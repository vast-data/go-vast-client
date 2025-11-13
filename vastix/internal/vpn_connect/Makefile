# VPN Connect - Makefile

.PHONY: all build clean test install help client server static

# Variables
CLIENT_BIN := vpn-client
SERVER_BIN := vpn-server
CLIENT_CMD := ./client/cmd
SERVER_CMD := ./server/cmd
BUILD_DIR := ./build
INSTALL_DIR := /usr/local/bin

# Build flags
LDFLAGS := -s -w
GO_BUILD := go build -ldflags="$(LDFLAGS)"
GO_BUILD_STATIC := CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)"

# Default target
all: build

# Build client and server
build: client server

# Build client binary
client:
	@echo "Building VPN client..."
	@mkdir -p $(BUILD_DIR)
	$(GO_BUILD) -o $(BUILD_DIR)/$(CLIENT_BIN) $(CLIENT_CMD)
	@echo "✓ Client built: $(BUILD_DIR)/$(CLIENT_BIN)"

# Build server binary
server:
	@echo "Building VPN server..."
	@mkdir -p $(BUILD_DIR)
	$(GO_BUILD) -o $(BUILD_DIR)/$(SERVER_BIN) $(SERVER_CMD)
	@echo "✓ Server built: $(BUILD_DIR)/$(SERVER_BIN)"

# Build static binaries (for remote deployment)
static:
	@echo "Building static binaries..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO_BUILD_STATIC) -o $(BUILD_DIR)/$(CLIENT_BIN)-static $(CLIENT_CMD)
	GOOS=linux GOARCH=amd64 $(GO_BUILD_STATIC) -o $(BUILD_DIR)/$(SERVER_BIN)-static $(SERVER_CMD)
	@echo "✓ Static binaries built:"
	@echo "  - $(BUILD_DIR)/$(CLIENT_BIN)-static"
	@echo "  - $(BUILD_DIR)/$(SERVER_BIN)-static"

# Build for multiple platforms
cross-compile:
	@echo "Cross-compiling for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	# Linux amd64
	GOOS=linux GOARCH=amd64 $(GO_BUILD_STATIC) -o $(BUILD_DIR)/$(CLIENT_BIN)-linux-amd64 $(CLIENT_CMD)
	GOOS=linux GOARCH=amd64 $(GO_BUILD_STATIC) -o $(BUILD_DIR)/$(SERVER_BIN)-linux-amd64 $(SERVER_CMD)
	# Linux arm64
	GOOS=linux GOARCH=arm64 $(GO_BUILD_STATIC) -o $(BUILD_DIR)/$(CLIENT_BIN)-linux-arm64 $(CLIENT_CMD)
	GOOS=linux GOARCH=arm64 $(GO_BUILD_STATIC) -o $(BUILD_DIR)/$(SERVER_BIN)-linux-arm64 $(SERVER_CMD)
	# macOS amd64
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) -o $(BUILD_DIR)/$(CLIENT_BIN)-darwin-amd64 $(CLIENT_CMD)
	# macOS arm64
	GOOS=darwin GOARCH=arm64 $(GO_BUILD) -o $(BUILD_DIR)/$(CLIENT_BIN)-darwin-arm64 $(CLIENT_CMD)
	@echo "✓ Cross-compilation complete"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -cover ./...
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

# Install binaries to system
install: build
	@echo "Installing binaries to $(INSTALL_DIR)..."
	@sudo cp $(BUILD_DIR)/$(CLIENT_BIN) $(INSTALL_DIR)/
	@sudo cp $(BUILD_DIR)/$(SERVER_BIN) $(INSTALL_DIR)/
	@sudo chmod +x $(INSTALL_DIR)/$(CLIENT_BIN)
	@sudo chmod +x $(INSTALL_DIR)/$(SERVER_BIN)
	@echo "✓ Installed successfully"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "✓ Clean complete"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "✓ Dependencies updated"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✓ Code formatted"

# Lint code
lint:
	@echo "Linting code..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Generate documentation
docs:
	@echo "Generating documentation..."
	@if command -v godoc > /dev/null; then \
		echo "Starting godoc server at http://localhost:6060"; \
		godoc -http=:6060; \
	else \
		echo "godoc not installed. Install with:"; \
		echo "  go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

# Quick deploy example (customize for your needs)
deploy-example:
	@echo "Example deployment command:"
	@echo "./$(BUILD_DIR)/$(CLIENT_BIN) -mode deploy \\"
	@echo "  -remote-host 10.27.14.107 \\"
	@echo "  -remote-user centos \\"
	@echo "  -remote-password 'your-password' \\"
	@echo "  -vpn-server-ip 10.99.1.1 \\"
	@echo "  -vpn-network 10.99.1.0/24 \\"
	@echo "  -vpn-port 51821"

# Help target
help:
	@echo "VPN Connect - Build Targets"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all              Build client and server (default)"
	@echo "  build            Build client and server"
	@echo "  client           Build client binary only"
	@echo "  server           Build server binary only"
	@echo "  static           Build static binaries for deployment"
	@echo "  cross-compile    Build for multiple platforms"
	@echo "  test             Run tests"
	@echo "  test-coverage    Run tests with coverage report"
	@echo "  install          Install binaries to $(INSTALL_DIR)"
	@echo "  clean            Remove build artifacts"
	@echo "  deps             Download and update dependencies"
	@echo "  fmt              Format code"
	@echo "  lint             Lint code (requires golangci-lint)"
	@echo "  docs             Start godoc server"
	@echo "  deploy-example   Show example deployment command"
	@echo "  help             Show this help message"

