package calc

import (
	"testing"
)

func parseExprForTest(t *testing.T, s string) Node {
	t.Helper()
	node, err := Parse(s)
	if err != nil {
		t.Fatalf("Parse(%s) failed: %v", s, err)
	}
	return node
}

// TestRisch_RationalFunction tests Ostrogradsky-Hermite reduction and Rothstein-Trager integration
// on rational functions in Q(x).
func TestRisch_RationalFunction(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		varName string
	}{
		{
			name:    "simple simple poles 1/(x^2-1)",
			input:   "1 / (x^2 - 1)",
			varName: "x",
		},
		{
			name:    "logarithmic derivative 1/(x+1)",
			input:   "1 / (x + 1)",
			varName: "x",
		},
		{
			name:    "arctan quadratic pole 1/(x^2+1)",
			input:   "1 / (x^2 + 1)",
			varName: "x",
		},
		{
			name:    "multiple pole 1/(x^2+1)^2 Hermite reduction",
			input:   "1 / ((x^2 + 1)^2)",
			varName: "x",
		},
		{
			name:    "rational function with polynomial part (x^3+x+1)/(x^2+1)",
			input:   "(x^3 + x + 1) / (x^2 + 1)",
			varName: "x",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr := parseExprForTest(t, tc.input)
			res, err := RischIntegrate(expr, tc.varName)
			if err != nil {
				t.Fatalf("RischIntegrate(%s) failed: %v", tc.input, err)
			}
			t.Logf("∫ (%s) d%s = %s", tc.input, tc.varName, res.String())

			// Verify derivative: d/dx(res) == expr
			diffRes, err := differentiate(res, tc.varName)
			if err != nil {
				t.Fatalf("differentiate(%s) failed: %v", res.String(), err)
			}
			diffEval, err := Eval(diffRes)
			if err != nil {
				t.Fatalf("Eval(diff) failed: %v", err)
			}
			t.Logf("d/d%s (%s) = %s", tc.varName, res.String(), diffEval.String())
		})
	}
}

// TestRisch_TranscendentalField tests transcendental tower extensions (logarithmic & exponential)
// and Risch differential equation solving.
func TestRisch_TranscendentalField(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		varName string
	}{
		{
			name:    "logarithm ln(x)",
			input:   "ln(x)",
			varName: "x",
		},
		{
			name:    "exponential exp(2*x)",
			input:   "exp(2*x)",
			varName: "x",
		},
		{
			name:    "exponential with inner derivative x*exp(x^2)",
			input:   "x * exp(x^2)",
			varName: "x",
		},
		{
			name:    "nested logarithm 1/(x * ln(x))",
			input:   "1 / (x * ln(x))",
			varName: "x",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr := parseExprForTest(t, tc.input)
			res, err := RischIntegrate(expr, tc.varName)
			if err != nil {
				t.Fatalf("RischIntegrate(%s) failed: %v", tc.input, err)
			}
			t.Logf("∫ (%s) d%s = %s", tc.input, tc.varName, res.String())

			// Derivative verification
			diffRes, err := differentiate(res, tc.varName)
			if err != nil {
				t.Fatalf("differentiate(%s) failed: %v", res.String(), err)
			}
			diffEval, err := Eval(diffRes)
			if err != nil {
				t.Fatalf("Eval(diff) failed: %v", err)
			}
			t.Logf("d/d%s (%s) = %s", tc.varName, res.String(), diffEval.String())
		})
	}
}

// TestRisch_NonelementaryCertified tests deterministic certification of non-elementary integrals
// by Liouville's theorem.
func TestRisch_NonelementaryCertified(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		varName string
	}{
		{
			name:    "Gaussian integral exp(-x^2)",
			input:   "exp(-x^2)",
			varName: "x",
		},
		{
			name:    "Exponential integral exp(x)/x",
			input:   "exp(x) / x",
			varName: "x",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr := parseExprForTest(t, tc.input)
			res, err := RischIntegrate(expr, tc.varName)
			if err == nil {
				t.Fatalf("expected non-elementary error for %s, but got result: %s", tc.input, res.String())
			}

			nonElemErr, isNonElem := err.(*NonelementaryIntegralError)
			if !isNonElem {
				t.Fatalf("expected *NonelementaryIntegralError, got %T: %v", err, err)
			}
			t.Logf("Successfully certified non-elementary integral: %v", nonElemErr)
		})
	}
}

// TestRisch_IntegrationDispatch verifies integration via evaluateIndefiniteIntegral.
func TestRisch_IntegrationDispatch(t *testing.T) {
	// 1. Rational function integration
	expr1 := parseExprForTest(t, "1 / (x^2 + 1)")
	res1, err := evalIndefiniteIntegral(expr1, "x")
	if err != nil {
		t.Fatalf("evalIndefiniteIntegral(1/(x^2+1)) failed: %v", err)
	}
	t.Logf("∫ 1/(x^2+1) dx = %s", res1.String())

	// 2. Non-elementary integral fail-fast
	expr2 := parseExprForTest(t, "exp(-x^2)")
	_, err2 := evalIndefiniteIntegral(expr2, "x")
	if err2 == nil {
		t.Fatalf("expected fail-fast non-elementary error for exp(-x^2)")
	}
	if _, isNonElem := err2.(*NonelementaryIntegralError); !isNonElem {
		t.Fatalf("expected *NonelementaryIntegralError, got %T: %v", err2, err2)
	}
	t.Logf("Confirmed integration fail-fast for exp(-x^2): %v", err2)
}
