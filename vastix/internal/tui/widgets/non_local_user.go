package widgets

import (
	"fmt"
	"net/http"
	"vastix/internal/database"
	"vastix/internal/msg_types"
	"vastix/internal/tui/widgets/common"

	tea "github.com/charmbracelet/bubbletea"
)

type NonLocalUser struct {
	*BaseWidget
}

func NewNonLocalUser(db *database.Service) common.Widget {
	resourceType := "nonlocal users"
	listHeaders := []string{"action"}

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(
			http.MethodPost, "users/non_local_keys",
			http.MethodGet, "users/non_local_keys",
		),
	}

	extraNav := []common.ExtraWidget{
		NewGetNonLocalUser(db),
		NewGGenerateNonLocalAccessKey(db),
	}

	keyRestrictions := &common.NavigatorKeyRestrictions{
		Main: common.KeyRestrictions{
			NotAllowedListKeys: []string{"x"}, // Block extra actions in main list mode
		},
		Extra: common.NewDefaultKeyRestrictions(), // No restrictions for extra widgets
	}

	widget := &NonLocalUser{
		BaseWidget: NewBaseWidget(db, listHeaders, formHints, resourceType, extraNav, keyRestrictions),
	}
	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (NonLocalUser) API(rest *VMSRest) VastResourceAPI {
	// NonLocalUsers is not a separate resource in the untyped API
	// It's handled through extra methods on the Users resource
	return rest.Users
}

func (w *NonLocalUser) SetListData() tea.Msg {
	data := [][]string{
		{"get_nonlocal_user"},
		{"generate_nonlocal_user_key"},
	}

	w.ListAdapter.SetListData(data, w.GetFuzzyListSearchString())
	return nil
}

func (w *NonLocalUser) Select(rowData common.RowData) (tea.Cmd, error) {
	w.SetModeMust(common.NavigatorModeExtra)
	return w.BaseWidget.Select(rowData)

}

// -------------------------------
// Get Non Local User
// -------------------------------

type GetNonLocalUser struct {
	*BaseWidget
}

func NewGetNonLocalUser(db *database.Service) *GetNonLocalUser {
	resourceType := "get_nonlocal_user"

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPatch, "users/query", http.MethodGet, "users/query"),
	}

	widget := &GetNonLocalUser{
		BaseWidget: NewBaseWidget(db, nil, formHints, resourceType, nil, nil),
	}

	// Set custom title for the details view
	widget.DetailsAdapter.SetPredefinedTitle("nonlocal user content")
	widget.CreateAdapter.SetPredefinedTitle("get nonlocal user")
	return widget
}

func (*GetNonLocalUser) InitialExtraMode() common.ExtraNavigatorMode {
	return common.ExtraNavigatorModeCreate
}

func (w *GetNonLocalUser) ViewCreateForm() string {
	return w.viewCreateForm()
}

func (w *GetNonLocalUser) GetInputs() (common.Inputs, error) {
	inputs, err := w.formHints.GetInputsFromQueryParamsWithCustom(true)
	if err != nil {
		return nil, err
	}
	return inputs, nil
}

func (w *GetNonLocalUser) CreateFromInputs(inputs common.Inputs) (tea.Cmd, error) {
	rest, err := getActiveRest(w.db)
	if err != nil {
		return nil, err
	}
	data := inputs.ToParams()

	w.SetExtraMode(common.ExtraNavigatorModeDetails)

	return func() tea.Msg {
		// Query non-local users via the UserQuery extra method
		record, err := rest.Users.UserQuery_GET(data)
		if err != nil {
			w.DetailsAdapter.SetContent(nil)
			return msg_types.ErrorMsg{
				Err: err,
			}
		}

		w.DetailsAdapter.SetContent(record)

		return nil
	}, nil
}

func (w *GetNonLocalUser) ViewDetails() string {
	return w.viewDetails()
}

// -------------------------------
// Non Local User Keys
// -------------------------------

type GenerateNonLocalAccessKey struct {
	*BaseWidget
}

func NewGGenerateNonLocalAccessKey(db *database.Service) *GenerateNonLocalAccessKey {
	resourceType := "generate_nonlocal_user_key"

	formHints := &common.FormHints{
		SchemaRef: common.NewSchemaReference(http.MethodPost, "users/non_local_keys", "", ""),
	}

	widget := &GenerateNonLocalAccessKey{
		BaseWidget: NewBaseWidget(db, nil, formHints, resourceType, nil, nil),
	}
	// Parent will be set by the main widget via SetParentForExtraWidgets()

	// Set custom title for the details view
	widget.DetailsAdapter.SetPredefinedTitle("nonlocal user-key content")
	widget.CreateAdapter.SetPredefinedTitle("create nonlocal user-key")

	return widget
}

func (*GenerateNonLocalAccessKey) InitialExtraMode() common.ExtraNavigatorMode {
	return common.ExtraNavigatorModeCreate
}

func (w *GenerateNonLocalAccessKey) ViewCreateForm() string {
	return w.viewCreateForm()
}

func (w *GenerateNonLocalAccessKey) GetInputs() (common.Inputs, error) {
	inputs, err := w.formHints.GetInputsFromCreateSchemaWithCustom(true)
	if err != nil {
		return nil, err
	}
	return inputs, nil
}

func (w *GenerateNonLocalAccessKey) CreateFromInputs(inputs common.Inputs) (tea.Cmd, error) {
	data := inputs.ToParams()

	rest, err := getActiveRest(w.db)
	if err != nil {
		return nil, err
	}

	w.SetExtraMode(common.ExtraNavigatorModeDetails)

	return func() tea.Msg {
		// Query non-local user first to retrieve uid and name
		userRecord, err := rest.Users.UserQuery_GET(data)
		if err != nil {
			// Clear the details adapter content so it shows "No content"
			w.DetailsAdapter.SetContent(nil)
			return msg_types.ErrorMsg{
				Err: fmt.Errorf("failed to query non-local user: %w", err),
			}
		}
		uid := common.ToIntMust(userRecord["uid"])
		name := userRecord.RecordName()

		// Create non-local user key via extra method
		record, err := rest.Users.UserNonLocalKeys_POST(data)
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
			_, err = w.db.CreateNonLocalUserKey(activeProfile.ID, name, uid, accessKey, secretKey)
			if err != nil {
				// Clear the details adapter content so it shows "No content"
				w.DetailsAdapter.SetContent(nil)
				return msg_types.ErrorMsg{
					Err: fmt.Errorf("failed to store user key for user %d in database: %w", uid, err),
				}
			} else {
				w.auxlog.Printf("Successfully stored user key for user %d in database", uid)
			}
		}

		// Set the record for display in details view
		w.DetailsAdapter.SetContent(record)

		return nil
	}, nil
}
func (w *GenerateNonLocalAccessKey) ViewDetails() string {
	return w.viewDetails()
}
