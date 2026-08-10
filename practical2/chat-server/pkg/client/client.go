package client

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

// ChatClient represents a chat client
type ChatClient struct {
	Conn     net.Conn
	Reader   *bufio.Reader
	Writer   *bufio.Writer
	Username string
	Messages chan string
	Done     chan bool
}

// NewChatClient creates a new chat client
func NewChatClient(addr, username string) (*ChatClient, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	client := &ChatClient{
		Conn:     conn,
		Reader:   bufio.NewReader(conn),
		Writer:   bufio.NewWriter(conn),
		Username: username,
		Messages: make(chan string, 10),
		Done:     make(chan bool),
	}

	return client, nil
}

// SendMessage sends a message to the server
func (c *ChatClient) SendMessage(content string) error {
	msg := fmt.Sprintf("%s: %s\n", c.Username, content)
	_, err := c.Writer.WriteString(msg)
	if err != nil {
		return err
	}
	return c.Writer.Flush()
}

// ReceiveMessages receives messages from the server in a goroutine
func (c *ChatClient) ReceiveMessages() {
	defer c.Close()

	for {
		line, err := c.Reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading from server: %v\n", err)
			c.Done <- true
			break
		}

		if len(line) > 0 {
			c.Messages <- line
		}
	}
}

// GetServerMessages returns the channel for receiving messages
func (c *ChatClient) GetServerMessages() <-chan string {
	return c.Messages
}

// Close closes the connection
func (c *ChatClient) Close() error {
	return c.Conn.Close()
}

// IsConnected checks if the client is connected
func (c *ChatClient) IsConnected() bool {
	if c.Conn == nil {
		return false
	}

	// Try to write a dummy message to check connection
	c.Conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
	_, err := c.Conn.Write([]byte{})
	c.Conn.SetWriteDeadline(time.Time{})

	return err == nil
}
