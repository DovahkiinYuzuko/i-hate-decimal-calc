package calc_test

import (
	"strings"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
)

// TestVerifierGovernance_CircularVerificationEliminated ensures that
// functions without independent asymmetric verifiers (e.g. qe, cad) return UNVERIFIED
// instead of circularly certifying via self-re-evaluation.
func TestVerifierGovernance_CircularVerificationEliminated(t *testing.T) {
	env := calc.NewEnv()

	// 1. qe computation: should be UNVERIFIED, NOT VERIFIED
	qeExpr, err := calc.Parse("qe(forall([x], exists([y], y^3 + x*y + a == 0)))")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	qeResult, err := calc.EvalWithEnv(qeExpr, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	cert, err := calc.VerifyComputation(qeExpr, qeResult, env)
	if err != nil {
		t.Fatalf("VerifyComputation returned unexpected error: %v", err)
	}
	if cert.IsVerified {
		t.Fatalf("CIRCULAR VERIFICATION DETECTED: qe must not be certified, but got: %s", cert.String())
	}
	if cert.State != calc.VerifyStateUnsupportedDomain {
		t.Fatalf("expected state VerifyStateUnsupportedDomain, got: %v", cert.State)
	}
	if !strings.Contains(cert.String(), "[UNVERIFIED:") {
		t.Fatalf("expected '[UNVERIFIED:' in certificate output, got: %s", cert.String())
	}

	// 2. factor computation: HAS independent asymmetric verifier (expansion), MUST be VERIFIED
	factorExpr, err := calc.Parse("factor(x^2 - 1)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	factorResult, err := calc.EvalWithEnv(factorExpr, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	factorCert, err := calc.VerifyComputation(factorExpr, factorResult, env)
	if err != nil {
		t.Fatalf("VerifyComputation factor error: %v", err)
	}
	if !factorCert.IsVerified {
		t.Fatalf("expected factor to be verified, got: %s", factorCert.String())
	}
	if !strings.Contains(factorCert.String(), "[VERIFIED:") {
		t.Fatalf("expected '[VERIFIED:' in factor cert, got: %s", factorCert.String())
	}
}

// TestVerifierGovernance_AbelRuffiniCertificate ensures that
// is_solvable_by_radicals emits an Abel-Ruffini Impossibility Certificate for non-solvable quintics,
// and correctly transpiles to formal Lean 4 without dummy variables like (false x : ℚ).
func TestVerifierGovernance_AbelRuffiniCertificate(t *testing.T) {
	env := calc.NewEnv()

	// is_solvable_by_radicals(x^5 - 4*x + 2) is false (S5, non-solvable)
	expr, err := calc.Parse("is_solvable_by_radicals(x^5 - 4*x + 2)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	result, err := calc.EvalWithEnv(expr, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	if result.String() != "false" {
		t.Fatalf("expected false, got: %s", result.String())
	}

	cert, err := calc.VerifyComputation(expr, result, env)
	if err != nil {
		t.Fatalf("VerifyComputation error: %v", err)
	}
	if !cert.IsVerified {
		t.Fatalf("expected Abel-Ruffini certificate to be verified, got: %s", cert.String())
	}
	if cert.Domain != calc.DomainImpossibility {
		t.Fatalf("expected DomainImpossibility, got: %s", cert.Domain)
	}

	// Transpile to Lean 4
	leanCode, err := calc.GenerateLeanSource("quintic_insolvability", cert, expr, result)
	if err != nil {
		t.Fatalf("GenerateLeanSource failed: %v", err)
	}
	// Must NOT contain dummy variable declarations like variable (false ... : ℚ)
	if strings.Contains(leanCode, "false") && strings.Contains(leanCode, "variable") && strings.Contains(leanCode, "false x : ℚ") {
		t.Fatalf("corrupted variable declaration in Lean source: %s", leanCode)
	}
	if !strings.Contains(leanCode, "IsSolvable") {
		t.Fatalf("expected 'IsSolvable' in Abel-Ruffini Lean theorem, got: %s", leanCode)
	}
}

// TestVerifierGovernance_LeanSafeguards ensures that functions without Lean proof generation
// (e.g. galois_group) return explicit unsupported errors instead of corrupted "by ring" proofs.
func TestVerifierGovernance_LeanSafeguards(t *testing.T) {
	env := calc.NewEnv()

	// 1. galois_group: should NOT generate broken "variable (A3 x : ℚ) ... by ring"
	expr, err := calc.Parse("galois_group(x^3 - 3*x + 1)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	res, err := calc.EvalWithEnv(expr, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	cert, err := calc.VerifyComputation(expr, res, env)
	if err != nil {
		t.Fatalf("VerifyComputation error: %v", err)
	}
	// Verification must be unverified since galois_group has no independent certificate verifier
	if cert.IsVerified {
		t.Fatalf("galois_group must not be verified without independent certificate, got: %s", cert.String())
	}

	// Lean generation must fail gracefully with explicit error
	_, err = calc.GenerateLeanSource("galois_proof", cert, expr, res)
	if err == nil {
		t.Fatalf("expected error when transpiling unverified / unsupported function to Lean, got nil")
	}

	// 2. CollectFreeVariables must not include bool or reserved keywords as variables
	freeVars := calc.CollectFreeVariables(
		&calc.VarNode{Name: "false"},
		&calc.VarNode{Name: "true"},
		&calc.VarNode{Name: "x"},
	)
	for _, v := range freeVars {
		if v == "false" || v == "true" {
			t.Fatalf("CollectFreeVariables erroneously included boolean '%s'", v)
		}
	}
}
