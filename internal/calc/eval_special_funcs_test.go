package calc

import (
	"testing"
)

func TestEvalSpecialFuncs(t *testing.T) {
	tests := []struct {
		input       string
		expected    string
		expectError bool
	}{
		// 1. Gamma function
		{"gamma(1)", "1", false},
		{"gamma(2)", "1", false},
		{"gamma(5)", "24", false},
		{"gamma(1/2)", "sqrt(pi)", false},
		{"gamma(3/2)", "sqrt(pi)/2", false},
		{"gamma(5/2)", "3/4 * sqrt(pi)", false},
		{"gamma(-1/2)", "-2 * sqrt(pi)", false},
		{"gamma(-3/2)", "4/3 * sqrt(pi)", false},
		{"gamma(0)", "", true},
		{"gamma(-1)", "", true},
		{"gamma(x)", "gamma(x)", false},

		// 2. Beta function
		{"beta(1, 1)", "1", false},
		{"beta(2, 3)", "1/12", false},
		{"beta(1/2, 1/2)", "pi", false},
		{"beta(1/2, 1)", "2", false},
		{"beta(x, y)", "beta(x, y)", false},

		// 3. Bernoulli numbers
		{"bernoulli(0)", "1", false},
		{"bernoulli(1)", "-1/2", false},
		{"bernoulli(2)", "1/6", false},
		{"bernoulli(3)", "0", false},
		{"bernoulli(4)", "-1/30", false},
		{"bernoulli(5)", "0", false},
		{"bernoulli(6)", "1/42", false},
		{"bernoulli(10)", "5/66", false},
		{"bernoulli(-1)", "", true},

		// 4. Riemann zeta function
		{"zeta(0)", "-1/2", false},
		{"zeta(2)", "pi^2/6", false},
		{"zeta(4)", "pi^4/90", false},
		{"zeta(6)", "pi^6/945", false},
		{"zeta(1)", "", true},
		{"zeta(3)", "zeta(3)", false},
		{"zeta(s)", "zeta(s)", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error for %s: %v", tt.input, err)
			}
			env := NewEnv()
			res, err := EvalWithEnv(node, env)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error for %s, but got %v", tt.input, res)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			if res == nil {
				t.Fatalf("got nil result for %s", tt.input)
			}
			got := res.String()
			if got != tt.expected {
				t.Errorf("for %s, expected %q, got %q", tt.input, tt.expected, got)
			}
		})
	}
}
