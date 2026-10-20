package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

func TestSupportedShells(t *testing.T) {
	shells := SupportedShells()
	expected := []string{"bash", "zsh", "fish", "powershell"}

	for _, exp := range expected {
		found := false
		for _, s := range shells {
			if s == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected shell %q in SupportedShells(), got %v", exp, shells)
		}
	}
}

func TestRunCompletion_AllShells(t *testing.T) {
	tests := []struct {
		shell       string
		mustContain []string
	}{
		{
			shell: "bash",
			mustContain: []string{
				"_ihd_completion",
				"complete -F _ihd_completion ihd",
				"--format",
				"--typst",
				"diff",
				"integrate",
				"solve",
			},
		},
		{
			shell: "zsh",
			mustContain: []string{
				"#compdef ihd",
				"_arguments",
				"--format",
				"diff",
				"integrate",
			},
		},
		{
			shell: "fish",
			mustContain: []string{
				"complete -c ihd",
				"-l format",
				"diff",
				"integrate",
			},
		},
		{
			shell: "powershell",
			mustContain: []string{
				"Register-ArgumentCompleter -Native -CommandName 'ihd'",
				"CompletionResult",
				"--format",
				"--approx",
				"diff",
				"integrate",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			var buf bytes.Buffer
			err := RunCompletion(tt.shell, &buf)
			if err != nil {
				t.Fatalf("RunCompletion(%q) failed: %v", tt.shell, err)
			}

			output := buf.String()
			for _, exp := range tt.mustContain {
				if !strings.Contains(output, exp) {
					t.Errorf("shell %s completion output missing expected string %q", tt.shell, exp)
				}
			}
		})
	}
}

func TestRunCompletion_InvalidShell(t *testing.T) {
	var buf bytes.Buffer
	err := RunCompletion("unsupported_shell", &buf)
	if err == nil {
		t.Errorf("expected error for unsupported shell, got nil")
	}
}

func TestRunCompletion_I18n(t *testing.T) {
	_ = i18n.Init("ja")
	var jaBuf bytes.Buffer
	if err := RunCompletion("powershell", &jaBuf); err != nil {
		t.Fatalf("RunCompletion ja failed: %v", err)
	}
	jaOutput := jaBuf.String()
	if !strings.Contains(jaOutput, "指定変数に関する記号的微分を計算") {
		t.Errorf("expected Japanese description for diff in ja locale, got: %s", jaOutput)
	}

	_ = i18n.Init("en")
	var enBuf bytes.Buffer
	if err := RunCompletion("powershell", &enBuf); err != nil {
		t.Fatalf("RunCompletion en failed: %v", err)
	}
	enOutput := enBuf.String()
	if !strings.Contains(enOutput, "Symbolic differentiation with respect to variable") {
		t.Errorf("expected English description for diff in en locale, got: %s", enOutput)
	}
}
