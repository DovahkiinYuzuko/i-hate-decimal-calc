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

// TestEval_ConjugateRationalization validates binomial radical denominator rationalization.
func TestEval_ConjugateRationalization(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"1 / (1 + sqrt(2))", "-1 + √2"},
		{"1 / (2 + sqrt(3))", "2 - √3"},
		{"(1 + sqrt(3)) / (2 + sqrt(5))", "-2 - 2*√3 + √15 + √5"},
		{"1 / (sqrt(3) + sqrt(2))", "-√2 + √3"},
		{"6 / (sqrt(5) - sqrt(2))", "2*√2 + 2*√5"},
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

// TestEval_VariablesAndEnvironment tests evaluation with variable bindings in Env.
func TestEval_VariablesAndEnvironment(t *testing.T) {
	env := NewEnv()
	// Set x = 1/2
	half := mustRational(1, 2)
	env.Set("x", half)

	// Evaluate "x + 1"
	res, err := EvalStringWithEnv("x + 1", env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if Format(res) != "3/2" {
		t.Errorf("expected 3/2, got %s", Format(res))
	}

	// Set ans = 3/2 and evaluate "ans * 2"
	env.Set("ans", res)
	res2, err := EvalStringWithEnv("ans * 2", env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if Format(res2) != "3" {
		t.Errorf("expected 3, got %s", Format(res2))
	}
}

// TestEval_UnboundVariableSimplification tests algebraic collection of like terms with unbound variables.
func TestEval_UnboundVariableSimplification(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"2*x + 3*x", "5*x"},
		{"x - x", "0"},
		{"5*x - 2*x", "3*x"},
		{"(x + 1) * 2", "2 + 2*x"},
	}

	for _, tc := range testCases {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tc.input, err)
			continue
		}
		formatted := Format(res)
		if formatted != tc.expected {
			t.Errorf("input %q: expected %q, got %q", tc.input, tc.expected, formatted)
		}
	}
}

// TestEval_ZenkakuExpressions tests full-width character expression evaluation.
func TestEval_ZenkakuExpressions(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"１／２　＋　１／３", "5/6"},
		{"２×ｘ　＋　３×ｘ", "5*x"},
		{"ｓｑｒｔ（４）　＋　√（６４）", "10"},
		{"（１＋２）＊３", "9"},
		{"５．８８８", "736/125"},
	}

	for _, tc := range testCases {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tc.input, err)
			continue
		}
		formatted := Format(res)
		if formatted != tc.expected {
			t.Errorf("input %q: expected %q, got %q", tc.input, tc.expected, formatted)
		}
	}
}

// TestEval_Abs tests absolute value calculation for reals and complex numbers.
func TestEval_Abs(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"abs(-5)", "5"},
		{"abs(5)", "5"},
		{"abs(0)", "0"},
		{"abs(-3/4)", "3/4"},
		{"abs(3 + 4*i)", "5"},
		{"abs(1 + i)", "√2"},
		{"abs(-sqrt(2))", "√2"},
	}

	for _, tc := range testCases {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tc.input, err)
			continue
		}
		formatted := Format(res)
		if formatted != tc.expected {
			t.Errorf("input %q: expected %q, got %q", tc.input, tc.expected, formatted)
		}
	}
}

// TestEval_Cbrt tests cube root evaluation, cube-free factorization, and negative extraction.
func TestEval_Cbrt(t *testing.T) {
	testCases := []struct {
		input       string
		expected    string
		asciiExpect string
	}{
		{"cbrt(0)", "0", "0"},
		{"cbrt(8)", "2", "2"},
		{"cbrt(-8)", "-2", "-2"},
		{"cbrt(27/8)", "3/2", "3/2"},
		{"cbrt(16)", "2*³√2", "2*cbrt(2)"},
		{"cbrt(-54)", "-3*³√2", "-3*cbrt(2)"},
	}

	for _, tc := range testCases {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tc.input, err)
			continue
		}
		formatted := Format(res)
		if formatted != tc.expected {
			t.Errorf("input %q (unicode): expected %q, got %q", tc.input, tc.expected, formatted)
		}
		asciiFormatted := FormatWithOptions(res, FormatOptions{AsciiOnly: true})
		if asciiFormatted != tc.asciiExpect {
			t.Errorf("input %q (ascii): expected %q, got %q", tc.input, tc.asciiExpect, asciiFormatted)
		}
	}
}

// TestEval_NumberTheory tests gcd, lcm, and mod.
func TestEval_NumberTheory(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"gcd(24, 36)", "12"},
		{"gcd(-24, 36)", "12"},
		{"gcd(17, 13)", "1"},
		{"lcm(4, 6)", "12"},
		{"lcm(0, 5)", "0"},
		{"mod(17, 5)", "2"},
		{"mod(10, 2)", "0"},
		{"mod(-7, 3)", "2"},
	}

	for _, tc := range testCases {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tc.input, err)
			continue
		}
		formatted := Format(res)
		if formatted != tc.expected {
			t.Errorf("input %q: expected %q, got %q", tc.input, tc.expected, formatted)
		}
	}

	// Error cases
	errorCases := []string{
		"mod(5, 0)",
		"gcd(1/2, 3)",
		"lcm(2, 0.5)",
	}
	for _, ec := range errorCases {
		_, err := EvalString(ec)
		if err == nil {
			t.Errorf("expected error for %q, got nil", ec)
		}
	}
}

// TestEval_PermComb tests permutation and combination calculations.
func TestEval_PermComb(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"perm(5, 2)", "20"},
		{"perm(5, 0)", "1"},
		{"perm(5, 5)", "120"},
		{"perm(3, 5)", "0"},
		{"comb(5, 2)", "10"},
		{"comb(10, 3)", "120"},
		{"comb(5, 0)", "1"},
		{"comb(5, 5)", "1"},
		{"comb(3, 5)", "0"},
	}

	for _, tc := range testCases {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tc.input, err)
			continue
		}
		formatted := Format(res)
		if formatted != tc.expected {
			t.Errorf("input %q: expected %q, got %q", tc.input, tc.expected, formatted)
		}
	}

	// Error cases
	errorCases := []string{
		"comb(-1, 2)",
		"perm(1/2, 1)",
	}
	for _, ec := range errorCases {
		_, err := EvalString(ec)
		if err == nil {
			t.Errorf("expected error for %q, got nil", ec)
		}
	}
}

// TestEval_Rand tests deterministic and ranged integer random generation.
func TestEval_Rand(t *testing.T) {
	// Deterministic seed test
	res1, err := EvalString("rand(42, 1, 100)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	res2, err := EvalString("rand(42, 1, 100)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if Format(res1) != Format(res2) {
		t.Errorf("expected same result for same seed, got %s and %s", Format(res1), Format(res2))
	}

	// Range check
	rat, ok := res1.(*RationalNode)
	if !ok || !rat.Val.IsInt() {
		t.Fatalf("expected integer rational, got %T", res1)
	}
	val := rat.Val.Num().Int64()
	if val < 1 || val > 100 {
		t.Errorf("expected value in [1, 100], got %d", val)
	}
}



