package calc

import (
	"testing"
)

func TestEvalPolyGCD(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// 1. Univariate with shared factor
		{"poly_gcd(x^2 - 1, x^2 - 2*x + 1, x)", "-1 + x"},
		{"poly_gcd((x + 1)*(x + 2), (x + 1)*(x + 3), x)", "1 + x"},
		// 2. Coprime univariate polynomials
		{"poly_gcd(x + 1, x + 2, x)", "1"},
		// 3. Higher degree
		{"poly_gcd((x^2 + 1)*(x - 2), (x^2 + 1)*(x + 3), x)", "1 + x^2"},
		// 4. Multivariate polynomials
		{"poly_gcd(x^2 - y^2, x - y, x)", "-1 * y + x"},
		{"poly_gcd(x*y + x, x*y^2 + 2*x*y + x, x)", "x + y * x"},
		// 5. Zero polynomial edge cases
		{"poly_gcd(0, x^2 + 1, x)", "1 + x^2"},
		{"poly_gcd(x^2 + 1, 0, x)", "1 + x^2"},
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

func TestEvalPolyLCM(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"poly_lcm(x - 1, x + 1, x)", "-1 + x^2"},
		{"poly_lcm(x + 1, x + 2, x)", "2 + 3 * x + x^2"},
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
