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
- Updated build system to include TypeScript compilation

### Changed
- Updated from default Wails template to QMLauncher branding

### Technical Details
- Go version: 1.23
- Node.js dependencies installed
- Wails CLI v2.11.0 configured
- Vite development server integrated

### Changed
- Modified files: test_file.txt 
### Changed
- Modified files: test_file.txt 
### Changed
- Modified files: scripts/update_changelog.sh 
## [0.1.0] - 2025-12-27

### Added
- Project initialization
- Basic application framework
- Development environment setup
- Documentation files (README, CHANGELOG)
- Build configuration for multiple platforms
