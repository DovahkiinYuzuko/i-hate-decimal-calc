package calc

import (
	"testing"
)

func TestSolveCubicExact_Rational(t *testing.T) {
	// (x - 1)(x - 2)(x - 3) = x^3 - 6*x^2 + 11*x - 6 = 0
	roots, err := SolveCubicExact(mustRational(1, 1), mustRational(-6, 1), mustRational(11, 1), mustRational(-6, 1))
	if err != nil {
		t.Fatalf("SolveCubicExact failed: %v", err)
	}
	if len(roots) != 3 {
		t.Fatalf("expected 3 roots, got %d", len(roots))
	}

	expected := map[string]bool{"1": true, "2": true, "3": true}
	for _, r := range roots {
		s := Format(r)
		if !expected[s] {
			t.Errorf("unexpected root: %s", s)
		}
	}
}

func TestSolveCubicExact_RationalAndIrrationalQuadratic(t *testing.T) {
	// (x - 1)(x^2 - 2) = x^3 - x^2 - 2*x + 2 = 0
	roots, err := SolveCubicExact(mustRational(1, 1), mustRational(-1, 1), mustRational(-2, 1), mustRational(2, 1))
	if err != nil {
		t.Fatalf("SolveCubicExact failed: %v", err)
	}
	if len(roots) != 3 {
		t.Fatalf("expected 3 roots, got %d", len(roots))
	}

	expected := map[string]bool{"1": true, "√2": true, "-√2": true, "-1 * √2": true}
	foundRational := false
	for _, r := range roots {
		s := Format(r)
		if s == "1" {
			foundRational = true
		}
		if !expected[s] {
			t.Errorf("unexpected root: %s", s)
		}
	}
	if !foundRational {
		t.Errorf("expected root 1 not found in %v", roots)
	}
}

func TestSolveCubicExact_RepeatedRoots(t *testing.T) {
	// x^3 = 0
	roots, err := SolveCubicExact(mustRational(1, 1), mustRational(0, 1), mustRational(0, 1), mustRational(0, 1))
	if err != nil {
		t.Fatalf("SolveCubicExact failed: %v", err)
	}
	if len(roots) != 3 {
		t.Fatalf("expected 3 roots, got %d", len(roots))
	}
	for _, r := range roots {
		if Format(r) != "0" {
			t.Errorf("expected root 0, got %s", Format(r))
		}
	}
}

func TestSolveCubicExact_DepressedMonomial(t *testing.T) {
	// x^3 - 2 = 0
	roots, err := SolveCubicExact(mustRational(1, 1), mustRational(0, 1), mustRational(0, 1), mustRational(-2, 1))
	if err != nil {
		t.Fatalf("SolveCubicExact failed: %v", err)
	}
	if len(roots) != 3 {
		t.Fatalf("expected 3 roots, got %d", len(roots))
	}
	// Root 1 must be 2^1/3
	r1Str := Format(roots[0])
	if r1Str != "2^1/3" && r1Str != "2^(1/3)" {
		t.Errorf("expected real root 2^1/3, got %s", r1Str)
	}
}

func TestSolveCubicExact_CasusIrreducibilis(t *testing.T) {
	// x^3 - 3*x - 1 = 0
	roots, err := SolveCubicExact(mustRational(1, 1), mustRational(0, 1), mustRational(-3, 1), mustRational(-1, 1))
	if err != nil {
		t.Fatalf("SolveCubicExact failed: %v", err)
	}
	if len(roots) != 3 {
		t.Fatalf("expected 3 roots, got %d", len(roots))
	}

	// Verify that all 3 roots are non-nil and represent exact radical forms
	for i, r := range roots {
		if r == nil {
			t.Fatalf("root %d is nil", i)
		}
		s := Format(r)
		if len(s) == 0 {
			t.Errorf("root %d formatted to empty string", i)
		}
	}
}

func TestSolveCubicExact_QuadraticReduction(t *testing.T) {
	// 0*x^3 + x^2 - 4 = 0 => x = 2, -2
	roots, err := SolveCubicExact(mustRational(0, 1), mustRational(1, 1), mustRational(0, 1), mustRational(-4, 1))
	if err != nil {
		t.Fatalf("SolveCubicExact failed: %v", err)
	}
	if len(roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(roots))
	}
	expected := map[string]bool{"2": true, "-2": true}
	for _, r := range roots {
		s := Format(r)
		if !expected[s] {
			t.Errorf("unexpected root: %s", s)
		}
	}
}
