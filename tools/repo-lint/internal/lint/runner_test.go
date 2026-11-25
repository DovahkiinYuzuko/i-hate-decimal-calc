package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type stubASTAnalyzer struct {
	name     string
	severity Severity
	fn       func(ctx *LintContext, af *ASTFile) ([]Diagnostic, error)
}

func (s *stubASTAnalyzer) Name() string              { return s.name }
func (s *stubASTAnalyzer) Description() string       { return "stub analyzer for testing" }
func (s *stubASTAnalyzer) DefaultSeverity() Severity { return s.severity }
func (s *stubASTAnalyzer) AnalyzeFile(ctx *LintContext, af *ASTFile) ([]Diagnostic, error) {
	return s.fn(ctx, af)
}

type stubChecker struct {
	name  string
	diags []Diagnostic
}

func (s *stubChecker) Name() string        { return s.name }
func (s *stubChecker) Description() string { return "stub checker" }
func (s *stubChecker) Run(ctx *LintContext) ([]Diagnostic, error) {
	return s.diags, nil
}

type fakeGitClient struct {
	tracked []string
	staged  []string
	diffs   map[string]string
}

func (f *fakeGitClient) TrackedFiles(root string) ([]string, error) { return f.tracked, nil }
func (f *fakeGitClient) StagedFiles(root string) ([]string, error)  { return f.staged, nil }
func (f *fakeGitClient) StagedDiffUnified0(root, relPath string) (string, error) {
	if f.diffs == nil {
		return "", nil
	}
	return f.diffs[relPath], nil
}

func writeFixture(t *testing.T, root, relPath, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", abs, err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
}

func TestRunnerFilteringSeverityAndFormatting(t *testing.T) {
	r := NewRunner()
	r.RegisterChecker(&stubChecker{
		name: "alpha",
		diags: []Diagnostic{
			{
				RuleID:      "alpha-warn",
				Checker:     "alpha",
				Severity:    SeverityWarning,
				File:        "a.go",
				Line:        10,
				Col:         2,
				Message:     "alpha warning message",
				Reason:      "Alpha rule was violated.",
				Remediation: "Fix alpha violation.",
			},
		},
	})
	r.RegisterChecker(&stubChecker{
		name: "beta",
		diags: []Diagnostic{
			{
				RuleID:      "beta-err",
				Checker:     "beta",
				Severity:    SeverityError,
				File:        "b.go",
				Line:        3,
				Col:         1,
				Message:     "beta error message",
				Reason:      "Beta rule was violated.",
				Remediation: "Fix beta violation.",
			},
		},
	})

	// 1. --only alpha -> only warning returned; exit code 0 unless warn-as-error.
	res, err := r.Run(RunOptions{
		RepoRoot: t.TempDir(),
		Mode:     ModeAll,
		Only:     []string{"alpha"},
		Git:      &fakeGitClient{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Diagnostics) != 1 || res.Diagnostics[0].RuleID != "alpha-warn" {
		t.Fatalf("expected only alpha-warn, got %+v", res.Diagnostics)
	}
	if res.ExitCode(false) != 0 {
		t.Errorf("expected ExitCode(false)=0 for warnings only, got %d", res.ExitCode(false))
	}
	if res.ExitCode(true) != 1 {
		t.Errorf("expected ExitCode(true)=1 for warnings with warn-as-error, got %d", res.ExitCode(true))
	}

	// 2. --skip alpha -> only beta-err returned.
	res2, err := r.Run(RunOptions{
		RepoRoot: t.TempDir(),
		Mode:     ModeAll,
		Skip:     []string{"alpha"},
		Git:      &fakeGitClient{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res2.Diagnostics) != 1 || res2.Diagnostics[0].RuleID != "beta-err" {
		t.Fatalf("expected only beta-err, got %+v", res2.Diagnostics)
	}

	// 3. Formatted text output must explicitly include rule, location, summary, reason, and remediation.
	formatted := FormatText(res2)
	for _, want := range []string{"[ERROR]", "beta-err", "b.go:3:1", "beta error message", "Reason", "Beta rule was violated.", "Remediation", "Fix beta violation."} {
		if !strings.Contains(formatted, want) {
			t.Errorf("formatted output missing %q:\n%s", want, formatted)
		}
	}
}
