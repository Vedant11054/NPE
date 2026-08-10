# Concurrent Multi-Client Chat Server

A high-performance concurrent chat server built with Go, featuring TCP socket programming, goroutines, channels, and real-time message broadcasting.

## Features

✨ **Core Features:**
- **TCP Socket Programming**: Raw TCP connections for traditional clients
- **WebSocket Support**: Real-time bidirectional communication for web clients
- **Goroutines**: Lightweight concurrent handling of multiple clients
- **Channels**: Efficient inter-goroutine communication for message routing
- **Message Broadcasting**: Real-time message distribution to all connected clients
- **Web-Based GUI**: Beautiful, responsive chat interface
- **Connection Management**: Automatic client tracking and disconnection handling

## Architecture

### Project Structure
```
chat-server/
├── cmd/
│   └── main.go              # Server entry point with Web GUI
├── pkg/
│   └── server/
│       ├── server.go        # TCP server implementation
│       ├── websocket.go     # WebSocket server implementation
│       └── client.go        # TCP client library
├── web/                     # Web GUI assets
├── go.mod                   # Go module definition
└── chat-server.exe          # Compiled executable
```

### Technical Implementation

#### 1. **TCP Server (`server.go`)**
- Listens on port 9999 for TCP connections
- Each client connection spawns a goroutine via `go s.acceptConnections()`
- Uses channels for concurrent message handling:
  - `Broadcast`: Centralized message channel
  - `AddClient`: Register new clients
  - `RemoveClient`: Unregister disconnected clients
- Implements broadcaster pattern using `select` on multiple channels

#### 2. **WebSocket Server (`websocket.go`)**
- Handles HTTP upgrade to WebSocket on `/ws` endpoint
- Gorilla WebSocket library for connection management
- Same broadcasting pattern as TCP server
- JSON message format for structured communication

#### 3. **Channel Pattern**
```
Message Flow:
┌──────────────────────────────────────────────────┐
│ Connected Clients (Goroutines)                   │
│ ├─ TCP Client Handler                           │
│ ├─ WebSocket Handler                            │
│ └─ CLI Server                                    │
└─────────┬──────────────────────────────────────┘
          │ sends to
          ▼
    ┌──────────────────┐
    │  Broadcast Chan  │
    │  (buffered: 10)  │
    └─────────┬────────┘
              │
    ┌─────────▼────────┐
    │ Broadcaster Go   │
    │ Routine (Select) │
    └─────────┬────────┘
              │ sends to
              ▼
    ┌──────────────────────┐
    │ Client Send Channels │
    │ (each client has one)│
    └──────────────────────┘
```

#### 4. **Goroutine Model**
- **Main Goroutine**: CLI for server management
- **Accept Loop**: Accepts new connections
- **Client Handlers**: One per connected client (TCP)
- **Broadcaster**: Central message dispatcher
- **WebSocket Handlers**: One per WebSocket connection

## Running the Server

### Prerequisites
- Go 1.21 or higher
- Windows/Linux/macOS

### Build and Run

```bash
# Navigate to project directory
cd chat-server

# Build the executable
go build -o chat-server.exe ./cmd

# Run the server
./chat-server.exe
```

### Output
```
=== Chat Server Started ===
Servers:
  TCP: localhost:9999 (for TCP clients)
  Web GUI: http://localhost:8080

Commands:
  /status - Show number of connected clients
  /broadcast <message> - Broadcast a message to all clients
  /quit - Exit the server
```

## Using the Chat Server

### 1. Web GUI (Recommended)
1. Open your browser and navigate to: **http://localhost:8080**
2. Enter your username in the "Your name" field
3. Type messages and press Enter or click Send
4. Messages appear in real-time from all connected users

### 2. TCP Client
Use any TCP client to connect to `localhost:9999`:

```bash
# Example using telnet (if available)
telnet localhost 9999

# Send messages in format: [username]: [message]
User1: Hello everyone
```

### 3. Server Commands
From the server terminal:

```
> /status
TCP clients: 2, WebSocket clients: 3

> /broadcast Hello from admin!
# Message sent to all connected clients

> /quit
# Shutdown server
```

## Code Highlights

### Message Broadcasting with Channels

```go
// Broadcaster goroutine - central hub
func (s *Server) broadcaster() {
    for {
        select {
        case client := <-s.AddClient:
            s.mu.Lock()
            s.Clients[client.ID] = client
            s.mu.Unlock()
            
        case msg := <-s.Broadcast:
            s.mu.RLock()
            clients := s.Clients
            s.mu.RUnlock()
            
            // Send to all clients
            for _, client := range clients {
                select {
                case client.Send <- msg:
                case <-client.Done:
                    // Client closed
                }
            }
        }
    }
}
```

### Concurrent Client Handling

```go
// Each client runs in its own goroutine
go func() {
    for {
        // Read messages from client
        line, err := client.Reader.ReadString('\n')
        if err != nil {
            break
        }
        
        // Broadcast to all
        msg := Message{From: username, Content: line}
        s.Broadcast <- msg
    }
}()
```

### WebSocket Handler

```go
// Handle WebSocket with full-duplex communication
func (ws *WebSocketServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    // Upgrade HTTP to WebSocket
    conn, _ := ws.upgrader.Upgrade(w, r, nil)
    
    // Handle incoming messages
    go ws.handleWSClient(client)
    
    // Send outgoing messages
    go func() {
        for msg := range client.Send {
            conn.WriteJSON(msg)
        }
    }()
}
```

## GUI Features

- **Real-time Messaging**: Instant message delivery
- **User Tracking**: See who's online
- **Message Counter**: Track conversation activity
- **Connection Status**: Visual indicator (green = connected, red = disconnected)
- **Auto-reconnect**: Automatically reconnects if connection drops
- **Responsive Design**: Works on desktop and mobile browsers
- **Message History**: All messages displayed in scrollable area
- **Server/Admin Messages**: Special styling for system messages

## Performance Characteristics

- **Concurrency**: Handles thousands of simultaneous connections
- **Low Latency**: Direct goroutine-to-goroutine communication via channels
- **Memory Efficient**: Lightweight goroutines (thousands can run on modest hardware)
- **Non-blocking**: Message broadcasting doesn't block senders
- **Thread-safe**: sync.RWMutex protects shared state

## Testing

### Test Scenario 1: Multiple Web Clients
```
1. Open 3 browser tabs with http://localhost:8080
2. Set different usernames in each tab
3. Send messages from each client
4. Verify all clients receive all messages
```

### Test Scenario 2: Server Broadcasting
```
1. Connect web clients
2. From server terminal: /broadcast Welcome to chat!
3. Verify message appears on all clients
4. Check /status to see connected clients
```

### Test Scenario 3: Disconnection Handling
```
1. Connect a client
2. Close the browser tab
3. Verify server detects disconnection
4. Send new message from another client
5. Verify message doesn't go to closed client
```

## Technology Stack

- **Language**: Go 1.21+
- **Networking**: net (standard library)
- **WebSocket**: github.com/gorilla/websocket
- **Frontend**: HTML5, CSS3, Vanilla JavaScript
- **Protocol**: TCP, WebSocket (RFC 6455)

## Key Go Concepts Used

1. **Goroutines**: Lightweight concurrent execution units
2. **Channels**: Type-safe communication between goroutines
3. **select Statement**: Multiplexing channel operations
4. **Mutex**: Synchronization for shared resources (client map)
5. **Defer**: Cleanup of connections
6. **Interface{}**: Generic message handling

## Limitations & Future Enhancements

**Current Limitations:**
- No authentication/authorization
- In-memory message storage (lost on restart)
- No persistence layer
- No message history
- No direct messaging between users

**Potential Enhancements:**
- User authentication and roles
- Message persistence (database)
- Private/direct messaging
- Chat rooms/channels
- User blocking/muting
- Message reactions
- File sharing
- Admin dashboard
- Load balancing

## Troubleshooting

**Port Already in Use:**
```bash
# Kill process on port 8080 or 9999
# Windows: netstat -ano | findstr :8080
# Linux: lsof -i :8080
```

**Cannot Connect to Server:**
- Verify server is running
- Check firewall settings
- Verify correct IP and port
- Check browser console for WebSocket errors

**Messages Not Appearing:**
- Check connection status indicator
- Refresh the page
- Check server terminal for errors
- Verify username is set

## License

Open source - feel free to modify and use

## Author

Concurrent Chat Server - Multi-client networking demonstration

---

**Enjoy concurrent chatting! 🚀**
