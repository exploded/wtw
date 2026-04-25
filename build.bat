@echo off
setlocal enabledelayedexpansion

echo Building wtw...
go build -o wtw.exe ./cmd/wtw/
if errorlevel 1 (
    echo Build failed!
    exit /b 1
)

if not exist .env (
    if exist .env.example (
        copy .env.example .env
        echo Created .env from .env.example - edit with real values
    )
)

if exist .env (
    for /f "usebackq tokens=1,* delims==" %%a in (.env) do (
        echo %%a | findstr /r "^#" >nul
        if errorlevel 1 (
            if not "%%a"=="" (
                set "%%a=%%b"
            )
        )
    )
)

echo Starting wtw on port %PORT%...
wtw.exe
