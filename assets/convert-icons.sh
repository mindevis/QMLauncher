#!/bin/bash

# Convert PNG icon to ICO and ICNS formats

if [ ! -f "icon.png" ]; then
    echo "Error: icon.png not found"
    exit 1
fi

echo "Converting icon.png to other formats..."

# Windows ICO
if command -v convert >/dev/null 2>&1 || command -v magick >/dev/null 2>&1; then
    echo "Creating ICO for Windows..."
    ${IMGMAGICK:-convert} icon.png -define icon:auto-resize=256,128,64,48,32,16 icon.ico 2>/dev/null
    [ -f "icon.ico" ] && echo "✓ icon.ico created" || echo "✗ Failed to create ICO"
else
    echo "⚠ ImageMagick not found, skipping ICO conversion"
fi

# macOS ICNS
if command -v png2icns >/dev/null 2>&1; then
    echo "Creating ICNS for macOS..."
    png2icns icon.icns icon.png 2>/dev/null
    [ -f "icon.icns" ] && echo "✓ icon.icns created" || echo "✗ Failed to create ICNS"
else
    echo "⚠ png2icns not found, skipping ICNS conversion"
fi

echo "Conversion complete. Check files:"
ls -la icon.* 2>/dev/null || echo "No icon files found"
