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
2. Соберет Windows исполняемый файл (`wails build -platform windows/amd64`)

Результат будет в директории `build/bin/`

## Manual Build

Если нужно собрать вручную:

```bash
# 1. Собрать фронтенд
cd frontend
npm run build
cd ..

# 2. Собрать Windows executable
wails build -platform windows/amd64
```

## Requirements

- Node.js и npm (для сборки фронтенда)
- Go (для сборки бэкенда)
- Wails CLI (установлен через `go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

