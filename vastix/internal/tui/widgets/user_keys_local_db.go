package widgets

import (
	"context"
	"fmt"
	"strconv"
	"vastix/internal/database"
	"vastix/internal/msg_types"
	"vastix/internal/tui/widgets/common"

	tea "github.com/charmbracelet/bubbletea"
)

type UserKeysFromLocalDb struct {
	*BaseWidget
}

// NewUserKeysFromLocalDb creates a new user keys widget for local database records
func NewUserKeysFromLocalDb(db *database.Service, msgChan chan tea.Msg) common.Widget {
	resourceType := "user_keys [local database]"
	listHeaders := []string{
		"id", "user_id", "username", "non_local", "access_key",
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
func (w *UserKeysFromLocalDb) API(rest *VMSRest) VastResourceAPIWithContext {
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

// SetListDataWithContext overrides BaseWidget - context not needed for local DB access
func (w *UserKeysFromLocalDb) SetListDataWithContext(_ context.Context) tea.Msg {
	return w.SetListData()
}

// SetListData retrieves all UserKey records from the local database
func (w *UserKeysFromLocalDb) SetListData() tea.Msg {
	userKeys, err := w.db.GetUserKeysForActiveProfile()
	if err != nil {
		return msg_types.ErrorMsg{
			Err: fmt.Errorf("failed to retrieve user keys from database: %w", err),
		}
	}

	// Convert UserKey records to string data for display
	// Note: Secret key is NOT included in the list view for security
	// Additional fields (profile_id, user_uid, timestamps) are shown only in details view
	userKeyData := make([][]string, len(userKeys))
	for i, key := range userKeys {
		userKeyData[i] = []string{
			strconv.FormatUint(uint64(key.ID), 10),
			strconv.FormatInt(key.UserID, 10),
			key.Username,
			strconv.FormatBool(key.NonLocal),
			key.AccessKey,
		}
	}

	// Set the data in the list adapter
	w.ListAdapter.SetListData(userKeyData, w.GetFuzzyListSearchString())
	return msg_types.MockMsg{}
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
