package calc

import (
	"strings"
	"testing"
)

func TestLinearSolveVerificationAndLean(t *testing.T) {
	expr, err := Parse("solve_linear([[1, 2], [3, 4]], [5, 11])")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	res, err := Eval(expr)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	cert, err := VerifyComputation(expr, res, nil)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !cert.IsVerified {
		t.Fatalf("expected certified, got %v", cert)
	}
	if cert.Domain != DomainLinearSolve {
		t.Errorf("expected domain linear_solve, got %s", cert.Domain)
	}

	leanCode, err := GenerateLeanSource("ihd_solve_linear_test", cert, expr, res)
	if err != nil {
		t.Fatalf("generate lean source failed: %v", err)
	}
	if !strings.Contains(leanCode, "theorem ihd_solve_linear_test") {
		t.Errorf("lean code missing theorem name: %s", leanCode)
	}
	if !strings.Contains(leanCode, "Matrix") {
		t.Errorf("lean code missing Matrix: %s", leanCode)
	}
}

func TestPellVerificationAndLean(t *testing.T) {
	expr, err := Parse("solve_pell(2)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	res, err := Eval(expr)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	// [3, 2] satisfies 3^2 - 2*2^2 = 9 - 8 = 1
	formatted := Format(res)
	if formatted != "[3, 2]" {
		t.Errorf("expected [3, 2], got %s", formatted)
	}

	cert, err := VerifyComputation(expr, res, nil)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !cert.IsVerified {
		t.Fatalf("expected certified, got %v", cert)
	}
	if cert.Domain != DomainPell {
		t.Errorf("expected domain pell, got %s", cert.Domain)
	}

	leanCode, err := GenerateLeanSource("ihd_pell_test", cert, expr, res)
	if err != nil {
		t.Fatalf("generate lean source failed: %v", err)
	}
	if !strings.Contains(leanCode, "theorem ihd_pell_test") {
		t.Errorf("lean code missing theorem name: %s", leanCode)
	}
	if !strings.Contains(leanCode, "by decide") {
		t.Errorf("lean code missing by decide: %s", leanCode)
	}
}

func TestPolynomialRootVerificationAndLean(t *testing.T) {
	expr, err := Parse("solve(x^3 - 6*x^2 + 11*x - 6, x)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	res, err := Eval(expr)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	cert, err := VerifyComputation(expr, res, nil)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !cert.IsVerified {
		t.Fatalf("expected certified, got %v", cert)
	}

	leanCode, err := GenerateLeanSource("ihd_poly_root_test", cert, expr, res)
	if err != nil {
		t.Fatalf("generate lean source failed: %v", err)
	}
	if !strings.Contains(leanCode, "theorem ihd_poly_root_test") {
		t.Errorf("lean code missing theorem name: %s", leanCode)
	}
}

func TestEllipticAddVerificationAndLean(t *testing.T) {
	// Curve: y^2 = x^3 - x (a = -1, b = 0)
	// P1 = [-1, 0], P2 = [0, 0]
	expr, err := Parse("ec_add(-1, 0, [-1, 0], [0, 0])")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	res, err := Eval(expr)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	cert, err := VerifyComputation(expr, res, nil)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !cert.IsVerified {
		t.Fatalf("expected certified, got %v", cert)
	}
	if cert.Domain != DomainElliptic {
		t.Errorf("expected domain elliptic, got %s", cert.Domain)
	}

	leanCode, err := GenerateLeanSource("ihd_elliptic_test", cert, expr, res)
	if err != nil {
		t.Fatalf("generate lean source failed: %v", err)
	}
	if !strings.Contains(leanCode, "theorem ihd_elliptic_test") {
		t.Errorf("lean code missing theorem name: %s", leanCode)
	}
}

func TestExplainTraceCAS(t *testing.T) {
	// Test explain trace for Pell
	sink := NewMemoryTraceSink()
	WithTraceSink(sink, func() {
		_, err := EvalString("solve_pell(3)")
		if err != nil {
			t.Fatalf("eval solve_pell failed: %v", err)
		}
	})

	events := sink.Events()
	foundPell := false
	for _, e := range events {
		if e.Rule == RuleSolvePell {
			foundPell = true
			break
		}
	}
	if !foundPell {
		t.Errorf("expected trace event for RuleSolvePell, but not found among %d events", len(events))
	}
}
