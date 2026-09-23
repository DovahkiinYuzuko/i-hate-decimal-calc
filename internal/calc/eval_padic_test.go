package calc

import (
	"math/big"
	"testing"
)

func TestEvalPadicVal(t *testing.T) {
	// padic_val(50, 5) -> 2
	res, err := EvalString("padic_val(50, 5)")
	if err != nil {
		t.Fatalf("EvalString padic_val(50, 5) failed: %v", err)
	}
	rat, ok := res.(*RationalNode)
	if !ok || rat.Val.Cmp(new(big.Rat).SetInt64(2)) != 0 {
		t.Fatalf("expected 2, got %s", res.String())
	}

	// padic_val(3/20, 2) -> -2  (since 20 = 4*5 = 2^2 * 5, in denominator)
	res, err = EvalString("padic_val(3/20, 2)")
	if err != nil {
		t.Fatalf("EvalString padic_val(3/20, 2) failed: %v", err)
	}
	rat, ok = res.(*RationalNode)
	if !ok || rat.Val.Cmp(new(big.Rat).SetInt64(-2)) != 0 {
		t.Fatalf("expected -2, got %s", res.String())
	}
}

func TestEvalPadicNorm(t *testing.T) {
	// padic_norm(3/20, 2) -> 4  (2^(-(-2)) = 4)
	res, err := EvalString("padic_norm(3/20, 2)")
	if err != nil {
		t.Fatalf("EvalString padic_norm(3/20, 2) failed: %v", err)
	}
	rat, ok := res.(*RationalNode)
	if !ok || rat.Val.Cmp(new(big.Rat).SetInt64(4)) != 0 {
		t.Fatalf("expected 4, got %s", res.String())
	}

	// padic_norm(50, 5) -> 1/25
	res, err = EvalString("padic_norm(50, 5)")
	if err != nil {
		t.Fatalf("EvalString padic_norm(50, 5) failed: %v", err)
	}
	rat, ok = res.(*RationalNode)
	expected := new(big.Rat).SetFrac64(1, 25)
	if !ok || rat.Val.Cmp(expected) != 0 {
		t.Fatalf("expected 1/25, got %s", res.String())
	}
}

func TestEvalPadicExpand(t *testing.T) {
	// padic_expand(-1, 5, 3) -> 4 + 4*5 + 4*5^2 + O(5^3)
	res, err := EvalString("padic_expand(-1, 5, 3)")
	if err != nil {
		t.Fatalf("EvalString padic_expand(-1, 5, 3) failed: %v", err)
	}
	expected := "4 + 4*5 + 4*5^2 + O(5^3)"
	if res.String() != expected {
		t.Fatalf("expected '%s', got '%s'", expected, res.String())
	}
}
