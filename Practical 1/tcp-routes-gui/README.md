# TCP Routes GUI Viewer

A native Windows GUI application written in Go to display all active TCP connections on your machine.

## Features

- **Display TCP Routes**: Shows all active TCP connections with local address, remote address, and connection state
- **Process Information**: Displays the PID and process name associated with each connection
- **Refresh Capability**: Click the "Refresh Routes" button to update the list at any time
- **Native Windows UI**: Built with the Walk GUI library for a native Windows appearance

## Prerequisites

- Go 1.16 or later
- Windows 7, 8, 10, or 11
- Administrator privileges (recommended, for accessing system-level connection data)

## Installation

1. Navigate to the tcp-routes-gui directory:
```bash
cd "D:\NPE\Practical 1\tcp-routes-gui"
```

2. Download dependencies:
```bash
go mod download
go mod tidy
```

## Building

### Option 1: Run directly (with live output)
```bash
go run main.go
```

### Option 2: Build an executable
```bash
go build -o TCPRoutesViewer.exe
```

Then run the executable:
```
TCPRoutesViewer.exe
```

## Usage

1. **Launch the application** - The GUI window will open showing an empty list
2. **Click "Refresh Routes"** - This fetches all TCP connections from your system using the `netstat` command
3. **View Connections** - Each line displays:
   - **Local Address**: The local IP and port
   - **Remote Address**: The remote IP and port you're connected to
   - **State**: Connection state (ESTABLISHED, LISTENING, TIME_WAIT, etc.)
   - **PID**: Process ID of the application
   - **Process Name**: Name of the executable

## Column Descriptions

- **Local Addr**: Your machine's IP address and port number (format: IP:Port)
- **Remote Addr**: The remote system's IP address and port number
- **State**: Current state of the connection:
  - `ESTABLISHED` - Active connection
  - `LISTENING` - Waiting for incoming connections
  - `TIME_WAIT` - Waiting after connection close
  - `CLOSE_WAIT` - Waiting for close request
  - etc.
- **PID**: Process ID of the application using this connection
- **ProcessName**: Friendly name of the process/application

## Notes

- The application uses the Windows `netstat` command internally to retrieve connection data
- For best results, run with administrator privileges to see all connections
- The process name lookup uses the `tasklist` command and may show "Unknown" if the process is not accessible
- The application fetches data asynchronously, so the UI remains responsive

## Dependencies

- **github.com/lxn/walk**: Windows GUI library for Go
- **github.com/lxn/win**: Windows API bindings for Go

## Troubleshooting

### "Command 'go' not found"
Ensure Go is installed and added to your system PATH. Download from [golang.org](https://golang.org/dl/)

### Dependencies fail to download
Make sure you have an internet connection, then run:
```bash
go mod cache clean
go mod download
```

### GUI doesn't appear
The application uses system resources. Ensure your Windows installation is up to date and you have the latest drivers.

### Can't see some connections
Run the executable as Administrator to see all system connections.

## License

MIT
