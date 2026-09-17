package calc

import (
	"math/big"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

func TestAlgebraicNumber_BasicArithmetic(t *testing.T) {
	// Q(sqrt(2)) where m(x) = x^2 - 2
	minPoly := &ast.PolyNode{
		Vars:  []string{"x"},
		Order: ast.OrderLex,
		Terms: []ast.Monomial{
			{Coeff: big.NewRat(1, 1), Exponents: []int{2}},
			{Coeff: big.NewRat(-2, 1), Exponents: []int{0}},
		},
	}

	// a = 1 + x (representing 1 + sqrt(2))
	repA := &ast.PolyNode{
		Vars:  []string{"x"},
		Order: ast.OrderLex,
		Terms: []ast.Monomial{
			{Coeff: big.NewRat(1, 1), Exponents: []int{1}},
			{Coeff: big.NewRat(1, 1), Exponents: []int{0}},
		},
	}
	algA, err := NewAlgebraicNumber(minPoly, repA, "sqrt2")
	if err != nil {
		t.Fatalf("NewAlgebraicNumber failed: %v", err)
	}

	// (1 + sqrt(2)) + (1 + sqrt(2)) = 2 + 2*sqrt(2)
	sum, err := AddAlg(algA, algA)
	if err != nil {
		t.Fatalf("AddAlg failed: %v", err)
	}
	if len(sum.RepPoly.Terms) != 2 ||
		sum.RepPoly.Terms[0].Coeff.Cmp(big.NewRat(2, 1)) != 0 ||
		sum.RepPoly.Terms[1].Coeff.Cmp(big.NewRat(2, 1)) != 0 {
		t.Errorf("expected 2*x + 2, got %s", sum.RepPoly.String())
	}

	// (1 + sqrt(2)) * (1 + sqrt(2)) = 1 + 2*sqrt(2) + 2 = 3 + 2*sqrt(2)
	prod, err := MulAlg(algA, algA)
	if err != nil {
		t.Fatalf("MulAlg failed: %v", err)
	}
	if len(prod.RepPoly.Terms) != 2 ||
		prod.RepPoly.Terms[0].Coeff.Cmp(big.NewRat(2, 1)) != 0 ||
		prod.RepPoly.Terms[0].Exponents[0] != 1 ||
		prod.RepPoly.Terms[1].Coeff.Cmp(big.NewRat(3, 1)) != 0 ||
		prod.RepPoly.Terms[1].Exponents[0] != 0 {
		t.Errorf("expected 2*x + 3, got %s", prod.RepPoly.String())
	}

	// (1 + sqrt(2)) - (1 + sqrt(2)) = 0
	sub, err := SubAlg(algA, algA)
	if err != nil {
		t.Fatalf("SubAlg failed: %v", err)
	}
	if !IsZeroAlg(sub) {
		t.Errorf("expected IsZeroAlg to be true, got %s", sub.RepPoly.String())
	}
}

func TestAlgebraicNumber_Inverse(t *testing.T) {
	// Q(sqrt(2)) : m(x) = x^2 - 2
	minPoly := &ast.PolyNode{
		Vars:  []string{"x"},
		Order: ast.OrderLex,
		Terms: []ast.Monomial{
			{Coeff: big.NewRat(1, 1), Exponents: []int{2}},
			{Coeff: big.NewRat(-2, 1), Exponents: []int{0}},
		},
	}

	// 1 / (1 + x) = -1 + x (since (1 + sqrt(2)) * (-1 + sqrt(2)) = 2 - 1 = 1)
	repA := &ast.PolyNode{
		Vars:  []string{"x"},
		Order: ast.OrderLex,
		Terms: []ast.Monomial{
			{Coeff: big.NewRat(1, 1), Exponents: []int{1}},
			{Coeff: big.NewRat(1, 1), Exponents: []int{0}},
		},
	}
	algA, _ := NewAlgebraicNumber(minPoly, repA, "sqrt2")
	invA, err := InvAlg(algA)
	if err != nil {
		t.Fatalf("InvAlg failed: %v", err)
	}

	// Verify algA * invA = 1
	prod, err := MulAlg(algA, invA)
	if err != nil {
		t.Fatalf("MulAlg check failed: %v", err)
	}
	if len(prod.RepPoly.Terms) != 1 ||
		prod.RepPoly.Terms[0].Coeff.Cmp(big.NewRat(1, 1)) != 0 ||
		prod.RepPoly.Terms[0].Exponents[0] != 0 {
		t.Errorf("expected product to be 1, got %s", prod.RepPoly.String())
	}

	// Test Q(2^(1/3)) where m(x) = x^3 - 2
	// 1 / (1 + x) = 1/3 * (1 - x + x^2)
	minCbrt := &ast.PolyNode{
		Vars:  []string{"x"},
		Order: ast.OrderLex,
		Terms: []ast.Monomial{
			{Coeff: big.NewRat(1, 1), Exponents: []int{3}},
			{Coeff: big.NewRat(-2, 1), Exponents: []int{0}},
		},
	}
	algCbrt, _ := NewAlgebraicNumber(minCbrt, repA, "cbrt2")
	invCbrt, err := InvAlg(algCbrt)
	if err != nil {
		t.Fatalf("InvAlg for cbrt2 failed: %v", err)
	}

	prodCbrt, err := MulAlg(algCbrt, invCbrt)
	if err != nil {
		t.Fatalf("MulAlg check for cbrt2 failed: %v", err)
	}
	if len(prodCbrt.RepPoly.Terms) != 1 ||
		prodCbrt.RepPoly.Terms[0].Coeff.Cmp(big.NewRat(1, 1)) != 0 ||
		prodCbrt.RepPoly.Terms[0].Exponents[0] != 0 {
		t.Errorf("expected product to be 1 in Q(cbrt(2)), got %s", prodCbrt.RepPoly.String())
	}
}

func TestAlgebraicNumber_MinPolySum(t *testing.T) {
	// m1(x) = x^2 - 2 (sqrt(2))
	// m2(x) = x^2 - 3 (sqrt(3))
	// Sum sqrt(2) + sqrt(3) has minimal polynomial y^4 - 10*y^2 + 1
	m1 := &ast.PolyNode{
		Vars:  []string{"x"},
		Order: ast.OrderLex,
		Terms: []ast.Monomial{
			{Coeff: big.NewRat(1, 1), Exponents: []int{2}},
			{Coeff: big.NewRat(-2, 1), Exponents: []int{0}},
		},
	}
	m2 := &ast.PolyNode{
		Vars:  []string{"x"},
		Order: ast.OrderLex,
		Terms: []ast.Monomial{
			{Coeff: big.NewRat(1, 1), Exponents: []int{2}},
			{Coeff: big.NewRat(-3, 1), Exponents: []int{0}},
		},
	}

	minPoly, err := MinPolySum(m1, m2, "y")
	if err != nil {
		t.Fatalf("MinPolySum failed: %v", err)
	}

	// Expect y^4 - 10*y^2 + 1
	expected := &ast.PolyNode{
		Vars:  []string{"y"},
		Order: ast.OrderLex,
		Terms: []ast.Monomial{
			{Coeff: big.NewRat(1, 1), Exponents: []int{4}},
			{Coeff: big.NewRat(-10, 1), Exponents: []int{2}},
			{Coeff: big.NewRat(1, 1), Exponents: []int{0}},
		},
	}

	if !minPoly.Equal(expected) {
		t.Errorf("MinPolySum result = %s, want %s", minPoly.String(), expected.String())
	}
}
