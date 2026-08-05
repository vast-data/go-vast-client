package client

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	wgInterfaceName      = "wgvastix"
	wireGuardSudoersFile = "/etc/sudoers.d/vastix-wireguard"
)

// wireGuardSystemConfigDir returns the directory where wg-quick expects system configs.
func wireGuardSystemConfigDir() string {
	if runtime.GOOS == "darwin" {
		candidates := []string{
			"/opt/homebrew/etc/wireguard",
			"/usr/local/etc/wireguard",
			"/etc/wireguard",
		}
		for _, dir := range candidates {
			if info, err := os.Stat(dir); err == nil && info.IsDir() {
				return dir
			}
		}
		if _, err := os.Stat("/opt/homebrew"); err == nil {
			return "/opt/homebrew/etc/wireguard"
		}
		return "/usr/local/etc/wireguard"
	}
	return "/etc/wireguard"
}

// platformBash returns a Bash suitable for wg-quick scripts (Bash 4+ on macOS).
func platformBash() string {
	if runtime.GOOS == "darwin" {
		for _, path := range []string{
			"/opt/homebrew/bin/bash",
			"/usr/local/bin/bash",
		} {
			if fileExists(path) {
				return path
			}
		}
	}
	return lookPathOr("bash", "/bin/bash")
}

func lookPathOr(name, fallback string) string {
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	return fallback
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func appendLookPath(paths []string, names ...string) []string {
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			paths = append(paths, path)
		}
	}
	return paths
}

func appendExistingPaths(paths []string, candidates ...string) []string {
	for _, candidate := range candidates {
		if fileExists(candidate) {
			paths = append(paths, candidate)
		}
	}
	return paths
}

// wireGuardSudoNOPASSWDPaths builds the command list for a passwordless sudoers entry
// using resolved binary paths for the current platform.
func wireGuardSudoNOPASSWDPaths() []string {
	paths := appendLookPath(nil, "wg-quick", "wg", "bash", "install", "mkdir", "cp", "chmod", "rm", "visudo")

	if runtime.GOOS == "linux" {
		paths = appendLookPath(paths, "ip", "modprobe")
		paths = appendExistingPaths(paths,
			"/usr/sbin/ip",
			"/sbin/ip",
			"/usr/sbin/modprobe",
			"/sbin/modprobe",
		)
	}

	if runtime.GOOS == "darwin" {
		paths = append(paths, platformBash())
		if wgQuick := lookPathOr("wg-quick", ""); wgQuick != "" {
			paths = append(paths, wgQuick)
		}
		paths = appendLookPath(paths, "ifconfig", "route", "networksetup")
		paths = appendExistingPaths(paths,
			"/sbin/ifconfig",
			"/sbin/route",
			"/usr/sbin/ifconfig",
			"/usr/sbin/networksetup",
			"/opt/homebrew/bin/bash",
			"/usr/local/bin/bash",
			"/opt/homebrew/bin/wg-quick",
			"/usr/local/bin/wg-quick",
			"/opt/homebrew/bin/wg",
			"/usr/local/bin/wg",
		)
	}

	return uniquePaths(paths)
}

func wgQuickSudoArgs(action, target string) []string {
	wgQuick := lookPathOr("wg-quick", "wg-quick")
	if runtime.GOOS == "darwin" {
		return []string{platformBash(), wgQuick, action, target}
	}
	return []string{wgQuick, action, target}
}

func buildStaleStateCleanupScript() string {
	wgQuick := lookPathOr("wg-quick", "wg-quick")
	if runtime.GOOS == "darwin" {
		bash := platformBash()
		return fmt.Sprintf(`set -e
rm -rf /tmp/vastix 2>/dev/null || true
%q %q down %q 2>/dev/null || true
`, bash, wgQuick, wgInterfaceName)
	}

	return fmt.Sprintf(`set -e
rm -rf /tmp/vastix 2>/dev/null || true
ip link delete %q 2>/dev/null || true
%q down %q 2>/dev/null || true
`, wgInterfaceName, wgQuick, wgInterfaceName)
}

func buildConnectUpScript(systemConfigDir, localConfigPath, systemConfigPath string) string {
	wgQuick := lookPathOr("wg-quick", "wg-quick")
	systemDir := filepath.Dir(systemConfigPath)

	if runtime.GOOS == "darwin" {
		bash := platformBash()
		return fmt.Sprintf(`set -e
mkdir -p %q
install -m 600 -o root -g wheel %q %q 2>/dev/null || { cp %q %q && chmod 600 %q; }
%q %q down %q 2>/dev/null || true
%q %q up %q
`, systemDir, localConfigPath, systemConfigPath, localConfigPath, systemConfigPath, systemConfigPath,
			bash, wgQuick, wgInterfaceName,
			bash, wgQuick, wgInterfaceName)
	}

	return fmt.Sprintf(`set -e
mkdir -p %q
install -m 600 -o root -g root %q %q 2>/dev/null || { cp %q %q && chmod 600 %q; }
ip link delete %q 2>/dev/null || true
%q down %q 2>/dev/null || true
modprobe wireguard 2>/dev/null || true
%q up %q
`, systemDir, localConfigPath, systemConfigPath, localConfigPath, systemConfigPath, systemConfigPath,
		wgInterfaceName, wgQuick, wgInterfaceName, wgQuick, wgInterfaceName)
}

func buildDisconnectCleanupScript() string {
	if runtime.GOOS == "darwin" {
		return ""
	}
	return fmt.Sprintf(`ip link delete %q 2>/dev/null || true`, wgInterfaceName)
}

func connectFailureHints() string {
	if runtime.GOOS == "darwin" {
		return "  1. WireGuard is installed (brew install wireguard-tools or WireGuard app)\n" +
			"  2. You have sudo privileges\n" +
			"  3. Homebrew Bash 4+ is available for wg-quick"
	}
	return "  1. WireGuard is installed (wg-quick)\n" +
		"  2. You have sudo privileges\n" +
		"  3. The WireGuard kernel module is loaded"
}

func wireGuardInstallHints() string {
	if runtime.GOOS == "darwin" {
		return "wg-quick not found. Please install WireGuard:\n" +
			"  Homebrew:      brew install wireguard-tools bash\n" +
			"  Or download:   https://www.wireguard.com/install/"
	}
	return "wg-quick not found. Please install WireGuard:\n" +
		"  Ubuntu/Debian: sudo apt install wireguard-tools\n" +
		"  RHEL/CentOS:   sudo yum install wireguard-tools\n" +
		"  Arch:          sudo pacman -S wireguard-tools"
}

func needsWireGuardKernelModule() bool {
	return runtime.GOOS == "linux"
}

func buildSudoersInstallScript(tmpPath string) string {
	installBin := lookPathOr("install", "/usr/bin/install")
	visudoBin := lookPathOr("visudo", "/usr/sbin/visudo")
	return fmt.Sprintf(`set -e
[ -f %q ] && exit 0
%q -cf %q
%q -m 440 %q %q
`, wireGuardSudoersFile, visudoBin, tmpPath, installBin, tmpPath, wireGuardSudoersFile)
}

func buildSudoersContent(username string) string {
	paths := wireGuardSudoNOPASSWDPaths()
	return fmt.Sprintf("# Managed by Vastix - passwordless sudo for local VPN client operations\n%s ALL=(ALL) NOPASSWD: %s\n",
		username, strings.Join(paths, ", "))
}
