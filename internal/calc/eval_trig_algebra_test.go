package calc

import (
	"testing"
)

func TestTrigExpand(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Addition formulas
		{"trig_expand(sin(x + y))", "cos(x)*sin(y) + sin(x)*cos(y)"},
		{"trig_expand(cos(x + y))", "-1*sin(x)*sin(y) + cos(x)*cos(y)"},
		{"trig_expand(sin(x - y))", "-1*cos(x)*sin(y) + sin(x)*cos(y)"},
		{"trig_expand(cos(x - y))", "cos(x)*cos(y) + sin(x)*sin(y)"},

		// Double angle formulas
		{"trig_expand(sin(2*x))", "2*sin(x)*cos(x)"},
		{"trig_expand(cos(2*x))", "-sin(x)^2 + cos(x)^2"},
		{"trig_expand(tan(2*x))", "2*tan(x)*(1 - tan(x)^2)^-1"},

		// Triple angle formulas
		{"trig_expand(sin(3*x))", "3*cos(x)^2*sin(x) - sin(x)^3"},
		{"trig_expand(cos(3*x))", "-3*cos(x)*sin(x)^2 + cos(x)^3"},

		// Combined expression
		{"trig_expand(sin(2*x) + cos(2*x))", "2*sin(x)*cos(x) - sin(x)^2 + cos(x)^2"},
	}

	env := NewEnv()
	for _, tt := range tests {
		node, err := Parse(tt.input)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", tt.input, err)
		}
		res, err := EvalWithEnv(node, env)
		if err != nil {
			t.Fatalf("EvalWithEnv(%q) error: %v", tt.input, err)
		}
		got := Format(res)
		if got != tt.expected {
			t.Errorf("EvalWithEnv(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestTrigReduce(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Power reduction (half-angle reverse)
		{"trig_reduce(sin(x)^2)", "1/2 - cos(2*x)/2"},
		{"trig_reduce(cos(x)^2)", "1/2 + cos(2*x)/2"},

		// Product-to-sum
		{"trig_reduce(sin(x) * cos(x))", "sin(2*x)/2"},
		{"trig_reduce(cos(x) * sin(x))", "sin(2*x)/2"},
		{"trig_reduce(2 * sin(x) * cos(x))", "sin(2*x)"},

		// Harmonic addition (sin-cos linear combination)
		{"trig_reduce(sin(x) + cos(x))", "√2*sin(x + π/4)"},
		{"trig_reduce(sin(x) - cos(x))", "√2*sin(x - π/4)"},
		{"trig_reduce(-sin(x) + cos(x))", "√2*sin(x + 3/4*π)"},
		{"trig_reduce(-sin(x) - cos(x))", "√2*sin(x - 3/4*π)"},

		{"trig_reduce(sqrt(3)*sin(x) + cos(x))", "2*sin(x + π/6)"},
		{"trig_reduce(sin(x) + sqrt(3)*cos(x))", "2*sin(x + π/3)"},
		{"trig_reduce(sqrt(3)*sin(x) - cos(x))", "2*sin(x - π/6)"},
		{"trig_reduce(sin(x) - sqrt(3)*cos(x))", "2*sin(x - π/3)"},

		// Combination of power and harmonic
		{"trig_reduce(cos(x)^2 - sin(x)^2)", "cos(2*x)"},
	}

	env := NewEnv()
	for _, tt := range tests {
		node, err := Parse(tt.input)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", tt.input, err)
		}
		res, err := EvalWithEnv(node, env)
		if err != nil {
			t.Fatalf("EvalWithEnv(%q) error: %v", tt.input, err)
		}
		got := Format(res)
		if got != tt.expected {
			t.Errorf("EvalWithEnv(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}
