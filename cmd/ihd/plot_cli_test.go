package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI_Plot_VectorExport_And_OutFlag(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. --format svg one-shot stdout
	var out, errOut bytes.Buffer
	code := run([]string{"--format", "svg", "plot(1/x, [-3, 3])"}, strings.NewReader(""), &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit code 0 for --format svg, got %d, stderr: %s", code, errOut.String())
	}
	svgOut := out.String()
	if !strings.Contains(svgOut, "<svg") || !strings.Contains(svgOut, "viewBox=") {
		t.Errorf("expected valid SVG output from --format svg, got: %s", svgOut)
	}

	// 2. --format svg --out <file> writes UTF-8 without BOM
	svgPath := filepath.Join(tmpDir, "nested", "hyperbola.svg")
	out.Reset()
	errOut.Reset()
	code = run([]string{"--format", "svg", "--out", svgPath, "plot(x^2 - 1, [-3, 3])"}, strings.NewReader(""), &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit code 0 for --format svg --out, got %d, stderr: %s", code, errOut.String())
	}
	svgBytes, err := os.ReadFile(svgPath)
	if err != nil {
		t.Fatalf("failed reading SVG output file: %v", err)
	}
	if len(svgBytes) >= 3 && svgBytes[0] == 0xEF && svgBytes[1] == 0xBB && svgBytes[2] == 0xBF {
		t.Errorf("expected UTF-8 without BOM in SVG output file")
	}
	if !strings.Contains(string(svgBytes), "<svg") || !strings.Contains(string(svgBytes), "<path") {
		t.Errorf("expected SVG content in file, got: %s", string(svgBytes))
	}

	// 3. --typst --out <file> emits CeTZ plot Typst source cleanly
	typPath := filepath.Join(tmpDir, "curve.typ")
	out.Reset()
	errOut.Reset()
	code = run([]string{"--typst", "--out", typPath, "plot(sin(x), [-pi, pi])"}, strings.NewReader(""), &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit code 0 for --typst --out, got %d, stderr: %s", code, errOut.String())
	}
	typBytes, err := os.ReadFile(typPath)
	if err != nil {
		t.Fatalf("failed reading Typst output file: %v", err)
	}
	typStr := string(typBytes)
	if !strings.Contains(typStr, "cetz-plot") || !strings.Contains(typStr, "plot.add") {
		t.Errorf("expected CeTZ plot Typst markup, got: %s", typStr)
	}

	// 4. --explain --typst --out <file> emits Typst derivation document
	explainTypPath := filepath.Join(tmpDir, "apart.typ")
	out.Reset()
	errOut.Reset()
	code = run([]string{"--explain", "--typst", "--out", explainTypPath, "apart(1/(x^2 - 1), x)"}, strings.NewReader(""), &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit code 0 for --explain --typst --out, got %d, stderr: %s", code, errOut.String())
	}
	explainBytes, err := os.ReadFile(explainTypPath)
	if err != nil {
		t.Fatalf("failed reading Typst explain file: %v", err)
	}
	if !strings.Contains(string(explainBytes), "apart") && !strings.Contains(string(explainBytes), "=") {
		t.Errorf("expected Typst derivation output, got: %s", string(explainBytes))
	}

	// 5. --format svg on non-plot expression must return explicit error
	out.Reset()
	errOut.Reset()
	code = run([]string{"--format", "svg", "1 + 2"}, strings.NewReader(""), &out, &errOut)
	if code == 0 {
		t.Errorf("expected non-zero exit code for --format svg on non-plot expression, got stdout=%q", out.String())
	}

	// 6. --out with piped multi-line input must return explicit error
	out.Reset()
	errOut.Reset()
	code = run([]string{"--out", filepath.Join(tmpDir, "pipe.txt")}, strings.NewReader("1 + 1\n2 + 2\n"), &out, &errOut)
	if code == 0 {
		t.Errorf("expected non-zero exit code for --out with piped multi-line input")
	}
}
