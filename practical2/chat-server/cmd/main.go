package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"

	"chatserver/pkg/server"
)

func main() {
	// Create TCP server
	tcpServer, err := server.NewServer("9999")
	if err != nil {
		fmt.Printf("Error creating TCP server: %v\n", err)
		os.Exit(1)
	}

	// Start TCP server in a goroutine
	go func() {
		if err := tcpServer.Start(); err != nil {
			fmt.Printf("Error starting TCP server: %v\n", err)
		}
	}()

	// Create WebSocket server
	wsServer := server.NewWebSocketServer()
	wsServer.StartBroadcaster()

	// Setup HTTP routes
	http.HandleFunc("/ws", wsServer.HandleWebSocket)
	http.HandleFunc("/", serveGUI)
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		tcpClients := tcpServer.GetClientCount()
		wsClients := wsServer.GetWebSocketClientCount()
		fmt.Fprintf(w, `{"tcp_clients": %d, "ws_clients": %d}`, tcpClients, wsClients)
	})

	// Start HTTP server for Web GUI
	go func() {
		fmt.Println("Web server started at http://localhost:8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			fmt.Printf("Error starting HTTP server: %v\n", err)
		}
	}()

	// CLI for broadcasting messages or checking status
	fmt.Println("\n=== Chat Server Started ===")
	fmt.Println("Servers:")
	fmt.Println("  TCP: localhost:9999 (for TCP clients)")
	fmt.Println("  Web GUI: http://localhost:8080")
	fmt.Println("\nCommands:")
	fmt.Println("  /status - Show number of connected clients")
	fmt.Println("  /broadcast <message> - Broadcast a message to all clients")
	fmt.Println("  /quit - Exit the server")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		command := parts[0]

		switch command {
		case "/status":
			fmt.Printf("TCP clients: %d, WebSocket clients: %d\n",
				tcpServer.GetClientCount(), wsServer.GetWebSocketClientCount())

		case "/broadcast":
			if len(parts) < 2 {
				fmt.Println("Usage: /broadcast <message>")
				continue
			}
			tcpServer.BroadcastMessage("ADMIN", parts[1])
			wsServer.BroadcastToWebSocket("ADMIN", parts[1])

		case "/quit":
			fmt.Println("Shutting down server...")
			os.Exit(0)

		default:
			fmt.Println("Unknown command. Type /status, /broadcast <msg>, or /quit")
		}
	}
}

func serveGUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, getGuiHTML())
}

func getGuiHTML() string {
	return `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Lounge Chat Server</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-dark: #090d16;
            --accent-indigo: #6366f1;
            --accent-indigo-glow: rgba(99, 102, 241, 0.25);
            --accent-cyan: #06b6d4;
            --accent-rose: #f43f5e;
            --accent-gold: #eab308;
            
            --active-accent: var(--accent-indigo);
            --active-accent-glow: var(--accent-indigo-glow);
            
            --panel-bg: rgba(15, 23, 42, 0.55);
            --panel-border: rgba(255, 255, 255, 0.08);
            --text-main: #f3f4f6;
            --text-muted: #9ca3af;
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
            user-select: none;
        }

        body {
            font-family: 'Outfit', sans-serif;
            background-color: var(--bg-dark);
            color: var(--text-main);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
            overflow: hidden;
            position: relative;
        }

        /* Animated Mesh Background */
        .bg-orbs {
            position: absolute;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            overflow: hidden;
            z-index: -1;
            pointer-events: none;
        }

        .orb {
            position: absolute;
            border-radius: 50%;
            filter: blur(140px);
            opacity: 0.18;
            mix-blend-mode: screen;
            animation: floatOrb 25s infinite ease-in-out alternate;
        }

        .orb-1 {
            top: -10%;
            left: -10%;
            width: 50vw;
            height: 50vw;
            background: radial-gradient(circle, var(--accent-indigo) 0%, rgba(0,0,0,0) 70%);
        }

        .orb-2 {
            bottom: -10%;
            right: -10%;
            width: 55vw;
            height: 55vw;
            background: radial-gradient(circle, var(--accent-cyan) 0%, rgba(0,0,0,0) 70%);
            animation-duration: 30s;
        }

        .orb-3 {
            top: 40%;
            left: 50%;
            width: 35vw;
            height: 35vw;
            background: radial-gradient(circle, var(--accent-rose) 0%, rgba(0,0,0,0) 70%);
            animation-duration: 20s;
        }

        @keyframes floatOrb {
            0% { transform: translate(0, 0) scale(1); }
            100% { transform: translate(60px, 40px) scale(1.1); }
        }

        /* Welcome Overlay Modal */
        .welcome-overlay {
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(8, 10, 18, 0.85);
            backdrop-filter: blur(20px);
            z-index: 100;
            display: flex;
            align-items: center;
            justify-content: center;
            transition: opacity 0.5s ease, visibility 0.5s;
        }

        .welcome-overlay.hidden {
            opacity: 0;
            visibility: hidden;
            pointer-events: none;
        }

        .welcome-card {
            background: rgba(17, 24, 39, 0.65);
            border: 1px solid var(--panel-border);
            border-radius: 20px;
            padding: 40px;
            width: 100%;
            max-width: 460px;
            box-shadow: 0 20px 50px rgba(0, 0, 0, 0.4);
            text-align: center;
            transform: scale(1);
            transition: transform 0.5s cubic-bezier(0.34, 1.56, 0.64, 1);
        }

        .welcome-overlay.hidden .welcome-card {
            transform: scale(0.9);
        }

        .welcome-icon {
            width: 70px;
            height: 70px;
            background: linear-gradient(135deg, var(--accent-indigo) 0%, var(--accent-cyan) 100%);
            border-radius: 20px;
            margin: 0 auto 20px auto;
            display: flex;
            align-items: center;
            justify-content: center;
            box-shadow: 0 8px 30px var(--accent-indigo-glow);
        }

        .welcome-icon svg {
            width: 34px;
            height: 34px;
            color: white;
        }

        .welcome-card h1 {
            font-size: 28px;
            font-weight: 700;
            margin-bottom: 8px;
            letter-spacing: -0.5px;
        }

        .welcome-card p {
            color: var(--text-muted);
            font-size: 14px;
            margin-bottom: 30px;
        }

        .welcome-form {
            text-align: left;
            margin-bottom: 25px;
        }

        .welcome-form label {
            display: block;
            font-size: 13px;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            color: var(--text-muted);
            margin-bottom: 8px;
        }

        .welcome-input {
            width: 100%;
            background: rgba(10, 15, 30, 0.7);
            border: 1px solid var(--panel-border);
            border-radius: 12px;
            padding: 14px 18px;
            color: white;
            font-family: inherit;
            font-size: 15px;
            outline: none;
            transition: border-color 0.3s, box-shadow 0.3s;
        }

        .welcome-input:focus {
            border-color: var(--active-accent);
            box-shadow: 0 0 15px var(--active-accent-glow);
        }

        .welcome-avatar-select {
            margin-top: 20px;
        }

        .avatar-options {
            display: flex;
            justify-content: center;
            gap: 12px;
            margin-top: 10px;
        }

        .avatar-dot {
            width: 38px;
            height: 38px;
            border-radius: 50%;
            cursor: pointer;
            border: 3px solid transparent;
            transition: transform 0.2s, border-color 0.2s;
        }

        .avatar-dot:hover {
            transform: scale(1.1);
        }

        .avatar-dot.selected {
            border-color: white;
            transform: scale(1.1);
        }

        .enter-btn {
            width: 100%;
            background: linear-gradient(135deg, var(--active-accent) 0%, rgba(99, 102, 241, 0.75) 100%);
            color: white;
            border: none;
            border-radius: 12px;
            padding: 14px;
            font-family: inherit;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            box-shadow: 0 8px 24px var(--active-accent-glow);
            transition: opacity 0.3s, transform 0.2s;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 10px;
        }

        .enter-btn:hover {
            transform: translateY(-1px);
            opacity: 0.95;
        }

        .enter-btn:active {
            transform: translateY(1px);
        }

        .enter-btn svg {
            width: 18px;
            height: 18px;
        }

        /* App Container */
        .app-container {
            width: 100%;
            max-width: 1100px;
            height: 800px;
            max-height: 88vh;
            background: var(--panel-bg);
            border: 1px solid var(--panel-border);
            backdrop-filter: blur(25px);
            border-radius: 24px;
            box-shadow: 0 25px 60px rgba(0, 0, 0, 0.45);
            display: flex;
            overflow: hidden;
            animation: fadeInApp 0.6s cubic-bezier(0.16, 1, 0.3, 1);
            position: relative;
        }

        @keyframes fadeInApp {
            from { opacity: 0; transform: translateY(15px); }
            to { opacity: 1; transform: translateY(0); }
        }

        /* Sidebar styling */
        .sidebar {
            width: 280px;
            background: rgba(13, 18, 30, 0.5);
            border-right: 1px solid var(--panel-border);
            display: flex;
            flex-direction: column;
            flex-shrink: 0;
        }

        .sidebar-header {
            padding: 24px;
            border-bottom: 1px solid var(--panel-border);
            display: flex;
            align-items: center;
            gap: 12px;
        }

        .logo-ring {
            width: 36px;
            height: 36px;
            border-radius: 10px;
            background: linear-gradient(135deg, var(--active-accent) 0%, var(--accent-cyan) 100%);
            display: flex;
            align-items: center;
            justify-content: center;
            box-shadow: 0 0 15px var(--active-accent-glow);
        }

        .logo-ring svg {
            width: 18px;
            height: 18px;
            color: white;
        }

        .sidebar-header h2 {
            font-size: 18px;
            font-weight: 600;
            letter-spacing: -0.3px;
        }

        .sidebar-scroll {
            flex: 1;
            overflow-y: auto;
            padding: 20px;
        }

        /* Custom Scrollbar */
        ::-webkit-scrollbar {
            width: 6px;
            height: 6px;
        }
        ::-webkit-scrollbar-track {
            background: transparent;
        }
        ::-webkit-scrollbar-thumb {
            background: rgba(255, 255, 255, 0.1);
            border-radius: 10px;
        }
        ::-webkit-scrollbar-thumb:hover {
            background: rgba(255, 255, 255, 0.2);
        }

        .sidebar-section {
            margin-bottom: 24px;
        }

        .sidebar-section h3 {
            font-size: 11px;
            font-weight: 700;
            text-transform: uppercase;
            letter-spacing: 1px;
            color: var(--text-muted);
            margin-bottom: 12px;
        }

        /* Metrics Card */
        .metrics-container {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 10px;
        }

        .metric-card {
            background: rgba(255, 255, 255, 0.03);
            border: 1px solid rgba(255, 255, 255, 0.04);
            border-radius: 12px;
            padding: 12px;
            text-align: center;
            position: relative;
            overflow: hidden;
        }

        .metric-card::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            width: 100%;
            height: 2px;
            background: linear-gradient(90deg, var(--active-accent), transparent);
        }

        .metric-val {
            display: block;
            font-size: 18px;
            font-weight: 700;
            color: white;
        }

        .metric-lbl {
            font-size: 10px;
            color: var(--text-muted);
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-top: 2px;
            display: block;
        }

        /* Online Users List */
        .user-list {
            display: flex;
            flex-direction: column;
            gap: 8px;
        }

        .user-item {
            display: flex;
            align-items: center;
            gap: 10px;
            padding: 8px 10px;
            border-radius: 10px;
            background: rgba(255, 255, 255, 0.02);
            border: 1px solid transparent;
            transition: background 0.2s, border-color 0.2s;
            cursor: pointer;
        }

        .user-item:hover {
            background: rgba(255, 255, 255, 0.05);
            border-color: rgba(255, 255, 255, 0.05);
        }

        .user-avatar {
            width: 32px;
            height: 32px;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 13px;
            font-weight: 600;
            color: white;
            box-shadow: 0 2px 10px rgba(0, 0, 0, 0.2);
            position: relative;
        }

        .user-status-dot {
            position: absolute;
            bottom: -1px;
            right: -1px;
            width: 9px;
            height: 9px;
            border-radius: 50%;
            background: #10b981;
            border: 1.5px solid #0d121e;
        }

        .user-name-info {
            flex: 1;
            min-width: 0;
        }

        .user-name-txt {
            font-size: 14px;
            font-weight: 500;
            color: var(--text-main);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .user-role-lbl {
            font-size: 10px;
            color: var(--text-muted);
        }

        /* Sidebar Footer Settings Button */
        .sidebar-footer {
            padding: 16px 20px;
            border-top: 1px solid var(--panel-border);
            display: flex;
            flex-direction: column;
            gap: 12px;
        }

        .footer-action-row {
            display: flex;
            gap: 8px;
        }

        .btn-settings {
            flex: 1;
            background: rgba(255, 255, 255, 0.04);
            border: 1px solid var(--panel-border);
            border-radius: 10px;
            color: var(--text-main);
            padding: 8px 12px;
            font-family: inherit;
            font-size: 13px;
            font-weight: 500;
            cursor: pointer;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
            transition: background 0.2s;
        }

        .btn-settings:hover {
            background: rgba(255, 255, 255, 0.08);
        }

        .btn-settings svg {
            width: 15px;
            height: 15px;
            color: var(--text-muted);
        }

        .status-pill {
            background: rgba(16, 185, 129, 0.08);
            border: 1px solid rgba(16, 185, 129, 0.15);
            border-radius: 20px;
            padding: 6px 12px;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
            font-size: 11px;
            font-weight: 600;
            color: #10b981;
        }

        .status-pill.disconnected {
            background: rgba(239, 68, 68, 0.08);
            border: 1px solid rgba(239, 68, 68, 0.15);
            color: #ef4444;
        }

        .status-pill-dot {
            width: 7px;
            height: 7px;
            border-radius: 50%;
            background: currentColor;
            box-shadow: 0 0 8px currentColor;
        }

        /* Main Chat Window */
        .chat-main {
            flex: 1;
            display: flex;
            flex-direction: column;
            background: rgba(9, 13, 22, 0.3);
            position: relative;
        }

        /* Chat Header */
        .chat-header {
            padding: 20px 24px;
            border-bottom: 1px solid var(--panel-border);
            display: flex;
            align-items: center;
            justify-content: space-between;
            z-index: 2;
        }

        .room-details {
            display: flex;
            flex-direction: column;
        }

        .room-title {
            font-size: 18px;
            font-weight: 600;
            color: white;
        }

        .server-url-badge {
            display: flex;
            align-items: center;
            gap: 6px;
            font-size: 11px;
            color: var(--text-muted);
            margin-top: 3px;
            cursor: pointer;
            background: rgba(255, 255, 255, 0.03);
            padding: 3px 8px;
            border-radius: 6px;
            border: 1px solid rgba(255, 255, 255, 0.02);
            width: fit-content;
            transition: background 0.2s, color 0.2s;
        }

        .server-url-badge:hover {
            background: rgba(255, 255, 255, 0.06);
            color: var(--text-main);
        }

        .server-url-badge svg {
            width: 11px;
            height: 11px;
        }

        .header-search {
            position: relative;
            width: 200px;
            transition: width 0.3s;
        }

        .header-search:focus-within {
            width: 260px;
        }

        .search-input {
            width: 100%;
            background: rgba(255, 255, 255, 0.04);
            border: 1px solid var(--panel-border);
            border-radius: 10px;
            padding: 8px 12px 8px 32px;
            font-family: inherit;
            font-size: 13px;
            color: white;
            outline: none;
            transition: border-color 0.3s, box-shadow 0.3s;
        }

        .search-input:focus {
            border-color: var(--active-accent);
            box-shadow: 0 0 10px var(--active-accent-glow);
        }

        .search-icon-svg {
            position: absolute;
            left: 10px;
            top: 50%;
            transform: translateY(-50%);
            width: 14px;
            height: 14px;
            color: var(--text-muted);
            pointer-events: none;
        }

        /* Message Feed Area */
        .messages-feed {
            flex: 1;
            overflow-y: auto;
            padding: 24px;
            display: flex;
            flex-direction: column;
            gap: 16px;
        }

        /* Empty State */
        .empty-feed {
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            height: 100%;
            color: var(--text-muted);
            text-align: center;
            gap: 12px;
        }

        .empty-feed svg {
            width: 48px;
            height: 48px;
            color: rgba(255, 255, 255, 0.08);
            margin-bottom: 8px;
        }

        .empty-feed h4 {
            font-size: 16px;
            color: var(--text-main);
            font-weight: 500;
        }

        .empty-feed p {
            font-size: 13px;
            max-width: 250px;
        }

        /* Message Bubbles */
        .message-row {
            display: flex;
            gap: 12px;
            max-width: 75%;
            animation: slideUpMsg 0.3s cubic-bezier(0.16, 1, 0.3, 1) forwards;
            position: relative;
        }

        @keyframes slideUpMsg {
            from { opacity: 0; transform: translateY(12px); }
            to { opacity: 1; transform: translateY(0); }
        }

        .message-row.self {
            align-self: flex-end;
            flex-direction: row-reverse;
        }

        .message-avatar-wrap {
            align-self: flex-end;
        }

        .message-content-box {
            display: flex;
            flex-direction: column;
            gap: 4px;
        }

        .message-bubble {
            padding: 12px 16px;
            border-radius: 16px;
            font-size: 14.5px;
            line-height: 1.5;
            word-break: break-word;
            box-shadow: 0 4px 15px rgba(0, 0, 0, 0.08);
        }

        .message-row.self .message-bubble {
            background: linear-gradient(135deg, var(--active-accent) 0%, rgba(99, 102, 241, 0.8) 100%);
            color: white;
            border-bottom-right-radius: 4px;
            box-shadow: 0 4px 18px var(--active-accent-glow);
        }

        .message-row.other .message-bubble {
            background: rgba(255, 255, 255, 0.045);
            border: 1px solid rgba(255, 255, 255, 0.04);
            color: var(--text-main);
            border-bottom-left-radius: 4px;
        }

        .message-meta {
            display: flex;
            align-items: center;
            gap: 8px;
            font-size: 11px;
            color: var(--text-muted);
        }

        .message-row.self .message-meta {
            justify-content: flex-end;
        }

        .message-sender {
            font-weight: 600;
            color: var(--text-main);
        }

        .message-row.other .message-sender {
            color: var(--active-accent);
        }

        /* Hover Actions inside Chat */
        .message-row:hover .msg-hover-actions {
            opacity: 1;
        }

        .msg-hover-actions {
            position: absolute;
            top: 50%;
            transform: translateY(-50%);
            display: flex;
            gap: 4px;
            opacity: 0;
            transition: opacity 0.2s;
            z-index: 10;
        }

        .message-row.self .msg-hover-actions {
            left: -80px;
        }

        .message-row.other .msg-hover-actions {
            right: -80px;
        }

        .hover-btn {
            width: 28px;
            height: 28px;
            border-radius: 6px;
            background: rgba(17, 24, 39, 0.8);
            border: 1px solid var(--panel-border);
            color: var(--text-muted);
            cursor: pointer;
            display: flex;
            align-items: center;
            justify-content: center;
            transition: color 0.2s, background 0.2s;
        }

        .hover-btn:hover {
            color: white;
            background: rgba(17, 24, 39, 0.95);
        }

        .hover-btn svg {
            width: 13px;
            height: 13px;
        }

        /* System Messages styling */
        .message-system {
            align-self: center;
            background: rgba(255, 255, 255, 0.02);
            border: 1px solid rgba(255, 255, 255, 0.04);
            padding: 8px 16px;
            border-radius: 12px;
            display: flex;
            align-items: center;
            gap: 10px;
            max-width: 90%;
            margin: 4px 0;
        }

        .system-badge {
            font-size: 9px;
            font-weight: 700;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            padding: 2px 6px;
            background: var(--active-accent);
            color: white;
            border-radius: 4px;
        }

        .message-system.admin-type .system-badge {
            background: var(--accent-gold);
            color: black;
        }

        .system-txt {
            font-size: 12.5px;
            color: var(--text-muted);
        }

        /* Chat Link */
        .chat-link {
            color: var(--accent-cyan);
            text-decoration: underline;
            word-break: break-all;
        }

        .chat-link:hover {
            color: white;
        }

        /* Code Tags */
        code {
            background: rgba(0, 0, 0, 0.3);
            padding: 2px 6px;
            border-radius: 4px;
            font-family: monospace;
            font-size: 13px;
            color: var(--accent-rose);
        }

        /* File Attachment Cards styling */
        .file-card {
            display: flex;
            align-items: center;
            gap: 12px;
            background: rgba(0, 0, 0, 0.2);
            border: 1px solid rgba(255, 255, 255, 0.06);
            border-radius: 12px;
            padding: 10px 14px;
            margin-top: 6px;
            width: 250px;
        }

        .file-icon-box {
            width: 36px;
            height: 36px;
            background: rgba(255, 255, 255, 0.05);
            border-radius: 8px;
            display: flex;
            align-items: center;
            justify-content: center;
            color: var(--active-accent);
        }

        .file-icon-box svg {
            width: 18px;
            height: 18px;
        }

        .file-info {
            flex: 1;
            min-width: 0;
        }

        .file-name-txt {
            font-size: 13px;
            font-weight: 500;
            color: white;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
        }

        .file-size-txt {
            font-size: 11px;
            color: var(--text-muted);
            margin-top: 1px;
        }

        .file-download-arrow {
            color: var(--text-muted);
            cursor: pointer;
            transition: color 0.2s;
        }

        .file-download-arrow:hover {
            color: white;
        }

        .file-download-arrow svg {
            width: 16px;
            height: 16px;
        }

        .img-attachment-preview {
            max-width: 100%;
            max-height: 200px;
            border-radius: 10px;
            margin-top: 6px;
            border: 1px solid rgba(255, 255, 255, 0.1);
            display: block;
            object-fit: cover;
        }

        /* Simulated Upload Progress styling */
        .upload-progress-card {
            background: rgba(0,0,0,0.25);
            border: 1px dashed var(--panel-border);
            border-radius: 12px;
            padding: 12px;
            width: 220px;
            display: flex;
            flex-direction: column;
            gap: 6px;
        }

        .upload-progress-header {
            display: flex;
            justify-content: space-between;
            font-size: 11px;
            color: var(--text-muted);
        }

        .progress-bar-track {
            height: 4px;
            background: rgba(255, 255, 255, 0.08);
            border-radius: 20px;
            overflow: hidden;
        }

        .progress-bar-fill {
            height: 100%;
            background: var(--active-accent);
            width: 0%;
            transition: width 0.15s ease-out;
        }

        /* Typing Status Container */
        .typing-indicator-bar {
            padding: 8px 24px;
            display: flex;
            align-items: center;
            gap: 8px;
            font-size: 12px;
            color: var(--text-muted);
            background: rgba(9, 13, 22, 0.25);
            border-top: 1px solid var(--panel-border);
            transition: opacity 0.3s, transform 0.3s;
        }

        .typing-indicator-bar.hidden {
            opacity: 0;
            transform: translateY(5px);
            pointer-events: none;
            height: 0;
            padding: 0;
            border-top: none;
            overflow: hidden;
        }

        .typing-anim-dots {
            display: flex;
            align-items: center;
            gap: 3px;
            height: 12px;
        }

        .typing-anim-dots span {
            width: 5px;
            height: 5px;
            border-radius: 50%;
            background: var(--active-accent);
            display: inline-block;
            animation: bounceDot 1.4s infinite ease-in-out both;
        }

        .typing-anim-dots span:nth-child(1) { animation-delay: -0.32s; }
        .typing-anim-dots span:nth-child(2) { animation-delay: -0.16s; }

        @keyframes bounceDot {
            0%, 80%, 100% { transform: scale(0); }
            40% { transform: scale(1); }
        }

        /* Input Controls Panel */
        .chat-input-panel {
            padding: 20px 24px;
            background: rgba(13, 18, 30, 0.45);
            border-top: 1px solid var(--panel-border);
            display: flex;
            flex-direction: column;
            gap: 10px;
            position: relative;
        }

        .input-toolbar-row {
            display: flex;
            align-items: center;
            gap: 12px;
        }

        .btn-toolbar {
            width: 38px;
            height: 38px;
            border-radius: 10px;
            background: rgba(255, 255, 255, 0.03);
            border: 1px solid var(--panel-border);
            color: var(--text-muted);
            cursor: pointer;
            display: flex;
            align-items: center;
            justify-content: center;
            transition: color 0.2s, background 0.2s, border-color 0.2s;
        }

        .btn-toolbar:hover {
            color: white;
            background: rgba(255, 255, 255, 0.07);
            border-color: rgba(255, 255, 255, 0.1);
        }

        .btn-toolbar svg {
            width: 18px;
            height: 18px;
        }

        .chat-textarea {
            flex: 1;
            background: rgba(10, 15, 30, 0.6);
            border: 1px solid var(--panel-border);
            border-radius: 12px;
            padding: 10px 16px;
            color: white;
            font-family: inherit;
            font-size: 14.5px;
            line-height: 1.5;
            outline: none;
            resize: none;
            max-height: 120px;
            transition: border-color 0.3s, box-shadow 0.3s;
        }

        .chat-textarea:focus {
            border-color: var(--active-accent);
            box-shadow: 0 0 10px var(--active-accent-glow);
        }

        .btn-send {
            width: 40px;
            height: 40px;
            border-radius: 12px;
            background: var(--active-accent);
            border: none;
            color: white;
            cursor: pointer;
            display: flex;
            align-items: center;
            justify-content: center;
            box-shadow: 0 4px 12px var(--active-accent-glow);
            transition: opacity 0.2s, transform 0.2s;
        }

        .btn-send:hover {
            transform: translateY(-1px);
            opacity: 0.95;
        }

        .btn-send:active {
            transform: translateY(1px);
        }

        .btn-send svg {
            width: 18px;
            height: 18px;
        }

        /* Emoji Grid Panel */
        .emoji-grid-panel {
            position: absolute;
            bottom: 74px;
            left: 24px;
            background: rgba(17, 24, 39, 0.95);
            border: 1px solid var(--panel-border);
            backdrop-filter: blur(20px);
            border-radius: 14px;
            padding: 12px;
            width: 250px;
            box-shadow: 0 10px 30px rgba(0, 0, 0, 0.4);
            z-index: 50;
            display: grid;
            grid-template-columns: repeat(6, 1fr);
            gap: 6px;
            animation: popInEmoji 0.25s cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
        }

        .emoji-grid-panel.hidden {
            display: none;
        }

        @keyframes popInEmoji {
            from { opacity: 0; transform: scale(0.9) translateY(10px); }
            to { opacity: 1; transform: scale(1) translateY(0); }
        }

        .emoji-grid-panel span {
            font-size: 20px;
            text-align: center;
            padding: 4px;
            cursor: pointer;
            border-radius: 8px;
            transition: background 0.15s, transform 0.15s;
        }

        .emoji-grid-panel span:hover {
            background: rgba(255, 255, 255, 0.08);
            transform: scale(1.15);
        }

        /* Settings Drawer overlay */
        .settings-drawer {
            position: absolute;
            top: 0;
            right: 0;
            width: 320px;
            height: 100%;
            background: rgba(17, 24, 39, 0.85);
            backdrop-filter: blur(25px);
            border-left: 1px solid var(--panel-border);
            z-index: 90;
            display: flex;
            flex-direction: column;
            transform: translateX(100%);
            transition: transform 0.4s cubic-bezier(0.16, 1, 0.3, 1);
            box-shadow: -10px 0 35px rgba(0, 0, 0, 0.35);
        }

        .settings-drawer.open {
            transform: translateX(0);
        }

        .settings-drawer-header {
            padding: 24px;
            border-bottom: 1px solid var(--panel-border);
            display: flex;
            align-items: center;
            justify-content: space-between;
        }

        .settings-drawer-header h3 {
            font-size: 16px;
            font-weight: 600;
        }

        .btn-close-drawer {
            background: transparent;
            border: none;
            color: var(--text-muted);
            cursor: pointer;
            display: flex;
            align-items: center;
            justify-content: center;
            transition: color 0.2s;
        }

        .btn-close-drawer:hover {
            color: white;
        }

        .btn-close-drawer svg {
            width: 20px;
            height: 20px;
        }

        .settings-drawer-body {
            flex: 1;
            overflow-y: auto;
            padding: 24px;
            display: flex;
            flex-direction: column;
            gap: 22px;
        }

        .settings-group {
            display: flex;
            flex-direction: column;
            gap: 8px;
        }

        .settings-group.row-layout {
            flex-direction: row;
            align-items: center;
            justify-content: space-between;
        }

        .settings-label {
            font-size: 13px;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            color: var(--text-muted);
        }

        .settings-desc {
            font-size: 11px;
            color: var(--text-muted);
            margin-top: 1px;
            display: block;
        }

        /* Color Theme Options grid */
        .color-theme-grid {
            display: flex;
            gap: 10px;
            margin-top: 4px;
        }

        .color-theme-opt {
            width: 32px;
            height: 32px;
            border-radius: 8px;
            cursor: pointer;
            border: 2px solid transparent;
            transition: transform 0.15s, border-color 0.15s;
        }

        .color-theme-opt:hover {
            transform: scale(1.08);
        }

        .color-theme-opt.active {
            border-color: white;
            transform: scale(1.08);
        }

        /* Toggle switches */
        .toggle-switch {
            position: relative;
            display: inline-block;
            width: 44px;
            height: 24px;
        }

        .toggle-switch input {
            opacity: 0;
            width: 0;
            height: 0;
        }

        .toggle-slider {
            position: absolute;
            cursor: pointer;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background-color: rgba(255, 255, 255, 0.08);
            border: 1px solid var(--panel-border);
            transition: .3s cubic-bezier(0.16, 1, 0.3, 1);
            border-radius: 34px;
        }

        .toggle-slider:before {
            position: absolute;
            content: "";
            height: 16px;
            width: 16px;
            left: 3px;
            bottom: 3px;
            background-color: var(--text-muted);
            transition: .3s cubic-bezier(0.16, 1, 0.3, 1);
            border-radius: 50%;
        }

        .toggle-switch input:checked + .toggle-slider {
            background-color: var(--active-accent);
        }

        .toggle-switch input:checked + .toggle-slider:before {
            transform: translateX(20px);
            background-color: white;
        }

        .divider-line {
            height: 1px;
            background: var(--panel-border);
            margin: 10px 0;
        }

        .drawer-action-btn {
            background: rgba(255, 255, 255, 0.03);
            border: 1px solid var(--panel-border);
            color: var(--text-main);
            padding: 10px 14px;
            border-radius: 10px;
            font-family: inherit;
            font-size: 13.5px;
            font-weight: 500;
            cursor: pointer;
            display: flex;
            align-items: center;
            gap: 10px;
            transition: background 0.2s, color 0.2s;
        }

        .drawer-action-btn:hover {
            background: rgba(255, 255, 255, 0.06);
        }

        .drawer-action-btn.danger-opt:hover {
            background: rgba(239, 68, 68, 0.08);
            color: #ef4444;
            border-color: rgba(239, 68, 68, 0.15);
        }

        .drawer-action-btn svg {
            width: 16px;
            height: 16px;
        }

        /* Mobile Collapsed states helper */
        @media(max-width: 768px) {
            body {
                padding: 10px;
            }
            .app-container {
                height: 100%;
                max-height: 100vh;
                border-radius: 16px;
            }
            .sidebar {
                display: none;
            }
        }
    </style>
</head>
<body>

    <!-- Animated backgrounds mesh -->
    <div class="bg-orbs">
        <div class="orb orb-1"></div>
        <div class="orb orb-2"></div>
        <div class="orb orb-3"></div>
    </div>

    <!-- Join/Welcome Overlay Screen -->
    <div class="welcome-overlay" id="welcomeOverlay">
        <div class="welcome-card">
            <div class="welcome-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
            </div>
            <h1>Lounge Chat</h1>
            <p>Connect to Go multi-client concurrent server</p>
            
            <div class="welcome-form">
                <label for="usernameInput">Your Username</label>
                <input type="text" class="welcome-input" id="usernameInput" placeholder="Type a nickname..." maxlength="15" value="User">
                
                <div class="welcome-avatar-select">
                    <label>Choose Avatar Style</label>
                    <div class="avatar-options" id="avatarColorPicker">
                        <div class="avatar-dot selected" data-color="indigo" style="background: var(--accent-indigo);"></div>
                        <div class="avatar-dot" data-color="cyan" style="background: var(--accent-cyan);"></div>
                        <div class="avatar-dot" data-color="rose" style="background: var(--accent-rose);"></div>
                        <div class="avatar-dot" data-color="gold" style="background: var(--accent-gold);"></div>
                    </div>
                </div>
            </div>
            
            <button class="enter-btn" onclick="joinLoungeLobby()">
                <span>Enter Lounge Lounge</span>
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>
            </button>
        </div>
    </div>

    <!-- Main Chat Room Container -->
    <div class="app-container" id="appContainer">
        <!-- Left Sidebar Panel -->
        <aside class="sidebar">
            <div class="sidebar-header">
                <div class="logo-ring">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
                </div>
                <h2>Lounge Room</h2>
            </div>
            
            <div class="sidebar-scroll">
                <div class="sidebar-section">
                    <h3>Server Stats</h3>
                    <div class="metrics-container">
                        <div class="metric-card">
                            <span class="metric-val" id="wsClientsCount">0</span>
                            <span class="metric-lbl">Web Tabs</span>
                        </div>
                        <div class="metric-card">
                            <span class="metric-val" id="totalMessagesCount">0</span>
                            <span class="metric-lbl">Chat Logs</span>
                        </div>
                    </div>
                </div>
                
                <div class="sidebar-section">
                    <h3>Lobby Participants (<span id="userCount">0</span>)</h3>
                    <div class="user-list" id="userList">
                        <!-- Filled on activity dynamically -->
                    </div>
                </div>
            </div>
            
            <div class="sidebar-footer">
                <div class="footer-action-row">
                    <button class="btn-settings" onclick="openSettingsPanel()">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
                        <span>Settings</span>
                    </button>
                    
                    <div class="status-pill disconnected" id="connectionStatusBadge">
                        <span class="status-pill-dot"></span>
                        <span id="connectionBadgeTxt">Offline</span>
                    </div>
                </div>
            </div>
        </aside>
        
        <!-- Right Chat Area -->
        <main class="chat-main">
            <!-- Header bar -->
            <header class="chat-header">
                <div class="room-details">
                    <span class="room-title">#general-lounge</span>
                    <div class="server-url-badge" onclick="copyWebSocketURL()">
                        <span id="socketUrlDisplay">ws://localhost:8080/ws</span>
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                    </div>
                </div>
                
                <div class="header-search">
                    <svg class="search-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
                    <input type="text" class="search-input" id="searchChatField" placeholder="Search lounge messages..." oninput="triggerSearchFilter()">
                </div>
            </header>
            
            <!-- Message List -->
            <div class="messages-feed" id="chatFeed">
                <!-- Welcome system panel -->
                <div class="message-system">
                    <span class="system-badge">System info</span>
                    <span class="system-txt">Connected. Choose Settings at bottom-left to toggle Audio Notifications or Simulate Bot users.</span>
                </div>
            </div>
            
            <!-- Typing Area -->
            <div class="typing-indicator-bar hidden" id="typingIndicatorBox">
                <div class="typing-anim-dots">
                    <span></span>
                    <span></span>
                    <span></span>
                </div>
                <span id="typingIndicatorText">Lobby user is typing...</span>
            </div>
            
            <!-- Send panel -->
            <div class="chat-input-panel">
                <!-- Emoji Grid popover -->
                <div class="emoji-grid-panel hidden" id="emojiGridPopover">
                    <span onclick="addEmojiToInput('😀')">😀</span>
                    <span onclick="addEmojiToInput('😂')">😂</span>
                    <span onclick="addEmojiToInput('🔥')">🔥</span>
                    <span onclick="addEmojiToInput('👍')">👍</span>
                    <span onclick="addEmojiToInput('❤️')">❤️</span>
                    <span onclick="addEmojiToInput('🎉')">🎉</span>
                    <span onclick="addEmojiToInput('🚀')">🚀</span>
                    <span onclick="addEmojiToInput('💡')">💡</span>
                    <span onclick="addEmojiToInput('💻')">💻</span>
                    <span onclick="addEmojiToInput('👀')">👀</span>
                    <span onclick="addEmojiToInput('✨')">✨</span>
                    <span onclick="addEmojiToInput('🤔')">🤔</span>
                </div>

                <div class="input-toolbar-row">
                    <button class="btn-toolbar" onclick="toggleEmojiPopover()" title="Pick emoji">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M8 14s1.5 2 4 2 4-2 4-2"/><line x1="9" y1="9" x2="9.01" y2="9"/><line x1="15" y1="9" x2="15.01" y2="9"/></svg>
                    </button>
                    
                    <button class="btn-toolbar" onclick="triggerFileUpload()" title="Simulate File Attachment">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48"/></svg>
                    </button>
                    <input type="file" id="guiFileSelector" class="hidden" onchange="processSimulatedUpload(this)">

                    <textarea class="chat-textarea" id="chatInputArea" placeholder="Write a message... (Enter to send)" rows="1" oninput="reportWritingStatus()"></textarea>
                    
                    <button class="btn-send" onclick="transmitMessage()">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
                    </button>
                </div>
            </div>
        </aside>

        <!-- Right drawer settings -->
        <div class="settings-drawer" id="settingsDrawer">
            <div class="settings-drawer-header">
                <h3>Lounge Settings</h3>
                <button class="btn-close-drawer" onclick="closeSettingsPanel()">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                </button>
            </div>
            
            <div class="settings-drawer-body">
                <div class="settings-group">
                    <span class="settings-label">Workspace Accent</span>
                    <span class="settings-desc">Choose glowing accent color</span>
                    <div class="color-theme-grid" id="themePicker">
                        <div class="color-theme-opt active" data-theme="indigo" style="background: var(--accent-indigo)"></div>
                        <div class="color-theme-opt" data-theme="cyan" style="background: var(--accent-cyan)"></div>
                        <div class="color-theme-opt" data-theme="rose" style="background: var(--accent-rose)"></div>
                        <div class="color-theme-opt" data-theme="gold" style="background: var(--accent-gold)"></div>
                    </div>
                </div>
                
                <div class="divider-line"></div>
                
                <div class="settings-group row-layout">
                    <div>
                        <span class="settings-label">Chime Notifications</span>
                        <span class="settings-desc">Plays retro synthesized tones</span>
                    </div>
                    <label class="toggle-switch">
                        <input type="checkbox" id="soundToggleCheck" checked>
                        <span class="toggle-slider"></span>
                    </label>
                </div>
                
                <div class="settings-group row-layout">
                    <div>
                        <span class="settings-label">Lobby Simulators (Bots)</span>
                        <span class="settings-desc">Spawns virtual participants</span>
                    </div>
                    <label class="toggle-switch">
                        <input type="checkbox" id="botSimulationToggle" onchange="handleSimulatorChange(this)">
                        <span class="toggle-slider"></span>
                    </label>
                </div>
                
                <div class="divider-line"></div>
                
                <div class="settings-group">
                    <span class="settings-label">Local Toolkits</span>
                    <button class="drawer-action-btn" onclick="downloadLogsText()" style="margin-top: 8px;">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                        <span>Save Chat Transcript</span>
                    </button>
                    <button class="drawer-action-btn danger-opt" onclick="clearFeedScreen()">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><line x1="10" y1="11" x2="10" y2="17"/><line x1="14" y1="11" x2="14" y2="17"/></svg>
                        <span>Flush Chat History</span>
                    </button>
                </div>
            </div>
        </div>
    </div>

    <script>
        let wsSocket;
        let selfUsername = 'User';
        let selfAvatarColor = 'indigo';
        
        let clientParticipants = new Set();
        let clientLogsCount = 0;
        let activityTicker = null;
        let typingTimeout = null;

        // Avatar configuration gradients
        const themeGradients = {
            indigo: 'linear-gradient(135deg, #6366f1 0%, #4f46e5 100%)',
            cyan: 'linear-gradient(135deg, #06b6d4 0%, #0891b2 100%)',
            rose: 'linear-gradient(135deg, #f43f5e 0%, #e11d48 100%)',
            gold: 'linear-gradient(135deg, #eab308 0%, #ca8a04 100%)'
        };

        const themeGlows = {
            indigo: 'rgba(99, 102, 241, 0.35)',
            cyan: 'rgba(6, 182, 212, 0.35)',
            rose: 'rgba(244, 63, 94, 0.35)',
            gold: 'rgba(234, 179, 8, 0.35)'
        };

        // Initialize Avatar selectors
        const avatarPickers = document.querySelectorAll('#avatarColorPicker .avatar-dot');
        avatarPickers.forEach(el => {
            el.addEventListener('click', () => {
                avatarPickers.forEach(p => p.classList.remove('selected'));
                el.classList.add('selected');
                selfAvatarColor = el.getAttribute('data-color');
            });
        });

        // Initialize Settings Panel theme pickers
        const themePickers = document.querySelectorAll('#themePicker .color-theme-opt');
        themePickers.forEach(el => {
            el.addEventListener('click', () => {
                themePickers.forEach(p => p.classList.remove('active'));
                el.classList.add('active');
                const selectedTheme = el.getAttribute('data-theme');
                applyAccentStyle(selectedTheme);
            });
        });

        function applyAccentStyle(theme) {
            const root = document.documentElement;
            root.style.setProperty('--active-accent', 'var(--accent-' + theme + ')');
            root.style.setProperty('--active-accent-glow', 'var(--accent-' + theme + '-glow)');
        }

        // On entry click
        function joinLoungeLobby() {
            const nameField = document.getElementById('usernameInput');
            const nickname = nameField.value.trim();
            if (!nickname) {
                alert('Please supply a handle/username to connect.');
                return;
            }

            selfUsername = nickname;
            
            // Auto connect socket
            initializeSocketLink();
            
            // Animate transition out welcome card
            const welcomeCard = document.getElementById('welcomeOverlay');
            welcomeCard.classList.add('hidden');
            
            // Show main screen
            const appWrap = document.getElementById('appContainer');
            appWrap.classList.remove('hidden');

            // Apply accent theme from selection to interface
            applyAccentStyle(selfAvatarColor);
            
            // Update local user in user list initially
            registerParticipant(selfUsername, selfAvatarColor, 'You');
            queryServerActiveCount();
            
            // Periodically ping server counts
            setInterval(queryServerActiveCount, 6000);
        }

        function queryServerActiveCount() {
            fetch('/status')
                .then(r => r.json())
                .then(data => {
                    document.getElementById('wsClientsCount').textContent = data.ws_clients || 1;
                })
                .catch(e => console.log('Ping count failed'));
        }

        function initializeSocketLink() {
            const pathProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const requestEndpoint = pathProtocol + '//' + window.location.host + '/ws';
            document.getElementById('socketUrlDisplay').textContent = requestEndpoint;

            wsSocket = new WebSocket(requestEndpoint);

            wsSocket.onopen = function() {
                toggleNetworkStatus(true);
                postSystemMessage('Server Connection established.');
            };

            wsSocket.onmessage = function(e) {
                try {
                    const msgObject = JSON.parse(e.data);
                    renderIncomingMsg(msgObject);
                } catch(err) {
                    console.log('Error parsing incoming message', err);
                }
            };

            wsSocket.onerror = function() {
                toggleNetworkStatus(false);
            };

            wsSocket.onclose = function() {
                toggleNetworkStatus(false);
                postSystemMessage('Server disconnected. Attempting to reconnect in 3s...');
                setTimeout(initializeSocketLink, 3000);
            };
        }

        function toggleNetworkStatus(isActive) {
            const pill = document.getElementById('connectionStatusBadge');
            const txt = document.getElementById('connectionBadgeTxt');
            if (isActive) {
                pill.classList.remove('disconnected');
                txt.textContent = 'Online';
            } else {
                pill.classList.add('disconnected');
                txt.textContent = 'Offline';
            }
        }

        // Generate initials avatar
        function createAvatarSymbol(name, gradientColor) {
            const initials = name.slice(0, 2).toUpperCase();
            const fillGradient = themeGradients[gradientColor] || themeGradients.indigo;
            const shadowColor = themeGlows[gradientColor] || themeGlows.indigo;
            return '<div class="user-avatar" style="background: ' + fillGradient + '; box-shadow: 0 4px 10px ' + shadowColor + ';">' +
                   initials +
                   '<span class="user-status-dot"></span>' +
                   '</div>';
        }

        function registerParticipant(name, colorKey, tag) {
            const listContainer = document.getElementById('userList');
            if (clientParticipants.has(name)) return;
            
            clientParticipants.add(name);
            document.getElementById('userCount').textContent = clientParticipants.size;

            const wrap = document.createElement('div');
            wrap.className = 'user-item';
            wrap.id = 'user-node-' + name.replace(/[^a-zA-Z0-9]/g, '');

            const avatarHTML = createAvatarSymbol(name, colorKey);
            wrap.innerHTML = avatarHTML +
                             '<div class="user-name-info">' +
                             '<div class="user-name-txt">' + escapeHtmlText(name) + '</div>' +
                             '<div class="user-role-lbl">' + (tag || 'Participant') + '</div>' +
                             '</div>';
            listContainer.appendChild(wrap);
        }

        function transmitMessage() {
            const textarea = document.getElementById('chatInputArea');
            const textContent = textarea.value.trim();
            if (!textContent) return;

            if (wsSocket && wsSocket.readyState === WebSocket.OPEN) {
                const messageLoad = {
                    from: selfUsername,
                    content: textContent,
                    timestamp: new Date().toISOString()
                };
                wsSocket.send(JSON.stringify(messageLoad));
                textarea.value = '';
                textarea.focus();
                adjustTextareaHeight(textarea);
            } else {
                alert('Connection to the Go server is currently inactive.');
            }
        }

        // Support Enter key submit without Shift key
        document.getElementById('chatInputArea').addEventListener('keydown', function(e) {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                transmitMessage();
            }
        });

        function adjustTextareaHeight(el) {
            el.style.height = 'auto';
            el.style.height = el.scrollHeight + 'px';
        }

        function renderIncomingMsg(msg) {
            const feed = document.getElementById('chatFeed');
            const date = new Date(msg.timestamp);
            const timeFormatted = date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
            
            clientLogsCount++;
            document.getElementById('totalMessagesCount').textContent = clientLogsCount;

            // Handle system admin messages
            if (msg.from === 'SERVER' || msg.from === 'ADMIN') {
                const sysRow = document.createElement('div');
                sysRow.className = 'message-system' + (msg.from === 'ADMIN' ? ' admin-type' : '');
                sysRow.innerHTML = '<span class="system-badge">' + msg.from + '</span>' +
                                   '<span class="system-txt">' + escapeHtmlText(msg.content) + '</span>';
                feed.appendChild(sysRow);
                feed.scrollTop = feed.scrollHeight;
                triggerAudioBeep();
                return;
            }

            // Map user details to lists
            const isSelf = msg.from === selfUsername;
            
            // Random color for dynamic avatars of other users
            let userColor = 'indigo';
            if (!isSelf) {
                const colors = ['indigo', 'cyan', 'rose', 'gold'];
                const charCodeSum = msg.from.split('').reduce((acc, c) => acc + c.charCodeAt(0), 0);
                userColor = colors[charCodeSum % colors.length];
                registerParticipant(msg.from, userColor, 'Lobbyist');
            } else {
                userColor = selfAvatarColor;
            }

            // Create row bubble
            const row = document.createElement('div');
            row.className = 'message-row ' + (isSelf ? 'self' : 'other');

            const avatarHTML = createAvatarSymbol(msg.from, userColor);
            
            let bubbleContent = formatContentTokens(msg.content);

            // Check if content is dynamic base64 simulated image upload
            if (msg.content.startsWith('DATA_IMAGE:')) {
                const imgData = msg.content.substring(11);
                bubbleContent = '<div class="file-name-txt">Sent an image</div>' +
                                '<img class="img-attachment-preview" src="' + imgData + '" alt="Shared image">';
            } else if (msg.content.startsWith('DATA_FILE:')) {
                const parts = msg.content.substring(10).split('|');
                const fName = parts[0] || 'attachment';
                const fSize = parts[1] || 'Unknown';
                bubbleContent = '<div class="file-card">' +
                                '<div class="file-icon-box"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg></div>' +
                                '<div class="file-info">' +
                                '<div class="file-name-txt">' + escapeHtmlText(fName) + '</div>' +
                                '<div class="file-size-txt">' + escapeHtmlText(fSize) + '</div>' +
                                '</div>' +
                                '</div>';
            }

            row.innerHTML = '<div class="message-avatar-wrap">' + avatarHTML + '</div>' +
                            '<div class="message-content-box">' +
                            '<div class="message-meta">' +
                            '<span class="message-sender">' + escapeHtmlText(msg.from) + '</span>' +
                            '<span>' + timeFormatted + '</span>' +
                            '</div>' +
                            '<div class="message-bubble">' + bubbleContent + '</div>' +
                            '</div>' +
                            '<div class="msg-hover-actions">' +
                            '<button class="hover-btn" onclick="copyBubbleText(this)" title="Copy text">' +
                            '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>' +
                            '</button>' +
                            '<button class="hover-btn" onclick="replyToUser(\'' + msg.from + '\')" title="Reply">' +
                            '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 17 4 12 9 7"/><path d="M20 18v-2a4 4 0 0 0-4-4H4"/></svg>' +
                            '</button>' +
                            '</div>';
            
            feed.appendChild(row);
            feed.scrollTop = feed.scrollHeight;

            if (!isSelf) {
                triggerAudioBeep();
            }
        }

        function postSystemMessage(txt) {
            const feed = document.getElementById('chatFeed');
            const sysRow = document.createElement('div');
            sysRow.className = 'message-system';
            sysRow.innerHTML = '<span class="system-badge">System</span>' +
                               '<span class="system-txt">' + escapeHtmlText(txt) + '</span>';
            feed.appendChild(sysRow);
            feed.scrollTop = feed.scrollHeight;
        }

        // Web Audio synthesized chimes
        function triggerAudioBeep() {
            const isSoundOn = document.getElementById('soundToggleCheck').checked;
            if (!isSoundOn) return;

            try {
                const audioCtx = new (window.AudioContext || window.webkitAudioContext)();
                const oscNode = audioCtx.createOscillator();
                const gainNode = audioCtx.createGain();
                
                oscNode.connect(gainNode);
                gainNode.connect(audioCtx.destination);
                
                oscNode.type = 'sine';
                
                // Synthesize retro digital bubble chime
                oscNode.frequency.setValueAtTime(587.33, audioCtx.currentTime); // D5
                oscNode.frequency.setValueAtTime(880.00, audioCtx.currentTime + 0.08); // A5
                
                gainNode.gain.setValueAtTime(0, audioCtx.currentTime);
                gainNode.gain.linearRampToValueAtTime(0.06, audioCtx.currentTime + 0.03);
                gainNode.gain.exponentialRampToValueAtTime(0.0001, audioCtx.currentTime + 0.25);
                
                oscNode.start(audioCtx.currentTime);
                oscNode.stop(audioCtx.currentTime + 0.25);
            } catch(e) {
                console.log('Web Audio context not supported or user gesture required', e);
            }
        }

        // String styling parser (markdown subset)
        function formatContentTokens(text) {
            let processed = escapeHtmlText(text);
            
            // Format Bold **text**
            processed = processed.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
            // Format Italic *text*
            processed = processed.replace(/\*(.*?)\*/g, '<em>$1</em>');
            // Format Code blocks
            processed = processed.replace(new RegExp(String.fromCharCode(96) + '(.*?)' + String.fromCharCode(96), 'g'), '<code>$1</code>');
            
            // Format Links
            const urlRegex = /(https?:\/\/[^\s]+)/g;
            processed = processed.replace(urlRegex, function(url) {
                return '<a href="' + url + '" target="_blank" class="chat-link">' + url + '</a>';
            });
            
            return processed;
        }

        function escapeHtmlText(unsafe) {
            return unsafe
                .replace(/&/g, "&amp;")
                .replace(/</g, "&lt;")
                .replace(/>/g, "&gt;")
                .replace(/"/g, "&quot;")
                .replace(/'/g, "&#039;");
        }

        // Message Utilities
        function copyBubbleText(btn) {
            const bubble = btn.closest('.message-row').querySelector('.message-bubble');
            navigator.clipboard.writeText(bubble.innerText);
            
            // Flash color feedback
            const originalColor = btn.style.color;
            btn.style.color = '#10b981';
            setTimeout(() => btn.style.color = originalColor, 1200);
        }

        function replyToUser(sender) {
            const input = document.getElementById('chatInputArea');
            input.value = '@' + sender + ' ' + input.value;
            input.focus();
        }

        function copyWebSocketURL() {
            const text = document.getElementById('socketUrlDisplay').textContent;
            navigator.clipboard.writeText(text);
            alert('WebSocket endpoint address copied: ' + text);
        }

        // Emoji panel toggling
        function toggleEmojiPopover() {
            const el = document.getElementById('emojiGridPopover');
            el.classList.toggle('hidden');
        }

        function addEmojiToInput(emoji) {
            const input = document.getElementById('chatInputArea');
            input.value += emoji;
            input.focus();
            document.getElementById('emojiGridPopover').classList.add('hidden');
        }

        // Dynamic Filtering
        function triggerSearchFilter() {
            const query = document.getElementById('searchChatField').value.toLowerCase();
            const messages = document.querySelectorAll('#chatFeed > div');

            messages.forEach(el => {
                if (el.classList.contains('message-system')) return;
                
                const sender = el.querySelector('.message-sender');
                const bubble = el.querySelector('.message-bubble');
                
                if (sender && bubble) {
                    const textMatch = bubble.textContent.toLowerCase().includes(query);
                    const nameMatch = sender.textContent.toLowerCase().includes(query);
                    if (textMatch || nameMatch) {
                        el.style.display = 'flex';
                    } else {
                        el.style.display = 'none';
                    }
                }
            });
        }

        // Simulated file upload pipeline
        function triggerFileUpload() {
            document.getElementById('guiFileSelector').click();
        }

        function processSimulatedUpload(input) {
            if (!input.files || !input.files[0]) return;
            const fileObj = input.files[0];
            
            // Build dynamic loading bubble in the screen
            const feed = document.getElementById('chatFeed');
            const progressRow = document.createElement('div');
            progressRow.className = 'message-system';
            progressRow.id = 'simulatedUploadCard';
            progressRow.innerHTML = '<div class="upload-progress-card">' +
                                    '<div class="upload-progress-header">' +
                                    '<span>Uploading ' + escapeHtmlText(fileObj.name) + '</span>' +
                                    '<span id="uploadProgressPct">0%</span>' +
                                    '</div>' +
                                    '<div class="progress-bar-track"><div class="progress-bar-fill" id="uploadProgressBar"></div></div>' +
                                    '</div>';
            feed.appendChild(progressRow);
            feed.scrollTop = feed.scrollHeight;

            let percentage = 0;
            const stepInterval = setInterval(() => {
                percentage += 10;
                document.getElementById('uploadProgressBar').style.width = percentage + '%';
                document.getElementById('uploadProgressPct').textContent = percentage + '%';

                if (percentage >= 100) {
                    clearInterval(stepInterval);
                    progressRow.remove();
                    completeSimulatedUpload(fileObj);
                }
            }, 120);
        }

        function completeSimulatedUpload(fileObj) {
            const isImage = fileObj.type.startsWith('image/');
            const reader = new FileReader();

            reader.onload = function(e) {
                const finalData = e.target.result;
                let payload = '';

                if (isImage) {
                    payload = 'DATA_IMAGE:' + finalData;
                } else {
                    const sizeMB = (fileObj.size / (1024 * 1024)).toFixed(2) + ' MB';
                    payload = 'DATA_FILE:' + fileObj.name + '|' + sizeMB;
                }

                // Send through socket
                if (wsSocket && wsSocket.readyState === WebSocket.OPEN) {
                    const messageLoad = {
                        from: selfUsername,
                        content: payload,
                        timestamp: new Date().toISOString()
                    };
                    wsSocket.send(JSON.stringify(messageLoad));
                }
            };

            if (isImage) {
                // Compress/resize image locally if too large to fit in WebSocket
                reader.readAsDataURL(fileObj);
            } else {
                // Just use file metadata
                const sizeMB = (fileObj.size / (1024 * 1024)).toFixed(2) + ' MB';
                const simulatedPayload = 'DATA_FILE:' + fileObj.name + '|' + sizeMB;
                if (wsSocket && wsSocket.readyState === WebSocket.OPEN) {
                    wsSocket.send(JSON.stringify({
                        from: selfUsername,
                        content: simulatedPayload,
                        timestamp: new Date().toISOString()
                    }));
                }
            }
        }

        // Settings Drawer controls
        function openSettingsPanel() {
            document.getElementById('settingsDrawer').classList.add('open');
        }

        function closeSettingsPanel() {
            document.getElementById('settingsDrawer').classList.remove('open');
        }

        function clearFeedScreen() {
            if (confirm('Flush all local messages from screen?')) {
                const feed = document.getElementById('chatFeed');
                feed.innerHTML = '<div class="message-system"><span class="system-badge">System info</span><span class="system-txt">Screen cleared by user.</span></div>';
                clientLogsCount = 0;
                document.getElementById('totalMessagesCount').textContent = 0;
                closeSettingsPanel();
            }
        }

        function downloadLogsText() {
            const feed = document.getElementById('chatFeed');
            const messages = feed.querySelectorAll('.message-row, .message-system');
            let outputText = '=== LOUNGE CHAT ROOM LOGS ===\nExported: ' + new Date().toString() + '\n\n';

            messages.forEach(el => {
                if (el.classList.contains('message-system')) {
                    const txt = el.querySelector('.system-txt').textContent;
                    const b = el.querySelector('.system-badge').textContent;
                    outputText += '[' + b + '] ' + txt + '\n';
                } else {
                    const sender = el.querySelector('.message-sender').textContent;
                    const bubble = el.querySelector('.message-bubble').textContent;
                    const meta = el.querySelector('.message-meta span:last-child').textContent;
                    outputText += '[' + meta + '] ' + sender + ': ' + bubble + '\n';
                }
            });

            const link = document.createElement('a');
            link.href = 'data:text/plain;charset=utf-8,' + encodeURIComponent(outputText);
            link.download = 'lounge-chat-transcript.txt';
            link.click();
            closeSettingsPanel();
        }

        // Bot Simulator engine
        function handleSimulatorChange(chk) {
            if (chk.checked) {
                postSystemMessage('Bot Simulation Simulator started.');
                startLocalBotActivity();
            } else {
                postSystemMessage('Bot Simulation Simulator stopped.');
                if (activityTicker) {
                    clearInterval(activityTicker);
                    activityTicker = null;
                }
            }
        }

        function startLocalBotActivity() {
            const botNames = ['Trinity (Bot)', 'Neo (Bot)', 'Morpheus (Bot)', 'Cypher (Bot)'];
            const botColors = {
                'Trinity (Bot)': 'rose',
                'Neo (Bot)': 'cyan',
                'Morpheus (Bot)': 'gold',
                'Cypher (Bot)': 'indigo'
            };
            const botMessages = [
                'Are you connected via Telnet too? Or raw Go sockets?',
                'I know kung fu.',
                'There is no spoon.',
                'The Matrix is everywhere. It is all around us.',
                'Fate, it seems, is not without a sense of irony.',
                'Remember... all I\'m offering is the truth. Nothing more.',
                'Did anyone try sending a file? Base64 parsing is awesome!',
                'Go concurrent broadcaster works perfectly in multiple terminals.',
                'This UI is so smooth. Love the frosted glass.'
            ];

            activityTicker = setInterval(() => {
                const randomBot = botNames[Math.floor(Math.random() * botNames.length)];
                const botColor = botColors[randomBot];
                const randomText = botMessages[Math.floor(Math.random() * botMessages.length)];

                // Show typing indicator
                showTypingIndicator(randomBot);

                setTimeout(() => {
                    hideTypingIndicator();
                    
                    // Transmit bot message through active socket so other windows see it!
                    if (wsSocket && wsSocket.readyState === WebSocket.OPEN) {
                        wsSocket.send(JSON.stringify({
                            from: randomBot,
                            content: randomText,
                            timestamp: new Date().toISOString()
                        }));
                    }
                }, 1800);

            }, 12000);
        }

        function showTypingIndicator(name) {
            const bar = document.getElementById('typingIndicatorBox');
            document.getElementById('typingIndicatorText').textContent = name + ' is writing...';
            bar.classList.remove('hidden');
        }

        function hideTypingIndicator() {
            const bar = document.getElementById('typingIndicatorBox');
            bar.classList.add('hidden');
        }

        // Self typing status reporter (local throttle)
        function reportWritingStatus() {
            if (typingTimeout) clearTimeout(typingTimeout);
            
            // To simulate local self writing status for bot responses
            typingTimeout = setTimeout(() => {
                // Done writing
            }, 1500);
        }
    </script>
</body>
</html>
`
}
