@echo off
setlocal
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0verify-storage.ps1"
exit /b %ERRORLEVEL%
