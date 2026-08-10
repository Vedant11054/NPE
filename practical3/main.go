package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"practical3/scanner"
)

//go:embed frontend/*
var frontendFS embed.FS

var activeScanner = scanner.NewPortScanner()

// JSON Response Helpers
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func handleInterfaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get interfaces: "+err.Error())
		return
	}

	type InterfaceInfo struct {
		Name         string   `json:"name"`
		HardwareAddr string   `json:"hardware_addr"`
		IPs          []string `json:"ips"`
	}

	var result []InterfaceInfo
	for _, iface := range interfaces {
		// Filter out down or loopback interfaces
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		var ips []string
		for _, addr := range addrs {
			ips = append(ips, addr.String())
		}

		// Keep interface if it has assigned IPs
		if len(ips) > 0 {
			result = append(result, InterfaceInfo{
				Name:         iface.Name,
				HardwareAddr: iface.HardwareAddr.String(),
				IPs:          ips,
			})
		}
	}

	respondWithJSON(w, http.StatusOK, result)
}

func handleScanStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Target  string `json:"target"`
		Ports   string `json:"ports"`
		Workers int    `json:"workers"`
		Timeout int    `json:"timeout"` // in milliseconds
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate inputs
	if req.Target == "" {
		respondWithError(w, http.StatusBadRequest, "Target is required")
		return
	}
	if req.Ports == "" {
		respondWithError(w, http.StatusBadRequest, "Ports list is required")
		return
	}

	// Start scanning
	err = activeScanner.Start(req.Target, req.Ports, req.Workers, req.Timeout)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get starting status (like total ports scanned size)
	status := activeScanner.GetStatus()

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "started",
		"total_ports": status.TotalPorts,
	})
}

func handleScanStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	activeScanner.Stop()
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func handleScanStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	status := activeScanner.GetStatus()
	respondWithJSON(w, http.StatusOK, status)
}

// openBrowser launches the local web server link in default web browser
func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "windows":
		err = exec.Command("cmd", "/c", "start", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default: // Linux / BSD
		err = exec.Command("xdg-open", url).Start()
	}
	if err != nil {
		log.Printf("Could not open browser automatically: %v (Please visit %s manually)", err, url)
	}
}

func main() {
	// Configure embedded filesystem prefix strip
	subFS, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		log.Fatalf("Error reading embedded filesystem: %v", err)
	}

	// Routers
	http.Handle("/", http.FileServer(http.FS(subFS)))
	http.HandleFunc("/api/interfaces", handleInterfaces)
	http.HandleFunc("/api/scan/start", handleScanStart)
	http.HandleFunc("/api/scan/stop", handleScanStop)
	http.HandleFunc("/api/scan/status", handleScanStatus)

	// Pick an available port, or default to 8085
	port := 8085
	serverAddr := fmt.Sprintf("127.0.0.1:%d", port)
	
	// Create listener to verify port availability
	listener, err := net.Listen("tcp", serverAddr)
	if err != nil {
		// If port 8085 is in use, listen on a random available port
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Fatalf("Failed to bind server to any interface: %v", err)
		}
		serverAddr = listener.Addr().String()
	} else {
		listener.Close()
	}

	serverUrl := fmt.Sprintf("http://%s", serverAddr)
	fmt.Printf("\n=======================================================\n")
	fmt.Printf("   Antigravity Concurrent Network Port Scanner   \n")
	fmt.Printf("=======================================================\n")
	fmt.Printf("Server listening on: %s\n", serverUrl)
	fmt.Printf("Opening default web browser...\n")
	fmt.Printf("Press Ctrl+C to terminate application\n")
	fmt.Printf("=======================================================\n\n")

	// Start browser in a brief delay to allow server startup completion
	go func() {
		time.Sleep(500 * time.Millisecond)
		openBrowser(serverUrl)
	}()

	err = http.ListenAndServe(serverAddr, nil)
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
