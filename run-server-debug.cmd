@echo off
cd /d "%~dp0"
set GODEBUG=tls13=1,tlsdebug=2
echo Starting server with TLS debug enabled...
echo GODEBUG=%GODEBUG%
echo.
.\build\fluidity-server.exe --config configs\server.local.yaml
pause
