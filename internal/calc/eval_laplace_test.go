package calc

import (
	"testing"
)

func TestEvalLaplace_Basic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "laplace(1, t, s)",
			expected: "s^-1",
		},
		{
			input:    "laplace(t, t, s)",
			expected: "s^-2",
		},
		{
			input:    "laplace(t^2, t, s)",
			expected: "2 * s^-3",
		},
		{
			input:    "laplace(exp(3*t), t, s)",
			expected: "(-3 + s)^-1",
		},
		{
			input:    "laplace(sin(2*t), t, s)",
			expected: "2 * (4 + s^2)^-1",
		},
		{
			input:    "laplace(cos(3*t), t, s)",
			expected: "s * (9 + s^2)^-1",
		},
		{
			input:    "laplace(exp(-t) * sin(2*t), t, s)",
			expected: "2 * (4 + (1 + s)^2)^-1",
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

func TestEvalInvLaplace_Basic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "inv_laplace(1 / (s - 3), s, t)",
			expected: "exp(3 * t)",
		},
		{
			input:    "inv_laplace(1 / s^2, s, t)",
			expected: "t",
		},
		{
			input:    "inv_laplace(2 / (s^2 + 4), s, t)",
			expected: "sin(2 * t)",
		},
		{
			input:    "inv_laplace(s / (s^2 + 9), s, t)",
			expected: "cos(3 * t)",
		},
		{
			// 1 / (s^2 - 1) = -1/(2*(s+1)) + 1/(2*(s-1)) => -exp(-t)/2 + exp(t)/2
			input: "inv_laplace(1 / (s^2 - 1), s, t)",
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
