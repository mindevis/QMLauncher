# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
