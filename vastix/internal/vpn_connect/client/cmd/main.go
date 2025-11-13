// VPN Client - Standalone executable and deployment tool
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"vastix/internal/vpn_connect/client"
	"vastix/internal/vpn_connect/common"
)

func main() {
	// Command modes
	mode := flag.String("mode", "connect", "Mode: connect, deploy, start-remote, stop-remote, status-remote")

	// Connection flags
	serverEndpoint := flag.String("server", "", "Server endpoint (host:port)")
	serverPublicKey := flag.String("server-key", "", "Server public key")
	clientIP := flag.String("client-ip", "", "Client VPN IP (e.g., 10.99.1.2)")
	serverIP := flag.String("server-ip", "", "Server VPN IP (e.g., 10.99.1.1)")
	privateNetwork := flag.String("private-network", "172.21.101.0/24", "Private network CIDR to access")
	privateKey := flag.String("private-key", "", "Client private key (generate if empty)")

	// Deployment flags
	remoteHost := flag.String("remote-host", "", "Remote host for deployment")
	remotePort := flag.Int("remote-port", 22, "Remote SSH port")
	remoteUser := flag.String("remote-user", "centos", "Remote SSH username")
	remotePassword := flag.String("remote-password", "", "Remote SSH password")
	remoteKeyFile := flag.String("remote-key", "", "Remote SSH private key file")
	remoteWorkDir := flag.String("remote-dir", "/tmp/vpn-server", "Remote working directory")
	vpnPort := flag.Uint("vpn-port", 51820, "VPN server port")
	vpnServerIP := flag.String("vpn-server-ip", "", "VPN server IP for remote deployment")
	vpnNetwork := flag.String("vpn-network", "", "VPN network CIDR for remote deployment")

	verbose := flag.Bool("verbose", false, "Enable verbose logging")

	flag.Parse()

	// Set up logger
	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	switch *mode {
	case "connect":
		runConnectMode(logger, *serverEndpoint, *serverPublicKey, *clientIP, *serverIP, *privateNetwork, *privateKey)

	case "deploy":
		runDeployMode(logger, *remoteHost, *remotePort, *remoteUser, *remotePassword, *remoteKeyFile, *remoteWorkDir, *vpnPort, *vpnServerIP, *vpnNetwork, *privateNetwork)

	case "start-remote":
		runStartRemoteMode(logger, *remoteHost, *remotePort, *remoteUser, *remotePassword, *remoteKeyFile, *remoteWorkDir, *vpnPort)

	case "stop-remote":
		runStopRemoteMode(logger, *remoteHost, *remotePort, *remoteUser, *remotePassword, *remoteKeyFile, *vpnPort)

	case "status-remote":
		runStatusRemoteMode(logger, *remoteHost, *remotePort, *remoteUser, *remotePassword, *remoteKeyFile, *vpnPort)

	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s\n", *mode)
		flag.Usage()
		os.Exit(1)
	}
}

func runConnectMode(logger *slog.Logger, serverEndpoint, serverPublicKey, clientIPStr, serverIPStr, privateNetworkStr, privateKey string) {
	if serverEndpoint == "" || serverPublicKey == "" || clientIPStr == "" || serverIPStr == "" {
		logger.Error("Missing required flags for connect mode")
		fmt.Fprintf(os.Stderr, "Required flags: -server, -server-key, -client-ip, -server-ip\n")
		os.Exit(1)
	}

	// Generate keys if not provided
	var privKey, pubKey string
	var err error

	if privateKey == "" {
		privKey, pubKey, err = common.GenerateKeyPair()
		if err != nil {
			logger.Error("Failed to generate key pair", slog.Any("error", err))
			os.Exit(1)
		}
		logger.Info("Generated new key pair", slog.String("publicKey", pubKey))
	} else {
		privKey = privateKey
		pubKey, err = common.GetPublicKey(privKey)
		if err != nil {
			logger.Error("Failed to derive public key", slog.Any("error", err))
			os.Exit(1)
		}
	}

	// Parse IPs
	clientIP, err := netip.ParseAddr(clientIPStr)
	if err != nil {
		logger.Error("Invalid client IP", slog.Any("error", err))
		os.Exit(1)
	}

	serverIP, err := netip.ParseAddr(serverIPStr)
	if err != nil {
		logger.Error("Invalid server IP", slog.Any("error", err))
		os.Exit(1)
	}

	privNet, err := netip.ParsePrefix(privateNetworkStr)
	if err != nil {
		logger.Error("Invalid private network", slog.Any("error", err))
		os.Exit(1)
	}

	// Create client configuration
	config := &common.ClientConfig{
		PrivateKey:      privKey,
		PublicKey:       pubKey,
		ServerPublicKey: serverPublicKey,
		ServerEndpoint:  serverEndpoint,
		ClientIP:        clientIP,
		ServerIP:        serverIP,
		PrivateNetwork:  privNet,
	}

	// Create client
	c, err := client.NewClient(config, nil, nil)
	if err != nil {
		logger.Error("Failed to create client", slog.Any("error", err))
		os.Exit(1)
	}

	// Set up signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		logger.Info("Received shutdown signal")
		cancel()
	}()

	// Connect
	logger.Info("Connecting to VPN server", slog.String("endpoint", serverEndpoint))

	if err := c.Connect(ctx); err != nil {
		logger.Error("Failed to connect", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("Connected successfully!")
	fmt.Println("\nClient Public Key:", pubKey)
	fmt.Println("(Provide this to the server administrator to authorize your connection)\n")

	// Wait for shutdown
	<-ctx.Done()

	// Disconnect
	if err := c.Disconnect(); err != nil {
		logger.Error("Failed to disconnect", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("Disconnected gracefully")
}

func runDeployMode(logger *slog.Logger, host string, port int, user, password, keyFile, workDir string, vpnPort uint, vpnServerIP, vpnNetwork, privateNetwork string) {
	if host == "" {
		logger.Error("Remote host is required for deployment")
		os.Exit(1)
	}

	if vpnServerIP == "" || vpnNetwork == "" {
		logger.Error("VPN server IP and network are required for deployment")
		fmt.Fprintf(os.Stderr, "Required flags: -vpn-server-ip, -vpn-network\n")
		os.Exit(1)
	}

	// Parse network configs
	srvIP, err := netip.ParseAddr(vpnServerIP)
	if err != nil {
		logger.Error("Invalid VPN server IP", slog.Any("error", err))
		os.Exit(1)
	}

	vpnNet, err := netip.ParsePrefix(vpnNetwork)
	if err != nil {
		logger.Error("Invalid VPN network", slog.Any("error", err))
		os.Exit(1)
	}

	privNet, err := netip.ParsePrefix(privateNetwork)
	if err != nil {
		logger.Error("Invalid private network", slog.Any("error", err))
		os.Exit(1)
	}

	// Generate server keys
	privKey, pubKey, err := common.GenerateKeyPair()
	if err != nil {
		logger.Error("Failed to generate keys", slog.Any("error", err))
		os.Exit(1)
	}

	serverConfig := &common.ServerConfig{
		PrivateKey:     privKey,
		PublicKey:      pubKey,
		ListenPort:     uint16(vpnPort),
		ServerIP:       srvIP,
		VPNNetwork:     vpnNet,
		PrivateNetwork: privNet,
		Interface:      "", // Will be auto-detected on remote
	}

	// Create deployer
	deployer := client.NewDeployer(nil, nil)

	deployConfig := &client.DeploymentConfig{
		Host:           host,
		Port:           port,
		Username:       user,
		Password:       password,
		PrivateKeyPath: keyFile,
		RemoteWorkDir:  workDir,
	}

	// Connect
	ctx := context.Background()
	if err := deployer.Connect(ctx, deployConfig); err != nil {
		logger.Error("Failed to connect to remote host", slog.Any("error", err))
		os.Exit(1)
	}
	defer deployer.Disconnect()

	// Deploy
	logger.Info("Deploying VPN server to remote host")

	if err := deployer.Deploy(ctx, deployConfig, serverConfig); err != nil {
		logger.Error("Deployment failed", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("Deployment completed successfully!")
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("VPN Server Deployed Successfully!")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Remote Host:       %s\n", host)
	fmt.Printf("Server Public Key: %s\n", pubKey)
	fmt.Printf("Server VPN IP:     %s\n", vpnServerIP)
	fmt.Printf("VPN Port:          %d\n", vpnPort)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("\nTo start the server, run:")
	fmt.Printf("  %s -mode start-remote -remote-host %s -vpn-port %d\n", os.Args[0], host, vpnPort)
	fmt.Println()
}

func runStartRemoteMode(logger *slog.Logger, host string, port int, user, password, keyFile, workDir string, vpnPort uint) {
	if host == "" {
		logger.Error("Remote host is required")
		os.Exit(1)
	}

	deployer := client.NewDeployer(nil, nil)

	deployConfig := &client.DeploymentConfig{
		Host:           host,
		Port:           port,
		Username:       user,
		Password:       password,
		PrivateKeyPath: keyFile,
		RemoteWorkDir:  workDir,
	}

	ctx := context.Background()
	if err := deployer.Connect(ctx, deployConfig); err != nil {
		logger.Error("Failed to connect to remote host", slog.Any("error", err))
		os.Exit(1)
	}
	defer deployer.Disconnect()

	if err := deployer.StartServer(workDir, uint16(vpnPort)); err != nil {
		logger.Error("Failed to start remote server", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("Remote server started successfully")
}

func runStopRemoteMode(logger *slog.Logger, host string, port int, user, password, keyFile string, vpnPort uint) {
	if host == "" {
		logger.Error("Remote host is required")
		os.Exit(1)
	}

	deployer := client.NewDeployer(nil, nil)

	deployConfig := &client.DeploymentConfig{
		Host:           host,
		Port:           port,
		Username:       user,
		Password:       password,
		PrivateKeyPath: keyFile,
	}

	ctx := context.Background()
	if err := deployer.Connect(ctx, deployConfig); err != nil {
		logger.Error("Failed to connect to remote host", slog.Any("error", err))
		os.Exit(1)
	}
	defer deployer.Disconnect()

	if err := deployer.StopServer(uint16(vpnPort)); err != nil {
		logger.Error("Failed to stop remote server", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("Remote server stopped successfully")
}

func runStatusRemoteMode(logger *slog.Logger, host string, port int, user, password, keyFile string, vpnPort uint) {
	if host == "" {
		logger.Error("Remote host is required")
		os.Exit(1)
	}

	deployer := client.NewDeployer(nil, nil)

	deployConfig := &client.DeploymentConfig{
		Host:           host,
		Port:           port,
		Username:       user,
		Password:       password,
		PrivateKeyPath: keyFile,
	}

	ctx := context.Background()
	if err := deployer.Connect(ctx, deployConfig); err != nil {
		logger.Error("Failed to connect to remote host", slog.Any("error", err))
		os.Exit(1)
	}
	defer deployer.Disconnect()

	running, pid, err := deployer.GetServerStatus(uint16(vpnPort))
	if err != nil {
		logger.Error("Failed to check server status", slog.Any("error", err))
		os.Exit(1)
	}

	if running {
		fmt.Printf("✓ VPN Server is running (PID: %s, Port: %d)\n", pid, vpnPort)
	} else {
		fmt.Printf("✗ VPN Server is not running (Port: %d)\n", vpnPort)
	}
}
