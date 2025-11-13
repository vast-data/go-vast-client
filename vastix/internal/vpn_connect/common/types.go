// Package common provides shared types and utilities for VPN client/server
package common

import (
	"fmt"
	"net/netip"
)

// VPNConfig represents a VPN configuration
type VPNConfig struct {
	// Client configuration
	ClientPrivateKey string
	ClientPublicKey  string
	ClientIP         netip.Addr

	// Server configuration
	ServerPublicKey string
	ServerEndpoint  string // host:port
	ServerIP        netip.Addr

	// Network configuration
	PrivateIPs []netip.Addr // List of individual IPs to route (e.g., VIP pool IPs)
	VPNNetwork netip.Prefix // e.g., 10.99.0.0/24

	// Connection settings
	ListenPort          uint16
	PersistentKeepalive int // seconds, 0 to disable
}

// ServerConfig represents server-side configuration
type ServerConfig struct {
	PrivateKey string
	PublicKey  string
	ListenPort uint16
	ServerIP   netip.Addr
	VPNNetwork netip.Prefix
	PrivateIPs []netip.Addr // List of individual IPs to route (e.g., VIP pool IPs)
	Interface  string       // Network interface for NAT (e.g., eno1, eth0)
}

// ClientConfig represents client-side configuration
type ClientConfig struct {
	PrivateKey      string
	PublicKey       string
	ServerPublicKey string
	ServerEndpoint  string
	ClientIP        netip.Addr
	ServerIP        netip.Addr
	PrivateIPs      []netip.Addr // List of individual IPs to route (e.g., VIP pool IPs)
}

// DeploymentInfo contains information about a deployed VPN server
type DeploymentInfo struct {
	Host            string
	Port            uint16
	ServerPublicKey string
	ServerIP        netip.Addr
	ProcessPID      int
	LogFile         string
}

// ConnectionStatus represents the current VPN connection status
type ConnectionStatus struct {
	Connected      bool
	ServerEndpoint string
	BytesSent      uint64
	BytesReceived  uint64
	LastHandshake  int64 // Unix timestamp
	Latency        int   // milliseconds
}

// GenerateVPNNetwork generates a unique VPN network for a client
// This allows multiple concurrent VPN connections without conflicts
func GenerateVPNNetwork(clientID int) (netip.Prefix, netip.Addr, netip.Addr, error) {
	// Use different /24 networks for each client: 10.99.X.0/24
	// where X is the client ID (1-254)
	if clientID < 1 || clientID > 254 {
		return netip.Prefix{}, netip.Addr{}, netip.Addr{}, fmt.Errorf("client ID must be between 1 and 254")
	}

	network := fmt.Sprintf("10.99.%d.0/24", clientID)
	serverIP := fmt.Sprintf("10.99.%d.1", clientID)
	clientIP := fmt.Sprintf("10.99.%d.2", clientID)

	vpnNet, err := netip.ParsePrefix(network)
	if err != nil {
		return netip.Prefix{}, netip.Addr{}, netip.Addr{}, err
	}

	srvIP, err := netip.ParseAddr(serverIP)
	if err != nil {
		return netip.Prefix{}, netip.Addr{}, netip.Addr{}, err
	}

	cliIP, err := netip.ParseAddr(clientIP)
	if err != nil {
		return netip.Prefix{}, netip.Addr{}, netip.Addr{}, err
	}

	return vpnNet, srvIP, cliIP, nil
}

// GetListenPort generates a unique listen port for a client
// This allows multiple servers to run simultaneously
func GetListenPort(clientID int) uint16 {
	// Use ports 51820 + clientID (51821, 51822, ...)
	return uint16(51820 + clientID)
}

// Error types
type VPNError struct {
	Operation string
	Err       error
}

func (e *VPNError) Error() string {
	return fmt.Sprintf("VPN %s error: %v", e.Operation, e.Err)
}

func NewVPNError(operation string, err error) *VPNError {
	return &VPNError{Operation: operation, Err: err}
}
