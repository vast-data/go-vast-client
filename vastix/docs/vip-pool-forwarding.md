# VIP Pool Forwarding

## Overview

VIP Pool Forwarding provides secure, encrypted access to VAST cluster VIP pool IPs through a WireGuard VPN tunnel. This feature is useful when you need direct network access to VIP pool addresses that are not routable from your local machine.

**How it works:**
1. Vastix automatically deploys a VPN server to the remote VAST node via SSH
2. Creates a WireGuard tunnel between your local machine and the remote server
3. Routes VIP pool IPs through the tunnel
4. Monitors connection health and auto-recovers on failures
5. Cleans up all resources (local and remote) when disconnected

## Prerequisites

### Local Machine (Where Vastix Runs)

#### 1. WireGuard Tools
WireGuard must be installed for managing VPN interfaces.

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install wireguard wireguard-tools
```

**CentOS/RHEL 8+:**
```bash
sudo dnf install wireguard-tools
```

**macOS:**
```bash
brew install wireguard-tools
```

**Verify installation:**
```bash
wg --version
wg-quick --version
```

#### 2. Sudo Privileges
Your user must have sudo access (for WireGuard interface management):
```bash
# Test sudo access
sudo -v
```

If you need passwordless sudo for WireGuard (recommended):
```bash
# Add to /etc/sudoers.d/wireguard
echo "$USER ALL=(ALL) NOPASSWD: /usr/bin/wg-quick, /usr/bin/wg" | sudo tee /etc/sudoers.d/wireguard
sudo chmod 0440 /etc/sudoers.d/wireguard
```

#### 3. Go Toolchain
Required for building Vastix:
```bash
# Install Go 1.21 or later
# Download from https://go.dev/dl/

# Verify installation
go version
```

#### 4. SSH Access
Ensure you can SSH to the remote VAST node:
```bash
# Test SSH connection
ssh <username>@<vast-node-ip>

# For passwordless access (recommended), set up SSH keys:
ssh-keygen -t ed25519 -C "vastix-vpn"
ssh-copy-id <username>@<vast-node-ip>
```

### Remote Machine (VAST Node)

#### 1. WireGuard (Automatically Installed)
**Vastix automatically installs WireGuard** on the remote system if it's not already present. Supported operating systems:
- Ubuntu/Debian (installs via `apt`)
- CentOS/RHEL/Rocky Linux (installs via `yum`)
- Custom kernels (installs `wireguard-go` userspace implementation)

**No manual installation required!**

If automatic installation fails, Vastix will show clear error messages with manual installation instructions.

**To verify WireGuard is installed (optional):**
```bash
# Check for kernel module
lsmod | grep wireguard

# Or check for wireguard-go (userspace)
which wireguard-go

# Or check wireguard tools
which wg
```

#### 2. Sudo Privileges (Required)
The remote user must have sudo access for:
- WireGuard interface management
- iptables/firewall rules

```bash
# Test sudo access on remote
sudo -v

# For passwordless sudo (recommended):
echo "<username> ALL=(ALL) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/vastix-vpn
sudo chmod 0440 /etc/sudoers.d/vastix-vpn
```

Ensure SSH server is running and accessible:
```bash
sudo systemctl status sshd
sudo systemctl enable sshd
sudo systemctl start sshd
```

#### 4. Firewall Configuration
If firewall is enabled, allow the VPN port (default: 51821):
```bash
# For firewalld (CentOS/RHEL)
sudo firewall-cmd --permanent --add-port=51821/udp
sudo firewall-cmd --reload

# For ufw (Ubuntu/Debian)
sudo ufw allow 51821/udp
```

#### 6. Network Access to VIP Pool
The remote VAST node must have network connectivity to the VIP pool IPs:
```bash
# Test connectivity to a VIP pool IP
ping -c 3 <vip-pool-ip>
```

## Installation

### 1. Build Vastix
```bash
cd /path/to/go-vast-client/vastix
make build

# Or install system-wide
sudo make install
```

### 2. Configure VAST Connection
Launch Vastix and add your VAST cluster profile:
```bash
vastix
# or
./bin/vastix
```

In the TUI:
1. Go to **Profiles** (default view)
2. Press `<n>` to create new profile
3. Fill in VAST cluster details (VMS IP, username, password)
4. Press `<enter>` to save

### 3. Configure SSH Connection
1. Navigate to **SSH Connections** (use `<tab>` or type the resource name)
2. Press `<n>` to create new SSH connection
3. Fill in details:
   - **Name**: Descriptive name (e.g., "vast-node-1")
   - **Host**: Remote VAST node IP address
   - **Port**: SSH port (default: 22)
   - **Username**: SSH username with sudo privileges
   - **Password or SSH Key**: Authentication method
4. Press `<enter>` to save

## Using VIP Pool Forwarding

### Quick Start

1. **Select SSH Connection**
   - Navigate to **SSH Connections**
   - Use arrow keys to select your connection
   - Press `<enter>` to view details

2. **Activate VIP Pool Forwarding**
   - **Option A (Shortcut)**: Press `<1>` for direct activation
   - **Option B (Menu)**: Press `<x>` → select "vip pool forwarding" → press `<enter>`

3. **Select VIP Pool**
   - Choose the VIP pool you want to access
   - Press `<enter>` to confirm

4. **Wait for Connection**
   - Vastix will:
     - Deploy VPN server to remote node
     - Start WireGuard tunnel
     - Configure routes
     - Monitor connection health
   - **Status**: "Connected" when ready

5. **Use VIP Pool IPs**
   - VIP pool IPs are now accessible from your local machine
   - Example:
     ```bash
     # In another terminal
     ping 172.21.101.1
     curl http://172.21.101.1:8080
     ssh admin@172.21.101.2
     ```

6. **Disconnect**
   - **Option A**: Press `<esc>` to return to SSH connections list
   - **Option B**: Press `<ctrl+c>` to quit Vastix
   - Both options automatically clean up all resources

### Connection Details

When connected, you'll see:
```
VPN Connection Log

VPN connection established successfully
Accessible through VPN:
  - VPN Gateway: 10.99.1.1
  - Private IPs (4 total):
    • 172.21.101.1
    • 172.21.101.2
    • 172.21.101.3
    • 172.21.101.4
```

### Health Monitoring

Vastix automatically monitors connection health every **12 seconds**:
- **SSH connection** to VPN server
- **VPN tunnel** (ping gateway)
- **VIP pool connectivity** (random IP ping)

If connection is lost:
- Error is displayed in the TUI
- Local WireGuard interface is automatically cleaned up
- Remote VPN server self-destructs after heartbeat timeout (12s)

### Logs

Monitor VPN activity in real-time:
```bash
# Auxiliary log (health checks, connection status)
tail -f ~/.vastix/logs/aux.log

# Application log (errors, warnings)
tail -f ~/.vastix/logs/app.log
```

## Network Details

### IP Addressing

| Component | IP Range | Purpose |
|-----------|----------|---------|
| VPN Gateway | 10.99.1.1 | Remote VPN server endpoint |
| VPN Client | 10.99.1.2 | Your local machine in VPN |
| VPN Network | 10.99.1.0/24 | WireGuard tunnel network |
| VIP Pool IPs | 172.21.x.x | VAST cluster VIP pool (example) |

### Port Usage

| Port | Protocol | Purpose |
|------|----------|---------|
| 51820-51899 | UDP | WireGuard VPN tunnels (one per connection) |
| 22 | TCP | SSH to remote VAST node |

### Routing

Vastix creates **specific /32 routes** for each VIP pool IP:
```bash
# Example routes (created automatically)
10.99.1.1/32 dev wgvastix
172.21.101.1/32 dev wgvastix
172.21.101.2/32 dev wgvastix
172.21.101.3/32 dev wgvastix
172.21.101.4/32 dev wgvastix
```

This ensures:
- Only VIP pool traffic goes through the VPN
- No conflicts with other VPNs or networks
- Minimal routing table changes

## Troubleshooting

### Connection Fails Immediately

**Error**: `failed to bring up WireGuard interface`

**Cause**: WireGuard not installed or not in PATH

**Solution**:
```bash
# Check WireGuard installation
wg --version
wg-quick --version

# Install if missing (see Prerequisites)
```

---

**Error**: `SSH connection failed`

**Cause**: Cannot reach remote host or authentication failed

**Solution**:
```bash
# Test SSH manually
ssh <username>@<remote-host>

# Check SSH connection in Vastix database
cd ~/.vastix
sqlite3 vastix.db "SELECT * FROM ssh_connections;"

# Verify password or SSH key is correct
```

---

**Error**: `failed to register client peer`

**Cause**: VPN server failed to start (check deployment logs)

**Solution**:
```bash
# Check deployment logs in Vastix (auxlog zone shows real-time progress)

# SSH to remote and check WireGuard installation
ssh <username>@<remote-host>
lsmod | grep wireguard   # Kernel module
which wireguard-go       # Or userspace implementation
which wg                 # WireGuard tools

# Check VPN server logs
cat /tmp/vastix_vpn/*/server.log

# Check if VPN server process is running
pgrep -a vpn-server

# Check if VPN server port is available
sudo ss -tulpn | grep 51821
```

### Connection Drops Unexpectedly

**Symptom**: "SSH connection to VPN server lost" or "VPN tunnel connection lost"

**Possible Causes**:
1. Network connectivity issue
2. Remote server crashed
3. Firewall blocked VPN port

**Diagnosis**:
```bash
# Check local WireGuard status
sudo wg show

# Check aux log for health check failures
tail -50 ~/.vastix/logs/aux.log | grep -i "health\|error"

# Test SSH connectivity
ssh <username>@<remote-host>
```

**Solution**:
- Verify network connectivity
- Check remote server is still running
- Ensure firewall allows UDP port 51821
- Reconnect when network is stable

### Local Interface Not Cleaned Up

**Symptom**: `wgvastix` interface remains after disconnection

**Cause**: Cleanup failed or Vastix crashed

**Solution**:
```bash
# Manual cleanup
sudo wg-quick down wgvastix

# Or remove interface directly
sudo ip link delete wgvastix

# Check for remaining routes
ip route | grep wgvastix
sudo ip route del <route> dev wgvastix  # If any remain
```

### Remote Server Not Cleaned Up

**Symptom**: VPN server process still running on remote

**Cause**: Heartbeat timeout didn't trigger or process hung

**Solution**:
```bash
# SSH to remote
ssh <username>@<remote-host>

# Check for VPN server processes
pgrep -a vpn-server

# Kill if found
sudo pkill -9 vpn-server

# Check WireGuard interfaces
ip link show | grep wg

# Remove if found
sudo ip link delete wg21  # Or wg22, wg23, etc.

# Check iptables rules
sudo iptables -L -n -v | grep 10.99.1

# Clean up if found
sudo iptables -D FORWARD -i wg21 -j ACCEPT
sudo iptables -D FORWARD -o wg21 -j ACCEPT
sudo iptables -t nat -D POSTROUTING -s 10.99.1.0/24 -o <interface> -j MASQUERADE
```

### Permission Denied Errors

**Error**: `permission denied` when creating WireGuard interface

**Cause**: Missing sudo privileges

**Solution**:
```bash
# Test sudo locally
sudo -v

# Set up passwordless sudo for WireGuard (see Prerequisites)
echo "$USER ALL=(ALL) NOPASSWD: /usr/bin/wg-quick, /usr/bin/wg" | sudo tee /etc/sudoers.d/wireguard
sudo chmod 0440 /etc/sudoers.d/wireguard
```

### VIP Pool IPs Not Reachable

**Error**: Can ping VPN gateway but not VIP pool IPs

**Cause**: Remote node doesn't have access to VIP pool network

**Solution**:
```bash
# SSH to remote
ssh <username>@<remote-host>

# Test connectivity to VIP pool
ping -c 3 <vip-pool-ip>

# Check routing on remote
ip route | grep <vip-pool-network>

# Verify iptables FORWARD rules
sudo iptables -L FORWARD -n -v
```

## Advanced Configuration

### Multiple Simultaneous Connections

You can only have **one active VPN connection** at a time. Starting a new connection automatically terminates the previous one. This is by design to avoid:
- Port conflicts (WireGuard ports)
- Interface conflicts (same `wgvastix` interface name)
- Route conflicts (overlapping VPN networks)

### Custom VPN Network

Currently, the VPN network (10.99.1.0/24) is hardcoded. To change it, you would need to modify the source code:

**File**: `vastix/internal/tui/widgets/vip_pool_forwarding.go`
```go
// Line ~700
ServerIP:     netip.MustParseAddr("10.99.1.1"),    // VPN gateway
ClientIP:     netip.MustParseAddr("10.99.1.2"),    // Your local IP
VpnNetworkStr: "10.99.1.0/24",                     // VPN subnet
```

### Persistent Connections

VPN connections do not persist across Vastix restarts. When you quit Vastix (or it crashes), all VPN resources are cleaned up. To maintain persistent access, you would need to:
1. Keep Vastix running (e.g., in `screen` or `tmux`)
2. Or implement a standalone VPN connection outside Vastix

## Security Considerations

### Encryption
- All VPN traffic uses **WireGuard** with modern cryptography (ChaCha20, Poly1305)
- Keys are generated dynamically for each connection
- Keys are stored temporarily and cleaned up on disconnect

### Authentication
- SSH authentication (password or key) required
- Sudo access required (but limited to WireGuard/iptables commands)

### Network Isolation
- VPN traffic is isolated in a dedicated network (10.99.1.0/24)
- Only specific VIP pool IPs are routed (no broad routing)
- No split tunneling (default route unchanged)

### Cleanup
- Automatic cleanup on disconnect (local and remote)
- Self-destruction on connection loss (12s heartbeat timeout)
- No persistent state or credentials

## Architecture

```
┌─────────────────┐         SSH (TCP/22)           ┌─────────────────┐
│  Local Machine  │◄───────────────────────────────►│  Remote Node    │
│   (Vastix TUI)  │                                 │   (VPN Server)  │
└────────┬────────┘                                 └────────┬────────┘
         │                                                   │
         │ WireGuard (UDP/51821)                            │
         │ Encrypted Tunnel                                 │
         │◄─────────────────────────────────────────────────►│
         │                                                   │
         │ 10.99.1.2                              10.99.1.1 │
         │                                                   │
         │                                                   │
         │  Access VIP Pool IPs:                            │
         │  172.21.101.1 ◄──────────────────────────────────┤
         │  172.21.101.2 ◄──────────────────────────────────┤
         │  172.21.101.3 ◄──────────────────────────────────┤
         │  172.21.101.4 ◄──────────────────────────────────┤
         │                                                   │
         │                                          ┌────────▼────────┐
         │                                          │  VAST Cluster   │
         │                                          │  (VIP Pool)     │
         │                                          └─────────────────┘
         │
    wgvastix interface
    (local WireGuard)
```

## FAQ

**Q: Do I need to install anything on the VAST cluster itself?**  
A: No, the VAST cluster doesn't need any special software. You only need WireGuard/Go on the remote **VAST node** (not the cluster).

**Q: Can I use this feature without sudo?**  
A: No, sudo is required for WireGuard interface management (both local and remote).

**Q: Does this work with VPN already running (e.g., corporate VPN)?**  
A: Yes, as long as there's no IP/route conflict. Vastix uses specific /32 routes to avoid conflicts.

**Q: What happens if my laptop goes to sleep?**  
A: The connection will be lost. Health monitoring will detect it, auto-cleanup will run, and you'll need to reconnect.

**Q: Can I access VAST GUI through this VPN?**  
A: Yes, if the VIP pool IP is included in the VIP pool. Just open `http://<vip-pool-ip>` in your browser.

**Q: How do I know which VIP pool IPs are available?**  
A: Vastix queries the VAST API to get VIP pool IPs. They're displayed in the VIP pool selection list and connection log.

**Q: Can I use this with multiple VAST clusters?**  
A: Yes, create separate SSH connections for each cluster and connect to the one you need.

**Q: What's the performance/latency impact?**  
A: WireGuard has minimal overhead (~5-10%). Latency depends on your network connection to the remote node.

## Getting Help

- **Logs**: `~/.vastix/logs/aux.log` and `~/.vastix/logs/app.log`
- **Database**: `~/.vastix/vastix.db` (SQLite)
- **GitHub Issues**: Report bugs or request features
- **Check Remote Server**: SSH to the node and check `/tmp/vastix_vpn/*/server.log`

## Related Documentation

- [WireGuard Documentation](https://www.wireguard.com/)
- [VAST Data Platform Documentation](https://support.vastdata.com/)
- [Go SSH Library](https://pkg.go.dev/golang.org/x/crypto/ssh)

