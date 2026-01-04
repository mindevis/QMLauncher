package updater

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"QMLauncher/internal/network"
)

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
	Prerelease  bool      `json:"prerelease"`
}

// Asset represents a release asset
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// Updater handles application updates
type Updater struct {
	Owner       string
	Repo        string
	CurrentVer  string
	CacheDir    string
	APIEndpoint string
}

// UpdateInfo contains information about available updates
type UpdateInfo struct {
	Available   bool
	LatestVer   string
	ReleaseURL  string
	Changelog   string
	DownloadURL string
	Size        int64
}

// New creates a new updater instance
func New(owner, repo, currentVer, cacheDir string) *Updater {
	return &Updater{
		Owner:       owner,
		Repo:        repo,
		CurrentVer:  currentVer,
		CacheDir:    cacheDir,
		APIEndpoint: "https://api.github.com",
	}
}

// CheckForUpdates checks if there's a newer version available
func (u *Updater) CheckForUpdates() (*UpdateInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", u.APIEndpoint, u.Owner, u.Repo)

	cache := network.Cache[GitHubRelease]{
		Path:        filepath.Join(u.CacheDir, "updater", "latest_release.json"),
		URL:         url,
		AlwaysFetch: false,
	}

	var release GitHubRelease
	if err := cache.Get(&release); err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}

	// Skip prereleases unless current version is also a prerelease
	if release.Prerelease && !strings.Contains(u.CurrentVer, "-") {
		return &UpdateInfo{Available: false}, nil
	}

	latestVer := strings.TrimPrefix(release.TagName, "v")
	currentVer := strings.TrimPrefix(u.CurrentVer, "v")

	if latestVer == currentVer {
		return &UpdateInfo{Available: false}, nil
	}

	// Find appropriate asset for current platform
	asset := u.findAssetForPlatform(release.Assets)
	if asset == nil {
		return nil, fmt.Errorf("no suitable download found for platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	return &UpdateInfo{
		Available:   true,
		LatestVer:   latestVer,
		ReleaseURL:  fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", u.Owner, u.Repo, release.TagName),
		Changelog:   release.Body,
		DownloadURL: asset.BrowserDownloadURL,
		Size:        asset.Size,
	}, nil
}

// findAssetForPlatform finds the appropriate asset for current platform
func (u *Updater) findAssetForPlatform(assets []Asset) *Asset {
	os := runtime.GOOS
	arch := runtime.GOARCH

	// Normalize OS names for matching
	osNormalized := os
	switch os {
	case "windows":
		osNormalized = "windows"
	case "darwin":
		osNormalized = "darwin"
	case "linux":
		osNormalized = "linux"
	}

	// Normalize architecture names
	archNormalized := arch
	switch arch {
	case "amd64":
		archNormalized = "amd64"
	case "386":
		archNormalized = "386"
	}

	for _, asset := range assets {
		name := strings.ToLower(asset.Name)

		// For our release naming convention:
		// - Linux: QMLauncher-cli-linux-amd64 (no extension)
		// - macOS: QMLauncher-cli-macos-amd64 (no extension)
		// - Windows: QMLauncher-cli-windows-amd64.exe

		// Check platform by release naming convention
		switch os {
		case "windows":
			if strings.Contains(name, "windows") && strings.Contains(name, "amd64") && strings.HasSuffix(name, ".exe") {
				return &asset
			}
		case "darwin":
			if strings.Contains(name, "macos") && strings.Contains(name, "amd64") && !strings.Contains(name, ".exe") {
				return &asset
			}
		case "linux":
			if strings.Contains(name, "linux") && strings.Contains(name, "amd64") && !strings.Contains(name, ".exe") {
				return &asset
			}
		}

		// Fallback: check for platform and arch in filename
		if strings.Contains(name, osNormalized) && strings.Contains(name, archNormalized) {
			return &asset
		}
	}

	return nil
}

// DownloadUpdate downloads and installs the update
func (u *Updater) DownloadUpdate(updateInfo *UpdateInfo, progressCallback func(float64)) error {
	if updateInfo == nil || !updateInfo.Available {
		return fmt.Errorf("no update available")
	}

	// Create temp directory for download
	tempDir := filepath.Join(u.CacheDir, "updater", "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	tempFile := filepath.Join(tempDir, "update.zip")

	// Download the update
	if err := u.downloadFile(updateInfo.DownloadURL, tempFile, progressCallback); err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}

	// Extract the update
	if err := u.extractUpdate(tempFile, tempDir); err != nil {
		return fmt.Errorf("failed to extract update: %w", err)
	}

	// Find the new binary
	newBinary, err := u.findNewBinary(tempDir)
	if err != nil {
		return fmt.Errorf("failed to find new binary: %w", err)
	}

	// Replace current binary
	if err := u.replaceBinary(newBinary); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Cleanup
	os.RemoveAll(tempDir)

	return nil
}

// downloadFile downloads a file with progress callback
func (u *Updater) downloadFile(url, destPath string, progressCallback func(float64)) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	counter := &ProgressReader{
		Reader:   resp.Body,
		Total:    resp.ContentLength,
		Callback: progressCallback,
	}

	_, err = io.Copy(out, counter)
	return err
}

// ProgressReader tracks download progress
type ProgressReader struct {
	Reader   io.Reader
	Total    int64
	Current  int64
	Callback func(float64)
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Current += int64(n)

	if pr.Callback != nil && pr.Total > 0 {
		progress := float64(pr.Current) / float64(pr.Total) * 100
		pr.Callback(progress)
	}

	return n, err
}

// extractUpdate extracts the ZIP archive
func (u *Updater) extractUpdate(zipPath, destDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		path := filepath.Join(destDir, file.Name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		outFile, err := os.Create(path)
		if err != nil {
			return err
		}

		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}

		// Set executable permissions on binary files
		if u.isBinaryFile(file.Name) {
			os.Chmod(path, 0755)
		}
	}

	return nil
}

// isBinaryFile checks if a file is likely a binary executable
func (u *Updater) isBinaryFile(filename string) bool {
	name := strings.ToLower(filename)

	// Skip archive files
	if strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tar.bz2") {
		return false
	}

	// Check for common executable extensions
	if runtime.GOOS == "windows" {
		return strings.HasSuffix(name, ".exe")
	}

	// On Unix-like systems, check for files without extensions or with executable names
	if !strings.Contains(name, ".") {
		return true
	}

	// Common executable names (but not in archive names)
	execNames := []string{"qmlauncher", "qm"}
	for _, execName := range execNames {
		if strings.Contains(name, execName) && !strings.Contains(name, ".zip") && !strings.Contains(name, ".tar") {
			return true
		}
	}

	return false
}

// findNewBinary finds the new binary in the extracted files
func (u *Updater) findNewBinary(extractDir string) (string, error) {
	var candidates []string

	err := filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && u.isBinaryFile(info.Name()) {
			candidates = append(candidates, path)
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no binary found in update package")
	}

	// Return the first (and likely only) binary
	return candidates[0], nil
}

// replaceBinary replaces the current binary with the new one
func (u *Updater) replaceBinary(newBinary string) error {
	// Get current executable path
	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	// Create backup of current binary
	backupPath := currentBinary + ".backup"
	if err := copyFile(currentBinary, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Replace the binary
	if err := copyFile(newBinary, currentBinary); err != nil {
		// Restore backup on failure
		copyFile(backupPath, currentBinary)
		os.Remove(backupPath)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Set executable permissions
	if err := os.Chmod(currentBinary, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

// GetVersionInfo returns current version information
func (u *Updater) GetVersionInfo() map[string]string {
	return map[string]string{
		"current":  u.CurrentVer,
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
		"platform": fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
