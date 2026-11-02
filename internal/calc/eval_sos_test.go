package calc

import (
	"math/big"
	"testing"
)

func TestDecomposeSOS_UnivariateSquare(t *testing.T) {
	// P(x) = x^2 + 2x + 1 = (x + 1)^2
	vars := []string{"x"}
	order := OrderGRevLex
	varMap := map[string]int{"x": 0}

	p, err := nodeToMPoly(mustParseNode(t, "x^2 + 2*x + 1"), varMap, 1, order)
	if err != nil {
		t.Fatalf("nodeToMPoly error: %v", err)
	}

	res, err := DecomposeSOS(p, vars, order)
	if err != nil {
		t.Fatalf("DecomposeSOS error: %v", err)
	}

	if res.Status != SOSCertified {
		t.Fatalf("expected SOSCertified, got %v", res.Status)
	}

	// Verify reconstruction: sum c_i * q_i^2 == P
	reconstructed := NewZeroMPoly(1)
	for _, term := range res.Terms {
		sq := term.Poly.Mul(term.Poly, order)
		sqScaled := sq.MulTerm(Term{Coeff: term.Weight, Mon: zeroMonomial(1)}, order)
		reconstructed = reconstructed.Add(sqScaled, order)
	}

	diff := p.Sub(reconstructed, order)
	if !diff.IsZero() {
		t.Fatalf("reconstruction mismatch: p - sum c_i*q_i^2 != 0, got %v", mpolyToNode(diff, vars))
	}
}

func TestDecomposeSOS_BivariateQuadratic(t *testing.T) {
	// P(x, y) = 2*x^2 + 2*x*y + 2*y^2
	// Expected: 2 * (x + 1/2*y)^2 + 3/2 * y^2
	vars := []string{"x", "y"}
	order := OrderGRevLex
	varMap := map[string]int{"x": 0, "y": 1}

	p, err := nodeToMPoly(mustParseNode(t, "2*x^2 + 2*x*y + 2*y^2"), varMap, 2, order)
	if err != nil {
		t.Fatalf("nodeToMPoly error: %v", err)
	}

	res, err := DecomposeSOS(p, vars, order)
	if err != nil {
		t.Fatalf("DecomposeSOS error: %v", err)
	}

	if res.Status != SOSCertified {
		t.Fatalf("expected SOSCertified, got %v", res.Status)
	}

	for _, term := range res.Terms {
		if term.Weight.Sign() <= 0 {
			t.Fatalf("expected positive weight, got %v", term.Weight)
		}
	}

	// Verify reconstruction
	reconstructed := NewZeroMPoly(2)
	for _, term := range res.Terms {
		sq := term.Poly.Mul(term.Poly, order)
		sqScaled := sq.MulTerm(Term{Coeff: term.Weight, Mon: zeroMonomial(2)}, order)
		reconstructed = reconstructed.Add(sqScaled, order)
	}

	diff := p.Sub(reconstructed, order)
	if !diff.IsZero() {
		t.Fatalf("reconstruction mismatch: p - sum c_i*q_i^2 != 0, got %v", mpolyToNode(diff, vars))
	}
}

func TestDecomposeSOS_Refuted(t *testing.T) {
	// P(x) = x^2 - 1 => P(0) = -1 < 0
	vars := []string{"x"}
	order := OrderGRevLex
	varMap := map[string]int{"x": 0}

	p, err := nodeToMPoly(mustParseNode(t, "x^2 - 1"), varMap, 1, order)
	if err != nil {
		t.Fatalf("nodeToMPoly error: %v", err)
	}

	res, err := DecomposeSOS(p, vars, order)
	if err != nil {
		t.Fatalf("DecomposeSOS error: %v", err)
	}

	if res.Status != SOSRefuted {
		t.Fatalf("expected SOSRefuted, got %v", res.Status)
	}
	if len(res.Counterexample) == 0 {
		t.Fatalf("expected non-empty counterexample point")
	}
}

func TestDecomposeSOS_MotzkinPolynomial(t *testing.T) {
	// Motzkin polynomial: M(x, y) = x^4*y^2 + x^2*y^4 - 3*x^2*y^2 + 1 >= 0 on R, but NOT a sum of squares!
	vars := []string{"x", "y"}
	order := OrderGRevLex
	varMap := map[string]int{"x": 0, "y": 1}

	p, err := nodeToMPoly(mustParseNode(t, "x^4*y^2 + x^2*y^4 - 3*x^2*y^2 + 1"), varMap, 2, order)
	if err != nil {
		t.Fatalf("nodeToMPoly error: %v", err)
	}

	res, err := DecomposeSOS(p, vars, order)
	if err != nil {
		t.Fatalf("DecomposeSOS error: %v", err)
	}

	// Must NOT be falsely certified!
	if res.Status == SOSCertified {
		t.Fatalf("Motzkin polynomial must not be certified as SOS!")
	}
	if res.Status != SOSNoCertificateFound {
		t.Fatalf("expected SOSNoCertificateFound, got %v", res.Status)
	}
}


func TestEvalSOSDecompose_Builtin(t *testing.T) {
	env := NewEnv()
	node, err := Parse("sos_decompose(x^2 + 2*x + 1, [x])")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	res, err := EvalWithEnv(node, env)
	if err != nil {
		t.Fatalf("EvalWithEnv error: %v", err)
	}

	listNode, ok := res.(*ListNode)
	if !ok || len(listNode.Elements) != 2 {
		t.Fatalf("expected 2-element list result, got: %s", Format(res))
	}
	statusVar, ok := listNode.Elements[0].(*VarNode)
	if !ok || statusVar.Name != "certified" {
		t.Fatalf("expected certified, got: %s", Format(listNode.Elements[0]))
	}
}

func TestDecomposeSOS_OddDegreeCauchyRefutation(t *testing.T) {
	// P(x) = x^3 + 10000 (degree 3, odd)
	// Must mathematically prove a negative witness where P(w) < 0, NOT a fake witness like -10 (which gives +9000).
	p := &MPoly{
		Terms: []Term{
			{Coeff: big.NewRat(1, 1), Mon: Monomial{Exps: []int{3}, TotalDeg: 3}},
			{Coeff: big.NewRat(10000, 1), Mon: Monomial{Exps: []int{0}, TotalDeg: 0}},
		},
		Sugar: 3,
	}
	order := OrderGRevLex
	res, err := DecomposeSOS(p, []string{"x"}, order)
	if err != nil {
		t.Fatalf("DecomposeSOS error: %v", err)
	}
	if res.Status != SOSRefuted {
		t.Fatalf("expected SOSRefuted, got %v", res.Status)
	}
	if len(res.Counterexample) == 0 {
		t.Fatalf("expected counterexample witness point")
	}
	// Verify that P(w) is strictly negative
	pt := []*big.Rat{res.Counterexample["x"]}
	val := evaluateAt(p, pt)
	if val.Sign() >= 0 {
		t.Fatalf("counterexample %v is invalid! P(w) = %s >= 0", res.Counterexample, val.String())
	}
}

func TestDecomposeSOS_SymmetricAveraging(t *testing.T) {
	// P(x, y) = x^4 + 2*x^2*y^2 + y^4 = (x^2 + y^2)^2
	// Monomial basis includes [x^2, x*y, y^2].
	// The cross term x^2 * y^2 can be formed by (x^2) * (y^2) and (x*y) * (x*y).
	// Averaging and LDL^T must successfully decompose it into a valid SOS without double-subtraction.
	p := &MPoly{
		Terms: []Term{
			{Coeff: big.NewRat(1, 1), Mon: Monomial{Exps: []int{4, 0}, TotalDeg: 4}},
			{Coeff: big.NewRat(2, 1), Mon: Monomial{Exps: []int{2, 2}, TotalDeg: 4}},
			{Coeff: big.NewRat(1, 1), Mon: Monomial{Exps: []int{0, 4}, TotalDeg: 4}},
		},
		Sugar: 4,
	}
	order := OrderGRevLex
	res, err := DecomposeSOS(p, []string{"x", "y"}, order)
	if err != nil {
		t.Fatalf("DecomposeSOS error: %v", err)
	}
	if res.Status != SOSCertified {
		t.Fatalf("expected SOSCertified for (x^2 + y^2)^2, got %v", res.Status)
	}
}
