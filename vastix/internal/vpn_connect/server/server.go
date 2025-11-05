// Package server provides VPN server functionality
package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"vastix/internal/vpn_connect/common"
)

// Server represents a VPN server instance
type Server struct {
	config  *common.ServerConfig
	logger  *slog.Logger
	process *exec.Cmd
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	configDir     string
	logFile       *os.File
	pidFile       string
	heartbeatFile string // Optional: path to heartbeat file for self-destruction

	mu      sync.RWMutex
	running bool
	clients map[string]*ClientSession // clientPublicKey -> session
}

// ClientSession represents a connected client
type ClientSession struct {
	PublicKey     string
	ClientIP      netip.Addr
	AllowedIPs    []netip.Prefix
	ConnectedAt   time.Time
	LastHandshake time.Time
	BytesSent     uint64
	BytesReceived uint64
}

// NewServer creates a new VPN server instance
func NewServer(config *common.ServerConfig, logger *slog.Logger) (*Server, error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}

	// Create configuration directory
	configDir := filepath.Join(os.TempDir(), fmt.Sprintf("vpn-server-%d", config.ListenPort))
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &Server{
		config:    config,
		logger:    logger,
		configDir: configDir,
		pidFile:   filepath.Join(configDir, "server.pid"),
		clients:   make(map[string]*ClientSession),
	}, nil
}

// SetHeartbeatFile sets the heartbeat file path for self-destruction monitoring
func (s *Server) SetHeartbeatFile(path string) {
	s.heartbeatFile = path
}

// Start starts the VPN server as a subprocess
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server is already running")
	}
	s.mu.Unlock()

	// Check if WireGuard is installed
	installed, err := CheckWireGuardInstalled()
	if err != nil {
		return fmt.Errorf("failed to check WireGuard installation: %w", err)
	}

	if !installed {
		s.logger.Warn("WireGuard not found, attempting to install...")
		if err := InstallWireGuard(); err != nil {
			return fmt.Errorf("failed to install WireGuard: %w", err)
		}
		s.logger.Info("WireGuard installed successfully")
	}

	// Get WireGuard command
	wgCmd, err := GetWireGuardCommand()
	if err != nil {
		return fmt.Errorf("failed to find WireGuard command: %w", err)
	}

	s.logger.Info("Using WireGuard", slog.String("command", wgCmd))

	// Create interface configuration
	// Use last 2 digits of port for interface name (e.g., port 51821 → wg21)
	ifaceName := fmt.Sprintf("wg%d", s.config.ListenPort%100)
	configPath := filepath.Join(s.configDir, "wg.conf")

	// Clean up any existing interface from previous crashed sessions
	s.logger.Info("Checking for existing interface", slog.String("interface", ifaceName))
	if err := s.deleteInterface(ifaceName); err != nil {
		s.logger.Debug("No existing interface to clean up (this is normal)", slog.String("interface", ifaceName))
	} else {
		s.logger.Info("Cleaned up existing interface from previous session", slog.String("interface", ifaceName))
	}

	if err := s.writeConfig(configPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Open log file
	logPath := filepath.Join(s.configDir, "server.log")
	s.logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	s.logger.Info("Starting VPN server",
		slog.String("interface", ifaceName),
		slog.Uint64("port", uint64(s.config.ListenPort)),
		slog.String("logFile", logPath))

	// Create context with cancellation
	ctxWithCancel, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	// Start wireguard-go as subprocess
	if strings.Contains(wgCmd, "wireguard-go") {
		s.process = exec.CommandContext(ctxWithCancel, wgCmd, "-f", ifaceName)
		s.process.Stdout = io.MultiWriter(s.logFile, logWriter{s.logger, slog.LevelInfo})
		s.process.Stderr = io.MultiWriter(s.logFile, logWriter{s.logger, slog.LevelError})

		if err := s.process.Start(); err != nil {
			s.logFile.Close()
			return fmt.Errorf("failed to start wireguard-go: %w", err)
		}

		// Wait for interface to be created
		time.Sleep(500 * time.Millisecond)

		// Configure the interface
		if err := s.configureInterface(ifaceName, configPath); err != nil {
			s.Stop()
			return fmt.Errorf("failed to configure interface: %w", err)
		}
	} else {
		// Using kernel module with wg-quick
		return fmt.Errorf("kernel module not yet supported, please use wireguard-go")
	}

	// Write PID file
	if err := os.WriteFile(s.pidFile, []byte(fmt.Sprintf("%d", s.process.Process.Pid)), 0600); err != nil {
		s.logger.Warn("Failed to write PID file", slog.Any("error", err))
	}

	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	// Monitor process in background
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.process.Wait(); err != nil {
			if ctxWithCancel.Err() == nil {
				s.logger.Error("VPN process exited unexpectedly", slog.Any("error", err))
			}
		}
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	// Monitor heartbeat for self-destruction (if heartbeat file is set)
	if s.heartbeatFile != "" {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.monitorHeartbeat(ctxWithCancel, s.heartbeatFile)
		}()
	}

	s.logger.Info("VPN server started successfully",
		slog.Int("pid", s.process.Process.Pid),
		slog.String("serverIP", s.config.ServerIP.String()))

	return nil
}

// Stop stops the VPN server and cleans up the WireGuard interface
func (s *Server) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return fmt.Errorf("server is not running")
	}
	s.mu.Unlock()

	s.logger.Info("Stopping VPN server...")

	// Delete WireGuard interface FIRST (before killing process)
	// Use last 2 digits of port for interface name (e.g., port 51821 → wg21)
	ifaceName := fmt.Sprintf("wg%d", s.config.ListenPort%100)
	s.logger.Info("Deleting WireGuard interface", slog.String("interface", ifaceName))
	if err := s.deleteInterface(ifaceName); err != nil {
		s.logger.Warn("Failed to delete interface", slog.Any("error", err))
		// Continue anyway - try to clean up the process
	} else {
		s.logger.Info("WireGuard interface deleted successfully")
	}

	// Cancel context
	if s.cancel != nil {
		s.cancel()
	}

	// Kill process if still running
	if s.process != nil && s.process.Process != nil {
		if err := s.process.Process.Kill(); err != nil {
			s.logger.Warn("Failed to kill process", slog.Any("error", err))
		}
	}

	// Wait for process to exit
	s.wg.Wait()

	// Close log file
	if s.logFile != nil {
		s.logFile.Close()
	}

	// Remove PID file
	os.Remove(s.pidFile)

	s.mu.Lock()
	s.running = false
	s.mu.Unlock()

	s.logger.Info("VPN server stopped")
	return nil
}

// AddClient adds a new client to the server
func (s *Server) AddClient(publicKey string, clientIP netip.Addr, allowedIPs []netip.Prefix) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.clients[publicKey]; exists {
		return fmt.Errorf("client already exists")
	}

	session := &ClientSession{
		PublicKey:   publicKey,
		ClientIP:    clientIP,
		AllowedIPs:  allowedIPs,
		ConnectedAt: time.Now(),
	}

	s.clients[publicKey] = session

	// Apply configuration to running interface
	if s.running {
		return s.applyClientConfig(publicKey, clientIP, allowedIPs)
	}

	return nil
}

// RemoveClient removes a client from the server
func (s *Server) RemoveClient(publicKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.clients[publicKey]; !exists {
		return fmt.Errorf("client not found")
	}

	delete(s.clients, publicKey)

	// Remove from running interface
	if s.running {
		return s.removeClientConfig(publicKey)
	}

	return nil
}

// GetStatus returns the server status
func (s *Server) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := map[string]interface{}{
		"running":     s.running,
		"port":        s.config.ListenPort,
		"serverIP":    s.config.ServerIP.String(),
		"clientCount": len(s.clients),
		"configDir":   s.configDir,
	}

	if s.process != nil && s.process.Process != nil {
		status["pid"] = s.process.Process.Pid
	}

	return status
}

// writeConfig writes the WireGuard configuration file
func (s *Server) writeConfig(path string) error {
	// For `wg setconf`, we don't include [Interface] section
	// Only PrivateKey and ListenPort (without [Interface] header)
	// Address is set separately via `ip address add`
	config := fmt.Sprintf(`[Interface]
PrivateKey = %s
ListenPort = %d
`,
		s.config.PrivateKey,
		s.config.ListenPort)

	// Add existing clients
	for publicKey, session := range s.clients {
		allowedIPsStr := make([]string, len(session.AllowedIPs))
		for i, ip := range session.AllowedIPs {
			allowedIPsStr[i] = ip.String()
		}

		config += fmt.Sprintf(`
[Peer]
PublicKey = %s
AllowedIPs = %s
`,
			publicKey,
			strings.Join(allowedIPsStr, ", "))
	}

	return os.WriteFile(path, []byte(config), 0600)
}

// configureInterface configures the WireGuard interface
func (s *Server) configureInterface(ifaceName, configPath string) error {
	// Set interface configuration
	cmd := exec.Command("wg", "setconf", ifaceName, configPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set interface config: %w\nOutput: %s", err, string(output))
	}

	// Bring interface up
	cmd = exec.Command("ip", "link", "set", "up", "dev", ifaceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring interface up: %w\nOutput: %s", err, string(output))
	}

	// Add IP address
	cmd = exec.Command("ip", "address", "add", "dev", ifaceName, s.config.ServerIP.String()+"/24")
	if output, err := cmd.CombinedOutput(); err != nil {
		// Ignore error if address already exists
		if !strings.Contains(string(output), "File exists") {
			return fmt.Errorf("failed to add IP address: %w\nOutput: %s", err, string(output))
		}
	}

	// Enable IP forwarding
	cmd = exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		s.logger.Warn("Failed to enable IP forwarding", slog.String("output", string(output)))
	}

	// Set up NAT if interface is specified
	if s.config.Interface != "" {
		// Add iptables rules for forwarding
		rules := [][]string{
			{"iptables", "-A", "FORWARD", "-i", ifaceName, "-j", "ACCEPT"},
			{"iptables", "-A", "FORWARD", "-o", ifaceName, "-j", "ACCEPT"},
			{"iptables", "-t", "nat", "-A", "POSTROUTING", "-s", s.config.VPNNetwork.String(), "-o", s.config.Interface, "-j", "MASQUERADE"},
		}

		for _, rule := range rules {
			cmd := exec.Command(rule[0], rule[1:]...)
			if output, err := cmd.CombinedOutput(); err != nil {
				// Ignore if rule already exists
				if !strings.Contains(string(output), "File exists") && !strings.Contains(string(output), "already") {
					s.logger.Warn("Failed to add iptables rule",
						slog.String("rule", strings.Join(rule, " ")),
						slog.String("output", string(output)))
				}
			}
		}
	}

	return nil
}

// deleteInterface deletes the WireGuard interface and cleans up iptables rules
func (s *Server) deleteInterface(ifaceName string) error {
	s.logger.Info("Cleaning up WireGuard interface", slog.String("interface", ifaceName))

	// Remove ALL iptables rules if interface was specified (loop until none remain)
	if s.config.Interface != "" {
		rules := [][]string{
			{"iptables", "-D", "FORWARD", "-i", ifaceName, "-j", "ACCEPT"},
			{"iptables", "-D", "FORWARD", "-o", ifaceName, "-j", "ACCEPT"},
			{"iptables", "-t", "nat", "-D", "POSTROUTING", "-s", s.config.VPNNetwork.String(), "-o", s.config.Interface, "-j", "MASQUERADE"},
		}

		for _, rule := range rules {
			// Keep deleting until the rule no longer exists (handles duplicates)
			deletedCount := 0
			for {
				cmd := exec.Command(rule[0], rule[1:]...)
				if _, err := cmd.CombinedOutput(); err != nil {
					// Rule doesn't exist anymore, move to next rule
					if deletedCount > 0 {
						s.logger.Info("Deleted duplicate iptables rules",
							slog.String("rule", strings.Join(rule, " ")),
							slog.Int("count", deletedCount))
					} else {
						s.logger.Debug("iptables rule not found (already deleted)",
							slog.String("rule", strings.Join(rule, " ")))
					}
					break
				}
				deletedCount++
				// Successfully deleted one instance, check for more
			}
		}
	}

	// Delete the interface
	cmd := exec.Command("ip", "link", "delete", ifaceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Check if interface doesn't exist - that's ok
		if strings.Contains(string(output), "Cannot find device") || strings.Contains(string(output), "does not exist") {
			s.logger.Debug("Interface already deleted", slog.String("interface", ifaceName))
			return nil
		}
		return fmt.Errorf("failed to delete interface: %w\nOutput: %s", err, string(output))
	}

	s.logger.Info("Interface deleted successfully", slog.String("interface", ifaceName))
	return nil
}

// applyClientConfig applies a new client configuration to the running interface
func (s *Server) applyClientConfig(publicKey string, clientIP netip.Addr, allowedIPs []netip.Prefix) error {
	// Use last 2 digits of port for interface name (e.g., port 51821 → wg21)
	ifaceName := fmt.Sprintf("wg%d", s.config.ListenPort%100)

	allowedIPsStr := make([]string, len(allowedIPs))
	for i, ip := range allowedIPs {
		allowedIPsStr[i] = ip.String()
	}

	cmd := exec.Command("wg", "set", ifaceName,
		"peer", publicKey,
		"allowed-ips", strings.Join(allowedIPsStr, ","))

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add client: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// removeClientConfig removes a client configuration from the running interface
func (s *Server) removeClientConfig(publicKey string) error {
	// Use last 2 digits of port for interface name (e.g., port 51821 → wg21)
	ifaceName := fmt.Sprintf("wg%d", s.config.ListenPort%100)

	cmd := exec.Command("wg", "set", ifaceName, "peer", publicKey, "remove")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove client: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// logWriter wraps slog.Logger to implement io.Writer
type logWriter struct {
	logger *slog.Logger
	level  slog.Level
}

func (w logWriter) Write(p []byte) (n int, err error) {
	w.logger.Log(context.Background(), w.level, string(p))
	return len(p), nil
}

// monitorHeartbeat monitors the heartbeat file and self-destructs if no heartbeat is received
func (s *Server) monitorHeartbeat(ctx context.Context, heartbeatFile string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	const heartbeatTimeout = 12 * time.Second

	s.logger.Info("Heartbeat monitoring started",
		slog.String("timeout", heartbeatTimeout.String()),
		slog.String("file", heartbeatFile))

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Heartbeat monitoring stopped (context canceled)")
			return
		case <-ticker.C:
			// Check heartbeat file
			data, err := os.ReadFile(heartbeatFile)
			if err != nil {
				// Heartbeat file doesn't exist yet - wait for first heartbeat
				if os.IsNotExist(err) {
					s.logger.Debug("Waiting for initial heartbeat...")
					continue
				}
				s.logger.Warn("Failed to read heartbeat file", slog.String("error", err.Error()))
				continue
			}

			// Parse timestamp
			timestampStr := strings.TrimSpace(string(data))
			timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
			if err != nil {
				s.logger.Warn("Failed to parse heartbeat timestamp", slog.String("error", err.Error()))
				continue
			}

			// Check if heartbeat is stale
			lastHeartbeat := time.Unix(timestamp, 0)
			timeSinceHeartbeat := time.Since(lastHeartbeat)

			if timeSinceHeartbeat > heartbeatTimeout {
				s.logger.Error("⚠️  HEARTBEAT TIMEOUT - CLIENT CONNECTION LOST",
					slog.Duration("timeSinceLastHeartbeat", timeSinceHeartbeat),
					slog.Duration("timeout", heartbeatTimeout),
					slog.Time("lastHeartbeat", lastHeartbeat),
					slog.String("heartbeatFile", heartbeatFile))

				s.logger.Warn("Initiating server self-destruction due to heartbeat timeout")

				// Self-destruct: stop the server
				s.logger.Info("Stopping VPN server (self-destruction triggered by heartbeat timeout)")
				if err := s.Stop(); err != nil {
					s.logger.Error("Failed to stop server during self-destruction", slog.String("error", err.Error()))
				} else {
					s.logger.Info("Server stopped successfully during self-destruction")
				}

				s.logger.Info("✓ Server self-destructed due to heartbeat timeout - cleanup complete")
				s.logger.Info("Terminating server process now...")

				// Exit the process - this is critical for self-destruction
				// Without this, the process sits idle even though all resources are cleaned up
				os.Exit(0)
			}

			s.logger.Debug("Heartbeat OK", slog.Duration("age", timeSinceHeartbeat))
		}
	}
}

// GetClients returns a list of connected clients
func (s *Server) GetClients() []*ClientSession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clients := make([]*ClientSession, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, client)
	}

	return clients
}

// IsRunning returns whether the server is running
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetConfigDir returns the configuration directory
func (s *Server) GetConfigDir() string {
	return s.configDir
}

// ReadLogs returns the last N lines of the server log
func (s *Server) ReadLogs(lines int) ([]string, error) {
	logPath := filepath.Join(s.configDir, "server.log")

	file, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Read file backwards to get last N lines efficiently
	var result []string
	scanner := bufio.NewScanner(file)

	// Read all lines (for simplicity, can optimize for large files)
	var allLines []string
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	// Get last N lines
	start := len(allLines) - lines
	if start < 0 {
		start = 0
	}
	result = allLines[start:]

	return result, nil
}

// DetectExternalInterface attempts to detect the external network interface
func DetectExternalInterface() (string, error) {
	cmd := exec.Command("ip", "route", "show", "default")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get default route: %w", err)
	}

	// Parse output like: "default via 10.27.14.1 dev eno1 proto static metric 100"
	fields := strings.Fields(string(output))
	for i, field := range fields {
		if field == "dev" && i+1 < len(fields) {
			return fields[i+1], nil
		}
	}

	return "", fmt.Errorf("could not detect external interface")
}

// GetNextAvailablePort finds the next available port starting from basePort
func GetNextAvailablePort(basePort uint16) (uint16, error) {
	for port := basePort; port < basePort+100; port++ {
		// Check if port is in use
		cmd := exec.Command("ss", "-tuln")
		output, err := cmd.Output()
		if err != nil {
			return 0, fmt.Errorf("failed to check ports: %w", err)
		}

		portStr := strconv.Itoa(int(port))
		if !strings.Contains(string(output), ":"+portStr) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports found in range %d-%d", basePort, basePort+100)
}
