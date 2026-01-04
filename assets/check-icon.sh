#!/bin/bash

# Script to check if icon files exist and are valid

echo "Проверка наличия иконок / Checking for icon files..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check for icon files
ICON_ICO="icon.ico"
ICON_ICNS="icon.icns"
ICON_PNG="icon.png"

missing_files=()

# Check Windows ICO
if [ -f "$ICON_ICO" ]; then
    echo -e "${GREEN}✓${NC} $ICON_ICO найден / found"
else
    echo -e "${RED}✗${NC} $ICON_ICO отсутствует / missing"
    missing_files+=("$ICON_ICO")
fi

# Check macOS ICNS
if [ -f "$ICON_ICNS" ]; then
    echo -e "${GREEN}✓${NC} $ICON_ICNS найден / found"
else
    echo -e "${YELLOW}⚠${NC} $ICON_ICNS отсутствует / missing (optional for macOS builds)"
fi

# Check PNG (for Linux .desktop files and potential conversion)
if [ -f "$ICON_PNG" ]; then
    echo -e "${GREEN}✓${NC} $ICON_PNG найден / found"

    # Check if PNG is a reasonable size for an icon
    if command -v identify >/dev/null 2>&1; then
        size=$(identify -format "%wx%h" "$ICON_PNG" 2>/dev/null | head -1)
        if [ $? -eq 0 ] && [ -n "$size" ]; then
            echo -e "   Размер: $size / Size: $size"
            if [[ $size =~ ^([0-9]+)x([0-9]+)$ ]]; then
                width="${BASH_REMATCH[1]}"
                height="${BASH_REMATCH[2]}"
                if [ "$width" -ge 256 ] && [ "$height" -ge 256 ]; then
                    echo -e "   ${GREEN}✓${NC} Подходящий размер для иконки / Suitable size for icon"
                else
                    echo -e "   ${YELLOW}⚠${NC} Рекомендуется размер не менее 256x256 / Recommended size 256x256 or larger"
                fi
            fi
        fi
    else
        echo -e "   ${YELLOW}⚠${NC} ImageMagick не установлен, пропуск проверки размера / ImageMagick not installed, skipping size check"
    fi
else
    echo -e "${YELLOW}⚠${NC} $ICON_PNG отсутствует / missing (recommended for Linux)"
fi

echo
echo "Сводка / Summary:"

# Check if we have at least PNG (required for Linux)
png_found=false
if [ -f "$ICON_PNG" ]; then
    png_found=true
fi

if [ ${#missing_files[@]} -eq 0 ]; then
    echo -e "${GREEN}Все файлы найдены! / All files found!${NC}"
    exit 0
elif [ "$png_found" = true ]; then
    echo -e "${GREEN}PNG найден - достаточно для Linux сборки / PNG found - sufficient for Linux builds${NC}"
    echo -e "${YELLOW}Для Windows/macOS запустите: ./convert-icons.sh${NC}"
    exit 0
else
    echo -e "${RED}Отсутствуют файлы: ${missing_files[*]}${NC}"
    echo "Добавьте недостающие файлы в папку assets/"
    echo "Или используйте ./convert-icons.sh для конвертации из PNG"
    exit 1
fi