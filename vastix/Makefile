# Makefile for Vastix TUI Application

.PHONY: build build-all build-release run run-debug clean help test test-verbose test-coverage test-bench test-inputs install uninstall package release build-vpn-binaries clean-vpn-binaries clean-dist

# Build VPN server binaries for embedding
build-vpn-binaries:
	@echo "Building VPN server binaries for all platforms..."
	@./scripts/build-vpn-binaries.sh

# Build the vastix TUI application (includes VPN binaries)
build: bin build-vpn-binaries
	@echo "Building vastix..."
	@go build -o bin/vastix cmd/main.go

# Build binaries for all platforms (basic - no VPN binaries embedded)
build-all: bin
	@echo "Building vastix for all platforms..."
	@echo "Building for Linux amd64..."
	@GOOS=linux GOARCH=amd64 go build -o bin/vastix-linux-amd64 cmd/main.go
	@echo "Building for Linux arm64..."
	@GOOS=linux GOARCH=arm64 go build -o bin/vastix-linux-arm64 cmd/main.go
	@echo "Building for Windows amd64..."
	@GOOS=windows GOARCH=amd64 go build -o bin/vastix-windows-amd64.exe cmd/main.go
	@echo "Building for Windows arm64..."
	@GOOS=windows GOARCH=arm64 go build -o bin/vastix-windows-arm64.exe cmd/main.go
	@echo "Building for macOS amd64..."
	@GOOS=darwin GOARCH=amd64 go build -o bin/vastix-darwin-amd64 cmd/main.go
	@echo "Building for macOS arm64..."
	@GOOS=darwin GOARCH=arm64 go build -o bin/vastix-darwin-arm64 cmd/main.go
	@echo "Cross-platform build complete. Binaries available in bin/ directory."

# Build release binaries for all platforms (with VPN binaries embedded and proper naming)
build-release: dist
	@echo "Building release binaries for all platforms..."
	@./scripts/build-multi-platform.sh

# Package all platform binaries into a zip archive
package: build-all
	@echo "Creating release package..."
	@rm -f vastix-*.zip
	@zip -j vastix-$(shell date +%Y%m%d-%H%M%S).zip bin/vastix-linux-* bin/vastix-darwin-* bin/vastix-windows-*.exe README.md 2>/dev/null || zip -j vastix-$(shell date +%Y%m%d-%H%M%S).zip bin/vastix-linux-* bin/vastix-darwin-* bin/vastix-windows-*.exe
	@echo "Package created: vastix-$(shell date +%Y%m%d-%H%M%S).zip"

# Create a release package with version tag
release: build-all
	@echo "Creating versioned release package..."
	@rm -f vastix-*.zip
	@if [ -n "$(VERSION)" ]; then \
		zip -j vastix-$(VERSION).zip bin/vastix-linux-* bin/vastix-darwin-* bin/vastix-windows-*.exe README.md 2>/dev/null || zip -j vastix-$(VERSION).zip bin/vastix-linux-* bin/vastix-darwin-* bin/vastix-windows-*.exe; \
		echo "Release package created: vastix-$(VERSION).zip"; \
	else \
		echo "Error: VERSION not specified. Use: make release VERSION=v1.0.0"; \
		exit 1; \
	fi

# Run the vastix TUI application
run: build
	@echo "Starting vastix TUI..."
	@./bin/vastix

# Run the vastix TUI application with debug logging
run-debug: build
	@echo "Starting vastix TUI with debug logging..."
	@LOG_LEVEL=DEBUG ./bin/vastix

# Install the vastix binary to system as 'vx'
install: build
	@echo "Installing vastix as 'vx' to /usr/local/bin/..."
	@sudo cp bin/vastix /usr/local/bin/vx
	@sudo chmod +x /usr/local/bin/vx
	@echo "Installation complete. You can now run 'vx' from anywhere."

# Uninstall the vx binary from system
uninstall:
	@echo "Removing 'vx' from /usr/local/bin/..."
	@sudo rm -f /usr/local/bin/vx
	@echo "Uninstallation complete."

# Test the application
test:
	@echo "Running tests..."
	@go test ./...

# Test with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	@go test ./... -v

# Test with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@go test ./... -cover -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmark tests
test-bench:
	@echo "Running benchmark tests..."
	@go test ./... -bench=. -benchmem

# Test specific inputs functionality
test-inputs:
	@echo "Testing inputs ToParams functionality..."
	@go test ./internal/tui/widgets/common -v -run TestInputs_ToParams

# Clean VPN binaries
clean-vpn-binaries:
	@echo "Cleaning VPN server binaries..."
	@rm -rf internal/vpn_connect/client/binaries/

# Clean release binaries
clean-dist:
	@echo "Cleaning release binaries..."
	@rm -rf dist/

# Clean build artifacts
clean: clean-vpn-binaries clean-dist
	@echo "Cleaning up..."
	@rm -rf bin/ coverage.out coverage.html

# Show help
help:
	@echo "Available targets:"
	@echo "  build              - Build the vastix application (includes VPN binaries)"
	@echo "  build-vpn-binaries - Build VPN server binaries for all platforms"
	@echo "  build-all          - Build binaries for all platforms (Linux, Windows, macOS)"
	@echo "  build-release      - Build release binaries with VPN binaries embedded (for GitHub releases)"
	@echo "  run                - Build and run the vastix application"
	@echo "  run-debug          - Build and run with debug logging enabled"
	@echo "  install            - Install vastix as 'vx' to /usr/local/bin/ (requires sudo)"
	@echo "  uninstall          - Remove 'vx' from /usr/local/bin/ (requires sudo)"
	@echo "  package            - Build all platforms and create timestamped zip archive"
	@echo "  release            - Create versioned release zip (use: make release VERSION=v1.0.0)"
	@echo "  test               - Run all tests"
	@echo "  test-verbose       - Run tests with verbose output"
	@echo "  test-coverage      - Run tests with coverage report"
	@echo "  test-bench         - Run benchmark tests"
	@echo "  test-inputs        - Test inputs ToParams functionality specifically"
	@echo "  clean              - Remove all build artifacts and test outputs"
	@echo "  clean-vpn-binaries - Remove VPN server binaries"
	@echo "  clean-dist         - Remove release binaries from dist/"
	@echo "  help               - Show this help message"

# Create bin directory if it doesn't exist
bin:
	@mkdir -p bin

# Create dist directory if it doesn't exist
dist:
	@mkdir -p dist
