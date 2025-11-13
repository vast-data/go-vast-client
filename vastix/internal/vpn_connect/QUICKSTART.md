# VPN Connect - Quick Start Guide

Get started with VPN Connect in 5 minutes!

## What You Need

- Remote server with SSH access (e.g., `10.27.14.107`)
- Go 1.22+ installed locally
- Access credentials (SSH password or key)

## Step-by-Step Setup

### 1. Build the Client Tool

```bash
cd /home/fnn45/VastData/go-vast-client/vastix/vpn_connect
go build -o vpn-client ./client/cmd
```

### 2. Deploy VPN Server

```bash
./vpn-client -mode deploy \
  -remote-host 10.27.14.107 \
  -remote-user centos \
  -remote-password "your-password" \
  -vpn-server-ip 10.99.1.1 \
  -vpn-network 10.99.1.0/24 \
  -vpn-port 51821 \
  -private-network 172.21.101.0/24
```

**Output Example:**
```
============================================================
VPN Server Deployed Successfully!
============================================================
Remote Host:       10.27.14.107
Server Public Key: G6irfdBQO8dVanH8L33Qm9hWGrVcCW5wIew8ucSR+lU=
Server VPN IP:     10.99.1.1
VPN Port:          51821
============================================================
```

**Save the Server Public Key!** You'll need it for connecting.

### 3. Start the Server

```bash
./vpn-client -mode start-remote \
  -remote-host 10.27.14.107 \
  -remote-password "your-password" \
  -vpn-port 51821
```

**Output:**
```
✓ VPN Server started successfully (PID: 12345)
```

### 4. Connect from Your Local Machine

```bash
./vpn-client -mode connect \
  -server 10.27.14.107:51821 \
  -server-key "G6irfdBQO8dVanH8L33Qm9hWGrVcCW5wIew8ucSR+lU=" \
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

### 5. Test the Connection

Open a new terminal and test:

```bash
# Ping VPN server
ping 10.99.1.1

# Ping private network (e.g., NFS server)
ping 172.21.101.1

# Mount NFS share
sudo mkdir -p /mnt/nfs
sudo mount -t nfs 172.21.101.1:/test /mnt/nfs

# Verify files
ls /mnt/nfs/
```

**Success!** 🎉 You're now connected to the private network via VPN!

---

## Common Use Cases

### Use Case 1: Access NFS Server

```bash
# Mount NFS share through VPN
sudo mount -t nfs 172.21.101.1:/data /mnt/nfs-data

# Access files
cd /mnt/nfs-data
ls -la
```

### Use Case 2: Access Private Web Service

```bash
# Connect via VPN (keep running in one terminal)
./vpn-client -mode connect ...

# In another terminal, access private service
curl http://172.21.101.2:8080
```

### Use Case 3: SSH to Private Server

```bash
# Connect via VPN
./vpn-client -mode connect ...

# SSH through VPN
ssh user@172.21.101.3
```

---

## Multiple Clients

To support multiple simultaneous clients, use different VPN networks and ports:

**Client 1:**
```bash
# Deploy
./vpn-client -mode deploy \
  -remote-host 10.27.14.107 \
  -remote-password "password" \
  -vpn-server-ip 10.99.1.1 \
  -vpn-network 10.99.1.0/24 \
  -vpn-port 51821

# Connect
./vpn-client -mode connect \
  -server 10.27.14.107:51821 \
  -server-key "<KEY1>" \
  -client-ip 10.99.1.2 \
  -server-ip 10.99.1.1
```

**Client 2:**
```bash
# Deploy
./vpn-client -mode deploy \
  -remote-host 10.27.14.107 \
  -remote-password "password" \
  -vpn-server-ip 10.99.2.1 \
  -vpn-network 10.99.2.0/24 \
  -vpn-port 51822

# Connect
./vpn-client -mode connect \
  -server 10.27.14.107:51822 \
  -server-key "<KEY2>" \
  -client-ip 10.99.2.2 \
  -server-ip 10.99.2.1
```

---

## Management Commands

### Check Server Status

```bash
./vpn-client -mode status-remote \
  -remote-host 10.27.14.107 \
  -remote-password "password" \
  -vpn-port 51821
```

### Stop Server

```bash
./vpn-client -mode stop-remote \
  -remote-host 10.27.14.107 \
  -remote-password "password" \
  -vpn-port 51821
```

### Restart Server

```bash
# Stop
./vpn-client -mode stop-remote \
  -remote-host 10.27.14.107 \
  -remote-password "password" \
  -vpn-port 51821

# Start
./vpn-client -mode start-remote \
  -remote-host 10.27.14.107 \
  -remote-password "password" \
  -vpn-port 51821
```

---

## Troubleshooting

### Can't Connect to Server

**Check if server is running:**
```bash
./vpn-client -mode status-remote \
  -remote-host 10.27.14.107 \
  -remote-password "password" \
  -vpn-port 51821
```

**Check network connectivity:**
```bash
# Test SSH
ssh centos@10.27.14.107

# Test UDP port
nc -zvu 10.27.14.107 51821
```

### Can't Access Private Network

**From local machine, after connecting:**
```bash
# Test VPN connectivity
ping 10.99.1.1

# Check routes
ip route show | grep 172.21

# Test private network
ping 172.21.101.1
```

**On remote server:**
```bash
ssh centos@10.27.14.107

# Check IP forwarding
sysctl net.ipv4.ip_forward

# Check NAT rules
sudo iptables -t nat -L -n -v
```

### VPN Connection Drops

**Use verbose logging:**
```bash
./vpn-client -mode connect -verbose \
  -server 10.27.14.107:51821 \
  -server-key "..." \
  -client-ip 10.99.1.2 \
  -server-ip 10.99.1.1
```

---

## Using SSH Keys Instead of Passwords

For better security, use SSH keys:

```bash
# Generate SSH key (if you don't have one)
ssh-keygen -t ed25519 -f ~/.ssh/vpn-deploy

# Copy to remote server
ssh-copy-id -i ~/.ssh/vpn-deploy.pub centos@10.27.14.107

# Deploy using SSH key
./vpn-client -mode deploy \
  -remote-host 10.27.14.107 \
  -remote-user centos \
  -remote-key ~/.ssh/vpn-deploy \
  -vpn-server-ip 10.99.1.1 \
  -vpn-network 10.99.1.0/24 \
  -vpn-port 51821
```

---

## Integration with Your TUI Application

### Example: Programmatic VPN Management

```go
package main

import (
    "context"
    "log"
    "net/netip"

    "github.com/vastdata/go-vast-client/vastix/vpn_connect/client"
    "github.com/vastdata/go-vast-client/vastix/vpn_connect/common"
)

func setupVPN(clientID int) error {
    // Generate unique network for this client
    vpnNetwork, serverIP, clientIP, _ := common.GenerateVPNNetwork(clientID)
    port := common.GetListenPort(clientID)

    // Deploy server
    deployer := client.NewDeployer(nil)
    
    deployConfig := &client.DeploymentConfig{
        Host:          "10.27.14.107",
        Port:          22,
        Username:      "centos",
        Password:      "your-password",
        RemoteWorkDir: "/tmp/vpn-" + string(clientID),
    }

    ctx := context.Background()
    if err := deployer.Connect(ctx, deployConfig); err != nil {
        return err
    }
    defer deployer.Disconnect()

    // Generate keys
    privKey, pubKey, _ := common.GenerateKeyPair()

    serverConfig := &common.ServerConfig{
        PrivateKey:     privKey,
        PublicKey:      pubKey,
        ListenPort:     port,
        ServerIP:       serverIP,
        VPNNetwork:     vpnNetwork,
        PrivateNetwork: netip.MustParsePrefix("172.21.101.0/24"),
    }

    // Deploy and start
    if err := deployer.Deploy(ctx, deployConfig, serverConfig); err != nil {
        return err
    }

    if err := deployer.StartServer(deployConfig.RemoteWorkDir, port); err != nil {
        return err
    }

    log.Printf("VPN setup complete! Server key: %s", pubKey)
    
    // Now create client and connect...
    clientConfig := &common.ClientConfig{
        PrivateKey:      "...", // Generate client keys
        PublicKey:       "...",
        ServerPublicKey: pubKey,
        ServerEndpoint:  "10.27.14.107:" + string(port),
        ClientIP:        clientIP,
        ServerIP:        serverIP,
        PrivateNetwork:  netip.MustParsePrefix("172.21.101.0/24"),
    }

    vpnClient, _ := client.NewClient(clientConfig, nil)
    return vpnClient.Connect(ctx)
}

func main() {
    if err := setupVPN(1); err != nil {
        log.Fatal(err)
    }
    
    // VPN is now active and you can access the private network!
}
```

---

## Next Steps

- **Read the full documentation**: See `docs/README.md`
- **API Reference**: Check `docs/API.md` for Go library usage
- **CLI Guide**: Full CLI options in `docs/CLI.md`
- **Production Deployment**: See `docs/DEPLOYMENT.md`

---

## Quick Reference Card

```bash
# Deploy
./vpn-client -mode deploy -remote-host <HOST> -remote-password <PASS> \
  -vpn-server-ip <IP> -vpn-network <CIDR> -vpn-port <PORT>

# Start
./vpn-client -mode start-remote -remote-host <HOST> -remote-password <PASS> \
  -vpn-port <PORT>

# Connect
./vpn-client -mode connect -server <HOST:PORT> -server-key <KEY> \
  -client-ip <IP> -server-ip <IP>

# Status
./vpn-client -mode status-remote -remote-host <HOST> -remote-password <PASS> \
  -vpn-port <PORT>

# Stop
./vpn-client -mode stop-remote -remote-host <HOST> -remote-password <PASS> \
  -vpn-port <PORT>
```

---

## Get Help

- Check logs: `tail -f /tmp/vpn-server/server.log` (on remote)
- Use `-verbose` flag for detailed output
- Review error messages carefully
- Ensure firewall allows VPN ports (UDP)

**Happy VPN'ing!** 🚀

