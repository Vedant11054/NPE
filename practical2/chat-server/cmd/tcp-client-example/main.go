package main
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: tcp-client <server:port> <username>")
		fmt.Println("Example: tcp-client localhost:9999 Alice")
		os.Exit(1)
	}

	server := os.Args[1]
	username := os.Args[2]

	// Connect to server
	conn, err := net.Dial("tcp", server)
	if err != nil {
		fmt.Printf("Error connecting to server: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("Connected to %s as %s\n", server, username)
	fmt.Println("Type messages to send (type 'quit' to exit):")

	// Create reader and writer
	reader := bufio.NewReader(os.Stdin)
	connReader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// Goroutine to receive messages
	go func() {
		for {
			line, err := connReader.ReadString('\n')
			if err != nil {
				fmt.Printf("\nDisconnected from server\n")
				os.Exit(0)
			}
			if len(line) > 0 {
				fmt.Printf("\n%s", line)
				fmt.Print("> ")
			}
		}
	}()

	// Main loop to send messages
	for {
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "quit" {
			fmt.Println("Disconnecting...")
			break
		}

		if input == "" {
			continue
		}

		// Send message with username
		message := fmt.Sprintf("%s: %s\n", username, input)
		_, err := writer.WriteString(message)
		if err != nil {
			fmt.Printf("Error sending message: %v\n", err)
			break
		}
		writer.Flush()
	}
}
