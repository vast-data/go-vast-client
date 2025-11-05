// Package client provides remote server deployment functionality
package client

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"math/rand"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"vastix/internal/vpn_connect/common"

	"golang.org/x/crypto/ssh"
)

// Embedded pre-compiled VPN server binaries for different platforms
// These are built by scripts/build-vpn-binaries.sh and placed in client/binaries/

//go:embed binaries/vpn-server-linux-amd64
var vpnServerLinuxAmd64 []byte

//go:embed binaries/vpn-server-linux-arm64
var vpnServerLinuxArm64 []byte

//go:embed binaries/vpn-server-darwin-amd64
var vpnServerDarwinAmd64 []byte

//go:embed binaries/vpn-server-darwin-arm64
var vpnServerDarwinArm64 []byte

// Deployer handles deployment of VPN server to remote machines
type Deployer struct {
	writer          io.Writer // Writer for streaming logs to UI (includes multiwriter with auxlog)
	sshClient       *ssh.Client
	sshConfig       *ssh.ClientConfig
	sshHost         string             // SSH host for error messages
	sshUser         string             // SSH user for error messages
	heartbeatCancel context.CancelFunc // Cancel function for heartbeat goroutine
	vipPoolIPs      []netip.Addr       // VIP pool IPs to check during health monitoring
}

// DeploymentConfig contains configuration for remote deployment
type DeploymentConfig struct {
	Host             string
	Port             int
	Username         string
	Password         string
	PrivateKeyPath   string
	ServerBinaryPath string // Local path to server binary
	RemoteWorkDir    string // Remote directory for VPN server
}

// NewDeployer creates a new deployer instance
func NewDeployer(writer io.Writer) *Deployer {
	// Writer must never be nil - this is a programming error
	if writer == nil {
		panic("BUG: writer cannot be nil in NewDeployer - this indicates improper initialization")
	}

	return &Deployer{
		writer: writer,
	}
}

// Connect establishes SSH connection to remote host
func (d *Deployer) Connect(ctx context.Context, config *DeploymentConfig) error {
	var authMethods []ssh.AuthMethod

	// Add password authentication if provided
	if config.Password != "" {
		authMethods = append(authMethods, ssh.Password(config.Password))
	}

	// Add public key authentication if provided
	if config.PrivateKeyPath != "" {
		key, err := os.ReadFile(config.PrivateKeyPath)
		if err != nil {
			return fmt.Errorf("failed to read private key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	if len(authMethods) == 0 {
		return fmt.Errorf("no authentication method provided")
	}

	d.sshConfig = &ssh.ClientConfig{
		User:            config.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	d.writef("Connecting to remote host: %s\n", addr)

	client, err := ssh.Dial("tcp", addr, d.sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect via SSH: %w", err)
	}

	d.sshClient = client
	d.sshHost = config.Host
	d.sshUser = config.Username
	d.writef("SSH connection established\n")

	return nil
}

// Disconnect closes the SSH connection
func (d *Deployer) Disconnect() error {
	// Stop heartbeat if running
	if d.heartbeatCancel != nil {
		d.heartbeatCancel()
		d.heartbeatCancel = nil
	}

	if d.sshClient != nil {
		return d.sshClient.Close()
	}
	return nil
}

// Deploy deploys the VPN server to the remote machine
func (d *Deployer) Deploy(ctx context.Context, config *DeploymentConfig, serverConfig *common.ServerConfig) error {
	if d.sshClient == nil {
		return fmt.Errorf("not connected to remote host")
	}

	d.writef("=== Starting Deployment ===\n")
	d.writef("Starting deployment to remote host\n")

	// Step 0: Ensure WireGuard is available (check or auto-install)
	if err := d.ensureWireGuard(ctx); err != nil {
		return fmt.Errorf("WireGuard setup failed: %w", err)
	}

	// Step 1: Kill any existing vpn-server processes to release the binary file lock
	d.writef("Checking for existing VPN server processes...\n")
	checkCmd := "pgrep -x vpn-server"
	if output, _ := d.runCommandWithOutput(checkCmd); strings.TrimSpace(output) != "" {
		d.writef("Found running server process, stopping it gracefully...\n")

		// Step 1: Send SIGTERM for graceful shutdown
		termCmd := "sh -c 'sudo pkill -TERM -x vpn-server 2>/dev/null; exit 0'"
		d.runCommand(termCmd)

		// Step 2: Wait up to 3 seconds for graceful shutdown
		stopped := false
		for i := 0; i < 6; i++ {
			time.Sleep(500 * time.Millisecond)
			if output, _ := d.runCommandWithOutput(checkCmd); strings.TrimSpace(output) == "" {
				stopped = true
				break
			}
		}

		// Step 3: Force kill if still running
		if !stopped {
			d.writef("Server did not stop gracefully, force-killing...\n")
			killCmd := "sh -c 'sudo pkill -9 -x vpn-server 2>/dev/null; exit 0'"
			d.runCommand(killCmd)
			time.Sleep(500 * time.Millisecond)
		}

		d.writef("Old server processes stopped\n")
	} else {
		d.writef("No existing server processes found\n")
	}

	// Create remote working directory
	if err := d.runCommand(fmt.Sprintf("mkdir -p %s", config.RemoteWorkDir)); err != nil {
		return fmt.Errorf("failed to create remote directory: %w", err)
	}

	// Build server binary if not provided
	localBinary := config.ServerBinaryPath
	if localBinary == "" {
		d.writef("Building VPN server binary\n")
		var err error
		localBinary, err = d.buildServerBinary(ctx)
		if err != nil {
			return fmt.Errorf("failed to build server binary: %w", err)
		}
		defer os.Remove(localBinary)
	}

	// Upload server binary
	remoteBinary := filepath.Join(config.RemoteWorkDir, "vpn-server")
	if err := d.uploadFile(localBinary, remoteBinary); err != nil {
		return fmt.Errorf("failed to upload server binary: %w", err)
	}

	// Make binary executable
	if err := d.runCommand(fmt.Sprintf("chmod +x %s", remoteBinary)); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Generate server configuration file
	configFile := d.generateServerConfig(serverConfig)
	remoteConfigPath := filepath.Join(config.RemoteWorkDir, "server-config.yaml")
	if err := d.uploadContent(configFile, remoteConfigPath); err != nil {
		return fmt.Errorf("failed to upload config file: %w", err)
	}

	d.writef("Deployment completed successfully\n  Binary: %s\n  Config: %s\n",
		remoteBinary, remoteConfigPath)

	return nil
}

// StartServer starts the VPN server on the remote machine
// StartServer starts the VPN server in foreground mode with stdout/stderr streaming.
// This is a blocking call - run it in a goroutine. The server will stop when ctx is cancelled.
func (d *Deployer) StartServer(ctx context.Context, workDir string, config *common.ServerConfig) error {
	if d.sshClient == nil {
		return fmt.Errorf("not connected to remote host")
	}

	d.writef("Starting VPN server on remote host (port: %d)\n", config.ListenPort)

	// Pre-flight checks
	remoteBinary := filepath.Join(workDir, "vpn-server")

	// Check 1: Test sudo access (without password)
	d.writef("Checking sudo access...\n")
	if err := d.runCommand("sudo -n true 2>/dev/null"); err != nil {
		return fmt.Errorf(`sudo requires password authentication.

Please configure passwordless sudo for your user on the remote server:
  1. SSH to the remote: ssh %s
  2. Edit sudoers: sudo visudo
  3. Add this line: %s ALL=(ALL) NOPASSWD: ALL

Or configure sudo for specific commands only.
After configuring, try connecting again.`, d.sshHost, d.sshUser)
	}
	d.writef("✓ Sudo access confirmed\n")

	// Check 2: Verify binary is executable
	d.writef("Verifying VPN server binary...\n")
	if err := d.runCommand(fmt.Sprintf("test -x %s", remoteBinary)); err != nil {
		return fmt.Errorf("VPN server binary is not executable: %s", remoteBinary)
	}
	d.writef("✓ Binary is executable\n")

	// Create SSH session for running the server
	session, err := d.sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Note: VPN server needs root privileges to create TUN devices
	// We preserve the PATH environment variable so wireguard-go can be found in /usr/local/bin
	// remoteBinary already defined above in pre-flight checks

	// Build private IPs string (comma-separated)
	var privateIPsStr string
	if len(config.PrivateIPs) > 0 {
		ipStrs := make([]string, len(config.PrivateIPs))
		for i, ip := range config.PrivateIPs {
			ipStrs[i] = ip.String()
		}
		privateIPsStr = strings.Join(ipStrs, ",")
	}

	// Log file path (next to the binary)
	logFile := filepath.Join(workDir, "server.log")

	// Heartbeat file path (next to the binary)
	heartbeatFile := filepath.Join(workDir, "heartbeat")

	// Run server in foreground using setsid to create a new session
	// This ensures the process dies when the SSH session closes
	// setsid creates a new session and detaches from the controlling terminal
	startCmd := fmt.Sprintf("setsid sudo env PATH=$PATH %s -port %d -server-ip %s -vpn-network %s -private-ips '%s' -private-key '%s' -log-file '%s' -heartbeat-file '%s'",
		remoteBinary,
		config.ListenPort,
		config.ServerIP.String(),
		config.VPNNetwork.String(),
		privateIPsStr,
		config.PrivateKey,
		logFile,
		heartbeatFile)

	// Connect stdout/stderr to writer for real-time log streaming
	session.Stdout = d.writer
	session.Stderr = d.writer

	d.writef("Starting server in foreground mode (will run until cancelled)...\n")
	d.writef("Server log file: %s\n", logFile)
	d.writef("Command: %s\n\n", startCmd)
	d.writef("--- Server Output (streaming to TUI) ---\n")

	// Start the command (blocking) - will run until context is cancelled or command exits
	errChan := make(chan error, 1)
	go func() {
		errChan <- session.Run(startCmd)
	}()

	// Wait for either context cancellation or command completion
	select {
	case <-ctx.Done():
		d.writef("\n--- Server Context Cancelled, initiating graceful shutdown ---\n")

		// Step 1: Send SIGTERM for graceful shutdown (allows cleanup to run)
		d.writef("Sending SIGTERM to vpn-server for graceful cleanup...\n")
		termCmd := "sh -c 'sudo pkill -TERM -x vpn-server 2>/dev/null; exit 0'"
		if termErr := d.runCommand(termCmd); termErr != nil {
			d.writef("Warning: Failed to send SIGTERM: %v\n", termErr)
		}

		// Step 2: Wait up to 5 seconds for graceful shutdown
		d.writef("Waiting for server to clean up (max 5 seconds)...\n")
		for i := 0; i < 10; i++ {
			time.Sleep(500 * time.Millisecond)
			// Check if process is still running
			checkCmd := "pgrep -x vpn-server"
			if output, _ := d.runCommandWithOutput(checkCmd); strings.TrimSpace(output) == "" {
				d.writef("Server stopped gracefully\n")
				return nil
			}
		}

		// Step 3: If still running after 5 seconds, force kill
		d.writef("Server did not stop gracefully, sending SIGKILL...\n")
		killCmd := "sh -c 'sudo pkill -9 -x vpn-server 2>/dev/null; exit 0'"
		if killErr := d.runCommand(killCmd); killErr != nil {
			d.writef("Warning: Failed to kill remote server: %v\n", killErr)
		} else {
			d.writef("Remote vpn-server process force-killed\n")
		}

		// Close session
		session.Close()
		return fmt.Errorf("server stopped: %w", ctx.Err())
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("server exited with error: %w", err)
		}
		d.writef("\n--- Server Exited Normally ---\n")
		return nil
	}
}

// RegisterPeer registers a client peer on the remote VPN server
func (d *Deployer) RegisterPeer(clientPublicKey string, clientIP string, port uint16) error {
	if d.sshClient == nil {
		return fmt.Errorf("not connected to remote host")
	}

	// Calculate interface name from port: wg + (port % 100)
	// This matches the server's naming scheme
	// e.g., port 51821 -> wg21, port 51822 -> wg22, etc.
	interfaceName := fmt.Sprintf("wg%d", port%100)

	d.writef("Registering client peer on server\n  Public Key: %s\n  Client IP: %s\n  Interface: %s\n",
		clientPublicKey, clientIP, interfaceName)

	// Add peer to WireGuard interface
	cmd := fmt.Sprintf("sudo wg set %s peer %s allowed-ips %s/32", interfaceName, clientPublicKey, clientIP)

	if err := d.runCommand(cmd); err != nil {
		return fmt.Errorf("failed to register peer: %w", err)
	}

	d.writef("Client peer registered successfully\n")

	return nil
}

// GetServerStatus checks if the server is running
func (d *Deployer) GetServerStatus(port uint16) (bool, string, error) {
	if d.sshClient == nil {
		return false, "", fmt.Errorf("not connected to remote host")
	}

	// Use -x for exact match on process name to avoid matching wrapper processes
	checkCmd := "pgrep -x vpn-server"
	output, err := d.runCommandWithOutput(checkCmd)

	pid := strings.TrimSpace(output)
	if pid == "" || err != nil {
		return false, "", nil
	}

	// Take only the first PID if multiple matched
	pids := strings.Split(pid, "\n")
	return true, pids[0], nil
}

// IsPortInUse checks if a UDP port is in use on the remote host
func (d *Deployer) IsPortInUse(port uint16) (bool, error) {
	if d.sshClient == nil {
		return false, fmt.Errorf("not connected to remote host")
	}

	// Check if port is listening using ss command (more reliable than netstat)
	checkCmd := fmt.Sprintf("ss -ulnH | grep -q ':%d ' && echo 'in-use' || echo 'available'", port)
	output, err := d.runCommandWithOutput(checkCmd)
	if err != nil {
		// If command fails, assume port is available
		return false, nil
	}

	return strings.TrimSpace(output) == "in-use", nil
}

// AllocatePort finds an available port in the given range
func (d *Deployer) AllocatePort(startPort, endPort uint16) (uint16, error) {
	if d.sshClient == nil {
		return 0, fmt.Errorf("not connected to remote host")
	}

	for port := startPort; port <= endPort; port++ {
		inUse, err := d.IsPortInUse(port)
		if err != nil {
			return 0, fmt.Errorf("failed to check port %d: %w", port, err)
		}

		if !inUse {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports in range %d-%d", startPort, endPort)
}

// runCommand executes a command on the remote host
func (d *Deployer) runCommand(cmd string) error {
	_, err := d.runCommandWithOutput(cmd)
	return err
}

// runCommandWithOutput executes a command and returns its output
func (d *Deployer) runCommandWithOutput(cmd string) (string, error) {
	session, err := d.sshClient.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}

// runCommandWithLogging executes a command and streams its output to the writer (for TUI display)
func (d *Deployer) runCommandWithLogging(cmd string) error {
	session, err := d.sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Stream stdout and stderr to the writer (goes to TUI and auxlog)
	session.Stdout = d.writer
	session.Stderr = d.writer

	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

// uploadFile uploads a local file to the remote host
func (d *Deployer) uploadFile(localPath, remotePath string) error {
	// Read local file
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	return d.uploadContent(string(data), remotePath)
}

// uploadContent uploads content to a remote file
func (d *Deployer) uploadContent(content, remotePath string) error {
	session, err := d.sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Use cat to write file
	cmd := fmt.Sprintf("cat > %s", remotePath)
	session.Stdin = strings.NewReader(content)

	if output, err := session.CombinedOutput(cmd); err != nil {
		return fmt.Errorf("failed to upload file: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// buildServerBinary gets the appropriate pre-compiled VPN server binary for the target system
// The binary is selected based on remote OS and architecture detection
func (d *Deployer) buildServerBinary(ctx context.Context) (string, error) {
	// Detect remote OS and architecture
	d.writef("Detecting remote system...\n")
	remoteOS, err := d.detectRemoteOS()
	if err != nil {
		return "", fmt.Errorf("failed to detect remote OS: %w", err)
	}

	remoteArch, err := d.detectRemoteArch()
	if err != nil {
		return "", fmt.Errorf("failed to detect remote architecture: %w", err)
	}

	d.writef("  OS: %s\n  Architecture: %s\n", remoteOS, remoteArch)

	// Get the appropriate embedded binary
	binaryData, err := d.getEmbeddedBinary(remoteOS, remoteArch)
	if err != nil {
		return "", err
	}

	// Write binary to temporary file
	tmpDir := os.TempDir()
	binaryPath := filepath.Join(tmpDir, "vpn-server-"+strconv.FormatInt(time.Now().Unix(), 10))

	d.writef("Writing VPN server binary to %s\n", binaryPath)
	if err := os.WriteFile(binaryPath, binaryData, 0755); err != nil {
		return "", fmt.Errorf("failed to write binary: %w", err)
	}

	size := float64(len(binaryData)) / 1024 / 1024
	d.writef("✓ VPN server binary prepared (%.1f MB)\n", size)

	return binaryPath, nil
}

// detectRemoteOS detects the operating system on the remote machine
func (d *Deployer) detectRemoteOS() (string, error) {
	// Try /etc/os-release first (most modern Linux distros)
	output, err := d.runCommandWithOutput("cat /etc/os-release 2>/dev/null")
	if err == nil && output != "" {
		lower := strings.ToLower(output)
		if strings.Contains(lower, "ubuntu") || strings.Contains(lower, "debian") ||
			strings.Contains(lower, "centos") || strings.Contains(lower, "rocky") ||
			strings.Contains(lower, "red hat") || strings.Contains(lower, "rhel") {
			return "linux", nil
		}
	}

	// Check for macOS
	_, err = d.runCommandWithOutput("sw_vers")
	if err == nil {
		return "darwin", nil
	}

	// Default to linux if we can't detect (most VAST nodes are Linux)
	return "linux", nil
}

// detectRemoteArch detects the CPU architecture on the remote machine
func (d *Deployer) detectRemoteArch() (string, error) {
	output, err := d.runCommandWithOutput("uname -m")
	if err != nil {
		return "", fmt.Errorf("failed to detect architecture: %w", err)
	}

	arch := strings.TrimSpace(output)
	switch arch {
	case "x86_64", "amd64":
		return "amd64", nil
	case "aarch64", "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("unsupported architecture: %s (supported: x86_64/amd64, aarch64/arm64)", arch)
	}
}

// getEmbeddedBinary returns the appropriate pre-compiled binary for the target system
func (d *Deployer) getEmbeddedBinary(os, arch string) ([]byte, error) {
	key := fmt.Sprintf("%s-%s", os, arch)
	switch key {
	case "linux-amd64":
		return vpnServerLinuxAmd64, nil
	case "linux-arm64":
		return vpnServerLinuxArm64, nil
	case "darwin-amd64":
		return vpnServerDarwinAmd64, nil
	case "darwin-arm64":
		return vpnServerDarwinArm64, nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s/%s\nSupported platforms: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64", os, arch)
	}
}

// ensureWireGuard ensures WireGuard is available on the remote system
// Checks for kernel module or wireguard-go, attempts auto-installation if possible
func (d *Deployer) ensureWireGuard(ctx context.Context) error {
	d.writef("Checking WireGuard availability...\n")

	// Check 1: WireGuard kernel module (try to create a test interface)
	d.writef("Checking for WireGuard kernel module...\n")
	checkKernel := "sudo ip link add name wgtest type wireguard 2>/dev/null && sudo ip link del wgtest 2>/dev/null"
	if err := d.runCommand(checkKernel); err == nil {
		d.writef("✓ WireGuard kernel module available\n")
		return nil
	}

	// Check 2: wireguard-go (userspace implementation)
	d.writef("Checking for wireguard-go (userspace implementation)...\n")
	checkWgGo := "which wireguard-go"
	if err := d.runCommand(checkWgGo); err == nil {
		d.writef("✓ wireguard-go available\n")
		return nil
	}

	// Check 3: wg command exists (might help with installation)
	checkWg := "which wg"
	wgExists := d.runCommand(checkWg) == nil

	// Get kernel info for error message
	kernelInfo, _ := d.runCommandWithOutput("uname -r")
	kernelInfo = strings.TrimSpace(kernelInfo)

	// Detect if this is a custom kernel (Lightbits, etc.)
	isCustomKernel := strings.Contains(strings.ToLower(kernelInfo), ".lb") ||
		strings.Contains(strings.ToLower(kernelInfo), "lightbits")

	// Try auto-installation for both standard and custom kernels
	if !wgExists {
		d.writef("WireGuard not found, attempting auto-installation...\n")
		if isCustomKernel {
			d.writef("Note: Custom kernel detected (%s), kernel module won't work\n", kernelInfo)
			d.writef("Will install wireguard-tools for userspace implementation\n")
		}
		return d.installWireGuard(ctx)
	}

	// wg command exists but kernel module doesn't work
	// For custom kernels, wireguard-go is REQUIRED (kernel module won't work)
	if isCustomKernel {
		d.writef("✓ WireGuard tools (wg command) found\n")
		d.writef("Note: Custom kernel detected, wireguard-go is required\n")
		// Install wireguard-go for userspace implementation
		return d.installWireGuardGo(ctx)
	}

	// Standard kernel but module isn't working - this is unexpected
	return fmt.Errorf(`WireGuard tools are installed but the kernel module is not available.

Kernel: %s

This might be because:
1. The kernel module is not loaded: Try 'sudo modprobe wireguard'
2. The kernel doesn't support WireGuard: Upgrade your kernel

SOLUTIONS:

Try loading the module:
  sudo modprobe wireguard

After fixing, try connecting again.`, kernelInfo)
}

// installWireGuard attempts to auto-install WireGuard on supported systems
func (d *Deployer) installWireGuard(ctx context.Context) error {
	// Detect OS for installation
	osRelease, _ := d.runCommandWithOutput("cat /etc/os-release 2>/dev/null")
	lower := strings.ToLower(osRelease)

	var installCmd string
	var osName string

	switch {
	case strings.Contains(lower, "ubuntu"), strings.Contains(lower, "debian"):
		osName = "Ubuntu/Debian"
		installCmd = "sudo apt-get update -qq && sudo apt-get install -y wireguard"

	case strings.Contains(lower, "centos"), strings.Contains(lower, "rocky"):
		osName = "CentOS/Rocky"
		version, _ := d.detectOSVersion()
		if strings.HasPrefix(version, "7") {
			installCmd = `
				sudo yum install -y epel-release elrepo-release && \
				sudo yum install -y kmod-wireguard wireguard-tools
			`
		} else {
			// CentOS/Rocky 8+
			installCmd = `
				sudo yum install -y epel-release elrepo-release && \
				sudo yum install -y kmod-wireguard wireguard-tools
			`
		}

	case strings.Contains(lower, "red hat"), strings.Contains(lower, "rhel"):
		osName = "Red Hat Enterprise Linux"
		version, _ := d.detectOSVersion()
		if strings.HasPrefix(version, "7") {
			installCmd = `
				sudo yum install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm \
					https://www.elrepo.org/elrepo-release-7.el7.elrepo.noarch.rpm && \
				sudo yum install -y kmod-wireguard wireguard-tools
			`
		} else {
			// RHEL 8+
			installCmd = `
				sudo yum install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm \
					https://www.elrepo.org/elrepo-release-8.el8.elrepo.noarch.rpm && \
				sudo yum install -y kmod-wireguard wireguard-tools
			`
		}

	default:
		return fmt.Errorf(`WireGuard not found and OS not supported for auto-installation.

Please install WireGuard manually:
  • Installation guide: https://www.wireguard.com/install/
  • Supported OS: Ubuntu, Debian, CentOS, RHEL, Rocky Linux, macOS

After installation, try connecting again.`)
	}

	d.writef("Installing WireGuard on %s...\n", osName)
	d.writef("This may take a few minutes...\n")
	d.writef("--- Installation Output ---\n")

	// Use runCommandWithLogging to stream all installation output to TUI/auxlog
	if err := d.runCommandWithLogging(installCmd); err != nil {
		return fmt.Errorf(`WireGuard installation failed: %w

Please install WireGuard manually:
  • Installation guide: https://www.wireguard.com/install/
  • For %s, run:
    %s

After installation, try connecting again.`, err, osName, installCmd)
	}

	d.writef("--- Installation Complete ---\n")

	d.writef("✓ WireGuard tools installed successfully\n")

	// Try to load the WireGuard kernel module and verify it actually works
	d.writef("Checking if WireGuard kernel module can be loaded...\n")
	checkModuleCmd := "sudo modprobe wireguard 2>/dev/null && sudo ip link add name wgtest type wireguard 2>/dev/null && sudo ip link del wgtest 2>/dev/null"
	if err := d.runCommand(checkModuleCmd); err == nil {
		d.writef("✓ WireGuard kernel module loaded and working\n")
		return nil // Kernel module works, no need for wireguard-go
	}

	// Kernel module failed or doesn't work, install wireguard-go for userspace implementation
	d.writef("Kernel module not available or not working, installing wireguard-go...\n")
	return d.installWireGuardGo(ctx)
}

// installWireGuardGo attempts to install wireguard-go for userspace WireGuard
func (d *Deployer) installWireGuardGo(ctx context.Context) error {
	d.writef("Installing wireguard-go (userspace WireGuard)...\n")

	// Check if wireguard-go is already installed
	if err := d.runCommand("which wireguard-go"); err == nil {
		d.writef("✓ wireguard-go is already installed\n")
		return nil
	}

	// Method 1: Check if Go is available on remote for direct installation
	d.writef("Checking if Go is available on remote server...\n")
	if err := d.runCommand("which go"); err == nil {
		d.writef("✓ Go is installed on remote, using 'go install'\n")
		installCmd := "go install golang.zx2c4.com/wireguard/cmd/wireguard-go@latest"
		if err := d.runCommandWithLogging(installCmd); err == nil {
			// Link from ~/go/bin to /usr/local/bin
			if err := d.runCommand("test -f ~/go/bin/wireguard-go && sudo ln -sf ~/go/bin/wireguard-go /usr/local/bin/wireguard-go"); err == nil {
				d.writef("✓ wireguard-go installed and linked\n")
				return nil
			}
		}
	}

	// Method 2: Build locally and upload
	d.writef("Building wireguard-go locally and uploading...\n")
	localBinary, err := d.buildWireGuardGo(ctx)
	if err != nil {
		d.writef("✗ Failed to build wireguard-go locally: %v\n", err)
		d.writef("Note: wireguard-go is required for the VPN server to work\n")
		return fmt.Errorf("failed to install wireguard-go: %w", err)
	}
	defer os.Remove(localBinary)

	// Upload to remote
	remoteBinary := "/tmp/wireguard-go"
	if err := d.uploadFile(localBinary, remoteBinary); err != nil {
		return fmt.Errorf("failed to upload wireguard-go: %w", err)
	}

	// Make executable and move to /usr/local/bin
	cmds := []string{
		fmt.Sprintf("chmod +x %s", remoteBinary),
		fmt.Sprintf("sudo mv %s /usr/local/bin/wireguard-go", remoteBinary),
	}

	for _, cmd := range cmds {
		if err := d.runCommand(cmd); err != nil {
			return fmt.Errorf("failed to install wireguard-go: %w", err)
		}
	}

	d.writef("✓ wireguard-go installed successfully\n")
	return nil
}

// buildWireGuardGo builds wireguard-go from the git repository for upload to remote
func (d *Deployer) buildWireGuardGo(ctx context.Context) (string, error) {
	d.writef("Cloning wireguard-go from official repository...\n")

	tmpDir, err := os.MkdirTemp("", "wireguard-go-git-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Clone the repository
	cloneCmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", "https://git.zx2c4.com/wireguard-go", tmpDir)
	if output, err := cloneCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to clone wireguard-go: %w\nOutput: %s", err, string(output))
	}

	d.writef("✓ Repository cloned\n")

	// Build it
	d.writef("Compiling wireguard-go for Linux...\n")
	buildCmd := exec.CommandContext(ctx, "make")
	buildCmd.Dir = tmpDir
	buildCmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS=linux",
		"GOARCH=amd64",
	)

	if output, err := buildCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to build wireguard-go: %w\nOutput: %s", err, string(output))
	}

	// The binary should be in tmpDir/wireguard-go
	builtBinary := filepath.Join(tmpDir, "wireguard-go")
	if _, err := os.Stat(builtBinary); err != nil {
		return "", fmt.Errorf("wireguard-go binary not found after build: %w", err)
	}

	// Copy to a persistent temp location
	finalBinary := filepath.Join(os.TempDir(), "wireguard-go-upload")
	if err := copyFile(builtBinary, finalBinary); err != nil {
		return "", fmt.Errorf("failed to prepare binary: %w", err)
	}

	d.writef("✓ wireguard-go built successfully (ready for upload)\n")
	return finalBinary, nil
}

// detectOSVersion detects the OS version from /etc/os-release
func (d *Deployer) detectOSVersion() (string, error) {
	output, err := d.runCommandWithOutput("grep -E '^VERSION_ID=' /etc/os-release 2>/dev/null | cut -d'=' -f2 | tr -d '\"'")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// generateServerConfig generates a YAML configuration file for the server
func (d *Deployer) generateServerConfig(config *common.ServerConfig) string {
	// Build private IPs list (comma-separated)
	var privateIPsStr string
	if len(config.PrivateIPs) > 0 {
		ipStrs := make([]string, len(config.PrivateIPs))
		for i, ip := range config.PrivateIPs {
			ipStrs[i] = ip.String()
		}
		privateIPsStr = strings.Join(ipStrs, ", ")
	}

	return fmt.Sprintf(`# VPN Server Configuration
privateKey: %s
publicKey: %s
listenPort: %d
serverIP: %s
vpnNetwork: %s
privateIPs: %s
interface: %s
`,
		config.PrivateKey,
		config.PublicKey,
		config.ListenPort,
		config.ServerIP.String(),
		config.VPNNetwork.String(),
		privateIPsStr,
		config.Interface,
	)
}

// GetServerLogs retrieves the server logs from remote host
func (d *Deployer) GetServerLogs(workDir string, lines int) (string, error) {
	if d.sshClient == nil {
		return "", fmt.Errorf("not connected to remote host")
	}

	logFile := filepath.Join(workDir, "server.log")
	cmd := fmt.Sprintf("tail -n %d %s 2>/dev/null || echo 'No logs available'", lines, logFile)

	output, err := d.runCommandWithOutput(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve logs: %w", err)
	}

	return output, nil
}

// DownloadFile downloads a file from the remote host
func (d *Deployer) DownloadFile(remotePath, localPath string) error {
	if d.sshClient == nil {
		return fmt.Errorf("not connected to remote host")
	}

	session, err := d.sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Read remote file
	var buf bytes.Buffer
	session.Stdout = &buf

	cmd := fmt.Sprintf("cat %s", remotePath)
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("failed to read remote file: %w", err)
	}

	// Write to local file
	if err := os.WriteFile(localPath, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("failed to write local file: %w", err)
	}

	return nil
}

// ExecuteInteractive executes a command and streams output
func (d *Deployer) ExecuteInteractive(cmd string, stdout, stderr io.Writer) error {
	if d.sshClient == nil {
		return fmt.Errorf("not connected to remote host")
	}

	session, err := d.sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	session.Stdout = stdout
	session.Stderr = stderr

	return session.Run(cmd)
}

// StartHeartbeat starts sending periodic heartbeat signals to the remote server
// The server will self-destruct if it doesn't receive a heartbeat for 12 seconds
func (d *Deployer) StartHeartbeat(workDir string) error {
	if d.sshClient == nil {
		return fmt.Errorf("not connected to remote host")
	}

	// Stop any existing heartbeat
	if d.heartbeatCancel != nil {
		d.heartbeatCancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.heartbeatCancel = cancel

	heartbeatFile := filepath.Join(workDir, "heartbeat")

	// Start heartbeat goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		d.writef("Heartbeat monitoring started (interval: 5s)\n")

		// Send initial heartbeat
		if err := d.sendHeartbeat(heartbeatFile); err != nil {
			d.writef("Warning: Failed to send initial heartbeat: %v\n", err)
		}

		for {
			select {
			case <-ctx.Done():
				d.writef("Heartbeat monitoring stopped\n")
				return
			case <-ticker.C:
				if err := d.sendHeartbeat(heartbeatFile); err != nil {
					d.writef("Warning: Heartbeat failed: %v\n", err)
				}
				// Heartbeat sent successfully - no need to log every 5 seconds
			}
		}
	}()

	return nil
}

// sendHeartbeat sends a single heartbeat signal by updating the timestamp file
func (d *Deployer) sendHeartbeat(heartbeatFile string) error {
	timestamp := time.Now().Unix()
	cmd := fmt.Sprintf("echo %d > %s", timestamp, heartbeatFile)
	return d.runCommand(cmd)
}

// writef writes a formatted message to the writer if available
// Also logs to aux logger for TUI integration
func (d *Deployer) writef(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	// Format with timestamp like log.Logger does
	timestamp := time.Now().Format("2006/01/02 15:04:05")

	// Ensure the message ends with a newline for proper display
	if !strings.HasSuffix(msg, "\n") {
		msg = msg + "\n"
	}

	formattedMsg := fmt.Sprintf("%s [vpn deployer] %s", timestamp, msg)
	d.writer.Write([]byte(formattedMsg))
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// SetVipPoolIPs sets the VIP pool IPs for health monitoring
func (d *Deployer) SetVipPoolIPs(ips []netip.Addr) {
	d.vipPoolIPs = ips
}

// CheckSSHHealth checks if the SSH connection is still alive
// Also pings a random VIP pool IP to verify end-to-end connectivity
// Returns error if the connection is dead or IP is unreachable
func (d *Deployer) CheckSSHHealth() error {
	if d.sshClient == nil {
		return fmt.Errorf("SSH client not initialized")
	}

	// Use a context with timeout to prevent blocking indefinitely
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Channel to receive the result
	resultChan := make(chan error, 1)

	go func() {
		// Try to create a new session - this will fail if the connection is dead
		session, err := d.sshClient.NewSession()
		if err != nil {
			resultChan <- fmt.Errorf("SSH connection dead: %w", err)
			return
		}
		defer session.Close()

		// If we have VIP pool IPs, ping a random one to verify end-to-end connectivity
		if len(d.vipPoolIPs) > 0 {
			// Pick a random IP
			randomIP := d.vipPoolIPs[rand.Intn(len(d.vipPoolIPs))]

			// Ping the IP (1 ping, 2 sec timeout, no output logged)
			pingCmd := fmt.Sprintf("ping -c 1 -W 2 %s > /dev/null 2>&1", randomIP)
			if err := session.Run(pingCmd); err != nil {
				resultChan <- fmt.Errorf("VIP pool IP %s unreachable: %w", randomIP, err)
				return
			}
		} else {
			// No VIP pool IPs to check, just run a simple command
			if err := session.Run("echo health_check"); err != nil {
				resultChan <- fmt.Errorf("SSH health check command failed: %w", err)
				return
			}
		}

		resultChan <- nil
	}()

	// Wait for either the result or timeout
	select {
	case err := <-resultChan:
		return err
	case <-ctx.Done():
		return fmt.Errorf("SSH health check timeout (connection likely lost)")
	}
}
