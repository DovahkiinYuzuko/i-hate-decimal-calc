package calc

import (
	"testing"
)

func evalTestExpr(t *testing.T, exprStr string) Node {
	t.Helper()
	parsed, err := Parse(exprStr)
	if err != nil {
		t.Fatalf("Parse(%q) failed: %v", exprStr, err)
	}
	node, err := EvalWithEnv(parsed, NewEnv())
	if err != nil {
		t.Fatalf("EvalWithEnv(%q) failed: %v", exprStr, err)
	}
	return node
}

func evalTestExprErr(exprStr string) (Node, error) {
	parsed, err := Parse(exprStr)
	if err != nil {
		return nil, err
	}
	return EvalWithEnv(parsed, NewEnv())
}

func TestGraphLaplacian(t *testing.T) {
	// K3 complete graph
	res := evalTestExpr(t, "graph_laplacian([[0, 1, 1], [1, 0, 1], [1, 1, 0]])")
	mat, ok := res.(*MatrixNode)
	if !ok {
		t.Fatalf("expected MatrixNode, got %T", res)
	}
	expected := "[[2, -1, -1], [-1, 2, -1], [-1, -1, 2]]"
	if mat.String() != expected {
		t.Errorf("graph_laplacian K3: got %s, want %s", mat.String(), expected)
	}

	// Asymmetric matrix should error
	if _, err := evalTestExprErr("graph_laplacian([[0, 1], [0, 0]])"); err == nil {
		t.Errorf("expected error for asymmetric adjacency matrix, got nil")
	}

	// Oversized weight (> int64) should error
	if _, err := evalTestExprErr("graph_laplacian([[0, 18446744073709551616], [18446744073709551616, 0]])"); err == nil {
		t.Errorf("expected error for oversized edge weight, got nil")
	}
}

func TestSpanningTrees(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected string
	}{
		{
			name:     "K3",
			expr:     "spanning_trees([[0, 1, 1], [1, 0, 1], [1, 1, 0]])",
			expected: "3",
		},
		{
			name:     "K4",
			expr:     "spanning_trees([[0, 1, 1, 1], [1, 0, 1, 1], [1, 1, 0, 1], [1, 1, 1, 0]])",
			expected: "16", // 4^(4-2) = 16
		},
		{
			name:     "Disconnected",
			expr:     "spanning_trees([[0, 1, 0, 0], [1, 0, 0, 0], [0, 0, 0, 1], [0, 0, 1, 0]])",
			expected: "0",
		},
		{
			name:     "SingleVertex",
			expr:     "spanning_trees([[0]])",
			expected: "1",
		},
		{
			name:     "MultigraphTwoVerticesThreeEdges",
			expr:     "spanning_trees([[0, 3], [3, 0]])",
			expected: "3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := evalTestExpr(t, tt.expr)
			if res.String() != tt.expected {
				t.Errorf("spanning_trees %s: got %s, want %s", tt.name, res.String(), tt.expected)
			}
		})
	}
}

func TestChromaticPolynomial(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected string
	}{
		{
			name:     "SelfLoop",
			expr:     "chromatic_poly([[1, 1], [1, 0]])",
			expected: "0",
		},
		{
			name:     "EmptyGraphV3",
			expr:     "chromatic_poly([[0, 0, 0], [0, 0, 0], [0, 0, 0]])",
			expected: "k^3",
		},
		{
			name:     "K3_EvaluatedAt4",
			expr:     "chromatic_poly([[0, 1, 1], [1, 0, 1], [1, 1, 0]], 4)",
			expected: "24", // 4 * 3 * 2 = 24
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := evalTestExpr(t, tt.expr)
			if res.String() != tt.expected {
				t.Errorf("chromatic_poly %s: got %s, want %s", tt.name, res.String(), tt.expected)
			}
		})
	}
}
