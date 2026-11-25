package lint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestParseIgnoreDirectives(t *testing.T) {
	src := `package sample

func demo() {
	// repo-lint:ignore go-hygiene-blank-err intentional best-effort flush
	_ = flush()
	_ = closeIt() // repo-lint:ignore go-hygiene-blank-err close error logged elsewhere
	_ = badIgnore() // repo-lint:ignore go-hygiene-blank-err
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse fixture: %v", err)
	}

	ignores, invalid := ParseIgnoreDirectives("sample.go", fset, file, []byte(src))

	// Line 5 is suppressed by preceding line 4 comment.
	if !IsSuppressed(ignores, 5, "go-hygiene-blank-err") {
		t.Errorf("expected line 5 to be suppressed for go-hygiene-blank-err")
	}
	if IsSuppressed(ignores, 5, "other-rule") {
		t.Errorf("expected line 5 NOT to be suppressed for other-rule")
	}

	// Line 6 is suppressed by inline comment on line 6.
	if !IsSuppressed(ignores, 6, "go-hygiene-blank-err") {
		t.Errorf("expected line 6 to be suppressed for go-hygiene-blank-err")
	}

	// Line 7 has ignore without reason: must NOT suppress and must report an invalid directive diagnostic.
	if IsSuppressed(ignores, 7, "go-hygiene-blank-err") {
		t.Errorf("expected line 7 NOT to be suppressed when reason is missing")
	}
	if len(invalid) != 1 {
		t.Fatalf("expected 1 missing-reason diagnostic, got %d", len(invalid))
	}
	if invalid[0].RuleID != "repo-lint-ignore-missing-reason" {
		t.Errorf("expected rule repo-lint-ignore-missing-reason, got %q", invalid[0].RuleID)
	}
	if invalid[0].Line != 7 {
		t.Errorf("expected line 7, got %d", invalid[0].Line)
	}
	if invalid[0].Reason == "" || invalid[0].Remediation == "" {
		t.Errorf("expected non-empty Reason and Remediation in diagnostic: %+v", invalid[0])
	}
}

func TestASTAnalyzerExtensionSeam(t *testing.T) {
	// Verifies that an external ASTAnalyzer (e.g., future issue-183 CAS rules)
	// can plug into Runner and emit structured diagnostics with ignore support.
	tmp := t.TempDir()
	writeFixture(t, tmp, "internal/calc/sample.go", `package calc

func compute() {
	forbiddenCall()
	forbiddenCall() // repo-lint:ignore cas-mock-rule approved fixture exception
}
`)

	mockAnalyzer := &stubASTAnalyzer{
		name:     "cas-mock-rule",
		severity: SeverityError,
		fn: func(ctx *LintContext, af *ASTFile) ([]Diagnostic, error) {
			var diags []Diagnostic
			ast.Inspect(af.File, func(n ast.Node) bool {
				ident, ok := n.(*ast.Ident)
				if !ok || ident.Name != "forbiddenCall" {
					return true
				}
				pos := af.Fset.Position(ident.Pos())
				diags = append(diags, Diagnostic{
					RuleID:      "cas-mock-rule",
					Checker:     "ast-analyzers",
					Severity:    SeverityError,
					File:        af.RelPath,
					Line:        pos.Line,
					Col:         pos.Column,
					Message:     "forbiddenCall is disallowed in CAS core",
					Reason:      "Non-exact helper breaks algebraic determinism.",
					Remediation: "Replace with exact rational helper or add '// repo-lint:ignore cas-mock-rule <reason>'.",
				})
				return true
			})
			return diags, nil
		},
	}

	r := NewRunner()
	r.RegisterASTAnalyzer(mockAnalyzer)

	res, err := r.Run(RunOptions{
		RepoRoot: tmp,
		Mode:     ModeAll,
		Git:      &fakeGitClient{},
	})
	if err != nil {
		t.Fatalf("Runner.Run failed: %v", err)
	}

	if len(res.Diagnostics) != 1 {
		t.Fatalf("expected 1 unsuppressed diagnostic, got %d: %+v", len(res.Diagnostics), res.Diagnostics)
	}
	d := res.Diagnostics[0]
	if d.RuleID != "cas-mock-rule" || d.Line != 4 {
		t.Errorf("unexpected diagnostic: %+v", d)
	}
	if res.ExitCode(false) != 1 {
		t.Errorf("expected ExitCode(false)=1 on error diagnostic, got %d", res.ExitCode(false))
	}
}
