# Changelog

All notable changes to QMLauncher will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.1] - 2025-01-11

### Security
- **Fixed hardcoded access token**: Replaced hardcoded `'token'` with empty string `''` for `--accessToken` argument in Minecraft launch (offline mode requirement)
- **Removed unused secureStorage**: Deleted `secureStorage.ts` that was created but never used in the codebase

### Changed
- **Improved logging**: Replaced critical `console.log/error` calls with `logger` utility (only active in development mode)
- **Production build optimization**: Configured Vite with terser to automatically remove all `console.log` statements in production builds
- **Code cleanup**: Removed unused code and improved code quality

### Technical Details
- Added terser as dev dependency for production builds
- Configured `drop_console: true` in Vite build configuration
- Logger utility now respects `import.meta.env.DEV` for conditional logging

## [1.0.0] - 2025-01-11

### Added
- **Complete migration from Electron to Wails framework**
  - Go backend with Wails v2 integration
  - React + TypeScript frontend with shadcn/ui components
  - Native performance and smaller application size

- **Minecraft Client Installation**
  - Automatic download and installation of Minecraft client JAR
  - Library management with parallel downloads
  - Asset downloading with SHA1 verification
  - Native library extraction (LWJGL) for Windows, Linux, and macOS
  - Platform-specific native library handling
  - Comprehensive installation progress tracking

- **Java Runtime Management**
  - Automatic Java installation (OpenJDK)
  - Vendor and version selection per server
  - Retry logic with exponential backoff for downloads
  - Temporary file handling to prevent corrupted downloads
  - Java path validation and caching

- **Minecraft Client Launching**
  - Full JVM argument construction
  - Game argument processing with variable substitution
  - Automatic server connection via `--server` and `--port` arguments
  - Support for `--quickPlayMultiplayer` for modern Minecraft versions
  - Path normalization (tilde expansion)
  - Working directory management

- **Server Management**
  - Server list display with search functionality
  - Server installation status tracking
  - Client installation per server
  - Server-specific settings (Minecraft path, Java path, JVM arguments, memory)
  - Server uninstallation with proper cleanup
  - Loader icons display (Forge, Fabric, Quilt, NeoForge, Vanilla)

- **Authentication & Session Management**
  - Login form with username/password
  - Token-based authentication
  - Session persistence with localStorage
  - "Soft" offline mode - keeps token but shows warning when server unavailable
  - Background token validation
  - Automatic token refresh on app start

- **User Interface**
  - Modern UI built with React, TypeScript, and Tailwind CSS
  - shadcn/ui component library integration
  - Internationalization (i18n) support (Russian and English)
  - Toast notifications (sonner)
  - Progress bars for installation
  - Dialog modals for confirmations
  - Responsive design

- **Windows-specific Features**
  - Console window hiding using `javaw.exe` and `CREATE_NO_WINDOW` flag
  - Process output suppression on Windows
  - Windows-specific path handling

- **Process Management**
  - Minecraft process tracking
  - Automatic detection of process termination
  - Play button state management based on running processes
  - Periodic process status checks

- **Error Handling & Logging**
  - Comprehensive logging throughout the application
  - Detailed error messages for users
  - Network error handling with retries
  - File system error handling
  - Installation error recovery

- **Configuration Management**
  - Server configuration caching
  - Embedded server support
  - Launcher settings persistence
  - Server UUID management for proper uninstallation

### Changed
- **Architecture**: Migrated from Electron (Node.js) to Wails (Go + WebView)
- **IPC Communication**: Replaced Electron IPC with Wails context methods
- **File Operations**: Migrated from Node.js fs to Go os package
- **Process Management**: Migrated from Node.js child_process to Go exec package
- **Build System**: Replaced Electron Builder with Wails build system

### Fixed
- **Native Library Extraction**: Fixed LWJGL DLL loading issues on Windows
- **Path Handling**: Fixed tilde expansion and path normalization issues
- **Resource Loading**: Fixed "Can't open the resource index file" error
- **Session Persistence**: Fixed token not being saved on app restart
- **Uninstallation**: Fixed server UUID detection for proper cleanup
- **Console Window**: Fixed flickering console window on Windows during Minecraft launch
- **Play Button State**: Fixed button remaining disabled after client closure
- **Server Connection**: Fixed automatic server connection on launch
- **Java Downloads**: Fixed 503 errors and zero-sized file downloads with retry logic

### Technical Details

#### Backend (Go)
- `app.go`: Main application structure and Wails context
- `minecraft.go`: Minecraft client installation, launching, and management
- `java.go`: Java runtime installation and management
- `config.go`: Configuration management and server data caching
- `mods.go`: Mod management (prepared for future implementation)

#### Frontend (React + TypeScript)
- `App.tsx`: Main application component with authentication and routing
- `ServersTab.tsx`: Server list, installation, and launch management
- `ServerSettingsDialog.tsx`: Server-specific settings management
- `bridge.ts`: TypeScript bridge for Go backend methods
- `locales/`: Internationalization files (ru.json, en.json)

#### Key Features Implementation
- **Native Library Extraction**: Extracts `.dll`, `.so`, `.dylib` files from JAR archives based on platform
- **Automatic Server Connection**: Uses both `--server`/`--port` and `--quickPlayMultiplayer` for compatibility
- **Process Hiding**: Uses `syscall.SysProcAttr` with `CREATE_NO_WINDOW` flag on Windows
- **Session Persistence**: Stores auth token in localStorage and validates in background
- **Retry Logic**: Implements exponential backoff for network requests

## [Unreleased]

### Planned
- Mod management and installation
- Resource pack management
- Profile management
- Auto-update functionality
- Advanced settings UI
- Log viewer
- Performance monitoring

