@echo off
:: InferNode Setup — double-click to run
:: Launches setup-windows.ps1 with execution policy bypass
cd /d "%~dp0"
powershell.exe -ExecutionPolicy Bypass -NoProfile -File "%~dp0setup-windows.ps1"
if %errorlevel% neq 0 pause
pause
