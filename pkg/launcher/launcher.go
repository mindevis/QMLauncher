// Package launcher provides the necessary functions to start the game.
package launcher

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"QMLauncher/internal/meta"
	"QMLauncher/internal/network"
	"QMLauncher/internal/version"
	env "QMLauncher/pkg"
	"QMLauncher/pkg/auth"
)

// Loader represents a game mod loader.
type Loader string

const (
	LoaderVanilla  Loader = "vanilla"
	LoaderFabric   Loader = "fabric"
	LoaderQuilt    Loader = "quilt"
	LoaderNeoForge Loader = "neoforge"
	LoaderForge    Loader = "forge"
)

// LaunchOptions represents configuration options when preparing an instance to be launched.
type LaunchOptions struct {
	Session auth.Session

	InstanceConfig
	QuickPlayServer    string
	QuickPlayWorld     string
	Demo               bool
	DisableMultiplayer bool
	DisableChat        bool
	NoJavaWindow       bool

	// SkinURL/CapeURL for CustomSkinLoader LocalSkin (Cloud accounts with Ely.by etc.)
	SkinURL string
	CapeURL string

	skipAssets    bool
	skipLibraries bool
}

// An EventWatcher is a controller that can handle multiple types of events.
type EventWatcher func(event any)

// MetadataResolvedEvent is called when all metadata has been retrieved
type MetadataResolvedEvent struct{}

// LibrariesResolvedEvent is called when all game libraries have been identified and filtered.
type LibrariesResolvedEvent struct {
	Total int
}

// AssetsResolvedEvent is called when all game assets have been identified and filtered.
type AssetsResolvedEvent struct {
	Total int
}

// DownloadingEvent is called when a download has progressed.
type DownloadingEvent struct {
	Completed int
	Total     int
}

// PostProcessingEvent is called when, usually Forge, pre-processing begins.
type PostProcessingEvent struct{}

// A Runner is a controller which manages the starting of the game.
type Runner func(cmd *exec.Cmd) error

// An ConsoleRunner is an implementation of Runner which logs game output to the console.
func ConsoleRunner(cmd *exec.Cmd) error {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// JavaVersion represents an installed Java version.
type JavaVersion struct {
	Name string
	Path string
}

// ListInstalledJavaVersions returns a list of all installed Java versions.
func ListInstalledJavaVersions() ([]JavaVersion, error) {
	entries, err := os.ReadDir(env.JavaDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var javas []JavaVersion
	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			path := filepath.Join(env.JavaDir, name)

			// Check if this directory contains a valid Java installation
			binDir := filepath.Join(path, "bin")
			if _, err := os.Stat(binDir); err == nil {
				javas = append(javas, JavaVersion{
					Name: name,
					Path: path,
				})
			}
		}
	}

	// Sort by name
	sort.Slice(javas, func(i, j int) bool {
		return strings.ToLower(javas[i].Name) < strings.ToLower(javas[j].Name)
	})

	return javas, nil
}

// A LaunchEnvironment represents the information needed to start the game.
type LaunchEnvironment struct {
	GameDir   string
	Java      string
	MainClass string
	Classpath []string
	JavaArgs  []string
	GameArgs  []string
}

// ApplyResourcePacksToOptions updates options.txt so that resource packs in the
// resourcepacks folder are automatically selected when the game starts.
// Preserves system/built-in packs (vanilla, fabric, mod_resources, etc.) and replaces
// file-based packs with the actual .zip files present in resourcepacks/.
// orderedPaths: optional, resourcepack paths (e.g. resourcepacks/Name.zip) in desired load order from QMAdmin load_order.
func ApplyResourcePacksToOptions(gameDir string, orderedPaths []string) error {
	rpDir := filepath.Join(gameDir, "resourcepacks")
	entries, err := os.ReadDir(rpDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	allFiles := make(map[string]bool)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".zip") {
			allFiles[name] = true
		}
	}
	var filePacks []string
	if len(orderedPaths) > 0 {
		seen := make(map[string]bool)
		for _, p := range orderedPaths {
			name := filepath.Base(p)
			if allFiles[name] && !seen[name] {
				filePacks = append(filePacks, "file/"+name)
				seen[name] = true
			}
		}
		for name := range allFiles {
			if !seen[name] {
				filePacks = append(filePacks, "file/"+name)
			}
		}
	} else {
		for name := range allFiles {
			filePacks = append(filePacks, "file/"+name)
		}
		sort.Strings(filePacks)
	}
	if len(filePacks) == 0 {
		return nil
	}
	optsPath := filepath.Join(gameDir, "options.txt")
	data, err := os.ReadFile(optsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	lines := strings.Split(string(data), "\n")
	var newLines []string
	modified := false
	for _, line := range lines {
		if !strings.HasPrefix(line, "resourcePacks:") {
			newLines = append(newLines, line)
			continue
		}
		val := strings.TrimPrefix(line, "resourcePacks:")
		val = strings.TrimSpace(val)
		var arr []string
		if err := json.Unmarshal([]byte(val), &arr); err != nil {
			newLines = append(newLines, line)
			continue
		}
		var systemPacks []string
		for _, s := range arr {
			if !strings.HasPrefix(s, "file/") {
				systemPacks = append(systemPacks, s)
			}
		}
		newArr := append(systemPacks, filePacks...)
		b, _ := json.Marshal(newArr)
		newLines = append(newLines, "resourcePacks:"+string(b))
		modified = true
	}
	if !modified {
		if !strings.Contains(string(data), "resourcePacks:") {
			b, _ := json.Marshal(append([]string{"vanilla"}, filePacks...))
			newLines = append(newLines, "resourcePacks:"+string(b))
			modified = true
		}
	}
	if !modified {
		return nil
	}
	return os.WriteFile(optsPath, []byte(strings.Join(newLines, "\n")), 0644)
}

// Launch starts a LaunchEnvironment with the specified runner.
//
// The Java executable is checked and the classpath and command arguments are finalized.
func Launch(launchEnv LaunchEnvironment, runner Runner) error {
	info, err := os.Stat(launchEnv.Java)
	if err != nil {
		return fmt.Errorf("Java executable does not exist") //nolint:staticcheck // error message capitalization
	}
	if info.IsDir() {
		return fmt.Errorf("Java binary is not executable") //nolint:staticcheck // error message capitalization
	}

	// On Windows, check for .exe extension
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(launchEnv.Java), ".exe") {
		return fmt.Errorf("Java binary is not executable") //nolint:staticcheck // error message capitalization
	}

	// On Unix systems, check execute permissions
	if runtime.GOOS != "windows" && info.Mode()&0111 == 0 {
		return fmt.Errorf("Java binary is not executable") //nolint:staticcheck // error message capitalization
	}

	javaArgs := append(launchEnv.JavaArgs, "-cp", strings.Join(launchEnv.Classpath, string(os.PathListSeparator)), launchEnv.MainClass)
	cmd := exec.Command(launchEnv.Java, append(javaArgs, launchEnv.GameArgs...)...)
	cmd.Dir = launchEnv.GameDir
	return runner(cmd)
}

// linkSharedDir creates a symlink or junction so playerDir/name points to baseDir/name.
// On Windows: tries os.Symlink first, then mklink /j (junction) as fallback.
func linkSharedDir(playerDir, baseDir, name string) error {
	linkPath := filepath.Join(playerDir, name)
	targetPath := filepath.Join(baseDir, name)
	if _, err := os.Stat(targetPath); err != nil {
		return err // directory doesn't exist
	}
	if err := os.Symlink(targetPath, linkPath); err == nil {
		return nil
	}
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "mklink", "/j", linkPath, targetPath)
		setCmdNoWindow(cmd)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("could not link %s directory", name)
}

// ensurePlayerGameDir creates a per-account game directory: config, saves, options.txt isolated;
// mods, kubejs, journeymap, etc. shared via symlink/junction.
func ensurePlayerGameDir(inst Instance, accountUUID string) (string, error) {
	baseDir := inst.Dir()
	playerDir := filepath.Join(baseDir, "players", accountUUID)

	if err := os.MkdirAll(playerDir, 0755); err != nil {
		return baseDir, err
	}

	// Link mods (required for modded instances)
	modsLink := filepath.Join(playerDir, "mods")
	if _, err := os.Stat(modsLink); os.IsNotExist(err) {
		if err := linkSharedDir(playerDir, baseDir, "mods"); err != nil {
			return baseDir, nil // fallback to shared dir
		}
	}

	// Link optional shared dirs (kubejs, shaderpacks, schematics, resourcepacks, CustomSkinLoader, journeymap)
	for _, name := range []string{"kubejs", "shaderpacks", "schematics", "resourcepacks", "CustomSkinLoader", "journeymap"} {
		linkPath := filepath.Join(playerDir, name)
		if _, err := os.Stat(linkPath); os.IsNotExist(err) {
			_ = linkSharedDir(playerDir, baseDir, name)
		}
	}

	for _, sub := range []string{"config", "saves"} {
		dir := filepath.Join(playerDir, sub)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return baseDir, err
		}
	}

	optsDest := filepath.Join(playerDir, "options.txt")
	if _, err := os.Stat(optsDest); os.IsNotExist(err) {
		optsSrc := filepath.Join(baseDir, "options.txt")
		if data, err := os.ReadFile(optsSrc); err == nil {
			_ = os.WriteFile(optsDest, data, 0644)
		} else {
			_ = os.WriteFile(optsDest, []byte("lang:ru_ru\n"), 0644)
		}
	}

	return playerDir, nil
}

// skinDownloadClient fetches skin/cape with proper User-Agent (some hosts block default Go client).
// Uses debug HTTP tracing when launcher_debug is enabled.
var skinDownloadClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: network.WrapRoundTripperWithDebug(&roundTripperWithUA{
		rt:        &http.Transport{TLSHandshakeTimeout: 15 * time.Second},
		userAgent: "QMLauncher/" + version.Current,
	}),
}

type roundTripperWithUA struct {
	rt        http.RoundTripper
	userAgent string
}

func (r *roundTripperWithUA) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.Header.Set("User-Agent", r.userAgent)
	return r.rt.RoundTrip(req2)
}

// saveLocalSkin downloads skin/cape and saves to CustomSkinLoader/LocalSkin for the given username.
// CustomSkinLoader mod loads from LocalSkin/skins/{username}.png and LocalSkin/capes/{username}.png.
func saveLocalSkin(gameDir, username, skinURL, capeURL string) error {
	base := filepath.Join(gameDir, "CustomSkinLoader", "LocalSkin")
	if skinURL == "" && capeURL == "" {
		return nil
	}
	download := func(url, path string) error {
		resp, err := skinDownloadClient.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		out, err := os.Create(path)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, resp.Body)
		return err
	}
	if skinURL != "" {
		skinPath := filepath.Join(base, "skins", username+".png")
		if err := download(skinURL, skinPath); err != nil {
			return fmt.Errorf("download skin: %w", err)
		}
	}
	if capeURL != "" {
		capePath := filepath.Join(base, "capes", username+".png")
		if err := download(capeURL, capePath); err != nil {
			return fmt.Errorf("download cape: %w", err)
		}
	}
	return nil
}

// GetAccountGameDir returns the game directory for the given account.
// When accountUUID is set, returns the per-account dir (players/<uuid>/); otherwise inst.Dir().
func GetAccountGameDir(inst Instance, accountUUID string) (string, error) {
	if accountUUID == "" {
		return inst.Dir(), nil
	}
	return ensurePlayerGameDir(inst, accountUUID)
}

func Prepare(inst Instance, options LaunchOptions, watcher EventWatcher) (LaunchEnvironment, error) {
	var downloads []network.DownloadEntry

	version, err := fetchVersion(inst.Loader, inst.GameVersion, inst.LoaderVersion, inst.CachesDir(), inst.LibrariesDir(), inst.TmpDir())
	if err != nil {
		return LaunchEnvironment{}, fmt.Errorf("retrieve metadata: %w", err)
	}

	gameDir := inst.Dir()
	if options.Session.UUID != "" {
		if playerDir, err := ensurePlayerGameDir(inst, options.Session.UUID); err == nil && playerDir != inst.Dir() {
			gameDir = playerDir
		}
	}

	// Save skin/cape for CustomSkinLoader LocalSkin (Cloud accounts with Ely.by etc.)
	if options.Session.Username != "" {
		if err := saveLocalSkin(gameDir, options.Session.Username, options.SkinURL, options.CapeURL); err != nil {
			// Log but don't fail launch — CustomSkinLoader may not be installed
			log.Printf("[SaveLocalSkin] %v", err)
		}
	}

	launchEnv := LaunchEnvironment{
		GameDir:   gameDir,
		Java:      options.Java,
		MainClass: version.MainClass,
	}

	// On Windows, replace java.exe with javaw.exe if NoJavaWindow is requested
	if runtime.GOOS == "windows" && options.NoJavaWindow && strings.HasSuffix(strings.ToLower(launchEnv.Java), "java.exe") {
		launchEnv.Java = strings.TrimSuffix(launchEnv.Java, "java.exe") + "javaw.exe"
	}
	if watcher != nil {
		watcher(MetadataResolvedEvent{})
	}

	// Filter libraries, and add necessary artifact download entries
	if options.CustomJar == "" {
		version.Libraries = append(version.Libraries, version.Client())
	}

	installedLibs, requiredLibs := filterLibraries(version.Libraries, inst.LibrariesDir())
	if !options.skipLibraries {
		for _, library := range requiredLibs {
			downloads = append(downloads, library.Artifact.DownloadEntry(inst.LibrariesDir()))
		}
	}
	if watcher != nil {
		watcher(LibrariesResolvedEvent{
			Total: len(installedLibs) + len(requiredLibs),
		})
	}

	// Download asset index and add all necessary asset download entries
	assetIndex, err := meta.DownloadAssetIndex(version, inst.AssetsDir())
	if err != nil {
		return LaunchEnvironment{}, fmt.Errorf("retrieve asset index: %w", err)
	}
	if !options.skipAssets {
		downloads = append(downloads, assetIndex.DownloadEntries(inst.AssetsDir())...)
	}
	if watcher != nil {
		watcher(AssetsResolvedEvent{Total: len(assetIndex.Objects)})
	}

	// If no Java path is present, fetch Mojang Java downloads
	var symlinks map[string]string
	if launchEnv.Java == "" {
		manifest, err := meta.FetchJavaManifest(version.JavaVersion.Component, inst.CachesDir())
		if err != nil {
			return LaunchEnvironment{}, fmt.Errorf("fetch Java manifest: %w", err)
		}
		var entries []network.DownloadEntry
		entries, symlinks = manifest.DownloadEntries(version.JavaVersion.Component)
		downloads = append(downloads, entries...)

		exeName := "java"
		if runtime.GOOS == "windows" {
			if options.NoJavaWindow {
				exeName = "javaw.exe"
			} else {
				exeName = "java.exe"
			}
		}
		launchEnv.Java = filepath.Join(env.JavaDir, version.JavaVersion.Component, "bin", exeName)
	}

	if err := download(downloads, symlinks, watcher); err != nil {
		return LaunchEnvironment{}, fmt.Errorf("download files: %w", err)
	}

	// Fetch Forge post processors, if any

	var processors []meta.ForgeProcessor
	switch inst.Loader {
	case LoaderForge:
		processors, err = meta.Forge.FetchPostProcessors(version.ID, version.LoaderID, inst.CachesDir(), inst.LibrariesDir(), inst.TmpDir())
		if err != nil {
			return LaunchEnvironment{}, fmt.Errorf("fetch Forge post processors: %w", err)
		}
	case LoaderNeoForge:
		processors, err = meta.Neoforge.FetchPostProcessors(version.ID, version.LoaderID, inst.CachesDir(), inst.LibrariesDir(), inst.TmpDir())
		if err != nil {
			return LaunchEnvironment{}, fmt.Errorf("fetch NeoForge post processors: %w", err)
		}
	}

	if len(processors) > 0 {
		if watcher != nil {
			watcher(PostProcessingEvent{})
		}
		// Run any available processors
		if err := postProcess(launchEnv, processors); err != nil {
			return LaunchEnvironment{}, fmt.Errorf("run post processors: %w", err)
		}
	}

	launchEnv.JavaArgs, launchEnv.GameArgs = createArgs(launchEnv, version, options, inst)

	// Finalize classpath
	for _, library := range append(installedLibs, requiredLibs...) {
		if library.SkipOnClasspath {
			continue
		}
		launchEnv.Classpath = append(launchEnv.Classpath, library.Artifact.RuntimePath(inst.LibrariesDir()))
	}
	if options.CustomJar != "" {
		launchEnv.Classpath = append(launchEnv.Classpath, options.CustomJar)
	}
	return launchEnv, nil
}

// download takes a list of download entries and executes them, reporting download events to watcher.
//
// It also creates all symlinks specified.
func download(entries []network.DownloadEntry, symlinks map[string]string, watcher EventWatcher) error {
	for link, target := range symlinks {
		if err := os.MkdirAll(filepath.Dir(link), 0755); err != nil {
			return fmt.Errorf("create directory for symlink %q: %w", link, err)
		}
		if err := os.Symlink(target, link); err != nil {
			return fmt.Errorf("create symlink %q: %w", link, err)
		}
	}
	if len(entries) > 0 {
		results := network.StartDownloadEntries(entries)
		i := 0
		for err := range results {
			if err != nil {
				return err
			}
			if watcher != nil {
				watcher(DownloadingEvent{
					Completed: i,
					Total:     len(entries),
				})
			}
			i++
		}
	}
	return nil
}

// createArgs takes data from a launch environment, version metadata, and environment options to
// create a set of game and Java arguments to pass when starting the game.
func createArgs(launchEnv LaunchEnvironment, version meta.VersionMeta, options LaunchOptions, inst Instance) (java, game []string) {
	// Determine user type based on session
	userType := "msa"
	if options.Session.AccessToken == "" {
		userType = "legacy" // Use legacy for offline/local accounts
	}

	// Game arguments
	game = []string{
		"--username", options.Session.Username,
		"--accessToken", options.Session.AccessToken,
		"--userType", userType,
		"--gameDir", launchEnv.GameDir,
		"--assetsDir", inst.AssetsDir(),
		"--assetIndex", version.AssetIndex.ID,
		"--version", version.ID,
		"--versionType", version.Type,
	}

	gameOptions, _ := os.ReadFile(filepath.Join(launchEnv.GameDir, "options.txt"))
	if !strings.Contains(string(gameOptions), "fullscreen:true") {
		game = append(game, "--width", strconv.Itoa(options.WindowResolution.Width))
		game = append(game, "--height", strconv.Itoa(options.WindowResolution.Height))
	}

	switch {
	case options.QuickPlayServer != "":
		game = append(game, "--quickPlayMultiplayer", options.QuickPlayServer)
	case options.QuickPlayWorld != "":
		game = append(game, "--quickPlaySingleplayer", options.QuickPlayWorld)
	}
	if options.Session.UUID != "" {
		game = append(game, "--uuid", options.Session.UUID)
	}
	if options.Demo {
		game = append(game, "--demo")
	}
	if options.DisableChat {
		game = append(game, "--disableChat")
	}
	if options.DisableMultiplayer {
		game = append(game, "--disableMultiplayer")
	}

	// Java arguments
	if runtime.GOOS == "darwin" {
		java = append(java, "-XstartOnFirstThread")
	}
	if options.MinMemory != 0 {
		java = append(java, fmt.Sprintf("-Xms%dm", options.MinMemory))
	}
	if options.MaxMemory != 0 {
		java = append(java, fmt.Sprintf("-Xmx%dm", options.MaxMemory))
	}
	if options.JavaArgs != "" {
		java = append(java, strings.Split(options.JavaArgs, " ")...)
	}
	for _, arg := range version.Arguments.Game {
		if arg, ok := arg.(string); ok {
			game = append(game, arg)
		}
	}
	for _, arg := range version.Arguments.Jvm {
		// Replace any templates
		if arg, ok := arg.(string); ok {
			arg = strings.ReplaceAll(arg, "${version_name}", version.ID)
			arg = strings.ReplaceAll(arg, "${library_directory}", inst.LibrariesDir())
			arg = strings.ReplaceAll(arg, "${classpath_separator}", string(os.PathListSeparator))
			java = append(java, arg)
		}
	}
	return java, game
}

// postProcess takes all Forge post processors and runs them with specified launch environment.
func postProcess(launchEnv LaunchEnvironment, processors []meta.ForgeProcessor) error {
	for _, processor := range processors {
		cmd := exec.Command(launchEnv.Java, processor.JavaArgs...)
		cmd.Dir = launchEnv.GameDir
		cmd.Stderr = os.Stdout
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

// fetchVersion returns a VersionMeta containing both information for the base game, and specified mod loader.
func fetchVersion(loader Loader, gameVersion string, loaderVersion string, cachesDir string, librariesDir string, tmpDir string) (meta.VersionMeta, error) {
	var loaderMeta meta.VersionMeta
	var err error

	version, err := meta.FetchVersionMeta(gameVersion, cachesDir)
	if err != nil {
		// For Forge/NeoForge, try extracting game version from loaderVersion (e.g. "1.20.1-47.2.0" -> "1.20.1")
		if (loader == LoaderForge || loader == LoaderNeoForge) && loaderVersion != "" {
			if extracted := meta.ExtractGameVersionFromLoader(loaderVersion); extracted != "" {
				if v2, err2 := meta.FetchVersionMeta(extracted, cachesDir); err2 == nil {
					version = v2
					err = nil
				}
			}
		}
		if err != nil {
			return meta.VersionMeta{}, fmt.Errorf("retrieve version metadata: %w", err)
		}
	}

	switch loader {
	case LoaderFabric, LoaderQuilt:
		api := meta.Fabric
		if loader == LoaderQuilt {
			api = meta.Quilt
		}
		loaderMeta, err = api.FetchMeta(version.ID, loaderVersion, cachesDir)
		if err != nil {
			return meta.VersionMeta{}, fmt.Errorf("retrieve Fabric/Quilt metadata: %w", err)
		}
	case LoaderNeoForge:
		if loaderVersion == "latest" {
			loaderVersion, err = meta.FetchNeoforgeVersion(version.ID)
			if err != nil {
				return meta.VersionMeta{}, fmt.Errorf("retrieve NeoForge version: %w", err)
			}
		}
		loaderMeta, _, err = meta.Neoforge.FetchMeta(loaderVersion, cachesDir, librariesDir, tmpDir)
		if err != nil {
			return meta.VersionMeta{}, fmt.Errorf("retrieve NeoForge metadata: %w", err)
		}
	case LoaderForge:
		if loaderVersion == "latest" {
			loaderVersion, err = meta.FetchForgeVersion(version.ID)
			if err != nil {
				return meta.VersionMeta{}, fmt.Errorf("retrieve Forge version: %w", err)
			}
		}
		loaderMeta, _, err = meta.Forge.FetchMeta(loaderVersion, cachesDir, librariesDir, tmpDir)
		if err != nil {
			return meta.VersionMeta{}, fmt.Errorf("retrieve Forge metadata: %w", err)
		}
	}

	return meta.MergeVersionMeta(version, loaderMeta), nil
}

// SanitizeInstanceName normalizes an instance name to be filesystem-safe
// by replacing spaces with underscores and removing problematic characters.
// This ensures cross-platform compatibility and prevents filesystem issues.
func SanitizeInstanceName(name string) string {
	if name == "" {
		return "Instance"
	}

	// Replace spaces with underscores
	result := strings.ReplaceAll(name, " ", "_")

	// Remove problematic characters that can cause issues on different filesystems
	result = strings.ReplaceAll(result, "'", "")
	result = strings.ReplaceAll(result, "\"", "")
	result = strings.ReplaceAll(result, "(", "")
	result = strings.ReplaceAll(result, ")", "")
	result = strings.ReplaceAll(result, "[", "")
	result = strings.ReplaceAll(result, "]", "")
	result = strings.ReplaceAll(result, "{", "")
	result = strings.ReplaceAll(result, "}", "")
	result = strings.ReplaceAll(result, "<", "")
	result = strings.ReplaceAll(result, ">", "")
	result = strings.ReplaceAll(result, "|", "")
	result = strings.ReplaceAll(result, "\\", "")
	result = strings.ReplaceAll(result, "/", "")
	result = strings.ReplaceAll(result, ":", "")
	result = strings.ReplaceAll(result, "*", "")
	result = strings.ReplaceAll(result, "?", "")
	result = strings.ReplaceAll(result, "\"", "")

	// Remove multiple consecutive underscores
	for strings.Contains(result, "__") {
		result = strings.ReplaceAll(result, "__", "_")
	}

	// Trim underscores from start and end
	result = strings.Trim(result, "_")

	// Ensure result is not empty after sanitization
	if result == "" {
		return "Instance"
	}

	return result
}
