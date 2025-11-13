package main

import (
	"context"
	"fmt"
	"log"
	"net/netip"
	"os"
	"os/exec"

	"vastix/internal/vpn_connect/client"
	"vastix/internal/vpn_connect/common"
)

func main() {
	fmt.Println("=================================================================")
	fmt.Println("VPN Connect End-to-End Test")
	fmt.Println("=================================================================")

	// Get local hostname for remote directory
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Failed to get hostname: %v", err)
	}

	// Remote directory structure: /tmp/vastix_vpn/<local_hostname>
	remoteWorkDir := fmt.Sprintf("/tmp/vastix_vpn/%s", hostname)

	fmt.Printf("Local hostname: %s\n", hostname)
	fmt.Printf("Remote work directory: %s\n", remoteWorkDir)
	fmt.Println()

	// Configuration
	remoteHost := "10.27.14.107"
	remoteUser := "centos"
	remotePassword := ""                                    // Set your password here or use key
	remoteKeyPath := os.Getenv("HOME") + "/.ssh/id_ed25519" // Using SSH key
	clientID := 1

	if remotePassword == "" && remoteKeyPath == "" {
		log.Fatal("Please set either remotePassword or remoteKeyPath in the test script")
	}

	ctx := context.Background()

	// Step 1: Generate network configuration
	fmt.Println("Step 1: Generating network configuration...")
	vpnNetwork, serverIP, clientIP, err := common.GenerateVPNNetwork(clientID)
	if err != nil {
		log.Fatalf("Failed to generate VPN network: %v", err)
	}

	port := common.GetListenPort(clientID)
	privateNetwork := netip.MustParsePrefix("172.21.101.0/24")

	fmt.Printf("  VPN Network: %s\n", vpnNetwork)
	fmt.Printf("  Server IP: %s\n", serverIP)
	fmt.Printf("  Client IP: %s\n", clientIP)
	fmt.Printf("  Port: %d\n", port)
	fmt.Printf("  Private Network: %s\n", privateNetwork)
	fmt.Println()

	// Step 2: Create deployer and connect via SSH
	fmt.Println("Step 2: Connecting to remote host via SSH...")
	deployer := client.NewDeployer(nil, nil)

	deployConfig := &client.DeploymentConfig{
		Host:           remoteHost,
		Port:           22,
		Username:       remoteUser,
		Password:       remotePassword,
		PrivateKeyPath: remoteKeyPath,
		RemoteWorkDir:  remoteWorkDir,
	}

	if err := deployer.Connect(ctx, deployConfig); err != nil {
		log.Fatalf("Failed to connect to remote host: %v", err)
	}
	defer deployer.Disconnect()

	fmt.Println("  ✓ SSH connection established")
	fmt.Println()

	// Step 3: Generate server keys
	fmt.Println("Step 3: Generating WireGuard keys...")
	serverPrivKey, serverPubKey, err := common.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate server keys: %v", err)
	}

	fmt.Printf("  Server Public Key: %s\n", serverPubKey)
	fmt.Println()

	// Step 4: Deploy server
	fmt.Println("Step 4: Deploying server binary to remote...")
	serverConfig := &common.ServerConfig{
		PrivateKey:     serverPrivKey,
		PublicKey:      serverPubKey,
		ListenPort:     port,
		ServerIP:       serverIP,
		VPNNetwork:     vpnNetwork,
		PrivateNetwork: privateNetwork,
	}

	if err := deployer.Deploy(ctx, deployConfig, serverConfig); err != nil {
		log.Fatalf("Failed to deploy server: %v", err)
	}

	fmt.Println("  ✓ Server deployed successfully")
	fmt.Println()

	// Step 5: Start server
	fmt.Println("Step 5: Starting VPN server on remote...")
	if err := deployer.StartServer(remoteWorkDir, serverConfig); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	fmt.Println("  ✓ Server started successfully")
	fmt.Println()

	// Step 6: Generate client keys
	fmt.Println("Step 6: Generating client keys...")
	clientPrivKey, clientPubKey, err := common.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate client keys: %v", err)
	}

	fmt.Printf("  Client Public Key: %s\n", clientPubKey)
	fmt.Println()

	// Step 7: Create and connect VPN client
	fmt.Println("Step 7: Connecting VPN client...")
	clientConfig := &common.ClientConfig{
		PrivateKey:      clientPrivKey,
		PublicKey:       clientPubKey,
		ServerPublicKey: serverPubKey,
		ServerEndpoint:  fmt.Sprintf("%s:%d", remoteHost, port),
		ClientIP:        clientIP,
		ServerIP:        serverIP,
		PrivateNetwork:  privateNetwork,
	}

	vpnClient, err := client.NewClient(clientConfig, nil, nil)
	if err != nil {
		log.Fatalf("Failed to create VPN client: %v", err)
	}

	if err := vpnClient.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect VPN client: %v", err)
	}
	defer vpnClient.Disconnect("") // Empty password for passwordless sudo in tests

	fmt.Println("  ✓ VPN client connected successfully")
	fmt.Println()

	// Step 8: Test connectivity
	fmt.Println("Step 8: Testing connectivity...")

	// Test VPN server IP
	fmt.Printf("  Testing ping to VPN server (%s)...\n", serverIP)
	if err := testPing(serverIP.String()); err != nil {
		fmt.Printf("    ✗ Ping failed: %v\n", err)
	} else {
		fmt.Println("    ✓ Ping successful")
	}

	// Test private network
	targetIP := "172.21.101.1"
	fmt.Printf("  Testing ping to private network (%s)...\n", targetIP)
	if err := testPing(targetIP); err != nil {
		fmt.Printf("    ✗ Ping failed: %v\n", err)
	} else {
		fmt.Println("    ✓ Ping successful!")
	}

	fmt.Println()

	// Success!
	fmt.Println("=================================================================")
	fmt.Println("✅ VPN Connection Test SUCCESSFUL!")
	fmt.Println("=================================================================")
	fmt.Printf("Server endpoint: %s:%d\n", remoteHost, port)
	fmt.Printf("VPN Network: %s\n", vpnNetwork)
	fmt.Printf("Client IP: %s\n", clientIP)
	fmt.Printf("Private Network: %s (accessible!)\n", privateNetwork)
	fmt.Println()
	fmt.Println("Press Ctrl+C to disconnect and exit...")

	// Wait for interrupt
	select {}
}

func testPing(ip string) error {
	cmd := exec.Command("ping", "-c", "3", "-W", "2", ip)
	return cmd.Run()
}
