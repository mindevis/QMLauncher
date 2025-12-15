# Build Instructions

## Windows Build

Для сборки Windows установщика используйте скрипт:

```bash
./build-windows.sh
```

Или напрямую:

```bash
chmod +x build-windows.sh
./build-windows.sh
```

Скрипт автоматически:
1. Соберет фронтенд (`npm run build` в директории `frontend/`)
2. Соберет Windows установщик (`wails build -platform windows/amd64 -nsis`)

Результат будет в директории `build/bin/` в виде NSIS установщика (`.msi` файл).

## Manual Build

Если нужно собрать вручную:

```bash
# 1. Собрать фронтенд
cd frontend
npm run build
cd ..

# 2. Собрать Windows установщик
wails build -platform windows/amd64 -nsis
```

**Примечание:** С флагом `-nsis` будет создан установочный пакет (`.msi`). Без этого флага будет создан только исполняемый файл (`.exe`). В `wails.json` настроено `"nsisType": "nsis"`, что означает создание установщика по умолчанию.

## Linux Build

Для сборки Linux пакета:

```bash
# 1. Собрать фронтенд
cd frontend
npm run build
cd ..

# 2. Собрать Linux AppImage
wails build -platform linux/amd64
```

Результат будет в директории `build/bin/` в виде AppImage файла (`.AppImage`).

## macOS Build

Для сборки macOS пакета:

```bash
# 1. Собрать фронтенд
cd frontend
npm run build
cd ..

# 2. Собрать macOS disk image
wails build -platform darwin/amd64
```

Результат будет в директории `build/bin/` в виде disk image (`.dmg`) или package (`.pkg`).

## Requirements

- Node.js и npm (для сборки фронтенда)
- Go (для сборки бэкенда)
- Wails CLI (установлен через `go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- **Для Windows установщиков**: NSIS (автоматически обрабатывается Wails)
- **Для Linux пакетов**: `dpkg-deb` для `.deb`, `rpmbuild` для `.rpm` (опционально, по умолчанию создается AppImage)
- **Для macOS пакетов**: Xcode Command Line Tools (если сборка выполняется на macOS)

