package padic

import (
	"math/big"
	"testing"
)

func TestValuationAndNorm(t *testing.T) {
	p5 := big.NewInt(5)

	// v_5(50) = v_5(2 * 5^2) = 2
	r50 := new(big.Rat).SetInt64(50)
	v, err := Valuation(r50, p5)
	if err != nil || v != 2 {
		t.Fatalf("expected v_5(50)=2, got %d, err=%v", v, err)
	}

	norm50, err := Norm(r50, p5)
	if err != nil {
		t.Fatalf("unexpected err in Norm: %v", err)
	}
	expectedNorm50 := new(big.Rat).SetFrac64(1, 25)
	if norm50.Cmp(expectedNorm50) != 0 {
		t.Fatalf("expected norm=1/25, got %s", norm50.String())
	}

	// v_5(1/25) = -2
	r1_25 := new(big.Rat).SetFrac64(1, 25)
	v, err = Valuation(r1_25, p5)
	if err != nil || v != -2 {
		t.Fatalf("expected v_5(1/25)=-2, got %d, err=%v", v, err)
	}
	norm1_25, err := Norm(r1_25, p5)
	if err != nil || norm1_25.Cmp(new(big.Rat).SetInt64(25)) != 0 {
		t.Fatalf("expected norm=25, got %s, err=%v", norm1_25.String(), err)
	}

	// v_3(7) = 0
	p3 := big.NewInt(3)
	r7 := new(big.Rat).SetInt64(7)
	v, err = Valuation(r7, p3)
	if err != nil || v != 0 {
		t.Fatalf("expected v_3(7)=0, got %d, err=%v", v, err)
	}

	// zero valuation
	_, err = Valuation(new(big.Rat), p5)
	if err == nil {
		t.Fatal("expected error for Valuation(0)")
	}
	norm0, err := Norm(new(big.Rat), p5)
	if err != nil || norm0.Sign() != 0 {
		t.Fatalf("expected norm(0)=0, got %s, err=%v", norm0.String(), err)
	}
}

func TestPadicArithmetic(t *testing.T) {
	p5 := big.NewInt(5)
	prec := 5

	// a = 2/3, b = 4/7 in Q_5
	aRat := new(big.Rat).SetFrac64(2, 3)
	bRat := new(big.Rat).SetFrac64(4, 7)

	a, err := NewPadicFromRat(aRat, p5, prec)
	if err != nil {
		t.Fatalf("NewPadicFromRat failed: %v", err)
	}
	b, err := NewPadicFromRat(bRat, p5, prec)
	if err != nil {
		t.Fatalf("NewPadicFromRat failed: %v", err)
	}

	// a + b in Q: 2/3 + 4/7 = (14 + 12)/21 = 26/21
	sumRat := new(big.Rat).Add(aRat, bRat)
	expectedSum, _ := NewPadicFromRat(sumRat, p5, prec)

	sum, err := Add(a, b)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if sum.Valuation != expectedSum.Valuation || sum.Unit.Cmp(expectedSum.Unit) != 0 {
		t.Fatalf("Add mismatch: got v=%d u=%s, expected v=%d u=%s", sum.Valuation, sum.Unit.String(), expectedSum.Valuation, expectedSum.Unit.String())
	}

	// a * b in Q: 8/21
	prodRat := new(big.Rat).Mul(aRat, bRat)
	expectedProd, _ := NewPadicFromRat(prodRat, p5, prec)

	prod, err := Mul(a, b)
	if err != nil {
		t.Fatalf("Mul failed: %v", err)
	}
	if prod.Valuation != expectedProd.Valuation || prod.Unit.Cmp(expectedProd.Unit) != 0 {
		t.Fatalf("Mul mismatch: got v=%d u=%s, expected v=%d u=%s", prod.Valuation, prod.Unit.String(), expectedProd.Valuation, expectedProd.Unit.String())
	}

	// a / b in Q: (2/3)/(4/7) = 14/12 = 7/6
	divRat := new(big.Rat).Quo(aRat, bRat)
	expectedDiv, _ := NewPadicFromRat(divRat, p5, prec)

	div, err := Div(a, b)
	if err != nil {
		t.Fatalf("Div failed: %v", err)
	}
	if div.Valuation != expectedDiv.Valuation || div.Unit.Cmp(expectedDiv.Unit) != 0 {
		t.Fatalf("Div mismatch: got v=%d u=%s, expected v=%d u=%s", div.Valuation, div.Unit.String(), expectedDiv.Valuation, expectedDiv.Unit.String())
	}

	// Ultrametric verification
	if !IsUltrametric(a, b) {
		t.Fatal("IsUltrametric failed for a, b")
	}
}

func TestPadicExpansionString(t *testing.T) {
	p5 := big.NewInt(5)
	// -1 in Q_5 with prec 3:
	// -1 = 4 + 4*5 + 4*5^2 (mod 5^3) = 124
	negOneRat := new(big.Rat).SetInt64(-1)
	negOne, err := NewPadicFromRat(negOneRat, p5, 3)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	s := ExpansionString(negOne)
	expected := "4 + 4*5 + 4*5^2 + O(5^3)"
	if s != expected {
		t.Fatalf("expected '%s', got '%s'", expected, s)
	}
}
