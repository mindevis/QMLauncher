@echo off
REM Build QMLauncher for Windows on Windows (native - icon works)
REM Run from QMLauncher dir: scripts\build-windows-native.bat

cd /d "%~dp0\.."
if not exist assets\icon.png goto :noicon
mkdir build 2>nul
copy /Y assets\icon.png build\appicon.png >nul
if exist assets\icon.ico (
    mkdir build\windows 2>nul
    copy /Y assets\icon.ico build\windows\icon.ico >nul
)

wails build -platform windows/amd64 -tags webkit2_41 -clean
if exist build\bin\QMLauncher-windows-amd64.exe (
    move /Y build\bin\QMLauncher-windows-amd64.exe build\
    rmdir /s /q build\bin 2>nul
    rmdir /s /q build\windows 2>nul
    echo Built: build\QMLauncher-windows-amd64.exe
)
goto :eof

:noicon
echo assets\icon.png not found
exit /b 1
