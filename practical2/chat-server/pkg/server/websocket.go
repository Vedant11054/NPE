package server

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketClient represents a WebSocket client connection
type WebSocketClient struct {
	ID       string
	Username string
	Conn     *websocket.Conn
	Send     chan Message
	mu       sync.Mutex
}

// WebSocketServer manages WebSocket connections
type WebSocketServer struct {
	Clients        map[string]*WebSocketClient
	Broadcast      chan Message
	AddWSClient    chan *WebSocketClient
	RemoveWSClient chan string
	mu             sync.RWMutex
	upgrader       websocket.Upgrader
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer() *WebSocketServer {
	return &WebSocketServer{
		Clients:        make(map[string]*WebSocketClient),
		Broadcast:      make(chan Message, 10),
		AddWSClient:    make(chan *WebSocketClient, 10),
		RemoveWSClient: make(chan string, 10),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all connections in this demo
			},
		},
	}
}

// HandleWebSocket handles WebSocket connections
func (ws *WebSocketServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("WebSocket upgrade error: %v\n", err)
		return
	}

	client := &WebSocketClient{
		ID:   conn.RemoteAddr().String(),
		Conn: conn,
		Send: make(chan Message, 10),
	}

	ws.AddWSClient <- client

	fmt.Printf("WebSocket client %s connected\n", client.ID)

	// Handle incoming messages
	go ws.handleWSClient(client)

	// Send messages to WebSocket client
	go func() {
		for msg := range client.Send {
			client.mu.Lock()
			err := conn.WriteJSON(msg)
			client.mu.Unlock()
			if err != nil {
				fmt.Printf("WebSocket write error: %v\n", err)
				break
			}
		}
	}()
}

// handleWSClient handles incoming messages from a WebSocket client
func (ws *WebSocketServer) handleWSClient(client *WebSocketClient) {
	defer func() {
		ws.RemoveWSClient <- client.ID
		client.Conn.Close()
	}()

	for {
		var msg Message
		err := client.Conn.ReadJSON(&msg)
		if err != nil {
			fmt.Printf("WebSocket read error: %v\n", err)
			break
		}

		if msg.From != "" {
			client.Username = msg.From
			msg.Timestamp = getNowTime()
			ws.Broadcast <- msg
		}
	}
}

// StartBroadcaster starts the broadcaster goroutine
func (ws *WebSocketServer) StartBroadcaster() {
	go func() {
		for {
			select {
			case client := <-ws.AddWSClient:
				ws.mu.Lock()
				ws.Clients[client.ID] = client
				ws.mu.Unlock()
				fmt.Printf("WebSocket client added. Total: %d\n", len(ws.Clients))

			case clientID := <-ws.RemoveWSClient:
				ws.mu.Lock()
				delete(ws.Clients, clientID)
				ws.mu.Unlock()
				fmt.Printf("WebSocket client removed. Total: %d\n", len(ws.Clients))

			case msg := <-ws.Broadcast:
				ws.mu.RLock()
				clients := ws.Clients
				ws.mu.RUnlock()

				for _, client := range clients {
					select {
					case client.Send <- msg:
					default:
						// Channel full, skip
					}
				}

				fmt.Printf("[WS] %s: %s\n", msg.From, msg.Content)
			}
		}
	}()
}

// BroadcastToWebSocket broadcasts a message to all WebSocket clients
func (ws *WebSocketServer) BroadcastToWebSocket(from, content string) {
	msg := Message{
		From:      from,
		Content:   content,
		Timestamp: getNowTime(),
	}
	select {
	case ws.Broadcast <- msg:
	default:
		// Channel full
	}
}

// GetWebSocketClientCount returns the number of connected WebSocket clients
func (ws *WebSocketServer) GetWebSocketClientCount() int {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return len(ws.Clients)
}

// Helper function to get current time
func getNowTime() time.Time {
	return time.Now()
}
