# Concurrent Multi-Client Chat Server - Quick Start

## Step 1: Build the Server
```bash
cd chat-server
go build -o chat-server.exe ./cmd
```

## Step 2: Run the Server
```bash
./chat-server.exe
```

You should see:
```
=== Chat Server Started ===
Servers:
  TCP: localhost:9999 (for TCP clients)
  Web GUI: http://localhost:8080

Commands:
  /status - Show number of connected clients
  /broadcast <message> - Broadcast a message to all clients
  /quit - Exit the server

> 
```

## Step 3: Open the Web GUI
Open your browser and go to: **http://localhost:8080**

You should see a beautiful chat interface with:
- User list on the left
- Message area in the center
- Input box at the bottom

## Step 4: Chat!
1. Enter your name
2. Type a message
3. Click "Send" or press Enter
4. Open the GUI in another browser tab to test multiple users

## Architecture Overview

### TCP Server (`port 9999`)
```
Client 1 ──┐
Client 2 ──┼──> TCP Listener ──> Goroutine per client ──┐
Client 3 ──┘                                              │
                                                          ├──> Broadcast Channel ──> Broadcaster ──> All Clients
WebSocket Client ──────> Upgrade ──> WebSocket Handler ──┤
                                                          │
Server CLI ────────────────> /broadcast command ──────────┘
```

### Key Technologies

1. **Goroutines** - Lightweight threads for concurrent client handling
   - Each TCP client gets its own goroutine
   - Each WebSocket client gets a read/write goroutine
   - One broadcaster goroutine for all routing

2. **Channels** - Type-safe communication between goroutines
   ```
   Broadcast Channel    → Main hub for all messages
   AddClient Channel    → Register new connections
   RemoveClient Channel → Clean up disconnections
   Client.Send Channel  → Individual message routing
   ```

3. **TCP Socket Programming**
   - Raw TCP connections on port 9999
   - bufio for efficient reading/writing
   - Automatic cleanup with defer

4. **Message Broadcasting Pattern**
   ```
   Send: Client 1 ──> Broadcast Channel
                              │
   Receive: select over channels in broadcaster
                              │
         ┌─────────────────────┼─────────────────────┐
         │                     │                     │
   Client 1.Send         Client 2.Send         Client 3.Send
         │                     │                     │
         ▼                     ▼                     ▼
   All clients receive the message
   ```

## Usage Examples

### Example 1: Web GUI Chat
1. Open 3 browser tabs
2. Set usernames: "Alice", "Bob", "Charlie"
3. Send messages from each - all tabs receive them instantly

### Example 2: Server Broadcasting
Terminal:
```
> /broadcast Server maintenance in 5 minutes!
[SERVER] System message broadcast to all clients
```
All clients see: "ADMIN: Server maintenance in 5 minutes!"

### Example 3: Check Status
Terminal:
```
> /status
TCP clients: 2, WebSocket clients: 3
```

## Testing Concurrent Connections

### Test 1: 10 Web Clients
1. Open 10 browser tabs
2. Each enters different username
3. All send messages
4. Verify delivery time (should be near-instant)

### Test 2: Connection Stability
1. Connect client
2. Keep connection idle for 10 minutes
3. Send message - should arrive instantly
4. Close browser - server detects and notifies others

### Test 3: High-Volume Messages
1. Connect client
2. Send 100 messages rapidly
3. Verify all arrive in order
4. Check server resource usage (should be minimal)

## Message Flow Diagram

```
┌─────────────────────────────────────────────┐
│ User 1 sends "Hello" from browser           │
└────────────────────┬────────────────────────┘
                     │
                     ▼
        ┌────────────────────────────┐
        │ WebSocket Handler receives │
        └────────────┬───────────────┘
                     │
                     ▼
        ┌────────────────────────────┐
        │ Convert to Message struct  │
        │ {From: "User1",            │
        │  Content: "Hello",         │
        │  Timestamp: now}           │
        └────────────┬───────────────┘
                     │
                     ▼
        ┌────────────────────────────┐
        │ Send to Broadcast Channel  │
        └────────────┬───────────────┘
                     │
                     ▼
        ┌────────────────────────────┐
        │ Broadcaster Goroutine      │
        │ (select on channels)       │
        └────────────┬───────────────┘
                     │
        ┌────────────┴────────────┬──────────────┐
        │                         │              │
        ▼                         ▼              ▼
   To User1          To User2          To User3
   (WebSocket)       (WebSocket)       (WebSocket)
        │                  │                 │
        └──────────────────┴─────────────────┘
                     │
                     ▼
          Browsers receive via WebSocket
          Message appears in chat instantly
```

## Performance Notes

- **Latency**: Sub-millisecond message delivery
- **Scalability**: Can handle thousands of concurrent connections
- **Memory**: Each goroutine uses ~2KB (can run 100k+ on modest hardware)
- **CPU**: Minimal CPU usage due to efficient channel operations

## File Structure

```
chat-server/
├── README.md                    # Full documentation
├── QUICKSTART.md               # This file
├── go.mod                      # Go modules
├── chat-server.exe             # Compiled server (after build)
│
├── cmd/
│   ├── main.go                 # Server with Web GUI
│   └── tcp-client-example/     # Example TCP client
│       └── main.go
│
├── pkg/
│   └── server/
│       ├── server.go           # TCP server core
│       ├── websocket.go        # WebSocket handler
│       └── client.go           # TCP client library
│
└── web/                        # (Web GUI embedded in main.go)
```

## Troubleshooting

**Q: Can't connect to http://localhost:8080**
A: Check if server is running, ports might be blocked by firewall

**Q: Messages not appearing in other tabs**
A: Refresh the page, check browser console for WebSocket errors

**Q: Server crashes with "port already in use"**
A: Another application is using port 8080 or 9999
   Windows: `netstat -ano | findstr :8080`
   Linux: `lsof -i :8080`

**Q: Very slow message delivery**
A: Check network latency, browser performance, or server load

## Key Learning Points

This chat server demonstrates:
1. ✅ Concurrent programming with goroutines
2. ✅ Type-safe inter-goroutine communication with channels
3. ✅ TCP socket programming fundamentals
4. ✅ WebSocket protocol handling
5. ✅ Broadcaster/pub-sub pattern
6. ✅ Connection lifecycle management
7. ✅ Synchronization with mutexes
8. ✅ Web GUI integration with backend

---

**Next Steps:**
- Modify the code to add features (private messaging, rooms, etc.)
- Integrate with a database for persistence
- Add authentication
- Deploy to a server

Have fun coding! 🎉
