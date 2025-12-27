package main

import (
	"embed"

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
