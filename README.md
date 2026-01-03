# QMLauncher

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Минималистичный лаунчер Minecraft для командной строки.

## 📋 Описание

QMLauncher - это кроссплатформенное приложение командной строки для запуска Minecraft. Предоставляет удобный интерфейс для управления инстансами Minecraft, аутентификации, поиска версий и управления Java runtime.

## ✨ Возможности

- 🚀 **Управление инстансами** - создание, настройка и запуск инстансов Minecraft
- 🔐 **Аутентификация** - поддержка различных методов аутентификации Mojang/Microsoft
- 🔍 **Поиск версий** - поиск и установка версий Minecraft и модлоадеров
- ☕ **Управление Java** - автоматическое обнаружение и управление установками Java
- 📦 **Автообновления** - встроенная система обновлений приложения
- 🌍 **Локализация** - поддержка русского и английского языков

## 🛠️ Установка

### Скачивание релиза

Скачайте подходящую версию для вашей платформы из [релизов](https://github.com/telecter/QMLauncher/releases):

- **Linux**: `QMLauncher-cli-linux-amd64`
- **macOS**: `QMLauncher-cli-darwin-amd64` или `QMLauncher-cli-darwin-arm64`
- **Windows**: `QMLauncher-cli-windows-amd64.exe`

### Сборка из исходников

```bash
git clone https://github.com/telecter/QMLauncher.git
cd QMLauncher
make build  # или make linux/macos/windows
```

## 🚀 Использование

### Основные команды

```bash
# Показать справку
./QMLauncher-cli --help

# Показать информацию о версии
./QMLauncher-cli about

# Управление инстансами (с алиасами)
./QMLauncher-cli -i list           # Список инстансов (-i = instance)
./QMLauncher-cli instance list     # Альтернативный вариант
./QMLauncher-cli -i create         # Создать новый инстанс
./QMLauncher-cli -i delete         # Удалить инстанс

# Аутентификация
./QMLauncher-cli auth login        # Войти в аккаунт
./QMLauncher-cli auth logout       # Выйти из аккаунта

# Запуск игры (быстрые алиасы)
./QMLauncher-cli -is <instance>    # Запустить инстанс (-is = instance start)
./QMLauncher-cli -i -s <instance>  # Альтернативный вариант
./QMLauncher-cli instance start <instance>  # Полная форма

# Примеры с опциями запуска
./QMLauncher-cli -is 'qDev RPG' -u Devis --server 178.172.172.41:25565 --min-memory=4096 --max-memory=6192
./QMLauncher-cli -i -s 'qDev RPG' -u Devis --server 178.172.172.41:25565 --min-memory=4096 --max-memory=6192

# Поиск версий
./QMLauncher-cli search versions   # Найти версии Minecraft
./QMLauncher-cli search fabric     # Найти версии Fabric
./QMLauncher-cli search forge      # Найти версии Forge
```

### Опции командной строки

```bash
--verbosity string    Уровень подробности вывода [info, extra, debug] (default "info")
--dir string          Корневая директория для файлов лаунчера
--no-color           Отключить цветовую подсветку (также NO_COLOR=1)
```

## 📁 Структура проекта

```
.
├── main.go                 # Точка входа CLI приложения
├── internal/
│   ├── cli/               # CLI логика и команды
│   │   ├── cmd/          # Команды CLI
│   │   └── output/       # Вывод и локализация
│   ├── meta/             # Метаданные Minecraft
│   └── network/          # Сетевая логика
├── pkg/
│   ├── auth/             # Аутентификация
│   ├── launcher/         # Логика запуска
│   ├── env.go            # Переменные окружения
│   └── updater/          # Обновления
├── go.mod
├── go.sum
└── Makefile              # Скрипты сборки
```

## 🏗️ Сборка и разработка

### Сборка для текущей платформы
```bash
make build
```

### Кроссплатформенная сборка
```bash
make linux      # Linux AMD64
make macos      # macOS AMD64
make macos-arm64 # macOS ARM64
make windows    # Windows AMD64
make release    # Все платформы
```

### Разработка
```bash
make test       # Запуск тестов
make lint       # Линтинг кода
make fmt        # Форматирование
make vet        # Статический анализ
make check      # Все проверки
```

## 📚 Документация

- [CHANGELOG.md](CHANGELOG.md) - История изменений
- [CHANGELOG_en.md](CHANGELOG_en.md) - Changelog (English)

## 🤝 Вклад в проект

1. Форкните репозиторий
2. Создайте ветку для вашей фичи (`git checkout -b feature/AmazingFeature`)
3. Зафиксируйте изменения (`git commit -m 'Add some AmazingFeature'`)
4. Запушьте ветку (`git push origin feature/AmazingFeature`)
5. Создайте Pull Request

## 📄 Лицензия

Этот проект распространяется под лицензией MIT. Смотрите файл [LICENSE](LICENSE) для подробностей.

## 🙏 Благодарности

- [alecthomas/kong](https://github.com/alecthomas/kong) - CLI фреймворк
- [fatih/color](https://github.com/fatih/color) - Цветной вывод в терминале
- [jedib0t/go-pretty](https://github.com/jedib0t/go-pretty) - Форматирование таблиц
- [schollz/progressbar](https://github.com/schollz/progressbar) - Прогресс-бары

## 📞 Контакты

- Автор: telecter
- Репозиторий: [github.com/telecter/QMLauncher](https://github.com/telecter/QMLauncher)