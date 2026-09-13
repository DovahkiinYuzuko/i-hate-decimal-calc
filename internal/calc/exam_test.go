package calc

import (
	"strings"
	"testing"
)

func evalExpr(t *testing.T, expr string) string {
	t.Helper()
	ast, err := Parse(expr)
	if err != nil {
		t.Fatalf("parse error for %q: %v", expr, err)
	}
	env := NewEnv()
	res, err := EvalWithEnv(ast, env)
	if err != nil {
		t.Fatalf("eval error for %q: %v", expr, err)
	}
	return Format(res)
}

func TestHighSchoolCookbook(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected string
	}{
		{
			name:     "01_SymmetricIdentity",
			expr:     "(3)^3 - 3*(1)*(3)",
			expected: "18",
		},
		{
			name:     "02_RadicalDenestingAndRationalization",
			expr:     "1 / sqrt(5 - 2*sqrt(6))",
			expected: "√2 + √3",
		},
		{
			name:     "03_GcdLcmRelation",
			expr:     "gcd(123456, 789012) * lcm(123456, 789012) - 123456 * 789012",
			expected: "0",
		},
		{
			name:     "04_RepeatedTrialsProbability",
			expr:     "comb(10, 3) * (1/6)^3 * (5/6)^7",
			expected: "390625/2519424",
		},
		{
			name:     "05_ComplexPolarForm",
			expr:     "polar(1 + i)",
			expected: "√2*(cos(π/4) + i*sin(π/4))",
		},
		{
			name:     "06_ExactTrigAdditionIdentity",
			expr:     "sin(pi/3) * cos(pi/6) + cos(pi/3) * sin(pi/6)",
			expected: "1",
		},
		{
			name:     "07_MultiBaseLogarithm",
			expr:     "log(2, 8) + log(3, 27) - log(10, 1000)",
			expected: "3",
		},
		{
			name:     "08_FaulhaberSumOfCubes",
			expr:     "sum(k^3, k, 1, n)",
			expected: "n^2/4 + n^3/2 + n^4/4",
		},
		{
			name:     "09_SymbolicProductRuleDerivative",
			expr:     "diff(x * sin(x), x)",
			expected: "cos(x)*x + sin(x)",
		},
		{
			name:     "10_MaclaurinSeriesExpansion",
			expr:     "taylor(sin(x), x, 0, 5)",
			expected: "x - x^3/6 + x^5/120",
		},
		{
			name:     "11_VectorCrossProductOrthogonality",
			expr:     "dot([1, 2, 3], cross([1, 2, 3], [4, 5, 6]))",
			expected: "0",
		},
		{
			name:     "12_LineCircleIntersection",
			expr:     "solve(2*x^2 + 2*x, x)",
			expected: "[-1, 0]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evalExpr(t, tt.expr)
			if got != tt.expected {
				t.Errorf("expr %q: got %q, want %q", tt.expr, got, tt.expected)
			}
		})
	}
}

func TestAdvancedCompetitionCookbook(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected string
	}{
		{
			name:     "13_PellsEquationMinimalSolution",
			expr:     "1766319049^2 - 61 * 226153980^2",
			expected: "1",
		},
		{
			name:     "14_Mersenne67ColeFactorization",
			expr:     "2^67 - 1 - 193707721 * 761838257287",
			expected: "0",
		},
		{
			name:     "15_ChineseRemainderTheoremSystem",
			expr:     "[mod(23, 3), mod(23, 5), mod(23, 7)]",
			expected: "[2, 3, 2]",
		},
		{
			name:     "16_ExactComplexArgument",
			expr:     "arg(1 + i*sqrt(3))",
			expected: "π/3",
		},
		{
			name:     "17_RadicalDenesting",
			expr:     "sqrt(7 + 2*sqrt(10))",
			expected: "√2 + √5",
		},
		{
			name:     "18_PolynomialExpansionIdentity",
			expr:     "expand((x + 1)^3 - (x^3 + 3*x^2 + 3*x + 1))",
			expected: "0",
		},
		{
			name:     "19_ContinuedFractionPeriodicity",
			expr:     "cfrac(sqrt(2))",
			expected: "[1, [2]]",
		},
		{
			name:     "20_VandermondeDeterminantEvaluation",
			expr:     "det([[1, 1, 1], [1, 2, 3], [1^2, 2^2, 3^2]])",
			expected: "2",
		},
		{
			name:     "21_CirculantMatrixInverse",
			expr:     "inv([[1, 2, 3], [3, 1, 2], [2, 3, 1]])",
			expected: "[[-5/18, 7/18, 1/18], [1/18, -5/18, 7/18], [7/18, 1/18, -5/18]]",
		},
		{
			name:     "22_HarmonicPotentialLaplacian",
			expr:     "diff(diff(x^3 - 3*x*y^2, x), x) + diff(diff(x^3 - 3*x*y^2, y), y)",
			expected: "0",
		},
		{
			name:     "23_EulerLineTriangleCenters",
			expr:     "triangle_centers([0, 0], [4, 0], [0, 3])",
			expected: "[[4/3, 1], [2, 3/2], [0, 0], [1, 1]]",
		},
		{
			name:     "24_CatalanNumber",
			expr:     "1/6 * comb(10, 5)",
			expected: "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evalExpr(t, tt.expr)
			if got != tt.expected {
				t.Errorf("expr %q: got %q, want %q", tt.expr, got, tt.expected)
			}
		})
	}
}

func TestExplainerShowcase(t *testing.T) {
	// Test --explain functionality for pedagogical steps
	ast, err := Parse("1 / sqrt(5 - 2*sqrt(6))")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	res, steps, err := EvalWithTrace(ast)
	if err != nil {
		t.Fatalf("eval with trace error: %v", err)
	}
	if len(steps) == 0 {
		t.Errorf("expected pedagogical steps for denesting + rationalization, got none")
	}

	formatted := FormatTrace("1 / sqrt(5 - 2*sqrt(6))", steps, res)
	if !strings.Contains(formatted, "√2 + √3") {
		t.Errorf("formatted trace missing final result: %s", formatted)
	}
}
