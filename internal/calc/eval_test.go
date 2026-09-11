package calc

import (
	"testing"
)

// TestEval_RationalArithmetic validates basic rational arithmetic and automatic reduction.
func TestEval_RationalArithmetic(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"1/2 + 1/3", "5/6"},
		{"4/6", "2/3"},
		{"3/4 * 2/3", "1/2"},
		{"10 / 2", "5"},
		{"4 + 4 * 6.441", "7441/250"},
	}

	for _, tc := range testCases {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tc.input, err)
			continue
		}
		if res.String() != tc.expected {
			t.Errorf("input %q: expected %s, got %s", tc.input, tc.expected, res.String())
		}
	}
}

// TestEval_SquareRoots validates square-free reduction and rational square roots.
func TestEval_SquareRoots(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"sqrt(4)", "2"},
		{"sqrt(8)", "2 * sqrt(2)"},
		{"sqrt(12)", "2 * sqrt(3)"},
		{"sqrt(4/9)", "2/3"},
		{"sqrt(2) * sqrt(2)", "2"},
		{"sqrt(2) * sqrt(3)", "sqrt(6)"},
	}

	for _, tc := range testCases {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tc.input, err)
			continue
		}
		if res.String() != tc.expected {
			t.Errorf("input %q: expected %s, got %s", tc.input, tc.expected, res.String())
		}
	}
}

// TestEval_TrigonometricSpecialValues validates table simplification for sin, cos, tan.
func TestEval_TrigonometricSpecialValues(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"sin(0)", "0"},
		{"sin(pi/6)", "1/2"},
		{"sin(pi/4)", "sqrt(2)/2"},
		{"sin(pi/3)", "sqrt(3)/2"},
		{"sin(pi/2)", "1"},
		{"cos(0)", "1"},
		{"cos(pi/3)", "1/2"},
		{"cos(pi/2)", "0"},
		{"tan(0)", "0"},
		{"tan(pi/4)", "1"},
		{"tan(pi/6)", "sqrt(3)/3"},
		{"sin(1)", "sin(1)"}, // Non-special value retained as symbol
	}

	for _, tc := range testCases {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tc.input, err)
			continue
		}
		if res.String() != tc.expected {
			t.Errorf("input %q: expected %s, got %s", tc.input, tc.expected, res.String())
		}
	}
}

// TestEval_DistributiveExpansionAndLikeTerms validates FOIL expansion and like-term collection.
func TestEval_DistributiveExpansionAndLikeTerms(t *testing.T) {
	// (1 + sqrt(2)) * (1 - sqrt(2)) = 1 - 2 = -1
	res, err := EvalString("(1 + sqrt(2)) * (1 - sqrt(2))")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.String() != "-1" {
		t.Errorf("expected '-1', got %s", res.String())
	}

	// sqrt(2) + sqrt(2) = 2*sqrt(2)
	res, err = EvalString("sqrt(2) + sqrt(2)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.String() != "2 * sqrt(2)" {
		t.Errorf("expected '2 * sqrt(2)', got %s", res.String())
	}

	// 3*sqrt(2) - sqrt(2) = 2*sqrt(2)
	res, err = EvalString("3*sqrt(2) - sqrt(2)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.String() != "2 * sqrt(2)" {
		t.Errorf("expected '2 * sqrt(2)', got %s", res.String())
	}

	// sqrt(2) - sqrt(2) = 0
	res, err = EvalString("sqrt(2) - sqrt(2)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.String() != "0" {
		t.Errorf("expected '0', got %s", res.String())
	}
}

// TestEval_Rationalization validates rationalization of monomial radical denominators.
func TestEval_Rationalization(t *testing.T) {
	// 1 / sqrt(2) -> sqrt(2)/2
	res, err := EvalString("1 / sqrt(2)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.String() != "sqrt(2)/2" {
		t.Errorf("expected 'sqrt(2)/2', got %s", res.String())
	}

	// 2 / (3 * sqrt(2)) -> sqrt(2)/3
	res, err = EvalString("2 / (3 * sqrt(2))")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.String() != "sqrt(2)/3" {
		t.Errorf("expected 'sqrt(2)/3', got %s", res.String())
	}

	// (2 + 2*sqrt(2)) / 2 -> 1 + sqrt(2)
	res, err = EvalString("(2 + 2*sqrt(2)) / 2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.String() != "1 + sqrt(2)" {
		t.Errorf("expected '1 + sqrt(2)', got %s", res.String())
	}
}

// TestEval_ComplexPromotion validates promotion of negative roots to complex numbers.
func TestEval_ComplexPromotion(t *testing.T) {
	// sqrt(-4) = 2*i
	res, err := EvalString("sqrt(-4)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.String() != "0 + 2*i" {
		t.Errorf("expected '0 + 2*i', got %s", res.String())
	}

	// sqrt(-1) * sqrt(-1) = -1
	res, err = EvalString("sqrt(-1) * sqrt(-1)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.String() != "-1" {
		t.Errorf("expected '-1', got %s", res.String())
	}
}

// TestEval_PowersLogsAndFactorials validates powers, logs, and factorials.
func TestEval_PowersLogsAndFactorials(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"0^0", "1"},
		{"2^3^2", "512"},
		{"-3^2", "-9"},
		{"(-3)^2", "9"},
		{"3!", "6"},
		{"0!", "1"},
		{"log(100)", "2"},
		{"log(2, 4)", "2"},
		{"ln(e)", "1"},
	}

	for _, tc := range testCases {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tc.input, err)
			continue
		}
		if res.String() != tc.expected {
			t.Errorf("input %q: expected %s, got %s", tc.input, tc.expected, res.String())
		}
	}
}

// TestEval_DomainErrors validates all specification domain errors are detected.
func TestEval_DomainErrors(t *testing.T) {
	errorInputs := []string{
		"5 + 3/0",
		"0^(-1)",
		"tan(pi/2)",
		"tan(3*pi/2)",
		"log(0)",
		"log(-5)",
		"log(1, 4)",
		"log(-2, 4)",
		"ln(0)",
		"ln(-1)",
		"(-3)!",
		"(3/2)!",
	}

	for _, input := range errorInputs {
		_, err := EvalString(input)
		if err == nil {
			t.Errorf("expected error for domain violation %q, got nil", input)
		}
	}
}

// TestEval_AdvancedCasFixes validates edge cases reported in Issue 5.1.
func TestEval_AdvancedCasFixes(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"(1 + 2*i) * (1 - 2*i)", "5"},
		{"sin(pi/6)^2 + cos(pi/6)^2", "1"},
		{"i^2", "-1"},
		{"i^3", "-i"},
		{"i^4", "1"},
		{"(1 + i)^2", "2*i"},
		{"1 / i", "-i"},
		{"sqrt(8) / sqrt(2)", "2"},
		{"sqrt(2)^2", "2"},
		{"(-sqrt(2))^2", "2"},
		{"(2*sqrt(3))^2", "12"},
		{"((1 + sqrt(5)) / 2) * ((1 - sqrt(5)) / 2)", "-1"},
	}

	for _, tc := range testCases {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tc.input, err)
			continue
		}
		formatted := Format(res)
		if formatted != tc.expected {
			t.Errorf("input %q: expected %q, got %q (raw node: %s)", tc.input, tc.expected, formatted, res.String())
		}
	}
}

// TestEval_DenestSqrt validates Borodin (1985) radical denesting algorithm.
func TestEval_DenestSqrt(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"sqrt(5 + 2*sqrt(6))", "√2 + √3"},
		{"sqrt(5 - 2*sqrt(6))", "-√2 + √3"},
		{"sqrt(3 + sqrt(8))", "1 + √2"},
		{"sqrt(7 + 4*sqrt(3))", "2 + √3"},
		{"sqrt(1 + sqrt(2))", "√(1 + √2)"},
	}

	for _, tc := range testCases {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tc.input, err)
			continue
		}
		formatted := Format(res)
		if formatted != tc.expected {
			t.Errorf("input %q: expected %q, got %q (raw node: %s)", tc.input, tc.expected, formatted, res.String())
		}
	}
}

