# QMLauncher

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Minimalistic Minecraft launcher for command line.

## 📋 Description

QMLauncher is a cross-platform command-line application for launching Minecraft. It provides a convenient interface for managing Minecraft instances, authentication, version searching, and Java runtime management.

## ✨ Features

- 🚀 **Instance Management** - create, configure, and launch Minecraft instances
- 🔐 **Authentication** - support for various Mojang/Microsoft authentication methods
- 🔍 **Version Search** - search and install Minecraft versions and mod loaders
- ☕ **Java Management** - automatic detection and management of Java installations
- 📦 **Auto-updates** - built-in application update system
- 🌍 **Localization** - support for Russian and English languages

## 🛠️ Installation

### Download Release

Download the appropriate version for your platform from [releases](https://github.com/telecter/QMLauncher/releases):

- **Linux**: `QMLauncher-cli-linux-amd64`
- **macOS**: `QMLauncher-cli-darwin-amd64` or `QMLauncher-cli-darwin-arm64`
- **Windows**: `QMLauncher-cli-windows-amd64.exe`

### Build from Source

```bash
git clone https://github.com/telecter/QMLauncher.git
cd QMLauncher
make build  # or make linux/macos/windows
```

## 🚀 Usage

### Basic Commands

```bash
# Show help
./QMLauncher-cli --help

# Show version information
./QMLauncher-cli about

# Instance management
./QMLauncher-cli instance list     # List instances
./QMLauncher-cli instance create   # Create new instance
./QMLauncher-cli instance delete   # Delete instance

# Authentication
./QMLauncher-cli auth login        # Login to account
./QMLauncher-cli auth logout       # Logout from account

# Launch game
./QMLauncher-cli instance start <instance>  # Launch instance

# Version search
./QMLauncher-cli search versions   # Find Minecraft versions
./QMLauncher-cli search fabric     # Find Fabric versions
./QMLauncher-cli search forge      # Find Forge versions
```

### Command Line Options

```bash
--verbosity string    Output verbosity level [info, extra, debug] (default "info")
--dir string          Root directory for launcher files
--no-color           Disable color highlighting (also NO_COLOR=1)
```

## 📁 Project Structure

```
.
├── main.go                 # CLI application entry point
├── internal/
│   ├── cli/               # CLI logic and commands
│   │   ├── cmd/          # CLI commands
│   │   └── output/       # Output and localization
│   ├── meta/             # Minecraft metadata
│   └── network/          # Network logic
├── pkg/
│   ├── auth/             # Authentication
│   ├── launcher/         # Launch logic
│   ├── env.go            # Environment variables
│   └── updater/          # Updates
├── go.mod
├── go.sum
└── Makefile              # Build scripts
```

## 🏗️ Building and Development

### Build for Current Platform
```bash
make build
```

### Cross-Platform Builds
```bash
make linux      # Linux AMD64
make macos      # macOS AMD64
make macos-arm64 # macOS ARM64
make windows    # Windows AMD64
make release    # All platforms
```

### Development
```bash
make test       # Run tests
make lint       # Code linting
make fmt        # Code formatting
make vet        # Static analysis
make check      # All checks
```

## 📚 Documentation

- [CHANGELOG.md](CHANGELOG.md) - Change history
- [CHANGELOG_en.md](CHANGELOG_en.md) - Changelog (English)

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Create a Pull Request

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [alecthomas/kong](https://github.com/alecthomas/kong) - CLI framework
- [fatih/color](https://github.com/fatih/color) - Terminal color output
- [jedib0t/go-pretty](https://github.com/jedib0t/go-pretty) - Table formatting
- [schollz/progressbar](https://github.com/schollz/progressbar) - Progress bars

## 📞 Contact

- Author: telecter
- Repository: [github.com/telecter/QMLauncher](https://github.com/telecter/QMLauncher)