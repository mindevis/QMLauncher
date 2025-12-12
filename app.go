package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetAppVersion returns the application version
func (a *App) GetAppVersion() string {
	return "1.0.0"
}

// GetPlatform returns the current platform
func (a *App) GetPlatform() string {
	return runtime.GOOS
}

// APIRequest makes an HTTP request via Go backend (for CORS handling)
func (a *App) APIRequest(url string, method string, headers map[string]string, body string) (map[string]interface{}, error) {
	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data interface{}
	if err := json.Unmarshal(respBody, &data); err != nil {
		// If not JSON, return as string
		data = string(respBody)
	}

	return map[string]interface{}{
		"ok":         resp.StatusCode >= 200 && resp.StatusCode < 300,
		"status":     resp.StatusCode,
		"statusText": resp.Status,
		"data":       data,
	}, nil
}

// WindowMinimize minimizes the window
func (a *App) WindowMinimize() {
	wailsRuntime.WindowMinimise(a.ctx)
}

// WindowMaximize toggles window maximized state
func (a *App) WindowMaximize() {
	if wailsRuntime.WindowIsMaximised(a.ctx) {
		wailsRuntime.WindowUnmaximise(a.ctx)
	} else {
		wailsRuntime.WindowMaximise(a.ctx)
	}
}

// WindowClose closes the window
func (a *App) WindowClose() {
	wailsRuntime.Quit(a.ctx)
}

// WindowIsMaximized returns if window is maximized
func (a *App) WindowIsMaximized() bool {
	return wailsRuntime.WindowIsMaximised(a.ctx)
}

// GetSettings возвращает настройки
func (a *App) GetSettings() (*Settings, error) {
	configService := NewConfigService(a)
	return configService.GetSettings()
}

// SaveSettings сохраняет настройки
func (a *App) SaveSettings(settings *Settings) error {
	configService := NewConfigService(a)
	return configService.SaveSettings(settings)
}

// InstallJava устанавливает Java
func (a *App) InstallJava(vendor string, version string, serverUuid string) error {
	javaService := NewJavaService(a)
	return javaService.InstallJava(vendor, version, serverUuid)
}

// GetJavaPath возвращает путь к Java
func (a *App) GetJavaPath(serverUuid string) (string, error) {
	javaService := NewJavaService(a)
	return javaService.GetJavaPath(serverUuid)
}

// GetHWID возвращает hardware ID
func (a *App) GetHWID() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}

	// Комбинируем несколько идентификаторов для уникальности
	identifiers := []string{
		hostname,
		runtime.GOOS,
		runtime.GOARCH,
	}

	// Получаем информацию о памяти
	memInfo, err := getMemoryInfo()
	if err == nil {
		identifiers = append(identifiers, memInfo)
	}

	// Создаем хеш
	hwidString := strings.Join(identifiers, "-")
	hash := sha256.Sum256([]byte(hwidString))
	return hex.EncodeToString(hash[:]), nil
}

// getMemoryInfo возвращает информацию о памяти для HWID
func getMemoryInfo() (string, error) {
	// Упрощенная версия - в реальности можно использовать syscall для получения детальной информации
	return runtime.GOARCH, nil
}

// GetScreenResolutions возвращает доступные разрешения экрана
func (a *App) GetScreenResolutions() ([]string, error) {
	// В Wails нет прямого доступа к разрешениям экрана через runtime
	// Возвращаем стандартные разрешения
	return []string{
		"1920x1080",
		"1366x768",
		"1280x720",
		"1024x768",
		"1600x900",
		"2560x1440",
		"3840x2160",
	}, nil
}

// ValidateJavaPath проверяет путь к Java
func (a *App) ValidateJavaPath(javaPath string) (*JavaValidationResult, error) {
	javaService := NewJavaService(a)
	return javaService.ValidateJavaPath(javaPath)
}

// GetEmbeddedServers возвращает embedded servers из конфигурации
func (a *App) GetEmbeddedServers() ([]EmbeddedServer, error) {
	configService := NewConfigService(a)
	servers, err := configService.GetServers()
	if err != nil {
		return []EmbeddedServer{}, nil
	}

	embeddedServers := make([]EmbeddedServer, 0, len(servers))
	for _, server := range servers {
		embeddedServers = append(embeddedServers, EmbeddedServer{
			ServerID:         server.ID,
			ServerUUID:       server.ServerUUID,
			ServerName:       server.ServerName,
			ServerAddress:    server.ServerAddress,
			ServerPort:       server.ServerPort,
			MinecraftVersion: server.MinecraftVersion,
			Description:      server.Description,
			PreviewImageURL:  server.PreviewImageURL,
			Enabled:          1,
		})
	}

	return embeddedServers, nil
}

// GetLauncherDbConfig возвращает конфигурацию для сервера
func (a *App) GetLauncherDbConfig(serverId int) (*LauncherDbConfig, error) {
	configService := NewConfigService(a)
	return configService.GetLauncherDbConfig(serverId)
}

// GetServerMods получает список модов с сервера
func (a *App) GetServerMods(serverId int, apiBaseUrl string) (*ServerModsResult, error) {
	modsService := NewModsService(a)
	return modsService.GetServerMods(serverId, apiBaseUrl)
}

// DownloadMod скачивает мод
func (a *App) DownloadMod(downloadUrl string, savePath string) (*DownloadResult, error) {
	modsService := NewModsService(a)
	return modsService.DownloadMod(downloadUrl, savePath)
}

// CheckAndUpdateMods проверяет и обновляет моды
func (a *App) CheckAndUpdateMods(serverId int, apiBaseUrl string) (*ModsUpdateResult, error) {
	modsService := NewModsService(a)
	return modsService.CheckAndUpdateMods(serverId, apiBaseUrl)
}

// UninstallMinecraft удаляет установленный Minecraft клиент
func (a *App) UninstallMinecraft(serverId int) (*UninstallResult, error) {
	minecraftService := NewMinecraftService(a)
	return minecraftService.UninstallMinecraft(serverId)
}

// CheckClientInstalled проверяет, установлен ли клиент
func (a *App) CheckClientInstalled(serverId int, serverUuid string) (*ClientCheckResult, error) {
	minecraftService := NewMinecraftService(a)
	return minecraftService.CheckClientInstalled(serverId, serverUuid)
}

// CheckQMLauncherDirExists проверяет, является ли это не первым запуском.
// Если директория .qmlauncher уже существует, считаем что пользователь прошел стартовый экран,
// даже если в ней пока нет конфигурационных файлов.
func (a *App) CheckQMLauncherDirExists() bool {
	home := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		home = os.Getenv("USERPROFILE")
	}
	baseDir := filepath.Join(home, ".qmlauncher")

	// Если директория существует — сразу возвращаем true
	if _, err := os.Stat(baseDir); err == nil {
		return true
	}

	// Считаем, что конфигурация создана, если есть один из файлов:
	// config.json, servers.json или маркер .mode_selected
	configPath := filepath.Join(baseDir, "config.json")
	serversPath := filepath.Join(baseDir, "servers.json")
	modeMarker := filepath.Join(baseDir, ".mode_selected")

	if _, err := os.Stat(configPath); err == nil {
		return true
	}
	if _, err := os.Stat(serversPath); err == nil {
		return true
	}
	if _, err := os.Stat(modeMarker); err == nil {
		return true
	}

	return false
}

// EmbeddedServer структура embedded сервера
type EmbeddedServer struct {
	ServerID         int    `json:"server_id"`
	ServerUUID       string `json:"server_uuid"`
	ServerName       string `json:"server_name"`
	ServerAddress    string `json:"server_address"`
	ServerPort       int    `json:"server_port"`
	MinecraftVersion string `json:"minecraft_version"`
	Description      string `json:"description"`
	PreviewImageURL  string `json:"preview_image_url"`
	Enabled          int    `json:"enabled"`
}
