#!/bin/bash

set -e

echo "🏗️  Building QMLauncher CLI for $RUNNER_OS..."

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
OUTPUT_NAME="QMLauncher-cli-${OS}-${ARCH}${EXT}"

echo "📦 Building for ${OS}/${ARCH}..."
go build -tags cli -o "$OUTPUT_NAME" .

# Platform-specific post-processing
case "$OS" in
    "linux")
        echo "🐧 Creating Linux desktop integration..."

        # Copy icon for desktop integration
        if [ -f "assets/icon.png" ]; then
            cp assets/icon.png QMLauncher.png
            echo "✅ Icon copied for desktop integration"
        fi

        # Create desktop file
        printf "[Desktop Entry]\nVersion=1.0\nName=QMLauncher\nComment=Minecraft Launcher CLI\nExec=./QMLauncher-cli-linux-amd64\nIcon=QMLauncher\nTerminal=true\nType=Application\nCategories=Game;\n" > QMLauncher.desktop

        chmod +x QMLauncher.desktop
        echo "✅ Desktop file created"
        ;;
        
    "windows")
        echo "🪟 Building Windows with embedded icon..."
        
        # Create resource file
        printf "1 ICON \"assets/icon.ico\"\n" > icon.rc
        
        # Compile resource (windres should be available on Windows runners)
        if command -v windres >/dev/null 2>&1; then
            windres -i icon.rc -o icon.syso
            echo "✅ Icon resource compiled"
        else
            echo "⚠️  windres not found, building without embedded icon"
        fi
        
        # Rebuild with icon resource
        go build -tags cli -o "$OUTPUT_NAME" .
        echo "✅ Windows binary built with icon"
        
        # Cleanup
        rm -f icon.rc icon.syso
        ;;
        
    "macos")
        echo "🍎 Creating macOS app bundle..."
        
        # Create app bundle structure
        mkdir -p "QMLauncher.app/Contents/MacOS"
        mkdir -p "QMLauncher.app/Contents/Resources"
        
        # Copy binary
        cp "$OUTPUT_NAME" "QMLauncher.app/Contents/MacOS/"
        
        # Copy icon if available
        if [ -f "assets/icon.icns" ]; then
            cp "assets/icon.icns" "QMLauncher.app/Contents/Resources/AppIcon.icns"
            echo "✅ ICNS icon copied"
        else
            echo "⚠️  ICNS icon not found"
        fi
        
        # Create Info.plist
        printf '<?xml version="1.0" encoding="UTF-8"?>\n<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">\n<plist version="1.0">\n<dict>\n    <key>CFBundleExecutable</key>\n    <string>QMLauncher-cli-macos-amd64</string>\n    <key>CFBundleIconFile</key>\n    <string>AppIcon</string>\n    <key>CFBundleIdentifier</key>\n    <string>com.qmlauncher.cli</string>\n    <key>CFBundleName</key>\n    <string>QMLauncher</string>\n    <key>CFBundleVersion</key>\n    <string>1.0.0</string>\n    <key>LSUIElement</key>\n    <true/>\n</dict>\n</plist>\n' > "QMLauncher.app/Contents/Info.plist"
        
        echo "✅ macOS app bundle created"
        ;;
esac

echo "✅ Build completed for $RUNNER_OS"
echo "📦 Output: $OUTPUT_NAME"
