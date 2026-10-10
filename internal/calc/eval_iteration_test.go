package calc

import (
	"strings"
	"testing"
)

func TestEvalIteration_Table_Basic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"table(k^2, k, 1, 5)", "[1, 4, 9, 16, 25]"},
		{"table(k, k, 1, 9, 2)", "[1, 3, 5, 7, 9]"},
		{"table(k, k, 5, 1, -1)", "[5, 4, 3, 2, 1]"},
		{"table(x, x, 1/3, 5/3, 1/3)", "[1/3, 2/3, 1, 4/3, 5/3]"},
		{"table(k, k, 1, 5, -1)", "[]"},
		{"table(table(m * j, j, 1, 3), m, 1, 3)", "[[1, 2, 3], [2, 4, 6], [3, 6, 9]]"},
		{"table(i * k, k, 1, 3)", "[0 + 1*i, 0 + 2*i, 0 + 3*i]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			if node.String() != tt.expected {
				t.Errorf("input %s: got %s, expected %s", tt.input, node.String(), tt.expected)
			}
		})
	}
}

func TestEvalIteration_Table_RationalEps(t *testing.T) {
	// 0 to 1 with step 1/10 -> exactly 11 elements
	node, err := EvalString("table(x, x, 0, 1, 1/10)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := node.(*ListNode)
	if !ok {
		t.Fatalf("expected *ListNode, got %T", node)
	}
	if len(list.Elements) != 11 {
		t.Fatalf("expected 11 elements, got %d: %s", len(list.Elements), list.String())
	}
	if list.Elements[10].String() != "1" {
		t.Errorf("expected final element to be 1, got %s", list.Elements[10].String())
	}
}

func TestEvalIteration_Table_ZeroStepError(t *testing.T) {
	_, err := EvalString("table(k, k, 1, 5, 0)")
	if err == nil {
		t.Fatalf("expected error for step=0, got nil")
	}
	if !strings.Contains(err.Error(), "0") && !strings.Contains(err.Error(), "zero") {
		t.Errorf("expected zero step error message, got %v", err)
	}
}

func TestEvalIteration_ReservedImaginaryUnitError(t *testing.T) {
	_, err := EvalString("table(i^2, i, 1, 5)")
	if err == nil {
		t.Fatalf("expected error when using 'i' as loop variable, got nil")
	}
	if !strings.Contains(err.Error(), "imaginary unit") {
		t.Errorf("expected helpful imaginary unit error note, got %v", err)
	}
}

func TestEvalIteration_Table_ScopingAndRollback(t *testing.T) {
	env := NewEnv()
	env.Set("k", mustRational(99, 1))

	// Outer variable preservation
	res, err := EvalStringWithEnv("table(k^2, k, 1, 3)", env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.String() != "[1, 4, 9]" {
		t.Errorf("got %s, expected [1, 4, 9]", res.String())
	}
	val, ok := env.Get("k")
	if !ok || val.String() != "99" {
		t.Errorf("outer variable k was modified or lost, got %v", val)
	}

	// Error rollback: division by zero in loop
	env.Set("n", mustRational(777, 1))
	_, err = EvalStringWithEnv("table(1 / (n - 2), n, 1, 3)", env)
	if err == nil {
		t.Fatalf("expected division by zero error, got nil")
	}
	valN, okN := env.Get("n")
	if !okN || valN.String() != "777" {
		t.Errorf("outer variable n was not restored on error, got %v", valN)
	}

	// Unbound variable shouldn't linger in env
	envFresh := NewEnv()
	_, err = EvalStringWithEnv("table(x, x, 1, 3)", envFresh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, exists := envFresh.Get("x"); exists {
		t.Errorf("unbound variable x lingered in env after table evaluation")
	}
}

func TestEvalIteration_Table_SameNameNested(t *testing.T) {
	// Outer m is 10, inner m runs 1..2
	node, err := EvalString("table(table(m + j, m, 1, 2), j, 10, 10)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.String() != "[[11, 12]]" {
		t.Errorf("got %s, expected [[11, 12]]", node.String())
	}
}

func TestEvalIteration_Product_Basic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"product(k, k, 1, 5)", "120"},
		{"product(2*k, k, 1, 3)", "48"},
		{"product(k, k, -100, 100)", "0"},
		{"product(k / (k + 1), k, 1, 100)", "1/101"},
		{"product(-1, k, 1, 101)", "-1"},
		{"product(-1, k, 1, 100)", "1"},
		{"product(k, k, 5, 1, 1)", "1"}, // empty set returns 1
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			if node.String() != tt.expected {
				t.Errorf("input %s: got %s, expected %s", tt.input, node.String(), tt.expected)
			}
		})
	}
}

func TestEvalIteration_Product_ShortCircuit(t *testing.T) {
	// Even though range is large, 0 is encountered early
	node, err := EvalString("product(k, k, -5, 5000)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.String() != "0" {
		t.Errorf("expected 0 from short-circuit, got %s", node.String())
	}
}

func TestEvalIteration_For_AccumulationAndSideEffects(t *testing.T) {
	env := NewEnv()
	env.Set("s", mustRational(0, 1))

	// s = 0; for(k, 1, 10, s = s + k); s
	res, err := EvalStringWithEnv("for(k, 1, 10, s = s + k)", env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.String() != "55" {
		t.Errorf("for returned %s, expected 55", res.String())
	}

	sVal, ok := env.Get("s")
	if !ok || sVal.String() != "55" {
		t.Errorf("env s got %v, expected 55", sVal)
	}

	// Step specified: for(k, 1, 9, 2, s = s + k)
	env.Set("s", mustRational(0, 1))
	res2, err := EvalStringWithEnv("for(k, 1, 9, 2, s = s + k)", env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.String() != "25" {
		t.Errorf("for step=2 returned %s, expected 25", res2.String())
	}
}

func TestEvalIteration_For_OuterVariableProtected(t *testing.T) {
	env := NewEnv()
	env.Set("k", mustRational(42, 1))
	env.Set("s", mustRational(0, 1))

	_, err := EvalStringWithEnv("for(k, 1, 5, s = s + k)", env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	kVal, ok := env.Get("k")
	if !ok || kVal.String() != "42" {
		t.Errorf("outer variable k was corrupted: got %v, expected 42", kVal)
	}

	sVal, ok := env.Get("s")
	if !ok || sVal.String() != "15" {
		t.Errorf("accumulator s got %v, expected 15", sVal)
	}
}

func TestEvalIteration_For_EmptyLoop(t *testing.T) {
	env := NewEnv()
	res, err := EvalStringWithEnv("for(k, 5, 1, 1, 100)", env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.String() != "0" {
		t.Errorf("empty for loop expected 0, got %s", res.String())
	}
}

func TestEvalIteration_For_StressFastAccumulation(t *testing.T) {
	env := NewEnv()
	env.Set("c", mustRational(0, 1))
	res, err := EvalStringWithEnv("for(k, 1, 10000, c = c + 1)", env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.String() != "10000" {
		t.Errorf("stress for loop expected 10000, got %s", res.String())
	}
}
