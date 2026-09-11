package calc

import (
	"strings"
	"testing"
)

func TestApprox_Basic(t *testing.T) {
	testCases := []struct {
		input    string
		contains string
	}{
		{"1/2", "0.5"},
		{"1/3", "0.333333333333333"},
		{"sqrt(2)", "1.414213562373095"},
		{"pi", "3.141592653589793"},
		{"e", "2.718281828459045"},
		{"sin(pi/2)", "1"},
		{"cos(0)", "1"},
		{"ln(e)", "1"},
		{"log(100)", "2"},
		{"2^10", "1024"},
		{"5!", "120"},
		{"sqrt(-4)", "2i"},
		{"3 + 4*i", "3 + 4i"},
		{"3 - 4*i", "3 - 4i"},
	}

	for _, tc := range testCases {
		node, err := EvalString(tc.input)
		if err != nil {
			t.Fatalf("eval error for %q: %v", tc.input, err)
		}
		app, err := Approx(node)
		if err != nil {
			t.Fatalf("approx error for %q: %v", tc.input, err)
		}
		if !strings.Contains(app, tc.contains) {
			t.Errorf("input %q: expected approx to contain %q, got %q", tc.input, tc.contains, app)
		}
	}
}
