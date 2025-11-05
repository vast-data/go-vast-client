# VPN Connect - Go API Reference

Complete API documentation for using VPN Connect as a Go library.

## Table of Contents

- [Common Package](#common-package)
- [Server Package](#server-package)
- [Client Package](#client-package)
- [Examples](#examples)

## Common Package

### Types

#### `VPNConfig`

Represents a complete VPN configuration.

```go
type VPNConfig struct {
    ClientPrivateKey string
    ClientPublicKey  string
    ClientIP         netip.Addr
    ServerPublicKey string
    ServerEndpoint  string
    ServerIP        netip.Addr
    PrivateNetwork netip.Prefix
    VPNNetwork     netip.Prefix
    ListenPort       uint16
    PersistentKeepalive int
}
```

#### `ServerConfig`

Server-side configuration.

```go
type ServerConfig struct {
    PrivateKey     string
    PublicKey      string
    ListenPort     uint16
    ServerIP       netip.Addr
    VPNNetwork     netip.Prefix
    PrivateNetwork netip.Prefix
    Interface      string
}
```

#### `ClientConfig`

Client-side configuration.

```go
type ClientConfig struct {
    PrivateKey      string
    PublicKey       string
    ServerPublicKey string
    ServerEndpoint  string
    ClientIP        netip.Addr
    ServerIP        netip.Addr
    PrivateNetwork  netip.Prefix
}
```

### Functions

#### `GenerateKeyPair`

Generates a new WireGuard key pair.

```go
func GenerateKeyPair() (privateKey, publicKey string, err error)
```

**Example:**
```go
privKey, pubKey, err := common.GenerateKeyPair()
if err != nil {
    log.Fatal(err)
}
fmt.Println("Public Key:", pubKey)
```

#### `GetPublicKey`

Derives the public key from a private key.

```go
func GetPublicKey(privateKeyBase64 string) (string, error)
```

**Example:**
```go
pubKey, err := common.GetPublicKey(privateKey)
```

#### `ValidateKey`

Validates a base64-encoded key.

```go
func ValidateKey(keyBase64 string) error
```

#### `GenerateVPNNetwork`

Generates a unique VPN network for a client ID (1-254).

```go
func GenerateVPNNetwork(clientID int) (netip.Prefix, netip.Addr, netip.Addr, error)
```

**Returns:** `(vpnNetwork, serverIP, clientIP, error)`

**Example:**
```go
// For client 1: generates 10.99.1.0/24, 10.99.1.1, 10.99.1.2
network, serverIP, clientIP, err := common.GenerateVPNNetwork(1)
```

#### `GetListenPort`

Gets a unique port for a client ID.

```go
func GetListenPort(clientID int) uint16
```

**Example:**
```go
port := common.GetListenPort(1) // Returns 51821
```

---

## Server Package

### Types

#### `Server`

Main VPN server instance.

```go
type Server struct {
    // private fields
}
```

#### `ClientSession`

Represents a connected client.

```go
type ClientSession struct {
    PublicKey     string
    ClientIP      netip.Addr
    AllowedIPs    []netip.Prefix
    ConnectedAt   time.Time
    LastHandshake time.Time
    BytesSent     uint64
    BytesReceived uint64
}
```

### Functions

#### `NewServer`

Creates a new VPN server instance.

```go
func NewServer(config *common.ServerConfig, logger *slog.Logger) (*Server, error)
```

**Parameters:**
- `config`: Server configuration
- `logger`: slog.Logger for logging (can be nil for default)

**Example:**
```go
logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

config := &common.ServerConfig{
    PrivateKey:     privKey,
    PublicKey:      pubKey,
    ListenPort:     51821,
    ServerIP:       netip.MustParseAddr("10.99.1.1"),
    VPNNetwork:     netip.MustParsePrefix("10.99.1.0/24"),
    PrivateNetwork: netip.MustParsePrefix("172.21.101.0/24"),
    Interface:      "eno1",
}

srv, err := server.NewServer(config, logger)
```

### Methods

#### `Start`

Starts the VPN server.

```go
func (s *Server) Start(ctx context.Context) error
```

**Example:**
```go
ctx := context.Background()
if err := srv.Start(ctx); err != nil {
    log.Fatal(err)
}
```

#### `Stop`

Stops the VPN server.

```go
func (s *Server) Stop() error
```

#### `AddClient`

Adds a client to the server.

```go
func (s *Server) AddClient(publicKey string, clientIP netip.Addr, allowedIPs []netip.Prefix) error
```

**Example:**
```go
clientPubKey := "client-public-key-here"
clientIP := netip.MustParseAddr("10.99.1.2")
allowedIPs := []netip.Prefix{
    netip.MustParsePrefix("10.99.1.2/32"),
    netip.MustParsePrefix("172.21.101.0/24"),
}
err := srv.AddClient(clientPubKey, clientIP, allowedIPs)
```

#### `RemoveClient`

Removes a client from the server.

```go
func (s *Server) RemoveClient(publicKey string) error
```

#### `GetStatus`

Returns server status information.

```go
func (s *Server) GetStatus() map[string]interface{}
```

**Returns:** Map with keys: `running`, `port`, `serverIP`, `clientCount`, `configDir`, `pid`

#### `GetClients`

Returns list of connected clients.

```go
func (s *Server) GetClients() []*ClientSession
```

#### `IsRunning`

Checks if server is running.

```go
func (s *Server) IsRunning() bool
```

#### `ReadLogs`

Returns the last N lines of server logs.

```go
func (s *Server) ReadLogs(lines int) ([]string, error)
```

### Utility Functions

#### `CheckWireGuardInstalled`

Checks if WireGuard is installed.

```go
func CheckWireGuardInstalled() (bool, error)
```

#### `InstallWireGuard`

Attempts to install WireGuard.

```go
func InstallWireGuard() error
```

#### `DetectExternalInterface`

Detects the external network interface.

```go
func DetectExternalInterface() (string, error)
```

**Example:**
```go
iface, err := server.DetectExternalInterface()
// Returns "eno1", "eth0", etc.
```

#### `GetNextAvailablePort`

Finds next available port starting from basePort.

```go
func GetNextAvailablePort(basePort uint16) (uint16, error)
```

---

## Client Package

### Types

#### `Client`

Main VPN client instance.

```go
type Client struct {
    // private fields
}
```

#### `ConnectionStats`

Connection statistics.

```go
type ConnectionStats struct {
    BytesSent      uint64
    BytesReceived  uint64
    LastHandshake  time.Time
    ConnectedAt    time.Time
    PacketsSent    uint64
    PacketsReceived uint64
}
```

#### `Deployer`

Handles remote server deployment.

```go
type Deployer struct {
    // private fields
}
```

#### `DeploymentConfig`

Configuration for remote deployment.

```go
type DeploymentConfig struct {
    Host            string
    Port            int
    Username        string
    Password        string
    PrivateKeyPath  string
    ServerBinaryPath string
    RemoteWorkDir   string
}
```

### Client Functions

#### `NewClient`

Creates a new VPN client instance.

```go
func NewClient(config *common.ClientConfig, logger *slog.Logger) (*Client, error)
```

**Example:**
```go
config := &common.ClientConfig{
    PrivateKey:      privKey,
    PublicKey:       pubKey,
    ServerPublicKey: serverPubKey,
    ServerEndpoint:  "10.27.14.107:51821",
    ClientIP:        netip.MustParseAddr("10.99.1.2"),
    ServerIP:        netip.MustParseAddr("10.99.1.1"),
    PrivateNetwork:  netip.MustParsePrefix("172.21.101.0/24"),
}

client, err := client.NewClient(config, logger)
```

### Client Methods

#### `Connect`

Establishes VPN connection.

```go
func (c *Client) Connect(ctx context.Context) error
```

#### `Disconnect`

Closes VPN connection.

```go
func (c *Client) Disconnect() error
```

#### `IsConnected`

Checks if client is connected.

```go
func (c *Client) IsConnected() bool
```

#### `GetStats`

Returns connection statistics.

```go
func (c *Client) GetStats() ConnectionStats
```

#### `Ping`

Tests connectivity through VPN.

```go
func (c *Client) Ping(ctx context.Context, host string, timeout time.Duration) (time.Duration, error)
```

**Example:**
```go
latency, err := client.Ping(ctx, "172.21.101.1", 5*time.Second)
fmt.Printf("Latency: %v\n", latency)
```

#### `DialTCP`

Creates TCP connection through VPN.

```go
func (c *Client) DialTCP(ctx context.Context, addr string) (net.Conn, error)
```

**Example:**
```go
conn, err := client.DialTCP(ctx, "172.21.101.1:2049")
// Use conn for NFS or other TCP protocols
```

#### `GetStatus`

Returns client status.

```go
func (c *Client) GetStatus() map[string]interface{}
```

#### `MonitorConnection`

Monitors connection and returns stats channel.

```go
func (c *Client) MonitorConnection(ctx context.Context, interval time.Duration) <-chan ConnectionStats
```

**Example:**
```go
statsCh := client.MonitorConnection(ctx, 5*time.Second)
for stats := range statsCh {
    fmt.Printf("Sent: %d bytes, Received: %d bytes\n", stats.BytesSent, stats.BytesReceived)
}
```

### Deployer Functions

#### `NewDeployer`

Creates a new deployer instance.

```go
func NewDeployer(logger *slog.Logger) *Deployer
```

### Deployer Methods

#### `Connect`

Connects to remote host via SSH.

```go
func (d *Deployer) Connect(ctx context.Context, config *DeploymentConfig) error
```

**Example:**
```go
deployer := client.NewDeployer(logger)

config := &client.DeploymentConfig{
    Host:          "10.27.14.107",
    Port:          22,
    Username:      "centos",
    Password:      "password",
    RemoteWorkDir: "/tmp/vpn-server",
}

err := deployer.Connect(ctx, config)
defer deployer.Disconnect()
```

#### `Disconnect`

Closes SSH connection.

```go
func (d *Deployer) Disconnect() error
```

#### `Deploy`

Deploys VPN server to remote machine.

```go
func (d *Deployer) Deploy(ctx context.Context, config *DeploymentConfig, serverConfig *common.ServerConfig) error
```

#### `StartServer`

Starts the remote VPN server.

```go
func (d *Deployer) StartServer(workDir string, port uint16) error
```

#### `StopServer`

Stops the remote VPN server.

```go
func (d *Deployer) StopServer(port uint16) error
```

#### `GetServerStatus`

Checks if remote server is running.

```go
func (d *Deployer) GetServerStatus(port uint16) (bool, string, error)
```

**Returns:** `(running, pid, error)`

#### `GetServerLogs`

Retrieves server logs.

```go
func (d *Deployer) GetServerLogs(workDir string, lines int) (string, error)
```

---

## Complete Examples

### Basic Server

```go
package main

import (
    "context"
    "log/slog"
    "net/netip"
    "os"
    "os/signal"
    "syscall"

    "github.com/vastdata/go-vast-client/vastix/vpn_connect/common"
    "github.com/vastdata/go-vast-client/vastix/vpn_connect/server"
)

func main() {
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

    // Generate keys
    privKey, pubKey, err := common.GenerateKeyPair()
    if err != nil {
        logger.Error("Failed to generate keys", slog.Any("error", err))
        return
    }

    // Detect interface
    iface, _ := server.DetectExternalInterface()

    // Configure server
    config := &common.ServerConfig{
        PrivateKey:     privKey,
        PublicKey:      pubKey,
        ListenPort:     51821,
        ServerIP:       netip.MustParseAddr("10.99.1.1"),
        VPNNetwork:     netip.MustParsePrefix("10.99.1.0/24"),
        PrivateNetwork: netip.MustParsePrefix("172.21.101.0/24"),
        Interface:      iface,
    }

    // Create server
    srv, err := server.NewServer(config, logger)
    if err != nil {
        logger.Error("Failed to create server", slog.Any("error", err))
        return
    }

    // Start server
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    if err := srv.Start(ctx); err != nil {
        logger.Error("Failed to start server", slog.Any("error", err))
        return
    }

    logger.Info("Server started", slog.String("publicKey", pubKey))

    // Handle signals
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
    <-sigCh

    // Stop server
    srv.Stop()
}
```

### Basic Client

```go
package main

import (
    "context"
    "log/slog"
    "net/netip"
    "os"
    "time"

    "github.com/vastdata/go-vast-client/vastix/vpn_connect/client"
    "github.com/vastdata/go-vast-client/vastix/vpn_connect/common"
)

func main() {
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

    // Generate keys
    privKey, pubKey, _ := common.GenerateKeyPair()

    // Configure client
    config := &common.ClientConfig{
        PrivateKey:      privKey,
        PublicKey:       pubKey,
        ServerPublicKey: "server-public-key-here",
        ServerEndpoint:  "10.27.14.107:51821",
        ClientIP:        netip.MustParseAddr("10.99.1.2"),
        ServerIP:        netip.MustParseAddr("10.99.1.1"),
        PrivateNetwork:  netip.MustParsePrefix("172.21.101.0/24"),
    }

    // Create client
    c, _ := client.NewClient(config, logger)

    // Connect
    ctx := context.Background()
    if err := c.Connect(ctx); err != nil {
        logger.Error("Failed to connect", slog.Any("error", err))
        return
    }
    defer c.Disconnect()

    logger.Info("Connected!", slog.String("publicKey", pubKey))

    // Test connectivity
    latency, err := c.Ping(ctx, "172.21.101.1", 5*time.Second)
    if err == nil {
        logger.Info("Ping successful", slog.Duration("latency", latency))
    }

    // Keep connection alive
    time.Sleep(10 * time.Minute)
}
```

### Multi-Client Deployment

```go
package main

import (
    "context"
    "fmt"
    "log/slog"
    "net/netip"
    "os"

    "github.com/vastdata/go-vast-client/vastix/vpn_connect/client"
    "github.com/vastdata/go-vast-client/vastix/vpn_connect/common"
)

func deployForClient(clientID int, host, user, password string) error {
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    
    // Generate unique network for this client
    vpnNetwork, serverIP, _, err := common.GenerateVPNNetwork(clientID)
    if err != nil {
        return err
    }

    port := common.GetListenPort(clientID)

    // Generate server keys
    privKey, pubKey, _ := common.GenerateKeyPair()

    serverConfig := &common.ServerConfig{
        PrivateKey:     privKey,
        PublicKey:      pubKey,
        ListenPort:     port,
        ServerIP:       serverIP,
        VPNNetwork:     vpnNetwork,
        PrivateNetwork: netip.MustParsePrefix("172.21.101.0/24"),
    }

    // Deploy
    deployer := client.NewDeployer(logger)
    
    deployConfig := &client.DeploymentConfig{
        Host:          host,
        Port:          22,
        Username:      user,
        Password:      password,
        RemoteWorkDir: fmt.Sprintf("/tmp/vpn-server-%d", clientID),
    }

    ctx := context.Background()
    if err := deployer.Connect(ctx, deployConfig); err != nil {
        return err
    }
    defer deployer.Disconnect()

    if err := deployer.Deploy(ctx, deployConfig, serverConfig); err != nil {
        return err
    }

    if err := deployer.StartServer(deployConfig.RemoteWorkDir, port); err != nil {
        return err
    }

    fmt.Printf("Client %d: Server deployed on port %d, public key: %s\n", 
        clientID, port, pubKey)

    return nil
}

func main() {
    // Deploy for 3 different clients
    for i := 1; i <= 3; i++ {
        if err := deployForClient(i, "10.27.14.107", "centos", "password"); err != nil {
            fmt.Printf("Failed to deploy client %d: %v\n", i, err)
        }
    }
}
```

## Error Handling

All functions return standard Go errors. Use `errors.Is()` or `errors.As()` for error checking:

```go
if err := srv.Start(ctx); err != nil {
    if vpnErr, ok := err.(*common.VPNError); ok {
        log.Printf("VPN operation %s failed: %v", vpnErr.Operation, vpnErr.Err)
    } else {
        log.Printf("Unknown error: %v", err)
    }
}
```

## Thread Safety

- `Server` methods are thread-safe and can be called from multiple goroutines
- `Client` methods are thread-safe
- `Deployer` is NOT thread-safe (one operation at a time)

## Resource Cleanup

Always ensure proper cleanup:

```go
// Server
defer srv.Stop()

// Client
defer client.Disconnect()

// Deployer
defer deployer.Disconnect()
```

