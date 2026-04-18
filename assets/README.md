# Assets / Ресурсы QMLauncher

Эта папка содержит статические ресурсы приложения.

## ✅ Система готова! / System is ready!

**Все платформы теперь автоматически собираются с иконками!**

| Платформа | Формат иконки | Статус |
|-----------|---------------|--------|
| **Windows** | ICO (встроен) | ✅ Работает |
| **macOS** | ICNS (встроен) | ✅ Работает |
| **Linux** | PNG (скопирован) | ✅ Работает |

## 🚀 Быстрый старт / Quick Start

Просто запускайте сборку - иконки добавятся автоматически:

```bash
# Сборка для текущей платформы
make build

# Сборка для всех платформ
make release
```

### Ручная настройка (опционально) / Manual setup (optional)

1. **Конвертация иконок вручную:**
   ```bash
   ./assets/convert-icons.sh
   ```

2. **Проверка статуса:**
   ```bash
   ./assets/check-icon.sh
   ```

## Иконка приложения / Application Icon

### Windows (.ico)
- **Файл:** `icon.ico`
- **Размер:** Рекомендуется 256x256 пикселей
- **Формат:** ICO с несколькими размерами (16x16, 32x32, 48x48, 256x256)

### macOS (.icns)
- **Файл:** `icon.icns`
- **Формат:** Apple ICNS

### Linux (.png)
- **Файл:** `icon.png`
- **Размер:** 512x512 пикселей
- **Формат:** PNG с прозрачностью

## Использование / Usage

Иконки автоматически включаются в исполняемый файл при сборке с помощью соответствующих флагов компилятора.

### Сборка через Makefile

```bash
# С иконкой (если поддерживается)
make build-with-icon

# Обычная сборка
make build
```

## 🤖 CI/CD Автоматизация / CI/CD Automation

### GitHub Actions
Сборка и проверки выполняются в `.github/workflows/ci.yml` (тесты, линт, порог покрытия на `main`). Иконки подключаются локальными шагами `Makefile` / скриптами в этом каталоге.

**Поддерживаемые платформы:**
- ✅ **Ubuntu**: ImageMagick + icnsutils
- ✅ **macOS**: Встроенные инструменты + ImageMagick
- ✅ **Windows**: ImageMagick (через Chocolatey)

### Тестирование CI/CD локально / Local CI/CD Testing
```bash
# Протестировать процесс сборки как в GitHub Actions
.github/workflows/test-local-build.sh
```

---

### Ручная сборка / Manual Build

#### Windows
```bash
# Если есть ICO файл
go build -ldflags "-H windowsgui -icon assets/icon.ico" -o qmlauncher.exe

# Или конвертировать PNG в ICO и собрать
./assets/convert-icons.sh
go build -ldflags "-H windowsgui -icon assets/icon.ico" -o qmlauncher.exe
```

#### macOS
```bash
# Если есть ICNS файл
go build -ldflags "-icon assets/icon.icns" -o qmlauncher

# Или конвертировать PNG в ICNS и собрать
./assets/convert-icons.sh
go build -ldflags "-icon assets/icon.icns" -o qmlauncher
```

#### Linux
Иконка обычно указывается в .desktop файле, а не встраивается в исполняемый файл:
```bash
go build -o qmlauncher
```

### Проверка иконок

Используйте скрипт проверки перед сборкой:

```bash
# Из корневой директории проекта
./assets/check-icon.sh

# Или из папки assets
cd assets && ./check-icon.sh
```

Пример вывода:
```
Проверка наличия иконок / Checking for icon files...
✓ icon.ico найден / found
⚠ icon.icns отсутствует / missing (optional for macOS builds)
⚠ icon.png отсутствует / missing (recommended for Linux)

Сводка / Summary:
Отсутствуют файлы: icon.ico
```

### Проверка поддержки иконок

```bash
# Проверить версию Go
go version

# Для Windows может потребоваться установка дополнительных инструментов
go install github.com/akavel/rsrc@latest

# Проверить наличие иконки перед сборкой
ls -la assets/

# Или использовать скрипт проверки
./assets/check-icon.sh
```

### Зависимости / Dependencies

Для сборки с иконками могут потребоваться дополнительные инструменты:

#### Windows
- **rsrc** для встраивания ресурсов: `go install github.com/akavel/rsrc@latest`

#### macOS
- Стандартные инструменты разработки Xcode

#### Linux
- Иконки обычно указываются в .desktop файлах, не встраиваются в исполняемый файл

### Примеры .desktop файла для Linux

Готовый шаблон .desktop файла находится в `assets/.desktop`. Скопируйте его и настройте пути:

```bash
# Скопировать шаблон
cp assets/.desktop ~/.local/share/applications/qmlauncher.desktop

# Отредактировать пути
nano ~/.local/share/applications/qmlauncher.desktop
```

Пример содержимого .desktop файла:

```ini
[Desktop Entry]
Name=QMLauncher
Comment=Minecraft launcher with mod support
Exec=/usr/local/bin/qmlauncher
Icon=/usr/local/share/qmlauncher/icon.png
Terminal=false
Type=Application
Categories=Game;
Keywords=minecraft;launcher;mod;
```

### Установка иконки для Linux

```bash
# Создать директорию для иконок
mkdir -p /usr/local/share/qmlauncher

# Скопировать иконку
cp assets/icon.png /usr/local/share/qmlauncher/

# Обновить кэш иконок
update-icon-caches /usr/local/share/icons/*
```

## Создание иконок / Icon Creation

### Из PNG файла
```bash
# Windows ICO
convert icon.png -define icon:auto-resize=256,128,64,48,32,16 icon.ico

# macOS ICNS
png2icns icon.icns icon.png
```

### Онлайн инструменты
- [Favicon.io](https://favicon.io/favicon-converter/)
- [RealFaviconGenerator](https://realfavicongenerator.net/)
- [IconConverter](https://iconconverter.com/)

## Примечания / Notes

- Убедитесь, что иконка имеет правильный размер и формат для целевой платформы
- Для Windows рекомендуется ICO с несколькими размерами
- Для macOS важен ICNS формат
- Иконки должны иметь квадратную форму для лучшего отображения