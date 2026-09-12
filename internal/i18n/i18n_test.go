package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

func TestI18n_EmbeddedLocales(t *testing.T) {
	err := Init("en")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if CurrentLocale() != "en" {
		t.Errorf("expected current locale 'en', got %q", CurrentLocale())
	}
	if CurrentLocaleName() != "English" {
		t.Errorf("expected 'English', got %q", CurrentLocaleName())
	}

	res := T("cli.repl_welcome")
	if res != "ihd: Exact Arithmetic Calculator" {
		t.Errorf("unexpected welcome message: %q", res)
	}

	// Switch to ja
	err = SetLocale("ja")
	if err != nil {
		t.Fatalf("SetLocale('ja') failed: %v", err)
	}
	if CurrentLocale() != "ja" {
		t.Errorf("expected current locale 'ja', got %q", CurrentLocale())
	}
	if CurrentLocaleName() != "日本語" {
		t.Errorf("expected '日本語', got %q", CurrentLocaleName())
	}

	// Test formatting with arguments
	zeroMsg := T("plot.zero", "√2")
	if zeroMsg != "* 零点 (Zero):       x = √2\n" {
		t.Errorf("unexpected formatted zero message: %q", zeroMsg)
	}
}

func TestI18n_FallbackToDefault(t *testing.T) {
	err := Init("ja")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Test a key that only exists in en or fallback behavior
	// If a key doesn't exist in ja, it should fall back to en.
	// For testing, mock a key or ensure nonexistent key falls back to key name
	nonExistent := T("non.existent.key")
	if nonExistent != "non.existent.key" {
		t.Errorf("expected key name fallback, got %q", nonExistent)
	}
}

func TestI18n_ExternalDiscovery(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("IHD_DIR", tmpDir)

	localesDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localesDir, 0755); err != nil {
		t.Fatalf("failed to create temp locales dir: %v", err)
	}

	frJson := `{
  "meta": {
    "name": "French",
    "name_local": "Français"
  },
  "cli": {
    "repl_welcome": "ihd: Calculateur Arithmétique Exact"
  }
}`
	if err := os.WriteFile(filepath.Join(localesDir, "fr.json"), []byte(frJson), 0644); err != nil {
		t.Fatalf("failed to write fr.json: %v", err)
	}

	// Re-init with fr
	err := Init("fr")
	if err != nil {
		t.Fatalf("Init('fr') failed: %v", err)
	}

	if CurrentLocale() != "fr" {
		t.Errorf("expected 'fr', got %q", CurrentLocale())
	}
	if CurrentLocaleName() != "Français" {
		t.Errorf("expected 'Français', got %q", CurrentLocaleName())
	}

	res := T("cli.repl_welcome")
	if res != "ihd: Calculateur Arithmétique Exact" {
		t.Errorf("expected French welcome, got %q", res)
	}

	// Un-translated key in French should fall back to English (default)
	errPrefix := T("cli.error_prefix")
	if errPrefix != "Error: " {
		t.Errorf("expected fallback to English 'Error: ', got %q", errPrefix)
	}
}

func TestConfig_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("IHD_DIR", tmpDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Locale != "" {
		t.Errorf("expected empty initial config, got %q", cfg.Locale)
	}

	err = SaveConfigLocale("ja")
	if err != nil {
		t.Fatalf("SaveConfigLocale failed: %v", err)
	}

	cfgAfter, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig after save failed: %v", err)
	}
	if cfgAfter.Locale != "ja" {
		t.Errorf("expected saved locale 'ja', got %q", cfgAfter.Locale)
	}
}
