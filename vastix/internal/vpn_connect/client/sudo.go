// Package client provides VPN client functionality
package client

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// CheckSudoNeedsPassword checks if sudo requires a password
// Returns true if password is required, false otherwise
func CheckSudoNeedsPassword() bool {
	cmd := exec.Command("sudo", "-n", "true") // #nosec G204 -- fixed argv; checks whether sudo needs a password
	err := cmd.Run()
	// If command succeeds, no password needed
	// If it fails, password is required
	return err != nil
}

// CheckWgQuickNeedsPassword checks if wg-quick specifically requires a password.
// This is more accurate than CheckSudoNeedsPassword because wg-quick might be
// configured in sudoers for passwordless execution even if other sudo commands require a password.
func CheckWgQuickNeedsPassword() bool {
	// wg-quick exits 1 when invoked without arguments, so we cannot rely on exit code alone.
	// Distinguish sudo authentication failure from wg-quick actually running.
	cmd := exec.Command("sudo", "-n", "wg-quick") // #nosec G204 -- fixed argv; checks whether wg-quick is passwordless
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return false
	}

	stderrStr := stderr.String()
	if strings.Contains(stderrStr, "password") ||
		strings.Contains(stderrStr, "authentication") ||
		strings.Contains(stderrStr, "try again") {
		return true
	}

	// wg-quick ran (usage error, missing args, etc.) — passwordless sudo works
	return false
}

// ValidateSudoPassword validates a sudo password by trying to run a simple command
// Returns nil if password is valid, error otherwise
func ValidateSudoPassword(password string) error {
	cmd := exec.Command("sudo", "-S", "-k", "true") // #nosec G204 -- fixed argv; validates a sudo password

	// Pass password via stdin
	cmd.Stdin = strings.NewReader(password + "\n")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "incorrect password") ||
			strings.Contains(stderrStr, "Sorry, try again") {
			return fmt.Errorf("invalid sudo password")
		}
		return fmt.Errorf("sudo validation failed: %w (stderr: %s)", err, stderrStr)
	}

	return nil
}
