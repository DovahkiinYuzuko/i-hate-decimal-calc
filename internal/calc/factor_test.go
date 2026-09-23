package calc

import (
	"strings"
	"testing"
)

func TestFactor_Integer(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"factor(0)", "0"},
		{"factor(1)", "1"},
		{"factor(-1)", "-1"},
		{"factor(17)", "17"},
		{"factor(360)", "2^3*3^2*5"},
		{"factor(-24)", "-1*2^3*3"},
		{"factor(100)", "2^2*5^2"},
		{"factor(9797)", "97*101"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			resStr := Format(res)
			if resStr != tt.expected {
				t.Errorf("got %q, want %q", resStr, tt.expected)
			}
		})
	}
}

func TestFactor_Polynomial_Quadratic(t *testing.T) {
	tests := []struct {
		input    string
		expected []string // multiple valid factor orderings
	}{
		{
			"factor(x^2 - 5*x + 6)",
			[]string{"(-2 + x)*(-3 + x)", "(-3 + x)*(-2 + x)"},
		},
		{
			"factor(x^2 - 4*x + 4)",
			[]string{"(-2 + x)^2"},
		},
		{
			"factor(2*x^2 + 5*x + 2)",
			[]string{"(1 + 2*x)*(2 + x)", "(2 + x)*(1 + 2*x)"},
		},
		{
			"factor(x^2 - 9)",
			[]string{"(-3 + x)*(3 + x)", "(3 + x)*(-3 + x)"},
		},
		{
			"factor(x^2 + 1)",
			[]string{"1 + x^2"}, // irreducible over Q
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			resStr := Format(res)
			matched := false
			for _, exp := range tt.expected {
				if resStr == exp {
					matched = true
					break
				}
			}
			if !matched {
				t.Errorf("got %q, want one of %v", resStr, tt.expected)
			}
		})
	}
}

func TestFactor_Polynomial_CubicAndHigher(t *testing.T) {
	tests := []struct {
		input         string
		expectedParts []string
	}{
		{
			"factor(x^3 - 6*x^2 + 11*x - 6)",
			[]string{"(-1 + x)", "(-2 + x)", "(-3 + x)"},
		},
		{
			"factor(2*x^3 - 3*x^2 - 11*x + 6)",
			[]string{"(-1 + 2*x)", "(2 + x)", "(-3 + x)"},
		},
		{
			"factor(x^4 - 5*x^2 + 4)",
			[]string{"(-1 + x)", "(1 + x)", "(-2 + x)", "(2 + x)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			resStr := Format(res)
			for _, part := range tt.expectedParts {
				if !strings.Contains(resStr, part) {
					t.Errorf("expected result %q to contain factor %q", resStr, part)
				}
			}
		})
	}
}

func TestFactor_CommonFactors(t *testing.T) {
	// Common monomial factor: 2*x^2 + 4*x => 2*x*(2 + x)
	res, err := EvalString("factor(2*x^2 + 4*x)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resStr := Format(res)
	if !strings.Contains(resStr, "(2 + x)") || !strings.Contains(resStr, "x") || !strings.Contains(resStr, "2") {
		t.Errorf("expected 2*x*(2 + x), got %q", resStr)
	}

	// Common variable power: x^4 - 5*x^3 + 6*x^2 => x^2*(-2 + x)*(-3 + x)
	res2, err2 := EvalString("factor(x^4 - 5*x^3 + 6*x^2)")
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}
	resStr2 := Format(res2)
	if !strings.Contains(resStr2, "x^2") || !strings.Contains(resStr2, "(-2 + x)") || !strings.Contains(resStr2, "(-3 + x)") {
		t.Errorf("expected x^2*(-2 + x)*(-3 + x), got %q", resStr2)
	}
}

func TestFactor_ExplicitVariable(t *testing.T) {
	res, err := EvalString("factor(y^2 - 4, y)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resStr := Format(res)
	if !(resStr == "(-2 + y)*(2 + y)" || resStr == "(2 + y)*(-2 + y)") {
		t.Errorf("expected (-2 + y)*(2 + y), got %q", resStr)
	}
}

func TestFactor_BigInt(t *testing.T) {
	// Large prime beyond 64-bit integer: 10^20 + 39 is prime
	primeStr := "100000000000000000039"
	res, err := EvalString("factor(" + primeStr + ")")
	if err != nil {
		t.Fatalf("unexpected error for large prime: %v", err)
	}
	if Format(res) != primeStr {
		t.Errorf("got %q, want %q", Format(res), primeStr)
	}

	// Large semiprime: 2 * (10^20 + 39) = 200000000000000000078
	semiStr := "200000000000000000078"
	resSemi, err := EvalString("factor(" + semiStr + ")")
	if err != nil {
		t.Fatalf("unexpected error for large semiprime: %v", err)
	}
	expectedSemi := "2*" + primeStr
	if Format(resSemi) != expectedSemi {
		t.Errorf("got %q, want %q", Format(resSemi), expectedSemi)
	}
}

func TestFactor_Polynomial_Hensel(t *testing.T) {
	// (x^2 + 1) * (x^2 + 2) = x^4 + 3*x^2 + 2
	res, err := EvalString("factor(x^4 + 3*x^2 + 2)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resStr := Format(res)
	if !(resStr == "(1 + x^2)*(2 + x^2)" || resStr == "(2 + x^2)*(1 + x^2)") {
		t.Errorf("expected (1 + x^2)*(2 + x^2), got %q", resStr)
	}

	// (x^2 - 2) * (x^2 + 2) = x^4 - 4
	res2, err := EvalString("factor(x^4 - 4)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resStr2 := Format(res2)
	if !(resStr2 == "(-2 + x^2)*(2 + x^2)" || resStr2 == "(2 + x^2)*(-2 + x^2)") {
		t.Errorf("expected (-2 + x^2)*(2 + x^2), got %q", resStr2)
	}
}




