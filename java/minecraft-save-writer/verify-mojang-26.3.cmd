@echo off
setlocal
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0verify-mojang-26.3.ps1" %*
exit /b %ERRORLEVEL%
