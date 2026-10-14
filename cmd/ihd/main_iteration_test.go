package main

import (
	"bytes"
	"math/big"
	"strings"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

func TestMain_CLI_MaxLoopFlag(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("IHD_DIR", tmpDir)
	t.Setenv("IHD_LANG", "en")

	in := strings.NewReader("")
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	args := []string{"--max-loop", "5", "table(k, k, 1, 10)"}
	exitCode := run(args, in, out, errOut)

	if exitCode == 0 {
		t.Fatalf("expected non-zero exit code due to max-loop limit, got 0. stdout: %s", out.String())
	}

	errStr := errOut.String()
	if !strings.Contains(errStr, "exceeded maximum limit") {
		t.Errorf("expected errOut to mention exceeded maximum limit, got: %s", errStr)
	}
}

func TestMain_CLI_WarnLoopFlag_CleanStdout(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("IHD_DIR", tmpDir)
	t.Setenv("IHD_LANG", "en")

	in := strings.NewReader("")
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	args := []string{"--warn-loop", "3", "table(k, k, 1, 5)"}
	exitCode := run(args, in, out, errOut)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", exitCode, errOut.String())
	}

	// stdout must be clean with only calculation result
	outStr := strings.TrimSpace(out.String())
	if outStr != "[1, 2, 3, 4, 5]" {
		t.Errorf("expected stdout to be '[1, 2, 3, 4, 5]', got: %q", outStr)
	}

	// stderr must receive the non-destructive warning
	errStr := errOut.String()
	if !strings.Contains(errStr, "warning:") || !strings.Contains(errStr, "exceeded") {
		t.Errorf("expected warning in errOut, got: %q", errStr)
	}
}

func TestMain_REPL_LoopCommands(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("IHD_DIR", tmpDir)
	t.Setenv("IHD_LANG", "en")

	inputCommands := "config loop\nwarn-loop 500\nmax-loop 2000\nconfig loop\nexit\n"
	in := strings.NewReader(inputCommands)
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	exitCode := run([]string{}, in, out, errOut)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", exitCode, errOut.String())
	}

	outStr := out.String()
	if !strings.Contains(outStr, "Current loop settings") {
		t.Errorf("expected 'Current loop settings' in output, got: %s", outStr)
	}
	if !strings.Contains(outStr, "500") {
		t.Errorf("expected '500' in output, got: %s", outStr)
	}
	if !strings.Contains(outStr, "2000") {
		t.Errorf("expected '2000' in output, got: %s", outStr)
	}

	// Verify persistence in config.json
	cfg, err := i18n.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	warnBig := i18n.GetWarnIterationsBig(cfg)
	if warnBig.Cmp(big.NewInt(500)) != 0 {
		t.Errorf("expected saved warn iterations 500, got %v", warnBig)
	}
	maxBig := i18n.GetMaxIterationsBig(cfg)
	if maxBig.Cmp(big.NewInt(2000)) != 0 {
		t.Errorf("expected saved max iterations 2000, got %v", maxBig)
	}
}
