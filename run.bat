@echo off
setlocal enabledelayedexpansion

if not exist wtw.exe (
    echo No binary found, running build.bat...
    call build.bat
    exit /b %errorlevel%
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
