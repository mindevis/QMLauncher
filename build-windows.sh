#!/bin/bash

# Build script for QMLauncher Windows
# This script builds the frontend and then builds the Windows installer

set -e  # Exit on error

echo "=========================================="
echo "QMLauncher Windows Build Script"
echo "=========================================="

# Add Go bin directory to PATH for garble (needed for obfuscation)
export PATH="$HOME/go/bin:$PATH"

# Get the script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Step 1: Build frontend
echo ""
echo "Step 1: Building frontend..."
echo "----------------------------------------"
cd frontend

if ! command -v npm &> /dev/null; then
    echo "Error: npm is not installed or not in PATH"
    exit 1
fi

npm run build

if [ $? -ne 0 ]; then
    echo "Error: Frontend build failed"
    exit 1
fi

echo "✓ Frontend build completed successfully"
cd ..

# Step 2: Build Windows installer
echo ""
echo "Step 2: Building Windows installer..."
echo "----------------------------------------"

if ! command -v wails &> /dev/null; then
    echo "Error: wails is not installed or not in PATH"
    echo "Please install Wails: https://wails.io/docs/gettingstarted/installation"
    exit 1
fi

wails build -platform windows/amd64 -nsis

if [ $? -ne 0 ]; then
    echo "Error: Windows build failed"
    exit 1
fi

echo ""
echo "=========================================="
echo "✓ Build completed successfully!"
echo "=========================================="
echo ""
echo "Windows installer (.msi) should be in: build/bin/"
echo ""

