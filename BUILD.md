# Build Instructions

## Windows Build

Для сборки Windows исполняемого файла используйте скрипт:

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
2. Соберет Windows исполняемый файл с обфускацией (`wails build -platform windows/amd64 -obfuscated`)

Результат будет в директории `build/bin/` в виде исполняемого файла (`.exe` файл).

## Manual Build

Если нужно собрать вручную:

```bash
# 1. Собрать фронтенд
cd frontend
npm run build
cd ..

# 2. Собрать Windows исполняемый файл с обфускацией
wails build -platform windows/amd64 -obfuscated
```

**Примечание:** Флаг `-obfuscated` используется для всех платформ и обфусцирует код с помощью garble для защиты от реверс-инжиниринга. Без этого флага будет создан обычный исполняемый файл без обфускации.

## Linux Build

Для сборки Linux пакета:

```bash
# 1. Собрать фронтенд
cd frontend
npm run build
cd ..

# 2. Собрать Linux AppImage с обфускацией
wails build -platform linux/amd64 -obfuscated
```

Результат будет в директории `build/bin/` в виде AppImage файла (`.AppImage`) с обфусцированным кодом.

## macOS Build

Для сборки macOS пакета:

```bash
# 1. Собрать фронтенд
cd frontend
npm run build
cd ..

# 2. Собрать macOS disk image с обфускацией
wails build -platform darwin/amd64 -obfuscated
```

Результат будет в директории `build/bin/` в виде disk image (`.dmg`) или package (`.pkg`) с обфусцированным кодом.

## Requirements

- Node.js и npm (для сборки фронтенда)
- Go (для сборки бэкенда)
- Wails CLI (установлен через `go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- **Для обфускации**: Garble (установлен через `go install mvdan.cc/garble@latest`)
- **Для Linux пакетов**: `dpkg-deb` для `.deb`, `rpmbuild` для `.rpm` (опционально, по умолчанию создается AppImage)
- **Для macOS пакетов**: Xcode Command Line Tools (если сборка выполняется на macOS)

