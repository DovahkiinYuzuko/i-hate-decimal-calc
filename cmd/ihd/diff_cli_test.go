package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI_DiffSubcommand(t *testing.T) {
	tempDir := t.TempDir()
	f1 := filepath.Join(tempDir, "script1.ihd")
	f2 := filepath.Join(tempDir, "script2.ihd")

	c1 := `
f = (x + 1)^2
val = 42
`
	c2 := `
f = x^2 + 2*x + 1
val = 42
extra = 100
`

	if err := os.WriteFile(f1, []byte(c1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte(c2), 0644); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	code := run([]string{"diff", f1, f2}, nil, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected code 0, got %d. errOut: %s", code, errOut.String())
	}

	outStr := out.String()
	if !strings.Contains(outStr, "Semantic Diff") && !strings.Contains(outStr, "意味論的数式差分") {
		t.Errorf("expected diff header, got: %s", outStr)
	}
	if !strings.Contains(outStr, "EQUIVALENT") && !strings.Contains(outStr, "意味論的等価") {
		t.Errorf("expected equivalent marker for f, got: %s", outStr)
	}
	if !strings.Contains(outStr, "ADDED") && !strings.Contains(outStr, "追加") {
		t.Errorf("expected added marker for extra, got: %s", outStr)
	}

	// Test direct expression diff without files
	var outInline, errOutInline bytes.Buffer
	codeInline := run([]string{"diff", "(x + 1)^2", "x^2 + 2*x + 1"}, nil, &outInline, &errOutInline)
	if codeInline != 0 {
		t.Fatalf("expected code 0 for inline diff, got %d. errOut: %s", codeInline, errOutInline.String())
	}
	outInlineStr := outInline.String()
	if !strings.Contains(outInlineStr, "EQUIVALENT") && !strings.Contains(outInlineStr, "意味論的等価") {
		t.Errorf("expected EQUIVALENT for inline expression diff, got: %s", outInlineStr)
	}
}

func TestCLI_TraceFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"--trace", "groebner([x^2 - y, x^3 - z], [x, y, z])"}, nil, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected code 0, got %d. errOut: %s", code, errOut.String())
	}

	outStr := out.String()
	if !strings.Contains(outStr, "buchberger") && !strings.Contains(outStr, "groebner.pair_selected") {
		t.Errorf("expected groebner pair events in JSON dump, got: %s", outStr)
	}
}

func TestCLI_REPL_ProveCommand(t *testing.T) {
	input := strings.NewReader("prove expand((x + 1)^2)\nq\nexit\n")
	var out, errOut bytes.Buffer
	ro := runOptions{noColor: true}
	code := runREPL(input, &out, &errOut, ro)
	if code != 0 {
		t.Fatalf("expected code 0, got %d. errOut: %s", code, errOut.String())
	}

	outStr := out.String()
	if !strings.Contains(outStr, "Derivation Step") && !strings.Contains(outStr, "導出ステップ") {
		t.Errorf("expected step explorer in prove command output, got: %s", outStr)
	}
}
