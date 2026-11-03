package calc

import (
	"math/big"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

func TestSimplifyAdd_ComplexRealDuplication(t *testing.T) {
	// Regression test for issue-173: Adding a rational number to a ComplexNode
	// with a rational Real part and non-rational Imag part must not duplicate the Real part.
	// Input: (1 + x*i) + (-10) + 0
	// Correct: -9 + x*i
	// Bug was: -8 + x*i (real part 1 was counted twice)
	complexNode := ast.NewComplex(
		ast.NewRationalFromBigRat(big.NewRat(1, 1)),
		&ast.VarNode{Name: "x"},
	)
	ratNode := ast.NewRationalFromBigRat(big.NewRat(-10, 1))
	zeroNode := ast.NewRationalFromBigRat(big.NewRat(0, 1))

	res, err := simplifyAdd([]ast.Node{complexNode, ratNode, zeroNode})
	if err != nil {
		t.Fatalf("simplifyAdd failed: %v", err)
	}

	addRes, ok := res.(*ast.AddNode)
	if !ok {
		t.Fatalf("expected *ast.AddNode, got %T: %s", res, res.String())
	}

	// Should have two terms: rational -9 and pure complex x*i
	if len(addRes.Terms) != 2 {
		t.Fatalf("expected 2 terms in AddNode, got %d: %s", len(addRes.Terms), addRes.String())
	}

	rPart, okR := addRes.Terms[0].(*ast.RationalNode)
	if !okR || rPart.Val.Cmp(big.NewRat(-9, 1)) != 0 {
		t.Errorf("expected first term to be -9, got %s", addRes.Terms[0].String())
	}

	cPart, okC := addRes.Terms[1].(*ast.ComplexNode)
	if !okC {
		t.Fatalf("expected second term to be ComplexNode, got %T: %s", addRes.Terms[1], addRes.Terms[1].String())
	}
	if !isZero(cPart.Real) {
		t.Errorf("expected ComplexNode Real part to be 0, got %s", cPart.Real.String())
	}
	vPart, okV := cPart.Imag.(*ast.VarNode)
	if !okV || vPart.Name != "x" {
		t.Errorf("expected ComplexNode Imag part to be 'x', got %s", cPart.Imag.String())
	}
}
