package tui

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"vastix/internal/colors"
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
	ctx           context.Context   // Base context for API calls

	// Message channel for sending messages to the app
	msgChan chan tea.Msg

	// Widget management
	widgets       map[string]common.Widget
	currentWidget common.Widget

	// Spinner display (moved from status zone)
	spinnerMsg string

	// Log buffer for spinner mode
	logBuffer   []string // Buffer of log lines
	maxLogLines int      // Maximum number of log lines to keep

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

func NewWorkingZone(db *database.Service, errorHandler func(string), clearErrors func(), enableTickers func(), disableTickers func(), restClient interface{}, msgChan chan tea.Msg, ctx context.Context) *WorkingZone {
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
	// Pass msgChan so it can be propagated to all child widgets (including VipPoolForwarding for health monitoring)
	resourcesWidget := widgets.NewResources(db, msgChan).(*widgets.Resources)

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
		ctx:            ctx,
		msgChan:        msgChan,
		widgets:        registeredWidgets,
		currentWidget:  currentWidget,
		errorHandler:   errorHandler,
		clearErrors:    clearErrors,
		enableTickers:  enableTickers,
		disableTickers: disableTickers,
		logBuffer:      []string{},
		maxLogLines:    100, // Keep last 100 lines
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
func (w *WorkingZone) InitializeAPIWidgets(restClient *rest.UntypedVMSRest, profile *database.Profile) error {
	// If restClient is nil (e.g., due to network error), we cannot initialize API widgets
	if restClient == nil {
		log.Warn("REST client is nil, cannot initialize API widgets (will retry when profile is activated)")
		return nil // Not an error - just means widgets aren't available yet
	}

	log.Info("Initializing API widgets")

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

		for name, widget := range allWidgets {
			// Check if this widget was already registered
			if _, exists := w.widgets[name]; exists {
				// Widget already exists - replace it with the new one to pick up new REST client
				log.Debug("Replacing existing widget with new one (profile switch)", zap.String("resource", name))
			} else {
				log.Debug("Registered new widget", zap.String("resource", name))
			}
			// Always set/replace the widget
			w.widgets[name] = widget
			widget.SetSize(w.width, w.height)
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

	// Set the navigator mode for the current widget
	w.currentWidget.SetMode(mode)

	// Control tickers based on navigator mode
	// Tickers are enabled only in NavigatorModeList to provide real-time data updates
	// when browsing lists. They are disabled in other modes (create, delete, details, extra)
	// to avoid interference and unnecessary processing during focused operations.
	// IMPORTANT: Check the widget's ACTUAL mode after SetMode, not the input parameter,
	// because widgets can override SetMode and change the mode (e.g., VSettings)
	if w.currentWidget.GetMode() == common.NavigatorModeList {
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
		auxlog.Printf("Widgets history: previous='%s' -> current='%s'", currentFromDB, resourceType)
		if err := w.db.SetResourceHistory(resourceType, currentFromDB); err != nil {
			auxlog.Printf("Failed to update resource history: %v", err)
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
// Also enables aux logger output to working zone
func (w *WorkingZone) SetSpinner(msg string) {
	w.spinnerMsg = msg

	// Enable aux logger to write to working zone when spinner starts
	if msg != "" {
		log.SetAuxLogWriter(w)
	}
}

// ClearSpinner clears the spinner message and log buffer from the working zone
// Also disables aux logger output to working zone
func (w *WorkingZone) ClearSpinner() {
	w.spinnerMsg = ""
	w.ClearLogs()

	// Disable aux logger output to working zone when spinner ends
	log.ClearAuxLogWriter()
}

func (w *WorkingZone) ClearWidget() {
	// Set the current widget to nil (cleanup is handled by caller via Clean())
	w.currentWidget = nil
}

// HasSpinner returns true if there's a spinner message to display
func (w *WorkingZone) HasSpinner() bool {
	return w.spinnerMsg != ""
}

// Write implements io.Writer interface for streaming logs to the working zone
// Only accepts logs when in spinner mode
func (w *WorkingZone) Write(p []byte) (n int, err error) {
	// Only buffer logs if we're in spinner mode
	if !w.HasSpinner() {
		// Not in spinner mode - silently discard
		return len(p), nil
	}

	lines := strings.Split(string(p), "\n")

	for _, line := range lines {
		if line == "" {
			continue // Skip empty lines
		}

		w.logBuffer = append(w.logBuffer, line)

		// Keep only the last maxLogLines
		if len(w.logBuffer) > w.maxLogLines {
			w.logBuffer = w.logBuffer[len(w.logBuffer)-w.maxLogLines:]
		}
	}

	return len(p), nil
}

// AppendLog appends a log line to the buffer
func (w *WorkingZone) AppendLog(text string) {
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		w.logBuffer = append(w.logBuffer, line)

		// Keep only the last maxLogLines
		if len(w.logBuffer) > w.maxLogLines {
			w.logBuffer = w.logBuffer[len(w.logBuffer)-w.maxLogLines:]
		}
	}
}

// ClearLogs clears the log buffer
func (w *WorkingZone) ClearLogs() {
	w.logBuffer = []string{}
}

// GetLogWriter returns the working zone as an io.Writer for logs
func (w *WorkingZone) GetLogWriter() *WorkingZone {
	return w
}

// GetMsgChan returns the message channel for widgets to send messages to the app
func (w *WorkingZone) GetMsgChan() chan tea.Msg {
	return w.msgChan
}

func (w *WorkingZone) SetListData() tea.Msg {
	return w.SetListDataWithContext(w.ctx)
}

// SetListDataWithContext fetches list data with the provided context.
// This allows callers to pass context with special flags (e.g., to skip interceptor logging for ticker updates).
func (w *WorkingZone) SetListDataWithContext(ctx context.Context) tea.Msg {
	if w.currentWidget == nil {
		log.Error("SetListData: currentWidget is nil")
		return msg_types.MockMsg{}
	}

	// Call the widget's SetListDataWithContext method if available, otherwise fall back to SetListData
	type contextAwareWidget interface {
		SetListDataWithContext(context.Context) tea.Msg
	}

	var result tea.Msg
	if ctxWidget, ok := w.currentWidget.(contextAwareWidget); ok {
		result = ctxWidget.SetListDataWithContext(ctx)
	} else {
		result = w.currentWidget.SetListData()
	}

	// Update throttling fields after successful execution
	w.lastSetDataTime = time.Now()
	w.lastSetDataWidget = w.currentWidget

	return result
}

// SetListDataThrottled calls SetListData only if:
// - The current widget has changed since the last call, OR
// - More than 1 minute has passed since the last call for the same widget
func (w *WorkingZone) SetListDataThrottled() tea.Msg {
	return w.SetListDataThrottledWithContext(w.ctx)
}

// SetListDataThrottledWithContext calls SetListDataWithContext only if throttle conditions are met.
// This allows callers to pass context with special flags (e.g., to skip interceptor logging for ticker updates).
func (w *WorkingZone) SetListDataThrottledWithContext(ctx context.Context) tea.Msg {
	now := time.Now()

	// Check if widget has changed
	widgetChanged := w.lastSetDataWidget != w.currentWidget

	// Check if more than the throttle interval has passed since last call for the same widget
	timeSinceLastCall := now.Sub(w.lastSetDataTime)
	timePassed := timeSinceLastCall >= ThrottleInterval
	// Call SetListDataWithContext if widget changed or enough time has passed
	if widgetChanged || timePassed {
		// Call SetListDataWithContext - it will update the throttling fields after execution
		return w.SetListDataWithContext(ctx)
	}

	// Throttled - return nil (no-op)
	return nil
}

func (w *WorkingZone) SetDetailsData(details any) {
	w.currentWidget.SetDetailsData(details)
}

func (w *WorkingZone) GetWidgetBindings() []common.KeyBinding {
	if w.currentWidget == nil {
		return []common.KeyBinding{}
	}
	bindings := w.currentWidget.GetKeyBindings()
	return bindings
}

// renderSpinnerOnly renders content split into two horizontal zones:
// - Top zone: Latest logs that fit the width
// - Bottom zone: Spinner
func (w *WorkingZone) renderSpinnerOnly() string {
	if w.spinnerMsg == "" {
		return ""
	}

	// Calculate inner content dimensions (accounting for borders that will be added)
	innerWidth := w.width - 2       // Account for left and right borders
	innerHeight := w.height - 3 + 1 // Account for top, bottom borders, and title area + 1 line to match normal content

	if innerWidth < 1 {
		innerWidth = w.width
	}
	if innerHeight < 1 {
		innerHeight = 5 // Minimum height
	}

	// Reserve space for spinner at bottom (3 lines: 1 empty, 1 spinner, 1 margin)
	spinnerHeight := 3
	logHeight := innerHeight - spinnerHeight
	if logHeight < 0 {
		logHeight = 0
	}

	// Create all lines
	lines := make([]string, innerHeight)

	// Top zone: Render logs with left padding and gray color
	leftPadding := "  "                                         // 2 spaces left padding
	grayStyle := lipgloss.NewStyle().Foreground(colors.Grey240) // Gray color

	if logHeight > 0 && len(w.logBuffer) > 0 {
		// Get the latest logs that fit in the log zone
		startIdx := 0
		if len(w.logBuffer) > logHeight {
			startIdx = len(w.logBuffer) - logHeight
		}

		logLines := w.logBuffer[startIdx:]

		// Render each log line with padding and gray color
		for i := 0; i < logHeight; i++ {
			if i < len(logLines) {
				logLine := logLines[i]

				// Remove ANSI codes for width calculation
				cleanLine := regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(logLine, "")

				// Calculate available width after padding
				availableWidth := innerWidth - len(leftPadding)
				if availableWidth < 1 {
					availableWidth = 1
				}

				// Apply gray color to the log line
				styledLine := grayStyle.Render(cleanLine)

				// Remove ANSI from styled line for width calculation
				cleanStyled := regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(styledLine, "")

				if len(cleanLine) > availableWidth {
					// Truncate to fit available width
					// Find where to cut in styled line (approximate)
					cutPoint := availableWidth
					styledLine = grayStyle.Render(cleanLine[:cutPoint])
				}

				// Add left padding and right padding to fill width
				logWithPadding := leftPadding + styledLine
				cleanWithPadding := leftPadding + cleanStyled
				rightPadding := innerWidth - len(cleanWithPadding)
				if rightPadding > 0 {
					lines[i] = logWithPadding + strings.Repeat(" ", rightPadding)
				} else {
					lines[i] = logWithPadding
				}
			} else {
				// Empty line
				lines[i] = strings.Repeat(" ", innerWidth)
			}
		}
	} else {
		// No logs - fill with empty lines
		for i := 0; i < logHeight; i++ {
			lines[i] = strings.Repeat(" ", innerWidth)
		}
	}

	// Bottom zone: Render spinner
	leftMargin := "    " // 4 spaces margin

	// Add separator line (empty)
	if logHeight < innerHeight {
		lines[logHeight] = strings.Repeat(" ", innerWidth)
	}

	// Add spinner line
	if logHeight+1 < innerHeight {
		spinnerWithMargin := leftMargin + w.spinnerMsg
		cleanSpinnerWidth := w.getSpinnerDisplayWidth()
		totalVisualWidth := cleanSpinnerWidth + 6 // +6 for [  ] brackets with spaces
		paddingNeeded := common.Max(0, innerWidth-len(leftMargin)-totalVisualWidth)
		lines[logHeight+1] = spinnerWithMargin + strings.Repeat(" ", paddingNeeded)
	}

	// Add bottom margin line
	if logHeight+2 < innerHeight {
		lines[logHeight+2] = strings.Repeat(" ", innerWidth)
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
		Background(colors.Orange). // Orange background
		Foreground(colors.BlackTerm) // Black text

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
