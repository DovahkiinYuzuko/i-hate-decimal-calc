package domain

import (
	"math/big"
	"testing"
)

func TestRationalDomain(t *testing.T) {
	q := NewRationalDomain()

	if q.Name() != "Q" {
		t.Errorf("expected name Q, got %s", q.Name())
	}

	zero := q.Zero()
	one := q.One()

	if !q.IsZero(zero) {
		t.Errorf("expected IsZero(0) to be true")
	}
	if !q.IsOne(one) {
		t.Errorf("expected IsOne(1) to be true")
	}

	a := big.NewRat(2, 3)
	b := big.NewRat(3, 4)

	// Add: 2/3 + 3/4 = 17/12
	addRes := q.Add(a, b)
	if addRes.Cmp(big.NewRat(17, 12)) != 0 {
		t.Errorf("expected 17/12, got %s", addRes.RatString())
	}

	// Sub: 2/3 - 3/4 = -1/12
	subRes := q.Sub(a, b)
	if subRes.Cmp(big.NewRat(-1, 12)) != 0 {
		t.Errorf("expected -1/12, got %s", subRes.RatString())
	}

	// Mul: 2/3 * 3/4 = 1/2
	mulRes := q.Mul(a, b)
	if mulRes.Cmp(big.NewRat(1, 2)) != 0 {
		t.Errorf("expected 1/2, got %s", mulRes.RatString())
	}

	// Inv & Div: (2/3) / (3/4) = 8/9
	divRes, err := q.Div(a, b)
	if err != nil {
		t.Fatalf("unexpected div error: %v", err)
	}
	if divRes.Cmp(big.NewRat(8, 9)) != 0 {
		t.Errorf("expected 8/9, got %s", divRes.RatString())
	}

	// Zero division
	_, err = q.Div(a, zero)
	if err == nil {
		t.Errorf("expected zero division error, got nil")
	}
}

func TestFiniteFieldDomain(t *testing.T) {
	// F_17
	p := big.NewInt(17)
	f17, err := NewFiniteFieldDomain(p)
	if err != nil {
		t.Fatalf("failed to create F_17: %v", err)
	}

	if f17.Name() != "F_17" {
		t.Errorf("expected F_17, got %s", f17.Name())
	}

	// Non-prime check
	composite := big.NewInt(15)
	_, err = NewFiniteFieldDomain(composite)
	if err == nil {
		t.Errorf("expected error for non-prime modulus 15, got nil")
	}

	a := big.NewInt(10)
	b := big.NewInt(12)

	// Add: (10 + 12) mod 17 = 22 mod 17 = 5
	addRes := f17.Add(a, b)
	if addRes.Cmp(big.NewInt(5)) != 0 {
		t.Errorf("expected 5, got %s", addRes.String())
	}

	// Sub: (10 - 12) mod 17 = -2 mod 17 = 15
	subRes := f17.Sub(a, b)
	if subRes.Cmp(big.NewInt(15)) != 0 {
		t.Errorf("expected 15, got %s", subRes.String())
	}

	// Mul: (10 * 12) mod 17 = 120 mod 17 = 1
	mulRes := f17.Mul(a, b)
	if mulRes.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("expected 1, got %s", mulRes.String())
	}

	// Inv of 10 mod 17: since 10 * 12 = 120 = 17 * 7 + 1, 10^(-1) = 12
	invA, err := f17.Inv(a)
	if err != nil {
		t.Fatalf("failed to invert 10 mod 17: %v", err)
	}
	if invA.Cmp(big.NewInt(12)) != 0 {
		t.Errorf("expected 12, got %s", invA.String())
	}

	// Div: 5 / 10 mod 17 = 5 * 12 mod 17 = 60 mod 17 = 9
	// Check: 9 * 10 mod 17 = 90 mod 17 = 5
	divRes, err := f17.Div(big.NewInt(5), a)
	if err != nil {
		t.Fatalf("failed to divide 5 / 10: %v", err)
	}
	if divRes.Cmp(big.NewInt(9)) != 0 {
		t.Errorf("expected 9, got %s", divRes.String())
	}

	// Zero division
	_, err = f17.Div(a, f17.Zero())
	if err == nil {
		t.Errorf("expected zero division error, got nil")
	}
}

func TestAlgExtensionDomain(t *testing.T) {
	// Q(sqrt(2)) where minPoly = x^2 - 2, coeffs: [-2, 0, 1]
	minPoly := []*big.Rat{big.NewRat(-2, 1), big.NewRat(0, 1), big.NewRat(1, 1)}
	qSqrt2, err := NewAlgExtensionDomain("sqrt2", minPoly)
	if err != nil {
		t.Fatalf("failed to create Q(sqrt2): %v", err)
	}

	if qSqrt2.Name() != "Q(sqrt2)" {
		t.Errorf("expected Q(sqrt2), got %s", qSqrt2.Name())
	}

	// alpha = sqrt(2), coeffs: [0, 1]
	alpha := []*big.Rat{big.NewRat(0, 1), big.NewRat(1, 1)}

	// alpha^2 = 2, coeffs: [2]
	alphaSq := qSqrt2.Mul(alpha, alpha)
	if len(alphaSq) != 1 || alphaSq[0].Cmp(big.NewRat(2, 1)) != 0 {
		t.Errorf("expected alpha^2 == 2, got %s", qSqrt2.String(alphaSq))
	}

	// 1 / alpha = sqrt(2) / 2, coeffs: [0, 1/2]
	invAlpha, err := qSqrt2.Inv(alpha)
	if err != nil {
		t.Fatalf("failed to invert alpha: %v", err)
	}
	if len(invAlpha) != 2 || invAlpha[0].Sign() != 0 || invAlpha[1].Cmp(big.NewRat(1, 2)) != 0 {
		t.Errorf("expected 1/sqrt(2) = 1/2*sqrt(2), got %s", qSqrt2.String(invAlpha))
	}

	// 1 / (1 + sqrt(2)) = sqrt(2) - 1, coeffs: [-1, 1]
	onePlusAlpha := []*big.Rat{big.NewRat(1, 1), big.NewRat(1, 1)}
	invOnePlusAlpha, err := qSqrt2.Inv(onePlusAlpha)
	if err != nil {
		t.Fatalf("failed to invert 1 + sqrt(2): %v", err)
	}
	if len(invOnePlusAlpha) != 2 || invOnePlusAlpha[0].Cmp(big.NewRat(-1, 1)) != 0 || invOnePlusAlpha[1].Cmp(big.NewRat(1, 1)) != 0 {
		t.Errorf("expected 1/(1+sqrt(2)) = -1 + sqrt(2), got %s", qSqrt2.String(invOnePlusAlpha))
	}

	// Check multiplication: (1 + sqrt2) * (sqrt2 - 1) = 2 - 1 = 1
	prod := qSqrt2.Mul(onePlusAlpha, invOnePlusAlpha)
	if !qSqrt2.IsOne(prod) {
		t.Errorf("expected prod == 1, got %s", qSqrt2.String(prod))
	}
}
