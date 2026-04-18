package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"QMLauncher/internal/version"
	env "QMLauncher/pkg"
)

// GitHubBinaryUpdateAvailable reports whether GitHub releases/latest is newer than version.Current.
func GitHubBinaryUpdateAvailable() bool {
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		return false
	}
	up := New("mindevis", "QMLauncher", version.Current, env.CachesDir)
	info, err := up.CheckForUpdates()
	return err == nil && info != nil && info.Available
}

// CheckAndApplyGitHubBinaryUpdate uses GitHub releases/latest (raw exe / linux binary, not zip). Returns true if process exits.
func CheckAndApplyGitHubBinaryUpdate(logFn func(string)) bool {
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		return false
	}
	up := New("mindevis", "QMLauncher", version.Current, env.CachesDir)
	info, err := up.CheckForUpdates()
	if err != nil || info == nil || !info.Available {
		if err != nil && logFn != nil {
			logFn(fmt.Sprintf("[AutoUpdate] GitHub check: %v", err))
		}
		return false
	}
	dlURL := strings.TrimSpace(info.DownloadURL)
	if dlURL == "" {
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
	var expectName string
	switch runtime.GOOS {
	case "windows":
		expectName = launcherExeName
	case "linux":
		expectName = "QMLauncher-linux-amd64"
	default:
		return false
	}
	if !strings.EqualFold(filepath.Base(exePath), expectName) {
		return false
	}

	tempDir := filepath.Join(os.TempDir(), "qmlauncher-github-update")
	_ = os.MkdirAll(tempDir, 0755)
	tempBin := filepath.Join(tempDir, expectName)
	if err := downloadFileToPath(dlURL, tempBin); err != nil {
		if logFn != nil {
			logFn(fmt.Sprintf("[AutoUpdate] GitHub download failed: %v", err))
		}
		return false
	}
	if runtime.GOOS == "linux" {
		_ = os.Chmod(tempBin, 0755)
	}

	curSum, e1 := ComputeFileMD5(exePath)
	newSum, e2 := ComputeFileMD5(tempBin)
	if e1 == nil && e2 == nil && strings.EqualFold(curSum, newSum) {
		_ = os.Remove(tempBin)
		if logFn != nil {
			logFn("[AutoUpdate] Skip GitHub update: running binary matches release (embedded version may lag)")
		}
		return false
	}

	if logFn != nil {
		logFn(fmt.Sprintf("[AutoUpdate] Applying GitHub release %s", info.LatestVer))
	}

	switch runtime.GOOS {
	case "windows":
		if err := runWindowsUpdater(exePath, tempBin, logFn); err != nil {
			if logFn != nil {
				logFn(fmt.Sprintf("[AutoUpdate] GitHub apply failed: %v", err))
			}
			return false
		}
		os.Exit(0)
	case "linux":
		if err := linuxReplaceExecutableAndRelaunch(exePath, tempBin); err != nil {
			if logFn != nil {
				logFn(fmt.Sprintf("[AutoUpdate] GitHub apply failed: %v", err))
			}
			return false
		}
	}
	return false
}
