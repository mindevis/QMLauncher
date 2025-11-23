# QMLauncher

A Minecraft launcher built with Electron, React, and TypeScript.

## Features

- **Server Management**: Browse and connect to servers from QMServer
- **Profile Management**: Create and manage multiple launch profiles
- **Customizable Settings**: Configure Java path and memory settings
- **Modern GUI**: Clean and intuitive user interface built with React
- **Cross-platform**: Works on Windows, macOS, and Linux

## Requirements

- Node.js 18+
- npm or yarn

## Installation

1. Install dependencies:
```bash
npm install
```

2. Run in development mode:
```bash
npm run dev
```

3. Build for production:
```bash
npm run build
```

4. Package the application:
```bash
npm run dist
```

## Configuration

The launcher stores its configuration and game files in `~/.qmlauncher/` (or `%APPDATA%/.qmlauncher/` on Windows):
- `versions/` - Installed Minecraft versions
- `profiles/` - Launch profiles
- `assets/` - Game assets
- `libraries/` - Game libraries
- `config.json` - Launcher configuration

## Usage

1. **Servers Tab**: Browse available servers from QMServer and launch Minecraft
2. **Profiles Tab**: Create profiles with custom settings (version, memory, username)
3. **Settings Tab**: Configure Java path, memory settings, and API URL
4. **Launch**: Click "Запустить" button to start Minecraft with selected server

## Development

The project is structured as follows:
- `electron/` - Electron main process and preload scripts
- `src/renderer/` - React application (UI)
- `src/shared/` - Shared TypeScript types and utilities
- `dist/` - Built React application
- `dist-electron/` - Built Electron main process

## Integration with QMServer

QMLauncher integrates with QMServer API to:
- Fetch available servers
- Get server configurations
- Use server settings for Minecraft launch

Set the API URL in Settings tab (default: `http://localhost:8000/api/v1`)

## License

This project is part of the QMProject ecosystem.

