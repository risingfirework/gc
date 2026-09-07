@echo off
cd /d "%~dp0"
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\install-autostart.ps1"
if errorlevel 1 (
  echo.
  echo Gagal memasang startup otomatis. Baca pesan di atas.
)
echo.
pause
