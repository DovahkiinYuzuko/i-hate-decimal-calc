package calc

import (
	"testing"
)

func TestEvalFourierSeries(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// 1. Sawtooth wave f(t) = t over [-pi, pi]
		// a0 = 0, bk = 2*(-1)^(k+1)/k -> b1=2, b2=-1, b3=2/3
		{"fourier_series(t, t, pi, 3)", "-sin(2*t) + 2*sin(t) + 2/3*sin(3*t)"},

		// 2. Parabolic wave f(t) = t^2 over [-pi, pi]
		// a0 = pi^2/3, a1 = -4, a2 = 1, bk = 0
		{"fourier_series(t^2, t, pi, 2)", "-4*cos(t) + cos(2*t) + pi^2/3"},

		// 3. Pure harmonic function orthogonality
		{"fourier_series(cos(t), t, pi, 3)", "cos(t)"},
		{"fourier_series(sin(2*t), t, pi, 3)", "sin(2*t)"},

		// 4. Constant function f(t) = 5
		{"fourier_series(5, t, pi, 3)", "5"},

		// 5. Default arguments: fourier_series(t) defaults to var=t, L=pi, n=3
		{"fourier_series(t)", "-sin(2*t) + 2*sin(t) + 2/3*sin(3*t)"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error for %s: %v", tt.input, err)
			}
			res, err := Eval(node)
			if err != nil {
				t.Fatalf("Eval error for %s: %v", tt.input, err)
			}
			got := FormatWithOptions(res, FormatOptions{AsciiOnly: true})
			if got != tt.expected {
				t.Errorf("fourier_series mismatch\ninput:    %s\ngot:      %s\nexpected: %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestEvalFourierSeriesErrors(t *testing.T) {
	errorInputs := []string{
		"fourier_series()",                // too few arguments
		"fourier_series(t, t, pi, 3, 5)", // too many arguments
		"fourier_series(t, t, 0, 3)",     // L = 0
		"fourier_series(t, t, pi, 0)",    // n = 0
		"fourier_series(t, t, pi, -2)",   // n < 0
	}

	for _, input := range errorInputs {
		t.Run(input, func(t *testing.T) {
			node, err := Parse(input)
			if err != nil {
				// Parser error is also an acceptable rejection
				return
			}
			_, err = Eval(node)
			if err == nil {
				t.Errorf("expected error for %s, got nil", input)
			}
		})
	}
}
