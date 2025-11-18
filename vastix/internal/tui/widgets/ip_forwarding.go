package widgets

import (
	"context"
	"fmt"
	"io"
	"net/netip"
	"os"
	"time"
	"vastix/internal/database"
	"vastix/internal/msg_types"
	"vastix/internal/tui/widgets/common"

	vpn_client "vastix/internal/vpn_connect/client"
	vpn_common "vastix/internal/vpn_connect/common"

	tea "github.com/charmbracelet/bubbletea"
)

// IpForwarding is an extra widget for establishing VPN connection to a single IP via SSH
type IpForwarding struct {
	*BaseWidget

	// VPN components
	deployer  *vpn_client.Deployer
	vpnClient *vpn_client.Client
	logWriter interface{ Write([]byte) (int, error) } // Writer for logs (working zone or details adapter)
	msgChan   chan tea.Msg                            // Channel to send messages to the app

	// State
	connected           bool
	deploying           bool
	needingSudoPassword bool   // True when waiting for sudo password input
	sudoPassword        string // Cached sudo password for this session
	serverKey           string
	clientID            int
	serverPort          uint16 // Track server port for cleanup
	ctx                 context.Context
	cancel              context.CancelFunc
	lastStatus          string
	lastError           error

	// Connection details
	targetIP       string // Single IP address to route through VPN
	remoteHost     string
	remoteUser     string
	remotePassword string
	remoteKeyPath  string
	privateIPs     []netip.Addr // Single IP as array for compatibility
}

// NewIpForwarding creates a new IP forwarding extra widget
func NewIpForwarding(db *database.Service, msgChan chan tea.Msg) *IpForwarding {
	ctx, cancel := context.WithCancel(context.Background())

	// Custom key restrictions: disable ctrl+e in details mode (can't "edit & resubmit" logs)
	keyRestrictions := &common.NavigatorKeyRestrictions{
		Main: common.NewDefaultKeyRestrictions(),
		Extra: common.KeyRestrictions{
			NotAllowedListKeys:    []string{},
			NotAllowedCreateKeys:  []string{},
			NotAllowedDeleteKeys:  []string{},
			NotAllowedDetailsKeys: []string{"ctrl+e"}, // Hide "edit & resubmit" for log view
		},
	}

	widget := &IpForwarding{
		BaseWidget: NewBaseWidget(db, nil, nil, "ip_forwarding", nil, keyRestrictions),
		clientID:   1, // Fixed client ID for now (can be made configurable)
		ctx:        ctx,
		cancel:     cancel,
		privateIPs: []netip.Addr{}, // Will contain single IP
		lastStatus: "Not connected",
		msgChan:    msgChan, // Store msgChan for health monitoring
	}

	// Set this widget as NOT resourceless - it requires SSH connections
	widget.SetResourceless(false)

	// Set custom title for the details view
	widget.DetailsAdapter.SetPredefinedTitle("Forwarding Logs")

	// Initialize logWriter with DetailsAdapter by default
	widget.logWriter = widget.DetailsAdapter

	return widget
}

// SetLogWriter sets the writer for VPN logs (working zone or details adapter)
func (w *IpForwarding) SetLogWriter(writer interface{ Write([]byte) (int, error) }) {
	w.logWriter = writer

	// Recreate deployer with new writer
	w.deployer = vpn_client.NewDeployer(writer)
}

// Init initializes the IP forwarding widget
func (w *IpForwarding) Init() tea.Msg {
	w.auxlog.Printf("IpForwarding.Init() called")

	if w.msgChan == nil {
		w.auxlog.Printf("WARNING: msgChan not set in IpForwarding.Init, health monitoring will be limited")
	}

	// Cancel old context if exists, create new one for fresh start
	if w.cancel != nil {
		w.cancel()
	}
	w.ctx, w.cancel = context.WithCancel(context.Background())

	// Reset state for fresh start - user can enter new IP each time
	w.targetIP = ""
	w.privateIPs = []netip.Addr{}
	w.needingSudoPassword = false
	w.connected = false
	w.deploying = false
	w.auxlog.Printf("IpForwarding state reset for new connection")

	return nil
}

// ShortCut returns the keyboard shortcut for this extra widget
func (*IpForwarding) ShortCut() *common.KeyBinding {
	return &common.KeyBinding{
		Key:           "<1>",
		Desc:          "ip forwarding",
		IsExtraAction: true,
	}
}

// GetSummary returns a short description of this extra widget for display in the extra actions list
func (*IpForwarding) GetSummary() string {
	return "Deploy VPN server via SSH and route single IP address through tunnel"
}

// GetAllowedExtraNavigatorModes restricts which modes are available for this widget
func (*IpForwarding) GetAllowedExtraNavigatorModes() []common.ExtraNavigatorMode {
	return []common.ExtraNavigatorMode{
		common.ExtraNavigatorModeList,    // Select SSH connection
		common.ExtraNavigatorModeCreate,  // Initial IP address input form
		common.ExtraNavigatorModeDetails, // View connection logs
		common.ExtraNavigatorModePrompt,  // Disconnect confirmation
		// ExtraNavigatorModeDelete is intentionally excluded (not applicable)
	}
}

// GetDetailsKeyBindings returns key bindings for details mode (viewing logs)
func (w *IpForwarding) GetDetailsKeyBindings() []common.KeyBinding {
	return []common.KeyBinding{
		{Key: "</>", Desc: "search", Generic: true},
		{Key: "<↑/↓>", Desc: "scroll"},
		{Key: "<pgup/pgdn>", Desc: "page"},
		{Key: "<ctrl+s>", Desc: "copy to clipboard"},
		// <ctrl+e> "edit & resubmit" intentionally excluded - doesn't make sense for logs
		{Key: "<esc>", Desc: "back"},
	}
}

// InitialExtraMode returns the initial mode for this extra widget
func (*IpForwarding) InitialExtraMode() common.ExtraNavigatorMode {
	// Start with Create mode to ask for IP address first
	return common.ExtraNavigatorModeCreate
}

// GetInputs returns the input fields for the create form
func (w *IpForwarding) GetInputs() (common.Inputs, error) {
	inputs := common.Inputs{}

	// IP Address input
	inputs.NewTextInput("ip_address", "Enter IP address (e.g., 192.168.1.100)", true, "")

	return inputs, nil
}

// ViewCreateForm displays the create form
func (w *IpForwarding) ViewCreateForm() string {
	return w.viewCreateForm()
}

// ViewPrompt displays the IP forwarding connection prompt
func (w *IpForwarding) ViewPrompt() string {
	selectedRowData := w.selectedRowData

	// Extract SSH connection details
	sshName := selectedRowData.GetStringMust("name")
	sshHost := selectedRowData.GetStringMust("ssh_host")
	sshUser := selectedRowData.GetStringMust("ssh_user_name")

	// Create the prompt message
	promptMsg := fmt.Sprintf(
		"Deploy and connect to IP '%s' via %s (%s@%s)?\n\n"+
			"This will:\n"+
			"  • Deploy VPN server via SSH\n"+
			"  • Route traffic to %s through VPN tunnel\n"+
			"  • Allow direct access to the IP address",
		w.targetIP, sshName, sshUser, sshHost, w.targetIP,
	)
	promptTitle := fmt.Sprintf("Connect to IP: %s", w.targetIP)

	// Use the prompt adapter to render the prompt
	width := w.GetWidth()
	height := w.GetHeight()

	return w.PromptAdapter.PromptDo(promptMsg, promptTitle, width, height)
}

// getSudoPassword gets or validates sudo password, returns error if not available/valid
func (w *IpForwarding) getSudoPassword() error {
	w.auxlog.Printf("DEBUG getSudoPassword: checking if password is needed...")

	// Check if wg-quick specifically needs a password
	if !vpn_client.CheckWgQuickNeedsPassword() {
		// wg-quick doesn't need password, use empty string
		w.sudoPassword = ""
		w.auxlog.Printf("wg-quick configured for passwordless sudo execution")
		return nil
	}

	// Check if we already have a password in memory (from popup submission)
	if w.sudoPassword != "" {
		w.auxlog.Printf("DEBUG getSudoPassword: using in-memory password")
		return nil
	}

	// wg-quick requires password - try to get it from database
	w.auxlog.Printf("DEBUG getSudoPassword: trying to get password from database...")
	sudoPwd, err := w.db.GetSudoPassword()
	if err == nil && sudoPwd != nil {
		// Validate the stored password
		if err := vpn_client.ValidateSudoPassword(sudoPwd.Password); err == nil {
			w.sudoPassword = sudoPwd.Password
			w.auxlog.Printf("Using stored sudo password from database")
			return nil
		}
		// Invalid password, delete it
		w.auxlog.Printf("Stored sudo password invalid, removing from database")
		w.db.DeleteSudoPassword()
	}

	// No valid password available
	w.auxlog.Printf("DEBUG getSudoPassword: no password available, need to prompt user")
	return fmt.Errorf("sudo password required but not available")
}

// CreateFromInputs initiates the IP forwarding connection
func (w *IpForwarding) CreateFromInputs(inputs common.Inputs) (tea.Cmd, error) {
	w.auxlog.Printf("DEBUG CreateFromInputs: called, needingSudoPassword=%v", w.needingSudoPassword)

	// Step 1: Handle IP address input (first time entering Create mode)
	if w.targetIP == "" && !w.needingSudoPassword {
		w.auxlog.Printf("DEBUG CreateFromInputs: extracting IP address from inputs")

		// Find the ip_address input
		var ipAddress string
		for _, input := range inputs {
			if input.GetLabel() == "ip_address" {
				ipAddress = input.Value()
				break
			}
		}

		if ipAddress == "" {
			return nil, fmt.Errorf("IP address cannot be empty")
		}

		// Parse and validate IP address
		ip, err := netip.ParseAddr(ipAddress)
		if err != nil {
			return nil, fmt.Errorf("invalid IP address: %w", err)
		}

		// Store the IP address
		w.targetIP = ipAddress
		w.privateIPs = []netip.Addr{ip}
		w.auxlog.Printf("IP address set to: %s", w.targetIP)

		// Switch to Prompt mode for confirmation
		w.SetExtraMode(common.ExtraNavigatorModePrompt)
		return nil, nil
	}

	// Step 2: Check if we're waiting for sudo password from popup
	if w.needingSudoPassword {
		w.auxlog.Printf("DEBUG CreateFromInputs: checking popup state...")
		if !w.DetailsAdapter.IsPopupHidden() {
			w.auxlog.Printf("DEBUG CreateFromInputs: popup still visible, waiting for user input")
			return nil, nil // Popup is still visible, wait for user to submit
		}

		popupContent := w.DetailsAdapter.GetPopupContent()
		w.auxlog.Printf("DEBUG CreateFromInputs: popup hidden, content length=%d", len(popupContent))

		if popupContent == "" {
			w.auxlog.Printf("DEBUG CreateFromInputs: no content submitted yet")
			return nil, nil // User hasn't entered anything yet
		}

		w.auxlog.Printf("DEBUG CreateFromInputs: processing popup submission")
		// Validate the password
		if err := vpn_client.ValidateSudoPassword(popupContent); err != nil {
			w.auxlog.Printf("Invalid sudo password: %v", err)
			w.DetailsAdapter.ClearPopupContent()

			// Show the popup again with error message
			w.DetailsAdapter.ShowPopup(
				"Invalid Sudo Password - Try Again",
				"Enter your local system password",
				true, // isSecret
			)
			return nil, nil
		}

		// Save the password to database
		db := database.New()
		if err := db.SaveSudoPassword(popupContent); err != nil {
			w.auxlog.Printf("Warning: Failed to save sudo password: %v", err)
			// Continue anyway with the in-memory password
		} else {
			w.auxlog.Printf("Sudo password validated and saved to database")
		}

		// Store password and clear the flag
		w.sudoPassword = popupContent
		w.needingSudoPassword = false
		w.DetailsAdapter.ClearPopupContent()

		w.auxlog.Printf("Sudo password accepted, continuing with deployment...")
		// Fall through to continue with deployment
	}

	selectedRowData := w.selectedRowData

	// Extract SSH connection details from selected row
	sshHost := selectedRowData.GetStringMust("ssh_host")
	sshUser := selectedRowData.GetStringMust("ssh_user_name")
	sshPort := int(selectedRowData.GetInt64Must("ssh_port"))
	authMethod := selectedRowData.GetStringMust("auth_method")

	// Get the SSH connection ID to fetch full details
	sshConnID, err := selectedRowData.GetIntID()
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH connection ID: %w", err)
	}

	// Fetch full SSH connection details from database
	sshConn, err := w.db.GetSshConnection(uint(sshConnID))
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH connection details: %w", err)
	}

	// Store connection details
	w.remoteHost = sshHost
	w.remoteUser = sshUser
	w.remotePassword = sshConn.SshPassword
	w.remoteKeyPath = sshConn.SshKey

	// Validate SSH connection details
	if authMethod == "key" && w.remoteKeyPath == "" {
		return nil, fmt.Errorf("SSH key path is required for key-based authentication")
	}
	if authMethod == "password" && w.remotePassword == "" {
		return nil, fmt.Errorf("SSH password is required for password-based authentication")
	}

	// IP should already be set
	if len(w.privateIPs) == 0 {
		return nil, fmt.Errorf("no IP address available - this should not happen")
	}
	w.auxlog.Printf("Using IP address %s for VPN routing", w.targetIP)

	// Check/get sudo password before proceeding
	if err := w.getSudoPassword(); err != nil {
		// Need to ask for sudo password via popup
		w.needingSudoPassword = true
		w.DetailsAdapter.ShowPopup(
			"Sudo Password Required",
			"Enter your local system password",
			true, // isSecret
		)
		// Stay in prompt mode and wait for popup submission
		return nil, nil
	}

	// Return async command to deploy and connect
	return func() tea.Msg {
		w.deploying = true
		w.lastError = nil

		w.DetailsAdapter.ClearContent()

		// Create writer that writes to DetailsAdapter AND auxlog
		var writer io.Writer
		auxLogWriter := w.auxlog.Writer()

		if w.logWriter == w.DetailsAdapter {
			// logWriter is just the DetailsAdapter, write to both DetailsAdapter and auxlog
			writer = io.MultiWriter(w.DetailsAdapter, auxLogWriter)
		} else {
			// logWriter is the working zone writer, write to all three destinations
			writer = io.MultiWriter(w.logWriter, w.DetailsAdapter, auxLogWriter)
		}

		w.deployer = vpn_client.NewDeployer(writer)

		// Set IP for health monitoring
		w.deployer.SetVipPoolIPs(w.privateIPs)

		// Step 0: Check if WireGuard is installed
		w.lastStatus = "Checking WireGuard installation..."
		if err := vpn_client.CheckWireGuardInstalled(); err != nil {
			w.lastError = err
			w.lastStatus = "WireGuard not installed"
			w.deploying = false
			w.auxlog.Printf("Error: WireGuard check failed: %v", err)
			return msg_types.ErrorMsg{Err: fmt.Errorf("WireGuard check failed:\n%w", err)}
		}
		w.auxlog.Printf("WireGuard is installed")

		w.lastStatus = "Deploying VPN server..."

		// Get local hostname for remote directory structure
		hostname, err := os.Hostname()
		if err != nil {
			w.lastError = err
			w.lastStatus = "Failed to get local hostname"
			w.deploying = false
			w.auxlog.Printf("Error: Failed to get local hostname: %v", err)
			return msg_types.ErrorMsg{Err: fmt.Errorf("failed to get local hostname: %w", err)}
		}

		// Step 1: Connect to remote host via SSH
		w.lastStatus = "Connecting to remote host..."
		deployConfig := &vpn_client.DeploymentConfig{
			Host:           w.remoteHost,
			Port:           sshPort,
			Username:       w.remoteUser,
			Password:       w.remotePassword,
			PrivateKeyPath: w.remoteKeyPath,
			RemoteWorkDir:  "/tmp", // Temporary, will update after port allocation
		}

		if err := w.deployer.Connect(w.ctx, deployConfig); err != nil {
			w.lastError = err
			w.lastStatus = "Failed to connect to remote host"
			w.deploying = false
			w.auxlog.Printf("Error: Failed to connect to remote host: %v", err)
			return msg_types.ErrorMsg{Err: fmt.Errorf("failed to connect to remote host: %w", err)}
		}

		// Immediately verify SSH connection is working before proceeding with deployment
		w.lastStatus = "Verifying SSH connection..."
		w.auxlog.Printf("Verifying SSH connection health before deployment...")
		if err := w.deployer.CheckSSHHealth(); err != nil {
			w.lastError = err
			w.lastStatus = "SSH connection verification failed"
			w.deploying = false
			w.auxlog.Printf("Error: SSH connection verification failed: %v", err)
			return msg_types.ErrorMsg{Err: fmt.Errorf("SSH connection verification failed:\n%w\n\nThe remote host may be unreachable or the target IP is not accessible from this host", err)}
		}
		w.auxlog.Printf("SSH connection verified successfully")

		// Step 2: Allocate an available port on the remote host
		w.lastStatus = "Allocating VPN port..."
		const portRangeStart = 51821
		const portRangeEnd = 51920 // Support up to 100 concurrent VPN connections

		port, err := w.deployer.AllocatePort(portRangeStart, portRangeEnd)
		if err != nil {
			w.lastError = err
			w.lastStatus = "Failed to allocate port"
			w.deploying = false
			w.auxlog.Printf("Error: Failed to allocate port: %v", err)
			return msg_types.ErrorMsg{Err: fmt.Errorf("failed to allocate port: %w", err)}
		}

		w.serverPort = port // Store port for cleanup
		w.auxlog.Printf("Allocated port: %d", port)

		// Calculate clientID from port for network generation
		clientID := int(port - 51820)

		// Step 3: Generate network configuration
		vpnNetwork, serverIP, clientIP, err := vpn_common.GenerateVPNNetwork(clientID)
		if err != nil {
			w.lastError = err
			w.lastStatus = "Failed to generate network"
			w.deploying = false
			w.auxlog.Printf("Error: Failed to generate VPN network: %v", err)
			return msg_types.ErrorMsg{Err: fmt.Errorf("failed to generate VPN network: %w", err)}
		}

		// Remote directory: /tmp/vastix_vpn/<local_hostname>-<port>
		remoteWorkDir := fmt.Sprintf("/tmp/vastix_vpn/%s-port%d", hostname, port)
		deployConfig.RemoteWorkDir = remoteWorkDir // Update the work directory

		w.auxlog.Printf("VPN Network: %s, Server IP: %s, Client IP: %s, Port: %d",
			vpnNetwork, serverIP, clientIP, port)
		w.auxlog.Printf("Remote work directory: %s", remoteWorkDir)

		// Step 3: Generate server keys
		w.lastStatus = "Generating encryption keys..."
		serverPrivKey, serverPubKey, err := vpn_common.GenerateKeyPair()
		if err != nil {
			w.lastError = err
			w.lastStatus = "Failed to generate keys"
			w.deploying = false
			w.auxlog.Printf("Error: Failed to generate server keys: %v", err)
			return msg_types.ErrorMsg{Err: fmt.Errorf("failed to generate server keys: %w", err)}
		}

		w.serverKey = serverPubKey
		w.auxlog.Printf("Server Public Key: %s", serverPubKey)

		// Step 4: Deploy server binary
		w.lastStatus = "Uploading server binary..."
		w.auxlog.Printf("Preparing server configuration...")
		serverConfig := &vpn_common.ServerConfig{
			PrivateKey: serverPrivKey,
			PublicKey:  serverPubKey,
			ListenPort: port,
			ServerIP:   serverIP,
			VPNNetwork: vpnNetwork,
			PrivateIPs: w.privateIPs,
		}
		w.auxlog.Printf("Initiating deployment to remote host...")

		if err := w.deployer.Deploy(w.ctx, deployConfig, serverConfig); err != nil {
			w.lastError = err
			w.lastStatus = "Failed to deploy server"
			w.deploying = false
			w.auxlog.Printf("Error: Failed to deploy server: %v", err)
			return msg_types.ErrorMsg{Err: fmt.Errorf("failed to deploy server: %w", err)}
		}

		// Step 5: Start server in background
		w.lastStatus = "Starting VPN server..."

		// Start server in goroutine
		go func() {
			if err := w.deployer.StartServer(w.ctx, deployConfig.RemoteWorkDir, serverConfig); err != nil {
				w.auxlog.Printf("Server stopped: %v", err)
				// Send error message to main app if server exits unexpectedly
				if w.msgChan != nil && w.ctx.Err() == nil {
					w.msgChan <- msg_types.ErrorMsg{Err: fmt.Errorf("VPN server stopped unexpectedly: %w", err)}
				}
			}
		}()

		// Give server a moment to initialize
		w.auxlog.Printf("Waiting for VPN server to initialize...")
		time.Sleep(3 * time.Second)

		// Start heartbeat monitoring
		w.auxlog.Printf("Starting heartbeat to remote server...")
		if err := w.deployer.StartHeartbeat(deployConfig.RemoteWorkDir); err != nil {
			w.lastError = err
			w.lastStatus = "Failed to start heartbeat"
			w.deploying = false
			w.auxlog.Printf("Error: Failed to start heartbeat: %v", err)
			return msg_types.ErrorMsg{Err: fmt.Errorf("failed to start heartbeat: %w", err)}
		}
		time.Sleep(3 * time.Second)
		w.auxlog.Printf("VPN server should be running now")

		// Step 6: Generate client keys
		w.lastStatus = "Generating client keys..."
		clientPrivKey, clientPubKey, err := vpn_common.GenerateKeyPair()
		if err != nil {
			w.lastError = err
			w.lastStatus = "Failed to generate client keys"
			w.deploying = false
			w.auxlog.Printf("Error: Failed to generate client keys: %v", err)
			return msg_types.ErrorMsg{Err: fmt.Errorf("failed to generate client keys: %w", err)}
		}

		w.auxlog.Printf("Client Public Key: %s", clientPubKey)

		// Step 6.5: Register client peer on server
		w.lastStatus = "Registering client on server..."
		if err := w.deployer.RegisterPeer(clientPubKey, clientIP.String(), port); err != nil {
			w.lastError = err
			w.lastStatus = "Failed to register client"
			w.deploying = false
			w.auxlog.Printf("Error: Failed to register client peer: %v", err)
			return msg_types.ErrorMsg{Err: fmt.Errorf("failed to register client peer: %w", err)}
		}

		w.auxlog.Printf("Client peer registered on server")

		// Step 7: Create client configuration
		w.lastStatus = "Preparing VPN client..."

		clientConfig := &vpn_common.ClientConfig{
			PrivateKey:      clientPrivKey,
			PublicKey:       clientPubKey,
			ServerPublicKey: serverPubKey,
			ServerEndpoint:  fmt.Sprintf("%s:%d", w.remoteHost, port),
			ClientIP:        clientIP,
			ServerIP:        serverIP,
			PrivateIPs:      w.privateIPs, // Single IP
		}

		// Step 8: Create VPN client
		w.lastStatus = "Preparing VPN connection..."
		w.auxlog.Printf("Initiating VPN connection with output display...")
		w.deploying = false

		w.vpnClient, err = vpn_client.NewClient(clientConfig, writer)
		if err != nil {
			w.lastError = err
			w.lastStatus = "Failed to create VPN client"
			w.auxlog.Printf("Error: Failed to create VPN client: %v", err)
			return msg_types.ErrorMsg{Err: fmt.Errorf("failed to create VPN client:\n%w", err)}
		}

		// Switch to details mode if using only DetailsAdapter
		if w.logWriter == w.DetailsAdapter {
			w.SetExtraMode(common.ExtraNavigatorModeDetails)
		}

		// Step 9: Connect VPN
		if err := w.vpnClient.Connect(w.sudoPassword); err != nil {
			w.lastError = err
			w.lastStatus = "Failed to connect"
			w.auxlog.Printf("Error: VPN connection failed: %v", err)
			return msg_types.ErrorMsg{Err: fmt.Errorf("VPN connection failed:\n%w", err)}
		}

		// Step 10: Connection successful!
		w.connected = true
		w.lastStatus = "Connected successfully"
		w.lastError = nil

		// Start health monitoring
		w.StartHealthMonitoring(w.ctx)

		return nil
	}, nil
}

// UpdateViewPort overrides the base UpdateViewPort to handle popup submissions
func (w *IpForwarding) UpdateViewPort(msg tea.Msg) tea.Cmd {
	w.auxlog.Printf("DEBUG UpdateViewPort: called, needingSudoPassword=%v", w.needingSudoPassword)

	// First, let the DetailsAdapter handle the message (including popup input)
	cmd := w.DetailsAdapter.UpdateViewPort(msg)

	// Check if we're waiting for sudo password and popup was just submitted
	if w.needingSudoPassword && w.DetailsAdapter.IsPopupHidden() {
		popupContent := w.DetailsAdapter.GetPopupContent()
		w.auxlog.Printf("DEBUG UpdateViewPort: popup hidden, content length=%d", len(popupContent))

		if popupContent != "" {
			w.auxlog.Printf("DEBUG UpdateViewPort: popup has content, re-calling CreateFromInputs")
			// Call CreateFromInputs to process the password
			createCmd, err := w.CreateFromInputs(common.Inputs{})
			if err != nil {
				w.auxlog.Printf("Error in CreateFromInputs after popup: %v", err)
				return tea.Batch(cmd, func() tea.Msg {
					return msg_types.ErrorMsg{Err: err}
				})
			}
			// Wrap with ProcessWithSpinner to trigger spinner display
			return tea.Batch(cmd, msg_types.ProcessWithSpinner(createCmd))
		}
	}

	return cmd
}

// Update handles messages for the IP forwarding widget
func (w *IpForwarding) Update(msg tea.Msg) tea.Cmd {
	// Debug: Log when Update is called while needing sudo password
	if w.needingSudoPassword {
		isHidden := w.DetailsAdapter.IsPopupHidden()
		popupContent := w.DetailsAdapter.GetPopupContent()
		w.auxlog.Printf("DEBUG Update: needingSudoPassword=true, popupHidden=%v, popupContent=%q",
			isHidden, popupContent)
	}

	// Check if popup was just submitted (hidden with content)
	if w.needingSudoPassword && w.DetailsAdapter.IsPopupHidden() {
		popupContent := w.DetailsAdapter.GetPopupContent()
		w.auxlog.Printf("DEBUG: Popup hidden with content, processing password")
		if popupContent != "" {
			// Clear popup content
			w.DetailsAdapter.ClearPopupContent()

			// Validate the password
			if err := vpn_client.ValidateSudoPassword(popupContent); err != nil {
				// Invalid password, show popup again with error
				w.auxlog.Printf("Invalid sudo password: %v", err)
				w.DetailsAdapter.ShowPopup(
					"Invalid Sudo Password",
					"Try again (password was incorrect)",
					true,
				)
				return nil
			}

			// Save password to database
			if err := w.db.SaveSudoPassword(popupContent); err != nil {
				w.auxlog.Printf("Warning: Failed to save sudo password: %v", err)
			}

			// Store password and clear flag
			w.sudoPassword = popupContent
			w.needingSudoPassword = false

			// Clear details content and prepare for logs
			w.DetailsAdapter.ClearContent()
			w.DetailsAdapter.AppendContent("Starting VPN deployment...\n\n")

			// Switch to Details mode to show deployment logs
			w.SetExtraMode(common.ExtraNavigatorModeDetails)

			// Now proceed with deployment
			cmd, err := w.CreateFromInputs(common.Inputs{})
			if err != nil {
				w.auxlog.Printf("Error calling CreateFromInputs: %v", err)
				return func() tea.Msg { return msg_types.ErrorMsg{Err: err} }
			}

			w.auxlog.Printf("Deployment command created successfully")
			return cmd
		}
	}

	// Handle keyboard input for prompt navigation
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		extraMode := w.GetExtraMode()
		if extraMode == common.ExtraNavigatorModePrompt {
			switch keyMsg.String() {
			case "left", "right", "tab":
				// Toggle between Yes/No buttons
				w.PromptAdapter.ToggleSelection()
				return nil
			case "y":
				// Select Yes directly
				w.PromptAdapter.SetSelection(false)
				return nil
			case "n":
				// Select No directly
				w.PromptAdapter.SetSelection(true)
				return nil
			}
		}
	}

	// Let base widget handle other messages
	return nil
}

// ViewDetails displays the IP forwarding connection details
func (w *IpForwarding) ViewDetails() string {
	// Just show the base details view (logs)
	return w.viewDetails()
}

// Disconnect closes the IP forwarding connection and stops the remote server
func (w *IpForwarding) Disconnect() error {
	w.auxlog.Printf("Disconnecting VPN...")

	// Step 1: Disconnect local VPN client
	if w.vpnClient != nil {
		w.auxlog.Printf("Bringing down local WireGuard interface...")
		if err := w.vpnClient.Disconnect(w.sudoPassword); err != nil {
			w.auxlog.Printf("Warning: Failed to disconnect VPN client: %v", err)
		} else {
			w.auxlog.Printf("Local WireGuard interface cleaned up successfully")
		}
		w.connected = false
		w.lastStatus = "Disconnected"
	} else {
		w.auxlog.Printf("No VPN client to disconnect")
	}

	// Step 2: Close SSH connection (this will automatically stop the remote server)
	if w.deployer != nil {
		w.auxlog.Printf("Closing SSH connection to remote server...")
		if err := w.deployer.Disconnect(); err != nil {
			w.auxlog.Printf("Warning: Failed to close SSH connection: %v", err)
		} else {
			w.auxlog.Printf("SSH connection closed successfully")
		}
	} else {
		w.auxlog.Printf("No SSH deployer to disconnect")
	}

	w.auxlog.Printf("VPN disconnected and cleaned up (local + remote)")
	return nil
}

// LeaveWidget is called when user leaves the widget
func (w *IpForwarding) LeaveWidget() error {
	w.auxlog.Printf("Leaving IP Forwarding Widget")

	// Cancel context to stop the server goroutine
	if w.cancel != nil {
		w.auxlog.Printf("Cancelling VPN context - this will stop remote server")
		w.cancel()
	}

	// Disconnect local VPN client
	w.auxlog.Printf("Cleaning up local VPN connection")
	err := w.Disconnect()
	time.Sleep(1 * time.Second)
	return err
}

// IsConnected returns whether VPN is connected
func (w *IpForwarding) IsConnected() bool {
	return w.connected && w.vpnClient != nil && w.vpnClient.IsConnected()
}

// StartHealthMonitoring starts monitoring VPN connection health
func (w *IpForwarding) StartHealthMonitoring(ctx context.Context) {
	if w.msgChan == nil {
		w.auxlog.Printf("WARNING: msgChan is nil, health monitoring disabled")
		return
	}

	go func() {
		ticker := time.NewTicker(12 * time.Second)
		defer ticker.Stop()

		w.auxlog.Printf("VPN health monitoring started (interval: 12s)")

		for {
			select {
			case <-ctx.Done():
				w.auxlog.Printf("VPN health monitoring stopped")
				return
			case <-ticker.C:
				w.checkHealth()
			}
		}
	}()
}

// checkHealth performs health checks on the VPN connection
func (w *IpForwarding) checkHealth() {
	// Check if we're supposed to be connected
	if !w.connected && !w.deploying {
		return // Not connected, no need to check
	}

	w.auxlog.Printf("🔍 Running VPN health check...")

	// Check 1: SSH connection health
	if w.deployer != nil {
		w.auxlog.Printf("   → Checking SSH connection to VPN server...")
		if err := w.deployer.CheckSSHHealth(); err != nil {
			w.auxlog.Printf("ERROR: SSH connection lost: %v", err)
			w.sendError(fmt.Errorf("SSH connection to VPN server lost: %w", err))

			// Automatically clean up local resources when connection is lost
			w.auxlog.Printf("Automatically cleaning up local VPN interface due to connection loss...")
			w.Disconnect()

			return
		}
		w.auxlog.Printf("   ✓ SSH connection OK")
	}

	// Check 2: VPN tunnel health
	if w.vpnClient != nil && w.connected {
		w.auxlog.Printf("   → Checking VPN tunnel (ping gateway)...")
		if err := w.vpnClient.CheckTunnelHealth(); err != nil {
			w.auxlog.Printf("ERROR: VPN tunnel unhealthy: %v", err)
			w.sendError(fmt.Errorf("VPN tunnel connection lost: %w", err))

			// Automatically clean up local resources when tunnel is lost
			w.auxlog.Printf("Automatically cleaning up local VPN interface due to tunnel failure...")
			w.Disconnect()

			return
		}
		w.auxlog.Printf("   ✓ VPN tunnel OK")
	}

	w.auxlog.Printf("Health check passed")
}

// sendError sends an error message to the app's message channel
func (w *IpForwarding) sendError(err error) {
	if w.msgChan == nil {
		w.auxlog.Printf("WARNING: Cannot send error, msgChan is nil: %v", err)
		return
	}

	// Send error message to the app
	select {
	case w.msgChan <- msg_types.ErrorMsg{Err: err}:
		w.auxlog.Printf("Error message sent to app: %v", err)
	default:
		w.auxlog.Printf("WARNING: msgChan full, could not send error: %v", err)
	}
}
