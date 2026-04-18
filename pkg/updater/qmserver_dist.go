package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"QMLauncher/internal/network"
	"QMLauncher/internal/version"

	"golang.org/x/mod/semver"
)

// qmserverDistJSON matches GET /api/v1/launcher/distribution on QMServer.
type qmserverDistJSON struct {
	Version     string `json:"version"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
	Windows     struct {
		Filename    string `json:"filename"`
		DownloadURL string `json:"download_url"`
		MD5URL      string `json:"md5_url"`
		Size        int64  `json:"size"`
	} `json:"windows"`
	Linux struct {
		Filename    string `json:"filename"`
		DownloadURL string `json:"download_url"`
		MD5URL      string `json:"md5_url"`
		Size        int64  `json:"size"`
	} `json:"linux"`
}

func canonicalSemverStr(s string) string {
	s = strings.TrimSpace(strings.TrimPrefix(s, "v"))
	if s == "" {
		return "v0.0.0"
	}
	v := s
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if semver.IsValid(v) {
		return semver.Canonical(v)
	}
	return "v0.0.0"
}

func fetchQMServerDistribution() (*qmserverDistJSON, error) {
	base := strings.TrimSuffix(network.EffectiveQMServerAPIBase(), "/")
	u := base + "/api/v1/launcher/distribution"
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", network.QMServerUserAgent)
	resp, err := network.QMServerHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("distribution %s: %s", resp.Status, strings.TrimSpace(string(body[:min(len(body), 512)])))
	}
	var dist qmserverDistJSON
	if err := json.Unmarshal(body, &dist); err != nil {
		return nil, err
	}
	return &dist, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// qmserverSkipUpdateSameMD5 is true when the running binary already matches QMServer's published MD5 (guards stale embedded semver).
func qmserverSkipUpdateSameMD5(exePath, remoteMD5 string, logFn func(string)) bool {
	localMD5, err := ComputeFileMD5(exePath)
	if err != nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(localMD5), strings.TrimSpace(remoteMD5)) {
		if logFn != nil {
			logFn("[AutoUpdate] QMServer distribution: skip (binary MD5 matches; version metadata may lag)")
		}
		return true
	}
	return false
}

func fetchTextURL(u string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", network.QMServerUserAgent)
	client := qmWebHTTPClient(90 * time.Second)
	resp, err := client.Do(req)
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

func downloadFileToPath(rawURL, dest string) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", network.QMServerUserAgent)
	client := qmWebHTTPClient(8 * time.Minute)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download status %d", resp.StatusCode)
	}
	tmp := dest + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, resp.Body)
	cerr := out.Close()
	if err != nil {
		os.Remove(tmp)
		return err
	}
	if cerr != nil {
		os.Remove(tmp)
		return cerr
	}
	return os.Rename(tmp, dest)
}

// QMServerDistributionUpdateAvailable returns true if QMServer hosts a newer semver than the running binary.
func QMServerDistributionUpdateAvailable(logFn func(string)) bool {
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		return false
	}
	dist, err := fetchQMServerDistribution()
	if err != nil {
		if logFn != nil {
			logFn(fmt.Sprintf("[AutoUpdate] QMServer distribution unavailable: %v", err))
		}
		return false
	}
	remote := canonicalSemverStr(dist.Version)
	local := canonicalSemverStr(version.Current)
	if semver.Compare(remote, local) <= 0 {
		return false
	}
	var md5URL string
	switch runtime.GOOS {
	case "windows":
		md5URL = strings.TrimSpace(dist.Windows.MD5URL)
	case "linux":
		md5URL = strings.TrimSpace(dist.Linux.MD5URL)
	default:
		return false
	}
	if md5URL == "" {
		return true
	}
	exePath, err := os.Executable()
	if err != nil {
		return true
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return true
	}
	remoteMD5, err := fetchTextURL(md5URL)
	if err != nil || strings.TrimSpace(remoteMD5) == "" {
		return true
	}
	if qmserverSkipUpdateSameMD5(exePath, remoteMD5, logFn) {
		return false
	}
	return true
}

// CheckAndApplyQMServerDistributionUpdate downloads from QMServer and restarts when newer. Returns true if process exits for update.
func CheckAndApplyQMServerDistributionUpdate(logFn func(string)) bool {
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		return false
	}
	dist, err := fetchQMServerDistribution()
	if err != nil {
		if logFn != nil {
			logFn(fmt.Sprintf("[AutoUpdate] QMServer distribution unavailable: %v", err))
		}
		return false
	}
	remote := canonicalSemverStr(dist.Version)
	local := canonicalSemverStr(version.Current)
	if semver.Compare(remote, local) <= 0 {
		return false
	}

	var dlURL, md5URL, expectName string
	switch runtime.GOOS {
	case "windows":
		dlURL = strings.TrimSpace(dist.Windows.DownloadURL)
		md5URL = strings.TrimSpace(dist.Windows.MD5URL)
		expectName = launcherExeName
	case "linux":
		dlURL = strings.TrimSpace(dist.Linux.DownloadURL)
		md5URL = strings.TrimSpace(dist.Linux.MD5URL)
		expectName = "QMLauncher-linux-amd64"
	default:
		return false
	}
	if dlURL == "" || md5URL == "" {
		return false
	}

	exePath, err := os.Executable()
	if err != nil {
		return false
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return false
	}
	baseExe := strings.ToLower(filepath.Base(exePath))
	if runtime.GOOS == "windows" && baseExe != strings.ToLower(expectName) {
		if logFn != nil {
			logFn(fmt.Sprintf("[AutoUpdate] Skip QMServer update: exe must be %s", expectName))
		}
		return false
	}
	if runtime.GOOS == "linux" && baseExe != strings.ToLower(expectName) {
		if logFn != nil {
			logFn(fmt.Sprintf("[AutoUpdate] Skip QMServer update: binary must be %s", expectName))
		}
		return false
	}

	remoteMD5, err := fetchTextURL(md5URL)
	if err != nil {
		if logFn != nil {
			logFn(fmt.Sprintf("[AutoUpdate] QMServer MD5 fetch failed: %v", err))
		}
		return false
	}
	if qmserverSkipUpdateSameMD5(exePath, remoteMD5, logFn) {
		return false
	}

	tempDir := filepath.Join(os.TempDir(), "qmlauncher-qmserver-update")
	_ = os.MkdirAll(tempDir, 0755)
	tempBin := filepath.Join(tempDir, expectName)
	if err := downloadFileToPath(dlURL, tempBin); err != nil {
		if logFn != nil {
			logFn(fmt.Sprintf("[AutoUpdate] QMServer download failed: %v", err))
		}
		return false
	}
	if runtime.GOOS == "linux" {
		_ = os.Chmod(tempBin, 0755)
	}
	sum, err := ComputeFileMD5(tempBin)
	if err != nil {
		os.Remove(tempBin)
		return false
	}
	if !strings.EqualFold(sum, remoteMD5) {
		os.Remove(tempBin)
		if logFn != nil {
			logFn("[AutoUpdate] QMServer package MD5 mismatch")
		}
		return false
	}

	if logFn != nil {
		logFn(fmt.Sprintf("[AutoUpdate] Applying QMServer release %s (was %s)", dist.Version, version.Current))
	}

	switch runtime.GOOS {
	case "windows":
		if err := runWindowsUpdater(exePath, tempBin, logFn); err != nil {
			if logFn != nil {
				logFn(fmt.Sprintf("[AutoUpdate] QMServer apply failed: %v", err))
			}
			return false
		}
		os.Exit(0)
	case "linux":
		if err := linuxReplaceExecutableAndRelaunch(exePath, tempBin); err != nil {
			if logFn != nil {
				logFn(fmt.Sprintf("[AutoUpdate] QMServer apply failed: %v", err))
			}
			return false
		}
	}
	return false
}

func linuxReplaceExecutableAndRelaunch(currentExe, newExe string) error {
	backupPath := currentExe + ".backup"
	if err := copyFileLinux(currentExe, backupPath); err != nil {
		return fmt.Errorf("backup: %w", err)
	}
	if err := copyFileLinux(newExe, currentExe); err != nil {
		_ = copyFileLinux(backupPath, currentExe)
		return fmt.Errorf("replace: %w", err)
	}
	_ = os.Chmod(currentExe, 0755)
	cmd := exec.Command(currentExe, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}

func copyFileLinux(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
