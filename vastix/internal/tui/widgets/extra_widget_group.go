package widgets

import (
	"context"
	"log"
	"maps"
	"slices"
	"sort"
	"strings"
	"vastix/internal/database"
	"vastix/internal/logging"
	"vastix/internal/msg_types"
	"vastix/internal/tui/widgets/adapters"
	"vastix/internal/tui/widgets/common"

	vast_client "github.com/vast-data/go-vast-client"

	"go.uber.org/zap"

	tea "github.com/charmbracelet/bubbletea"
)

type ExtraWidgetGroup struct {
	resourceType      string
	entries           map[string]common.ExtraWidget // Map of extra navigators in the group
	activeExtraWidget string

	*common.ExtraWidgetNavigator
	*adapters.DimensionAdapter
	db *database.Service

	// Action Adapters
	*adapters.ListAdapter
	*adapters.CreateAdapter
	*adapters.DetailsAdapter
	*adapters.PromptAdapter
	*adapters.SearchAdapter

	// Loggers
	log    *zap.Logger
	auxlog *log.Logger

	// Selected row data
	selectedRowData common.RowData // Currently selected row data of parent widget
	mainWidget      common.Widget  // Reference to parent widget for validation
}

// NewExtraWidgetGroup creates a new extra widget group.
func NewExtraWidgetGroup(db *database.Service, parentNavigator *common.WidgetNavigator, extraWidgets ...common.ExtraWidget) *ExtraWidgetGroup {
	log := logging.GetGlobalLogger()
	auxlog := logging.GetAuxLogger()

	auxlog.Printf("[ExtraWidgetGroup] NewExtraWidgetGroup called with %d extra widgets", len(extraWidgets))
	log.Debug("Initializing ExtraWidgetGroup")
	defer log.Debug("ExtraWidgetGroup initialized")

	resourceType := "extra actions"
	listHeaders := []string{"method", "action", "summary"}

	entries := make(map[string]common.ExtraWidget)
	// Initialize entries from provided extra widgets
	for i, widget := range extraWidgets {
		extraResourceType := widget.GetExtraResourceType()
		auxlog.Printf("[ExtraWidgetGroup] Adding extra widget %d: type=%s", i, extraResourceType)
		entries[extraResourceType] = widget
	}
	auxlog.Printf("[ExtraWidgetGroup] Total entries created: %d", len(entries))

	extraWidgetGroup := &ExtraWidgetGroup{
		resourceType: resourceType,

		// Adapters
		ListAdapter:      adapters.NewListAdapter(db, resourceType, listHeaders),
		CreateAdapter:    adapters.NewCreateAdapter(db, resourceType),
		DetailsAdapter:   adapters.NewDetailsAdapter(db, resourceType),
		PromptAdapter:    adapters.NewPromptAdapter(db, resourceType),
		DimensionAdapter: &adapters.DimensionAdapter{},
		SearchAdapter:    adapters.NewSearchAdapter(db, resourceType),

		// Database
		db:      db,
		entries: entries,

		// Loggers
		log:    log,
		auxlog: auxlog,
	}

	// Initialize the group's own navigator with parent navigator
	extraWidgetGroup.ExtraWidgetNavigator = common.NewExtraWidgetNavigator(
		extraWidgetGroup, // The navigator controls this group
		parentNavigator,  // Pass the parent navigator
		nil, nil, nil, nil,
	)

	extraWidgetGroup.SetListData()
	return extraWidgetGroup
}

// SetMainWidgetParent updates the parent reference of all individual extra widgets
// to point to the main widget that contains this ExtraWidgetGroup
func (eg *ExtraWidgetGroup) SetMainWidgetParent(mainWidget common.Widget) {
	// Store reference to main widget for validation
	eg.mainWidget = mainWidget

	for _, widget := range eg.entries {
		if baseWidget, ok := widget.(interface{ SetParentForBaseWidget(common.Widget, bool) }); ok {
			baseWidget.SetParentForBaseWidget(mainWidget, true)
		}
	}
}

func (eg *ExtraWidgetGroup) currentExtraWidget() common.ExtraWidget {
	if eg.activeExtraWidget == "" {
		panic("activeExtraWidget is not set")
	}
	return eg.entries[eg.activeExtraWidget]
}

func (eg *ExtraWidgetGroup) GetName() string {
	if eg.activeExtraWidget == "" {
		return ""
	}
	return eg.currentExtraWidget().GetName()
}

// SetActiveWidget sets the active widget by resource type (used when individual widgets call SetExtraMode)
func (eg *ExtraWidgetGroup) SetActiveWidget(resourceType string) {
	if _, ok := eg.entries[resourceType]; ok {
		eg.auxlog.Printf("🎯 SETTING activeExtraWidget: %s -> %s", eg.activeExtraWidget, resourceType)
		eg.activeExtraWidget = resourceType
	} else {
		panic("ExtraWidgetGroup.SetActiveWidget: unknown resource type: " + resourceType)
	}
}

// Helper function to debug available resource types
func (eg *ExtraWidgetGroup) getAvailableResourceTypes() []string {
	var types []string
	for resourceType := range eg.entries {
		types = append(types, resourceType)
	}
	return types
}

func (eg *ExtraWidgetGroup) Init() tea.Msg {
	cmds := make([]tea.Cmd, 0, len(eg.entries))
	for _, entry := range eg.entries {
		cmds = append(cmds, entry.Init)
	}
	return tea.Batch(cmds...)
}

func (eg *ExtraWidgetGroup) CanUseExtra() bool {
	if len(eg.entries) == 0 {
		return false
	}

	// Check if ALL extra widgets are not resourceless (all require resources)
	allNeedResources := true
	for _, widget := range eg.entries {
		// Check if widget has IsResourceless method (works for both BaseWidget and ExtraMethodWidget)
		if resourcelessChecker, ok := widget.(interface{ IsResourceless() bool }); ok {
			if resourcelessChecker.IsResourceless() {
				allNeedResources = false
				break
			}
		}
	}

	// If all widgets need resources, check if parent widget has data
	if allNeedResources && eg.mainWidget != nil {
		if listAdapterWidget, ok := any(eg.mainWidget).(interface{ GetFilteredDataCount() int }); ok {
			filteredCount := listAdapterWidget.GetFilteredDataCount()
			if filteredCount == 0 {
				return false // No data, can't use extra actions
			}
		}
	}

	return true
}

// GetAllExtraWidgets returns all extra widgets in the group
func (eg *ExtraWidgetGroup) GetAllExtraWidgets() map[string]common.ExtraWidget {
	return eg.entries
}

// ShortCut returns nil since ExtraWidgetGroup doesn't have its own shortcut
func (eg *ExtraWidgetGroup) ShortCut() *common.KeyBinding {
	return nil
}

// ShortCuts returns a map of shortcut keys to their corresponding extra widgets
// Filters out shortcuts for widgets that require resources when no data is available
func (eg *ExtraWidgetGroup) ShortCuts() map[string]common.ExtraWidget {
	shortcuts := make(map[string]common.ExtraWidget)

	// Check if parent widget has data
	hasData := true
	if eg.mainWidget != nil {
		if listAdapterWidget, ok := any(eg.mainWidget).(interface{ GetFilteredDataCount() int }); ok {
			filteredCount := listAdapterWidget.GetFilteredDataCount()
			hasData = filteredCount > 0
		}
	}

	// Sort entry keys for deterministic iteration order
	// This ensures shortcuts are always displayed in consistent order (e.g., <1>, <2>, <3>)
	entryKeys := make([]string, 0, len(eg.entries))
	for key := range eg.entries {
		entryKeys = append(entryKeys, key)
	}
	sort.Strings(entryKeys)

	// Iterate in sorted order to ensure consistent shortcut ordering
	for _, entryKey := range entryKeys {
		widget := eg.entries[entryKey]

		// Skip widgets that need resources when no data is available
		// Check if widget has IsResourceless method (works for both BaseWidget and ExtraMethodWidget)
		if resourcelessChecker, ok := widget.(interface{ IsResourceless() bool }); ok {
			if !resourcelessChecker.IsResourceless() && !hasData {
				continue
			}
		}

		if shortcut := widget.ShortCut(); shortcut != nil {
			// Remove angle brackets from key for lookup
			key := strings.Trim(shortcut.Key, "<>")
			shortcuts[key] = widget
		}
	}
	return shortcuts
}

// GetEntries returns the entries map for accessing extra widgets
func (eg *ExtraWidgetGroup) GetEntries() map[string]common.ExtraWidget {
	return eg.entries
}

func (eg *ExtraWidgetGroup) ViewExtra() string {
	// If no extra widget is active, display the list of available actions
	if eg.activeExtraWidget == "" {
		return eg.ViewList()
	}

	mode := eg.GetExtraMode()

	switch mode {
	case common.ExtraNavigatorModeList:
		// Show the ExtraWidgetGroup's own list (list of extra actions)
		return eg.ViewList()
	case common.ExtraNavigatorModeCreate:
		currentWidget := eg.currentExtraWidget()
		return currentWidget.ViewCreateForm()
	case common.ExtraNavigatorModeDetails:
		currentWidget := eg.currentExtraWidget()
		return currentWidget.ViewDetails()
	case common.ExtraNavigatorModeDelete:
		currentWidget := eg.currentExtraWidget()
		return currentWidget.ViewPrompt()
	case common.ExtraNavigatorModePrompt:
		currentWidget := eg.currentExtraWidget()
		return currentWidget.ViewPrompt()
	default:
		panic("unknown ExtraNavigatorMode: " + mode.String())
	}
}

func (eg *ExtraWidgetGroup) GetExtraResourceType() string {
	return eg.currentExtraWidget().GetExtraResourceType()
}

// ResetCreateForm resets the create form state
func (eg *ExtraWidgetGroup) ResetCreateForm() {
	eg.currentExtraWidget().ResetCreateForm()
}

// ExtraNavigate allows ExtraWidgetGroup to satisfy common.ExtraNavigator
func (eg *ExtraWidgetGroup) ExtraNavigate(msg tea.Msg) (tea.Cmd, bool) {
	// The ExtraWidgetGroup handles ALL navigation using its own ExtraWidgetNavigator
	// Individual widgets should NOT handle navigation to avoid recursion
	// The group's navigator knows about the current widget and can handle form inputs properly
	return eg.ExtraWidgetNavigator.ExtraNavigate(msg)
}

func (eg *ExtraWidgetGroup) GetExtraMode() common.ExtraNavigatorMode {
	// ExtraWidgetGroup has limited capacity - when no active widget, always return List mode
	if eg.activeExtraWidget == "" {
		return common.ExtraNavigatorModeList
	}

	mode := eg.currentExtraWidget().GetExtraMode()
	return mode
}

func (eg *ExtraWidgetGroup) SetExtraMode(mode common.ExtraNavigatorMode) {
	// Panic if attempting to set non-List mode when no active widget is selected
	if eg.activeExtraWidget == "" && mode != common.ExtraNavigatorModeList {
		errMsg := "Cannot set ExtraNavigatorMode on ExtraWidgetGroup when no active widget is set. " +
			"ExtraWidgetGroup has limited capacity and must stay in List mode when no widget is active. " +
			"Select a widget first before changing modes."
		eg.auxlog.Println(errMsg)
		panic(errMsg)
	}
	eg.currentExtraWidget().SetExtraMode(mode)

}

func (eg *ExtraWidgetGroup) InitialExtraMode() common.ExtraNavigatorMode {
	// Default to list mode if no extra widget is active
	if eg.activeExtraWidget == "" {
		return common.ExtraNavigatorModeList
	}
	return eg.currentExtraWidget().InitialExtraMode()
}

// ----------------------------
// Dimensions
// ----------------------------

// SetSize overrides DimensionAdapter.SetSize to propagate size changes to adapters
func (eg *ExtraWidgetGroup) SetSize(width, height int) {
	// Call the embedded DimensionAdapter's SetSize
	eg.DimensionAdapter.SetSize(width, height)
	for _, entry := range eg.entries {
		entry.SetSize(width, height)
	}
}

// ----------------------------
// ListWidget methods
// ----------------------------

// SetListDataWithContext overrides BaseWidget to pass context to active extra widget if present
func (eg *ExtraWidgetGroup) SetListDataWithContext(ctx context.Context) tea.Msg {
	if eg.activeExtraWidget == "" {
		// If no active extra widget, use the ListAdapter to set data (no API call needed)
		return eg.SetListData()
	}

	// If there's an active extra widget, try to call SetListDataWithContext on it
	type contextAwareWidget interface {
		SetListDataWithContext(context.Context) tea.Msg
	}
	if ctxWidget, ok := eg.currentExtraWidget().(contextAwareWidget); ok {
		return ctxWidget.SetListDataWithContext(ctx)
	}
	return eg.currentExtraWidget().SetListData()
}

func (eg *ExtraWidgetGroup) SetListData() tea.Msg {
	if eg.activeExtraWidget == "" {
		// If no active extra widget, use the ListAdapter to set data
		// Convert to data format
		data := make([][]string, 0, len(eg.entries))

		// Sort the keys with custom order: GET, POST, PATCH, PUT, DELETE
		// Within each method group, sort by path
		keys := make([]string, 0, len(eg.entries))
		for key := range eg.entries {
			keys = append(keys, key)
		}

		// Custom sort: by method priority, then by path
		sort.Slice(keys, func(i, j int) bool {
			// Extract method and path from keys (format: "METHOD:/path/")
			methodI, pathI := "", ""
			methodJ, pathJ := "", ""

			if idx := strings.Index(keys[i], ":"); idx >= 0 {
				methodI = keys[i][:idx]
				pathI = keys[i][idx+1:]
			}
			if idx := strings.Index(keys[j], ":"); idx >= 0 {
				methodJ = keys[j][:idx]
				pathJ = keys[j][idx+1:]
			}

			// Define method priority: GET=1, POST=2, PATCH=3, PUT=4, DELETE=5
			methodPriority := map[string]int{
				"GET":    1,
				"POST":   2,
				"PATCH":  3,
				"PUT":    4,
				"DELETE": 5,
			}

			priorityI := methodPriority[methodI]
			priorityJ := methodPriority[methodJ]

			// If priorities are different, sort by priority
			if priorityI != priorityJ {
				return priorityI < priorityJ
			}

			// If same priority, sort by path
			return pathI < pathJ
		})

		// Iterate in sorted order
		for _, key := range keys {
			widget := eg.entries[key]
			// Try to get HTTP method, path, and summary from ExtraMethodWidget
			method := "-"
			path := ""
			summary := ""
			if extraMethodWidget, ok := widget.(*ExtraMethodWidget); ok {
				method = extraMethodWidget.GetHTTPMethod()
				path = extraMethodWidget.GetPath()
				summary = extraMethodWidget.GetSummary()
			} else {
				// For custom widgets, use the resource type (which is the key) as the action
				path = widget.GetExtraResourceType()
				// Try to get summary from custom widget if it has a GetSummary method
				if summaryWidget, ok := widget.(interface{ GetSummary() string }); ok {
					summary = summaryWidget.GetSummary()
				}
			}
			data = append(data, []string{method, path, summary})
		}
		eg.ListAdapter.SetListData(data, eg.GetFuzzyListSearchString())
		return msg_types.SetDataMsg{}
	}

	return eg.currentExtraWidget().SetListData()
}

func (eg *ExtraWidgetGroup) ViewList() string {
	// Always show the ExtraWidgetGroup's own list of extra actions
	// Individual extra widgets (like VipPoolForwarding) don't have a list view
	return eg.ListAdapter.ViewList(eg)
}

func (eg *ExtraWidgetGroup) ClearListData() {
	if eg.activeExtraWidget == "" {
		return
	}
	eg.currentExtraWidget().ClearListData()
}

// ----------------------------
// SearchableWidget interface implementation
// ----------------------------

// SetFuzzyListSearchString sets the fuzzy list search query for the widget
func (eg *ExtraWidgetGroup) SetFuzzyListSearchString(query string) {
	if eg.activeExtraWidget == "" {
		// No active extra widget, nothing to set
		return
	}
	eg.currentExtraWidget().SetFuzzyDetailsSearchString(query)
}

// GetFuzzyListSearchString returns the current fuzzy list search query
func (eg *ExtraWidgetGroup) GetFuzzyListSearchString() string {
	if eg.activeExtraWidget == "" {
		return ""
	}
	return eg.currentExtraWidget().GetFuzzyListSearchString()
}

// SetFuzzyDetailsSearchString sets the fuzzy details search query for the widget
func (eg *ExtraWidgetGroup) SetFuzzyDetailsSearchString(query string) {
	if eg.activeExtraWidget == "" {
		// No active extra widget, nothing to set
		return
	}
	eg.currentExtraWidget().SetFuzzyDetailsSearchString(query)
}

// GetFuzzyDetailsSearchString returns the current fuzzy details search query
func (eg *ExtraWidgetGroup) GetFuzzyDetailsSearchString() string {
	if eg.activeExtraWidget == "" {
		return ""
	}
	return eg.currentExtraWidget().GetFuzzyDetailsSearchString()
}

// SetServerSearchParams sets the server-side search parameters string
func (eg *ExtraWidgetGroup) SetServerSearchParams(paramStr string) {
	if eg.activeExtraWidget == "" {
		// No active extra widget, nothing to set
		return
	}
	eg.currentExtraWidget().SetServerSearchParams(paramStr)
}

// GetServerSearchParams returns the current server-side search parameters as string
func (eg *ExtraWidgetGroup) GetServerSearchParams() string {
	if eg.activeExtraWidget == "" {
		return ""
	}
	return eg.currentExtraWidget().GetServerSearchParams()
}

// ClearFuzzyListSearch clears the fuzzy list search
func (eg *ExtraWidgetGroup) ClearFuzzyListSearch() {
	if eg.activeExtraWidget == "" {
		// No active extra widget, nothing to clear
		return
	}
	eg.currentExtraWidget().ClearFuzzyListSearch()
}

// ClearFuzzyDetailsSearch clears the fuzzy details search
func (eg *ExtraWidgetGroup) ClearFuzzyDetailsSearch() {
	if eg.activeExtraWidget == "" {
		// No active extra widget, nothing to clear
		return
	}
	eg.currentExtraWidget().ClearFuzzyDetailsSearch()
}

func (eg *ExtraWidgetGroup) IsServerSearchable() bool {
	if eg.activeExtraWidget == "" {
		return false
	}
	extraMode := eg.GetExtraMode()
	return extraMode == common.ExtraNavigatorModeList
}

func (eg *ExtraWidgetGroup) IsFuzzySearchable() bool {
	if eg.activeExtraWidget == "" {
		return false
	}
	extraMode := eg.GetExtraMode()
	return extraMode == common.ExtraNavigatorModeList || extraMode == common.ExtraNavigatorModeDetails
}

// ClearFilters removes any active filters
func (eg *ExtraWidgetGroup) ClearFilters() {
	if eg.activeExtraWidget == "" {
		// No active extra widget, nothing to clear
		return
	}
	eg.currentExtraWidget().ClearFilters()
}

func (eg *ExtraWidgetGroup) ClearServerSearchParams() {
	if eg.activeExtraWidget == "" {
		// No active extra widget, nothing to clear
		return
	}
	eg.currentExtraWidget().ClearServerSearchParams()
}

func (eg *ExtraWidgetGroup) GetServerParams() *vast_client.Params {
	return eg.currentExtraWidget().GetServerParams()
}

// ----------------------------
// CreateWidget methods
// ----------------------------

func (eg *ExtraWidgetGroup) GetInputs() (common.Inputs, error) {
	// Use dynamic schema-based input generation
	return eg.currentExtraWidget().GetInputs()
}

func (eg *ExtraWidgetGroup) SetInputs(inputs common.Inputs) {
	// Set inputs for the current extra widget
	eg.currentExtraWidget().SetInputs(inputs)

}

// ViewCreateForm renders the profile creation form
func (eg *ExtraWidgetGroup) ViewCreateForm() string {
	// Ensure inputs are set
	return eg.currentExtraWidget().ViewCreateForm()
}

func (eg *ExtraWidgetGroup) ResetCreate() {
	eg.currentExtraWidget().ResetCreateForm()
}

func (eg *ExtraWidgetGroup) CreateFromInputs(inputs common.Inputs) (tea.Cmd, error) {
	return eg.currentExtraWidget().CreateFromInputs(inputs)
}

// ----------------------------
// FormNavigateAdaptor methods
// ----------------------------

// NextInput delegates navigation to the current extra widget
func (eg *ExtraWidgetGroup) NextInput() {
	// Delegate to the current extra widget
	if adapter, ok := any(eg.currentExtraWidget()).(common.FormNavigateAdaptor); ok {
		adapter.NextInput()
	} else {
		// Fallback to embedded CreateAdapter if widget doesn't implement interface
		if eg.CreateAdapter != nil {
			eg.CreateAdapter.NextInput()
		}
	}
}

// PrevInput delegates navigation to the current extra widget
func (eg *ExtraWidgetGroup) PrevInput() {
	// Delegate to the current extra widget
	if adapter, ok := any(eg.currentExtraWidget()).(common.FormNavigateAdaptor); ok {
		adapter.PrevInput()
	} else {
		// Fallback to embedded CreateAdapter if widget doesn't implement interface
		if eg.CreateAdapter != nil {
			eg.CreateAdapter.PrevInput()
		}
	}
}

// UpdateCurrentInput delegates input updates to the current extra widget
func (eg *ExtraWidgetGroup) UpdateCurrentInput(msg tea.Msg) {
	// Delegate to the current extra widget
	if adapter, ok := any(eg.currentExtraWidget()).(common.UpdateInputAdapter); ok {
		adapter.UpdateCurrentInput(msg)
	} else {
		// Fallback to embedded CreateAdapter if widget doesn't implement interface
		if eg.CreateAdapter != nil {
			eg.CreateAdapter.UpdateCurrentInput(msg)
		}
	}
}

// CreateFromInputsDo delegates create action to the current extra widget
func (eg *ExtraWidgetGroup) CreateFromInputsDo(widget common.CreateWidget) tea.Cmd {
	// Delegate to the current extra widget
	if adapter, ok := any(eg.currentExtraWidget()).(common.CreateFromInputsAdapter); ok {
		return adapter.CreateFromInputsDo(widget)
	} else {
		// Fallback to embedded CreateAdapter if widget doesn't implement interface
		if eg.CreateAdapter != nil {
			return eg.CreateAdapter.CreateFromInputsDo(widget)
		}
		return nil
	}
}

// HasInputs delegates to the current extra widget
func (eg *ExtraWidgetGroup) HasInputs() bool {
	if eg.activeExtraWidget == "" {
		if eg.CreateAdapter != nil {
			return eg.CreateAdapter.HasInputs()
		}
		return false
	}

	// Delegate to the current extra widget
	if adapter, ok := any(eg.currentExtraWidget()).(common.CreateWidget); ok {
		result := adapter.HasInputs()
		return result
	} else {
		// Fallback to embedded CreateAdapter if widget doesn't implement interface
		if eg.CreateAdapter != nil {
			return eg.CreateAdapter.HasInputs()
		}
		return false
	}
}

// ----------------------------
// DetailsWidget methods
// ----------------------------

func (eg *ExtraWidgetGroup) Details(selectedRowData common.RowData) (tea.Cmd, error) {
	currentWidget := eg.currentExtraWidget()
	return currentWidget.Details(selectedRowData)
}

func (eg *ExtraWidgetGroup) Select(selectedRowData common.RowData) (tea.Cmd, error) {
	httpMethod := selectedRowData.GetString("method")
	actionPath := selectedRowData.GetString("action")

	// Try to find the widget by matching the action
	var uniqueKey string
	var found bool

	// First, try direct lookup (for custom widgets, the action name IS the key)
	if _, ok := eg.entries[actionPath]; ok {
		uniqueKey = actionPath
		found = true
		eg.auxlog.Printf("Found widget by direct action lookup: %s", uniqueKey)
	} else {
		// For auto-generated widgets, try METHOD:PATH format
		if httpMethod != "-" && httpMethod != "" {
			testKey := httpMethod + ":" + actionPath
			if _, ok := eg.entries[testKey]; ok {
				uniqueKey = testKey
				found = true
				eg.auxlog.Printf("Found widget by METHOD:PATH lookup: %s", uniqueKey)
			}
		}
	}

	if !found {
		availableActions := slices.Collect(maps.Keys(eg.entries))
		panic(
			"ExtraWidgetGroup.Select: selected action '" +
				actionPath +
				"' not found in entries. Available actions: " +
				strings.Join(availableActions, ", "),
		)
	}

	eg.SetActiveWidget(uniqueKey)
	// Initialize the selected widget to its initial mode
	currentWidget := eg.currentExtraWidget()
	initialMode := currentWidget.InitialExtraMode()

	currentWidget.SetSelectedRowData(eg.selectedRowData)

	eg.SetExtraMode(initialMode) // Set mode on group, not individual widget

	// If we just switched to create mode, initialize inputs now that activeExtraWidget is set
	if initialMode == common.ExtraNavigatorModeCreate {
		eg.auxlog.Printf("Post-select: Initializing inputs for create mode")
		if inputsWidget, ok := currentWidget.(interface{ GetInputs() (common.Inputs, error) }); ok {
			if createWidget, ok := currentWidget.(interface{ SetInputs(common.Inputs) }); ok {
				inputs, err := inputsWidget.GetInputs()
				if err != nil {
					eg.auxlog.Printf("Failed to get inputs after select: %v", err)
				} else {
					createWidget.SetInputs(inputs)
					// Note: Don't reset the form here as it would clear pre-populated values from parent row data
					eg.auxlog.Printf("Post-select: Extra create inputs initialized with pre-populated values")
				}
			}
		}
	}

	return msg_types.ProcessWithSpinner(func() tea.Msg {
		return nil
	}), nil
}

func (eg *ExtraWidgetGroup) SetDetailsData(details any) {
	eg.currentExtraWidget().SetDetailsData(details)
}

func (eg *ExtraWidgetGroup) GetSelectedRowData() common.RowData {
	return eg.currentExtraWidget().GetSelectedRowData()
}

func (eg *ExtraWidgetGroup) SetSelectedRowData(data common.RowData) {
	if eg.activeExtraWidget == "" {
		eg.selectedRowData = data
	} else {
		eg.currentExtraWidget().SetSelectedRowData(data)
	}
}

// ToggleFormJSONMode delegates to the current extra widget's form/JSON toggle
func (eg *ExtraWidgetGroup) ToggleFormJSONMode() {
	if eg.activeExtraWidget != "" {
		if toggleWidget, ok := eg.currentExtraWidget().(interface{ ToggleFormJSONMode() }); ok {
			toggleWidget.ToggleFormJSONMode()
		}
	}
}

// IsEditingJSON delegates to the current extra widget's JSON editing state
func (eg *ExtraWidgetGroup) IsEditingJSON() bool {
	if eg.activeExtraWidget != "" {
		if jsonWidget, ok := eg.currentExtraWidget().(interface{ IsEditingJSON() bool }); ok {
			return jsonWidget.IsEditingJSON()
		}
	}
	return false
}

// UpdateJSONTextarea delegates to the current extra widget's JSON textarea update
func (eg *ExtraWidgetGroup) UpdateJSONTextarea(msg tea.Msg) tea.Cmd {
	if eg.activeExtraWidget != "" {
		if jsonWidget, ok := eg.currentExtraWidget().(interface{ UpdateJSONTextarea(tea.Msg) tea.Cmd }); ok {
			return jsonWidget.UpdateJSONTextarea(msg)
		}
	}
	return nil
}

// SaveJSONEdits delegates to the current extra widget's JSON save
func (eg *ExtraWidgetGroup) SaveJSONEdits() error {
	if eg.activeExtraWidget != "" {
		if jsonWidget, ok := eg.currentExtraWidget().(interface{ SaveJSONEdits() error }); ok {
			return jsonWidget.SaveJSONEdits()
		}
	}
	return nil
}

// HasActiveWidget checks if there's an active extra widget
func (eg *ExtraWidgetGroup) HasActiveWidget() bool {
	return eg.activeExtraWidget != ""
}

func (eg *ExtraWidgetGroup) Reset() {
	// Reset all contained extra widgets
	for _, widget := range eg.entries {
		widget.Reset()
	}
	// Reset the group's own state
	eg.activeExtraWidget = ""
	eg.CreateAdapter.ResetCreateForm()
}

// Clean performs cleanup when the widget group is being closed
// Delegates to LeaveWidget if the active extra widget implements it
func (eg *ExtraWidgetGroup) Clean() {
	// Call LeaveWidget which already handles the delegation logic
	if err := eg.LeaveWidget(); err != nil {
		eg.auxlog.Printf("Error during ExtraWidgetGroup cleanup: %v", err)
	}
	// Reset all widgets
	eg.Reset()
}

// LeaveWidget implements the LeaveWidget interface
// Delegates to the active extra widget if it implements LeaveWidget
func (eg *ExtraWidgetGroup) LeaveWidget() error {
	if eg.activeExtraWidget == "" {
		eg.auxlog.Println("ExtraWidgetGroup.LeaveWidget: no active widget")
		return nil
	}

	currentWidget := eg.currentExtraWidget()
	eg.auxlog.Printf("ExtraWidgetGroup.LeaveWidget: delegating to %s", eg.activeExtraWidget)

	// Check if the active widget implements LeaveWidget
	if leaveWidget, ok := currentWidget.(common.LeaveWidget); ok {
		eg.auxlog.Printf("Active widget %s implements LeaveWidget, calling it", eg.activeExtraWidget)
		return leaveWidget.LeaveWidget()
	}

	eg.auxlog.Printf("Active widget %s does NOT implement LeaveWidget", eg.activeExtraWidget)
	return nil
}

func (eg *ExtraWidgetGroup) ViewDetails() string {
	return eg.currentExtraWidget().ViewDetails()
}

// UpdateViewPort delegates to the DetailsAdapter for viewport navigation (scrolling)
func (eg *ExtraWidgetGroup) UpdateViewPort(msg tea.Msg) tea.Cmd {
	if eg.activeExtraWidget == "" {
		panic("UpdateViewPort called on ExtraWidgetGroup with no active extra widget")
	}

	currentWidget := eg.currentExtraWidget()
	// Pass other keys (arrows, pgup/pgdn, etc.) to the details adapter for scrolling
	if adapter, ok := any(currentWidget).(common.ViewPortAdapter); ok {
		return adapter.UpdateViewPort(msg)
	} else {
		panic("ExtraWidget: widget does not implement ViewPortAdapter interface")
	}
}

// ----------------------------
// DeleteWidget methods
// ----------------------------

// Delete implements the DeleteWidget interface for view deletion
func (eg *ExtraWidgetGroup) Delete(selectedRowData common.RowData) (tea.Cmd, error) {
	return eg.currentExtraWidget().Delete(selectedRowData)
}

// ----------------------------
// PromptWidget methods
// ----------------------------

func (eg *ExtraWidgetGroup) ViewPrompt() string {
	return eg.currentExtraWidget().ViewPrompt()
}

// ------------------------------
// KeyBindings methods
// ------------------------------

func (eg *ExtraWidgetGroup) GetKeyBindings() []common.KeyBinding {
	var keyBindings []common.KeyBinding

	// Check if there's an active widget set first (before calling currentExtraWidget which panics)
	hasActiveWidget := eg.activeExtraWidget != ""
	var activeWidget common.ExtraWidget
	if hasActiveWidget {
		activeWidget = eg.currentExtraWidget()
	}

	switch eg.GetExtraMode() {
	case common.ExtraNavigatorModeList:
		if hasActiveWidget && activeWidget != nil {
			keyBindings = activeWidget.GetListKeyBindings()
		} else {
			keyBindings = eg.GetListKeyBindings()
		}
	case common.ExtraNavigatorModeCreate:
		if hasActiveWidget && activeWidget != nil {
			keyBindings = activeWidget.GetCreateKeyBindings()
		} else {
			keyBindings = eg.GetCreateKeyBindings()
		}
	case common.ExtraNavigatorModeDelete:
		if hasActiveWidget && activeWidget != nil {
			keyBindings = activeWidget.GetDeleteKeyBindings()
		} else {
			keyBindings = eg.GetDeleteKeyBindings()
		}
	case common.ExtraNavigatorModeDetails:
		// Delegate to active widget for customized details key bindings
		// This allows widgets like VipPoolForwarding to exclude <ctrl+e>
		if hasActiveWidget && activeWidget != nil {
			keyBindings = activeWidget.GetDetailsKeyBindings()
		} else {
			keyBindings = eg.GetDetailsKeyBindings()
		}
	case common.ExtraNavigatorModePrompt:
		// Prompt mode doesn't have a method in KeyBindingGetter interface
		// Always use the group's method
		keyBindings = eg.GetPromptKeyBindings()
	}

	return keyBindings
}

func (eg *ExtraWidgetGroup) GetListKeyBindings() []common.KeyBinding {
	return []common.KeyBinding{}
}
func (eg *ExtraWidgetGroup) GetCreateKeyBindings() []common.KeyBinding {
	availableBindings := []common.KeyBinding{
		{Key: "<tab>", Desc: "next input"},
		{Key: "<shift+tab>", Desc: "previous input"},
		{Key: "<ctrl+s>", Desc: "submit"},
		{Key: "<ctrl+t>", Desc: "toggle form/json"},
		{Key: "<esc>", Desc: "back"},
		{Key: "<space>", Desc: "toggle boolean"},
		{Key: "<,>", Desc: "array delimiter"},
	}

	bindings := make([]common.KeyBinding, 0, len(availableBindings))
	notAllowedKeys := eg.ExtraWidgetNavigator.NotAllowedCreateKeys

	for _, binding := range availableBindings {
		strippedBinding := strings.Trim(binding.Key, "<>")
		if _, ok := notAllowedKeys[strippedBinding]; !ok {
			bindings = append(bindings, binding)
		}
	}
	return bindings
}
func (eg *ExtraWidgetGroup) GetDeleteKeyBindings() []common.KeyBinding {
	availableBindings := []common.KeyBinding{
		{Key: "<y or enter>", Desc: "confirm"},
		{Key: "<n or esc>", Desc: "cancel"},
	}

	bindings := make([]common.KeyBinding, 0, len(availableBindings))
	notAllowedKeys := eg.ExtraWidgetNavigator.NotAllowedDeleteKeys

	for _, binding := range availableBindings {
		strippedBinding := strings.Trim(binding.Key, "<>")
		if _, ok := notAllowedKeys[strippedBinding]; !ok {
			bindings = append(bindings, binding)
		}
	}
	return bindings
}

func (eg *ExtraWidgetGroup) GetPromptKeyBindings() []common.KeyBinding {
	return []common.KeyBinding{
		{Key: "<←/→/tab>", Desc: "navigate"},
		{Key: "<y/n>", Desc: "quick select"},
		{Key: "<enter>", Desc: "confirm"},
		{Key: "<esc>", Desc: "cancel"},
	}
}
func (eg *ExtraWidgetGroup) GetDetailsKeyBindings() []common.KeyBinding {
	availableBindings := []common.KeyBinding{
		{Key: "</>", Desc: "search", Generic: true},
		{Key: "<↑/↓>", Desc: "scroll"},
		{Key: "<pgup/pgdn>", Desc: "page"},
		{Key: "<ctrl+s>", Desc: "copy to clipboard"},
		{Key: "<ctrl+e>", Desc: "edit & resubmit"},
		{Key: "<esc>", Desc: "back"},
	}

	bindings := make([]common.KeyBinding, 0, len(availableBindings))
	notAllowedKeys := eg.ExtraWidgetNavigator.NotAllowedDetailsKeys

	for _, binding := range availableBindings {
		strippedBinding := strings.Trim(binding.Key, "<>")
		if _, ok := notAllowedKeys[strippedBinding]; !ok {
			bindings = append(bindings, binding)
		}
	}
	return bindings
}

// ------------------------------
// Prompt adapter methods - delegate to current extra widget
// ------------------------------

// TogglePromptSelection delegates to the active extra widget's prompt adapter
func (eg *ExtraWidgetGroup) TogglePromptSelection() {
	if eg.activeExtraWidget == "" {
		return
	}
	if widget, ok := eg.entries[eg.activeExtraWidget].(common.PromptToggleAdapter); ok {
		widget.TogglePromptSelection()
	}
}

// IsPromptNoSelected delegates to the active extra widget's prompt adapter
func (eg *ExtraWidgetGroup) IsPromptNoSelected() bool {
	if eg.activeExtraWidget == "" {
		return false
	}
	if widget, ok := eg.entries[eg.activeExtraWidget].(common.PromptSelectionAdapter); ok {
		return widget.IsPromptNoSelected()
	}
	return false
}

// SetPromptSelection delegates to the active extra widget's prompt adapter
func (eg *ExtraWidgetGroup) SetPromptSelection(selectNo bool) {
	if eg.activeExtraWidget == "" {
		return
	}
	if widget, ok := eg.entries[eg.activeExtraWidget].(common.PromptSelectionSetAdapter); ok {
		widget.SetPromptSelection(selectNo)
	}
}

func (eg *ExtraWidgetGroup) GetAllowedExtraNavigatorModes() []common.ExtraNavigatorMode {
	if eg.activeExtraWidget != "" {
		return eg.currentExtraWidget().GetAllowedExtraNavigatorModes()
	}
	// ExtraWidgetGroup itself only supports list mode
	return []common.ExtraNavigatorMode{
		common.ExtraNavigatorModeList,
	}
}

// Mimic Widget interface methods for passing to Adapters

func (eg *ExtraWidgetGroup) View() string {
	panic("View should not be called on ExtraWidgetGroup")
}

func (eg *ExtraWidgetGroup) GetMode() common.NavigatorMode {
	panic("GetMode should not be called on ExtraWidgetGroup")
}

func (eg *ExtraWidgetGroup) SetMode(mode common.NavigatorMode) {
	panic("SetMode should not be called on ExtraWidgetGroup")
}

func (eg *ExtraWidgetGroup) GetWidget() common.Widget {
	panic("GetWidget should not be called on ExtraWidgetGroup")
}

func (eg *ExtraWidgetGroup) SetWidget(widget common.Widget) {
	panic("SetWidget should not be called on ExtraWidgetGroup")
}

func (eg *ExtraWidgetGroup) Navigate(msg tea.Msg) tea.Cmd {
	panic("Navigate should not be called on ExtraWidgetGroup")
}

func (eg *ExtraWidgetGroup) GetAllowedNavigatorModes() []common.NavigatorMode {
	panic("GetAllowedNavigatorModes should not be called on ExtraWidgetGroup")
}

func (eg *ExtraWidgetGroup) GetNotAllowedNavigatorModes() []common.NavigatorMode {
	panic("GetNotAllowedNavigatorModes should not be called on ExtraWidgetGroup")
}

func (eg *ExtraWidgetGroup) GetResourceType() string {
	return eg.resourceType
}

func (eg *ExtraWidgetGroup) InitialMode() common.NavigatorMode {
	panic("InitialMode should not be called on ExtraWidgetGroup")
}
