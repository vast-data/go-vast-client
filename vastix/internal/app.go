package internal

import (
	"context"
	"fmt"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"vastix/internal/client"
	"vastix/internal/database"
	"vastix/internal/logging"
	"vastix/internal/msg_types"
	"vastix/internal/tui"
	"vastix/internal/tui/widgets/common"

	"github.com/vast-data/go-vast-client/openapi_schema"
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

	// UI zones
	profile         *tui.ProfileZone
	keybindings     *tui.KeybindingsZone
	logo            *tui.LogoZone
	workingZone     *tui.WorkingZone
	statusZone      *tui.StatusZone   // Status zone for errors and spinner
	filtersZone     *tui.FiltersZone  // Filters zone for search and filtering
	currentResource string            // Track current resource type (views, quotas, etc.)
	db              *database.Service // Database service

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

// NewApp creates a new TUI application
func NewApp(appVersion string, spinnerCtrl *SpinnerControl, tickerCtrl *TickerControl) *App {
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

	// Initialize REST client service
	restService := client.InitGlobalRestService()
	auxlog.Println("REST client service initialized")

	app := &App{
		appVersion:        appVersion,
		profile:           tui.NewProfileZone(db),
		keybindings:       tui.NewKeybindingsZone(db),
		logo:              tui.NewLogoZone(db),
		statusZone:        tui.NewStatusZone(db),
		currentResource:   "profiles", // Default resource type
		db:                db,
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
	app.workingZone = tui.NewWorkingZone(db, app.SetError, app.ClearError, app.EnableTickers, app.DisableTickers)

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
	initAppCmd := func() tea.Msg {
		// Hot preload OpenAPI documentation
		a.auxlog.Println("[app.Init] - start")
		if _, err := openapi_schema.GetOpenApiResource("/"); err != nil {
			a.log.Error("Failed to load OpenAPI document", zap.Error(err))
		}
		a.profile.Init()
		a.keybindings.Init()
		a.logo.Init()
		a.workingZone.Init()
		a.statusZone.Init()
		a.filtersZone.Init()
		a.ready = true
		a.ClearError()

		a.auxlog.Println("[app.Init] - completed successfully")
		return nil
	}

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

// Update handles messages and updates the application state
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		a.auxlog.Printf("Spinner started, errors cleared. Active: %t", a.spinnerActive)
		return a, nil

	case msg_types.SpinnerStopMsg:
		spinnerId := msg.SpinnerId
		spinnerStart, exists := activeSpinners[spinnerId]

		// If spinner doesn't exist, it might have been stopped already
		if !exists {
			a.auxlog.Printf("Spinner stop requested but spinner not found, id: %d", spinnerId)
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
			a.auxlog.Printf(
				"Spinner stop received too early, delaying stop. Elapsed: %s, Remaining: %s",
				duration, remainingWait,
			)

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

		a.auxlog.Printf("Spinner stopped. Total duration: %s", duration)
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
		return a, tea.Batch(a.profile.SetData, a.workingZone.SetListData)

	case msg_types.SetDataMsg:
		if msg.UseSpinner {
			// Use spinner for data loading
			return a, msg_types.ProcessWithSpinner(a.workingZone.SetListData)
		}
		return a, a.workingZone.SetListData

	case msg_types.UpdateProfileMsg:
		cmd := tea.Sequence(a.workingZone.ResetAllWidgets, tea.Batch(a.profile.SetData, a.workingZone.SetListData))
		return a, cmd

	case msg_types.TickerSetDataMsg:
		// Force redraw when in list mode to show updated state
		if a.workingZone.GetCurrentWidget() != nil && a.workingZone.GetCurrentWidget().GetMode() == common.NavigatorModeList {
			// Combine list data update with forced UI redraw for immediate visual feedback
			cmd := tea.Sequence(a.workingZone.SetListDataThrottled, func() tea.Msg {
				return tea.WindowSizeMsg{Width: a.width, Height: a.height}
			})
			return a, cmd
		}
		return a, a.workingZone.SetListDataThrottled

	case msg_types.TickerUpdateProfileMsg:
		return a, a.profile.SetData

	case msg_types.ProfileDataMsg:
		a.profile.UpdateData(msg)
		return a, nil

	case msg_types.DetailsContentMsg:
		a.updateSizes()
		a.workingZone.SetDetailsData(msg.Content)
		return a, nil

	case msg_types.SetResourceTypeMsg:
		a.auxlog.Printf("SetResourceTypeMsg received, switching resource type to: %q", msg.ResourceType)
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
			a.forceStopAllSpinners() // Clean up all spinners before quitting
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
			a.auxlog.Println("Splash screen spinner stopped - initial app transition completed")
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
		a.auxlog.Println("Enabling dataGetter Tickers.")
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
		zoneWidth := a.width / 3
		profileWidth := zoneWidth
		keybindingsWidth := zoneWidth
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

	zoneWidth := a.width / 3

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

	a.profile.SetSize(zoneWidth, HeaderHeight)
	a.keybindings.SetSize(zoneWidth, HeaderHeight)
	a.logo.SetSize(a.width-zoneWidth*2, HeaderHeight)
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
		if err := db.Close(); err != nil {
			auxlog.Printf("Error closing database: %v", err)
		} else {
			auxlog.Println("Database connection closed successfully")
		}
	}()

	ch, spinnerCtrl, tickerCtrl, unsub := SetupSubscriptions(ctx, cancel)
	defer unsub()

	app := NewApp(appVersion, spinnerCtrl, tickerCtrl)

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
		_, err := p.Run()
		resultChan <- err
	}()

	// Wait for either program completion or interrupt signal
	select {
	case err := <-resultChan:
		auxlog.Println("TUI program finished")
		cancel() // Cancel context to stop background goroutines
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
