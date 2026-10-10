package calc_test

import (
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
)

// TestCAD_QE_CubicRealRoots ensures that CAD and Quantifier Elimination correctly
// handle cubic polynomials with 3 real roots (casus irreducibilis) without erroneously
// dropping real roots via flawed string-based "i" detection.
func TestCAD_QE_CubicRealRoots(t *testing.T) {
	env := calc.NewEnv()

	// 1. y^3 + x*y + a == 0 has at least one real root y for all x, a (odd degree).
	// Therefore forall x, exists y, y^3 + x*y + a == 0 is identically TRUE.
	expr1, err := calc.Parse("qe(forall([x], exists([y], y^3 + x*y + a == 0)))")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	res1, err := calc.EvalWithEnv(expr1, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	if res1.String() != "true" {
		t.Fatalf("expected 'true' for forall x, exists y, y^3 + x*y + a == 0, but got: %s", res1.String())
	}

	// 2. y^3 - 3*y - x == 0 also identically has real root y for all x
	expr2, err := calc.Parse("qe(forall([x], exists([y], y^3 - 3*y - x == 0)))")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	res2, err := calc.EvalWithEnv(expr2, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	if res2.String() != "true" {
		t.Fatalf("expected 'true' for forall x, exists y, y^3 - 3*y - x == 0, but got: %s", res2.String())
	}
}
