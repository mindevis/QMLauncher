# QMLauncher

[![Version](https://img.shields.io/badge/version-1.1.0-blue.svg)](https://github.com/qdevis/QMLauncher/releases/tag/v1.1.0)
[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8.svg)](https://golang.org)
[![Vue.js](https://img.shields.io/badge/Vue.js-3.2+-4FC08D.svg)](https://vuejs.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.3+-3178C6.svg)](https://www.typescriptlang.org)
[![Wails](https://img.shields.io/badge/Wails-2.11+-00ADD8.svg)](https://wails.io)

A modern desktop application built with Go and React using the Wails framework.

## About

QMLauncher is a cross-platform desktop application that serves as a launcher for various applications and tools. Built using the Wails framework, it combines the power of Go for the backend with React for the frontend to create a native desktop experience. The UI is built with shadcn/ui components and styled with Tailwind CSS for a modern, accessible, and beautiful user interface.

## Features

- 🚀 Fast and responsive desktop application
- 🔄 Hot reload during development
- 📦 Cross-platform support (Windows, macOS, Linux)
- 🎨 Modern React frontend with shadcn/ui components
- 🔷 Full TypeScript support for type safety
- ⚡ Native performance with Go backend
- 🛠️ Easy configuration and customization
- 🎯 Beautiful UI with Tailwind CSS
- 📱 Responsive design components

## API Documentation

For programmatic usage of the Minecraft launcher functionality:

- [English API Documentation](docs/API.md)
- [Russian API Documentation](docs/API_ru.md)

The API allows you to create, manage, and launch Minecraft instances programmatically from your Go applications.

## Requirements

- Go 1.19 or later
- Node.js 16 or later
- Wails CLI

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd QMLauncher
```

2. Install Go dependencies:
```bash
go mod tidy
```

3. Install frontend dependencies:
```bash
cd frontend
npm install
cd ..
```

## Development

### Prerequisites

Before starting development, install the required tools:

```bash
# Install development tools
make install-tools

# Install all dependencies
make deps
```

### Running Modes

#### GUI Mode (Default)

To run the application in GUI mode with Wails desktop interface:

```bash
wails dev
# or
make dev
```

This will start:
- The Go backend
- Vite development server for the frontend
- Hot reload for both backend and frontend changes

#### CLI Mode

To run the application in command-line mode for Minecraft launcher functionality:

```bash
./qmlauncher --no-gui [command]
```

Available CLI commands:
- `start` - Start the specified Minecraft instance
- `instance list` - List all instances with their paths
- `instance create <name> -v <version> -l <loader>` - Create a new instance
- `instance delete <name>` - Delete an instance
- `instance rename <old> <new>` - Rename an instance
- `java list` - List all installed Java versions
- `auth` - Manage account authentication
- `search` - Search Minecraft versions
- `completions` - Generate shell completions
- `about` - Display launcher version and information

The launcher uses `~/.qmlauncher` as the default working directory for storing instances, assets, libraries, and configuration files. You can override this with the `--dir` flag.

### Instance Structure

Each instance is stored in its own directory with a unique UUID for isolation:

```
~/.qmlauncher/instances/
└── InstanceName/
    └── uuid/
        ├── instance.toml    # Instance configuration
        ├── minecraft.jar    # Minecraft client
        ├── forge.jar        # Mod loader (if applicable)
        ├── mods/           # Mods directory
        ├── saves/          # World saves
        └── config/         # Configuration files
```

Shared resources remain in the root directories:
- `libraries/` - Java libraries
- `assets/` - Game assets and textures
- `caches/` - Downloaded metadata and manifests
- `java/` - Java runtime installations

Examples:
```bash
# Show help
./qmlauncher --no-gui --help

# Display version info
./qmlauncher --no-gui about

# Search Minecraft versions
./qmlauncher --no-gui search 1.20

# List instances
./qmlauncher --no-gui instance list
```

### Browser Development

For frontend development with browser access to Go methods, connect to:
- http://localhost:34115 (dev server)

### Code Quality

#### Linting and Formatting

Run all code quality checks:

```bash
make check
```

This includes:
- Go code formatting (`make fmt`)
- Go linting (`make lint`)
- Frontend linting (`make frontend-lint`)
- Frontend formatting (`make frontend-fmt`)
- TypeScript type checking (`npm run type-check` in frontend/)

#### Individual Commands

```bash
# Go commands
make lint          # Run golangci-lint
make fmt           # Format Go code
make vet           # Run go vet
make test          # Run tests

# Frontend commands
make frontend-lint     # Run ESLint
make frontend-fmt      # Format with Prettier
make frontend-install  # Install dependencies

# Combined commands
make deps          # Install all dependencies
make clean         # Clean build artifacts
make check         # Run all checks
```

### Project Structure

```
QMLauncher/
├── main.go              # Application entry point
├── app.go               # Main application logic
├── wails.json          # Wails configuration
├── go.mod              # Go module file
├── frontend/           # React frontend
│   ├── src/
│   │   ├── App.tsx
│   │   ├── main.tsx
│   │   ├── components/
│   │   ├── hooks/
│   │   ├── contexts/
│   │   └── types/
│   ├── package.json
│   └── vite.config.js
└── build/              # Build output
```

## Building

### Development Build

```bash
wails build
```

### Production Build

```bash
wails build -production
```

### Platform-Specific Builds

Use convenient Makefile commands for cross-platform builds:

```bash
# Build for current platform
make linux    # Linux
make macos    # macOS
make windows  # Windows

# Build for specific architectures
make linux-amd64 linux-arm64     # Linux AMD64 + ARM64
make macos-amd64 macos-arm64     # macOS Intel + Apple Silicon
make windows-amd64 windows-arm64 # Windows AMD64 + ARM64

# Build for all major platforms (AMD64)
make release

# Build for all platforms and architectures
make release-all
```

Built applications will be placed in `build/bin/` with descriptive names including platform and architecture.

#### Manual Wails Commands (if needed)

```bash
# Windows
wails build -platform windows/amd64

# macOS
wails build -platform darwin/amd64

# Linux
wails build -platform linux/amd64
```

## Configuration

The project can be configured by editing `wails.json`. More information about project settings can be found at:
https://wails.io/docs/reference/project-config

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## Versioning

This project follows [Semantic Versioning](https://semver.org/) and [Conventional Commits](https://conventionalcommits.org/) specifications.

### Release Types

- **MAJOR** version (X.y.z) - Breaking changes
- **MINOR** version (x.Y.z) - New features (backward compatible)
- **PATCH** version (x.y.Z) - Bug fixes (backward compatible)

### Commit Message Format

```
type(scope): description

[optional body]

[optional footer]
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

Examples:
- `feat: add user authentication`
- `fix: resolve memory leak in file handler`
- `docs: update API documentation`

## CI/CD

This project uses GitHub Actions for automated building and releasing:

- **Build**: Runs on every push/PR (excluding docs), tests compilation on Windows/macOS/Linux
- **Release**: Triggers on release creation, builds binaries for all platforms and uploads to GitHub Releases

## Contributing

1. Follow Conventional Commits for commit messages
2. Run `make check` before submitting PR
3. Update CHANGELOG.md for significant changes
4. Test on multiple platforms when possible
5. Ensure CI passes before merging

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

If you encounter any issues or have questions:

- Check the [Wails documentation](https://wails.io/docs)
- Open an issue on GitHub
- Join the [Wails Discord community](https://discord.gg/7FY4VQ4)
