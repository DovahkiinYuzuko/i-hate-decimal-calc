package calc

import (
	"math/big"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

func TestCompareMonomials_Lex(t *testing.T) {
	// vars: [x, y, z]
	// m1: x^2 y (2, 1, 0)
	// m2: x y^2 (1, 2, 0)
	// m3: x y   (1, 1, 0)
	m1 := ast.Monomial{Coeff: big.NewRat(1, 1), Exponents: []int{2, 1, 0}}
	m2 := ast.Monomial{Coeff: big.NewRat(1, 1), Exponents: []int{1, 2, 0}}
	m3 := ast.Monomial{Coeff: big.NewRat(1, 1), Exponents: []int{1, 1, 0}}

	if ComparePolyMonomials(m1, m2, ast.OrderLex) <= 0 {
		t.Errorf("expected m1 > m2 in OrderLex")
	}
	if ComparePolyMonomials(m2, m3, ast.OrderLex) <= 0 {
		t.Errorf("expected m2 > m3 in OrderLex")
	}
	if ComparePolyMonomials(m1, m1, ast.OrderLex) != 0 {
		t.Errorf("expected m1 == m1 in OrderLex")
	}
}

func TestComparePolyMonomials_GrevLex(t *testing.T) {
	// vars: [x, y, z]
	// m1: x^2 y (2, 1, 0) -> deg 3
	// m2: x y^2 (1, 2, 0) -> deg 3
	// In GrevLex: total degree equal, look from rightmost:
	// z exponents: 0 == 0
	// y exponents: m1 has 1, m2 has 2. Smaller exponent is greater => m1 > m2
	m1 := ast.Monomial{Coeff: big.NewRat(1, 1), Exponents: []int{2, 1, 0}}
	m2 := ast.Monomial{Coeff: big.NewRat(1, 1), Exponents: []int{1, 2, 0}}

	if ComparePolyMonomials(m1, m2, ast.OrderGrevLex) <= 0 {
		t.Errorf("expected m1 > m2 in OrderGrevLex")
	}

	// m3: x^4 (4, 0, 0) -> deg 4
	// m3 > m1 because deg 4 > deg 3
	m3 := ast.Monomial{Coeff: big.NewRat(1, 1), Exponents: []int{4, 0, 0}}
	if ComparePolyMonomials(m3, m1, ast.OrderGrevLex) <= 0 {
		t.Errorf("expected m3 > m1 in OrderGrevLex")
	}
}

func TestNewPolyNode_CanonicalInvariants(t *testing.T) {
	vars := []string{"x", "y"}
	// Input with duplicate exponent vectors and a zero term
	terms := []ast.Monomial{
		{Coeff: big.NewRat(2, 1), Exponents: []int{1, 1}},  // 2xy
		{Coeff: big.NewRat(3, 1), Exponents: []int{2, 0}},  // 3x^2
		{Coeff: big.NewRat(4, 1), Exponents: []int{1, 1}},  // 4xy
		{Coeff: big.NewRat(0, 1), Exponents: []int{0, 1}},  // 0y
		{Coeff: big.NewRat(-6, 1), Exponents: []int{1, 1}}, // -6xy (cancels xy to 0)
	}

	poly := NewPolyNode(vars, ast.OrderLex, terms)

	// After combining: 2xy + 4xy - 6xy = 0xy (removed), 3x^2 remains
	if len(poly.Terms) != 1 {
		t.Fatalf("expected 1 term, got %d", len(poly.Terms))
	}
	if poly.Terms[0].Coeff.Cmp(big.NewRat(3, 1)) != 0 || poly.Terms[0].Exponents[0] != 2 {
		t.Errorf("unexpected term: %v", poly.Terms[0])
	}
}

func TestAddSubPoly(t *testing.T) {
	vars := []string{"x"}
	// p1 = 2x^2 + 3x + 1
	p1 := NewPolyNode(vars, ast.OrderLex, []ast.Monomial{
		{Coeff: big.NewRat(2, 1), Exponents: []int{2}},
		{Coeff: big.NewRat(3, 1), Exponents: []int{1}},
		{Coeff: big.NewRat(1, 1), Exponents: []int{0}},
	})
	// p2 = x^2 - 3x + 4
	p2 := NewPolyNode(vars, ast.OrderLex, []ast.Monomial{
		{Coeff: big.NewRat(1, 1), Exponents: []int{2}},
		{Coeff: big.NewRat(-3, 1), Exponents: []int{1}},
		{Coeff: big.NewRat(4, 1), Exponents: []int{0}},
	})

	// sum = 3x^2 + 5
	sum := AddPoly(p1, p2)
	if sum.String() != "3*x^2 + 5" {
		t.Errorf("AddPoly result = %q, want \"3*x^2 + 5\"", sum.String())
	}

	// diff = x^2 + 6x - 3
	diff := SubPoly(p1, p2)
	if diff.String() != "x^2 + 6*x - 3" {
		t.Errorf("SubPoly result = %q, want \"x^2 + 6*x - 3\"", diff.String())
	}
}

func TestMulPoly_MonaganPearceHeap(t *testing.T) {
	vars := []string{"x", "y"}
	// (x + y) * (x - y) = x^2 - y^2
	p1 := NewPolyNode(vars, ast.OrderLex, []ast.Monomial{
		{Coeff: big.NewRat(1, 1), Exponents: []int{1, 0}},
		{Coeff: big.NewRat(1, 1), Exponents: []int{0, 1}},
	})
	p2 := NewPolyNode(vars, ast.OrderLex, []ast.Monomial{
		{Coeff: big.NewRat(1, 1), Exponents: []int{1, 0}},
		{Coeff: big.NewRat(-1, 1), Exponents: []int{0, 1}},
	})

	prod := MulPoly(p1, p2)
	expected := "x^2 - y^2"
	if prod.String() != expected {
		t.Errorf("MulPoly (x+y)(x-y) = %q, want %q", prod.String(), expected)
	}

	// (x + y)^3 = x^3 + 3x^2y + 3xy^2 + y^3
	prod2 := MulPoly(p1, p1) // (x+y)^2
	prod3 := MulPoly(prod2, p1) // (x+y)^3
	expectedCube := "x^3 + 3*x^2*y + 3*x*y^2 + y^3"
	if prod3.String() != expectedCube {
		t.Errorf("MulPoly (x+y)^3 = %q, want %q", prod3.String(), expectedCube)
	}
}

func TestPseudoDivRem(t *testing.T) {
	vars := []string{"x"}
	// A = 2x^3 - 4x + 5
	A := NewPolyNode(vars, ast.OrderLex, []ast.Monomial{
		{Coeff: big.NewRat(2, 1), Exponents: []int{3}},
		{Coeff: big.NewRat(-4, 1), Exponents: []int{1}},
		{Coeff: big.NewRat(5, 1), Exponents: []int{0}},
	})
	// B = x^2 + 1
	B := NewPolyNode(vars, ast.OrderLex, []ast.Monomial{
		{Coeff: big.NewRat(1, 1), Exponents: []int{2}},
		{Coeff: big.NewRat(1, 1), Exponents: []int{0}},
	})

	quo, rem, factor, err := PseudoDivRem(A, B, "x")
	if err != nil {
		t.Fatalf("PseudoDivRem error: %v", err)
	}

	// Verify identity: factor * A == quo * B + rem
	factorTerm := ast.Monomial{Coeff: factor, Exponents: []int{0}}
	factorPoly := NewPolyNode(vars, ast.OrderLex, []ast.Monomial{factorTerm})
	scaledA := MulPoly(A, factorPoly)

	quoTimesB := MulPoly(quo, B)
	reconstructedA := AddPoly(quoTimesB, rem)

	if !scaledA.Equal(reconstructedA) {
		t.Errorf("identity failed: scaledA = %s, quo*B + rem = %s", scaledA.String(), reconstructedA.String())
	}
}

func TestNodeToPoly_And_PolyToNode(t *testing.T) {
	// Build AST: (x + y)^2
	xNode := ast.NewVar("x")
	yNode := ast.NewVar("y")
	addNode := ast.NewAdd([]ast.Node{xNode, yNode})
	twoRat, _ := ast.NewRational(2, 1)
	powNode, _ := ast.NewPow(addNode, twoRat)

	poly, err := NodeToPoly(powNode, []string{"x", "y"}, ast.OrderLex)
	if err != nil {
		t.Fatalf("NodeToPoly failed: %v", err)
	}

	expected := "x^2 + 2*x*y + y^2"
	if poly.String() != expected {
		t.Errorf("poly.String() = %q, want %q", poly.String(), expected)
	}

	// Convert back to AST
	backAST := PolyToNode(poly)
	if backAST == nil {
		t.Fatalf("PolyToNode returned nil")
	}

	// Re-converting backAST should produce the same canonical poly
	poly2, err := NodeToPoly(backAST, []string{"x", "y"}, ast.OrderLex)
	if err != nil {
		t.Fatalf("re-converting backAST failed: %v", err)
	}
	if !poly.Equal(poly2) {
		t.Errorf("roundtrip conversion mismatch: %s vs %s", poly.String(), poly2.String())
	}
}

func BenchmarkMulPoly_Heap(b *testing.B) {
	vars := []string{"x", "y", "z"}
	// (x + y + z)^5
	base := NewPolyNode(vars, ast.OrderLex, []ast.Monomial{
		{Coeff: big.NewRat(1, 1), Exponents: []int{1, 0, 0}},
		{Coeff: big.NewRat(1, 1), Exponents: []int{0, 1, 0}},
		{Coeff: big.NewRat(1, 1), Exponents: []int{0, 0, 1}},
	})
	p := base
	for i := 0; i < 4; i++ {
		p = MulPoly(p, base)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = MulPoly(p, p)
	}
}
