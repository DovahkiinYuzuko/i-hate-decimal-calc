package calc

import (
	"math/big"
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

func TestVerifyEquivalencePIT_Multivariate_Inconclusive(t *testing.T) {
	// x * y vs x (False identity, but contains 2 variables!)
	// PIT must NOT falsely certify this by setting y = 1.
	lhs := NewMul([]Node{NewVar("x"), NewVar("y")})
	rhs := NewVar("x")

	cert, _, err := VerifyEquivalencePIT(lhs, rhs, "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cert != VerifyCertaintyInconclusive {
		t.Errorf("expected Inconclusive for multivariate PIT, got %s", cert)
	}
}

func TestVerifyEquivalencePIT_RationalFunction_Inconclusive(t *testing.T) {
	// 1 / (x^2 + 2) vs -1/6 * x^2 + 1/2
	// Has negative exponent, so not pure polynomial. Must fail safe to Inconclusive.
	x2, _ := NewPow(NewVar("x"), mustRational(2, 1))
	denom := NewAdd([]Node{x2, mustRational(2, 1)})
	lhs, _ := NewPow(denom, mustRational(-1, 1))

	negSixth := mustRational(-1, 6)
	term1 := NewMul([]Node{negSixth, x2})
	rhs := NewAdd([]Node{term1, mustRational(1, 2)})

	cert, _, err := VerifyEquivalencePIT(lhs, rhs, "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cert != VerifyCertaintyInconclusive {
		t.Errorf("expected Inconclusive for non-polynomial rational function, got %s", cert)
	}
}

func TestRatPowBigInt_BinaryExponentiation(t *testing.T) {
	// Test 2^10 = 1024
	base := big.NewRat(2, 1)
	res, err := ratPowBigInt(base, big.NewInt(10))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Cmp(big.NewRat(1024, 1)) != 0 {
		t.Errorf("expected 1024, got %s", res.String())
	}

	// Test (1/2)^(-5) = 32
	half := big.NewRat(1, 2)
	resNeg, err := ratPowBigInt(half, big.NewInt(-5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resNeg.Cmp(big.NewRat(32, 1)) != 0 {
		t.Errorf("expected 32, got %s", resNeg.String())
	}
}
