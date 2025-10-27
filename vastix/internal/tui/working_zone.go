package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"vastix/internal/database"
	log "vastix/internal/logging"
	"vastix/internal/msg_types"
	"vastix/internal/tui/widgets"
	"vastix/internal/tui/widgets/common"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vast-data/go-vast-client/rest"
	"go.uber.org/zap"
)

// Initialize global spinner accessor for the common package
func init() {
	common.SetGlobalSpinnerAccessor(func() bool {
		return GetGlobalSpinnerState().IsActive()
	})
}

// WorkingZone represents the main working area
type WorkingZone struct {
	width, height int
	resourceType  string
	db            *database.Service // Database service

	// Widget management
	widgets       map[string]common.Widget
	currentWidget common.Widget

	// Spinner display (moved from status zone)
	spinnerMsg string

	// Error handling
	errorHandler func(string) // Function to display errors
	clearErrors  func()       // Function to clear errors

	// Ticker control
	enableTickers  func() // Function to enable tickers
	disableTickers func() // Function to disable tickers

	// setResourceCallbacks
	BeforeSetResourceCb func()
	AfterSetResourceCb  func()

	// Throttling fields for SetListDataThrottled
	lastSetDataTime   time.Time
	lastSetDataWidget common.Widget
}

func NewWorkingZone(db *database.Service, errorHandler func(string), clearErrors func(), enableTickers func(), disableTickers func(), restClient interface{}) *WorkingZone {
	log.Debug("WorkingZone initializing")

	// Note: REST client may be nil at startup - widgets will be initialized later
	// when InitializeAPIWidgets() is called after profile connection
	if restClient != nil {
		if untypedRest, ok := restClient.(*rest.UntypedVMSRest); ok {
			factory, err := widgets.InitializeWidgets(db, untypedRest)
			if err != nil {
				log.Error("Failed to initialize widgets", zap.Error(err))
			} else {
				log.Info("Widgets initialized successfully",
					zap.Int("widget_count", len(factory.GetSupportedResources())))
			}
		}
	} else {
		log.Info("REST client not available yet, widgets will be initialized after profile connection")
	}

	// Initialize Resources widget first - it manages all other widgets
	resourcesWidget := widgets.NewResources(db).(*widgets.Resources)

	// Get all widgets from Resources widget (includes Profile + any generated widgets)
	registeredWidgets := resourcesWidget.GetAllWidgets()

	log.Debug("All widgets initialized from Resources widget",
		zap.Int("widget_count", len(registeredWidgets)))

	// Get current resource from database history, fallback to "profiles" if none exists
	auxlog := log.GetAuxLogger()
	currentResourceType, err := db.GetCurrentResource()
	if err != nil {
		auxlog.Printf("Failed to get current resource from database: %v, using default 'profiles'", err)
		log.Debug("Failed to get current resource from database, using default", zap.Error(err))
		currentResourceType = "profiles"
	} else {
		auxlog.Printf("Read from database: current resource = '%s'", currentResourceType)
	}

	// Initialize resource history if it doesn't exist
	if err := db.InitializeResourceHistory(currentResourceType); err != nil {
		auxlog.Printf("Failed to initialize resource history: %v", err)
		log.Debug("Failed to initialize resource history", zap.Error(err))
	} else {
		auxlog.Printf("Resource history initialized with: '%s'", currentResourceType)
	}

	navigatorMode := common.NavigatorModeList // Default mode

	// Fix any existing data corruption where multiple profiles are active
	if err := db.EnsureSingleActiveProfile(); err != nil {
		log.Error("Failed to ensure single active profile", zap.Error(err))
		// Don't panic here, just log the error and continue
	}

	// Check if there are active profiles - if not, force profile creation
	hasActive, err := db.HasActiveProfile()
	if err != nil {
		panic(err)
	}

	if !hasActive {
		log.Warn("No active profiles found, entering forced create mode",
			zap.Error(err),
			zap.Bool("has_active", hasActive))

		// Store current resource as previous and switch to profiles
		if err := db.SetResourceHistory("profiles", currentResourceType); err != nil {
			log.Debug("Failed to update resource history for profile creation", zap.Error(err))
		}

		currentResourceType = "profiles"
		navigatorMode = common.NavigatorModeCreate

	} else {
		log.Debug("Active profiles found, using normal mode",
			zap.String("currentResource", currentResourceType))
	}

	// Check if the current resource type has a registered widget, fallback to "profiles" if not
	currentWidget := registeredWidgets[currentResourceType]
	if currentWidget == nil {
		auxlog.Printf("Widget '%s' not found yet (may be API-generated), temporarily using 'profiles'", currentResourceType)
		log.Warn("Current resource type not found in registered widgets, temporarily using profiles for display",
			zap.String("savedResourceType", currentResourceType))

		// Use profiles widget for display, but DON'T overwrite the database
		// The saved widget name in the database is still valid, it just hasn't been loaded yet
		// When API widgets are loaded, we'll restore to the correct widget
		currentWidget = registeredWidgets["profiles"]

		// NOTE: We do NOT call SetResourceHistory here because:
		// 1. The widget might be API-generated and will be loaded soon after profile connects
		// 2. We want to preserve the user's last selected widget in the database
		// 3. InitializeAPIWidgets will restore to the saved widget when it becomes available
		auxlog.Printf("Database still has '%s' saved, will restore after API widgets load", currentResourceType)

		// Temporarily display profiles, but the saved resource name stays in database for restoration
		currentResourceType = "profiles"
	}

	// Final safety check
	if currentWidget == nil {
		panic("Failed to initialize any widget - both currentResourceType and fallback failed")
	}

	workingZone := &WorkingZone{
		resourceType:   currentResourceType,
		db:             db,
		widgets:        registeredWidgets,
		currentWidget:  currentWidget,
		errorHandler:   errorHandler,
		clearErrors:    clearErrors,
		enableTickers:  enableTickers,
		disableTickers: disableTickers,
	}

	workingZone.setNavigatorMode(navigatorMode)

	log.Debug("WorkingZone initialized successfully",
		zap.String("resource_type", currentResourceType),
		zap.Any("navigator_mode", navigatorMode),
	)

	return workingZone
}

func (w *WorkingZone) Init() {
	for _, widget := range w.widgets {
		widget.Init()
	}
}

// InitializeAPIWidgets initializes widgets from API after profile connection
func (w *WorkingZone) InitializeAPIWidgets(restClient *rest.UntypedVMSRest) error {
	if restClient == nil {
		return fmt.Errorf("rest client is nil")
	}

	log.Info("Initializing API widgets after profile connection")

	factory, err := widgets.InitializeWidgets(w.db, restClient)
	if err != nil {
		log.Error("Failed to initialize API widgets", zap.Error(err))
		return err
	}

	log.Info("API widgets initialized successfully",
		zap.Int("widget_count", len(factory.GetSupportedResources())))

	// Get the Resources widget and refresh all widgets from it
	if resourcesWidget, ok := w.widgets["resources"].(*widgets.Resources); ok {
		// Get all widgets (including newly generated ones)
		allWidgets := resourcesWidget.GetAllWidgets()

		// Update the widgets map with any new widgets
		for name, widget := range allWidgets {
			if _, exists := w.widgets[name]; !exists {
				w.widgets[name] = widget
				widget.SetSize(w.width, w.height)
				log.Debug("Registered new widget", zap.String("resource", name))
			}
		}

		// Refresh the Resources widget to show the new widgets
		resourcesWidget.SetListData()
		log.Info("Resources widget refreshed with new API widgets")
	}

	log.Info("All API widgets registered", zap.Int("total_widgets", len(w.widgets)))

	// Now try to restore to the saved widget from database if it wasn't available during init
	auxlog := log.GetAuxLogger()
	savedResource, err := w.db.GetCurrentResource()
	if err == nil && savedResource != "" && savedResource != w.resourceType {
		// Check if the saved widget now exists
		if savedWidget, exists := w.widgets[savedResource]; exists {
			auxlog.Printf("Restoring to saved widget '%s' (now available after API load)", savedResource)
			w.currentWidget = savedWidget
			w.resourceType = savedResource
			w.currentWidget.SetSize(w.width, w.height)
			log.Info("Restored to saved widget from database",
				zap.String("widget", savedResource))
		} else {
			auxlog.Printf("Saved widget '%s' still not available, staying on current widget", savedResource)
		}
	}

	return nil
}

// SetSize sets the dimensions of the working zone
func (w *WorkingZone) SetSize(width, height int) {
	w.width = width
	w.height = height

	// Set size for all widget list renderers
	for _, widget := range w.widgets {
		widget.SetSize(width, height)
	}
}

// Update handles messages for the working zone
func (w *WorkingZone) Update(msg tea.Msg) (*WorkingZone, tea.Cmd) {
	log.Debug("WorkingZone Update called", zap.Any("msg_type", msg))

	// Disable all interactions when spinner is active
	if GetGlobalSpinnerState().IsActive() {
		log.Debug("WorkingZone: ignoring input - spinner active")
		return w, nil
	}

	if w.currentWidget == nil {
		log.Error("WorkingZone: currentWidget is nil - cannot process navigation")
		return w, nil
	}

	log.Debug("WorkingZone: calling currentWidget.Navigate", zap.String("widget_type", w.currentWidget.GetResourceType()))
	return w, w.currentWidget.Navigate(msg)
}

// View renders the working zone
func (w *WorkingZone) View() string {
	if w.width == 0 || w.height == 0 {
		return ""
	}

	if w.HasSpinner() {
		// When spinner is active, show only the spinner
		return w.renderSpinnerOnly()
	}

	if w.currentWidget == nil {
		return "ERROR: No widget available for current resource"
	}

	return w.currentWidget.View()
}

func (w *WorkingZone) setNavigatorMode(mode common.NavigatorMode) {
	auxlog := log.GetAuxLogger()

	// Check if currentWidget is nil to prevent panic
	if w.currentWidget == nil {
		log.Error("Cannot set navigator mode: currentWidget is nil")
		return
	}

	auxlog.Printf("setNavigatorMode: setting mode to %v for widget %T", mode, w.currentWidget)

	// Set the navigator mode for the current widget
	w.currentWidget.SetMode(mode)

	// Control tickers based on navigator mode
	// Tickers are enabled only in NavigatorModeList to provide real-time data updates
	// when browsing lists. They are disabled in other modes (create, delete, details, extra)
	// to avoid interference and unnecessary processing during focused operations.
	if mode == common.NavigatorModeList {
		// Enable tickers only in list mode
		auxlog.Println("setNavigatorMode: enabling tickers (list mode)")
		if w.enableTickers != nil {
			w.enableTickers()
		}
		// Get the widget navigator and reset its creation.
		w.currentWidget.ResetCreateForm()
	} else {
		// Disable tickers for all other modes (create, delete, details, extra)
		auxlog.Printf("setNavigatorMode: disabling tickers (mode: %v)", mode)
		if w.disableTickers != nil {
			w.disableTickers()
		}
	}
}

func (w *WorkingZone) ResetAllWidgets() tea.Msg {
	for _, widget := range w.widgets {
		widget.Reset()
	}
	return nil
}

func (w *WorkingZone) SetListNavigatorMode() {
	w.setNavigatorMode(common.NavigatorModeList)
}

// SetResourceType changes the current resource type and updates the current widget
func (w *WorkingZone) SetResourceType(resourceType string) {
	// Get current resource from database before switching for history tracking
	auxlog := log.GetAuxLogger()
	currentFromDB, err := w.db.GetCurrentResource()
	if err != nil {
		log.Debug("Failed to get current resource from database", zap.Error(err))
		currentFromDB = w.resourceType // Fallback to in-memory value
	}

	auxlog.Printf("Switching resource type from %s to %s", currentFromDB, resourceType)
	// Don't update history if we're not actually changing
	if currentFromDB == resourceType {
		log.Debug("Resource type unchanged, skipping history update")
		return
	}

	// Call before callback if set
	if w.BeforeSetResourceCb != nil {
		w.BeforeSetResourceCb()
	}

	// Update resource type
	w.resourceType = resourceType

	// Check if the widget exists before setting it as current
	if widget, exists := w.widgets[resourceType]; exists {
		w.currentWidget = widget
		w.currentWidget.SetSize(w.width, w.height)
		// Update resource history in database: current becomes previous, new becomes current
		auxlog.Printf("Updating database: current='%s' -> previous='%s'", resourceType, currentFromDB)
		if err := w.db.SetResourceHistory(resourceType, currentFromDB); err != nil {
			auxlog.Printf("Failed to update resource history: %v", err)
			log.Debug("Failed to update resource history in database",
				zap.Error(err),
				zap.String("new_current", resourceType),
				zap.String("new_previous", currentFromDB))
		} else {
			auxlog.Printf("Resource history saved to database: current='%s', previous='%s'", resourceType, currentFromDB)
			log.Debug("Resource history updated successfully",
				zap.String("new_current", resourceType),
				zap.String("new_previous", currentFromDB))
		}
	} else {
		// Handle the case where widget doesn't exist - could log an error or fallback to a default
		log.Error("Widget not found for resource type", zap.String("resourceType", resourceType))
		return
	}

	// Call after callback if set
	if w.AfterSetResourceCb != nil {
		w.AfterSetResourceCb()
	}

}

// GetCurrentWidget returns the current widget
func (w *WorkingZone) GetCurrentWidget() common.Widget {
	return w.currentWidget
}

// Ready returns whether the working zone is ready to be displayed
func (w *WorkingZone) Ready() bool {
	// Working zone is ready when it has widgets and current widget is set
	return w.widgets != nil && w.currentWidget != nil
}

// SetSpinner sets the spinner message to display in the working zone
func (w *WorkingZone) SetSpinner(msg string) {
	w.spinnerMsg = msg
}

// ClearSpinner clears the spinner message from the working zone
func (w *WorkingZone) ClearSpinner() {
	w.spinnerMsg = ""
}

// HasSpinner returns true if there's a spinner message to display
func (w *WorkingZone) HasSpinner() bool {
	return w.spinnerMsg != ""
}

func (w *WorkingZone) SetListData() tea.Msg {
	if w.currentWidget == nil {
		log.Error("SetListData: currentWidget is nil")
		return msg_types.MockMsg{}
	}

	// Call the widget's SetListData method
	result := w.currentWidget.SetListData()

	// Update throttling fields after successful execution
	w.lastSetDataTime = time.Now()
	w.lastSetDataWidget = w.currentWidget

	return result
}

// SetListDataThrottled calls SetListData only if:
// - The current widget has changed since the last call, OR
// - More than 1 minute has passed since the last call for the same widget
func (w *WorkingZone) SetListDataThrottled() tea.Msg {
	auxlog := log.GetAuxLogger()
	now := time.Now()

	// Check if widget has changed
	widgetChanged := w.lastSetDataWidget != w.currentWidget

	// Check if more than the throttle interval has passed since last call for the same widget
	timeSinceLastCall := now.Sub(w.lastSetDataTime)
	timePassed := timeSinceLastCall >= ThrottleInterval

	// Get widget type for logging
	var currentWidgetType string
	if w.currentWidget != nil {
		currentWidgetType = fmt.Sprintf("%T", w.currentWidget)
	} else {
		currentWidgetType = "<nil>"
	}

	// Call SetListData if widget changed or enough time has passed
	if widgetChanged || timePassed {
		// Call SetListData - it will update the throttling fields after execution
		return w.SetListData()
	}

	// Throttled - return nil (no-op)
	auxlog.Printf("SetListDataThrottled: throttled for widget %s (time since last: %v)", currentWidgetType, timeSinceLastCall)
	return nil
}

func (w *WorkingZone) SetDetailsData(details any) {
	w.currentWidget.SetDetailsData(details)
}

func (w *WorkingZone) GetWidgetBindings() []common.KeyBinding {
	if w.currentWidget == nil {
		return []common.KeyBinding{}
	}
	return w.currentWidget.GetKeyBindings()
}

// renderSpinnerOnly renders empty content with spinner in top-left, will get bordered
func (w *WorkingZone) renderSpinnerOnly() string {
	if w.spinnerMsg == "" {
		return ""
	}

	// Create empty content area of the same size as normal content
	// The border will be added by the widget rendering system (and will be gray due to global state)

	// Calculate inner content dimensions (accounting for borders that will be added)
	innerWidth := w.width - 2       // Account for left and right borders
	innerHeight := w.height - 3 + 1 // Account for top, bottom borders, and title area + 1 line to match normal content

	if innerWidth < 1 {
		innerWidth = w.width
	}
	if innerHeight < 1 {
		innerHeight = 5 // Minimum height
	}

	// Create empty lines
	lines := make([]string, innerHeight)
	for i := range lines {
		lines[i] = strings.Repeat(" ", innerWidth)
	}

	// Put spinner in bottom-left corner with same margins as splash screen
	leftMargin := "    " // Same as splash screen (4 spaces)
	if len(lines) > 2 {  // Make sure we have enough lines for bottom positioning
		// Position spinner near the bottom (leave 1 line for bottom margin)
		bottomLineIndex := len(lines) - 2

		// Spinner already has brackets built-in now
		spinnerWithMargin := leftMargin + w.spinnerMsg

		// Calculate padding (spinner msg already includes brackets and spaces)
		// Note: getSpinnerDisplayWidth strips ANSI codes but doesn't account for brackets
		// Since spinner now includes brackets, we need to estimate total visual width
		cleanSpinnerWidth := w.getSpinnerDisplayWidth()
		totalVisualWidth := cleanSpinnerWidth + 6 // +6 for [  ] brackets with spaces
		paddingNeeded := common.Max(0, innerWidth-len(leftMargin)-totalVisualWidth)
		lines[bottomLineIndex] = spinnerWithMargin + strings.Repeat(" ", paddingNeeded)
	}

	content := strings.Join(lines, "\n")

	embeddedText := map[common.BorderPosition]string{
		common.TopMiddleBorder: w.getFormTitle(),
	}

	return common.Borderize(content, true, embeddedText)
}

// getSpinnerDisplayWidth calculates display width of spinner (excluding ANSI codes)
func (w *WorkingZone) getSpinnerDisplayWidth() int {
	cleanSpinner := regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(w.spinnerMsg, "")
	return len(cleanSpinner)
}

// getFormTitle returns the same title that the normal form would have
func (w *WorkingZone) getFormTitle() string {
	resourceNameStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("214")). // Orange background
		Foreground(lipgloss.Color("0"))    // Black text

	switch w.currentWidget.GetMode() {
	case common.NavigatorModeCreate:
		// Create resource type label with same styling as normal create form
		return resourceNameStyle.Render(" create: " + w.resourceType + " ")
	case common.NavigatorModeDelete:
		return resourceNameStyle.Render(" delete: " + w.resourceType + " ")
	default:
		return resourceNameStyle.Render(fmt.Sprintf(" %s ", w.resourceType))
	}
}

// GetPreviousResource returns the previous resource type from history
func (w *WorkingZone) GetPreviousResource() (string, error) {
	return w.db.GetPreviousResource()
}

// GetCurrentResource returns the current resource type from history
func (w *WorkingZone) GetCurrentResource() (string, error) {
	return w.db.GetCurrentResource()
}

// GoToPreviousResource switches back to the previous resource if available
func (w *WorkingZone) GoToPreviousResource() error {
	previousResource, err := w.GetPreviousResource()
	if err != nil {
		return err
	}

	if previousResource == "" {
		log.Debug("No previous resource available")
		return nil
	}

	log.Debug("Going back to previous resource", zap.String("previousResource", previousResource))
	w.SetResourceType(previousResource)
	return nil
}
