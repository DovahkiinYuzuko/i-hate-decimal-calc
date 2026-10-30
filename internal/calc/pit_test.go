package calc

import (
	"testing"
)

func TestVerifyEquivalencePIT_Positive(t *testing.T) {
	// (x - 1) * (x + 1) vs x^2 - 1
	xMinus1 := NewAdd([]Node{NewVar("x"), mustRational(-1, 1)})
	xPlus1 := NewAdd([]Node{NewVar("x"), mustRational(1, 1)})
	lhs := NewMul([]Node{xMinus1, xPlus1})

	x2, _ := NewPow(NewVar("x"), mustRational(2, 1))
	rhs := NewAdd([]Node{x2, mustRational(-1, 1)})

	cert, wit, err := VerifyEquivalencePIT(lhs, rhs, "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wit != nil {
		t.Errorf("expected no witness for true identity, got %v", wit)
	}
	if cert != VerifyCertaintyExact && cert != VerifyCertaintyProbabilistic {
		t.Errorf("expected exact or probabilistic pass, got %s", cert)
	}
}

func TestVerifyEquivalencePIT_Negative(t *testing.T) {
	// (x - 1)^2 vs x^2 - 1  (False identity!)
	xMinus1 := NewAdd([]Node{NewVar("x"), mustRational(-1, 1)})
	lhs, _ := NewPow(xMinus1, mustRational(2, 1))

	x2, _ := NewPow(NewVar("x"), mustRational(2, 1))
	rhs := NewAdd([]Node{x2, mustRational(-1, 1)})

	cert, wit, err := VerifyEquivalencePIT(lhs, rhs, "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cert != VerifyCertaintyRefuted {
		t.Errorf("expected REFUTED, got %s", cert)
	}
	if wit == nil {
		t.Fatalf("expected non-nil witness for refuted identity")
	}
	if wit.Kind != WitnessValueMismatch {
		t.Errorf("expected ValueMismatch, got %v", wit.Kind)
	}
}

func TestVerifyEquivalencePIT_Cubic(t *testing.T) {
	// (x - 1) * (x^2 + x + 1) vs x^3 - 1
	xMinus1 := NewAdd([]Node{NewVar("x"), mustRational(-1, 1)})
	x2, _ := NewPow(NewVar("x"), mustRational(2, 1))
	quad := NewAdd([]Node{x2, NewVar("x"), mustRational(1, 1)})
	lhs := NewMul([]Node{xMinus1, quad})

	x3, _ := NewPow(NewVar("x"), mustRational(3, 1))
	rhs := NewAdd([]Node{x3, mustRational(-1, 1)})

	cert, wit, err := VerifyEquivalencePIT(lhs, rhs, "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wit != nil {
		t.Errorf("expected no witness for cubic identity, got %v", wit)
	}
	if cert != VerifyCertaintyExact {
		t.Errorf("expected VerifyCertaintyExact, got %s", cert)
	}
}
