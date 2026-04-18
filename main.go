package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	for _, a := range os.Args[1:] {
		if a == "-version" || a == "--version" {
			fmt.Printf("QMLauncher %s build=%s\n", version, buildStamp)
			return
		}
	}
	runGUI()
}

func runGUI() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options. Bind exposes only App methods to the WebView — keep that API minimal and input-safe (see SECURITY.md).
	err := wails.Run(&options.App{
		Title:  fmt.Sprintf("QMLauncher %s", version),
		Width:  1200,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
