package lint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// NoticeSyncChecker verifies that go.mod and tools/reference-builder/package.json dependencies
// are documented in NOTICE.md and tools/reference-builder/NOTICE.md respectively.
type NoticeSyncChecker struct{}

func NewNoticeSyncChecker() *NoticeSyncChecker { return &NoticeSyncChecker{} }

func (c *NoticeSyncChecker) Name() string { return "notice-sync" }
func (c *NoticeSyncChecker) Description() string {
	return "Checks that go.mod dependencies and tools/reference-builder/package.json dependencies are listed in NOTICE.md."
}

func (c *NoticeSyncChecker) Run(ctx *LintContext) ([]Diagnostic, error) {
	if ctx.Mode == ModeStaged && !hasRelevantNoticeStage(ctx.StagedFiles) {
		return nil, nil
	}

	var diags []Diagnostic

	// 1. go.mod -> root NOTICE.md
	goModPath := filepath.Join(ctx.RepoRoot, "go.mod")
	rootNoticePath := filepath.Join(ctx.RepoRoot, "NOTICE.md")
	if goModBytes, err := os.ReadFile(goModPath); err == nil {
		noticeBytes, noticeErr := os.ReadFile(rootNoticePath)
		noticeLower := ""
		if noticeErr == nil {
			noticeLower = strings.ToLower(string(noticeBytes))
		}
		deps := parseGoModRequires(string(goModBytes))
		for _, dep := range deps {
			if noticeErr != nil || !strings.Contains(noticeLower, strings.ToLower(dep)) {
				diags = append(diags, Diagnostic{
					RuleID:      "notice-go-missing",
					Checker:     c.Name(),
					Severity:    SeverityError,
					File:        "NOTICE.md",
					Message:     "Go dependency `" + dep + "` from go.mod is missing from NOTICE.md",
					Reason:      "All third-party Go modules declared in `go.mod` must carry license and copyright notices in `NOTICE.md`.",
					Remediation: "Add an entry for `" + dep + "` (version, license, copyright) to `NOTICE.md` (follow `managing-libraries-is-a-drag` skill).",
				})
			}
		}
	}

	// 2. tools/reference-builder/package.json -> tools/reference-builder/NOTICE.md (fallback root NOTICE.md)
	pkgJSONPath := filepath.Join(ctx.RepoRoot, "tools", "reference-builder", "package.json")
	if pkgBytes, err := os.ReadFile(pkgJSONPath); err == nil {
		var pkg struct {
			Dependencies map[string]string `json:"dependencies"`
		}
		if err := json.Unmarshal(pkgBytes, &pkg); err == nil {
			subNoticePath := filepath.Join(ctx.RepoRoot, "tools", "reference-builder", "NOTICE.md")
			subNotice, _ := os.ReadFile(subNoticePath)
			rootNotice, _ := os.ReadFile(rootNoticePath)
			combinedLower := strings.ToLower(string(subNotice) + "\n" + string(rootNotice))

			names := make([]string, 0, len(pkg.Dependencies))
			for name := range pkg.Dependencies {
				names = append(names, name)
			}
			sort.Strings(names)

			for _, name := range names {
				if !strings.Contains(combinedLower, strings.ToLower(name)) {
					diags = append(diags, Diagnostic{
						RuleID:      "notice-npm-missing",
						Checker:     c.Name(),
						Severity:    SeverityError,
						File:        "tools/reference-builder/NOTICE.md",
						Message:     "npm dependency `" + name + "` from tools/reference-builder/package.json is missing from NOTICE.md",
						Reason:      "All third-party npm packages used by `tools/reference-builder` must carry third-party license notices.",
						Remediation: "Add `" + name + "` (version, license, copyright) to `tools/reference-builder/NOTICE.md`.",
					})
				}
			}
		}
	}

	return diags, nil
}

func hasRelevantNoticeStage(staged []string) bool {
	for _, f := range staged {
		switch f {
		case "go.mod", "NOTICE.md", "tools/reference-builder/package.json", "tools/reference-builder/NOTICE.md":
			return true
		}
	}
	return false
}

func parseGoModRequires(content string) []string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var deps []string
	seen := make(map[string]bool)
	inBlock := false

	for _, raw := range lines {
		line := raw
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "require (") || line == "require(" {
			inBlock = true
			continue
		}
		if inBlock {
			if line == ")" {
				inBlock = false
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 2 && !seen[fields[0]] {
				seen[fields[0]] = true
				deps = append(deps, fields[0])
			}
			continue
		}
		if strings.HasPrefix(line, "require ") {
			fields := strings.Fields(strings.TrimPrefix(line, "require "))
			if len(fields) >= 2 && !seen[fields[0]] {
				seen[fields[0]] = true
				deps = append(deps, fields[0])
			}
		}
	}
	sort.Strings(deps)
	return deps
}
