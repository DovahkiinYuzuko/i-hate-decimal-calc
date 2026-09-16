package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// CurrentVersion represents the active application version.
// It can be overridden at build time via -ldflags="-X github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/updater.CurrentVersion=v...".
var CurrentVersion = "v1.4.0"

const (
	RepoOwner      = "DovahkiinYuzuko"
	RepoName       = "i-hate-decimal-calc"
	CheckInterval  = 24 * time.Hour
	RequestTimeout = 1200 * time.Millisecond
)

// UpdateInfo holds details about an available newer release.
type UpdateInfo struct {
	CurrentVersion string
	LatestVersion  string
	ReleaseURL     string
	UpdateCommand  string
}

// CacheData stores the timestamp and version of the last update check.
type CacheData struct {
	LastCheckedAt time.Time `json:"last_checked_at"`
	LatestVersion string    `json:"latest_version"`
	ReleaseURL    string    `json:"release_url"`
}

// GetCachePath returns the path to ~/.ihd/update_cache.json.
func GetCachePath() string {
	return filepath.Join(i18n.GetIhdDir(), "update_cache.json")
}

// loadCache reads ~/.ihd/update_cache.json.
func loadCache() (*CacheData, error) {
	path := GetCachePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cache CacheData
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	return &cache, nil
}

// saveCache writes data to ~/.ihd/update_cache.json.
func saveCache(cache *CacheData) error {
	dir := i18n.GetIhdDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(GetCachePath(), data, 0644)
}

// IsUpdateCheckEnabled checks if update checks are enabled via env vars and config.
func IsUpdateCheckEnabled() bool {
	envVal := strings.ToLower(os.Getenv("IHD_NO_UPDATE_CHECK"))
	if envVal == "1" || envVal == "true" || envVal == "yes" {
		return false
	}

	cfg, err := i18n.LoadConfig()
	if err == nil && cfg != nil && cfg.CheckUpdates != nil && !*cfg.CheckUpdates {
		return false
	}

	return true
}

// CheckUpdate checks whether a newer release is available.
// When force is false, it respects the 24-hour cache interval.
// If an error occurs (e.g. offline, rate limit, timeout), it fails silently and returns nil, nil.
func CheckUpdate(force bool) (*UpdateInfo, error) {
	return checkUpdateWithClient(force, http.DefaultClient, fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", RepoOwner, RepoName))
}

// checkUpdateWithClient is an internal helper that accepts custom client and URL for testing.
func checkUpdateWithClient(force bool, client *http.Client, apiURL string) (*UpdateInfo, error) {
	if !IsUpdateCheckEnabled() && !force {
		return nil, nil
	}

	// 1. Check local cache
	cache, err := loadCache()
	if err == nil && cache != nil && !force {
		if time.Since(cache.LastCheckedAt) < CheckInterval {
			if IsNewerVersion(cache.LatestVersion, CurrentVersion) {
				return makeUpdateInfo(cache.LatestVersion, cache.ReleaseURL), nil
			}
			return nil, nil
		}
	}

	// 2. Fetch latest release from GitHub API
	ctx, cancel := context.WithTimeout(context.Background(), RequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, nil // Silent fail
	}
	req.Header.Set("User-Agent", "ihd-updater/"+CurrentVersion)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil // Silent fail on network error / timeout
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil // Silent fail on 403 rate limit, 404, 500, etc.
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, nil
	}

	if release.TagName == "" {
		return nil, nil
	}

	// 3. Update cache
	_ = saveCache(&CacheData{
		LastCheckedAt: time.Now().UTC(),
		LatestVersion: release.TagName,
		ReleaseURL:    release.HTMLURL,
	})

	// 4. Compare version
	if IsNewerVersion(release.TagName, CurrentVersion) {
		return makeUpdateInfo(release.TagName, release.HTMLURL), nil
	}

	return nil, nil
}

func makeUpdateInfo(latest, htmlURL string) *UpdateInfo {
	cmd := "curl -fsSL https://raw.githubusercontent.com/" + RepoOwner + "/" + RepoName + "/main/install.sh | bash"
	if runtime.GOOS == "windows" {
		cmd = "irm https://raw.githubusercontent.com/" + RepoOwner + "/" + RepoName + "/main/install.ps1 | iex"
	}

	if htmlURL == "" {
		htmlURL = fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", RepoOwner, RepoName, latest)
	}

	return &UpdateInfo{
		CurrentVersion: CurrentVersion,
		LatestVersion:  latest,
		ReleaseURL:     htmlURL,
		UpdateCommand:  cmd,
	}
}

// FormatNotification produces a clean, human-readable update notice.
func FormatNotification(info *UpdateInfo) string {
	if info == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(i18n.T("updater.new_version", info.LatestVersion, info.CurrentVersion))
	if info.ReleaseURL != "" {
		sb.WriteString(i18n.T("updater.release", info.ReleaseURL))
	}
	sb.WriteString(i18n.T("updater.to_update", info.UpdateCommand))
	return sb.String()
}

// IsNewerVersion returns true if latest is strictly newer than current according to SemVer.
func IsNewerVersion(latest, current string) bool {
	lParts := parseSemVer(latest)
	cParts := parseSemVer(current)

	for i := 0; i < 3; i++ {
		if lParts[i] > cParts[i] {
			return true
		}
		if lParts[i] < cParts[i] {
			return false
		}
	}
	return false
}

// parseSemVer extracts [major, minor, patch] integers from strings like "v1.4.2" or "1.4".
func parseSemVer(v string) [3]int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	v = strings.TrimPrefix(v, "V")

	// Strip pre-release or build metadata (e.g., "-beta", "+build")
	if idx := strings.IndexAny(v, "-+"); idx != -1 {
		v = v[:idx]
	}

	parts := strings.Split(v, ".")
	var res [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		if n, err := strconv.Atoi(parts[i]); err == nil {
			res[i] = n
		}
	}
	return res
}
