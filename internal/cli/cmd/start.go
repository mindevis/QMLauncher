package cmd

import (
	env "QMLauncher/pkg"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"QMLauncher/internal/cli/output"
	"QMLauncher/pkg/auth"
	"QMLauncher/pkg/launcher"

	"github.com/alecthomas/kong"
	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

// ServerConnection represents a server connection entry
type ServerConnection struct {
	Username string `json:"username"`
	Server   string `json:"server"`
	Instance string `json:"instance"`
	Time     int64  `json:"time"`
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
func addRecentConnection(username, server, instance string) error {
	connections, err := loadRecentConnections()
	if err != nil {
		return err
	}

	// Remove duplicates (same username + server + instance)
	connections = filterConnections(connections, func(c ServerConnection) bool {
		return !(c.Username == username && c.Server == server && c.Instance == instance)
	})

	// Add new connection at the beginning
	newConnection := ServerConnection{
		Username: username,
		Server:   server,
		Instance: instance,
		Time:     time.Now().Unix(),
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

	config := inst.Config

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
		// Save to global recent connections
		if err := addRecentConnection(session.Username, c.Options.Server, c.ID); err != nil {
			output.Warning("Не удалось сохранить информацию о подключении: %v", err)
		}

		// Save to instance config
		if config.LastServer != c.Options.Server || config.LastUser != session.Username {
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
	output.Success(output.Translate("start.launch"), color.New(color.Bold).Sprint(session.Username))

	// Choose runner based on verbosity level
	var runner launcher.Runner
	if verbosity == 0 {
		// Default verbosity - hide Minecraft logs
		runner = QuietRunner
	} else {
		// Extra/debug verbosity - show Minecraft logs
		runner = launcher.ConsoleRunner
	}

	return launcher.Launch(launchEnv, runner)
}
