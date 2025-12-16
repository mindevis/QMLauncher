package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

//go:embed embedded_config.json
var embeddedConfigData []byte

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

	// If config doesn't exist, try to copy from embedded config (if built by QMServer)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Try to copy from embedded config (embedded in .exe via Go embed)
		if err := c.copyEmbeddedConfig(configPath, configDir); err != nil {
			// If embedded config doesn't exist or is empty, return default settings
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

// HasEmbeddedConfig проверяет наличие встроенного конфига в бинарнике
// и проверяет, содержит ли он apiBaseUrl (режим 3 - сборка через QMServer)
func (c *ConfigService) HasEmbeddedConfig() (bool, error) {
	// Check if embedded config data exists and is not empty
	if len(embeddedConfigData) == 0 {
		return false, nil
	}

	// Try to decrypt and parse embedded config
	config, err := c.decryptEmbeddedConfig()
	if err != nil {
		// If decryption fails, config is not embedded (empty {} or invalid)
		return false, nil
	}

	// Check if apiBaseUrl exists and is not empty
	if apiBaseURL, ok := config["apiBaseUrl"].(string); ok && apiBaseURL != "" {
		return true, nil
	}

	return false, nil
}

// decryptEmbeddedConfig decrypts the embedded config data
func (c *ConfigService) decryptEmbeddedConfig() (map[string]interface{}, error) {
	// Check if embedded config data exists
	if len(embeddedConfigData) == 0 {
		return nil, fmt.Errorf("embedded config is empty")
	}

	// Remove quotes if it's a JSON string
	dataStr := strings.TrimSpace(string(embeddedConfigData))
	if strings.HasPrefix(dataStr, `"`) && strings.HasSuffix(dataStr, `"`) {
		// It's a JSON string, unquote it
		var unquoted string
		if err := json.Unmarshal(embeddedConfigData, &unquoted); err != nil {
			return nil, fmt.Errorf("failed to unquote embedded config: %v", err)
		}
		dataStr = unquoted
	}

	// Decode from base64
	encryptedData, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		// If it's not base64, try to parse as plain JSON (for empty {} case)
		var config map[string]interface{}
		if err := json.Unmarshal(embeddedConfigData, &config); err == nil {
			return config, nil
		}
		return nil, fmt.Errorf("failed to decode base64: %v", err)
	}

	// Use the same encryption key as QMServer
	secret := "qmlauncher-embedded-config-encryption-key-v1"

	// Derive a 32-byte key from the secret using PBKDF2
	key := pbkdf2.Key([]byte(secret), []byte("qmlauncher-config-salt"), 10000, 32, sha256.New)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %v", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %v", err)
	}

	// Extract nonce and ciphertext
	nonceSize := gcm.NonceSize()
	if len(encryptedData) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]

	// Decrypt the data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %v", err)
	}

	// Parse JSON
	var config map[string]interface{}
	if err := json.Unmarshal(plaintext, &config); err != nil {
		return nil, fmt.Errorf("failed to parse decrypted config: %v", err)
	}

	return config, nil
}

// copyEmbeddedConfig copies config from embedded data (embedded in .exe via Go embed)
// This is used when QMLauncher is built by QMServer with pre-configured settings
func (c *ConfigService) copyEmbeddedConfig(destPath, destDir string) error {
	// Decrypt embedded config
	config, err := c.decryptEmbeddedConfig()
	if err != nil {
		return fmt.Errorf("failed to decrypt embedded config: %v", err)
	}

	// Check if apiBaseUrl exists (required for embedded config to be valid)
	if apiBaseURL, ok := config["apiBaseUrl"].(string); !ok || apiBaseURL == "" {
		return fmt.Errorf("embedded config does not contain apiBaseUrl")
	}

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// Marshal decrypted config to JSON
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// Write decrypted config to user's config file
	if err := os.WriteFile(destPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
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
