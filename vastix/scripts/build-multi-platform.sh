#!/bin/bash
# Build vastix for multiple platforms
# Usage: ./scripts/build-multi-platform.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VASTIX_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
OUTPUT_DIR="$VASTIX_ROOT/dist"
VERSION=$(cat "$VASTIX_ROOT/version" 2>/dev/null || echo "unknown")

echo -e "${GREEN}Building vastix for multiple platforms...${NC}"
echo "Vastix root: $VASTIX_ROOT"
echo "Output directory: $OUTPUT_DIR"
echo "Version: $VERSION"
echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Ensure VPN binaries are built first
echo -e "${BLUE}Step 1: Building VPN server binaries...${NC}"
"$SCRIPT_DIR/build-vpn-binaries.sh"
echo ""

# Build function
build_binary() {
    local GOOS=$1
    local GOARCH=$2
    local OUTPUT_NAME="vx-${GOOS}-${GOARCH}.${VERSION}"
    
    echo -e "${YELLOW}Building ${OUTPUT_NAME}...${NC}"
    
    cd "$VASTIX_ROOT"
    
    # Build with optimizations
    GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 go build \
        -ldflags "-s -w -X main.Version=$VERSION" \
        -o "$OUTPUT_DIR/$OUTPUT_NAME" \
        cmd/main.go
    
    if [ $? -ne 0 ]; then
        echo -e "${RED}✗ Failed to build ${OUTPUT_NAME}${NC}"
        return 1
    fi
    
    # Get file size
    if [ "$GOOS" = "darwin" ]; then
        SIZE=$(ls -lh "$OUTPUT_DIR/$OUTPUT_NAME" | awk '{print $5}')
    else
        SIZE=$(du -h "$OUTPUT_DIR/$OUTPUT_NAME" | cut -f1)
    fi
    
    echo -e "${GREEN}✓ Built ${OUTPUT_NAME} (${SIZE})${NC}"
    echo ""
}

echo -e "${BLUE}Step 2: Building vastix binaries for all platforms...${NC}"
echo ""

# Build for all supported platforms
build_binary "linux" "amd64"
build_binary "linux" "arm64"
build_binary "darwin" "amd64"
build_binary "darwin" "arm64"

echo -e "${GREEN}✓ All vastix binaries built successfully${NC}"
echo ""
echo "Binaries in $OUTPUT_DIR:"
ls -lh "$OUTPUT_DIR"/vx-* 2>/dev/null || echo "No binaries found"
echo ""
echo -e "${GREEN}Ready for release!${NC}"

