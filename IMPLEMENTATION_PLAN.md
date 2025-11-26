# План реализации полного функционала QMLauncher

## ✅ Выполнено (Backend - QMServer):

1. ✅ Обновлена функция `create_config_db` - добавлена таблица `embedded_servers` для хранения данных всех серверов
2. ✅ Обновлен процесс сборки - передача всех серверов в `create_config_db`
3. ✅ Обновлен процесс сборки Docker - передача переменных окружения `QM_SERVER_API_HOST`, `QM_SERVER_API_PORT`, `QM_SERVER_API_PROTOCOL`, `QM_SERVER_API_BASE_PATH` через build args
4. ✅ Обновлены все Dockerfile для поддержки build args с API конфигурацией
5. ✅ Добавлен IPC handler `get-embedded-servers` в Electron main process

## 🔄 В процессе (Frontend - QMLauncher):

1. ✅ Создан модуль `embeddedServers.ts` для чтения встроенных данных
2. ⏳ Реализовать проверку доступности QMServer при запуске
3. ⏳ Реализовать форму авторизации
4. ⏳ Обновить ServersTab для использования встроенных данных
5. ⏳ Реализовать кнопку "Установить" для установки клиента и модов
6. ⏳ Реализовать кнопку "Играть" с автоматическим коннектом

## 📋 Следующие шаги:

### 1. Проверка доступности QMServer
- При запуске лаунчера проверять доступность API по встроенному адресу
- Показывать ошибку если сервер недоступен
- Показывать форму авторизации если сервер доступен

### 2. Форма авторизации
- Компонент `LoginForm.tsx` с полями email и password
- Интеграция с `/api/v1/auth/login` endpoint
- Обработка ошибок (пользователь не существует, неверный пароль)
- Ссылка на регистрацию в QMWeb при ошибке "пользователь не существует"
- Сохранение токена в Electron storage

### 3. Обновление ServersTab
- Использовать встроенные данные из `embedded_servers` таблицы
- Отображать название, описание, логотип, количество игроков
- Кнопка "Установить" если клиент не установлен
- Кнопка "Играть" если клиент установлен

### 4. Установка клиента и модов
- Проверка наличия установленного клиента
- Загрузка и установка Minecraft клиента нужной версии
- Загрузка и установка модов из встроенной конфигурации

### 5. Запуск игры
- Получение параметров запуска с QMServer по server_id и server_uuid
- Запуск Minecraft с правильными аргументами
- Автоматический коннект к серверу по встроенному адресу и порту

## API Endpoints для использования:

- `POST /api/v1/auth/login` - авторизация пользователя
- `GET /api/v1/servers` - получение списка серверов (для обновления данных)
- `GET /api/v1/servers/{server_id}` - получение данных сервера
- `GET /api/v1/servers/{server_id}/mods` - получение списка модов
- `POST /api/v1/minecraft-server/status` - проверка статуса сервера

## Структура данных:

### Embedded Server (из SQLite):
```typescript
{
  server_id: number
  server_uuid: string
  server_name: string | null
  server_address: string | null
  server_port: number | null
  minecraft_version: string | null
  description: string | null
  preview_image_url: string | null
  enabled: number
}
```

### API Config (из SQLite launcher_config):
```typescript
{
  api_base_url: string
  server_id: string
  server_uuid: string
}
```

