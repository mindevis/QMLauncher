# Build QMLauncher for Windows on Windows (native build - icon works correctly)
# Run from project root: .\scripts\build-windows-native.ps1
# Requires: Go, Node.js, Wails CLI, npm

$ErrorActionPreference = "Stop"
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent $scriptDir
Set-Location $projectRoot

# Prepare icon
if (Test-Path "assets\icon.png") {
    New-Item -ItemType Directory -Force -Path build | Out-Null
    Copy-Item -Force assets\icon.png build\appicon.png
}
if (Test-Path "assets\icon.ico") {
    New-Item -ItemType Directory -Force -Path build\windows | Out-Null
    Copy-Item -Force assets\icon.ico build\windows\icon.ico
}

# Build (standard Wails - icon embeds correctly on native Windows)
wails build -platform windows/amd64 -tags webkit2_41 -clean

# Move exe
if (Test-Path "build\bin\QMLauncher-windows-amd64.exe") {
    Move-Item -Force build\bin\QMLauncher-windows-amd64.exe build\QMLauncher-windows-amd64.exe
    Remove-Item -Recurse -Force build\bin, build\windows -ErrorAction SilentlyContinue
    Write-Host "Built: build\QMLauncher-windows-amd64.exe"
}
