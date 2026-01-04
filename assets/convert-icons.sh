#!/bin/bash

# Script to convert PNG icon to various formats needed for different platforms

echo "Конвертация иконок / Converting icons..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SOURCE_PNG="icon.png"
TARGET_ICO="icon.ico"
TARGET_ICNS="icon.icns"

# Check if source PNG exists
if [ ! -f "$SOURCE_PNG" ]; then
    echo -e "${RED}Ошибка: $SOURCE_PNG не найден${NC}"
    echo -e "${RED}Error: $SOURCE_PNG not found${NC}"
    exit 1
fi

echo -e "${BLUE}Исходный файл: $SOURCE_PNG${NC}"
echo -e "${BLUE}Source file: $SOURCE_PNG${NC}"
echo

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Convert to Windows ICO
echo "Конвертация в ICO для Windows / Converting to ICO for Windows..."
if command_exists "convert"; then
    echo "Используем ImageMagick / Using ImageMagick..."
    convert "$SOURCE_PNG" \
        -define icon:auto-resize=256,128,64,48,32,16 \
        -colors 256 \
        "$TARGET_ICO" 2>/dev/null

    if [ $? -eq 0 ] && [ -f "$TARGET_ICO" ]; then
        echo -e "${GREEN}✓ ICO создан: $TARGET_ICO${NC}"
        ico_size=$(ls -lh "$TARGET_ICO" | awk '{print $5}')
        echo -e "  Размер файла: $ico_size"
    else
        echo -e "${RED}✗ Ошибка создания ICO${NC}"
    fi
elif command_exists "magick"; then
    echo "Используем ImageMagick v7 / Using ImageMagick v7..."
    magick "$SOURCE_PNG" \
        -define icon:auto-resize=256,128,64,48,32,16 \
        -colors 256 \
        "$TARGET_ICO" 2>/dev/null

    if [ $? -eq 0 ] && [ -f "$TARGET_ICO" ]; then
        echo -e "${GREEN}✓ ICO создан: $TARGET_ICO${NC}"
        ico_size=$(ls -lh "$TARGET_ICO" | awk '{print $5}')
        echo -e "  Размер файла: $ico_size"
    else
        echo -e "${RED}✗ Ошибка создания ICO${NC}"
    fi
else
    echo -e "${YELLOW}⚠ ImageMagick не найден. Установите для конвертации ICO.${NC}"
    echo -e "${YELLOW}⚠ ImageMagick not found. Install for ICO conversion.${NC}"
    echo "  Ubuntu/Debian: sudo apt install imagemagick"
    echo "  CentOS/RHEL: sudo yum install ImageMagick"
    echo "  macOS: brew install imagemagick"
fi

echo

# Convert to macOS ICNS
echo "Конвертация в ICNS для macOS / Converting to ICNS for macOS..."

# Try different tools in order of preference
icns_created=false

# Method 1: Try png2icns (if available)
if command_exists "png2icns" && [ "$icns_created" = false ]; then
    echo "Пробуем png2icns / Trying png2icns..."
    png2icns "$TARGET_ICNS" "$SOURCE_PNG" 2>/dev/null

    if [ $? -eq 0 ] && [ -f "$TARGET_ICNS" ]; then
        echo -e "${GREEN}✓ ICNS создан с помощью png2icns: $TARGET_ICNS${NC}"
        icns_size=$(ls -lh "$TARGET_ICNS" | awk '{print $5}')
        echo -e "  Размер файла: $icns_size"
        icns_created=true
    else
        echo -e "${YELLOW}⚠ png2icns не сработал, пробуем другие инструменты...${NC}"
        echo -e "${YELLOW}⚠ png2icns failed, trying other tools...${NC}"
    fi
fi

# Method 2: Try iconutil + sips (macOS built-in)
if command_exists "iconutil" && command_exists "sips" && [ "$icns_created" = false ]; then
    echo "Пробуем iconutil + sips (macOS) / Trying iconutil + sips (macOS)..."

    # Create iconset directory
    ICONSET_DIR="icon.iconset"
    mkdir -p "$ICONSET_DIR"

    # Generate different sizes using sips
    echo "  Генерируем размеры иконок / Generating icon sizes..."
    SIZES=("16x16" "32x32" "128x128" "256x256" "512x512")
    for size in "${SIZES[@]}"; do
        width_height=$size
        sips -z "$width_height" "$SOURCE_PNG" --out "$ICONSET_DIR/icon_${size}.png" >/dev/null 2>&1
        if [ $? -ne 0 ]; then
            echo -e "${YELLOW}    Предупреждение: не удалось создать ${size}${NC}"
        fi
    done

    # Also create @2x versions for retina
    RETINA_SIZES=("32x32" "64x64" "256x256" "512x512" "1024x1024")
    for i in "${!SIZES[@]}"; do
        size="${SIZES[$i]}"
        retina_size="${RETINA_SIZES[$i]}"
        sips -z "$retina_size" "$SOURCE_PNG" --out "$ICONSET_DIR/icon_${size}@2x.png" >/dev/null 2>&1
    done

    # Create ICNS using iconutil
    echo "  Создаем ICNS файл / Creating ICNS file..."
    iconutil -c icns "$ICONSET_DIR" -o "$TARGET_ICNS" 2>/dev/null

    # Cleanup
    rm -rf "$ICONSET_DIR"

    if [ $? -eq 0 ] && [ -f "$TARGET_ICNS" ]; then
        echo -e "${GREEN}✓ ICNS создан с помощью iconutil: $TARGET_ICNS${NC}"
        icns_size=$(ls -lh "$TARGET_ICNS" | awk '{print $5}')
        echo -e "  Размер файла: $icns_size"
        icns_created=true
    else
        echo -e "${YELLOW}⚠ iconutil не сработал${NC}"
    fi
fi

# Method 3: Try ImageMagick (fallback)
if (command_exists "convert" || command_exists "magick") && [ "$icns_created" = false ]; then
    echo "Пробуем ImageMagick / Trying ImageMagick..."

    # ImageMagick can create ICNS, but it's experimental
    if command_exists "convert"; then
        convert "$SOURCE_PNG" "$TARGET_ICNS" 2>/dev/null
    else
        magick "$SOURCE_PNG" "$TARGET_ICNS" 2>/dev/null
    fi

    if [ $? -eq 0 ] && [ -f "$TARGET_ICNS" ]; then
        echo -e "${GREEN}✓ ICNS создан с помощью ImageMagick: $TARGET_ICNS${NC}"
        echo -e "${YELLOW}  Предупреждение: ICNS от ImageMagick может не работать корректно${NC}"
        icns_size=$(ls -lh "$TARGET_ICNS" | awk '{print $5}')
        echo -e "  Размер файла: $icns_size"
        icns_created=true
    else
        echo -e "${YELLOW}⚠ ImageMagick тоже не сработал${NC}"
    fi
fi

# Final result
if [ "$icns_created" = false ]; then
    echo -e "${RED}✗ Не удалось создать ICNS файл ни одним из доступных инструментов${NC}"
    echo -e "${RED}✗ Failed to create ICNS file with any available tool${NC}"
    echo "  Доступные инструменты / Available tools:"
    echo "  - png2icns (sudo apt install icnsutils)"
    echo "  - iconutil + sips (встроенные в macOS / built-in on macOS)"
    echo "  - ImageMagick (brew install imagemagick)"
fi

echo

# Summary
echo "Сводка / Summary:"
echo "=================="

files_created=0

if [ -f "$TARGET_ICO" ]; then
    echo -e "${GREEN}✓ Windows ICO: $TARGET_ICO${NC}"
    ((files_created++))
else
    echo -e "${RED}✗ Windows ICO: не создан${NC}"
fi

if [ -f "$TARGET_ICNS" ]; then
    echo -e "${GREEN}✓ macOS ICNS: $TARGET_ICNS${NC}"
    ((files_created++))
else
    echo -e "${RED}✗ macOS ICNS: не создан${NC}"
fi

echo -e "${GREEN}✓ Linux PNG: $SOURCE_PNG (готов к использованию)${NC}"

echo
if [ $files_created -gt 0 ]; then
    echo -e "${GREEN}Готово! Создано $files_created форматов иконок.${NC}"
    echo -e "${GREEN}Done! Created $files_created icon formats.${NC}"
    echo
    echo "Теперь можно собирать приложение с иконками / Now you can build with icons:"
    echo "  Windows: go build -ldflags \"-H windowsgui -icon assets/icon.ico\" -o qmlauncher.exe"
    echo "  macOS:   go build -ldflags \"-icon assets/icon.icns\" -o qmlauncher"
    echo "  Linux:   go build -o qmlauncher  # иконка в .desktop файле"
else
    echo -e "${YELLOW}Не удалось создать ни одного формата иконки.${NC}"
    echo -e "${YELLOW}Failed to create any icon formats.${NC}"
fi