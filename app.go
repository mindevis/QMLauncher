package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"QMLauncher/internal/debuglog"
	"QMLauncher/internal/i18n"
	"QMLauncher/internal/meta"
	"QMLauncher/internal/network"
	env "QMLauncher/pkg"
	"QMLauncher/pkg/auth"
	"QMLauncher/pkg/launcher"
	"QMLauncher/pkg/serversdat"
	"QMLauncher/pkg/updater"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/text/language"
)

// App struct
type App struct {
	ctx context.Context
}

var (
	logFile     *os.File
	lastQMError string // last error from GetQMServersList, for UI display

	curseForgeCloudMu    sync.Mutex
	curseForgeCloudKey   string
	curseForgeCloudKeyAt time.Time
)

const curseForgeCloudKeyTTL = 25 * time.Minute

// curseForgeEffectiveKeySource describes where meta.CurseForgeAPIKey() comes from (after CURSEFORGE_API_KEY in env).
func curseForgeEffectiveKeySource() string {
	if strings.TrimSpace(os.Getenv("CURSEFORGE_API_KEY")) != "" {
		return "env"
	}
	if curseForgeAPIKeyFromSettingsMap(readLauncherSettingsMap()) != "" {
		return "file"
	}
	if strings.TrimSpace(meta.CurseForgeAPIKey()) != "" {
		return "cloud"
	}
	return "none"
}

func curseForgeAPIKeyFromSettingsMap(cfg map[string]interface{}) string {
	if cfg == nil {
		return ""
	}
	s, ok := cfg["curseforge_api_key"].(string)
	if !ok {
		return ""
	}
	return meta.NormalizeCurseForgeAPIKey(s)
}

// computeLauncherCurseForgeKey resolves key: optional env is handled in meta.CurseForgeAPIKey.
// Priority: non-empty key in ~/.qmlauncher/settings.json → QMServer GET /launcher/curseforge-api-key when cloud-logged-in
// (Pro + CurseForge module + configured key). Local key must win when the user pasted it in the launcher, even if cloud has another key.
func computeLauncherCurseForgeKey() string {
	cfg := readLauncherSettingsMap()
	localKey := curseForgeAPIKeyFromSettingsMap(cfg)
	if localKey != "" {
		return localKey
	}
	acc := auth.GetDefaultCloudAccount()
	if acc != nil && strings.TrimSpace(acc.Token) != "" {
		if k := strings.TrimSpace(resolveCurseForgeKeyFromQMServerCloud()); k != "" {
			return k
		}
	}
	return ""
}

func resolveCurseForgeKeyFromQMServerCloud() string {
	if meta.IsCurseForgeCloudThrottled() {
		return ""
	}
	acc := auth.GetDefaultCloudAccount()
	if acc == nil || acc.Token == "" {
		curseForgeCloudMu.Lock()
		curseForgeCloudKey, curseForgeCloudKeyAt = "", time.Time{}
		curseForgeCloudMu.Unlock()
		return ""
	}
	curseForgeCloudMu.Lock()
	if curseForgeCloudKey != "" && time.Since(curseForgeCloudKeyAt) < curseForgeCloudKeyTTL {
		k := curseForgeCloudKey
		curseForgeCloudMu.Unlock()
		debuglog.LogCurseForgeKeyFromQMServer(k)
		return k
	}
	curseForgeCloudMu.Unlock()

	apiBase := network.EffectiveQMServerAPIBase()
	reqURL := apiBase + "/launcher/curseforge-api-key?token=" + url.QueryEscape(acc.Token)
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return ""
	}
	resp, err := network.QMServerHTTPClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusOK {
		logMessage(fmt.Sprintf("[CurseForge] QMServer GET /launcher/curseforge-api-key: status=%d base=%s (if 401, cloud token may be for a different API host than this base)", resp.StatusCode, apiBase))
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
			meta.MarkCurseForgeCloudKeyMiss()
		}
		curseForgeCloudMu.Lock()
		if resp.StatusCode == http.StatusUnauthorized {
			curseForgeCloudKey, curseForgeCloudKeyAt = "", time.Time{}
		}
		curseForgeCloudMu.Unlock()
		return ""
	}
	var data struct {
		APIKey string `json:"api_key"`
	}
	if json.Unmarshal(body, &data) != nil {
		meta.MarkCurseForgeCloudKeyMiss()
		return ""
	}
	key := meta.NormalizeCurseForgeAPIKey(data.APIKey)
	if key == "" {
		meta.MarkCurseForgeCloudKeyMiss()
		return ""
	}
	curseForgeCloudMu.Lock()
	curseForgeCloudKey = key
	curseForgeCloudKeyAt = time.Now()
	curseForgeCloudMu.Unlock()
	debuglog.LogCurseForgeKeyFromQMServer(key)
	return key
}

func clearCurseForgeCloudKeyCache() {
	curseForgeCloudMu.Lock()
	curseForgeCloudKey, curseForgeCloudKeyAt = "", time.Time{}
	curseForgeCloudMu.Unlock()
	meta.ResetCurseForgeCloudKeyMiss()
}

// initAppLogging initializes logging for the entire GUI application
func initAppLogging() error {
	// Use centralized logs directory - cross-platform home directory detection
	var homeDir string
	if home := os.Getenv("HOME"); home != "" {
		homeDir = home
	} else if home := os.Getenv("USERPROFILE"); home != "" {
		homeDir = home // Windows
	} else {
		// Fallback - try to get home directory
		if h, err := os.UserHomeDir(); err == nil {
			homeDir = h
		} else {
			return fmt.Errorf("cannot determine home directory")
		}
	}

	logsDir := filepath.Join(homeDir, ".qmlauncher", "logs")

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Create log filename for application
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFilename := fmt.Sprintf("qmlauncher-gui_%s.log", timestamp)
	logPath := filepath.Join(logsDir, logFilename)

	var err error
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Set log output to file
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	return nil
}

// logMessage logs a message to the log file
func logMessage(message string) {
	if logFile != nil {
		log.Println(message)
	}
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	meta.SetCurseForgeKeyChooser(computeLauncherCurseForgeKey)
	meta.RegisterCurseForgeAPI403Handler(clearCurseForgeCloudKeyCache)

	// Initialize logging first (needed for update check)
	if err := initAppLogging(); err != nil {
		_ = err
	}

	var startupCfg map[string]interface{}
	if homeDir, err := os.UserHomeDir(); err == nil {
		settingsPath := filepath.Join(homeDir, ".qmlauncher", "settings.json")
		if data, err := os.ReadFile(settingsPath); err == nil {
			var cfg map[string]interface{}
			if json.Unmarshal(data, &cfg) == nil {
				startupCfg = cfg
			}
		}
	}
	applyLauncherDebugFromSettings(startupCfg)

	// Encrypted vault: Microsoft + offline + cloud accounts
	if err := auth.LoadCredentials(); err != nil {
		logMessage(fmt.Sprintf("[Auth] LoadCredentials: %v", err))
	}

	// Load language and QMServer API target from settings file (default UI language: Russian)
	langConfigured := false
	if startupCfg != nil {
		applyAPITargetFromSettingsMap(startupCfg)
		if l, ok := startupCfg["language"].(string); ok && (l == "en" || l == "ru") {
			langConfigured = true
			if l == "en" {
				i18n.SetLang(language.English)
			} else {
				i18n.SetLang(language.Russian)
			}
		}
	}
	if !langConfigured {
		i18n.SetLang(language.Russian)
	}

	// Auto-update: QMServer-hosted release first, then GitHub, then legacy QMWeb /uploads (Windows MD5).
	updater.CheckAndApplyQMServerDistributionUpdate(logMessage)
	updater.CheckAndApplyGitHubBinaryUpdate(logMessage)
	updater.CheckAndApplyQMWebUpdate(logMessage)

	// Start periodic update check (every 30 min)
	go startPeriodicUpdateCheck(ctx, logMessage)
}

func launcherSettingsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".qmlauncher", "settings.json"), nil
}

func readLauncherSettingsMap() map[string]interface{} {
	path, err := launcherSettingsPath()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cfg map[string]interface{}
	if json.Unmarshal(data, &cfg) != nil {
		return nil
	}
	return cfg
}

func parseBoolish(v interface{}, defaultTrue bool) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		if s == "0" || s == "false" || s == "no" || s == "off" {
			return false
		}
		if s == "1" || s == "true" || s == "yes" || s == "on" {
			return true
		}
		return defaultTrue
	default:
		return defaultTrue
	}
}

func applyLauncherDebugFromSettings(cfg map[string]interface{}) {
	enabled := false
	if cfg != nil {
		if v, ok := cfg["launcher_debug"]; ok {
			enabled = parseBoolish(v, false)
		}
	}
	if err := debuglog.SetEnabled(enabled); err != nil {
		if enabled {
			logMessage(fmt.Sprintf("[debug] failed to start debug log file: %v", err))
		}
		return
	}
	if enabled {
		if p := debuglog.CurrentLogPath(); p != "" {
			logMessage(fmt.Sprintf("[debug] HTTP tracing enabled → %s", p))
		}
	}
}

// GetLauncherDebug returns whether ~/.qmlauncher/settings.json has launcher_debug enabled.
func (a *App) GetLauncherDebug() bool {
	cfg := readLauncherSettingsMap()
	if cfg == nil {
		return false
	}
	v, ok := cfg["launcher_debug"]
	if !ok {
		return false
	}
	return parseBoolish(v, false)
}

// SetLauncherDebug persists launcher_debug and opens or closes the *debug* trace log file.
func (a *App) SetLauncherDebug(enabled bool) string {
	path, err := launcherSettingsPath()
	if err != nil {
		return err.Error()
	}
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	var existing map[string]interface{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &existing)
	}
	if existing == nil {
		existing = make(map[string]interface{})
	}
	existing["launcher_debug"] = enabled
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err.Error()
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err.Error()
	}
	if err := debuglog.SetEnabled(enabled); err != nil {
		return err.Error()
	}
	if enabled {
		if p := debuglog.CurrentLogPath(); p != "" {
			logMessage(fmt.Sprintf("[debug] HTTP tracing enabled → %s", p))
		}
	} else {
		logMessage("[debug] HTTP tracing disabled")
	}
	return ""
}

func applyAPITargetFromSettingsMap(cfg map[string]interface{}) {
	useCloud := true
	custom := ""
	if cfg != nil {
		if v, ok := cfg["use_qmserver_cloud"]; ok {
			useCloud = parseBoolish(v, true)
		}
		if s, ok := cfg["custom_api_base"].(string); ok {
			custom = s
		}
	}
	if !useCloud && strings.TrimSpace(custom) == "" {
		useCloud = true
	}
	network.ApplyLauncherAPITarget(useCloud, custom)
	if useCloud && strings.TrimSpace(custom) != "" {
		logMessage("[Launcher API] use_qmserver_cloud is true but custom_api_base is set — custom URL is ignored. Set use_qmserver_cloud to false to use your QMServer, or remove custom_api_base. CurseForge key is read from the same host as Effective API base.")
	}
	logMessage(fmt.Sprintf("[Launcher API] effective QMServer API base: %s", network.EffectiveQMServerAPIBase()))
}

// startPeriodicUpdateCheck runs update check every 30 minutes; emits event when update found.
// Stops when ctx is cancelled (e.g. app shutdown) so the goroutine does not leak.
func startPeriodicUpdateCheck(ctx context.Context, logFn func(string)) {
	const interval = 30 * time.Minute
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !updater.QMServerDistributionUpdateAvailable(nil) &&
				!updater.GitHubBinaryUpdateAvailable() &&
				!updater.CheckForQMWebUpdate(nil) {
				continue
			}
			logFn("[AutoUpdate] Update available, emitting event")
			runtime.EventsEmit(ctx, "launcher-update-available", nil)
		}
	}
}

// ApplyLauncherUpdate applies available launcher update and restarts. Returns empty string on success;
// on success the process exits. Returns error message on failure.
func (a *App) ApplyLauncherUpdate() string {
	updater.CheckAndApplyQMServerDistributionUpdate(logMessage)
	updater.CheckAndApplyGitHubBinaryUpdate(logMessage)
	if err := updater.ApplyAndRestartQMWebUpdate(logMessage); err != nil {
		return err.Error()
	}
	return ""
}

// GetInstances returns list of available Minecraft instances
func (a *App) GetInstances() []launcher.Instance {
	instances, err := launcher.FetchAllInstances()
	if err != nil {
		return []launcher.Instance{}
	}
	return instances
}

// InstanceDetails represents extended information about an instance for the frontend.
type InstanceDetails struct {
	Name          string `json:"name"`
	UUID          string `json:"uuid"`
	GameVersion   string `json:"gameVersion"`
	Loader        string `json:"loader"`
	LoaderVersion string `json:"loaderVersion"`
	Dir           string `json:"dir"`

	Mods          []string `json:"mods"`
	Shaderpacks   []string `json:"shaderpacks"`
	Resourcepacks []string `json:"resourcepacks"`
	Datapacks     []string `json:"datapacks"`
	Modpacks      []string `json:"modpacks"`
	Schematics    []string `json:"schematics"`
	ConfigFiles   []string `json:"configFiles"`
	KubeJSFiles   []string `json:"kubejsFiles"`

	IsUsingQMServerCloud bool   `json:"isUsingQMServerCloud"`
	IsPremium            bool   `json:"isPremium"`
	LastServer           string `json:"lastServer"`
	LastUser             string `json:"lastUser"`

	// RemoteInstalls maps "category/path" (e.g. mods/foo.jar) to catalog install metadata.
	RemoteInstalls map[string]launcher.RemoteInstallMeta `json:"remoteInstalls"`

	// Catalog flags from QMServer when using cloud (false,false if unavailable). Local instances: both true.
	CatalogCurseforgeEnabled bool `json:"catalogCurseforgeEnabled"`
	CatalogModrinthEnabled   bool `json:"catalogModrinthEnabled"`
}

// ResourceStoreLinks holds store page URLs resolved from an on-disk resource filename.
type ResourceStoreLinks struct {
	CurseforgeURL string `json:"curseforgeUrl"`
	ModrinthURL   string `json:"modrinthUrl"`
}

// RemoteStoreSearchResponse is the catalog search result (resource store UI).
type RemoteStoreSearchResponse struct {
	Hits  []meta.RemoteStoreHit `json:"hits"`
	Error string                `json:"error"`
}

// GetInstanceDetails returns extended information about a specific instance,
// including lists of installed mods, shaderpacks, resourcepacks and other folders.
func (a *App) GetInstanceDetails(instanceName string) InstanceDetails {
	inst, err := launcher.FetchInstance(instanceName)
	if err != nil {
		// Return empty struct on error
		return InstanceDetails{}
	}

	instanceDir := inst.Dir()

	// Helper to list files (non-recursive) in a subdirectory
	listFiles := func(subdir string) []string {
		path := filepath.Join(instanceDir, subdir)
		entries, err := os.ReadDir(path)
		if err != nil {
			return []string{}
		}
		var names []string
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			names = append(names, e.Name())
		}
		return names
	}

	listDatapacks := func() []string {
		var out []string
		rootDP := filepath.Join(instanceDir, "datapacks")
		if entries, err := os.ReadDir(rootDP); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				out = append(out, filepath.ToSlash(filepath.Join("datapacks", e.Name())))
			}
		}
		savesDir := filepath.Join(instanceDir, "saves")
		worlds, err := os.ReadDir(savesDir)
		if err != nil {
			return out
		}
		for _, w := range worlds {
			if !w.IsDir() {
				continue
			}
			wname := w.Name()
			dpPath := filepath.Join(savesDir, wname, "datapacks")
			entries, err := os.ReadDir(dpPath)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				out = append(out, filepath.ToSlash(filepath.Join("saves", wname, "datapacks", e.Name())))
			}
		}
		return out
	}

	listModpackRootMarkerFiles := func() []string {
		candidates := []string{
			"manifest.json",
			"modrinth.index.json",
			"pack.toml",
			"minecraftinstance.json",
		}
		var out []string
		for _, c := range candidates {
			if _, err := os.Stat(filepath.Join(instanceDir, c)); err == nil {
				out = append(out, c)
			}
			dis := c + ".disabled"
			if _, err := os.Stat(filepath.Join(instanceDir, dis)); err == nil {
				out = append(out, dis)
			}
		}
		return out
	}

	modsList := listFiles("mods")

	details := InstanceDetails{
		Name:          inst.Name,
		UUID:          inst.UUID,
		GameVersion:   inst.GameVersion,
		Loader:        string(inst.Loader),
		LoaderVersion: inst.LoaderVersion,
		Dir:           instanceDir,

		Mods:          modsList,
		Shaderpacks:   listFiles("shaderpacks"),
		Resourcepacks: listFiles("resourcepacks"),
		Datapacks:     listDatapacks(),
		Modpacks:      listModpackRootMarkerFiles(),
		Schematics:    listFiles("schematics"),
		ConfigFiles:   listFiles("config"),
		KubeJSFiles:   listFiles("kubejs"),

		IsUsingQMServerCloud: inst.Config.IsUsingQMServerCloud,
		IsPremium:            inst.Config.IsPremium,
		LastServer:           inst.Config.LastServer,
		LastUser:             inst.Config.LastUser,
	}

	if inst.Loader == launcher.LoaderFabric {
		launcher.EnsureFabricAPIRemoteInstallRecord(instanceDir, details.Mods)
	}

	remoteInstalls := launcher.LoadRemoteInstalls(instanceDir)
	if remoteInstalls == nil {
		remoteInstalls = map[string]launcher.RemoteInstallMeta{}
	}
	details.RemoteInstalls = remoteInstalls

	cfCat, mrCat := instanceCatalogFlags(&inst)
	details.CatalogCurseforgeEnabled = cfCat
	details.CatalogModrinthEnabled = mrCat

	return details
}

// instanceCatalogFlags returns which remote-store tabs are enabled for the resource catalog.
// Toggles: ~/.qmlauncher/settings.json catalog_curseforge_enabled / catalog_modrinth_enabled (default true).
func instanceCatalogFlags(_ *launcher.Instance) (curseforge, modrinth bool) {
	cf := true
	mr := true
	if cfg := readLauncherSettingsMap(); cfg != nil {
		cf = parseBoolish(cfg["catalog_curseforge_enabled"], true)
		mr = parseBoolish(cfg["catalog_modrinth_enabled"], true)
	}
	return cf, mr
}

func resourceHasDisabledSuffix(name string) bool {
	return len(name) > len(".disabled") && strings.HasSuffix(strings.ToLower(name), ".disabled")
}

func resourceStripDisabledSuffix(name string) string {
	if resourceHasDisabledSuffix(name) {
		return name[:len(name)-len(".disabled")]
	}
	return name
}

func validateModsBasename(base string) error {
	base = strings.TrimSpace(base)
	if base == "" || base != filepath.Base(base) || strings.Contains(base, "..") {
		return fmt.Errorf("invalid mod file name")
	}
	lower := strings.ToLower(base)
	if strings.HasSuffix(lower, ".jar.disabled") {
		return nil
	}
	if strings.HasSuffix(lower, ".jar") {
		return nil
	}
	return fmt.Errorf("only .jar / .jar.disabled in mods")
}

func validateFlatBasename(base string) error {
	base = strings.TrimSpace(base)
	if base == "" || base != filepath.Base(base) || strings.Contains(base, "..") {
		return fmt.Errorf("invalid file name")
	}
	return nil
}

func modpackMarkerAllowed(name string) bool {
	stem := resourceStripDisabledSuffix(name)
	switch strings.ToLower(stem) {
	case "manifest.json", "modrinth.index.json", "pack.toml", "minecraftinstance.json":
		return true
	default:
		return false
	}
}

func validateDatapackRel(rel string) error {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" || strings.Contains(rel, "..") {
		return fmt.Errorf("invalid datapack path")
	}
	parts := strings.Split(rel, "/")
	switch len(parts) {
	case 2:
		if parts[0] == "datapacks" && parts[1] != "" {
			return nil
		}
	case 4:
		if parts[0] == "saves" && parts[1] != "" && parts[2] == "datapacks" && parts[3] != "" {
			return nil
		}
	}
	return fmt.Errorf("invalid datapack path")
}

func instanceResourceAbsPath(inst *launcher.Instance, category, relKey string) (string, error) {
	relKey = filepath.ToSlash(strings.TrimSpace(relKey))
	instanceDir := inst.Dir()
	switch category {
	case "mods":
		if err := validateModsBasename(relKey); err != nil {
			return "", err
		}
		return filepath.Join(instanceDir, "mods", filepath.Base(relKey)), nil
	case "resourcepacks", "shaderpacks":
		if err := validateFlatBasename(relKey); err != nil {
			return "", err
		}
		return filepath.Join(instanceDir, category, filepath.Base(relKey)), nil
	case "modpacks":
		if err := validateFlatBasename(relKey); err != nil {
			return "", err
		}
		base := filepath.Base(relKey)
		if !modpackMarkerAllowed(base) {
			return "", fmt.Errorf("not a modpack marker file")
		}
		return filepath.Join(instanceDir, base), nil
	case "datapacks":
		if err := validateDatapackRel(relKey); err != nil {
			return "", err
		}
		return filepath.Join(instanceDir, filepath.FromSlash(relKey)), nil
	default:
		return "", fmt.Errorf("unknown category")
	}
}

func applyResourceDisabledToggle(absPath string, enabled bool) error {
	dir := filepath.Dir(absPath)
	base := filepath.Base(absPath)
	if enabled {
		if !resourceHasDisabledSuffix(base) {
			return nil
		}
		newBase := resourceStripDisabledSuffix(base)
		newPath := filepath.Join(dir, newBase)
		if _, err := os.Stat(newPath); err == nil {
			return fmt.Errorf("target file already exists")
		}
		return os.Rename(absPath, newPath)
	}
	if resourceHasDisabledSuffix(base) {
		return nil
	}
	dis := absPath + ".disabled"
	if _, err := os.Stat(dis); err == nil {
		return fmt.Errorf("disabled file already exists")
	}
	return os.Rename(absPath, dis)
}

// SetInstanceResourceEnabled toggles *.{ext} <-> *.{ext}.disabled (mods, resourcepacks, shaderpacks, modpack root markers, datapacks).
// category: mods | resourcepacks | shaderpacks | modpacks | datapacks
// resourcePath: basename для mods/resourcepacks/shaderpacks/modpacks; datapacks — datapacks/a.zip или saves/W/datapacks/a.zip
func (a *App) SetInstanceResourceEnabled(instanceName, category, resourcePath string, enabled bool) string {
	instanceName = strings.TrimSpace(instanceName)
	category = strings.TrimSpace(strings.ToLower(category))
	inst, err := launcher.FetchInstance(instanceName)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	absPath, err := instanceResourceAbsPath(&inst, category, resourcePath)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	if _, err := os.Stat(absPath); err != nil {
		return fmt.Sprintf("Error: file not found: %s", filepath.Base(absPath))
	}
	if err := applyResourceDisabledToggle(absPath, enabled); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return ""
}

// DeleteInstanceResource removes a file in the given category (active or .disabled).
func (a *App) DeleteInstanceResource(instanceName, category, resourcePath string) string {
	instanceName = strings.TrimSpace(instanceName)
	category = strings.TrimSpace(strings.ToLower(category))
	inst, err := launcher.FetchInstance(instanceName)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	absPath, err := instanceResourceAbsPath(&inst, category, resourcePath)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	if err := os.Remove(absPath); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	launcher.RemoveRemoteInstall(inst.Dir(), category, resourcePath)
	return ""
}

// SetInstanceModEnabled is a shortcut for SetInstanceResourceEnabled(..., "mods", ...).
func (a *App) SetInstanceModEnabled(instanceName, modFileName string, enabled bool) string {
	return a.SetInstanceResourceEnabled(instanceName, "mods", modFileName, enabled)
}

// DeleteInstanceMod is a shortcut for DeleteInstanceResource(..., "mods", ...).
func (a *App) DeleteInstanceMod(instanceName, modFileName string) string {
	return a.DeleteInstanceResource(instanceName, "mods", modFileName)
}

func startDefaultBrowser(target string) error {
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	case "darwin":
		cmd = exec.Command("open", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	return cmd.Start()
}

// OpenBrowserURL opens an http(s) URL in the system default browser.
func (a *App) OpenBrowserURL(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return "Error: empty URL"
	}
	u, err := url.Parse(target)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "Error: invalid URL"
	}
	if err := startDefaultBrowser(target); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return ""
}

// ResolveInstanceResourceStoreLinks resolves CurseForge and Modrinth links for a resource row (best effort).
// category: mods | resourcepacks | shaderpacks | datapacks — for modpacks returns empty URLs.
func (a *App) ResolveInstanceResourceStoreLinks(instanceName, category, storagePath string) ResourceStoreLinks {
	var out ResourceStoreLinks
	category = strings.ToLower(strings.TrimSpace(category))
	instanceName = strings.TrimSpace(instanceName)
	inst, err := launcher.FetchInstance(instanceName)
	if err != nil {
		return out
	}
	absPath, err := instanceResourceAbsPath(&inst, category, storagePath)
	if err != nil {
		return out
	}
	if _, err := os.Stat(absPath); err != nil {
		return out
	}

	caches := inst.CachesDir()
	loader := string(inst.Loader)
	gameVer := inst.GameVersion

	base := filepath.Base(filepath.ToSlash(strings.TrimSpace(storagePath)))
	base = resourceStripDisabledSuffix(base)

	switch category {
	case "modpacks":
		return out
	case "mods":
		if !strings.HasSuffix(strings.ToLower(base), ".jar") {
			return out
		}
		mi := meta.ExtractModInfoFromFilename(base)
		mi = meta.GetModLinks(mi, caches, loader, gameVer)
		out.CurseforgeURL = mi.CurseForgeURL
		out.ModrinthURL = mi.ModrinthURL
		if out.ModrinthURL == "" {
			out.ModrinthURL = "https://modrinth.com/search?q=" + url.QueryEscape(mi.Slug)
		}
	case "resourcepacks":
		rp := meta.ExtractResourcePackInfo(base)
		rp = meta.GetResourcePackLinks(rp, caches, gameVer)
		out.CurseforgeURL = meta.SearchResourcePackOnCurseForge(rp.Slug, caches, gameVer)
		out.ModrinthURL = rp.ModrinthURL
		if out.ModrinthURL == "" {
			if slug, err := meta.SearchModrinthProjectByType(rp.Slug, "resourcepack", caches); err == nil {
				out.ModrinthURL = "https://modrinth.com/resourcepack/" + slug
			} else {
				out.ModrinthURL = "https://modrinth.com/search?q=" + url.QueryEscape(rp.Slug)
			}
		}
	case "shaderpacks":
		rp := meta.ExtractResourcePackInfo(base)
		out.CurseforgeURL = meta.CurseForgeMinecraftSearchURL(rp.Slug)
		if slug, err := meta.SearchModrinthProjectByType(rp.Slug, "shader", caches); err == nil {
			out.ModrinthURL = "https://modrinth.com/shader/" + slug
		} else {
			out.ModrinthURL = "https://modrinth.com/search?q=" + url.QueryEscape(rp.Slug)
		}
	case "datapacks":
		rp := meta.ExtractResourcePackInfo(base)
		out.CurseforgeURL = meta.CurseForgeMinecraftSearchURL(rp.Slug)
		if slug, err := meta.SearchModrinthProjectByType(rp.Slug, "datapack", caches); err == nil {
			out.ModrinthURL = "https://modrinth.com/datapack/" + slug
		} else {
			out.ModrinthURL = "https://modrinth.com/search?q=" + url.QueryEscape(rp.Slug)
		}
	}
	cfOn, mrOn := instanceCatalogFlags(&inst)
	if !cfOn {
		out.CurseforgeURL = ""
	}
	if !mrOn {
		out.ModrinthURL = ""
	}
	return out
}

// SearchRemoteStore searches CurseForge or Modrinth (or both interleaved) for installable content.
// source: curseforge | modrinth | both
// curseSort / modrinthSort: popularity | downloads
func (a *App) SearchRemoteStore(instanceName, category, source, query, curseSort, modrinthSort string, page int) RemoteStoreSearchResponse {
	instanceName = strings.TrimSpace(instanceName)
	inst, err := launcher.FetchInstance(instanceName)
	if err != nil {
		return RemoteStoreSearchResponse{Error: err.Error()}
	}
	caches := inst.CachesDir()
	query = strings.TrimSpace(query)
	if query == "" {
		query = "minecraft"
	}
	source = strings.ToLower(strings.TrimSpace(source))
	if page < 0 {
		page = 0
	}
	const pageSize = 20
	cfOn, mrOn := instanceCatalogFlags(&inst)
	switch source {
	case "curseforge":
		if !cfOn {
			return RemoteStoreSearchResponse{Error: "Каталог CurseForge отключён в настройках лаунчера"}
		}
		hits, err := meta.SearchCurseForgeStore(category, query, curseSort, page, pageSize, caches)
		if err != nil {
			return RemoteStoreSearchResponse{Error: err.Error()}
		}
		return RemoteStoreSearchResponse{Hits: hits}
	case "modrinth":
		if !mrOn {
			return RemoteStoreSearchResponse{Error: "Каталог Modrinth отключён в настройках лаунчера"}
		}
		hits, err := meta.SearchModrinthStore(category, query, modrinthSort, page, pageSize, caches)
		if err != nil {
			return RemoteStoreSearchResponse{Error: err.Error()}
		}
		return RemoteStoreSearchResponse{Hits: hits}
	case "both":
		if !cfOn && !mrOn {
			return RemoteStoreSearchResponse{Error: "Каталоги CurseForge и Modrinth отключены в настройках лаунчера"}
		}
		if !cfOn {
			mr, errMr := meta.SearchModrinthStore(category, query, modrinthSort, page, pageSize, caches)
			if errMr != nil {
				return RemoteStoreSearchResponse{Error: errMr.Error()}
			}
			return RemoteStoreSearchResponse{Hits: mr}
		}
		if !mrOn {
			cf, errCf := meta.SearchCurseForgeStore(category, query, curseSort, page, pageSize, caches)
			if errCf != nil {
				return RemoteStoreSearchResponse{Error: errCf.Error()}
			}
			return RemoteStoreSearchResponse{Hits: cf}
		}
		cf, errCf := meta.SearchCurseForgeStore(category, query, curseSort, page, pageSize, caches)
		mr, errMr := meta.SearchModrinthStore(category, query, modrinthSort, page, pageSize, caches)
		if errMr != nil {
			return RemoteStoreSearchResponse{Error: errMr.Error()}
		}
		var cfHits []meta.RemoteStoreHit
		if errCf == nil {
			cfHits = cf
		}
		return RemoteStoreSearchResponse{Hits: meta.MergeRemoteStoreHits(cfHits, mr)}
	default:
		return RemoteStoreSearchResponse{Error: "неизвестный источник каталога"}
	}
}

func remoteStoreDestDir(inst *launcher.Instance, category string) (string, error) {
	instanceDir := inst.Dir()
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "mods":
		return filepath.Join(instanceDir, "mods"), nil
	case "resourcepacks":
		return filepath.Join(instanceDir, "resourcepacks"), nil
	case "shaderpacks":
		return filepath.Join(instanceDir, "shaderpacks"), nil
	case "datapacks":
		return filepath.Join(instanceDir, "datapacks"), nil
	case "modpacks":
		return instanceDir, nil
	default:
		return "", fmt.Errorf("unknown category")
	}
}

// DownloadRemoteStoreProject downloads a file from Modrinth or CurseForge.
// title is optional display name from the catalog for stored metadata.
// iconURL is optional HTTPS thumbnail URL from the catalog (shown in instance resource lists).
func (a *App) DownloadRemoteStoreProject(instanceName, category, storeSource, projectID, slug, title, iconURL string) string {
	instanceName = strings.TrimSpace(instanceName)
	inst, err := launcher.FetchInstance(instanceName)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	destDir, err := remoteStoreDestDir(&inst, category)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	storeSource = strings.ToLower(strings.TrimSpace(storeSource))
	cfOn, mrOn := instanceCatalogFlags(&inst)
	catKey := strings.ToLower(strings.TrimSpace(category))
	var savedPath string
	switch storeSource {
	case "modrinth":
		if !mrOn {
			return "Error: Modrinth catalog is disabled in launcher settings"
		}
		s := strings.TrimSpace(slug)
		if s == "" {
			s = strings.TrimSpace(projectID)
		}
		savedPath, err = meta.DownloadModrinthProjectTo(s, inst.GameVersion, string(inst.Loader), catKey, destDir)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
	case "curseforge":
		if !cfOn {
			return "Error: CurseForge catalog is disabled in launcher settings"
		}
		savedPath, err = meta.DownloadCurseForgeProjectTo(projectID, inst.GameVersion, string(inst.Loader), catKey, meta.CurseForgeAPIKey(), destDir)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
	default:
		return "Error: unknown store"
	}
	baseName := filepath.Base(savedPath)
	launcher.RecordRemoteInstall(inst.Dir(), catKey, baseName, launcher.RemoteInstallMeta{
		Category:  catKey,
		Source:    storeSource,
		ProjectID: strings.TrimSpace(projectID),
		Slug:      strings.TrimSpace(slug),
		Title:     strings.TrimSpace(title),
		IconURL:   strings.TrimSpace(iconURL),
	})
	return ""
}

// SetInstanceMemory sets min (-Xms) and max (-Xmx) memory for an instance in MB.
// Both default to 4096. minMemoryMB must be <= maxMemoryMB. Returns error string on failure.
func (a *App) SetInstanceMemory(instanceName string, minMemoryMB int, maxMemoryMB int) string {
	if instanceName == "" {
		return "Error: empty instance name"
	}
	if minMemoryMB < 128 || minMemoryMB > 32768 {
		return "Error: min memory must be between 128 and 32768 MB"
	}
	if maxMemoryMB < 512 || maxMemoryMB > 32768 {
		return "Error: max memory must be between 512 and 32768 MB"
	}
	if minMemoryMB > maxMemoryMB {
		return "Error: min memory cannot exceed max memory"
	}
	inst, err := launcher.FetchInstance(instanceName)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	inst.Config.MinMemory = minMemoryMB
	inst.Config.MaxMemory = maxMemoryMB
	if err := inst.WriteConfig(); err != nil {
		return fmt.Sprintf("Error: failed to save config: %v", err)
	}
	return ""
}

// CreateInstance creates a new Minecraft instance.
// loader: "vanilla", "fabric", "quilt", "forge", "neoforge"
// gameVersion: e.g. "1.20.1", "release" for latest
// loaderVersion: e.g. "latest" for Fabric/Quilt/Forge; empty for vanilla
// Returns empty string on success, error message on failure.
func (a *App) CreateInstance(name string, gameVersion string, loader string, loaderVersion string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Error: имя инстанса не может быть пустым"
	}
	var ldr launcher.Loader
	switch strings.ToLower(strings.TrimSpace(loader)) {
	case "fabric":
		ldr = launcher.LoaderFabric
	case "quilt":
		ldr = launcher.LoaderQuilt
	case "neoforge":
		ldr = launcher.LoaderNeoForge
	case "forge":
		ldr = launcher.LoaderForge
	default:
		ldr = launcher.LoaderVanilla
	}
	if gameVersion == "" {
		gameVersion = "release"
	}
	if loaderVersion == "" && ldr != launcher.LoaderVanilla {
		loaderVersion = "latest"
	}
	defaultConfig := launcher.InstanceConfig{
		WindowResolution: struct {
			Width  int `toml:"width" json:"width"`
			Height int `toml:"height" json:"height"`
		}{Width: 1708, Height: 960},
		MinMemory: 4096,
		MaxMemory: 4096,
	}
	options := launcher.InstanceOptions{
		Name:          name,
		GameVersion:   gameVersion,
		Loader:        ldr,
		LoaderVersion: loaderVersion,
		Config:        defaultConfig,
	}
	_, err := launcher.CreateInstance(options)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return ""
}

// GetCreateInstanceMinecraftVersions returns Minecraft release version ids (newest first) for the create-instance UI.
// Prefer QMServer Cloud; on failure or empty list fall back to Mojang manifest via local cache.
func (a *App) GetCreateInstanceMinecraftVersions() []string {
	cloudBase := network.EffectiveQMServerAPIBase()
	if v, err := meta.FetchCreateInstanceMinecraftFromQMServerCloud(cloudBase); err == nil && len(v) > 0 {
		return v
	} else if err != nil {
		logMessage(fmt.Sprintf("[CreateInstance] QMServer Cloud minecraft versions: %v (using fallback)", err))
	}
	v, err := meta.ListCreateInstanceMinecraftReleases(env.CachesDir)
	if err != nil {
		logMessage(fmt.Sprintf("[CreateInstance] GetCreateInstanceMinecraftVersions: %v", err))
		return nil
	}
	return v
}

// GetCreateInstanceLoaderVersions returns loader versions for the given loader type and Minecraft version.
// loader: fabric | quilt | forge | neoforge
// Prefer QMServer Cloud; on failure or empty list fall back to Fabric/Quilt/Forge/NeoForge upstream APIs.
func (a *App) GetCreateInstanceLoaderVersions(loader string, gameVersion string) []string {
	cloudBase := network.EffectiveQMServerAPIBase()
	if v, err := meta.FetchCreateInstanceLoaderFromQMServerCloud(cloudBase, loader, gameVersion); err == nil && len(v) > 0 {
		return v
	} else if err != nil {
		logMessage(fmt.Sprintf("[CreateInstance] QMServer Cloud loader versions (%q, %q): %v (using fallback)", loader, gameVersion, err))
	}
	v, err := meta.ListCreateInstanceLoaderVersions(loader, gameVersion)
	if err != nil {
		logMessage(fmt.Sprintf("[CreateInstance] GetCreateInstanceLoaderVersions(%q, %q): %v", loader, gameVersion, err))
		return nil
	}
	return v
}

// DeleteInstance removes a Minecraft instance by name. Returns empty string on success.
func (a *App) DeleteInstance(instanceName string) string {
	instanceName = strings.TrimSpace(instanceName)
	if instanceName == "" {
		return "Error: имя инстанса не может быть пустым"
	}
	if err := launcher.RemoveInstance(instanceName); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return ""
}

// OpenPath opens a path (file or folder) in the system default application.
// For directories, opens in file explorer. Returns empty on success, error message on failure.
func (a *App) OpenPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "Error: path is empty"
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	if _, err := os.Stat(absPath); err != nil {
		return fmt.Sprintf("Error: path not found: %v", err)
	}
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", absPath)
	case "darwin":
		cmd = exec.Command("open", absPath)
	default:
		cmd = exec.Command("xdg-open", absPath)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return ""
}

// ServerInfo represents server information for frontend
type ServerInfo struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Address          string `json:"address"`
	Port             int    `json:"port"`
	Online           bool   `json:"online"`
	Enabled          bool   `json:"enabled"`
	GameServerOnline *bool  `json:"gameServerOnline,omitempty"`
	Players          int    `json:"players"`
	MaxPlayers       int    `json:"maxPlayers"`
	Version          string `json:"version"`
	ModLoader        string `json:"modLoader,omitempty"`
	ModLoaderVersion string `json:"modLoaderVersion,omitempty"`
	IsPremium        bool   `json:"isPremium,omitempty"`
	ServerID         uint   `json:"serverID,omitempty"`
}

// GetQMServersError returns the last error from loading servers (empty if none)
func (a *App) GetQMServersError() string {
	return lastQMError
}

// GetRecentServers returns list of servers from QMServer Cloud
func (a *App) GetRecentServers() []ServerInfo {
	lastQMError = ""
	serversResponse, err := network.GetQMServersList()
	if err != nil {
		lastQMError = err.Error()
		log.Printf("[QMServer] Failed to fetch servers: %v", err)
		return []ServerInfo{}
	}

	if serversResponse.Error != "" {
		lastQMError = serversResponse.Error
		log.Printf("[QMServer] API error: %s", serversResponse.Error)
		return []ServerInfo{}
	}

	// Convert QMServerInfo to ServerInfo
	servers := make([]ServerInfo, 0, len(serversResponse.ServerProfiles))
	for _, server := range serversResponse.ServerProfiles {
		enabled := network.QMServerProfileEnabled(server)
		gameUp := true
		if server.GameServerOnline != nil {
			gameUp = *server.GameServerOnline
		}
		pl := 0
		if server.Players != nil {
			pl = *server.Players
		}
		maxPlayers := 0
		if server.MaxPlayers != nil {
			maxPlayers = *server.MaxPlayers
		}
		servers = append(servers, ServerInfo{
			ID:               strconv.Itoa(int(server.ID)),
			Name:             server.Name,
			Address:          server.Host,
			ServerID:         server.ID,
			Port:             server.Port,
			Online:           enabled && gameUp,
			Enabled:          enabled,
			GameServerOnline: server.GameServerOnline,
			Players:          pl,
			MaxPlayers:       maxPlayers,
			Version:          server.Version,
			ModLoader:        server.ModLoader,
			ModLoaderVersion: server.ModLoaderVersion,
			IsPremium:        server.IsPremium,
		})
	}

	return servers
}

// LaunchInstance launches an instance with optional server connection - exact copy of TUI launchInstance
// syncConfigFromServer: when true and serverID > 0, sync config/ and options.txt from QMServer Cloud (overwrite local)
func (a *App) LaunchInstance(instanceName string, serverAddress string, serverID uint, syncConfigFromServer bool) string {
	return a.LaunchInstanceWithAccount(instanceName, serverAddress, serverID, syncConfigFromServer, "", "", "", "")
}

// LaunchInstanceWithAccount launches an instance with a specific account selected
// selectedAccountUsername: username of the selected account (can be local, cloud_game, or microsoft)
// disabledModsJSON: optional JSON array of mod paths to exclude from sync (e.g. ["mods/sodium.jar"])
// enabledResourcepacksOrderJSON: optional JSON array of resourcepack paths in load order (from QMAdmin load_order)
// serverName: display name for servers.dat when connecting to server (optional)
func (a *App) LaunchInstanceWithAccount(instanceName string, serverAddress string, serverID uint, syncConfigFromServer bool, selectedAccountUsername string, disabledModsJSON string, enabledResourcepacksOrderJSON string, serverName string) string {
	// Fetch instance
	inst, err := launcher.FetchInstance(instanceName)
	if err != nil {
		return fmt.Sprintf("Error: Instance '%s' not found: %v", instanceName, err)
	}

	if err := network.CheckServerProfileConnectAllowed(serverID); err != nil {
		if errors.Is(err, network.ErrServerProfileDisabled) {
			return "Error: " + i18n.Translate("ui.server.disabled_error")
		}
	}

	// Call internal launchInstance function with selected account
	err = a.launchInstance(inst, serverAddress, serverID, syncConfigFromServer, selectedAccountUsername, disabledModsJSON, enabledResourcepacksOrderJSON, serverName)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return fmt.Sprintf("Success: Launched %s", instanceName)
}

// launchInstance is exact copy of TUI launchInstance function
// Note: This function needs access to App context for events, so it's now a method
// syncConfigFromServer: when true and serverID > 0, sync only config/ and options.txt from QMServer (overwrite)
// selectedAccountUsername: if not empty, use this specific account instead of default
// disabledModsJSON: JSON array of mod paths to exclude from sync and remove from local instance
// enabledResourcepacksOrderJSON: optional JSON array of resourcepack paths in load order for options.txt
func (a *App) launchInstance(inst launcher.Instance, serverAddress string, serverID uint, syncConfigFromServer bool, selectedAccountUsername string, disabledModsJSON string, enabledResourcepacksOrderJSON string, serverName string) error {
	logMessage(fmt.Sprintf("=== Запуск инстанса: %s (serverID: %d) ===", inst.Name, serverID))
	if serverAddress != "" {
		logMessage(fmt.Sprintf("Автоподключение к серверу: %s", serverAddress))
	}

	// Initialize logging
	logMessage("Инициализация логирования")
	if err := initLogging(inst.Name); err != nil {
		logMessage(fmt.Sprintf("Не удалось инициализировать логирование: %v", err))
	}

	logMessage(fmt.Sprintf("Запуск инстанса: %s", inst.Name))

	// Require specific account selection - no default accounts allowed when connecting to server
	if serverAddress != "" && selectedAccountUsername == "" {
		return fmt.Errorf("необходимо выбрать игровой аккаунт для подключения к серверу")
	}

	// Resolve session: if selectedAccountUsername is provided, use it; otherwise use default priority
	var session auth.Session
	var cloudSkinURL, cloudCapeURL string

	if selectedAccountUsername != "" {
		// User selected a specific account
		logMessage(fmt.Sprintf("Выбран конкретный аккаунт: %s", selectedAccountUsername))

		// Check if it's a Microsoft account
		if auth.Store.MSA.RefreshToken != "" && auth.Store.Minecraft.Username == selectedAccountUsername {
			if sess, err := auth.Authenticate(); err == nil {
				session = sess
				logMessage(fmt.Sprintf("Использование выбранного аккаунта Microsoft: %s", session.Username))
			}
		}

		// Check if it's a Cloud game account
		if session.Username == "" {
			if cloudAcc := auth.GetDefaultCloudAccount(); cloudAcc != nil && cloudAcc.Token != "" {
				gas, _ := a.GetCloudGameAccounts()
				for _, ga := range gas {
					if ga.Username == selectedAccountUsername {
						// Use ServerUUID for --uuid so Minecraft server sees the correct UUID (inventory sync)
						launchUUID := ga.ServerUUID
						if launchUUID == "" {
							launchUUID = ga.UUID
						}
						session = auth.Session{
							Username:    ga.Username,
							AccessToken: "",
							UUID:        launchUUID,
						}
						cloudSkinURL = ga.SkinURL
						cloudCapeURL = ga.CapeURL
						logMessage(fmt.Sprintf("Использование выбранного аккаунта QMServer Cloud: %s", session.Username))
						break
					}
				}
			}
		}

		// Check if it's a local account
		if session.Username == "" {
			for _, acc := range auth.LocalStore.Accounts {
				if acc.Name == selectedAccountUsername {
					session = auth.Session{
						Username:    acc.Name,
						AccessToken: "",
						UUID:        acc.UUID,
					}
					logMessage(fmt.Sprintf("Использование выбранного локального аккаунта: %s", session.Username))
					break
				}
			}
		}

		if session.Username == "" {
			return fmt.Errorf("выбранный аккаунт '%s' не найден", selectedAccountUsername)
		}
	} else {
		// No specific account selected, use default priority: Microsoft > Cloud > Local

		// 1. Microsoft account (if exists and active)
		if auth.Store.MSA.RefreshToken != "" && auth.Store.Minecraft.Username != "" {
			if sess, err := auth.Authenticate(); err == nil {
				session = sess
				logMessage(fmt.Sprintf("Использование аккаунта Microsoft: %s", session.Username))
			}
		}

		// 2. QMServer Cloud account (if no Microsoft session)
		if session.Username == "" {
			if cloudAcc := auth.GetDefaultCloudAccount(); cloudAcc != nil && cloudAcc.Token != "" {
				gas, _ := a.GetCloudGameAccounts()
				if len(gas) > 0 {
					ga := gas[0]
					// Use ServerUUID for --uuid so Minecraft server sees the correct UUID (inventory sync)
					launchUUID := ga.ServerUUID
					if launchUUID == "" {
						launchUUID = ga.UUID
					}
					session = auth.Session{
						Username:    ga.Username,
						AccessToken: "",
						UUID:        launchUUID,
					}
					cloudSkinURL = ga.SkinURL
					cloudCapeURL = ga.CapeURL
					logMessage(fmt.Sprintf("Использование аккаунта QMServer Cloud: %s", session.Username))
				}
			}
		}

		// 3. Local account (if no Microsoft/Cloud)
		if session.Username == "" {
			defaultAccountName := auth.LocalStore.GetDefaultAccount()
			if defaultAccountName == "" && len(auth.LocalStore.Accounts) > 0 {
				defaultAccountName = auth.LocalStore.Accounts[0].Name
				auth.LocalStore.SetDefaultAccount(defaultAccountName)
			}
			if defaultAccountName == "" {
				return fmt.Errorf("нет доступных аккаунтов, создайте аккаунт в разделе 'Аккаунты'")
			}
			var account *auth.LocalAccount
			for _, acc := range auth.LocalStore.Accounts {
				if acc.Name == defaultAccountName {
					account = &acc
					break
				}
			}
			if account == nil {
				// DefaultAccount may point to removed/synced account — clear invalid default
				auth.LocalStore.ClearDefaultIfInvalid()
				return fmt.Errorf("аккаунт '%s' не найден. Выберите аккаунт в разделе 'Аккаунты'", defaultAccountName)
			}
			session = auth.Session{
				Username:    account.Name,
				AccessToken: "",
				UUID:        account.UUID,
			}
			logMessage(fmt.Sprintf("Использование локального аккаунта: %s", session.Username))
		}
	}

	// Prepare launch options - use full instance config like CLI does
	options := launcher.LaunchOptions{
		Session:        session,
		InstanceConfig: inst.Config, // Use full instance config like CLI
		// Используем javaw на Windows (NoJavaWindow=true), чтобы не открывалось отдельное консольное окно
		NoJavaWindow:       true,
		Demo:               false, // Default values like CLI
		DisableMultiplayer: false,
		DisableChat:        false,
		SkinURL:            cloudSkinURL,
		CapeURL:            cloudCapeURL,
	}

	// Set server for auto-connect if specified
	if serverAddress != "" {
		options.QuickPlayServer = serverAddress
		logMessage(fmt.Sprintf("Автоматическое подключение к серверу: %s", serverAddress))
	}

	logMessage(fmt.Sprintf("Подготовка опций запуска для пользователя: %s", session.Username))

	// Sync config and options.txt from QMServer when user requested (checkbox in account picker)
	// Sync only to the selected account's directory (per-account isolation)
	if syncConfigFromServer && serverID > 0 {
		logMessage(fmt.Sprintf("Запрошена синхронизация конфигурации с сервера (serverID=%d) для аккаунта %s", serverID, session.Username))
		runtime.EventsEmit(a.ctx, "launch-progress", map[string]interface{}{
			"type":    "sync-start",
			"message": "Синхронизация конфигурации с QMServer Cloud...",
		})
		if err := syncConfigFromQMServer(inst, serverID, session.UUID); err != nil {
			logMessage(fmt.Sprintf("Ошибка синхронизации конфигурации: %v", err))
			runtime.EventsEmit(a.ctx, "launch-progress", map[string]interface{}{
				"type":    "sync-error",
				"message": fmt.Sprintf("Ошибка синхронизации конфигурации: %v", err),
			})
		} else {
			logMessage("Синхронизация конфигурации с QMServer Cloud завершена")
			runtime.EventsEmit(a.ctx, "launch-progress", map[string]interface{}{
				"type":    "sync-complete",
				"message": "Синхронизация конфигурации завершена",
			})
		}
	}

	// Sync files with QMServer Cloud if this instance uses it (full manifest sync, e.g. mods)
	logMessage(fmt.Sprintf("Проверка условий синхронизации: IsUsingQMServerCloud=%v, QMServerHost='%s', serverID=%d",
		inst.Config.IsUsingQMServerCloud, inst.Config.QMServerHost, serverID))
	logMessage(fmt.Sprintf("Инстанс %s: IsUsingQMServerCloud=%v", inst.Name, inst.Config.IsUsingQMServerCloud))
	if inst.Config.IsUsingQMServerCloud && inst.Config.QMServerHost != "" && serverID > 0 {
		logMessage(fmt.Sprintf("Обнаружена QMServer Cloud конфигурация для инстанса %s", inst.Name))
		logMessage(fmt.Sprintf("QMServer: %s:%d, ServerID: %d", inst.Config.QMServerHost, inst.Config.QMServerPort, serverID))

		// Upload local skins/capes to QMServer before sync (for distribution to other users)
		if cloudAcc := auth.GetDefaultCloudAccount(); cloudAcc != nil && cloudAcc.Token != "" {
			_ = network.UploadLocalSkinsToQMServer(inst.Dir(), inst.Config.QMServerHost, inst.Config.QMServerPort, cloudAcc.Token, logMessage)
		}

		runtime.EventsEmit(a.ctx, "launch-progress", map[string]interface{}{
			"type":    "sync-start",
			"message": "Синхронизация с QMServer Cloud...",
		})

		var disabledMods []string
		if disabledModsJSON != "" {
			_ = json.Unmarshal([]byte(disabledModsJSON), &disabledMods)
		}
		emitSync := func(phase, msg, file string, pct float64) {
			runtime.EventsEmit(a.ctx, "launch-progress", map[string]interface{}{
				"type":        "sync-progress",
				"phase":       phase,
				"message":     msg,
				"currentFile": file,
				"progress":    pct,
			})
		}
		if err := syncQMServerFiles(inst, serverID, disabledMods, emitSync); err != nil {
			logMessage(fmt.Sprintf("Ошибка синхронизации с QMServer Cloud: %v", err))
			runtime.EventsEmit(a.ctx, "launch-progress", map[string]interface{}{
				"type":    "sync-error",
				"message": fmt.Sprintf("Ошибка синхронизации: %v", err),
			})
		} else {
			logMessage("Синхронизация с QMServer Cloud завершена успешно")
			// Обновить servers.dat — добавить/обновить сервер для быстрого доступа в Minecraft
			if serverAddress != "" {
				name := serverName
				if name == "" {
					name = serverAddress
				}
				if err := serversdat.UpdateOrAddServer(inst.Dir(), name, serverAddress); err != nil {
					logMessage(fmt.Sprintf("Не удалось обновить servers.dat: %v", err))
				} else {
					logMessage("servers.dat обновлён")
				}
			}
			runtime.EventsEmit(a.ctx, "launch-progress", map[string]interface{}{
				"type":    "sync-complete",
				"message": "Синхронизация завершена",
			})
		}
	}

	logMessage("Начало загрузки Minecraft и компонентов")

	runtime.EventsEmit(a.ctx, "launch-progress", map[string]interface{}{
		"type":    "prepare-start",
		"message": "Начало загрузки Minecraft и компонентов",
	})

	// Prepare launch environment
	// Create progress watcher for GUI - sends events to frontend
	watcher := func(event any) {
		switch e := event.(type) {
		case launcher.DownloadingEvent:
			if e.Total > 0 {
				progress := float64(e.Completed) / float64(e.Total) * 100
				logMessage(fmt.Sprintf("Загрузка: %d/%d (%.1f%%)", e.Completed, e.Total, progress))
				// Send progress event to frontend
				runtime.EventsEmit(a.ctx, "launch-progress", map[string]interface{}{
					"type":      "downloading",
					"completed": e.Completed,
					"total":     e.Total,
					"progress":  progress,
					"message":   fmt.Sprintf("Загрузка: %d/%d (%.1f%%)", e.Completed, e.Total, progress),
				})
				if e.Completed >= e.Total {
					logMessage("Загрузка Minecraft завершена")
					runtime.EventsEmit(a.ctx, "launch-progress", map[string]interface{}{
						"type":    "downloading-complete",
						"message": "Загрузка завершена",
					})
				}
			}
		case launcher.AssetsResolvedEvent:
			logMessage(fmt.Sprintf("Ассеты обработаны: %d", e.Total))
			runtime.EventsEmit(a.ctx, "launch-progress", map[string]interface{}{
				"type":    "assets-resolved",
				"total":   e.Total,
				"message": fmt.Sprintf("Ассеты обработаны: %d", e.Total),
			})
		case launcher.LibrariesResolvedEvent:
			logMessage(fmt.Sprintf("Библиотеки обработаны: %d", e.Total))
			runtime.EventsEmit(a.ctx, "launch-progress", map[string]interface{}{
				"type":    "libraries-resolved",
				"total":   e.Total,
				"message": fmt.Sprintf("Библиотеки обработаны: %d", e.Total),
			})
		case launcher.MetadataResolvedEvent:
			logMessage("Метаданные Minecraft разрешены")
			runtime.EventsEmit(a.ctx, "launch-progress", map[string]interface{}{
				"type":    "metadata-resolved",
				"message": "Метаданные Minecraft разрешены",
			})
		case launcher.PostProcessingEvent:
			logMessage("Начата пост-обработка (Forge/Minecraft)")
			runtime.EventsEmit(a.ctx, "launch-progress", map[string]interface{}{
				"type":    "post-processing",
				"message": "Пост-обработка (Forge/Minecraft)",
			})
		}
	}
	launchEnv, err := launcher.Prepare(inst, options, watcher)
	if err != nil {
		logMessage(fmt.Sprintf("Ошибка подготовки инстанса: %v", err))
		return fmt.Errorf("failed to prepare instance: %w", err)
	}

	logMessage("Подготовка завершена успешно")

	// Apply selected resource packs to options.txt so they are enabled automatically in-game
	var rpOrder []string
	if enabledResourcepacksOrderJSON != "" {
		_ = json.Unmarshal([]byte(enabledResourcepacksOrderJSON), &rpOrder)
	}
	if err := launcher.ApplyResourcePacksToOptions(launchEnv.GameDir, rpOrder); err != nil {
		logMessage(fmt.Sprintf("[ResourcePacks] Не удалось обновить options.txt: %v", err))
	}

	runtime.EventsEmit(a.ctx, "launch-progress", map[string]interface{}{
		"type":    "prepare-complete",
		"message": "Подготовка завершена, запуск Minecraft...",
	})

	// Launch the instance in background (don't wait for completion)
	logMessage("Запуск Minecraft...")
	err = launcher.Launch(launchEnv, func(cmd *exec.Cmd) error {
		return cmd.Start() // Start in background, don't wait
	})

	if err != nil {
		logMessage(fmt.Sprintf("Ошибка запуска: %v", err))
		runtime.EventsEmit(a.ctx, "launch-error", map[string]interface{}{
			"error": fmt.Sprintf("Ошибка запуска: %v", err),
		})
		return fmt.Errorf("failed to launch instance: %w", err)
	}

	logMessage("Minecraft запущен успешно")
	logMessage(fmt.Sprintf("=== Завершение запуска инстанса: %s ===", inst.Name))
	return nil
}

// EnsureInstanceForServer creates or gets instance for server - exact copy of TUI logic
func (a *App) EnsureInstanceForServer(serverName string, serverAddress string, serverVersion string, serverModLoader string, serverModLoaderVersion string, serverID uint) string {
	if err := network.CheckServerProfileConnectAllowed(serverID); err != nil {
		if errors.Is(err, network.ErrServerProfileDisabled) {
			return "Error: " + i18n.Translate("ui.server.disabled_error")
		}
	}

	// Parse mod loader info (format: "loader version" or just "loader") - exact copy of TUI
	var loaderType launcher.Loader
	var loaderVersion string

	// Combine loader and version like TUI does (modLoaderInfo format)
	modLoaderInfo := serverModLoader
	if serverModLoaderVersion != "" {
		modLoaderInfo = serverModLoader + " " + serverModLoaderVersion
	}

	parts := strings.Fields(modLoaderInfo)
	if len(parts) >= 1 {
		loaderStr := strings.ToLower(parts[0])
		switch loaderStr {
		case "fabric":
			loaderType = launcher.LoaderFabric
		case "forge":
			loaderType = launcher.LoaderForge
		case "neoforge":
			loaderType = launcher.LoaderNeoForge
		case "quilt":
			loaderType = launcher.LoaderQuilt
		default:
			loaderType = launcher.LoaderVanilla
		}

		if len(parts) >= 2 {
			loaderVersion = parts[1]
		}
	} else {
		loaderType = launcher.LoaderVanilla
	}

	// Instance name will be sanitized automatically by CreateInstance
	instanceName := launcher.SanitizeInstanceName(serverName)

	// Validate input data
	if serverVersion == "" {
		return "Error: Empty game version"
	}

	// Create instance automatically with server parameters
	// Use default config values like CLI does
	defaultConfig := launcher.InstanceConfig{
		WindowResolution: struct {
			Width  int `toml:"width" json:"width"`
			Height int `toml:"height" json:"height"`
		}{
			Width:  1708,
			Height: 960,
		},
		MinMemory: 4096,
		MaxMemory: 4096,
		// QMServer Cloud configuration
		IsUsingQMServerCloud: true,
		QMServerHost:         defaultQMServerHost,
		QMServerPort:         defaultQMServerPort,
	}

	// For Forge, the version should be in format "gameVersion-loaderVersion" - exact copy of TUI
	finalLoaderVersion := loaderVersion
	if loaderType == launcher.LoaderForge && loaderVersion != "" {
		finalLoaderVersion = serverVersion + "-" + loaderVersion
		logMessage(fmt.Sprintf("Формируем полную версию Forge: %s", finalLoaderVersion))
	}

	options := launcher.InstanceOptions{
		Name:          instanceName,
		GameVersion:   serverVersion,
		Loader:        loaderType,
		LoaderVersion: finalLoaderVersion,
		Config:        defaultConfig,
	}

	// First, try to find existing instance - exact copy of TUI
	logMessage("Проверяем существование инстанса")
	inst, err := launcher.FetchInstance(instanceName)
	if err != nil {
		logMessage(fmt.Sprintf("Инстанс не найден, создаём новый: %v", err))
		runtime.EventsEmit(a.ctx, "launch-progress", map[string]interface{}{
			"type":    "instance-creating",
			"message": fmt.Sprintf("Создание инстанса '%s'...", instanceName),
		})
		// Instance doesn't exist, create it
		logMessage("Вызов launcher.CreateInstance")
		inst, err = launcher.CreateInstance(options)
		if err != nil {
			logMessage(fmt.Sprintf("Ошибка создания инстанса: %v", err))
			runtime.EventsEmit(a.ctx, "launch-error", map[string]interface{}{
				"error": fmt.Sprintf("Ошибка создания инстанса: %v", err),
			})
			return fmt.Sprintf("Error creating instance: %v", err)
		}
		logMessage(fmt.Sprintf("Инстанс '%s' создан успешно", instanceName))
		runtime.EventsEmit(a.ctx, "launch-progress", map[string]interface{}{
			"type":    "instance-created",
			"message": fmt.Sprintf("Инстанс '%s' создан успешно", instanceName),
		})
	} else {
		logMessage(fmt.Sprintf("Найден существующий инстанс: %s", instanceName))
		runtime.EventsEmit(a.ctx, "launch-progress", map[string]interface{}{
			"type":    "instance-found",
			"message": fmt.Sprintf("Найден существующий инстанс '%s'", instanceName),
		})
		// Update existing instance config with QMServer settings if needed
		config := inst.Config
		configNeedsUpdate := false

		if !config.IsUsingQMServerCloud {
			config.IsUsingQMServerCloud = true
			configNeedsUpdate = true
			logMessage("Обновление конфигурации: включена поддержка QMServer Cloud")
		}

		if config.QMServerHost != defaultQMServerHost {
			config.QMServerHost = defaultQMServerHost
			configNeedsUpdate = true
			logMessage("Обновление конфигурации: установлен QMServerHost")
		}

		if config.QMServerPort != defaultQMServerPort {
			config.QMServerPort = defaultQMServerPort
			configNeedsUpdate = true
			logMessage("Обновление конфигурации: установлен QMServerPort")
		}

		// Update loader, loader version, and game version to match server profile if needed
		loaderNeedsUpdate := false

		if inst.GameVersion != serverVersion && serverVersion != "" {
			logMessage(fmt.Sprintf("Обновление версии игры: %s -> %s", inst.GameVersion, serverVersion))
			inst.GameVersion = serverVersion
			loaderNeedsUpdate = true
		}

		if inst.Loader != loaderType {
			logMessage(fmt.Sprintf("Обновление типа загрузчика: %s -> %s", inst.Loader, loaderType))
			inst.Loader = loaderType
			loaderNeedsUpdate = true
		}

		// Если сервер передал конкретную версию загрузчика – синхронизируем её
		if finalLoaderVersion != "" && inst.LoaderVersion != finalLoaderVersion {
			logMessage(fmt.Sprintf("Обновление версии загрузчика: %s -> %s", inst.LoaderVersion, finalLoaderVersion))
			inst.LoaderVersion = finalLoaderVersion
			loaderNeedsUpdate = true
		}

		// Apply updates if anything changed
		if configNeedsUpdate {
			inst.Config = config
		}

		if configNeedsUpdate || loaderNeedsUpdate {
			if err := inst.WriteConfig(); err != nil {
				logMessage(fmt.Sprintf("Ошибка сохранения обновленной конфигурации инстанса: %v", err))
			} else {
				logMessage("Конфигурация инстанса успешно обновлена для QMServer Cloud и загрузчика")
			}
		}
	}

	return instanceName
}

// initLogging initializes logging to a centralized logs directory - exact copy of TUI
func initLogging(instanceName string) error {
	logMessage("Определение домашней директории")

	// Use centralized logs directory - cross-platform home directory detection
	var homeDir string
	if home := os.Getenv("HOME"); home != "" {
		homeDir = home
		logMessage(fmt.Sprintf("Используется HOME: %s", homeDir))
	} else if home := os.Getenv("USERPROFILE"); home != "" {
		homeDir = home // Windows
		logMessage(fmt.Sprintf("Используется USERPROFILE: %s", homeDir))
	} else {
		// Fallback - try to get home directory
		if h, err := os.UserHomeDir(); err == nil {
			homeDir = h
			logMessage(fmt.Sprintf("Используется UserHomeDir: %s", homeDir))
		} else {
			logMessage(fmt.Sprintf("Не удалось определить домашнюю директорию: %v", err))
			return fmt.Errorf("cannot determine home directory")
		}
	}

	logsDir := filepath.Join(homeDir, ".qmlauncher", "logs")
	logMessage(fmt.Sprintf("Директория для логов: %s", logsDir))

	// Create logs directory if it doesn't exist
	logMessage("Создание директории для логов")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		logMessage(fmt.Sprintf("Ошибка создания директории логов: %v", err))
		return fmt.Errorf("failed to create logs directory: %w", err)
	}
	logMessage(fmt.Sprintf("Директория создана: %s", logsDir))

	// Create log filename with instance name and timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFilename := fmt.Sprintf("qmlauncher_%s_%s.log", instanceName, timestamp)
	logPath := filepath.Join(logsDir, logFilename)

	logMessage(fmt.Sprintf("Создание файла логов: %s", logFilename))

	var err error
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logMessage(fmt.Sprintf("Ошибка создания файла логов: %v", err))
		return fmt.Errorf("failed to open log file: %w", err)
	}
	logMessage("Файл логов создан успешно")

	// Set log output to file
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	logMessage(fmt.Sprintf("Инстанс: %s", instanceName))
	return nil
}

// Translate returns translated string for the given key
func (a *App) Translate(key string) string {
	return i18n.Translate(key)
}

// SetLang sets the application language (ru, en). Persists to ~/.qmlauncher/settings.json. Call Translate to refresh UI.
func (a *App) SetLang(langTag string) {
	switch langTag {
	case "en", "en-US", "en-GB":
		i18n.SetLang(language.English)
	case "ru", "ru-RU":
		i18n.SetLang(language.Russian)
	default:
		i18n.SetLang(language.Russian)
	}
	// Persist language to settings file
	if homeDir, err := os.UserHomeDir(); err == nil {
		settingsPath := filepath.Join(homeDir, ".qmlauncher", "settings.json")
		os.MkdirAll(filepath.Dir(settingsPath), 0755)
		var existing map[string]interface{}
		if data, err := os.ReadFile(settingsPath); err == nil {
			json.Unmarshal(data, &existing)
		}
		if existing == nil {
			existing = make(map[string]interface{})
		}
		existing["language"] = i18n.GetLang()
		if data, err := json.MarshalIndent(existing, "", "  "); err == nil {
			os.WriteFile(settingsPath, data, 0644)
		}
	}
}

// GetLang returns the current language tag ("ru" or "en").
func (a *App) GetLang() string {
	return i18n.GetLang()
}

// GetLauncherVersion returns semver with a "v" prefix for the window title and header (e.g. v1.0.10).
func (a *App) GetLauncherVersion() string {
	return "v" + version
}

// LauncherAboutInfo describes the running binary for the About dialog.
type LauncherAboutInfo struct {
	Version string `json:"version"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

// GetLauncherAboutInfo returns version and platform for the About dialog.
func (a *App) GetLauncherAboutInfo() LauncherAboutInfo {
	return LauncherAboutInfo{
		Version: "v" + version,
		OS:      goruntime.GOOS,
		Arch:    goruntime.GOARCH,
	}
}

// CheckLauncherUpdateAvailable reports whether an update is available (QMServer distribution, GitHub binary, or QMWeb).
func (a *App) CheckLauncherUpdateAvailable() bool {
	return updater.QMServerDistributionUpdateAvailable(nil) ||
		updater.GitHubBinaryUpdateAvailable() ||
		updater.CheckForQMWebUpdate(nil)
}

// LauncherAPITargetSettings is read/written via ~/.qmlauncher/settings.json (use_qmserver_cloud, custom_api_base).
type LauncherAPITargetSettings struct {
	UseQMServerCloud bool   `json:"use_qmserver_cloud"`
	CustomAPIBase    string `json:"custom_api_base"`
	EffectiveAPIBase string `json:"effective_api_base"`
}

// GetLauncherAPITarget returns persisted API target fields and the effective base URL.
func (a *App) GetLauncherAPITarget() LauncherAPITargetSettings {
	cfg := readLauncherSettingsMap()
	useCloud := true
	custom := ""
	if cfg != nil {
		if v, ok := cfg["use_qmserver_cloud"]; ok {
			useCloud = parseBoolish(v, true)
		}
		if s, ok := cfg["custom_api_base"].(string); ok {
			custom = s
		}
	}
	return LauncherAPITargetSettings{
		UseQMServerCloud: useCloud,
		CustomAPIBase:    custom,
		EffectiveAPIBase: network.EffectiveQMServerAPIBase(),
	}
}

// SetLauncherAPITarget persists API target and refreshes in-memory base. Returns empty string on success.
func (a *App) SetLauncherAPITarget(useCloud bool, customBase string) string {
	customBase = strings.TrimSpace(customBase)
	if !useCloud && customBase == "" {
		return "Укажите базовый URL API (например https://example.com/api/v1) или включите QMServer Cloud."
	}
	path, err := launcherSettingsPath()
	if err != nil {
		return err.Error()
	}
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	var existing map[string]interface{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &existing)
	}
	if existing == nil {
		existing = make(map[string]interface{})
	}
	existing["use_qmserver_cloud"] = useCloud
	existing["custom_api_base"] = customBase
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err.Error()
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err.Error()
	}
	network.ApplyLauncherAPITarget(useCloud, customBase)
	return ""
}

// CatalogStoreSettings controls CurseForge / Modrinth availability in the resource catalog UI and API.
type CatalogStoreSettings struct {
	CurseforgeEnabled bool `json:"curseforge_enabled"`
	ModrinthEnabled   bool `json:"modrinth_enabled"`
}

// GetCatalogStoreSettings returns catalog_curseforge_enabled and catalog_modrinth_enabled (default true if absent).
func (a *App) GetCatalogStoreSettings() CatalogStoreSettings {
	cfg := readLauncherSettingsMap()
	cf := true
	mr := true
	if cfg != nil {
		cf = parseBoolish(cfg["catalog_curseforge_enabled"], true)
		mr = parseBoolish(cfg["catalog_modrinth_enabled"], true)
	}
	return CatalogStoreSettings{CurseforgeEnabled: cf, ModrinthEnabled: mr}
}

// SetCatalogStoreSettings persists catalog toggles and emits catalog-store-settings-changed.
func (a *App) SetCatalogStoreSettings(curseforgeEnabled, modrinthEnabled bool) string {
	path, err := launcherSettingsPath()
	if err != nil {
		return err.Error()
	}
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	var existing map[string]interface{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &existing)
	}
	if existing == nil {
		existing = make(map[string]interface{})
	}
	existing["catalog_curseforge_enabled"] = curseforgeEnabled
	existing["catalog_modrinth_enabled"] = modrinthEnabled
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err.Error()
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err.Error()
	}
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "catalog-store-settings-changed", nil)
	}
	return ""
}

// CurseForgeKeySettings describes persisted CurseForge API key state for the UI (no key material exposed).
type CurseForgeKeySettings struct {
	HasEffectiveKey bool   `json:"has_effective_key"`
	KeySavedInFile  bool   `json:"key_saved_in_file"`
	UseMyKeyDefault bool   `json:"use_my_key_default"`
	EffectiveSource string `json:"effective_source"`
}

// GetCurseForgeKeySettings returns whether a key is available for API calls and the source label only (no key preview).
func (a *App) GetCurseForgeKeySettings() CurseForgeKeySettings {
	cfg := readLauncherSettingsMap()
	saved := false
	useMy := false
	if cfg != nil {
		if s, ok := cfg["curseforge_api_key"].(string); ok && meta.NormalizeCurseForgeAPIKey(s) != "" {
			saved = true
		}
		useMy = parseBoolish(cfg["curseforge_use_my_key_default"], false)
	}
	effective := strings.TrimSpace(meta.CurseForgeAPIKey())
	return CurseForgeKeySettings{
		HasEffectiveKey: effective != "",
		KeySavedInFile:  saved,
		UseMyKeyDefault: useMy,
		EffectiveSource: curseForgeEffectiveKeySource(),
	}
}

// HasCurseForgeAPIKey reports whether CurseForge Core API requests can be authenticated.
func (a *App) HasCurseForgeAPIKey() bool {
	return strings.TrimSpace(meta.CurseForgeAPIKey()) != ""
}

// SetCurseForgeSettingsKey persists curseforge_api_key and curseforge_use_my_key_default in ~/.qmlauncher/settings.json.
// If clearKey is true, the stored API key is removed. If clearKey is false and apiKey is empty, the previous key is kept (only the useMyKeyDefault / prefs update).
func (a *App) SetCurseForgeSettingsKey(apiKey string, useMyKeyDefault bool, clearKey bool) string {
	apiKey = meta.NormalizeCurseForgeAPIKey(apiKey)
	path, err := launcherSettingsPath()
	if err != nil {
		return err.Error()
	}
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	var existing map[string]interface{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &existing)
	}
	if existing == nil {
		existing = make(map[string]interface{})
	}
	if clearKey {
		delete(existing, "curseforge_api_key")
	} else if apiKey != "" {
		existing["curseforge_api_key"] = apiKey
	}
	existing["curseforge_use_my_key_default"] = useMyKeyDefault
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err.Error()
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err.Error()
	}
	clearCurseForgeCloudKeyCache()
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "curseforge-settings-changed", nil)
	}
	return ""
}

// AccountInfo represents account information for frontend
type AccountInfo struct {
	Type          string `json:"type"` // "microsoft", "local", "cloud", "cloud_game"
	Username      string `json:"username"`
	Status        string `json:"status"` // "active", "expired", "none"
	IsDefault     bool   `json:"isDefault"`
	SkinModel     string `json:"skinModel"`     // "steve" or "alex" for local; empty for microsoft (use SkinUuid)
	SkinUuid      string `json:"skinUuid"`      // Mojang UUID for microsoft (to fetch skin); empty for local
	SkinURL       string `json:"skinUrl"`       // Custom skin URL (Ely.by, etc.) for cloud/local
	Email         string `json:"email"`         // For cloud accounts - used for logout
	GameAccountID uint   `json:"gameAccountId"` // ID of cloud_game account (for deletion)
}

// CloudGameAccountInfo represents a game account on QMServer Cloud (for skin editing)
type CloudGameAccountInfo struct {
	ID         uint   `json:"id"`
	Username   string `json:"username"`
	UUID       string `json:"uuid"`       // Internal game account UUID
	ServerUUID string `json:"serverUuid"` // UUID that Minecraft server sees (for --uuid at launch)
	SkinModel  string `json:"skinModel"`
	SkinURL    string `json:"skinUrl"`
	CapeURL    string `json:"capeUrl"`
	MojangUUID string `json:"mojangUuid"` // For mc-heads avatar when skin_url empty
}

// GetAccounts returns list of all accounts (Microsoft and local)
func (a *App) GetAccounts() []AccountInfo {
	accounts := make([]AccountInfo, 0)

	// Check Microsoft account
	if auth.Store.MSA.RefreshToken != "" {
		status := "expired"
		if auth.Store.Minecraft.Username != "" && auth.Store.Minecraft.Expires.After(time.Now()) {
			status = "active"
		}
		accounts = append(accounts, AccountInfo{
			Type:      "microsoft",
			Username:  auth.Store.Minecraft.Username,
			Status:    status,
			IsDefault: true, // Microsoft account is always default if exists
			SkinUuid:  auth.Store.Minecraft.UUID,
		})
	}

	// Add local accounts
	for _, localAccount := range auth.LocalStore.Accounts {
		isDefault := localAccount.Name == auth.LocalStore.DefaultAccount
		skinModel := localAccount.SkinModel
		if skinModel != auth.SkinModelAlex {
			skinModel = auth.SkinModelSteve
		}
		accounts = append(accounts, AccountInfo{
			Type:      "local",
			Username:  localAccount.Name,
			Status:    "active",
			IsDefault: isDefault,
			SkinModel: skinModel,
		})
	}

	// Add QMServer Cloud accounts
	// Display only real game account usernames (Minecraft nicks). Exclude QMWeb profile and Ely.by
	// — even if they appear in game_accounts, they are auth/identity, not Minecraft accounts.
	cloudStore, _ := auth.ReadCloudStore()
	var gameAccountUsername, gameAccountSkinURL, gameAccountSkinUuid string
	if defCloud := auth.GetDefaultCloudAccount(); defCloud != nil && defCloud.Token != "" {
		elyLinked, profileUser, elyUser, elyUserRaw := a.getCloudProfileNonGameUsernames()
		exclude := map[string]bool{}
		if profileUser != "" {
			exclude[profileUser] = true
		}
		if elyUser != "" {
			exclude[elyUser] = true
		}
		if gas, _ := a.GetCloudGameAccounts(); len(gas) > 0 {
			for _, ga := range gas {
				u := strings.TrimSpace(strings.ToLower(ga.Username))
				if u != "" && !exclude[u] {
					gameAccountUsername = ga.Username
					gameAccountSkinURL = ga.SkinURL
					gameAccountSkinUuid = ga.MojangUUID
					// Fallback: if no skin_url but user has Ely.by linked, use Ely.by skin
					if gameAccountSkinURL == "" && elyLinked && elyUserRaw != "" {
						gameAccountSkinURL = "https://skinsystem.ely.by/skins/" + elyUserRaw + ".png"
					}
					break
				}
			}
		}
	}
	for _, cloud := range cloudStore.Accounts {
		if cloud.Token == "" {
			continue
		}
		isDefault := auth.GetDefaultCloudAccount() != nil && auth.GetDefaultCloudAccount().Email == cloud.Email
		displayName := cloud.Email // fallback: email, not profile/Ely.by username
		skinURL := ""
		skinUuid := ""
		if isDefault && gameAccountUsername != "" {
			displayName = gameAccountUsername
			skinURL = gameAccountSkinURL
			skinUuid = gameAccountSkinUuid
		}
		accounts = append(accounts, AccountInfo{
			Type:      "cloud",
			Username:  displayName,
			Status:    "active",
			IsDefault: isDefault,
			SkinModel: auth.SkinModelSteve,
			SkinURL:   skinURL,
			SkinUuid:  skinUuid,
			Email:     cloud.Email,
		})
	}

	// Add QMServer Cloud game accounts (separate entries for deletion and management)
	if defCloud := auth.GetDefaultCloudAccount(); defCloud != nil && defCloud.Token != "" {
		elyLinked, profileUser, elyUser, _ := a.getCloudProfileNonGameUsernames()
		exclude := map[string]bool{}
		if profileUser != "" {
			exclude[profileUser] = true
		}
		if elyUser != "" {
			exclude[elyUser] = true
		}
		if gas, _ := a.GetCloudGameAccounts(); len(gas) > 0 {
			for _, ga := range gas {
				u := strings.TrimSpace(strings.ToLower(ga.Username))
				if u != "" && !exclude[u] {
					// Use Ely.by skin as fallback
					skinURL := ga.SkinURL
					if skinURL == "" && elyLinked && ga.MojangUUID != "" {
						// Try to get Ely.by username from profile
						if _, _, elyUsername, elyUsernameRaw := a.getCloudProfileNonGameUsernames(); elyUsername != "" {
							skinURL = "https://skinsystem.ely.by/skins/" + elyUsernameRaw + ".png"
						}
					}
					// Link UUID for UI: prefer Mojang profile id; else server/Minecraft UUID (matches Microsoft session when mojang_uuid was not stored)
					linkUUID := strings.TrimSpace(ga.MojangUUID)
					if linkUUID == "" {
						linkUUID = strings.TrimSpace(ga.ServerUUID)
					}
					accounts = append(accounts, AccountInfo{
						Type:          "cloud_game",
						Username:      ga.Username,
						Status:        "active",
						IsDefault:     false,
						SkinModel:     ga.SkinModel,
						SkinURL:       skinURL,
						SkinUuid:      linkUUID,
						GameAccountID: ga.ID,
					})
				}
			}
		}
	}

	return accounts
}

// LoginAccount logs in to Microsoft account
func (a *App) LoginAccount(noBrowser bool) string {
	if network.IsMSAAuthDisabledByQMServer() {
		return "Microsoft-вход отключён в настройках QMAdmin."
	}
	if id := network.GetQMLauncherMSAClientID(); id != "" {
		auth.ClientID = id
	} else {
		return "Microsoft/Mojang вход недоступен."
	}

	// Try to authenticate with existing session first
	session, err := auth.Authenticate()
	if err == nil {
		return fmt.Sprintf(i18n.Translate("login.complete"), session.Username)
	}

	// If no existing session, start new authentication
	if noBrowser {
		// Device code flow
		resp, err := auth.FetchDeviceCode()
		if err != nil {
			return fmt.Sprintf("Error fetching device code: %v", err)
		}
		// Return device code info for user to enter
		return fmt.Sprintf(i18n.Translate("login.code"), resp.UserCode, resp.VerificationURI)
	} else {
		// Browser flow - this will block, so we need to handle it differently
		// For now, return URL for user to open
		url := auth.AuthCodeURL()
		return fmt.Sprintf(i18n.Translate("login.url"), url.String())
	}
}

// LogoutAccount logs out from Microsoft account
func (a *App) LogoutAccount() string {
	err := auth.Store.Clear()
	if err != nil {
		return fmt.Sprintf("Error logging out: %v", err)
	}
	return i18n.Translate("logout.complete")
}

// OpenBrowserForMicrosoft opens browser to Microsoft OAuth for login. Starts a local callback server
// and emits "microsoft-auth-success" or "microsoft-auth-error" when done.
func (a *App) OpenBrowserForMicrosoft() string {
	if network.IsMSAAuthDisabledByQMServer() {
		msg := "Microsoft-вход отключён в настройках QMAdmin (Настройки → Microsoft авторизация)."
		logMessage(fmt.Sprintf("[MicrosoftAuth] %s", msg))
		emitMicrosoftAuthError(a.ctx, msg)
		return "error"
	}
	// Use fixed redirect URI registered in Azure AD (must match exactly)
	var err error
	auth.RedirectURI, err = url.Parse(auth.DefaultRedirectURI)
	if err != nil {
		logMessage(fmt.Sprintf("[MicrosoftAuth] Failed to parse redirect URI: %v", err))
		emitMicrosoftAuthError(a.ctx, "Не удалось создать redirect URI")
		return "error"
	}

	id := network.GetQMLauncherMSAClientID()
	if id == "" {
		msg := "Microsoft/Mojang auth is not available (disabled in QMAdmin: Microsoft authorization)."
		logMessage(fmt.Sprintf("[MicrosoftAuth] %s", msg))
		emitMicrosoftAuthError(a.ctx, msg)
		return "error"
	}
	auth.ClientID = id
	msa := network.FetchMSAServerSettings()
	switch {
	case msa.FetchOK && msa.ClientID != "":
		logMessage("[MicrosoftAuth] Using MSA Client ID from QMServer")
	case strings.TrimSpace(os.Getenv("QMLAUNCHER_MSA_CLIENT_ID")) != "":
		logMessage("[MicrosoftAuth] Using MSA Client ID from QMLAUNCHER_MSA_CLIENT_ID")
	default:
		logMessage("[MicrosoftAuth] Using built-in QMLauncher MSA Client ID")
	}

	// Start listener on port 8000 (matches redirect URI)
	listener, err := net.Listen("tcp", "127.0.0.1:8000")
	if err != nil {
		logMessage(fmt.Sprintf("[MicrosoftAuth] Failed to start callback server (port 8000 may be in use): %v", err))
		emitMicrosoftAuthError(a.ctx, "Порт 8000 занят. Закройте другие приложения или перезапустите лаунчер.")
		return "error"
	}

	// Get Microsoft OAuth URL
	authURL := auth.AuthCodeURL()

	// Handle callback in goroutine
	go func() {
		defer listener.Close()
		server := &http.Server{ReadHeaderTimeout: 5 * time.Second}
		server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/signin" {
				http.NotFound(w, r)
				return
			}
			code := r.URL.Query().Get("code")
			errorParam := r.URL.Query().Get("error")
			errorDesc := r.URL.Query().Get("error_description")

			if errorParam != "" {
				logMessage(fmt.Sprintf("[MicrosoftAuth] OAuth error: %s - %s", errorParam, errorDesc))
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(microsoftAuthCallbackHTML("Ошибка авторизации", errorDesc, "Окно можно закрыть.", true)))
				emitMicrosoftAuthError(a.ctx, errorDesc)
				go server.Shutdown(context.Background())
				return
			}

			if code == "" {
				logMessage("[MicrosoftAuth] No code in callback")
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(microsoftAuthCallbackHTML("Ошибка", "Код авторизации не получен.", "Окно можно закрыть.", true)))
				emitMicrosoftAuthError(a.ctx, "Код авторизации не получен")
				go server.Shutdown(context.Background())
				return
			}

			// Exchange code for tokens
			resp, err := auth.ExchangeAuthCode(code)
			if err != nil {
				logMessage(fmt.Sprintf("[MicrosoftAuth] Failed to authenticate with code: %v", err))
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(microsoftAuthCallbackHTML("Ошибка", "Не удалось завершить авторизацию.", "Окно можно закрыть.", true)))
				emitMicrosoftAuthError(a.ctx, fmt.Sprintf("Не удалось завершить авторизацию: %v", err))
				go server.Shutdown(context.Background())
				return
			}

			// Success
			logMessage(fmt.Sprintf("[MicrosoftAuth] Successfully authenticated as %s", resp.Username))
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(microsoftAuthCallbackHTML("Успешно!", fmt.Sprintf("Вход выполнен как %s", resp.Username), "Окно можно закрыть.", false)))
			runtime.EventsEmit(a.ctx, "microsoft-auth-success", nil)
			go server.Shutdown(context.Background())
		})
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logMessage(fmt.Sprintf("[MicrosoftAuth] Server error: %v", err))
		}
	}()

	// Open browser (platform-specific). Use Start() to not block.
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", authURL.String())
	case "darwin":
		cmd = exec.Command("open", authURL.String())
	default:
		cmd = exec.Command("xdg-open", authURL.String())
	}
	if err := cmd.Start(); err != nil {
		logMessage(fmt.Sprintf("[MicrosoftAuth] Failed to open browser: %v, URL: %s", err, authURL.String()))
		emitMicrosoftAuthError(a.ctx, "Не удалось открыть браузер. Скопируйте ссылку: "+authURL.String())
		return authURL.String()
	}
	logMessage(fmt.Sprintf("[MicrosoftAuth] Opened browser with URL: %s", authURL.String()))
	return authURL.String()
}

func emitMicrosoftAuthError(ctx context.Context, msg string) {
	if ctx != nil {
		runtime.EventsEmit(ctx, "microsoft-auth-error", msg)
	}
}

// microsoftAuthCallbackHTML returns HTML for the Microsoft OAuth callback page
func microsoftAuthCallbackHTML(title, message, footer string, isError bool) string {
	accentColor := "#3c8527"
	if isError {
		accentColor = "#d4441a"
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link href="https://fonts.googleapis.com/css2?family=Noto+Sans:wght@400;500;600;700&display=swap" rel="stylesheet">
  <title>Microsoft Auth</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: 'Noto Sans', ui-sans-serif, system-ui, sans-serif;
      background: #1d1e1e;
      color: #f1edec;
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 24px;
    }
    .card {
      background: #313131;
      border: 1px solid rgba(255,255,255,0.1);
      border-radius: 0;
      padding: 32px;
      max-width: 400px;
      text-align: center;
    }
    h1 {
      font-family: 'Noto Sans', sans-serif;
      font-size: 1.5rem;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 1px;
      color: %s;
      margin-bottom: 16px;
    }
    p { font-size: 0.95rem; line-height: 1.6; color: #f1edec; }
    .message { margin-bottom: 12px; }
    .footer { font-size: 0.85rem; color: #949494; }
  </style>
</head>
<body>
  <div class="card">
    <h1>%s</h1>
    <p class="message">%s</p>
    <p class="footer">%s</p>
  </div>
</body>
</html>`, accentColor, title, message, footer)
}

// OpenBrowserForQMServerCloud opens browser to QMWeb for login/register. Starts a local callback server
// and emits "cloud-auth-success" or "cloud-auth-error" when done.
func (a *App) OpenBrowserForQMServerCloud() string {
	// Start listener on random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		logMessage(fmt.Sprintf("[CloudAuth] Failed to start callback server: %v", err))
		emitCloudAuthError(a.ctx, "Не удалось запустить сервер")
		return "error"
	}
	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	// QMWeb URL - use production or configurable
	qmWebBase := "https://web.qx-dev.ru"
	if base := os.Getenv("QMWEB_URL"); base != "" {
		qmWebBase = strings.TrimSuffix(base, "/")
	}
	authURL := fmt.Sprintf("%s/auth?callback=%s", qmWebBase, url.QueryEscape(callbackURL))

	// Handle callback in goroutine
	go func() {
		defer listener.Close()
		server := &http.Server{ReadHeaderTimeout: 5 * time.Second}
		server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}
			token := r.URL.Query().Get("token")
			email := r.URL.Query().Get("email")
			username := r.URL.Query().Get("username")
			if token == "" {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(cloudAuthCallbackHTML("Ошибка", "Токен не получен.", "Окно можно закрыть.", true)))
				emitCloudAuthError(a.ctx, "Токен не получен")
				go server.Shutdown(context.Background())
				return
			}
			if err := auth.AddCloudAccount(token, email, username); err != nil {
				logMessage(fmt.Sprintf("[CloudAuth] Failed to save: %v", err))
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(cloudAuthCallbackHTML("Ошибка сохранения", "Не удалось сохранить аккаунт.", "Окно можно закрыть.", true)))
				emitCloudAuthError(a.ctx, "Ошибка сохранения")
			} else {
				clearCurseForgeCloudKeyCache()
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write([]byte(cloudAuthCallbackHTML("Успешно!", "Аккаунт добавлен в QMLauncher.", "Окно можно закрыть.", false)))
				runtime.EventsEmit(a.ctx, "cloud-auth-success", nil)
			}
			go server.Shutdown(context.Background())
		})
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logMessage(fmt.Sprintf("[CloudAuth] Server error: %v", err))
		}
	}()

	// Open browser (platform-specific). Use Start() to not block.
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", authURL)
	case "darwin":
		cmd = exec.Command("open", authURL)
	default:
		cmd = exec.Command("xdg-open", authURL)
	}
	if err := cmd.Start(); err != nil {
		logMessage(fmt.Sprintf("[CloudAuth] Failed to open browser: %v, URL: %s", err, authURL))
		emitCloudAuthError(a.ctx, "Не удалось открыть браузер. Скопируйте ссылку: "+authURL)
		return authURL
	}
	return authURL
}

func emitCloudAuthError(ctx context.Context, msg string) {
	if ctx != nil {
		runtime.EventsEmit(ctx, "cloud-auth-error", msg)
	}
}

// cloudAuthCallbackHTML returns HTML for the callback page in QMWeb style (dark theme, Minecraft fonts).
func cloudAuthCallbackHTML(title, message, footer string, isError bool) string {
	accentColor := "#3c8527"
	if isError {
		accentColor = "#d4441a"
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link href="https://fonts.googleapis.com/css2?family=Noto+Sans:wght@400;500;600;700&display=swap" rel="stylesheet">
  <title>QMServer Cloud</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: 'Noto Sans', ui-sans-serif, system-ui, sans-serif;
      background: #1d1e1e;
      color: #f1edec;
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 24px;
    }
    .card {
      background: #313131;
      border: 1px solid rgba(255,255,255,0.1);
      border-radius: 0;
      padding: 32px;
      max-width: 400px;
      text-align: center;
    }
    h1 {
      font-family: 'Noto Sans', sans-serif;
      font-size: 1.5rem;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 1px;
      color: %s;
      margin-bottom: 16px;
    }
    p { font-size: 0.95rem; line-height: 1.6; color: #f1edec; }
    .message { margin-bottom: 12px; }
    .footer { font-size: 0.85rem; color: #949494; }
  </style>
</head>
<body>
  <div class="card">
    <h1>%s</h1>
    <p class="message">%s</p>
    <p class="footer">%s</p>
  </div>
</body>
</html>`, accentColor, title, message, footer)
}

// LogoutCloudAccount removes the default cloud account
func (a *App) LogoutCloudAccount(email string) string {
	if err := auth.RemoveCloudAccount(email); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	clearCurseForgeCloudKeyCache()
	return "Cloud account removed"
}

// SyncLocalAccountToCloud syncs a local account to QMServer Cloud (creates game account, removes local).
// The local account's UUID and username are always preserved for player data continuity.
// skinURL: optional custom skin URL (Ely.by, custom); empty = no custom skin
func (a *App) SyncLocalAccountToCloud(localAccountName string, skinURL string) string {
	acc := auth.GetLocalAccountByName(localAccountName)
	if acc == nil {
		return "Local account not found"
	}
	if strings.TrimSpace(acc.Name) == "" {
		return "Local account has no name"
	}
	// Ensure UUID is set (required for sync — preserves player dirs, skins, etc.)
	localUUID := auth.EnsureLocalAccountUUID(acc)
	if localUUID == "" {
		return "Local account has no UUID"
	}
	cloudAcc := auth.GetDefaultCloudAccount()
	if cloudAcc == nil || cloudAcc.Token == "" {
		return "No QMServer Cloud account. Log in first."
	}
	apiBase := network.EffectiveQMServerAPIBase()
	bodyMap := map[string]interface{}{
		"username":   acc.Name,
		"skin_model": acc.SkinModel,
		"local_uuid": localUUID,
	}
	if skinURL != "" {
		bodyMap["skin_url"] = skinURL
	}
	body, _ := json.Marshal(bodyMap)
	req, err := http.NewRequest("POST", apiBase+"/game-accounts/sync", bytes.NewReader(body))
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cloudAcc.Token)
	resp, err := network.QMServerHTTPClient.Do(req)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Sprintf("API error %d: %s", resp.StatusCode, string(b))
	}
	auth.LocalStore.RemoveLocalAccount(localAccountName)
	// Do NOT remove players/<uuid> — preserve saves, config, options.txt for continuity
	// Update cloud account display name to synced game account so it appears correctly in the list
	_ = auth.UpdateDefaultCloudAccountUsername(acc.Name)
	// Emit event so frontend refreshes accounts list without restart
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "cloud-auth-success", nil)
	}
	return fmt.Sprintf("Account '%s' synced to QMServer Cloud", localAccountName)
}

// SyncMicrosoftAccountToCloud registers the current Microsoft/Mojang Minecraft profile as a QMServer Cloud game account
// (same family as POST /game-accounts/sync with mojang_uuid). Does not remove the local Microsoft session.
func (a *App) SyncMicrosoftAccountToCloud() string {
	cloudAcc := auth.GetDefaultCloudAccount()
	if cloudAcc == nil || cloudAcc.Token == "" {
		return "No QMServer Cloud account. Log in under Connected accounts first."
	}
	if strings.TrimSpace(auth.Store.MSA.RefreshToken) == "" {
		return "No Microsoft sign-in. Add a Mojang/Microsoft profile first."
	}
	if _, err := auth.Authenticate(); err != nil {
		return fmt.Sprintf("Minecraft session unavailable: %v", err)
	}
	u := strings.TrimSpace(auth.Store.Minecraft.Username)
	mjUUID := strings.TrimSpace(auth.Store.Minecraft.UUID)
	if u == "" || mjUUID == "" {
		return "No Minecraft profile (license). Sign in with Microsoft."
	}
	apiBase := network.EffectiveQMServerAPIBase()
	bodyMap := map[string]interface{}{
		"username":    u,
		"skin_model":  auth.SkinModelSteve,
		"local_uuid":  uuid.New().String(),
		"server_uuid": mjUUID,
		"mojang_uuid": mjUUID,
	}
	body, _ := json.Marshal(bodyMap)
	req, err := http.NewRequest("POST", apiBase+"/game-accounts/sync", bytes.NewReader(body))
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cloudAcc.Token)
	resp, err := network.QMServerHTTPClient.Do(req)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("API error %d: %s", resp.StatusCode, string(b))
	}
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "cloud-auth-success", nil)
	}
	return fmt.Sprintf("Minecraft profile '%s' linked to QMServer Cloud", u)
}

// NewsItem represents a news entry for QMWeb/QMLauncher
type NewsItem struct {
	ID        uint   `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// GetNews returns news from QMServer (public API)
func (a *App) GetNews() ([]NewsItem, error) {
	apiBase := network.EffectiveQMServerAPIBase()
	req, err := http.NewRequest("GET", apiBase+"/news?limit=10", nil)
	if err != nil {
		return nil, err
	}
	resp, err := network.QMServerHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d", resp.StatusCode)
	}
	var data struct {
		News []struct {
			ID        uint   `json:"id"`
			Title     string `json:"title"`
			Content   string `json:"content"`
			CreatedAt string `json:"created_at"`
		} `json:"news"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	result := make([]NewsItem, 0, len(data.News))
	for _, n := range data.News {
		result = append(result, NewsItem{
			ID:        n.ID,
			Title:     n.Title,
			Content:   n.Content,
			CreatedAt: n.CreatedAt,
		})
	}
	return result, nil
}

// GetSkinProviderConfig returns enabled skin providers from QMServer (public API)
func (a *App) GetSkinProviderConfig() (map[string]bool, error) {
	apiBase := network.EffectiveQMServerAPIBase()
	req, err := http.NewRequest("GET", apiBase+"/settings/skin-providers", nil)
	if err != nil {
		return nil, err
	}
	resp, err := network.QMServerHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d", resp.StatusCode)
	}
	var out map[string]bool
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]bool{"ely_by": true}
	}
	return out, nil
}

// GetCloudElyLinked returns whether the current QMServer Cloud user has Ely.by linked
func (a *App) GetCloudElyLinked() (bool, error) {
	linked, _, _, _ := a.getCloudProfileNonGameUsernames()
	return linked, nil
}

// getCloudProfileNonGameUsernames returns profile username and Ely.by username — these must NOT
// be displayed as game accounts (auth/identity, not Minecraft accounts).
// elyByUsernameRaw is the original Ely.by username for URL construction (case-sensitive).
func (a *App) getCloudProfileNonGameUsernames() (elyLinked bool, profileUsername, elyByUsername string, elyByUsernameRaw string) {
	cloudAcc := auth.GetDefaultCloudAccount()
	if cloudAcc == nil || cloudAcc.Token == "" {
		return false, "", "", ""
	}
	apiBase := network.EffectiveQMServerAPIBase()
	req, err := http.NewRequest("GET", apiBase+"/auth/me", nil)
	if err != nil {
		return false, "", "", ""
	}
	req.Header.Set("Authorization", "Bearer "+cloudAcc.Token)
	resp, err := network.QMServerHTTPClient.Do(req)
	if err != nil {
		return false, "", "", ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, "", "", ""
	}
	var data struct {
		Username      string `json:"username"`
		ElyByUsername string `json:"ely_by_username"`
		ElyByLinked   bool   `json:"ely_by_linked"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return false, "", "", ""
	}
	profileUsername = strings.TrimSpace(strings.ToLower(data.Username))
	elyByUsername = strings.TrimSpace(strings.ToLower(data.ElyByUsername))
	elyByUsernameRaw = strings.TrimSpace(data.ElyByUsername)
	return data.ElyByLinked, profileUsername, elyByUsername, elyByUsernameRaw
}

// GetCloudGameAccounts returns game accounts for the current QMServer Cloud user (for skin editing)
func (a *App) GetCloudGameAccounts() ([]CloudGameAccountInfo, string) {
	cloudAcc := auth.GetDefaultCloudAccount()
	if cloudAcc == nil || cloudAcc.Token == "" {
		return nil, "No QMServer Cloud account. Log in first."
	}
	apiBase := network.EffectiveQMServerAPIBase()
	req, err := http.NewRequest("GET", apiBase+"/game-accounts", nil)
	if err != nil {
		return nil, fmt.Sprintf("Error: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+cloudAcc.Token)
	resp, err := network.QMServerHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Sprintf("Error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Sprintf("API error %d: %s", resp.StatusCode, string(b))
	}
	var data struct {
		GameAccounts []struct {
			ID         uint    `json:"id"`
			Username   string  `json:"username"`
			UUID       string  `json:"uuid"`
			ServerUUID *string `json:"server_uuid"`
			SkinModel  string  `json:"skin_model"`
			SkinURL    *string `json:"skin_url"`
			CapeURL    *string `json:"cape_url"`
			MojangUUID *string `json:"mojang_uuid"`
		} `json:"game_accounts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Sprintf("Parse error: %v", err)
	}
	result := make([]CloudGameAccountInfo, 0, len(data.GameAccounts))
	for _, ga := range data.GameAccounts {
		skinURL := ""
		if ga.SkinURL != nil {
			skinURL = *ga.SkinURL
		}
		capeURL := ""
		if ga.CapeURL != nil {
			capeURL = *ga.CapeURL
		}
		mojangUUID := ""
		if ga.MojangUUID != nil {
			mojangUUID = *ga.MojangUUID
		}
		serverUUID := ""
		if ga.ServerUUID != nil {
			serverUUID = *ga.ServerUUID
		}
		result = append(result, CloudGameAccountInfo{
			ID:         ga.ID,
			Username:   ga.Username,
			UUID:       ga.UUID,
			ServerUUID: serverUUID,
			SkinModel:  ga.SkinModel,
			SkinURL:    skinURL,
			CapeURL:    capeURL,
			MojangUUID: mojangUUID,
		})
	}
	return result, ""
}

// UpdateCloudGameAccount updates skin for a QMServer Cloud game account
func (a *App) UpdateCloudGameAccount(gameAccountID uint, skinURL string, skinModel string) string {
	cloudAcc := auth.GetDefaultCloudAccount()
	if cloudAcc == nil || cloudAcc.Token == "" {
		return "No QMServer Cloud account. Log in first."
	}
	apiBase := network.EffectiveQMServerAPIBase()
	bodyMap := map[string]interface{}{}
	if skinModel != "" {
		bodyMap["skin_model"] = skinModel
	}
	if skinURL == "" {
		bodyMap["skin_url"] = nil
	} else {
		bodyMap["skin_url"] = skinURL
	}
	body, _ := json.Marshal(bodyMap)
	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/game-accounts/%d", apiBase, gameAccountID), bytes.NewReader(body))
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cloudAcc.Token)
	resp, err := network.QMServerHTTPClient.Do(req)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Sprintf("API error %d: %s", resp.StatusCode, string(b))
	}
	return "Skin updated"
}

// DeleteCloudGameAccount deletes a QMServer Cloud game account
func (a *App) DeleteCloudGameAccount(gameAccountID uint) string {
	cloudAcc := auth.GetDefaultCloudAccount()
	if cloudAcc == nil || cloudAcc.Token == "" {
		return "No QMServer Cloud account. Log in first."
	}
	apiBase := network.EffectiveQMServerAPIBase()
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/game-accounts/%d", apiBase, gameAccountID), nil)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cloudAcc.Token)
	resp, err := network.QMServerHTTPClient.Do(req)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Sprintf("API error %d: %s", resp.StatusCode, string(b))
	}
	// Emit event so frontend refreshes accounts list without restart
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "cloud-auth-success", nil)
	}
	return "Game account deleted"
}

// InventorySlot represents a single inventory slot
type InventorySlot struct {
	Slot  int    `json:"slot"`
	Item  string `json:"item"`
	Count int    `json:"count"`
	NBT   string `json:"nbt,omitempty"`
}

// InventoryEntry represents inventory from one server
type InventoryEntry struct {
	ServerID      string          `json:"server_id"`
	ServerName    string          `json:"server_name,omitempty"`
	ServerVersion string          `json:"server_version,omitempty"`
	PlayerName    string          `json:"player_name"`
	Timestamp     int64           `json:"timestamp"`
	Main          []InventorySlot `json:"main"`
	Armor         []InventorySlot `json:"armor"`
	Offhand       *InventorySlot  `json:"offhand,omitempty"`
}

// GameAccountInventoryResponse is the response for GetGameAccountInventory
type GameAccountInventoryResponse struct {
	Inventories []InventoryEntry `json:"inventories"`
	Error       string           `json:"error,omitempty"`
}

// GetGameAccountInventory fetches inventory for a QMServer Cloud game account.
// Requires Cloud account to be logged in. Returns inventories and optional error message.
func (a *App) GetGameAccountInventory(gameAccountID uint) GameAccountInventoryResponse {
	cloudAcc := auth.GetDefaultCloudAccount()
	if cloudAcc == nil || cloudAcc.Token == "" {
		return GameAccountInventoryResponse{Error: "No QMServer Cloud account. Log in first."}
	}
	apiBase := network.EffectiveQMServerAPIBase()
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/game-accounts/%d/inventory", apiBase, gameAccountID), nil)
	if err != nil {
		return GameAccountInventoryResponse{Error: fmt.Sprintf("Error: %v", err)}
	}
	req.Header.Set("Authorization", "Bearer "+cloudAcc.Token)
	resp, err := network.QMServerHTTPClient.Do(req)
	if err != nil {
		return GameAccountInventoryResponse{Error: fmt.Sprintf("Error: %v", err)}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return GameAccountInventoryResponse{Error: fmt.Sprintf("API error %d: %s", resp.StatusCode, string(b))}
	}
	var data struct {
		Inventories []struct {
			ServerID      string          `json:"server_id"`
			ServerName    string          `json:"server_name"`
			ServerVersion string          `json:"server_version"`
			PlayerName    string          `json:"player_name"`
			Timestamp     int64           `json:"timestamp"`
			Main          []InventorySlot `json:"main"`
			Armor         []InventorySlot `json:"armor"`
			Offhand       *InventorySlot  `json:"offhand,omitempty"`
		} `json:"inventories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return GameAccountInventoryResponse{Error: fmt.Sprintf("Parse error: %v", err)}
	}
	result := make([]InventoryEntry, 0, len(data.Inventories))
	for _, inv := range data.Inventories {
		result = append(result, InventoryEntry{
			ServerID:      inv.ServerID,
			ServerName:    inv.ServerName,
			ServerVersion: inv.ServerVersion,
			PlayerName:    inv.PlayerName,
			Timestamp:     inv.Timestamp,
			Main:          inv.Main,
			Armor:         inv.Armor,
			Offhand:       inv.Offhand,
		})
	}
	return GameAccountInventoryResponse{Inventories: result}
}

// CreateLocalAccount creates a new local account.
// skinModel: "steve" (male/classic) or "alex" (female/slim) - determines default skin in game
func (a *App) CreateLocalAccount(name string, skinModel string) string {
	if name == "" {
		return "Account name cannot be empty"
	}
	if skinModel != auth.SkinModelAlex {
		skinModel = auth.SkinModelSteve
	}
	auth.AddLocalAccount(name, skinModel)
	return fmt.Sprintf("Local account '%s' created", name)
}

// CreateCloudGameAccount creates a new QMServer Cloud game account.
// username: Minecraft nickname (required)
// skinModel: "steve" (male/classic) or "alex" (female/slim) - determines default skin in game
func (a *App) CreateCloudGameAccount(username string, skinModel string) string {
	if username == "" {
		return "Username cannot be empty"
	}
	if skinModel != auth.SkinModelAlex {
		skinModel = auth.SkinModelSteve
	}
	cloudAcc := auth.GetDefaultCloudAccount()
	if cloudAcc == nil || cloudAcc.Token == "" {
		return "No QMServer Cloud account. Log in first."
	}
	apiBase := network.EffectiveQMServerAPIBase()
	bodyMap := map[string]interface{}{
		"username":   username,
		"skin_model": skinModel,
	}
	body, _ := json.Marshal(bodyMap)
	req, err := http.NewRequest("POST", apiBase+"/game-accounts", bytes.NewReader(body))
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cloudAcc.Token)
	resp, err := network.QMServerHTTPClient.Do(req)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Sprintf("API error %d: %s", resp.StatusCode, string(b))
	}
	// Emit event so frontend refreshes accounts list without restart
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "cloud-auth-success", nil)
	}
	return fmt.Sprintf("Game account '%s' created", username)
}

// DeleteLocalAccount deletes a local account and removes its player dirs from all instances.
// Symlinks (mods, kubejs, etc.) are removed, not followed — shared files stay intact.
func (a *App) DeleteLocalAccount(name string) string {
	acc := auth.GetLocalAccountByName(name)
	auth.LocalStore.RemoveLocalAccount(name)
	if acc != nil && acc.UUID != "" {
		launcher.RemovePlayerDirsForAccount(acc.UUID)
	}
	return fmt.Sprintf("Local account '%s' deleted", name)
}

// SetDefaultAccount sets the default local account
func (a *App) SetDefaultAccount(name string) string {
	auth.LocalStore.SetDefaultAccount(name)
	return fmt.Sprintf("Default account set to '%s'", name)
}

// GetCloudProfile fetches the QMServer Cloud user profile (avatar_url, etc.) for the default cloud account
func (a *App) GetCloudProfile() map[string]interface{} {
	result := make(map[string]interface{})
	cloudAcc := auth.GetDefaultCloudAccount()
	if cloudAcc == nil || cloudAcc.Token == "" {
		return result
	}
	apiBase := network.EffectiveQMServerAPIBase()
	req, err := http.NewRequest("GET", apiBase+"/auth/me?token="+url.QueryEscape(cloudAcc.Token), nil)
	if err != nil {
		return result
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := network.QMServerHTTPClient.Do(req)
	if err != nil {
		return result
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return result
	}
	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return result
	}
	if v, ok := data["avatar_url"]; ok && v != nil {
		if s, ok := v.(string); ok && s != "" {
			result["avatar_url"] = s
		}
	}
	if v, ok := data["is_premium"]; ok {
		switch t := v.(type) {
		case bool:
			result["is_premium"] = t
		case float64:
			result["is_premium"] = t != 0
		}
	}
	result["email"] = data["email"]
	result["username"] = data["username"]
	return result
}

// GetCurrentAccount returns the current default account information
func (a *App) GetCurrentAccount() map[string]interface{} {
	result := make(map[string]interface{})

	// Check if Microsoft account exists and is active (Microsoft account is always default if exists)
	if auth.Store.MSA.RefreshToken != "" && auth.Store.Minecraft.Username != "" {
		if auth.Store.Minecraft.Expires.After(time.Now()) {
			result["name"] = auth.Store.Minecraft.Username
			result["email"] = strings.ToLower(auth.Store.Minecraft.Username) + "@qmlauncher.local"
			result["type"] = "microsoft"
			result["isDefault"] = true
			return result
		}
		// Even if expired, Microsoft account is still considered default
		if auth.Store.Minecraft.Username != "" {
			result["name"] = auth.Store.Minecraft.Username
			result["email"] = strings.ToLower(auth.Store.Minecraft.Username) + "@qmlauncher.local"
			result["type"] = "microsoft"
			result["isDefault"] = true
			return result
		}
	}

	// Check for default cloud account (takes precedence over local when set)
	// Display QMServer Cloud account (email), NOT game account username
	if cloudAcc := auth.GetDefaultCloudAccount(); cloudAcc != nil && cloudAcc.Token != "" {
		result["name"] = cloudAcc.Email
		result["email"] = cloudAcc.Email
		result["type"] = "cloud"
		result["isDefault"] = true
		return result
	}

	// Check for default local account — show "Локальный аккаунт" (local account, not game account)
	defaultAccountName := auth.LocalStore.GetDefaultAccount()
	if defaultAccountName != "" {
		result["name"] = "Локальный аккаунт"
		result["email"] = defaultAccountName
		result["type"] = "local"
		result["isDefault"] = true
		return result
	}

	// If no default account, return empty values
	result["name"] = ""
	result["email"] = ""
	result["type"] = ""
	result["isDefault"] = false
	return result
}

const defaultQMServerHost = "api.qx-dev.ru"
const defaultQMServerPort = 443

func getQMServerBaseURL(host string, port int) string {
	scheme := "http"
	if port == 443 {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, host, port)
}

// GetQMServerAPIBase returns the effective QMServer API base URL (cloud or custom; for proxy, etc.)
func (a *App) GetQMServerAPIBase() string {
	return network.EffectiveQMServerAPIBase()
}

// GetMicrosoftAuthAvailable returns true if Microsoft/Mojang login is allowed and a Client ID is available (server, env, or built-in default).
func (a *App) GetMicrosoftAuthAvailable() bool {
	if network.IsMSAAuthDisabledByQMServer() {
		return false
	}
	return network.GetQMLauncherMSAClientID() != ""
}

// syncConfigFromQMServer syncs only config/ folder and options.txt from QMServer Cloud.
// When accountUUID is set, syncs to the per-account directory (players/<uuid>/); otherwise to inst.Dir().
func syncConfigFromQMServer(inst launcher.Instance, serverID uint, accountUUID string) error {
	qmHost := inst.Config.QMServerHost
	qmPort := inst.Config.QMServerPort
	if qmHost == "" {
		qmHost = defaultQMServerHost
		qmPort = defaultQMServerPort
		logMessage(fmt.Sprintf("[SyncConfig] Using default QMServer: %s:%d", qmHost, qmPort))
	}
	logMessage(fmt.Sprintf("[SyncConfig] Downloading manifest for server ID: %d", serverID))
	manifest, err := downloadDataManifest(serverID, qmHost, qmPort)
	if err != nil {
		return fmt.Errorf("failed to download manifest: %w", err)
	}
	// Filter: only config/*, journeymap/* and options.txt
	var toSync []FileInfo
	for _, f := range manifest.Files {
		if f.Path == "options.txt" || strings.HasPrefix(f.Path, "config/") || strings.HasPrefix(f.Path, "journeymap/") {
			toSync = append(toSync, f)
		}
	}
	if len(toSync) == 0 {
		logMessage("[SyncConfig] No config/, journeymap/ or options.txt in server manifest, nothing to sync")
		return nil
	}
	logMessage(fmt.Sprintf("[SyncConfig] Found %d config/, journeymap/ or options.txt file(s) on server, syncing and overwriting", len(toSync)))
	targetDir, err := launcher.GetAccountGameDir(inst, accountUUID)
	if err != nil {
		return fmt.Errorf("get account game dir: %w", err)
	}
	logMessage(fmt.Sprintf("[SyncConfig] Target directory for sync: %s", targetDir))
	for _, fileInfo := range toSync {
		destPath := filepath.Join(targetDir, fileInfo.Path)
		logMessage(fmt.Sprintf("[SyncConfig] Downloading %s -> %s", fileInfo.Path, destPath))
		if err := downloadFile(serverID, fileInfo.Path, qmHost, qmPort, destPath); err != nil {
			logMessage(fmt.Sprintf("[SyncConfig] Error downloading %s: %v", fileInfo.Path, err))
			continue
		}
		logMessage(fmt.Sprintf("[SyncConfig] Synced: %s", fileInfo.Path))
	}
	return nil
}

// SyncProgressEmitter sends progress updates to frontend (nil = no-op)
type SyncProgressEmitter func(phase, message, currentFile string, progress float64)

// syncQMServerFiles synchronizes instance files with QMServer Cloud (like TUI does)
// disabledMods: mod paths to exclude from sync and remove from local instance (e.g. mods/sodium.jar)
// emitProgress: optional callback to send progress to UI (phase: checking|downloading|disabling, message, currentFile, progress 0-100)
func syncQMServerFiles(inst launcher.Instance, serverID uint, disabledMods []string, emitProgress SyncProgressEmitter) error {
	logMessage(fmt.Sprintf("[ConnectToServer] Starting file sync with QMServer Cloud for server ID: %d", serverID))

	// Get QMServer configuration from instance
	config := inst.Config
	if config.QMServerHost == "" {
		logMessage("[ConnectToServer] QMServerHost not configured, skipping sync")
		return nil
	}

	if serverID == 0 {
		logMessage("[ConnectToServer] ServerID not set, skipping sync")
		return nil
	}

	logMessage(fmt.Sprintf("[ConnectToServer] Connecting to QMServer: %s:%d", config.QMServerHost, config.QMServerPort))

	// Download data manifest
	logMessage(fmt.Sprintf("[ConnectToServer] Downloading data manifest for server ID: %d", serverID))
	manifest, err := downloadDataManifest(serverID, config.QMServerHost, config.QMServerPort)
	if err != nil {
		logMessage(fmt.Sprintf("[ConnectToServer] Error downloading manifest: %v", err))
		return fmt.Errorf("failed to download manifest: %w", err)
	}

	logMessage(fmt.Sprintf("[ConnectToServer] Manifest downloaded successfully, files in manifest: %d", len(manifest.Files)))

	// Create a map of files from manifest for quick lookup
	manifestFiles := make(map[string]FileInfo)
	for _, file := range manifest.Files {
		manifestFiles[file.Path] = file
	}

	instanceDir := inst.Dir()
	logMessage(fmt.Sprintf("[ConnectToServer] Instance directory: %s", instanceDir))

	// Build disabled set for quick lookup
	disabledSet := make(map[string]bool)
	for _, p := range disabledMods {
		disabledSet[filepath.ToSlash(strings.TrimPrefix(p, "/"))] = true
	}
	if len(disabledMods) > 0 {
		logMessage(fmt.Sprintf("[ConnectToServer] Disabled mods: %v", disabledMods))
	}

	// Count files to sync (for progress)
	var filesToSync []string
	for filePath := range manifestFiles {
		if filePath == "options.txt" || strings.HasPrefix(filePath, "config/") {
			continue
		}
		filesToSync = append(filesToSync, filePath)
	}
	totalFiles := len(filesToSync)

	// Remove orphaned files before syncing
	logMessage("[ConnectToServer] Checking for orphaned files")
	if err := removeOrphanedFiles(instanceDir, manifestFiles); err != nil {
		logMessage(fmt.Sprintf("[ConnectToServer] Error removing orphaned files: %v", err))
	} else {
		logMessage("[ConnectToServer] Orphaned files check completed")
	}

	// Sync files from manifest
	filesProcessed := 0
	filesDownloaded := 0
	filesSkipped := 0
	filesUpdated := 0
	processedForProgress := 0

	// Re-enable mods that are no longer disabled (rename .jar.disabled → .jar so we can sync)
	for modPath := range manifestFiles {
		if !strings.HasPrefix(modPath, "mods/") || disabledSet[modPath] {
			continue
		}
		disabledPath := modPath + ".disabled"
		localDisabled := filepath.Join(instanceDir, disabledPath)
		if _, err := os.Stat(localDisabled); err == nil {
			localJar := filepath.Join(instanceDir, modPath)
			if err := os.Rename(localDisabled, localJar); err == nil {
				logMessage(fmt.Sprintf("[ConnectToServer] Re-enabled mod for sync: %s", modPath))
			}
		}
	}

	for filePath, fileInfo := range manifestFiles {
		filesProcessed++
		instanceFilePath := filepath.Join(instanceDir, filePath)

		logMessage(fmt.Sprintf("[ConnectToServer] Processing file: %s", filePath))

		// By default do not sync config/ and options.txt
		if filePath == "options.txt" || strings.HasPrefix(filePath, "config/") {
			logMessage(fmt.Sprintf("[ConnectToServer] Skipping (sync only via config checkbox): %s", filePath))
			filesSkipped++
			continue
		}

		// Skip disabled paths (mods, resourcepacks, shaderpacks)
		if disabledSet[filePath] {
			logMessage(fmt.Sprintf("[ConnectToServer] Skipping (disabled by user): %s", filePath))
			filesSkipped++
			// Remove local file if it exists (user disabled, shouldn't keep stale copy)
			if strings.HasPrefix(filePath, "resourcepacks/") || strings.HasPrefix(filePath, "shaderpacks/") {
				if _, err := os.Stat(instanceFilePath); err == nil {
					if err := os.Remove(instanceFilePath); err == nil {
						logMessage(fmt.Sprintf("[ConnectToServer] Removed disabled: %s", filePath))
					}
				}
			}
			continue
		}

		processedForProgress++
		fileName := filepath.Base(filePath)
		pct := 0.0
		if totalFiles > 0 {
			pct = float64(processedForProgress-1) / float64(totalFiles) * 100
		}
		if emitProgress != nil {
			emitProgress("verifying", "Проверка: "+fileName, filePath, pct)
		}

		// Check if file exists and has matching MD5
		if _, err := os.Stat(instanceFilePath); err == nil {
			existingMD5, err := calculateFileMD5(instanceFilePath)
			if err != nil {
				logMessage(fmt.Sprintf("[ConnectToServer] Error calculating MD5 for file %s: %v", instanceFilePath, err))
				continue
			}
			if existingMD5 == fileInfo.MD5 {
				logMessage(fmt.Sprintf("[ConnectToServer] File unchanged, skipping: %s", filePath))
				filesSkipped++
				if emitProgress != nil && totalFiles > 0 {
					emitProgress("skipped", "Пропуск: "+fileName, filePath, float64(processedForProgress)/float64(totalFiles)*100)
				}
				continue
			}
			filesUpdated++
		} else {
			filesDownloaded++
		}

		if emitProgress != nil && totalFiles > 0 {
			emitProgress("downloading", "Скачивание: "+fileName, filePath, float64(processedForProgress-1)/float64(totalFiles)*100)
		}

		// Download file
		logMessage(fmt.Sprintf("[ConnectToServer] Downloading file: %s", filePath))
		if err := downloadFile(serverID, filePath, config.QMServerHost, config.QMServerPort, instanceFilePath); err != nil {
			logMessage(fmt.Sprintf("[ConnectToServer] Error downloading file %s: %v", filePath, err))
			continue
		}
		logMessage(fmt.Sprintf("[ConnectToServer] File downloaded successfully: %s", filePath))
	}

	// Disable mods by renaming .jar → .jar.disabled (Minecraft mod loaders skip .disabled files)
	disabledCount := 0
	for modPath := range disabledSet {
		if !strings.HasPrefix(modPath, "mods/") {
			continue
		}
		disabledCount++
		if emitProgress != nil {
			denom := len(disabledSet)
			if denom < 1 {
				denom = 1
			}
			emitProgress("disabling", "Отключение мода: "+filepath.Base(modPath), modPath, 95+float64(disabledCount)*5/float64(denom))
		}
		localPath := filepath.Join(instanceDir, modPath)
		disabledPath := localPath + ".disabled"
		if _, err := os.Stat(localPath); err == nil {
			_ = os.Remove(disabledPath) // Remove old .disabled if exists (we have fresh from sync)
			if err := os.Rename(localPath, disabledPath); err == nil {
				logMessage(fmt.Sprintf("[ConnectToServer] Disabled mod: %s → .disabled", modPath))
			}
		}
	}

	logMessage(fmt.Sprintf("[ConnectToServer] Sync completed: processed %d files, downloaded %d, updated %d, skipped %d",
		filesProcessed, filesDownloaded, filesUpdated, filesSkipped))

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

// FileInfo represents file information from QMServer manifest
type FileInfo struct {
	Path     string `json:"path"`
	MD5      string `json:"md5"`
	Size     int64  `json:"size"`
	Modified int64  `json:"modified"`
}

// DataManifest represents the data.json structure from QMServer
type DataManifest struct {
	ServerID   uint       `json:"server_id"`
	ServerUUID string     `json:"server_uuid"`
	Files      []FileInfo `json:"files"`
	Generated  int64      `json:"generated"`
}

// downloadDataManifest downloads data manifest from QMServer
func downloadDataManifest(serverID uint, qmServerHost string, qmServerPort int) (*DataManifest, error) {
	base := getQMServerBaseURL(qmServerHost, qmServerPort)
	url := fmt.Sprintf("%s/api/v1/check/data/%d", base, serverID)

	resp, err := network.QMServerHTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to QMServer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(network.ReadQMServerError(resp))
		if msg != "" {
			return nil, fmt.Errorf("QMServer does not serve QMLauncher: %s", msg)
		}
		return nil, fmt.Errorf("QMServer returned status %d", resp.StatusCode)
	}

	var manifest DataManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to parse data manifest: %w", err)
	}

	return &manifest, nil
}

// downloadFile downloads a file from QMServer
func downloadFile(serverID uint, filePath string, qmServerHost string, qmServerPort int, destPath string) error {
	base := getQMServerBaseURL(qmServerHost, qmServerPort)
	url := fmt.Sprintf("%s/api/v1/download/%d/%s", base, serverID, filePath)

	resp, err := network.QMServerHTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(network.ReadQMServerError(resp))
		if msg != "" {
			return fmt.Errorf("QMServer does not serve QMLauncher: %s", msg)
		}
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

	// Copy data
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// removeOrphanedFiles removes files and directories from mods/ that don't exist in server manifest
func removeOrphanedFiles(instanceDir string, manifestFiles map[string]FileInfo) error {
	logMessage("[ConnectToServer] Checking mods/ for orphaned files")

	modsDir := filepath.Join(instanceDir, "mods")

	// Check if mods directory exists
	if _, err := os.Stat(modsDir); os.IsNotExist(err) {
		logMessage("[ConnectToServer] mods/ directory does not exist - creating")
		if err := os.MkdirAll(modsDir, 0755); err != nil {
			logMessage(fmt.Sprintf("[ConnectToServer] Error creating mods/ directory: %v", err))
			return err
		}
		return nil
	}

	removedCount := 0
	checkedCount := 0

	// Walk through mods directory and check each file/folder
	err := filepath.Walk(modsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root mods directory itself
		if path == modsDir {
			return nil
		}

		// Get relative path from instance directory
		relPath, err := filepath.Rel(instanceDir, path)
		if err != nil {
			logMessage(fmt.Sprintf("[ConnectToServer] Error calculating path for %s: %v", path, err))
			return err
		}

		// Convert to forward slashes for consistency
		relPath = filepath.ToSlash(relPath)
		checkedCount++

		// Check if this file/directory exists in manifest (or is a .disabled variant of a manifest mod)
		_, exists := manifestFiles[relPath]
		if !exists && strings.HasSuffix(relPath, ".disabled") {
			basePath := strings.TrimSuffix(relPath, ".disabled")
			_, exists = manifestFiles[basePath]
		}
		if !exists {
			if info.IsDir() {
				logMessage(fmt.Sprintf("[ConnectToServer] Removing orphaned directory: %s", relPath))
				if err := os.RemoveAll(path); err != nil {
					logMessage(fmt.Sprintf("[ConnectToServer] Error removing directory %s: %v", relPath, err))
					return err
				}
				removedCount++
				return filepath.SkipDir // Skip walking into removed directory
			} else {
				logMessage(fmt.Sprintf("[ConnectToServer] Removing orphaned file: %s", relPath))
				if err := os.Remove(path); err != nil {
					logMessage(fmt.Sprintf("[ConnectToServer] Error removing file %s: %v", relPath, err))
					return err
				}
				removedCount++
			}
		}

		return nil
	})

	if err != nil {
		logMessage(fmt.Sprintf("[ConnectToServer] Error walking mods directory: %v", err))
		return err
	}

	logMessage(fmt.Sprintf("[ConnectToServer] Orphaned files check: checked %d items, removed %d", checkedCount, removedCount))
	return nil
}
