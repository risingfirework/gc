@echo off
cd /d "%~dp0"
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\start-local.ps1"
if errorlevel 1 (
  echo.
  echo Gagal menjalankan TKA Local. Baca pesan di atas.
  pause
)
