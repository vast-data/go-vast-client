package client

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"runtime"
	"strings"
)

// PrepareLocalEnvironment performs all local setup needed before a VPN connection.
// It is safe to call on every connect; stale state is cleaned automatically.
func PrepareLocalEnvironment(sudoPassword string, writer io.Writer) error {
	writeStep(writer, "Checking WireGuard tools...")
	if err := EnsureWireGuardTools(sudoPassword, writer); err != nil {
		return err
	}
	writeStep(writer, "WireGuard tools ready")

	writeStep(writer, "Cleaning up stale WireGuard state...")
	if err := cleanupStaleWireGuardState(sudoPassword, writer); err != nil {
		return fmt.Errorf("failed to clean up stale WireGuard state: %w", err)
	}
	writeStep(writer, "Stale state cleaned")

	if CheckSudoNeedsPassword() {
		writeStep(writer, "Configuring passwordless sudo for WireGuard (one-time setup)...")
		if err := ensurePasswordlessWireGuardSudo(sudoPassword, writer); err != nil {
			writeStep(writer, fmt.Sprintf("Note: could not configure passwordless sudo: %v", err))
			writeStep(writer, "Continuing with provided sudo password")
		} else {
			writeStep(writer, "Passwordless sudo configured for future connections")
		}
	}

	if needsWireGuardKernelModule() {
		writeStep(writer, "Loading WireGuard kernel module...")
		modprobe := lookPathOr("modprobe", "/sbin/modprobe")
		_ = runSudo(sudoPassword, writer, modprobe, "wireguard")
	}

	return nil
}

// EnsureWireGuardTools verifies wg-quick is available and installs it when possible.
func EnsureWireGuardTools(sudoPassword string, writer io.Writer) error {
	if err := CheckWireGuardInstalled(); err == nil {
		return nil
	}

	writeStep(writer, "WireGuard not found, attempting automatic installation...")
	if err := installWireGuardTools(sudoPassword, writer); err != nil {
		return fmt.Errorf("wireguard is not installed and automatic installation failed: %w", err)
	}

	return CheckWireGuardInstalled()
}

func cleanupStaleWireGuardState(sudoPassword string, writer io.Writer) error {
	script := buildStaleStateCleanupScript()

	if err := runSudoScript(sudoPassword, writer, script); err != nil {
		if sudoPassword == "" && CheckSudoNeedsPassword() {
			return fmt.Errorf("sudo password required to clean up stale WireGuard state")
		}
		return err
	}

	hostname, err := os.Hostname()
	if err == nil {
		if workDir, err := userWorkDir(hostname); err == nil {
			_ = ensureUserWorkDir(workDir, sudoPassword)
		}
	}

	return nil
}

func ensurePasswordlessWireGuardSudo(sudoPassword string, writer io.Writer) error {
	if _, err := os.Stat(wireGuardSudoersFile); err == nil {
		return nil
	}

	username, err := currentUsername()
	if err != nil {
		return err
	}

	content := buildSudoersContent(username)

	tmpFile, err := os.CreateTemp("", "vastix-wireguard-sudoers-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(content); err != nil {
		if closeErr := tmpFile.Close(); closeErr != nil {
			return fmt.Errorf("%w (also failed to close temp file: %v)", err, closeErr)
		}
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	return runSudoScript(sudoPassword, writer, buildSudoersInstallScript(tmpPath))
}

func installWireGuardTools(sudoPassword string, writer io.Writer) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("automatic installation is only supported on Linux; on macOS install via Homebrew: brew install wireguard-tools bash")
	}

	distro, err := detectLinuxDistro()
	if err != nil {
		return err
	}

	var installScript string
	switch distro {
	case "ubuntu", "debian":
		installScript = "apt-get update -qq && apt-get install -y wireguard-tools"
	case "centos", "rhel", "fedora", "rocky":
		installScript = "dnf install -y wireguard-tools || yum install -y wireguard-tools"
	case "arch":
		installScript = "pacman -Sy --noconfirm wireguard-tools"
	default:
		return fmt.Errorf("unsupported Linux distribution %q; install wireguard-tools manually", distro)
	}

	return runSudoScript(sudoPassword, writer, installScript)
}

func detectLinuxDistro() (string, error) {
	for _, path := range []string{"/etc/os-release", "/usr/lib/os-release"} {
		data, err := os.ReadFile(path) // #nosec G304 -- standard Linux distro identification paths
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "ID=") {
				return strings.Trim(strings.TrimPrefix(line, "ID="), `"`), nil
			}
		}
	}
	return "", fmt.Errorf("could not detect Linux distribution")
}

func currentUsername() (string, error) {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username, nil
	}
	if username := os.Getenv("USER"); username != "" {
		return username, nil
	}
	if username := os.Getenv("LOGNAME"); username != "" {
		return username, nil
	}
	return "", fmt.Errorf("could not determine current username")
}

func writeStep(writer io.Writer, message string) {
	if writer == nil {
		return
	}
	fmt.Fprintf(writer, "[vpn client] %s\n", message)
}
