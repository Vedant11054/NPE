package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type TCPConnection struct {
	LocalAddr   string `json:"local_addr"`
	RemoteAddr  string `json:"remote_addr"`
	State       string `json:"state"`
	PID         string `json:"pid"`
	ProcessName string `json:"process_name"`
	Timestamp   string `json:"timestamp,omitempty"`
	ID          int    `json:"id,omitempty"`
}

type APIResponse struct {
	Success     bool             `json:"success"`
	Message     string           `json:"message"`
	Connections []*TCPConnection `json:"connections"`
	Count       int              `json:"count"`
	Timestamp   string           `json:"timestamp"`
}

var db *sql.DB

func getTCPConnections() []*TCPConnection {
	var connections []*TCPConnection

	cmd := exec.Command("netstat", "-ano", "-p", "tcp")
	output, err := cmd.CombinedOutput()
	if err != nil {
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

	sort.Slice(connections, func(i, j int) bool {
		return connections[i].LocalAddr < connections[j].LocalAddr
	})

	return connections
}

func parseTCPLine(line string) *TCPConnection {
	line = strings.Join(strings.Fields(line), " ")
	parts := strings.Split(line, " ")

	if len(parts) < 5 {
		return nil
	}

	if strings.EqualFold(parts[0], "Proto") {
		return nil
	}

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

func getProcessName(pid string) string {
	if pid == "" || pid == "PID" {
		return "N/A"
	}

	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %s", pid), "/NH")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "Unknown"
	}

	result := strings.TrimSpace(string(output))
	if result != "" && !strings.Contains(result, "No tasks") {
		parts := strings.Fields(result)
		if len(parts) > 0 {
			return parts[0]
		}
		return result
	}

	return "Unknown"
}

func initDB() error {
	var err error
	db, err = sql.Open("sqlite3", "./tcp_routes_history.db")
	if err != nil {
		return err
	}

	// Create tables if they don't exist
	schema := `
	CREATE TABLE IF NOT EXISTS tcp_connections (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		local_addr TEXT NOT NULL,
		remote_addr TEXT NOT NULL,
		state TEXT NOT NULL,
		pid TEXT NOT NULL,
		process_name TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_timestamp ON tcp_connections(timestamp);
	CREATE INDEX IF NOT EXISTS idx_process ON tcp_connections(process_name);
	`

	_, err = db.Exec(schema)
	return err
}

func storeConnections(connections []*TCPConnection) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	now := time.Now().Format(time.RFC3339)
	stmt, err := db.Prepare(`
		INSERT INTO tcp_connections (local_addr, remote_addr, state, pid, process_name, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, conn := range connections {
		_, err := stmt.Exec(conn.LocalAddr, conn.RemoteAddr, conn.State, conn.PID, conn.ProcessName, now)
		if err != nil {
			fmt.Printf("Error storing connection: %v\n", err)
		}
	}

	return nil
}

func getHistory(limit int, offset int) ([]*TCPConnection, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	if limit <= 0 {
		limit = 100
	}

	rows, err := db.Query(`
		SELECT id, local_addr, remote_addr, state, pid, process_name, timestamp
		FROM tcp_connections
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []*TCPConnection
	for rows.Next() {
		var conn TCPConnection
		err := rows.Scan(&conn.ID, &conn.LocalAddr, &conn.RemoteAddr, &conn.State, &conn.PID, &conn.ProcessName, &conn.Timestamp)
		if err != nil {
			fmt.Printf("Error scanning row: %v\n", err)
			continue
		}
		connections = append(connections, &conn)
	}

	return connections, rows.Err()
}

func getHistoryStats() (map[string]interface{}, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var totalCount int
	var firstRecord, lastRecord string

	err := db.QueryRow("SELECT COUNT(*) FROM tcp_connections").Scan(&totalCount)
	if err != nil {
		return nil, err
	}

	err = db.QueryRow("SELECT MIN(timestamp) FROM tcp_connections").Scan(&firstRecord)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	err = db.QueryRow("SELECT MAX(timestamp) FROM tcp_connections").Scan(&lastRecord)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return map[string]interface{}{
		"total_records": totalCount,
		"first_record":  firstRecord,
		"last_record":   lastRecord,
	}, nil
}

func openBrowser(url string) {
	cmd := exec.Command("cmd", "/c", "start", url)
	cmd.Run()
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, getHTML())
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	connections := getTCPConnections()
	
	// Store to database
	if err := storeConnections(connections); err != nil {
		fmt.Printf("Error storing connections: %v\n", err)
	}
	
	response := APIResponse{
		Success:     true,
		Message:     fmt.Sprintf("Found %d TCP connections", len(connections)),
		Connections: connections,
		Count:       len(connections),
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

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

func main() {
	port := 9999
	url := fmt.Sprintf("http://localhost:%d", port)

	fmt.Println("===============================================")
	fmt.Println("   TCP Routes Viewer - Desktop Application")
	fmt.Println("===============================================")
	fmt.Printf("Open: %s\n", url)

	// Open browser after short delay
	go func() {
		time.Sleep(1200 * time.Millisecond)
		fmt.Println("Opening application window...")
		openBrowser(url)
	}()

	// Setup routes
	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/api/connections", handleConnections)
	http.HandleFunc("/api/resolve", handleResolve)

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Server running on %s\n", url)
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println("===============================================\n")

	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

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
			font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
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
			white-space: nowrap;
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
		}
		
		.status {
			flex: 1;
			padding: 10px 15px;
			background: white;
			border-radius: 6px;
			font-size: 0.95em;
			min-width: 200px;
		}
		
		.table-wrapper {
			overflow-x: auto;
			max-height: 70vh;
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
		
		.state-established { color: #28a745; font-weight: 600; }
		.state-listening { color: #0066cc; font-weight: 600; }
		.state-time-wait { color: #ff9800; font-weight: 600; }
		.state-close-wait { color: #ff6b6b; font-weight: 600; }
		
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
			<p>Windows Desktop Application - Real-time TCP Connection Monitoring</p>
		</div>
		
		<div class="controls">
			<button class="btn" onclick="refreshConnections()">Refresh Routes</button>
			<button class="btn secondary" onclick="exportCSV()">Export CSV</button>
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
			<p>Last updated: <span id="lastUpdate">Never</span> | Total connections displayed: <span id="connCount">0</span></p>
		</div>
	</div>
	
	<script>
		let allConnections = [];
		
		document.getElementById("filterInput").addEventListener("input", filterConnections);
		
		async function refreshConnections() {
			const statusEl = document.getElementById("status");
			const btn = event ? event.target : null;
			
			statusEl.textContent = "Loading TCP routes...";
			if (btn) btn.disabled = true;
			
			try {
				const response = await fetch("/api/connections");
				const data = await response.json();
				
				if (data.success) {
					allConnections = data.connections || [];
					displayConnections(allConnections);
					statusEl.textContent = "✓ Found " + data.count + " TCP connections";
					document.getElementById("lastUpdate").textContent = new Date().toLocaleTimeString();
					document.getElementById("connCount").textContent = allConnections.length;
				} else {
					statusEl.textContent = "Error: " + data.message;
				}
			} catch (error) {
				statusEl.textContent = "Error: " + error;
			} finally {
				if (btn) btn.disabled = false;
			}
		}
		
		function displayConnections(connections) {
			const tbody = document.getElementById("connectionsBody");
			
			if (!connections || connections.length === 0) {
				tbody.innerHTML = "<tr><td colspan=\"5\" class=\"empty\">No TCP connections found</td></tr>";
				return;
			}
			
			tbody.innerHTML = connections.map(conn => {
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
			const filterText = document.getElementById("filterInput").value.toLowerCase();
			const filtered = allConnections.filter(conn => {
				const text = (conn.local_addr + conn.remote_addr + conn.state + conn.process_name + conn.pid).toLowerCase();
				return text.includes(filterText);
			});
			displayConnections(filtered);
			document.getElementById("connCount").textContent = filtered.length;
		}
		
		function exportCSV() {
			if (!allConnections || allConnections.length === 0) {
				alert("No connections to export. Refresh first!");
				return;
			}
			
			let csv = "Local Address,Remote Address,State,PID,Process Name\n";
			csv += allConnections.map(conn =>
				'"' + conn.local_addr + '","' + conn.remote_addr + '","' + conn.state + '","' + conn.pid + '","' + conn.process_name + '"'
			).join("\n");
			
			const blob = new Blob([csv], { type: "text/csv" });
			const url = window.URL.createObjectURL(blob);
			const a = document.createElement("a");
			a.href = url;
			a.download = "tcp-routes-" + new Date().toISOString().slice(0, 19) + ".csv";
			a.click();
			window.URL.revokeObjectURL(url);
		}
		
		// Auto-refresh on page load
		window.addEventListener("load", refreshConnections);
	</script>
</body>
</html>`
}
