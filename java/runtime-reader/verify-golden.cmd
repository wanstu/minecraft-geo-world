@echo off
setlocal
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0verify-golden.ps1" %*
exit /b %ERRORLEVEL%
