package lint

import (
	"testing"
)

func hasRule(diags []Diagnostic, ruleID string) bool {
	for _, d := range diags {
		if d.RuleID == ruleID {
			return true
		}
	}
	return false
}

func assertAllHaveReasonAndRemediation(t *testing.T, diags []Diagnostic) {
	t.Helper()
	for _, d := range diags {
		if d.RuleID == "" || d.Message == "" || d.Reason == "" || d.Remediation == "" {
			t.Errorf("diagnostic missing mandatory fields (RuleID/Message/Reason/Remediation): %+v", d)
		}
	}
}

func TestCheckerGitWhitelist(t *testing.T) {
	t.Run("clean tracked files and valid gitignores", func(t *testing.T) {
		tmp := t.TempDir()
		writeFixture(t, tmp, "formal/lean-toolchain", "leanprover/lean4:v4.36.0-rc2\n")
		writeFixture(t, tmp, ".gitignores/formal.gitignore", "!formal/\n!formal/lean-toolchain\n")
		writeFixture(t, tmp, ".gitignores/root.gitignore", "!README.md\n")

		git := &fakeGitClient{
			tracked: []string{
				"README.md",
				"README.en.md",
				"NOTICE.md",
				"LICENSE.MIT",
				"docs/development.ja.md",
				"docs/development.en.md",
				"docs/references/ja/reference.ja.md",
				"docs/references/ja/reference.ja.html",
				"exam/problems.ja.md",
				"sample/sample.en.md",
				"tools/reference-builder/NOTICE.md",
				"tools/reference-builder/README.md",
				"tools/sympy-oracle/README.md",
				".githooks/pre-commit",
				"cmd/ihd/main.go",
			},
		}

		c := NewGitWhitelistChecker()
		diags, err := c.Run(&LintContext{RepoRoot: tmp, Mode: ModeAll, Git: git, TrackedFiles: git.tracked})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(diags) != 0 {
			t.Errorf("expected 0 diagnostics, got %+v", diags)
		}
	})

	t.Run("detects forbidden artifacts, unlisted docs, and gitignores defects", func(t *testing.T) {
		tmp := t.TempDir()
		writeFixture(t, tmp, "formal/lean-toolchain", "leanprover/lean4:v4.36.0-rc2\n")
		writeFixture(t, tmp, ".gitignores/fomal.gitignore", "!formal/\n")
		writeFixture(t, tmp, ".gitignores/formal.gitignore", "!formal/lake-manifest.json!formal/Tools/\nnon_whitelist_line\n")

		git := &fakeGitClient{
			tracked: []string{
				"docs/local-issues/wip/issue-195.md",
				"formal/.lake/build/foo.olean",
				".env",
				"variables_functions_audit_report.md",
				"e2e.md",
			},
		}

		c := NewGitWhitelistChecker()
		diags, err := c.Run(&LintContext{RepoRoot: tmp, Mode: ModeAll, Git: git, TrackedFiles: git.tracked})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertAllHaveReasonAndRemediation(t, diags)

		for _, rule := range []string{
			"git-forbidden-artifact",
			"git-unlisted-doc",
			"gitignores-dir-mismatch",
			"gitignores-structure",
		} {
			if !hasRule(diags, rule) {
				t.Errorf("expected rule %q to trigger, got %+v", rule, diags)
			}
		}
	})

	t.Run("gracefully skips gitignores check when directory absent (CI)", func(t *testing.T) {
		tmp := t.TempDir()
		git := &fakeGitClient{tracked: []string{"README.md", "go.mod"}}
		c := NewGitWhitelistChecker()
		diags, err := c.Run(&LintContext{RepoRoot: tmp, Mode: ModeAll, Git: git, TrackedFiles: git.tracked})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(diags) != 0 {
			t.Errorf("expected 0 diagnostics in CI without .gitignores, got %+v", diags)
		}
	})
}

func TestCheckerI18nAST(t *testing.T) {
	t.Run("clean i18n keys and matching format verbs", func(t *testing.T) {
		tmp := t.TempDir()
		writeFixture(t, tmp, "internal/i18n/locales/en.json", `{
  "meta": {"name": "English", "name_local": "English"},
  "errors": {
    "div_zero": "division by zero",
    "bad_arg": "invalid argument %s (expected %d, 100%% exact)"
  },
  "dyn": {
    "item": "dynamic item"
  }
}`)
		writeFixture(t, tmp, "internal/i18n/locales/ja.json", `{
  "meta": {"name": "Japanese", "name_local": "日本語"},
  "errors": {
    "div_zero": "ゼロ除算です",
    "bad_arg": "無効な引数 %s (期待値 %d, 100%% 厳密)"
  },
  "dyn": {
    "item": "動的項目"
  }
}`)
		writeFixture(t, tmp, "internal/calc/demo.go", `package calc

import (
	"fmt"
	"os"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

func demo(k string) {
	_ = i18n.T("errors.div_zero")
	_ = i18n.T("errors.bad_arg", "x", 2)
	_, _ = fmt.Fprintf(os.Stderr, i18n.T("errors.bad_arg")+"\n", "y", 3)
	_ = i18n.T("dyn." + k)
}
`)

		c := NewI18nASTChecker()
		diags, err := c.Run(&LintContext{RepoRoot: tmp, Mode: ModeAll})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(diags) != 0 {
			t.Errorf("expected 0 diagnostics, got %+v", diags)
		}
	})

	t.Run("detects parity mismatch, percent-w, missing key, arg mismatch, and unused key", func(t *testing.T) {
		tmp := t.TempDir()
		writeFixture(t, tmp, "internal/i18n/locales/en.json", `{
  "meta": {"name": "English", "name_local": "English"},
  "errors": {
    "only_en": "only in en",
    "bad_w": "wrap %w",
    "fmt_two": "got %s and %d",
    "unused_key": "never referenced"
  }
}`)
		writeFixture(t, tmp, "internal/i18n/locales/ja.json", `{
  "meta": {"name": "Japanese", "name_local": "日本語"},
  "errors": {
    "bad_w": "ラップ %w",
    "fmt_two": "%s と %d",
    "unused_key": "未使用"
  }
}`)
		writeFixture(t, tmp, "cmd/ihd/demo.go", `package main

import "github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"

func demo() {
	_ = i18n.T("errors.does_not_exist")
	_ = i18n.T("errors.fmt_two", "only-one-arg")
	_ = i18n.T("errors.bad_w", "x")
}
`)

		c := NewI18nASTChecker()
		diags, err := c.Run(&LintContext{RepoRoot: tmp, Mode: ModeAll})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertAllHaveReasonAndRemediation(t, diags)

		for _, rule := range []string{
			"i18n-locale-parity",
			"i18n-percent-w",
			"i18n-missing-key",
			"i18n-format-args",
			"i18n-unused-key",
		} {
			if !hasRule(diags, rule) {
				t.Errorf("expected rule %q to trigger, got %+v", rule, diags)
			}
		}
	})
}

func TestCheckerNoticeSync(t *testing.T) {
	t.Run("clean go.mod and reference-builder package.json", func(t *testing.T) {
		tmp := t.TempDir()
		writeFixture(t, tmp, "go.mod", `module example.com/test

go 1.26.5

require (
	github.com/nyaosorg/go-readline-ny v1.16.1
)

require (
	golang.org/x/sys v0.30.0 // indirect
)
`)
		writeFixture(t, tmp, "NOTICE.md", "# Third-Party Notices\n### github.com/nyaosorg/go-readline-ny\n### golang.org/x/sys\n")
		writeFixture(t, tmp, "tools/reference-builder/package.json", `{"dependencies": {"katex": "^0.19.0", "unified": "^11.0.5"}}`)
		writeFixture(t, tmp, "tools/reference-builder/NOTICE.md", "### katex\n### unified\n")

		c := NewNoticeSyncChecker()
		diags, err := c.Run(&LintContext{RepoRoot: tmp, Mode: ModeAll})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(diags) != 0 {
			t.Errorf("expected 0 diagnostics, got %+v", diags)
		}
	})

	t.Run("detects missing go and npm notices", func(t *testing.T) {
		tmp := t.TempDir()
		writeFixture(t, tmp, "go.mod", `module example.com/test
go 1.26.5
require github.com/example/unlisted v1.0.0
`)
		writeFixture(t, tmp, "NOTICE.md", "# Third-Party Notices\n")
		writeFixture(t, tmp, "tools/reference-builder/package.json", `{"dependencies": {"unlisted-pkg": "^1.0.0"}}`)
		writeFixture(t, tmp, "tools/reference-builder/NOTICE.md", "# Third-Party Notices\n")

		c := NewNoticeSyncChecker()
		diags, err := c.Run(&LintContext{RepoRoot: tmp, Mode: ModeAll})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertAllHaveReasonAndRemediation(t, diags)

		if !hasRule(diags, "notice-go-missing") {
			t.Errorf("expected notice-go-missing, got %+v", diags)
		}
		if !hasRule(diags, "notice-npm-missing") {
			t.Errorf("expected notice-npm-missing, got %+v", diags)
		}
	})
}

func TestCheckerReadmeQuickStart(t *testing.T) {
	readmeJA := "# Title\n\n## インストール\ninstall line\n\n## クイックスタート\nline1\nline2\n\n## 機能\nfeat\n"
	readmeEN := "# Title\n\n## Installation\ninstall line\n\n## Quick Start\nline1\nline2\n\n## Features\nfeat\n"

	t.Run("blocks staged edit inside Quick Start section", func(t *testing.T) {
		tmp := t.TempDir()
		writeFixture(t, tmp, "README.md", readmeJA)
		writeFixture(t, tmp, "README.en.md", readmeEN)

		git := &fakeGitClient{
			staged: []string{"README.md", "README.en.md"},
			diffs: map[string]string{
				"README.md":    "@@ -7,1 +7,1 @@\n-line1\n+changed\n",
				"README.en.md": "@@ -7,1 +7,1 @@\n-line1\n+changed\n",
			},
		}

		c := NewReadmeQuickStartChecker()
		diags, err := c.Run(&LintContext{RepoRoot: tmp, Mode: ModeStaged, Git: git, StagedFiles: git.staged})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertAllHaveReasonAndRemediation(t, diags)
		if len(diags) != 2 || !hasRule(diags, "readme-quickstart-modified") {
			t.Errorf("expected 2 readme-quickstart-modified errors, got %+v", diags)
		}
	})

	t.Run("allows edit outside Quick Start and respects IHD_ALLOW_QUICKSTART_EDIT=1", func(t *testing.T) {
		tmp := t.TempDir()
		writeFixture(t, tmp, "README.md", readmeJA)

		gitOutside := &fakeGitClient{
			staged: []string{"README.md"},
			diffs:  map[string]string{"README.md": "@@ -4,1 +4,1 @@\n-install line\n+updated\n"},
		}
		c := NewReadmeQuickStartChecker()
		diags, err := c.Run(&LintContext{RepoRoot: tmp, Mode: ModeStaged, Git: gitOutside, StagedFiles: gitOutside.staged})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(diags) != 0 {
			t.Errorf("expected 0 diagnostics for edit outside Quick Start, got %+v", diags)
		}

		gitInside := &fakeGitClient{
			staged: []string{"README.md"},
			diffs:  map[string]string{"README.md": "@@ -7,1 +7,1 @@\n-line1\n+changed\n"},
		}
		diagsOverride, err := c.Run(&LintContext{
			RepoRoot:    tmp,
			Mode:        ModeStaged,
			Git:         gitInside,
			StagedFiles: gitInside.staged,
			EnvLookup: func(k string) string {
				if k == "IHD_ALLOW_QUICKSTART_EDIT" {
					return "1"
				}
				return ""
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(diagsOverride) != 0 {
			t.Errorf("expected 0 diagnostics with IHD_ALLOW_QUICKSTART_EDIT=1, got %+v", diagsOverride)
		}
	})
}

func TestCheckerLeanPinning(t *testing.T) {
	t.Run("passes pinned toolchain, 40-char rev, and comment mentioning lake update", func(t *testing.T) {
		tmp := t.TempDir()
		writeFixture(t, tmp, "formal/lean-toolchain", "leanprover/lean4:v4.36.0-rc2\n")
		writeFixture(t, tmp, "formal/lake-manifest.json", `{
  "packages": [
    {"name": "mathlib", "rev": "7db326f5280be7ca10231aef6711b09783b5445f"}
  ]
}`)
		writeFixture(t, tmp, ".github/workflows/lean-verify.yml", `jobs:
  verify:
    steps:
      - run: |
          # Pinned manifest used: lake update is skipped
          lake exe cache get
`)
		c := NewLeanPinningChecker()
		diags, err := c.Run(&LintContext{RepoRoot: tmp, Mode: ModeAll})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(diags) != 0 {
			t.Errorf("expected 0 diagnostics, got %+v", diags)
		}
	})

	t.Run("detects unpinned toolchain, non-sha rev, and active lake update in CI", func(t *testing.T) {
		tmp := t.TempDir()
		writeFixture(t, tmp, "formal/lean-toolchain", "leanprover/lean4:nightly\n")
		writeFixture(t, tmp, "formal/lake-manifest.json", `{
  "packages": [
    {"name": "mathlib", "rev": "master"}
  ]
}`)
		writeFixture(t, tmp, ".github/workflows/lean-verify.yml", `jobs:
  verify:
    steps:
      - run: lake update mathlib
`)
		c := NewLeanPinningChecker()
		diags, err := c.Run(&LintContext{RepoRoot: tmp, Mode: ModeAll})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertAllHaveReasonAndRemediation(t, diags)

		for _, rule := range []string{"lean-toolchain-unpinned", "lean-manifest-unpinned", "lean-ci-lake-update"} {
			if !hasRule(diags, rule) {
				t.Errorf("expected rule %q to trigger, got %+v", rule, diags)
			}
		}
	})
}

func TestCheckerGoHygiene(t *testing.T) {
	tmp := t.TempDir()
	writeFixture(t, tmp, "internal/i18n/locales/en.json", `{"ok": "sqrt(2) = √2 ± 0", "bad": "done \ud83d\ude80"}`)
	writeFixture(t, tmp, "internal/i18n/locales/ja.json", `{"ok": "ok", "bad": "完了"}`)
	writeFixture(t, tmp, "internal/calc/hygiene.go", `package calc

import "fmt"

func bad(dir string, err error) {
	if err != nil {
	}
	_ = err
	_ = err // repo-lint:ignore go-hygiene-blank-err intentionally discarded in test fixture
	p := dir + "/config.json"
	url := "https://" + dir
	frac := "1/2"
	fmt.Println("status: \U0001F525", p, url, frac, "√2 ± 1 → ⣿")
}
`)

	r := NewRunner()
	RegisterGoHygieneAnalyzers(r)

	res, err := r.Run(RunOptions{RepoRoot: tmp, Mode: ModeAll, Git: &fakeGitClient{}})
	if err != nil {
		t.Fatalf("Runner.Run failed: %v", err)
	}
	assertAllHaveReasonAndRemediation(t, res.Diagnostics)

	if !hasRule(res.Diagnostics, "go-hygiene-empty-err-branch") {
		t.Errorf("expected go-hygiene-empty-err-branch, got %+v", res.Diagnostics)
	}
	if !hasRule(res.Diagnostics, "go-hygiene-emoji") {
		t.Errorf("expected go-hygiene-emoji, got %+v", res.Diagnostics)
	}
	if !hasRule(res.Diagnostics, "go-hygiene-blank-err") {
		t.Errorf("expected go-hygiene-blank-err warning, got %+v", res.Diagnostics)
	}
	if !hasRule(res.Diagnostics, "go-hygiene-os-path-concat") {
		t.Errorf("expected go-hygiene-os-path-concat warning, got %+v", res.Diagnostics)
	}

	// Verify severities: empty-err & emoji are errors; blank-err & os-path-concat are warnings; line 9 suppressed.
	blankErrCount := 0
	pathConcatCount := 0
	for _, d := range res.Diagnostics {
		switch d.RuleID {
		case "go-hygiene-empty-err-branch", "go-hygiene-emoji":
			if d.Severity != SeverityError {
				t.Errorf("expected %s to be error, got %s", d.RuleID, d.Severity)
			}
		case "go-hygiene-blank-err":
			blankErrCount++
			if d.Severity != SeverityWarning {
				t.Errorf("expected go-hygiene-blank-err to be warning, got %s", d.Severity)
			}
		case "go-hygiene-os-path-concat":
			pathConcatCount++
			if d.Severity != SeverityWarning {
				t.Errorf("expected go-hygiene-os-path-concat to be warning, got %s", d.Severity)
			}
		}
	}
	if blankErrCount != 1 {
		t.Errorf("expected exactly 1 unsuppressed go-hygiene-blank-err (line 9 suppressed), got %d", blankErrCount)
	}
	if pathConcatCount != 1 {
		t.Errorf("expected exactly 1 go-hygiene-os-path-concat (url/frac ignored), got %d", pathConcatCount)
	}
}

func TestCheckerE2ECSV(t *testing.T) {
	header := "id,category,flags,pipe_input,match_mode,expected_stdout,expected_stderr,expected_exit_code,deduction_method,description\n"

	t.Run("clean E2E CSV suite passes", func(t *testing.T) {
		tmp := t.TempDir()
		writeFixture(t, tmp, "tools/e2e/test-cases/01/a.csv", header+`TC-001,arith,"1+2",,exact,3,,0,deductive,basic addition`+"\n")
		writeFixture(t, tmp, "tools/e2e/test-cases/02/b.csv", header+`TC-002,arith,,"1/2+1/3",exact,5/6,,0,roundtrip,fraction sum`+"\n")

		c := NewE2ECSVChecker()
		diags, err := c.Run(&LintContext{RepoRoot: tmp, Mode: ModeAll})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(diags) != 0 {
			t.Errorf("expected 0 diagnostics, got %+v", diags)
		}
	})

	t.Run("detects schema, column count, required fields, enum, and cross-file duplicate IDs", func(t *testing.T) {
		tmp := t.TempDir()
		writeFixture(t, tmp, "tools/e2e/test-cases/01/a.csv", header+
			`TC-DUP,arith,"1+2",,exact,3,,0,deductive,valid row`+"\n"+
			`,arith,"1+2",,invalid_mode,3,,0,bad_method,`+"\n"+
			`TC-SHORT,arith,only_three_cols`+"\n")
		writeFixture(t, tmp, "tools/e2e/test-cases/02/b.csv", header+
			`TC-DUP,arith,,,exact,3,,0,deductive,duplicate id and empty input`+"\n")
		writeFixture(t, tmp, "tools/e2e/test-cases/03/bad_header.csv", "id,category\nTC-H,arith\n")

		c := NewE2ECSVChecker()
		diags, err := c.Run(&LintContext{RepoRoot: tmp, Mode: ModeAll})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertAllHaveReasonAndRemediation(t, diags)

		for _, rule := range []string{
			"e2e-csv-header",
			"e2e-csv-columns",
			"e2e-csv-required-fields",
			"e2e-csv-enum",
			"e2e-csv-duplicate-id",
		} {
			if !hasRule(diags, rule) {
				t.Errorf("expected rule %q to trigger, got %+v", rule, diags)
			}
		}
	})
}
