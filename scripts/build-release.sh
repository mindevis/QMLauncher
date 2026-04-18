#!/bin/bash

set -e

echo "🏗️  Building QMLauncher (GUI) for $RUNNER_OS..."

# Normalize OS names
case "$RUNNER_OS" in
    "Linux")
        OS="linux"
        EXT=""
        ;;
    "Windows")
        OS="windows"
        EXT=".exe"
        ;;
    "macOS")
        OS="macos"
        EXT=""
        ;;
    *)
        echo "❌ Unsupported OS: $RUNNER_OS"
        exit 1
        ;;
esac

ARCH="amd64"
OUTPUT_NAME="QMLauncher-${OS}-${ARCH}${EXT}"

echo "📦 Frontend build..."
(cd frontend && npm ci && npm run build)

echo "📦 Building Go binary for ${OS}/${ARCH}..."
go build -o "$OUTPUT_NAME" .

# Platform-specific post-processing
case "$OS" in
    "linux")
        echo "🐧 Creating Linux desktop integration..."

        if [ -f "assets/icon.png" ]; then
            cp assets/icon.png QMLauncher.png
            echo "✅ Icon copied for desktop integration"
        fi

        printf "[Desktop Entry]\nVersion=1.0\nName=QMLauncher\nComment=Minecraft Launcher\nExec=./QMLauncher-linux-amd64\nIcon=QMLauncher\nTerminal=false\nType=Application\nCategories=Game;\n" > QMLauncher.desktop

        chmod +x QMLauncher.desktop
        echo "✅ Desktop file created"
        ;;

    "windows")
        echo "🪟 Building Windows with embedded icon..."

        printf "1 ICON \"assets/icon.ico\"\n" > icon.rc

        if command -v windres >/dev/null 2>&1; then
            windres -i icon.rc -o icon.syso
            echo "✅ Icon resource compiled"
        else
            echo "⚠️  windres not found, building without embedded icon"
        fi

        go build -o "$OUTPUT_NAME" .
        echo "✅ Windows binary built"

        rm -f icon.rc icon.syso
        ;;

    "macos")
        echo "🍎 Creating macOS app bundle..."

        mkdir -p "QMLauncher.app/Contents/MacOS"
        mkdir -p "QMLauncher.app/Contents/Resources"

        cp "$OUTPUT_NAME" "QMLauncher.app/Contents/MacOS/"

        if [ -f "assets/icon.icns" ]; then
            cp "assets/icon.icns" "QMLauncher.app/Contents/Resources/AppIcon.icns"
            echo "✅ ICNS icon copied"
        else
            echo "⚠️  ICNS icon not found"
        fi

        EXEC_NAME="$OUTPUT_NAME"
        printf '<?xml version="1.0" encoding="UTF-8"?>\n<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">\n<plist version="1.0">\n<dict>\n    <key>CFBundleExecutable</key>\n    <string>%s</string>\n    <key>CFBundleIconFile</key>\n    <string>AppIcon</string>\n    <key>CFBundleIdentifier</key>\n    <string>com.qmlauncher.gui</string>\n    <key>CFBundleName</key>\n    <string>QMLauncher</string>\n    <key>CFBundleVersion</key>\n    <string>1.0.0</string>\n    <key>LSUIElement</key>\n    <false/>\n</dict>\n</plist>\n' "$EXEC_NAME" > "QMLauncher.app/Contents/Info.plist"

        echo "✅ macOS app bundle created"
        ;;
esac

echo "✅ Build completed for $RUNNER_OS"
echo "📦 Output: $OUTPUT_NAME"
