package widgets

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
	"vastix/internal/database"
	"vastix/internal/msg_types"
	"vastix/internal/tui/widgets/common"

	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

type SshConnections struct {
	*BaseWidget
}

// NewSshConnections creates a new SSH connections widget
func NewSshConnections(db *database.Service) common.Widget {
	resourceType := "ssh_connections"
	listHeaders := []string{"id", "name", "ssh_host", "ssh_user_name", "ssh_port", "auth_method"}

	extraNav := []common.ExtraWidget{}

	widget := &SshConnections{
		NewBaseWidget(db, listHeaders, nil, resourceType, extraNav, nil),
	}

	widget.SetParentForBaseWidget(widget, false)
	return widget
}

// testSshConnection tests an SSH connection by executing the "test" command
func testSshConnection(host string, port int, username, password, keyPath string) error {
	// Skip testing for "local [pseudo ssh]" connection
	if username == "-" && password == "-" && keyPath == "-" {
		return nil // Local connection doesn't need testing
	}

	// Create SSH client config
	config := &ssh.ClientConfig{
		User:            username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: In production, use proper host key verification
		Timeout:         10 * time.Second,
	}

	// Set up authentication
	if password != "" && password != "-" {
		config.Auth = append(config.Auth, ssh.Password(password))
	}

	if keyPath != "" && keyPath != "-" {
		// Read private key file
		key, err := ioutil.ReadFile(keyPath)
		if err != nil {
			return fmt.Errorf("failed to read private key file %s: %w", keyPath, err)
		}

		// Parse private key
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}

		config.Auth = append(config.Auth, ssh.PublicKeys(signer))
	}

	// Connect to SSH server
	address := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server %s: %w", address, err)
	}
	defer client.Close()

	// Create SSH session
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Execute a safe test command that should always succeed
	output, err := session.CombinedOutput("echo 'connection test'")
	if err != nil {
		return fmt.Errorf("failed to execute test command: %w (output: %s)", err, string(output))
	}

	return nil
}

func (s *SshConnections) SetListData() tea.Msg {
	// Initialize with SSH connection data from database
	db := s.BaseWidget.db
	connections, err := db.GetAllSshConnections()

	if err != nil {
		return msg_types.ErrorMsg{
			Err: err,
		}
	} else {
		connectionData := make([][]string, len(connections))
		for i, conn := range connections {
			// Determine auth method
			authMethod := ""
			if conn.SshPassword != "" && conn.SshKey != "" {
				authMethod = "password+key"
			} else if conn.SshPassword != "" {
				authMethod = "password"
			} else if conn.SshKey != "" {
				authMethod = "key"
			} else {
				authMethod = "none"
			}

			connectionData[i] = []string{
				fmt.Sprintf("%d", conn.ID),
				conn.Name,
				conn.SshHost,
				conn.SshUserName,
				fmt.Sprintf("%d", conn.SshPort),
				authMethod,
			}
		}
		s.ListAdapter.SetListData(connectionData, s.GetFuzzyListSearchString())
	}
	return msg_types.MockMsg{}
}

func (s *SshConnections) GetInputs() (common.Inputs, error) {
	inputs := make(common.Inputs, 0, 6)

	inputs.NewTextInput("name", "My SSH Connection", true, "Connection name (required, e.g. 'server1')")
	inputs.NewTextInput("ssh_host", "192.168.1.100", true, "SSH host/IP address (required)")
	inputs.NewTextInput("ssh_user_name", "root", true, "SSH username (required)")
	inputs.NewSecretTextInput("ssh_password", "password123", false, "SSH password (optional if key provided)")
	inputs.NewTextInput("ssh_key", "/path/to/private/key", false, "Path to private SSH key (optional if password provided)")
	inputs.NewInt64Input("ssh_port", "22 (default)", false, 22)

	// Get existing SSH key paths for suggestions
	if sshKeySuggestions := s.getSshKeySuggestions(); len(sshKeySuggestions) > 0 {
		inputs.Field("ssh_key").SetSuggestions(sshKeySuggestions)
	}

	return inputs, nil
}

// getSshKeySuggestions returns a deduplicated list of existing SSH key paths from the database
func (s *SshConnections) getSshKeySuggestions() []string {
	db := s.BaseWidget.db
	connections, err := db.GetAllSshConnections()
	if err != nil {
		s.log.Debug("Failed to get SSH connections for key suggestions", zap.Error(err))
		return nil
	}

	// Use a map to deduplicate SSH key paths
	keyPathsMap := make(map[string]bool)

	for _, conn := range connections {
		// Skip empty, "-", and whitespace-only SSH keys
		keyPath := strings.TrimSpace(conn.SshKey)
		if keyPath != "" && keyPath != "-" {
			keyPathsMap[keyPath] = true
		}
	}

	// Convert map keys to slice
	var suggestions []string
	for keyPath := range keyPathsMap {
		suggestions = append(suggestions, keyPath)
	}

	s.log.Debug("Generated SSH key suggestions",
		zap.Int("total_connections", len(connections)),
		zap.Int("unique_keys", len(suggestions)))

	return suggestions
}

func (s *SshConnections) CreateFromInputs(inputs common.Inputs) (tea.Cmd, error) {
	s.log.Info("Creating SSH connection from inputs", zap.Any("inputs", inputs.GetValues()))
	db := s.BaseWidget.db

	if err := inputs.Validate(); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(inputs.Field("name").String())
	sshHost := strings.TrimSpace(inputs.Field("ssh_host").String())
	sshUserName := strings.TrimSpace(inputs.Field("ssh_user_name").String())
	sshPassword := inputs.Field("ssh_password").String()
	sshKey := strings.TrimSpace(inputs.Field("ssh_key").String())
	sshPort := inputs.Field("ssh_port").Int64()

	// Validate required fields
	if name == "" {
		return nil, fmt.Errorf("connection name is required")
	}
	if sshHost == "" {
		return nil, fmt.Errorf("SSH host is required")
	}
	if sshUserName == "" {
		return nil, fmt.Errorf("SSH username is required")
	}

	// Validate that at least one authentication method is provided
	if sshPassword == "" && sshKey == "" {
		return nil, fmt.Errorf("either SSH password or SSH key must be provided")
	}

	// Set default port if not provided
	if sshPort <= 0 {
		sshPort = 22
	}

	createConnectionFn := func() tea.Msg {
		// Test SSH connection first (unless it's a "local [pseudo ssh]" connection)
		if name != "local [pseudo ssh]" {
			s.log.Info("Testing SSH connection",
				zap.String("host", sshHost),
				zap.String("username", sshUserName),
				zap.Int64("port", sshPort))

			if err := testSshConnection(sshHost, int(sshPort), sshUserName, sshPassword, sshKey); err != nil {
				s.log.Error("SSH connection test failed", zap.Error(err))
				return msg_types.ErrorMsg{
					Err: fmt.Errorf("SSH connection test failed: %w", err),
				}
			}

			s.log.Info("SSH connection test successful")
		}

		// Create SSH connection in database
		sshConn := &database.SshConnection{
			Name:        name,
			SshHost:     sshHost,
			SshUserName: sshUserName,
			SshPassword: sshPassword,
			SshKey:      sshKey,
			SshPort:     int(sshPort),
		}

		if err := db.CreateSshConnection(sshConn); err != nil {
			return msg_types.ErrorMsg{
				Err: fmt.Errorf("failed to create SSH connection: %w", err),
			}
		}

		s.log.Info("SSH connection created successfully",
			zap.String("name", sshConn.Name),
			zap.String("host", sshConn.SshHost),
			zap.String("username", sshConn.SshUserName))

		// Refresh the list and switch to list mode
		s.SetListData()
		s.SetModeMust(common.NavigatorModeList)
		return msg_types.SetDataMsg{
			UseSpinner: false,
		}
	}

	return createConnectionFn, nil
}

// Select implements the Selectable interface - called when an SSH connection row is selected
func (s *SshConnections) Select(selectedRowData common.RowData) (tea.Cmd, error) {
	if selectedRowData.Len() == 0 {
		return nil, fmt.Errorf("no row data provided for selection")
	}

	s.log.Debug("SSH connection selection received",
		zap.Any("rowData", selectedRowData),
		zap.Int("dataLength", selectedRowData.Len()))

	// For SSH connections, selection just shows details (no activation like profiles)
	return nil, nil
}

func (s *SshConnections) GetKeyBindings() []common.KeyBinding {
	var keyBindings []common.KeyBinding
	switch s.WidgetNavigator.GetMode() {
	case common.NavigatorModeList:
		keyBindings = []common.KeyBinding{
			{Key: "<:>", Desc: "resources", Generic: true},
			{Key: "</>", Desc: "search", Generic: true},
			{Key: "<↑/↓>", Desc: "navigate"},
			{Key: "<enter>", Desc: "details"},
			{Key: "<d>", Desc: "describe"},
			{Key: "<n>", Desc: "new"},
			{Key: "<ctrl+d>", Desc: "delete"},
		}
	case common.NavigatorModeCreate:
		keyBindings = []common.KeyBinding{
			{Key: "<tab>", Desc: "next input"},
			{Key: "<shift+tab>", Desc: "previous input"},
			{Key: "<enter>", Desc: "submit"},
			{Key: "<esc>", Desc: "back"},
			{Key: "<space>", Desc: "toggle boolean"},
		}
	case common.NavigatorModeDelete:
		keyBindings = []common.KeyBinding{
			{Key: "<y or enter>", Desc: "confirm"},
			{Key: "<n or esc>", Desc: "cancel"},
		}
	case common.NavigatorModeDetails:
		keyBindings = []common.KeyBinding{
			{Key: "</>", Desc: "search", Generic: true},
			{Key: "<↑/↓>", Desc: "scroll"},
			{Key: "<pgup/pgdn>", Desc: "page"},
			{Key: "<esc>", Desc: "back"},
		}
	}

	return keyBindings
}

// RenderRow implements the RenderRow interface for custom SSH connection row styling
func (s *SshConnections) RenderRow(rowData common.RowData, isSelected bool, colWidth int) []string {
	if rowData.Len() == 0 {
		return []string{}
	}

	// Get ordered slice from RowData - no special styling, use default colors
	styledRow := rowData.ToSlice()

	return styledRow
}

// Delete implements the DeleteWidget interface for SSH connection deletion
func (s *SshConnections) Delete(selectedRowData common.RowData) (tea.Cmd, error) {
	if selectedRowData.Len() == 0 {
		return nil, fmt.Errorf("no row data provided for deletion")
	}
	db := s.BaseWidget.db

	s.log.Debug("SSH connection deletion received",
		zap.Any("rowData", selectedRowData),
		zap.Int("dataLength", selectedRowData.Len()))

	// Extract ID from the row data
	id := selectedRowData.GetInt64Must("id")
	connection, err := db.GetSshConnection(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("SSH connection not found")
		}
		return nil, fmt.Errorf("failed to find SSH connection: %w", err)
	}

	// Protect the default "local [pseudo ssh]" connection from deletion
	if connection.Name == "local [pseudo ssh]" {
		return nil, fmt.Errorf("cannot delete default 'local [pseudo ssh]' connection")
	}

	// Delete the SSH connection from database
	if err := db.DeleteSshConnection(connection.ID); err != nil {
		return nil, fmt.Errorf("failed to delete SSH connection: %w", err)
	}

	s.log.Info("SSH connection deleted successfully",
		zap.String("name", connection.Name),
		zap.String("username", connection.SshUserName))

	return func() tea.Msg {
		s.SetListData()
		s.SetModeMust(common.NavigatorModeList)
		return nil
	}, nil
}

// Details implements the Detailable interface for SSH connection details
func (s *SshConnections) Details(selectedRowData common.RowData) (tea.Cmd, error) {
	if selectedRowData.Len() == 0 {
		return func() tea.Msg {
			return msg_types.DetailsContentMsg{
				Content:      "No SSH connection selected",
				ResourceType: s.resourceType,
				Error:        nil,
			}
		}, nil
	}

	s.log.Debug("SSH connection details requested",
		zap.Any("rowData", selectedRowData),
		zap.Int("dataLength", selectedRowData.Len()))

	// Extract connection ID from the row data
	idStr := selectedRowData.GetID()
	if idStr == "" {
		return func() tea.Msg {
			return msg_types.DetailsContentMsg{
				Content:      "Invalid SSH connection data: missing ID",
				ResourceType: s.resourceType,
				Error:        fmt.Errorf("missing connection ID"),
			}
		}, nil
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return func() tea.Msg {
			return msg_types.DetailsContentMsg{
				Content:      fmt.Sprintf("Invalid connection ID: %s", idStr),
				ResourceType: s.resourceType,
				Error:        err,
			}
		}, nil
	}

	// Return async command that will load details in background
	return func() tea.Msg {
		db := s.BaseWidget.db
		connection, err := db.GetSshConnection(uint(id))
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return msg_types.DetailsContentMsg{
					Content:      "SSH connection not found",
					ResourceType: s.resourceType,
					Error:        err,
				}
			}
			return msg_types.DetailsContentMsg{
				Content:      fmt.Sprintf("Failed to load SSH connection: %v", err),
				ResourceType: s.resourceType,
				Error:        err,
			}
		}
		// Convert connection to map[string]any for JSON formatting
		connectionMap := make(map[string]any)
		connectionMap["id"] = connection.ID
		connectionMap["name"] = connection.Name
		connectionMap["ssh_host"] = connection.SshHost
		connectionMap["ssh_user_name"] = connection.SshUserName
		connectionMap["ssh_password"] = connection.SshPassword
		connectionMap["ssh_key"] = connection.SshKey
		connectionMap["ssh_port"] = connection.SshPort
		connectionMap["created_at"] = connection.CreatedAt.Format("2006-01-02T15:04:05Z")
		connectionMap["updated_at"] = connection.UpdatedAt.Format("2006-01-02T15:04:05Z")

		s.log.Info("SSH connection details generated",
			zap.String("name", connection.Name),
			zap.Uint("connection_id", connection.ID))

		return msg_types.DetailsContentMsg{
			Content:      connectionMap,
			ResourceType: s.resourceType,
			Error:        nil,
		}
	}, nil
}

func (*SshConnections) DetailsOnSelect() bool {
	// Enable "Enter" key to select details for SSH connections
	return true
}
