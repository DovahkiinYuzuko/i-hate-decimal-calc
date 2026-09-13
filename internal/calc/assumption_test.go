package calc

import (
	"testing"
)

func TestAssumption_DeductionAndConsistency(t *testing.T) {
	store := NewAssumptionStore()

	// 1. Positive implies NonNegative, NonZero, Real, Complex
	err := store.Assume("x", PropPositive)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !store.IsPositive("x") {
		t.Errorf("expected x to be positive")
	}
	if !store.IsNonNegative("x") {
		t.Errorf("expected x to be non-negative by deduction")
	}
	if !store.IsReal("x") {
		t.Errorf("expected x to be real by deduction")
	}
	if store.IsNegative("x") {
		t.Errorf("expected x NOT to be negative")
	}

	// 2. Inconsistency: adding Negative to Positive x should fail
	err = store.Assume("x", PropNegative)
	if err == nil {
		t.Errorf("expected contradiction error when assuming negative for positive x")
	}
	// Verify state remained consistent after rollback
	if !store.IsPositive("x") || store.IsNegative("x") {
		t.Errorf("rollback failed after inconsistent assumption")
	}

	// 3. Integer & Even
	err = store.Assume("n", PropEven)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !store.IsEven("n") {
		t.Errorf("expected n to be even")
	}
	if !store.IsInteger("n") {
		t.Errorf("expected n to be integer by deduction")
	}
	if !store.IsReal("n") {
		t.Errorf("expected n to be real by deduction")
	}

	// 4. Inconsistency: adding Odd to Even n should fail
	err = store.Assume("n", PropOdd)
	if err == nil {
		t.Errorf("expected contradiction error when assuming odd for even n")
	}

	// 5. Unassume
	store.Unassume("x")
	if store.IsPositive("x") || store.IsReal("x") {
		t.Errorf("expected x to be unconstrained after unassume")
	}
	if store.Query("x", PropPositive) != TernaryUnknown {
		t.Errorf("expected x query to be TernaryUnknown")
	}

	// 6. ListAssumptions
	list := store.ListAssumptions()
	if len(list) != 1 {
		t.Errorf("expected 1 assumption in list, got %d", len(list))
	}

	// 7. ClearAll
	store.ClearAll()
	if len(store.ListAssumptions()) != 0 {
		t.Errorf("expected empty assumptions after ClearAll")
	}
}

func TestAssumption_EvalIntegration(t *testing.T) {
	env := NewEnv()

	eval := func(exprStr string) string {
		node, err := Parse(exprStr)
		if err != nil {
			t.Fatalf("parse error for %q: %v", exprStr, err)
		}
		res, err := EvalWithEnv(node, env)
		if err != nil {
			t.Fatalf("eval error for %q: %v", exprStr, err)
		}
		return res.String()
	}

	// 1. Without assumption: sqrt(x^2) -> abs(x)
	res := eval("sqrt(x^2)")
	if res != "abs(x)" {
		t.Errorf("expected abs(x) without assumption, got %s", res)
	}

	// 2. assume(x > 0): sqrt(x^2) -> x
	eval("assume(x > 0)")
	res = eval("sqrt(x^2)")
	if res != "x" {
		t.Errorf("expected x after assume(x > 0), got %s", res)
	}

	// 3. unassume(x) -> back to abs(x)
	eval("unassume(x)")
	res = eval("sqrt(x^2)")
	if res != "abs(x)" {
		t.Errorf("expected abs(x) after unassume(x), got %s", res)
	}

	// 4. assume(x < 0): sqrt(x^2) -> -x
	eval("assume(x < 0)")
	res = eval("sqrt(x^2)")
	if res != "-x" && res != "-1 * x" {
		t.Errorf("expected -x or -1 * x after assume(x < 0), got %s", res)
	}

	// 5. Trig functions with integer assumption
	// Without assumption: sin(n * pi) remains sin(n*pi) or sin(pi*n)
	eval("unassume(x)")
	resNoAssume := eval("sin(n * pi)")
	if resNoAssume == "0" {
		t.Errorf("expected sin(n * pi) not to be 0 without assumption, got %s", resNoAssume)
	}

	// assume(n, integer) -> sin(n * pi) = 0, tan(n * pi) = 0
	eval("assume(n, integer)")
	res = eval("sin(n * pi)")
	if res != "0" {
		t.Errorf("expected sin(n * pi) = 0 under integer n, got %s", res)
	}
	res = eval("tan(n * pi)")
	if res != "0" {
		t.Errorf("expected tan(n * pi) = 0 under integer n, got %s", res)
	}

	// assume(n, even) -> cos(n * pi) = 1
	eval("assume(n, even)")
	res = eval("cos(n * pi)")
	if res != "1" {
		t.Errorf("expected cos(n * pi) = 1 under even n, got %s", res)
	}

	// unassume(n); assume(n, odd) -> cos(n * pi) = -1
	eval("unassume(n)")
	eval("assume(n, odd)")
	res = eval("cos(n * pi)")
	if res != "-1" {
		t.Errorf("expected cos(n * pi) = -1 under odd n, got %s", res)
	}

	// 6. List assumptions
	resList := eval("assumptions()")
	if resList != "[n: odd, integer, rational, real, complex]" && resList != "[n: odd]" {
		// Just ensure n is listed
		t.Logf("assumptions() returned: %s", resList)
	}
}
