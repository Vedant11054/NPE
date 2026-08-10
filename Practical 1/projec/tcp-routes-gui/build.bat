@echo off
REM Build script for TCP Routes GUI Viewer
REM Run this file to build the executable

echo Building TCP Routes GUI Viewer...
go build -o TCPRoutesViewer.exe
if %ERRORLEVEL% NEQ 0 (
    echo Build failed!
    pause
    exit /b 1
)
echo Build successful! Run TCPRoutesViewer.exe to start the application.
pause
