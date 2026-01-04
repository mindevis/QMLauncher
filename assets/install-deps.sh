#!/bin/bash

# Script to install dependencies for icon conversion

echo "Установка зависимостей для конвертации иконок / Installing icon conversion dependencies..."
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Detect OS
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS="linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS="macos"
elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "win32" ]]; then
    OS="windows"
else
    OS="unknown"
fi

echo -e "${BLUE}Обнаружена ОС: $OS / Detected OS: $OS${NC}"
echo

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Install dependencies based on OS
case $OS in
    linux)
        echo "Установка зависимостей для Linux / Installing dependencies for Linux..."
        echo

        # Check for package manager
        if command_exists apt; then
            echo "Используем apt (Ubuntu/Debian) / Using apt (Ubuntu/Debian)"
            sudo apt update
            sudo apt install -y imagemagick icnsutils
        elif command_exists yum; then
            echo "Используем yum (CentOS/RHEL) / Using yum (CentOS/RHEL)"
            sudo yum install -y ImageMagick
            # icnsutils may not be available in yum
        elif command_exists pacman; then
            echo "Используем pacman (Arch Linux) / Using pacman (Arch Linux)"
            sudo pacman -S --noconfirm imagemagick
        else
            echo -e "${YELLOW}Неизвестный менеджер пакетов. Установите ImageMagick вручную.${NC}"
            echo -e "${YELLOW}Unknown package manager. Please install ImageMagick manually.${NC}"
        fi
        ;;

    macos)
        echo "Установка зависимостей для macOS / Installing dependencies for macOS..."
        echo

        if command_exists brew; then
            echo "Используем Homebrew / Using Homebrew"
            brew install imagemagick
            echo -e "${BLUE}Примечание: macOS имеет встроенные инструменты iconutil и sips${NC}"
            echo -e "${BLUE}Note: macOS has built-in iconutil and sips tools${NC}"
        else
            echo -e "${YELLOW}Homebrew не найден. Установите Homebrew и затем ImageMagick:${NC}"
            echo -e "${YELLOW}Homebrew not found. Install Homebrew and then ImageMagick:${NC}"
            echo "  /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
            echo "  brew install imagemagick"
        fi
        ;;

    windows)
        echo "Установка зависимостей для Windows / Installing dependencies for Windows..."
        echo

        if command_exists choco; then
            echo "Используем Chocolatey / Using Chocolatey"
            choco install imagemagick
        elif command_exists winget; then
            echo "Используем winget / Using winget"
            winget install ImageMagick.ImageMagick
        else
            echo -e "${YELLOW}Не найден менеджер пакетов. Скачайте ImageMagick с официального сайта:${NC}"
            echo -e "${YELLOW}No package manager found. Download ImageMagick from official website:${NC}"
            echo "  https://imagemagick.org/script/download.php#windows"
        fi
        ;;

    *)
        echo -e "${RED}Неподдерживаемая ОС: $OSTYPE${NC}"
        echo -e "${RED}Unsupported OS: $OSTYPE${NC}"
        echo "Пожалуйста, установите ImageMagick вручную / Please install ImageMagick manually"
        exit 1
        ;;
esac

echo
echo "Проверка установки / Verifying installation..."

# Check installations
if command_exists convert || command_exists magick; then
    echo -e "${GREEN}✓ ImageMagick установлен / ImageMagick installed${NC}"
else
    echo -e "${RED}✗ ImageMagick не найден / ImageMagick not found${NC}"
fi

if command_exists png2icns; then
    echo -e "${GREEN}✓ png2icns установлен / png2icns installed${NC}"
elif [[ "$OS" == "macos" ]] && command_exists iconutil; then
    echo -e "${GREEN}✓ iconutil доступен (macOS) / iconutil available (macOS)${NC}"
else
    echo -e "${YELLOW}⚠ png2icns не найден (опционально) / png2icns not found (optional)${NC}"
fi

echo
echo -e "${GREEN}Готово! Теперь можно конвертировать иконки / Done! You can now convert icons${NC}"
echo "Запустите: ./convert-icons.sh"
echo