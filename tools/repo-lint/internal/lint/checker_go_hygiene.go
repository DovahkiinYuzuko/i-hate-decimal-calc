package lint

import (
	"encoding/json"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// RegisterGoHygieneAnalyzers registers all built-in Go AST hygiene analyzers
// (demonstrating the ASTAnalyzer seam reused by issue-183) plus the locale emoji checker.
func RegisterGoHygieneAnalyzers(r *Runner) {
	r.RegisterASTAnalyzer(&emptyErrBranchAnalyzer{})
	r.RegisterASTAnalyzer(&goEmojiAnalyzer{})
	r.RegisterASTAnalyzer(&blankErrAssignAnalyzer{})
	r.RegisterASTAnalyzer(&osPathConcatAnalyzer{})
	r.RegisterChecker(&localeEmojiChecker{})
}

// 1. Empty `if err != nil {}` branch (ERROR)
type emptyErrBranchAnalyzer struct{}

func (a *emptyErrBranchAnalyzer) Name() string { return "go-hygiene-empty-err-branch" }
func (a *emptyErrBranchAnalyzer) Description() string {
	return "Detects empty `if err != nil {}` branches that silently swallow errors."
}
func (a *emptyErrBranchAnalyzer) DefaultSeverity() Severity { return SeverityError }

func (a *emptyErrBranchAnalyzer) AnalyzeFile(ctx *LintContext, af *ASTFile) ([]Diagnostic, error) {
	var diags []Diagnostic
	ast.Inspect(af.File, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok || ifStmt.Body == nil || len(ifStmt.Body.List) != 0 || ifStmt.Else != nil {
			return true
		}
		if isErrNotNilCond(ifStmt.Cond) {
			pos := af.Fset.Position(ifStmt.Pos())
			diags = append(diags, Diagnostic{
				RuleID:      a.Name(),
				Checker:     "go-hygiene",
				Severity:    SeverityError,
				File:        af.RelPath,
				Line:        pos.Line,
				Col:         pos.Column,
				Message:     "Empty `if err != nil {}` block silently swallows error",
				Reason:      "Swallowing errors hides root causes and violates anti-band-aid engineering standards.",
				Remediation: "Handle, wrap, or return the error explicitly instead of leaving an empty `if err != nil {}` block.",
			})
		}
		return true
	})
	return diags, nil
}

func isErrNotNilCond(expr ast.Expr) bool {
	bin, ok := expr.(*ast.BinaryExpr)
	if !ok || bin.Op != token.NEQ {
		return false
	}
	return (isErrIdent(bin.X) && isNilIdent(bin.Y)) || (isNilIdent(bin.X) && isErrIdent(bin.Y))
}

func isErrIdent(expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	return id.Name == "err" || strings.HasSuffix(id.Name, "Err")
}

func isNilIdent(expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == "nil"
}

// 2. Emoji in Go string literals (ERROR)
type goEmojiAnalyzer struct{}

func (a *goEmojiAnalyzer) Name() string { return "go-hygiene-emoji" }
func (a *goEmojiAnalyzer) Description() string {
	return "Bans emojis in Go production CLI/error string literals."
}
func (a *goEmojiAnalyzer) DefaultSeverity() Severity { return SeverityError }

func (a *goEmojiAnalyzer) AnalyzeFile(ctx *LintContext, af *ASTFile) ([]Diagnostic, error) {
	var diags []Diagnostic
	ast.Inspect(af.File, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		val, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		if r, found := findForbiddenEmoji(val); found {
			pos := af.Fset.Position(lit.Pos())
			diags = append(diags, Diagnostic{
				RuleID:      a.Name(),
				Checker:     "go-hygiene",
				Severity:    SeverityError,
				File:        af.RelPath,
				Line:        pos.Line,
				Col:         pos.Column,
				Message:     "Emoji character " + strconv.QuoteRuneToASCII(r) + " found in Go string literal",
				Reason:      "Project policy prohibits emojis in CLI/system output strings to maintain formal tone and terminal compatibility.",
				Remediation: "Remove the emoji from the string literal and use plain ASCII/localized text instead.",
			})
		}
		return true
	})
	return diags, nil
}

// 2b. Emoji in internal/i18n/locales/*.json (ERROR)
type localeEmojiChecker struct{}

func (c *localeEmojiChecker) Name() string { return "go-hygiene-locale-emoji" }
func (c *localeEmojiChecker) Description() string {
	return "Bans emojis in internal/i18n/locales/*.json translation strings."
}

func (c *localeEmojiChecker) Run(ctx *LintContext) ([]Diagnostic, error) {
	localesDir := filepath.Join(ctx.RepoRoot, "internal", "i18n", "locales")
	entries, err := os.ReadDir(localesDir)
	if err != nil {
		return nil, nil
	}
	var diags []Diagnostic
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		rel := filepath.ToSlash(filepath.Join("internal", "i18n", "locales", entry.Name()))
		if ctx.Mode == ModeStaged && !containsString(ctx.StagedFiles, rel) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(localesDir, entry.Name()))
		if err != nil {
			continue
		}
		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}
		flat, err := loadLocaleFlat(filepath.Join(localesDir, entry.Name()))
		if err != nil {
			continue
		}
		for _, k := range sortedKeys(flat) {
			if r, found := findForbiddenEmoji(flat[k]); found {
				diags = append(diags, Diagnostic{
					RuleID:      "go-hygiene-emoji",
					Checker:     "go-hygiene",
					Severity:    SeverityError,
					File:        rel,
					Message:     "Emoji character " + strconv.QuoteRuneToASCII(r) + " found in locale key `" + k + "`",
					Reason:      "Project policy prohibits emojis in localized UI/CLI messages to maintain formal tone and terminal portability.",
					Remediation: "Remove the emoji from key `" + k + "` in `" + rel + "`.",
				})
			}
		}
	}
	return diags, nil
}

func findForbiddenEmoji(s string) (rune, bool) {
	for _, r := range s {
		if isForbiddenEmojiRune(r) {
			return r, true
		}
	}
	return 0, false
}

// isForbiddenEmojiRune matches pictographic emoji ranges while explicitly excluding
// mathematical symbols (U+2200..U+22FF), arrows (U+2190..U+21FF), Box Drawing (U+2500..U+257F),
// Geometric Shapes (U+25A0..U+25FF), and Braille Patterns (U+2800..U+28FF).
func isForbiddenEmojiRune(r rune) bool {
	switch {
	case r >= 0x1F300 && r <= 0x1FAFF:
		return true
	case r >= 0x1F000 && r <= 0x1F02F:
		return true
	case r >= 0x1F0A0 && r <= 0x1F0FF:
		return true
	case r >= 0x1F1E6 && r <= 0x1F1FF:
		return true
	case r >= 0x23E9 && r <= 0x23F3:
		return true
	case r >= 0x23F8 && r <= 0x23FA:
		return true
	case r == 0x26A0 || r == 0x26A1 || r == 0x2705 || r == 0x2728 || r == 0x274C || r == 0x274E || r == 0x2757 || r == 0x2764:
		return true
	case r >= 0x2753 && r <= 0x2755:
		return true
	}
	return false
}

// 3. Unjustified `_ = err` (WARNING)
type blankErrAssignAnalyzer struct{}

func (a *blankErrAssignAnalyzer) Name() string { return "go-hygiene-blank-err" }
func (a *blankErrAssignAnalyzer) Description() string {
	return "Warns on unjustified `_ = err` assignments that discard errors without comment."
}
func (a *blankErrAssignAnalyzer) DefaultSeverity() Severity { return SeverityWarning }

func (a *blankErrAssignAnalyzer) AnalyzeFile(ctx *LintContext, af *ASTFile) ([]Diagnostic, error) {
	var diags []Diagnostic
	ast.Inspect(af.File, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != len(assign.Rhs) {
			return true
		}
		for i := range assign.Lhs {
			lhsIdent, ok := assign.Lhs[i].(*ast.Ident)
			if !ok || lhsIdent.Name != "_" {
				continue
			}
			if isErrIdent(assign.Rhs[i]) {
				pos := af.Fset.Position(assign.Pos())
				diags = append(diags, Diagnostic{
					RuleID:      a.Name(),
					Checker:     "go-hygiene",
					Severity:    SeverityWarning,
					File:        af.RelPath,
					Line:        pos.Line,
					Col:         pos.Column,
					Message:     "Error value discarded via `_ = err` without justification",
					Reason:      "Unexplained `_ = err` assignments hide failure modes and make debugging harder.",
					Remediation: "Handle or return the error, or if intentionally ignored, add `// repo-lint:ignore go-hygiene-blank-err <concrete reason>`.",
				})
			}
		}
		return true
	})
	return diags, nil
}

// 4. OS-dependent path concatenation (WARNING)
type osPathConcatAnalyzer struct{}

func (a *osPathConcatAnalyzer) Name() string { return "go-hygiene-os-path-concat" }
func (a *osPathConcatAnalyzer) Description() string {
	return "Warns on string concatenation building filesystem paths instead of filepath.Join."
}
func (a *osPathConcatAnalyzer) DefaultSeverity() Severity { return SeverityWarning }

var pathLikeSuffixes = []string{
	".json", ".go", ".md", ".html", ".txt", ".csv", ".lean", ".ihd", ".log", ".yaml", ".yml", ".toml", ".svg", ".typ", ".exe",
}

var pathLikeDirs = []string{
	"locales/", "docs/", "cmd/", "internal/", "tools/", "formal/", "test-scripts/", "sample/", "exam/", "assets/",
}

func (a *osPathConcatAnalyzer) AnalyzeFile(ctx *LintContext, af *ASTFile) ([]Diagnostic, error) {
	var diags []Diagnostic
	ast.Inspect(af.File, func(n ast.Node) bool {
		bin, ok := n.(*ast.BinaryExpr)
		if !ok || bin.Op != token.ADD {
			return true
		}
		if isOSPathConcatOperand(bin.X) || isOSPathConcatOperand(bin.Y) {
			pos := af.Fset.Position(bin.Pos())
			diags = append(diags, Diagnostic{
				RuleID:      a.Name(),
				Checker:     "go-hygiene",
				Severity:    SeverityWarning,
				File:        af.RelPath,
				Line:        pos.Line,
				Col:         pos.Column,
				Message:     "Filesystem path constructed via string concatenation (`+`)",
				Reason:      "Hardcoded `/` or `\\` path concatenation breaks Windows/POSIX portability.",
				Remediation: "Use `filepath.Join(...)` for OS-agnostic path construction, or add `// repo-lint:ignore go-hygiene-os-path-concat <reason>` if this is not a filesystem path.",
			})
		}
		return true
	})
	return diags, nil
}

func isOSPathConcatOperand(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return false
	}
	val, err := strconv.Unquote(lit.Value)
	if err != nil || len(val) < 2 {
		return false
	}
	lower := strings.ToLower(val)

	// Exclude URLs and schemes
	if strings.Contains(lower, "://") {
		return false
	}
	// Backslash path segment like `\file` or `dir\` (excluding pure escape/LaTeX commands like `\frac`)
	if strings.HasPrefix(val, "\\") && hasPathExt(lower) {
		return true
	}
	if strings.HasSuffix(val, "\\") && len(val) >= 3 && isAlphaNum(val[len(val)-2]) {
		return true
	}
	// Forward slash with explicit file extension like `"/config.json"`
	if strings.HasPrefix(val, "/") && hasPathExt(lower) {
		return true
	}
	// Known repo directory prefix like `"locales/" + name`
	for _, d := range pathLikeDirs {
		if strings.HasSuffix(lower, d) && !strings.ContainsAny(lower, " \t\n") {
			return true
		}
	}
	return false
}

func hasPathExt(lower string) bool {
	for _, ext := range pathLikeSuffixes {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

func isAlphaNum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func containsString(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}
