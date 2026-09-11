package calc

import (
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
