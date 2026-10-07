@echo off
title NilOS Studio Device Emulator - Nilang Ecosystem
color 0b
echo ============================================================
echo   📱 NILOS STUDIO DEVICE EMULATOR (NILANG ECOSYSTEM)
echo   Launching Full Virtual Device Studio & Multi-App OS...
echo ============================================================
echo.

cd /d "%~dp0"
start msedge --app="http://localhost:8085/device.html" --window-size=1280,880 || start chrome --app="http://localhost:8085/device.html" --window-size=1280,880 || start http://localhost:8085/device.html

echo [OK] NilOS Studio Emulator launched at http://localhost:8085/device.html
go run .
pause
