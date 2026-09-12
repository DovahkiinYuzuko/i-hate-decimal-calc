package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed locales/*.json
var embeddedLocales embed.FS

// DefaultLocale is the fallback language code.
const DefaultLocale = "en"

// EngineState represents the FSM state of the i18n engine.
type EngineState int

const (
	StateUninitialized EngineState = iota
	StateReady
	StateError
)

// LocaleMeta contains language metadata defined inside the JSON.
type LocaleMeta struct {
	Name      string `json:"name"`
	NameLocal string `json:"name_local"`
}

// Engine manages loaded locales, FSM state, and key translations.
type Engine struct {
	mu             sync.RWMutex
	state          EngineState
	currentLocale  string
	locales        map[string]map[string]any
	metadata       map[string]LocaleMeta
	availableList  []string
}

var globalEngine = &Engine{
	state:   StateUninitialized,
	locales: make(map[string]map[string]any),
	metadata: make(map[string]LocaleMeta),
}

// Init initializes the global i18n engine.
// Cascades: customLocale > env IHD_LANG > config.json > OS environment > default "en".
func Init(customLocale string) error {
	return globalEngine.Init(customLocale)
}

// T translates a key into the localized text with optional fmt formatting.
func T(key string, args ...any) string {
	return globalEngine.T(key, args...)
}

// SetLocale changes the active locale.
func SetLocale(locale string) error {
	return globalEngine.SetLocale(locale)
}

// CurrentLocale returns the currently active locale code.
func CurrentLocale() string {
	return globalEngine.CurrentLocale()
}

// CurrentLocaleName returns the local display name of the current locale.
func CurrentLocaleName() string {
	return globalEngine.CurrentLocaleName()
}

// AvailableLocales returns all discovered locales and their metadata.
func AvailableLocales() map[string]LocaleMeta {
	return globalEngine.AvailableLocales()
}

// GetState returns the current FSM state of the engine.
func GetState() EngineState {
	globalEngine.mu.RLock()
	defer globalEngine.mu.RUnlock()
	return globalEngine.state
}

// Init implements the Engine initialization logic.
func (e *Engine) Init(customLocale string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.locales = make(map[string]map[string]any)
	e.metadata = make(map[string]LocaleMeta)

	// 1. Load embedded locales
	entries, err := embeddedLocales.ReadDir("locales")
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			localeCode := strings.TrimSuffix(entry.Name(), ".json")
			data, readErr := embeddedLocales.ReadFile("locales/" + entry.Name())
			if readErr == nil {
				e.parseAndRegisterLocale(localeCode, data)
			}
		}
	}

	// 2. Discover external locales (rokeeru plumbing pattern)
	searchDirs := []string{
		filepath.Join(GetIhdDir(), "locales"),
		"locales",
	}
	for _, dir := range searchDirs {
		dirEntries, dirErr := os.ReadDir(dir)
		if dirErr == nil {
			for _, entry := range dirEntries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
					continue
				}
				localeCode := strings.TrimSuffix(entry.Name(), ".json")
				data, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
				if readErr == nil {
					e.parseAndRegisterLocale(localeCode, data)
				}
			}
		}
	}

	// 3. Resolve target locale via cascade
	target := e.resolveLocaleCascade(customLocale)
	if _, exists := e.locales[target]; !exists {
		target = DefaultLocale
	}
	e.currentLocale = target
	e.state = StateReady
	return nil
}

func (e *Engine) parseAndRegisterLocale(code string, data []byte) {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return
	}

	// Extract meta if present
	var meta LocaleMeta
	if m, ok := root["meta"].(map[string]any); ok {
		if name, ok := m["name"].(string); ok {
			meta.Name = name
		}
		if nameLocal, ok := m["name_local"].(string); ok {
			meta.NameLocal = nameLocal
		}
	}
	if meta.NameLocal == "" {
		meta.NameLocal = code
	}
	if meta.Name == "" {
		meta.Name = code
	}

	e.metadata[code] = meta
	e.locales[code] = root
}

func (e *Engine) resolveLocaleCascade(custom string) string {
	if custom != "" {
		return normalizeLocaleCode(custom)
	}

	if env := os.Getenv("IHD_LANG"); env != "" {
		return normalizeLocaleCode(env)
	}

	if cfg, err := LoadConfig(); err == nil && cfg.Locale != "" {
		return normalizeLocaleCode(cfg.Locale)
	}

	// OS environment (LANG, LC_ALL, LC_MESSAGES)
	for _, envVar := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if val := os.Getenv(envVar); val != "" {
			return normalizeLocaleCode(val)
		}
	}

	return DefaultLocale
}

func normalizeLocaleCode(code string) string {
	code = strings.TrimSpace(code)
	if idx := strings.IndexAny(code, "._@"); idx != -1 {
		code = code[:idx]
	}
	return strings.ToLower(code)
}

// T resolves a dot-notated key with fallback to DefaultLocale and fmt formatting.
func (e *Engine) T(key string, args ...any) string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.state != StateReady {
		// Auto-initialize with default if called before Init
		e.mu.RUnlock()
		_ = e.Init("")
		e.mu.RLock()
	}

	tmpl := e.lookupKey(e.currentLocale, key)
	if tmpl == "" && e.currentLocale != DefaultLocale {
		tmpl = e.lookupKey(DefaultLocale, key)
	}
	if tmpl == "" {
		return key // fallback to key name
	}

	if len(args) > 0 {
		return fmt.Sprintf(tmpl, args...)
	}
	return tmpl
}

func (e *Engine) lookupKey(locale, key string) string {
	dict, exists := e.locales[locale]
	if !exists {
		return ""
	}

	parts := strings.Split(key, ".")
	var curr any = dict

	for _, part := range parts {
		m, ok := curr.(map[string]any)
		if !ok {
			return ""
		}
		curr, ok = m[part]
		if !ok {
			return ""
		}
	}

	if s, ok := curr.(string); ok {
		return s
	}
	return ""
}

// SetLocale sets the active locale.
func (e *Engine) SetLocale(locale string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	code := normalizeLocaleCode(locale)
	if _, exists := e.locales[code]; !exists {
		return fmt.Errorf("unsupported locale: %s", locale)
	}

	e.currentLocale = code
	return nil
}

// CurrentLocale returns the code of the active locale.
func (e *Engine) CurrentLocale() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.currentLocale
}

// CurrentLocaleName returns the localized name of the active locale.
func (e *Engine) CurrentLocaleName() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if meta, exists := e.metadata[e.currentLocale]; exists && meta.NameLocal != "" {
		return meta.NameLocal
	}
	return e.currentLocale
}

// AvailableLocales returns a copy of all loaded locale metadata.
func (e *Engine) AvailableLocales() map[string]LocaleMeta {
	e.mu.RLock()
	defer e.mu.RUnlock()
	res := make(map[string]LocaleMeta, len(e.metadata))
	for k, v := range e.metadata {
		res[k] = v
	}
	return res
}
