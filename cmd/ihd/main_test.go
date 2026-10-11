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

func TestCLI_OneShot_Interval(t *testing.T) {
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	code := run([]string{"-i", "sqrt(2)"}, strings.NewReader(""), out, errOut)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, errOut.String())
	}
	actual := strings.TrimSpace(out.String())
	if !strings.Contains(actual, "width:") || !strings.Contains(actual, "1.41421") {
		t.Errorf("expected interval output with width and 1.41421, got %q", actual)
	}

	// Test with --interval and custom eps
	out.Reset()
	errOut.Reset()
	code = run([]string{"--interval", "-interval-eps", "1/1000", "pi"}, strings.NewReader(""), out, errOut)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, errOut.String())
	}
	actualPi := strings.TrimSpace(out.String())
	if !strings.Contains(actualPi, "3.14") {
		t.Errorf("expected interval output with 3.14, got %q", actualPi)
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
	code := run([]string{"--lang", "ja", "--explain", "1 / (sqrt(2) + 1)"}, nil, out, errOut)
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
	input := "lang ja\nexplain 1 / (sqrt(2) + 1)\nexit\n"
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
	if !strings.Contains(errStr, "error.ihd:4:") || (!strings.Contains(errStr, "division by zero") && !strings.Contains(errStr, "ゼロ除算")) {
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

func TestCLI_CachedUpdateNotice(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("IHD_DIR", tmpDir)
	t.Setenv("IHD_NO_UPDATE_CHECK", "")

	// Write mock cache with v99.0.0
	cachePath := filepath.Join(tmpDir, "update_cache.json")
	cacheJSON := `{"last_checked_at": "2026-10-06T00:00:00Z", "latest_version": "v99.0.0", "release_url": "https://example.com/v99.0.0"}`
	_ = os.WriteFile(cachePath, []byte(cacheJSON), 0644)

	// 1. REPL mode: should show notification in welcome section of out
	outREPL := new(bytes.Buffer)
	errOutREPL := new(bytes.Buffer)
	codeREPL := runREPL(strings.NewReader("exit\n"), outREPL, errOutREPL, runOptions{opts: calc.FormatOptions{AsciiOnly: false}})
	if codeREPL != 0 {
		t.Fatalf("expected exit code 0, got %d", codeREPL)
	}
	if !strings.Contains(outREPL.String(), "v99.0.0") {
		t.Errorf("expected REPL output to contain update notice for v99.0.0, got: %s", outREPL.String())
	}

	// 2. One-shot mode: stdout should strictly be "2", errOut should contain notification
	outOneShot := new(bytes.Buffer)
	errOutOneShot := new(bytes.Buffer)
	codeOneShot := run([]string{"1 + 1"}, strings.NewReader(""), outOneShot, errOutOneShot)
	if codeOneShot != 0 {
		t.Fatalf("expected exit code 0, got %d", codeOneShot)
	}
	if strings.TrimSpace(outOneShot.String()) != "2" {
		t.Errorf("expected stdout to be exactly '2', got: %q", outOneShot.String())
	}
	if !strings.Contains(errOutOneShot.String(), "v99.0.0") {
		t.Errorf("expected stderr to contain update notice for v99.0.0, got: %s", errOutOneShot.String())
	}

	// 3. With --no-update-check flag: errOut should NOT contain notification
	outDisabled := new(bytes.Buffer)
	errOutDisabled := new(bytes.Buffer)
	codeDisabled := run([]string{"--no-update-check", "1 + 1"}, strings.NewReader(""), outDisabled, errOutDisabled)
	if codeDisabled != 0 {
		t.Fatalf("expected exit code 0, got %d", codeDisabled)
	}
	if strings.Contains(errOutDisabled.String(), "v99.0.0") {
		t.Errorf("expected no update notice when --no-update-check is passed, got: %s", errOutDisabled.String())
	}
}


func TestCLI_LeanAndLeanFile(t *testing.T) {
	tmpDir := t.TempDir()
	leanFilePath := filepath.Join(tmpDir, "proof.lean")

	// 1. Test --lean flag
	outLean := new(bytes.Buffer)
	errOutLean := new(bytes.Buffer)
	codeLean := run([]string{"--lean", "factor(x^2 - 1)"}, strings.NewReader(""), outLean, errOutLean)
	if codeLean != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", codeLean, errOutLean.String())
	}
	leanOutStr := outLean.String()
	if !strings.Contains(leanOutStr, "import Mathlib.Tactic.Ring") {
		t.Errorf("expected Mathlib import in output, got %q", leanOutStr)
	}
	if !strings.Contains(leanOutStr, "by ring") {
		t.Errorf("expected 'by ring' in output, got %q", leanOutStr)
	}

	// 2. Test --lean-file flag
	outLeanFile := new(bytes.Buffer)
	errOutLeanFile := new(bytes.Buffer)
	codeLeanFile := run([]string{"--lean-file", leanFilePath, "factor(x^2 - 1)"}, strings.NewReader(""), outLeanFile, errOutLeanFile)
	if codeLeanFile != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", codeLeanFile, errOutLeanFile.String())
	}
	content, err := os.ReadFile(leanFilePath)
	if err != nil {
		t.Fatalf("failed reading generated lean file: %v", err)
	}
	fileStr := string(content)
	if !strings.Contains(fileStr, "theorem ihd_certified_proof") {
		t.Errorf("expected theorem in saved file, got %q", fileStr)
	}
	if !strings.Contains(fileStr, "variable (x : ℚ)") {
		t.Errorf("expected variable declaration in saved file, got %q", fileStr)
	}
	if !strings.Contains(fileStr, "#print axioms ihd_certified_proof") {
		t.Errorf("expected '#print axioms ihd_certified_proof' in saved file, got %q", fileStr)
	}

	// 3. Test 'lean <expr>' prefix
	outPrefix := new(bytes.Buffer)
	errOutPrefix := new(bytes.Buffer)
	codePrefix := run([]string{"lean factor(x^2 - 1)"}, strings.NewReader(""), outPrefix, errOutPrefix)
	if codePrefix != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", codePrefix, errOutPrefix.String())
	}
	if !strings.Contains(outPrefix.String(), "by ring") {
		t.Errorf("expected 'by ring' for prefix command, got %q", outPrefix.String())
	}

	// 4. Test calculus integration lean generation: integrate(exp(x), x)
	outExp := new(bytes.Buffer)
	errOutExp := new(bytes.Buffer)
	codeExp := run([]string{"--lean", "integrate(exp(x), x)"}, strings.NewReader(""), outExp, errOutExp)
	if codeExp != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", codeExp, errOutExp.String())
	}
	expStr := outExp.String()
	if !strings.Contains(expStr, "by simp") {
		t.Errorf("expected 'by simp' for exp deriv proof, got %q", expStr)
	}
	if !strings.Contains(expStr, "#print axioms ihd_certified_proof") {
		t.Errorf("expected '#print axioms ihd_certified_proof' in output, got %q", expStr)
	}

	// 5. Test factor(x^3 - 1)
	outCube := new(bytes.Buffer)
	errOutCube := new(bytes.Buffer)
	codeCube := run([]string{"--lean", "factor(x^3 - 1)"}, strings.NewReader(""), outCube, errOutCube)
	if codeCube != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", codeCube, errOutCube.String())
	}
	if !strings.Contains(outCube.String(), "by ring") {
		t.Errorf("expected 'by ring' for factor(x^3 - 1), got %q", outCube.String())
	}

	// 6. Test linear algebra matrix inversion: inv([[1, 2], [3, 4]])
	outInv := new(bytes.Buffer)
	errOutInv := new(bytes.Buffer)
	codeInv := run([]string{"--lean", "inv([[1, 2], [3, 4]])"}, strings.NewReader(""), outInv, errOutInv)
	if codeInv != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", codeInv, errOutInv.String())
	}
	invStr := outInv.String()
	if !strings.Contains(invStr, "fin_cases") && !strings.Contains(invStr, "norm_num") && !strings.Contains(invStr, "by ext") {
		t.Errorf("expected matrix inv proof tactic, got %q", invStr)
	}

	// 7. Test structured geometric deduction theorem: midpoint theorem
	outGeo := new(bytes.Buffer)
	errOutGeo := new(bytes.Buffer)
	codeGeo := run([]string{"--lean", "geo_prove([midpoint(M, A, B), midpoint(N, A, C)], parallel(M, N, B, C))"}, strings.NewReader(""), outGeo, errOutGeo)
	if codeGeo != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", codeGeo, errOutGeo.String())
	}
	geoStr := outGeo.String()
	if !strings.Contains(geoStr, "ihd_certified_proof_algebraic_identity") {
		t.Errorf("expected algebraic identity lemma in geo proof, got %q", geoStr)
	}
	if !strings.Contains(geoStr, "rw [h_M_x, h_M_y, h_N_x, h_N_y]") {
		t.Errorf("expected hypothesis rewrites in geo proof, got %q", geoStr)
	}
}

func TestCLI_CertificateAndReplay(t *testing.T) {
	tempDir := t.TempDir()
	certFile := filepath.Join(tempDir, "factor_cert.json")

	// 1. Generate certificate via CLI
	var out, errOut bytes.Buffer
	code := run([]string{"--certificate", certFile, "factor(x^2 - 1)"}, strings.NewReader(""), &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, err: %s", code, errOut.String())
	}

	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		t.Fatalf("expected certificate file to be created at %s", certFile)
	}

	// 2. Replay certificate via CLI
	var outReplay, errOutReplay bytes.Buffer
	replayCode := run([]string{"--replay", certFile}, strings.NewReader(""), &outReplay, &errOutReplay)
	if replayCode != 0 {
		t.Fatalf("expected replay exit code 0, got %d, err: %s", replayCode, errOutReplay.String())
	}

	replayOutput := outReplay.String()
	if !strings.Contains(replayOutput, certFile) {
		t.Errorf("expected replay output to reference cert file, got: %s", replayOutput)
	}
}

func TestCLI_CertificateAndReplay_MultipleDomains(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name string
		expr string
	}{
		{"Calculus_Integrate", "integrate(exp(x), x)"},
		{"LinearAlgebra_Inv", "inv([[1, 2], [3, 4]])"},
		{"NumberTheory_Pell", "solve_pell(2)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			certFile := filepath.Join(tempDir, tc.name+".json")

			var out, errOut bytes.Buffer
			code := run([]string{"--certificate", certFile, tc.expr}, strings.NewReader(""), &out, &errOut)
			if code != 0 {
				t.Fatalf("expected exit code 0 for %s, got %d, err: %s", tc.expr, code, errOut.String())
			}

			if _, err := os.Stat(certFile); os.IsNotExist(err) {
				t.Fatalf("expected cert file for %s to exist", tc.expr)
			}

			var outReplay, errOutReplay bytes.Buffer
			replayCode := run([]string{"--replay", certFile}, strings.NewReader(""), &outReplay, &errOutReplay)
			if replayCode != 0 {
				t.Fatalf("expected replay exit code 0 for %s, got %d, err: %s", tc.expr, replayCode, errOutReplay.String())
			}
		})
	}
}

func TestCLI_CliffordGeometricAlgebra(t *testing.T) {
	testCases := []struct {
		expr     string
		expected string
	}{
		{"clifford(e0 * e1 + e1 * e0, 3, 0)", "0"},
		{"clifford(e0 * e0, 3, 0)", "1"},
		{"clifford(e1 * e1, 1, 3)", "-1"},
		{"clifford(e3 * e3, 3, 0, 1)", "0"},
		{"clifford_wedge(e0, e1, 3, 0)", "e0^e1"},
		{"clifford_contract(e0, e0, \"left\", 3, 0)", "1"},
		{"clifford_dual(e3, 3, 0, 1)", "-e0^e1^e2"},
	}

	for _, tc := range testCases {
		t.Run(tc.expr, func(t *testing.T) {
			out := new(bytes.Buffer)
			errOut := new(bytes.Buffer)
			code := run([]string{tc.expr}, strings.NewReader(""), out, errOut)
			if code != 0 {
				t.Fatalf("expected exit code 0 for %s, got %d. stderr: %s", tc.expr, code, errOut.String())
			}
			actual := strings.TrimSpace(out.String())
			if actual != tc.expected {
				t.Errorf("expr %s: expected %q, got %q", tc.expr, tc.expected, actual)
			}
		})
	}
}

func TestCLI_QuantumCircuitEquivalence(t *testing.T) {
	testCases := []struct {
		name     string
		expr     string
		expected string
	}{
		{
			name:     "H H identity",
			expr:     "quantum_equiv([[\"H\", 0], [\"H\", 0]], [])",
			expected: "[1, 0, -1]",
		},
		{
			name:     "X Z equivalent to Y up to global phase omega^2",
			expr:     "quantum_equiv([[\"X\", 0], [\"Z\", 0]], [[\"Y\", 0]])",
			expected: "[1, 2, -1]",
		},
		{
			name:     "X Z strict equivalence to Y fails",
			expr:     "quantum_equiv([[\"X\", 0], [\"Z\", 0]], [[\"Y\", 0]], true)",
			expected: "[0, 2, 0]",
		},
		{
			name:     "Pauli X evaluation",
			expr:     "quantum_eval([[\"X\", 0]])",
			expected: "[[0, 1], [1, 0]]",
		},
		{
			name:     "CNOT commutativity refutation with counterexample",
			expr:     "quantum_equiv([[\"CNOT\", 0, 1]], [[\"CNOT\", 1, 0]])",
			expected: "[0, -1, 1]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := new(bytes.Buffer)
			errOut := new(bytes.Buffer)
			code := run([]string{tc.expr}, strings.NewReader(""), out, errOut)
			if code != 0 {
				t.Fatalf("expected exit code 0 for %s, got %d. stderr: %s", tc.expr, code, errOut.String())
			}
			actual := strings.TrimSpace(out.String())
			if actual != tc.expected {
				t.Errorf("expr %s: expected %q, got %q", tc.expr, tc.expected, actual)
			}
		})
	}
}

func TestCLI_Clifford_Quantum_Adversarial(t *testing.T) {
	errorCases := []struct {
		name string
		expr string
	}{
		{
			name: "PGA singular blade inverse division",
			expr: "clifford(1 / e3, 3, 0, 1)",
		},
		{
			name: "Unknown quantum gate",
			expr: "quantum_equiv([[\"FOOBAR\", 0]], [])",
		},
		{
			name: "CNOT target equals control",
			expr: "quantum_equiv([[\"CNOT\", 0, 0]], [])",
		},
		{
			name: "Out of range qubit count (> 6)",
			expr: "quantum_eval([[\"H\", 7]])",
		},
	}

	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			out := new(bytes.Buffer)
			errOut := new(bytes.Buffer)
			code := run([]string{tc.expr}, strings.NewReader(""), out, errOut)
			if code == 0 {
				t.Errorf("expected error exit code != 0 for adversarial case %s, got 0. stdout: %s", tc.expr, out.String())
			}
		})
	}
}

func TestCLI_ThemeAndNoColorFlags(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("IHD_DIR", tmpDir)

	// 1. One-shot evaluation with --theme and --no-color flags
	out1 := new(bytes.Buffer)
	errOut1 := new(bytes.Buffer)
	code1 := run([]string{"--theme", "light", "1 + 1"}, strings.NewReader(""), out1, errOut1)
	if code1 != 0 {
		t.Fatalf("expected code 0, got %d", code1)
	}
	if strings.TrimSpace(out1.String()) != "2" {
		t.Errorf("expected '2', got %q", out1.String())
	}

	out2 := new(bytes.Buffer)
	errOut2 := new(bytes.Buffer)
	code2 := run([]string{"--no-color", "2 * 3"}, strings.NewReader(""), out2, errOut2)
	if code2 != 0 {
		t.Fatalf("expected code 0, got %d", code2)
	}
	if strings.TrimSpace(out2.String()) != "6" {
		t.Errorf("expected '6', got %q", out2.String())
	}

	// 2. Interactive REPL via run with --theme=light
	outREPL := new(bytes.Buffer)
	errOutREPL := new(bytes.Buffer)
	codeREPL := run([]string{"--theme=light"}, strings.NewReader("theme\nexit\n"), outREPL, errOutREPL)
	if codeREPL != 0 {
		t.Fatalf("expected code 0, got %d", codeREPL)
	}
	if !strings.Contains(outREPL.String(), "light") {
		t.Errorf("expected REPL output to reflect light theme, got: %s", outREPL.String())
	}

	// 3. Piped stdin has zero ANSI escape codes
	outPipe := new(bytes.Buffer)
	errOutPipe := new(bytes.Buffer)
	codePipe := run([]string{}, strings.NewReader("sin(0)\nexit\n"), outPipe, errOutPipe)
	if codePipe != 0 {
		t.Fatalf("expected code 0, got %d", codePipe)
	}
	if strings.Contains(outPipe.String(), "\x1B[") {
		t.Errorf("piped output should not contain ANSI escape codes: %q", outPipe.String())
	}
}


