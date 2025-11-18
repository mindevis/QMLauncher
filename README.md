# QMLauncher

A Minecraft launcher built with PySide6 (Qt for Python).

## Features

- **Version Management**: Browse and install Minecraft versions
- **Profile Management**: Create and manage multiple launch profiles
- **Customizable Settings**: Configure Java path and memory settings
- **Modern GUI**: Clean and intuitive user interface

## Requirements

- Python 3.8+
- PySide6
- requests

## Installation

1. Install dependencies:
```bash
pip install -r requirements.txt
```

2. Run the launcher:
```bash
python main.py
```

## Configuration

The launcher stores its configuration and game files in `~/.qmlauncher/`:
- `versions/` - Installed Minecraft versions
- `profiles/` - Launch profiles
- `assets/` - Game assets
- `libraries/` - Game libraries
- `config.json` - Launcher configuration

## Usage

1. **Versions Tab**: Browse available Minecraft versions and install them
2. **Profiles Tab**: Create profiles with custom settings (version, memory, username)
3. **Settings Tab**: Configure Java path and default memory settings
4. **Launch**: Click "Launch Minecraft" button to start the game

## Development

The project is structured as follows:
- `main.py` - Entry point
- `core/` - Core functionality (config, version management, profiles, launcher)
- `ui/` - User interface components

## License

This project is part of the QMProject ecosystem.

