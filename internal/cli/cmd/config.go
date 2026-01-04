package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"QMLauncher/internal/cli/output"
	env "QMLauncher/pkg"

	"github.com/alecthomas/kong"
)

// InteractiveConfig represents the interactive mode configuration
type InteractiveConfig struct {
	Theme          string `json:"theme" default:"default"`
	AutoComplete   bool   `json:"autocomplete" default:"true"`
	ShowStatusBar  bool   `json:"show_status_bar" default:"true"`
	DebugMode      bool   `json:"debug_mode" default:"false"`
	MaxHistorySize int    `json:"max_history_size" default:"1000"`
	ProgressStyle  string `json:"progress_style" default:"default"`
	ColorScheme    string `json:"color_scheme" default:"default"`
}

// ConfigCmd represents the config command
type ConfigCmd struct {
	Args []string `arg:"" optional:"" help:"Аргументы команды"`
}

func (c *ConfigCmd) Run(ctx *kong.Context) error {
	args := c.Args
	if len(args) == 0 {
		config, err := loadConfig()
		if err != nil {
			return fmt.Errorf("не удалось загрузить конфигурацию: %w", err)
		}
		return listConfig(config)
	}

	action := args[0]
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("не удалось загрузить конфигурацию: %w", err)
	}

	switch action {
	case "list":
		return listConfig(config)
	case "reset":
		return resetConfig()
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("укажите параметры для получения")
		}
		return getConfigValues(config, args[1:])
	case "set":
		if len(args) < 2 {
			return fmt.Errorf("укажите параметры для установки (key=value)")
		}
		// Parse key=value pairs
		values := make(map[string]string)
		for _, arg := range args[1:] {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("некорректный формат параметра: %s (ожидается key=value)", arg)
			}
			values[parts[0]] = parts[1]
		}
		return setConfigValues(config, values)
	case "export":
		if len(args) < 2 {
			return fmt.Errorf("укажите путь к файлу экспорта")
		}
		return exportConfig(config, args[1])
	case "import":
		if len(args) < 2 {
			return fmt.Errorf("укажите путь к файлу импорта")
		}
		return importConfig(args[1])
	default:
		return fmt.Errorf("неизвестное действие: %s", action)
	}
}

// getConfigFilePath returns the path to the config file
func getConfigFilePath() string {
	return filepath.Join(env.RootDir, "qmlauncher.json")
}

// loadConfig loads the interactive configuration
func loadConfig() (*InteractiveConfig, error) {
	configPath := getConfigFilePath()

	// Default configuration
	config := &InteractiveConfig{
		Theme:          "default",
		AutoComplete:   true,
		ShowStatusBar:  true,
		DebugMode:      false,
		MaxHistorySize: 1000,
		ProgressStyle:  "default",
		ColorScheme:    "default",
	}

	// Try to load existing configuration
	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config doesn't exist, return defaults
			return config, nil
		}
		return nil, err
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(config); err != nil {
		output.Warning("Не удалось разобрать конфигурационный файл, используются значения по умолчанию: %v", err)
		return config, nil
	}

	return config, nil
}

// saveConfig saves the interactive configuration
func saveConfig(config *InteractiveConfig) error {
	configPath := getConfigFilePath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(config)
}

// setConfigValues sets configuration values
func setConfigValues(config *InteractiveConfig, values map[string]string) error {
	updated := false

	for key, value := range values {
		switch key {
		case "theme":
			config.Theme = value
			updated = true
		case "autocomplete":
			config.AutoComplete = parseBool(value)
			updated = true
		case "show_status_bar":
			config.ShowStatusBar = parseBool(value)
			updated = true
		case "debug_mode":
			config.DebugMode = parseBool(value)
			updated = true
		case "max_history_size":
			if size := parseInt(value); size > 0 {
				config.MaxHistorySize = size
				updated = true
			}
		case "progress_style":
			config.ProgressStyle = value
			updated = true
		case "color_scheme":
			config.ColorScheme = value
			updated = true
		default:
			output.Warning("Неизвестный параметр конфигурации: %s", key)
		}
	}

	if updated {
		if err := saveConfig(config); err != nil {
			return fmt.Errorf("не удалось сохранить конфигурацию: %w", err)
		}
		output.Success(output.Translate("config.updated"))
	} else {
		output.Info("Никакие параметры не были изменены")
	}

	return nil
}

// getConfigValues gets configuration values
func getConfigValues(config *InteractiveConfig, keys []string) error {
	for _, key := range keys {
		var value interface{}
		switch key {
		case "theme":
			value = config.Theme
		case "autocomplete":
			value = config.AutoComplete
		case "show_status_bar":
			value = config.ShowStatusBar
		case "debug_mode":
			value = config.DebugMode
		case "max_history_size":
			value = config.MaxHistorySize
		case "progress_style":
			value = config.ProgressStyle
		case "color_scheme":
			value = config.ColorScheme
		default:
			output.Error("Неизвестный параметр конфигурации: %s", key)
			continue
		}
		fmt.Printf("%s = %v\n", key, value)
	}
	return nil
}

// listConfig shows all configuration values
func listConfig(config *InteractiveConfig) error {
	output.Header("Конфигурация интерактивного режима")
	fmt.Println()

	fmt.Printf("Theme:           %s\n", config.Theme)
	fmt.Printf("AutoComplete:    %t\n", config.AutoComplete)
	fmt.Printf("ShowStatusBar:   %t\n", config.ShowStatusBar)
	fmt.Printf("DebugMode:       %t\n", config.DebugMode)
	fmt.Printf("MaxHistorySize:  %d\n", config.MaxHistorySize)
	fmt.Printf("ProgressStyle:   %s\n", config.ProgressStyle)
	fmt.Printf("ColorScheme:     %s\n", config.ColorScheme)

	fmt.Println()
	output.Status(fmt.Sprintf("Файл конфигурации: %s", getConfigFilePath()))

	return nil
}

// resetConfig resets configuration to defaults
func resetConfig() error {
	configPath := getConfigFilePath()
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("не удалось удалить файл конфигурации: %w", err)
	}
	output.Success("Конфигурация сброшена к значениям по умолчанию")
	return nil
}

// exportConfig exports configuration to a file
func exportConfig(config *InteractiveConfig, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("не удалось создать файл экспорта: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("не удалось экспортировать конфигурацию: %w", err)
	}

	output.Success("Конфигурация экспортирована в: %s", filePath)
	return nil
}

// importConfig imports configuration from a file
func importConfig(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("не удалось открыть файл импорта: %w", err)
	}
	defer file.Close()

	var config InteractiveConfig
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return fmt.Errorf("не удалось разобрать файл импорта: %w", err)
	}

	if err := saveConfig(&config); err != nil {
		return fmt.Errorf("не удалось сохранить импортированную конфигурацию: %w", err)
	}

	output.Success("Конфигурация импортирована из: %s", filePath)
	return nil
}

// parseBool parses a string to boolean
func parseBool(s string) bool {
	switch s {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return false
	}
}

// parseInt parses a string to int
func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}
