# VPN Connect - CLI Usage Guide

Complete command-line interface documentation for VPN Connect tools.

## Table of Contents

- [Installation](#installation)
- [Client Tool](#client-tool)
- [Server Tool](#server-tool)
- [Common Workflows](#common-workflows)
- [Configuration](#configuration)

## Installation

### Build from Source

```bash
cd /home/fnn45/VastData/go-vast-client/vastix/vpn_connect

# Build client tool
go build -o vpn-client ./client/cmd

# Build server tool (optional, can be auto-built during deployment)
go build -o vpn-server ./server/cmd
```

### Build Static Binary (for remote deployment)

```bash
# Build static binary that works across Linux distributions
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -o vpn-server-static \
  -ldflags="-s -w" \
  ./server/cmd
```

## Client Tool

The client tool (`vpn-client`) provides multiple operation modes.

### General Syntax

```bash
./vpn-client -mode <MODE> [OPTIONS]
```

### Modes

| Mode | Description |
|------|-------------|
| `connect` | Connect to a VPN server |
| `deploy` | Deploy server to remote machine |
| `start-remote` | Start remote VPN server |
| `stop-remote` | Stop remote VPN server |
| `status-remote` | Check remote server status |

---

### Mode: `deploy`

Deploys VPN server to a remote machine via SSH.

#### Required Flags

```bash
-mode deploy
-remote-host <HOST>          # Remote server IP/hostname
-vpn-server-ip <VPN_IP>      # Server's VPN IP (e.g., 10.99.1.1)
-vpn-network <CIDR>          # VPN network (e.g., 10.99.1.0/24)
```

#### Optional Flags

```bash
-remote-port <PORT>          # SSH port (default: 22)
-remote-user <USER>          # SSH username (default: centos)
-remote-password <PASSWORD>  # SSH password
-remote-key <PATH>           # SSH private key file
-remote-dir <PATH>           # Remote working directory (default: /tmp/vpn-server)
-vpn-port <PORT>             # VPN listen port (default: 51820)
-private-network <CIDR>      # Private network to route (default: 172.21.101.0/24)
-verbose                     # Enable verbose logging
```

#### Examples

**Deploy with password authentication:**
```bash
./vpn-client -mode deploy \
  -remote-host 10.27.14.107 \
  -remote-user centos \
  -remote-password "Kristin1109" \
  -vpn-server-ip 10.99.1.1 \
  -vpn-network 10.99.1.0/24 \
  -vpn-port 51821
```

**Deploy with SSH key:**
```bash
./vpn-client -mode deploy \
  -remote-host 10.27.14.107 \
  -remote-user centos \
  -remote-key ~/.ssh/id_rsa \
  -vpn-server-ip 10.99.1.1 \
  -vpn-network 10.99.1.0/24
```

**Deploy with custom private network:**
```bash
./vpn-client -mode deploy \
  -remote-host 10.27.14.107 \
  -remote-user centos \
  -remote-password "password" \
  -vpn-server-ip 10.99.2.1 \
  -vpn-network 10.99.2.0/24 \
  -private-network 192.168.1.0/24 \
  -vpn-port 51822
```

---

### Mode: `start-remote`

Starts the VPN server on a remote machine.

#### Required Flags

```bash
-mode start-remote
-remote-host <HOST>          # Remote server IP/hostname
-vpn-port <PORT>             # VPN port to start
```

#### Optional Flags

```bash
-remote-port <PORT>          # SSH port (default: 22)
-remote-user <USER>          # SSH username (default: centos)
-remote-password <PASSWORD>  # SSH password
-remote-key <PATH>           # SSH private key file
-remote-dir <PATH>           # Remote working directory (default: /tmp/vpn-server)
-verbose                     # Enable verbose logging
```

#### Example

```bash
./vpn-client -mode start-remote \
  -remote-host 10.27.14.107 \
  -remote-user centos \
  -remote-password "password" \
  -vpn-port 51821
```

---

### Mode: `stop-remote`

Stops the VPN server on a remote machine.

#### Required Flags

```bash
-mode stop-remote
-remote-host <HOST>          # Remote server IP/hostname
-vpn-port <PORT>             # VPN port to stop
```

#### Optional Flags

Same as `start-remote`.

#### Example

```bash
./vpn-client -mode stop-remote \
  -remote-host 10.27.14.107 \
  -remote-password "password" \
  -vpn-port 51821
```

---

### Mode: `status-remote`

Checks if VPN server is running on remote machine.

#### Required Flags

```bash
-mode status-remote
-remote-host <HOST>          # Remote server IP/hostname
-vpn-port <PORT>             # VPN port to check
```

#### Optional Flags

Same as `start-remote`.

#### Example

```bash
./vpn-client -mode status-remote \
  -remote-host 10.27.14.107 \
  -remote-password "password" \
  -vpn-port 51821
```

**Output:**
```
✓ VPN Server is running (PID: 12345, Port: 51821)
```

---

### Mode: `connect`

Connects to a VPN server.

#### Required Flags

```bash
-mode connect
-server <HOST:PORT>          # Server endpoint
-server-key <KEY>            # Server's public key
-client-ip <IP>              # Client's VPN IP (e.g., 10.99.1.2)
-server-ip <IP>              # Server's VPN IP (e.g., 10.99.1.1)
```

#### Optional Flags

```bash
-private-network <CIDR>      # Private network to access (default: 172.21.101.0/24)
-private-key <KEY>           # Client's private key (auto-generated if not provided)
-verbose                     # Enable verbose logging
```

#### Example

```bash
# Get server public key from deployment output
SERVER_KEY="G6irfdBQO8dVanH8L33Qm9hWGrVcCW5wIew8ucSR+lU="

./vpn-client -mode connect \
  -server 10.27.14.107:51821 \
  -server-key "$SERVER_KEY" \
  -client-ip 10.99.1.2 \
  -server-ip 10.99.1.1 \
  -private-network 172.21.101.0/24
```

**Output:**
```
Connected successfully!

Client Public Key: B+bKd/4IjwKOfPNfnqr/IkFZE3ufMrpM9t6WD7PmG08=
(Provide this to the server administrator to authorize your connection)
```

---

## Server Tool

The server tool (`vpn-server`) runs the VPN server directly (usually deployed automatically).

### Syntax

```bash
./vpn-server [OPTIONS]
```

### Flags

```bash
-port <PORT>                 # VPN listen port (default: 51820)
-server-ip <IP>              # Server VPN IP (required, e.g., 10.99.1.1)
-vpn-network <CIDR>          # VPN network (required, e.g., 10.99.1.0/24)
-private-network <CIDR>      # Private network to route (default: 172.21.101.0/24)
-interface <NAME>            # External interface (auto-detect if empty)
-private-key <KEY>           # Server private key (auto-generated if empty)
-verbose                     # Enable verbose logging
```

### Example

```bash
./vpn-server \
  -port 51821 \
  -server-ip 10.99.1.1 \
  -vpn-network 10.99.1.0/24 \
  -private-network 172.21.101.0/24 \
  -interface eno1
```

**Output:**
```
============================================================
VPN Server Started Successfully!
============================================================
Server Public Key: G6irfdBQO8dVanH8L33Qm9hWGrVcCW5wIew8ucSR+lU=
Server IP:         10.99.1.1
Listen Port:       51821
VPN Network:       10.99.1.0/24
Private Network:   172.21.101.0/24
External Interface: eno1
============================================================

Provide this public key to clients to connect.
Press Ctrl+C to stop the server.
```

---

## Common Workflows

### Workflow 1: Single Client VPN Setup

**Step 1: Deploy Server**
```bash
./vpn-client -mode deploy \
  -remote-host 10.27.14.107 \
  -remote-user centos \
  -remote-password "password" \
  -vpn-server-ip 10.99.1.1 \
  -vpn-network 10.99.1.0/24 \
  -vpn-port 51821
```

**Output:**
```
Server Public Key: G6irfdBQO8dVanH8L33Qm9hWGrVcCW5wIew8ucSR+lU=
```

**Step 2: Start Server**
```bash
./vpn-client -mode start-remote \
  -remote-host 10.27.14.107 \
  -remote-password "password" \
  -vpn-port 51821
```

**Step 3: Connect Client**
```bash
./vpn-client -mode connect \
  -server 10.27.14.107:51821 \
  -server-key "G6irfdBQO8dVanH8L33Qm9hWGrVcCW5wIew8ucSR+lU=" \
  -client-ip 10.99.1.2 \
  -server-ip 10.99.1.1
```

**Step 4: Use VPN (in another terminal)**
```bash
# Test connectivity
ping 10.99.1.1          # Ping VPN server
ping 172.21.101.1       # Ping private network

# Mount NFS
sudo mount -t nfs 172.21.101.1:/test /mnt/nfs
```

---

### Workflow 2: Multiple Concurrent Clients

Deploy separate VPN tunnels for multiple clients using different networks and ports.

**Client 1:**
```bash
# Deploy
./vpn-client -mode deploy \
  -remote-host 10.27.14.107 \
  -remote-password "password" \
  -vpn-server-ip 10.99.1.1 \
  -vpn-network 10.99.1.0/24 \
  -vpn-port 51821

# Start
./vpn-client -mode start-remote \
  -remote-host 10.27.14.107 \
  -remote-password "password" \
  -vpn-port 51821

# Connect (on client machine)
./vpn-client -mode connect \
  -server 10.27.14.107:51821 \
  -server-key "<SERVER_KEY_1>" \
  -client-ip 10.99.1.2 \
  -server-ip 10.99.1.1
```

**Client 2 (different network):**
```bash
# Deploy
./vpn-client -mode deploy \
  -remote-host 10.27.14.107 \
  -remote-password "password" \
  -vpn-server-ip 10.99.2.1 \
  -vpn-network 10.99.2.0/24 \
  -vpn-port 51822

# Start
./vpn-client -mode start-remote \
  -remote-host 10.27.14.107 \
  -remote-password "password" \
  -vpn-port 51822

# Connect (on different client machine)
./vpn-client -mode connect \
  -server 10.27.14.107:51822 \
  -server-key "<SERVER_KEY_2>" \
  -client-ip 10.99.2.2 \
  -server-ip 10.99.2.1
```

---

### Workflow 3: Programmatic Management

**Bash script to manage VPN lifecycle:**

```bash
#!/bin/bash

VPN_CLIENT="./vpn-client"
REMOTE_HOST="10.27.14.107"
REMOTE_USER="centos"
REMOTE_PASSWORD="password"
VPN_PORT=51821
VPN_SERVER_IP="10.99.1.1"
VPN_NETWORK="10.99.1.0/24"

# Deploy
echo "Deploying VPN server..."
$VPN_CLIENT -mode deploy \
  -remote-host $REMOTE_HOST \
  -remote-user $REMOTE_USER \
  -remote-password "$REMOTE_PASSWORD" \
  -vpn-port $VPN_PORT \
  -vpn-server-ip $VPN_SERVER_IP \
  -vpn-network $VPN_NETWORK > deploy.log 2>&1

# Extract server public key
SERVER_KEY=$(grep "Server Public Key:" deploy.log | awk '{print $4}')
echo "Server Key: $SERVER_KEY"

# Start server
echo "Starting VPN server..."
$VPN_CLIENT -mode start-remote \
  -remote-host $REMOTE_HOST \
  -remote-password "$REMOTE_PASSWORD" \
  -vpn-port $VPN_PORT

# Check status
$VPN_CLIENT -mode status-remote \
  -remote-host $REMOTE_HOST \
  -remote-password "$REMOTE_PASSWORD" \
  -vpn-port $VPN_PORT

echo "VPN setup complete!"
echo "To connect, run:"
echo "$VPN_CLIENT -mode connect -server $REMOTE_HOST:$VPN_PORT -server-key \"$SERVER_KEY\" -client-ip 10.99.1.2 -server-ip 10.99.1.1"
```

---

## Configuration

### Environment Variables

```bash
# Set log level
export VPN_LOG_LEVEL=debug

# Set configuration directory
export VPN_CONFIG_DIR=/var/lib/vpn
```

### Configuration Files

When using the server tool directly, you can create a config file:

**`/etc/vpn/server.conf`:**
```ini
[Server]
ListenPort = 51821
ServerIP = 10.99.1.1
VPNNetwork = 10.99.1.0/24
PrivateNetwork = 172.21.101.0/24
Interface = eno1
PrivateKey = <base64-encoded-key>
```

Then run:
```bash
./vpn-server -config /etc/vpn/server.conf
```

---

## Troubleshooting

### Check Server Status

```bash
# Via CLI
./vpn-client -mode status-remote \
  -remote-host 10.27.14.107 \
  -remote-password "password" \
  -vpn-port 51821

# Via SSH
ssh centos@10.27.14.107 "pgrep -f 'vpn-server.*51821'"
```

### View Server Logs

```bash
# SSH to remote
ssh centos@10.27.14.107

# View logs
tail -f /tmp/vpn-server/server.log

# Or check wireguard-go logs
journalctl -f | grep wireguard
```

### Test VPN Connectivity

```bash
# From client (after connecting)

# Test VPN interface
ping 10.99.1.1

# Test private network routing
ping 172.21.101.1

# Check routes
ip route show | grep 172.21

# Check firewall
sudo iptables -L -n -v
```

### Common Issues

**Issue: "Connection refused"**
```bash
# Check if server is running
./vpn-client -mode status-remote -remote-host <HOST> -vpn-port <PORT>

# Check firewall on remote
ssh <HOST> "sudo iptables -L -n | grep <PORT>"
```

**Issue: "Can't access private network"**
```bash
# Check NAT rules on server
ssh <HOST> "sudo iptables -t nat -L -n -v"

# Check IP forwarding
ssh <HOST> "sysctl net.ipv4.ip_forward"
```

**Issue: "Authentication failed"**
```bash
# Verify client public key is added to server
# Contact server administrator to add your public key
```

---

## Tips & Best Practices

1. **Use unique VPN networks for each client** to avoid conflicts
2. **Save server public keys securely** - you'll need them for client connections
3. **Use SSH keys instead of passwords** for production deployments
4. **Monitor server status** regularly with `status-remote`
5. **Keep WireGuard updated** on both client and server
6. **Use `-verbose` flag** when troubleshooting
7. **Document port assignments** when running multiple VPN servers

---

## Quick Reference

### Deploy & Start
```bash
./vpn-client -mode deploy -remote-host <HOST> -remote-password <PASS> -vpn-server-ip <IP> -vpn-network <CIDR> -vpn-port <PORT>
./vpn-client -mode start-remote -remote-host <HOST> -remote-password <PASS> -vpn-port <PORT>
```

### Connect
```bash
./vpn-client -mode connect -server <HOST:PORT> -server-key <KEY> -client-ip <IP> -server-ip <IP>
```

### Manage
```bash
./vpn-client -mode status-remote -remote-host <HOST> -remote-password <PASS> -vpn-port <PORT>
./vpn-client -mode stop-remote -remote-host <HOST> -remote-password <PASS> -vpn-port <PORT>
```

