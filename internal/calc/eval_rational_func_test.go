package calc

import (
	"testing"
)

func evalAt(t *testing.T, expr Node, varName string, val int64) Node {
	t.Helper()
	env := NewEnv()
	env.Set(varName, mustRational(val, 1))
	res, err := EvalWithEnv(expr, env)
	if err != nil {
		t.Fatalf("evalAt %s=%d error: %v", varName, val, err)
	}
	return res
}

func TestPolyExtendedGCD(t *testing.T) {
	pA, _ := extractPoly(&AddNode{Terms: []Node{&VarNode{Name: "x"}, mustRational(-1, 1)}}, "x")
	pB, _ := extractPoly(&AddNode{Terms: []Node{&VarNode{Name: "x"}, mustRational(1, 1)}}, "x")

	S, T, G, ok := polyExtendedGCD(pA, pB)
	if !ok {
		t.Fatalf("polyExtendedGCD failed")
	}
	if G.degree() != 0 || isZero(G.coeffs[0]) {
		t.Errorf("expected degree 0 constant monic GCD (1), got %v", G.toNode().String())
	}

	// Verify S*A + T*B = G = 1
	sa := polyMul(S, pA)
	tb := polyMul(T, pB)
	sum := polySub(sa, polyScale(tb, mustRational(-1, 1))) // sa + tb
	sumNode, _ := Eval(sum.toNode())
	if !isOne(sumNode) {
		t.Errorf("Bézout identity failed: S*A + T*B = %v, expected 1", sumNode.String())
	}
}

func TestEvalApart_DistinctLinear(t *testing.T) {
	// apart(1 / (x^2 - 1)) = 1/(2*(x - 1)) - 1/(2*(x + 1))
	expr, err := ParseStatement("apart(1 / (x^2 - 1))")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	res, err := Eval(expr.(Node))
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	t.Logf("apart(1 / (x^2 - 1)) = %s", Format(res))

	// Check that substituting x = 2 gives 1/3 in both original and decomposed
	orig, _ := ParseStatement("1 / (x^2 - 1)")
	origVal := evalAt(t, orig.(Node), "x", 2)
	apartVal := evalAt(t, res, "x", 2)

	if origVal.String() != apartVal.String() {
		t.Errorf("value mismatch at x=2: original=%s, apart=%s", origVal.String(), apartVal.String())
	}
	if apartVal.String() != "1/3" {
		t.Errorf("expected 1/3 at x=2, got %s", apartVal.String())
	}
}

func TestEvalApart_ImproperFraction(t *testing.T) {
	// apart((x^2 + 1) / (x - 1)) = x + 1 + 2/(x - 1)
	expr, err := ParseStatement("apart((x^2 + 1) / (x - 1))")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	res, err := Eval(expr.(Node))
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	t.Logf("apart((x^2 + 1) / (x - 1)) = %s", Format(res))

	// Check x = 3: (9 + 1)/2 = 5. Decomposed: 3 + 1 + 2/2 = 5
	orig, _ := ParseStatement("(x^2 + 1) / (x - 1)")
	origVal := evalAt(t, orig.(Node), "x", 3)
	apartVal := evalAt(t, res, "x", 3)

	if origVal.String() != apartVal.String() {
		t.Errorf("value mismatch at x=3: original=%s, apart=%s", origVal.String(), apartVal.String())
	}
	if apartVal.String() != "5" {
		t.Errorf("expected 5 at x=3, got %s", apartVal.String())
	}
}

func TestEvalApart_RepeatedFactor(t *testing.T) {
	// apart(1 / (x * (x + 1)^2))
	expr, err := ParseStatement("apart(1 / (x * (x + 1)^2))")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	res, err := Eval(expr.(Node))
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	t.Logf("apart(1 / (x * (x + 1)^2)) = %s", Format(res))

	// Check x = 2: 1 / (2 * 9) = 1/18
	orig, _ := ParseStatement("1 / (x * (x + 1)^2)")
	origVal := evalAt(t, orig.(Node), "x", 2)
	apartVal := evalAt(t, res, "x", 2)

	if origVal.String() != apartVal.String() {
		t.Errorf("value mismatch at x=2: original=%s, apart=%s", origVal.String(), apartVal.String())
	}
	if apartVal.String() != "1/18" {
		t.Errorf("expected 1/18 at x=2, got %s", apartVal.String())
	}
}

func TestEvalApart_ThreeFactors(t *testing.T) {
	// apart(1 / ((x - 1) * (x - 2) * (x - 3)))
	expr, err := ParseStatement("apart(1 / ((x - 1) * (x - 2) * (x - 3)))")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	res, err := Eval(expr.(Node))
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	t.Logf("apart(1 / ((x - 1) * (x - 2) * (x - 3))) = %s", Format(res))

	// Check x = 4: 1 / (3 * 2 * 1) = 1/6
	orig, _ := ParseStatement("1 / ((x - 1) * (x - 2) * (x - 3))")
	origVal := evalAt(t, orig.(Node), "x", 4)
	apartVal := evalAt(t, res, "x", 4)

	if origVal.String() != apartVal.String() {
		t.Errorf("value mismatch at x=4: original=%s, apart=%s", origVal.String(), apartVal.String())
	}
	if apartVal.String() != "1/6" {
		t.Errorf("expected 1/6 at x=4, got %s", apartVal.String())
	}
}

func TestEvalTogether_Simple(t *testing.T) {
	// together(1/x + 1/(x + 1)) = (2*x + 1) / (x * (x + 1))
	expr, err := ParseStatement("together(1/x + 1/(x + 1))")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	res, err := Eval(expr.(Node))
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	t.Logf("together(1/x + 1/(x + 1)) = %s", Format(res))

	// Check x = 3: 1/3 + 1/4 = 7/12
	togetherVal := evalAt(t, res, "x", 3)
	if togetherVal.String() != "7/12" {
		t.Errorf("expected 7/12 at x=3, got %s", togetherVal.String())
	}
}

func TestEvalTogether_Subtraction(t *testing.T) {
	// together(1/(x - 1) - 1/(x + 1)) = 2 / ((x - 1)*(x + 1))
	expr, err := ParseStatement("together(1/(x - 1) - 1/(x + 1))")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	res, err := Eval(expr.(Node))
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	t.Logf("together(1/(x - 1) - 1/(x + 1)) = %s", Format(res))

	// Check x = 3: 1/2 - 1/4 = 1/4
	togetherVal := evalAt(t, res, "x", 3)
	if togetherVal.String() != "1/4" {
		t.Errorf("expected 1/4 at x=3, got %s", togetherVal.String())
	}
}
