package calc

import (
	"math/big"
	"testing"
)

func TestLLLFSM_LifecycleTransitions(t *testing.T) {
	fsm := NewLLLLifecycleFSM()
	if fsm.CurrentState() != LLLStateInit {
		t.Fatalf("expected initial state Init, got %v", fsm.CurrentState())
	}

	// Valid step-by-step transitions
	if err := fsm.TransitionTo(LLLStateGramSchmidtComputed); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fsm.TransitionTo(LLLStateSizeReduced); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fsm.TransitionTo(LLLStateLovaszTested); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fsm.TransitionTo(LLLStateVectorSwapped); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fsm.TransitionTo(LLLStateGramSchmidtComputed); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fsm.TransitionTo(LLLStateBasisReduced); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fsm.TransitionTo(LLLStateRelationIdentified); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Invalid transition from RelationIdentified
	if err := fsm.TransitionTo(LLLStateInit); err == nil {
		t.Fatalf("expected error for invalid transition to Init, got nil")
	}

	history := fsm.History()
	if len(history) < 7 {
		t.Fatalf("expected history length >= 7, got %d", len(history))
	}
}

func TestExactLLLReduction_SmallLattice(t *testing.T) {
	// Standard 3D lattice test
	// Basis:
	// b1 = [1, -1, 3]
	// b2 = [1, 0, 5]
	// b3 = [1, 2, 6]
	basis := [][]*big.Int{
		{big.NewInt(1), big.NewInt(-1), big.NewInt(3)},
		{big.NewInt(1), big.NewInt(0), big.NewInt(5)},
		{big.NewInt(1), big.NewInt(2), big.NewInt(6)},
	}

	res, err := ExactLLLReduction(basis, big.NewRat(3, 4))
	if err != nil {
		t.Fatalf("LLL reduction failed: %v", err)
	}

	if res.FSM.CurrentState() != LLLStateBasisReduced {
		t.Fatalf("expected FSM state BasisReduced, got %v", res.FSM.CurrentState())
	}

	// Verify that the reduced basis vectors are non-zero
	for i, row := range res.ReducedBasis {
		norm := vectorDotBigInt(row, row)
		if norm.Sign() == 0 {
			t.Fatalf("reduced basis row %d is zero vector", i)
		}
	}

	// Verify that Gram-Schmidt coefficients satisfy size reduction |mu_{i,j}| <= 1/2
	half := big.NewRat(1, 2)
	negHalf := big.NewRat(-1, 2)
	for i := 0; i < len(res.GramSchmidtCoeffs); i++ {
		for j := 0; j < i; j++ {
			mu := res.GramSchmidtCoeffs[i][j]
			if mu.Cmp(half) > 0 || mu.Cmp(negHalf) < 0 {
				t.Errorf("size reduction violated: mu[%d][%d] = %s", i, j, mu.RatString())
			}
		}
	}
}

func TestFindIntegerRelation_Simple(t *testing.T) {
	// Simple integer linear relation: 2 * (1/2) + 3 * (1/3) - 2 * (1) = 1 + 1 - 2 = 0
	values := []*big.Rat{
		big.NewRat(1, 2),
		big.NewRat(1, 3),
		big.NewRat(1, 1),
	}

	coeffs, err := FindIntegerRelation(values, 64)
	if err != nil {
		t.Fatalf("FindIntegerRelation failed: %v", err)
	}

	// Check relation: sum c_i * v_i == 0
	sum := big.NewRat(0, 1)
	for i := 0; i < len(values); i++ {
		term := new(big.Rat).Mul(new(big.Rat).SetInt(coeffs[i]), values[i])
		sum.Add(sum, term)
	}
	if sum.Sign() != 0 {
		t.Fatalf("relation not satisfied: sum = %s", sum.RatString())
	}
}

func TestFindMinimalPolynomial_AlgebraicNumbers(t *testing.T) {
	env := NewEnv()

	// Test 1: sqrt(2) -> x^2 - 2
	sqrt2 := &PowNode{
		Base: &RationalNode{Val: big.NewRat(2, 1)},
		Exp:  &RationalNode{Val: big.NewRat(1, 2)},
	}

	p1, err := FindMinimalPolynomial(sqrt2, 2, env)
	if err != nil {
		t.Fatalf("FindMinimalPolynomial for sqrt(2) failed: %v", err)
	}
	p1Str := p1.String()
	t.Logf("sqrt(2) min poly: %s", p1Str)

	// Test 2: sqrt(2) + 1 -> x^2 - 2*x - 1
	sqrt2Plus1 := &AddNode{
		Terms: []Node{
			sqrt2,
			&RationalNode{Val: big.NewRat(1, 1)},
		},
	}
	p2, err := FindMinimalPolynomial(sqrt2Plus1, 2, env)
	if err != nil {
		t.Fatalf("FindMinimalPolynomial for sqrt(2) + 1 failed: %v", err)
	}
	p2Str := p2.String()
	t.Logf("sqrt(2) + 1 min poly: %s", p2Str)

	// Test 3: sqrt(2) + sqrt(3) -> degree 4: x^4 - 10*x^2 + 1
	sqrt3 := &PowNode{
		Base: &RationalNode{Val: big.NewRat(3, 1)},
		Exp:  &RationalNode{Val: big.NewRat(1, 2)},
	}
	sqrt2Plus3 := &AddNode{
		Terms: []Node{sqrt2, sqrt3},
	}
	p3, err := FindMinimalPolynomial(sqrt2Plus3, 4, env)
	if err != nil {
		t.Fatalf("FindMinimalPolynomial for sqrt(2) + sqrt(3) failed: %v", err)
	}
	p3Str := p3.String()
	t.Logf("sqrt(2) + sqrt(3) min poly: %s", p3Str)
}

func TestHandleLLL_Builtin(t *testing.T) {
	env := NewEnv()

	// Test lll([[1, 2], [3, 4]])
	mat := &MatrixNode{
		Rows: 2,
		Cols: 2,
		Data: [][]Node{
			{&RationalNode{Val: big.NewRat(1, 1)}, &RationalNode{Val: big.NewRat(2, 1)}},
			{&RationalNode{Val: big.NewRat(3, 1)}, &RationalNode{Val: big.NewRat(4, 1)}},
		},
	}

	res, err := HandleLLL([]Node{mat}, env)
	if err != nil {
		t.Fatalf("HandleLLL failed: %v", err)
	}

	matRes, ok := res.(*MatrixNode)
	if !ok {
		t.Fatalf("expected MatrixNode result, got %T", res)
	}
	if matRes.Rows != 2 || matRes.Cols != 2 {
		t.Fatalf("expected 2x2 matrix, got %dx%d", matRes.Rows, matRes.Cols)
	}
}
