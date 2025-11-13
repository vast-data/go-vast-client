#!/bin/bash
# Multi-Client VPN Deployment Script
# Deploy VPN for multiple clients simultaneously

set -e

# Configuration
REMOTE_HOST="${REMOTE_HOST:-10.27.14.107}"
REMOTE_USER="${REMOTE_USER:-centos}"
REMOTE_PASSWORD="${REMOTE_PASSWORD}"
REMOTE_KEY="${REMOTE_KEY:-$HOME/.ssh/id_rsa}"
START_CLIENT_ID="${START_CLIENT_ID:-1}"
END_CLIENT_ID="${END_CLIENT_ID:-3}"
PRIVATE_NETWORK="${PRIVATE_NETWORK:-172.21.101.0/24}"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Check if vpn-client exists
if [ ! -f "./build/vpn-client" ] && [ ! -f "./vpn-client" ]; then
    echo -e "${RED}Error: vpn-client not found. Run 'make build' first.${NC}"
    exit 1
fi

VPN_CLIENT="./build/vpn-client"
if [ ! -f "$VPN_CLIENT" ]; then
    VPN_CLIENT="./vpn-client"
fi

# Determine authentication
AUTH_ARGS=""
if [ -n "$REMOTE_PASSWORD" ]; then
    AUTH_ARGS="-remote-password $REMOTE_PASSWORD"
elif [ -f "$REMOTE_KEY" ]; then
    AUTH_ARGS="-remote-key $REMOTE_KEY"
else
    echo -e "${RED}Error: No authentication method provided.${NC}"
    exit 1
fi

echo -e "${GREEN}=== Multi-Client VPN Deployment ===${NC}"
echo "Remote Host: $REMOTE_HOST"
echo "Deploying clients: $START_CLIENT_ID to $END_CLIENT_ID"
echo "Private Network: $PRIVATE_NETWORK"
echo ""

# Create output directory
mkdir -p deployments

# Deploy each client
for CLIENT_ID in $(seq $START_CLIENT_ID $END_CLIENT_ID); do
    VPN_NETWORK="10.99.${CLIENT_ID}.0/24"
    VPN_SERVER_IP="10.99.${CLIENT_ID}.1"
    VPN_CLIENT_IP="10.99.${CLIENT_ID}.2"
    VPN_PORT=$((51820 + CLIENT_ID))
    
    echo -e "${YELLOW}Deploying Client ${CLIENT_ID}...${NC}"
    echo "  Network: $VPN_NETWORK"
    echo "  Port: $VPN_PORT"
    
    # Deploy
    $VPN_CLIENT -mode deploy \
        -remote-host "$REMOTE_HOST" \
        -remote-user "$REMOTE_USER" \
        $AUTH_ARGS \
        -vpn-server-ip "$VPN_SERVER_IP" \
        -vpn-network "$VPN_NETWORK" \
        -vpn-port "$VPN_PORT" \
        -private-network "$PRIVATE_NETWORK" \
        > "deployments/client-${CLIENT_ID}.log" 2>&1
    
    if [ $? -ne 0 ]; then
        echo -e "${RED}  ✗ Deployment failed${NC}"
        continue
    fi
    
    # Extract server key
    SERVER_KEY=$(grep "Server Public Key:" "deployments/client-${CLIENT_ID}.log" | awk '{print $4}')
    
    # Start server
    $VPN_CLIENT -mode start-remote \
        -remote-host "$REMOTE_HOST" \
        -remote-user "$REMOTE_USER" \
        $AUTH_ARGS \
        -vpn-port "$VPN_PORT" > /dev/null 2>&1
    
    if [ $? -ne 0 ]; then
        echo -e "${RED}  ✗ Failed to start server${NC}"
        continue
    fi
    
    # Save configuration
    cat > "deployments/client-${CLIENT_ID}-connect.sh" <<EOF
#!/bin/bash
# VPN Client ${CLIENT_ID} Connection Script
$VPN_CLIENT -mode connect \\
  -server ${REMOTE_HOST}:${VPN_PORT} \\
  -server-key "${SERVER_KEY}" \\
  -client-ip ${VPN_CLIENT_IP} \\
  -server-ip ${VPN_SERVER_IP} \\
  -private-network ${PRIVATE_NETWORK}
EOF
    
    chmod +x "deployments/client-${CLIENT_ID}-connect.sh"
    
    echo -e "${GREEN}  ✓ Client ${CLIENT_ID} deployed successfully${NC}"
    echo "    Server Key: ${SERVER_KEY:0:20}..."
    echo "    Connect script: deployments/client-${CLIENT_ID}-connect.sh"
    echo ""
done

# Generate summary
SUMMARY_FILE="deployments/SUMMARY.md"
cat > "$SUMMARY_FILE" <<EOF
# Multi-Client VPN Deployment Summary

**Deployment Date**: $(date)
**Remote Host**: $REMOTE_HOST
**Private Network**: $PRIVATE_NETWORK

## Deployed Clients

EOF

for CLIENT_ID in $(seq $START_CLIENT_ID $END_CLIENT_ID); do
    VPN_PORT=$((51820 + CLIENT_ID))
    VPN_NETWORK="10.99.${CLIENT_ID}.0/24"
    
    if [ -f "deployments/client-${CLIENT_ID}-connect.sh" ]; then
        SERVER_KEY=$(grep "Server Public Key:" "deployments/client-${CLIENT_ID}.log" | awk '{print $4}')
        
        cat >> "$SUMMARY_FILE" <<EOF
### Client ${CLIENT_ID}

- **VPN Network**: $VPN_NETWORK
- **VPN Port**: $VPN_PORT
- **Server Public Key**: \`$SERVER_KEY\`
- **Connect Script**: \`./deployments/client-${CLIENT_ID}-connect.sh\`

**Connect Command**:
\`\`\`bash
$VPN_CLIENT -mode connect \\
  -server ${REMOTE_HOST}:${VPN_PORT} \\
  -server-key "$SERVER_KEY" \\
  -client-ip 10.99.${CLIENT_ID}.2 \\
  -server-ip 10.99.${CLIENT_ID}.1
\`\`\`

**Stop Command**:
\`\`\`bash
$VPN_CLIENT -mode stop-remote \\
  -remote-host $REMOTE_HOST \\
  $AUTH_ARGS \\
  -vpn-port $VPN_PORT
\`\`\`

---

EOF
    fi
done

echo -e "${GREEN}=== Deployment Complete ===${NC}"
echo ""
echo "Summary saved to: $SUMMARY_FILE"
echo ""
echo "Client connection scripts:"
for CLIENT_ID in $(seq $START_CLIENT_ID $END_CLIENT_ID); do
    if [ -f "deployments/client-${CLIENT_ID}-connect.sh" ]; then
        echo "  - deployments/client-${CLIENT_ID}-connect.sh"
    fi
done
echo ""
echo "To connect a client:"
echo -e "${YELLOW}  ./deployments/client-1-connect.sh${NC}"

