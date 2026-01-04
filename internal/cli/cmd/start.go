package cmd

import (
	env "QMLauncher/pkg"
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"QMLauncher/internal/cli/output"
	"QMLauncher/pkg/auth"
	"QMLauncher/pkg/launcher"

	"github.com/alecthomas/kong"
	"github.com/schollz/progressbar/v3"
)

var (
	logFile              *os.File
	interactiveDebugMode bool
)

// initLogging initializes logging to a file in the instance directory
func initLogging(instanceDir string) error {
	logsDir := filepath.Join(instanceDir, "logs")

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	logPath := filepath.Join(logsDir, "qmlauncher.log")
	var err error
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Set log output to file
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	logMessage("=== Запуск лаунчера ===")
	return nil
}

// logMessage logs a message to the log file and optionally to console in interactive debug mode
func logMessage(message string) {
	if logFile != nil {
		log.Println(message)
	}

	// Also print to console if in interactive debug mode
	if interactiveDebugMode {
		fmt.Printf("[DEBUG] %s\n", message)
	}
}

// SetInteractiveDebugMode sets the interactive debug mode for logging
func SetInteractiveDebugMode(enabled bool) {
	interactiveDebugMode = enabled
}

// closeLogging closes the log file
func closeLogging() {
	if logFile != nil {
		logMessage("=== Завершение работы лаунчера ===")
		logFile.Close()
		logFile = nil
	}
}

// QMServerCheckResponse represents the response from QMServer Cloud check/server endpoint
type QMServerCheckResponse struct {
	Exists    bool   `json:"exists"`
	ServerID  uint   `json:"server_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Version   string `json:"version,omitempty"`
	IsPremium bool   `json:"is_premium,omitempty"`
	Error     string `json:"error,omitempty"`
}

// QMServerInfo represents server information from QMServer Cloud servers list
type QMServerInfo struct {
	ID               uint   `json:"id"`
	UUID             string `json:"uuid"`
	Name             string `json:"name"`
	Host             string `json:"host"`
	Port             int    `json:"port"`
	Version          string `json:"version"`
	ModLoader        string `json:"mod_loader"`
	ModLoaderVersion string `json:"mod_loader_version"`
	IsPremium        bool   `json:"is_premium"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// QMServersResponse represents the response from QMServer Cloud servers endpoint
type QMServersResponse struct {
	Count          int            `json:"count"`
	ServerProfiles []QMServerInfo `json:"server_profiles"`
	Error          string         `json:"error,omitempty"`
}

// FileInfo represents file information with MD5 hash
type FileInfo struct {
	Path     string `json:"path"`
	MD5      string `json:"md5"`
	Size     int64  `json:"size"`
	Modified int64  `json:"modified"`
}

// DataManifest represents the data.json structure
type DataManifest struct {
	ServerID   uint       `json:"server_id"`
	ServerUUID string     `json:"server_uuid"`
	Files      []FileInfo `json:"files"`
	Generated  int64      `json:"generated"`
}

// ServerConnection represents a server connection entry
type ServerConnection struct {
	Username             string `json:"username"`
	Server               string `json:"server"`
	Instance             string `json:"instance"`
	Time                 int64  `json:"time"`
	IsUsingQMServerCloud bool   `json:"is_using_qmserver_cloud,omitempty"`
	IsPremium            bool   `json:"is_premium,omitempty"`
}

// getRecentConnectionsFile returns the path to the recent connections file
func getRecentConnectionsFile() string {
	return filepath.Join(env.RootDir, ".recent_connections.json")
}

// loadRecentConnections loads recent server connections from file
func loadRecentConnections() ([]ServerConnection, error) {
	filePath := getRecentConnectionsFile()
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ServerConnection{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var connections []ServerConnection
	if err := json.NewDecoder(file).Decode(&connections); err != nil {
		return nil, err
	}

	// Sort by time (newest first)
	sort.Slice(connections, func(i, j int) bool {
		return connections[i].Time > connections[j].Time
	})

	return connections, nil
}

// LoadRecentConnectionsFromFile loads recent server connections from file
func LoadRecentConnectionsFromFile() ([]ServerConnection, error) {
	return loadRecentConnections()
}

// saveRecentConnections saves recent server connections to file
func saveRecentConnections(connections []ServerConnection) error {
	filePath := getRecentConnectionsFile()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(connections)
}

// addRecentConnection adds a new connection to the recent connections list
//
//nolint:unused
func addRecentConnection(username, server, instance string) error {
	return addRecentConnectionWithCloudInfo(username, server, instance, false, false)
}

// addRecentConnectionWithCloudInfo adds a new connection to the recent connections list with QMServer Cloud info
func addRecentConnectionWithCloudInfo(username, server, instance string, isUsingQMServerCloud, isPremium bool) error {
	connections, err := loadRecentConnections()
	if err != nil {
		return err
	}

	// Remove duplicates (same username + server + instance)
	connections = filterConnections(connections, func(c ServerConnection) bool {
		return c.Username != username || c.Server != server || c.Instance != instance
	})

	// Add new connection at the beginning
	newConnection := ServerConnection{
		Username:             username,
		Server:               server,
		Instance:             instance,
		Time:                 time.Now().Unix(),
		IsUsingQMServerCloud: isUsingQMServerCloud,
		IsPremium:            isPremium,
	}
	connections = append([]ServerConnection{newConnection}, connections...)

	// Keep only last 20 connections
	if len(connections) > 20 {
		connections = connections[:20]
	}

	return saveRecentConnections(connections)
}

// filterConnections filters connections based on predicate
func filterConnections(connections []ServerConnection, predicate func(ServerConnection) bool) []ServerConnection {
	var filtered []ServerConnection
	for _, c := range connections {
		if predicate(c) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// checkQMServerCloud checks if a server exists in QMServer Cloud
func checkQMServerCloud(serverAddr string) (*QMServerCheckResponse, error) {
	// Parse server address (host:port)
	parts := strings.Split(serverAddr, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid server address format: %s", serverAddr)
	}
	host := parts[0]

	var port int
	if _, err := fmt.Sscanf(parts[1], "%d", &port); err != nil {
		return nil, fmt.Errorf("invalid port in server address: %s", serverAddr)
	}

	// QMServer Cloud endpoint
	qmServerURL := "http://178.172.201.248:8240/api/v1/check/server"

	// Create request payload
	requestBody := map[string]interface{}{
		"host": host,
		"port": port,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(qmServerURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to QMServer Cloud: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var response QMServerCheckResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if response.Error != "" {
			return nil, fmt.Errorf("QMServer Cloud error: %s", response.Error)
		}
		return nil, fmt.Errorf("QMServer Cloud returned status %d", resp.StatusCode)
	}

	return &response, nil
}

// downloadDataManifest downloads data.json from QMServer Cloud for the given server
func downloadDataManifest(serverID uint, qmServerHost string, qmServerPort int) (*DataManifest, error) {
	url := fmt.Sprintf("http://%s:%d/api/v1/check/data/%d", qmServerHost, qmServerPort, serverID)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to QMServer Cloud: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("QMServer Cloud returned status %d", resp.StatusCode)
	}

	var manifest DataManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to parse data manifest: %w", err)
	}

	return &manifest, nil
}

// getQMServersList fetches the list of servers from QMServer Cloud
func getQMServersList() (*QMServersResponse, error) {
	url := "http://178.172.201.248:8240/api/v1/servers"

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to QMServer Cloud: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("QMServer Cloud returned status %d", resp.StatusCode)
	}

	var serversResponse QMServersResponse
	if err := json.NewDecoder(resp.Body).Decode(&serversResponse); err != nil {
		return nil, fmt.Errorf("failed to parse servers list: %w", err)
	}

	return &serversResponse, nil
}

// downloadFile downloads a file from QMServer Cloud
func downloadFile(serverID uint, filePath string, qmServerHost string, qmServerPort int, destPath string) error {
	url := fmt.Sprintf("http://%s:%d/api/v1/download/%d/%s", qmServerHost, qmServerPort, serverID, filePath)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file, status: %d", resp.StatusCode)
	}

	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create destination file
	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer file.Close()

	// Copy content
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

// calculateFileMD5 calculates MD5 hash of a file
func calculateFileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// syncInstanceFiles synchronizes files between instance and QMServer Cloud data manifest
func syncInstanceFiles(inst launcher.Instance, manifest *DataManifest, qmServerHost string, qmServerPort int) error {
	logMessage("Начало синхронизации файлов инстанса")
	instanceDir := inst.Dir()

	// Create a map of files from manifest for quick lookup
	manifestFiles := make(map[string]FileInfo)
	for _, file := range manifest.Files {
		manifestFiles[file.Path] = file
	}

	// Sync files from manifest
	for filePath, fileInfo := range manifestFiles {
		instanceFilePath := filepath.Join(instanceDir, filePath)

		// Special handling for certain files and directories
		shouldSkipDownload := false

		// Don't overwrite options.txt if it exists
		if filePath == "options.txt" {
			if _, err := os.Stat(instanceFilePath); err == nil {
				shouldSkipDownload = true
			}
		}

		// Allow updating files in config directory (but not options.txt)
		// Config files will be updated if MD5 differs, but options.txt is protected

		if shouldSkipDownload {
			continue
		}

		// Check if file exists and has matching MD5
		if _, err := os.Stat(instanceFilePath); err == nil {
			// Calculate MD5 of existing file
			existingMD5, err := calculateFileMD5(instanceFilePath)
			if err != nil {
				output.Warning("Не удалось рассчитать MD5 для существующего файла %s: %v", instanceFilePath, err)
				continue
			}

			// If MD5 matches, skip download
			if existingMD5 == fileInfo.MD5 {
				continue
			}
		}

		// Download file with progress indication
		logMessage(fmt.Sprintf("Скачивание файла: %s", filePath))
		output.Progress("Скачивание: %s", filePath)

		if err := downloadFile(manifest.ServerID, filePath, qmServerHost, qmServerPort, instanceFilePath); err != nil {
			logMessage(fmt.Sprintf("Ошибка скачивания файла %s: %v", filePath, err))
			output.Error("Не удалось скачать файл %s: %v", filePath, err)
			continue
		}
		logMessage(fmt.Sprintf("Файл успешно скачан: %s", filePath))
	}

	// Remove files that exist locally but not in manifest (only for mods directory)
	logMessage("Удаление устаревших файлов")
	if err := removeOrphanedFiles(instanceDir, manifestFiles); err != nil {
		logMessage(fmt.Sprintf("Ошибка при удалении устаревших файлов: %v", err))
		output.Warning("Ошибка при удалении устаревших файлов: %v", err)
	}

	logMessage("Синхронизация файлов завершена")
	return nil
}

// manifestsEqual compares two data manifests for equality
func manifestsEqual(a, b *DataManifest) bool {
	if a == nil || b == nil {
		return false
	}

	// Compare basic fields
	if a.ServerID != b.ServerID || a.ServerUUID != b.ServerUUID || a.Generated != b.Generated {
		return false
	}

	// Compare files count
	if len(a.Files) != len(b.Files) {
		return false
	}

	// Compare files (create maps for efficient lookup)
	aFiles := make(map[string]FileInfo)
	bFiles := make(map[string]FileInfo)

	for _, f := range a.Files {
		aFiles[f.Path] = f
	}
	for _, f := range b.Files {
		bFiles[f.Path] = f
	}

	// Check if all files from a exist in b with same properties
	for path, aFile := range aFiles {
		bFile, exists := bFiles[path]
		if !exists {
			return false
		}
		if aFile.MD5 != bFile.MD5 || aFile.Size != bFile.Size || aFile.Modified != bFile.Modified {
			return false
		}
	}

	return true
}

// loadLocalDataManifest loads data.json from local instance directory
func loadLocalDataManifest(filePath string) (*DataManifest, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open data.json: %w", err)
	}
	defer file.Close()

	var manifest DataManifest
	if err := json.NewDecoder(file).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to decode data.json: %w", err)
	}

	return &manifest, nil
}

// saveDataManifest saves the data manifest as data.json locally
func saveDataManifest(filePath string, manifest *DataManifest) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for data.json: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create data.json: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		return fmt.Errorf("failed to encode manifest: %w", err)
	}

	return nil
}

// removeOrphanedFiles removes files that exist locally but not in the manifest
// Removes files from mods and config directories
func removeOrphanedFiles(instanceDir string, manifestFiles map[string]FileInfo) error {
	// Directories to check for orphaned files
	directoriesToClean := []string{"mods", "config", "shaderpacks", "resourcepacks", "schematics"}

	for _, dirName := range directoriesToClean {
		dirPath := filepath.Join(instanceDir, dirName)

		// Check if directory exists
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			continue // Directory doesn't exist, skip
		}

		// Walk through directory and find files not in manifest
		err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			// Get relative path from instance directory
			relPath, err := filepath.Rel(instanceDir, path)
			if err != nil {
				return err
			}

			// Convert backslashes to forward slashes for consistency with manifest
			relPath = filepath.ToSlash(relPath)

			// Check if file exists in manifest
			if _, exists := manifestFiles[relPath]; !exists {
				logMessage(fmt.Sprintf("Удаление устаревшего файла: %s", relPath))
				if err := os.Remove(path); err != nil {
					logMessage(fmt.Sprintf("Ошибка удаления файла %s: %v", relPath, err))
					output.Warning("Не удалось удалить устаревший файл %s: %v", relPath, err)
				}
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to walk directory %s: %w", dirPath, err)
		}
	}

	return nil
}

// QuietRunner runs the game without showing its console output
func QuietRunner(cmd *exec.Cmd) error {
	return cmd.Run()
}

func watcher(verbosity int) launcher.EventWatcher {
	var bar = progressbar.NewOptions(0,
		progressbar.OptionSetDescription(output.Translate("start.launch.downloading")),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionOnCompletion(func() {
			fmt.Print("\n")
		}),
		progressbar.OptionFullWidth())
	return func(event any) {
		switch e := event.(type) {
		case launcher.DownloadingEvent:
			bar.ChangeMax(e.Total)
			bar.Add(1)
		case launcher.AssetsResolvedEvent:
			if verbosity > 0 {
				output.Info(output.Translate("start.launch.assets"), e.Total)
			}
		case launcher.LibrariesResolvedEvent:
			if verbosity > 0 {
				output.Info(output.Translate("start.launch.libraries"), e.Total)
			}
		case launcher.MetadataResolvedEvent:
			if verbosity > 0 {
				output.Info(output.Translate("start.launch.metadata"))
			}
		case launcher.PostProcessingEvent:
			output.Info(output.Translate("start.processing"))
		}
	}
}

// StartCmd runs an instance with the specified options.
type StartCmd struct {
	ID string `arg:"" help:"${start_arg_id}"`

	Prepare bool `help:"${start_arg_prepare}"`

	NoJavaWindow bool `help:"${start_arg_nojavawindow}"`

	Options struct {
		Username    string `help:"${start_arg_username}" short:"u"`
		Server      string `help:"${start_arg_server}" placeholder:"IP" xor:"quickplay"`
		World       string `help:"${start_arg_world}" short:"w" placeholder:"NAME" xor:"quickplay"`
		Demo        bool   `help:"${start_arg_demo}"`
		DisableMP   bool   `help:"${start_arg_disablemp}"`
		DisableChat bool   `help:"${start_arg_disablechat}"`
	} `embed:"" group:"opts"`
	Overrides struct {
		Width     int    `help:"${start_arg_width}" and:"size"`
		Height    int    `help:"${start_arg_height}" and:"size"`
		JVM       string `help:"${start_arg_jvm}" type:"path" placeholder:"PATH"`
		JVMArgs   string `help:"${start_arg_jvmargs}"`
		MinMemory int    `help:"${start_arg_minmemory}" placeholder:"MB" and:"memory"`
		MaxMemory int    `help:"${start_arg_maxmemory}" placeholder:"MB" and:"memory"`
	} `embed:"" group:"overrides"`
}

func (c *StartCmd) Run(ctx *kong.Context, verbosity int) error {
	inst, err := launcher.FetchInstance(c.ID)
	if err != nil {
		return err
	}

	// Initialize logging
	if err := initLogging(inst.Dir()); err != nil {
		output.Warning("Не удалось инициализировать логирование: %v", err)
	}
	defer closeLogging()

	logMessage(fmt.Sprintf("Запуск инстанса: %s (ID: %s)", inst.Name, c.ID))

	config := inst.Config

	// Initialize cloud response variable
	var cloudResponse *QMServerCheckResponse

	// Handle memory settings - only save to config if values differ from saved ones
	configChanged := false
	if c.Overrides.MinMemory != 0 && c.Overrides.MinMemory != config.MinMemory {
		config.MinMemory = c.Overrides.MinMemory
		configChanged = true
	}
	if c.Overrides.MaxMemory != 0 && c.Overrides.MaxMemory != config.MaxMemory {
		config.MaxMemory = c.Overrides.MaxMemory
		configChanged = true
	}

	// Save updated config to instance only if something changed
	if configChanged {
		inst.Config = config
		if err := inst.WriteConfig(); err != nil {
			output.Warning(output.Translate("start.instance.save_error"), err)
		}
	}

	override := launcher.InstanceConfig{
		WindowResolution: struct {
			Width  int "toml:\"width\" json:\"width\""
			Height int "toml:\"height\" json:\"height\""
		}{
			Width:  c.Overrides.Width,
			Height: c.Overrides.Height,
		},
		Java:     c.Overrides.JVM,
		JavaArgs: c.Overrides.JVMArgs,
		// Memory settings are already handled above and saved to instance config
		MinMemory: config.MinMemory,
		MaxMemory: config.MaxMemory,
	}

	if override.WindowResolution.Width != 0 && override.WindowResolution.Height != 0 {
		config.WindowResolution = override.WindowResolution
	}
	if override.Java != "" {
		config.Java = override.Java
	}
	if override.JavaArgs != "" {
		config.JavaArgs = override.JavaArgs
	}

	// Use saved values as defaults if not specified
	if c.Options.Username == "" && config.LastUser != "" {
		c.Options.Username = config.LastUser
	}
	if c.Options.Server == "" && config.LastServer != "" {
		c.Options.Server = config.LastServer
	}

	session := auth.Session{
		Username: c.Options.Username,
	}
	if c.Options.Username == "" {
		session, err = auth.Authenticate()
		if err != nil {
			return fmt.Errorf("authenticate session: %w", err)
		}
	}

	// Save connection info if server is specified
	if c.Options.Server != "" && session.Username != "" {

		// Check QMServer Cloud for this server
		logMessage(fmt.Sprintf("Проверка QMServer Cloud для сервера: %s", c.Options.Server))
		var cloudErr error

		cloudResponse, cloudErr = checkQMServerCloud(c.Options.Server)
		if cloudErr != nil {
			logMessage(fmt.Sprintf("Ошибка проверки QMServer Cloud: %v", cloudErr))
			output.Warning("Не удалось проверить QMServer Cloud для сервера %s: %v", c.Options.Server, cloudErr)
		}

		// Update instance config with QMServer Cloud information
		configChanged := false
		if cloudResponse != nil && cloudResponse.Exists {
			logMessage(fmt.Sprintf("Сервер найден в QMServer Cloud: ID=%d, Name=%s, Premium=%v", cloudResponse.ServerID, cloudResponse.Name, cloudResponse.IsPremium))
			// Server exists in QMServer Cloud
			if !config.IsUsingQMServerCloud {
				config.IsUsingQMServerCloud = true
				config.QMServerHost = "178.172.201.248"
				config.QMServerPort = 8240
				configChanged = true
			}
			if config.IsPremium != cloudResponse.IsPremium {
				config.IsPremium = cloudResponse.IsPremium
				configChanged = true
			}
		} else {
			logMessage("Сервер не найден в QMServer Cloud или проверка не удалась")
			// Server not found in QMServer Cloud or check failed
			if config.IsUsingQMServerCloud {
				config.IsUsingQMServerCloud = false
				config.QMServerHost = ""
				config.QMServerPort = 0
				config.IsPremium = false
				configChanged = true
			}
		}

		// Save to global recent connections with QMServer Cloud info
		isUsingCloud := cloudResponse != nil && cloudResponse.Exists
		isPremium := cloudResponse != nil && cloudResponse.IsPremium
		if err := addRecentConnectionWithCloudInfo(session.Username, c.Options.Server, c.ID, isUsingCloud, isPremium); err != nil {
			output.Warning("Не удалось сохранить информацию о подключении: %v", err)
		}

		// Save to instance config
		if config.LastServer != c.Options.Server || config.LastUser != session.Username || configChanged {
			config.LastServer = c.Options.Server
			config.LastUser = session.Username
			inst.Config = config
			if err := inst.WriteConfig(); err != nil {
				output.Warning("Не удалось сохранить конфигурацию инстанса: %v", err)
			}
		}
	}

	launchEnv, err := launcher.Prepare(
		inst,
		launcher.LaunchOptions{
			Session: session,

			InstanceConfig:     config,
			QuickPlayServer:    c.Options.Server,
			QuickPlayWorld:     c.Options.World,
			Demo:               c.Options.Demo,
			DisableMultiplayer: c.Options.DisableMP,
			DisableChat:        c.Options.DisableChat,
			NoJavaWindow:       c.NoJavaWindow,
		},
		watcher(verbosity))

	if err != nil {
		return err
	}

	// Sync files with QMServer Cloud if enabled
	if config.IsUsingQMServerCloud && cloudResponse != nil && cloudResponse.Exists {
		logMessage("Начало синхронизации файлов с QMServer Cloud")

		// Always download fresh manifest from server
		logMessage(fmt.Sprintf("Скачивание манифеста данных для сервера ID: %d", cloudResponse.ServerID))
		freshManifest, err := downloadDataManifest(cloudResponse.ServerID, config.QMServerHost, config.QMServerPort)
		if err != nil {
			logMessage(fmt.Sprintf("Ошибка скачивания манифеста: %v", err))
			output.Warning("Не удалось скачать манифест данных: %v", err)
			return nil
		}
		logMessage(fmt.Sprintf("Манифест успешно скачан, файлов в манифесте: %d", len(freshManifest.Files)))

		dataJsonPath := filepath.Join(inst.Dir(), inst.UUID, "data.json")

		// Check if local data.json needs updating
		needsUpdate := true
		if localManifest, err := loadLocalDataManifest(dataJsonPath); err == nil {
			logMessage("Сравнение локального и свежего манифестов")
			// Compare manifests
			if manifestsEqual(localManifest, freshManifest) {
				logMessage("Манифесты идентичны, синхронизация не требуется")
				needsUpdate = false
			} else {
				logMessage("Манифесты отличаются, требуется синхронизация")
			}
		} else {
			logMessage("Локальный манифест не найден, требуется первая синхронизация")
		}

		// Save/update local data.json
		if err := saveDataManifest(dataJsonPath, freshManifest); err != nil {
			logMessage(fmt.Sprintf("Ошибка сохранения data.json: %v", err))
			output.Warning("Не удалось сохранить data.json: %v", err)
		} else {
			logMessage(fmt.Sprintf("data.json сохранен по пути: %s", dataJsonPath))
		}

		// Perform sync only if manifest was updated or first time
		if needsUpdate {
			logMessage("Начало синхронизации файлов инстанса")
			if err := syncInstanceFiles(inst, freshManifest, config.QMServerHost, config.QMServerPort); err != nil {
				logMessage(fmt.Sprintf("Ошибка синхронизации файлов: %v", err))
				output.Warning("Ошибка синхронизации файлов: %v", err)
			} else {
				logMessage("Синхронизация файлов успешно завершена")
				output.Success("Синхронизация файлов завершена")
			}
		}
	}

	if c.Prepare {
		output.Success(output.Translate("start.prepared"))
		return nil
	}

	if verbosity > 1 {
		output.Debug(output.Translate("start.launch.jvmargs"), launchEnv.JavaArgs)

		var gameArgs []string
		var hideNext bool
		for _, arg := range launchEnv.GameArgs {
			if hideNext {
				gameArgs = append(gameArgs, "***")
			} else {
				gameArgs = append(gameArgs, arg)
			}
			if arg == "--accessToken" || arg == "--uuid" {
				hideNext = true
			} else {
				hideNext = false
			}
		}
		output.Debug(output.Translate("start.launch.gameargs"), gameArgs)
		output.Debug(output.Translate("start.launch.info"), launchEnv.MainClass, launchEnv.GameDir)
	}

	// Show launch progress bar
	launchBar := progressbar.NewOptions(100,
		progressbar.OptionSetDescription(""),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(50),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionShowIts(),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionOnCompletion(func() {
			fmt.Print("\n")
		}),
	)

	// Simulate launch progress
	for i := 0; i <= 100; i += 10 {
		launchBar.Set(i)
		time.Sleep(50 * time.Millisecond)
	}
	launchBar.Finish()

	// Silent launch - no account message

	// Choose runner based on verbosity level
	var runner launcher.Runner
	if verbosity == 0 {
		// Default verbosity - hide Minecraft logs
		runner = QuietRunner
		logMessage("Запуск игры в тихом режиме (без показа логов Minecraft)")
	} else {
		// Extra/debug verbosity - show Minecraft logs
		runner = launcher.ConsoleRunner
		logMessage("Запуск игры с показом логов Minecraft")
	}

	logMessage("Запуск Minecraft с Java: " + strings.Join(launchEnv.JavaArgs, " "))
	return launcher.Launch(launchEnv, runner)
}
