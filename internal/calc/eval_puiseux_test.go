package calc

import (
	"strings"
	"testing"
)

func TestEvalPuiseux_Cusp(t *testing.T) {
	// y^2 - x^3 == 0  => y = x^(3/2), -x^(3/2)
	res, err := EvalString("puiseux(y^2 - x^3, y, x, 3)")
	if err != nil {
		t.Fatalf("unexpected error for cusp: %v", err)
	}

	str := Format(res)
	if !strings.Contains(str, "x^3/2") {
		t.Fatalf("expected x^3/2 in result, got %s", str)
	}
}

func TestEvalPuiseux_Node(t *testing.T) {
	// y^2 - x^2 - x^3 == 0 => y = x + 1/2*x^2 + ..., -x - 1/2*x^2 + ...
	res, err := EvalString("puiseux(y^2 - x^2 - x^3, y, x, 2)")
	if err != nil {
		t.Fatalf("unexpected error for node: %v", err)
	}

	str := Format(res)
	if !strings.Contains(str, "x") {
		t.Fatalf("expected leading term x in node, got %s", str)
	}
}

func TestEvalPuiseux_CubeRootBranch(t *testing.T) {
	// y^3 - x == 0 => y = x^(1/3)
	res, err := EvalString("puiseux(y^3 - x, y, x, 3)")
	if err != nil {
		t.Fatalf("unexpected error for y^3 - x: %v", err)
	}

	str := Format(res)
	if !strings.Contains(str, "x^1/3") {
		t.Fatalf("expected x^1/3 in result, got %s", str)
	}
}

func TestEvalPuiseux_TaylorType(t *testing.T) {
	// y - (1 + 2*x + 3*x^2) == 0 => y = 1 + 2*x + 3*x^2
	res, err := EvalString("puiseux(y - (1 + 2*x + 3*x^2), y, x, 3)")
	if err != nil {
		t.Fatalf("unexpected error for polynomial: %v", err)
	}

	str := Format(res)
	if !strings.Contains(str, "1") || !strings.Contains(str, "2*x") {
		t.Fatalf("expected 1 + 2*x + ... in result, got %s", str)
	}
}

func TestEvalPuiseux_NegativeCases(t *testing.T) {
	// 1. Missing target variable y
	_, err := EvalString("puiseux(x^2 - 1, y, x)")
	if err == nil {
		t.Fatalf("expected error when equation contains no y, got nil")
	}

	// 2. Identity equation
	_, err = EvalString("puiseux(0, y, x)")
	if err == nil {
		t.Fatalf("expected error for identity equation, got nil")
	}

	// 3. Negative order
	_, err = EvalString("puiseux(y^2 - x, y, x, -1)")
	if err == nil {
		t.Fatalf("expected error for negative order, got nil")
	}

	// 4. Non-variable argument for y
	_, err = EvalString("puiseux(y^2 - x, 42, x)")
	if err == nil {
		t.Fatalf("expected error for non-variable y, got nil")
	}

	// 5. Wrong argument count
	_, err = EvalString("puiseux(y^2 - x, y)")
	if err == nil {
		t.Fatalf("expected error for insufficient arguments, got nil")
	}
}
