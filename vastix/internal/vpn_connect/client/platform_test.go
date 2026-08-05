package client

import (
	"runtime"
	"strings"
	"testing"
)

func TestWireGuardSystemConfigDir(t *testing.T) {
	dir := wireGuardSystemConfigDir()
	if dir == "" {
		t.Fatal("expected non-empty config dir")
	}
	if runtime.GOOS == "linux" && dir != "/etc/wireguard" {
		t.Fatalf("linux config dir = %q, want /etc/wireguard", dir)
	}
}

func TestBuildStaleStateCleanupScript(t *testing.T) {
	script := buildStaleStateCleanupScript()
	if !strings.Contains(script, wgInterfaceName) {
		t.Fatalf("cleanup script missing interface: %s", script)
	}
	if runtime.GOOS == "linux" && !strings.Contains(script, "ip link delete") {
		t.Fatalf("linux cleanup should use ip link delete: %s", script)
	}
	if runtime.GOOS == "darwin" && strings.Contains(script, "ip link delete") {
		t.Fatalf("darwin cleanup should not use ip link delete: %s", script)
	}
}

func TestBuildConnectUpScript(t *testing.T) {
	script := buildConnectUpScript("/etc/wireguard", "/tmp/local.conf", "/etc/wireguard/wgvastix.conf")
	if !strings.Contains(script, "wgvastix") {
		t.Fatalf("connect script missing interface: %s", script)
	}
	if runtime.GOOS == "linux" && !strings.Contains(script, "modprobe") {
		t.Fatalf("linux connect should load kernel module: %s", script)
	}
	if runtime.GOOS == "darwin" && strings.Contains(script, "modprobe") {
		t.Fatalf("darwin connect should not use modprobe: %s", script)
	}
}

func TestWireGuardSudoNOPASSWDPathsIncludesCoreTools(t *testing.T) {
	paths := wireGuardSudoNOPASSWDPaths()
	if len(paths) == 0 {
		t.Fatal("expected sudoers paths")
	}
	joined := strings.Join(paths, " ")
	if !strings.Contains(joined, "wg-quick") && !strings.Contains(joined, "wg") {
		t.Fatalf("expected wg-quick or wg in paths: %v", paths)
	}
}

func TestBuildSudoersContent(t *testing.T) {
	content := buildSudoersContent("testuser")
	if !strings.Contains(content, "testuser") {
		t.Fatalf("missing username: %s", content)
	}
	if !strings.Contains(content, "NOPASSWD") {
		t.Fatalf("missing NOPASSWD: %s", content)
	}
}
