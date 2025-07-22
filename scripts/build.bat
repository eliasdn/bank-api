@echo off
REM Build script for bank-api (Windows)
REM This script builds the application for Windows

setlocal enabledelayedexpansion

REM Colors for output
set "RED=[91m"
set "GREEN=[92m"
set "YELLOW=[93m"
set "NC=[0m"

REM Get the directory of this script
set "SCRIPT_DIR=%~dp0"
set "PROJECT_ROOT=%SCRIPT_DIR%.."

REM Build configuration
set "BINARY_NAME=bank-api"
set "VERSION=%VERSION%"
if "%VERSION%"=="" set "VERSION=dev"

for /f "tokens=*" %%a in ('powershell -Command "Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ'"') do set "BUILD_TIME=%%a"
for /f "tokens=*" %%a in ('git rev-parse --short HEAD 2^>nul') do set "GIT_COMMIT=%%a"
if "%GIT_COMMIT%"=="" set "GIT_COMMIT=unknown"

echo %GREEN%Building %BINARY_NAME%...%NC%

REM Create build directory
set "BUILD_DIR=%PROJECT_ROOT%\build"
if not exist "%BUILD_DIR%" mkdir "%BUILD_DIR%"

REM Build flags
set "LDFLAGS=-X main.Version=%VERSION% -X main.BuildTime=%BUILD_TIME% -X main.GitCommit=%GIT_COMMIT%"

REM Build for Windows AMD64
echo %YELLOW%Building for Windows AMD64...%NC%
cd /d "%PROJECT_ROOT%"
go build -ldflags "%LDFLAGS%" -o "%BUILD_DIR%\%BINARY_NAME%-windows-amd64.exe" cmd/server/main.go

echo %GREEN%Build completed successfully!%NC%
echo Binaries available in: %BUILD_DIR%
dir "%BUILD_DIR%"