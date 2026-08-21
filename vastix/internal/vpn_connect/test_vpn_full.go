//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"net/netip"
	"os"
	"os/exec"
	"time"

	"vastix/internal/vpn_connect/client"
	"vastix/internal/vpn_connect/common"
)

func main() {
	fmt.Println("=================================================================")
	fmt.Println("VPN Connect FULL End-to-End Test (with WireGuard client)")
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
	remoteKeyPath := os.Getenv("HOME") + "/.ssh/id_ed25519"
	clientID := 1
	sudoPassword := os.Getenv("VASTIX_SUDO_PASSWORD")

	ctx := context.Background()

	// Step 1: Generate network configuration
	fmt.Println("Step 1: Generating network configuration...")
	vpnNetwork, serverIP, clientIP, err := common.GenerateVPNNetwork(clientID)
	if err != nil {
		log.Fatalf("Failed to generate VPN network: %v", err)
	}

	port := common.GetListenPort(clientID)
	privateIPs := []netip.Addr{netip.MustParseAddr("172.21.101.1")}
	privateNetwork := netip.MustParsePrefix("172.21.101.0/24")

	fmt.Printf("  VPN Network: %s\n", vpnNetwork)
	fmt.Printf("  Server IP: %s\n", serverIP)
	fmt.Printf("  Client IP: %s\n", clientIP)
	fmt.Printf("  Port: %d\n", port)
	fmt.Printf("  Private Network: %s\n", privateNetwork)
	fmt.Println()

	// Step 2: Deploy server
	fmt.Println("Step 2: Deploying VPN server...")
	deployer := client.NewDeployer(os.Stdout)

	deployConfig := &client.DeploymentConfig{
		Host:           remoteHost,
		Port:           22,
		Username:       remoteUser,
		PrivateKeyPath: remoteKeyPath,
		RemoteWorkDir:  remoteWorkDir,
	}

	if err := deployer.Connect(ctx, deployConfig); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer deployer.Disconnect()

	serverPrivKey, serverPubKey, _ := common.GenerateKeyPair()

	serverConfig := &common.ServerConfig{
		PrivateKey:     serverPrivKey,
		PublicKey:      serverPubKey,
		ListenPort:     port,
		ServerIP:       serverIP,
		VPNNetwork:     vpnNetwork,
		PrivateIPs: privateIPs,
	}

	if err := deployer.Deploy(ctx, deployConfig, serverConfig); err != nil {
		log.Fatalf("Failed to deploy: %v", err)
	}

	go func() {
		if err := deployer.StartServer(ctx, remoteWorkDir, serverConfig); err != nil {
			log.Printf("VPN server stopped: %v", err)
		}
	}()
	time.Sleep(3 * time.Second)

	fmt.Println("  ✓ Server running")
	fmt.Printf("  Server Public Key: %s\n", serverPubKey)
	fmt.Println()

	// Step 3: Setup client-side WireGuard
	fmt.Println("Step 3: Setting up local WireGuard client...")

	clientPrivKey, clientPubKey, _ := common.GenerateKeyPair()
	fmt.Printf("  Client Public Key: %s\n", clientPubKey)

	// Create WireGuard config
	wgConfig := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s/24

[Peer]
PublicKey = %s
Endpoint = %s:%d
AllowedIPs = %s, %s
PersistentKeepalive = 25
`, clientPrivKey, clientIP, serverPubKey, remoteHost, port, vpnNetwork, privateNetwork)

	if err := os.WriteFile("/tmp/wg_vastix.conf", []byte(wgConfig), 0600); err != nil { // #nosec G303 -- manual VPN lab helper. nosemgrep: go.lang.security.audit.io.bad-tmp-file-creation.bad-tmp-file-creation
		log.Fatalf("Failed to write config: %v", err)
	}

	// Bring up WireGuard interface
	fmt.Println("  Bringing up WireGuard interface (wg_vastix)...")
	cmd := exec.Command("bash", "-c", fmt.Sprintf("echo '%s' | sudo -S wg-quick up /tmp/wg_vastix.conf", sudoPassword)) // #nosec G204 -- manual VPN lab helper
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("wg-quick output: %s", output)
		log.Fatalf("Failed to bring up interface: %v", err)
	}

	defer func() {
		fmt.Println("\nCleaning up...")
		cmd := exec.Command("bash", "-c", fmt.Sprintf("echo '%s' | sudo -S wg-quick down /tmp/wg_vastix.conf 2>/dev/null", sudoPassword)) // #nosec G204 -- manual VPN lab helper
		cmd.Run() // #nosec G104 -- best-effort teardown in defer
	}()

	fmt.Println("  ✓ WireGuard interface up")
	fmt.Println()

	// Wait for connection to establish
	time.Sleep(3 * time.Second)

	// Step 4: Test connectivity
	fmt.Println("Step 4: Testing connectivity...")

	fmt.Printf("  Ping VPN server (%s)...\n", serverIP)
	cmd = exec.Command("ping", "-c", "3", "-W", "2", serverIP.String()) // #nosec G204 -- manual VPN lab helper; dest is a typed IP
	if err := cmd.Run(); err != nil {
		fmt.Println("    ✗ Ping failed")
	} else {
		fmt.Println("    ✓ Ping successful")
	}

	fmt.Printf("  Ping private network (172.21.101.1)...\n")
	cmd = exec.Command("ping", "-c", "3", "-W", "2", "172.21.101.1") // #nosec G204 -- manual VPN lab helper; fixed lab IP
	if err := cmd.Run(); err != nil {
		fmt.Println("    ✗ Ping failed")
	} else {
		fmt.Println("    ✓ Ping successful!")
	}
	fmt.Println()

	// Step 5: Mount NFS share
	fmt.Println("Step 5: Mounting NFS share 172.21.101.1:/test...")

	mountPoint := "/tmp/vastix_nfs_test"
	os.MkdirAll(mountPoint, 0755) // #nosec G301,G104 -- manual VPN lab helper; temp NFS mount point

	cmd = exec.Command("bash", "-c", fmt.Sprintf("echo '%s' | sudo -S mount -t nfs 172.21.101.1:/test %s", sudoPassword, mountPoint)) // #nosec G204 -- manual VPN lab helper
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("  ✗ Mount failed: %v\n", err)
		fmt.Printf("  Output: %s\n", output)
	} else {
		fmt.Println("  ✓ NFS share mounted")

		defer func() {
			cmd := exec.Command("bash", "-c", fmt.Sprintf("echo '%s' | sudo -S umount %s 2>/dev/null", sudoPassword, mountPoint)) // #nosec G204 -- manual VPN lab helper
			cmd.Run() // #nosec G104 -- best-effort umount in defer
		}()

		// List contents
		fmt.Println("\n  Contents of NFS share:")
		cmd = exec.Command("ls", "-la", mountPoint) // #nosec G204 -- manual VPN lab helper; lists the NFS mount
		if output, err := cmd.CombinedOutput(); err == nil {
			fmt.Printf("%s\n", output)
		}

		// Check for text.txt
		textFile := mountPoint + "/text.txt"
		if content, err := os.ReadFile(textFile); err == nil {
			fmt.Printf("  ✓ Found text.txt!\n")
			fmt.Printf("  Content: %s\n", string(content))
		} else {
			fmt.Printf("  text.txt not found: %v\n", err)
		}
	}

	fmt.Println("\n=================================================================")
	fmt.Println("✅ VPN + NFS TEST COMPLETE!")
	fmt.Println("=================================================================")
}
