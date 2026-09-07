@echo off
cd /d "%~dp0"
docker compose stop
if errorlevel 1 (
  echo.
  echo Docker Desktop belum aktif atau aplikasi gagal dihentikan.
  pause
) else (
  echo.
  echo TKA Local sudah dihentikan. Data database tetap tersimpan.
  pause
)
