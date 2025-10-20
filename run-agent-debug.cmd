@echo off
cd /d "%~dp0"
set GODEBUG=tls13=1,tlsdebug=2
echo Starting agent with TLS debug enabled...
echo GODEBUG=%GODEBUG%
echo.
.\build\fluidity-agent.exe --config configs\agent.local.yaml
pause
