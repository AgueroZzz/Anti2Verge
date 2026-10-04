@echo off
chcp 65001 >nul
title Anti2Verge 控制台守护
cd /d "%~dp0"

if exist "bin\anti2verge-cli.exe" (
    bin\anti2verge-cli.exe run
) else (
    anti2verge-cli.exe run
)

pause
