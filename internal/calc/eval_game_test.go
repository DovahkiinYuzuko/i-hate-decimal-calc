package calc

import (
	"testing"
)

func TestMex(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		{"mex([0, 1, 2, 4])", "3"},
		{"mex([])", "0"},
		{"mex([1, 2, 3])", "0"},
		{"mex([0, 2, 3])", "1"},
		{"mex(0, 1, 3)", "2"},
		{"mex([0, 1, 18446744073709551616, 2])", "3"},
		{"mex([1000000000000000000000000])", "0"},
		{"mex([0, 0, 1, 1, 5])", "2"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			res := evalTestExpr(t, tt.expr)
			if res.String() != tt.expected {
				t.Errorf("mex %s: got %s, want %s", tt.expr, res.String(), tt.expected)
			}
		})
	}
}

func TestNimSum(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		{"nim_sum(1, 2, 3)", "0"},
		{"nim_sum(3, 4, 5)", "2"},
		{"nim_sum(0)", "0"},
		{"nim_sum(7, 7)", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			res := evalTestExpr(t, tt.expr)
			if res.String() != tt.expected {
				t.Errorf("nim_sum %s: got %s, want %s", tt.expr, res.String(), tt.expected)
			}
		})
	}

	// Negative should error
	_, err := evalTestExprErr("nim_sum(-1, 2)")
	if err == nil {
		t.Errorf("expected error for negative nim_sum argument, got nil")
	}
}

func TestNashZeroSum(t *testing.T) {
	// Rock-Paper-Scissors: payoff matrix
	// [ 0, -1,  1]
	// [ 1,  0, -1]
	// [-1,  1,  0]
	// Value should be 0, strategy should be [1/3, 1/3, 1/3]
	res := evalTestExpr(t, "nash_zero_sum([[0, -1, 1], [1, 0, -1], [-1, 1, 0]])")
	list, ok := res.(*ListNode)
	if !ok || len(list.Elements) < 3 {
		t.Fatalf("expected ListNode with [value, x_strategy, y_strategy], got %s", res.String())
	}

	val := list.Elements[0].String()
	strat1 := list.Elements[1].String()
	strat2 := list.Elements[2].String()

	if val != "0" {
		t.Errorf("RPS game value: got %s, want 0", val)
	}
	if strat1 != "[1/3, 1/3, 1/3]" {
		t.Errorf("RPS Player 1 strategy: got %s, want [1/3, 1/3, 1/3]", strat1)
	}
	if strat2 != "[1/3, 1/3, 1/3]" {
		t.Errorf("RPS Player 2 strategy: got %s, want [1/3, 1/3, 1/3]", strat2)
	}
}
