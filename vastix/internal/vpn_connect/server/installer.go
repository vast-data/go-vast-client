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

	case "centos", "rhel", "fedora", "rocky":
		// Try to install wireguard-tools via package manager first (available in RHEL 9+, Rocky 9+, Fedora)
		// Fall back to wireguard-go if package manager installation fails
		cmd = exec.Command("sudo", "dnf", "install", "-y", "wireguard-tools")
		if err := cmd.Run(); err != nil {
			// Fallback: try yum for older systems
			cmd = exec.Command("sudo", "yum", "install", "-y", "wireguard-tools")
			if err := cmd.Run(); err != nil {
				// If package manager fails, try wireguard-go (requires Go)
				return installWireGuardGo()
			}
		}
		return nil

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

// HasKernelWireGuard checks whether the WireGuard kernel module is available on
// the running system. It first checks the fast sysfs path and then tries a
// modprobe (which is a no-op if the module cannot be loaded).
func HasKernelWireGuard() bool {
	// Fast path: module is already loaded.
	if _, err := os.Stat("/sys/module/wireguard"); err == nil {
		return true
	}
	// Try to load the module; the server runs as root so this is allowed.
	// This is a safe no-op on kernels that don't have the module at all.
	if err := exec.Command("modprobe", "wireguard").Run(); err == nil {
		return true
	}
	return false
}

// GetWireGuardCommand returns the appropriate WireGuard command.
// Returns the sentinel string "kernel" when the native kernel module is
// available (preferred over wireguard-go to avoid the sendmmsg issue),
// otherwise returns the path to wireguard-go.
func GetWireGuardCommand() (string, error) {
	// Kernel module is the reliable choice: it avoids the
	// "sendmmsg: invalid argument" errors that wireguard-go produces on
	// kernels with native WireGuard support.
	if HasKernelWireGuard() {
		return "kernel", nil
	}

	// Fall back to wireguard-go (userspace implementation).
	if path, err := exec.LookPath("wireguard-go"); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("no WireGuard installation found (kernel module unavailable and wireguard-go not in PATH)")
}
