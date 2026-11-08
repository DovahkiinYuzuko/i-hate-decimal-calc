package calc

import (
	"strings"
	"testing"
)

func TestIntegration_Issue162_REPL(t *testing.T) {
	env := NewEnv()

	// 1. isolate_roots_vas(x^2 - 2, x)
	expr1, err := Parse("isolate_roots_vas(x^2 - 2, x)")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	res1, err := EvalWithEnv(expr1, env)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	list1, ok1 := res1.(*ListNode)
	if !ok1 || len(list1.Elements) != 2 {
		t.Fatalf("expected 2 root intervals, got %v", res1)
	}

	// 2. alg_real([-2, 0, 1], [1, 2], 1)
	expr2, err := Parse("alg_real([-2, 0, 1], [1, 2], 1)")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	res2, err := EvalWithEnv(expr2, env)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	str2 := res2.String()
	if !strings.Contains(str2, "AlgReal") || !strings.Contains(str2, "x^2-2") {
		t.Errorf("expected AlgReal representation, got %s", str2)
	}

	// 3. is_constructible(x^3 - 2, x) -> false
	expr3, err := Parse("is_constructible(x^3 - 2, x)")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	res3, err := EvalWithEnv(expr3, env)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	v3, ok3 := res3.(*VarNode)
	if !ok3 || v3.Name != "false" {
		t.Errorf("expected false, got %v", res3)
	}

	// 4. is_constructible(x^4 - 2, x) -> true
	expr4, err := Parse("is_constructible(x^4 - 2, x)")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	res4, err := EvalWithEnv(expr4, env)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	v4, ok4 := res4.(*VarNode)
	if !ok4 || v4.Name != "true" {
		t.Errorf("expected true, got %v", res4)
	}

	// 5. is_constructible(x^4 - x - 1, x) -> false
	expr5, err := Parse("is_constructible(x^4 - x - 1, x)")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	res5, err := EvalWithEnv(expr5, env)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	v5, ok5 := res5.(*VarNode)
	if !ok5 || v5.Name != "false" {
		t.Errorf("expected false, got %v", res5)
	}

	// 6. is_constructible(x^8 - 2, x) -> undetermined
	expr6, err := Parse("is_constructible(x^8 - 2, x)")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	res6, err := EvalWithEnv(expr6, env)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	v6, ok6 := res6.(*VarNode)
	if !ok6 || v6.Name != "undetermined" {
		t.Errorf("expected undetermined, got %v", res6)
	}

	// 7. alg_real with rational coefficients: [-1/2, 0, 1] -> 2x^2 - 1 = 0
	expr7, err := Parse("alg_real([-1/2, 0, 1], [0, 1], 1)")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	res7, err := EvalWithEnv(expr7, env)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	str7 := res7.String()
	if !strings.Contains(str7, "AlgReal") || !strings.Contains(str7, "2x^2-1") {
		t.Errorf("expected scaled integer polynomial 2x^2-1, got %s", str7)
	}
}
