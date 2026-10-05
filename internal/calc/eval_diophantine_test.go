package calc

import (
	"math/big"
	"testing"
)

func TestPellSolve(t *testing.T) {
	tests := []struct {
		d        int64
		expectX  string
		expectY  string
		hasError bool
	}{
		{d: 2, expectX: "3", expectY: "2", hasError: false},
		{d: 3, expectX: "2", expectY: "1", hasError: false},
		{d: 5, expectX: "9", expectY: "4", hasError: false},
		{d: 7, expectX: "8", expectY: "3", hasError: false},
		{d: 13, expectX: "649", expectY: "180", hasError: false},
		{d: 61, expectX: "1766319049", expectY: "226153980", hasError: false},
		{d: 109, expectX: "158070671986249", expectY: "15140424455100", hasError: false},
		{d: 4, hasError: true},  // Square
		{d: 9, hasError: true},  // Square
		{d: -2, hasError: true}, // Negative
		{d: 0, hasError: true},  // Zero
	}

	for _, tt := range tests {
		x, y, err := PellSolve(big.NewInt(tt.d))
		if tt.hasError {
			if err == nil {
				t.Errorf("PellSolve(%d) expected error, got nil", tt.d)
			}
			continue
		}
		if err != nil {
			t.Fatalf("PellSolve(%d) unexpected error: %v", tt.d, err)
		}
		if x.String() != tt.expectX || y.String() != tt.expectY {
			t.Errorf("PellSolve(%d) = (%s, %s), expected (%s, %s)", tt.d, x.String(), y.String(), tt.expectX, tt.expectY)
		}

		// Mathematical verification: x^2 - d * y^2 == 1
		xSq := new(big.Int).Mul(x, x)
		ySq := new(big.Int).Mul(y, y)
		dySq := new(big.Int).Mul(big.NewInt(tt.d), ySq)
		diff := new(big.Int).Sub(xSq, dySq)
		if diff.Cmp(big.NewInt(1)) != 0 {
			t.Errorf("PellSolve(%d) mathematical identity failed: x^2 - d*y^2 = %s != 1", tt.d, diff.String())
		}
	}
}

func TestPellNegativeSolve(t *testing.T) {
	tests := []struct {
		d        int64
		expectX  string
		expectY  string
		hasError bool
	}{
		{d: 2, expectX: "1", expectY: "1", hasError: false},
		{d: 5, expectX: "2", expectY: "1", hasError: false},
		{d: 13, expectX: "18", expectY: "5", hasError: false},
		{d: 3, hasError: true},  // Unsolvable
		{d: 7, hasError: true},  // Unsolvable
		{d: 4, hasError: true},  // Square
	}

	for _, tt := range tests {
		x, y, err := PellNegativeSolve(big.NewInt(tt.d))
		if tt.hasError {
			if err == nil {
				t.Errorf("PellNegativeSolve(%d) expected error, got (%s, %s)", tt.d, x.String(), y.String())
			}
			continue
		}
		if err != nil {
			t.Fatalf("PellNegativeSolve(%d) unexpected error: %v", tt.d, err)
		}
		if x.String() != tt.expectX || y.String() != tt.expectY {
			t.Errorf("PellNegativeSolve(%d) = (%s, %s), expected (%s, %s)", tt.d, x.String(), y.String(), tt.expectX, tt.expectY)
		}

		// Mathematical verification: x^2 - d * y^2 == -1
		xSq := new(big.Int).Mul(x, x)
		ySq := new(big.Int).Mul(y, y)
		dySq := new(big.Int).Mul(big.NewInt(tt.d), ySq)
		diff := new(big.Int).Sub(xSq, dySq)
		if diff.Cmp(big.NewInt(-1)) != 0 {
			t.Errorf("PellNegativeSolve(%d) mathematical identity failed: x^2 - d*y^2 = %s != -1", tt.d, diff.String())
		}
	}
}

func TestSolveLinearDiophantineSingle(t *testing.T) {
	// 3*x + 5*y = 1
	xSol, ySol, err := SolveLinearDiophantineSingle(big.NewInt(3), big.NewInt(5), big.NewInt(1), "x", "y", "t")
	if err != nil {
		t.Fatalf("SolveLinearDiophantineSingle unexpected error: %v", err)
	}
	if xSol == nil || ySol == nil {
		t.Fatal("expected non-nil solutions")
	}

	// 2*x + 4*y = 3 (Insoluble: gcd(2, 4)=2 does not divide 3)
	_, _, err = SolveLinearDiophantineSingle(big.NewInt(2), big.NewInt(4), big.NewInt(3), "x", "y", "t")
	if err == nil {
		t.Error("expected error for insoluble linear diophantine equation, got nil")
	}
}

func TestEvalPellSolveAndDiophantineCAS(t *testing.T) {
	// pell_solve(61)
	res, err := EvalPellSolve([]Node{mustRational(61, 1)})
	if err != nil {
		t.Fatalf("EvalPellSolve(61) unexpected error: %v", err)
	}
	list, ok := res.(*ListNode)
	if !ok || len(list.Elements) != 2 {
		t.Fatalf("expected 2-element list, got %s", res.String())
	}
	if list.Elements[0].String() != "1766319049" || list.Elements[1].String() != "226153980" {
		t.Errorf("EvalPellSolve(61) = %s, expected [1766319049, 226153980]", res.String())
	}

	// solve_diophantine(x^2 - 2*y^2 == 1, [x, y])
	eqPell := &RelOpNode{
		Op: "==",
		LHS: &AddNode{Terms: []Node{
			&PowNode{Base: &VarNode{Name: "x"}, Exp: mustRational(2, 1)},
			&MulNode{Factors: []Node{mustRational(-2, 1), &PowNode{Base: &VarNode{Name: "y"}, Exp: mustRational(2, 1)}}},
		}},
		RHS: mustRational(1, 1),
	}
	varsPell := &ListNode{Elements: []Node{&VarNode{Name: "x"}, &VarNode{Name: "y"}}}
	solPell, err := EvalSolveDiophantine([]Node{eqPell, varsPell})
	if err != nil {
		t.Fatalf("EvalSolveDiophantine(Pell) unexpected error: %v", err)
	}
	solPellList, ok := solPell.(*ListNode)
	if !ok || len(solPellList.Elements) != 2 {
		t.Fatalf("expected 2-element list for Pell solution, got %s", solPell.String())
	}
	if solPellList.Elements[0].String() != "x == 3" || solPellList.Elements[1].String() != "y == 2" {
		t.Errorf("EvalSolveDiophantine(Pell) = %s, expected [x == 3, y == 2]", solPell.String())
	}

	// solve_diophantine(x^2 + y^2 == z^2, [x, y, z])
	eqPyth := &RelOpNode{
		Op: "==",
		LHS: &AddNode{Terms: []Node{
			&PowNode{Base: &VarNode{Name: "x"}, Exp: mustRational(2, 1)},
			&PowNode{Base: &VarNode{Name: "y"}, Exp: mustRational(2, 1)},
		}},
		RHS: &PowNode{Base: &VarNode{Name: "z"}, Exp: mustRational(2, 1)},
	}
	varsPyth := &ListNode{Elements: []Node{&VarNode{Name: "x"}, &VarNode{Name: "y"}, &VarNode{Name: "z"}}}
	solPyth, err := EvalSolveDiophantine([]Node{eqPyth, varsPyth})
	if err != nil {
		t.Fatalf("EvalSolveDiophantine(Pythagorean) unexpected error: %v", err)
	}
	if solPyth == nil {
		t.Fatal("expected non-nil pythagorean solution")
	}
}

func TestDiophantineFSMInvalidTransition(t *testing.T) {
	fsm := NewDiophantineSolverFSM()
	// Trying invalid direct transition INIT -> SUCCESS must fail
	err := fsm.Transition(DiophantineStateSuccess)
	if err == nil {
		t.Error("expected error for invalid transition INIT -> SUCCESS, got nil")
	}
	if fsm.Current() != DiophantineStateFailure {
		t.Errorf("expected state FAILURE after invalid transition, got %s", fsm.Current())
	}
}
