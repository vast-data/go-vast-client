package widgets

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"vastix/internal/client"
	"vastix/internal/database"
	log "vastix/internal/logging"
	"vastix/internal/msg_types"
	"vastix/internal/tui/widgets/common"

	"gorm.io/gorm"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.uber.org/zap"
)

type Profile struct {
	*BaseWidget
}

// NewProfile creates a new profile widget
func NewProfile(db *database.Service) common.Widget {
	resourceType := "profiles"
	listHeaders := []string{"id", "endpoint", "alias", "vast", "status", "username", "password", "token", "tenant", "api_version"}

	extraNav := []common.ExtraWidget{}

	widget := &Profile{
		NewBaseWidget(db, listHeaders, nil, resourceType, extraNav, nil),
	}

	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (p *Profile) SetListData() tea.Msg {
	// Initialize with profile data from database
	db := p.BaseWidget.db
	profiles, err := db.GetAllProfiles()

	if err != nil {
		return msg_types.ErrorMsg{
			Err: err,
		}
	} else {
		profileData := make([][]string, len(profiles))
		for i, p := range profiles {
			status := ""
			if p.Active {
				status = "[active]"
			}
			password := ""
			if p.Password != "" {
				password = "*****"
			}
			token := ""
			if p.Token != "" {
				token = "*****"
			}

			profileData[i] = []string{
				fmt.Sprintf("%d", p.ID),
				fmt.Sprintf("%s:%d", p.Endpoint, p.Port),
				p.Alias,
				p.VastVersion,
				status,
				p.Username,
				password,
				token,
				p.Tenant,
				p.ApiVersion,
			}
		}
		p.ListAdapter.SetListData(profileData, p.GetFuzzyListSearchString())
	}
	return msg_types.MockMsg{}
}

func (p *Profile) GetInputs() (common.Inputs, error) {
	inputs := make(common.Inputs, 0, 9)

	inputs.NewTextInput("endpoint", "vast.example.com", true, "")
	inputs.NewTextInput("username", "admin", false, "")
	inputs.NewSecretTextInput("password", "123456", false, "")
	inputs.NewTextInput("token", "auth-token (optional)", false, "")
	inputs.NewTextInput("tenant", "tenant (optional)", false, "")
	inputs.NewInt64Input("port", "443 (default)", false, 0)
	inputs.NewBoolInput("ssl_verify", "Verify SSL certificates", false, false)
	inputs.NewTextInput("alias", "Short profile name (optional)", false, "")
	inputs.NewTextInput("api_version", "API version e.g., v1, latest (default: latest)", false, "")

	return inputs, nil
}

func (p *Profile) CreateFromInputs(inputs common.Inputs) (tea.Cmd, error) {
	p.log.Info("Creating profile from inputs", zap.Any("inputs", inputs.GetValues()))
	db := p.BaseWidget.db

	if err := inputs.Validate(); err != nil {
		return nil, err
	}

	alias := inputs.Field("alias").String()
	endpoint := inputs.Field("endpoint").String()
	username := inputs.Field("username").String()
	password := inputs.Field("password").String()
	token := inputs.Field("token").String()
	tenant := inputs.Field("tenant").String()
	port := inputs.Field("port").Int64()
	sslVerify := inputs.Field("ssl_verify").Bool()
	apiVersion := strings.TrimSpace(inputs.Field("api_version").String())

	log.Debug("Profile creation values",
		zap.String("endpoint", endpoint),
		zap.String("username", username),
		zap.String("tenant", tenant),
		zap.Int64("port", port),
		zap.Bool("ssl_verify", sslVerify))

	// Validate authentication: either token OR username+password must be provided
	hasToken := strings.TrimSpace(token) != ""
	hasUsernamePassword := strings.TrimSpace(username) != "" && strings.TrimSpace(password) != ""

	if !hasToken && !hasUsernamePassword {
		return nil, fmt.Errorf("authentication required: provide either API token or both username and password")
	}

	// Validate apiVersion: allow empty (defaults to latest), or validate "latest" or v<digits>
	var lower string
	if apiVersion == "" {
		lower = "latest"
	} else {
		lower = strings.ToLower(apiVersion)
		if lower != "latest" {
			if !(strings.HasPrefix(lower, "v") && len(lower) > 1) {
				return nil, fmt.Errorf("invalid api_version: must be 'latest' or 'v1', 'v2', 'v5' etc")
			}
			if _, err := strconv.Atoi(lower[1:]); err != nil {
				return nil, fmt.Errorf("invalid api_version: version number after 'v' must be digits")
			}
		}
	}
	apiVersion = lower

	initProfileFn := func() tea.Msg {
		// Create configuration for the new cached client system
		config := client.RestClientConfig{
			Host:       endpoint,
			Port:       port,
			Username:   username,
			Password:   password,
			ApiToken:   token,
			Tenant:     tenant,
			SslVerify:  sslVerify,
			ApiVersion: apiVersion,
		}

		// Use the new cached client system
		rest, err := client.GetGlobalClient(config)
		if err != nil {
			return msg_types.ErrorMsg{
				Err: err,
			}
		}

		// Test the connection by getting version
		version, err := rest.Versions.GetVersionWithContext(context.Background())
		if err != nil {
			return msg_types.ErrorMsg{
				Err: err,
			}
		}

		p.log.Info("Rest client initialized", zap.Any("VAST version", version))

		// Create profile in database
		profile := &database.Profile{
			Alias:       alias,
			Endpoint:    endpoint,
			Username:    username,
			Password:    password,
			Token:       token,
			Tenant:      tenant,
			Port:        port,
			SSLVerify:   sslVerify,
			VastVersion: version.String(),
			ApiVersion:  apiVersion,
		}

		log.Debug("Profile struct before DB creation",
			zap.String("endpoint", profile.Endpoint),
			zap.Bool("ssl_verify", profile.SSLVerify))

		// Use atomic operation to create profile as active, deactivating all others
		if err = db.CreateProfileAsActive(profile); err != nil {
			return msg_types.ErrorMsg{
				Err: fmt.Errorf("failed to create profile as active: %w", err),
			}
		}

		p.log.Info("Profile created successfully", zap.String("profile_name", profile.ProfileName()))

		// Return the REST client to propagate it to other widgets
		return msg_types.InitProfileMsg{
			Client: rest,
		}
	}

	return initProfileFn, nil

}

// Select implements the Selectable interface - called when a profile row is selected
func (p *Profile) Select(selectedRowData common.RowData) (tea.Cmd, error) {
	if selectedRowData.Len() == 0 {
		return nil, fmt.Errorf("no row data provided for selection")
	}
	db := p.BaseWidget.db

	log.Debug("Profile selection received",
		zap.Any("rowData", selectedRowData),
		zap.Int("dataLength", selectedRowData.Len()))

	activeProfile, err := db.GetActiveProfile()
	if err != nil {
		return nil, err
	}

	// Extract ID from the row data
	id := selectedRowData.GetInt64Must("id")
	if int64(activeProfile.ID) == id {
		// If the selected profile is already active, do nothing
		log.Debug("Profile already active, no action taken",
			zap.Uint("profile_id", activeProfile.ID),
			zap.String("endpoint", selectedRowData.GetString("endpoint")))
		return nil, nil
	}

	profile, err := db.GetProfile(uint64(id))
	if err != nil {
		return nil, fmt.Errorf("failed to find profile: %w", err)
	}

	// Set this profile as active
	if err := db.SetActiveProfile(profile.ID); err != nil {
		return nil, fmt.Errorf("failed to set active profile: %w", err)
	}

	p.log.Info("Profile activated",
		zap.String("endpoint", profile.Endpoint),
		zap.String("username", profile.Username))

	updateVastVersionFn := func() tea.Msg {
		newActiveProfile, err := db.GetActiveProfile()
		if err != nil {
			log.Error("Failed to get active profile", zap.Error(err))
			return msg_types.ErrorMsg{Err: fmt.Errorf("failed to get active profile: %w", err)}
		}
		rest, err := newActiveProfile.RestClientFromProfile()
		if err != nil {
			log.Error("Failed to get REST client from profile", zap.Error(err))
			return msg_types.ErrorMsg{Err: fmt.Errorf("failed to get REST client from profile: %w", err)}
		}
		// Test the connection by getting version
		version, err := rest.Versions.GetVersionWithContext(context.Background())
		if err != nil {
			log.Error("Failed to get VAST version from new active profile", zap.Error(err))
			// Revert to previous active profile
			db.SetActiveProfile(activeProfile.ID)
			return msg_types.ErrorMsg{
				Err: err,
			}
		} else {
			if version.String() != newActiveProfile.VastVersion {
				newActiveProfile.VastVersion = version.String()
				if err := db.UpdateProfile(newActiveProfile); err != nil {
					log.Error("Failed to update profile with new version", zap.Error(err))
					return msg_types.ErrorMsg{Err: fmt.Errorf("failed to update profile with new version: %w", err)}
				} else {
					p.log.Info("Profile updated with new VAST version", zap.String("version", version.String()))
				}
			}
		}

		p.log.Info("Rest client initialized", zap.Any("VAST version", version))
		return msg_types.UpdateProfileMsg{}
	}

	// Return command to update the profile zone and other components
	return msg_types.ProcessWithSpinner(updateVastVersionFn), nil
}

func (p *Profile) GetKeyBindings() []common.KeyBinding {
	var keyBindings []common.KeyBinding
	switch p.WidgetNavigator.GetMode() {
	case common.NavigatorModeList:
		keyBindings = []common.KeyBinding{
			{Key: "<:>", Desc: "resources", Generic: true},
			{Key: "</>", Desc: "search", Generic: true},
			{Key: "<↑/↓>", Desc: "navigate"},
			{Key: "<enter>", Desc: "select"},
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

// RenderRow implements the RenderRow interface for custom profile row styling
func (p *Profile) RenderRow(rowData common.RowData, isSelected bool, colWidth int) []string {
	if rowData.Len() == 0 {
		return []string{}
	}

	// Get ordered slice from RowData
	styledRow := rowData.ToSlice()

	// Apply styling to cells that contain "[active]"
	for i, cell := range styledRow {
		if strings.Contains(cell, "[active]") && !isSelected {
			activeNormalStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("2")) // Green
			styledRow[i] = activeNormalStyle.Render(cell)
		}
	}

	return styledRow
}

// Delete implements the DeleteWidget interface for profile deletion
func (p *Profile) Delete(selectedRowData common.RowData) (tea.Cmd, error) {
	if selectedRowData.Len() == 0 {
		return nil, fmt.Errorf("no row data provided for deletion")
	}
	db := p.BaseWidget.db

	log.Debug("Profile deletion received",
		zap.Any("rowData", selectedRowData),
		zap.Int("dataLength", selectedRowData.Len()))

	// Extract ID from the row data
	id := selectedRowData.GetInt64Must("id")
	profile, err := db.GetProfile(uint64(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No active profile found - return nil profile with no error
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find profile: %w", err)
	}

	if profile.Active {
		// If the profile is active, we cannot delete it
		return nil, fmt.Errorf("cannot delete active profile: %s", profile.ProfileName())
	}

	// Delete the profile from database
	if err := db.DeleteProfile(profile.ID); err != nil {
		return nil, fmt.Errorf("failed to delete profile: %w", err)
	}

	p.log.Info("Profile deleted successfully",
		zap.String("endpoint", profile.Endpoint),
		zap.String("username", profile.Username))

	return func() tea.Msg {
		p.SetListData()
		p.SetModeMust(common.NavigatorModeList)
		return nil
	}, nil
}

// Details implements the Detailable interface for profile details
func (p *Profile) Details(selectedRowData common.RowData) (tea.Cmd, error) {
	if selectedRowData.Len() == 0 {
		return func() tea.Msg {
			return msg_types.DetailsContentMsg{
				Content:      "No profile selected",
				ResourceType: p.resourceType,
				Error:        nil,
			}
		}, nil
	}

	log.Debug("Profile details requested",
		zap.Any("rowData", selectedRowData),
		zap.Int("dataLength", selectedRowData.Len()))

	// Extract profile ID from the row data
	idStr := selectedRowData.GetID()
	if idStr == "" {
		return func() tea.Msg {
			return msg_types.DetailsContentMsg{
				Content:      "Invalid profile data: missing ID",
				ResourceType: p.resourceType,
				Error:        fmt.Errorf("missing profile ID"),
			}
		}, nil
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return func() tea.Msg {
			return msg_types.DetailsContentMsg{
				Content:      fmt.Sprintf("Invalid profile ID: %s", idStr),
				ResourceType: p.resourceType,
				Error:        err,
			}
		}, nil
	}

	// Return async command that will load details in background
	return func() tea.Msg {
		db := p.BaseWidget.db
		profile, err := db.GetProfile(id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return msg_types.DetailsContentMsg{
					Content:      "Profile not found",
					ResourceType: p.resourceType,
					Error:        err,
				}
			}
			return msg_types.DetailsContentMsg{
				Content:      fmt.Sprintf("Failed to load profile: %v", err),
				ResourceType: p.resourceType,
				Error:        err,
			}
		}

		// Format profile details with JSON-style syntax highlighting

		// Convert profile to map[string]any for generic formatting
		profileMap := make(map[string]any)
		profileMap["id"] = profile.ID

		if profile.Alias != "" {
			profileMap["alias"] = profile.Alias
		}

		profileMap["endpoint"] = profile.Endpoint
		profileMap["port"] = profile.Port
		profileMap["username"] = profile.Username

		// Show raw password
		if profile.Password != "" {
			profileMap["password"] = profile.Password
		} else {
			profileMap["password"] = nil
		}

		// Show raw token
		if profile.Token != "" {
			profileMap["token"] = profile.Token
		} else {
			profileMap["token"] = nil
		}

		if profile.Tenant != "" {
			profileMap["tenant"] = profile.Tenant
		}

		profileMap["ssl_verify"] = profile.SSLVerify
		profileMap["active"] = profile.Active

		if profile.VastVersion != "" {
			profileMap["vast_version"] = profile.VastVersion
		}

		// API version (required)
		profileMap["api_version"] = profile.ApiVersion

		profileMap["created_at"] = profile.CreatedAt.Format("2006-01-02T15:04:05Z")
		profileMap["updated_at"] = profile.UpdatedAt.Format("2006-01-02T15:04:05Z")

		p.log.Info("Profile details generated",
			zap.String("endpoint", profile.Endpoint),
			zap.Uint("profile_id", profile.ID))

		return msg_types.DetailsContentMsg{
			Content:      profileMap,
			ResourceType: p.resourceType,
			Error:        nil,
		}
	}, nil
}

func (*Profile) DetailsOnSelect() bool {
	// Whether to use "Enter" key to select details. For this widget it is disabled.
	return false
}
