// Package server provides VPN server functionality
package server

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// CheckWireGuardInstalled checks if WireGuard tools are installed
func CheckWireGuardInstalled() (bool, error) {
	// Check for wireguard-go first (userspace implementation)
	if path, err := exec.LookPath("wireguard-go"); err == nil {
		return true, nil
	} else if path != "" {
		return true, nil
	}

	// Check for wg command (kernel module)
	if path, err := exec.LookPath("wg"); err == nil {
		return true, nil
	} else if path != "" {
		return true, nil
	}

	return false, nil
}

// InstallWireGuard attempts to install WireGuard based on the OS
func InstallWireGuard() error {
	switch runtime.GOOS {
	case "linux":
		return installWireGuardLinux()
	case "darwin":
		return fmt.Errorf("macOS installation not implemented. Please install WireGuard manually from https://www.wireguard.com/install/")
	case "windows":
		return fmt.Errorf("Windows installation not implemented. Please install WireGuard manually from https://www.wireguard.com/install/")
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// installWireGuardLinux installs WireGuard on Linux
func installWireGuardLinux() error {
	// Detect Linux distribution
	distro, err := detectLinuxDistro()
	if err != nil {
		return fmt.Errorf("failed to detect Linux distribution: %w", err)
	}

	var cmd *exec.Cmd

	switch distro {
	case "ubuntu", "debian":
		cmd = exec.Command("sudo", "apt-get", "update")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to update package lists: %w", err)
		}
		cmd = exec.Command("sudo", "apt-get", "install", "-y", "wireguard")

	case "centos", "rhel", "fedora":
		// For CentOS/RHEL, wireguard-go is safer as kernel module may not be available
		return installWireGuardGo()

	case "arch":
		cmd = exec.Command("sudo", "pacman", "-Sy", "--noconfirm", "wireguard-tools")

	default:
		// Fallback to wireguard-go for unknown distributions
		return installWireGuardGo()
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install WireGuard: %w", err)
	}

	return nil
}

// installWireGuardGo installs the userspace wireguard-go implementation
func installWireGuardGo() error {
	// Check if Go is installed
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("Go is not installed. Please install Go or WireGuard manually")
	}

	// Install wireguard-go
	cmd := exec.Command("go", "install", "golang.zx2c4.com/wireguard/cmd/wireguard-go@latest")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install wireguard-go: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// detectLinuxDistro detects the Linux distribution
func detectLinuxDistro() (string, error) {
	// Try reading /etc/os-release
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", err
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "ID=") {
			distro := strings.TrimPrefix(line, "ID=")
			distro = strings.Trim(distro, `"`)
			return strings.ToLower(distro), nil
		}
	}

	return "", fmt.Errorf("could not determine Linux distribution")
}

// GetWireGuardCommand returns the appropriate WireGuard command
func GetWireGuardCommand() (string, error) {
	// Prefer wireguard-go (userspace) for better compatibility
	if path, err := exec.LookPath("wireguard-go"); err == nil {
		return path, nil
	}

	// Fallback to kernel module
	if path, err := exec.LookPath("wg"); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("no WireGuard installation found")
}
