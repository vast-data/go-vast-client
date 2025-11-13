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
	cmd := exec.Command("sudo", "-n", "true")
	err := cmd.Run()
	// If command succeeds, no password needed
	// If it fails, password is required
	return err != nil
}

// CheckWgQuickNeedsPassword checks if wg-quick specifically requires a password.
// This is more accurate than CheckSudoNeedsPassword because wg-quick might be
// configured in sudoers for passwordless execution even if other sudo commands require a password.
// It runs `sudo -n wg-quick` to check if passwordless execution is possible.
func CheckWgQuickNeedsPassword() bool {
	// Try to run wg-quick with -n flag (non-interactive) to test passwordless execution
	// Using just the command name without arguments to test sudo access
	cmd := exec.Command("sudo", "-n", "wg-quick")
	err := cmd.Run()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode := exitError.ExitCode()
			// Exit code 1 means password is required
			// Exit code 2 or other might mean wg-quick ran but failed due to missing args (which is fine for our test)
			// We only care if it asks for password (exit code 1)
			return exitCode == 1
		}
		// Other errors - assume password might be needed
		return true
	}

	// Command succeeded (or failed for non-password reasons), no password needed
	return false
}

// ValidateSudoPassword validates a sudo password by trying to run a simple command
// Returns nil if password is valid, error otherwise
func ValidateSudoPassword(password string) error {
	cmd := exec.Command("sudo", "-S", "-k", "true")

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
