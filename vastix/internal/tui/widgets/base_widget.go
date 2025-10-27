package widgets

import (
	"fmt"
	"log"
	"strings"
	"vastix/internal/database"
	"vastix/internal/logging"
	"vastix/internal/msg_types"
	"vastix/internal/tui/widgets/adapters"
	"vastix/internal/tui/widgets/common"

	tea "github.com/charmbracelet/bubbletea"
	vast_client "github.com/vast-data/go-vast-client"
	"go.uber.org/zap"
)

type (
	params          = vast_client.Params
	VMSRest         = vast_client.VMSRest
	VastResourceAPI = vast_client.VastResourceAPI
	RecordSet       = vast_client.RecordSet
	Record          = vast_client.Record
)

// BaseWidget common properties and methods for all widgets
type BaseWidget struct {
	resourceType string // Type of resource this widget represents, e.g., "views", "quotas", "users" etc.

	// cache
	cachedInputs common.Inputs // Cached inputs for create form, generated from OpenAPI schema

	// Key bindings cache
	cachedListKeyBindings    []common.KeyBinding
	cachedCreateKeyBindings  []common.KeyBinding
	cachedDeleteKeyBindings  []common.KeyBinding
	cachedDetailsKeyBindings []common.KeyBinding

	*common.WidgetNavigator
	// ExtraNavigator for additional navigation capabilities
	*common.ExtraWidgetNavigator

	db *database.Service

	// Action Adapters
	*adapters.DimensionAdapter
	*adapters.ListAdapter
	*adapters.CreateAdapter
	*adapters.DetailsAdapter
	*adapters.PromptAdapter
	*adapters.SearchAdapter

	// Parent widget
	parent  common.Widget
	isExtra bool // Indicates if this is an extra widget

	// FormHints
	formHints *common.FormHints

	log    *zap.Logger
	auxlog *log.Logger

	// Selected row data
	selectedRowData common.RowData // Currently selected row data of parent widget
}

// NewBaseWidget creates a new base widget
//
// Example usage with custom key restrictions:
//
//	keyRestrictions := &common.NavigatorKeyRestrictions{
//		Main:  common.NewReadOnlyKeyRestrictions(),        // Block create/delete for main widget
//		Extra: common.NewSearchOnlyKeyRestrictions(),      // Only allow search for extra widgets
//	}
//	widget := NewBaseWidget(db, headers, hints, "users", extraWidgets, keyRestrictions)
//
// Or use preset configurations:
//
//	readOnlyConfig := &common.NavigatorKeyRestrictions{
//		Main:  common.NewReadOnlyKeyRestrictions(),
//		Extra: common.NewDefaultKeyRestrictions(),
//	}
//
// Or pass nil for default behavior (all keys allowed):
//
//	widget := NewBaseWidget(db, headers, hints, "users", extraWidgets, nil)
func NewBaseWidget(db *database.Service, listHeaders []string, formHints *common.FormHints, resourceType string, extraWidgets []common.ExtraWidget, keyRestrictions *common.NavigatorKeyRestrictions) *BaseWidget {
	log := logging.GetGlobalLogger()
	auxlog := logging.GetAuxLogger()

	log.Debug("Initializing resource", zap.String("resourceType", resourceType))
	defer log.Debug("Resource initialized", zap.String("resourceType", resourceType))

	// Use default key restrictions if none provided
	if keyRestrictions == nil {
		defaultRestrictions := common.NewDefaultNavigatorKeyRestrictions()
		keyRestrictions = &defaultRestrictions
	}

	widgetNavigator := common.NewWidgetNavigator(
		keyRestrictions.Main.NotAllowedListKeys,
		keyRestrictions.Main.NotAllowedCreateKeys,
		keyRestrictions.Main.NotAllowedDeleteKeys,
		keyRestrictions.Main.NotAllowedDetailsKeys,
	)
	extraWidgetGroup := NewExtraWidgetGroup(db, widgetNavigator, extraWidgets...)
	baseWidget := &BaseWidget{
		resourceType: resourceType,

		// Navigation
		WidgetNavigator: widgetNavigator,
		ExtraWidgetNavigator: common.NewExtraWidgetNavigator(
			extraWidgetGroup,
			widgetNavigator,
			keyRestrictions.Extra.NotAllowedListKeys,
			keyRestrictions.Extra.NotAllowedCreateKeys,
			keyRestrictions.Extra.NotAllowedDeleteKeys,
			keyRestrictions.Extra.NotAllowedDetailsKeys,
		),

		// Adapters
		ListAdapter:      adapters.NewListAdapter(db, resourceType, listHeaders),
		CreateAdapter:    adapters.NewCreateAdapter(db, resourceType),
		DetailsAdapter:   adapters.NewDetailsAdapter(db, resourceType),
		DimensionAdapter: &adapters.DimensionAdapter{},
		PromptAdapter:    adapters.NewPromptAdapter(db, resourceType),
		SearchAdapter:    adapters.NewSearchAdapter(db, resourceType),

		// Database
		db: db,

		// CreationForm Hints
		formHints: formHints,

		// Loggers
		log:    log,
		auxlog: auxlog,
	}

	log.Debug("Resource initialized", zap.String("resourceType", resourceType))
	return baseWidget
}

func (bw *BaseWidget) Init() tea.Msg {
	return nil
}

func (bw *BaseWidget) GetName() string {
	return fmt.Sprintf("%T[%s]", bw.parent, bw.resourceType)
}

// TraceLog logs the widget hierarchy and modes for debugging
func (bw *BaseWidget) TraceLog() {
	widgetName := bw.GetName()

	if bw.isExtra {
		// This is an extra widget
		bw.auxlog.Printf("EXTRA WIDGET: [%s]", widgetName)
		return
	}

	// This is a main widget
	widgetMode := bw.GetMode()

	if widgetMode == common.NavigatorModeExtra {
		if extraWidget, ok := bw.parent.(common.ExtraNavigator); ok {
			childGroup := extraWidget.GetExtraWidget()
			childGroupMode := childGroup.GetExtraMode()
			childGroupName := fmt.Sprintf("%T", childGroup)

			// Try to get active extra widget name if possible
			activeExtraWidgetName := "none"
			if extraGroup, ok := childGroup.(*ExtraWidgetGroup); ok {
				if extraGroup.activeExtraWidget != "" {
					activeExtraWidgetName = extraGroup.activeExtraWidget
				}
			}

			bw.auxlog.Printf("MAIN WIDGET: [%s - %s] -> [%s - %s] -> ACTIVE[%s - %s]",
				widgetName, widgetMode,
				childGroupName, childGroupMode,
				activeExtraWidgetName, childGroup.GetExtraMode())
		}
	} else {
		bw.auxlog.Printf("MAIN WIDGET: [%s - %s]", widgetName, widgetMode)
	}
}

func (bw *BaseWidget) View() string {
	if bw.isExtra {
		panic("BaseWidget.View() should not be called directly on extra widgets. Extra widgets should be rendered through their parent ExtraWidgetGroup.")
	}
	switch bw.WidgetNavigator.GetMode() {
	case common.NavigatorModeList:
		return bw.ViewList()
	case common.NavigatorModeCreate:
		return bw.ViewCreateForm()
	case common.NavigatorModeDetails:
		return bw.ViewDetails()
	case common.NavigatorModeDelete:
		return bw.ViewPrompt()
	case common.NavigatorModeExtra:
		return bw.ViewExtra()
	default:
		panic("invalid view mode")
	}
}

func (bw *BaseWidget) ViewExtra() string {
	return bw.GetExtraWidget().ViewExtra()
}

func (bw *BaseWidget) GetResourceType() string {
	return bw.resourceType
}

func (bw *BaseWidget) SetParentForBaseWidget(parent common.Widget, isExtra bool) {
	bw.parent = parent
	// Set the parent widget reference in WidgetNavigator
	if isExtra {
		bw.WidgetNavigator = nil
	} else {
		bw.WidgetNavigator.SetWidget(parent)

		// For main widgets, automatically set up parent relationships for extra widgets
		if extraGroup, ok := bw.GetExtraWidget().(*ExtraWidgetGroup); ok {
			extraGroup.SetMainWidgetParent(parent)
		}
	}
	bw.isExtra = isExtra
}

// ----------------------------
// Navigator modes
// ----------------------------

func (bw *BaseWidget) GetMode() common.NavigatorMode {
	if bw.isExtra {
		return bw.GetParentNavigatorMode()
	}
	return bw.WidgetNavigator.GetMode()
}

// SetMode sets the navigator mode and initializes inputs for create mode
func (bw *BaseWidget) SetMode(mode common.NavigatorMode) {
	if bw.isExtra {
		panic("SetMode should not be called on extra widgets. Extra widgets don't control app navigation.")
	}
	bw.WidgetNavigator.SetMode(mode)

	// Initialize CreateAdapter inputs when switching to create mode
	if mode == common.NavigatorModeCreate {
		// Clear the cached inputs to ensure fresh schema generation
		bw.cachedInputs = nil

		// Always refresh inputs to ensure fresh state every time (like extra widgets)
		inputs, err := bw.parent.GetInputs()
		if err != nil {
			bw.log.Error("Failed to get inputs", zap.Error(err))
			return
		}
		bw.CreateAdapter.SetInputs(inputs)

		// Reset the form to clear any previous values
		bw.CreateAdapter.ResetCreateForm()
		bw.log.Debug("Profile create inputs refreshed and form reset for navigation")
	}

	// Initialize DetailsAdapter content when switching to details mode
	if mode == common.NavigatorModeDetails {
		// Set loading content initially
		bw.log.Debug("Profile details mode activated, content will be loaded asynchronously")
	}

	if mode == common.NavigatorModeExtra {
		selecterRow := bw.ListAdapter.GetSelectedRowData()
		bw.GetExtraWidget().SetSelectedRowData(selecterRow)
	}
}

func (bw *BaseWidget) InitialMode() common.NavigatorMode {
	return common.NavigatorModeList
}

// ----------------------------
// Dimensions
// ----------------------------

// SetSize overrides DimensionAdapter.SetSize to propagate size changes to adapters
func (bw *BaseWidget) SetSize(width, height int) {
	// Call the embedded DimensionAdapter's SetSize
	bw.DimensionAdapter.SetSize(width, height)
	// Propagate size changes to DetailsAdapter for proper viewport sizing
	bw.DetailsAdapter.SetSize(width, height)
	// Propagate size changes to ExtraWidget for proper viewport sizing (only if ExtraWidgetNavigator is available)
	if bw.ExtraWidgetNavigator != nil {
		bw.GetExtraWidget().SetSize(width, height)
	}
}

// ----------------------------
// ListWidget methods
// ----------------------------

func (bw *BaseWidget) SetListData() tea.Msg {
	// Initialize with profile data from database
	if bw.isExtra {
		bw.auxlog.Println("[ERROR] SetListData called for extra widget, using GetExtraWidget")
		panic(
			"BaseWidget SetListData should not be called for extra widgets. " +
				"Each extra widget should implement its own SetListData method.",
		)
	}

	bw.auxlog.Printf("Setting list data for list widget: %T", bw.parent)

	rest, err := getActiveRest(bw.db)
	if err != nil {
		bw.log.Error("Error getting active REST client", zap.Error(err))
		return msg_types.ErrorMsg{
			Err: err,
		}
	}

	var records RecordSet
	if vastResourceListGetter, ok := any(bw.parent).(common.VastResourceListGetter); ok {
		if records, err = vastResourceListGetter.List(rest); err != nil {
			bw.log.Error("Error fetching records", zap.Error(err))
			return msg_types.ErrorMsg{
				Err: err,
			}
		}
	} else if vastAPIGetter, ok := any(bw.parent).(common.VastAPIGetter); ok {
		var params vast_client.Params
		if query := bw.GetServerParams(); query != nil {
			params = *query
		}
		if records, err = vastAPIGetter.API(rest).List(params); err != nil {
			bw.log.Error("Error fetching records from VastResourceAPI", zap.Error(err))
			return msg_types.ErrorMsg{
				Err: err,
			}
		}
	} else {
		panic("Neither VastResourceListGetter nor VastAPIGetter implemented on parent widget")
	}

	headers := bw.ListAdapter.GetHeaders()
	if len(headers) == 0 {
		panic("ListAdapter headers not set for widget " + bw.GetName())
	}

	data := make([][]string, 0, len(records))
	for _, record := range records {
		row := make([]string, 0, len(headers))
		for _, key := range headers {
			lowerKey := strings.ToLower(key)
			if val, ok := record[lowerKey]; ok {
				row = append(row, bw.formatFieldValue(key, val))
			} else {
				row = append(row, "")
			}
		}
		data = append(data, row)
	}

	bw.ListAdapter.SetListData(data, bw.GetFuzzyListSearchString())
	return nil
}

// formatFieldValue formats field values appropriately for display
// Numeric fields like UID are formatted as integers to avoid scientific notation
func (bw *BaseWidget) formatFieldValue(key string, value interface{}) string {
	lowerKey := strings.ToLower(key)

	// Handle numeric fields that should be displayed as integers
	if isNumericField(lowerKey) {
		if floatVal, ok := value.(float64); ok {
			// Convert float64 to int64 to avoid scientific notation
			return fmt.Sprintf("%.0f", floatVal)
		}
		if intVal, ok := value.(int64); ok {
			return fmt.Sprintf("%d", intVal)
		}
		if intVal, ok := value.(int); ok {
			return fmt.Sprintf("%d", intVal)
		}
	}

	// Default formatting for other fields
	return fmt.Sprintf("%v", value)
}

// isNumericField returns true if the field should be treated as a numeric field
func isNumericField(lowerKey string) bool {
	numericFields := []string{
		"id", "uid", "gid", "port", "size", "count", "number", "num",
		"pid", "tid", "sid", "index", "offset", "length", "bytes",
		"capacity", "used", "free", "available", "total",
	}

	for _, field := range numericFields {
		if lowerKey == field || strings.Contains(lowerKey, field) {
			return true
		}
	}

	return false
}

func (bw *BaseWidget) ViewList() string {
	if bw.isExtra {
		navMode := bw.GetParentNavigatorMode()
		extraNavMode := bw.ExtraWidgetNavigator.GetExtraMode()
		resourceType := bw.GetResourceType()

		panic(fmt.Sprintf(
			"BaseWidget ViewList should not be called for extra widgets. "+
				"Each extra widget should implement its own ViewList method. "+
				"[navMode=%s, extraNavMode=%s, resourceType=%s, widget=%T]",
			navMode.String(),
			extraNavMode.String(),
			resourceType,
			bw,
		))
	}

	return bw.viewList()
}

func (bw *BaseWidget) viewList() string {
	return bw.ListAdapter.ViewList(bw.parent)
}

func (bw *BaseWidget) ClearListData() {
	if bw.GetMode() == common.NavigatorModeExtra {
		// Delegate to the extra widget if in extra mode
		bw.GetExtraWidget().ClearListData()
		return
	}
	bw.ListAdapter.ClearListData()
}

// ----------------------------
// SearchableWidget interface implementation
// ----------------------------

// SetFuzzyListSearchString sets the fuzzy list search query for the widget
func (bw *BaseWidget) SetFuzzyListSearchString(query string) {
	// If this is an individual extra widget, use its own SearchAdapter directly
	if bw.isExtra {
		bw.SearchAdapter.SetFuzzyListSearchString(query)
		return
	}

	// If this is the main widget in extra mode, delegate to the extra widget group
	if bw.GetMode() == common.NavigatorModeExtra {
		bw.GetExtraWidget().SetFuzzyListSearchString(query)
		return
	}

	// Regular case: use own SearchAdapter
	bw.SearchAdapter.SetFuzzyListSearchString(query)
}

// GetFuzzyListSearchString returns the current fuzzy list search query
func (bw *BaseWidget) GetFuzzyListSearchString() string {
	// If this is an individual extra widget, use its own SearchAdapter directly
	if bw.isExtra {
		return bw.SearchAdapter.GetFuzzyListSearchString()
	}

	// If this is the main widget in extra mode, delegate to the extra widget group
	if bw.GetMode() == common.NavigatorModeExtra {
		return bw.GetExtraWidget().GetFuzzyListSearchString()
	}

	// Regular case: use own SearchAdapter
	return bw.SearchAdapter.GetFuzzyListSearchString()
}

// SetFuzzyDetailsSearchString sets the fuzzy details search query for the widget
func (bw *BaseWidget) SetFuzzyDetailsSearchString(query string) {
	// If this is an individual extra widget, use its own SearchAdapter directly
	if bw.isExtra {
		bw.SearchAdapter.SetFuzzyDetailsSearchString(query)
		return
	}

	// If this is the main widget in extra mode, delegate to the extra widget group
	if bw.GetMode() == common.NavigatorModeExtra {
		bw.GetExtraWidget().SetFuzzyDetailsSearchString(query)
		return
	}

	// Regular case: use own SearchAdapter
	bw.SearchAdapter.SetFuzzyDetailsSearchString(query)
}

// GetFuzzyDetailsSearchString returns the current fuzzy details search query
func (bw *BaseWidget) GetFuzzyDetailsSearchString() string {
	// If this is an individual extra widget, use its own SearchAdapter directly
	if bw.isExtra {
		return bw.SearchAdapter.GetFuzzyDetailsSearchString()
	}

	// If this is the main widget in extra mode, delegate to the extra widget group
	if bw.GetMode() == common.NavigatorModeExtra {
		return bw.GetExtraWidget().GetFuzzyDetailsSearchString()
	}

	// Regular case: use own SearchAdapter
	return bw.SearchAdapter.GetFuzzyDetailsSearchString()
}

// SetServerSearchParams sets the server-side search parameters string
func (bw *BaseWidget) SetServerSearchParams(paramStr string) {
	if bw.GetMode() == common.NavigatorModeExtra {
		// Delegate to the extra widget if in extra mode
		bw.GetExtraWidget().SetServerSearchParams(paramStr)
		return
	}
	bw.SearchAdapter.SetServerSearchParams(paramStr)
}

// GetServerSearchParams returns the current server-side search parameters as string
func (bw *BaseWidget) GetServerSearchParams() string {
	if bw.GetMode() == common.NavigatorModeExtra {
		// Delegate to the extra widget if in extra mode
		return bw.GetExtraWidget().GetServerSearchParams()
	}
	return bw.SearchAdapter.GetServerSearchParams()
}

// ClearFuzzyListSearch clears the fuzzy list search
func (bw *BaseWidget) ClearFuzzyListSearch() {
	if bw.GetMode() == common.NavigatorModeExtra {
		// Delegate to the extra widget if in extra mode
		bw.GetExtraWidget().ClearFuzzyListSearch()
		return
	}
	bw.SearchAdapter.ClearFuzzyListSearch()
}

// ClearFuzzyDetailsSearch clears the fuzzy details search
func (bw *BaseWidget) ClearFuzzyDetailsSearch() {
	if bw.GetMode() == common.NavigatorModeExtra {
		// Delegate to the extra widget if in extra mode
		bw.GetExtraWidget().ClearFuzzyDetailsSearch()
		return
	}
	bw.SearchAdapter.ClearFuzzyDetailsSearch()
}

func (bw *BaseWidget) IsServerSearchable() bool {
	if bw.GetMode() == common.NavigatorModeExtra {
		// Delegate to the extra widget if in extra mode
		return bw.GetExtraWidget().IsServerSearchable()
	}
	return bw.GetMode() == common.NavigatorModeList
}

func (bw *BaseWidget) IsFuzzySearchable() bool {
	if bw.GetMode() == common.NavigatorModeExtra {
		// Delegate to the extra widget if in extra mode
		return bw.GetExtraWidget().IsFuzzySearchable()
	}
	return bw.GetMode() == common.NavigatorModeList || bw.GetMode() == common.NavigatorModeDetails
}

// ClearFilters removes any active filters
func (bw *BaseWidget) ClearFilters() {
	if bw.GetMode() == common.NavigatorModeExtra {
		// Delegate to the extra widget if in extra mode
		bw.GetExtraWidget().ClearFilters()
		return
	}
	bw.SearchAdapter.ClearFilters()
}

func (bw *BaseWidget) ClearServerSearchParams() {
	if bw.GetMode() == common.NavigatorModeExtra {
		// Delegate to the extra widget if in extra mode
		bw.GetExtraWidget().ClearServerSearchParams()
		return
	}
	bw.SearchAdapter.ClearServerSearchParams()
}

func (bw *BaseWidget) GetServerParams() *vast_client.Params {
	if bw.GetMode() == common.NavigatorModeExtra {
		// Delegate to the extra widget if in extra mode
		return bw.GetExtraWidget().GetServerParams()
	}
	return bw.SearchAdapter.GetServerParams()
}

// ----------------------------
// CreateWidget methods
// ----------------------------

func (bw *BaseWidget) GetInputs() (common.Inputs, error) {
	// Use dynamic schema-based input generation
	// Extra widgets can now also generate inputs from their FormHints
	if bw.cachedInputs == nil && bw.formHints != nil {
		inputs, err := bw.getInputsWithError()
		if err != nil {
			return nil, err
		}
		bw.cachedInputs = inputs
	}

	return bw.cachedInputs, nil
}

func (bw *BaseWidget) getInputs() common.Inputs {
	// Use dynamic schema-based input generation
	inputs, err := bw.formHints.GetInputsFromCreateSchemaWithCustom(false)
	if err != nil {
		panic(err)
	}
	bw.log.Debug("Generated dynamic inputs from OpenAPI schema",
		zap.Int("input_count", len(inputs)))

	return inputs
}

func (bw *BaseWidget) getInputsWithError() (common.Inputs, error) {
	// Use dynamic schema-based input generation
	inputs, err := bw.formHints.GetInputsFromCreateSchemaWithCustom(false)
	if err != nil {
		return nil, err
	}
	bw.log.Debug("Generated dynamic inputs from OpenAPI schema",
		zap.Int("input_count", len(inputs)))

	return inputs, nil
}

func (bw *BaseWidget) SetInputs(inputs common.Inputs) {
	// For extra widgets, use their own CreateAdapter directly to avoid delegation loops
	if bw.isExtra {
		bw.auxlog.Println("SetInputs called for extra widget - using own CreateAdapter")
		bw.CreateAdapter.SetInputs(inputs)
		return
	}
	// Set inputs for the base widget
	bw.CreateAdapter.SetInputs(inputs)

}

// ViewCreateForm renders the profile creation form
func (bw *BaseWidget) ViewCreateForm() string {
	// Extra widgets can now use the base implementation for form rendering
	// The form will be generated from the FormHints (OpenAPI schema reference)
	return bw.viewCreateForm()

}

// ViewCreateForm renders the profile creation form
func (bw *BaseWidget) viewCreateForm() string {
	// Ensure inputs are set
	if !bw.CreateAdapter.HasInputs() {
		inputs, err := bw.parent.GetInputs()
		if err != nil {
			bw.log.Error("Failed to get inputs for create form", zap.Error(err))
			return "Error: Failed to load create form inputs"
		}
		bw.CreateAdapter.SetInputs(inputs)
		// Start cursor blinking
		bw.CreateAdapter.Init()
	}
	width := bw.GetWidth()
	height := bw.GetHeight()
	return bw.CreateAdapter.RenderCreate(width, height)
}

func (bw *BaseWidget) ResetCreate() {
	bw.CreateAdapter.ResetCreateForm()
}

// UpdateCurrentInput implements UpdateInputAdapter by delegating to CreateAdapter
func (bw *BaseWidget) UpdateCurrentInput(msg tea.Msg) {
	if bw.CreateAdapter != nil {
		bw.CreateAdapter.UpdateCurrentInput(msg)
	}
}

// ToggleFormJSONMode toggles between form and JSON mode in the create adapter
func (bw *BaseWidget) ToggleFormJSONMode() {
	if bw.CreateAdapter != nil {
		bw.CreateAdapter.ToggleFormJSONMode()
	}
}

// StartJSONEditing starts the embedded JSON editor
func (bw *BaseWidget) StartJSONEditing() {
	if bw.CreateAdapter != nil {
		bw.CreateAdapter.StartJSONEditing()
	}
}

// SaveJSONEdits saves the JSON edits back to form
func (bw *BaseWidget) SaveJSONEdits() error {
	if bw.CreateAdapter != nil {
		return bw.CreateAdapter.SaveJSONEdits()
	}
	return nil
}

// CancelJSONEditing cancels JSON editing without saving
func (bw *BaseWidget) CancelJSONEditing() {
	if bw.CreateAdapter != nil {
		bw.CreateAdapter.CancelJSONEditing()
	}
}

// UpdateJSONTextarea updates the JSON textarea
func (bw *BaseWidget) UpdateJSONTextarea(msg tea.Msg) tea.Cmd {
	if bw.CreateAdapter != nil {
		return bw.CreateAdapter.UpdateJSONTextarea(msg)
	}
	return nil
}

// IsJSONMode returns whether the create adapter is in JSON mode
func (bw *BaseWidget) IsJSONMode() bool {
	if bw.CreateAdapter != nil {
		return bw.CreateAdapter.IsJSONMode()
	}
	return false
}

// IsEditingJSON returns whether actively editing JSON
func (bw *BaseWidget) IsEditingJSON() bool {
	if bw.CreateAdapter != nil {
		return bw.CreateAdapter.IsEditingJSON()
	}
	return false
}

func (bw *BaseWidget) CreateFromInputs(inputs common.Inputs) (tea.Cmd, error) {
	if bw.isExtra {
		panic(
			"BaseWidget CreateFromInputs should not be called for extra widgets. " +
				"Each extra widget should implement its own CreateFromInputs method.",
		)
	}
	return bw.createFromInputs(inputs)

}

func (bw *BaseWidget) createFromInputs(inputs common.Inputs) (tea.Cmd, error) {
	bw.log.Info(
		"Creating from inputs",
		zap.String("resource", bw.resourceType),
		zap.Any("inputs", inputs.GetValues()),
	)

	// Get active REST client
	rest, err := getActiveRest(bw.db)
	if err != nil {
		return nil, fmt.Errorf("failed to get REST client: %w", err)
	}

	if vastResourceCreator, ok := any(bw.parent).(common.VastResourceCreator); ok {
		return func() tea.Msg {
			if msg, err := vastResourceCreator.Create(rest); err != nil {
				bw.log.Error("Failed to create resource", zap.Error(err))
				return msg_types.ErrorMsg{
					Err: err,
				}
			} else {
				return msg
			}
		}, nil
	}

	var vastAPIGetter common.VastAPIGetter
	var ok bool
	if vastAPIGetter, ok = any(bw.parent).(common.VastAPIGetter); !ok {
		panic("VastAPIGetter not implemented on parent widget")
	}
	api := vastAPIGetter.API(rest)

	if err := inputs.Validate(); err != nil {
		return nil, err
	}

	// Convert inputs to API payload
	payload := inputs.ToParams()

	bw.log.Debug("Payload prepared", zap.Any("payload", payload))

	// Return async command to create the view
	createFn := func() tea.Msg {

		if beforeVastCreator, ok := any(bw.parent).(common.BeforeVastResourceCreator); ok {
			// Call the before creator hook if implemented
			if err := beforeVastCreator.BeforeCreate(payload); err != nil {
				bw.log.Error("Before create hook failed",
					zap.String("resourceType", bw.resourceType),
					zap.Any("payload", payload),
					zap.Error(err))
				return msg_types.ErrorMsg{
					Err: fmt.Errorf("before create hook failed: %w", err),
				}
			}
		}

		record, err := api.Create(payload)
		if err != nil {
			bw.log.Error("Failed to create resource",
				zap.String("resourceType", bw.resourceType),
				zap.Any("payload", payload),
				zap.Error(err))
			return msg_types.ErrorMsg{
				Err: err,
			}
		}

		if afterVastCreator, ok := any(bw.parent).(common.AfterVastResourceCreator); ok {
			// Call the after creator hook if implemented
			msg, err := afterVastCreator.AfterCreate(record)
			if err != nil {
				bw.log.Error("After create hook failed",
					zap.String("resourceType", bw.resourceType),
					zap.Any("record", record),
					zap.Error(err))
				return msg_types.ErrorMsg{
					Err: fmt.Errorf("after create hook failed: %w", err),
				}
			}
			return msg
		} else {
			bw.SetContent(record)
			bw.SetModeMust(common.NavigatorModeDetails)
			return msg_types.SetDataMsg{
				UseSpinner: false, // No spinner needed for create, just updated data in the background for list representation
			}
		}
	}

	return createFn, nil
}

// ----------------------------
// DetailsWidget methods
// ----------------------------

func (bw *BaseWidget) Details(selectedRowData common.RowData) (tea.Cmd, error) {
	if bw.isExtra {
		panic(
			"Base Details implementation should not be called for extra widgets." +
				" Each extra widget should implement its own Details method.",
		)
	}
	return bw.details(selectedRowData)

}

func (bw *BaseWidget) details(selectedRowData common.RowData) (tea.Cmd, error) {
	bw.log.Debug("Details initiated",
		zap.Any("rowData", selectedRowData),
		zap.Int("dataLength", selectedRowData.Len()))

	// Get active REST client
	rest, err := getActiveRest(bw.db)
	if err != nil {
		return nil, fmt.Errorf("failed to get REST client: %w", err)
	}

	if vastResourceDetailsGetter, ok := any(bw.parent).(common.VastResourceDetailsGetter); ok {
		return func() tea.Msg {
			record, err := vastResourceDetailsGetter.GetDetails(rest, selectedRowData)
			if err != nil {
				bw.log.Error("Failed to load details", zap.Error(err))
				return msg_types.DetailsContentMsg{
					Content:      fmt.Sprintf("Failed to load details: %s", err.Error()),
					ResourceType: bw.resourceType,
					Error:        err,
				}
			}

			return msg_types.DetailsContentMsg{
				Content:      record,
				ResourceType: bw.resourceType,
				Error:        nil,
			}
		}, nil
	}

	if selectedRowData.Len() == 0 {
		return func() tea.Msg {
			return msg_types.DetailsContentMsg{
				Content:      "No row selected",
				ResourceType: bw.resourceType,
				Error:        nil,
			}
		}, nil
	}

	var vastAPIGetter common.VastAPIGetter
	var ok bool
	if vastAPIGetter, ok = any(bw.parent).(common.VastAPIGetter); !ok {
		panic("Details: VastAPIGetter not implemented on parent widget")
	}
	api := vastAPIGetter.API(rest)

	// Extract ID from the row data
	id := selectedRowData.GetID()
	if id == "" {
		return func() tea.Msg {
			return msg_types.DetailsContentMsg{
				Content:      "Invalid data: missing ID",
				ResourceType: bw.resourceType,
				Error:        fmt.Errorf("missing ID"),
			}
		}, nil
	}

	// Return async command that will load details in background
	return func() tea.Msg {
		record, err := api.GetById(id)
		if err != nil {
			bw.log.Error("Failed to fetch details",
				zap.Any("id", id),
				zap.Error(err))
			return msg_types.DetailsContentMsg{
				Content:      fmt.Sprintf("Failed to load details: %s", err.Error()),
				ResourceType: bw.resourceType,
				Error:        err,
			}
		}

		return msg_types.DetailsContentMsg{
			Content:      record,
			ResourceType: bw.resourceType,
			Error:        nil,
		}
	}, nil
}

func (bw *BaseWidget) DetailsOnSelect() bool {
	return true
}

func (bw *BaseWidget) Select(selectedData common.RowData) (tea.Cmd, error) {
	// For extra widgets, handle selection differently since WidgetNavigator is nil
	if bw.isExtra {
		// Extra widgets should implement their own Select method
		panic("BaseWidget.Select should not be called on extra widgets. Extra widgets should implement their own Select method.")
	}

	if bw.WidgetNavigator.GetMode() == common.NavigatorModeExtra {
		// Delegate selection to the extra widget
		return bw.GetExtraWidget().Select(selectedData)
	}
	// No-op for BaseWidget, should be implemented by specific widgets
	return nil, nil
}

func (bw *BaseWidget) SetDetailsData(details any) {
	bw.DetailsAdapter.SetContent(details)
}

func (bw *BaseWidget) GetSelectedRowData() common.RowData {
	return bw.ListAdapter.GetSelectedRowData()
}

func (bw *BaseWidget) SetSelectedRowData(data common.RowData) {
	bw.selectedRowData = data
}

func (bw *BaseWidget) Reset() {
	bw.ClearListData()
	bw.ResetCreate()
	bw.CreateAdapter.ResetCreateForm()
	bw.ListAdapter.ClearListData()
}

func (bw *BaseWidget) ViewDetails() string {
	// Extra widgets can now use the base implementation for details rendering
	// The details will show response data set via SetDetailsData()
	return bw.viewDetails()
}

func (bw *BaseWidget) viewDetails() string {
	width := bw.GetWidth()
	height := bw.GetHeight()
	return bw.DetailsAdapter.ViewDetails(width, height, bw.SearchAdapter.GetFuzzyDetailsSearchString())
}

// ----------------------------
// DeleteWidget methods
// ----------------------------

// Delete implements the DeleteWidget interface for view deletion
func (bw *BaseWidget) Delete(selectedRowData common.RowData) (tea.Cmd, error) {
	if bw.isExtra {
		panic("BaseWidget Delete should not be called for extra widgets. " +
			"Each extra widget should implement its own Delete method.",
		)
	}
	return bw.delete(selectedRowData)

}

func (bw *BaseWidget) delete(selectedRowData common.RowData) (tea.Cmd, error) {
	bw.log.Debug("Deletion initiated",
		zap.Any("rowData", selectedRowData),
		zap.Int("dataLength", selectedRowData.Len()))

	// Get active REST client
	rest, err := getActiveRest(bw.db)
	if err != nil {
		return nil, fmt.Errorf("failed to get REST client: %w", err)
	}

	if vastResourceDeleter, ok := any(bw.parent).(common.VastResourceDeleter); ok {
		return func() tea.Msg {
			if msg, err := vastResourceDeleter.Delete(rest, selectedRowData); err != nil {
				bw.log.Error("Failed to delete resource", zap.Error(err))
				return msg_types.ErrorMsg{
					Err: err,
				}
			} else {
				return msg
			}
		}, nil
	}

	if selectedRowData.Len() == 0 {
		return nil, fmt.Errorf("no row data provided for deletion")
	}

	var vastAPIGetter common.VastAPIGetter
	var ok bool
	if vastAPIGetter, ok = any(bw.parent).(common.VastAPIGetter); !ok {
		panic("VastAPIGetter not implemented on parent widget")
	}
	api := vastAPIGetter.API(rest)

	// Extract ID from the row data
	idStr := selectedRowData.GetID()
	if idStr == "" {
		return nil, fmt.Errorf("invalid view data: missing ID")
	}

	return func() tea.Msg {
		if _, err := api.DeleteById(idStr, nil, nil); err != nil {
			bw.log.Error("Failed to delete",
				zap.String("id", idStr),
				zap.Error(err))
			return msg_types.ErrorMsg{
				Err: err,
			}
		}
		bw.SetListData() // Refresh the list after deletion
		bw.SetModeMust(common.NavigatorModeList)
		return nil
	}, nil
}

// ------------------------------
// ExtraWidget methods
// ------------------------------

func (bw *BaseWidget) GetAllowedExtraNavigatorModes() []common.ExtraNavigatorMode {
	// Return nil - no restrictions on navigator modes
	return nil
}

func (bw *BaseWidget) GetExtraResourceType() string {
	// This was causing a recursive call. An extra widget should return its own resource type.
	return bw.resourceType
}

func (bw *BaseWidget) CanUseExtra() bool {
	// Check if the extra widget is enabled
	if bw.ExtraWidgetNavigator == nil {
		return false
	}
	return bw.GetExtraWidget().CanUseExtra()
}

func (bw *BaseWidget) InitialExtraMode() common.ExtraNavigatorMode {
	if bw.ExtraWidgetNavigator == nil {
		return common.ExtraNavigatorModeList
	}
	return bw.GetExtraWidget().InitialExtraMode()
}

// ------------------------------
// PromptWidget methods
// ------------------------------

func (bw *BaseWidget) ViewPrompt() string {
	// Extra widgets can also use the default prompt implementation
	return bw.viewPrompt()
}

func (bw *BaseWidget) viewPrompt() string {
	// Safety check: ensure PromptAdapter is initialized
	if bw.PromptAdapter == nil {
		bw.log.Error("PromptAdapter is nil, cannot display prompt dialog")
		return "Error: Prompt dialog not available"
	}

	var selectedInfo string
	selectedRowData := bw.GetSelectedRowData()
	if selectedRowData.Len() > 0 {
		// Convert RowData to string representation using ordered slice
		values := selectedRowData.ToSlice()
		selectedInfo = strings.Join(values, " ")
	}

	if selectedInfo == "" {
		selectedInfo = "the selected item"
	}

	msg := fmt.Sprintf("Are you sure you want to delete this %s?\n\n%s", bw.resourceType, selectedInfo)
	title := fmt.Sprintf(" delete: %s ", bw.resourceType)

	return bw.PromptAdapter.PromptDo(msg, title, bw.GetWidth(), bw.GetHeight())
}

// TogglePromptSelection toggles between Yes and No buttons in the prompt
func (bw *BaseWidget) TogglePromptSelection() {
	if bw.PromptAdapter != nil {
		bw.PromptAdapter.ToggleSelection()
	}
}

// IsPromptNoSelected returns true if "No" button is currently selected
func (bw *BaseWidget) IsPromptNoSelected() bool {
	if bw.PromptAdapter != nil {
		return bw.PromptAdapter.IsNoSelected()
	}
	return false // Default to Yes (false means No is not selected)
}

// ------------------------------
// FormNavigateAdaptor methods
// ------------------------------

// NextInput moves focus to the next input field in the form
func (bw *BaseWidget) NextInput() {
	if bw.CreateAdapter != nil {
		bw.CreateAdapter.NextInput()
	}
}

// PrevInput moves focus to the previous input field in the form
func (bw *BaseWidget) PrevInput() {
	if bw.CreateAdapter != nil {
		bw.CreateAdapter.PrevInput()
	}
}

// ------------------------------
// ExtraNavigator methods (for extra widgets)
// ------------------------------

// GetExtraMode delegates to the embedded ExtraWidgetNavigator
func (bw *BaseWidget) GetExtraMode() common.ExtraNavigatorMode {
	if bw.isExtra {
		return bw.ExtraWidgetNavigator.GetExtraMode()
	}

	if bw.GetMode() == common.NavigatorModeExtra {
		bw.auxlog.Printf("contex of GetExtraMode: %T", bw.parent)

		if bw.ExtraWidgetNavigator == nil {
			panic("GetExtraMode: ExtraWidgetNavigator is nil but mode is NavigatorModeExtra")
		}
		return bw.GetExtraWidget().GetExtraMode()
	}

	panic(
		"GetExtraMode should not be called on non-extra widgets. " +
			"Each extra widget should implement its own GetExtraMode method.",
	)

}

// SetExtraMode delegates to the embedded ExtraWidgetNavigator
func (bw *BaseWidget) SetExtraMode(mode common.ExtraNavigatorMode) {
	// Add trace logging
	bw.TraceLog()
	bw.auxlog.Printf("🟢 SET_EXTRA_MODE: widget=%T mode=%s", bw.parent, mode.String())

	// For extra widgets, delegate to parent widget but also provide context
	if bw.isExtra {
		bw.ExtraWidgetNavigator.SetExtraMode(mode)
		return
	}

	if bw.ExtraWidgetNavigator != nil {
		bw.GetExtraWidget().SetExtraMode(mode)
	}
}

// UpdateViewPort delegates to the DetailsAdapter for viewport navigation (scrolling)
func (bw *BaseWidget) UpdateViewPort(msg tea.Msg) tea.Cmd {

	if bw.GetMode() == common.NavigatorModeExtra {
		adapter := bw.GetExtraWidget().(common.ViewPortAdapter)
		return adapter.UpdateViewPort(msg)
	}
	// For main widgets, use own DetailsAdapter
	return bw.DetailsAdapter.UpdateViewPort(msg)
}

// ExtraNavigate delegates to the ExtraWidgetGroup if available, otherwise to the embedded ExtraWidgetNavigator
func (bw *BaseWidget) ExtraNavigate(msg tea.Msg) (tea.Cmd, bool) {
	// For main widgets, delegate to the ExtraWidgetGroup
	if bw.ExtraWidgetNavigator != nil && bw.ExtraWidgetNavigator.GetExtraWidget() != nil {
		return bw.ExtraWidgetNavigator.GetExtraWidget().ExtraNavigate(msg)
	}
	if bw.ExtraWidgetNavigator != nil {
		return bw.ExtraWidgetNavigator.ExtraNavigate(msg)
	}
	return nil, false
}

// ------------------------------
// KeyBindings methods
// ------------------------------

func (bw *BaseWidget) GetKeyBindings() []common.KeyBinding {
	if bw.isExtra {
		// Extra widgets should implement their own key bindings
		return []common.KeyBinding{} // Return empty for now
	}
	var keyBindings []common.KeyBinding
	switch bw.WidgetNavigator.GetMode() {
	case common.NavigatorModeList:
		keyBindings = bw.GetListKeyBindings()
	case common.NavigatorModeCreate:
		keyBindings = bw.GetCreateKeyBindings()
	case common.NavigatorModeDelete:
		keyBindings = bw.GetDeleteKeyBindings()
	case common.NavigatorModeDetails:
		keyBindings = bw.GetDetailsKeyBindings()
	case common.NavigatorModeExtra:
		// Delegate to the extra widget group
		keyBindings = bw.GetExtraWidget().GetKeyBindings()
	}

	return keyBindings
}

func (bw *BaseWidget) GetListKeyBindings() []common.KeyBinding {
	// Return cached bindings if available
	if bw.cachedListKeyBindings == nil {
		availableBindings := []common.KeyBinding{
			{Key: "<:>", Desc: "resources", Generic: true},
			{Key: "</>", Desc: "search", Generic: true},
			{Key: "<?>", Desc: "query params", Generic: true},
			{Key: "<↑/↓>", Desc: "navigate"},
			{Key: "<enter>", Desc: "select"},
		}

		// Add delete key binding only if Delete mode is allowed
		if bw.isModeAllowed(common.NavigatorModeDelete) {
			availableBindings = append(availableBindings, common.KeyBinding{Key: "<ctrl+d>", Desc: "delete"})
		}

		// Add details key binding only if Details mode is allowed
		if bw.isModeAllowed(common.NavigatorModeDetails) {
			availableBindings = append(availableBindings, common.KeyBinding{Key: "<d>", Desc: "describe"})
		}

		if bw.CanUseExtra() {
			availableBindings = append(availableBindings, common.KeyBinding{Key: "<x>", Desc: "extra actions"})
		}

		// Add create key binding only if Create mode is allowed and inputs are available
		inputs, err := bw.parent.GetInputs()
		if err == nil && len(inputs) > 0 && bw.isModeAllowed(common.NavigatorModeCreate) {
			availableBindings = append(availableBindings, common.KeyBinding{Key: "<n>", Desc: "new"})
		}

		// Collect shortcuts from all extra widgets
		if bw.CanUseExtra() {
			for _, shortcut := range bw.ShortCuts() {
				if keyBinding := shortcut.ShortCut(); keyBinding != nil {
					availableBindings = append(availableBindings, *keyBinding)
				}
			}
		}

		bindings := make([]common.KeyBinding, 0, len(availableBindings))
		notAllowedKeys := bw.WidgetNavigator.NotAllowedListKeys

		for _, binding := range availableBindings {
			strippedBinding := strings.Trim(binding.Key, "<>")
			if _, ok := notAllowedKeys[strippedBinding]; !ok {
				bindings = append(bindings, binding)
			}
		}
		bw.cachedListKeyBindings = bindings

	}

	return bw.cachedListKeyBindings

}

func (bw *BaseWidget) GetCreateKeyBindings() []common.KeyBinding {
	if bw.cachedCreateKeyBindings == nil {
		availableBindings := []common.KeyBinding{
			{Key: "<tab>", Desc: "next input"},
			{Key: "<shift+tab>", Desc: "previous input"},
			{Key: "<ctrl+s>", Desc: "submit"},
			{Key: "<esc>", Desc: "back"},
			{Key: "<space>", Desc: "toggle boolean"},
			{Key: "<,>", Desc: "array delimiter"},
			{Key: "<ctrl+t>", Desc: "toggle form/json"},
		}

		bindings := make([]common.KeyBinding, 0, len(availableBindings))
		notAllowedKeys := bw.WidgetNavigator.NotAllowedCreateKeys

		for _, binding := range availableBindings {
			strippedBinding := strings.Trim(binding.Key, "<>")
			if _, ok := notAllowedKeys[strippedBinding]; !ok {
				bindings = append(bindings, binding)
			}
		}
		bw.cachedCreateKeyBindings = bindings
	}
	return bw.cachedCreateKeyBindings
}

func (bw *BaseWidget) GetDeleteKeyBindings() []common.KeyBinding {
	if bw.cachedDeleteKeyBindings == nil {
		availableBindings := []common.KeyBinding{
			{Key: "<←/→/tab>", Desc: "toggle"},
			{Key: "<n or esc>", Desc: "cancel"},
			{Key: "<enter>", Desc: "confirm"},
		}

		bindings := make([]common.KeyBinding, 0, len(availableBindings))
		notAllowedKeys := bw.WidgetNavigator.NotAllowedDeleteKeys

		for _, binding := range availableBindings {
			strippedBinding := strings.Trim(binding.Key, "<>")
			if _, ok := notAllowedKeys[strippedBinding]; !ok {
				bindings = append(bindings, binding)
			}
		}
		bw.cachedDeleteKeyBindings = bindings
	}
	return bw.cachedDeleteKeyBindings
}
func (bw *BaseWidget) GetDetailsKeyBindings() []common.KeyBinding {
	if bw.cachedDetailsKeyBindings == nil {
		availableBindings := []common.KeyBinding{
			{Key: "</>", Desc: "search", Generic: true},
			{Key: "<↑/↓>", Desc: "scroll"},
			{Key: "<pgup/pgdn>", Desc: "page"},
			{Key: "<ctrl+s>", Desc: "copy to clipboard"},
			{Key: "<ctrl+r>", Desc: "refresh"},
			{Key: "<esc>", Desc: "back"},
		}

		bindings := make([]common.KeyBinding, 0, len(availableBindings))
		notAllowedKeys := bw.WidgetNavigator.NotAllowedDetailsKeys

		for _, binding := range availableBindings {
			strippedBinding := strings.Trim(binding.Key, "<>")
			if _, ok := notAllowedKeys[strippedBinding]; !ok {
				bindings = append(bindings, binding)
			}
		}
		bw.cachedDetailsKeyBindings = bindings
	}
	return bw.cachedDetailsKeyBindings
}

// GetAllowedNavigatorModes and GetNotAllowedNavigatorModes provide navigation mode restrictions
// for widgets. These methods control which key bindings are shown in the UI:
//
// - If Details mode is prohibited: <d> key binding will be hidden
// - If Create mode is prohibited: <n> key binding will be hidden
// - If Delete mode is prohibited: <ctrl+d> key binding will be hidden
//
// Example implementation for a read-only widget:
//
//   func (w *MyReadOnlyWidget) GetNotAllowedNavigatorModes() []common.NavigatorMode {
//       return []common.NavigatorMode{
//           common.NavigatorModeCreate,  // Hide <n> key
//           common.NavigatorModeDelete,  // Hide <ctrl+d> key
//       }
//   }
//
// Example implementation for a details-only widget:
//
//   func (w *MyDetailsOnlyWidget) GetAllowedNavigatorModes() []common.NavigatorMode {
//       return []common.NavigatorMode{
//           common.NavigatorModeList,     // Allow list mode
//           common.NavigatorModeDetails,  // Allow <d> key
//       }
//   }
//
// Note: GetAllowedNavigatorModes takes precedence over GetNotAllowedNavigatorModes.
// If GetAllowedNavigatorModes returns a non-nil slice, only those modes will be allowed.

func (bw *BaseWidget) GetAllowedNavigatorModes() []common.NavigatorMode {
	// Return nil - no restrictions on navigator modes
	return nil
}

func (bw *BaseWidget) GetNotAllowedNavigatorModes() []common.NavigatorMode {
	// Return nil - no restrictions on navigator modes
	return nil
}

// isModeAllowed checks if a navigator mode is allowed based on the widget's restrictions
func (bw *BaseWidget) isModeAllowed(mode common.NavigatorMode) bool {
	// Check allowed modes first (if specified, this takes precedence)
	if allowedModes := bw.parent.GetAllowedNavigatorModes(); allowedModes != nil {
		for _, allowedMode := range allowedModes {
			if allowedMode == mode {
				return true
			}
		}
		return false // Mode not in allowed list
	}

	// Check not allowed modes (if no allowed list is specified)
	if notAllowedModes := bw.parent.GetNotAllowedNavigatorModes(); notAllowedModes != nil {
		for _, notAllowedMode := range notAllowedModes {
			if notAllowedMode == mode {
				return false // Mode is explicitly not allowed
			}
		}
	}

	// Default: allow all modes if no restrictions are specified
	return true
}

// ShortCut returns nil by default - widgets can override this to provide shortcuts
func (bw *BaseWidget) ShortCut() *common.KeyBinding {
	return nil
}

func (bw *BaseWidget) ShortCuts() map[string]common.ExtraWidget {
	return bw.GetExtraWidget().(*ExtraWidgetGroup).ShortCuts()

}
