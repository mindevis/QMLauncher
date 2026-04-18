#!/bin/sh
# Prepare icons for Wails build (preBuildHook runs from build/bin/)
# Use our icon.ico directly - Wails uses it for exe embedding when it exists
cd "$(dirname "$0")/.." || exit 0
mkdir -p build build/windows
[ -f assets/icon.png ] && cp -f assets/icon.png build/appicon.png
[ -f assets/icon.ico ] && cp -f assets/icon.ico build/windows/icon.ico
[ -f assets/icon.ico ] && ! [ -f build/appicon.png ] && (convert assets/icon.ico build/appicon.png 2>/dev/null || magick assets/icon.ico build/appicon.png 2>/dev/null) || true
