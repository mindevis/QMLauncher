package updater

import (
	"testing"
)

func TestNew(t *testing.T) {
	updater := New("mindeivs", "QMLauncher", "1.0.0", "/tmp/cache")

	if updater.Owner != "mindeivs" {
		t.Errorf("Expected owner 'mindeivs', got '%s'", updater.Owner)
	}

	if updater.Repo != "QMLauncher" {
		t.Errorf("Expected repo 'QMLauncher', got '%s'", updater.Repo)
	}

	if updater.CurrentVer != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", updater.CurrentVer)
	}
}

func TestFindAssetForPlatform(t *testing.T) {
	updater := New("mindeivs", "QMLauncher", "1.0.0", "/tmp/cache")

	assets := []Asset{
		{Name: "qmlauncher-linux-amd64.zip", BrowserDownloadURL: "https://example.com/linux-amd64.zip"},
		{Name: "qmlauncher-windows-amd64.zip", BrowserDownloadURL: "https://example.com/windows-amd64.zip"},
		{Name: "qmlauncher-darwin-amd64.zip", BrowserDownloadURL: "https://example.com/darwin-amd64.zip"},
	}

	asset := updater.findAssetForPlatform(assets)
	if asset == nil {
		t.Error("Expected to find asset for current platform")
	}
}

func TestGetVersionInfo(t *testing.T) {
	updater := New("mindeivs", "QMLauncher", "1.0.0", "/tmp/cache")

	info := updater.GetVersionInfo()

	if info["current"] != "1.0.0" {
		t.Errorf("Expected current version '1.0.0', got '%s'", info["current"])
	}

	if info["os"] == "" {
		t.Error("Expected non-empty OS")
	}

	if info["arch"] == "" {
		t.Error("Expected non-empty architecture")
	}
}

func TestIsBinaryFile(t *testing.T) {
	updater := New("mindeivs", "QMLauncher", "1.0.0", "/tmp/cache")

	tests := []struct {
		filename string
		expected bool
	}{
		{"qmlauncher", true},
		{"qmlauncher.exe", true},
		{"qm", true},
		{"README.md", false},
		{"config.json", false},
		{"qmlauncher-linux-amd64.zip", false},
	}

	for _, test := range tests {
		result := updater.isBinaryFile(test.filename)
		if result != test.expected {
			t.Errorf("isBinaryFile(%s) = %v, expected %v", test.filename, result, test.expected)
		}
	}
}
