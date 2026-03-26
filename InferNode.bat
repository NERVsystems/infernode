@echo off
cd /d "%~dp0"
setlocal enabledelayedexpansion
set W=980
set H=700
for /f %%a in ('powershell -NoProfile -Command "Add-Type -AssemblyName System.Windows.Forms; [math]::Floor([System.Windows.Forms.Screen]::PrimaryScreen.Bounds.Width * 0.85)"') do set W=%%a
for /f %%a in ('powershell -NoProfile -Command "Add-Type -AssemblyName System.Windows.Forms; [math]::Floor([System.Windows.Forms.Screen]::PrimaryScreen.Bounds.Height * 0.85)"') do set H=%%a
start "" o.emu.exe -c0 -g !W!x!H! -pheap=512m -pmain=512m -pimage=512m -r . sh /dis/lucifer-start.sh
endlocal
