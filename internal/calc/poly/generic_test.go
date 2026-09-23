package poly

import (
	"math/big"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/domain"
)

func TestGenericPolynomialRational(t *testing.T) {
	qDom := domain.NewRationalDomain()
	vars := []string{"x", "y"}

	// p1 = 2*x^2 + 3*x*y + 1
	// p2 = x*y + 4
	p1 := NewPolynomial(qDom, vars, ast.OrderLex, []Term[*big.Rat]{
		{Coeff: big.NewRat(2, 1), Exponents: []int{2, 0}},
		{Coeff: big.NewRat(3, 1), Exponents: []int{1, 1}},
		{Coeff: big.NewRat(1, 1), Exponents: []int{0, 0}},
	})

	p2 := NewPolynomial(qDom, vars, ast.OrderLex, []Term[*big.Rat]{
		{Coeff: big.NewRat(1, 1), Exponents: []int{1, 1}},
		{Coeff: big.NewRat(4, 1), Exponents: []int{0, 0}},
	})

	// Add: p1 + p2 = 2*x^2 + 4*x*y + 5
	sum, err := p1.Add(p2)
	if err != nil {
		t.Fatalf("Add error: %v", err)
	}
	if len(sum.Terms) != 3 {
		t.Fatalf("expected 3 terms, got %d: %s", len(sum.Terms), sum.String())
	}
	if sum.Terms[0].Coeff.Cmp(big.NewRat(2, 1)) != 0 ||
		sum.Terms[1].Coeff.Cmp(big.NewRat(4, 1)) != 0 ||
		sum.Terms[2].Coeff.Cmp(big.NewRat(5, 1)) != 0 {
		t.Errorf("unexpected sum terms: %s", sum.String())
	}

	// Sub: p1 - p2 = 2*x^2 + 2*x*y - 3
	diff, err := p1.Sub(p2)
	if err != nil {
		t.Fatalf("Sub error: %v", err)
	}
	if diff.Terms[1].Coeff.Cmp(big.NewRat(2, 1)) != 0 ||
		diff.Terms[2].Coeff.Cmp(big.NewRat(-3, 1)) != 0 {
		t.Errorf("unexpected diff terms: %s", diff.String())
	}

	// Mul: (x + 1) * (x - 1) = x^2 - 1
	px1 := NewPolynomial(qDom, []string{"x"}, ast.OrderLex, []Term[*big.Rat]{
		{Coeff: big.NewRat(1, 1), Exponents: []int{1}},
		{Coeff: big.NewRat(1, 1), Exponents: []int{0}},
	})
	px2 := NewPolynomial(qDom, []string{"x"}, ast.OrderLex, []Term[*big.Rat]{
		{Coeff: big.NewRat(1, 1), Exponents: []int{1}},
		{Coeff: big.NewRat(-1, 1), Exponents: []int{0}},
	})
	prod, err := px1.Mul(px2)
	if err != nil {
		t.Fatalf("Mul error: %v", err)
	}
	if len(prod.Terms) != 2 ||
		prod.Terms[0].Coeff.Cmp(big.NewRat(1, 1)) != 0 || prod.Terms[0].Exponents[0] != 2 ||
		prod.Terms[1].Coeff.Cmp(big.NewRat(-1, 1)) != 0 || prod.Terms[1].Exponents[0] != 0 {
		t.Errorf("expected x^2 - 1, got %s", prod.String())
	}

	// DivRem: (x^3 - 1) / (x - 1) = x^2 + x + 1, rem = 0
	pxCubeMinus1 := NewPolynomial(qDom, []string{"x"}, ast.OrderLex, []Term[*big.Rat]{
		{Coeff: big.NewRat(1, 1), Exponents: []int{3}},
		{Coeff: big.NewRat(-1, 1), Exponents: []int{0}},
	})
	quot, rem, err := DivRem(pxCubeMinus1, px2, 0) // px2 is (x - 1)
	if err != nil {
		t.Fatalf("DivRem error: %v", err)
	}
	if !rem.IsZero() {
		t.Errorf("expected rem 0, got %s", rem.String())
	}
	if len(quot.Terms) != 3 ||
		quot.Terms[0].Exponents[0] != 2 ||
		quot.Terms[1].Exponents[0] != 1 ||
		quot.Terms[2].Exponents[0] != 0 {
		t.Errorf("expected x^2 + x + 1, got %s", quot.String())
	}

	// Monic: 2*x^2 + 4 -> x^2 + 2
	p2x := NewPolynomial(qDom, []string{"x"}, ast.OrderLex, []Term[*big.Rat]{
		{Coeff: big.NewRat(2, 1), Exponents: []int{2}},
		{Coeff: big.NewRat(4, 1), Exponents: []int{0}},
	})
	monic, err := p2x.Monic()
	if err != nil {
		t.Fatalf("Monic error: %v", err)
	}
	if monic.Terms[0].Coeff.Cmp(big.NewRat(1, 1)) != 0 || monic.Terms[1].Coeff.Cmp(big.NewRat(2, 1)) != 0 {
		t.Errorf("expected x^2 + 2, got %s", monic.String())
	}
}

func TestGenericPolynomialFiniteField(t *testing.T) {
	// F_17[x]
	f17, err := domain.NewFiniteFieldDomain(big.NewInt(17))
	if err != nil {
		t.Fatalf("failed to create F_17: %v", err)
	}

	// p1 = 10*x + 5
	// p2 = 12*x + 3
	p1 := NewPolynomial(f17, []string{"x"}, ast.OrderLex, []Term[*big.Int]{
		{Coeff: big.NewInt(10), Exponents: []int{1}},
		{Coeff: big.NewInt(5), Exponents: []int{0}},
	})
	p2 := NewPolynomial(f17, []string{"x"}, ast.OrderLex, []Term[*big.Int]{
		{Coeff: big.NewInt(12), Exponents: []int{1}},
		{Coeff: big.NewInt(3), Exponents: []int{0}},
	})

	// Add: (10*x + 5) + (12*x + 3) = 22*x + 8 = 5*x + 8 mod 17
	sum, err := p1.Add(p2)
	if err != nil {
		t.Fatalf("Add error: %v", err)
	}
	if sum.Terms[0].Coeff.Cmp(big.NewInt(5)) != 0 || sum.Terms[1].Coeff.Cmp(big.NewInt(8)) != 0 {
		t.Errorf("expected 5*x + 8, got %s", sum.String())
	}

	// Monic of (10*x + 5) in F_17:
	// 10^(-1) = 12 mod 17
	// 12 * (10*x + 5) = 120*x + 60 = 1*x + 9 mod 17
	monic, err := p1.Monic()
	if err != nil {
		t.Fatalf("Monic error: %v", err)
	}
	if monic.Terms[0].Coeff.Cmp(big.NewInt(1)) != 0 || monic.Terms[1].Coeff.Cmp(big.NewInt(9)) != 0 {
		t.Errorf("expected x + 9, got %s", monic.String())
	}
}

func TestPseudoDivRem(t *testing.T) {
	qDom := domain.NewRationalDomain()
	// f = 3*x^2 + 5*x + 2
	// g = 2*x + 1
	f := NewPolynomial(qDom, []string{"x"}, ast.OrderLex, []Term[*big.Rat]{
		{Coeff: big.NewRat(3, 1), Exponents: []int{2}},
		{Coeff: big.NewRat(5, 1), Exponents: []int{1}},
		{Coeff: big.NewRat(2, 1), Exponents: []int{0}},
	})
	g := NewPolynomial(qDom, []string{"x"}, ast.OrderLex, []Term[*big.Rat]{
		{Coeff: big.NewRat(2, 1), Exponents: []int{1}},
		{Coeff: big.NewRat(1, 1), Exponents: []int{0}},
	})

	q, r, delta, err := PseudoDivRem(f, g, 0)
	if err != nil {
		t.Fatalf("PseudoDivRem error: %v", err)
	}
	if delta != 2 {
		t.Errorf("expected delta 2, got %d", delta)
	}
	if r.Degree(0) >= g.Degree(0) {
		t.Errorf("expected deg(r) < deg(g), got deg(r)=%d, deg(g)=%d", r.Degree(0), g.Degree(0))
	}

	// Verify identity: lc(g)^delta * f == q * g + r
	// lc(g) = 2, delta = 2 -> 2^2 = 4
	// 4 * (3*x^2 + 5*x + 2) = 12*x^2 + 20*x + 8
	// q * g + r
	qTimesG, err := q.Mul(g)
	if err != nil {
		t.Fatalf("Mul error: %v", err)
	}
	rhs, err := qTimesG.Add(r)
	if err != nil {
		t.Fatalf("Add error: %v", err)
	}
	lhs := f.ScalarMul(big.NewRat(4, 1))

	diff, err := lhs.Sub(rhs)
	if err != nil {
		t.Fatalf("Sub error: %v", err)
	}
	if !diff.IsZero() {
		t.Errorf("Pseudo-division identity failed: lhs - rhs = %s", diff.String())
	}
}

func TestCompatPolyNode(t *testing.T) {
	node := &ast.PolyNode{
		Vars:  []string{"x", "y"},
		Order: ast.OrderLex,
		Terms: []ast.Monomial{
			{Coeff: big.NewRat(3, 2), Exponents: []int{2, 1}},
			{Coeff: big.NewRat(-1, 1), Exponents: []int{0, 0}},
		},
	}

	gen := FromPolyNode(node)
	if len(gen.Terms) != 2 {
		t.Fatalf("expected 2 terms in FromPolyNode, got %d", len(gen.Terms))
	}

	back := ToPolyNode(gen)
	if len(back.Terms) != 2 {
		t.Fatalf("expected 2 terms in ToPolyNode, got %d", len(back.Terms))
	}
	if back.Terms[0].Coeff.Cmp(big.NewRat(3, 2)) != 0 ||
		back.Terms[1].Coeff.Cmp(big.NewRat(-1, 1)) != 0 {
		t.Errorf("unexpected round-trip: %s", back.String())
	}
}
