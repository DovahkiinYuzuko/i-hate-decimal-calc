package calc

import (
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
		{"integrate(e^(2*x), x)", "e^2*x/2"},
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
