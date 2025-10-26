package common

import (
	tea "github.com/charmbracelet/bubbletea"
	vast_client "github.com/vast-data/go-vast-client"
)

type Widget interface {
	Navigator                                     // Implements the Navigator interface for handling navigation modes
	Dimensional                                   // Implements the Dimensional interface for size management
	KeyBindingGetter                              // Implements the KeyBindingGetter interface for key bindings
	ListWidget                                    // Implements the ListWidget interface for list views
	CreateWidget                                  // Implements the CreateWidget interface for creation forms
	DetailsWidget                                 // Implements the DetailsWidget interface for detailed views
	DeleteWidget                                  // Implements the DeleteWidget interface for deletion actions
	PromptWidget                                  // Implements the PromptWidget interface for prompting user input
	SearchableWidget                              // Implements the SearchableWidget interface for search functionality
	GetName() string                              // Returns the name of the widget, typically used for display purposes
	Init() tea.Msg                                // Initializes the widget, typically called once at startup
	Reset()                                       // Resets the widget to its initial state
	GetAllowedNavigatorModes() []NavigatorMode    // Returns a list of allowed navigator modes for this widget
	GetNotAllowedNavigatorModes() []NavigatorMode // Returns a list of not allowed navigator modes for this widget
	GetResourceType() string                      // Returns the type of resource this widget represents
	View() string                                 // Renders the main view for the widget, typically used for displaying lists or forms
	ShortCuts() map[string]ExtraWidget            // Returns a map of shortcut keys to extra widgets for quick access
}

// ------------------------
// Navigator interface for handling different modes of navigation
// -------------------------

type Navigator interface {
	InitialMode() NavigatorMode // Returns the initial mode of the navigator
	SetMode(NavigatorMode)
	GetMode() NavigatorMode
	GetWidget() Widget
	SetWidget(Widget)
	Navigate(msg tea.Msg) tea.Cmd
}

type ExtraWidget interface {
	ExtraNavigator                                       // Implements the ExtraNavigator interface for handling navigation modes
	Dimensional                                          // Implements the Dimensional interface for size management
	KeyBindingGetter                                     // Implements the KeyBindingGetter interface for key bindings
	ListWidget                                           // Implements the ListWidget interface for list views
	CreateWidget                                         // Implements the CreateWidget interface for creation forms
	DetailsWidget                                        // Implements the DetailsWidget interface for detailed views
	DeleteWidget                                         // Implements the DeleteWidget interface for deletion actions
	PromptWidget                                         // Implements the PromptWidget interface for prompting user input
	SearchableWidget                                     // Implements the SearchableWidget interface for search functionality
	GetName() string                                     // Returns the name of the extra widget, typically used for display purposes
	Init() tea.Msg                                       // Initializes the widget, typically called once at startup
	Reset()                                              // Resets the widget to its initial state
	GetAllowedExtraNavigatorModes() []ExtraNavigatorMode // Returns a list of allowed extra navigator modes for this widget
	GetExtraResourceType() string                        // Returns the type of resource this widget represents
	ViewExtra() string                                   // Renders the extra view for the widget, typically used for additional actions or information
	CanUseExtra() bool                                   // Checks if the extra widget is enabled, allowing it to be used in the UI
	ShortCut() *KeyBinding                               // Returns the key binding for quick access to this extra widget
}

// ------------------------
// ExtraNavigator interface for handling extra navigation modes
// -------------------------

type ExtraNavigator interface {
	InitialExtraMode() ExtraNavigatorMode // Returns the initial mode of the extra navigator
	SetExtraMode(ExtraNavigatorMode)
	GetExtraMode() ExtraNavigatorMode
	GetExtraWidget() ExtraWidget
	SetExtraWidget(ExtraWidget)
	ExtraNavigate(msg tea.Msg) (tea.Cmd, bool)
}

// ------------------------
// Dimensional interface for managing widget dimensions
// -------------------------

type Dimensional interface {
	GetWidth() int
	GetHeight() int
	SetWidth(w int)
	SetHeight(h int)
	SetSize(w, h int)
}

// ------------------------
// KeyBindingGetter interface for widgets that support key bindings
// -------------------------

type KeyBindingGetter interface {
	GetListKeyBindings() []KeyBinding    // Returns a list of key bindings for the widget in list mode
	GetCreateKeyBindings() []KeyBinding  // Returns a list of key bindings for the widget in create mode
	GetDeleteKeyBindings() []KeyBinding  // Returns a list of key bindings for the widget in delete mode
	GetDetailsKeyBindings() []KeyBinding // Returns a list of key bindings for the widget in details mode
	GetKeyBindings() []KeyBinding        // Returns a list of key bindings for the widget (dynamically determined based on mode)
}

// ------------------------
// ListWidget interface for widgets that display lists
// -------------------------

type ListWidget interface {
	ViewList() string     // Renders the list view
	SetListData() tea.Msg // Sets the data for the list, typically used to refresh or update the view
	ClearListData()       // Clears the current list data, typically used to reset the view
}

type ListAdapter interface {
	MoveUp()
	MoveDown()
	MoveHome()
	MoveEnd()
	PageUp()
	PageDown()
}

// ------------------------
// CreateWidget interface for widgets that support form creation
// -------------------------

type CreateWidget interface {
	ViewCreateForm() string                   // Renders the form view for creating new entries
	GetInputs() (Inputs, error)               // Returns a list of input fields for the form
	HasInputs() bool                          // Checks if the form has any input fields defined
	SetInputs(Inputs)                         // Sets the input fields for the form, typically used to initialize or reset the form
	CreateFromInputs(Inputs) (tea.Cmd, error) // Creates a new entry from the provided inputs, returning a command and an error if any
	ResetCreateForm()                         // Resets the form to its initial state, clearing any input values
}

type FormNavigateAdaptor interface {
	NextInput() // Moves to the next input field in the form
	PrevInput() // Moves to the previous input field in the form
}

type CreateFromInputsAdapter interface {
	CreateFromInputsDo(CreateWidget) tea.Cmd // Creates a new entry from the provided inputs, returning a command
}

type UpdateInputAdapter interface {
	UpdateCurrentInput(msg tea.Msg) // Updates the currently focused input field based on the provided message
}

type FormJSONToggleAdapter interface {
	ToggleFormJSONMode() // Toggles between form and JSON mode
}

// ------------------------
// DetailsWidget interface for widgets that display detailed information
// -------------------------

type DetailsWidget interface {
	Details(RowData) (tea.Cmd, error) // Get details for the selected item, returning a command and an error if any
	ViewDetails() string              // Renders the details view for the selected item
	SetDetailsData(any)               // Sets the data for the details view, typically used to display information about a selected item
	GetSelectedRowData() RowData      // Returns the data of the currently selected row, typically used for actions like deletion or detail viewing
	SetSelectedRowData(RowData)       // Sets the currently selected row data, typically used to update the details view when a new item is selected
	Select(RowData) (tea.Cmd, error)  // Selects an item based on the provided row data, returning a command and an error if any
}

type DetailsAdapter interface {
	DetailsDo(DetailsWidget) tea.Cmd // Get details for the selected item, returning a command
	DetailsOnSelect() bool           // Determines if on select (pressing "enter") should trigger details view
}

// SelectAdapter interface for widgets that support selection
type SelectAdapter interface {
	SelectDo(widget DetailsWidget) tea.Cmd
}

type ViewPortAdapter interface {
	UpdateViewPort(tea.Msg) tea.Cmd
}

type CopyToClipboardAdapter interface {
	CopyToClipboard() tea.Msg // Copies the currently selected content to the clipboard
}

// ------------------------
// DeleteWidget interface for widgets that support deletion of items
// -------------------------

type DeleteWidget interface {
	Delete(RowData) (tea.Cmd, error) // Deletes the selected item based on the provided row data, returning a command and an error if any
}

type DeleteAdapter interface {
	DeleteDo(DeleteWidget) tea.Cmd // Deletes the selected item, returning a command
}

// ------------------------
// PromptWidget interface for widgets that support prompting the user for input
// -------------------------

type PromptWidget interface {
	ViewPrompt() string
}

type PromptAdapter interface {
	PromptDo(msg, title string, width, height int) string // Returns a prompt string
}

type SearchableWidget interface {
	IsServerSearchable() bool             // Returns true if the widget can currently search its data on the server
	IsFuzzySearchable() bool              // Returns true if the widget can currently apply fuzzy search to its local data
	SetFuzzyListSearchString(string)      // Sets a fuzzy search query for the widget
	GetFuzzyListSearchString() string     // Returns the current fuzzy search query
	SetFuzzyDetailsSearchString(string)   // Sets a fuzzy search query for the widget
	GetFuzzyDetailsSearchString() string  // Returns the current fuzzy search query
	SetServerSearchParams(string)         // Sets a server-side search query string for the widget
	GetServerSearchParams() string        // Returns the current server-side search parameters as string
	ClearFilters()                        // Clears any active filters, resetting the search state
	ClearFuzzyListSearch()                // Clears the local fuzzy search query for list filtering
	ClearFuzzyDetailsSearch()             // Clears the local fuzzy search query for details filtering
	ClearServerSearchParams()             // Clears the server-side search parameters string
	GetServerParams() *vast_client.Params // Returns the current server-side search parameters as vast_client.Params
}

// InputField Common interface for all input types
type InputField interface {
	Update(msg tea.Msg) tea.Cmd
	View() string
	Focus()
	Blur()
	Value() string
	SetValue(val string)
	Validate() error
}

// RenderRow interface for widgets that want to customize their row rendering
type RenderRow interface {
	RenderRow(rowData RowData, isSelected bool, colWidth int) []string
}

type CreateFromInputs interface {
	CreateFromInputs(Inputs) (tea.Cmd, error)
}

type KeyBinding struct {
	Key     string // Key combination (e.g., "ctrl+c")
	Desc    string // Description of the key binding
	Generic bool   // Whether this is a generic key binding (IOW move to another resource or adjusting search settings)
}

// ------------------------
// Key Restrictions for Navigator Configuration
// ------------------------

// KeyRestrictions defines which keys are not allowed for a specific navigator
type KeyRestrictions struct {
	NotAllowedListKeys    []string // Keys blocked in list mode
	NotAllowedCreateKeys  []string // Keys blocked in create mode
	NotAllowedDeleteKeys  []string // Keys blocked in delete mode
	NotAllowedDetailsKeys []string // Keys blocked in details mode
}

// NavigatorKeyRestrictions defines key restrictions for both main and extra navigators
type NavigatorKeyRestrictions struct {
	Main  KeyRestrictions // Key restrictions for the main widget navigator
	Extra KeyRestrictions // Key restrictions for the extra widget navigator
}

// Helper functions for common key restriction configurations

// NewDefaultKeyRestrictions creates a KeyRestrictions with no restrictions (all keys allowed)
func NewDefaultKeyRestrictions() KeyRestrictions {
	return KeyRestrictions{
		NotAllowedListKeys:    []string{},
		NotAllowedCreateKeys:  []string{},
		NotAllowedDeleteKeys:  []string{},
		NotAllowedDetailsKeys: []string{},
	}
}

// NewDefaultNavigatorKeyRestrictions creates a NavigatorKeyRestrictions with no restrictions for both main and extra
func NewDefaultNavigatorKeyRestrictions() NavigatorKeyRestrictions {
	return NavigatorKeyRestrictions{
		Main:  NewDefaultKeyRestrictions(),
		Extra: NewDefaultKeyRestrictions(),
	}
}

// NewNavigatorKeyRestrictions creates a NavigatorKeyRestrictions with specified restrictions
func NewNavigatorKeyRestrictions(main, extra KeyRestrictions) NavigatorKeyRestrictions {
	return NavigatorKeyRestrictions{
		Main:  main,
		Extra: extra,
	}
}

// NewKeyRestrictions creates a KeyRestrictions with specified restrictions for each mode
func NewKeyRestrictions(list, create, delete, details []string) KeyRestrictions {
	return KeyRestrictions{
		NotAllowedListKeys:    list,
		NotAllowedCreateKeys:  create,
		NotAllowedDeleteKeys:  delete,
		NotAllowedDetailsKeys: details,
	}
}

// Common preset configurations

// NewReadOnlyKeyRestrictions creates restrictions suitable for read-only widgets (blocks create/delete operations)
func NewReadOnlyKeyRestrictions() KeyRestrictions {
	return KeyRestrictions{
		NotAllowedListKeys:    []string{},
		NotAllowedCreateKeys:  []string{"n", "enter"}, // Block new and submit
		NotAllowedDeleteKeys:  []string{"y", "enter"}, // Block confirm delete
		NotAllowedDetailsKeys: []string{},
	}
}

// NewMinimalNavigationKeyRestrictions creates restrictions that block navigation keys but allow basic operations
func NewMinimalNavigationKeyRestrictions() KeyRestrictions {
	return KeyRestrictions{
		NotAllowedListKeys:    []string{":", "x"},   // Block resource switching and extra actions
		NotAllowedCreateKeys:  []string{"esc"},      // Block cancel in create mode
		NotAllowedDeleteKeys:  []string{"n", "esc"}, // Block cancel in delete mode
		NotAllowedDetailsKeys: []string{},
	}
}

// NewSearchOnlyKeyRestrictions creates restrictions that only allow search operations
func NewSearchOnlyKeyRestrictions() KeyRestrictions {
	return KeyRestrictions{
		NotAllowedListKeys:    []string{"enter", "d", "n", "ctrl+d", "x"}, // Block all except search
		NotAllowedCreateKeys:  []string{"tab", "shift+tab", "enter"},      // Block all create operations
		NotAllowedDeleteKeys:  []string{"y", "enter"},                     // Block delete confirm
		NotAllowedDetailsKeys: []string{"ctrl+s", "esc"},                  // Block clipboard and back
	}
}

// ------------------------
// VAST API
// -------------------------

type VastAPIGetter interface {
	API(rest *vast_client.VMSRest) vast_client.VastResourceAPI
}

type VastResourceGetter interface {
	Get(*vast_client.VMSRest) (vast_client.Record, error)
}

type VastResourceDetailsGetter interface {
	GetDetails(*vast_client.VMSRest, RowData) (vast_client.Record, error)
}

type VastResourceListGetter interface {
	List(*vast_client.VMSRest) (vast_client.RecordSet, error)
}

type VastResourceCreator interface {
	Create(*vast_client.VMSRest) (tea.Msg, error)
}

type BeforeVastResourceCreator interface {
	BeforeCreate(params vast_client.Params) error
}

type AfterVastResourceCreator interface {
	AfterCreate(record vast_client.Record) (tea.Msg, error)
}

type VastResourceDeleter interface {
	Delete(*vast_client.VMSRest, RowData) (tea.Msg, error)
}
