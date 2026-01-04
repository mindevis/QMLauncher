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
- 🌐 **QMServer Cloud** - QMServer Cloud integration for server verification and premium status display
- 🌍 **Localization** - support for Russian and English languages

## 🛠️ Installation

### Download Release

Download the appropriate version for your platform from [releases](https://github.com/mindevis/QMLauncher/releases):

- **Linux**: `QMLauncher-cli-linux-amd64`
- **macOS**: `QMLauncher-cli-darwin-amd64` or `QMLauncher-cli-darwin-arm64`
- **Windows**: `QMLauncher-cli-windows-amd64.exe`

### Build from Source

```bash
git clone https://github.com/mindevis/QMLauncher.git
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

# Instance management (with aliases)
./QMLauncher-cli -i list           # List instances (-i = instance)
./QMLauncher-cli instance list     # Alternative form
./QMLauncher-cli -i create         # Create new instance
./QMLauncher-cli -i delete         # Delete instance

# Authentication
./QMLauncher-cli auth login        # Login to account
./QMLauncher-cli auth logout       # Logout from account

# Launch game (quick aliases)
./QMLauncher-cli -is <instance>    # Launch instance (-is = instance start)
./QMLauncher-cli -i -s <instance>  # Alternative form
./QMLauncher-cli instance start <instance>  # Full form

# Examples with launch options
./QMLauncher-cli -is 'qDev RPG' -u Devis --server 178.172.172.41:25565 --min-memory=4096 --max-memory=6192
./QMLauncher-cli -i -s 'qDev RPG' -u Devis --server 178.172.172.41:25565 --min-memory=4096 --max-memory=6192

# Version search
./QMLauncher-cli search versions   # Find Minecraft versions
./QMLauncher-cli search fabric     # Find Fabric versions
./QMLauncher-cli search forge      # Find Forge versions
```

### Interactive Mode

QMLauncher supports interactive mode for convenient operation:

```bash
# Windows: automatic interactive mode launch
QMLauncher.exe

# Linux/macOS: automatic interactive mode launch
./QMLauncher-cli

# Force interactive mode (even with arguments)
./QMLauncher-cli --interactive
```

#### Interactive mode features:
- **Management commands**:
  - `help`, `h`, `?` - show help
  - `exit`, `quit`, `q` - exit interactive mode
- **Command aliases** - `-i`, `-s`, `-is` work as usual
- **Auto-completion** - convenient command input

```bash
QMLauncher> -i list          # Show instances
QMLauncher> -is "My Server"   # Launch instance
QMLauncher> help             # Show help
QMLauncher> exit             # Exit
```

#### Memory Settings Persistence

Memory settings (`--min-memory`, `--max-memory`) are saved in the instance configuration (`instance.toml`). When you first launch with memory parameters, they are saved and used in subsequent launches:

```bash
# First launch with memory settings - saves to instance config
./QMLauncher-cli instance start "My Server" --min-memory=4096 --max-memory=8192

# Subsequent launches will use saved settings
./QMLauncher-cli instance start "My Server"

# You can change settings anytime
./QMLauncher-cli instance start "My Server" --max-memory=16384
```

### Command Line Options

```bash
--verbosity string    Output verbosity level [info, extra, debug] (default "info")
--dir string          Root directory for launcher files
--no-color           Disable color highlighting (also NO_COLOR=1)
--interactive        Start in interactive mode
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

- Author: mindevis
- Repository: [github.com/mindevis/QMLauncher](https://github.com/mindevis/QMLauncher)