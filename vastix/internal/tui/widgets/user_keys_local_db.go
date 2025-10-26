package widgets

import (
	"fmt"
	"strconv"
	"strings"
	"vastix/internal/database"
	"vastix/internal/msg_types"
	"vastix/internal/tui/widgets/common"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type UserKeysFromLocalDb struct {
	*BaseWidget
}

// NewUserKeysFromLocalDb creates a new user keys widget for local database records
func NewUserKeysFromLocalDb(db *database.Service) common.Widget {
	resourceType := "user_keys [local store]"
	listHeaders := []string{
		"id", "user_id", "username", "non_local", "access_key", "status",
	}

	// No form hints needed - this is a read-only widget
	extraNav := []common.ExtraWidget{}

	widget := &UserKeysFromLocalDb{
		BaseWidget: NewBaseWidget(db, listHeaders, nil, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

// API method is not needed for local database widget, but required by interface
func (w *UserKeysFromLocalDb) API(rest *VMSRest) VastResourceAPI {
	// This widget reads from local database, not from REST API
	// Return nil since we don't use REST API for this widget
	return nil
}

// GetNotAllowedNavigatorModes returns not allowed navigator modes for this widget
func (w *UserKeysFromLocalDb) GetNotAllowedNavigatorModes() []common.NavigatorMode {
	return []common.NavigatorMode{
		common.NavigatorModeCreate,
	}
}

// SetListData retrieves all UserKey records from the local database
func (w *UserKeysFromLocalDb) SetListData() tea.Msg {
	rest, err := getActiveRest(w.db)
	if err != nil {
		return msg_types.ErrorMsg{
			Err: fmt.Errorf("failed to get active REST API: %w", err),
		}
	}

	userKeys, err := w.db.GetUserKeysForActiveProfile()
	if err != nil {
		return msg_types.ErrorMsg{
			Err: fmt.Errorf("failed to retrieve user keys from database: %w", err),
		}
	}

	allLocalIDs := make([]int64, 0)
	for _, key := range userKeys {
		if key.UserID != 0 {
			allLocalIDs = append(allLocalIDs, key.UserID)
		}
	}

	// Get users from REST API to check key status
	var userMap map[int64]map[string]interface{}
	if len(allLocalIDs) > 0 {
		users, err := rest.Users.List(params{"id__in": allLocalIDs})
		if err != nil {
			return msg_types.ErrorMsg{
				Err: fmt.Errorf("failed to get users from REST API: %w", err),
			}
		}

		// Build map of user ID to user data for quick lookup
		userMap = make(map[int64]map[string]interface{})
		for _, user := range users {
			if idVal, exists := user["id"]; exists {
				if id, ok := idVal.(float64); ok { // JSON numbers are float64
					userMap[int64(id)] = user
				}
			}
		}
	}

	// Convert UserKey records to string data for display
	// Note: Secret key is NOT included in the list view for security
	// Additional fields (profile_id, user_uid, timestamps) are shown only in details view
	userKeyData := make([][]string, len(userKeys))
	for i, key := range userKeys {
		status := w.determineKeyStatus(key, userMap)

		userKeyData[i] = []string{
			strconv.FormatUint(uint64(key.ID), 10),
			strconv.FormatInt(key.UserID, 10),
			key.Username,
			strconv.FormatBool(key.NonLocal),
			key.AccessKey,
			status,
		}
	}

	// Set the data in the list adapter
	w.ListAdapter.SetListData(userKeyData, w.GetFuzzyListSearchString())
	return msg_types.MockMsg{}
}

// RenderRow implements the RenderRow interface for custom user key row styling
func (w *UserKeysFromLocalDb) RenderRow(rowData common.RowData, isSelected bool, colWidth int) []string {
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
		case strings.Contains(status, "enabled"):
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // Green
		case strings.Contains(status, "disabled"):
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // Red
		default:
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // Gray
		}

		styledRow[5] = statusStyle.Render(status)
	}

	return styledRow
}

// determineKeyStatus determines the status of a user key based on REST API data
func (w *UserKeysFromLocalDb) determineKeyStatus(key database.UserKey, userMap map[int64]map[string]interface{}) string {
	// If UserID is 0, it's a non-local user
	if key.UserID == 0 {
		return "n/a (non-local)"
	}

	userID := key.UserID

	// Check if user exists in the API response
	userData, userExists := userMap[userID]
	if !userExists {
		return "user not found"
	}

	// Get access_keys field from user data
	accessKeysVal, accessKeysExists := userData["access_keys"]
	if !accessKeysExists {
		return "no access keys field"
	}

	// Parse access_keys - it should be an array of maps
	accessKeys, ok := accessKeysVal.([]interface{})
	if !ok {
		return "invalid access keys format"
	}

	// Look for our specific access key in the user's access_keys
	for _, keyData := range accessKeys {
		keyMap, ok := keyData.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if this is our key
		keyVal, keyExists := keyMap["key"]
		if !keyExists {
			continue
		}

		keyStr, ok := keyVal.(string)
		if !ok || keyStr != key.AccessKey {
			continue
		}

		// Found our key, check status
		statusVal, statusExists := keyMap["status"]
		if !statusExists {
			return "status field missing"
		}

		statusStr, ok := statusVal.(string)
		if !ok {
			return "invalid status format"
		}

		// Return the actual status
		switch statusStr {
		case "enabled":
			return "enabled"
		case "disabled":
			return "disabled"
		default:
			return fmt.Sprintf("unknown: %s", statusStr)
		}
	}

	// Key not found in user's access_keys
	return "key not found for user"
}

// Details implements the DetailsWidget interface - called when pressing 'd' to view details
func (w *UserKeysFromLocalDb) Details(selectedRowData common.RowData) (tea.Cmd, error) {
	if selectedRowData.Len() == 0 {
		return func() tea.Msg {
			return msg_types.DetailsContentMsg{
				Content:      "No user key selected",
				ResourceType: w.resourceType,
				Error:        fmt.Errorf("no user key selected"),
			}
		}, nil
	}

	userKeyId, err := selectedRowData.GetIntID()
	if err != nil {
		return func() tea.Msg {
			return msg_types.DetailsContentMsg{
				Content:      "Invalid user key ID",
				ResourceType: w.resourceType,
				Error:        fmt.Errorf("failed to get user key ID: %w", err),
			}
		}, nil
	}

	// Return async command that will load details from database
	return func() tea.Msg {
		// Fetch the complete UserKey record from database including secret key
		userKey, err := w.db.GetUserKey(uint(userKeyId))
		if err != nil {
			return msg_types.DetailsContentMsg{
				Content:      fmt.Sprintf("Failed to fetch user key details: %v", err),
				ResourceType: w.resourceType,
				Error:        err,
			}
		}

		// Create a map with only UserKey fields, excluding the Profile relationship entirely
		var userUID any = nil
		if userKey.UserUID != nil {
			userUID = *userKey.UserUID
		}

		userKeyData := map[string]any{
			"created_at": userKey.CreatedAt,
			"user_id":    userKey.UserID,
			"username":   userKey.Username,
			"user_uid":   userUID,
			"non_local":  userKey.NonLocal,
			"access_key": userKey.AccessKey,
			"secret_key": userKey.SecretKey,
		}

		// Set the details data and return success message
		w.DetailsAdapter.SetContent(userKeyData)
		return msg_types.DetailsContentMsg{
			Content:      userKeyData, // The DetailsAdapter will format this as JSON
			ResourceType: w.resourceType,
			Error:        nil,
		}
	}, nil
}

// Delete implements the DeleteWidget interface - called when pressing Ctrl+d to delete a user key
func (w *UserKeysFromLocalDb) Delete(selectedRowData common.RowData) (tea.Cmd, error) {
	if selectedRowData.Len() == 0 {
		return nil, fmt.Errorf("no row data provided for deletion")
	}

	// Extract ID from the row data
	userKeyId, err := selectedRowData.GetIntID()
	if err != nil {
		return nil, fmt.Errorf("failed to get user key ID from selected row: %w", err)
	}

	// Return async command that will delete the user key from database
	return func() tea.Msg {
		// Delete the user key from database
		err := w.db.DeleteUserKey(uint(userKeyId))
		if err != nil {
			return msg_types.ErrorMsg{
				Err: fmt.Errorf("failed to delete user key: %w", err),
			}
		}

		w.SetListData()
		w.SetModeMust(common.NavigatorModeList)
		return nil
	}, nil
}
