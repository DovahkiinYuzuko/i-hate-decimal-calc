package calc

import (
	"math/big"
	"testing"
)

func TestMonomialOrdering(t *testing.T) {
	// 3 variables: x, y, z
	// m1: x^1 y^2 z^3 (total deg 6)
	// m2: x^2 y^0 z^4 (total deg 6)
	m1 := newMonomial([]int{1, 2, 3})
	m2 := newMonomial([]int{2, 0, 4})

	// Lex order: compare x first. m2 has x^2 > m1's x^1 => m2 > m1
	if cmp := CompareMonomials(m1, m2, OrderLex); cmp != -1 {
		t.Errorf("expected m1 < m2 in Lex, got %d", cmp)
	}
	if cmp := CompareMonomials(m2, m1, OrderLex); cmp != 1 {
		t.Errorf("expected m2 > m1 in Lex, got %d", cmp)
	}

	// GRevLex order: both have total deg 6.
	// Compare from z backwards: m1 has z^3, m2 has z^4.
	// In GRevLex, smaller exponent from right wins!
	// So m1 has smaller z => m1 > m2.
	if cmp := CompareMonomials(m1, m2, OrderGRevLex); cmp != 1 {
		t.Errorf("expected m1 > m2 in GRevLex, got %d", cmp)
	}
}

func TestEvalGroebner_CircleLine(t *testing.T) {
	// Intersection of circle x^2 + y^2 - 1 and line x - y
	// Under lex order (x > y), x is eliminated from the second equation:
	// Yields: [x - y, 2*y^2 - 1] (or monic: y^2 - 1/2)
	p1 := mustParseNode(t, "x^2 + y^2 - 1")
	p2 := mustParseNode(t, "x - y")

	polys := NewList([]Node{p1, p2})
	vars := NewList([]Node{&VarNode{Name: "x"}, &VarNode{Name: "y"}})
	order := &VarNode{Name: "lex"}

	res, err := EvalGroebner(polys, vars, order, nil)
	if err != nil {
		t.Fatalf("EvalGroebner failed: %v", err)
	}

	list, ok := res.(*ListNode)
	if !ok {
		t.Fatalf("expected ListNode, got %T", res)
	}

	// Reduced Groebner basis should have 2 elements:
	// x - y and y^2 - 1/2
	if len(list.Elements) != 2 {
		t.Fatalf("expected 2 basis elements, got %d: %s", len(list.Elements), list.String())
	}

	t.Logf("CircleLine Groebner basis (lex): %s", list.String())
}

func TestEvalGroebner_HyperbolaEllipse(t *testing.T) {
	// x*y - 1 = 0, x^2 + y^2 - 4 = 0
	// Under lex order (x > y):
	// x = (4*y - y^3), y^4 - 4*y^2 + 1 = 0
	p1 := mustParseNode(t, "x*y - 1")
	p2 := mustParseNode(t, "x^2 + y^2 - 4")

	polys := NewList([]Node{p1, p2})
	vars := NewList([]Node{&VarNode{Name: "x"}, &VarNode{Name: "y"}})
	order := &VarNode{Name: "lex"}

	res, err := EvalGroebner(polys, vars, order, nil)
	if err != nil {
		t.Fatalf("EvalGroebner failed: %v", err)
	}

	list, ok := res.(*ListNode)
	if !ok {
		t.Fatalf("expected ListNode, got %T", res)
	}

	if len(list.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d: %s", len(list.Elements), list.String())
	}

	t.Logf("HyperbolaEllipse Groebner basis (lex): %s", list.String())
}

func TestEvalGroebner_InconsistentSystem(t *testing.T) {
	// x = 1, x = 2 -> [x - 1, x - 2] -> ideal is <1>
	p1 := mustParseNode(t, "x - 1")
	p2 := mustParseNode(t, "x - 2")

	polys := NewList([]Node{p1, p2})
	vars := NewList([]Node{&VarNode{Name: "x"}})

	res, err := EvalGroebner(polys, vars, nil, nil)
	if err != nil {
		t.Fatalf("EvalGroebner failed: %v", err)
	}

	list, ok := res.(*ListNode)
	if !ok {
		t.Fatalf("expected ListNode, got %T", res)
	}

	if len(list.Elements) != 1 || list.Elements[0].String() != "1" {
		t.Fatalf("expected [1] for inconsistent system, got %s", list.String())
	}
}

func TestEvalGroebner_ThreeVariables(t *testing.T) {
	// Katsura-like or simple 3-var system:
	// x + y + z - 1, x - y, z - 2
	p1 := mustParseNode(t, "x + y + z - 1")
	p2 := mustParseNode(t, "x - y")
	p3 := mustParseNode(t, "z - 2")

	polys := NewList([]Node{p1, p2, p3})
	vars := NewList([]Node{&VarNode{Name: "x"}, &VarNode{Name: "y"}, &VarNode{Name: "z"}})
	order := &VarNode{Name: "lex"}

	res, err := EvalGroebner(polys, vars, order, nil)
	if err != nil {
		t.Fatalf("EvalGroebner failed: %v", err)
	}

	list, ok := res.(*ListNode)
	if !ok {
		t.Fatalf("expected ListNode, got %T", res)
	}

	// Should completely solve linear system:
	// z = 2, y = -1/2, x = -1/2
	// Monic in lex: x + 1/2, y + 1/2, z - 2
	if len(list.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d: %s", len(list.Elements), list.String())
	}
	t.Logf("ThreeVariables solved Groebner basis: %s", list.String())
}

func mustParseNode(t *testing.T, s string) Node {
	t.Helper()
	node, err := Parse(s)
	if err != nil {
		t.Fatalf("Parse error on %q: %v", s, err)
	}
	return node
}

func TestEvalGroebner_ViaEval(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		{
			expr:     "groebner([x^2 + y^2 - 1, x - y], [x, y], lex)",
			expected: "[x + -1 * y, y^2 + -1/2]",
		},
		{
			expr:     "groebner([x - 1, x - 2], [x])",
			expected: "[1]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			node, err := Parse(tt.expr)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			res, err := Eval(node)
			if err != nil {
				t.Fatalf("Eval failed: %v", err)
			}
			t.Logf("%s => %s", tt.expr, res.String())
		})
	}
}

func TestGebauerMoller_PruningLogic(t *testing.T) {
	// Monomial tests for Gebauer-Möller
	// p1 = x^2, p2 = x*y, p3 = y^2
	// LCM(p1, p3) = x^2 * y^2.
	// LM(p2) = x*y divides LCM(p1, p3).
	// If (p1, p2) and (p2, p3) are processed, (p1, p3) must be pruned by Criterion 2.
	m1 := newMonomial([]int{2, 0}) // x^2
	m2 := newMonomial([]int{1, 1}) // x*y
	m3 := newMonomial([]int{0, 2}) // y^2

	p1 := &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: m1}}}
	p2 := &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: m2}}}
	p3 := &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: m3}}}

	G := []*MPoly{p1, p2, p3}

	// When both (0, 1) and (1, 2) are processed, (0, 2) must be pruned by Criterion 2 through p2.
	processed := map[uint64]bool{
		critPairKey(0, 1): true,
		critPairKey(1, 2): true,
	}
	if !canApplyChainCriterion(p1, p3, 0, 2, G, processed) {
		t.Errorf("expected canApplyChainCriterion(p1, p3) to return true through p2")
	}

	// If (0, 1) is NOT processed, canApplyChainCriterion should return false
	delete(processed, critPairKey(0, 1))
	if canApplyChainCriterion(p1, p3, 0, 2, G, processed) {
		t.Errorf("expected canApplyChainCriterion to return false when (0, 1) is not processed")
	}
}

func TestEvalGroebner_GebauerMollerCriterion2(t *testing.T) {
	// Classical example from Cox, Little, O'Shea Chapter 2 §9:
	// f1 = x^3 - 2*x*y, f2 = x^2*y - 2*y^2 + x
	// In grevlex order, this computation generates critical pairs where Criterion 2 prunes redundant S-pairs.
	expr := "groebner([x^3 - 2*x*y, x^2*y - 2*y^2 + x], [x, y], grevlex)"
	node, err := Parse(expr)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	res, err := Eval(node)
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}
	list, ok := res.(*ListNode)
	if !ok {
		t.Fatalf("expected ListNode, got %T", res)
	}
	// Reduced Groebner basis contains 3 elements: x^2, x*y, and y^2 - 1/2*x
	if len(list.Elements) != 3 {
		t.Fatalf("expected 3 elements in reduced basis, got %d: %s", len(list.Elements), list.String())
	}
	t.Logf("CLO Example Groebner basis: %s", list.String())

	// Another non-trivial 3-variable system:
	// x + y + z = 1, x^2 + y^2 + z^2 = 2, x^3 + y^3 + z^3 = 3
	expr3 := "groebner([x + y + z - 1, x^2 + y^2 + z^2 - 2, x^3 + y^3 + z^3 - 3], [x, y, z], lex)"
	node3, err := Parse(expr3)
	if err != nil {
		t.Fatalf("Parse 3-var failed: %v", err)
	}
	res3, err := Eval(node3)
	if err != nil {
		t.Fatalf("Eval 3-var failed: %v", err)
	}
	list3, ok := res3.(*ListNode)
	if !ok {
		t.Fatalf("expected ListNode, got %T", res3)
	}
	t.Logf("Symmetric 3-var Groebner basis: %s", list3.String())
	if len(list3.Elements) < 3 {
		t.Fatalf("expected at least 3 elements in 3-var Groebner basis, got %d", len(list3.Elements))
	}
}

func BenchmarkGroebner_ChainCriterion(b *testing.B) {
	node, err := Parse("groebner([x^2 + y^2 + z^2 - 1, x^2 + z^2 - y, x - z], [x, y, z], grevlex)")
	if err != nil {
		b.Fatalf("Parse error: %v", err)
	}

	for b.Loop() {
		_, err := Eval(node)
		if err != nil {
			b.Fatalf("Eval error: %v", err)
		}
	}
}

