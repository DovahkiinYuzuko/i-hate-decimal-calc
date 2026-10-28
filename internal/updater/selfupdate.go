package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// UpdateState represents the discrete step of self-update FSM.
type UpdateState int

const (
	StateIdle UpdateState = iota
	StateChecking
	StateDownloading
	StateVerifying
	StateStaging
	StateCommitting
	StateSyncingAssets
	StateCompleted
	StateRollingBack
	StateFailed
)

// SelfUpdateOptions configures the self-update execution.
type SelfUpdateOptions struct {
	Force      bool
	CheckOnly  bool
	Out        io.Writer
	ErrOut     io.Writer
	ReleaseURL string // Overrides default GitHub releases URL (for testing)
	HTTPClient *http.Client
}

// DetectPackageManager detects whether the executable is inside a managed package manager directory.
func DetectPackageManager(executablePath string) string {
	cleanPath := filepath.ToSlash(filepath.Clean(executablePath))
	lowerPath := strings.ToLower(cleanPath)

	// Homebrew detection (macOS / Linux)
	if strings.Contains(lowerPath, "/opt/homebrew/") ||
		strings.Contains(lowerPath, "/cellar/") ||
		strings.Contains(lowerPath, "/.linuxbrew/") {
		return "homebrew"
	}

	// Scoop detection (Windows)
	if strings.Contains(lowerPath, "/scoop/apps/") || strings.Contains(lowerPath, "/scoop/shims/") {
		return "scoop"
	}

	return ""
}

// VerifyChecksum validates the SHA256 hex string of the provided data.
func VerifyChecksum(data []byte, expectedHex string) error {
	expected := strings.TrimSpace(strings.ToLower(expectedHex))
	hasher := sha256.New()
	hasher.Write(data)
	actual := hex.EncodeToString(hasher.Sum(nil))

	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}

// ApplyBinaryReplacement replaces targetPath with newBinaryPath atomically or via 2-stage staged rename.
func ApplyBinaryReplacement(newBinaryPath, targetPath string) error {
	dir := filepath.Dir(targetPath)
	filename := filepath.Base(targetPath)
	oldPath := filepath.Join(dir, fmt.Sprintf(".%s.old", filename))

	// Remove any leftover .old file from previous update (crucial on Windows)
	_ = os.Remove(oldPath)

	// Step 1: Move current binary out of the way
	if err := os.Rename(targetPath, oldPath); err != nil {
		return fmt.Errorf("failed to move current binary to old backup: %w", err)
	}

	// Step 2: Move new binary into target place
	if err := os.Rename(newBinaryPath, targetPath); err != nil {
		// Rollback immediately
		rerr := os.Rename(oldPath, targetPath)
		if rerr != nil {
			return fmt.Errorf("failed to replace binary (%v), and rollback also failed (%v)", err, rerr)
		}
		return fmt.Errorf("failed to move new binary into place (rolled back): %w", err)
	}

	// Step 3: Best-effort remove old binary (on Windows, active running binary may block deletion, so ignore error)
	_ = os.Remove(oldPath)

	return nil
}

type gitHubReleaseResponse struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// PerformSelfUpdate executes the full self-update workflow with FSM state progression.
func PerformSelfUpdate(opts SelfUpdateOptions) error {
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}
	errOut := opts.ErrOut
	if errOut == nil {
		errOut = os.Stderr
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot resolve current executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("cannot resolve symlinks for executable: %w", err)
	}

	// Step 1: Check package manager
	if pkg := DetectPackageManager(exePath); pkg != "" {
		updateCmd := "brew upgrade ihd"
		if pkg == "scoop" {
			updateCmd = "scoop update ihd"
		}
		fmt.Fprintf(errOut, i18n.T("updater.pkg_manager_detected"), pkg, updateCmd)
		return nil
	}

	// Step 2: Check latest version
	fmt.Fprint(out, i18n.T("updater.checking"))
	apiURL := opts.ReleaseURL
	if apiURL == "" {
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", RepoOwner, RepoName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "ihd-updater/"+CurrentVersion)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("network error while fetching release metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned status: %s", resp.Status)
	}

	var rel gitHubReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return fmt.Errorf("failed to decode release response: %w", err)
	}

	latestTag := normalizeVersion(rel.TagName)
	if !opts.Force && !IsNewerVersion(latestTag, CurrentVersion) {
		fmt.Fprintf(out, i18n.T("updater.already_latest"), CurrentVersion)
		return nil
	}

	if opts.CheckOnly {
		fmt.Fprintf(out, i18n.T("updater.new_version"), latestTag, CurrentVersion)
		fmt.Fprintf(out, i18n.T("updater.release"), rel.HTMLURL)
		return nil
	}

	// Determine matching asset name based on OS and Arch
	archiveExt := "tar.gz"
	if runtime.GOOS == "windows" {
		archiveExt = "zip"
	}
	expectedAsset := fmt.Sprintf("ihd-%s-%s.%s", runtime.GOOS, runtime.GOARCH, archiveExt)

	var assetURL string
	var checksumsURL string

	for _, a := range rel.Assets {
		if a.Name == expectedAsset {
			assetURL = a.BrowserDownloadURL
		}
		if a.Name == "checksums.txt" {
			checksumsURL = a.BrowserDownloadURL
		}
	}

	if assetURL == "" {
		return fmt.Errorf("no suitable asset found for platform %s/%s (%s)", runtime.GOOS, runtime.GOARCH, expectedAsset)
	}

	// Download checksums.txt if available
	var expectedChecksum string
	if checksumsURL != "" {
		fmt.Fprint(out, i18n.T("updater.verifying"))
		csData, err := downloadBytes(ctx, client, checksumsURL)
		if err == nil {
			expectedChecksum = parseChecksumForAsset(string(csData), expectedAsset)
		}
	}

	// Download release archive
	fmt.Fprintf(out, i18n.T("updater.downloading"), latestTag, expectedAsset)
	archiveData, err := downloadBytes(ctx, client, assetURL)
	if err != nil {
		return fmt.Errorf("failed to download release asset: %w", err)
	}

	// Verify SHA256 checksum
	if expectedChecksum != "" {
		if err := VerifyChecksum(archiveData, expectedChecksum); err != nil {
			return err
		}
	}

	// Staging: Unpack into temporary directory
	fmt.Fprint(out, i18n.T("updater.staging"))
	stageDir, err := os.MkdirTemp("", "ihd-update-stage-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary staging directory: %w", err)
	}
	defer os.RemoveAll(stageDir)

	if archiveExt == "zip" {
		if err := extractZip(archiveData, stageDir); err != nil {
			return fmt.Errorf("failed to extract zip archive: %w", err)
		}
	} else {
		if err := extractTarGz(archiveData, stageDir); err != nil {
			return fmt.Errorf("failed to extract tar.gz archive: %w", err)
		}
	}

	// Locate new binary in stageDir
	binName := "ihd"
	if runtime.GOOS == "windows" {
		binName = "ihd.exe"
	}
	stagedBinaryPath := filepath.Join(stageDir, binName)
	if _, err := os.Stat(stagedBinaryPath); err != nil {
		return fmt.Errorf("staged binary %s not found in release archive: %w", binName, err)
	}
	_ = os.Chmod(stagedBinaryPath, 0755)

	// Step: Binary Replacement
	fmt.Fprint(out, i18n.T("updater.applying"))
	// Copy staged binary to a temporary file on the SAME volume/directory as target binary
	targetDir := filepath.Dir(exePath)
	tempNewBinary := filepath.Join(targetDir, fmt.Sprintf(".%s.new", filepath.Base(exePath)))
	if err := copyFile(stagedBinaryPath, tempNewBinary, 0755); err != nil {
		if os.IsPermission(err) {
			fmt.Fprintf(errOut, i18n.T("updater.permission_denied"), targetDir)
			return err
		}
		return fmt.Errorf("failed to stage new binary in target directory: %w", err)
	}
	defer os.Remove(tempNewBinary)

	if err := ApplyBinaryReplacement(tempNewBinary, exePath); err != nil {
		return err
	}

	// Also replace micro-verifier if present in stageDir and target directory
	verifierName := "micro-verifier"
	if runtime.GOOS == "windows" {
		verifierName = "micro-verifier.exe"
	}
	stagedVerifier := filepath.Join(stageDir, verifierName)
	targetVerifier := filepath.Join(targetDir, verifierName)
	if _, err := os.Stat(stagedVerifier); err == nil {
		tempNewVerifier := filepath.Join(targetDir, fmt.Sprintf(".%s.new", verifierName))
		if copyErr := copyFile(stagedVerifier, tempNewVerifier, 0755); copyErr == nil {
			_ = ApplyBinaryReplacement(tempNewVerifier, targetVerifier)
		}
	}

	// Step: Sync bundled assets (exam, references, tools) if standard ~/.ihd structure
	ihdHome := i18n.GetIhdDir()
	if strings.HasPrefix(filepath.Clean(exePath), filepath.Clean(ihdHome)) {
		fmt.Fprint(out, i18n.T("updater.syncing_assets"))
		syncAssets(stageDir, ihdHome)
	}

	fmt.Fprintf(out, i18n.T("updater.success"), CurrentVersion, latestTag)
	return nil
}

func downloadBytes(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ihd-updater/"+CurrentVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func parseChecksumForAsset(checksumsContent, assetName string) string {
	lines := strings.Split(checksumsContent, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			filename := filepath.Base(fields[1])
			if filename == assetName {
				return fields[0]
			}
		}
	}
	return ""
}

func extractZip(data []byte, destDir string) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, f := range r.File {
		targetPath := filepath.Join(destDir, f.Name)
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(destDir)) {
			continue // Protect against ZipSlip
		}
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(targetPath, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTarGz(data []byte, destDir string) error {
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		targetPath := filepath.Join(destDir, header.Name)
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(destDir)) {
			continue // Protect against ZipSlip
		}
		switch header.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(targetPath, 0755)
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			_, err = io.Copy(outFile, tr)
			outFile.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func syncAssets(stageDir, ihdHome string) {
	// Sync exam directory
	stageExam := filepath.Join(stageDir, "exam")
	if info, err := os.Stat(stageExam); err == nil && info.IsDir() {
		_ = copyDir(stageExam, filepath.Join(ihdHome, "exam"))
	}
	// Sync references directory
	stageRef := filepath.Join(stageDir, "docs", "references")
	if info, err := os.Stat(stageRef); err == nil && info.IsDir() {
		_ = copyDir(stageRef, filepath.Join(ihdHome, "references"))
	}
	// Sync root docs
	for _, doc := range []string{"LICENSE.MIT", "NOTICE.md", "README.md"} {
		src := filepath.Join(stageDir, doc)
		if _, err := os.Stat(src); err == nil {
			_ = copyFile(src, filepath.Join(ihdHome, doc), 0644)
		}
	}
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target, info.Mode())
	})
}
