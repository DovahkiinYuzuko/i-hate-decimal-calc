package calc

import (
	"math/big"
	"strings"
	"testing"
)

func TestRSolve_FirstOrder_Geometric(t *testing.T) {
	// a(n+1) == 3*a(n), a(0) == 2 => 2 * 3^n
	res, err := EvalString("rsolve(a(n+1) == 3*a(n), a(n), [a(0) == 2])")
	if err != nil {
		t.Fatalf("rsolve failed: %v", err)
	}

	// Verify at n=0, 1, 2
	env := NewEnv()
	env.Set("n", &RationalNode{Val: big.NewRat(0, 1)})
	v0, _ := EvalWithEnv(res, env)
	if r, ok := v0.(*RationalNode); !ok || r.Val.Cmp(big.NewRat(2, 1)) != 0 {
		t.Errorf("at n=0 expected 2, got %s", v0.String())
	}

	env.Set("n", &RationalNode{Val: big.NewRat(2, 1)})
	v2, _ := EvalWithEnv(res, env)
	if r, ok := v2.(*RationalNode); !ok || r.Val.Cmp(big.NewRat(18, 1)) != 0 {
		t.Errorf("at n=2 expected 18, got %s", v2.String())
	}
}

func TestRSolve_FirstOrder_Hanoi(t *testing.T) {
	// a(n+1) == 2*a(n) + 1, a(0) == 0 => 2^n - 1
	res, err := EvalString("rsolve(a(n+1) == 2*a(n) + 1, a(n), [a(0) == 0])")
	if err != nil {
		t.Fatalf("rsolve failed: %v", err)
	}

	env := NewEnv()
	// n=3 => 2^3 - 1 = 7
	env.Set("n", &RationalNode{Val: big.NewRat(3, 1)})
	v3, _ := EvalWithEnv(res, env)
	if r, ok := v3.(*RationalNode); !ok || r.Val.Cmp(big.NewRat(7, 1)) != 0 {
		t.Errorf("at n=3 expected 7, got %s", v3.String())
	}
}

func TestRSolve_FirstOrder_Polynomial(t *testing.T) {
	// a(n+1) == a(n) + n, a(0) == 1 => 1 + n*(n-1)/2
	res, err := EvalString("rsolve(a(n+1) == a(n) + n, a(n), [a(0) == 1])")
	if err != nil {
		t.Fatalf("rsolve failed: %v", err)
	}

	env := NewEnv()
	testValues := map[int64]int64{0: 1, 1: 1, 2: 2, 3: 4, 4: 7}
	for nVal, expected := range testValues {
		env.Set("n", &RationalNode{Val: big.NewRat(nVal, 1)})
		vn, _ := EvalWithEnv(res, env)
		if r, ok := vn.(*RationalNode); !ok || r.Val.Cmp(big.NewRat(expected, 1)) != 0 {
			t.Errorf("at n=%d expected %d, got %s", nVal, expected, vn.String())
		}
	}
}

func TestRSolve_SecondOrder_DistinctRationalRoots(t *testing.T) {
	// a(n+2) - 5*a(n+1) + 6*a(n) == 0, a(0) == 2, a(1) == 5
	// roots are 2 and 3 => a(n) = 2^n + 3^n
	res, err := EvalString("rsolve(a(n+2) - 5*a(n+1) + 6*a(n) == 0, a(n), [a(0) == 2, a(1) == 5])")
	if err != nil {
		t.Fatalf("rsolve failed: %v", err)
	}

	env := NewEnv()
	// n=2 => 2^2 + 3^2 = 4 + 9 = 13
	env.Set("n", &RationalNode{Val: big.NewRat(2, 1)})
	v2, _ := EvalWithEnv(res, env)
	if r, ok := v2.(*RationalNode); !ok || r.Val.Cmp(big.NewRat(13, 1)) != 0 {
		t.Errorf("at n=2 expected 13, got %s", v2.String())
	}
}

func TestRSolve_SecondOrder_DoubleRoot(t *testing.T) {
	// a(n+2) - 4*a(n+1) + 4*a(n) == 0, a(0) == 1, a(1) == 4
	// root is 2 (multiplicity 2) => a(n) = (1 + n) * 2^n
	res, err := EvalString("rsolve(a(n+2) - 4*a(n+1) + 4*a(n) == 0, a(n), [a(0) == 1, a(1) == 4])")
	if err != nil {
		t.Fatalf("rsolve failed: %v", err)
	}

	env := NewEnv()
	// n=2 => (1 + 2) * 2^2 = 3 * 4 = 12
	env.Set("n", &RationalNode{Val: big.NewRat(2, 1)})
	v2, _ := EvalWithEnv(res, env)
	if r, ok := v2.(*RationalNode); !ok || r.Val.Cmp(big.NewRat(12, 1)) != 0 {
		t.Errorf("at n=2 expected 12, got %s", v2.String())
	}
}

func TestRSolve_Fibonacci(t *testing.T) {
	// a(n+2) == a(n+1) + a(n), a(0) == 0, a(1) == 1
	res, err := EvalString("rsolve(a(n+2) == a(n+1) + a(n), a(n), [a(0) == 0, a(1) == 1])")
	if err != nil {
		t.Fatalf("rsolve failed: %v", err)
	}

	expectedFibs := []int64{0, 1, 1, 2, 3, 5, 8, 13}
	env := NewEnv()
	for nVal, expected := range expectedFibs {
		env.Set("n", &RationalNode{Val: big.NewRat(int64(nVal), 1)})
		vn, err := EvalWithEnv(res, env)
		if err != nil {
			t.Fatalf("eval at n=%d failed: %v", nVal, err)
		}
		if r, ok := vn.(*RationalNode); ok {
			if r.Val.Cmp(big.NewRat(expected, 1)) != 0 {
				t.Errorf("F_%d expected %d, got %s", nVal, expected, r.String())
			}
		} else {
			approxStr, err := Approx(vn)
			if err == nil {
				if !strings.HasPrefix(approxStr, big.NewInt(expected).String()) {
					t.Errorf("F_%d approx mismatch: got %s, expected %d", nVal, approxStr, expected)
				}
			}
		}
	}
}

func TestRSolve_GeneralSolution_SymbolicConstants(t *testing.T) {
	// a(n+1) == 2*a(n) without inits => C1 * 2^n
	res, err := EvalString("rsolve(a(n+1) == 2*a(n), a(n))")
	if err != nil {
		t.Fatalf("rsolve failed: %v", err)
	}

	if !containsVarName(res, "C1") {
		t.Errorf("expected C1 in general solution, got: %s", res.String())
	}
}

func containsVarName(n Node, name string) bool {
	found := false
	var walk func(curr Node)
	walk = func(curr Node) {
		if curr == nil || found {
			return
		}
		if v, ok := curr.(*VarNode); ok && v.Name == name {
			found = true
			return
		}
		switch val := curr.(type) {
		case *AddNode:
			for _, t := range val.Terms {
				walk(t)
			}
		case *MulNode:
			for _, f := range val.Factors {
				walk(f)
			}
		case *PowNode:
			walk(val.Base)
			walk(val.Exp)
		}
	}
	walk(n)
	return found
}

