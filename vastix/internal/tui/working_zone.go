package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"vastix/internal/database"
	"vastix/internal/logging"
	log "vastix/internal/logging"
	"vastix/internal/tui/widgets"
	"vastix/internal/tui/widgets/common"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func NewWorkingZone(db *database.Service, errorHandler func(string), clearErrors func(), enableTickers func(), disableTickers func()) *WorkingZone {
	log.Debug("WorkingZone initializing")

	registeredWidgetsList := []common.Widget{
		widgets.NewProfile(db),
		widgets.NewSshConnections(db),
		widgets.NewResources(db),
		widgets.NewView(db),
		widgets.NewUser(db),
		widgets.NewUserKeysFromLocalDb(db),
		widgets.NewViewPolicy(db),
		widgets.NewVipPool(db),
		widgets.NewEventDefinition(db),
		widgets.NewEventDefinitionConfig(db),
		widgets.NewBGPConfig(db),
		widgets.NewNonLocalUser(db),
		widgets.NewApiToken(db),
		widgets.NewApiTokensFromLocalDb(db),
		widgets.NewActiveDirectory(db),
		widgets.NewAdministratorManager(db),
		widgets.NewAdministratorRealm(db),
		widgets.NewAdministratorRole(db),
		widgets.NewBlockHost(db),
		widgets.NewDNS(db),
		widgets.NewEncryptionGroup(db),
		widgets.NewGlobalSnapshotStream(db),
		widgets.NewGroup(db),
		widgets.NewLDAP(db),
		widgets.NewLocalProvider(db),
		widgets.NewLocalS3Key(db),
		widgets.NewNIS(db),
		widgets.NewProtectedPath(db),
		widgets.NewProtectionPolicy(db),
		widgets.NewQosPolicy(db),
		widgets.NewQuota(db),
		widgets.NewReplicationPeer(db),
		widgets.NewS3LifeCycleRule(db),
		widgets.NewS3Policy(db),
		widgets.NewS3ReplicationPeer(db),
		widgets.NewSnapshot(db),
		widgets.NewTenant(db),
		widgets.NewVms(db),
		widgets.NewVolume(db),
		widgets.NewBlockHostMapping(db),
	}

	registeredWidgets := make(map[string]common.Widget, len(registeredWidgetsList))
	for _, widget := range registeredWidgetsList {
		resourceType := widget.GetResourceType()

		if resourceType != "resources" {
			widgets.SupportedResources = append(widgets.SupportedResources, widget.GetResourceType())

		}
		registeredWidgets[resourceType] = widget
	}

	log.Debug("Base widgets initialized",
		zap.Int("widget_count", len(registeredWidgets)))

	// Get current resource from database history, fallback to "profiles" if none exists
	currentResourceType, err := db.GetCurrentResource()
	if err != nil {
		log.Debug("Failed to get current resource from database, using default", zap.Error(err))
		currentResourceType = "profiles"
	}

	// Initialize resource history if it doesn't exist
	if err := db.InitializeResourceHistory(currentResourceType); err != nil {
		log.Debug("Failed to initialize resource history", zap.Error(err))
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

	// Check if the current resource type has a registered widget, fallback to "views" if not
	currentWidget := registeredWidgets[currentResourceType]
	if currentWidget == nil {
		log.Warn("Current resource type not found in registered widgets, falling back to views",
			zap.String("currentResourceType", currentResourceType))
		currentResourceType = "views"
		currentWidget = registeredWidgets["views"]

		// Update the database with the fallback resource
		if err := db.SetResourceHistory("views", currentResourceType); err != nil {
			log.Debug("Failed to update resource history after fallback", zap.Error(err))
		}
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
	// Check if currentWidget is nil to prevent panic
	if w.currentWidget == nil {
		log.Error("Cannot set navigator mode: currentWidget is nil")
		return
	}

	// Set the navigator mode for the current widget
	w.currentWidget.SetMode(mode)

	// Control tickers based on navigator mode
	// Tickers are enabled only in NavigatorModeList to provide real-time data updates
	// when browsing lists. They are disabled in other modes (create, delete, details)
	// to avoid interference and unnecessary processing during focused operations.
	if mode == common.NavigatorModeList {
		// Enable tickers only in list mode
		if w.enableTickers != nil {
			w.enableTickers()
		}
		// Get the widget navigator and reset its creation.
		w.currentWidget.ResetCreateForm()
	} else {
		// Disable tickers for all other modes (create, delete, details)
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
		if err := w.db.SetResourceHistory(resourceType, currentFromDB); err != nil {
			log.Debug("Failed to update resource history in database",
				zap.Error(err),
				zap.String("new_current", resourceType),
				zap.String("new_previous", currentFromDB))
		} else {
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
	auxlog := logging.GetAuxLogger()
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
