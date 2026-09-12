package calc

import (
	"strings"
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

// TestEval_InverseTrig tests inverse trigonometric exact simplification.
func TestEval_InverseTrig(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		// asin
		{"asin(0)", "0"},
		{"asin(1/2)", "π/6"},
		{"asin(-1/2)", "-π/6"},
		{"asin(sqrt(2)/2)", "π/4"},
		{"asin(-sqrt(2)/2)", "-π/4"},
		{"asin(sqrt(3)/2)", "π/3"},
		{"asin(-sqrt(3)/2)", "-π/3"},
		{"asin(1)", "π/2"},
		{"asin(-1)", "-π/2"},

		// acos
		{"acos(0)", "π/2"},
		{"acos(1/2)", "π/3"},
		{"acos(-1/2)", "2/3*π"},
		{"acos(sqrt(2)/2)", "π/4"},
		{"acos(-sqrt(2)/2)", "3/4*π"},
		{"acos(sqrt(3)/2)", "π/6"},
		{"acos(-sqrt(3)/2)", "5/6*π"},
		{"acos(1)", "0"},
		{"acos(-1)", "π"},

		// atan
		{"atan(0)", "0"},
		{"atan(sqrt(3)/3)", "π/6"},
		{"atan(-sqrt(3)/3)", "-π/6"},
		{"atan(1)", "π/4"},
		{"atan(-1)", "-π/4"},
		{"atan(sqrt(3))", "π/3"},
		{"atan(-sqrt(3))", "-π/3"},
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

	// Domain error cases
	errorCases := []string{
		"asin(2)",
		"asin(-3/2)",
		"acos(2)",
		"acos(-5)",
	}
	for _, ec := range errorCases {
		_, err := EvalString(ec)
		if err == nil {
			t.Errorf("expected error for %q, got nil", ec)
		}
	}
}

// TestEval_DegreeMode tests deg constant and ApplyDegreeMode AST transformations.
func TestEval_DegreeMode(t *testing.T) {
	// 1. Direct evaluation using built-in constant deg
	degCases := []struct {
		input    string
		expected string
	}{
		{"sin(30*deg)", "1/2"},
		{"cos(60*deg)", "1/2"},
		{"tan(45*deg)", "1"},
		{"asin(1/2) / deg", "30"},
		{"atan(1) / deg", "45"},
	}
	for _, tc := range degCases {
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

	// 2. ApplyDegreeMode transformation
	modeCases := []struct {
		input    string
		expected string
	}{
		{"sin(30)", "1/2"},
		{"cos(60)", "1/2"},
		{"tan(45)", "1"},
		{"asin(1/2)", "30"},
		{"acos(1/2)", "60"},
		{"atan(1)", "45"},
	}
	for _, tc := range modeCases {
		parsed, err := Parse(tc.input)
		if err != nil {
			t.Fatalf("parse error for %q: %v", tc.input, err)
		}
		transformed := ApplyDegreeMode(parsed)
		res, err := Eval(transformed)
		if err != nil {
			t.Errorf("eval error for %q in degree mode: %v", tc.input, err)
			continue
		}
		formatted := Format(res)
		if formatted != tc.expected {
			t.Errorf("degree mode %q: expected %q, got %q", tc.input, tc.expected, formatted)
		}
	}
}

// TestEval_SymbolPowerCancellation tests merging and cancellation of identical base powers (sub-issue-15.1).
func TestEval_SymbolPowerCancellation(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"pi * pi^-1", "1"},
		{"(2*pi) * (3*pi^-1)", "6"},
		{"x * x^-1", "1"},
		{"x^2 * x^3", "x^5"},
		{"x^3 * x^-3", "1"},
		{"x * x", "x^2"},
	}

	for _, tc := range cases {
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

// TestEval_Expand tests polynomial expansion.
func TestEval_Expand(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"expand((x + 1) * (x - 2))", "-2 - x + x^2"},
		{"expand((x + 1)^2)", "1 + 2*x + x^2"},
		{"expand((x + 1) * (x + 1))", "1 + 2*x + x^2"},
		{"expand(2 * (x + 3))", "6 + 2*x"},
		{"expand((x + 1)^3)", "1 + 3*x + 3*x^2 + x^3"},
		{"expand((x + 2) * (x - 2))", "-4 + x^2"},
	}

	for _, tc := range cases {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tc.input, err)
			continue
		}
		formatted := Format(res)
		if formatted != tc.expected {
			t.Errorf("expand %q: expected %q, got %q", tc.input, tc.expected, formatted)
		}
	}
}

// TestEval_Diff tests exact symbolic differentiation.
func TestEval_Diff(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		// Polynomial
		{"diff(x^3 - 3*x^2 + 2*x - 5, x)", "2 - 6*x + 3*x^2"},
		{"diff(5, x)", "0"},
		{"diff(pi, x)", "0"},
		{"diff(y, x)", "0"},
		{"diff(x, x)", "1"},
		{"diff(2*x, x)", "2"},

		// Trigonometric
		{"diff(sin(x), x)", "cos(x)"},
		{"diff(cos(x), x)", "-sin(x)"},
		{"diff(sin(2*x), x)", "2*cos(2*x)"},

		// Exponential & Logarithm
		{"diff(ln(x), x)", "x^-1"},
		{"diff(e^x, x)", "e^x"},

		// Power & Fraction
		{"diff(x^-1, x)", "-x^-2"},
		{"diff(x^2, x)", "2*x"},
	}

	for _, tc := range cases {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tc.input, err)
			continue
		}
		formatted := Format(res)
		if formatted != tc.expected {
			t.Errorf("diff %q: expected %q, got %q", tc.input, tc.expected, formatted)
		}
	}
}

// TestEval_Solve tests algebraic equation solving.
func TestEval_Solve(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		// Linear
		{"solve(2*x + 4, x)", "[-2]"},
		{"solve(3*x - 1, x)", "[1/3]"},
		{"solve(x - 5, x)", "[5]"},

		// Quadratic (integer roots)
		{"solve(x^2 - 4, x)", "[-2, 2]"},
		{"solve(x^2 - 5*x + 6, x)", "[2, 3]"},
		{"solve(x^2 - 4*x + 4, x)", "[2]"}, // Repeated root

		// Quadratic (irrational / radical roots)
		{"solve(x^2 - 2, x)", "[-√2, √2]"},

		// Quadratic (complex / imaginary roots)
		{"solve(x^2 + 1, x)", "[-i, i]"},
		{"solve(x^2 + 4, x)", "[-2*i, 2*i]"},
	}

	for _, tc := range cases {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tc.input, err)
			continue
		}
		formatted := Format(res)
		if formatted != tc.expected {
			t.Errorf("solve %q: expected %q, got %q", tc.input, tc.expected, formatted)
		}
	}
}

// TestEval_Matrix tests exact matrix operations (addition, multiplication, det, inv, transpose).
func TestEval_Matrix(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		// Parsing and addition / subtraction
		{"[[1, 2], [3, 4]] + [[5, 6], [7, 8]]", "[[6, 8], [10, 12]]"},
		{"[[1, 2], [3, 4]] - [[1, 0], [0, 1]]", "[[0, 2], [3, 3]]"},

		// Scalar multiplication
		{"2 * [[1, 2], [3, 4]]", "[[2, 4], [6, 8]]"},
		{"[[1, 2], [3, 4]] * 3", "[[3, 6], [9, 12]]"},

		// Matrix multiplication
		{"[[1, 2], [3, 4]] * [[2, 0], [1, 2]]", "[[4, 4], [10, 8]]"},
		{"[[1, 2, 3], [4, 5, 6]] * [[7, 8], [9, 1], [2, 3]]", "[[31, 19], [85, 55]]"},

		// Determinant
		{"det([[1, 2], [3, 4]])", "-2"},
		{"det([[1, 2, 3], [0, 4, 5], [1, 0, 6]])", "22"},
		{"det([[sqrt(2), 1], [1, sqrt(2)]])", "1"},
		{"det([[i, 1], [1, i]])", "-2"},

		// Inverse
		{"inv([[1, 2], [3, 4]])", "[[-2, 1], [3/2, -1/2]]"},
		{"[[1, 2], [3, 4]] * inv([[1, 2], [3, 4]])", "[[1, 0], [0, 1]]"},

		// Transpose
		{"transpose([[1, 2, 3], [4, 5, 6]])", "[[1, 4], [2, 5], [3, 6]]"},
	}

	for _, tc := range cases {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tc.input, err)
			continue
		}
		formatted := Format(res)
		if formatted != tc.expected {
			t.Errorf("matrix %q: expected %q, got %q", tc.input, tc.expected, formatted)
		}
	}

	// Error cases
	errorCases := []struct {
		input       string
		errContains string
	}{
		{"inv([[1, 2], [2, 4]])", "singular matrix"},
		{"det([[1, 2, 3], [4, 5, 6]])", "square matrix"},
		{"[[1, 2]] * [[1, 2]]", "dimension mismatch"},
		{"[[1, 2]] + [[1], [2]]", "dimension mismatch"},
	}
	for _, tc := range errorCases {
		_, err := EvalString(tc.input)
		if err == nil {
			t.Errorf("expected error containing %q for %q, got nil", tc.errContains, tc.input)
			continue
		}
		if !strings.Contains(err.Error(), tc.errContains) {
			t.Errorf("expected error containing %q for %q, got %v", tc.errContains, tc.input, err)
		}
	}
}

func TestTaylor(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"taylor(sin(x), x, 0, 3)", "x - x^3/6"},
		{"taylor(cos(x), x, 0, 4)", "1 - x^2/2 + x^4/24"},
		{"taylor(1/(1 - x), x, 0, 3)", "1 + x + x^2 + x^3"},
		{"taylor(x^3 + 2*x + 1, x, 0, 2)", "1 + 2*x"},
	}

	for _, tc := range cases {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tc.input, err)
			continue
		}
		formatted := Format(res)
		if formatted != tc.expected {
			t.Errorf("taylor %q: expected %q, got %q", tc.input, tc.expected, formatted)
		}
	}

	// Error cases
	errorCases := []struct {
		input       string
		errContains string
	}{
		{"taylor(sin(x), x, 0, -1)", "order must be non-negative"},
		{"taylor(sin(x), 1, 0, 2)", "second argument must be a variable"},
	}
	for _, tc := range errorCases {
		_, err := EvalString(tc.input)
		if err == nil {
			t.Errorf("expected error for %q, got nil", tc.input)
			continue
		}
		if !strings.Contains(err.Error(), tc.errContains) {
			t.Errorf("expected error containing %q for %q, got %v", tc.errContains, tc.input, err)
		}
	}
}

func TestSum(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"sum(k, k, 1, 10)", "55"},
		{"sum(k^2, k, 1, 5)", "55"},
		{"sum(2^k, k, 0, 4)", "31"},
		{"sum(1/k, k, 1, 4)", "25/12"},
		{"sum(1, k, 1, n)", "n"},
		{"sum(k, k, 1, n)", "n/2 + n^2/2"},
		{"sum(k^2, k, 1, n)", "n/6 + n^2/2 + n^3/3"},
	}

	for _, tc := range cases {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tc.input, err)
			continue
		}
		formatted := Format(res)
		if formatted != tc.expected {
			t.Errorf("sum %q: expected %q, got %q", tc.input, tc.expected, formatted)
		}
	}

	// Error cases
	errorCases := []struct {
		input       string
		errContains string
	}{
		{"sum(k, 1, 1, 10)", "second argument must be a variable"},
		{"sum(k, k, 5, 2)", "start value exceeds end value"},
	}
	for _, tc := range errorCases {
		_, err := EvalString(tc.input)
		if err == nil {
			t.Errorf("expected error for %q, got nil", tc.input)
			continue
		}
		if !strings.Contains(err.Error(), tc.errContains) {
			t.Errorf("expected error containing %q for %q, got %v", tc.errContains, tc.input, err)
		}
	}
}

func TestVectorCalculus(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"dot([1, 2, 3], [4, 5, 6])", "32"},
		{"cross([1, 0, 0], [0, 1, 0])", "[0, 0, 1]"},
		{"cross([1, 2, 3], [4, 5, 6])", "[-3, 6, -3]"},
		{"norm([3, 4])", "5"},
		{"norm([1, 1, 1])", "√3"},
		{"norm([1, 2, 2])", "3"},
		{"grad(x^2 + y^2 + z^2, [x, y, z])", "[2*x, 2*y, 2*z]"},
		{"div([x^2, y^2, z^2], [x, y, z])", "2*x + 2*y + 2*z"},
		{"curl([y, -x, 0], [x, y, z])", "[0, 0, -2]"},
	}

	for _, tc := range cases {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Errorf("eval error for %q: %v", tc.input, err)
			continue
		}
		formatted := Format(res)
		if formatted != tc.expected {
			t.Errorf("vector calc %q: expected %q, got %q", tc.input, tc.expected, formatted)
		}
	}

	// Error cases
	errorCases := []struct {
		input       string
		errContains string
	}{
		{"dot([1, 2], [1, 2, 3])", "dimension mismatch"},
		{"cross([1, 2], [3, 4])", "cross product requires 3-dimensional vectors"},
		{"curl([x, y], [x, y])", "curl requires 3-dimensional vectors"},
	}
	for _, tc := range errorCases {
		_, err := EvalString(tc.input)
		if err == nil {
			t.Errorf("expected error for %q, got nil", tc.input)
			continue
		}
		if !strings.Contains(err.Error(), tc.errContains) {
			t.Errorf("expected error containing %q for %q, got %v", tc.errContains, tc.input, err)
		}
	}
}

func TestGeometryBuiltins(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"line_intersect([1, 1, -4], [1, -1, 0])", "[2, 2]"},
		{"triangle_area([0, 0], [4, 0], [0, 3])", "6"},
		{"triangle_centers([0, 0], [4, 0], [0, 3])", "[[4/3, 1], [2, 3/2], [0, 0], [1, 1]]"},
		{"circle_intersect([0, 0], 1, [3, 0], 2)", "[[1, 0]]"},
		{"circle_intersect([0, 0], 2, [0, 0], 1)", "[]"},
	}

	for _, tc := range cases {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tc.input, err)
		}
		formatted := Format(res)
		if formatted != tc.expected {
			t.Errorf("geometry %q: expected %q, got %q", tc.input, tc.expected, formatted)
		}
	}

	// Two circles intersection with radicals
	resCircles, err := EvalString("circle_intersect([0, 0], 2, [2, 0], 2)")
	if err != nil {
		t.Fatalf("unexpected error for circle_intersect: %v", err)
	}
	listNode, ok := resCircles.(*ListNode)
	if !ok || len(listNode.Elements) != 2 {
		t.Fatalf("expected list of 2 points, got %s", Format(resCircles))
	}

	// Error cases
	errorCases := []struct {
		input       string
		errContains string
	}{
		{"line_intersect([1, 1, 1], [1, 1, 5])", "lines are parallel"},
		{"triangle_centers([0, 0], [1, 1], [2, 2])", "points are collinear"},
		{"circle_intersect([0, 0], 2, [0, 0], 2)", "circles are coincident"},
	}
	for _, tc := range errorCases {
		_, err := EvalString(tc.input)
		if err == nil {
			t.Errorf("expected error for %q, got nil", tc.input)
			continue
		}
		if !strings.Contains(err.Error(), tc.errContains) {
			t.Errorf("expected error containing %q for %q, got %v", tc.errContains, tc.input, err)
		}
	}
}




