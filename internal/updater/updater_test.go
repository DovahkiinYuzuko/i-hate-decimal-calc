package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestParseSemVer(t *testing.T) {
	tests := []struct {
		input    string
		expected [3]int
	}{
		{"v1.4.0", [3]int{1, 4, 0}},
		{"1.2.3", [3]int{1, 2, 3}},
		{"v2.0.0-beta.1", [3]int{2, 0, 0}},
		{"v3.1", [3]int{3, 1, 0}},
		{"5", [3]int{5, 0, 0}},
	}

	for _, tt := range tests {
		got := parseSemVer(tt.input)
		if got != tt.expected {
			t.Errorf("parseSemVer(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		latest   string
		current  string
		expected bool
	}{
		{"v1.4.1", "v1.4.0", true},
		{"v1.5.0", "v1.4.9", true},
		{"v2.0.0", "v1.9.9", true},
		{"v1.4.0", "v1.4.0", false},
		{"v1.3.9", "v1.4.0", false},
		{"v1.4.0", "v1.4.1", false},
		{"v1.4.1-alpha", "v1.4.0", true}, // 1.4.1 > 1.4.0
	}

	for _, tt := range tests {
		got := IsNewerVersion(tt.latest, tt.current)
		if got != tt.expected {
			t.Errorf("IsNewerVersion(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.expected)
		}
	}
}

func TestFormatNotification(t *testing.T) {
	info := &UpdateInfo{
		CurrentVersion: "v1.4.0",
		LatestVersion:  "v1.4.1",
		ReleaseURL:     "https://github.com/DovahkiinYuzuko/i-hate-decimal-calc/releases/tag/v1.4.1",
		UpdateCommand:  "irm https://... | iex",
	}

	msg := FormatNotification(info)
	if msg == "" {
		t.Fatal("expected non-empty notification message")
	}
	if !testing.Short() {
		t.Logf("Formatted message:\n%s", msg)
	}
}

func TestCheckUpdateWithClient_MockServer(t *testing.T) {
	// Set custom IHD_DIR to a temporary folder
	tmpDir := t.TempDir()
	t.Setenv("IHD_DIR", tmpDir)
	t.Setenv("IHD_NO_UPDATE_CHECK", "")

	requestCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"tag_name": "v1.5.0",
			"html_url": "https://github.com/DovahkiinYuzuko/i-hate-decimal-calc/releases/tag/v1.5.0",
		})
	}))
	defer ts.Close()

	// Initial check: should fetch from server and return update
	info, err := checkUpdateWithClient(false, ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected update info, got nil")
	}
	if info.LatestVersion != "v1.5.0" {
		t.Errorf("got latest version %s, want v1.5.0", info.LatestVersion)
	}
	if requestCount != 1 {
		t.Errorf("expected 1 HTTP request, got %d", requestCount)
	}

	// Verify cache was created
	cachePath := filepath.Join(tmpDir, "update_cache.json")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("cache file was not created")
	}

	// Second check: within 24 hours, should use cache and not make another HTTP request
	info2, err := checkUpdateWithClient(false, ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info2 == nil {
		t.Fatal("expected update info from cache, got nil")
	}
	if requestCount != 1 {
		t.Errorf("expected HTTP request count to stay 1 due to cache, got %d", requestCount)
	}

	// Third check: with force=true, should re-query
	_, err = checkUpdateWithClient(true, ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error on force: %v", err)
	}
	if requestCount != 2 {
		t.Errorf("expected HTTP request count to be 2 after force check, got %d", requestCount)
	}
}

func TestCheckUpdateWithClient_SilentFailOnServerError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("IHD_DIR", tmpDir)
	t.Setenv("IHD_NO_UPDATE_CHECK", "")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	// Should silently fail (return nil, nil)
	info, err := checkUpdateWithClient(true, ts.Client(), ts.URL)
	if err != nil {
		t.Errorf("expected silent fail (nil error), got %v", err)
	}
	if info != nil {
		t.Errorf("expected nil info, got %v", info)
	}
}

func TestCheckUpdateWithClient_DisabledByEnv(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("IHD_DIR", tmpDir)
	t.Setenv("IHD_NO_UPDATE_CHECK", "1")

	requestCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
	}))
	defer ts.Close()

	info, err := checkUpdateWithClient(false, ts.Client(), ts.URL)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if info != nil {
		t.Errorf("expected nil info when disabled, got %v", info)
	}
	if requestCount != 0 {
		t.Errorf("expected 0 HTTP requests when disabled, got %d", requestCount)
	}
}
