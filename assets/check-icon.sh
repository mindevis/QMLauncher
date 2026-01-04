#!/bin/bash

# Simple script to check if icon files exist

echo "Checking for icon files..."

[ -f "icon.ico" ] && echo "✓ icon.ico found" || echo "✗ icon.ico missing"
[ -f "icon.icns" ] && echo "✓ icon.icns found" || echo "⚠ icon.icns missing (optional)"
[ -f "icon.png" ] && echo "✓ icon.png found" || echo "⚠ icon.png missing (recommended)"

echo
echo "Note: PNG is required for Linux builds, ICO for Windows, ICNS for macOS"
echo "Run ./convert-icons.sh to convert PNG to other formats"