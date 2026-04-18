// Package env provides directories used by the launcher for various data.
package env

import (
	"fmt"
	"os"
	"path/filepath"
)

var RootDir string // Base launcher directory. Defaults to "$HOME/.qmlauncher"

var LibrariesDir string // Java libraries directory

var InstancesDir string // Instances directory

var CachesDir string // Caches directory, e.g. version metadata, version manifest.

var AssetsDir string // Game assets directory and asset index

var TmpDir string // Directory for temporary files

var JavaDir string // Mojang Java installations

// CredentialsVaultPath is the encrypted file storing Microsoft session, offline and QMServer Cloud accounts.
var CredentialsVaultPath string

// AuthStorePath is an alias for CredentialsVaultPath (single vault file replaces legacy account.json).
var AuthStorePath string

// SetDirs sets all directories to defaults from rootDir. These values can also be changed individually.
// However, they should not be changed between operations, as the launcher will not be able to find necessary files.
func SetDirs(rootDir string) error {
	RootDir = rootDir
	InstancesDir = filepath.Join(RootDir, "instances")
	LibrariesDir = filepath.Join(RootDir, "libraries")
	CachesDir = filepath.Join(RootDir, "caches")
	AssetsDir = filepath.Join(RootDir, "assets")
	TmpDir = filepath.Join(RootDir, "tmp")
	JavaDir = filepath.Join(RootDir, "java")
	CredentialsVaultPath = filepath.Join(RootDir, "credentials.vault")
	AuthStorePath = CredentialsVaultPath

	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return fmt.Errorf("create root directory: %w", err)
	}
	return nil
}

func init() {
	home, _ := os.UserHomeDir()
	SetDirs(filepath.Join(home, ".qmlauncher"))
}
