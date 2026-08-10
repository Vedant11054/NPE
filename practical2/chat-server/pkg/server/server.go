package server

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"
)

// Message represents a chat message
type Message struct {
	From      string    `json:"from"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Client represents a connected client
type Client struct {
	ID       string
	Conn     net.Conn
	Reader   *bufio.Reader
	Send     chan Message
	Done     chan bool
	Username string
}

// Server represents the chat server
type Server struct {
	Listener     net.Listener
	Clients      map[string]*Client
	Broadcast    chan Message
	AddClient    chan *Client
	RemoveClient chan string
	mu           sync.RWMutex
}

// NewServer creates a new chat server
func NewServer(port string) (*Server, error) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, err
	}

	server := &Server{
		Listener:     listener,
		Clients:      make(map[string]*Client),
		Broadcast:    make(chan Message, 10),
		AddClient:    make(chan *Client, 10),
		RemoveClient: make(chan string, 10),
	}

	return server, nil
}

// Start starts the server
func (s *Server) Start() error {
	fmt.Printf("Chat server started on %s\n", s.Listener.Addr().String())

	// Start the broadcaster goroutine
	go s.broadcaster()

	// Start accepting connections
	go s.acceptConnections()

	// Block forever
	select {}
}

// acceptConnections accepts incoming TCP connections
func (s *Server) acceptConnections() {
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}

		// Create a new client for each connection
		client := &Client{
			ID:     conn.RemoteAddr().String(),
			Conn:   conn,
			Reader: bufio.NewReader(conn),
			Send:   make(chan Message, 10),
			Done:   make(chan bool),
		}

		// Send client to add channel
		s.AddClient <- client

		// Handle client communication in a separate goroutine
		go s.handleClient(client)
	}
}

// handleClient handles a single client connection
func (s *Server) handleClient(client *Client) {
	defer func() {
		s.RemoveClient <- client.ID
		client.Conn.Close()
	}()

	// Send welcome message
	welcomeMsg := Message{
		From:      "SERVER",
		Content:   fmt.Sprintf("%s connected to the chat", client.ID),
		Timestamp: time.Now(),
	}
	s.Broadcast <- welcomeMsg

	// Read messages from client
	for {
		line, err := client.Reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Client %s disconnected\n", client.ID)
			break
		}

		if len(line) > 1 {
			// Remove newline
			msg := Message{
				From:      client.Username,
				Content:   line[:len(line)-1],
				Timestamp: time.Now(),
			}
			s.Broadcast <- msg
		}
	}

	// Send disconnect message
	disconnectMsg := Message{
		From:      "SERVER",
		Content:   fmt.Sprintf("%s disconnected from the chat", client.Username),
		Timestamp: time.Now(),
	}
	s.Broadcast <- disconnectMsg
}

// broadcaster handles message broadcasting to all clients
func (s *Server) broadcaster() {
	for {
		select {
		case client := <-s.AddClient:
			s.mu.Lock()
			s.Clients[client.ID] = client
			s.mu.Unlock()
			fmt.Printf("Client %s connected. Total clients: %d\n", client.ID, len(s.Clients))

		case clientID := <-s.RemoveClient:
			s.mu.Lock()
			delete(s.Clients, clientID)
			s.mu.Unlock()
			fmt.Printf("Client %s removed. Total clients: %d\n", clientID, len(s.Clients))

		case msg := <-s.Broadcast:
			s.mu.RLock()
			clients := s.Clients
			s.mu.RUnlock()

			// Send message to all connected clients
			for _, client := range clients {
				select {
				case client.Send <- msg:
				case <-client.Done:
					// Client is already closed
				default:
					// Channel full, skip
				}
			}

			// Print message to server console
			fmt.Printf("[%s] %s: %s\n", msg.Timestamp.Format("15:04:05"), msg.From, msg.Content)
		}
	}
}

// SendToClient sends a message to a specific client
func (s *Server) SendToClient(client *Client, msg Message) {
	select {
	case client.Send <- msg:
	case <-client.Done:
		fmt.Printf("Client %s is closed\n", client.ID)
	}
}

// GetClientCount returns the number of connected clients
func (s *Server) GetClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Clients)
}

// BroadcastMessage broadcasts a message to all clients
func (s *Server) BroadcastMessage(from, content string) {
	msg := Message{
		From:      from,
		Content:   content,
		Timestamp: time.Now(),
	}
	s.Broadcast <- msg
}
