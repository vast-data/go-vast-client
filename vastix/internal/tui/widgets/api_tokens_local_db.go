package widgets

import (
	"vastix/internal/colors"
	"context"
	"fmt"
	"strings"
	"time"
	"vastix/internal/database"
	"vastix/internal/msg_types"
	"vastix/internal/tui/widgets/common"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ApiTokensFromLocalDb struct {
	*BaseWidget
}

// NewApiTokensFromLocalDb creates a new API tokens widget for local database records
func NewApiTokensFromLocalDb(db *database.Service, msgChan chan tea.Msg, extraWidgets []common.ExtraWidget) common.Widget {
	resourceType := "api_tokens [local database]"
	listHeaders := []string{
		"id", "owner", "name", "created", "expiry_date", "status",
	}

	// Use provided extra widgets (e.g., revoke action from auto-generated widget)
	// If none provided, default to empty array
	if extraWidgets == nil {
		extraWidgets = []common.ExtraWidget{}
	}

	widget := &ApiTokensFromLocalDb{
		BaseWidget: NewBaseWidget(db, listHeaders, nil, resourceType, extraWidgets, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

// API method is not needed for local database widget, but required by interface
func (w *ApiTokensFromLocalDb) API(rest *VMSRest) VastResourceAPIWithContext {
	// This widget reads from local database, not from REST API
	// Return nil since we don't use REST API for this widget
	return nil
}

// GetNotAllowedNavigatorModes returns not allowed navigator modes for this widget
func (w *ApiTokensFromLocalDb) GetNotAllowedNavigatorModes() []common.NavigatorMode {
	return []common.NavigatorMode{
		common.NavigatorModeCreate,
	}
}

// SetListDataWithContext overrides BaseWidget - context not needed for local DB access
func (w *ApiTokensFromLocalDb) SetListDataWithContext(_ context.Context) tea.Msg {
	return w.SetListData()
}

// SetListData retrieves all ApiToken records from the local database
func (w *ApiTokensFromLocalDb) SetListData() tea.Msg {
	rest, err := getActiveRest(w.db)
	if err != nil {
		return msg_types.ErrorMsg{
			Err: fmt.Errorf("failed to get active REST API: %w", err),
		}
	}

	apiTokens, err := w.db.GetApiTokensForActiveProfile()
	if err != nil {
		return msg_types.ErrorMsg{
			Err: fmt.Errorf("failed to retrieve API tokens from database: %w", err),
		}
	}

	// Collect all unique owner IDs to check if they exist
	allOwnerIDs := make([]int64, 0)
	ownerIDMap := make(map[uint]bool)
	for _, token := range apiTokens {
		if token.OwnerID != 0 && !ownerIDMap[token.OwnerID] {
			allOwnerIDs = append(allOwnerIDs, int64(token.OwnerID))
			ownerIDMap[token.OwnerID] = true
		}
	}

	// Collect all token IDs to check their status from backend
	allTokenIDs := make([]string, 0)
	tokenIDMap := make(map[string]bool)
	for _, token := range apiTokens {
		if token.TokenID != "" && !tokenIDMap[token.TokenID] {
			allTokenIDs = append(allTokenIDs, token.TokenID)
			tokenIDMap[token.TokenID] = true
		}
	}

	// Get managers from REST API to check owner existence
	var existingOwnerIDs map[uint]bool
	if len(allOwnerIDs) > 0 {
		managers, err := rest.Managers.List(params{"id__in": allOwnerIDs})
		if err != nil {
			return msg_types.ErrorMsg{
				Err: fmt.Errorf("failed to get managers from REST API: %w", err),
			}
		}

		// Build map of owner ID to existence for quick lookup
		existingOwnerIDs = make(map[uint]bool)
		for _, manager := range managers {
			if idVal, exists := manager["id"]; exists {
				if id, ok := idVal.(float64); ok { // JSON numbers are float64
					existingOwnerIDs[uint(id)] = true
				}
			}
		}
	}

	// Get tokens from REST API to check their status and existence
	var backendTokens map[string]map[string]interface{}
	if len(allTokenIDs) > 0 {
		tokens, err := rest.ApiTokens.List(params{"id__in": allTokenIDs})
		if err != nil {
			return msg_types.ErrorMsg{
				Err: fmt.Errorf("failed to get tokens from REST API: %w", err),
			}
		}

		// Build map of token ID to token data for quick lookup
		backendTokens = make(map[string]map[string]interface{})
		for _, token := range tokens {
			if idVal, exists := token["id"]; exists {
				if id, ok := idVal.(string); ok {
					backendTokens[id] = token
				}
			}
		}
	}

	// Convert ApiToken records to string data for display
	apiTokenData := make([][]string, len(apiTokens))
	for i, token := range apiTokens {
		status := w.determineTokenStatus(token, existingOwnerIDs, backendTokens)

		// Format created date
		createdStr := token.VastCreated.Format("2006-01-02 15:04:05")

		// Format expiry date
		expiryStr := "never"
		if token.ExpireDate != nil {
			expiryStr = token.ExpireDate.Format("2006-01-02 15:04:05")
		}

		apiTokenData[i] = []string{
			token.TokenID, // Use TokenID (API token ID string), not ID (database row number)
			token.Owner,
			token.Name,
			createdStr,
			expiryStr,
			status,
		}
	}

	// Set the data in the list adapter
	w.ListAdapter.SetListData(apiTokenData, w.GetFuzzyListSearchString())

	// Clear any cached selected row data to ensure fresh data is used
	w.SetSelectedRowData(common.RowData{})

	return msg_types.MockMsg{}
}

// RenderRow implements the RenderRow interface for custom API token row styling
func (w *ApiTokensFromLocalDb) RenderRow(rowData common.RowData, isSelected bool, colWidth int) []string {
	if rowData.Len() == 0 {
		return []string{}
	}

	// Get ordered slice from RowData
	styledRow := rowData.ToSlice()

	// Apply styling to status column (index 5) based on status value
	// Only apply styling to non-selected rows
	if len(styledRow) > 5 && !isSelected {
		status := styledRow[5]

		var statusStyle lipgloss.Style
		switch {
		case strings.Contains(status, "active"):
			statusStyle = lipgloss.NewStyle().Foreground(colors.GreenTerm) // Green
		case strings.Contains(status, "expired") || strings.Contains(status, "revoked"):
			statusStyle = lipgloss.NewStyle().Foreground(colors.DarkRed) // Red
		default:
			statusStyle = lipgloss.NewStyle().Foreground(colors.BlackishTerm) // Gray
		}

		styledRow[5] = statusStyle.Render(status)
	}

	return styledRow
}

// determineTokenStatus determines the status of an API token
func (w *ApiTokensFromLocalDb) determineTokenStatus(token database.ApiToken, existingOwnerIDs map[uint]bool, backendTokens map[string]map[string]interface{}) string {
	// Check if token exists in backend - if not found, it means it's revoked
	_, tokenExists := backendTokens[token.TokenID]
	if !tokenExists {
		return "revoked"
	}

	// Check if owner exists by ID
	if token.OwnerID == 0 {
		return "owner not found"
	}

	ownerExists := existingOwnerIDs[token.OwnerID]
	if !ownerExists {
		return "owner not found"
	}

	// Check if token is expired
	if token.ExpireDate != nil && token.ExpireDate.Before(time.Now()) {
		return "expired"
	}

	return "active"
}

// Details implements the DetailsWidget interface - called when pressing 'd' to view details
func (w *ApiTokensFromLocalDb) Details(selectedRowData common.RowData) (tea.Cmd, error) {
	if selectedRowData.Len() == 0 {
		return func() tea.Msg {
			return msg_types.DetailsContentMsg{
				Content:      "No API token selected",
				ResourceType: w.resourceType,
				Error:        fmt.Errorf("no API token selected"),
			}
		}, nil
	}

	// Get tokenID as a string (e.g., "XryvUqOg")
	tokenID := selectedRowData.GetString("id")
	if tokenID == "" {
		return func() tea.Msg {
			return msg_types.DetailsContentMsg{
				Content:      "Invalid API token ID",
				ResourceType: w.resourceType,
				Error:        fmt.Errorf("API token ID is empty"),
			}
		}, nil
	}

	// Return async command that will load details from database
	return func() tea.Msg {
		// Get the active profile
		profile, err := w.db.GetActiveProfile()
		if err != nil {
			return msg_types.DetailsContentMsg{
				Content:      fmt.Sprintf("Failed to get active profile: %v", err),
				ResourceType: w.resourceType,
				Error:        err,
			}
		}

		// Fetch the complete ApiToken record from database by TokenID
		apiToken, err := w.db.GetApiTokenByTokenID(tokenID, profile.ID)
		if err != nil {
			return msg_types.DetailsContentMsg{
				Content:      fmt.Sprintf("Failed to fetch API token details: %v", err),
				ResourceType: w.resourceType,
				Error:        err,
			}
		}

		// Create a map with all ApiToken fields
		tokenData := map[string]any{
			"id":          apiToken.TokenID,
			"token":       apiToken.Token,
			"name":        apiToken.Name,
			"owner":       apiToken.Owner,
			"owner_id":    apiToken.OwnerID,
			"expire_date": apiToken.ExpireDate,
			"created":     apiToken.VastCreated,
		}

		// Set the details data and return success message
		w.DetailsAdapter.SetContent(tokenData)
		return msg_types.DetailsContentMsg{
			Content:      tokenData, // The DetailsAdapter will format this as JSON
			ResourceType: w.resourceType,
			Error:        nil,
		}
	}, nil
}

// Delete implements the DeleteWidget interface - called when pressing Ctrl+d to delete an API token
func (w *ApiTokensFromLocalDb) Delete(selectedRowData common.RowData) (tea.Cmd, error) {
	if selectedRowData.Len() == 0 {
		return nil, fmt.Errorf("no row data provided for deletion")
	}

	// Extract ID from the row data
	tokenId, err := selectedRowData.GetIntID()
	if err != nil {
		return nil, fmt.Errorf("failed to get API token ID from selected row: %w", err)
	}

	// Return async command that will delete the API token from database
	return func() tea.Msg {
		// Delete the API token from database
		err := w.db.DeleteApiToken(uint(tokenId))
		if err != nil {
			return msg_types.ErrorMsg{
				Err: fmt.Errorf("failed to delete API token: %w", err),
			}
		}

		w.SetListData()
		w.SetModeMust(common.NavigatorModeList)
		return nil

	}, nil
}
