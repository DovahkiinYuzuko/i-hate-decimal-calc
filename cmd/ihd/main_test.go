package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
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
		{[]string{"0.(3) + 0.1(6)"}, "1/2", 0},
		{[]string{"sqrt(5 + 2*sqrt(6))"}, "√2 + √3", 0},
		{[]string{"(1 + sqrt(3)) / (2 + sqrt(5))"}, "-2 - 2*√3 + √15 + √5", 0},
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

func TestCLI_OneShot_LaTeX(t *testing.T) {
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	code := run([]string{"--latex", "1/2 + sqrt(2)"}, strings.NewReader(""), out, errOut)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, errOut.String())
	}
	actual := strings.TrimSpace(out.String())
	expected := "$$ \\frac{1}{2} + \\sqrt{2} $$"
	if actual != expected {
		t.Errorf("expected %q, got %q", expected, actual)
	}
}

func TestCLI_OneShot_Pretty(t *testing.T) {
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	code := run([]string{"--pretty", "1/2"}, strings.NewReader(""), out, errOut)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, errOut.String())
	}
	actual := strings.TrimSpace(out.String())
	if !strings.Contains(actual, "---") || !strings.Contains(actual, "1") || !strings.Contains(actual, "2") {
		t.Errorf("expected 2D pretty fraction, got:\n%s", actual)
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

	code := runREPL(in, out, errOut, runOptions{opts: calc.FormatOptions{AsciiOnly: false}})
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}

	output := out.String()
	if !strings.Contains(output, i18n.T("cli.prompt")) {
		t.Errorf("expected REPL prompt, got %q", output)
	}
	if !strings.Contains(output, "2") {
		t.Errorf("expected output 2, got %q", output)
	}
	if !strings.Contains(output, "6") {
		t.Errorf("expected output 6, got %q", output)
	}
	if !strings.Contains(output, i18n.T("cli.repl_exit")) {
		t.Errorf("expected exit message, got %q", output)
	}
}

func TestCLI_REPL_VariablesAndCommands(t *testing.T) {
	input := "x = 1/2\nx + 1\nans * 2\nvars\nexit\n"
	in := strings.NewReader(input)
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)

	code := runREPL(in, out, errOut, runOptions{opts: calc.FormatOptions{AsciiOnly: false}})
	if code != 0 {
		t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
	}

	output := out.String()
	if !strings.Contains(output, "1/2") {
		t.Errorf("expected output to contain '1/2', got %q", output)
	}
	if !strings.Contains(output, "3/2") {
		t.Errorf("expected output to contain '3/2', got %q", output)
	}
	if !strings.Contains(output, "3") {
		t.Errorf("expected output to contain '3', got %q", output)
	}
	if !strings.Contains(output, "x = 1/2") {
		t.Errorf("expected vars output to contain 'x = 1/2', got %q", output)
	}
	if !strings.Contains(output, "ans = 3") {
		t.Errorf("expected vars output to contain 'ans = 3', got %q", output)
	}
}

func TestCLI_Pipe_Variables(t *testing.T) {
	input := "x = 1/2\nx + 1\nans * 2\n"
	in := strings.NewReader(input)
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)

	code := run([]string{}, in, out, errOut)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, errOut.String())
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	expected := []string{"1/2", "3/2", "3"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d output lines, got %d (%v)", len(expected), len(lines), lines)
	}
	for i, exp := range expected {
		if strings.TrimSpace(lines[i]) != exp {
			t.Errorf("line %d: expected %q, got %q", i, exp, strings.TrimSpace(lines[i]))
		}
	}
}

func TestCLI_Explain(t *testing.T) {
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	code := run([]string{"--explain", "1 / (sqrt(2) + 1)"}, nil, out, errOut)
	if code != 0 {
		t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
	}
	output := out.String()
	if !strings.Contains(output, "Step 1: 分母の有理化") {
		t.Errorf("expected step output, got %q", output)
	}
	if !strings.Contains(output, "= -1 + √2") {
		t.Errorf("expected final result, got %q", output)
	}
}

func TestCLI_REPL_ExplainCommand(t *testing.T) {
	input := "explain 1 / (sqrt(2) + 1)\nexit\n"
	in := strings.NewReader(input)
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)

	code := runREPL(in, out, errOut, runOptions{opts: calc.FormatOptions{AsciiOnly: false}})
	if code != 0 {
		t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
	}
	output := out.String()
	if !strings.Contains(output, "Step 1: 分母の有理化") {
		t.Errorf("expected REPL explain step output, got %q", output)
	}
}

func TestCLI_OneShot_CaseInsensitive(t *testing.T) {
	testCases := []struct {
		expr     string
		expected string
	}{
		{"100*5/3+Sqrt(541)", "500/3 + √541"},
		{"SIN(pi/6)", "1/2"},
		{"Abs(-5)", "5"},
		{"Cbrt(8)", "2"},
	}

	for _, tc := range testCases {
		out := new(bytes.Buffer)
		errOut := new(bytes.Buffer)
		code := run([]string{tc.expr}, strings.NewReader(""), out, errOut)
		if code != 0 {
			t.Errorf("expr %q: expected exit code 0, got %d. stderr: %s", tc.expr, code, errOut.String())
		}
		actual := strings.TrimSpace(out.String())
		if actual != tc.expected {
			t.Errorf("expr %q: expected stdout %q, got %q", tc.expr, tc.expected, actual)
		}
	}
}

func TestCLI_REPL_CaseInsensitiveCommands(t *testing.T) {
	// Test Exit, Quit, Vars in mixed case
	input := "x = 10\nVars\nExit\n"
	in := strings.NewReader(input)
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)

	code := runREPL(in, out, errOut, runOptions{opts: calc.FormatOptions{AsciiOnly: false}})
	if code != 0 {
		t.Fatalf("expected code 0, got %d. stderr: %s", code, errOut.String())
	}
	output := out.String()
	if !strings.Contains(output, "x = 10") {
		t.Errorf("expected Vars output to contain 'x = 10', got %q", output)
	}
}

func TestCLI_RunScriptFile(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Basic script with variables and semicolon suppression
	scriptContent := `# Sample script
x = 1/2;
y = sqrt(2); # inline comment
x + 1
y * 2
`
	scriptPath := filepath.Join(tmpDir, "sample.ihd")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		t.Fatalf("failed writing test script: %v", err)
	}

	// Test with "run" subcommand: ihd run sample.ihd
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	code := run([]string{"run", scriptPath}, strings.NewReader(""), out, errOut)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, errOut.String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	expected := []string{"3/2", "2*√2"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d: %v", len(expected), len(lines), lines)
	}
	for i, exp := range expected {
		if strings.TrimSpace(lines[i]) != exp {
			t.Errorf("line %d: expected %q, got %q", i, exp, strings.TrimSpace(lines[i]))
		}
	}

	// Test with direct invocation: ihd sample.ihd
	outDirect := new(bytes.Buffer)
	errOutDirect := new(bytes.Buffer)
	codeDirect := run([]string{scriptPath}, strings.NewReader(""), outDirect, errOutDirect)
	if codeDirect != 0 {
		t.Fatalf("direct invocation expected exit code 0, got %d. stderr: %s", codeDirect, errOutDirect.String())
	}
	if strings.TrimSpace(outDirect.String()) != strings.TrimSpace(out.String()) {
		t.Errorf("direct invocation output mismatch: expected %q, got %q", out.String(), outDirect.String())
	}

	// 2. Script with line number error reporting
	errScriptContent := `# Line 1 comment
a = 10;
b = 0;
a / b
`
	errScriptPath := filepath.Join(tmpDir, "error.ihd")
	if err := os.WriteFile(errScriptPath, []byte(errScriptContent), 0644); err != nil {
		t.Fatalf("failed writing test script: %v", err)
	}

	outErr := new(bytes.Buffer)
	errOutErr := new(bytes.Buffer)
	codeErr := run([]string{"run", errScriptPath}, strings.NewReader(""), outErr, errOutErr)
	if codeErr == 0 {
		t.Errorf("expected error exit code 1, got 0")
	}
	errStr := errOutErr.String()
	if !strings.Contains(errStr, "error.ihd:4:") || !strings.Contains(errStr, "division by zero") {
		t.Errorf("expected line number 4 and zero division in stderr, got %q", errStr)
	}

	// 3. Non-existent script file
	outNotFound := new(bytes.Buffer)
	errOutNotFound := new(bytes.Buffer)
	codeNotFound := run([]string{"run", filepath.Join(tmpDir, "non_existent.ihd")}, strings.NewReader(""), outNotFound, errOutNotFound)
	if codeNotFound == 0 {
		t.Errorf("expected error exit code 1 for non-existent file, got 0")
	}
}

func TestCLI_VersionAndNoUpdateCheck(t *testing.T) {
	t.Setenv("IHD_NO_UPDATE_CHECK", "1")

	// 1. Test -v and --version
	for _, flag := range []string{"-v", "--version"} {
		out := new(bytes.Buffer)
		errOut := new(bytes.Buffer)
		code := run([]string{flag}, strings.NewReader(""), out, errOut)
		if code != 0 {
			t.Errorf("expected exit code 0 for %s, got %d. stderr: %s", flag, code, errOut.String())
		}
		if !strings.HasPrefix(strings.TrimSpace(out.String()), "ihd v") {
			t.Errorf("expected output to start with 'ihd v', got %q", out.String())
		}
	}

	// 2. Test --no-update-check flag
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	code := run([]string{"--no-update-check", "1 + 1"}, strings.NewReader(""), out, errOut)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if strings.TrimSpace(out.String()) != "2" {
		t.Errorf("expected '2', got %q", out.String())
	}
}




