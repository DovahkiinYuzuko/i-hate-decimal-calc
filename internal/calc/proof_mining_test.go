package calc

import (
	"math/big"
	"testing"
)

func TestVerifyIdealMembership_Success(t *testing.T) {
	// f = x^2 - y^2
	// f1 = x - y, f2 = x + y
	// h1 = x + y, h2 = 0
	f := mustParseNode(t, "x^2 - y^2")
	f1 := mustParseNode(t, "x - y")
	f2 := mustParseNode(t, "x + y")
	h1 := mustParseNode(t, "x + y")
	h2 := mustParseNode(t, "0")

	vars := []string{"x", "y"}
	order := OrderGRevLex

	residual, ok, err := VerifyIdealMembership(f, []Node{f1, f2}, []Node{h1, h2}, vars, order)
	if err != nil {
		t.Fatalf("VerifyIdealMembership returned error: %v", err)
	}
	if !ok {
		t.Errorf("expected identity to hold, residual was: %v", residual)
	}
	if !residual.IsZero() {
		t.Errorf("expected residual to be zero, got: %v", residual)
	}
}

func TestVerifyIdealMembership_Mismatch(t *testing.T) {
	// Wrong cofactor: h1 = x instead of x + y
	f := mustParseNode(t, "x^2 - y^2")
	f1 := mustParseNode(t, "x - y")
	f2 := mustParseNode(t, "x + y")
	h1 := mustParseNode(t, "x")
	h2 := mustParseNode(t, "0")

	vars := []string{"x", "y"}
	order := OrderGRevLex

	residual, ok, err := VerifyIdealMembership(f, []Node{f1, f2}, []Node{h1, h2}, vars, order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Errorf("expected identity to fail for wrong cofactor")
	}
	if residual.IsZero() {
		t.Errorf("expected non-zero residual for wrong cofactor")
	}
}

func TestVerifySOSDecomposition_Success(t *testing.T) {
	// P = x^2 + 2*x + 1 = 1 * (x + 1)^2
	p := mustParseNode(t, "x^2 + 2*x + 1")
	q := mustParseNode(t, "x + 1")

	terms := []SOSCertificateTerm{
		{
			Weight: big.NewRat(1, 1),
			Poly:   q,
		},
	}
	vars := []string{"x"}
	order := OrderGRevLex

	residual, ok, err := VerifySOSDecomposition(p, terms, vars, order)
	if err != nil {
		t.Fatalf("VerifySOSDecomposition returned error: %v", err)
	}
	if !ok {
		t.Errorf("expected SOS identity to hold, residual was: %v", residual)
	}
	if !residual.IsZero() {
		t.Errorf("expected residual to be zero, got: %v", residual)
	}
}

func TestVerifySOSDecomposition_Mismatch(t *testing.T) {
	p := mustParseNode(t, "x^2 + 2*x + 1")
	q := mustParseNode(t, "x") // Wrong q

	terms := []SOSCertificateTerm{
		{
			Weight: big.NewRat(1, 1),
			Poly:   q,
		},
	}
	vars := []string{"x"}
	order := OrderGRevLex

	residual, ok, err := VerifySOSDecomposition(p, terms, vars, order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Errorf("expected SOS identity to fail for wrong term")
	}
	if residual.IsZero() {
		t.Errorf("expected non-zero residual for wrong term")
	}
}

func TestVerifySOSDecomposition_NegativeWeight(t *testing.T) {
	p := mustParseNode(t, "x^2")
	q := mustParseNode(t, "x")

	terms := []SOSCertificateTerm{
		{
			Weight: big.NewRat(-1, 1), // Negative weight
			Poly:   q,
		},
	}
	vars := []string{"x"}
	order := OrderGRevLex

	_, _, err := VerifySOSDecomposition(p, terms, vars, order)
	if err == nil {
		t.Fatalf("expected error for negative weight, got nil")
	}
}

func TestVerifySOSDecomposition_NilWeightOrPoly(t *testing.T) {
	p := mustParseNode(t, "x^2")
	q := mustParseNode(t, "x")
	vars := []string{"x"}
	order := OrderGRevLex

	// Test nil weight
	termsNilWeight := []SOSCertificateTerm{
		{
			Weight: nil,
			Poly:   q,
		},
	}
	_, _, err := VerifySOSDecomposition(p, termsNilWeight, vars, order)
	if err == nil {
		t.Fatalf("expected error for nil weight, got nil")
	}

	// Test nil poly
	termsNilPoly := []SOSCertificateTerm{
		{
			Weight: big.NewRat(1, 1),
			Poly:   nil,
		},
	}
	_, _, err = VerifySOSDecomposition(p, termsNilPoly, vars, order)
	if err == nil {
		t.Fatalf("expected error for nil poly, got nil")
	}
}
