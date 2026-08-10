package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// TCPConnection represents a TCP connection entry
type TCPConnection struct {
	LocalAddr   string `json:"local_addr"`
	RemoteAddr  string `json:"remote_addr"`
	State       string `json:"state"`
	PID         string `json:"pid"`
	ProcessName string `json:"process_name"`
}

// APIResponse represents the API response
type APIResponse struct {
	Success     bool             `json:"success"`
	Message     string           `json:"message"`
	Connections []*TCPConnection `json:"connections"`
	Count       int              `json:"count"`
	Timestamp   string           `json:"timestamp"`
}

// getTCPConnections retrieves all TCP connections from the system
func getTCPConnections() []*TCPConnection {
	var connections []*TCPConnection

	// Get IPv4 TCP connections using netstat
	cmd := exec.Command("netstat", "-ano", "-p", "tcp")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error running netstat: %v", err)
		return connections
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "TCP") || strings.HasPrefix(line, "  [") {
			conn := parseTCPLine(line)
			if conn != nil {
				conn.ProcessName = getProcessName(conn.PID)
				connections = append(connections, conn)
			}
		}
	}

	return connections
}

// parseTCPLine parses a single line from netstat output
func parseTCPLine(line string) *TCPConnection {
	// Remove extra spaces
	line = strings.Join(strings.Fields(line), " ")
	parts := strings.Split(line, " ")

	if len(parts) < 5 {
		return nil
	}

	// Skip header lines
	if strings.EqualFold(parts[0], "Proto") {
		return nil
	}

	// Extract fields based on netstat -ano output format
	localAddr := parts[1]
	remoteAddr := parts[2]
	state := parts[3]
	pid := parts[4]

	if strings.Contains(localAddr, ":") && strings.Contains(remoteAddr, ":") {
		return &TCPConnection{
			LocalAddr:  localAddr,
			RemoteAddr: remoteAddr,
			State:      state,
			PID:        pid,
		}
	}

	return nil
}

// getProcessName retrieves the process name from PID
func getProcessName(pid string) string {
	if pid == "" || pid == "PID" {
		return "N/A"
	}

	// Use tasklist to get process name
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %s"), "/NH", "/FO", "CSV")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "Unknown"
	}

	result := strings.TrimSpace(string(output))
	if result != "" && !strings.Contains(result, "No tasks") {
		// Remove quotes and extract process name
		result = strings.Trim(result, "\"")
		return result
	}

	return "Unknown"
}

// openBrowser opens the application in the default browser
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Run(); err != nil {
		log.Printf("Could not open browser: %v", err)
	}
}

func main() {
	port := 8080
	url := fmt.Sprintf("http://localhost:%d", port)

	log.Println("Starting TCP Routes Viewer...")
	log.Printf("Open your browser at: %s", url)

	// Open browser after short delay to ensure server is ready
	go func() {
		time.Sleep(1 * time.Second)
		openBrowser(url)
	}()

	// Serve static files
	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/api/connections", handleConnections)
	http.HandleFunc("/api/resolve", handleResolve)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Server listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// serveHTML serves the HTML interface
func serveHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, getHTML())
}

// handleConnections returns all TCP connections as JSON
func handleConnections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	connections := getTCPConnections()
	response := APIResponse{
		Success:     true,
		Message:     fmt.Sprintf("Found %d TCP connections", len(connections)),
		Connections: connections,
		Count:       len(connections),
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// handleResolve resolves an IP address to a hostname
func handleResolve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ip := r.URL.Query().Get("ip")

	if ip == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "IP parameter required",
		})
		return
	}

	names, err := net.LookupAddr(ip)
	hostname := "Unknown"
	if err == nil && len(names) > 0 {
		hostname = strings.TrimSuffix(names[0], ".")
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"ip":       ip,
		"hostname": hostname,
	})
}

// getHTML returns the HTML content
func getHTML() string {
	return `<!DOCTYPE html>
<html>
<head>
	<title>TCP Routes Viewer</title>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>
		* {
			margin: 0;
			padding: 0;
			box-sizing: border-box;
		}
		
		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
			background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
			min-height: 100vh;
			padding: 20px;
		}
		
		.container {
			max-width: 1400px;
			margin: 0 auto;
			background: white;
			border-radius: 12px;
			box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
			overflow: hidden;
		}
		
		.header {
			background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
			color: white;
			padding: 30px;
			text-align: center;
		}
		
		.header h1 {
			font-size: 2.5em;
			margin-bottom: 10px;
		}
		
		.header p {
			font-size: 1.1em;
			opacity: 0.9;
		}
		
		.controls {
			padding: 20px 30px;
			background: #f8f9fa;
			border-bottom: 1px solid #e0e0e0;
			display: flex;
			gap: 10px;
			flex-wrap: wrap;
			align-items: center;
		}
		
		.btn {
			padding: 10px 20px;
			background: #667eea;
			color: white;
			border: none;
			border-radius: 6px;
			cursor: pointer;
			font-size: 1em;
			transition: all 0.3s ease;
		}
		
		.btn:hover {
			background: #764ba2;
			transform: translateY(-2px);
			box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
		}
		
		.btn.secondary {
			background: #6c757d;
		}
		
		.btn.secondary:hover {
			background: #5a6268;
		}
		
		.btn:disabled {
			opacity: 0.6;
			cursor: not-allowed;
			transform: none;
		}
		
		.status {
			flex: 1;
			padding: 10px 15px;
			background: white;
			border-radius: 6px;
			font-size: 0.95em;
		}
		
		.status.loading {
			color: #667eea;
		}
		
		.status.success {
			color: #28a745;
		}
		
		.status.error {
			color: #dc3545;
		}
		
		.table-wrapper {
			overflow-x: auto;
		}
		
		table {
			width: 100%;
			border-collapse: collapse;
			font-size: 0.95em;
		}
		
		thead {
			background: #f8f9fa;
			position: sticky;
			top: 0;
		}
		
		th {
			padding: 15px;
			text-align: left;
			font-weight: 600;
			color: #333;
			border-bottom: 2px solid #e0e0e0;
		}
		
		td {
			padding: 12px 15px;
			border-bottom: 1px solid #e0e0e0;
		}
		
		tbody tr:hover {
			background: #f8f9fa;
		}
		
		tbody tr:nth-child(even) {
			background: #fafbfc;
		}
		
		.state-established {
			color: #28a745;
			font-weight: 600;
		}
		
		.state-listening {
			color: #0066cc;
			font-weight: 600;
		}
		
		.state-time-wait {
			color: #ff9800;
			font-weight: 600;
		}
		
		.state-close-wait {
			color: #ff6b6b;
			font-weight: 600;
		}
		
		.mono {
			font-family: "Courier New", monospace;
			font-size: 0.9em;
		}
		
		.footer {
			padding: 20px 30px;
			background: #f8f9fa;
			border-top: 1px solid #e0e0e0;
			text-align: center;
			color: #666;
			font-size: 0.9em;
		}
		
		.empty {
			text-align: center;
			padding: 50px 30px;
			color: #999;
		}
		
		.filter-box {
			padding: 10px 15px;
			background: white;
			border: 1px solid #ddd;
			border-radius: 6px;
			flex: 1;
			min-width: 250px;
		}
		
		.filter-box input {
			width: 100%;
			border: none;
			outline: none;
			font-size: 1em;
		}
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1>TCP Routes Viewer</h1>
			<p>Real-time TCP connection monitoring for Windows</p>
		</div>
		
		<div class="controls">
			<button class="btn" onclick="refreshConnections()">
				Refresh Routes
			</button>
			<button class="btn secondary" onclick="exportCSV()">
				Export CSV
			</button>
			<div class="filter-box">
				<input type="text" id="filterInput" placeholder="Filter by process name, IP, or state...">
			</div>
			<div class="status" id="status"></div>
		</div>
		
		<div class="table-wrapper">
			<table id="connectionsTable">
				<thead>
					<tr>
						<th>Local Address</th>
						<th>Remote Address</th>
						<th>State</th>
						<th>PID</th>
						<th>Process Name</th>
					</tr>
				</thead>
				<tbody id="connectionsBody">
					<tr>
						<td colspan="5" class="empty">Click "Refresh Routes" to load TCP connections...</td>
					</tr>
				</tbody>
			</table>
		</div>
		
		<div class="footer">
			<p>Last updated: <span id="lastUpdate">Never</span></p>
		</div>
	</div>
	
	<script>
		let allConnections = [];
		
		const filterInput = document.getElementById("filterInput");
		filterInput.addEventListener("input", filterConnections);
		
		async function refreshConnections() {
			const statusEl = document.getElementById("status");
			statusEl.textContent = "Loading TCP routes...";
			statusEl.className = "status loading";
			
			const btn = event.target;
			btn.disabled = true;
			
			try {
				const response = await fetch("/api/connections");
				const data = await response.json();
				
				if (data.success) {
					allConnections = data.connections || [];
					displayConnections(allConnections);
					statusEl.textContent = "Found " + data.count + " TCP connections";
					statusEl.className = "status success";
					document.getElementById("lastUpdate").textContent = new Date().toLocaleTimeString();
				} else {
					statusEl.textContent = "Error: " + data.message;
					statusEl.className = "status error";
				}
			} catch (error) {
				statusEl.textContent = "Error loading connections: " + error;
				statusEl.className = "status error";
			} finally {
				btn.disabled = false;
			}
		}
		
		function displayConnections(connections) {
			const tbody = document.getElementById("connectionsBody");
			
			if (!connections || connections.length === 0) {
				tbody.innerHTML = "<tr><td colspan=\"5\" class=\"empty\">No TCP connections found</td></tr>";
				return;
			}
			
			tbody.innerHTML = connections.map(function(conn) {
				const stateClass = getStateClass(conn.state);
				return "<tr>" +
					"<td class=\"mono\">" + escapeHtml(conn.local_addr) + "</td>" +
					"<td class=\"mono\">" + escapeHtml(conn.remote_addr) + "</td>" +
					"<td class=\"" + stateClass + "\">" + escapeHtml(conn.state) + "</td>" +
					"<td class=\"mono\">" + escapeHtml(conn.pid) + "</td>" +
					"<td>" + escapeHtml(conn.process_name) + "</td>" +
					"</tr>";
			}).join("");
		}
		
		function escapeHtml(text) {
			const div = document.createElement("div");
			div.textContent = text;
			return div.innerHTML;
		}
		
		function getStateClass(state) {
			const normalized = state.toUpperCase().replace(/-/g, "_");
			if (normalized === "ESTABLISHED") return "state-established";
			if (normalized === "LISTENING") return "state-listening";
			if (normalized === "TIME_WAIT") return "state-time-wait";
			if (normalized === "CLOSE_WAIT") return "state-close-wait";
			return "";
		}
		
		function filterConnections() {
			const filterText = filterInput.value.toLowerCase();
			const filtered = allConnections.filter(function(conn) {
				const text = (
					conn.local_addr +
					conn.remote_addr +
					conn.state +
					conn.process_name +
					conn.pid
				).toLowerCase();
				return text.includes(filterText);
			});
			displayConnections(filtered);
		}
		
		function exportCSV() {
			if (!allConnections || allConnections.length === 0) {
				alert("No connections to export. Refresh first!");
				return;
			}
			
			let csv = "Local Address,Remote Address,State,PID,Process Name\n";
			csv += allConnections.map(function(conn) {
				return "\"" + conn.local_addr + "\",\"" + conn.remote_addr + "\",\"" + conn.state + "\",\"" + conn.pid + "\",\"" + conn.process_name + "\"";
			}).join("\n");
			
			const blob = new Blob([csv], { type: "text/csv" });
			const url = window.URL.createObjectURL(blob);
			const a = document.createElement("a");
			a.href = url;
			a.download = "tcp-routes-" + new Date().toISOString() + ".csv";
			a.click();
			window.URL.revokeObjectURL(url);
		}
		
		document.addEventListener("DOMContentLoaded", refreshConnections);
	</script>
</body>
</html>`
}
