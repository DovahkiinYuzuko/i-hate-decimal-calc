package calc

import (
	"strings"
	"testing"
)

func TestArg_Axis(t *testing.T) {
	tests := []struct {
		input       string
		expected    string
		expectError bool
	}{
		{"arg(5)", "0", false},
		{"arg(-3)", "π", false},
		{"arg(2*i)", "π/2", false},
		{"arg(-4*i)", "-π/2", false},
		{"arg(0)", "", true},
		{"arg(0 + 0*i)", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res, err := EvalString(tt.input)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error for %s, got result %v", tt.input, res)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			resStr := Format(res)
			if resStr != tt.expected {
				t.Errorf("got %q, want %q", resStr, tt.expected)
			}
		})
	}
}

func TestArg_Quadrants_SpecialAngles(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Quadrant 1
		{"arg(1 + i)", "π/4"},
		{"arg(sqrt(3) + i)", "π/6"},
		{"arg(1 + sqrt(3)*i)", "π/3"},

		// Quadrant 2
		{"arg(-1 + i)", "3/4*π"},
		{"arg(-sqrt(3) + i)", "5/6*π"},
		{"arg(-1 + sqrt(3)*i)", "2/3*π"},

		// Quadrant 3
		{"arg(-1 - i)", "-3/4*π"},
		{"arg(-sqrt(3) - i)", "-5/6*π"},
		{"arg(-1 - sqrt(3)*i)", "-2/3*π"},

		// Quadrant 4
		{"arg(1 - i)", "-π/4"},
		{"arg(sqrt(3) - i)", "-π/6"},
		{"arg(1 - sqrt(3)*i)", "-π/3"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			resStr := Format(res)
			if resStr != tt.expected {
				t.Errorf("got %q, want %q", resStr, tt.expected)
			}
		})
	}
}

func TestPolar(t *testing.T) {
	tests := []struct {
		input         string
		expectedParts []string
	}{
		{
			"polar(1 + i)",
			[]string{"√2", "cos(π/4)", "sin(π/4)"},
		},
		{
			"polar(2*i)",
			[]string{"2", "cos(π/2)", "sin(π/2)"},
		},
		{
			"polar(-3)",
			[]string{"3", "cos(π)", "sin(π)"},
		},
		{
			"polar(0)",
			[]string{"0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			resStr := Format(res)
			for _, part := range tt.expectedParts {
				if !strings.Contains(resStr, part) {
					t.Errorf("expected result %q to contain %q", resStr, part)
				}
			}
		})
	}
}

func TestPolarExp(t *testing.T) {
	tests := []struct {
		input         string
		expectedParts []string
	}{
		{
			"polar_exp(1 + i)",
			[]string{"√2", "e^i*1/4*π"},
		},
		{
			"polar_exp(2*i)",
			[]string{"2", "e^i*1/2*π"},
		},
		{
			"polar_exp(-3)",
			[]string{"3", "e^i*π"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			resStr := Format(res)
			for _, part := range tt.expectedParts {
				if !strings.Contains(resStr, part) {
					t.Errorf("expected result %q to contain %q", resStr, part)
				}
			}
		})
	}
}

func TestRect(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"rect(sqrt(2), pi/4)", "1 + i"},
		{"rect(2, pi)", "-2"},
		{"rect(3, pi/2)", "3*i"},
		{"rect(2, 2/3*pi)", "-1 + √3*i"},
		{"rect(2, -pi/2)", "-3*i"}, // wait, 2 * -i = -2*i
	}

	// Fix the last test case: rect(2, -pi/2) -> -2*i
	tests[4].expected = "-2*i"

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			resStr := Format(res)
			if resStr != tt.expected {
				t.Errorf("got %q, want %q", resStr, tt.expected)
			}
		})
	}
}
