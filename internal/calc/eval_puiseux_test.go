package calc

import (
	"math/big"
	"strings"
	"testing"
)

func TestEvalPuiseux_Cusp(t *testing.T) {
	// y^2 - x^3 == 0  => y = x^(3/2), -x^(3/2)
	res, err := EvalString("puiseux(y^2 - x^3, y, x, 3)")
	if err != nil {
		t.Fatalf("unexpected error for cusp: %v", err)
	}

	str := Format(res)
	if !strings.Contains(str, "x^(3/2)") {
		t.Fatalf("expected x^(3/2) in result, got %s", str)
	}
}

func TestEvalPuiseux_Node(t *testing.T) {
	// y^2 - x^2 - x^3 == 0 => y = x + 1/2*x^2 + ..., -x - 1/2*x^2 + ...
	res, err := EvalString("puiseux(y^2 - x^2 - x^3, y, x, 2)")
	if err != nil {
		t.Fatalf("unexpected error for node: %v", err)
	}

	str := Format(res)
	if !strings.Contains(str, "x") {
		t.Fatalf("expected leading term x in node, got %s", str)
	}
}

func TestEvalPuiseux_CubeRootBranch(t *testing.T) {
	// y^3 - x == 0 => y = x^(1/3)
	res, err := EvalString("puiseux(y^3 - x, y, x, 3)")
	if err != nil {
		t.Fatalf("unexpected error for y^3 - x: %v", err)
	}

	str := Format(res)
	if !strings.Contains(str, "x^(1/3)") {
		t.Fatalf("expected x^(1/3) in result, got %s", str)
	}
}

func assertPuiseuxBranchTerms(t *testing.T, node Node, wantCoeffs, wantExps []*big.Rat, wantOrder *big.Rat) {
	t.Helper()
	pNode, ok := node.(*PuiseuxNode)
	if !ok {
		t.Fatalf("expected *PuiseuxNode, got %T (%v)", node, node)
	}
	if pNode.TruncationOrder.Cmp(wantOrder) != 0 {
		t.Fatalf("truncation order = %v, want %v", pNode.TruncationOrder, wantOrder)
	}
	if len(pNode.Terms) != len(wantCoeffs) {
		t.Fatalf("expected %d terms, got %d (%s)", len(wantCoeffs), len(pNode.Terms), Format(pNode))
	}
	for i := range wantCoeffs {
		if pNode.Terms[i].Coeff.Cmp(wantCoeffs[i]) != 0 || pNode.Terms[i].Exp.Cmp(wantExps[i]) != 0 {
			t.Fatalf("term[%d] = %v*x^(%v), want %v*x^(%v)", i, pNode.Terms[i].Coeff, pNode.Terms[i].Exp, wantCoeffs[i], wantExps[i])
		}
	}
}

func TestEvalPuiseux_TaylorType(t *testing.T) {
	// y - (1 + 2*x + 3*x^2) == 0 => y = 1 + 2*x + 3*x^2 + O(x^3)
	res, err := EvalString("puiseux(y - (1 + 2*x + 3*x^2), y, x, 3)")
	if err != nil {
		t.Fatalf("unexpected error for polynomial: %v", err)
	}

	list, ok := res.(*ListNode)
	if !ok || len(list.Elements) != 1 {
		t.Fatalf("expected 1 branch, got %T (%v)", res, res)
	}
	assertPuiseuxBranchTerms(
		t,
		list.Elements[0],
		[]*big.Rat{big.NewRat(1, 1), big.NewRat(2, 1), big.NewRat(3, 1)},
		[]*big.Rat{big.NewRat(0, 1), big.NewRat(1, 1), big.NewRat(2, 1)},
		big.NewRat(3, 1),
	)
}

func TestEvalPuiseux_HigherOrderFractional(t *testing.T) {
	// y^2 - x*(1 + x) == 0 => y = +-(x^(1/2) + (1/2)*x^(3/2) - (1/8)*x^(5/2)) + O(x^3)
	res, err := EvalString("puiseux(y^2 - x*(1 + x), y, x, 3)")
	if err != nil {
		t.Fatalf("unexpected error for y^2 - x*(1 + x): %v", err)
	}

	list, ok := res.(*ListNode)
	if !ok || len(list.Elements) != 2 {
		t.Fatalf("expected 2 branches, got %T (%v)", res, res)
	}

	wantExps := []*big.Rat{big.NewRat(1, 2), big.NewRat(3, 2), big.NewRat(5, 2)}
	assertPuiseuxBranchTerms(
		t,
		list.Elements[0],
		[]*big.Rat{big.NewRat(-1, 1), big.NewRat(-1, 2), big.NewRat(1, 8)},
		wantExps,
		big.NewRat(3, 1),
	)
	assertPuiseuxBranchTerms(
		t,
		list.Elements[1],
		[]*big.Rat{big.NewRat(1, 1), big.NewRat(1, 2), big.NewRat(-1, 8)},
		wantExps,
		big.NewRat(3, 1),
	)
}

func TestEvalPuiseux_SubsequentBranching(t *testing.T) {
	// (y - x)^2 - x^3 == 0 => y = x - x^(3/2) + O(x^3), x + x^(3/2) + O(x^3)
	res, err := EvalString("puiseux((y - x)^2 - x^3, y, x, 3)")
	if err != nil {
		t.Fatalf("unexpected error for (y - x)^2 - x^3: %v", err)
	}

	list, ok := res.(*ListNode)
	if !ok || len(list.Elements) != 2 {
		t.Fatalf("expected 2 branches after secondary split, got %T (%v)", res, res)
	}

	wantExps := []*big.Rat{big.NewRat(1, 1), big.NewRat(3, 2)}
	assertPuiseuxBranchTerms(
		t,
		list.Elements[0],
		[]*big.Rat{big.NewRat(1, 1), big.NewRat(-1, 1)},
		wantExps,
		big.NewRat(3, 1),
	)
	assertPuiseuxBranchTerms(
		t,
		list.Elements[1],
		[]*big.Rat{big.NewRat(1, 1), big.NewRat(1, 1)},
		wantExps,
		big.NewRat(3, 1),
	)
}

func TestEvalPuiseux_DistinctHigherEdges(t *testing.T) {
	// (y - x - x^2)*(y - x - x^3) == 0 => up to O(x^3): [x + x^2 + O(x^3), x + O(x^3)]
	res, err := EvalString("puiseux((y - x - x^2)*(y - x - x^3), y, x, 3)")
	if err != nil {
		t.Fatalf("unexpected error for multi-edge secondary split: %v", err)
	}

	list, ok := res.(*ListNode)
	if !ok || len(list.Elements) != 2 {
		t.Fatalf("expected 2 distinct branches, got %T (%v)", res, res)
	}
	assertPuiseuxBranchTerms(
		t,
		list.Elements[0],
		[]*big.Rat{big.NewRat(1, 1), big.NewRat(1, 1)},
		[]*big.Rat{big.NewRat(1, 1), big.NewRat(2, 1)},
		big.NewRat(3, 1),
	)
	assertPuiseuxBranchTerms(
		t,
		list.Elements[1],
		[]*big.Rat{big.NewRat(1, 1)},
		[]*big.Rat{big.NewRat(1, 1)},
		big.NewRat(3, 1),
	)
}

func TestEvalPuiseux_NodeOrder3AndResidual(t *testing.T) {
	// y^2 - x^2 - x^3 == 0 at order 3 => +-(x + (1/2)*x^2) + O(x^3)
	res, err := EvalString("puiseux(y^2 - x^2 - x^3, y, x, 3)")
	if err != nil {
		t.Fatalf("unexpected error for node order 3: %v", err)
	}

	list, ok := res.(*ListNode)
	if !ok || len(list.Elements) != 2 {
		t.Fatalf("expected 2 branches, got %T (%v)", res, res)
	}

	wantExps := []*big.Rat{big.NewRat(1, 1), big.NewRat(2, 1)}
	assertPuiseuxBranchTerms(
		t,
		list.Elements[0],
		[]*big.Rat{big.NewRat(-1, 1), big.NewRat(-1, 2)},
		wantExps,
		big.NewRat(3, 1),
	)
	assertPuiseuxBranchTerms(
		t,
		list.Elements[1],
		[]*big.Rat{big.NewRat(1, 1), big.NewRat(1, 2)},
		wantExps,
		big.NewRat(3, 1),
	)

	// Verify algebraic substitution residual vanishes below order + min valuation (here >= 4)
	expr, err := Parse("y^2 - x^2 - x^3")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	for idx, elem := range list.Elements {
		pNode := elem.(*PuiseuxNode)
		var yExprTerms []Node
		for _, term := range pNode.Terms {
			cNode := &RationalNode{Val: new(big.Rat).Set(term.Coeff)}
			xPow := &PowNode{Base: NewVar("x"), Exp: &RationalNode{Val: new(big.Rat).Set(term.Exp)}}
			m, _ := simplifyMul([]Node{cNode, xPow})
			yExprTerms = append(yExprTerms, m)
		}
		yExpr, _ := simplifyAdd(yExprTerms)
		residual := expandAndEval(Substitute(expr, "y", yExpr))
		coeffs, err := extractPolyCoeffs(residual, "x")
		if err != nil {
			t.Fatalf("branch[%d] residual extractPolyCoeffs failed: %v", idx, err)
		}
		for deg, c := range coeffs {
			if deg < 4 && !isZero(c) {
				t.Fatalf("branch[%d] residual has non-zero term at x^%d: %s", idx, deg, Format(residual))
			}
		}
	}
}

func TestEvalPuiseux_FineFractionalOrder(t *testing.T) {
	// y^3 - x*(1 + x) == 0 => real branch x^(1/3) + (1/3)*x^(4/3) + O(x^2)
	// Verify that fractional step size does not get cut off prematurely.
	res, err := EvalString("puiseux(y^3 - x*(1 + x), y, x, 2)")
	if err != nil {
		t.Fatalf("unexpected error for y^3 - x*(1 + x): %v", err)
	}

	list, ok := res.(*ListNode)
	if !ok || len(list.Elements) != 3 {
		t.Fatalf("expected 3 branches, got %T (%v)", res, res)
	}

	assertPuiseuxBranchTerms(
		t,
		list.Elements[0],
		[]*big.Rat{big.NewRat(1, 1), big.NewRat(1, 3)},
		[]*big.Rat{big.NewRat(1, 3), big.NewRat(4, 3)},
		big.NewRat(2, 1),
	)
}

func TestEvalPuiseux_NegativeCases(t *testing.T) {
	// 1. Missing target variable y
	_, err := EvalString("puiseux(x^2 - 1, y, x)")
	if err == nil {
		t.Fatalf("expected error when equation contains no y, got nil")
	}

	// 2. Identity equation
	_, err = EvalString("puiseux(0, y, x)")
	if err == nil {
		t.Fatalf("expected error for identity equation, got nil")
	}

	// 3. Negative order
	_, err = EvalString("puiseux(y^2 - x, y, x, -1)")
	if err == nil {
		t.Fatalf("expected error for negative order, got nil")
	}

	// 4. Non-variable argument for y
	_, err = EvalString("puiseux(y^2 - x, 42, x)")
	if err == nil {
		t.Fatalf("expected error for non-variable y, got nil")
	}

	// 5. Wrong argument count
	_, err = EvalString("puiseux(y^2 - x, y)")
	if err == nil {
		t.Fatalf("expected error for insufficient arguments, got nil")
	}
}
