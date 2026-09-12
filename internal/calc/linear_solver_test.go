package calc

import (
	"strings"
	"testing"
)

func TestLinear_RREF(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "rref([[1, 0], [0, 1]])",
			expected: "[[1, 0], [0, 1]]",
		},
		{
			input:    "rref([[1, 2], [3, 4]])",
			expected: "[[1, 0], [0, 1]]",
		},
		{
			input:    "rref([[1, 2, 3], [4, 5, 6], [7, 8, 9]])",
			expected: "[[1, 0, -1], [0, 1, 2], [0, 0, 0]]",
		},
		{
			input:    "rref([[1, 2, 3], [2, 4, 6]])",
			expected: "[[1, 2, 3], [0, 0, 0]]",
		},
		{
			input:    "rref([[0, 2], [3, 0]])",
			expected: "[[1, 0], [0, 1]]",
		},
	}

	for _, tt := range tests {
		ast, err := Parse(tt.input)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", tt.input, err)
		}
		res, err := Eval(ast)
		if err != nil {
			t.Fatalf("Eval(%q) error: %v", tt.input, err)
		}
		if res.String() != tt.expected {
			t.Errorf("Eval(%q) = %s, want %s", tt.input, res.String(), tt.expected)
		}
	}
}

func TestLinear_Rank(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "rank([[1, 0], [0, 1]])",
			expected: "2",
		},
		{
			input:    "rank([[1, 2], [3, 4]])",
			expected: "2",
		},
		{
			input:    "rank([[1, 2, 3], [4, 5, 6], [7, 8, 9]])",
			expected: "2",
		},
		{
			input:    "rank([[1, 2, 3], [2, 4, 6]])",
			expected: "1",
		},
		{
			input:    "rank([[0, 0], [0, 0]])",
			expected: "0",
		},
	}

	for _, tt := range tests {
		ast, err := Parse(tt.input)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", tt.input, err)
		}
		res, err := Eval(ast)
		if err != nil {
			t.Fatalf("Eval(%q) error: %v", tt.input, err)
		}
		if res.String() != tt.expected {
			t.Errorf("Eval(%q) = %s, want %s", tt.input, res.String(), tt.expected)
		}
	}
}

func TestLinear_SolveLinear_Unique(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			// 2x + y = 5, x - y = 1 => x = 2, y = 1
			input:    "solve_linear([[2, 1], [1, -1]], [5, 1])",
			expected: "[2, 1]",
		},
		{
			// column matrix for b
			input:    "linsolve([[2, 1], [1, -1]], [[5], [1]])",
			expected: "[2, 1]",
		},
		{
			// 3x3 system:
			// x + y + z = 6
			// 2y + 5z = -4
			// 2x + 5y - z = 27
			// Solution: x = 5, y = 3, z = -2
			input:    "solve_linear([[1, 1, 1], [0, 2, 5], [2, 5, -1]], [6, -4, 27])",
			expected: "[5, 3, -2]",
		},
	}

	for _, tt := range tests {
		ast, err := Parse(tt.input)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", tt.input, err)
		}
		res, err := Eval(ast)
		if err != nil {
			t.Fatalf("Eval(%q) error: %v", tt.input, err)
		}
		if res.String() != tt.expected {
			t.Errorf("Eval(%q) = %s, want %s", tt.input, res.String(), tt.expected)
		}
	}
}

func TestLinear_SolveLinear_Inconsistent(t *testing.T) {
	// Inconsistent system: x + y = 1, x + y = 2
	input := "solve_linear([[1, 1], [1, 1]], [1, 2])"
	ast, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	_, err = Eval(ast)
	if err == nil {
		t.Fatalf("expected error for inconsistent system, got nil")
	}
	if !strings.Contains(err.Error(), "inconsistent") && !strings.Contains(err.Error(), "解が存在しません") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestLinear_SolveLinear_Underdetermined(t *testing.T) {
	// Underdetermined system: x1 + x2 = 3
	// Expected: [3 - x2, x2] or equivalent
	input := "solve_linear([[1, 1]], [3])"
	ast, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	res, err := Eval(ast)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	listNode, ok := res.(*ListNode)
	if !ok || len(listNode.Elements) != 2 {
		t.Fatalf("expected 2-element ListNode, got %s", res.String())
	}
	// x2 should be free variable VarNode("x2")
	if listNode.Elements[1].String() != "x2" {
		t.Errorf("expected x2 for free variable, got %s", listNode.Elements[1].String())
	}
}

func TestLinear_SolveLinear_DimMismatch(t *testing.T) {
	input := "solve_linear([[1, 2], [3, 4]], [1, 2, 3])"
	ast, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	_, err = Eval(ast)
	if err == nil {
		t.Fatalf("expected dim mismatch error, got nil")
	}
}
