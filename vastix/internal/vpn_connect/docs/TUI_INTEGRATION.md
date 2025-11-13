# VPN Connect - TUI Application Integration Guide

Complete guide for integrating VPN Connect into your Terminal UI (TUI) application.

## Overview

This guide shows how to integrate VPN functionality into your TUI application, allowing users to:
- Deploy VPN servers programmatically
- Connect/disconnect on-demand
- Monitor connection status
- Support multiple simultaneous users

## Architecture for TUI Integration

```
┌─────────────────────────────────────────────────────────┐
│                  Your TUI Application                   │
│                                                         │
│  ┌────────────────┐  ┌──────────────┐  ┌────────────┐ │
│  │  VPN Widget    │  │  Status Bar  │  │  Settings  │ │
│  │  (User Input)  │  │  (Show VPN)  │  │  (Config)  │ │
│  └───────┬────────┘  └──────┬───────┘  └─────┬──────┘ │
│          │                  │                 │        │
│          └──────────────────┼─────────────────┘        │
│                             │                          │
│                     ┌───────▼────────┐                 │
│                     │  VPN Manager   │                 │
│                     │  (Your Code)   │                 │
│                     └───────┬────────┘                 │
└─────────────────────────────┼──────────────────────────┘
                              │
                    ┌─────────┴──────────┐
                    │                    │
          ┌─────────▼────────┐  ┌────────▼─────────┐
          │   VPN Connect    │  │   VPN Connect    │
          │    Deployer      │  │     Client       │
          │  (This Library)  │  │  (This Library)  │
          └──────────────────┘  └──────────────────┘
```

## VPN Manager Implementation

### Basic VPN Manager

```go
package vpn

import (
	"context"
	"fmt"
	"log/slog"
	"net/netip"
	"sync"
	"time"

	"github.com/vastdata/go-vast-client/vastix/vpn_connect/client"
	"github.com/vastdata/go-vast-client/vastix/vpn_connect/common"
)

// Manager handles VPN lifecycle for TUI app
type Manager struct {
	mu        sync.RWMutex
	deployer  *client.Deployer
	vpnClient *client.Client
	logger    *slog.Logger
	
	// State
	connected     bool
	deploying     bool
	deployError   error
	serverKey     string
	clientID      int
	
	// Configuration
	remoteHost     string
	remoteUser     string
	remotePassword string
	privateNetwork netip.Prefix
	
	// Monitoring
	stats         client.ConnectionStats
	lastPingTime  time.Duration
	statusUpdates chan StatusUpdate
}

// StatusUpdate represents a VPN status change
type StatusUpdate struct {
	Type      string // "deploying", "deployed", "connecting", "connected", "disconnected", "error"
	Message   string
	Error     error
	Timestamp time.Time
}

// Config holds VPN configuration
type Config struct {
	RemoteHost     string
	RemoteUser     string
	RemotePassword string
	RemoteKeyPath  string
	PrivateNetwork string
	ClientID       int
}

// NewManager creates a new VPN manager
func NewManager(config *Config, logger *slog.Logger) (*Manager, error) {
	if logger == nil {
		logger = slog.Default()
	}

	privNet, err := netip.ParsePrefix(config.PrivateNetwork)
	if err != nil {
		return nil, fmt.Errorf("invalid private network: %w", err)
	}

	return &Manager{
		deployer:       client.NewDeployer(logger),
		logger:         logger,
		clientID:       config.ClientID,
		remoteHost:     config.RemoteHost,
		remoteUser:     config.RemoteUser,
		remotePassword: config.RemotePassword,
		privateNetwork: privNet,
		statusUpdates:  make(chan StatusUpdate, 10),
	}, nil
}

// Deploy deploys the VPN server to remote machine
func (m *Manager) Deploy(ctx context.Context) error {
	m.mu.Lock()
	if m.deploying {
		m.mu.Unlock()
		return fmt.Errorf("deployment already in progress")
	}
	m.deploying = true
	m.deployError = nil
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.deploying = false
		m.mu.Unlock()
	}()

	m.sendStatus("deploying", "Deploying VPN server...", nil)

	// Generate network configuration
	vpnNetwork, serverIP, clientIP, err := common.GenerateVPNNetwork(m.clientID)
	if err != nil {
		m.deployError = err
		m.sendStatus("error", "Failed to generate network", err)
		return err
	}

	port := common.GetListenPort(m.clientID)

	// Connect to remote host
	deployConfig := &client.DeploymentConfig{
		Host:          m.remoteHost,
		Port:          22,
		Username:      m.remoteUser,
		Password:      m.remotePassword,
		RemoteWorkDir: fmt.Sprintf("/tmp/vpn-client-%d", m.clientID),
	}

	if err := m.deployer.Connect(ctx, deployConfig); err != nil {
		m.deployError = err
		m.sendStatus("error", "Failed to connect to remote host", err)
		return err
	}

	m.sendStatus("deploying", "Generating keys...", nil)

	// Generate server keys
	serverPrivKey, serverPubKey, err := common.GenerateKeyPair()
	if err != nil {
		m.deployError = err
		m.sendStatus("error", "Failed to generate keys", err)
		return err
	}

	m.mu.Lock()
	m.serverKey = serverPubKey
	m.mu.Unlock()

	m.sendStatus("deploying", "Uploading server binary...", nil)

	// Deploy server
	serverConfig := &common.ServerConfig{
		PrivateKey:     serverPrivKey,
		PublicKey:      serverPubKey,
		ListenPort:     port,
		ServerIP:       serverIP,
		VPNNetwork:     vpnNetwork,
		PrivateNetwork: m.privateNetwork,
	}

	if err := m.deployer.Deploy(ctx, deployConfig, serverConfig); err != nil {
		m.deployError = err
		m.sendStatus("error", "Failed to deploy server", err)
		return err
	}

	m.sendStatus("deploying", "Starting server...", nil)

	// Start server
	if err := m.deployer.StartServer(deployConfig.RemoteWorkDir, port); err != nil {
		m.deployError = err
		m.sendStatus("error", "Failed to start server", err)
		return err
	}

	m.sendStatus("deployed", "VPN server deployed successfully", nil)

	// Auto-connect after deployment
	return m.Connect(ctx)
}

// Connect establishes VPN connection
func (m *Manager) Connect(ctx context.Context) error {
	m.mu.Lock()
	if m.connected {
		m.mu.Unlock()
		return fmt.Errorf("already connected")
	}
	if m.serverKey == "" {
		m.mu.Unlock()
		return fmt.Errorf("server not deployed")
	}
	m.mu.Unlock()

	m.sendStatus("connecting", "Connecting to VPN...", nil)

	// Generate client keys
	clientPrivKey, clientPubKey, err := common.GenerateKeyPair()
	if err != nil {
		m.sendStatus("error", "Failed to generate client keys", err)
		return err
	}

	// Get network configuration
	_, serverIP, clientIP, _ := common.GenerateVPNNetwork(m.clientID)
	port := common.GetListenPort(m.clientID)

	// Create client
	clientConfig := &common.ClientConfig{
		PrivateKey:      clientPrivKey,
		PublicKey:       clientPubKey,
		ServerPublicKey: m.serverKey,
		ServerEndpoint:  fmt.Sprintf("%s:%d", m.remoteHost, port),
		ClientIP:        clientIP,
		ServerIP:        serverIP,
		PrivateNetwork:  m.privateNetwork,
	}

	m.mu.Lock()
	m.vpnClient, err = client.NewClient(clientConfig, m.logger)
	m.mu.Unlock()

	if err != nil {
		m.sendStatus("error", "Failed to create VPN client", err)
		return err
	}

	if err := m.vpnClient.Connect(ctx); err != nil {
		m.sendStatus("error", "Failed to connect", err)
		return err
	}

	m.mu.Lock()
	m.connected = true
	m.stats.ConnectedAt = time.Now()
	m.mu.Unlock()

	m.sendStatus("connected", "VPN connected successfully", nil)

	// Start monitoring
	go m.monitorConnection(ctx)

	return nil
}

// Disconnect closes VPN connection
func (m *Manager) Disconnect() error {
	m.mu.Lock()
	if !m.connected || m.vpnClient == nil {
		m.mu.Unlock()
		return fmt.Errorf("not connected")
	}
	m.mu.Unlock()

	if err := m.vpnClient.Disconnect(); err != nil {
		return err
	}

	m.mu.Lock()
	m.connected = false
	m.mu.Unlock()

	m.sendStatus("disconnected", "VPN disconnected", nil)

	return nil
}

// IsConnected returns connection status
func (m *Manager) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// GetStats returns connection statistics
func (m *Manager) GetStats() client.ConnectionStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

// GetLastPing returns last ping latency
func (m *Manager) GetLastPing() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastPingTime
}

// StatusUpdates returns a channel for status updates
func (m *Manager) StatusUpdates() <-chan StatusUpdate {
	return m.statusUpdates
}

// sendStatus sends a status update
func (m *Manager) sendStatus(typ, message string, err error) {
	select {
	case m.statusUpdates <- StatusUpdate{
		Type:      typ,
		Message:   message,
		Error:     err,
		Timestamp: time.Now(),
	}:
	default:
		// Channel full, skip
	}
}

// monitorConnection monitors VPN connection
func (m *Manager) monitorConnection(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !m.IsConnected() {
				return
			}

			// Update stats
			m.mu.Lock()
			if m.vpnClient != nil {
				m.stats = m.vpnClient.GetStats()
			}
			m.mu.Unlock()

			// Test connectivity
			latency, err := m.vpnClient.Ping(ctx, m.privateNetwork.Addr().String(), 5*time.Second)
			if err == nil {
				m.mu.Lock()
				m.lastPingTime = latency
				m.mu.Unlock()
			}
		}
	}
}

// Close cleans up resources
func (m *Manager) Close() error {
	if m.IsConnected() {
		m.Disconnect()
	}

	if m.deployer != nil {
		m.deployer.Disconnect()
	}

	close(m.statusUpdates)

	return nil
}
```

## TUI Widget Example (using Bubble Tea)

```go
package main

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vastdata/go-vast-client/vastix/vpn_connect/vpn"
)

// VPN Widget Model
type vpnWidget struct {
	manager     *vpn.Manager
	status      string
	error       error
	stats       string
	deploying   bool
	connected   bool
	ctx         context.Context
	cancel      context.CancelFunc
}

type vpnStatusMsg struct {
	update vpn.StatusUpdate
}

type vpnStatsMsg struct {
	stats string
}

func newVPNWidget(config *vpn.Config) *vpnWidget {
	ctx, cancel := context.WithCancel(context.Background())
	
	manager, _ := vpn.NewManager(config, nil)
	
	w := &vpnWidget{
		manager:   manager,
		status:    "Not connected",
		ctx:       ctx,
		cancel:    cancel,
	}

	return w
}

func (w *vpnWidget) Init() tea.Cmd {
	return tea.Batch(
		w.listenForStatusUpdates(),
		w.updateStats(),
	)
}

func (w *vpnWidget) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "d": // Deploy
			if !w.deploying && !w.connected {
				w.deploying = true
				go w.manager.Deploy(w.ctx)
				return w, w.listenForStatusUpdates()
			}

		case "c": // Connect
			if !w.connected && !w.deploying {
				go w.manager.Connect(w.ctx)
				return w, w.listenForStatusUpdates()
			}

		case "x": // Disconnect
			if w.connected {
				w.manager.Disconnect()
				w.connected = false
				w.status = "Disconnected"
			}
		}

	case vpnStatusMsg:
		w.status = msg.update.Message
		w.error = msg.update.Error

		switch msg.update.Type {
		case "deployed":
			w.deploying = false
		case "connected":
			w.connected = true
			w.deploying = false
			return w, w.updateStats()
		case "disconnected":
			w.connected = false
		case "error":
			w.deploying = false
		}

		return w, w.listenForStatusUpdates()

	case vpnStatsMsg:
		w.stats = msg.stats
		if w.connected {
			return w, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
				return w.getStatsMsg()
			})
		}
	}

	return w, nil
}

func (w *vpnWidget) View() string {
	var style = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2)

	var content string

	// Status indicator
	statusColor := "red"
	if w.connected {
		statusColor = "green"
	} else if w.deploying {
		statusColor = "yellow"
	}

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(statusColor)).
		Bold(true)

	content += statusStyle.Render("● ") + w.status + "\n\n"

	// Show error if any
	if w.error != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("red"))
		content += errorStyle.Render("Error: "+w.error.Error()) + "\n\n"
	}

	// Show stats if connected
	if w.connected {
		content += w.stats + "\n\n"
	}

	// Show controls
	content += lipgloss.NewStyle().Faint(true).Render(
		"Controls: [d] Deploy  [c] Connect  [x] Disconnect  [q] Quit",
	)

	return style.Render(content)
}

func (w *vpnWidget) listenForStatusUpdates() tea.Cmd {
	return func() tea.Msg {
		select {
		case update := <-w.manager.StatusUpdates():
			return vpnStatusMsg{update}
		case <-w.ctx.Done():
			return nil
		}
	}
}

func (w *vpnWidget) updateStats() tea.Cmd {
	return func() tea.Msg {
		return w.getStatsMsg()
	}
}

func (w *vpnWidget) getStatsMsg() tea.Msg {
	stats := w.manager.GetStats()
	ping := w.manager.GetLastPing()

	statsStr := fmt.Sprintf(
		"Uptime: %v\nBytes Sent: %d\nBytes Received: %d\nPing: %v",
		time.Since(stats.ConnectedAt).Round(time.Second),
		stats.BytesSent,
		stats.BytesReceived,
		ping,
	)

	return vpnStatsMsg{statsStr}
}

func (w *vpnWidget) Close() {
	w.cancel()
	w.manager.Close()
}
```

## Simple TUI Example

For a simpler TUI without Bubble Tea:

```go
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/vastdata/go-vast-client/vastix/vpn_connect/vpn"
)

func main() {
	ctx := context.Background()
	
	config := &vpn.Config{
		RemoteHost:     "10.27.14.107",
		RemoteUser:     "centos",
		RemotePassword: "your-password",
		PrivateNetwork: "172.21.101.0/24",
		ClientID:       1,
	}

	manager, err := vpn.NewManager(config, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer manager.Close()

	// Listen for status updates
	go func() {
		for update := range manager.StatusUpdates() {
			fmt.Printf("[%s] %s\n", update.Type, update.Message)
			if update.Error != nil {
				fmt.Printf("  Error: %v\n", update.Error)
			}
		}
	}()

	reader := bufio.NewReader(os.Stdin)

	for {
		printMenu(manager)
		
		fmt.Print("\nSelect option: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1": // Deploy
			if !manager.IsConnected() {
				fmt.Println("Deploying VPN server...")
				if err := manager.Deploy(ctx); err != nil {
					fmt.Printf("Deployment failed: %v\n", err)
				}
			} else {
				fmt.Println("Already connected!")
			}

		case "2": // Connect
			if !manager.IsConnected() {
				fmt.Println("Connecting to VPN...")
				if err := manager.Connect(ctx); err != nil {
					fmt.Printf("Connection failed: %v\n", err)
				}
			} else {
				fmt.Println("Already connected!")
			}

		case "3": // Disconnect
			if manager.IsConnected() {
				if err := manager.Disconnect(); err != nil {
					fmt.Printf("Disconnect failed: %v\n", err)
				} else {
					fmt.Println("Disconnected successfully")
				}
			} else {
				fmt.Println("Not connected!")
			}

		case "4": // Status
			if manager.IsConnected() {
				stats := manager.GetStats()
				ping := manager.GetLastPing()
				fmt.Printf("\nStatus: Connected\n")
				fmt.Printf("Uptime: %v\n", stats.ConnectedAt)
				fmt.Printf("Bytes Sent: %d\n", stats.BytesSent)
				fmt.Printf("Bytes Received: %d\n", stats.BytesReceived)
				fmt.Printf("Last Ping: %v\n", ping)
			} else {
				fmt.Println("\nStatus: Not connected")
			}

		case "q", "5": // Quit
			fmt.Println("Exiting...")
			return

		default:
			fmt.Println("Invalid option")
		}
	}
}

func printMenu(manager *vpn.Manager) {
	fmt.Println("\n=== VPN Manager ===")
	if manager.IsConnected() {
		fmt.Println("Status: ● Connected")
	} else {
		fmt.Println("Status: ○ Not Connected")
	}
	fmt.Println("\n1. Deploy VPN Server")
	fmt.Println("2. Connect")
	fmt.Println("3. Disconnect")
	fmt.Println("4. Show Status")
	fmt.Println("5. Quit")
}
```

## Best Practices for TUI Integration

### 1. Async Operations

Always run VPN operations in goroutines:

```go
go func() {
    if err := manager.Deploy(ctx); err != nil {
        // Handle error in UI
    }
}()
```

### 2. Status Monitoring

Use the status updates channel:

```go
for update := range manager.StatusUpdates() {
    // Update UI based on status
    updateStatusBar(update.Type, update.Message)
}
```

### 3. Error Handling

Provide clear error messages to users:

```go
if err != nil {
    showErrorDialog(fmt.Sprintf("VPN Error: %v\nPlease check network connectivity.", err))
}
```

### 4. User Feedback

Show progress for long operations:

```go
statusTypes := map[string]string{
    "deploying":   "🔄 Deploying server...",
    "deployed":    "✓ Server deployed",
    "connecting":  "🔄 Connecting...",
    "connected":   "✓ Connected",
    "disconnected": "○ Disconnected",
    "error":       "✗ Error",
}
```

### 5. Configuration Persistence

Save VPN configuration:

```go
type SavedConfig struct {
    RemoteHost     string
    ServerKey      string
    ClientID       int
    LastConnected  time.Time
}

// Save to ~/.config/yourapp/vpn.json
```

## Complete Integration Checklist

- [ ] Create VPN Manager wrapper
- [ ] Implement status update handling
- [ ] Add VPN widget to TUI
- [ ] Implement connection lifecycle
- [ ] Add status indicators
- [ ] Show connection statistics
- [ ] Handle errors gracefully
- [ ] Save/load configuration
- [ ] Add keyboard shortcuts
- [ ] Test multi-user scenarios
- [ ] Document VPN features for users

## Example: Full TUI App Structure

```
your-tui-app/
├── main.go                  # Entry point
├── ui/
│   ├── app.go              # Main TUI app
│   ├── vpn_widget.go       # VPN widget
│   ├── status_bar.go       # Status bar with VPN indicator
│   └── settings.go         # Settings screen
├── vpn/
│   ├── manager.go          # VPN manager (from above)
│   └── config.go           # VPN configuration
└── config/
    └── config.go           # App configuration
```

This structure keeps VPN functionality modular and easy to integrate!

