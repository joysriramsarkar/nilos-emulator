@echo off
chcp 65001 >nul
title Onuron OS - Native Device Simulator (Material Design 3)
color 0b

echo ============================================================
echo   [ONURON OS] Native Device Simulator (M3 / Material You)
echo   Direct Framebuffer Rendering (Zero HTML/CSS/JS)
echo ============================================================
echo.

cd /d "%~dp0"

if exist "bin\nilsim.exe" (
    echo [launch] Starting Onuron OS Desktop Simulator...
    start "" "%~dp0bin\nilsim.exe"
    goto :done
)

if exist "nilos-emulator.exe" (
    echo [launch] Starting NilOS Studio App Runtime Emulator...
    start "" "%~dp0nilos-emulator.exe"
    goto :done
)

echo [build] Building and starting NilOS Studio Emulator...
go run .

:done
echo [OK] Native Onuron OS Simulator is active!

