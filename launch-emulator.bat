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
    echo [launch] Starting Onuron OS High-Fidelity Simulator...
    start "" "%~dp0bin\nilsim.exe"
    goto :done
)

if exist "nilphone.exe" (
    echo [launch] Starting NilOS Native Simulator...
    start "" "%~dp0nilphone.exe"
    goto :done
)

if exist "simulator\main.go" (
    echo [build] Compiling native simulator binary...
    cd /d "%~dp0simulator"
    go build -o ..\nilphone.exe .
    cd /d "%~dp0"
    start "" "%~dp0nilphone.exe"
    goto :done
)

:done
echo [OK] Native Onuron OS Simulator is active!

