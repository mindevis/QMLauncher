package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizePathsInTextFiles(t *testing.T) {
	// Создаем временную директорию для теста
	tempDir, err := os.MkdirTemp("", "qmlauncher_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Создаем тестовые файлы с путями Windows
	testFiles := map[string]string{
		"options.txt":      "resourcePacks:[\"file:///C:\\\\Users\\\\User\\\\AppData\\\\Roaming\\\\.minecraft\\\\resourcepacks\\\\pack.zip\"]",
		"config/test.cfg":  "modPath=C:\\\\mods\\\\mod.jar\ntexturePath=C:\\\\textures\\\\texture.png",
		"config/test.toml": "path = \"C:\\\\config\\\\file.cfg\"\nother = \"normal/path\"",
	}

	// Создаем файлы с тестовыми данными
	for filePath, content := range testFiles {
		fullPath := filepath.Join(tempDir, filePath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create dir for %s: %v", filePath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", filePath, err)
		}
	}

	// Вызываем функцию нормализации
	if err := normalizePathsInTextFiles(tempDir); err != nil {
		t.Fatalf("normalizePathsInTextFiles failed: %v", err)
	}

	// Проверяем результаты
	expectedResults := map[string]string{
		"options.txt":      "resourcePacks:[\"file:///C://Users//User//AppData//Roaming//.minecraft//resourcepacks//pack.zip\"]",
		"config/test.cfg":  "modPath=C://mods//mod.jar\ntexturePath=C://textures//texture.png",
		"config/test.toml": "path = \"C://config//file.cfg\"\nother = \"normal/path\"",
	}

	for filePath, expectedContent := range expectedResults {
		fullPath := filepath.Join(tempDir, filePath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("Failed to read normalized file %s: %v", filePath, err)
		}

		actualContent := string(content)
		if actualContent != expectedContent {
			t.Errorf("File %s normalization failed.\nExpected:\n%s\nActual:\n%s", filePath, expectedContent, actualContent)
		}

		// Проверяем, что все обратные слэши были заменены на прямые
		if strings.Contains(actualContent, "\\") {
			t.Errorf("File %s still contains backslashes: %s", filePath, actualContent)
		}
	}
}

func TestNormalizeZipFilePaths(t *testing.T) {
	// Тестируем нормализацию путей файлов в ZIP архиве
	testCases := []struct {
		input    string
		expected string
	}{
		{"config\\jei\\jei-client.ini", "config/jei/jei-client.ini"},
		{"mods\\mod.jar", "mods/mod.jar"},
		{"resourcepacks\\pack.zip", "resourcepacks/pack.zip"},
		{"options.txt", "options.txt"}, // Уже нормальный путь
		{"config/subdir\\file.cfg", "config/subdir/file.cfg"},
	}

	for _, tc := range testCases {
		result := strings.ReplaceAll(tc.input, "\\", "/")
		if result != tc.expected {
			t.Errorf("Path normalization failed for %q: expected %q, got %q", tc.input, tc.expected, result)
		}
	}
}
