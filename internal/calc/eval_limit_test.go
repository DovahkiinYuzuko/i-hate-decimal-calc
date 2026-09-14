package calc

import (
	"testing"
)

func TestEvalLimit_DirectSubstitution(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"limit(x^2 + 3*x - 1, x, 2)", "9"},
		{"limit(sin(x), x, pi/6)", "1/2"},
		{"limit(cos(x), x, 0)", "1"},
		{"limit(e^x, x, 0)", "1"},
	}

	for _, tt := range tests {
		ast, err := Parse(tt.input)
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", tt.input, err)
		}
		res, err := Eval(ast)
		if err != nil {
			t.Fatalf("Eval(%q) failed: %v", tt.input, err)
		}
		got := Format(res)
		if got != tt.expected {
			t.Errorf("Eval(%q) = %s; want %s", tt.input, got, tt.expected)
		}
	}
}

func TestEvalLimit_FactorCancellation(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"limit((x^2 - 1)/(x - 1), x, 1)", "2"},
		{"limit((x^2 - 4)/(x - 2), x, 2)", "4"},
		{"limit((x^3 - 8)/(x - 2), x, 2)", "12"},
		{"limit((x^2 - 5*x + 6)/(x - 2), x, 2)", "-1"},
	}

	for _, tt := range tests {
		ast, err := Parse(tt.input)
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", tt.input, err)
		}
		res, err := Eval(ast)
		if err != nil {
			t.Fatalf("Eval(%q) failed: %v", tt.input, err)
		}
		got := Format(res)
		if got != tt.expected {
			t.Errorf("Eval(%q) = %s; want %s", tt.input, got, tt.expected)
		}
	}
}

func TestEvalLimit_LHopital(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"limit(sin(x)/x, x, 0)", "1"},
		{"limit((1 - cos(x))/(x^2), x, 0)", "1/2"},
		{"limit((e^x - 1)/x, x, 0)", "1"},
		{"limit((x - sin(x))/(x^3), x, 0)", "1/6"},
		{"limit(sin(3*x)/sin(2*x), x, 0)", "3/2"},
	}

	for _, tt := range tests {
		ast, err := Parse(tt.input)
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", tt.input, err)
		}
		res, err := Eval(ast)
		if err != nil {
			t.Fatalf("Eval(%q) failed: %v", tt.input, err)
		}
		got := Format(res)
		if got != tt.expected {
			t.Errorf("Eval(%q) = %s; want %s", tt.input, got, tt.expected)
		}
	}
}

func TestEvalLimit_Infinity(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"limit((2*x^2 + 3)/(5*x^2 - 1), x, inf)", "2/5"},
		{"limit(1/x, x, inf)", "0"},
		{"limit((3*x + 1)/(x^2 + 2), x, inf)", "0"},
		{"limit(x^2 / (x + 1), x, inf)", "inf"},
		{"limit(ln(x)/x, x, inf)", "0"},
		{"limit(x/e^x, x, inf)", "0"},
	}

	for _, tt := range tests {
		ast, err := Parse(tt.input)
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", tt.input, err)
		}
		res, err := Eval(ast)
		if err != nil {
			t.Fatalf("Eval(%q) failed: %v", tt.input, err)
		}
		got := Format(res)
		if got != tt.expected {
			t.Errorf("Eval(%q) = %s; want %s", tt.input, got, tt.expected)
		}
	}
}

func TestEvalLimit_OneSided(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"limit(1/x, x, 0, 1)", "inf"},
		{"limit(1/x, x, 0, -1)", "-inf"},
		{"limit(1/x^2, x, 0)", "inf"},
	}

	for _, tt := range tests {
		ast, err := Parse(tt.input)
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", tt.input, err)
		}
		res, err := Eval(ast)
		if err != nil {
			t.Fatalf("Eval(%q) failed: %v", tt.input, err)
		}
		got := Format(res)
		if got != tt.expected {
			t.Errorf("Eval(%q) = %s; want %s", tt.input, got, tt.expected)
		}
	}
}

func TestEvalLimit_Explain(t *testing.T) {
	ast, err := Parse("limit((x^2 - 1)/(x - 1), x, 1)")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	res, steps, err := EvalWithTrace(ast)
	if err != nil {
		t.Fatalf("EvalWithTrace failed: %v", err)
	}
	if Format(res) != "2" {
		t.Fatalf("expected 2, got %s", Format(res))
	}
	if len(steps) == 0 {
		t.Fatalf("expected rewrite steps, got 0")
	}
	output := FormatTrace("limit((x^2 - 1)/(x - 1), x, 1)", steps, res)
	if len(output) == 0 {
		t.Fatalf("FormatTrace output is empty")
	}
}
