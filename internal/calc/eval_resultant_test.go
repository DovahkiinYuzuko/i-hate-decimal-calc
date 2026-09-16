package calc

import (
	"testing"
)

func TestEvalResultant(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// 1. Common root exists -> resultant is 0
		{"resultant(x^2 - 3*x + 2, x^2 - 5*x + 6, x)", "0"},
		{"resultant((x - 1)*(x - 2), (x - 2)*(x - 3), x)", "0"},
		{"resultant(x^2 - 1, x - 1, x)", "0"},

		// 2. No common roots
		{"resultant(x - 1, x - 2, x)", "-1"},
		{"resultant(x^2 + 1, x - 1, x)", "2"},
		{"resultant(x^2 - 2, 2*x - 4, x)", "8"},

		// 3. Multivariate algebraic elimination (intersection of circle and line)
		// Circle: x^2 + y^2 - 1 = 0, Line: x + y - 1 = 0
		// Eliminating x yields: 2*y^2 - 2*y
		{"resultant(x^2 + y^2 - 1, x + y - 1, x)", "-2 * y + 2 * y^2"},
		{"resultant(x^2 + y^2 - 1, x + y - 1, y)", "-2 * x + 2 * x^2"},

		// 4. Boundary cases with constants
		{"resultant(x^2 + 1, 3, x)", "9"},
		{"resultant(2, x^3 + 1, x)", "8"},
		{"resultant(0, x + 1, x)", "0"},
		{"resultant(x + 1, 0, x)", "0"},
		{"resultant(3, 5, x)", "1"},

		// 5. Automatic variable deduction
		{"resultant(x^2 - 3*x + 2, x^2 - 5*x + 6)", "0"},
		{"resultant(x^2 + 1, x - 1)", "2"},
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
				t.Errorf("input %q: got %q, expected %q", tt.input, res.String(), tt.expected)
			}
		})
	}
}

func TestEvalResultantErrors(t *testing.T) {
	errorTests := []string{
		"resultant(x)",          // too few arguments
		"resultant(x, y, z, w)", // too many arguments
		"resultant(x, y, 123)",  // third argument must be variable
	}

	for _, input := range errorTests {
		t.Run(input, func(t *testing.T) {
			node, err := Parse(input)
			if err == nil {
				_, err = Eval(node)
			}
			if err == nil {
				t.Errorf("expected error for input %q, got nil", input)
			}
		})
	}
}
