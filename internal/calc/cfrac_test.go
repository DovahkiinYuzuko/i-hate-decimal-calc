package calc

import (
	"testing"
)

func TestCFrac_Rational(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "cfrac(355/113)",
			expected: "[3, 7, 16]",
		},
		{
			input:    "cfrac(43/19)",
			expected: "[2, 3, 1, 4]",
		},
		{
			input:    "cfrac(5)",
			expected: "[5]",
		},
		{
			input:    "cfrac(0)",
			expected: "[0]",
		},
		{
			input:    "cfrac(1/3)",
			expected: "[0, 3]",
		},
		{
			input:    "cfrac(-7/5)",
			expected: "[-2, 1, 1, 2]",
		},
	}

	for _, tt := range tests {
		ast, err := Parse(tt.input)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", tt.input, err)
		}
		res, err := Eval(ast)
		if err != nil {
			t.Fatalf("Eval(%q) error: %v", tt.input, err)
		}
		if res.String() != tt.expected {
			t.Errorf("Eval(%q) = %s, want %s", tt.input, res.String(), tt.expected)
		}
	}
}

func TestCFrac_Sqrt(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "cfrac(sqrt(2))",
			expected: "[1, [2]]",
		},
		{
			input:    "cfrac(sqrt(3))",
			expected: "[1, [1, 2]]",
		},
		{
			input:    "cfrac(sqrt(7))",
			expected: "[2, [1, 1, 1, 4]]",
		},
		{
			input:    "cfrac(sqrt(19))",
			expected: "[4, [2, 1, 3, 1, 2, 8]]",
		},
		{
			input:    "cfrac(sqrt(4))",
			expected: "[2]",
		},
		{
			input:    "cfrac(sqrt(0))",
			expected: "[0]",
		},
	}

	for _, tt := range tests {
		ast, err := Parse(tt.input)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", tt.input, err)
		}
		res, err := Eval(ast)
		if err != nil {
			t.Fatalf("Eval(%q) error: %v", tt.input, err)
		}
		if res.String() != tt.expected {
			t.Errorf("Eval(%q) = %s, want %s", tt.input, res.String(), tt.expected)
		}
	}
}

func TestFromCFrac(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "from_cfrac([3, 7, 16])",
			expected: "355/113",
		},
		{
			input:    "from_cfrac([2, 3, 1, 4])",
			expected: "43/19",
		},
		{
			input:    "from_cfrac([-2, 1, 1, 2])",
			expected: "-7/5",
		},
		{
			input:    "from_cfrac([0, 3])",
			expected: "1/3",
		},
		{
			input:    "from_cfrac([5])",
			expected: "5",
		},
	}

	for _, tt := range tests {
		ast, err := Parse(tt.input)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", tt.input, err)
		}
		res, err := Eval(ast)
		if err != nil {
			t.Fatalf("Eval(%q) error: %v", tt.input, err)
		}
		if res.String() != tt.expected {
			t.Errorf("Eval(%q) = %s, want %s", tt.input, res.String(), tt.expected)
		}
	}
}

func TestCFrac_Errors(t *testing.T) {
	errorInputs := []string{
		"cfrac(x + 1)",
		"from_cfrac([])",
		"from_cfrac([1, 1/2])",
	}

	for _, input := range errorInputs {
		ast, err := Parse(input)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", input, err)
		}
		_, err = Eval(ast)
		if err == nil {
			t.Errorf("Eval(%q) expected error, got nil", input)
		}
	}
}
