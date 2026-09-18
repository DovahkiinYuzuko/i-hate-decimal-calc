package calc

import (
	"math/big"
	"testing"
)

func TestEllipticGroupArithmetic(t *testing.T) {
	// Curve: y^2 = x^3 + 1 (A = 0, B = 1)
	a := big.NewRat(0, 1)
	b := big.NewRat(1, 1)

	p := NewEllipticPoint(big.NewRat(2, 1), big.NewRat(3, 1))
	if !IsOnCurve(p, a, b) {
		t.Fatalf("Point (2, 3) must be on y^2 = x^3 + 1")
	}

	// 1. Identity laws
	inf := InfinityPoint()
	pPlusInf, err := AddEllipticPoints(p, inf, a, b)
	if err != nil || !pPlusInf.Equal(p) {
		t.Errorf("P + O != P: got %v, err %v", pPlusInf, err)
	}

	infPlusP, err := AddEllipticPoints(inf, p, a, b)
	if err != nil || !infPlusP.Equal(p) {
		t.Errorf("O + P != P: got %v, err %v", infPlusP, err)
	}

	// 2. Inverse
	negP := NewEllipticPoint(big.NewRat(2, 1), big.NewRat(-3, 1))
	pPlusNegP, err := AddEllipticPoints(p, negP, a, b)
	if err != nil || !pPlusNegP.IsInf {
		t.Errorf("P + (-P) != O: got %v, err %v", pPlusNegP, err)
	}

	// 3. Doubling: 2 * (2, 3) = (0, 1)
	twoP, err := AddEllipticPoints(p, p, a, b)
	if err != nil {
		t.Fatalf("AddEllipticPoints doubling error: %v", err)
	}
	expectedTwoP := NewEllipticPoint(big.NewRat(0, 1), big.NewRat(1, 1))
	if !twoP.Equal(expectedTwoP) {
		t.Errorf("2*(2, 3) expected (0, 1), got %v", twoP)
	}

	// 4. Scalar multiplication: 3 * (2, 3) = (-1, 0)
	threeP, err := ScalarMulEllipticPoint(p, big.NewInt(3), a, b)
	if err != nil {
		t.Fatalf("ScalarMul 3*P error: %v", err)
	}
	expectedThreeP := NewEllipticPoint(big.NewRat(-1, 1), big.NewRat(0, 1))
	if !threeP.Equal(expectedThreeP) {
		t.Errorf("3*(2, 3) expected (-1, 0), got %v", threeP)
	}

	// 5. Order 6 verification: 6 * (2, 3) = O
	sixP, err := ScalarMulEllipticPoint(p, big.NewInt(6), a, b)
	if err != nil {
		t.Fatalf("ScalarMul 6*P error: %v", err)
	}
	if !sixP.IsInf {
		t.Errorf("6*(2, 3) expected O, got %v", sixP)
	}
}

func TestEllipticFindTorsionSubgroup(t *testing.T) {
	tests := []struct {
		name          string
		a             int64
		b             int64
		expectedOrder int
		expectedGroup string
	}{
		{
			name:          "Klein four-group Z/2Z x Z/2Z on y^2 = x^3 - x",
			a:             -1,
			b:             0,
			expectedOrder: 4,
			expectedGroup: "Z/2Z x Z/2Z",
		},
		{
			name:          "Cyclic group Z/6Z on y^2 = x^3 + 1",
			a:             0,
			b:             1,
			expectedOrder: 6,
			expectedGroup: "Z/6Z",
		},
		{
			name:          "Cyclic group Z/2Z on y^2 = x^3 - 2",
			a:             0,
			b:             -2,
			expectedOrder: 1, // Only O (x^3 - 2 = 0 has no rational roots)
			expectedGroup: "Z/1Z",
		},
		{
			name:          "Cyclic group Z/4Z on y^2 = x^3 + 4x",
			a:             4,
			b:             0,
			expectedOrder: 4,
			expectedGroup: "Z/4Z",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			aRat := big.NewRat(tc.a, 1)
			bRat := big.NewRat(tc.b, 1)
			fsm := NewEllipticLifecycleFSM()

			points, groupStr, err := FindTorsionSubgroup(aRat, bRat, fsm)
			if err != nil {
				t.Fatalf("FindTorsionSubgroup failed: %v", err)
			}

			if len(points) != tc.expectedOrder {
				t.Errorf("expected order %d, got %d (points: %v)", tc.expectedOrder, len(points), points)
			}
			if groupStr != tc.expectedGroup {
				t.Errorf("expected group %q, got %q", tc.expectedGroup, groupStr)
			}

			// Validate FSM final state
			if fsm.CurrentState() != EcStateGroupDetermined {
				t.Errorf("expected FSM state %v, got %v", EcStateGroupDetermined, fsm.CurrentState())
			}
		})
	}
}

func TestEllipticSingularCurve(t *testing.T) {
	// y^2 = x^3 (A = 0, B = 0): cusp singularity (Delta = 0)
	aRat := big.NewRat(0, 1)
	bRat := big.NewRat(0, 1)
	fsm := NewEllipticLifecycleFSM()

	_, _, err := FindTorsionSubgroup(aRat, bRat, fsm)
	if err == nil {
		t.Fatalf("expected error on singular curve y^2 = x^3, got nil")
	}
	if fsm.CurrentState() != EcStateFailed {
		t.Errorf("expected FSM state EcStateFailed, got %v", fsm.CurrentState())
	}
}

func TestEllipticHandlersViaEval(t *testing.T) {
	env := NewEnv()

	// 1. ec_add
	resAdd, err := EvalStringWithEnv("ec_add(-1, 0, [-1, 0], [0, 0])", env)
	if err != nil {
		t.Fatalf("ec_add failed: %v", err)
	}
	listAdd, ok := resAdd.(*ListNode)
	if !ok || len(listAdd.Elements) != 2 {
		t.Fatalf("expected 2-element list, got %v", resAdd)
	}
	if listAdd.Elements[0].String() != "1" || listAdd.Elements[1].String() != "0" {
		t.Errorf("expected [1, 0], got %s", listAdd.String())
	}

	// 2. ec_mul
	resMul, err := EvalStringWithEnv("ec_mul(0, 1, 3, [2, 3])", env)
	if err != nil {
		t.Fatalf("ec_mul failed: %v", err)
	}
	listMul, ok := resMul.(*ListNode)
	if !ok || len(listMul.Elements) != 2 {
		t.Fatalf("expected 2-element list, got %v", resMul)
	}
	if listMul.Elements[0].String() != "-1" || listMul.Elements[1].String() != "0" {
		t.Errorf("expected [-1, 0], got %s", listMul.String())
	}

	// 3. ec_mul to Infinity O
	resMulInf, err := EvalStringWithEnv("ec_mul(0, 1, 6, [2, 3])", env)
	if err != nil {
		t.Fatalf("ec_mul to inf failed: %v", err)
	}
	if varInf, ok := resMulInf.(*VarNode); !ok || varInf.Name != "O" {
		t.Errorf("expected O, got %v", resMulInf)
	}

	// 4. ec_torsion
	resTorsion, err := EvalStringWithEnv("ec_torsion(0, 1)", env)
	if err != nil {
		t.Fatalf("ec_torsion failed: %v", err)
	}
	listTorsion, ok := resTorsion.(*ListNode)
	if !ok || len(listTorsion.Elements) != 2 {
		t.Fatalf("expected 2-element list [group, points], got %v", resTorsion)
	}
	if groupNode, ok := listTorsion.Elements[0].(*VarNode); !ok || groupNode.Name != "Z/6Z" {
		t.Errorf("expected group Z/6Z, got %v", listTorsion.Elements[0])
	}
	ptsList, ok := listTorsion.Elements[1].(*ListNode)
	if !ok || len(ptsList.Elements) != 6 {
		t.Errorf("expected 6 torsion points, got %v", listTorsion.Elements[1])
	}
}
