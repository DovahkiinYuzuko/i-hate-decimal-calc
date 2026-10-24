package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

func TestREPL_ThemeAndHighlightCommands(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("IHD_DIR", tmpDir)

	input := "theme\ntheme light\ntheme\nhighlight off\nhighlight\nhighlight on\ntheme invalid\nexit\n"
	in := strings.NewReader(input)
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	ro := runOptions{}
	code := runREPL(in, out, errOut, ro)
	if code != 0 {
		t.Fatalf("runREPL returned non-zero code %d", code)
	}

	outStr := out.String()
	if !strings.Contains(outStr, "light") {
		t.Errorf("expected output to mention 'light', got: %s", outStr)
	}

	// Verify configuration was saved
	cfg, err := i18n.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Theme != "light" {
		t.Errorf("expected saved theme 'light', got %q", cfg.Theme)
	}
	if cfg.SyntaxHighlight == nil || *cfg.SyntaxHighlight != true {
		t.Errorf("expected SyntaxHighlight true, got %v", cfg.SyntaxHighlight)
	}
}

func TestREPL_PipedOutputHasNoANSIEscape(t *testing.T) {
	input := "1 + 1\nsin(pi)\nexit\n"
	in := strings.NewReader(input)
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	ro := runOptions{}
	code := runREPL(in, out, errOut, ro)
	if code != 0 {
		t.Fatalf("runREPL returned non-zero code %d", code)
	}

	outStr := out.String()
	if strings.Contains(outStr, "\x1B[") {
		t.Errorf("piped REPL output contains ANSI escape sequences: %q", outStr)
	}
}

func TestREPL_SessionCommands(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("IHD_DIR", tmpDir)

	sessionFile := filepath.Join(tmpDir, "saved_session.ihd")
	escapedSessionFile := strings.ReplaceAll(sessionFile, "\\", "/")

	input := "x = 100\nassume(y, positive)\nsave \"" + escapedSessionFile + "\"\nclear\nvars\nload \"" + escapedSessionFile + "\"\nvars\nsimilar(x + 1)\nexit\n"
	in := strings.NewReader(input)
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	ro := runOptions{}
	code := runREPL(in, out, errOut, ro)
	if code != 0 {
		t.Fatalf("runREPL returned non-zero code %d, err: %s", code, errOut.String())
	}

	outStr := out.String()
	if !strings.Contains(outStr, "x = 100") {
		t.Errorf("expected session load to restore x = 100, out:\n%s", outStr)
	}
	if !strings.Contains(outStr, "saved_session.ihd") {
		t.Errorf("expected save/load confirmation in output, out:\n%s", outStr)
	}
}

