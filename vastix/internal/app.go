package internal

import (
	"context"
	"fmt"
	stdlog "log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	"vastix/internal/client"
	"vastix/internal/database"
	"vastix/internal/logging"
	"vastix/internal/msg_types"
	"vastix/internal/tui"
	"vastix/internal/tui/widgets/common"

	"go.uber.org/zap"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const HeaderHeight = 6                         // Height of the header zones
var activeSpinners = make(map[int16]time.Time) // Track active spinners by ids

// App represents the main TUI application
type App struct {
	ready         bool
	width, height int
	appVersion    string

	// Base context for all app operations
	ctx context.Context

	// UI zones
	profile         *tui.ProfileZone
	keybindings     *tui.KeybindingsZone
	logo            *tui.LogoZone
	workingZone     *tui.WorkingZone
	statusZone      *tui.StatusZone   // Status zone for errors and spinner
	filtersZone     *tui.FiltersZone  // Filters zone for search and filtering
	currentResource string            // Track current resource type (views, quotas, etc.)
	db              *database.Service // Database service

	// Message channel for sending messages to the app from widgets/goroutines
	msgChan chan tea.Msg

	// Spinner for splash screen and status zone
	spinnerView       string
	spinnerControl    *SpinnerControl
	spinnerActive     bool // Track if spinner is currently active
	initialTransition bool // Track if we've completed initial splash screen transition

	// Ticker controls for data updates
	tickerControl *TickerControl

	// REST client service
	restService *client.Service

	// Info message debouncing
	infoTag int // Tag for debouncing info messages

	log    *zap.Logger
	auxlog *stdlog.Logger
}

// GetMsgChan returns the message channel for sending messages to the app
func (a *App) GetMsgChan() chan tea.Msg {
	return a.msgChan
}

// NewApp creates a new TUI application
func NewApp(appVersion string, spinnerCtrl *SpinnerControl, tickerCtrl *TickerControl, msgChan chan tea.Msg) *App {
	auxlog := logging.GetAuxLogger()
	log := logging.GetGlobalLogger()
	log.Info(fmt.Sprintf("App version: %s", appVersion))

	// Initialize database
	db := database.New()
	if db == nil {
		auxlog.Println("Failed to initialize database")
		panic("failed to initialize database")
	}
	auxlog.Println("Database initialized successfully")

	// Create base context for the application
	baseCtx := context.Background()
	// Set the base context on the database service so all components can access it
	db.SetContext(baseCtx)

	// Initialize REST client service
	restService := client.InitGlobalRestService()
	auxlog.Println("REST client service initialized")

	app := &App{
		appVersion:        appVersion,
		ctx:               baseCtx,
		profile:           tui.NewProfileZone(db, baseCtx),
		keybindings:       tui.NewKeybindingsZone(db),
		logo:              tui.NewLogoZone(db),
		statusZone:        tui.NewStatusZone(db),
		currentResource:   "profiles", // Default resource type
		db:                db,
		msgChan:           msgChan,
		spinnerView:       "",
		spinnerControl:    spinnerCtrl,
		spinnerActive:     false,
		initialTransition: false, // Track initial splash screen transition
		tickerControl:     tickerCtrl,
		restService:       restService,
		log:               log,
		auxlog:            auxlog,
	}

	// Create filters zone with callback functions
	app.filtersZone = tui.NewFiltersZone(
		db,
		app.updateSizes,
	)

	// Create working zone with error handlers and ticker control that reference the app
	// Pass nil for restClient initially - it will be set later when profile is loaded
	app.workingZone = tui.NewWorkingZone(db, app.SetError, app.ClearError, app.EnableTickers, app.DisableTickers, nil, msgChan, baseCtx)

	app.workingZone.BeforeSetResourceCb = func() {
		// Reset all filters and search state before switching resource types
		app.filtersZone.ResetFilters()
	}

	app.workingZone.AfterSetResourceCb = func() {
		// Update the current widget in filters zone after resource type change
		app.filtersZone.SetCurrentWidget(app.workingZone.GetCurrentWidget())
	}

	// Set initial current widget for filters zone
	app.filtersZone.SetCurrentWidget(app.workingZone.GetCurrentWidget())

	// Set keybindings getter function for dynamic updates
	app.keybindings.SetKeyBindingsGetter(app.workingZone.GetWidgetBindings)

	return app
}

// Init initializes the application
func (a *App) Init() tea.Cmd {
	// Log any panics that occur during init
	defer logging.LogPanic()

	var initError string // Store error to set after spinner completes

	initAppCmd := func() tea.Msg {
		a.auxlog.Println("[app.Init] - start")
		// OpenAPI document will be loaded automatically when first needed
		a.profile.Init()
		a.keybindings.Init()
		a.logo.Init()
		a.workingZone.Init()
		a.statusZone.Init()
		a.filtersZone.Init()

		// Initialize API widgets if there's an active profile
		if profile, err := a.db.GetActiveProfile(); err == nil && profile != nil {
			// Try to create a real REST client
			restClient, err := profile.RestClientFromProfile()
			if err != nil {
				// REST client creation failed - save error to display later
				a.log.Warn("Failed to create REST client", zap.Error(err), zap.String("profile", profile.ProfileName()))
				a.auxlog.Printf("[app.Init] - REST client creation failed: %v", err)
				initError = err.Error()
			} else {
				// REST client created successfully - verify connectivity by getting versions
				a.auxlog.Println("[app.Init] - verifying connectivity by getting versions")
				if _, err := restClient.Versions.ListWithContext(a.ctx, nil); err != nil {
					// Connectivity test failed
					a.auxlog.Printf("[app.Init] - API connectivity test failed: %v", err)
					initError = err.Error()
				} else {
					a.auxlog.Println("[app.Init] - API connectivity verified successfully")
				}
			}

			// Initialize widgets (will create mock client internally if restClient is nil)
			a.auxlog.Println("[app.Init] - initializing API widgets")
			if err := a.workingZone.InitializeAPIWidgets(restClient, profile); err != nil {
				a.log.Error("Failed to initialize API widgets", zap.Error(err))
				if initError == "" {
					initError = fmt.Sprintf("failed to initialize API widgets: %v", err)
				}
			} else {
				a.auxlog.Println("[app.Init] - API widgets initialized successfully")
			}
		}

		a.ready = true
		a.ClearError()

		a.auxlog.Println("[app.Init] - completed successfully")
		return nil
	}

	// Start a goroutine to set error after delay (avoids being cleared by spinner start)
	go func() {
		time.Sleep(1 * time.Second)
		if initError != "" {
			a.msgChan <- msg_types.ErrorMsg{Err: fmt.Errorf("%s", initError)}
		}
	}()

	cmd := tea.Sequence(
		initAppCmd, // Init app has splash screen spinner. No need to add spinner control here
		msg_types.ProcessWithSpinner(tea.Batch(a.profile.SetData, a.workingZone.SetListData)),
	)
	return cmd
}

// forceStopAllSpinners forcefully stops all active spinners and clears spinner state
func (a *App) forceStopAllSpinners() {
	a.auxlog.Printf("Force stopping all active spinners. Active count: %d", len(activeSpinners))

	// Clear all active spinners without delay
	for spinnerId := range activeSpinners {
		a.auxlog.Printf("Force stopping spinner ID: %d", spinnerId)
		delete(activeSpinners, spinnerId)
	}

	// Force suspend spinner control
	a.spinnerControl.Suspend()
	a.spinnerActive = false

	// Clear global spinner state
	tui.GetGlobalSpinnerState().SetActive(false)

	// Clear spinner from UI
	a.workingZone.ClearSpinner()

	a.auxlog.Println("All spinners force stopped and cleaned up")
}

func (a *App) clean() {
	a.auxlog.Println("App.clean: starting cleanup")

	// Stop all spinners first
	a.forceStopAllSpinners()

	// Clean up the current widget (calls LeaveWidget if in extra mode)
	if currentWidget := a.workingZone.GetCurrentWidget(); currentWidget != nil {
		a.auxlog.Printf("App.clean: cleaning current widget: %s", currentWidget.GetName())
		currentWidget.Clean()
	}

	// Clear the widget from the working zone
	a.workingZone.ClearWidget()

	a.auxlog.Println("App.clean: cleanup complete")
}

// Update handles messages and updates the application state
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Log any panics that occur during update
	defer logging.LogPanic()

	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.updateSizes()
		return a, nil

	case msg_types.MockMsg:
		// No-op for mock messages.
		return a, nil

	case msg_types.SpinnerTickMsg:
		a.spinnerView = string(msg)
		// Show spinner in working zone if:
		// 1. App is ready and there's an active operation (spinnerActive = true)
		// 2. During splash screen, always update the spinner view for display
		if a.Ready() && a.spinnerActive {
			a.workingZone.SetSpinner(string(msg))
		}
		// Note: splash screen uses a.spinnerView directly, so it's updated regardless of ready state
		return a, nil

	case msg_types.SpinnerStartMsg:
		a.spinnerControl.Resume()
		a.spinnerActive = true
		spinnerId := msg.SpinnerId
		if spinnerId != 0 {
			activeSpinners[spinnerId] = msg.SpinnerTs
		}
		tui.GetGlobalSpinnerState().SetActive(true) // Set global spinner state - affects borders and interactions
		a.ClearError()                              // Clear any previous errors when starting a new operation
		// If we have spinner content, immediately show it in working zone
		if a.spinnerView != "" {
			a.workingZone.SetSpinner(a.spinnerView)
		}
		a.updateSizes() // Recalculate sizes when spinner state changes
		return a, nil

	case msg_types.SpinnerStopMsg:
		spinnerId := msg.SpinnerId
		spinnerStart, exists := activeSpinners[spinnerId]

		// If spinner doesn't exist, it might have been stopped already
		if !exists {
			// If no spinners are active, suspend spinner control
			if len(activeSpinners) == 0 {
				a.spinnerControl.Suspend()
				a.spinnerActive = false
				tui.GetGlobalSpinnerState().SetActive(false)
				a.workingZone.ClearSpinner()
			}
			return a, nil
		}

		// Calculate elapsed time
		duration := time.Since(spinnerStart)
		minDuration := 300 * time.Millisecond

		if duration < minDuration {
			// Need to wait longer - schedule delayed stop (don't delete spinner entry yet)
			remainingWait := minDuration - duration
			cmd = tea.Tick(remainingWait, func(time.Time) tea.Msg {
				return msg_types.SpinnerStopMsg{SpinnerId: spinnerId} // Retry stop after delay
			})
			cmds = append(cmds, cmd)
			return a, tea.Batch(cmds...)
		}

		// Enough time has passed - stop immediately
		delete(activeSpinners, spinnerId) // Now safe to remove spinner from active list

		// Only suspend spinner control if no other spinners are active
		if len(activeSpinners) == 0 {
			a.spinnerControl.Suspend()
			a.spinnerActive = false
			tui.GetGlobalSpinnerState().SetActive(false) // Clear global spinner state - restores normal borders and interactions
			a.updateSizes()                              // Recalculate sizes when spinner state changes
			a.workingZone.ClearSpinner()                 // Clear spinner from working zone
		}

		return a, tea.Batch(cmds...)

	case msg_types.ErrorMsg:
		errorMessage := "Unknown error"
		if msg.Err != nil {
			errorMessage = msg.Err.Error()
		}
		a.SetError(errorMessage)
		a.auxlog.Printf("Application error: %v", msg.Err)
		return a, nil

	case msg_types.ClearErrorMsg:
		a.ClearError()
		a.auxlog.Println("Errors cleared before new operation")
		return a, nil

	case msg_types.InfoMsg:
		a.auxlog.Printf("InfoMsg received: %s", msg.Message)
		cmd := a.SetInfo(msg.Message)
		return a, cmd

	case msg_types.ClearInfoMsg:
		a.ClearInfo()
		a.auxlog.Println("Info message cleared")
		return a, nil

	case msg_types.InfoDebounceMsg:
		// If the tag in the message matches the current tag on the model then we
		// know that this is the last message sent and we can clear the info message.
		// Otherwise, another info message was sent after this one and we ignore this clear.
		if msg.Tag == a.infoTag {
			a.ClearInfo()
			a.auxlog.Println("Info message auto-cleared after debounce timeout")
		}
		return a, nil

	case msg_types.InitProfileMsg:
		a.workingZone.SetListNavigatorMode()

		// Try to initialize API widgets if we have a REST client now
		if msg.Client != nil {
			// Get active profile for widget initialization
			profile, _ := a.db.GetActiveProfile()
			a.auxlog.Println("[InitProfileMsg] - Initializing API widgets after profile activation")
			if err := a.workingZone.InitializeAPIWidgets(msg.Client, profile); err != nil {
				a.log.Error("Failed to initialize API widgets after profile activation", zap.Error(err))
				a.auxlog.Printf("[InitProfileMsg] - ERROR: Failed to initialize API widgets: %v", err)
				return a, tea.Batch(
					a.profile.SetData,
					a.workingZone.SetListData,
					func() tea.Msg {
						return msg_types.ErrorMsg{Err: fmt.Errorf("failed to initialize API widgets: %w", err)}
					},
				)
			}
			a.auxlog.Println("[InitProfileMsg] - API widgets initialized successfully")
		}

		return a, tea.Batch(a.profile.SetData, a.workingZone.SetListData)

	case msg_types.SetDataMsg:
		if msg.UseSpinner {
			// Use spinner for data loading
			return a, msg_types.ProcessWithSpinner(a.workingZone.SetListData)
		}
		return a, a.workingZone.SetListData

	case msg_types.UpdateProfileMsg:
		// Try to initialize/reinitialize API widgets when profile changes
		if profile, err := a.db.GetActiveProfile(); err == nil && profile != nil {
			if restClient, err := profile.RestClientFromProfile(); err == nil && restClient != nil {
				a.auxlog.Println("[UpdateProfileMsg] - Reinitializing API widgets after profile change")
				if err := a.workingZone.InitializeAPIWidgets(restClient, profile); err != nil {
					a.log.Error("Failed to reinitialize API widgets", zap.Error(err))
					a.auxlog.Printf("[UpdateProfileMsg] - ERROR: Failed to reinitialize API widgets: %v", err)
				} else {
					a.auxlog.Println("[UpdateProfileMsg] - API widgets reinitialized successfully")
				}
			}
		}

		cmd := tea.Sequence(a.workingZone.ResetAllWidgets, tea.Batch(a.profile.SetData, a.workingZone.SetListData))
		return a, cmd

	case msg_types.TickerSetDataMsg:
		// Log ticker activity for debugging
		if a.workingZone.GetCurrentWidget() != nil {
			currentMode := a.workingZone.GetCurrentWidget().GetMode()
			// Only process ticker data updates in list mode
			if currentMode == common.NavigatorModeList {
				// Derive context with ignore logging flag from app's base context for periodic ticker requests
				ctx := client.WithIgnoreLogging(a.ctx)
				// Combine list data update with forced UI redraw for immediate visual feedback
				cmd := tea.Sequence(func() tea.Msg {
					return a.workingZone.SetListDataThrottledWithContext(ctx)
				}, func() tea.Msg {
					return tea.WindowSizeMsg{Width: a.width, Height: a.height}
				})
				return a, cmd
			}
		} else {
			a.auxlog.Println("TickerSetDataMsg: currentWidget is nil")
		}
		return a, nil

	case msg_types.TickerUpdateProfileMsg:
		// Derive context with ignore logging flag from app's base context for periodic ticker requests
		ctx := client.WithIgnoreLogging(a.ctx)
		return a, func() tea.Msg {
			return a.profile.SetDataWithContext(ctx)
		}

	case msg_types.ProfileDataMsg:
		a.profile.UpdateData(msg)
		return a, nil

	case msg_types.DetailsContentMsg:
		a.updateSizes()

		// SetDetailsData now handles both normal and extra mode internally
		// (delegates to ExtraWidgetGroup if in extra mode, switches to details mode)
		a.workingZone.SetDetailsData(msg.Content)

		return a, nil

	case msg_types.SetResourceTypeMsg:
		cmd := func() tea.Msg {
			a.ClearError()
			// Reset to list mode before switching to new resource type
			a.workingZone.SetResourceType(msg.ResourceType)
			// Set list mode for the new resource type
			a.workingZone.SetListNavigatorMode()
			return a.workingZone.SetListData()
		}
		return a, msg_types.ProcessWithSpinner(cmd)

	case tea.KeyMsg:
		// Check if filters zone should handle this message
		if a.filtersZone.IsActive() || (msg.String() == "/" && a.filtersZone.IsFuzzySearchable()) || (msg.String() == "?" && a.filtersZone.IsServerSearchable()) {
			a.filtersZone, cmd = a.filtersZone.Update(msg)
			cmds = append(cmds, cmd)
			return a, tea.Batch(cmds...)
		}

		// Normal mode key handling
		switch msg.String() {
		case "ctrl+c":
			a.auxlog.Println("Ctrl+C pressed, initiating shutdown")
			a.clean()
			return a, tea.Quit
		case ":":
			// Switch to resources widget
			a.ClearError()
			a.workingZone.SetResourceType("resources")
			a.workingZone.SetListNavigatorMode()
			a.workingZone.SetListData()
			return a, nil
		case "esc":
			a.ClearError()
		}
	}

	// Update working zone
	a.workingZone, cmd = a.workingZone.Update(msg)
	cmds = append(cmds, cmd)

	return a, tea.Batch(cmds...)
}

// View renders the entire application
func (a *App) View() string {
	// Log any panics that occur during view rendering
	defer logging.LogPanic()

	if !a.allZonesReady() || !a.Ready() {
		return a.renderSplashScreen()
	}

	// Handle initial transition from splash screen to normal view (only once)
	if !a.initialTransition {
		if len(activeSpinners) == 0 {
			a.spinnerControl.Suspend()
			a.spinnerActive = false
			a.spinnerView = "" // Clear the spinner view too
			a.statusZone.ClearSpinner()
		} else {
			// Keep spinner active if there are other operations running
			a.auxlog.Printf("Keeping spinner active during transition - %d operations running", len(activeSpinners))
		}
		a.initialTransition = true // Mark that we've completed initial transition
	}

	// Normal rendering - spinner is now handled by status zone for operations
	return a.renderNormal()
}

// renderNormal renders the normal application layout
func (a *App) renderNormal() string {
	// Create the top zones layout (3 zones side by side)
	topZones := a.renderTopZones()

	// Calculate available height for filters and working zone
	remainingHeight := a.height - lipgloss.Height(topZones)

	// Render filters zone if active
	var filtersArea string
	if a.filtersZone.IsActive() {
		filtersArea = a.filtersZone.View()
		remainingHeight -= lipgloss.Height(filtersArea)
	}

	// Create working area
	workingArea := a.workingZone.View()
	if remainingHeight > 0 {
		workingArea = lipgloss.NewStyle().
			Height(remainingHeight).
			Width(a.width).
			Render(workingArea)
	}

	// Combine everything
	var content string
	if a.filtersZone.IsActive() {
		content = lipgloss.JoinVertical(lipgloss.Top, topZones, filtersArea, workingArea)
	} else {
		content = lipgloss.JoinVertical(lipgloss.Top, topZones, workingArea)
	}

	return content
}

// Ready returns whether the App is ready to be displayed
func (a *App) Ready() bool {
	return a.ready
}

// SetError sets an error message to be displayed in the errors zone
func (a *App) SetError(msg string) {
	a.statusZone.SetError(msg)
	a.updateSizes() // Recalculate sizes when error state changes
}

// ClearError clears any error message from the errors zone
func (a *App) ClearError() {
	a.statusZone.Clear()
	a.updateSizes() // Recalculate sizes when error state changes
}

// SetInfo sets an info message to be displayed in the status zone with auto-clear debounce
func (a *App) SetInfo(msg string) tea.Cmd {
	// Increment the tag for debouncing
	a.infoTag++
	currentTag := a.infoTag

	a.statusZone.SetInfo(msg)
	a.updateSizes() // Recalculate sizes when info state changes

	// Schedule auto-clear after 1 second using debounce pattern similar to user's example
	return tea.Tick(1*time.Second, func(_ time.Time) tea.Msg {
		return msg_types.InfoDebounceMsg{Tag: currentTag}
	})
}

// ClearInfo clears any info message from the status zone
func (a *App) ClearInfo() {
	a.statusZone.ClearInfo()
	a.updateSizes() // Recalculate sizes when info state changes
}

// EnableTickers starts both data and profile tickers
func (a *App) EnableTickers() {
	if a.tickerControl != nil {
		a.tickerControl.Enable()
	}
}

// DisableTickers stops both data and profile tickers
func (a *App) DisableTickers() {
	if a.tickerControl != nil {
		a.auxlog.Println("Disabling dataGetter Tickers.")
		a.tickerControl.Disable()
	}
}

// renderTopZones creates the three top zones without borders
func (a *App) renderTopZones() string {
	// Check if terminal width is less than 90 - hide keybindings if so
	var topZonesView string

	if a.width < 61 {
		logoWidth := a.width
		// Render logo zone
		logoView := a.logo.View()
		logoView = lipgloss.NewStyle().
			Width(logoWidth).
			Height(HeaderHeight).
			Align(lipgloss.Right, lipgloss.Center).
			Render(logoView)

		topZonesView = lipgloss.NewStyle().Width(a.width).Render(logoView)
	} else if a.width < 91 {
		// Only show profile and logo zones
		profileWidth := a.width / 2
		logoWidth := a.width - profileWidth

		// Render profile zone
		profileView := a.profile.View()
		profileView = lipgloss.NewStyle().
			Width(profileWidth).
			Height(HeaderHeight).
			Render(profileView)

		// Render logo zone
		logoView := a.logo.View()
		logoView = lipgloss.NewStyle().
			Width(logoWidth).
			Height(HeaderHeight).
			Align(lipgloss.Right, lipgloss.Center).
			Render(logoView)

		// Join only profile and logo horizontally
		topZones := lipgloss.JoinHorizontal(lipgloss.Top, profileView, logoView)
		topZonesView = lipgloss.NewStyle().Width(a.width).Render(topZones)
	} else {

		// Normal view with all 3 zones when width >= 90
		// Allocate more space to keybindings: Profile 25%, Keybindings 50%, Logo 25%
		profileWidth := a.width / 4
		keybindingsWidth := a.width / 2
		logoWidth := a.width - profileWidth - keybindingsWidth // Take remaining width to ensure full coverage

		// Render each zone with fixed dimensions
		profileView := a.profile.View()
		profileView = lipgloss.NewStyle().
			Width(profileWidth).
			Height(HeaderHeight).
			Render(profileView)

		keybindingsView := a.keybindings.View()
		keybindingsView = lipgloss.NewStyle().
			Width(keybindingsWidth).
			Height(HeaderHeight).
			Render(keybindingsView)

		logoView := a.logo.View()
		logoView = lipgloss.NewStyle().
			Width(logoWidth).
			Height(HeaderHeight).
			Align(lipgloss.Right, lipgloss.Center).
			Render(logoView)

		// Join zones horizontally to fill the full terminal width
		topZones := lipgloss.JoinHorizontal(lipgloss.Top, profileView, keybindingsView, logoView)

		// Ensure the top zones exactly match the terminal width
		topZonesView = lipgloss.NewStyle().Width(a.width).MaxHeight(HeaderHeight).Render(topZones)
	}

	// Handle error zone separately and return complete header with actual heights
	errMsg := a.statusZone.View()
	if errMsg != "" {
		// If there is an error message, add it to the top zones
		topZonesView = lipgloss.JoinVertical(lipgloss.Top, topZonesView, errMsg)
	}

	return topZonesView
}

// updateSizes updates the sizes of child components
func (a *App) updateSizes() {
	if a.width == 0 || a.height == 0 {
		return
	}

	// Allocate more space to keybindings: Profile 25%, Keybindings 50%, Logo 25%
	profileWidth := a.width / 4
	keybindingsWidth := a.width / 2
	logoWidth := a.width - profileWidth - keybindingsWidth

	// Set the errors zone width first
	a.statusZone.SetSize(a.width, 0) // Height will be calculated dynamically

	// Render the complete top zones to get the actual height
	topZones := a.renderTopZones()
	actualTopZoneHeight := lipgloss.Height(topZones)

	// Calculate working zone height dynamically using actual rendered height
	workingHeight := a.height - actualTopZoneHeight

	// Subtract filters zone height if active (use actual rendered height for consistency)
	filtersHeight := 0
	if a.filtersZone.IsActive() {
		filtersArea := a.filtersZone.View()
		filtersHeight = lipgloss.Height(filtersArea)
	}
	workingHeight -= filtersHeight

	// Ensure working height is not negative
	if workingHeight < 0 {
		workingHeight = 0
	}

	a.profile.SetSize(profileWidth, HeaderHeight)
	a.keybindings.SetSize(keybindingsWidth, HeaderHeight)
	a.logo.SetSize(logoWidth, HeaderHeight)
	a.workingZone.SetSize(a.width, workingHeight)
	a.filtersZone.SetSize(a.width, filtersHeight)
}

// allZonesReady checks if all zones are ready to be displayed
func (a *App) allZonesReady() bool {
	return a.profile.Ready() &&
		a.keybindings.Ready() &&
		a.logo.Ready() &&
		a.workingZone.Ready() &&
		a.statusZone.Ready() &&
		a.filtersZone.Ready()
}

// renderSplashScreen renders the splash screen with spinner
func (a *App) renderSplashScreen() string {
	// Create version and spinner content for bottom left with margin
	leftMargin := "    "

	// Build version content with faded colors
	versionContent := ""
	if a.appVersion != "" {
		versionStyle := lipgloss.NewStyle().
			Foreground(tui.LightGrey)

		versionText := "version: " + a.appVersion
		versionContent = leftMargin + versionStyle.Render(versionText)
	}

	// Create spinner content
	spinnerContent := ""
	if a.spinnerView != "" {
		spinnerContent = leftMargin + a.spinnerView
	}

	// Build the final splash screen - start from top
	content := ""

	// Add version and spinner at bottom
	if versionContent != "" || spinnerContent != "" {
		// Calculate spacing to position at bottom
		bottomSpacing := ""
		reservedLines := 2 // Reserve space for version and spinner
		if versionContent != "" {
			reservedLines++
		}
		if spinnerContent != "" {
			reservedLines++
		}

		totalSpacing := a.height - reservedLines
		for i := 0; i < totalSpacing; i++ {
			bottomSpacing += "\n"
		}
		content += bottomSpacing

		// Add version above spinner
		if versionContent != "" {
			content += versionContent + "\n"
		}

		// Add spinner
		if spinnerContent != "" {
			content += spinnerContent
		}
	}

	return content
}

// Run starts the TUI application
func Run(appVersion string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	closeLogger, err := logging.InitGlobalLogger()
	if err != nil {
		return err
	}
	if closeLogger != nil {
		defer closeLogger()
	}

	// Initialize auxlog AFTER logger initialization
	auxlog := logging.GetAuxLogger()

	// Initialize database for cleanup later
	db := database.New()
	defer func() {
		auxlog.Println("Closing database connection...")
		if err := db.Close(); err != nil {
			auxlog.Printf("Error closing database: %v", err)
		} else {
			auxlog.Println("Database connection closed successfully")
		}
	}()

	ch := make(chan tea.Msg)
	spinnerCtrl, tickerCtrl, unsub := SetupSubscriptions(ctx, cancel, ch)
	defer unsub()

	app := NewApp(appVersion, spinnerCtrl, tickerCtrl, ch)

	// Start with spinner active for splash screen
	spinnerCtrl.Resume()
	app.spinnerActive = true // Enable spinner for splash screen

	p := tea.NewProgram(app, tea.WithAltScreen())

	go func() {
		// Relay events to model in background
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return // Channel closed
				}
				p.Send(msg)
			case <-ctx.Done():
				return // Context canceled
			}
		}
	}()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run the program in a goroutine
	resultChan := make(chan error, 1)
	go func() {
		// Add panic logging for this goroutine (bubbletea has its own panic recovery)
		defer logging.LogPanic()
		_, err := p.Run()
		resultChan <- err
	}()

	// Wait for either program completion or interrupt signal
	select {
	case err := <-resultChan:
		auxlog.Println("TUI program finished")
		cancel() // Cancel context to stop background goroutines

		// Check if the error indicates a panic that was caught by bubbletea
		if err != nil {
			auxlog.Printf("TUI program error: %v", err)

			// Bubbletea returns specific error messages when panics occur
			errMsg := err.Error()
			if errMsg == "program was killed: program experienced a panic" {
				auxlog.Println("Detected panic caught by bubbletea, logging to panic.log")

				// Log this to panic.log file
				if vastixDir, dirErr := logging.GetVastixDir(); dirErr == nil {
					logsDir := filepath.Join(vastixDir, "logs")
					panicLogPath := filepath.Join(logsDir, "panic.log")

					if f, fileErr := os.OpenFile(panicLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); fileErr == nil {
						timestamp := time.Now().Format("2006-01-02 15:04:05")
						panicInfo := fmt.Sprintf("\n"+
							"================================================================================\n"+
							"PANIC at %s (caught by bubbletea)\n"+
							"================================================================================\n"+
							"Error: %v\n\n"+
							"Note: Full stack trace may be in terminal output above.\n"+
							"This panic was caught by bubbletea's internal panic handler.\n"+
							"================================================================================\n\n",
							timestamp, err)

						f.WriteString(panicInfo)
						f.Close()

						fmt.Fprintf(os.Stderr, "\n  Panic details saved to: %s\n", panicLogPath)
						fmt.Fprintf(os.Stderr, "Check terminal output above for full stack trace.\n\n")
					}
				}
			}
		}

		return err
	case <-sigChan:
		auxlog.Println("Received interrupt signal, shutting down...")
		cancel()                               // Cancel context first
		p.Send(tea.KeyMsg{Type: tea.KeyCtrlC}) // Send quit to program

		// Give the program a moment to clean up
		select {
		case err := <-resultChan:
			auxlog.Println("TUI program shut down gracefully")
			return err
		case <-time.After(2 * time.Second):
			auxlog.Println("TUI program shutdown timeout, forcing exit")
			return nil
		}
	}
}
