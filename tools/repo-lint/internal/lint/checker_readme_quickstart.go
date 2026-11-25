package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ReadmeQuickStartChecker blocks staged edits inside the Quick Start section of README.md / README.en.md
// unless IHD_ALLOW_QUICKSTART_EDIT=1 is explicitly set by a human user.
type ReadmeQuickStartChecker struct{}

func NewReadmeQuickStartChecker() *ReadmeQuickStartChecker { return &ReadmeQuickStartChecker{} }

func (c *ReadmeQuickStartChecker) Name() string { return "readme-quickstart" }
func (c *ReadmeQuickStartChecker) Description() string {
	return "Blocks staged modifications inside the Quick Start section of README.md and README.en.md."
}

var hunkHeaderRE = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

func (c *ReadmeQuickStartChecker) Run(ctx *LintContext) ([]Diagnostic, error) {
	if ctx.Mode != ModeStaged {
		return nil, nil
	}
	if strings.TrimSpace(ctx.LookupEnv("IHD_ALLOW_QUICKSTART_EDIT")) == "1" {
		return nil, nil
	}

	stagedSet := make(map[string]bool, len(ctx.StagedFiles))
	for _, f := range ctx.StagedFiles {
		stagedSet[f] = true
	}

	targets := []struct {
		file     string
		headings []string
	}{
		{"README.md", []string{"クイックスタート", "quick start"}},    // txthc-ignore
		{"README.en.md", []string{"quick start", "クイックスタート"}}, // txthc-ignore
	}

	var diags []Diagnostic
	for _, target := range targets {
		if !stagedSet[target.file] {
			continue
		}
		abs := filepath.Join(ctx.RepoRoot, filepath.FromSlash(target.file))
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		secStart, secEnd, found := findH2SectionRange(string(data), target.headings)
		if !found {
			continue
		}

		diffText, err := ctx.Git.StagedDiffUnified0(ctx.RepoRoot, target.file)
		if err != nil {
			return nil, err
		}
		if hitLine, hit := diffIntersectsRange(diffText, secStart, secEnd); hit {
			diags = append(diags, Diagnostic{
				RuleID:      "readme-quickstart-modified",
				Checker:     c.Name(),
				Severity:    SeverityError,
				File:        target.file,
				Line:        hitLine,
				Message:     fmt.Sprintf("Staged diff modifies the protected Quick Start section (L%d-L%d) in %s", secStart, secEnd, target.file),
				Reason:      "Repository policy forbids modifying the README Quick Start section without explicit human approval.",
				Remediation: "Revert changes inside the Quick Start section (`git checkout -p " + target.file + "`). AI agents must NOT bypass this rule; if the user explicitly approves a Quick Start edit, commit with `IHD_ALLOW_QUICKSTART_EDIT=1`.",
			})
		}
	}

	return diags, nil
}

func findH2SectionRange(markdown string, headingTargets []string) (int, int, bool) {
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	startLine := 0
	for i, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if !strings.HasPrefix(trimmed, "## ") {
			continue
		}
		title := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")))
		if startLine == 0 {
			for _, want := range headingTargets {
				if strings.Contains(title, strings.ToLower(want)) {
					startLine = i + 1
					break
				}
			}
		} else {
			return startLine, i, true
		}
	}
	if startLine > 0 {
		return startLine, len(lines), true
	}
	return 0, 0, false
}

func diffIntersectsRange(diffText string, secStart, secEnd int) (int, bool) {
	lines := strings.Split(strings.ReplaceAll(diffText, "\r\n", "\n"), "\n")
	for _, line := range lines {
		m := hunkHeaderRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		newStart, _ := strconv.Atoi(m[1])
		newCount := 1
		if m[2] != "" {
			newCount, _ = strconv.Atoi(m[2])
		}
		hunkStart := newStart
		hunkEnd := newStart + newCount - 1
		if newCount == 0 {
			// Pure deletion anchored between newStart and newStart+1.
			hunkStart = newStart
			hunkEnd = newStart + 1
		}
		if hunkEnd >= secStart && hunkStart <= secEnd {
			if hunkStart < secStart {
				return secStart, true
			}
			return hunkStart, true
		}
	}
	return 0, false
}
