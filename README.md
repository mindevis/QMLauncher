# QMLauncher

[![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)](https://github.com/mindeivs/QMLauncher/releases/tag/v1.0.0)
[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8.svg)](https://golang.org)
[![Vue.js](https://img.shields.io/badge/Vue.js-3.2+-4FC08D.svg)](https://vuejs.org)
[![Wails](https://img.shields.io/badge/Wails-2.11+-00ADD8.svg)](https://wails.io)

A modern desktop application built with Go and Vue.js using the Wails framework.

## About

QMLauncher is a cross-platform desktop application that serves as a launcher for various applications and tools. Built using the Wails framework, it combines the power of Go for the backend with Vue.js for the frontend to create a native desktop experience.

## Features

- 🚀 Fast and responsive desktop application
- 🔄 Hot reload during development
- 📦 Cross-platform support (Windows, macOS, Linux)
- 🎨 Modern Vue.js frontend
- ⚡ Native performance with Go backend
- 🛠️ Easy configuration and customization

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

### Running in Development Mode

To run the application in development mode with hot reload:

```bash
wails dev
# or
make dev
```

This will start:
- The Go backend
- Vite development server for the frontend
- Hot reload for both backend and frontend changes

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
├── frontend/           # Vue.js frontend
│   ├── src/
│   │   ├── App.vue
│   │   ├── main.js
│   │   └── components/
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

## Contributing

1. Follow Conventional Commits for commit messages
2. Run `make check` before submitting PR
3. Update CHANGELOG.md for significant changes
4. Test on multiple platforms when possible

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

If you encounter any issues or have questions:

- Check the [Wails documentation](https://wails.io/docs)
- Open an issue on GitHub
- Join the [Wails Discord community](https://discord.gg/7FY4VQ4)
