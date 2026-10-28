package updater

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectPackageManager(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"Homebrew opt", "/opt/homebrew/bin/ihd", "homebrew"},
		{"Homebrew Cellar", "/usr/local/Cellar/ihd/0.8.0/bin/ihd", "homebrew"},
		{"Linuxbrew", "/home/linuxbrew/.linuxbrew/bin/ihd", "homebrew"},
		{"Scoop apps", `C:\Users\user\scoop\apps\ihd\0.8.0\ihd.exe`, "scoop"},
		{"Scoop shims", `C:\Users\user\scoop\shims\ihd.exe`, "scoop"},
		{"Standard Unix ~/.ihd/bin", "/home/user/.ihd/bin/ihd", ""},
		{"Standard Windows ~/.ihd/bin", `C:\Users\user\.ihd\bin\ihd.exe`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectPackageManager(tt.path)
			if got != tt.expected {
				t.Errorf("DetectPackageManager(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestVerifyChecksum(t *testing.T) {
	data := []byte("hello world i hate decimal calc")
	hasher := sha256.New()
	hasher.Write(data)
	validHex := hex.EncodeToString(hasher.Sum(nil))

	if err := VerifyChecksum(data, validHex); err != nil {
		t.Fatalf("expected valid checksum to pass, got: %v", err)
	}

	invalidHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if err := VerifyChecksum(data, invalidHex); err == nil {
		t.Fatalf("expected invalid checksum to fail, got nil")
	}
}

func TestApplyBinaryReplacement(t *testing.T) {
	tempDir := t.TempDir()

	targetPath := filepath.Join(tempDir, "ihd-target.exe")
	newPath := filepath.Join(tempDir, "ihd-new.exe")

	if err := os.WriteFile(targetPath, []byte("version 1.0"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newPath, []byte("version 2.0"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := ApplyBinaryReplacement(newPath, targetPath); err != nil {
		t.Fatalf("ApplyBinaryReplacement failed: %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read replaced target: %v", err)
	}
	if string(content) != "version 2.0" {
		t.Fatalf("expected target to contain 'version 2.0', got: %s", string(content))
	}
}

func TestPerformSelfUpdate_CheckOnly(t *testing.T) {
	mockClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			body := `{"tag_name":"v99.0.0","html_url":"https://github.com/DovahkiinYuzuko/i-hate-decimal-calc/releases/tag/v99.0.0"}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}),
	}

	var out bytes.Buffer
	opts := SelfUpdateOptions{
		CheckOnly:  true,
		ReleaseURL: "https://api.github.com/mock/releases/latest",
		HTTPClient: mockClient,
		Out:        &out,
	}

	if err := PerformSelfUpdate(opts); err != nil {
		t.Fatalf("PerformSelfUpdate with CheckOnly failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "v99.0.0") {
		t.Errorf("expected output to contain v99.0.0, got: %s", output)
	}
}
