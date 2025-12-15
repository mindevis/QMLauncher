package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type ConfigService struct {
	app *App
}

func NewConfigService(app *App) *ConfigService {
	return &ConfigService{app: app}
}

// GetSettings возвращает настройки
func (c *ConfigService) GetSettings() (*Settings, error) {
	configPath := c.getConfigPath()
	configDir := filepath.Dir(configPath)

	// If config doesn't exist, try to copy from embedded resources (if built by QMServer)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Try to copy from embedded config (if available)
		if err := c.copyEmbeddedConfig(configPath, configDir); err != nil {
			// If embedded config doesn't exist, return default settings
			return c.getDefaultSettings(), nil
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return c.getDefaultSettings(), nil
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return c.getDefaultSettings(), nil
	}

	return &settings, nil
}

// copyEmbeddedConfig tries to copy config from embedded resources
// This is used when QMLauncher is built by QMServer with pre-configured settings
func (c *ConfigService) copyEmbeddedConfig(destPath, destDir string) error {
	// Check if config exists in build directory (relative to executable)
	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	execDir := filepath.Dir(execPath)

	// Try different possible locations for embedded config
	possiblePaths := []string{
		filepath.Join(execDir, "config", "config.json"),             // Windows/Linux: next to executable
		filepath.Join(execDir, "..", "config", "config.json"),       // macOS: in app bundle
		filepath.Join(execDir, "..", "Resources", "config.json"),    // macOS: in Resources
		filepath.Join(execDir, "..", "..", "config", "config.json"), // macOS: deeper in bundle
	}

	for _, srcPath := range possiblePaths {
		if _, err := os.Stat(srcPath); err == nil {
			// Create destination directory
			if err := os.MkdirAll(destDir, 0755); err != nil {
				return fmt.Errorf("failed to create config directory: %v", err)
			}

			// Copy file
			srcData, err := os.ReadFile(srcPath)
			if err != nil {
				continue
			}

			if err := os.WriteFile(destPath, srcData, 0644); err != nil {
				return fmt.Errorf("failed to write config file: %v", err)
			}

			return nil
		}
	}

	return fmt.Errorf("embedded config not found")
}

// SaveSettings сохраняет настройки
func (c *ConfigService) SaveSettings(settings *Settings) error {
	configPath := c.getConfigPath()
	configDir := filepath.Dir(configPath)

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории конфигурации: %v", err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации настроек: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("ошибка записи конфигурации: %v", err)
	}

	return nil
}

// SyncConfigWithServer синхронизирует конфигурацию с сервером
func (c *ConfigService) SyncConfigWithServer() error {
	settings, err := c.GetSettings()
	if err != nil {
		return err
	}

	apiBaseUrl := settings.APIBaseURL
	if apiBaseUrl == "" {
		apiBaseUrl = "http://localhost:8000/api/v1"
	}

	// Получаем серверы
	serversUrl := fmt.Sprintf("%s/servers", apiBaseUrl)
	result, err := c.app.APIRequest(serversUrl, "GET", nil, "")
	if err != nil {
		return fmt.Errorf("ошибка получения серверов: %v", err)
	}

	// Сохраняем серверы в конфигурацию
	// Упрощенная версия - в реальности нужно сохранять в зашифрованном виде
	serversData, _ := json.Marshal(result["data"])
	serversPath := c.getServersConfigPath()
	os.MkdirAll(filepath.Dir(serversPath), 0755)
	os.WriteFile(serversPath, serversData, 0644)

	return nil
}

// Вспомогательные методы

func (c *ConfigService) getConfigPath() string {
	home := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		home = os.Getenv("USERPROFILE")
	}
	return filepath.Join(home, ".qmlauncher", "config.json")
}

func (c *ConfigService) getServersConfigPath() string {
	home := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		home = os.Getenv("USERPROFILE")
	}
	return filepath.Join(home, ".qmlauncher", "servers.json")
}

func (c *ConfigService) getDefaultSettings() *Settings {
	return &Settings{
		APIBaseURL:    "http://localhost:8000/api/v1",
		MinecraftPath: "",
		JavaPath:      "",
		MinMemory:     512,
		MaxMemory:     2048,
		JVMArgs:       []string{},
	}
}

// GetServers возвращает список серверов из конфигурации
func (c *ConfigService) GetServers() ([]ServerData, error) {
	serversPath := c.getServersConfigPath()

	if _, err := os.Stat(serversPath); os.IsNotExist(err) {
		return []ServerData{}, nil
	}

	data, err := os.ReadFile(serversPath)
	if err != nil {
		return []ServerData{}, nil
	}

	var servers []ServerData
	if err := json.Unmarshal(data, &servers); err != nil {
		return []ServerData{}, nil
	}

	return servers, nil
}

// GetMods возвращает список модов для сервера
func (c *ConfigService) GetMods(serverId int) ([]ModData, error) {
	modsPath := c.getModsConfigPath()

	if _, err := os.Stat(modsPath); os.IsNotExist(err) {
		return []ModData{}, nil
	}

	data, err := os.ReadFile(modsPath)
	if err != nil {
		return []ModData{}, nil
	}

	var allMods []ModData
	if err := json.Unmarshal(data, &allMods); err != nil {
		return []ModData{}, nil
	}

	// Фильтруем моды по serverId
	var serverMods []ModData
	for _, mod := range allMods {
		if mod.ServerID == serverId {
			serverMods = append(serverMods, mod)
		}
	}

	return serverMods, nil
}

// GetLauncherDbConfig возвращает конфигурацию для сервера
func (c *ConfigService) GetLauncherDbConfig(serverId int) (*LauncherDbConfig, error) {
	settings, err := c.GetSettings()
	if err != nil {
		return nil, err
	}

	mods, err := c.GetMods(serverId)
	if err != nil {
		return nil, err
	}

	config := map[string]interface{}{
		"api_base_url": settings.APIBaseURL,
	}

	// Пытаемся получить server_uuid из embedded servers
	embeddedServers, err := c.app.GetEmbeddedServers()
	if err == nil {
		for _, server := range embeddedServers {
			if server.ServerID == serverId {
				if server.ServerUUID != "" {
					config["server_uuid"] = server.ServerUUID
				}
				break
			}
		}
	}

	// Если не нашли в embedded, пробуем из локального списка servers.json
	if _, ok := config["server_uuid"]; !ok {
		if servers, err := c.GetServers(); err == nil {
			for _, s := range servers {
				if s.ID == serverId && s.ServerUUID != "" {
					config["server_uuid"] = s.ServerUUID
					break
				}
			}
		}
	}

	return &LauncherDbConfig{
		Success: true,
		Config:  config,
		Mods:    mods,
	}, nil
}

func (c *ConfigService) getModsConfigPath() string {
	home := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		home = os.Getenv("USERPROFILE")
	}
	return filepath.Join(home, ".qmlauncher", "mods.json")
}

// Settings структура настроек
type Settings struct {
	APIBaseURL       string   `json:"apiBaseUrl"`
	ServerUUID       string   `json:"serverUuid,omitempty"` // Server UUID for launcher identification
	MinecraftPath    string   `json:"minecraftPath"`
	JavaPath         string   `json:"javaPath"`
	MinMemory        int      `json:"minMemory"`
	MaxMemory        int      `json:"maxMemory"`
	JVMArgs          []string `json:"jvmArgs"`
	WindowWidth      int      `json:"windowWidth"`
	WindowHeight     int      `json:"windowHeight"`
	Resolution       string   `json:"resolution"`
	CustomResolution string   `json:"customResolution"`
}

// ServerData структура данных сервера
type ServerData struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	ServerName       string `json:"server_name,omitempty"`
	ServerAddress    string `json:"server_address,omitempty"`
	ServerPort       int    `json:"server_port"`
	MinecraftVersion string `json:"minecraft_version"`
	Description      string `json:"description,omitempty"`
	PreviewImageURL  string `json:"preview_image_url,omitempty"`
	ServerUUID       string `json:"server_uuid,omitempty"`
	LoaderEnabled    bool   `json:"loader_enabled"`
	LoaderType       string `json:"loader_type"`
	LoaderVersion    string `json:"loader_version,omitempty"`
}

// ModData структура данных мода
type ModData struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Version  string `json:"version,omitempty"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	ServerID int    `json:"server_id"`
}

// LauncherDbConfig структура конфигурации для сервера
type LauncherDbConfig struct {
	Success bool                   `json:"success"`
	Config  map[string]interface{} `json:"config"`
	Mods    []ModData              `json:"mods"`
	Error   string                 `json:"error,omitempty"`
}
