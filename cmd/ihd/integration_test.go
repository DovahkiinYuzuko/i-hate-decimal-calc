package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
)

// TestIntegration_SpecificationExamples verifies all primary calculation examples from specifications.
func TestIntegration_SpecificationExamples(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		// 1. Decimals converted to fractions immediately
		{"4 + 4 * 6.441", "7441/250"},

		// 2. Distributive expansion and conjugate products
		{"(1 + sqrt(2)) * (1 - sqrt(2))", "-1"},
		{"(1 + 2*i) * (1 - 2*i)", "5"},
		{"((1 + sqrt(5)) / 2) * ((1 - sqrt(5)) / 2)", "-1"},

		// 3. Trigonometric special values & Pythagorean identity
		{"sin(pi/6)", "1/2"},
		{"sin(pi/4)", "√2/2"},
		{"cos(0)", "1"},
		{"cos(pi/6)", "√3/2"},
		{"tan(pi/4)", "1"},
		{"sin(pi/6)^2 + cos(pi/6)^2", "1"},
		{"2 * sin(pi/4) * cos(pi/4)", "1"},

		// 4. Square root simplifications and rationalizations
		{"sqrt(8)", "2*√2"},
		{"1 / sqrt(2)", "√2/2"},
		{"(2 + 2*sqrt(2)) / 2", "1 + √2"},
		{"sqrt(72) - sqrt(50) + sqrt(18)", "4*√2"},
		{"sqrt(12) * sqrt(75)", "30"},
		{"sqrt(8) / sqrt(2)", "2"},

		// 5. Complex numbers & imaginary promotion
		{"sqrt(-4)", "2*i"},
		{"sqrt(-1) * sqrt(-1)", "-1"},
		{"i^2", "-1"},
		{"i^3", "-i"},
		{"i^4", "1"},
		{"1 / i", "-i"},
		{"(1 + i)^2", "2*i"},

		// 6. Logarithms
		{"log(2, 4)", "2"},
		{"log(2, 8)", "3"},
		{"log(100)", "2"},
		{"ln(e)", "1"},
		{"ln(e^5)", "5"},

		// 7. Factorials
		{"3!", "6"},
		{"0!", "1"},
		{"5!", "120"},
		{"5! / (3! * 2!)", "10"},

		// 8. Repeating decimals & radical denesting (issue-11)
		{"0.(3) + 0.1(6)", "1/2"},
		{"sqrt(5 + 2*sqrt(6))", "√2 + √3"},
		{"sqrt(7 + 4*sqrt(3))", "2 + √3"},

		// 9. Binomial radical conjugate rationalization (issue-12)
		{"(1 + sqrt(3)) / (2 + sqrt(5))", "-2 - 2*√3 + √15 + √5"},
		{"1 / (2 + sqrt(3))", "2 - √3"},
		{"6 / (sqrt(5) - sqrt(2))", "2*√2 + 2*√5"},

		// 10. Power associativity and order of operations
		{"0^0", "1"},
		{"2^3^2", "512"},
		{"(2^3)^2", "64"},
		{"-3^2", "-9"},
		{"(-3)^2", "9"},
		{"(-sqrt(2))^2", "2"},

		// 11. Scientific calculator basic pack (issue-14)
		{"abs(-42)", "42"},
		{"abs(3 + 4*i)", "5"},
		{"abs(-sqrt(2))", "√2"},
		{"cbrt(8)", "2"},
		{"cbrt(-27)", "-3"},
		{"cbrt(16)", "2*³√2"},
		{"gcd(48, 18)", "6"},
		{"lcm(4, 6)", "12"},
		{"mod(23, 5)", "3"},
		{"perm(6, 2)", "30"},
		{"comb(6, 2)", "15"},

		// 12. Trigonometric pack (issue-15)
		{"asin(1/2)", "π/6"},
		{"acos(1/2)", "π/3"},
		{"atan(1)", "π/4"},
		{"sin(30*deg)", "1/2"},
		{"asin(1/2) / deg", "30"},
	}

	for _, tc := range testCases {
		res, err := calc.EvalString(tc.input)
		if err != nil {
			t.Errorf("input %q: unexpected error: %v", tc.input, err)
			continue
		}
		actual := calc.Format(res)
		if actual != tc.expected {
			t.Errorf("input %q: expected %q, got %q", tc.input, tc.expected, actual)
		}
	}
}

// TestIntegration_DomainAndSyntaxErrors verifies all invalid operations produce explicit errors.
func TestIntegration_DomainAndSyntaxErrors(t *testing.T) {
	errorInputs := []struct {
		input       string
		errContains string
	}{
		// Zero division
		{"1/0", "division by zero"},
		{"0^(-1)", "division by zero"},
		{"1 / (0 + 0*i)", "division by zero"},

		// Domain errors
		{"tan(pi/2)", "undefined"},
		{"tan(3*pi/2)", "undefined"},
		{"log(0)", "domain error"},
		{"log(-1)", "domain error"},
		{"log(1, 4)", "base cannot be 1"},
		{"log(-2, 4)", "base must be positive"},
		{"ln(0)", "domain error"},
		{"ln(-1)", "domain error"},
		{"(-3)!", "factorial requires non-negative integer"},
		{"(3/2)!", "factorial requires non-negative integer"},

		// Syntax errors (implicit multiplication prohibited)
		{"2pi", "implicit multiplication is not allowed"},
		{"(1+2)(3+4)", "implicit multiplication is not allowed"},
		{"2sqrt(2)", "implicit multiplication is not allowed"},

		// Syntax errors (parentheses & operators)
		{"(1 + 2", "syntax error"},
		{"1 + * 2", "syntax error"},
		{"", "syntax error"},
	}

	for _, tc := range errorInputs {
		_, err := calc.EvalString(tc.input)
		if err == nil {
			t.Errorf("input %q: expected error containing %q, got nil", tc.input, tc.errContains)
			continue
		}
		if !strings.Contains(err.Error(), tc.errContains) {
			t.Errorf("input %q: expected error containing %q, got %v", tc.input, tc.errContains, err)
		}
	}
}

// TestIntegration_CLIEndToEnd verifies full CLI execution paths through run().
func TestIntegration_CLIEndToEnd(t *testing.T) {
	// One-shot execution
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	code := run([]string{"--approx", "sqrt(2) + sqrt(8)"}, strings.NewReader(""), out, errOut)
	if code != 0 {
		t.Fatalf("CLI one-shot failed with exit code %d: %s", code, errOut.String())
	}
	output := strings.TrimSpace(out.String())
	if !strings.Contains(output, "3*√2") || !strings.Contains(output, "4.24264068") {
		t.Errorf("expected approx output for 3*√2, got %q", output)
	}

	// ASCII flag
	out.Reset()
	errOut.Reset()
	code = run([]string{"--ascii", "sqrt(2) + pi"}, strings.NewReader(""), out, errOut)
	if code != 0 {
		t.Fatalf("CLI ascii flag failed: %s", errOut.String())
	}
	if strings.TrimSpace(out.String()) != "sqrt(2) + pi" {
		t.Errorf("expected 'sqrt(2) + pi', got %q", strings.TrimSpace(out.String()))
	}

	// Degree mode flag (--deg)
	out.Reset()
	errOut.Reset()
	code = run([]string{"--deg", "sin(30) + cos(60) + asin(1/2)"}, strings.NewReader(""), out, errOut)
	if code != 0 {
		t.Fatalf("CLI degree mode failed: %s", errOut.String())
	}
	if strings.TrimSpace(out.String()) != "31" {
		t.Errorf("expected '31', got %q", strings.TrimSpace(out.String()))
	}
}
