package updater

import (
	"crypto/md5"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"QMLauncher/internal/network"
)

const (
	defaultQMWebBase    = "https://web.qx-dev.ru"
	launcherExeName     = "QMLauncher-windows-amd64.exe"
	launcherMD5Filename = "QMLauncher-windows-amd64.exe.md5"
)

// getQMWebBase returns QMWeb base URL for launcher downloads
func getQMWebBase() string {
	if base := os.Getenv("QMWEB_URL"); base != "" {
		return strings.TrimSuffix(base, "/")
	}
	return defaultQMWebBase
}

// ComputeFileMD5 returns MD5 hash of a file (hex string)
func ComputeFileMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// qmWebHTTPClient returns a client tuned for Cloudflare / high-latency TLS (longer handshakes than default).
func qmWebHTTPClient(totalTimeout time.Duration) *http.Client {
	base := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   45 * time.Second,
		ResponseHeaderTimeout: 45 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   4,
	}
	return &http.Client{
		Timeout:   totalTimeout,
		Transport: network.WrapRoundTripperWithDebug(base),
	}
}

func fetchRemoteMD5Once(rawURL string) (string, error) {
	client := qmWebHTTPClient(90 * time.Second)
	resp, err := client.Get(rawURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// FetchRemoteLauncherMD5 fetches MD5 from QMWeb uploads (retries on transient TLS / network errors).
func FetchRemoteLauncherMD5() (string, error) {
	base := getQMWebBase()
	rawURL := base + "/uploads/" + launcherMD5Filename
	const maxAttempts = 4
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			// Backoff: helps when Cloudflare or route is briefly slow.
			time.Sleep(time.Duration(attempt) * 1500 * time.Millisecond)
		}
		s, err := fetchRemoteMD5Once(rawURL)
		if err == nil {
			return s, nil
		}
		lastErr = err
	}
	return "", lastErr
}

// CheckForQMWebUpdate checks if an update is available (Windows only).
// Returns true if update is available, false otherwise.
func CheckForQMWebUpdate(logFn func(string)) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	exePath, err := os.Executable()
	if err != nil {
		return false
	}
	if !strings.HasSuffix(strings.ToLower(exePath), strings.ToLower(launcherExeName)) {
		return false
	}
	localMD5, err := ComputeFileMD5(exePath)
	if err != nil {
		return false
	}
	remoteMD5, err := FetchRemoteLauncherMD5()
	if err != nil {
		if logFn != nil {
			logFn(fmt.Sprintf("[AutoUpdate] Remote version check skipped: %v", err))
		}
		return false
	}
	return !strings.EqualFold(localMD5, remoteMD5)
}

// ApplyAndRestartQMWebUpdate downloads the update, applies it, and exits (Windows only).
// Does not return on success — process exits. Returns error only if download/apply fails.
func ApplyAndRestartQMWebUpdate(logFn func(string)) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("updates only supported on Windows")
	}
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	if !strings.HasSuffix(strings.ToLower(exePath), strings.ToLower(launcherExeName)) {
		return fmt.Errorf("exe name must be %s", launcherExeName)
	}
	localMD5, err := ComputeFileMD5(exePath)
	if err != nil {
		return err
	}
	remoteMD5, err := FetchRemoteLauncherMD5()
	if err != nil {
		return err
	}
	if strings.EqualFold(localMD5, remoteMD5) {
		return nil // already up to date
	}
	if logFn != nil {
		logFn(fmt.Sprintf("[AutoUpdate] Update available: local=%s remote=%s", localMD5, remoteMD5))
	}
	downloadURL := getQMWebBase() + "/uploads/" + launcherExeName
	tempDir := filepath.Join(os.TempDir(), "qmlauncher-update")
	os.MkdirAll(tempDir, 0755)
	tempExe := filepath.Join(tempDir, launcherExeName)
	if err := downloadFile(downloadURL, tempExe); err != nil {
		if logFn != nil {
			logFn(fmt.Sprintf("[AutoUpdate] Download failed: %v", err))
		}
		return err
	}
	downloadedMD5, err := ComputeFileMD5(tempExe)
	if err != nil || !strings.EqualFold(downloadedMD5, remoteMD5) {
		os.Remove(tempExe)
		return fmt.Errorf("downloaded file MD5 mismatch")
	}
	if err := runWindowsUpdater(exePath, tempExe, logFn); err != nil {
		return err
	}
	os.Exit(0)
	return nil // unreachable
}

// CheckAndApplyQMWebUpdate checks MD5 and applies update if needed (Windows only) at startup.
// If update is applied, the process exits and does not return.
func CheckAndApplyQMWebUpdate(logFn func(string)) bool {
	if !CheckForQMWebUpdate(logFn) {
		return false
	}
	if err := ApplyAndRestartQMWebUpdate(logFn); err != nil {
		if logFn != nil {
			logFn(fmt.Sprintf("[AutoUpdate] Failed: %v", err))
		}
		return false
	}
	return true
}

func downloadFile(url, dest string) error {
	client := qmWebHTTPClient(8 * time.Minute)
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func runWindowsUpdater(currentExe, newExe string, logFn func(string)) error {
	bat := fmt.Sprintf(`@echo off
ping -n 4 127.0.0.1 >nul
copy /Y "%s" "%s"
if errorlevel 1 exit /b 1
start "" "%s"
del "%%~f0"
`, newExe, currentExe, currentExe)
	batPath := filepath.Join(filepath.Dir(newExe), "qmlauncher-updater.bat")
	if err := os.WriteFile(batPath, []byte(bat), 0755); err != nil {
		return err
	}
	cmd := exec.Command("cmd", "/C", "start", "/B", "", batPath)
	setCmdNoWindow(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}
