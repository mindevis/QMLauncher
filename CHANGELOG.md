## 2025-11-29

### Added
- Encrypted configuration system with AES-256-GCM encryption
- Automatic synchronization with QMServer on first launch
- Config update detection and automatic sync
- Config manager for encrypted data storage

### Changed
- Replaced SQLite database with encrypted JSON config file
- Removed plugin support (launcher now works only with mods)
- Config data is now fetched from QMServer and cached locally in encrypted format
- Launcher settings are stored in encrypted config.json

### Removed
- SQLite database (better-sqlite3 dependency)
- Plugin management functionality
- Direct API calls for server/mod data (now uses cached encrypted config)

## 2024-11-28

### Fixed
- Fixed import path in select component (use alias @/lib/utils instead of relative path)
- Fixed launcher build process error handling and logging

## 2024-01-01

### Added
- Initial release of QMLauncher
- Server selection and management
- Game account management
- Automatic mod and resource downloads
- Launcher auto-update functionality
- Basic server connection
- Game launch functionality

### Changed
- Improved UI/UX design
- Enhanced error handling

### Fixed
- Fixed connection issues with QMServer
- Fixed authentication flow
