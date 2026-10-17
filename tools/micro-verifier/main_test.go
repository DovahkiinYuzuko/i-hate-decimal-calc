package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyCertificateFile_Valid(t *testing.T) {
	tempDir := t.TempDir()
	certPath := filepath.Join(tempDir, "valid_cert.json")

	validJSON := `{
  "version": "1.0",
  "original": "x^2 - 1",
  "result": "(x - 1)*(x + 1)",
  "is_verified": true,
  "total_steps": 2,
  "steps": [
    {
      "step_number": 1,
      "before": "x^2 - 1",
      "after": "(x - 1)*(x + 1)",
      "rule": "Factorization",
      "residual": "0",
      "is_self_verified": true
    },
    {
      "step_number": 2,
      "before": "(x - 1)*(x + 1)",
      "after": "(x - 1)*(x + 1)",
      "rule": "Identity",
      "residual": "0",
      "is_self_verified": true
    }
  ]
}`

	if err := os.WriteFile(certPath, []byte(validJSON), 0644); err != nil {
		t.Fatalf("failed to write test cert: %v", err)
	}

	ok, err := verifyCertificateFile(certPath)
	if err != nil || !ok {
		t.Fatalf("expected valid verification, got ok=%v, err=%v", ok, err)
	}
}

func TestVerifyCertificateFile_TransitivityBreak(t *testing.T) {
	tempDir := t.TempDir()
	certPath := filepath.Join(tempDir, "broken_chain.json")

	brokenJSON := `{
  "version": "1.0",
  "original": "x^2 - 1",
  "result": "(x - 1)*(x + 1)",
  "is_verified": true,
  "total_steps": 2,
  "steps": [
    {
      "step_number": 1,
      "before": "x^2 - 1",
      "after": "x^2 + 1",
      "rule": "Factorization",
      "residual": "0",
      "is_self_verified": true
    },
    {
      "step_number": 2,
      "before": "(x - 1)*(x + 1)",
      "after": "(x - 1)*(x + 1)",
      "rule": "Identity",
      "residual": "0",
      "is_self_verified": true
    }
  ]
}`

	if err := os.WriteFile(certPath, []byte(brokenJSON), 0644); err != nil {
		t.Fatalf("failed to write test cert: %v", err)
	}

	ok, err := verifyCertificateFile(certPath)
	if ok || err == nil {
		t.Fatalf("expected transitivity break error, got ok=%v, err=%v", ok, err)
	}
}

func TestVerifyCertificateFile_UnverifiedStep(t *testing.T) {
	tempDir := t.TempDir()
	certPath := filepath.Join(tempDir, "unverified.json")

	unverifiedJSON := `{
  "version": "1.0",
  "original": "x",
  "result": "x",
  "is_verified": false,
  "total_steps": 1,
  "steps": [
    {
      "step_number": 1,
      "before": "x",
      "after": "x",
      "rule": "Identity",
      "residual": "0",
      "is_self_verified": false
    }
  ]
}`

	if err := os.WriteFile(certPath, []byte(unverifiedJSON), 0644); err != nil {
		t.Fatalf("failed to write test cert: %v", err)
	}

	ok, err := verifyCertificateFile(certPath)
	if ok || err == nil {
		t.Fatalf("expected unverified step error, got ok=%v, err=%v", ok, err)
	}
}
