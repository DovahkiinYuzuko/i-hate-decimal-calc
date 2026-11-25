package lint

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// GitWhitelistChecker validates Git-tracked/staged paths against the repository allowlist
// and inspects local .gitignores/*.gitignore files when present.
type GitWhitelistChecker struct{}

func NewGitWhitelistChecker() *GitWhitelistChecker { return &GitWhitelistChecker{} }

func (c *GitWhitelistChecker) Name() string { return "git-whitelist" }
func (c *GitWhitelistChecker) Description() string {
	return "Enforces Git tracked/staged documentation whitelist, forbidden artifact bans, and .gitignores structure."
}

var forbiddenPrefixes = []string{
	".lake/",
	"docs/local-issues/",
	"docs/variables'n'functions/",
	"docs/domain-knowledge/",
	"docs/git-descriptions/",
	"docs/licenses-cache/",
	"docs/project-compiled/",
	"docs/task_queues/",
	"docs/using-library/",
	"briefing-board/",
	"scratch/",
	"gh-actions-log/",
	".agents/",
	".gitignores/",
	".research/",
	"research/",
	"somebody-help-me-answer/",
	".gotmp/",
	".history/",
}

var forbiddenExactFiles = map[string]bool{
	"variables_functions_audit_report.md": true,
	"docs/reference_audit_report.md":      true,
}

func (c *GitWhitelistChecker) Run(ctx *LintContext) ([]Diagnostic, error) {
	files := ctx.TrackedFiles
	if ctx.Mode == ModeStaged {
		files = ctx.StagedFiles
	}

	var diags []Diagnostic
	for _, raw := range files {
		p := strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/")
		if p == "" {
			continue
		}

		if reason, forbidden := matchForbiddenArtifact(p); forbidden {
			diags = append(diags, Diagnostic{
				RuleID:      "git-forbidden-artifact",
				Checker:     c.Name(),
				Severity:    SeverityError,
				File:        p,
				Message:     "Forbidden local/internal artifact is tracked or staged in Git",
				Reason:      reason,
				Remediation: "Remove the file from Git staging/index (`git restore --staged " + p + "` or `git rm --cached " + p + "`) and verify `.gitignore` via `/gitignore-soc-skill`.",
			})
			continue
		}

		if isDocPath(p) && !isAllowedPublicDoc(p) {
			diags = append(diags, Diagnostic{
				RuleID:      "git-unlisted-doc",
				Checker:     c.Name(),
				Severity:    SeverityError,
				File:        p,
				Message:     "Documentation file is outside the Git tracking allowlist",
				Reason:      "Only curated public documentation (`README*.md`, `NOTICE.md`, `LICENSE.*`, `docs/development.*.md`, `docs/references/*/*.{md,html}`, `exam/*.md`, `sample/sample.*.md`, `tools/*/{README,NOTICE}.md`) may be tracked in Git.",
				Remediation: "Unstage/untrack `" + p + "` (`git restore --staged " + p + "`), or if it is an intentional public document, add it to the allowlist in `tools/repo-lint` after user approval.",
			})
		}
	}

	// Inspect local .gitignores/ when present (skipped in CI where .gitignores/ is untracked).
	gitignoresDir := filepath.Join(ctx.RepoRoot, ".gitignores")
	if info, err := os.Stat(gitignoresDir); err == nil && info.IsDir() {
		diags = append(diags, c.checkGitignoresDir(ctx.RepoRoot, gitignoresDir)...)
	}

	return diags, nil
}

func matchForbiddenArtifact(p string) (string, bool) {
	lower := strings.ToLower(p)
	base := path.Base(lower)

	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return "Environment secret files (`.env*`) must never be tracked or committed to Git.", true
	}
	if strings.Contains("/"+lower, "/.lake/") {
		return "Lean build cache directory (`.lake/`) must stay excluded from Git to prevent repository bloat and cross-OS slowdowns.", true
	}
	if forbiddenExactFiles[lower] {
		return "Auto-generated audit reports must be resolved and deleted rather than committed.", true
	}
	for _, pref := range forbiddenPrefixes {
		if strings.HasPrefix(lower, strings.ToLower(pref)) {
			return "Internal/local directory (`" + pref + "`) is excluded from Git tracking by repository policy.", true
		}
	}
	return "", false
}

func isDocPath(p string) bool {
	lower := strings.ToLower(p)
	return strings.HasSuffix(lower, ".md") || strings.HasPrefix(lower, "docs/")
}

func isAllowedPublicDoc(p string) bool {
	if p == "README.md" || p == "README.en.md" || p == "NOTICE.md" {
		return true
	}
	if strings.HasPrefix(p, "LICENSE.") && !strings.Contains(p, "/") {
		return true
	}
	if ok, _ := path.Match("docs/development.*.md", p); ok {
		return true
	}
	if ok, _ := path.Match("docs/references/*/*.md", p); ok {
		return true
	}
	if ok, _ := path.Match("docs/references/*/*.html", p); ok {
		return true
	}
	if ok, _ := path.Match("exam/*.md", p); ok {
		return true
	}
	if ok, _ := path.Match("sample/sample.*.md", p); ok {
		return true
	}
	if ok, _ := path.Match("tools/*/README.md", p); ok {
		return true
	}
	if ok, _ := path.Match("tools/*/NOTICE.md", p); ok {
		return true
	}
	return false
}

func (c *GitWhitelistChecker) checkGitignoresDir(repoRoot, dir string) []Diagnostic {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var diags []Diagnostic

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".gitignore") {
			continue
		}
		relFile := path.Join(".gitignores", entry.Name())
		folderName := strings.TrimSuffix(entry.Name(), ".gitignore")

		if folderName != "root" {
			targetDir := filepath.Join(repoRoot, folderName)
			if st, err := os.Stat(targetDir); err != nil || !st.IsDir() {
				diags = append(diags, Diagnostic{
					RuleID:      "gitignores-dir-mismatch",
					Checker:     c.Name(),
					Severity:    SeverityWarning,
					File:        relFile,
					Line:        1,
					Message:     "Fragment filename `" + entry.Name() + "` does not match any top-level directory `" + folderName + "/`",
					Reason:      "Each fragment under `.gitignores/` should match its target directory name so typos (e.g. `fomal.gitignore`) do not silently break whitelist management.",
					Remediation: "Rename `" + relFile + "` to match the real directory name and re-merge `.gitignore` using `/gitignore-soc-skill`.",
				})
			}
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
		for idx, rawLine := range lines {
			line := strings.TrimSpace(rawLine)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			lineNum := idx + 1
			if !strings.HasPrefix(line, "!") {
				diags = append(diags, Diagnostic{
					RuleID:      "gitignores-structure",
					Checker:     c.Name(),
					Severity:    SeverityWarning,
					File:        relFile,
					Line:        lineNum,
					Message:     "Non-whitelist rule `" + line + "` found in fragment file",
					Reason:      "Repository `.gitignores/*.gitignore` fragments use whitelist (`!`) exception entries under a root `*` ignore policy.",
					Remediation: "Convert the rule to a `!` allowlist entry or remove it, then re-merge `.gitignore` via `/gitignore-soc-skill`.",
				})
				continue
			}
			if strings.Contains(line[1:], "!") {
				diags = append(diags, Diagnostic{
					RuleID:      "gitignores-structure",
					Checker:     c.Name(),
					Severity:    SeverityWarning,
					File:        relFile,
					Line:        lineNum,
					Message:     "Multiple `!` entries appear concatenated on a single line: `" + line + "`",
					Reason:      "A missing newline merges two whitelist patterns into one invalid pattern, silently excluding both intended files from Git.",
					Remediation: "Split the line into separate `!` lines in `" + relFile + "` and re-merge `.gitignore` via `/gitignore-soc-skill`.",
				})
			}
		}
	}

	return diags
}
