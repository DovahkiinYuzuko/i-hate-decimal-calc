package calc

import (
	"math"
	"testing"
)

func TestEvalIntegration_Polynomial(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Constant
		{"integrate(5, x)", "5*x"},
		// Monomial
		{"integrate(x, x)", "x^2/2"},
		// Polynomial
		{"integrate(3*x^2 + 2*x + 1, x)", "x + x^2 + x^3"},
		// Expanded polynomial
		{"integrate((x + 1)*(x + 2), x)", "2*x + 3/2*x^2 + x^3/3"},
		// Definite integral of polynomial
		{"integrate(x, x, 0, 2)", "2"},
		{"integrate(3*x^2, x, 0, 2)", "8"},
		{"integrate(x^2, x, 1, 3)", "26/3"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) failed: %v", tt.input, err)
			}
			got, err := Eval(node)
			if err != nil {
				t.Fatalf("Eval(%q) failed: %v", tt.input, err)
			}
			formatted := Format(got)
			if formatted != tt.want {
				t.Errorf("Format(Eval(%q)) = %q, want %q", tt.input, formatted, tt.want)
			}
		})
	}
}

func TestEvalIntegration_Trig(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Indefinite
		{"integrate(cos(x), x)", "sin(x)"},
		{"integrate(sin(x), x)", "-cos(x)"},
		{"integrate(cos(2*x), x)", "sin(2*x)/2"},
		{"integrate(sin(3*x), x)", "-cos(3*x)/3"},
		// Definite
		{"integrate(cos(x), x, 0, pi/2)", "1"},
		{"integrate(sin(x), x, 0, pi)", "2"},
		{"integrate(cos(2*x), x, 0, pi/4)", "1/2"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) failed: %v", tt.input, err)
			}
			got, err := Eval(node)
			if err != nil {
				t.Fatalf("Eval(%q) failed: %v", tt.input, err)
			}
			formatted := Format(got)
			if formatted != tt.want {
				t.Errorf("Format(Eval(%q)) = %q, want %q", tt.input, formatted, tt.want)
			}
		})
	}
}

func TestEvalIntegration_RationalAndExp(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Rational: 1/x -> ln(x)
		{"integrate(x^(-1), x)", "ln(x)"},
		{"integrate(x^(-1), x, 1, e)", "1"},
		// Exponential: e^x
		{"integrate(e^x, x)", "e^x"},
		{"integrate(e^(2*x), x)", "e^(2*x)/2"},
		{"integrate(e^x, x, 0, 1)", "-1 + e"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) failed: %v", tt.input, err)
			}
			got, err := Eval(node)
			if err != nil {
				t.Fatalf("Eval(%q) failed: %v", tt.input, err)
			}
			formatted := Format(got)
			if formatted != tt.want {
				t.Errorf("Format(Eval(%q)) = %q, want %q", tt.input, formatted, tt.want)
			}
		})
	}
}

func TestEvalIntegration_Parts(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// x * cos(x)
		{"integrate(x * cos(x), x)", "cos(x) + x*sin(x)"},
		{"integrate(x * cos(x), x, 0, pi/2)", "-1 + π/2"},
		// x * e^x
		{"integrate(x * e^x, x)", "e^x*x - e^x"},
		{"integrate(x * e^x, x, 0, 1)", "1"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) failed: %v", tt.input, err)
			}
			got, err := Eval(node)
			if err != nil {
				t.Fatalf("Eval(%q) failed: %v", tt.input, err)
			}
			formatted := Format(got)
			if formatted != tt.want {
				t.Errorf("Format(Eval(%q)) = %q, want %q", tt.input, formatted, tt.want)
			}
		})
	}
}

func TestEvalIntegration_Errors(t *testing.T) {
	invalidCases := []string{
		"integrate(x)",             // too few args
		"integrate(x, 1)",          // 2nd arg not variable
		"integrate(x, x, 0)",       // 3 args invalid
		"integrate(x, x, 0, 1, 2)", // too many args
	}

	for _, input := range invalidCases {
		t.Run(input, func(t *testing.T) {
			node, err := Parse(input)
			if err == nil {
				_, err = Eval(node)
			}
			if err == nil {
				t.Errorf("Expected error for %q, got nil", input)
			}
		})
	}
}

func TestEvalIntegration_LogBaseConsistency(t *testing.T) {
	// Deductive & Metamorphic round-trip: d/dx(integrate(log(x), x)) at x=10 is 1, at x=100 is 2
	cases := []struct {
		expr    string
		xVal    float64
		wantDer float64
	}{
		{"integrate(log(x), x)", 10, 1},
		{"integrate(log(x), x)", 100, 2},
		{"integrate(log(2*x + 1), x)", 49.5, 2}, // log10(100) = 2
		{"integrate(log(2, x), x)", 8, 3},
		{"integrate(log(2, 2*x + 2), x)", 3, 3}, // log2(8) = 3
	}

	for _, tc := range cases {
		F, err := EvalString(tc.expr)
		if err != nil {
			t.Fatalf("EvalString(%q) failed: %v", tc.expr, err)
		}
		dF, err := differentiate(F, "x")
		if err != nil {
			t.Fatalf("differentiate(%s) failed: %v", F.String(), err)
		}
		got, ok := evalSampleFloat(dF, "x", tc.xVal)
		if !ok {
			t.Fatalf("evalSampleFloat(dF=%s, x=%v) returned ok=false", dF.String(), tc.xVal)
		}
		if math.Abs(got-tc.wantDer) > 1e-12 {
			t.Errorf("d/dx(%s) at x=%v = %.15g, want %.15g (F=%s, dF=%s)", tc.expr, tc.xVal, got, tc.wantDer, F.String(), dF.String())
		}
	}

	// Contradiction (residue pinning): integrate(ln(x), x) / integrate(log(x), x) at x=3 == ln(10)
	fLn, err := EvalString("integrate(ln(x), x)")
	if err != nil {
		t.Fatalf("integrate(ln(x), x) failed: %v", err)
	}
	fLog, err := EvalString("integrate(log(x), x)")
	if err != nil {
		t.Fatalf("integrate(log(x), x) failed: %v", err)
	}
	vLn, ok1 := evalSampleFloat(fLn, "x", 3)
	vLog, ok2 := evalSampleFloat(fLog, "x", 3)
	if !ok1 || !ok2 {
		t.Fatalf("evalSampleFloat failed for ln/log integrals")
	}
	ratio := vLn / vLog
	if math.Abs(ratio-math.Ln10) > 1e-12 {
		t.Errorf("integrate(ln(x))/integrate(log(x)) at x=3 = %.15g, want ln(10)=%.15g", ratio, math.Ln10)
	}

	// Definite integral: int_1^10 log(x) dx = 10 - 9/ln(10) (~6.091349662870734)
	defNode, err := EvalString("integrate(log(x), x, 1, 10)")
	if err != nil {
		t.Fatalf("integrate(log(x), x, 1, 10) failed: %v", err)
	}
	defVal, err := evalNodeFloat(defNode)
	if err != nil {
		t.Fatalf("evalNodeFloat(%s) failed: %v", defNode.String(), err)
	}
	wantDef := 10.0 - 9.0/math.Ln10
	if math.Abs(defVal-wantDef) > 1e-10 {
		t.Errorf("integrate(log(x), x, 1, 10) = %.15g, want %.15g (node=%s)", defVal, wantDef, defNode.String())
	}
}
