package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLI_Update_Help(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"update", "--help"}, strings.NewReader(""), &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit code 0 for update --help, got %d. stderr: %s", code, errOut.String())
	}
	output := out.String()
	if !strings.Contains(output, "ihd update") {
		t.Errorf("expected output to contain 'ihd update', got: %s", output)
	}
}

func TestCLI_Update_UnknownFlag(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"update", "--nonexistent-flag"}, strings.NewReader(""), &out, &errOut)
	if code != 1 {
		t.Errorf("expected exit code 1 for invalid flag, got %d", code)
	}
}
