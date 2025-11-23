#!/bin/bash
# Скрипт для упаковки Windows версии QMLauncher без установщика

set -e

echo "🔨 Сборка QMLauncher для Windows..."

# Сборка приложения
npm run build

# Сборка portable версии (без установщика)
echo "📦 Создание portable версии..."
npm run build && npx electron-builder --win --config.win.target=portable

echo "✅ Готово! Portable версия находится в release/"
echo "📁 Файл: release/QMLauncher-1.0.0-portable.exe"
echo ""
echo "💡 Альтернатива: используйте уже готовую папку release/win-unpacked/"
echo "   Можно заархивировать её и использовать как portable версию"

