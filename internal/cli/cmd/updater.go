package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"QMLauncher/internal/cli/output"
	"QMLauncher/internal/version"
	"QMLauncher/pkg/updater"

	"github.com/alecthomas/kong"
	"github.com/fatih/color"
)

// UpdateCheckCmd checks for available updates
type UpdateCheckCmd struct{}

// UpdateDownloadCmd downloads and installs available updates
type UpdateDownloadCmd struct {
	Force bool `help:"Skip confirmation prompt" short:"f"`
}

// UpdateInfoCmd shows current version information
type UpdateInfoCmd struct{}

// UpdateCmd manages application updates
type UpdateCmd struct {
	Check    UpdateCheckCmd    `cmd:"" help:"${update_check}"`
	Download UpdateDownloadCmd `cmd:"" help:"${update_download}"`
	Info     UpdateInfoCmd     `cmd:"" help:"${update_info}"`
}

// updateCurrentVersionKey is the translation key for "Current version"
//
//nolint:unused
const updateCurrentVersionKey = "Текущая версия"

// updatePlatformKey is the translation key for "Platform"
//
//nolint:unused
const updatePlatformKey = "Платформа"

func (c *UpdateCheckCmd) Run(ctx *kong.Context) error {
	updater := createUpdater()

	output.Info("Checking for updates...")

	updateInfo, err := updater.CheckForUpdates()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if !updateInfo.Available {
		output.Success("You are running the latest version!")
		return nil
	}

	fmt.Printf("\n%s %s is available!\n", color.New(color.FgGreen, color.Bold).Sprint("Update available:"), updateInfo.LatestVer)
	fmt.Printf("%s %s\n", color.New(color.FgCyan).Sprint("Current version:"), updater.CurrentVer)
	fmt.Printf("%s %s\n", color.New(color.FgCyan).Sprint("Download size:"), formatFileSize(updateInfo.Size))
	fmt.Printf("%s %s\n", color.New(color.FgCyan).Sprint("Release URL:"), updateInfo.ReleaseURL)

	if updateInfo.Changelog != "" {
		fmt.Printf("\n%s\n", color.New(color.FgYellow, color.Bold).Sprint("Changelog:"))
		fmt.Println(updateInfo.Changelog)
	}

	fmt.Printf("\n%s Run '%s' to install the update.\n",
		color.New(color.FgGreen).Sprint("→"),
		color.New(color.Bold).Sprint("qm update download"))

	return nil
}

func (c *UpdateDownloadCmd) Run(ctx *kong.Context) error {
	updater := createUpdater()

	output.Info("Checking for updates...")

	updateInfo, err := updater.CheckForUpdates()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if !updateInfo.Available {
		output.Success("You are already running the latest version!")
		return nil
	}

	fmt.Printf("Update available: %s → %s\n",
		color.New(color.Bold).Sprint(updater.CurrentVer),
		color.New(color.FgGreen, color.Bold).Sprint(updateInfo.LatestVer))

	if !c.Force {
		fmt.Printf("Download size: %s\n", formatFileSize(updateInfo.Size))

		var confirm string
		fmt.Print("Do you want to download and install this update? [y/N]: ")
		fmt.Scanln(&confirm)

		if confirm != "y" && confirm != "Y" {
			output.Info("Update cancelled.")
			return nil
		}
	}

	fmt.Println()
	output.Info("Downloading update...")

	var lastProgress float64
	err = updater.DownloadUpdate(updateInfo, func(progress float64) {
		// Only print progress updates to avoid spam
		if progress-lastProgress >= 10 || progress >= 100 {
			fmt.Printf("\rDownload progress: %.1f%%", progress)
			lastProgress = progress
		}
	})

	if err != nil {
		fmt.Println() // New line after progress
		return fmt.Errorf("failed to download update: %w", err)
	}

	fmt.Println() // New line after progress
	output.Success("Update downloaded and installed successfully!")

	fmt.Printf("\n%s The application will restart in 3 seconds...\n",
		color.New(color.FgGreen).Sprint("✓"))

	// Wait a bit before restart
	time.Sleep(3 * time.Second)

	// Restart the application
	restartApplication()

	return nil
}

func (c *UpdateInfoCmd) Run(ctx *kong.Context) error {
	updater := createUpdater()

	info := updater.GetVersionInfo()

	fmt.Printf("%s: %s\n", output.Translate("update.current_version"), info["current"])
	fmt.Printf("%s: %s\n", output.Translate("update.platform"), info["platform"])

	// Check for updates in background
	go func() {
		updateInfo, err := updater.CheckForUpdates()
		if err != nil {
			return // Silently ignore errors in info command
		}

		if updateInfo.Available {
			fmt.Printf("\n%s %s is available! Run '%s' to update.\n",
				color.New(color.FgGreen).Sprint("Update available:"),
				updateInfo.LatestVer,
				color.New(color.Bold).Sprint("qm update check"))
		}
	}()

	return nil
}

// createUpdater creates a new updater instance with appropriate configuration
func createUpdater() *updater.Updater {
	// Get cache directory
	cacheDir := os.Getenv("QMLAUNCHER_CACHE_DIR")
	if cacheDir == "" {
		homeDir, _ := os.UserHomeDir()
		cacheDir = filepath.Join(homeDir, ".qmlauncher", "cache")
	}

	// Get current version from version package
	currentVer := version.Current

	return updater.New("mindevis", "QMLauncher", currentVer, cacheDir)
}

// restartApplication restarts the current application
func restartApplication() {
	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Warning: Could not get executable path: %v\n", err)
		os.Exit(0)
	}

	// On Windows, we need to use different approach
	if os.Getenv("OS") == "Windows_NT" {
		// For Windows, we'll just exit and let user restart manually
		fmt.Println("Please restart the application manually.")
		os.Exit(0)
	}

	// For Unix-like systems, we can try to restart
	// This is a simple approach - in production you might want more sophisticated restart logic
	fmt.Printf("Restarting %s...\n", execPath)
	os.Exit(0) // Exit with success, expecting external restart mechanism
}
