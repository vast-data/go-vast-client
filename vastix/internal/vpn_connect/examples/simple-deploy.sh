#!/bin/bash
# Simple VPN Deployment Script
# This script deploys a VPN server and provides connection info

set -e

# Configuration
REMOTE_HOST="${REMOTE_HOST:-10.27.14.107}"
REMOTE_USER="${REMOTE_USER:-centos}"
REMOTE_PASSWORD="${REMOTE_PASSWORD}"
REMOTE_KEY="${REMOTE_KEY:-$HOME/.ssh/id_rsa}"
CLIENT_ID="${CLIENT_ID:-1}"

# Derived configuration
VPN_NETWORK="10.99.${CLIENT_ID}.0/24"
VPN_SERVER_IP="10.99.${CLIENT_ID}.1"
VPN_CLIENT_IP="10.99.${CLIENT_ID}.2"
VPN_PORT=$((51820 + CLIENT_ID))
PRIVATE_NETWORK="${PRIVATE_NETWORK:-172.21.101.0/24}"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if vpn-client exists
if [ ! -f "./build/vpn-client" ] && [ ! -f "./vpn-client" ]; then
    echo -e "${RED}Error: vpn-client not found. Run 'make build' first.${NC}"
    exit 1
fi

VPN_CLIENT="./build/vpn-client"
if [ ! -f "$VPN_CLIENT" ]; then
    VPN_CLIENT="./vpn-client"
fi

echo -e "${GREEN}=== VPN Deployment Script ===${NC}"
echo "Client ID: $CLIENT_ID"
echo "Remote Host: $REMOTE_HOST"
echo "VPN Network: $VPN_NETWORK"
echo "VPN Port: $VPN_PORT"
echo ""

# Determine authentication method
AUTH_ARGS=""
if [ -n "$REMOTE_PASSWORD" ]; then
    AUTH_ARGS="-remote-password $REMOTE_PASSWORD"
elif [ -f "$REMOTE_KEY" ]; then
    AUTH_ARGS="-remote-key $REMOTE_KEY"
else
    echo -e "${RED}Error: No authentication method provided.${NC}"
    echo "Set REMOTE_PASSWORD or ensure SSH key exists at $REMOTE_KEY"
    exit 1
fi

# Deploy
echo -e "${YELLOW}Step 1: Deploying VPN server...${NC}"
$VPN_CLIENT -mode deploy \
    -remote-host "$REMOTE_HOST" \
    -remote-user "$REMOTE_USER" \
    $AUTH_ARGS \
    -vpn-server-ip "$VPN_SERVER_IP" \
    -vpn-network "$VPN_NETWORK" \
    -vpn-port "$VPN_PORT" \
    -private-network "$PRIVATE_NETWORK" \
    > "deployment-client-${CLIENT_ID}.log" 2>&1

if [ $? -ne 0 ]; then
    echo -e "${RED}Deployment failed. Check deployment-client-${CLIENT_ID}.log${NC}"
    cat "deployment-client-${CLIENT_ID}.log"
    exit 1
fi

# Extract server public key
SERVER_KEY=$(grep "Server Public Key:" "deployment-client-${CLIENT_ID}.log" | awk '{print $4}')

if [ -z "$SERVER_KEY" ]; then
    echo -e "${RED}Failed to extract server public key${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Deployment successful${NC}"
echo "Server Public Key: $SERVER_KEY"
echo ""

# Start server
echo -e "${YELLOW}Step 2: Starting VPN server...${NC}"
$VPN_CLIENT -mode start-remote \
    -remote-host "$REMOTE_HOST" \
    -remote-user "$REMOTE_USER" \
    $AUTH_ARGS \
    -vpn-port "$VPN_PORT"

if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to start server${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Server started${NC}"
echo ""

# Generate client configuration file
CONFIG_FILE="client-${CLIENT_ID}-config.sh"
cat > "$CONFIG_FILE" <<EOF
#!/bin/bash
# VPN Client Configuration for Client ${CLIENT_ID}
# Generated on $(date)

export VPN_SERVER_ENDPOINT="${REMOTE_HOST}:${VPN_PORT}"
export VPN_SERVER_KEY="${SERVER_KEY}"
export VPN_CLIENT_IP="${VPN_CLIENT_IP}"
export VPN_SERVER_IP="${VPN_SERVER_IP}"
export VPN_PRIVATE_NETWORK="${PRIVATE_NETWORK}"

# Connect command:
$VPN_CLIENT -mode connect \\
  -server \${VPN_SERVER_ENDPOINT} \\
  -server-key "\${VPN_SERVER_KEY}" \\
  -client-ip \${VPN_CLIENT_IP} \\
  -server-ip \${VPN_SERVER_IP} \\
  -private-network \${VPN_PRIVATE_NETWORK}
EOF

chmod +x "$CONFIG_FILE"

# Print summary
echo -e "${GREEN}=== Deployment Complete ===${NC}"
echo ""
echo "Configuration saved to: $CONFIG_FILE"
echo ""
echo "To connect from this machine:"
echo -e "${YELLOW}  source $CONFIG_FILE${NC}"
echo ""
echo "Or run directly:"
echo -e "${YELLOW}  $VPN_CLIENT -mode connect \\"
echo "    -server ${REMOTE_HOST}:${VPN_PORT} \\"
echo "    -server-key \"${SERVER_KEY}\" \\"
echo "    -client-ip ${VPN_CLIENT_IP} \\"
echo "    -server-ip ${VPN_SERVER_IP}${NC}"
echo ""
echo "After connecting, you can access the private network:"
echo -e "${YELLOW}  ping ${PRIVATE_NETWORK%/*}1${NC}"
echo ""
echo "To stop the server:"
echo -e "${YELLOW}  $VPN_CLIENT -mode stop-remote -remote-host $REMOTE_HOST $AUTH_ARGS -vpn-port $VPN_PORT${NC}"

