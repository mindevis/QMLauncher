# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.5.2] - 2026-01-04

### Added
- **NEW**: Automatic data.json generation when files are detected in QMServer data directory
- Interactive profile data management commands in QMServer

### Changed
- Improved GitHub Actions release creation process with separate stages
- Added additional debug information in release workflow

### Fixed
- Fixed missing data directories issue in QMServer
- Stabilized asset upload process for GitHub releases

## [1.5.1] - 2026-01-04

### Fixed
- Stabilized automatic release creation process in GitHub Actions
- Improved artifact handling across different platforms
- Fixed binary file paths in release workflow

## [1.5.0] - 2026-01-04

### Added
- **NEW**: Automatic release creation on GitHub when version tags are pushed
- GitHub Actions integration for automated release publishing
- Cross-platform release support (Linux, macOS, Windows)

### Changed
- Improved interactive mode - removed unnecessary command error messages
- Optimized build and publishing process

### Fixed
- Fixed binary file tracking in Git (updated .gitignore)
- Removed annoying "Command execution error" messages in interactive mode

## [1.4.0] - 2026-01-04

### Added
- **NEW**: QMServer Cloud integration for server verification before launch
- **NEW**: Premium server status display in instance and connection tables
- **NEW**: Minecraft client launch progress bar for better user experience
- Automatic QMServer Cloud information saving in instance config
- API integration with QMServer Cloud at http://178.172.201.248:8240

### Changed
- Removed timestamp column from recent connections table for compactness
- Updated instance table with QMServer Cloud and Premium status columns

### Fixed
- Fixed GitHub Actions workflow for cross-platform builds (added shell: bash)

## [1.3.0] - 2026-01-03

### Changed
- **BREAKING**: Project converted to CLI-only mode, GUI support (Wails) removed
- Simplified architecture - command line only without desktop interface
- Optimized dependencies, removed Wails and frontend packages
- Makefile updated for CLI builds only

### Removed
- Completely removed GUI mode and Wails integration
- Removed frontend part (React/TypeScript)
- Removed desktop-specific dependencies and configurations

## [1.2.0] - 2026-01-03

### Added
- **NEW**: Full command history in interactive mode with arrow key navigation ↑/↓
- **NEW**: Convenient command aliases: `i` (instance), `s` (start), `is` (instance start)
- **NEW**: Minecraft log output control - quiet mode by default, detailed logs with `--verbosity=extra`
- **NEW**: Recent server connections system with quick launch by number
- **NEW**: Last server and username saved in instance config (.toml)
- Quote support in command arguments for names with spaces
- Improved launch and status messages

### Changed
- Launch message: "Launching Minecraft client with account: %s"
- Interactive mode now shows recent connections table on startup
- Optimized command output without duplication

### Fixed
- Eliminated command duplication in interactive mode
- Fixed quote parsing in arguments
- Improved command history navigation

## [1.1.0] - 2026-01-02

### Added
- **NEW**: Comprehensive auto-update system with GitHub Releases integration
- **NEW**: Smart instance import with merge mode for adding missing files only
- UI update notifications with download progress and changelog display
- CLI commands for update management: `qm update check`, `qm update download`, `qm update info`
- Automatic binary replacement and application restart after updates
- Cross-platform update support for Windows, macOS, and Linux
- Enhanced instance import functionality with `--merge` flag
- Automatic release system triggered by CHANGELOG.md updates
- Improved import error messages with clear guidance for users
- Cross-platform path normalization in imported instance files (Windows/Linux)
- Automatic conversion of backslashes to forward slashes in text configuration files during import
- Full TypeScript configuration with strict type checking
- Custom hooks for Wails runtime and backend communication
- React Context providers for theme and app state management
- Comprehensive type definitions for Wails integration
- Enhanced development workflow with TypeScript tooling

### Changed
- **BREAKING**: Removed German language support, set Russian as default language
- Simplified localization system to support only English and Russian
- Russian language is now the primary language with English as fallback
- Russian is now the default language for CHANGELOG.md and README.md
- **BREAKING**: Migrated frontend from Vue.js to React with full TypeScript support
- Replaced shadcn-vue with shadcn/ui components
- Updated build system to use Vite with React instead of Vue
- Enhanced type safety with comprehensive TypeScript definitions
- Added custom hooks for Wails backend integration
- Implemented React Context for global state management
- Improved instance import commands with better error handling and user guidance

## [1.0.0] - 2025-12-27

### Added
- Initial project setup with Wails v2.11.0
- Vue.js 3 frontend with Vite build system
- Go backend with basic application structure
- Cross-platform desktop application support
- Hot reload development environment
- Basic UI components and styling
- Comprehensive Makefile with build, dev, and quality assurance commands
- Go linting with golangci-lint and code formatting tools
- Frontend linting with ESLint and Prettier
- Updated .gitignore with comprehensive exclusions
- Enhanced cross-platform build system with architecture-specific commands
- Automated version detection and build naming
- Convenient make targets for Linux, macOS, and Windows builds
- Fixed build directory duplication issue (build/bin/build/...)
- Added cross-compilation checks for unsupported platform combinations
- Improved build output handling with post-build file moving
- Integrated shadcn-vue UI component library
- Added Tailwind CSS for modern styling
- Created base UI components (Button, Card, Input, Label)
- Updated main application with new component showcase
- Added full TypeScript support for Vue components
- Configured TypeScript with proper Vue 3 integration
- Added type checking and IntelliSense support
- Updated build system to use Vite's built-in TypeScript support
- Fixed PostCSS configuration for ES modules compatibility
- Updated Vite to v4.5.14 and Vue plugin to v4.6.2
- Resolved vue-tsc compilation issues by using Vite's TypeScript handling
- Integrated QMLauncher CLI functionality for Minecraft launcher features
- Added --no-gui flag for command-line mode operation
- Imported CLI commands: start, instance, auth, search, completions, about
- Maintained backward compatibility with GUI mode as default
- Added GitHub Actions CI/CD workflows for automated building and releasing
- Configured cross-platform builds for Linux, macOS, and Windows
- Integrated frontend build process in CI pipeline
- Added automated release artifact uploads
- Created Russian translation of API documentation (API_ru.md)
- Added instance path column to `instance list` command output
- Added `java list` command to display all installed Java versions
- Changed default working directory from ~/.minecraft to ~/.qmlauncher
- Implemented UUID-based instance isolation with unique directories
- Updated instance structure: ~/.qmlauncher/instances/name/uuid/
- Added UUID generation for each new instance
- Modified instance management functions to work with new structure
- Added instance export/import functionality
- Added 'instance export' command to export instances to ZIP archives
- Added 'instance import' command to import instances from ZIP archives
- Added 'instance list-exports' command to list exported archives
- Improved Java executable validation for cross-platform compatibility
- Added NoJavaWindow option to launch options

### Changed
- Updated from default Wails template to QMLauncher branding

### Technical Details
- Go version: 1.23
- Node.js dependencies installed
- Wails CLI v2.11.0 configured
- Vite development server integrated

## [0.1.0] - 2025-12-27

### Added
- Project initialization
- Basic application framework
- Development environment setup
- Documentation files (README, CHANGELOG)
- Build configuration for multiple platforms
