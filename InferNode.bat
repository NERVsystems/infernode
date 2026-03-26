@echo off
cd /d "%~dp0"
start "" o.emu.exe -c0 -g 980x700 -pheap=512m -pmain=512m -pimage=512m -r . sh /dis/lucifer-start.sh
