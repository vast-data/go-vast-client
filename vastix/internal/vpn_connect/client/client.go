// Package client provides VPN client functionality
package client

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"vastix/internal/vpn_connect/common"
)

// Client represents a VPN client instance
type Client struct {
	config  *common.ClientConfig
	writer  io.Writer   // Writer for streaming logs to UI (includes multiwriter with auxlog)
	network interface{} // Will hold noisysockets.Network

	mu        sync.RWMutex
	connected bool
	stats     ConnectionStats
}

// ConnectionStats holds connection statistics
type ConnectionStats struct {
	BytesSent       uint64
	BytesReceived   uint64
	LastHandshake   time.Time
	ConnectedAt     time.Time
	PacketsSent     uint64
	PacketsReceived uint64
}

// CheckWireGuardInstalled checks if WireGuard tools are installed
func CheckWireGuardInstalled() error {
	// Check for wg-quick
	if _, err := exec.LookPath("wg-quick"); err != nil {
		return fmt.Errorf("wg-quick not found. Please install WireGuard:\n" +
			"  Ubuntu/Debian: sudo apt install wireguard-tools\n" +
			"  RHEL/CentOS:   sudo yum install wireguard-tools\n" +
			"  Arch:          sudo pacman -S wireguard-tools")
	}

	// Check for wg
	if _, err := exec.LookPath("wg"); err != nil {
		return fmt.Errorf("wg command not found. Please install WireGuard tools")
	}

	return nil
}

// NewClient creates a new VPN client instance
func NewClient(config *common.ClientConfig, writer io.Writer) (*Client, error) {
	// Writer must never be nil - this is a programming error
	if writer == nil {
		panic("BUG: writer cannot be nil in NewClient - this indicates improper initialization")
	}

	// Check if WireGuard is installed
	if err := CheckWireGuardInstalled(); err != nil {
		return nil, err
	}

	return &Client{
		config: config,
		writer: writer,
	}, nil
}

// Connect establishes the VPN connection using WireGuard
// sudoPassword MUST be provided if sudo requires a password (check with CheckSudoNeedsPassword)
func (c *Client) Connect(sudoPassword string) error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return fmt.Errorf("client is already connected")
	}
	c.mu.Unlock()

	c.writef("Connecting to VPN server %s (client IP: %s)\n", c.config.ServerEndpoint, c.config.ClientIP.String())

	// Build AllowedIPs list
	// Start with server IP only (no broad /16 route to avoid conflicts with other VPNs)
	allowedIPs := []string{c.config.ServerIP.String() + "/32"} // Allow server IP

	// Add private IPs - each as /32 (single host route)
	if len(c.config.PrivateIPs) > 0 {
		for _, ip := range c.config.PrivateIPs {
			allowedIPs = append(allowedIPs, ip.String()+"/32")
		}
		c.writef("Routing %d VIP pool IPs through VPN (specific /32 routes)\n", len(c.config.PrivateIPs))
	}

	// Join all AllowedIPs with commas
	allowedIPsStr := strings.Join(allowedIPs, ", ")

	// Create WireGuard configuration
	wgConfig := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s/24

[Peer]
PublicKey = %s
Endpoint = %s
AllowedIPs = %s
PersistentKeepalive = 25
`,
		c.config.PrivateKey,
		c.config.ClientIP,
		c.config.ServerPublicKey,
		c.config.ServerEndpoint,
		allowedIPsStr)

	// Get local hostname for directory structure
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get local hostname: %w", err)
	}

	// Create local work directory: /tmp/vastix/<hostname>/
	localWorkDir := fmt.Sprintf("/tmp/vastix/%s", hostname)
	if err := os.MkdirAll(localWorkDir, 0755); err != nil {
		return fmt.Errorf("failed to create local work directory: %w", err)
	}
	c.writef("Local work directory: %s\n", localWorkDir)

	// Write config to file in local work directory
	// Use simple interface name to avoid 15-character Linux interface name limit
	// Format: wgvastix.conf (8 chars interface name)
	configPath := fmt.Sprintf("%s/wgvastix.conf", localWorkDir)
	if err := os.WriteFile(configPath, []byte(wgConfig), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Clean up any existing wgvastix interface to avoid "File exists" errors
	cleanupCmd := exec.Command("sudo", "-S", "ip", "link", "delete", "wgvastix")
	cleanupCmd.Stdin = strings.NewReader(sudoPassword + "\n")
	// Ignore errors - interface might not exist
	cleanupCmd.Run()

	c.writef("Bringing up WireGuard interface: sudo wg-quick up %s\n", configPath)

	// Use sudo with -S to read password from stdin
	cmd := exec.Command("sudo", "-S", "wg-quick", "up", configPath)

	// Pass password via stdin
	cmd.Stdin = strings.NewReader(sudoPassword + "\n")

	// Send output to writer (which is already a multiwriter including auxlog and details)
	if c.writer == nil {
		panic("client writer is nil")
	}
	cmd.Stdout = c.writer
	cmd.Stderr = c.writer

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to bring up WireGuard interface. Please ensure:\n"+
			"  1. WireGuard is installed (wg-quick)\n"+
			"  2. You have sudo privileges\n"+
			"  3. The WireGuard kernel module is loaded\n"+
			"Error: %w", err)
	}

	c.mu.Lock()
	c.connected = true
	c.stats.ConnectedAt = time.Now()
	c.mu.Unlock()

	c.writef("VPN connection established successfully\n")
	c.writef("\nAccessible through VPN:\n")
	c.writef("  - VPN Gateway: %s\n", c.config.ServerIP)

	// Show routed IPs
	if len(c.config.PrivateIPs) > 0 {
		c.writef("  - Private IPs (%d total):\n", len(c.config.PrivateIPs))
		for _, ip := range c.config.PrivateIPs {
			c.writef("    • %s\n", ip)
		}
	}

	// Wait a moment for connection to stabilize
	time.Sleep(2 * time.Second)

	return nil
}

// writef writes a formatted message to the writer if available
// Also logs to aux logger for TUI integration
func (c *Client) writef(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	// Format with timestamp like log.Logger does
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	formattedMsg := fmt.Sprintf("%s [vpn client] %s", timestamp, msg)
	c.writer.Write([]byte(formattedMsg))
}

// Disconnect closes the VPN connection
func (c *Client) Disconnect(sudoPassword string) error {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return fmt.Errorf("client is not connected")
	}
	c.mu.Unlock()

	c.writef("Disconnecting from VPN server (may require sudo password)\n")

	// Get local hostname for directory structure
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get local hostname: %w", err)
	}

	// Bring down WireGuard interface
	// Use the same simple interface name as in Connect
	localWorkDir := fmt.Sprintf("/tmp/vastix/%s", hostname)
	configPath := fmt.Sprintf("%s/wgvastix.conf", localWorkDir)

	// Use sudo with -S to read password from stdin (same as Connect)
	cmd := exec.Command("sudo", "-S", "wg-quick", "down", configPath)

	// Pass password via stdin (empty password works for passwordless sudo)
	cmd.Stdin = strings.NewReader(sudoPassword + "\n")

	// Send output to writer (which is already a multiwriter including auxlog and details)
	cmd.Stdout = c.writer
	cmd.Stderr = c.writer

	if err := cmd.Run(); err != nil {
		c.writef("Warning: Failed to bring down interface: %v\n", err)
		// Don't fail disconnect if interface is already down
	}

	// Remove config file
	os.Remove(configPath)

	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()

	c.writef("VPN connection closed\n")

	return nil
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// GetStats returns connection statistics
func (c *Client) GetStats() ConnectionStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// Ping tests connectivity to a remote host through the VPN
func (c *Client) Ping(ctx context.Context, host string, timeout time.Duration) (time.Duration, error) {
	if !c.IsConnected() {
		return 0, fmt.Errorf("not connected to VPN")
	}

	start := time.Now()

	// Create a deadline context
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Try to establish TCP connection
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, "22"))
	if err != nil {
		return 0, fmt.Errorf("ping failed: %w", err)
	}
	defer conn.Close()

	latency := time.Since(start)
	return latency, nil
}

// DialTCP creates a TCP connection through the VPN
func (c *Client) DialTCP(ctx context.Context, addr string) (net.Conn, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to VPN")
	}

	// TODO: Use network.DialContext() for actual VPN routing

	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	return dialer.DialContext(ctx, "tcp", addr)
}

// GetStatus returns the client status
func (c *Client) GetStatus() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := map[string]interface{}{
		"connected":      c.connected,
		"serverEndpoint": c.config.ServerEndpoint,
		"clientIP":       c.config.ClientIP.String(),
		"serverIP":       c.config.ServerIP.String(),
	}

	if c.connected {
		status["connectedAt"] = c.stats.ConnectedAt
		status["uptime"] = time.Since(c.stats.ConnectedAt)
		status["bytesSent"] = c.stats.BytesSent
		status["bytesReceived"] = c.stats.BytesReceived
	}

	return status
}

// ResolveIP resolves a hostname to an IP address through the VPN
func (c *Client) ResolveIP(ctx context.Context, hostname string) ([]netip.Addr, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to VPN")
	}

	// Use system resolver for now
	// TODO: Use VPN's resolver if available

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s: %w", hostname, err)
	}

	result := make([]netip.Addr, 0, len(ips))
	for _, ip := range ips {
		if addr, ok := netip.AddrFromSlice(ip); ok {
			result = append(result, addr)
		}
	}

	return result, nil
}

// MonitorConnection monitors the VPN connection and returns stats
func (c *Client) MonitorConnection(ctx context.Context, interval time.Duration) <-chan ConnectionStats {
	statsChan := make(chan ConnectionStats)

	go func() {
		defer close(statsChan)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if c.IsConnected() {
					statsChan <- c.GetStats()
				}
			}
		}
	}()

	return statsChan
}

// CheckTunnelHealth checks if the VPN tunnel is still functional
// Tries to ping the VPN gateway to verify connectivity
func (c *Client) CheckTunnelHealth() error {
	c.mu.RLock()
	if !c.connected {
		c.mu.RUnlock()
		return fmt.Errorf("VPN not connected")
	}
	c.mu.RUnlock()

	// Ping the VPN gateway (server IP)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.Ping(ctx, c.config.ServerIP.String(), 5*time.Second)
	if err != nil {
		return fmt.Errorf("VPN tunnel unhealthy (cannot reach gateway %s): %w", c.config.ServerIP, err)
	}

	return nil
}
