# QMLauncher

A modern Minecraft launcher built with [Wails](https://wails.io/) (Go + WebView), React, TypeScript, and shadcn/ui.

## Features

- 🚀 **Fast & Lightweight**: Built with Go backend and native WebView, resulting in smaller binaries and better performance compared to Electron-based launchers
- 🎮 **Minecraft Client Management**: Automatic installation, updating, and launching of Minecraft clients
- ☕ **Java Runtime Management**: Automatic Java installation and version management per server
- 🌐 **Multi-Server Support**: Manage multiple Minecraft servers with individual configurations
- 🔐 **Secure Authentication**: Token-based authentication with session persistence
- 🌍 **Internationalization**: Support for Russian and English (easily extensible)
- 🎨 **Modern UI**: Beautiful interface built with React, TypeScript, Tailwind CSS, and shadcn/ui components
- 🔧 **Server-Specific Settings**: Customize Minecraft path, Java path, JVM arguments, and memory per server
- 🔄 **Automatic Server Connection**: Automatically connects to the server when launching Minecraft
- 💾 **Offline Mode**: "Soft" offline mode that preserves session when server is temporarily unavailable
- 🪟 **Windows Optimized**: Hides console window on Windows for cleaner user experience

## Requirements

- **Go**: Version 1.22 or higher
- **Node.js**: Version 18 or higher (for frontend development)
- **Wails**: Latest version (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

## Installation

### Development Setup

1. **Install Go dependencies:**
   ```bash
   go mod download
   ```

2. **Install frontend dependencies:**
   ```bash
   cd frontend
   npm install
   ```

3. **Run in development mode:**
   ```bash
   wails dev
   ```

### Building

**Build for current platform:**
```bash
wails build -obfuscated
```
This will create an executable with code obfuscation enabled using garble.

**Build for Windows (from Linux/macOS):**
```bash
wails build -platform windows/amd64 -obfuscated
```
This will create an executable file (`.exe`) with code obfuscation enabled using garble.

**Build for Linux:**
```bash
wails build -platform linux/amd64 -obfuscated
```
This will create an AppImage (`.AppImage`) with code obfuscation enabled. For `.deb` or `.rpm` packages, additional tools may be required.

**Build for macOS:**
```bash
wails build -platform darwin/amd64 -obfuscated
```
This will create a disk image (`.dmg`) or package (`.pkg`) with code obfuscation enabled.

**Note:** The built files (executables or installers) will be in the `build/bin/` directory. Installer packages are automatically created based on the platform and configuration in `wails.json`.

## Project Structure

```
QMLauncher/
├── app.go              # Main application structure and Wails context
├── minecraft.go        # Minecraft client installation and launching logic
├── java.go             # Java runtime installation and management
├── config.go           # Configuration management and server data caching
├── mods.go             # Mod management (prepared for future implementation)
├── main.go             # Application entry point
├── wails.json          # Wails configuration
├── go.mod              # Go dependencies
├── frontend/           # React frontend application
│   ├── src/
│   │   ├── components/ # React components
│   │   │   ├── ui/     # shadcn/ui components
│   │   │   ├── ServersTab.tsx
│   │   │   └── ServerSettingsDialog.tsx
│   │   ├── locales/    # Internationalization files
│   │   │   ├── ru.json
│   │   │   └── en.json
│   │   ├── lib/        # Utilities
│   │   ├── bridge.ts   # TypeScript bridge for Go backend methods
│   │   └── App.tsx     # Main application component
│   ├── package.json
│   └── vite.config.ts
└── build/              # Build output directory
```

## Architecture

### Backend (Go)
- **app.go**: Main application structure, Wails context methods, and API bridge
- **minecraft.go**: Handles Minecraft client installation, library/asset downloads, native extraction, and process launching
- **java.go**: Manages Java runtime installation with retry logic and error handling
- **config.go**: Server configuration caching, embedded server support, and settings management
- **mods.go**: Prepared for future mod management implementation

### Frontend (React + TypeScript)
- **App.tsx**: Main application component with authentication, routing, and state management
- **ServersTab.tsx**: Server list display, installation, launch, and uninstallation
- **ServerSettingsDialog.tsx**: Server-specific settings management UI
- **bridge.ts**: TypeScript definitions for Go backend methods exposed via Wails

## Key Features Implementation

### Native Library Extraction
Extracts platform-specific native libraries (`.dll`, `.so`, `.dylib`) from JAR archives. Handles LWJGL and other native dependencies correctly for Windows, Linux, and macOS.

### Automatic Server Connection
Uses both `--server`/`--port` arguments and `--quickPlayMultiplayer` flag for maximum compatibility across Minecraft versions.

### Process Management
Tracks Minecraft process lifecycle, automatically detects termination, and updates UI state accordingly.

### Session Persistence
Stores authentication token in localStorage and validates it in the background, providing seamless user experience.

### Windows Console Hiding
Uses `javaw.exe` and `CREATE_NO_WINDOW` flag to hide console window on Windows for cleaner user experience.

## Configuration

### Server Settings
Each server can have custom settings:
- **Minecraft Path**: Custom installation directory
- **Java Path**: Custom Java runtime location
- **JVM Arguments**: Custom JVM arguments
- **Memory**: Min/max memory allocation

### Launcher Settings
Global launcher settings are stored in `~/.qmlauncher/config.json`.

## Development

### Adding New Features

1. **Backend (Go)**: Add methods to `app.go` and implement logic in appropriate files
2. **Frontend (TypeScript)**: Add methods to `bridge.ts` and use them in React components
3. **UI Components**: Use shadcn/ui components from `frontend/src/components/ui/`

### Internationalization

Add translations to:
- `frontend/src/locales/en.json` (English)
- `frontend/src/locales/ru.json` (Russian)

Use the `t()` function from `useI18n()` hook in components.

## Troubleshooting

### Minecraft Client Not Launching
- Check Java installation and path
- Verify native libraries are extracted correctly
- Check logs in console output

### Native Libraries Not Found
- Ensure `extractNatives` is called during installation
- Check `libraries/natives` directory exists
- Verify platform-specific natives are downloaded

### Session Not Persisting
- Check browser localStorage is enabled
- Verify token is being saved correctly
- Check network connectivity for token validation

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## License

Copyright © 2024 QMProject

## Acknowledgments

- [Wails](https://wails.io/) - Framework for building desktop apps
- [shadcn/ui](https://ui.shadcn.com/) - Beautiful UI components
- [React](https://react.dev/) - UI library
- [Tailwind CSS](https://tailwindcss.com/) - Utility-first CSS framework
