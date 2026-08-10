@echo off
REM Run script for TCP Routes GUI Viewer
REM This runs the application directly without building an executable

cd /d "%~dp0"
echo Starting TCP Routes GUI Viewer...
go run main.go
