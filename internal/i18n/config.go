package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config represents persistent user configuration for ihd.
type Config struct {
	Locale string `json:"locale,omitempty"`
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
