# VPN Connect - Production Deployment Guide

This guide covers best practices for deploying VPN Connect in production environments.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Prerequisites](#prerequisites)
- [Security Considerations](#security-considerations)
- [Deployment Strategies](#deployment-strategies)
- [Multi-Client Setup](#multi-client-setup)
- [Monitoring & Logging](#monitoring--logging)
- [Performance Tuning](#performance-tuning)
- [High Availability](#high-availability)
- [Troubleshooting](#troubleshooting)

---

## Architecture Overview

### Network Topology

```
Internet
   │
   ├─── Client 1 (10.99.1.2) ──┐
   │                           │
   ├─── Client 2 (10.99.2.2) ──┤
   │                           │    VPN Tunnels (encrypted)
   ├─── Client 3 (10.99.3.2) ──┤
   │                           │
   │                           ▼
   │                    Remote Server
   │                    (10.27.14.107)
   │                           │
   │                    VPN Server Process
   │                    (Multiple instances)
   │                      │    │    │
   │                      ▼    ▼    ▼
   │                    10.99.1.1  10.99.2.1  10.99.3.1
   │                           │
   │                           ▼
   │                    NAT + Routing
   │                           │
   │                           ▼
   │                    Private Network
   │                    (172.21.101.0/24)
   │                      │    │    │    │
   │                      ▼    ▼    ▼    ▼
   │                    .101  .102  .103  .104
   │                    (NFS Servers)
```

### Component Layout

- **VPN Server**: Runs on remote machine, handles VPN tunnels
- **VPN Client**: Runs on local machines, connects to server
- **Deployer**: SSH-based deployment automation
- **Private Network**: The protected network (e.g., NFS servers)

---

## Prerequisites

### Server Requirements

**Hardware:**
- CPU: 2+ cores
- RAM: 2 GB minimum (+ 50 MB per client)
- Network: 1 Gbps+ recommended
- Disk: 1 GB for server + logs

**Software:**
- Linux kernel 3.10+ (CentOS 7+, Ubuntu 18.04+)
- WireGuard kernel module OR wireguard-go
- `iptables` for NAT/routing
- `iproute2` (`ip` command)
- SSH server for remote deployment

**Network:**
- Public IP or accessible endpoint
- UDP port 51820+ open (firewall configured)
- IP forwarding enabled

### Client Requirements

**Hardware:**
- CPU: 1+ core
- RAM: 512 MB minimum
- Network: Any

**Software:**
- Linux, macOS, or Windows (with WireGuard)
- Go 1.22+ (if building from source)

---

## Security Considerations

### Key Management

**Generation:**
```bash
# Generate keys securely
./vpn-client -mode deploy ... # Keys auto-generated

# Or manually
go run ./common/cmd/keygen/main.go
```

**Storage:**
- Store private keys in secure location (e.g., `/etc/vpn/keys/`)
- Use file permissions: `chmod 600 private.key`
- Never commit keys to version control
- Rotate keys periodically (every 90 days recommended)

**Distribution:**
- Share public keys only via secure channels (encrypted email, secure chat)
- Use SSH key-based authentication for deployment
- Avoid password authentication in production

### Network Security

**Firewall Configuration (Server):**
```bash
# Allow VPN ports (example for 3 clients)
sudo ufw allow 51821/udp comment "VPN Client 1"
sudo ufw allow 51822/udp comment "VPN Client 2"
sudo ufw allow 51823/udp comment "VPN Client 3"

# Allow SSH (for management)
sudo ufw allow 22/tcp

# Enable firewall
sudo ufw enable
```

**iptables Rules:**
```bash
# Allow forwarding from VPN to private network
sudo iptables -A FORWARD -i wg+ -j ACCEPT
sudo iptables -A FORWARD -o wg+ -j ACCEPT

# NAT for VPN traffic
sudo iptables -t nat -A POSTROUTING -s 10.99.0.0/16 -o eno1 -j MASQUERADE

# Save rules
sudo iptables-save > /etc/iptables/rules.v4
```

**IP Forwarding:**
```bash
# Enable permanently
echo "net.ipv4.ip_forward = 1" | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

### Access Control

**Client Authorization:**
```go
// Server side: Only allow authorized clients
srv.AddClient(clientPubKey, clientIP, allowedIPs)

// Reject unauthorized clients by not adding them
```

**Network Segmentation:**
- Use different VPN networks per client (`10.99.X.0/24`)
- Restrict `AllowedIPs` to only necessary networks
- Implement least-privilege access

---

## Deployment Strategies

### Strategy 1: Automated Deployment

**Bash Script (`deploy-vpn.sh`):**
```bash
#!/bin/bash
set -e

# Configuration
REMOTE_HOST="10.27.14.107"
REMOTE_USER="centos"
SSH_KEY="~/.ssh/id_rsa"
CLIENT_ID=1

# Derive network from client ID
VPN_NETWORK="10.99.${CLIENT_ID}.0/24"
VPN_SERVER_IP="10.99.${CLIENT_ID}.1"
VPN_CLIENT_IP="10.99.${CLIENT_ID}.2"
VPN_PORT=$((51820 + CLIENT_ID))

# Deploy
echo "Deploying VPN for client ${CLIENT_ID}..."
./vpn-client -mode deploy \
  -remote-host "$REMOTE_HOST" \
  -remote-user "$REMOTE_USER" \
  -remote-key "$SSH_KEY" \
  -vpn-server-ip "$VPN_SERVER_IP" \
  -vpn-network "$VPN_NETWORK" \
  -vpn-port "$VPN_PORT" \
  > "deployment-client-${CLIENT_ID}.log" 2>&1

# Extract server public key
SERVER_KEY=$(grep "Server Public Key:" "deployment-client-${CLIENT_ID}.log" | awk '{print $4}')

# Start server
echo "Starting VPN server..."
./vpn-client -mode start-remote \
  -remote-host "$REMOTE_HOST" \
  -remote-key "$SSH_KEY" \
  -vpn-port "$VPN_PORT"

# Generate client config
cat > "client-${CLIENT_ID}.conf" <<EOF
# VPN Client Configuration for Client ${CLIENT_ID}
Server Endpoint: ${REMOTE_HOST}:${VPN_PORT}
Server Public Key: ${SERVER_KEY}
Client VPN IP: ${VPN_CLIENT_IP}
Server VPN IP: ${VPN_SERVER_IP}

# Connect command:
./vpn-client -mode connect \\
  -server ${REMOTE_HOST}:${VPN_PORT} \\
  -server-key "${SERVER_KEY}" \\
  -client-ip ${VPN_CLIENT_IP} \\
  -server-ip ${VPN_SERVER_IP}
EOF

echo "✓ Deployment complete!"
echo "✓ Configuration saved to: client-${CLIENT_ID}.conf"
```

**Usage:**
```bash
chmod +x deploy-vpn.sh

# Deploy for client 1
CLIENT_ID=1 ./deploy-vpn.sh

# Deploy for client 2
CLIENT_ID=2 ./deploy-vpn.sh
```

### Strategy 2: Ansible Playbook

**`vpn-deploy.yml`:**
```yaml
---
- name: Deploy VPN Server
  hosts: vpn_server
  vars:
    client_id: 1
    vpn_network: "10.99.{{ client_id }}.0/24"
    vpn_server_ip: "10.99.{{ client_id }}.1"
    vpn_port: "{{ 51820 + client_id }}"
  
  tasks:
    - name: Copy VPN binary
      copy:
        src: ./vpn-server
        dest: /opt/vpn/vpn-server-{{ client_id }}
        mode: '0755'
    
    - name: Generate VPN keys
      command: /opt/vpn/keygen
      register: vpn_keys
    
    - name: Start VPN server
      shell: |
        nohup /opt/vpn/vpn-server-{{ client_id }} \
          -port {{ vpn_port }} \
          -server-ip {{ vpn_server_ip }} \
          -vpn-network {{ vpn_network }} \
          > /var/log/vpn-{{ client_id }}.log 2>&1 &
      
    - name: Save server info
      copy:
        content: "{{ vpn_keys.stdout }}"
        dest: "/opt/vpn/client-{{ client_id }}-info.txt"
```

### Strategy 3: Docker Deployment

**`Dockerfile`:**
```dockerfile
FROM golang:1.22 AS builder

WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
  -o vpn-server \
  -ldflags="-s -w" \
  ./server/cmd

FROM alpine:latest

RUN apk add --no-cache \
  wireguard-tools \
  iptables \
  iproute2

COPY --from=builder /build/vpn-server /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/vpn-server"]
```

**`docker-compose.yml`:**
```yaml
version: '3.8'

services:
  vpn-client1:
    build: .
    container_name: vpn-client1
    cap_add:
      - NET_ADMIN
      - SYS_MODULE
    environment:
      - VPN_PORT=51821
      - VPN_SERVER_IP=10.99.1.1
      - VPN_NETWORK=10.99.1.0/24
    ports:
      - "51821:51821/udp"
    restart: unless-stopped

  vpn-client2:
    build: .
    container_name: vpn-client2
    cap_add:
      - NET_ADMIN
      - SYS_MODULE
    environment:
      - VPN_PORT=51822
      - VPN_SERVER_IP=10.99.2.1
      - VPN_NETWORK=10.99.2.0/24
    ports:
      - "51822:51822/udp"
    restart: unless-stopped
```

---

## Multi-Client Setup

### Planning

**Network Allocation:**
- Client 1: `10.99.1.0/24`, Port `51821`
- Client 2: `10.99.2.0/24`, Port `51822`
- Client 3: `10.99.3.0/24`, Port `51823`
- ... up to Client 254

**Port Allocation:**
```
Base Port: 51820
Client 1: 51821
Client 2: 51822
Client 3: 51823
...
Client N: 51820 + N
```

### Bulk Deployment

**Python Script (`deploy-multi.py`):**
```python
#!/usr/bin/env python3
import subprocess
import sys

REMOTE_HOST = "10.27.14.107"
REMOTE_USER = "centos"
SSH_KEY = "~/.ssh/id_rsa"

def deploy_client(client_id):
    vpn_network = f"10.99.{client_id}.0/24"
    vpn_server_ip = f"10.99.{client_id}.1"
    vpn_port = 51820 + client_id
    
    cmd = [
        "./vpn-client", "-mode", "deploy",
        "-remote-host", REMOTE_HOST,
        "-remote-user", REMOTE_USER,
        "-remote-key", SSH_KEY,
        "-vpn-server-ip", vpn_server_ip,
        "-vpn-network", vpn_network,
        "-vpn-port", str(vpn_port),
    ]
    
    print(f"Deploying client {client_id}...")
    result = subprocess.run(cmd, capture_output=True, text=True)
    
    if result.returncode == 0:
        print(f"✓ Client {client_id} deployed successfully")
        # Extract and save server key
        for line in result.stdout.split('\n'):
            if "Server Public Key:" in line:
                key = line.split(':')[1].strip()
                with open(f"client-{client_id}-key.txt", "w") as f:
                    f.write(key)
        return True
    else:
        print(f"✗ Client {client_id} deployment failed")
        print(result.stderr)
        return False

if __name__ == "__main__":
    start_id = int(sys.argv[1]) if len(sys.argv) > 1 else 1
    end_id = int(sys.argv[2]) if len(sys.argv) > 2 else 3
    
    for client_id in range(start_id, end_id + 1):
        deploy_client(client_id)
```

**Usage:**
```bash
chmod +x deploy-multi.py

# Deploy for clients 1-10
./deploy-multi.py 1 10
```

---

## Monitoring & Logging

### Server Monitoring

**Check all running VPN servers:**
```bash
ssh centos@10.27.14.107 "ps aux | grep vpn-server"
```

**Check specific server status:**
```bash
./vpn-client -mode status-remote \
  -remote-host 10.27.14.107 \
  -remote-key ~/.ssh/id_rsa \
  -vpn-port 51821
```

### Log Collection

**Centralized logging script:**
```bash
#!/bin/bash
# collect-logs.sh

REMOTE_HOST="10.27.14.107"
SSH_KEY="~/.ssh/id_rsa"
LOG_DIR="./vpn-logs-$(date +%Y%m%d)"

mkdir -p "$LOG_DIR"

for port in {51821..51830}; do
  echo "Collecting logs for port $port..."
  ssh -i "$SSH_KEY" centos@$REMOTE_HOST \
    "cat /tmp/vpn-server-$((port - 51820))/server.log 2>/dev/null" \
    > "$LOG_DIR/server-$port.log" || echo "No logs for port $port"
done

echo "Logs collected in $LOG_DIR/"
```

### Metrics Collection

**Go monitoring client:**
```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/vastdata/go-vast-client/vastix/vpn_connect/client"
)

func monitorVPN(c *client.Client) {
    ctx := context.Background()
    statsCh := c.MonitorConnection(ctx, 30*time.Second)
    
    for stats := range statsCh {
        fmt.Printf("[%s] Sent: %d bytes, Received: %d bytes, Uptime: %v\n",
            time.Now().Format("15:04:05"),
            stats.BytesSent,
            stats.BytesReceived,
            time.Since(stats.ConnectedAt))
    }
}
```

---

## Performance Tuning

### Server Optimization

**Kernel Parameters (`/etc/sysctl.conf`):**
```ini
# Increase network buffer sizes
net.core.rmem_max = 134217728
net.core.wmem_max = 134217728
net.ipv4.tcp_rmem = 4096 87380 67108864
net.ipv4.tcp_wmem = 4096 65536 67108864

# Enable BBR congestion control
net.core.default_qdisc = fq
net.ipv4.tcp_congestion_control = bbr

# Increase connection tracking
net.netfilter.nf_conntrack_max = 1000000

# IP forwarding
net.ipv4.ip_forward = 1
```

Apply:
```bash
sudo sysctl -p
```

### WireGuard Optimization

**MTU Tuning:**
```bash
# Set optimal MTU for WireGuard interface
ip link set dev wg0 mtu 1420
```

**Queue Length:**
```bash
# Increase interface queue length
ip link set dev wg0 txqueuelen 1000
```

### Client Optimization

**Persistent Keepalive:**
```go
// For NAT traversal, use 25-second keepalive
config := &common.ClientConfig{
    // ...
    PersistentKeepalive: 25,
}
```

---

## High Availability

### Load Balancing

Deploy multiple VPN servers behind a load balancer:

```
             Load Balancer (HAProxy)
                      │
        ┌─────────────┼─────────────┐
        ▼             ▼             ▼
    VPN Server 1  VPN Server 2  VPN Server 3
    (10.27.14.107) (10.27.14.108) (10.27.14.109)
```

**HAProxy Configuration:**
```
frontend vpn_frontend
    bind *:51820 proto udp
    default_backend vpn_servers

backend vpn_servers
    mode udp
    balance roundrobin
    server vpn1 10.27.14.107:51820 check
    server vpn2 10.27.14.108:51820 check
    server vpn3 10.27.14.109:51820 check
```

### Failover

**Health Check Script:**
```bash
#!/bin/bash
# vpn-healthcheck.sh

PRIMARY="10.27.14.107:51821"
BACKUP="10.27.14.108:51821"

check_vpn() {
    ./vpn-client -mode status-remote \
      -remote-host $(echo $1 | cut -d: -f1) \
      -vpn-port $(echo $1 | cut -d: -f2) \
      > /dev/null 2>&1
    return $?
}

if check_vpn "$PRIMARY"; then
    echo "PRIMARY"
elif check_vpn "$BACKUP"; then
    echo "BACKUP"
else
    echo "NONE"
    exit 1
fi
```

---

## Troubleshooting

### Common Issues

**Issue: "Port already in use"**
```bash
# Find process using port
sudo ss -tuln | grep 51821
sudo lsof -i :51821

# Kill process
sudo kill <PID>
```

**Issue: "Connection timeout"**
```bash
# Test UDP connectivity
nc -zvu 10.27.14.107 51821

# Check firewall
sudo iptables -L -n | grep 51821
sudo ufw status | grep 51821
```

**Issue: "Can't ping private network"**
```bash
# Check routing
ip route show | grep 172.21

# Check NAT
sudo iptables -t nat -L -n -v | grep 172.21

# Check IP forwarding
sysctl net.ipv4.ip_forward
```

### Debug Mode

```bash
# Server with verbose logging
./vpn-server -verbose

# Client with verbose logging
./vpn-client -mode connect -verbose ...
```

### Packet Capture

```bash
# Capture VPN traffic
sudo tcpdump -i any -n port 51821

# Capture private network traffic
sudo tcpdump -i wg0 -n
```

---

## Backup & Disaster Recovery

### Backup Keys

```bash
# Backup all keys
tar -czf vpn-keys-backup-$(date +%Y%m%d).tar.gz \
  /tmp/vpn-server-*/wg.conf \
  client-*-key.txt

# Encrypt backup
gpg -c vpn-keys-backup-*.tar.gz
```

### Restore Configuration

```bash
# Extract backup
gpg -d vpn-keys-backup-*.tar.gz.gpg | tar -xz

# Redeploy with existing keys
./vpn-client -mode deploy \
  -private-key "$(cat /path/to/server-private.key)" \
  ...
```

---

## Checklist

### Pre-Deployment
- [ ] Server has public IP and firewall configured
- [ ] SSH access to server verified
- [ ] Required software installed (WireGuard, iptables)
- [ ] Network ports allocated and documented
- [ ] Keys generated and stored securely

### Deployment
- [ ] VPN server deployed successfully
- [ ] Server started and verified running
- [ ] Client can connect to VPN
- [ ] Client can ping VPN server IP
- [ ] Client can access private network

### Post-Deployment
- [ ] Monitoring configured
- [ ] Logging centralized
- [ ] Backup scheduled
- [ ] Documentation updated
- [ ] Users notified with connection details

---

## Best Practices Summary

1. **Use unique networks** for each client (`10.99.X.0/24`)
2. **Automate deployment** with scripts or Ansible
3. **Monitor server health** with status checks
4. **Centralize logs** for troubleshooting
5. **Rotate keys** periodically (every 90 days)
6. **Test failover** before production use
7. **Document everything** (ports, networks, keys)
8. **Use SSH keys** for deployment, not passwords
9. **Enable IP forwarding** and NAT rules
10. **Keep software updated** (WireGuard, kernel)

