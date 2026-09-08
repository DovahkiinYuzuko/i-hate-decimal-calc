package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLI_OneShot(t *testing.T) {
	testCases := []struct {
		args       []string
		expected   string
		expectCode int
	}{
		{[]string{"1/2 + 1/3"}, "5/6", 0},
		{[]string{"sqrt(2) + sqrt(8)"}, "3*√2", 0},
		{[]string{"--ascii", "sqrt(2) + pi"}, "sqrt(2) + pi", 0},
		{[]string{"--ascii", "sqrt(-4)"}, "2*i", 0},
		{[]string{"-2^3"}, "-8", 0},
		{[]string{"--ascii", "-3^2"}, "-9", 0},
	}

	for _, tc := range testCases {
		out := new(bytes.Buffer)
		errOut := new(bytes.Buffer)
		code := run(tc.args, strings.NewReader(""), out, errOut)

		if code != tc.expectCode {
			t.Errorf("args %v: expected exit code %d, got %d. stderr: %s", tc.args, tc.expectCode, code, errOut.String())
		}
		actual := strings.TrimSpace(out.String())
		if actual != tc.expected {
			t.Errorf("args %v: expected stdout %q, got %q", tc.args, tc.expected, actual)
		}
	}
}

func TestCLI_OneShot_Approx(t *testing.T) {
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	code := run([]string{"--approx", "sqrt(2)"}, strings.NewReader(""), out, errOut)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, errOut.String())
	}
	actual := strings.TrimSpace(out.String())
	if !strings.HasPrefix(actual, "√2 (≈ 1.41421356") {
		t.Errorf("expected approx output prefix '√2 (≈ 1.41421356', got %q", actual)
	}
}

func TestCLI_OneShot_Errors(t *testing.T) {
	testCases := []struct {
		args          []string
		expectedErrSub string
	}{
		{[]string{"1/0"}, "division by zero"},
		{[]string{"2pi"}, "implicit multiplication is not allowed"},
		{[]string{"log(-1)"}, "domain error"},
	}

	for _, tc := range testCases {
		out := new(bytes.Buffer)
		errOut := new(bytes.Buffer)
		code := run(tc.args, strings.NewReader(""), out, errOut)

		if code != 1 {
			t.Errorf("args %v: expected exit code 1, got %d", tc.args, code)
		}
		if !strings.Contains(errOut.String(), tc.expectedErrSub) {
			t.Errorf("args %v: expected stderr to contain %q, got %q", tc.args, tc.expectedErrSub, errOut.String())
		}
	}
}

func TestCLI_Help(t *testing.T) {
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	code := run([]string{"--help"}, strings.NewReader(""), out, errOut)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(out.String(), "ihd - Exact Arithmetic Calculator") {
		t.Errorf("expected help message, got %q", out.String())
	}
}

func TestCLI_Pipe(t *testing.T) {
	input := "1/2 + 1/3\nsqrt(4)\n# comment line\n\n3*5\n"
	in := strings.NewReader(input)
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)

	// Note: strings.NewReader is not an os.File, so isTerminal returns false (pipe mode)
	code := run([]string{}, in, out, errOut)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, errOut.String())
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	expected := []string{"5/6", "2", "15"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d output lines, got %d (%v)", len(expected), len(lines), lines)
	}
	for i, exp := range expected {
		if strings.TrimSpace(lines[i]) != exp {
			t.Errorf("line %d: expected %q, got %q", i, exp, strings.TrimSpace(lines[i]))
		}
	}
}

func TestCLI_REPL(t *testing.T) {
	input := "1+1\n2*3\nexit\n"
	in := strings.NewReader(input)
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)

	code := runREPL(in, out, errOut, FormatOptions{AsciiOnly: false}, false)
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}

	output := out.String()
	if !strings.Contains(output, PromptREPL) {
		t.Errorf("expected REPL prompt, got %q", output)
	}
	if !strings.Contains(output, "2") {
		t.Errorf("expected output 2, got %q", output)
	}
	if !strings.Contains(output, "6") {
		t.Errorf("expected output 6, got %q", output)
	}
	if !strings.Contains(output, MsgREPLExit) {
		t.Errorf("expected exit message, got %q", output)
	}
}
