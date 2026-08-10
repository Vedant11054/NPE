# Quick Start Guide

## Step 1: Setup Dependencies
Double-click `setup.bat` in the project folder. This will download all required Go packages.

## Step 2: Run the Application

### Option A: Quick Run (Recommended for Testing)
Double-click `run.bat` - the application will launch immediately.

### Option B: Build an Executable
Double-click `build.bat` to create `TCPRoutesViewer.exe`, then double-click the executable to run.

## Using the Application

1. **Window Opens**: The TCP Routes Viewer window will appear
2. **Click "Refresh Routes"**: This loads all TCP connections from your system
3. **View Results**: A list shows all active TCP connections with:
   - Local Address (your machine's IP:Port)
   - Remote Address (where you're connected to)
   - State (ESTABLISHED, LISTENING, etc.)
   - PID (Process ID)
   - Process Name (Application name)

## Tips

- For best results, **run as Administrator** to see all system connections
- Click Refresh multiple times to monitor connection changes in real-time
- The list updates asynchronously, so the window stays responsive
- Each connection line shows the complete network route information

## Troubleshooting

**Problem**: "go: command not found"
**Solution**: Install Go from https://golang.org/dl/

**Problem**: Dependencies fail to download
**Solution**: Run setup.bat again or check your internet connection

**Problem**: Can't see all connections
**Solution**: Run as Administrator (right-click > Run as Administrator)

**Problem**: "Failed to load TCP routes"
**Solution**: Check that netstat is available on your system (usually built-in on Windows)

---

For more detailed information, see README.md
