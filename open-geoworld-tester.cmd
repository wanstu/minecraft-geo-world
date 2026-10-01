@echo off
setlocal
cd /d "%~dp0"
echo Starting GeoWorld Tester...
echo URL: http://127.0.0.1:8788
echo.
go run .cmdgeoworld-tester
if errorlevel 1 (
  echo.
  echo GeoWorld Tester failed to start.
  pause
)
