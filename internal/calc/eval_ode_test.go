package calc

import (
	"testing"
)

func TestEvalDSolve_1stOrderLinearHomogeneous(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "dsolve(diff(y, x) == y, y, x)",
			expected: "y == C_1 * exp(x)",
		},
		{
			input:    "dsolve(diff(y, x) + 2 * y == 0, y, x)",
			expected: "y == C_1 * exp(-2 * x)",
		},
		{
			input:    "dsolve(diff(y, x) == 0, y, x)",
			expected: "y == C_1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			if res.String() != tt.expected {
				t.Errorf("got %s, expected %s", res.String(), tt.expected)
			}
		})
	}
}

func TestEvalDSolve_1stOrderLinearNonHomogeneous(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			// y' + y = 1 => mu = exp(x), int(exp(x)) = exp(x), y = 1 + C_1 * exp(-1 * x)
			input:    "dsolve(diff(y, x) + y == 1, y, x)",
			expected: "y == 1 + C_1 * exp(-1 * x)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			if res.String() != tt.expected {
				t.Errorf("got %s, expected %s", res.String(), tt.expected)
			}
		})
	}
}

func TestEvalDSolve_2ndOrderDistinctReal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			// y'' - 5y' + 6y = 0 => r = 2, 3 => C_1 * exp(3*x) + C_2 * exp(2*x) (or order of terms)
			input: "dsolve(diff(y, x, 2) - 5 * diff(y, x) + 6 * y == 0, y, x)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			s := res.String()
			// Check that it contains C_1, C_2, exp(2 * x), exp(3 * x)
			if s != "y == C_1 * exp(3 * x) + C_2 * exp(2 * x)" && s != "y == C_1 * exp(2 * x) + C_2 * exp(3 * x)" {
				t.Errorf("got %s, expected combination of exp(2*x) and exp(3*x)", s)
			}
		})
	}
}

func TestEvalDSolve_2ndOrderRepeatedReal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			// y'' - 4y' + 4y = 0 => (r-2)^2 = 0 => C_1 * exp(2 * x) + C_2 * x * exp(2 * x)
			input:    "dsolve(diff(y, x, 2) - 4 * diff(y, x) + 4 * y == 0, y, x)",
			expected: "y == C_1 * exp(2 * x) + C_2 * x * exp(2 * x)",
		},
		{
			// y'' = 0 => C_1 + C_2 * x
			input:    "dsolve(diff(y, x, 2) == 0, y, x)",
			expected: "y == C_1 + C_2 * x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			if res.String() != tt.expected {
				t.Errorf("got %s, expected %s", res.String(), tt.expected)
			}
		})
	}
}

func TestEvalDSolve_2ndOrderComplexConjugate(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			// y'' + 4y = 0 => r = +/- 2i => C_1 * cos(2 * x) + C_2 * sin(2 * x)
			input:    "dsolve(diff(y, x, 2) + 4 * y == 0, y, x)",
			expected: "y == C_1 * cos(2 * x) + C_2 * sin(2 * x)",
		},
		{
			// y'' + 2y' + 5y = 0 => r = -1 +/- 2i => C_1 * cos(2 * x) * exp(-1 * x) + C_2 * sin(2 * x) * exp(-1 * x)
			input:    "dsolve(diff(y, x, 2) + 2 * diff(y, x) + 5 * y == 0, y, x)",
			expected: "y == C_1 * cos(2 * x) * exp(-1 * x) + C_2 * sin(2 * x) * exp(-1 * x)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			if res.String() != tt.expected {
				t.Errorf("got %s, expected %s", res.String(), tt.expected)
			}
		})
	}
}

func TestEvalDSolve_2ndOrderNonHomogeneous(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			// y'' + 4y = 8 => yp = 2 => 2 + C_1 * cos(2 * x) + C_2 * sin(2 * x)
			input:    "dsolve(diff(y, x, 2) + 4 * y == 8, y, x)",
			expected: "y == 2 + C_1 * cos(2 * x) + C_2 * sin(2 * x)",
		},
		{
			// y'' - 3y' + 2y = exp(3*x) => P(3) = 9 - 9 + 2 = 2 => yp = (1/2) * exp(3*x)
			input: "dsolve(diff(y, x, 2) - 3 * diff(y, x) + 2 * y == exp(3 * x), y, x)",
		},
		{
			// y'' + y = sin(2*x) => yp = -1/3 * sin(2*x)
			input: "dsolve(diff(y, x, 2) + y == sin(2 * x), y, x)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			s := res.String()
			if tt.expected != "" && s != tt.expected {
				t.Errorf("got %s, expected %s", s, tt.expected)
			}
		})
	}
}
