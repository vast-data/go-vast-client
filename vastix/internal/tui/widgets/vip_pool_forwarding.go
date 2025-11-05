package widgets

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/netip"
	"os"
	"time"
	"vastix/internal/database"
	"vastix/internal/msg_types"
	"vastix/internal/tui/widgets/common"

	vpn_client "vastix/internal/vpn_connect/client"
	vpn_common "vastix/internal/vpn_connect/common"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/crypto/ssh"
)

// VipPoolForwarding is an extra widget for establishing VPN connection to VIP pool via SSH
type VipPoolForwarding struct {
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
	vipPoolName    string // Name of the VIP pool to connect to
	remoteHost     string
	remoteUser     string
	remotePassword string
	remoteKeyPath  string
	privateIPs     []netip.Addr // List of VIP pool IPs to route through VPN
}

// NewVipPoolForwarding creates a new VIP pool forwarding extra widget
func NewVipPoolForwarding(db *database.Service, msgChan chan tea.Msg) *VipPoolForwarding {
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

	widget := &VipPoolForwarding{
		BaseWidget: NewBaseWidget(db, nil, nil, "vip_pool_forwarding", nil, keyRestrictions),
		clientID:   1, // Fixed client ID for now (can be made configurable)
		ctx:        ctx,
		cancel:     cancel,
		privateIPs: []netip.Addr{}, // Will be populated from VIP pool data
		lastStatus: "Not connected",
		msgChan:    msgChan, // Store msgChan for health monitoring
	}

	// Set this widget as NOT resourceless - it requires SSH connections
	widget.SetResourceless(false)

	// Set custom title for the details view
	widget.DetailsAdapter.SetPredefinedTitle("Forwarding Logs")

	// Initialize logWriter with DetailsAdapter by default
	// This ensures we always have a valid writer even before SetLogWriter is called
	widget.logWriter = widget.DetailsAdapter

	return widget
}

// SetLogWriter sets the writer for VPN logs (working zone or details adapter)
func (w *VipPoolForwarding) SetLogWriter(writer interface{ Write([]byte) (int, error) }) {
	w.logWriter = writer

	// Recreate deployer with new writer
	w.deployer = vpn_client.NewDeployer(writer)
}

// Init initializes the VIP pool forwarding widget
// Called when the extra widget is activated
// msgChan is already set from constructor via NewVipPoolForwarding
func (w *VipPoolForwarding) Init() tea.Msg {
	w.auxlog.Printf("VipPoolForwarding.Init() called")

	if w.msgChan == nil {
		w.auxlog.Printf("WARNING: msgChan not set in VipPoolForwarding.Init, health monitoring will be limited")
	}

	// Cancel old context if exists, create new one for fresh start
	if w.cancel != nil {
		w.cancel()
	}
	w.ctx, w.cancel = context.WithCancel(context.Background())

	// Reset state for fresh start - user can enter new VIP pool each time
	w.vipPoolName = ""
	w.privateIPs = []netip.Addr{}
	w.needingSudoPassword = false
	w.connected = false
	w.deploying = false
	w.auxlog.Printf("VipPoolForwarding state reset for new connection")

	return nil
}

// ShortCut returns the keyboard shortcut for this extra widget
func (*VipPoolForwarding) ShortCut() *common.KeyBinding {
	return &common.KeyBinding{
		Key:           "<1>",
		Desc:          "vip pool forwarding",
		IsExtraAction: true,
	}
}

// GetSummary returns a short description of this extra widget for display in the extra actions list
func (*VipPoolForwarding) GetSummary() string {
	return "Deploy VPN server via SSH and route VIP pool traffic through tunnel"
}

// GetAllowedExtraNavigatorModes restricts which modes are available for this widget
// VIP Pool Forwarding uses all modes except Delete (not applicable for this action-based widget)
func (*VipPoolForwarding) GetAllowedExtraNavigatorModes() []common.ExtraNavigatorMode {
	return []common.ExtraNavigatorMode{
		common.ExtraNavigatorModeList,    // Select SSH connection / VIP pool
		common.ExtraNavigatorModeCreate,  // Initial VIP pool selection form
		common.ExtraNavigatorModeDetails, // View connection logs
		common.ExtraNavigatorModePrompt,  // Disconnect confirmation
		// ExtraNavigatorModeDelete is intentionally excluded (not applicable)
	}
}

// GetDetailsKeyBindings returns key bindings for details mode (viewing logs)
// Overrides base implementation to exclude <ctrl+e> (can't "edit & resubmit" logs)
func (w *VipPoolForwarding) GetDetailsKeyBindings() []common.KeyBinding {
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
func (*VipPoolForwarding) InitialExtraMode() common.ExtraNavigatorMode {
	// Start with Create mode to ask for VIP pool name first
	return common.ExtraNavigatorModeCreate
}

// GetInputs returns the input fields for the create form
func (w *VipPoolForwarding) GetInputs() (common.Inputs, error) {
	inputs := common.Inputs{}

	// VIP Pool Name input
	inputs.NewTextInput("vip_pool_name", "Enter VIP pool name (e.g., protocols-pool)", true, "")

	return inputs, nil
}

// ViewCreateForm displays the create form
func (w *VipPoolForwarding) ViewCreateForm() string {
	return w.viewCreateForm()
}

// ViewPrompt displays the VIP pool forwarding connection prompt
func (w *VipPoolForwarding) ViewPrompt() string {
	selectedRowData := w.selectedRowData

	// Extract SSH connection details
	sshName := selectedRowData.GetStringMust("name")
	sshHost := selectedRowData.GetStringMust("ssh_host")
	sshUser := selectedRowData.GetStringMust("ssh_user_name")

	// Create the prompt message
	promptMsg := fmt.Sprintf(
		"Deploy and connect to VIP Pool '%s' via %s (%s@%s)?\n\n"+
			"This will:\n"+
			"  • Deploy VPN server via SSH\n"+
			"  • Fetch IPs from VIP pool '%s'\n"+
			"  • Route VIP pool IPs through VPN tunnel\n"+
			"  • Allow access to VIP pool resources",
		w.vipPoolName, sshName, sshUser, sshHost, w.vipPoolName,
	)
	promptTitle := fmt.Sprintf("Connect to VIP Pool: %s", w.vipPoolName)

	// Use the prompt adapter to render the prompt
	width := w.GetWidth()
	height := w.GetHeight()

	return w.PromptAdapter.PromptDo(promptMsg, promptTitle, width, height)
}

// fetchVipPoolIPs fetches all VIP pool IPs from the VAST REST API
// This is a potentially long-running operation and should be called in a background goroutine
func (w *VipPoolForwarding) fetchVipPoolIPs(_ *database.SshConnection) error {
	w.auxlog.Printf("Fetching VIP pool IPs for pool '%s'", w.vipPoolName)

	// Get active profile from database
	activeProfile, err := w.db.GetActiveProfile()
	if err != nil {
		return fmt.Errorf("failed to get active profile: %w", err)
	}
	if activeProfile == nil {
		return fmt.Errorf("no active profile found")
	}

	// Create REST client from profile
	rest, err := activeProfile.RestClientFromProfile()
	if err != nil {
		return fmt.Errorf("failed to create REST client: %w", err)
	}

	w.auxlog.Printf("Fetching IPs from VIP pool '%s'", w.vipPoolName)

	// Fetch all IPs from the VIP pool
	ipStrings, err := rest.VipPools.IpRangeFor(w.vipPoolName)
	if err != nil {
		return fmt.Errorf("failed to fetch IPs from VIP pool '%s': %w", w.vipPoolName, err)
	}

	if len(ipStrings) == 0 {
		return fmt.Errorf("VIP pool '%s' has no IPs", w.vipPoolName)
	}

	w.auxlog.Printf("Fetched %d IPs from VIP pool", len(ipStrings))

	// Parse IP strings to netip.Addr
	w.privateIPs = make([]netip.Addr, 0, len(ipStrings))
	for _, ipStr := range ipStrings {
		ip, err := netip.ParseAddr(ipStr)
		if err != nil {
			w.auxlog.Printf("Warning: Failed to parse IP %s: %v", ipStr, err)
			continue
		}
		w.privateIPs = append(w.privateIPs, ip)
	}

	if len(w.privateIPs) == 0 {
		return fmt.Errorf("failed to parse any valid IPs from VIP pool '%s'", w.vipPoolName)
	}

	w.auxlog.Printf("Successfully parsed %d IPs from VIP pool '%s'", len(w.privateIPs), w.vipPoolName)
	return nil
}

// verifyIPReachability verifies that at least one random IP from the VIP pool is reachable via SSH
// This is called before establishing VPN to ensure IPs are actually accessible
func (w *VipPoolForwarding) verifyIPReachability(sshConn *database.SshConnection) error {
	if len(w.privateIPs) == 0 {
		return fmt.Errorf("no IPs to verify")
	}

	// Pick a random IP to test
	randomIP := w.privateIPs[rand.Intn(len(w.privateIPs))]
	w.auxlog.Printf("Testing reachability of IP: %s via SSH ping", randomIP)

	// Build SSH config
	var authMethods []ssh.AuthMethod

	// Add password authentication if provided
	if sshConn.SshPassword != "" {
		authMethods = append(authMethods, ssh.Password(sshConn.SshPassword))
	}

	// Add public key authentication if provided
	if sshConn.SshKey != "" {
		key, err := os.ReadFile(sshConn.SshKey)
		if err != nil {
			return fmt.Errorf("failed to read SSH private key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return fmt.Errorf("failed to parse SSH private key: %w", err)
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	sshConfig := &ssh.ClientConfig{
		User:            sshConn.SshUserName,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	// Connect to SSH
	addr := fmt.Sprintf("%s:%d", sshConn.SshHost, sshConn.SshPort)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect via SSH: %w", err)
	}
	defer client.Close()

	// Run ping command (2 pings, 2 sec timeout per ping = 10 sec total)
	pingCmd := fmt.Sprintf("ping -c 2 -W 2 %s", randomIP)
	w.auxlog.Printf("Running: %s", pingCmd)

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Capture output to auxlog for first ping
	session.Stdout = w.auxlog.Writer()
	session.Stderr = w.auxlog.Writer()

	if err := session.Run(pingCmd); err != nil {
		return fmt.Errorf("ping to %s failed: %w", randomIP, err)
	}

	w.auxlog.Printf("Successfully verified IP %s is reachable", randomIP)
	return nil
}

// getSudoPassword gets or validates sudo password, returns error if not available/valid
func (w *VipPoolForwarding) getSudoPassword() error {
	w.auxlog.Printf("DEBUG getSudoPassword: checking if password is needed...")

	// Check if wg-quick specifically needs a password
	// This is more accurate than checking generic sudo, as wg-quick might be
	// configured in sudoers for passwordless execution
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

// CreateFromInputs initiates the VIP pool forwarding connection
func (w *VipPoolForwarding) CreateFromInputs(inputs common.Inputs) (tea.Cmd, error) {
	w.auxlog.Printf("DEBUG CreateFromInputs: called, needingSudoPassword=%v", w.needingSudoPassword)

	// Step 1: Handle VIP pool name input (every time user enters Create mode)
	// User can specify different VIP pool each time they use this widget
	// Check if we need to fetch IPs (either no name set yet, or no IPs fetched yet)
	if (w.vipPoolName == "" || len(w.privateIPs) == 0) && !w.needingSudoPassword {
		w.auxlog.Printf("DEBUG CreateFromInputs: extracting VIP pool name from inputs")

		// Find the vip_pool_name input
		var vipPoolName string
		for _, input := range inputs {
			if input.GetLabel() == "vip_pool_name" {
				vipPoolName = input.Value()
				break
			}
		}

		if vipPoolName == "" {
			return nil, fmt.Errorf("VIP pool name cannot be empty")
		}

		// Store the name temporarily for the fetch
		tempVipPoolName := vipPoolName
		w.auxlog.Printf("Attempting to fetch IPs for VIP pool: %s", tempVipPoolName)

		// Get SSH connection for IP reachability check
		selectedRowData := w.selectedRowData
		sshConnID, err := selectedRowData.GetIntID()
		if err != nil {
			return nil, fmt.Errorf("failed to get SSH connection ID: %w", err)
		}

		// Fetch full SSH connection details from database
		sshConn, err := w.db.GetSshConnection(uint(sshConnID))
		if err != nil {
			return nil, fmt.Errorf("failed to get SSH connection details: %w", err)
		}

		// Fetch VIP pool IPs and verify reachability in background, then switch to Prompt mode
		return msg_types.ProcessWithSpinner(func() tea.Msg {
			// Set the VIP pool name in the widget for fetchVipPoolIPs to use
			w.vipPoolName = tempVipPoolName

			// Fetch VIP pool IPs from the active profile
			if err := w.fetchVipPoolIPs(nil); err != nil {
				w.auxlog.Printf("Error fetching VIP pool IPs: %v", err)
				// Reset state on error so user can try again with a different name
				w.vipPoolName = ""
				w.privateIPs = []netip.Addr{}
				return msg_types.ErrorMsg{Err: fmt.Errorf("Failed to fetch VIP pool IPs:\n%w", err)}
			}

			// Verify IP reachability via SSH
			w.auxlog.Printf("Verifying VIP pool IP reachability via SSH...")
			if err := w.verifyIPReachability(sshConn); err != nil {
				w.auxlog.Printf("VIP pool IP verification failed: %v", err)
				// Reset state on error so user can try again
				w.vipPoolName = ""
				w.privateIPs = []netip.Addr{}
				return msg_types.ErrorMsg{Err: fmt.Errorf("VIP pool IP verification failed:\n%w", err)}
			}

			// Successfully fetched IPs and verified reachability, switch to Prompt mode
			w.SetExtraMode(common.ExtraNavigatorModePrompt)
			return nil
		}), nil
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

	// VIP pool IPs should already be fetched and verified (done in background after user entered VIP pool name)
	if len(w.privateIPs) == 0 {
		return nil, fmt.Errorf("no VIP pool IPs available - this should not happen")
	}
	w.auxlog.Printf("Using %d VIP pool IPs for VPN routing", len(w.privateIPs))

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
		// This preserves logs in DetailsAdapter and shows real-time progress in auxlog zone
		// Note: logWriter is initialized to DetailsAdapter by default, and can be updated via SetLogWriter
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

		// Set VIP pool IPs for health monitoring
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

		// Get local hostname for remote directory structure (needed for initial SSH connection)
		hostname, err := os.Hostname()
		if err != nil {
			w.lastError = err
			w.lastStatus = "Failed to get local hostname"
			w.deploying = false
			w.auxlog.Printf("Error: Failed to get local hostname: %v", err)
			return msg_types.ErrorMsg{Err: fmt.Errorf("failed to get local hostname: %w", err)}
		}

		// Step 1: Connect to remote host via SSH (using temporary directory for port check)
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
		// Each connection gets its own directory to avoid conflicts
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

		// Step 5: Start server in background (long-running, streams logs)
		// The server will run until w.ctx is cancelled (when user leaves widget)
		w.lastStatus = "Starting VPN server..."

		// Start server in goroutine - it will stream logs to writer in real-time
		go func() {
			if err := w.deployer.StartServer(w.ctx, deployConfig.RemoteWorkDir, serverConfig); err != nil {
				w.auxlog.Printf("Server stopped: %v", err)
				// Send error message to main app if server exits unexpectedly
				if w.msgChan != nil && w.ctx.Err() == nil {
					w.msgChan <- msg_types.ErrorMsg{Err: fmt.Errorf("VPN server stopped unexpectedly: %w", err)}
				}
			}
		}()

		// Give server a moment to initialize before continuing
		w.auxlog.Printf("Waiting for VPN server to initialize...")
		time.Sleep(3 * time.Second)

		// Start heartbeat monitoring to enable server self-destruction if client disconnects
		w.auxlog.Printf("Starting heartbeat to remote server...")
		if err := w.deployer.StartHeartbeat(deployConfig.RemoteWorkDir); err != nil {
			w.lastError = err
			w.lastStatus = "Failed to start heartbeat"
			w.deploying = false
			w.auxlog.Printf("Error: Failed to start heartbeat: %v", err)
			return msg_types.ErrorMsg{Err: fmt.Errorf("failed to start heartbeat: %w", err)}
		}
		time.Sleep(3 * time.Second)
		w.auxlog.Printf("VPN server should be running now (streaming logs in background)")

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
			PrivateIPs:      w.privateIPs, // Use individual IPs instead of network
		}

		// Step 8: Create VPN client with appropriate writer
		w.lastStatus = "Preparing VPN connection..."
		w.auxlog.Printf("Initiating VPN connection with output display...")
		w.deploying = false

		// Use the same multi-writer approach for client
		// Note: writer was already set up above to write to DetailsAdapter, auxlog, and optionally working zone
		w.vpnClient, err = vpn_client.NewClient(clientConfig, writer)
		if err != nil {
			w.lastError = err
			w.lastStatus = "Failed to create VPN client"
			w.auxlog.Printf("Error: Failed to create VPN client: %v", err)
			return msg_types.ErrorMsg{Err: fmt.Errorf("failed to create VPN client:\n%w", err)}
		}

		// Switch to details mode if using only DetailsAdapter (not working zone)
		if w.logWriter == w.DetailsAdapter {
			w.SetExtraMode(common.ExtraNavigatorModeDetails)
		}

		// Step 9: Connect VPN (this will write logs to the writer)
		if err := w.vpnClient.Connect(w.sudoPassword); err != nil {
			w.lastError = err
			w.lastStatus = "Failed to connect"
			w.auxlog.Printf("Error: VPN connection failed: %v", err)
			return msg_types.ErrorMsg{Err: fmt.Errorf("VPN connection failed:\n%w", err)}
		}

		// Step 10: Connection successful!
		// Note: Server will run until w.ctx is cancelled (when user leaves widget)
		// No need for heartbeat files - the SSH session closing will stop the server

		w.connected = true
		w.lastStatus = "Connected successfully"
		w.lastError = nil

		// Inject msgChan from working zone if not already set
		if w.msgChan == nil {
			// Try to get msgChan from the working zone via database callback
			// This is a workaround since we don't have direct access to working zone here
			w.auxlog.Printf("WARNING: msgChan not set, attempting to retrieve from working zone")
			// For now, we'll set it in Init() method instead
		}

		// Start health monitoring
		w.StartHealthMonitoring(w.ctx)

		return nil
	}, nil
}

// UpdateViewPort overrides the base UpdateViewPort to handle popup submissions
func (w *VipPoolForwarding) UpdateViewPort(msg tea.Msg) tea.Cmd {
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

// Update handles messages for the VIP pool forwarding widget
func (w *VipPoolForwarding) Update(msg tea.Msg) tea.Cmd {
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
			// Call CreateFromInputs again with empty inputs
			cmd, err := w.CreateFromInputs(common.Inputs{})
			if err != nil {
				w.auxlog.Printf("Error calling CreateFromInputs: %v", err)
				return func() tea.Msg { return msg_types.ErrorMsg{Err: err} }
			}

			w.auxlog.Printf("Deployment command created successfully")
			return cmd
		}
	}

	// Check for popup submission (sudo password input)
	if w.needingSudoPassword {
		w.auxlog.Printf("DEBUG Update: needingSudoPassword=true, checking popup state...")
		if w.DetailsAdapter.IsPopupHidden() {
			popupContent := w.DetailsAdapter.GetPopupContent()
			w.auxlog.Printf("DEBUG Update: popup is hidden, content length=%d", len(popupContent))

			if popupContent != "" {
				w.auxlog.Printf("DEBUG Update: processing popup submission")
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
					return nil
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

				// Clear details content and prepare for logs
				w.DetailsAdapter.ClearContent()
				w.DetailsAdapter.AppendContent("Starting VPN deployment...\n\n")

				// Switch to Details mode to show deployment logs
				w.SetExtraMode(common.ExtraNavigatorModeDetails)

				// Now call CreateFromInputs to start deployment
				w.auxlog.Printf("Calling CreateFromInputs after password validated...")
				cmd, err := w.CreateFromInputs(common.Inputs{})
				if err != nil {
					w.auxlog.Printf("Error calling CreateFromInputs: %v", err)
					return func() tea.Msg { return msg_types.ErrorMsg{Err: err} }
				}
				w.auxlog.Printf("Deployment command created successfully")
				return cmd
			}
		} else {
			w.auxlog.Printf("DEBUG Update: popup is still visible, waiting for submission")
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

// ViewDetails displays the VIP pool forwarding connection details
func (w *VipPoolForwarding) ViewDetails() string {
	// Just show the base details view (logs)
	// No additional headers, footers, or status messages
	return w.viewDetails()
}

// Disconnect closes the VIP pool forwarding connection and stops the remote server
func (w *VipPoolForwarding) Disconnect() error {
	w.auxlog.Printf("Disconnecting VPN...")

	// Step 1: Disconnect local VPN client
	// Always clean up the local interface if vpnClient exists, regardless of connected state
	// (connection might be marked as lost but interface is still up)
	if w.vpnClient != nil {
		w.auxlog.Printf("Bringing down local WireGuard interface...")
		// Use cached sudo password (may have expired after long connection)
		// Empty string works if passwordless sudo is configured
		if err := w.vpnClient.Disconnect(w.sudoPassword); err != nil {
			w.auxlog.Printf("Warning: Failed to disconnect VPN client: %v", err)
			// Don't return error, try to clean up other resources
		} else {
			w.auxlog.Printf("Local WireGuard interface cleaned up successfully")
		}
		w.connected = false
		w.lastStatus = "Disconnected"
	} else {
		w.auxlog.Printf("No VPN client to disconnect")
	}

	// Step 2: Close SSH connection (this will automatically stop the remote server)
	// When SSH session closes, the remote server process receives SIGTERM/SIGHUP
	if w.deployer != nil {
		w.auxlog.Printf("Closing SSH connection to remote server...")
		if err := w.deployer.Disconnect(); err != nil {
			w.auxlog.Printf("Warning: Failed to close SSH connection: %v", err)
			// Don't return error
		} else {
			w.auxlog.Printf("SSH connection closed successfully")
		}
	} else {
		w.auxlog.Printf("No SSH deployer to disconnect")
	}

	w.auxlog.Printf("VPN disconnected and cleaned up (local + remote)")
	return nil
}

// LeaveWidget is called when user leaves the widget (e.g., by pressing escape)
// It implements the LeaveWidget interface
func (w *VipPoolForwarding) LeaveWidget() error {
	w.auxlog.Printf("Leaving VIP Pool Forwarding Widget")

	// Cancel context to stop the server goroutine
	// This will close the SSH session, which automatically sends SIGTERM to the remote server
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
func (w *VipPoolForwarding) IsConnected() bool {
	return w.connected && w.vpnClient != nil && w.vpnClient.IsConnected()
}

// StartHealthMonitoring starts monitoring VPN connection health
// Checks both SSH connection (with VIP pool IP ping) and VPN tunnel every 12 seconds
func (w *VipPoolForwarding) StartHealthMonitoring(ctx context.Context) {
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
func (w *VipPoolForwarding) checkHealth() {
	// Check if we're supposed to be connected
	if !w.connected && !w.deploying {
		return // Not connected, no need to check
	}

	w.auxlog.Printf("🔍 Running VPN health check...")

	// Check 1: SSH connection health (for deployer)
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

	// Check 2: VPN tunnel health (ping gateway)
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
func (w *VipPoolForwarding) sendError(err error) {
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
