package ast

import (
	"math/big"
	"testing"
)

func TestAlgebraicRealNode_Basics(t *testing.T) {
	// poly: x^2 - 2 -> coeffs: [-2, 0, 1]
	poly := []*big.Int{big.NewInt(-2), big.NewInt(0), big.NewInt(1)}
	low := big.NewRat(1, 1)
	high := big.NewRat(2, 1)
	node := NewAlgebraicRealNode(poly, low, high, 1)

	if node.Type() != NodeAlgebraicReal {
		t.Fatalf("expected NodeAlgebraicReal, got %v", node.Type())
	}

	if node.RootIndex != 1 {
		t.Errorf("expected RootIndex 1, got %d", node.RootIndex)
	}

	if node.Low.Cmp(low) != 0 || node.High.Cmp(high) != 0 {
		t.Errorf("interval mismatch: got [%v, %v]", node.Low, node.High)
	}

	str := node.String()
	if str == "" {
		t.Errorf("expected non-empty string representation")
	}

	node2 := NewAlgebraicRealNode(poly, low, high, 1)
	if !node.Equal(node2) {
		t.Errorf("expected identical nodes to be Equal")
	}

	nodeDifferent := NewAlgebraicRealNode(poly, big.NewRat(0, 1), high, 1)
	if node.Equal(nodeDifferent) {
		t.Errorf("expected different intervals not to be Equal")
	}
}

func TestCFNode_Basics(t *testing.T) {
	terms := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(2)}
	node := NewCFNode(terms, nil, false)

	if node.Type() != NodeCF {
		t.Fatalf("expected NodeCF, got %v", node.Type())
	}

	if len(node.Terms) != 3 {
		t.Errorf("expected 3 terms, got %d", len(node.Terms))
	}

	str := node.String()
	if str == "" {
		t.Errorf("expected non-empty string representation")
	}

	node2 := NewCFNode(terms, nil, false)
	if !node.Equal(node2) {
		t.Errorf("expected identical CF nodes to be Equal")
	}
}
