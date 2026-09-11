package calc

import (
	"strings"
	"testing"
)

// TestFormat_UnicodeDefault validates default human-readable Unicode formatting.
func TestFormat_UnicodeDefault(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"sqrt(2)", "√2"},
		{"-sqrt(2)", "-√2"},
		{"2*sqrt(3)", "2*√3"},
		{"1/sqrt(2)", "√2/2"},
		{"pi", "π"},
		{"-pi", "-π"},
		{"2*pi", "2*π"},
		{"e", "e"},
		{"4 + 4*6.441", "7441/250"},
		{"5", "5"},
	}

	for _, tc := range testCases {
		node, err := EvalString(tc.input)
		if err != nil {
			t.Fatalf("eval error for %q: %v", tc.input, err)
		}
		formatted := Format(node)
		if formatted != tc.expected {
			t.Errorf("input %q: expected %q, got %q", tc.input, tc.expected, formatted)
		}
	}
}

// TestFormat_AsciiMode validates --ascii flag output with standard ASCII characters.
func TestFormat_AsciiMode(t *testing.T) {
	opts := FormatOptions{AsciiOnly: true}
	testCases := []struct {
		input    string
		expected string
	}{
		{"sqrt(2)", "sqrt(2)"},
		{"-sqrt(2)", "-sqrt(2)"},
		{"2*sqrt(3)", "2*sqrt(3)"},
		{"1/sqrt(2)", "sqrt(2)/2"},
		{"pi", "pi"},
		{"-pi", "-pi"},
		{"2*pi", "2*pi"},
		{"e", "e"},
	}

	for _, tc := range testCases {
		node, err := EvalString(tc.input)
		if err != nil {
			t.Fatalf("eval error for %q: %v", tc.input, err)
		}
		formatted := FormatWithOptions(node, opts)
		if formatted != tc.expected {
			t.Errorf("input %q: expected %q, got %q", tc.input, tc.expected, formatted)
		}
	}
}

// TestFormat_ComplexNumbers validates complex number formatting.
func TestFormat_ComplexNumbers(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"sqrt(-4)", "2*i"},
		{"sqrt(-1)", "i"},
		{"-sqrt(-1)", "-i"},
		{"3 + 4*i", "3 + 4*i"},
		{"3 - 4*i", "3 - 4*i"},
		{"3 - i", "3 - i"},
		{"3 + i", "3 + i"},
	}

	for _, tc := range testCases {
		node, err := EvalString(tc.input)
		if err != nil {
			t.Fatalf("eval error for %q: %v", tc.input, err)
		}
		formatted := Format(node)
		if formatted != tc.expected {
			t.Errorf("input %q: expected %q, got %q", tc.input, tc.expected, formatted)
		}
	}
}

// TestFormat_CanonicalOrdering validates term ordering: Rational -> Sqrt -> Const -> Func.
func TestFormat_CanonicalOrdering(t *testing.T) {
	// sin(1) + pi + sqrt(2) + 5
	node, err := EvalString("sin(1) + pi + sqrt(2) + 5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "5 + √2 + π + sin(1)"
	formatted := Format(node)
	if formatted != expected {
		t.Errorf("expected %q, got %q", expected, formatted)
	}
}

func TestFormatLaTeX(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"42", "$$ 42 $$"},
		{"3/4", "$$ \\frac{3}{4} $$"},
		{"-3/4", "$$ -\\frac{3}{4} $$"},
		{"sqrt(2)", "$$ \\sqrt{2} $$"},
		{"3*sqrt(2)", "$$ 3\\sqrt{2} $$"},
		{"1 + 2*i", "$$ 1 + 2i $$"},
		{"1 - 2*i", "$$ 1 - 2i $$"},
		{"1/2 + sqrt(2)", "$$ \\frac{1}{2} + \\sqrt{2} $$"},
		{"pi", "$$ \\pi $$"},
		{"e", "$$ e $$"},
		{"log(2, 3)", "$$ \\log_{2}(3) $$"}, // log(2, 3) cannot simplify
		{"sin(1)", "$$ \\sin(1) $$"},
	}

	for _, tc := range testCases {
		node, err := EvalString(tc.input)
		if err != nil {
			t.Fatalf("eval error for %q: %v", tc.input, err)
		}
		actual := FormatLaTeX(node)
		if actual != tc.expected {
			t.Errorf("input %q: expected %q, got %q", tc.input, tc.expected, actual)
		}
	}

	// Test unsimplified AST (PowNode)
	powNode, err := NewPow(mustRational(2, 1), mustRational(3, 1))
	if err != nil {
		t.Fatalf("unexpected NewPow error: %v", err)
	}
	if actual := FormatLaTeX(powNode); actual != "$$ 2^{3} $$" {
		t.Errorf("expected '$$ 2^{3} $$', got %q", actual)
	}
}

func TestFormatPretty2D(t *testing.T) {
	// 1. Integer should be 1 line
	nInt := mustRational(42, 1)
	if actual := FormatPretty2D(nInt); actual != "42" {
		t.Errorf("expected '42', got %q", actual)
	}

	// 2. Simple fraction 3/4 should be 3 lines with horizontal bar
	nFrac := mustRational(3, 4)
	expectedFrac := " 3 \n---\n 4 "
	if actual := FormatPretty2D(nFrac); actual != expectedFrac {
		t.Errorf("expected:\n%s\ngot:\n%s", expectedFrac, actual)
	}

	// 3. Fraction with Sqrt: (1 + sqrt(2))/2
	// Evaluated: 1/2 + √2/2 -> AddNode with two fractions
	node, err := EvalString("1/2 + sqrt(2)/2")
	if err != nil {
		t.Fatalf("unexpected eval error: %v", err)
	}
	actual := FormatPretty2D(node)
	if !strings.Contains(actual, "---") || !strings.Contains(actual, "√2") {
		t.Errorf("expected pretty fraction with √2, got:\n%s", actual)
	}
}


