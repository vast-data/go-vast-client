package widgets

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"time"
	"vastix/internal/database"
	"vastix/internal/logging"
	"vastix/internal/msg_types"
	"vastix/internal/tui/widgets/adapters"
	"vastix/internal/tui/widgets/common"

	tea "github.com/charmbracelet/bubbletea"
	vast_client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/resources/untyped"
	"go.uber.org/zap"
)

type (
	params                     = vast_client.Params
	VMSRest                    = vast_client.VMSRest
	VastResourceAPI            = vast_client.VastResourceAPI
	VastResourceAPIWithContext = vast_client.VastResourceAPIWithContext
	RecordSet                  = vast_client.RecordSet
	Record                     = vast_client.Record
)

// BaseWidget common properties and methods for all widgets
type BaseWidget struct {
	resourceType string // Type of resource this widget represents, e.g., "views", "quotas", "users" etc.

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

	// Resourceless indicates if this extra widget can work without parent data
	// Default is true (most extra widgets don't need parent resources)
	// Set to false for widgets like VIP pool forwarding that require SSH connections
	resourceless bool

	// FormHints
	formHints *common.FormHints

	log    *zap.Logger
	auxlog *log.Logger

	// Selected row data
	selectedRowData common.RowData // Currently selected row data of parent widget

	// Callback functions for VAST resource operations
	// If nil, standard implementation via VastAPIGetter will be used
	listFn         common.ListFunc
	getDetailsFn   common.GetDetailsFunc
	createFn       common.CreateFunc
	beforeCreateFn common.BeforeCreateFunc
	afterCreateFn  common.AfterCreateFunc
	deleteFn       common.DeleteFunc
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
		resourceless: true, // Default: extra widgets don't need parent resources

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
// Callback setters for VAST resource operations
// ----------------------------

// SetListCallback sets a custom list function
// If nil, the standard VastAPIGetter.API().List() will be used
func (bw *BaseWidget) SetListCallback(fn common.ListFunc) {
	bw.listFn = fn
}

// SetGetDetailsCallback sets a custom get details function
// If nil, the standard VastAPIGetter.API().GetById() will be used
func (bw *BaseWidget) SetGetDetailsCallback(fn common.GetDetailsFunc) {
	bw.getDetailsFn = fn
}

// SetCreateCallback sets a custom create function
// If nil, the standard VastAPIGetter.API().Create() with before/after hooks will be used
func (bw *BaseWidget) SetCreateCallback(fn common.CreateFunc) {
	bw.createFn = fn
}

// SetBeforeCreateCallback sets a hook to run before resource creation
// Only used when createFn is nil (standard creation flow)
func (bw *BaseWidget) SetBeforeCreateCallback(fn common.BeforeCreateFunc) {
	bw.beforeCreateFn = fn
}

// SetAfterCreateCallback sets a hook to run after resource creation
// Only used when createFn is nil (standard creation flow)
func (bw *BaseWidget) SetAfterCreateCallback(fn common.AfterCreateFunc) {
	bw.afterCreateFn = fn
}

// SetDeleteCallback sets a custom delete function
// If nil, the standard VastAPIGetter.API().Delete() will be used
func (bw *BaseWidget) SetDeleteCallback(fn common.DeleteFunc) {
	bw.deleteFn = fn
}

// TogglePromptSelection toggles between Yes/No buttons in the prompt
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
	return false
}

// SetPromptSelection sets the button selection (true for No, false for Yes)
func (bw *BaseWidget) SetPromptSelection(selectNo bool) {
	if bw.PromptAdapter != nil {
		bw.PromptAdapter.SetSelection(selectNo)
	}
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
	// Use the base context from the database service
	return bw.SetListDataWithContext(bw.db.GetContext())
}

// SetListDataWithContext fetches list data with the provided context.
// This allows callers to pass context with special flags (e.g., to skip interceptor logging).
func (bw *BaseWidget) SetListDataWithContext(ctx context.Context) tea.Msg {
	// Initialize with profile data from database
	if bw.isExtra {
		bw.auxlog.Println("[ERROR] SetListData called for extra widget, using GetExtraWidget")
		panic(
			"BaseWidget SetListData should not be called for extra widgets. " +
				"Each extra widget should implement its own SetListData method.",
		)
	}

	// Check if the parent widget supports LIST operations
	if autoWidget, ok := bw.parent.(*AutoGeneratedWidget); ok {
		if !autoWidget.SupportsListOperation() {
			bw.log.Info("Resource does not support LIST operation, skipping list fetch",
				zap.String("resourceType", bw.resourceType))
			// Set empty list data and return nil (no error)
			bw.ListAdapter.SetListData([][]string{}, bw.GetFuzzyListSearchString())
			bw.SetSelectedRowData(common.RowData{})
			return nil
		}
	}

	rest, err := getActiveRest(bw.db)
	if err != nil {
		bw.log.Error("Error getting active REST client", zap.Error(err))
		return msg_types.ErrorMsg{
			Err: err,
		}
	}

	var records RecordSet
	// Use callback if provided, otherwise use standard VastAPIGetter
	if bw.listFn != nil {
		if records, err = bw.listFn(rest); err != nil {
			bw.log.Error("Error fetching records via callback", zap.Error(err))
			return msg_types.ErrorMsg{
				Err: err,
			}
		}
	} else if vastAPIGetter, ok := any(bw.parent).(common.VastAPIGetter); ok {
		var params vast_client.Params
		if query := bw.GetServerParams(); query != nil {
			params = *query
		}
		// VastAPIGetter now returns VastResourceAPIWithContext, so we can use ListWithContext directly
		if records, err = vastAPIGetter.API(rest).ListWithContext(ctx, params); err != nil {
			bw.log.Error("Error fetching records from VastResourceAPI", zap.Error(err))
			return msg_types.ErrorMsg{
				Err: err,
			}
		}
	} else {
		panic("Neither listFn callback nor VastAPIGetter implemented on parent widget")
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

	// Clear any cached selected row data to ensure fresh data is used on next selection
	bw.SetSelectedRowData(common.RowData{})

	return nil
}

// formatFieldValue formats field values appropriately for display
// Numeric fields like UID are formatted as integers to avoid scientific notation
func (bw *BaseWidget) formatFieldValue(key string, value interface{}) string {
	lowerKey := strings.ToLower(key)

	// Handle nil values
	if value == nil {
		return "<nil>"
	}

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

	// Handle maps - display as JSON-like syntax
	if mapVal, ok := value.(map[string]interface{}); ok {
		if len(mapVal) == 0 {
			return "{}"
		}
		// For non-empty maps, show abbreviated representation
		return fmt.Sprintf("{%d keys}", len(mapVal))
	}

	// Handle slices/arrays
	if reflect.TypeOf(value).Kind() == reflect.Slice {
		sliceVal := reflect.ValueOf(value)
		if sliceVal.Len() == 0 {
			return "[]"
		}
		// For non-empty arrays, show count
		return fmt.Sprintf("[%d items]", sliceVal.Len())
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
	if bw.formHints != nil {
		inputs, err := bw.getInputsWithError()
		return inputs, err
	}

	return nil, nil
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

// GetInputsWithError is an exported version of getInputsWithError for testing input availability
func (bw *BaseWidget) GetInputsWithError() (common.Inputs, error) {
	return bw.getInputsWithError()
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
		// Get inputs from the widget
		// For extra widgets (ExtraMethodWidget), parent points to self and has GetInputs override
		var inputs common.Inputs
		var err error

		bw.auxlog.Printf("[viewCreateForm] START: isExtra=%v, resourceType=%s, parent=%T",
			bw.isExtra, bw.resourceType, bw.parent)

		// Call GetInputs on parent (which should be the widget with potential override)
		if bw.parent != nil {
			inputs, err = bw.parent.GetInputs()
			bw.auxlog.Printf("[viewCreateForm] parent.GetInputs() returned: count=%d, err=%v",
				len(inputs), err)
		} else {
			// Fallback to self
			inputs, err = bw.GetInputs()
			bw.auxlog.Printf("[viewCreateForm] bw.GetInputs() returned: count=%d, err=%v",
				len(inputs), err)
		}

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

	// Use callback if provided
	if bw.createFn != nil {
		return func() tea.Msg {
			if msg, err := bw.createFn(rest); err != nil {
				bw.log.Error("Failed to create resource via callback", zap.Error(err))
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

		// Call before create callback if provided
		if bw.beforeCreateFn != nil {
			if err := bw.beforeCreateFn(payload); err != nil {
				bw.log.Error("Before create callback failed",
					zap.String("resourceType", bw.resourceType),
					zap.Any("payload", payload),
					zap.Error(err))
				return msg_types.ErrorMsg{
					Err: fmt.Errorf("before create callback failed: %w", err),
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

		// Handle potentially async task (wait for completion if needed)
		if err := bw.handleMaybeAsyncTask(rest, record); err != nil {
			bw.log.Error("Async task failed",
				zap.String("resourceType", bw.resourceType),
				zap.Error(err))
			return msg_types.ErrorMsg{
				Err: fmt.Errorf("async task failed: %w", err),
			}
		}

		// Call after create callback if provided (for additional work like saving to DB)
		if bw.afterCreateFn != nil {
			// For non-extra widgets, there's no parent row data, so pass an empty RowData
			emptyRowData := common.NewRowData([]string{}, []string{})
			_, err := bw.afterCreateFn(record, emptyRowData)
			if err != nil {
				bw.log.Error("After create callback failed",
					zap.String("resourceType", bw.resourceType),
					zap.Any("record", record),
					zap.Error(err))
				return msg_types.ErrorMsg{
					Err: fmt.Errorf("after create callback failed: %w", err),
				}
			}
			// Callback succeeded, continue with standard flow
		}

		// Always set content and switch to details mode after successful create
		bw.SetContent(record)
		bw.SetModeMust(common.NavigatorModeDetails)
		return msg_types.SetDataMsg{
			UseSpinner: false, // No spinner needed for create, just updated data in the background for list representation
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

	// Use callback if provided
	if bw.getDetailsFn != nil {
		return func() tea.Msg {
			record, err := bw.getDetailsFn(rest, selectedRowData)
			if err != nil {
				bw.log.Error("Failed to load details via callback", zap.Error(err))
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
	// If we're in extra mode, delegate to the ExtraWidgetGroup
	if bw.GetMode() == common.NavigatorModeExtra {
		// Set details on the extra widget group (which delegates to the active extra widget)
		bw.GetExtraWidget().SetDetailsData(details)
		// Switch to details mode to display the result
		if extraWidgetGroup, ok := bw.GetExtraWidget().(*ExtraWidgetGroup); ok {
			extraWidgetGroup.SetExtraModeMust(common.ExtraNavigatorModeDetails)
		}
		return
	}
	// Normal mode - set details on this widget
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

// Clean performs cleanup when the widget is being closed (e.g., Ctrl+C)
// If in extra mode, it checks if the active extra widget implements LeaveWidget and calls it
func (bw *BaseWidget) Clean() {
	// Check if we're in extra mode
	if bw.GetMode() == common.NavigatorModeExtra {
		extraWidget := bw.GetExtraWidget()
		if extraWidget != nil {
			// Check if the extra widget implements LeaveWidget
			if leaveWidget, ok := extraWidget.(common.LeaveWidget); ok {
				bw.auxlog.Printf("Extra widget implements LeaveWidget, calling it")
				if err := leaveWidget.LeaveWidget(); err != nil {
					bw.auxlog.Printf("Error leaving extra widget: %v", err)
				}
			} else {
				bw.auxlog.Printf("Extra widget does NOT implement LeaveWidget")
			}
		}
	}
	// Perform any additional cleanup for the base widget
	bw.Reset()
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

	// Use callback if provided
	if bw.deleteFn != nil {
		return func() tea.Msg {
			if msg, err := bw.deleteFn(rest, selectedRowData); err != nil {
				bw.log.Error("Failed to delete resource via callback", zap.Error(err))
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
		result, err := api.DeleteById(idStr, nil, nil)
		if err != nil {
			bw.log.Error("Failed to delete",
				zap.String("id", idStr),
				zap.Error(err))
			return msg_types.ErrorMsg{
				Err: err,
			}
		}

		// Handle potentially async task (wait for completion if needed)
		if err := bw.handleMaybeAsyncTask(rest, result); err != nil {
			bw.log.Error("Async delete task failed",
				zap.String("id", idStr),
				zap.Error(err))
			return msg_types.ErrorMsg{
				Err: fmt.Errorf("async delete task failed: %w", err),
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
		// For extra widgets, check the extra mode and return appropriate bindings
		if bw.ExtraWidgetNavigator != nil {
			mode := bw.ExtraWidgetNavigator.GetExtraMode()
			switch mode {
			case common.ExtraNavigatorModePrompt:
				bindings := bw.GetPromptKeyBindings()
				return bindings
			case common.ExtraNavigatorModeDetails:
				return bw.GetDetailsKeyBindings()
			// Add other extra modes as needed
			default:
				return []common.KeyBinding{}
			}
		}
		return []common.KeyBinding{}
	}
	var keyBindings []common.KeyBinding
	mode := bw.WidgetNavigator.GetMode()
	switch mode {
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
		extraWidget := bw.GetExtraWidget()
		keyBindings = extraWidget.GetKeyBindings()
	}

	return keyBindings
}

func (bw *BaseWidget) GetListKeyBindings() []common.KeyBinding {
	availableBindings := []common.KeyBinding{
		{Key: "<:>", Desc: "resources", Generic: true},
		{Key: "</>", Desc: "search", Generic: true},
		{Key: "<?>", Desc: "query params", Generic: true},
	}

	// Only add navigation and select keys if the widget supports list operations
	supportsListOps := true
	supportsReadOps := true
	if autoWidget, ok := bw.parent.(*AutoGeneratedWidget); ok {
		supportsListOps = autoWidget.SupportsListOperation()
		supportsReadOps = autoWidget.SupportsReadOperation()
	}

	// Show navigation keybinding if LIST is supported
	if supportsListOps {
		availableBindings = append(availableBindings,
			common.KeyBinding{Key: "<↑/↓>", Desc: "navigate"},
		)
	}

	// Show select keybinding only if both LIST and READ are supported
	if supportsListOps && supportsReadOps {
		availableBindings = append(availableBindings,
			common.KeyBinding{Key: "<enter>", Desc: "select"},
		)
	}

	// Add delete key binding only if Delete mode is allowed and list operations are supported
	if supportsListOps && bw.isModeAllowed(common.NavigatorModeDelete) {
		availableBindings = append(availableBindings, common.KeyBinding{Key: "<ctrl+d>", Desc: "delete"})
	}

	// Add details key binding only if Details mode is allowed and READ operations are supported
	if supportsReadOps && bw.isModeAllowed(common.NavigatorModeDetails) {
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

	// Collect shortcuts from all extra widgets (sorted by key for consistent display)
	if bw.CanUseExtra() {
		shortcuts := bw.ShortCuts()
		// Extract and sort keys to ensure consistent ordering (1, 2, 3, etc.)
		keys := make([]string, 0, len(shortcuts))
		for key := range shortcuts {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		// Add shortcuts in sorted order
		for _, key := range keys {
			shortcut := shortcuts[key]
			if keyBinding := shortcut.ShortCut(); keyBinding != nil {
				availableBindings = append(availableBindings, *keyBinding)
			}
		}
	}

	bindings := make([]common.KeyBinding, 0, len(availableBindings))

	// Get not allowed keys based on widget type (extra vs main)
	var notAllowedKeys map[string]struct{}
	if bw.isExtra && bw.ExtraWidgetNavigator != nil {
		notAllowedKeys = bw.ExtraWidgetNavigator.NotAllowedListKeys
	} else if bw.WidgetNavigator != nil {
		notAllowedKeys = bw.WidgetNavigator.NotAllowedListKeys
	}

	for _, binding := range availableBindings {
		strippedBinding := strings.Trim(binding.Key, "<>")
		if notAllowedKeys == nil || len(notAllowedKeys) == 0 {
			bindings = append(bindings, binding)
		} else if _, ok := notAllowedKeys[strippedBinding]; !ok {
			bindings = append(bindings, binding)
		}
	}

	return bindings
}

func (bw *BaseWidget) GetCreateKeyBindings() []common.KeyBinding {
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

	// For extra widgets, use ExtraWidgetNavigator; for main widgets, use WidgetNavigator
	var notAllowedKeys map[string]struct{}
	if bw.isExtra && bw.ExtraWidgetNavigator != nil {
		notAllowedKeys = bw.ExtraWidgetNavigator.NotAllowedCreateKeys
	} else if bw.WidgetNavigator != nil {
		notAllowedKeys = bw.WidgetNavigator.NotAllowedCreateKeys
	}

	for _, binding := range availableBindings {
		strippedBinding := strings.Trim(binding.Key, "<>")
		if notAllowedKeys == nil || len(notAllowedKeys) == 0 {
			bindings = append(bindings, binding)
		} else if _, ok := notAllowedKeys[strippedBinding]; !ok {
			bindings = append(bindings, binding)
		}
	}

	return bindings
}

func (bw *BaseWidget) GetDeleteKeyBindings() []common.KeyBinding {
	availableBindings := []common.KeyBinding{
		{Key: "<←/→/tab>", Desc: "toggle"},
		{Key: "<n or esc>", Desc: "cancel"},
		{Key: "<enter>", Desc: "confirm"},
	}

	bindings := make([]common.KeyBinding, 0, len(availableBindings))

	// For extra widgets, use ExtraWidgetNavigator; for main widgets, use WidgetNavigator
	var notAllowedKeys map[string]struct{}
	if bw.isExtra && bw.ExtraWidgetNavigator != nil {
		notAllowedKeys = bw.ExtraWidgetNavigator.NotAllowedDeleteKeys
	} else if bw.WidgetNavigator != nil {
		notAllowedKeys = bw.WidgetNavigator.NotAllowedDeleteKeys
	}

	for _, binding := range availableBindings {
		strippedBinding := strings.Trim(binding.Key, "<>")
		if notAllowedKeys == nil || len(notAllowedKeys) == 0 {
			bindings = append(bindings, binding)
		} else if _, ok := notAllowedKeys[strippedBinding]; !ok {
			bindings = append(bindings, binding)
		}
	}

	return bindings
}

func (bw *BaseWidget) GetPromptKeyBindings() []common.KeyBinding {
	availableBindings := []common.KeyBinding{
		{Key: "<←/→/tab>", Desc: "navigate"},
		{Key: "<y/n>", Desc: "quick select"},
		{Key: "<enter>", Desc: "confirm"},
		{Key: "<esc>", Desc: "cancel"},
	}

	bindings := make([]common.KeyBinding, 0, len(availableBindings))

	// For extra widgets, use ExtraWidgetNavigator; for main widgets, use WidgetNavigator
	// Use delete keys restrictions for prompt mode (same keys)
	var notAllowedKeys map[string]struct{}
	if bw.isExtra && bw.ExtraWidgetNavigator != nil {
		notAllowedKeys = bw.ExtraWidgetNavigator.NotAllowedDeleteKeys
	} else if bw.WidgetNavigator != nil {
		notAllowedKeys = bw.WidgetNavigator.NotAllowedDeleteKeys
	}

	for _, binding := range availableBindings {
		strippedBinding := strings.Trim(binding.Key, "<>")
		if notAllowedKeys == nil || len(notAllowedKeys) == 0 {
			bindings = append(bindings, binding)
		} else if _, ok := notAllowedKeys[strippedBinding]; !ok {
			bindings = append(bindings, binding)
		}
	}

	return bindings
}
func (bw *BaseWidget) GetDetailsKeyBindings() []common.KeyBinding {
	availableBindings := []common.KeyBinding{
		{Key: "</>", Desc: "search", Generic: true},
		{Key: "<↑/↓>", Desc: "scroll"},
		{Key: "<pgup/pgdn>", Desc: "page"},
		{Key: "<ctrl+s>", Desc: "copy to clipboard"},
		{Key: "<ctrl+r>", Desc: "refresh"},
		{Key: "<esc>", Desc: "back"},
	}

	bindings := make([]common.KeyBinding, 0, len(availableBindings))

	// For extra widgets, use ExtraWidgetNavigator; for main widgets, use WidgetNavigator
	var notAllowedKeys map[string]struct{}
	if bw.isExtra && bw.ExtraWidgetNavigator != nil {
		notAllowedKeys = bw.ExtraWidgetNavigator.NotAllowedDetailsKeys
	} else if bw.WidgetNavigator != nil {
		notAllowedKeys = bw.WidgetNavigator.NotAllowedDetailsKeys
	}

	for _, binding := range availableBindings {
		strippedBinding := strings.Trim(binding.Key, "<>")
		if notAllowedKeys == nil || len(notAllowedKeys) == 0 {
			bindings = append(bindings, binding)
		} else if _, ok := notAllowedKeys[strippedBinding]; !ok {
			bindings = append(bindings, binding)
		}
	}

	return bindings
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

// SetResourceless sets whether this extra widget requires parent resources
func (bw *BaseWidget) SetResourceless(resourceless bool) {
	bw.resourceless = resourceless
}

// IsResourceless returns whether this extra widget can work without parent data
func (bw *BaseWidget) IsResourceless() bool {
	return bw.resourceless
}

func (bw *BaseWidget) ShortCuts() map[string]common.ExtraWidget {
	return bw.GetExtraWidget().(*ExtraWidgetGroup).ShortCuts()
}

// GetExtraWidgets returns the list of extra widgets
func (bw *BaseWidget) GetExtraWidgets() []common.ExtraWidget {
	if eg, ok := bw.GetExtraWidget().(*ExtraWidgetGroup); ok {
		// Convert map values to slice
		widgets := make([]common.ExtraWidget, 0, len(eg.GetEntries()))
		for _, widget := range eg.GetEntries() {
			widgets = append(widgets, widget)
		}
		return widgets
	}
	return nil
}

// handleMaybeAsyncTask checks if a record represents an async task and waits for its completion
// This is similar to the terraform provider's handleMaybeAsyncTask function
func (bw *BaseWidget) handleMaybeAsyncTask(rest *vast_client.VMSRest, record vast_client.Record) error {
	if record == nil {
		return nil
	}

	// Use 10 minutes as default timeout for async operations
	timeout := 10 * time.Minute
	ctx := rest.GetCtx()

	asyncResult, err := untyped.MaybeWaitAsyncResultWithContext(ctx, record, rest, timeout)
	if err != nil {
		return err
	}
	if asyncResult != nil && asyncResult.Err != nil {
		return asyncResult.Err
	}
	return nil
}
