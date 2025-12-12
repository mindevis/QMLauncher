package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Настройка логирования
	setupLogging()

	log.Println("=== QMLauncher Starting ===")
	log.Println("Initializing services...")

	// Создаем экземпляр приложения
	app := NewApp()
	minecraftService := NewMinecraftService(app)
	javaService := NewJavaService(app)
	configService := NewConfigService(app)

	log.Println("Services initialized")

	// Настройки приложения
	// В Wails v2.11.0 сервисы регистрируются через поле Bind
	log.Println("Creating Wails application...")
	err := wails.Run(&options.App{
		Title:  "", // Пустой заголовок для frameless окна
		Width:  1000,
		Height: 700,
		MinWidth:  700,
		MinHeight: 500,
		Frameless: true, // Frameless окно - убирает стандартные кнопки и titlebar
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 26, G: 26, B: 26, A: 255},
		Windows: &windows.Options{
			WebviewIsTransparent: true,  // Прозрачный webview для frameless окна
			WindowIsTranslucent:  true,  // Frameless window - убирает стандартные кнопки управления окном
			DisableWindowIcon:    true,
		},
		StartHidden: false,
		OnStartup: func(ctx context.Context) {
			log.Println("OnStartup called")
			app.startup(ctx)
			minecraftService.Startup(ctx)
			
			// В Wails на Windows может потребоваться дополнительная настройка для скрытия стандартных кнопок
			// Попробуем установить размер окна явно, чтобы убедиться что настройки применены
			log.Println("Startup completed")
		},
		OnDomReady: func(ctx context.Context) {
			log.Println("OnDomReady called - frontend loaded")
		},
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			log.Println("OnBeforeClose called")
			return false
		},
		Bind: []interface{}{
			app,
			minecraftService,
			javaService,
			configService,
		},
	})

	if err != nil {
		log.Printf("Fatal error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	log.Println("Application closed")
}

func setupLogging() {
	// Создаем директорию для логов
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	logDir := filepath.Join(homeDir, ".qmlauncher", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// Если не можем создать директорию, просто используем stderr
		return
	}

	// Создаем файл лога
	logPath := filepath.Join(logDir, fmt.Sprintf("qmlauncher-wails-%d.log", os.Getpid()))
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// Если не можем создать файл, просто используем stderr
		return
	}

	// Настраиваем логирование в файл и stderr
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Logging to file: %s\n", logPath)
}

