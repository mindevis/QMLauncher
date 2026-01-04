# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.1] - 2026-01-04

### Fixed
- Completely restored and optimized auto-release workflow
- Fixed version detection logic in CHANGELOG.md
- Resolved conflicts between auto-release and build workflows
- Improved stability of automatic release creation process

## [1.1.0] - 2026-01-04

### Added
- **NEW**: Full integration with QMServer Cloud for server verification before launch
- **NEW**: Premium server status display in instance and connection tables
- **NEW**: Minecraft client launch progress bar for better user experience
- **NEW**: Automatic data.json generation when files are detected in QMServer data directory
- **NEW**: Interactive profile data management commands in QMServer
- Automatic QMServer Cloud information saving in instance config
- API integration with QMServer Cloud at http://178.172.201.248:8240
- Fully automated release system on GitHub
- Stable CI/CD process with cross-platform builds

### Changed
- Removed timestamp column from recent connections table for compactness
- Updated instance table with QMServer Cloud and Premium status columns
- Improved interactive mode - removed unnecessary command error messages

### Fixed
- Fixed binary file tracking in Git (updated .gitignore)
- Completely fixed GitHub Actions release creation process
- Added GitHub CLI authentication before creating releases
- Separated release creation and asset upload steps for reliability
- Resolved all asset upload issues in releases

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
- Improved build output file handling with post-build relocation
- Integrated shadcn-vue UI component library
- Added Tailwind CSS for modern styling
- Created basic UI component set (Button, Card, Input, Label)
- Updated main application with new component demonstrations
- Added full TypeScript support for Vue components
- Configured TypeScript with proper Vue 3 integration
- Added type checking and IntelliSense support
- Updated build system to use Vite's built-in TypeScript support
- Fixed PostCSS configuration for ES module compatibility
- Updated Vite to v4.5.14 and Vue plugin to v4.6.2
- Resolved vue-tsc compilation issues by using Vite's TypeScript processing
- Integrated QMLauncher CLI functionality for Minecraft launcher features
- Added --no-gui flag for command-line mode operation
- Imported CLI commands: start, instance, auth, search, completions, about
- Maintained backward compatibility with default GUI mode
- Added GitHub Actions CI/CD workflows for automated builds and releases
- Configured cross-platform builds for Linux, macOS, and Windows
- Integrated frontend build process into CI pipeline
- Added automated release artifact uploads
- Created Russian API documentation version (API_ru.md)
- Added instance path column to `instance list` command output
- Added `java list` command to display all installed Java versions
- Changed default working directory from ~/.minecraft to ~/.qmlauncher
- Implemented UUID-based instance isolation with unique directories
- Updated instance structure: ~/.qmlauncher/instances/name/uuid/
- Added UUID generation for each new instance
- Modified instance management functions for new structure
- Added instance export/import functionality
- Added 'instance export' command for exporting instances to ZIP archives
- Added 'instance import' command for importing instances from ZIP archives
- Added 'instance list-exports' command for listing exported archives
- Improved Java executability checking for cross-platform compatibility
- Added NoJavaWindow option for launch parameters

### Changed
- Updated from default Wails template to QMLauncher branding

### Technical Details
- Go Version: 1.23
- Node.js dependencies installed
- Wails CLI v2.11.0 configured
- Vite development server integrated

## [0.1.0] - 2025-12-27

### Added
- Project initialization
- Basic application framework
- Development environment setup
- Documentation files (README, CHANGELOG)
- Multi-platform build configuration