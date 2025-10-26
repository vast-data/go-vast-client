package widgets

import (
	"fmt"
	"net/http"
	"time"
	"vastix/internal/database"
	"vastix/internal/msg_types"
	"vastix/internal/tui/widgets/common"

	tea "github.com/charmbracelet/bubbletea"
	vast_client "github.com/vast-data/go-vast-client"
)

type ApiToken struct {
	*BaseWidget
}

func NewApiToken(db *database.Service) common.Widget {
	resourceType := "apitokens"
	listHeaders := []string{"id", "name", "owner", "expiry_date", "last_used", "revoked"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{}

	widget := &ApiToken{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (ApiToken) API(rest *VMSRest) VastResourceAPI {
	return rest.ApiTokens
}

func (w *ApiToken) GetInputs() (common.Inputs, error) {
	inputs := w.getInputs()
	rest, err := getActiveRest(w.db)
	if err != nil {
		return inputs, err
	}
	managers, err := rest.Managers.List(nil)
	if err != nil {
		return inputs, err
	}
	usernames := make([]string, 0)
	for _, manager := range managers {
		usernames = append(usernames, manager["username"].(string))
	}
	ownerField := inputs.MustField("owner")
	ownerField.SetSuggestions(usernames)
	return inputs, nil
}

func (w *ApiToken) AfterCreate(record vast_client.Record) (tea.Msg, error) {
	rest, err := getActiveRest(w.db)
	if err != nil {
		return nil, err
	}

	// Get active profile
	activeProfile, err := w.db.GetActiveProfile()
	if err != nil {
		return nil, err
	}

	// Extract ID from initial record to fetch complete details
	tokenID, ok := record["id"].(string)
	if !ok || tokenID == "" {
		return nil, fmt.Errorf("missing or invalid id field in record")
	}

	// Extract token string from initial record
	token, ok := record["token"].(string)
	if !ok || token == "" {
		return nil, fmt.Errorf("missing or invalid token field in record")
	}

	// Get complete token details by ID
	fullTokenRecord, err := rest.ApiTokens.GetById(tokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch complete token details: %w", err)
	}

	// Extract data from complete record with nil checks
	owner, ok := fullTokenRecord["owner"].(string)
	if !ok || owner == "" {
		return nil, fmt.Errorf("missing or invalid owner field in complete record")
	}

	name, ok := fullTokenRecord["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("missing or invalid name field in complete record")
	}

	createdStr, ok := fullTokenRecord["created"].(string)
	if !ok || createdStr == "" {
		return nil, fmt.Errorf("missing or invalid created field in complete record")
	}

	// Get owner details from managers
	ownerRecord, err := rest.Managers.Get(params{"username": owner})
	if err != nil {
		return nil, err
	}
	ownerID := ownerRecord.RecordID()

	// Parse created timestamp
	vastCreated, err := time.Parse(time.RFC3339, createdStr)
	if err != nil {
		return nil, err
	}

	// Parse expiry date (can be null)
	var expireDate *time.Time
	if expiryVal, exists := fullTokenRecord["expiry_date"]; exists && expiryVal != nil {
		if expiryStr, ok := expiryVal.(string); ok && expiryStr != "" {
			if parsed, err := time.Parse(time.RFC3339, expiryStr); err == nil {
				expireDate = &parsed
			}
		}
	}

	// Create local database record
	apiToken := &database.ApiToken{
		ProfileID:   activeProfile.ID,
		TokenID:     tokenID,
		Token:       token, // Use the actual token string from initial record
		Name:        name,
		Owner:       owner,
		OwnerID:     uint(ownerID),
		ExpireDate:  expireDate,
		VastCreated: vastCreated,
	}

	// Save to database
	if err := w.db.CreateApiToken(apiToken); err != nil {
		return nil, err
	}

	// Set content and switch to details mode
	w.SetContent(record)
	w.SetMode(common.NavigatorModeDetails)
	return msg_types.SetDataMsg{}, nil
}

// Delete implements the DeleteWidget interface - called when pressing Ctrl+d to delete an API token
func (w *ApiToken) Delete(selectedRowData common.RowData) (tea.Cmd, error) {
	if selectedRowData.Len() == 0 {
		return nil, fmt.Errorf("no row data provided for deletion")
	}

	rest, err := getActiveRest(w.db)
	if err != nil {
		return nil, err
	}

	// Extract ID from the row data
	tokenId := selectedRowData.GetString("id")
	// Return async command that will delete the API token from database
	return func() tea.Msg {
		// Delete the API token from database
		if _, err := rest.ApiTokens.ApiTokenRevoke_PATCH(tokenId, nil); err != nil {
			return msg_types.ErrorMsg{
				Err: fmt.Errorf("failed to revoke API token: %w", err),
			}
		}

		w.SetListData()
		w.SetModeMust(common.NavigatorModeList)
		return nil
	}, nil
}
