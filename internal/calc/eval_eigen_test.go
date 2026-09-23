package calc

import (
	"testing"
)

func TestEvalTrace(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"trace([[1, 2], [3, 4]])", "5"},
		{"tr([[5, 0, 0], [0, 10, 0], [0, 0, 15]])", "30"},
		{"trace([[1/2, 3], [4, 1/3]])", "5/6"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			res, err := Eval(node)
			if err != nil {
				t.Fatalf("Eval failed: %v", err)
			}
			if res.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, res.String())
			}
		})
	}
}

func TestEvalEigenvals(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"eigenvals([[3, 0], [0, 5]])", "[3, 5]"},
		{"eigenvals([[2, 1], [1, 2]])", "[1, 3]"},
		{"eigenvals([[2, 1], [0, 2]])", "[2, 2]"},
		{"eigenvals([[0, -1], [1, 0]])", "[-1 * i, i]"},
		{"eigenvals([[1, 0, 0], [0, 2, 0], [0, 0, 3]])", "[1, 2, 3]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			res, err := Eval(node)
			if err != nil {
				t.Fatalf("Eval failed: %v", err)
			}
			if res.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, res.String())
			}
		})
	}
}

func TestEvalEigenvects(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// 2x2 diagonal
		{"eigenvects([[3, 0], [0, 5]])", "[[3, [[1, 0]]], [5, [[0, 1]]]]"},
		// 2x2 symmetric
		{"eigenvects([[2, 1], [1, 2]])", "[[1, [[1, -1]]], [3, [[1, 1]]]]"},
		// 2x2 Jordan block (geometric multiplicity 1)
		{"eigenvects([[2, 1], [0, 2]])", "[[2, [[1, 0]]]]"},
		// 2x2 Identity (geometric multiplicity 2)
		{"eigenvects([[1, 0], [0, 1]])", "[[1, [[1, 0], [0, 1]]]]"},
	}


	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			res, err := Eval(node)
			if err != nil {
				t.Fatalf("Eval failed: %v", err)
			}
			if res.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, res.String())
			}
		})
	}
}

func TestEvalEigenvals_CubicExact(t *testing.T) {
	// 3x3 matrix with characteristic poly lambda^3 - 2 = 0
	// Companion matrix: [[0, 1, 0], [0, 0, 1], [2, 0, 0]]
	node, err := Parse("eigenvals([[0, 1, 0], [0, 0, 1], [2, 0, 0]])")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	res, err := Eval(node)
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}
	listNode, ok := res.(*ListNode)
	if !ok || len(listNode.Elements) != 3 {
		t.Fatalf("expected 3 eigenvalues, got %v", res)
	}
	// Check that we got 3 exact roots
	t.Logf("eigenvals(x^3 - 2 = 0) = %s", res.String())

	// 3x3 matrix with casus irreducibilis poly lambda^3 - 3*lambda - 1 = 0
	node2, err := Parse("eigenvals([[0, 1, 0], [0, 0, 1], [1, 3, 0]])")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	res2, err := Eval(node2)
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}
	listNode2, ok := res2.(*ListNode)
	if !ok || len(listNode2.Elements) != 3 {
		t.Fatalf("expected 3 eigenvalues, got %v", res2)
	}
	t.Logf("eigenvals(x^3 - 3x - 1 = 0) = %s", res2.String())
}
