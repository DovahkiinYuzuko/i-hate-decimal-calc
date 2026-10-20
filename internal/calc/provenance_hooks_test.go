package calc

import (
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/provenance"
)

func TestProvenanceHooks_AlgebraExpand(t *testing.T) {
	node, err := Parse("expand((x + 1)^2)")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	env := NewEnv()
	res, root, err := EvalWithProvenanceAndEnv(node, env)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	if res == nil {
		t.Fatalf("expected non-nil result")
	}
	if root == nil {
		t.Fatalf("expected non-nil provenance root")
	}
	if root.RuleID != provenance.RuleAlgebraExpand {
		t.Errorf("expected root rule %s, got %s", provenance.RuleAlgebraExpand, root.RuleID)
	}
}

func TestProvenanceHooks_AlgebraFactor(t *testing.T) {
	node, err := Parse("factor(x^2 - 1)")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	env := NewEnv()
	res, root, err := EvalWithProvenanceAndEnv(node, env)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	if res == nil {
		t.Fatalf("expected non-nil result")
	}
	if root == nil {
		t.Fatalf("expected non-nil provenance root")
	}
	if root.RuleID != provenance.RuleAlgebraFactor {
		t.Errorf("expected root rule %s, got %s", provenance.RuleAlgebraFactor, root.RuleID)
	}
}

func TestProvenanceHooks_AlgebraApartTogether(t *testing.T) {
	// 1. apart
	apartNode, err := Parse("apart(1 / (x * (x + 1)))")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	env := NewEnv()
	_, apartRoot, err := EvalWithProvenanceAndEnv(apartNode, env)
	if err != nil {
		t.Fatalf("apart eval error: %v", err)
	}
	if apartRoot == nil || apartRoot.RuleID != provenance.RuleAlgebraApart {
		t.Errorf("expected apart root %s, got %+v", provenance.RuleAlgebraApart, apartRoot)
	}

	// 2. together
	togetherNode, err := Parse("together(1/x - 1/(x + 1))")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	_, togetherRoot, err := EvalWithProvenanceAndEnv(togetherNode, env)
	if err != nil {
		t.Fatalf("together eval error: %v", err)
	}
	if togetherRoot == nil || togetherRoot.RuleID != provenance.RuleAlgebraTogether {
		t.Errorf("expected together root %s, got %+v", provenance.RuleAlgebraTogether, togetherRoot)
	}
}

func TestProvenanceHooks_CalculusDiffAndIntegrate(t *testing.T) {
	// 1. diff
	diffNode, err := Parse("diff(x^3, x)")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	env := NewEnv()
	_, diffRoot, err := EvalWithProvenanceAndEnv(diffNode, env)
	if err != nil {
		t.Fatalf("diff eval error: %v", err)
	}
	if diffRoot == nil || diffRoot.RuleID != "calculus.diff" {
		t.Errorf("expected diff root calculus.diff, got %+v", diffRoot)
	}

	// 2. integrate (indefinite with AntiderivativeCertificate)
	intNode, err := Parse("integrate(x^2, x)")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	_, intRoot, err := EvalWithProvenanceAndEnv(intNode, env)
	if err != nil {
		t.Fatalf("integrate eval error: %v", err)
	}
	if intRoot == nil || intRoot.RuleID != "calculus.integrate" {
		t.Errorf("expected integrate root calculus.integrate, got %+v", intRoot)
	}
	if intRoot.Certificate == nil || intRoot.Certificate.CertificateKind() != "Antiderivative" {
		t.Errorf("expected AntiderivativeCertificate, got %+v", intRoot.Certificate)
	}
}

func TestProvenanceHooks_Solve(t *testing.T) {
	solveNode, err := Parse("solve(x^2 - 4 == 0, x)")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	env := NewEnv()
	_, solveRoot, err := EvalWithProvenanceAndEnv(solveNode, env)
	if err != nil {
		t.Fatalf("solve eval error: %v", err)
	}
	if solveRoot == nil || solveRoot.RuleID != "solver.solve" {
		t.Errorf("expected solve root solver.solve, got %+v", solveRoot)
	}
}
