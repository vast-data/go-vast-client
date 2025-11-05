#!/bin/bash
# Build VPN server binaries for multiple platforms
# These binaries will be embedded in the main Vastix application

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VASTIX_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
OUTPUT_DIR="$VASTIX_ROOT/internal/vpn_connect/client/binaries"
SERVER_PACKAGE="./internal/vpn_connect/server/cmd"

echo -e "${GREEN}Building VPN server binaries...${NC}"
echo "Vastix root: $VASTIX_ROOT"
echo "Output directory: $OUTPUT_DIR"
echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Change to vastix directory for Go build
cd "$VASTIX_ROOT"

# Build function
build_binary() {
    local GOOS=$1
    local GOARCH=$2
    local OUTPUT_NAME="vpn-server-${GOOS}-${GOARCH}"
    
    echo -e "${YELLOW}Building ${OUTPUT_NAME}...${NC}"
    
    GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 go build \
        -ldflags "-s -w" \
        -o "$OUTPUT_DIR/$OUTPUT_NAME" \
        "$SERVER_PACKAGE"
    
    if [ $? -eq 0 ]; then
        SIZE=$(du -h "$OUTPUT_DIR/$OUTPUT_NAME" | cut -f1)
        echo -e "${GREEN}✓ Built ${OUTPUT_NAME} (${SIZE})${NC}"
    else
        echo -e "${RED}✗ Failed to build ${OUTPUT_NAME}${NC}"
        exit 1
    fi
}

# Build for all supported platforms
build_binary "linux" "amd64"
build_binary "linux" "arm64"
build_binary "darwin" "amd64"
build_binary "darwin" "arm64"

echo ""
echo -e "${GREEN}✓ All VPN server binaries built successfully${NC}"
echo ""
echo "Binaries:"
ls -lh "$OUTPUT_DIR/" | tail -n +2

echo ""
echo -e "${GREEN}These binaries will be embedded in the Vastix application${NC}"

