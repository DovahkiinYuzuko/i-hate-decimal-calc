package lint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// LeanPinningChecker validates formal/lean-toolchain pinning, 40-char SHA revisions in
// formal/lake-manifest.json, and bans `lake update` / unpinned branch tracking in CI workflows.
type LeanPinningChecker struct{}

func NewLeanPinningChecker() *LeanPinningChecker { return &LeanPinningChecker{} }

func (c *LeanPinningChecker) Name() string { return "lean-pinning" }
func (c *LeanPinningChecker) Description() string {
	return "Verifies Lean 4 toolchain/manifest SHA pinning and bans CI lake update or unpinned branch tracking."
}

var (
	sha40RE              = regexp.MustCompile(`^[0-9a-f]{40}$`)
	pinnedToolchainRE    = regexp.MustCompile(`^[^:\s]+:v\d+\.\d+\.\d+`)
	ciLakeUpdateRE       = regexp.MustCompile(`\blake\s+update\b`)
	ciUnpinnedToolchainRE = regexp.MustCompile(`\belan\s+(default|override\s+set)\s+\S*(nightly|master|main)\b`)
	ciUnpinnedMathlibRE  = regexp.MustCompile(`mathlib4[^\n]*(@master|@main|-b\s+(master|main)|--branch[=\s]+(master|main))`)
)

func (c *LeanPinningChecker) Run(ctx *LintContext) ([]Diagnostic, error) {
	if ctx.Mode == ModeStaged && !hasRelevantLeanStage(ctx.StagedFiles) {
		return nil, nil
	}

	var diags []Diagnostic

	// 1. formal/lean-toolchain
	tcPath := filepath.Join(ctx.RepoRoot, "formal", "lean-toolchain")
	if data, err := os.ReadFile(tcPath); err == nil {
		tc := strings.TrimSpace(string(data))
		lower := strings.ToLower(tc)
		if tc == "" || strings.Contains(lower, "nightly") || strings.HasSuffix(lower, ":master") || strings.HasSuffix(lower, ":main") || !pinnedToolchainRE.MatchString(tc) {
			diags = append(diags, Diagnostic{
				RuleID:      "lean-toolchain-unpinned",
				Checker:     c.Name(),
				Severity:    SeverityError,
				File:        "formal/lean-toolchain",
				Line:        1,
				Message:     "Lean toolchain `" + tc + "` is not pinned to an explicit version tag",
				Reason:      "Unpinned Lean toolchains (nightly/master/main) cause sudden CI breakage when Mathlib4 caches diverge.",
				Remediation: "Pin `formal/lean-toolchain` to a verified version tag (e.g. `leanprover/lean4:v4.36.0-rc2`) after local cache and proof verification.",
			})
		}
	}

	// 2. formal/lake-manifest.json
	manifestPath := filepath.Join(ctx.RepoRoot, "formal", "lake-manifest.json")
	if data, err := os.ReadFile(manifestPath); err == nil {
		var manifest struct {
			Packages []struct {
				Name string `json:"name"`
				Rev  string `json:"rev"`
			} `json:"packages"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			diags = append(diags, Diagnostic{
				RuleID:      "lean-manifest-unpinned",
				Checker:     c.Name(),
				Severity:    SeverityError,
				File:        "formal/lake-manifest.json",
				Line:        1,
				Message:     "Failed to parse formal/lake-manifest.json: " + err.Error(),
				Reason:      "Corrupted lake-manifest.json prevents reproducible Mathlib4 builds.",
				Remediation: "Regenerate `formal/lake-manifest.json` in local WSL Lean environment and verify `lake build Proofs`.",
			})
		} else {
			for _, pkg := range manifest.Packages {
				if !sha40RE.MatchString(pkg.Rev) {
					diags = append(diags, Diagnostic{
						RuleID:      "lean-manifest-unpinned",
						Checker:     c.Name(),
						Severity:    SeverityError,
						File:        "formal/lake-manifest.json",
						Message:     "Package `" + pkg.Name + "` has non-SHA revision `" + pkg.Rev + "`",
						Reason:      "Every package in `formal/lake-manifest.json` must be pinned to a full 40-character Git commit SHA to ensure reproducible CI cache hits.",
						Remediation: "Pin `" + pkg.Name + "` to a verified 40-char commit SHA via local `lake` update + `lake exe cache get` + `lake build Proofs`.",
					})
				}
			}
		}
	}

	// 3. .github/workflows/*.yml & *.yaml
	wfDir := filepath.Join(ctx.RepoRoot, ".github", "workflows")
	if entries, err := os.ReadDir(wfDir); err == nil {
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			lowerName := strings.ToLower(e.Name())
			if !strings.HasSuffix(lowerName, ".yml") && !strings.HasSuffix(lowerName, ".yaml") {
				continue
			}
			rel := filepath.ToSlash(filepath.Join(".github", "workflows", e.Name()))
			data, err := os.ReadFile(filepath.Join(wfDir, e.Name()))
			if err != nil {
				continue
			}
			lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
			for i, rawLine := range lines {
				codeLine := stripShellYAMLComment(rawLine)
				if codeLine == "" {
					continue
				}
				if ciLakeUpdateRE.MatchString(codeLine) || ciUnpinnedToolchainRE.MatchString(codeLine) || ciUnpinnedMathlibRE.MatchString(codeLine) {
					diags = append(diags, Diagnostic{
						RuleID:      "lean-ci-lake-update",
						Checker:     c.Name(),
						Severity:    SeverityError,
						File:        rel,
						Line:        i + 1,
						Message:     "CI workflow runs unpinned Lean/Mathlib update command: `" + strings.TrimSpace(codeLine) + "`",
						Reason:      "Running `lake update` or tracking unpinned `master`/`main`/`nightly` branches in GitHub Actions breaks Mathlib4 cache reproducibility and causes sudden CI failures.",
						Remediation: "Remove `lake update` / unpinned branch tracking from `" + rel + "` and rely on the pinned `formal/lean-toolchain` and `formal/lake-manifest.json` with `lake exe cache get`.",
					})
				}
			}
		}
	}

	return diags, nil
}

func hasRelevantLeanStage(staged []string) bool {
	for _, f := range staged {
		if f == "formal/lean-toolchain" || f == "formal/lake-manifest.json" || strings.HasPrefix(f, ".github/workflows/") {
			return true
		}
	}
	return false
}

func stripShellYAMLComment(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return ""
	}
	inSingle := false
	inDouble := false
	for i := 0; i < len(trimmed); i++ {
		ch := trimmed[i]
		switch ch {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle && (i == 0 || trimmed[i-1] != '\\') {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble && i > 0 && (trimmed[i-1] == ' ' || trimmed[i-1] == '\t') {
				return strings.TrimSpace(trimmed[:i])
			}
		}
	}
	return trimmed
}
