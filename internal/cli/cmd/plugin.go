package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sort"
	"strings"

	"QMLauncher/internal/cli/output"
	env "QMLauncher/pkg"

	"github.com/alecthomas/kong"
)

// Plugin represents a loaded plugin
type Plugin struct {
	Name        string
	Version     string
	Description string
	Author      string
	Enabled     bool
	Path        string
	Symbol      plugin.Symbol
}

// PluginManager manages plugins
type PluginManager struct {
	plugins map[string]*Plugin
}

// Global plugin manager instance
var pluginManager = &PluginManager{
	plugins: make(map[string]*Plugin),
}

// PluginCmd represents the plugin command
type PluginCmd struct {
	List   bool     `help:"Показать список плагинов"`
	Load   []string `help:"Загрузить плагины"`
	Unload []string `help:"Выгрузить плагины"`
	Enable []string `help:"Включить плагины"`
	Info   string   `help:"Информация о плагине"`
}

func (c *PluginCmd) Run(ctx *kong.Context) error {
	switch {
	case c.List:
		return listPlugins()
	case len(c.Load) > 0:
		return loadPlugins(c.Load)
	case len(c.Unload) > 0:
		return unloadPlugins(c.Unload)
	case len(c.Enable) > 0:
		return enablePlugins(c.Enable)
	case c.Info != "":
		return showPluginInfo(c.Info)
	default:
		return listPlugins()
	}
}

// getPluginsDir returns the plugins directory path
func getPluginsDir() string {
	return filepath.Join(env.RootDir, "plugins")
}

// discoverPlugins scans the plugins directory for available plugins
func discoverPlugins() ([]string, error) {
	pluginsDir := getPluginsDir()

	// Ensure plugins directory exists
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return nil, fmt.Errorf("не удалось создать директорию плагинов: %w", err)
	}

	var plugins []string

	// Scan for .so files (shared libraries)
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".so") {
			name := strings.TrimSuffix(entry.Name(), ".so")
			plugins = append(plugins, name)
		}
	}

	sort.Strings(plugins)
	return plugins, nil
}

// loadPlugin loads a single plugin
func loadPlugin(name string) error {
	pluginsDir := getPluginsDir()
	pluginPath := filepath.Join(pluginsDir, name+".so")

	// Check if plugin file exists
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return fmt.Errorf("плагин '%s' не найден в %s", name, pluginPath)
	}

	// Load the plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("не удалось загрузить плагин '%s': %w", name, err)
	}

	// Look for plugin metadata
	sym, err := p.Lookup("PluginInfo")
	if err != nil {
		return fmt.Errorf("плагин '%s' не содержит метаданных PluginInfo: %w", name, err)
	}

	// Try to cast to PluginInfo interface
	info, ok := sym.(*PluginInfo)
	if !ok {
		return fmt.Errorf("плагин '%s' имеет некорректный формат метаданных", name)
	}

	// Create plugin instance
	plugin := &Plugin{
		Name:        info.Name,
		Version:     info.Version,
		Description: info.Description,
		Author:      info.Author,
		Enabled:     true,
		Path:        pluginPath,
		Symbol:      sym,
	}

	pluginManager.plugins[name] = plugin

	// Call plugin initialization if available
	if initSym, err := p.Lookup("Init"); err == nil {
		if initFunc, ok := initSym.(func()); ok {
			initFunc()
		}
	}

	output.Success("Плагин '%s' загружен (%s)", name, info.Version)
	return nil
}

// unloadPlugin unloads a single plugin
func unloadPlugin(name string) error {
	plug, exists := pluginManager.plugins[name]
	if !exists {
		return fmt.Errorf("плагин '%s' не загружен", name)
	}

	// Call plugin cleanup if available
	if p, err := plugin.Open(plug.Path); err == nil {
		if cleanupSym, err := p.Lookup("Cleanup"); err == nil {
			if cleanupFunc, ok := cleanupSym.(func()); ok {
				cleanupFunc()
			}
		}
	}

	delete(pluginManager.plugins, name)
	output.Success("Плагин '%s' выгружен", name)
	return nil
}

// listPlugins shows all available and loaded plugins
func listPlugins() error {
	output.Header("Система плагинов QMLauncher")
	fmt.Println()

	// Show loaded plugins
	if len(pluginManager.plugins) > 0 {
		fmt.Println("Загруженные плагины:")
		for name, p := range pluginManager.plugins {
			status := "✓"
			if !p.Enabled {
				status = "✗"
			}
			fmt.Printf("  %s %s v%s - %s\n", status, name, p.Version, p.Description)
		}
		fmt.Println()
	} else {
		fmt.Println("Загруженных плагинов нет")
		fmt.Println()
	}

	// Show available plugins
	available, err := discoverPlugins()
	if err != nil {
		return fmt.Errorf("не удалось обнаружить плагины: %w", err)
	}

	if len(available) > 0 {
		fmt.Println("Доступные плагины:")
		for _, name := range available {
			if _, loaded := pluginManager.plugins[name]; !loaded {
				fmt.Printf("  ○ %s (не загружен)\n", name)
			}
		}
		fmt.Println()
	}

	fmt.Printf("Директория плагинов: %s\n", getPluginsDir())
	fmt.Println("Для загрузки плагина используйте: plugin load <name>")
	fmt.Println("Для создания плагина изучите документацию по API плагинов")

	return nil
}

// loadPlugins loads multiple plugins
func loadPlugins(names []string) error {
	successCount := 0

	for _, name := range names {
		if err := loadPlugin(name); err != nil {
			output.Error("Ошибка загрузки плагина '%s': %v", name, err)
		} else {
			successCount++
		}
	}

	if successCount > 0 {
		output.SuccessHighlight(fmt.Sprintf("Загружено плагинов: %d/%d", successCount, len(names)))
	}

	return nil
}

// unloadPlugins unloads multiple plugins
func unloadPlugins(names []string) error {
	successCount := 0

	for _, name := range names {
		if err := unloadPlugin(name); err != nil {
			output.Error("Ошибка выгрузки плагина '%s': %v", name, err)
		} else {
			successCount++
		}
	}

	if successCount > 0 {
		output.SuccessHighlight(fmt.Sprintf("Выгружено плагинов: %d/%d", successCount, len(names)))
	}

	return nil
}

// enablePlugins enables multiple plugins
func enablePlugins(names []string) error {
	successCount := 0

	for _, name := range names {
		if plugin, exists := pluginManager.plugins[name]; exists {
			plugin.Enabled = true
			output.Success("Плагин '%s' включен", name)
			successCount++
		} else {
			output.Error("Плагин '%s' не найден", name)
		}
	}

	if successCount > 0 {
		output.SuccessHighlight(fmt.Sprintf("Включено плагинов: %d/%d", successCount, len(names)))
	}

	return nil
}

// showPluginInfo shows detailed information about a plugin
func showPluginInfo(name string) error {
	plugin, exists := pluginManager.plugins[name]
	if !exists {
		return fmt.Errorf("плагин '%s' не загружен", name)
	}

	output.Header(fmt.Sprintf("Информация о плагине: %s", name))
	fmt.Println()

	fmt.Printf("Имя:        %s\n", plugin.Name)
	fmt.Printf("Версия:     %s\n", plugin.Version)
	fmt.Printf("Автор:      %s\n", plugin.Author)
	fmt.Printf("Описание:   %s\n", plugin.Description)
	fmt.Printf("Включен:    %t\n", plugin.Enabled)
	fmt.Printf("Путь:       %s\n", plugin.Path)

	fmt.Println()
	fmt.Println("Для управления плагином используйте:")
	fmt.Println("  plugin enable " + name + "    - включить плагин")
	fmt.Println("  plugin unload " + name + "    - выгрузить плагин")

	return nil
}

// PluginInfo represents plugin metadata
type PluginInfo struct {
	Name        string
	Version     string
	Description string
	Author      string
}

// RegisterPlugin registers a plugin (called by plugins themselves)
func RegisterPlugin(info *PluginInfo) {
	// This function would be called by plugins during initialization
	// Implementation depends on plugin architecture
}

// GetPluginManager returns the global plugin manager
func GetPluginManager() *PluginManager {
	return pluginManager
}
