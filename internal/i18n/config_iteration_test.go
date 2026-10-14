package i18n

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_IterationDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("IHD_DIR", tmpDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	warnBig := GetWarnIterationsBig(cfg)
	expectedWarn := big.NewInt(100000)
	if warnBig.Cmp(expectedWarn) != 0 {
		t.Errorf("expected default warn iterations %v, got %v", expectedWarn, warnBig)
	}

	maxBig := GetMaxIterationsBig(cfg)
	expectedMax := big.NewInt(-1)
	if maxBig.Cmp(expectedMax) != 0 {
		t.Errorf("expected default max iterations %v, got %v", expectedMax, maxBig)
	}

	// nil cfg should also safely return defaults
	if GetWarnIterationsBig(nil).Cmp(expectedWarn) != 0 {
		t.Errorf("nil cfg should return default warn iterations %v", expectedWarn)
	}
	if GetMaxIterationsBig(nil).Cmp(expectedMax) != 0 {
		t.Errorf("nil cfg should return default max iterations %v", expectedMax)
	}
}

func TestConfig_IterationSaveAndLoadBig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("IHD_DIR", tmpDir)

	warnVal := big.NewInt(50000)
	if err := SaveConfigWarnIterationsBig(warnVal); err != nil {
		t.Fatalf("SaveConfigWarnIterationsBig failed: %v", err)
	}

	// Giant integer exceeding 64-bit int (10^40)
	giantMax, ok := new(big.Int).SetString("10000000000000000000000000000000000000000", 10)
	if !ok {
		t.Fatalf("failed to parse test giantMax")
	}

	if err := SaveConfigMaxIterationsBig(giantMax); err != nil {
		t.Fatalf("SaveConfigMaxIterationsBig failed: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	loadedWarn := GetWarnIterationsBig(cfg)
	if loadedWarn.Cmp(warnVal) != 0 {
		t.Errorf("expected warn iterations %v, got %v", warnVal, loadedWarn)
	}

	loadedMax := GetMaxIterationsBig(cfg)
	if loadedMax.Cmp(giantMax) != 0 {
		t.Errorf("expected max iterations %v, got %v", giantMax, loadedMax)
	}

	// Reset to -1 (unlimited)
	if err := SaveConfigMaxIterationsBig(big.NewInt(-1)); err != nil {
		t.Fatalf("SaveConfigMaxIterationsBig reset failed: %v", err)
	}

	cfgAfterReset, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig after reset failed: %v", err)
	}
	if GetMaxIterationsBig(cfgAfterReset).Cmp(big.NewInt(-1)) != 0 {
		t.Errorf("expected max iterations -1 after reset, got %v", GetMaxIterationsBig(cfgAfterReset))
	}
}

func TestConfig_IterationJSONCompat(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("IHD_DIR", tmpDir)

	// User wrote integers as raw JSON numbers instead of strings
	rawJSON := `{
  "warn_iterations": 25000,
  "max_iterations": 999999
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(rawJSON), 0644); err != nil {
		t.Fatalf("failed to write raw JSON: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if GetWarnIterationsBig(cfg).Cmp(big.NewInt(25000)) != 0 {
		t.Errorf("expected warn iterations 25000, got %v", GetWarnIterationsBig(cfg))
	}
	if GetMaxIterationsBig(cfg).Cmp(big.NewInt(999999)) != 0 {
		t.Errorf("expected max iterations 999999, got %v", GetMaxIterationsBig(cfg))
	}
}
