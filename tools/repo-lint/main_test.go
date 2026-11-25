package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLIListRules(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"--list-rules"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d (stderr: %s)", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"git-whitelist", "i18n-ast", "notice-sync", "readme-quickstart", "lean-pinning", "e2e-csv", "go-hygiene-empty-err-branch"} {
		if !strings.Contains(out, want) {
			t.Errorf("--list-rules missing %q:\n%s", want, out)
		}
	}
}
