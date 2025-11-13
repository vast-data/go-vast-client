// Example: Integrating VPN Connect into a Go application
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/netip"
	"os"
	"time"

	"vastix/internal/vpn_connect/client"
	"vastix/internal/vpn_connect/common"
)

// VPNManager manages VPN lifecycle for your application
type VPNManager struct {
	deployer  *client.Deployer
	vpnClient *client.Client
	logger    *slog.Logger
	connected bool
	serverKey string
}

// NewVPNManager creates a new VPN manager
func NewVPNManager() *VPNManager {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	return &VPNManager{
		deployer: client.NewDeployer(nil, nil),
		logger:   logger,
	}
}

// DeployAndConnect deploys a VPN server and connects to it
func (m *VPNManager) DeployAndConnect(ctx context.Context, clientID int, remoteHost, remoteUser, remotePassword string) error {
	// Generate unique network for this client
	vpnNetwork, serverIP, clientIP, err := common.GenerateVPNNetwork(clientID)
	if err != nil {
		return fmt.Errorf("failed to generate VPN network: %w", err)
	}

	port := common.GetListenPort(clientID)

	m.logger.Info("Starting VPN deployment",
		slog.Int("clientID", clientID),
		slog.String("network", vpnNetwork.String()),
		slog.Uint64("port", uint64(port)))

	// Step 1: Connect to remote host
	deployConfig := &client.DeploymentConfig{
		Host:          remoteHost,
		Port:          22,
		Username:      remoteUser,
		Password:      remotePassword,
		RemoteWorkDir: fmt.Sprintf("/tmp/vpn-client-%d", clientID),
	}

	if err := m.deployer.Connect(ctx, deployConfig); err != nil {
		return fmt.Errorf("failed to connect to remote host: %w", err)
	}

	m.logger.Info("Connected to remote host", slog.String("host", remoteHost))

	// Step 2: Generate server keys
	serverPrivKey, serverPubKey, err := common.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate server keys: %w", err)
	}

	m.serverKey = serverPubKey
	m.logger.Info("Generated server keys", slog.String("publicKey", serverPubKey))

	// Step 3: Deploy server
	serverConfig := &common.ServerConfig{
		PrivateKey:     serverPrivKey,
		PublicKey:      serverPubKey,
		ListenPort:     port,
		ServerIP:       serverIP,
		VPNNetwork:     vpnNetwork,
		PrivateNetwork: netip.MustParsePrefix("172.21.101.0/24"),
	}

	if err := m.deployer.Deploy(ctx, deployConfig, serverConfig); err != nil {
		return fmt.Errorf("failed to deploy server: %w", err)
	}

	m.logger.Info("Server deployed successfully")

	// Step 4: Start server
	if err := m.deployer.StartServer(deployConfig.RemoteWorkDir, port); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	m.logger.Info("Server started successfully")

	// Step 5: Generate client keys
	clientPrivKey, clientPubKey, err := common.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate client keys: %w", err)
	}

	m.logger.Info("Generated client keys", slog.String("publicKey", clientPubKey))

	// Step 6: Create and connect client
	clientConfig := &common.ClientConfig{
		PrivateKey:      clientPrivKey,
		PublicKey:       clientPubKey,
		ServerPublicKey: serverPubKey,
		ServerEndpoint:  fmt.Sprintf("%s:%d", remoteHost, port),
		ClientIP:        clientIP,
		ServerIP:        serverIP,
		PrivateNetwork:  netip.MustParsePrefix("172.21.101.0/24"),
	}

	m.vpnClient, err = client.NewClient(clientConfig, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to create VPN client: %w", err)
	}

	if err := m.vpnClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect VPN client: %w", err)
	}

	m.connected = true
	m.logger.Info("VPN connection established successfully")

	return nil
}

// IsConnected returns whether the VPN is connected
func (m *VPNManager) IsConnected() bool {
	return m.connected && m.vpnClient != nil && m.vpnClient.IsConnected()
}

// Disconnect closes the VPN connection
func (m *VPNManager) Disconnect() error {
	if m.vpnClient != nil && m.connected {
		// Pass empty password for passwordless sudo or cached password
		// In real application, you may want to store and reuse the sudo password
		if err := m.vpnClient.Disconnect(""); err != nil {
			return fmt.Errorf("failed to disconnect: %w", err)
		}
		m.connected = false
		m.logger.Info("VPN disconnected")
	}

	if m.deployer != nil {
		if err := m.deployer.Disconnect(); err != nil {
			return fmt.Errorf("failed to close SSH connection: %w", err)
		}
	}

	return nil
}

// GetStats returns connection statistics
func (m *VPNManager) GetStats() (client.ConnectionStats, error) {
	if m.vpnClient == nil || !m.connected {
		return client.ConnectionStats{}, fmt.Errorf("not connected")
	}

	return m.vpnClient.GetStats(), nil
}

// TestConnectivity tests connectivity to a host through the VPN
func (m *VPNManager) TestConnectivity(ctx context.Context, host string) (time.Duration, error) {
	if m.vpnClient == nil || !m.connected {
		return 0, fmt.Errorf("not connected")
	}

	return m.vpnClient.Ping(ctx, host, 5*time.Second)
}

// Example usage in your application
func main() {
	ctx := context.Background()

	// Create VPN manager
	manager := NewVPNManager()
	defer manager.Disconnect()

	// Deploy and connect
	log.Println("Deploying and connecting VPN...")
	err := manager.DeployAndConnect(ctx, 1, "10.27.14.107", "centos", "your-password")
	if err != nil {
		log.Fatalf("Failed to setup VPN: %v", err)
	}

	log.Println("✓ VPN connected successfully!")

	// Test connectivity
	log.Println("Testing connectivity to private network...")
	latency, err := manager.TestConnectivity(ctx, "172.21.101.1")
	if err != nil {
		log.Printf("Warning: Connectivity test failed: %v", err)
	} else {
		log.Printf("✓ Ping successful! Latency: %v", latency)
	}

	// Get connection stats
	stats, err := manager.GetStats()
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
	} else {
		log.Printf("Connection stats: Sent=%d bytes, Received=%d bytes, Uptime=%v",
			stats.BytesSent, stats.BytesReceived, time.Since(stats.ConnectedAt))
	}

	// Your application logic here...
	// The VPN is now active and you can access the private network

	log.Println("Application running. Press Ctrl+C to exit...")

	// Keep running
	<-ctx.Done()

	log.Println("Shutting down...")
}
