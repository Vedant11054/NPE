@echo off
REM Setup script for TCP Routes GUI Viewer
REM This script downloads all required dependencies

echo =========================================
echo TCP Routes GUI Viewer - Setup Script
echo =========================================
echo.

REM Check if Go is installed
go version >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Go is not installed or not in PATH
    echo Please download Go from https://golang.org/dl/
    pause
    exit /b 1
)

echo [1/3] Detected Go installation
go version

echo.
echo [2/3] Downloading dependencies...
go mod download
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Failed to download dependencies
    echo Make sure you have an internet connection
    pause
    exit /b 1
)

echo.
echo [3/3] Cleaning and tidying dependencies...
go mod tidy

echo.
echo =========================================
echo Setup complete!
echo =========================================
echo.
echo You can now:
echo - Run: build.bat (to create TCPRoutesViewer.exe)
echo - Or:  run.bat (to run directly with go run)
echo.
pause
