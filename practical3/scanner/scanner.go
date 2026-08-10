package scanner

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Common port services mapping
var CommonServices = map[int]string{
	20:    "FTP Data",
	21:    "FTP Control",
	22:    "SSH",
	23:    "Telnet",
	25:    "SMTP",
	53:    "DNS",
	67:    "DHCP Server",
	68:    "DHCP Client",
	69:    "TFTP",
	80:    "HTTP",
	110:   "POP3",
	115:   "SFTP",
	123:   "NTP",
	135:   "RPC Endpoint Mapper",
	137:   "NetBIOS Name Service",
	138:   "NetBIOS Datagram",
	139:   "NetBIOS Session",
	143:   "IMAP",
	161:   "SNMP",
	179:   "BGP",
	389:   "LDAP",
	443:   "HTTPS",
	445:   "SMB",
	465:   "SMTPS",
	514:   "Syslog",
	587:   "SMTP Submission",
	636:   "LDAPS",
	993:   "IMAPS",
	995:   "POP3S",
	1433:  "Microsoft SQL Server",
	1521:  "Oracle DB",
	2049:  "NFS",
	3306:  "MySQL",
	3389:  "Remote Desktop (RDP)",
	5432:  "PostgreSQL",
	5672:  "RabbitMQ",
	5900:  "VNC Server",
	6379:  "Redis",
	8080:  "HTTP Alternate (Proxy/Tomcat)",
	8443:  "HTTPS Alternate",
	9000:  "Portainer / PHP-FPM",
	9200:  "Elasticsearch",
	27017: "MongoDB",
}

// ScanResult represents the result of scanning a single port
type ScanResult struct {
	IP        string `json:"ip"`
	Port      int    `json:"port"`
	State     string `json:"state"` // "open", "closed", "filtered"
	Service   string `json:"service"`
	Banner    string `json:"banner,omitempty"`
	Timestamp string `json:"timestamp"`
}

// LiveScannedPort represents a temporary status of a port being scanned
type LiveScannedPort struct {
	IP    string `json:"ip"`
	Port  int    `json:"port"`
	State string `json:"state"`
}

// ScanStatus represents the current progress of a scanning task
type ScanStatus struct {
	Target          string            `json:"target"`
	Ports           string            `json:"ports"`
	TotalPorts      int               `json:"total_ports"`
	ScannedPorts    int               `json:"scanned_ports"`
	Progress        float64           `json:"progress"`
	OpenPorts       []ScanResult      `json:"open_ports"`
	LiveScanned     []LiveScannedPort `json:"live_scanned"`
	IsRunning       bool              `json:"is_running"`
	ScanTimeElapsed string            `json:"scan_time_elapsed"`
	ScanSpeed       float64           `json:"scan_speed"` // ports/second
	ETA             string            `json:"eta"`
}

// PortScanner manages an active or completed scan job
type PortScanner struct {
	mu           sync.RWMutex
	target       string
	portsStr     string
	workers      int
	timeout      time.Duration
	cancelFunc   context.CancelFunc
	ctx          context.Context
	isRunning    bool
	startTime    time.Time
	endTime      time.Time
	totalPorts   int
	scannedPorts int
	openPorts    []ScanResult
	liveScanned  []LiveScannedPort
}

// NewPortScanner creates a new scanner instance
func NewPortScanner() *PortScanner {
	return &PortScanner{
		openPorts:   make([]ScanResult, 0),
		liveScanned: make([]LiveScannedPort, 0),
	}
}

// Start initiates the port scan in a background goroutine
func (ps *PortScanner) Start(targetStr string, portsStr string, workerCount int, timeoutMs int) error {
	ps.mu.Lock()
	if ps.isRunning {
		ps.mu.Unlock()
		return fmt.Errorf("a scan is already running")
	}

	// Parse IPs
	ips, err := ParseTargets(targetStr)
	if err != nil {
		ps.mu.Unlock()
		return fmt.Errorf("failed to parse targets: %w", err)
	}
	if len(ips) == 0 {
		ps.mu.Unlock()
		return fmt.Errorf("no valid target IP addresses found")
	}

	// Parse Ports
	ports, err := ParsePorts(portsStr)
	if err != nil {
		ps.mu.Unlock()
		return fmt.Errorf("failed to parse ports: %w", err)
	}
	if len(ports) == 0 {
		ps.mu.Unlock()
		return fmt.Errorf("no valid ports specified to scan")
	}

	// Setup Scan configuration
	ps.target = targetStr
	ps.portsStr = portsStr
	ps.workers = workerCount
	if ps.workers <= 0 {
		ps.workers = 100 // default
	}
	ps.timeout = time.Duration(timeoutMs) * time.Millisecond
	if ps.timeout <= 0 {
		ps.timeout = 1000 * time.Millisecond // default 1s
	}

	ps.totalPorts = len(ips) * len(ports)
	ps.scannedPorts = 0
	ps.openPorts = make([]ScanResult, 0)
	ps.liveScanned = make([]LiveScannedPort, 0)
	ps.isRunning = true
	ps.startTime = time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	ps.ctx = ctx
	ps.cancelFunc = cancel
	ps.mu.Unlock()

	// Launch scan orchestration in background
	go ps.runScan(ips, ports)

	return nil
}

// Stop cancels the running scan task
func (ps *PortScanner) Stop() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.isRunning && ps.cancelFunc != nil {
		ps.cancelFunc()
		ps.isRunning = false
		ps.endTime = time.Now()
	}
}

// GetStatus returns a snapshot of the current scan progress and newly scanned ports (clearing live buffer)
func (ps *PortScanner) GetStatus() ScanStatus {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	elapsed := time.Duration(0)
	if ps.isRunning {
		elapsed = time.Since(ps.startTime)
	} else if !ps.startTime.IsZero() && !ps.endTime.IsZero() {
		elapsed = ps.endTime.Sub(ps.startTime)
	}

	progress := 0.0
	if ps.totalPorts > 0 {
		progress = float64(ps.scannedPorts) / float64(ps.totalPorts) * 100.0
	}

	speed := 0.0
	if elapsed.Seconds() > 0 {
		speed = float64(ps.scannedPorts) / elapsed.Seconds()
	}

	etaStr := "Finished"
	if ps.isRunning {
		if speed > 0 && ps.scannedPorts < ps.totalPorts {
			remaining := ps.totalPorts - ps.scannedPorts
			remDuration := time.Duration(float64(remaining)/speed) * time.Second
			etaStr = remDuration.Round(time.Second).String()
		} else {
			etaStr = "Calculating..."
		}
	} else if ps.scannedPorts < ps.totalPorts && ps.totalPorts > 0 {
		etaStr = "Stopped"
	}

	// Copy and clear live scanned ports buffer
	live := make([]LiveScannedPort, len(ps.liveScanned))
	copy(live, ps.liveScanned)
	ps.liveScanned = ps.liveScanned[:0] // Reset capacity, clear contents

	return ScanStatus{
		Target:          ps.target,
		Ports:           ps.portsStr,
		TotalPorts:      ps.totalPorts,
		ScannedPorts:    ps.scannedPorts,
		Progress:        progress,
		OpenPorts:       ps.openPorts,
		LiveScanned:     live,
		IsRunning:       ps.isRunning,
		ScanTimeElapsed: elapsed.Round(time.Second).String(),
		ScanSpeed:       speed,
		ETA:             etaStr,
	}
}

// runScan orchestrates the worker pool and maps tasks
func (ps *PortScanner) runScan(ips []net.IP, ports []int) {
	numWorkers := ps.workers
	if numWorkers > ps.totalPorts {
		numWorkers = ps.totalPorts
	}

	portsChan := make(chan int, len(ports))
	resultsChan := make(chan ScanResult, numWorkers)
	var wg sync.WaitGroup

	// Start workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go ps.worker(ips, portsChan, resultsChan, &wg)
	}

	// Feed ports to channel
	go func() {
		for _, port := range ports {
			select {
			case <-ps.ctx.Done():
				close(portsChan)
				return
			case portsChan <- port:
			}
		}
		close(portsChan)
	}()

	// Wait for workers to finish in a separate routine and close results
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for res := range resultsChan {
		ps.mu.Lock()
		ps.scannedPorts++
		if res.State == "open" {
			ps.openPorts = append(ps.openPorts, res)
		}
		// Append to liveScanned list (keep max 1000 items in buffer before GUI retrieves it)
		if len(ps.liveScanned) < 1000 {
			ps.liveScanned = append(ps.liveScanned, LiveScannedPort{
				IP:    res.IP,
				Port:  res.Port,
				State: res.State,
			})
		}
		ps.mu.Unlock()
	}

	ps.mu.Lock()
	ps.isRunning = false
	ps.endTime = time.Now()
	ps.mu.Unlock()
}

// worker processes incoming ports and scans all IPs for each port
func (ps *PortScanner) worker(ips []net.IP, ports <-chan int, results chan<- ScanResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ps.ctx.Done():
			return
		case port, ok := <-ports:
			if !ok {
				return
			}
			for _, ip := range ips {
				select {
				case <-ps.ctx.Done():
					return
				default:
					res := ps.scanIPPort(ip, port)
					results <- res
				}
			}
		}
	}
}

// scanIPPort performs TCP connection check and banner grabbing
func (ps *PortScanner) scanIPPort(ip net.IP, port int) ScanResult {
	ipStr := ip.String()
	address := net.JoinHostPort(ipStr, strconv.Itoa(port))
	dialer := net.Dialer{Timeout: ps.timeout}

	conn, err := dialer.DialContext(ps.ctx, "tcp", address)

	result := ScanResult{
		IP:        ipStr,
		Port:      port,
		State:     "closed",
		Timestamp: time.Now().Format("15:04:05"),
	}

	if err != nil {
		// Distinguish between timeout (filtered) and connection refused (closed)
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			result.State = "filtered"
		}
		return result
	}
	defer conn.Close()

	// Port is open!
	result.State = "open"
	if svc, exists := CommonServices[port]; exists {
		result.Service = svc
	} else {
		result.Service = "Unknown"
	}

	// Attempt lightweight banner grabbing
	result.Banner = grabBanner(conn, port)

	return result
}

// grabBanner reads the introductory header bytes from the connection
func grabBanner(conn net.Conn, port int) string {
	conn.SetDeadline(time.Now().Add(1 * time.Second))
	defer conn.SetDeadline(time.Time{}) // Clear deadline

	// Send trigger for HTTP(S) protocol ports
	if port == 80 || port == 8080 {
		conn.Write([]byte("HEAD / HTTP/1.0\r\n\r\n"))
	} else if port == 443 || port == 8443 {
		// HTTPS uses TLS, raw write on TCP socket might fail or timeout,
		// but sometimes plain HTTP HEAD works or returns handshake error.
		conn.Write([]byte("HEAD / HTTP/1.0\r\n\r\n"))
	} else {
		// Just send a newline to trigger SSH/FTP/SMTP banners that greet on connect
		conn.Write([]byte("\r\n"))
	}

	buf := make([]byte, 128)
	n, err := conn.Read(buf)
	if err != nil || n == 0 {
		return ""
	}

	banner := string(buf[:n])
	var clean strings.Builder
	for _, r := range banner {
		if r >= 32 && r < 127 {
			clean.WriteRune(r)
		} else if r == '\n' || r == '\r' {
			clean.WriteRune(' ')
		}
	}

	res := strings.TrimSpace(clean.String())
	// Clean double spaces
	for strings.Contains(res, "  ") {
		res = strings.ReplaceAll(res, "  ", " ")
	}

	if len(res) > 80 {
		res = res[:77] + "..."
	}
	return res
}

// ParseTargets parses target input string supporting: IP, host, IP ranges, CIDR subnets
func ParseTargets(input string) ([]net.IP, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("target string is empty")
	}

	// CIDR Notation
	if strings.Contains(input, "/") {
		return expandCIDR(input)
	}

	// IP Range: e.g. 192.168.1.1-192.168.1.50 or 192.168.1.1-50
	if strings.Contains(input, "-") {
		parts := strings.Split(input, "-")
		if len(parts) == 2 {
			return expandIPRange(parts[0], parts[1])
		}
	}

	// Single Host or IP
	// Try parsing as IP first
	if ip := net.ParseIP(input); ip != nil {
		return []net.IP{ip}, nil
	}

	// Try resolving host
	ips, err := net.LookupIP(input)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve host '%s': %w", input, err)
	}

	var parsedIPs []net.IP
	for _, ip := range ips {
		// Focus on IPv4 for simple local scanning
		if ipv4 := ip.To4(); ipv4 != nil {
			parsedIPs = append(parsedIPs, ipv4)
		}
	}

	if len(parsedIPs) == 0 {
		return nil, fmt.Errorf("no IPv4 addresses resolved for host '%s'", input)
	}

	return parsedIPs, nil
}

// expandCIDR expands a CIDR range (e.g. 192.168.1.0/24) into a slice of IPs
func expandCIDR(cidr string) ([]net.IP, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []net.IP
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
		temp := make(net.IP, len(ip))
		copy(temp, ip)
		ips = append(ips, temp)
	}

	// For subnets larger than /31, remove network address (.0) and broadcast address (.255)
	ones, bits := ipnet.Mask.Size()
	if bits-ones > 1 && len(ips) > 2 {
		return ips[1 : len(ips)-1], nil
	}

	return ips, nil
}

// expandIPRange expands a range like 192.168.1.1-192.168.1.100 or 192.168.1.1-100
func expandIPRange(startStr, endStr string) ([]net.IP, error) {
	startStr = strings.TrimSpace(startStr)
	endStr = strings.TrimSpace(endStr)

	startIP := net.ParseIP(startStr)
	if startIP == nil {
		return nil, fmt.Errorf("invalid start IP: %s", startStr)
	}

	endIP := net.ParseIP(endStr)
	if endIP == nil {
		// Check if endStr is just a suffix (e.g. last octet "50" in "192.168.1.1-50")
		lastDot := strings.LastIndex(startStr, ".")
		if lastDot == -1 {
			return nil, fmt.Errorf("invalid IP format: %s", startStr)
		}
		prefix := startStr[:lastDot+1]
		endIP = net.ParseIP(prefix + endStr)
		if endIP == nil {
			return nil, fmt.Errorf("invalid end range: %s", endStr)
		}
	}

	// Ensure IPv4 conversion
	startIP4 := startIP.To4()
	endIP4 := endIP.To4()
	if startIP4 == nil || endIP4 == nil {
		return nil, fmt.Errorf("only IPv4 ranges are supported")
	}

	var ips []net.IP
	curr := make(net.IP, len(startIP4))
	copy(curr, startIP4)

	for {
		temp := make(net.IP, len(curr))
		copy(temp, curr)
		ips = append(ips, temp)

		if curr.Equal(endIP4) {
			break
		}

		incrementIP(curr)

		// Safety check to prevent infinite loop or wrapping if start > end
		if bytes.Compare(curr, endIP4) > 0 {
			break
		}
	}

	return ips, nil
}

// incrementIP increments the IP address byte by byte
func incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] > 0 {
			break
		}
	}
}

// ParsePorts parses port string formats like: "80", "80,443", "1-1000", "22,80-100,8080"
func ParsePorts(input string) ([]int, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("port list is empty")
	}

	portMap := make(map[int]bool)
	tokens := strings.Split(input, ",")

	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}

		if strings.Contains(token, "-") {
			// Range of ports
			parts := strings.Split(token, "-")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid port range: %s", token)
			}

			startPort, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s in range", parts[0])
			}

			endPort, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s in range", parts[1])
			}

			if startPort > endPort {
				startPort, endPort = endPort, startPort
			}

			if startPort < 1 || endPort > 65535 {
				return nil, fmt.Errorf("ports must be in range 1-65535")
			}

			for p := startPort; p <= endPort; p++ {
				portMap[p] = true
			}
		} else {
			// Single port
			port, err := strconv.Atoi(token)
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s", token)
			}

			if port < 1 || port > 65535 {
				return nil, fmt.Errorf("port must be in range 1-65535: %d", port)
			}

			portMap[port] = true
		}
	}

	// Convert map back to sorted/slice
	var ports []int
	for p := range portMap {
		ports = append(ports, p)
	}

	return ports, nil
}
