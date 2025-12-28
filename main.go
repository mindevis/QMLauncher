package main

import (
	"embed"
	"os"

	"QMLauncher/internal/cli"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// Application constants
const (
	windowWidth  = 1024
	windowHeight = 768
)

// Background color components
const (
	backgroundRed   = 27
	backgroundGreen = 38
	backgroundBlue  = 54
	backgroundAlpha = 1
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Check for --no-gui flag before parsing other arguments
	noGUI := false
	args := os.Args[1:]

	// Filter out --no-gui flag and check if it exists
	filteredArgs := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--no-gui" {
			noGUI = true
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	// Replace os.Args to exclude --no-gui for CLI parser
	os.Args = append([]string{os.Args[0]}, filteredArgs...)

	// If --no-gui flag is provided, run CLI mode
	if noGUI {
		runCLI()
		return
	}

	// Otherwise, run GUI mode
	runGUI()
}

func runGUI() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "QMLauncher",
		Width:  windowWidth,
		Height: windowHeight,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: backgroundRed, G: backgroundGreen, B: backgroundBlue, A: backgroundAlpha},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

func runCLI() {
	// Parse and run the CLI from QMLauncher
	exiter, code := cli.Run()
	exiter(code)
	os.Exit(code)
}
