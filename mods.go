package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

type ModsService struct {
	app *App
}

func NewModsService(app *App) *ModsService {
	return &ModsService{app: app}
}

// GetServerMods получает список модов с сервера
func (m *ModsService) GetServerMods(serverId int, apiBaseUrl string) (*ServerModsResult, error) {
	url := fmt.Sprintf("%s/servers/%d/mods", apiBaseUrl, serverId)
	
	resp, err := http.Get(url)
	if err != nil {
		return &ServerModsResult{
			Success: false,
			Error:   fmt.Sprintf("Ошибка получения модов: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &ServerModsResult{
			Success: false,
			Error:   fmt.Sprintf("Ошибка получения модов: %d", resp.StatusCode),
		}, nil
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return &ServerModsResult{
			Success: false,
			Error:   fmt.Sprintf("Ошибка парсинга ответа: %v", err),
		}, nil
	}

	mods, ok := data["mods"].([]interface{})
	if !ok {
		mods = []interface{}{}
	}

	return &ServerModsResult{
		Success: true,
		Mods:    mods,
	}, nil
}

// DownloadMod скачивает мод
func (m *ModsService) DownloadMod(downloadUrl string, savePath string) (*DownloadResult, error) {
	// Создаем директорию если не существует
	dir := filepath.Dir(savePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &DownloadResult{
			Success: false,
			Error:   fmt.Sprintf("Ошибка создания директории: %v", err),
		}, nil
	}

	resp, err := http.Get(downloadUrl)
	if err != nil {
		return &DownloadResult{
			Success: false,
			Error:   fmt.Sprintf("Ошибка скачивания: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &DownloadResult{
			Success: false,
			Error:   fmt.Sprintf("Ошибка скачивания: %d", resp.StatusCode),
		}, nil
	}

	file, err := os.Create(savePath)
	if err != nil {
		return &DownloadResult{
			Success: false,
			Error:   fmt.Sprintf("Ошибка создания файла: %v", err),
		}, nil
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(savePath)
		return &DownloadResult{
			Success: false,
			Error:   fmt.Sprintf("Ошибка записи файла: %v", err),
		}, nil
	}

	return &DownloadResult{Success: true}, nil
}

// CheckAndUpdateMods проверяет и обновляет моды
func (m *ModsService) CheckAndUpdateMods(serverId int, apiBaseUrl string) (*ModsUpdateResult, error) {
	settings, err := m.app.GetSettings()
	if err != nil {
		return &ModsUpdateResult{
			Success: false,
			Error:   fmt.Sprintf("Ошибка получения настроек: %v", err),
		}, nil
	}

	// Получаем server_uuid
	serverUuid := fmt.Sprintf("%d", serverId)
	
	// Определяем путь к Minecraft
	minecraftBasePath := m.getMinecraftBasePath(settings, serverUuid)
	modsDir := filepath.Join(minecraftBasePath, "mods")

	// Получаем моды с сервера
	serverModsResult, err := m.GetServerMods(serverId, apiBaseUrl)
	if err != nil {
		return &ModsUpdateResult{
			Success: false,
			Error:   fmt.Sprintf("Ошибка получения модов: %v", err),
		}, nil
	}

	if !serverModsResult.Success {
		return &ModsUpdateResult{
			Success: false,
			Error:   serverModsResult.Error,
		}, nil
	}

	// Если нет модов, возвращаем успех
	if len(serverModsResult.Mods) == 0 {
		return &ModsUpdateResult{
			Success:     true,
			Updated:     false,
			ModsUpdated: 0,
			ModsDir:     modsDir,
		}, nil
	}

	// Создаем директорию модов если не существует
	if err := os.MkdirAll(modsDir, 0755); err != nil {
		return &ModsUpdateResult{
			Success: false,
			Error:   fmt.Sprintf("Ошибка создания директории модов: %v", err),
		}, nil
	}

	needsUpdate := false
	modsToDownload := []ModToDownload{}

	// Проверяем каждый мод
	for _, modInterface := range serverModsResult.Mods {
		mod, ok := modInterface.(map[string]interface{})
		if !ok {
			continue
		}

		filename, _ := mod["filename"].(string)
		size, _ := mod["size"].(float64)
		downloadUrl, _ := mod["download_url"].(string)

		if filename == "" {
			continue
		}

		localModPath := filepath.Join(modsDir, filename)
		localModInfo, err := os.Stat(localModPath)

		// Проверяем, нужно ли обновление
		if err != nil || localModInfo.Size() != int64(size) {
			needsUpdate = true
			if downloadUrl == "" {
				downloadUrl = fmt.Sprintf("%s/servers/%d/mods/%s/download", apiBaseUrl, serverId, filename)
			}
			modsToDownload = append(modsToDownload, ModToDownload{
				Filename:    filename,
				DownloadURL: downloadUrl,
			})
		}
	}

	// Удаляем моды, которых больше нет на сервере
	if err := m.removeOldMods(modsDir, serverModsResult.Mods); err != nil {
		// Не критично, продолжаем
	}

	// Скачиваем моды
	if needsUpdate && len(modsToDownload) > 0 {
		for _, mod := range modsToDownload {
			savePath := filepath.Join(modsDir, mod.Filename)
			downloadResult, err := m.DownloadMod(mod.DownloadURL, savePath)
			if err != nil || !downloadResult.Success {
				return &ModsUpdateResult{
					Success: false,
					Error:   fmt.Sprintf("Ошибка скачивания %s: %s", mod.Filename, downloadResult.Error),
				}, nil
			}
		}
	}

	return &ModsUpdateResult{
		Success:     true,
		Updated:     needsUpdate,
		ModsUpdated: len(modsToDownload),
		ModsDir:     modsDir,
	}, nil
}

func (m *ModsService) removeOldMods(modsDir string, serverMods []interface{}) error {
	// Получаем список файлов модов на сервере
	serverFilenames := make(map[string]bool)
	for _, modInterface := range serverMods {
		mod, ok := modInterface.(map[string]interface{})
		if !ok {
			continue
		}
		filename, _ := mod["filename"].(string)
		if filename != "" {
			serverFilenames[filename] = true
		}
	}

	// Удаляем локальные моды, которых нет на сервере
	if _, err := os.Stat(modsDir); os.IsNotExist(err) {
		return nil
	}

	files, err := os.ReadDir(modsDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if filepath.Ext(file.Name()) != ".jar" {
			continue
		}
		if !serverFilenames[file.Name()] {
			os.Remove(filepath.Join(modsDir, file.Name()))
		}
	}

	return nil
}

func (m *ModsService) getMinecraftBasePath(settings *Settings, serverUuid string) string {
	if settings.MinecraftPath != "" {
		minecraftPath := settings.MinecraftPath
		if runtime.GOOS == "windows" {
			home := os.Getenv("USERPROFILE")
			minecraftPath = filepath.Join(home, ".qmlauncher", serverUuid, "minecraft")
		} else {
			home := os.Getenv("HOME")
			minecraftPath = filepath.Join(home, ".qmlauncher", serverUuid, "minecraft")
		}
		return minecraftPath
	}

	home := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		home = os.Getenv("USERPROFILE")
	}
	return filepath.Join(home, ".qmlauncher", serverUuid, "minecraft")
}

// Типы данных

type ServerModsResult struct {
	Success bool          `json:"success"`
	Mods    []interface{} `json:"mods"`
	Error   string        `json:"error,omitempty"`
}

type DownloadResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type ModsUpdateResult struct {
	Success     bool   `json:"success"`
	Updated     bool   `json:"updated"`
	ModsUpdated int    `json:"modsUpdated"`
	ModsDir     string `json:"modsDir"`
	Error       string `json:"error,omitempty"`
}

type ModToDownload struct {
	Filename    string
	DownloadURL string
}

