package widgets

import (
	"fmt"
	"net/http"
	"vastix/internal/database"
	"vastix/internal/msg_types"
	"vastix/internal/tui/widgets/common"

	tea "github.com/charmbracelet/bubbletea"
)

type User struct {
	*BaseWidget
}

func NewUser(db *database.Service) common.Widget {
	resourceType := "users"
	listHeaders := []string{"id", "name", "uid", "sid"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, resourceType, "", ""),
	}

	extraNav := []common.ExtraWidget{
		NewGenerateAccessKey(db),
	}

	widget := &User{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, nil),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (User) API(rest *VMSRest) VastResourceAPI {
	return rest.Users
}

// -------------------------------
// User Keys
// -------------------------------

type GenerateAccessKey struct {
	*BaseWidget
	// Note: Mode is managed by the embedded BaseWidget.ExtraWidgetNavigator
}

func NewGenerateAccessKey(db *database.Service) *GenerateAccessKey {
	resourceType := "generate_access_key"
	widget := &GenerateAccessKey{
		BaseWidget: NewBaseWidget(db, nil, nil, resourceType, nil, nil),
	}
	// Parent will be set by the main widget via SetParentForExtraWidgets()

	// Set custom title for the details view
	widget.DetailsAdapter.SetPredefinedTitle("user-key content")

	return widget
}

func (*GenerateAccessKey) ShortCut() *common.KeyBinding {
	return &common.KeyBinding{
		Key:  "<ctrl+g>",
		Desc: "generate access key",
	}
}

func (*GenerateAccessKey) InitialExtraMode() common.ExtraNavigatorMode {
	return common.ExtraNavigatorModePrompt
}

func (w *GenerateAccessKey) ViewPrompt() string {
	// Get the selected row data to extract username
	selectedRowData := w.selectedRowData

	// Extract username from selected row data using strict case-insensitive lookup
	// This will panic if "name" (or "NAME") key is not found, which is what we want
	username := selectedRowData.GetStringMust("name")

	// Create the prompt message
	promptMsg := "Do you want to create new access key for user \"" + username + "\"?"
	promptTitle := "generate access key"

	// Use the prompt adapter to render the prompt
	width := w.GetWidth()
	height := w.GetHeight()

	return w.PromptAdapter.PromptDo(promptMsg, promptTitle, width, height)
}

func (w *GenerateAccessKey) CreateFromInputs(_ common.Inputs) (tea.Cmd, error) {
	rest, err := getActiveRest(w.db)
	if err != nil {
		return nil, err
	}
	userId, err := w.selectedRowData.GetIntID()
	name := w.selectedRowData.GetStringMust("name")
	uid := w.selectedRowData.GetInt64Must("uid")
	if err != nil {
		return nil, err
	}
	// Get tenant ID from row data, default to 0 if not present
	tenantId := w.selectedRowData.GetInt64("tenant_id")

	return func() tea.Msg {
		// Create user access key via extra method
		record, err := rest.Users.UserAccessKeys_POST(userId, tenantId)
		if err != nil {
			// Clear the details adapter content so it shows "No content"
			w.DetailsAdapter.SetContent(nil)
			return msg_types.ErrorMsg{
				Err: err,
			}
		}

		// Get the active profile to store the key relationship
		activeProfile, err := w.db.GetActiveProfile()
		if err != nil {
			// Clear the details adapter content so it shows "No content"
			w.DetailsAdapter.SetContent(nil)
			return msg_types.ErrorMsg{
				Err: fmt.Errorf("failed to get active profile: %w", err),
			}
		}
		if activeProfile == nil {
			// Clear the details adapter content so it shows "No content"
			w.DetailsAdapter.SetContent(nil)
			return msg_types.ErrorMsg{
				Err: fmt.Errorf("no active profile found"),
			}
		}

		var accessKey, secretKey string
		if ak, exists := record["access_key"]; exists {
			accessKey = fmt.Sprintf("%v", ak)
		}
		if sk, exists := record["secret_key"]; exists {
			secretKey = fmt.Sprintf("%v", sk)
		}

		// Store the keys in the database (for local users, non_local = false)
		if accessKey != "" && secretKey != "" {
			_, err = w.db.CreateLocalUserKey(activeProfile.ID, userId, name, uid, accessKey, secretKey)
			if err != nil {
				// Clear the details adapter content so it shows "No content"
				w.DetailsAdapter.SetContent(nil)
				return msg_types.ErrorMsg{
					Err: fmt.Errorf("failed to store user key for user %d in database: %w", userId, err),
				}
			} else {
				w.auxlog.Printf("Successfully stored user key for user %d in database", userId)
			}
		}

		// Set the record for display in details view
		w.DetailsAdapter.SetContent(record)

		return nil
	}, nil
}

func (w *GenerateAccessKey) ViewDetails() string {
	return w.viewDetails()
}
