package lint

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunOptions configures a single Runner execution.
type RunOptions struct {
	RepoRoot          string
	Mode              RunMode
	Only              []string
	Skip              []string
	SeverityOverrides map[string]Severity
	Git               GitClient
	EnvLookup         func(string) string
}

// RunResult aggregates diagnostics and summary counts from Runner.Run.
type RunResult struct {
	Mode        RunMode      `json:"mode"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Errors      int          `json:"errors"`
	Warnings    int          `json:"warnings"`
}

// ExitCode returns 1 if blocking diagnostics exist, or 0 otherwise.
func (r RunResult) ExitCode(warnAsError bool) int {
	if r.Errors > 0 {
		return 1
	}
	if warnAsError && r.Warnings > 0 {
		return 1
	}
	return 0
}

// Runner orchestrates registered Checkers and pluggable ASTAnalyzers.
type Runner struct {
	checkers     []Checker
	astAnalyzers []ASTAnalyzer
}

// NewRunner creates an empty Runner for custom composition and testing.
func NewRunner() *Runner {
	return &Runner{}
}

// RegisterChecker adds a repository-level Checker.
func (r *Runner) RegisterChecker(c Checker) {
	if c != nil {
		r.checkers = append(r.checkers, c)
	}
}

// RegisterASTAnalyzer adds a pluggable Go ASTAnalyzer (extension seam for issue-183).
func (r *Runner) RegisterASTAnalyzer(a ASTAnalyzer) {
	if a != nil {
		r.astAnalyzers = append(r.astAnalyzers, a)
	}
}

// Rules returns metadata for all registered checkers and AST analyzers.
func (r *Runner) Rules() []RuleInfo {
	infos := make([]RuleInfo, 0, len(r.checkers)+len(r.astAnalyzers))
	for _, c := range r.checkers {
		infos = append(infos, RuleInfo{
			Name:            c.Name(),
			Kind:            "checker",
			DefaultSeverity: SeverityError,
			Description:     c.Description(),
		})
	}
	for _, a := range r.astAnalyzers {
		infos = append(infos, RuleInfo{
			Name:            a.Name(),
			Kind:            "ast-analyzer",
			DefaultSeverity: a.DefaultSeverity(),
			Description:     a.Description(),
		})
	}
	return infos
}

// Run executes selected checkers and AST analyzers against the configured target scope.
func (r *Runner) Run(opts RunOptions) (RunResult, error) {
	root := opts.RepoRoot
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return RunResult{}, err
	}

	mode := opts.Mode
	if mode == "" {
		mode = ModeAll
	}
	gitClient := opts.Git
	if gitClient == nil {
		gitClient = ExecGitClient{}
	}
	envFn := opts.EnvLookup
	if envFn == nil {
		envFn = os.Getenv
	}

	onlySet := toNormalizedSet(opts.Only)
	skipSet := toNormalizedSet(opts.Skip)

	ctx := &LintContext{
		RepoRoot:  absRoot,
		Mode:      mode,
		Git:       gitClient,
		EnvLookup: envFn,
	}

	if mode == ModeStaged {
		staged, err := gitClient.StagedFiles(absRoot)
		if err != nil {
			return RunResult{}, err
		}
		ctx.StagedFiles = staged
	} else {
		tracked, err := gitClient.TrackedFiles(absRoot)
		if err != nil {
			return RunResult{}, err
		}
		ctx.TrackedFiles = tracked
	}

	var allDiags []Diagnostic

	// 1. Run registered Checkers
	for _, c := range r.checkers {
		if !shouldRun(c.Name(), onlySet, skipSet) {
			continue
		}
		diags, err := c.Run(ctx)
		if err != nil {
			return RunResult{}, fmt.Errorf("checker %s: %w", c.Name(), err) // txthc-ignore
		}
		allDiags = append(allDiags, diags...)
	}

	// 2. Run registered ASTAnalyzers (single AST parse pass shared across all analyzers)
	activeAnalyzers := make([]ASTAnalyzer, 0, len(r.astAnalyzers))
	for _, a := range r.astAnalyzers {
		if shouldRunASTAnalyzer(a.Name(), onlySet, skipSet) {
			activeAnalyzers = append(activeAnalyzers, a)
		}
	}

	if len(activeAnalyzers) > 0 {
		astFiles, ignoreDiags, err := loadASTFiles(absRoot, mode, ctx.StagedFiles)
		if err != nil {
			return RunResult{}, err
		}
		ctx.ASTFiles = astFiles
		allDiags = append(allDiags, ignoreDiags...)

		for _, af := range astFiles {
			for _, a := range activeAnalyzers {
				diags, err := a.AnalyzeFile(ctx, af)
				if err != nil {
					return RunResult{}, fmt.Errorf("ast-analyzer %s (%s): %w", a.Name(), af.RelPath, err) // txthc-ignore
				}
				for _, d := range diags {
					if d.Severity == "" {
						d.Severity = a.DefaultSeverity()
					}
					if d.Checker == "" {
						d.Checker = "go-hygiene"
					}
					if IsSuppressed(af.Ignores, d.Line, d.RuleID) {
						continue
					}
					allDiags = append(allDiags, d)
				}
			}
		}
	}

	// 3. Apply severity overrides, ensure non-empty reason/remediation, and sort deterministically.
	res := RunResult{
		Mode:        mode,
		Diagnostics: make([]Diagnostic, 0, len(allDiags)),
	}
	for _, d := range allDiags {
		if sev, ok := opts.SeverityOverrides[d.RuleID]; ok {
			d.Severity = sev
		} else if sev, ok := opts.SeverityOverrides[d.Checker]; ok {
			d.Severity = sev
		}
		if d.Severity == "" {
			d.Severity = SeverityError
		}
		if d.Severity == SeverityError {
			res.Errors++
		} else {
			res.Warnings++
		}
		res.Diagnostics = append(res.Diagnostics, d)
	}

	sort.Slice(res.Diagnostics, func(i, j int) bool {
		a, b := res.Diagnostics[i], res.Diagnostics[j]
		if a.Severity != b.Severity {
			return a.Severity == SeverityError
		}
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Col != b.Col {
			return a.Col < b.Col
		}
		if a.RuleID != b.RuleID {
			return a.RuleID < b.RuleID
		}
		return a.Message < b.Message
	})

	return res, nil
}

func loadASTFiles(absRoot string, mode RunMode, stagedFiles []string) ([]*ASTFile, []Diagnostic, error) {
	var targetRelPaths []string

	if mode == ModeStaged {
		for _, rel := range stagedFiles {
			if isProdGoFile(rel) {
				targetRelPaths = append(targetRelPaths, rel)
			}
		}
	} else {
		for _, dir := range []string{"cmd", "internal"} {
			rootDir := filepath.Join(absRoot, dir)
			if _, err := os.Stat(rootDir); os.IsNotExist(err) {
				continue
			}
			err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				rel, err := filepath.Rel(absRoot, path)
				if err != nil {
					return err
				}
				relSlash := filepath.ToSlash(rel)
				if isProdGoFile(relSlash) {
					targetRelPaths = append(targetRelPaths, relSlash)
				}
				return nil
			})
			if err != nil {
				return nil, nil, err
			}
		}
	}

	sort.Strings(targetRelPaths)

	files := make([]*ASTFile, 0, len(targetRelPaths))
	var invalidDiags []Diagnostic

	for _, rel := range targetRelPaths {
		abs := filepath.Join(absRoot, filepath.FromSlash(rel))
		src, err := os.ReadFile(abs)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, nil, err
		}
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, rel, src, parser.ParseComments)
		if err != nil {
			invalidDiags = append(invalidDiags, Diagnostic{
				RuleID:      "go-ast-parse-error",
				Checker:     "ast-analyzers",
				Severity:    SeverityError,
				File:        rel,
				Message:     "Failed to parse Go source file: " + err.Error(),
				Reason:      "Syntax errors prevent deterministic AST verification.",
				Remediation: "Fix the Go syntax error and re-run `go test ./...`.",
			})
			continue
		}
		ignores, invalid := ParseIgnoreDirectives(rel, fset, parsed, src)
		invalidDiags = append(invalidDiags, invalid...)
		files = append(files, &ASTFile{
			RelPath: rel,
			AbsPath: abs,
			Fset:    fset,
			File:    parsed,
			Source:  src,
			Ignores: ignores,
		})
	}

	return files, invalidDiags, nil
}

func isProdGoFile(relSlash string) bool {
	if !strings.HasSuffix(relSlash, ".go") || strings.HasSuffix(relSlash, "_test.go") {
		return false
	}
	return strings.HasPrefix(relSlash, "cmd/") || strings.HasPrefix(relSlash, "internal/")
}

func toNormalizedSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, it := range items {
		it = strings.TrimSpace(strings.ToLower(it))
		if it != "" {
			set[it] = true
		}
	}
	return set
}

func shouldRun(name string, onlySet, skipSet map[string]bool) bool {
	n := strings.ToLower(name)
	if len(onlySet) > 0 && !onlySet[n] {
		return false
	}
	if skipSet[n] {
		return false
	}
	return true
}

func shouldRunASTAnalyzer(name string, onlySet, skipSet map[string]bool) bool {
	n := strings.ToLower(name)
	if len(onlySet) > 0 && !onlySet[n] && !onlySet["go-hygiene"] && !onlySet["ast-analyzers"] {
		return false
	}
	if skipSet[n] || skipSet["go-hygiene"] || skipSet["ast-analyzers"] {
		return false
	}
	return true
}

// FormatText renders diagnostics in a human- and LLM-readable report containing
// RuleID, location, summary, reason, and concrete remediation steps.
func FormatText(res RunResult) string {
	var sb strings.Builder
	if len(res.Diagnostics) == 0 {
		sb.WriteString(fmt.Sprintf("[repo-lint] PASS (%s mode): 0 errors, 0 warnings\n", res.Mode))
		return sb.String()
	}

	for _, d := range res.Diagnostics {
		sevLabel := "WARNING"
		if d.Severity == SeverityError {
			sevLabel = "ERROR"
		}
		loc := d.File
		if d.Line > 0 {
			if d.Col > 0 {
				loc = fmt.Sprintf("%s:%d:%d", d.File, d.Line, d.Col)
			} else {
				loc = fmt.Sprintf("%s:%d", d.File, d.Line)
			}
		}
		sb.WriteString(fmt.Sprintf("[%s] %s (%s)\n", sevLabel, d.RuleID, loc))
		sb.WriteString(fmt.Sprintf("  Summary    : %s\n", d.Message))
		sb.WriteString(fmt.Sprintf("  Reason     : %s\n", d.Reason))
		sb.WriteString(fmt.Sprintf("  Remediation: %s\n\n", d.Remediation))
	}

	status := "PASS"
	if res.Errors > 0 {
		status = "FAIL"
	}
	sb.WriteString(fmt.Sprintf("[repo-lint] %s (%s mode): %d error(s), %d warning(s)\n", status, res.Mode, res.Errors, res.Warnings))
	return sb.String()
}

// FormatJSON renders diagnostics as indented JSON.
func FormatJSON(res RunResult) (string, error) {
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b) + "\n", nil
}
