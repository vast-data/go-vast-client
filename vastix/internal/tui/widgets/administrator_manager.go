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

type AdministratorManager struct {
	*BaseWidget
}

func NewAdministratorManager(db *database.Service) common.Widget {
	resourceType := "managers"
	listHeaders := []string{"id", "username", "is_active", "tenant"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{
		NewGenerateToken(db),
	}

	keyRestrictions := &common.NavigatorKeyRestrictions{
		Main: common.KeyRestrictions{
			NotAllowedListKeys: []string{"x"}, // Block extra actions in main list mode
		},
		Extra: common.NewDefaultKeyRestrictions(), // No restrictions for extra widgets
	}

	widget := &AdministratorManager{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, keyRestrictions),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (AdministratorManager) API(rest *VMSRest) VastResourceAPI {
	return rest.Managers
}

// -------------------------------
// GenerateToken widget - extra navigator for AdministratorManager
// -------------------------------

type GenerateToken struct {
	*BaseWidget
}

func NewGenerateToken(db *database.Service) *GenerateToken {
	resourceType := "generate_token"

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, "apitokens", "", ""),
	}

	widget := &GenerateToken{
		BaseWidget: NewBaseWidget(db, nil, formHints, resourceType, nil, nil),
	}

	widget.DetailsAdapter.SetPredefinedTitle("api-token content")
	widget.CreateAdapter.SetPredefinedTitle("generate api-token")

	return widget
}

func (*GenerateToken) ShortCut() *common.KeyBinding {
	return &common.KeyBinding{
		Key:  "<ctrl+g>",
		Desc: "generate token",
	}
}

func (*GenerateToken) InitialExtraMode() common.ExtraNavigatorMode {
	return common.ExtraNavigatorModeCreate
}

func (w *GenerateToken) CreateFromInputs(inputs common.Inputs) (tea.Cmd, error) {
	rest, err := getActiveRest(w.db)
	if err != nil {
		return nil, err
	}

	userName := w.selectedRowData.GetString("username")
	w.SetExtraMode(common.ExtraNavigatorModeDetails)

	return func() tea.Msg {
		// Convert inputs to params
		params := inputs.ToParams()

		// Add the owner field from the selected manager
		params["owner"] = userName

		// Create the API token via REST API
		record, err := rest.ApiTokens.Create(params)
		if err != nil {
			// Clear the details adapter content so it shows "No content"
			w.DetailsAdapter.SetContent(nil)
			return msg_types.ErrorMsg{
				Err: err,
			}
		}

		// Store token to database (same as ApiToken.AfterCreate)
		if err := w.storeTokenToDatabase(record, rest); err != nil {
			w.auxlog.Printf("Failed to store token to database: %v", err)
			// Don't return error here - token was created successfully, just storage failed
		}

		// Set the record for display in details view
		w.DetailsAdapter.SetContent(record)

		return nil
	}, nil
}

func (w *GenerateToken) GetInputs() (common.Inputs, error) {
	inputs := w.getInputs()
	inputs.RemoveField("owner")

	userName := w.selectedRowData.GetString("username")
	w.CreateAdapter.SetPredefinedTitle(fmt.Sprintf("generate api-token for %q", userName))

	return inputs, nil
}

func (w *GenerateToken) ViewDetails() string {
	return w.viewDetails()
}

// storeTokenToDatabase stores the created token to the local database
// following the same pattern as ApiToken.AfterCreate
func (w *GenerateToken) storeTokenToDatabase(record vast_client.Record, rest *VMSRest) error {
	// Get active profile
	activeProfile, err := w.db.GetActiveProfile()
	if err != nil {
		return fmt.Errorf("failed to get active profile: %w", err)
	}

	// Extract ID from initial record to fetch complete details
	tokenID, ok := record["id"].(string)
	if !ok || tokenID == "" {
		return fmt.Errorf("missing or invalid id field in record")
	}

	// Extract token string from initial record
	token, ok := record["token"].(string)
	if !ok || token == "" {
		return fmt.Errorf("missing or invalid token field in record")
	}

	// Get complete token details by ID
	fullTokenRecord, err := rest.ApiTokens.GetById(tokenID)
	if err != nil {
		return fmt.Errorf("failed to fetch complete token details: %w", err)
	}

	// Extract data from complete record with nil checks
	owner, ok := fullTokenRecord["owner"].(string)
	if !ok || owner == "" {
		return fmt.Errorf("missing or invalid owner field in complete record")
	}

	name, ok := fullTokenRecord["name"].(string)
	if !ok || name == "" {
		return fmt.Errorf("missing or invalid name field in complete record")
	}

	createdStr, ok := fullTokenRecord["created"].(string)
	if !ok || createdStr == "" {
		return fmt.Errorf("missing or invalid created field in complete record")
	}

	// Get owner details from managers
	ownerRecord, err := rest.Managers.Get(params{"username": owner})
	if err != nil {
		return fmt.Errorf("failed to get manager details: %w", err)
	}
	ownerID := ownerRecord.RecordID()

	// Parse created timestamp
	vastCreated, err := time.Parse(time.RFC3339, createdStr)
	if err != nil {
		return fmt.Errorf("failed to parse created timestamp: %w", err)
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
		return fmt.Errorf("failed to save token to database: %w", err)
	}

	w.auxlog.Printf("Successfully stored API token %s to database for owner %s", tokenID, owner)
	return nil
}

func (w *GenerateToken) ViewCreateForm() string {
	return w.viewCreateForm()
}
