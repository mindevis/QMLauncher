# Сборка QMLauncher с настройкой API сервера

## Автоматическая подстановка API URL при сборке

QMLauncher поддерживает автоматическую подстановку адреса API сервера во время сборки. Это позволяет встраивать настройки API непосредственно в скомпилированный код, без необходимости конфигурационных файлов.

## Переменные окружения

При сборке можно указать следующие переменные окружения (рекомендуется использовать `QM_SERVER_API_*`):

- `QM_SERVER_API_HOST` - хост API сервера (по умолчанию: `localhost`)
- `QM_SERVER_API_PORT` - порт API сервера (по умолчанию: `8000`)
- `QM_SERVER_API_PROTOCOL` - протокол (по умолчанию: `http`)
- `QM_SERVER_API_BASE_PATH` - базовый путь API (по умолчанию: `/api/v1`)

Также поддерживаются переменные с префиксом `VITE_` для совместимости с Vite:
- `VITE_API_HOST`
- `VITE_API_PORT`
- `VITE_API_PROTOCOL`
- `VITE_API_BASE_PATH`

## Примеры сборки

### Сборка с настройками по умолчанию
```bash
npm run build
```
Использует: `http://localhost:8000/api/v1`

### Сборка с кастомным API сервером (Linux/macOS)
```bash
QM_SERVER_API_HOST=api.example.com QM_SERVER_API_PORT=443 QM_SERVER_API_PROTOCOL=https npm run build
```

### Сборка с кастомным API сервером (Windows PowerShell)
```powershell
$env:QM_SERVER_API_HOST="api.example.com"
$env:QM_SERVER_API_PORT="443"
$env:QM_SERVER_API_PROTOCOL="https"
npm run build
```

### Сборка для production с указанием API
```bash
QM_SERVER_API_HOST=api.example.com QM_SERVER_API_PORT=443 QM_SERVER_API_PROTOCOL=https npm run dist
```

### Сборка для Windows с кастомным API
```bash
QM_SERVER_API_HOST=api.example.com QM_SERVER_API_PORT=443 QM_SERVER_API_PROTOCOL=https npm run dist:win
```

## Использование в CI/CD

### GitHub Actions
```yaml
- name: Build QMLauncher
  env:
    QM_SERVER_API_HOST: api.example.com
    QM_SERVER_API_PORT: 443
    QM_SERVER_API_PROTOCOL: https
  run: npm run build
```

### GitLab CI
```yaml
build:
  variables:
    QM_SERVER_API_HOST: api.example.com
    QM_SERVER_API_PORT: 443
    QM_SERVER_API_PROTOCOL: https
  script:
    - npm run build
```

## Формат API URL

API URL формируется автоматически по формуле:
```
{PROTOCOL}://{HOST}:{PORT}{BASE_PATH}
```

Примеры:
- `http://localhost:8000/api/v1` (по умолчанию)
- `https://api.example.com:443/api/v1`
- `https://api.example.com/api/v1` (если порт 443, можно не указывать)

## Важные замечания

1. **Значения встраиваются на этапе сборки** - после сборки изменить API URL без пересборки невозможно
2. **Для разных окружений нужны разные сборки** - создавайте отдельные сборки для dev/staging/production
3. **Переменные окружения имеют приоритет** - `QM_SERVER_API_*` > `VITE_*`

## Проверка встроенного API URL

После сборки можно проверить, какой API URL был встроен, посмотрев в скомпилированный код:
- Renderer: `dist/assets/*.js` - поиск по `__QM_SERVER_API_BASE_URL__`
- Electron: `dist-electron/config/api.js` - будет содержать значения из переменных окружения

