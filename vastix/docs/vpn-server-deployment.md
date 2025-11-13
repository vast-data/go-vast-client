# VPN Server Deployment Architecture

## Overview

The VPN server deployment system uses pre-compiled binaries embedded in the Vastix application, eliminating the need for Go on remote systems and significantly speeding up deployment.

## Architecture

### Embedded Binaries

VPN server binaries are pre-compiled for multiple platforms and embedded directly in the Vastix binary using Go's `//go:embed` directive.

**Supported Platforms:**
- `linux/amd64` (x86_64)
- `linux/arm64` (aarch64)
- `darwin/amd64` (Intel Mac)
- `darwin/arm64` (Apple Silicon Mac)

**Location**: `internal/vpn_connect/client/binaries/`

### Build Process

#### 1. Build Script (`scripts/build-vpn-binaries.sh`)

Automatically builds VPN server binaries for all supported platforms:

```bash
# Build all binaries
make build-vpn-binaries

# Builds:
# - vpn-server-linux-amd64 (~2.7 MB)
# - vpn-server-linux-arm64 (~2.6 MB)  
# - vpn-server-darwin-amd64 (~2.7 MB)
# - vpn-server-darwin-arm64 (~2.6 MB)
```

#### 2. Integration with Main Build

The main `make build` command automatically builds VPN binaries first:

```makefile
build: bin build-vpn-binaries
    @echo "Building vastix..."
    @go build -o bin/vastix cmd/main.go
```

**Result**: Single Vastix binary (~39 MB) with all VPN server binaries embedded.

### Deployment Flow

```
1. User activates VIP Pool Forwarding
   ↓
2. Vastix connects to remote via SSH
   ↓
3. Detect remote OS (linux/darwin) and arch (amd64/arm64)
   ↓
4. Select appropriate embedded binary
   ↓
5. Check WireGuard availability
   ├─ Kernel module → ✓ Use it
   ├─ wireguard-go → ✓ Use it
   └─ Not found → Auto-install (if supported OS)
   ↓
6. Upload binary to remote
   ↓
7. Start VPN server
   ↓
8. Establish WireGuard tunnel
```

### OS Detection

**Remote OS Detection** (`detectRemoteOS`):
- Checks `/etc/os-release` for Linux distros
- Checks `sw_vers` for macOS
- Defaults to `linux` if uncertain

**Architecture Detection** (`detectRemoteArch`):
- Runs `uname -m` on remote
- Maps: `x86_64`/`amd64` → `amd64`, `aarch64`/`arm64` → `arm64`

### WireGuard Auto-Installation ✨

The deployer **automatically ensures WireGuard is available** on the remote system. This is a key feature that eliminates manual setup!

#### Detection & Installation Flow:

1. **Check for WireGuard kernel module** (fastest, best performance)
   ```bash
   sudo ip link add name wgtest type wireguard && sudo ip link del wgtest
   ```

2. **Check for wireguard-go** (userspace implementation)
   ```bash
   which wireguard-go
   ```

3. **Install if not found:**
   - Detects OS type and version
   - Installs wireguard-tools via package manager
   - If kernel module doesn't work (e.g., custom Lightbits kernel):
     - Clones wireguard-go from official git repository
     - Builds wireguard-go locally for Linux
     - Uploads binary to remote server
     - Installs to `/usr/local/bin/wireguard-go`

#### Auto-Installation Commands:

If WireGuard is not found, the deployer runs these commands automatically:

**Ubuntu/Debian:**
```bash
sudo apt-get update -qq && sudo apt-get install -y wireguard
```

**CentOS/Rocky Linux 7:**
```bash
sudo yum install -y epel-release elrepo-release && \
sudo yum install -y kmod-wireguard wireguard-tools
```

**CentOS/Rocky Linux 8+:**
```bash
sudo yum install -y epel-release elrepo-release && \
sudo yum install -y kmod-wireguard wireguard-tools
```

**Red Hat Enterprise Linux 7:**
```bash
sudo yum install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm \
    https://www.elrepo.org/elrepo-release-7.el7.elrepo.noarch.rpm && \
sudo yum install -y kmod-wireguard wireguard-tools
```

**Red Hat Enterprise Linux 8+:**
```bash
sudo yum install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm \
    https://www.elrepo.org/elrepo-release-8.el8.elrepo.noarch.rpm && \
sudo yum install -y kmod-wireguard wireguard-tools
```

**wireguard-go Fallback (Custom Kernels):**

For custom kernels (e.g., Lightbits) where kernel module isn't available:
```bash
# Automatically performed by Vastix:
# 1. Clone wireguard-go from git.zx2c4.com
# 2. Build for Linux (locally on your machine)
# 3. Upload to remote server
# 4. Install to /usr/local/bin/wireguard-go
```

**Unsupported OS:**
If the OS is not supported, a helpful error message is shown with:
- Manual installation instructions
- Link to https://www.wireguard.com/install/
- Specific commands for the detected OS

## File Structure

```
vastix/
├── scripts/
│   └── build-vpn-binaries.sh       # Builds all platform binaries
├── internal/
│   └── vpn_connect/
│       ├── server/
│       │   └── cmd/                # VPN server source code
│       └── client/
│           ├── deployer.go         # Deployment logic with embedded binaries
│           └── binaries/           # Generated binaries (gitignored)
│               ├── vpn-server-linux-amd64
│               ├── vpn-server-linux-arm64
│               ├── vpn-server-darwin-amd64
│               └── vpn-server-darwin-arm64
└── Makefile                         # Build orchestration
```

## Benefits

### 1. **No Go Required on Remote**
- ❌ **Before**: Remote system needed Go installed to build VPN server
- ✅ **After**: Only needs WireGuard (auto-installed if missing)

### 2. **Faster Deployment**
- ❌ **Before**: ~10-30 seconds to compile on remote
- ✅ **After**: <2 seconds to upload pre-compiled binary

### 3. **Offline Capable**
- ❌ **Before**: Needed internet on remote to download Go packages
- ✅ **After**: Everything embedded in Vastix binary

### 4. **Consistent Builds**
- ❌ **Before**: Different Go versions could cause issues
- ✅ **After**: Binaries built once in controlled environment

### 5. **Simplified Setup**
- ❌ **Before**: Manual WireGuard installation required
- ✅ **After**: Automatic installation for supported OSes

## Code Examples

### Embedding Binaries

```go
import _ "embed"

//go:embed binaries/vpn-server-linux-amd64
var vpnServerLinuxAmd64 []byte

//go:embed binaries/vpn-server-linux-arm64
var vpnServerLinuxArm64 []byte

//go:embed binaries/vpn-server-darwin-amd64
var vpnServerDarwinAmd64 []byte

//go:embed binaries/vpn-server-darwin-arm64
var vpnServerDarwinArm64 []byte
```

### Selecting Binary

```go
func (d *Deployer) getEmbeddedBinary(os, arch string) ([]byte, error) {
    key := fmt.Sprintf("%s-%s", os, arch)
    switch key {
    case "linux-amd64":
        return vpnServerLinuxAmd64, nil
    case "linux-arm64":
        return vpnServerLinuxArm64, nil
    case "darwin-amd64":
        return vpnServerDarwinAmd64, nil
    case "darwin-arm64":
        return vpnServerDarwinArm64, nil
    default:
        return nil, fmt.Errorf("unsupported platform: %s/%s", os, arch)
    }
}
```

### WireGuard Availability Check

```go
func (d *Deployer) ensureWireGuard(ctx context.Context) error {
    // Check kernel module
    if err := d.runCommand("lsmod | grep -q '^wireguard'"); err == nil {
        return nil // Kernel module available
    }

    // Check wireguard-go
    if err := d.runCommand("which wireguard-go"); err == nil {
        return nil // Userspace implementation available
    }

    // Check wg-quick
    if err := d.runCommand("which wg-quick"); err == nil {
        return nil // Tools available
    }

    // Auto-install
    return d.installWireGuard(ctx)
}
```

## Development

### Building Binaries Manually

```bash
# From vastix directory
./scripts/build-vpn-binaries.sh
```

### Testing with Different Platforms

To test platform detection without a real remote system:

```bash
# Mock OS detection
ssh remote "cat > /tmp/test-os.sh << 'EOF'
#!/bin/bash
echo 'ID=ubuntu'
echo 'VERSION_ID="22.04"'
EOF"
```

### Cleaning Binaries

```bash
make clean-vpn-binaries  # Remove generated binaries
make clean               # Remove all build artifacts
```

## Troubleshooting

### Binary Size Too Large

The embedded binaries add ~11 MB to the Vastix binary. If this is a concern:
- Binaries are already stripped (`-ldflags "-s -w"`)
- Consider UPX compression (not recommended for Go binaries)
- Only embed platforms you actually use

### Unsupported Platform Error

If you see: `unsupported platform: linux/arm`

The remote architecture is not supported. Currently supported:
- x86_64/amd64 (Intel/AMD 64-bit)
- aarch64/arm64 (ARM 64-bit)

32-bit architectures are not supported.

### WireGuard Installation Failed

If auto-installation fails, the error message will include:
- Manual installation instructions
- Link to https://www.wireguard.com/install/
- Specific commands for the detected OS

User should install WireGuard manually and retry.

## Security Considerations

### Binary Verification

The embedded binaries are built from source in a controlled environment. To verify:

```bash
# Check binary checksums
sha256sum internal/vpn_connect/client/binaries/*

# Rebuild from source
make clean-vpn-binaries
make build-vpn-binaries
```

### Sudo Requirements

WireGuard installation requires sudo on the remote system. The deployer:
- Uses `sudo` with `-y` flag for non-interactive installation
- Only runs package manager commands (apt/yum)
- Does not store or transmit sudo passwords

## Performance

### Build Times

| Operation | Time |
|-----------|------|
| Build all 4 VPN binaries | ~10 seconds |
| Build Vastix with embedded binaries | ~5 seconds |
| Total clean build | ~15 seconds |

### Deployment Times

| Stage | Time |
|-------|------|
| OS/arch detection | <1 second |
| WireGuard check | <1 second |
| WireGuard install (if needed) | 30-120 seconds |
| Binary upload | 1-2 seconds |
| VPN server start | <1 second |

**Total**: 2-5 seconds (or 32-122 seconds with first-time WireGuard install)

## Future Enhancements

Possible improvements:
1. Add support for 32-bit architectures (if needed)
2. Support for more OS flavors (Alpine, FreeBSD, etc.)
3. ~~wireguard-go auto-installation as fallback~~ ✅ **Implemented!**
4. Binary compression (experimental)
5. Incremental binary uploads (only if changed)
6. Pre-built wireguard-go binaries (avoid building locally)

## References

- [WireGuard Installation Guide](https://www.wireguard.com/install/)
- [Go embed Directive](https://pkg.go.dev/embed)
- [Cross-compilation in Go](https://golang.org/doc/install/source#environment)

