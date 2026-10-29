package i18n

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
)

// Config represents persistent user configuration for ihd.
type Config struct {
	Locale          string          `json:"locale,omitempty"`
	CheckUpdates    *bool           `json:"check_updates,omitempty"`
	Theme           string          `json:"theme,omitempty"`
	SyntaxHighlight *bool           `json:"syntax_highlight,omitempty"`
	WarnIterations  json.RawMessage `json:"warn_iterations,omitempty"`
	MaxIterations   json.RawMessage `json:"max_iterations,omitempty"`
	RewriteMaxSteps *int            `json:"rewrite_max_steps,omitempty"`
	Timeout         *string         `json:"timeout,omitempty"`
	BudgetSteps     *uint64         `json:"budget_steps,omitempty"`
	MaxNodes        *uint64         `json:"max_nodes,omitempty"`
	MaxDepth        *uint64         `json:"max_depth,omitempty"`
	Deterministic   *bool           `json:"deterministic,omitempty"`
}

// GetIhdDir returns the path to the ~/.ihd directory.
func GetIhdDir() string {
	if custom := os.Getenv("IHD_DIR"); custom != "" {
		return custom
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".ihd"
	}
	return filepath.Join(home, ".ihd")
}

// GetConfigPath returns the path to ~/.ihd/config.json.
func GetConfigPath() string {
	return filepath.Join(GetIhdDir(), "config.json")
}

// LoadConfig reads the configuration file from ~/.ihd/config.json.
// If the file does not exist, an empty Config is returned with nil error.
func LoadConfig() (*Config, error) {
	path := GetConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &Config{}, nil // fallback on parse error
	}
	return &cfg, nil
}

// SaveConfig writes the configuration to ~/.ihd/config.json.
func SaveConfig(cfg *Config) error {
	dir := GetIhdDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(GetConfigPath(), data, 0644)
}

// SaveConfigLocale updates the locale entry in ~/.ihd/config.json.
func SaveConfigLocale(locale string) error {
	cfg, _ := LoadConfig()
	if cfg == nil {
		cfg = &Config{}
	}
	cfg.Locale = locale
	return SaveConfig(cfg)
}

// SaveConfigTheme updates the theme entry in ~/.ihd/config.json.
func SaveConfigTheme(theme string) error {
	cfg, _ := LoadConfig()
	if cfg == nil {
		cfg = &Config{}
	}
	cfg.Theme = theme
	return SaveConfig(cfg)
}

// SaveConfigHighlight updates the syntax_highlight entry in ~/.ihd/config.json.
func SaveConfigHighlight(enabled bool) error {
	cfg, _ := LoadConfig()
	if cfg == nil {
		cfg = &Config{}
	}
	cfg.SyntaxHighlight = &enabled
	return SaveConfig(cfg)
}

func parseBigIntFromRaw(raw json.RawMessage, defaultVal *big.Int) *big.Int {
	if len(raw) == 0 {
		return new(big.Int).Set(defaultVal)
	}
	s := strings.Trim(string(raw), "\" \t\r\n")
	if s == "" || s == "null" {
		return new(big.Int).Set(defaultVal)
	}
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return new(big.Int).Set(defaultVal)
	}
	return n
}

// GetWarnIterationsBig returns the warning iteration threshold as *big.Int.
// Defaults to 100,000 if not specified or invalid.
func GetWarnIterationsBig(cfg *Config) *big.Int {
	defaultVal := big.NewInt(100000)
	if cfg == nil {
		return defaultVal
	}
	return parseBigIntFromRaw(cfg.WarnIterations, defaultVal)
}

// GetWarnIterationsBig returns the warning iteration threshold for this config as *big.Int.
func (c *Config) GetWarnIterationsBig() *big.Int {
	return GetWarnIterationsBig(c)
}

// GetMaxIterationsBig returns the maximum iteration limit as *big.Int.
// Defaults to -1 (unlimited) if not specified or invalid.
func GetMaxIterationsBig(cfg *Config) *big.Int {
	defaultVal := big.NewInt(-1)
	if cfg == nil {
		return defaultVal
	}
	return parseBigIntFromRaw(cfg.MaxIterations, defaultVal)
}

// GetMaxIterationsBig returns the maximum iteration limit for this config as *big.Int.
func (c *Config) GetMaxIterationsBig() *big.Int {
	return GetMaxIterationsBig(c)
}

// SaveConfigWarnIterationsBig updates the warn_iterations entry in ~/.ihd/config.json.
func SaveConfigWarnIterationsBig(val *big.Int) error {
	cfg, _ := LoadConfig()
	if cfg == nil {
		cfg = &Config{}
	}
	if val == nil {
		val = big.NewInt(100000)
	}
	cfg.WarnIterations = json.RawMessage([]byte(val.String()))
	return SaveConfig(cfg)
}

// SaveConfigMaxIterationsBig updates the max_iterations entry in ~/.ihd/config.json.
func SaveConfigMaxIterationsBig(val *big.Int) error {
	cfg, _ := LoadConfig()
	if cfg == nil {
		cfg = &Config{}
	}
	if val == nil {
		val = big.NewInt(-1)
	}
	cfg.MaxIterations = json.RawMessage([]byte(val.String()))
	return SaveConfig(cfg)
}

// GetRewriteMaxSteps returns the maximum rewrite steps.
// Defaults to 1000 if not specified or invalid.
func GetRewriteMaxSteps(cfg *Config) int {
	defaultVal := 1000
	if cfg == nil || cfg.RewriteMaxSteps == nil || *cfg.RewriteMaxSteps <= 0 {
		return defaultVal
	}
	return *cfg.RewriteMaxSteps
}

// GetRewriteMaxSteps returns the maximum rewrite steps for this config.
func (c *Config) GetRewriteMaxSteps() int {
	return GetRewriteMaxSteps(c)
}

// SaveConfigRewriteMaxSteps updates the rewrite_max_steps entry in ~/.ihd/config.json.
func SaveConfigRewriteMaxSteps(steps int) error {
	cfg, _ := LoadConfig()
	if cfg == nil {
		cfg = &Config{}
	}
	cfg.RewriteMaxSteps = &steps
	return SaveConfig(cfg)
}



