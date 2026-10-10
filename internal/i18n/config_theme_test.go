package i18n

import (
	"testing"
)

func TestConfig_SaveAndLoadThemeAndHighlight(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("IHD_DIR", tmpDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Theme != "" {
		t.Errorf("expected empty initial Theme, got %q", cfg.Theme)
	}
	if cfg.SyntaxHighlight != nil {
		t.Errorf("expected nil initial SyntaxHighlight, got %v", *cfg.SyntaxHighlight)
	}

	// Save Theme
	if err := SaveConfigTheme("light"); err != nil {
		t.Fatalf("SaveConfigTheme failed: %v", err)
	}

	// Save Highlight
	if err := SaveConfigHighlight(false); err != nil {
		t.Fatalf("SaveConfigHighlight failed: %v", err)
	}

	cfgAfter, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig after save failed: %v", err)
	}
	if cfgAfter.Theme != "light" {
		t.Errorf("expected Theme 'light', got %q", cfgAfter.Theme)
	}
	if cfgAfter.SyntaxHighlight == nil || *cfgAfter.SyntaxHighlight != false {
		t.Errorf("expected SyntaxHighlight false, got %v", cfgAfter.SyntaxHighlight)
	}
}
