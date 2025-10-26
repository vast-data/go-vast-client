package common

import (
	"fmt"
	"log"
	logging "vastix/internal/logging"
	"vastix/internal/msg_types"

	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
)

// WidgetNavigator uses empty interfaces to avoid type coupling
// The actual types will be asserted at runtime
type WidgetNavigator struct {
	mode                  NavigatorMode
	widget                Widget // Cross-reference to the parent widget
	NotAllowedListKeys    map[string]struct{}
	NotAllowedCreateKeys  map[string]struct{}
	NotAllowedDeleteKeys  map[string]struct{}
	NotAllowedDetailsKeys map[string]struct{}
	auxlog                *log.Logger // Add auxiliary logger
}

// NewWidgetNavigator creates a new WidgetNavigator with specified not-allowed keys for each mode.
// By default, ALL keys are allowed. Only keys in the notAllowed lists will be blocked.
//
// Example usage:
//
//	NewWidgetNavigator(
//	  []string{"ctrl+c"},     // Block ctrl+c in list mode
//	  []string{"esc", "/"},   // Block esc and / in create mode
//	  []string{},             // Allow all keys in delete mode
//	  []string{"tab"},        // Block tab in details mode
//	)
func NewWidgetNavigator(
	notAllowedListKeys []string,
	notAllowedCreateKeys []string,
	notAllowedDeleteKeys []string,
	notAllowedDetailsKeys []string,
) *WidgetNavigator {
	wn := &WidgetNavigator{
		mode:                  NavigatorModeList,
		NotAllowedListKeys:    make(map[string]struct{}),
		NotAllowedCreateKeys:  make(map[string]struct{}),
		NotAllowedDeleteKeys:  make(map[string]struct{}),
		NotAllowedDetailsKeys: make(map[string]struct{}),
		auxlog:                logging.GetAuxLogger(),
	}

	// Convert slices to maps for O(1) lookup
	for _, key := range notAllowedListKeys {
		wn.NotAllowedListKeys[key] = struct{}{}
	}
	for _, key := range notAllowedCreateKeys {
		wn.NotAllowedCreateKeys[key] = struct{}{}
	}
	for _, key := range notAllowedDeleteKeys {
		wn.NotAllowedDeleteKeys[key] = struct{}{}
	}
	for _, key := range notAllowedDetailsKeys {
		wn.NotAllowedDetailsKeys[key] = struct{}{}
	}

	return wn
}

func (wn *WidgetNavigator) SetWidget(widget Widget) {
	wn.widget = widget
}

func (wn *WidgetNavigator) GetWidget() Widget {
	return wn.widget
}

func (wn *WidgetNavigator) SetMode(mode NavigatorMode) {
	if wn.mode == mode {
		wn.auxlog.Printf("🟦 WIDGET_NAVIGATOR: mode already set to %s", mode.String())
		return
	}
	wn.auxlog.Printf("🟦 WIDGET_NAVIGATOR: setting mode to %s for widget %T", mode.String(), wn.widget)
	wn.mode = mode
}

// isModeAllowed checks if a mode is allowed by the widget based on allowed/not-allowed lists
// Only one of GetAllowedNavigatorModes or GetNotAllowedNavigatorModes should return non-nil
func (wn *WidgetNavigator) isModeAllowed(mode NavigatorMode) bool {
	allowedModes := wn.widget.GetAllowedNavigatorModes()
	notAllowedModes := wn.widget.GetNotAllowedNavigatorModes()

	// Validate that developer only overrides one of the methods
	if allowedModes != nil && notAllowedModes != nil {
		panic(
			fmt.Sprintf(
				"WidgetNavigator: widget %T cannot override "+
					"both GetAllowedNavigatorModes and GetNotAllowedNavigatorModes - choose only one",
				wn.widget,
			),
		)
	}

	// If neither method is overridden (both return nil), allow all modes
	if allowedModes == nil && notAllowedModes == nil {
		return true
	}

	// If allowed modes is specified, check if mode is in the allowed list
	if allowedModes != nil {
		for _, allowedMode := range allowedModes {
			if allowedMode == mode {
				return true
			}
		}
		return false // Mode not found in allowed list
	}

	// If not-allowed modes is specified, check if mode is NOT in the not-allowed list
	if notAllowedModes != nil {
		for _, notAllowedMode := range notAllowedModes {
			if notAllowedMode == mode {
				return false // Mode found in not-allowed list
			}
		}
		return true // Mode not found in not-allowed list, so it's allowed
	}

	// Should never reach here, but default to allowing mode
	return true
}

func (wn *WidgetNavigator) SetModeMust(m NavigatorMode) {
	if !wn.isModeAllowed(m) {
		// Mode not supported → panic
		panic(fmt.Sprintf("WidgetNavigator: mode %s not supported by widget %T", m.String(), wn.widget))
	}

	wn.widget.SetMode(m)
}

func (wn *WidgetNavigator) GetMode() NavigatorMode {
	return wn.mode
}

func (wn *WidgetNavigator) setModeIfSupported(m NavigatorMode) {
	if wn.isModeAllowed(m) {
		wn.widget.SetMode(m)
	}
}

func (wn *WidgetNavigator) Navigate(msg tea.Msg) tea.Cmd {
	currentMode := wn.GetMode()
	logging.Debug("Navigate called", zap.String("mode", currentMode.String()), zap.Any("msg_type", msg))

	// Add trace logging for full navigation chain
	wn.auxlog.Printf("WIDGET_NAVIGATOR: widget=%T mode=%s msg=%T", wn.widget, currentMode.String(), msg)

	switch currentMode {
	case NavigatorModeList:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if _, ok := wn.NotAllowedListKeys[msg.String()]; ok {
				logging.Debug("Ignoring key in list mode", zap.String("key", msg.String()))
				return nil // Ignore keys that are not allowed in list mode
			}

			adapter, ok := any(wn.widget).(ListAdapter)
			if !ok {
				panic("WidgetNavigator: widget does not implement ListAdapter interface")
			}

			switch msg.String() {
			case "up", "k":
				adapter.MoveUp()
			case "down", "j":
				adapter.MoveDown()
			case "home":
				adapter.MoveHome()
			case "end":
				adapter.MoveEnd()
			case "pgup":
				adapter.PageUp()
			case "pgdn":
				adapter.PageDown()
			case "n":
				// Switch to create mode and initialize inputs
				wn.setModeIfSupported(NavigatorModeCreate)
			case "d":
				// Check if there's visible content in the list (after fuzzy filtering) before allowing details mode
				if listAdapterWidget, ok := any(wn.widget).(interface{ GetFilteredDataCount() int }); ok {
					filteredCount := listAdapterWidget.GetFilteredDataCount()
					if filteredCount == 0 {
						logging.Debug("Preventing details mode: No visible content in list after filtering")
						return nil // Ignore the 'd' key when filtered list is empty
					}
				}

				wn.widget.ClearFuzzyDetailsSearch()
				// Switch to details mode and trigger async details loading
				logging.Debug("Switching to details mode via 'd' key")
				wn.setModeIfSupported(NavigatorModeDetails)
				if adapter, ok := any(wn.widget).(DetailsAdapter); ok {
					return adapter.DetailsDo(wn.widget)
				} else {
					panic("WidgetNavigator: widget does not implement DetailsAdapter interface")
				}

			case "enter":
				// Check if there's visible content in the list (after fuzzy filtering) before allowing selection
				if listAdapterWidget, ok := any(wn.widget).(interface{ GetFilteredDataCount() int }); ok {
					filteredCount := listAdapterWidget.GetFilteredDataCount()
					if filteredCount == 0 {
						logging.Debug("Preventing selection: No visible content in list after filtering")
						return nil // Ignore the 'enter' key when filtered list is empty
					}
				}

				wn.widget.ClearFuzzyDetailsSearch()
				intfSatisfied := false
				if adapter, ok := any(wn.widget).(SelectAdapter); ok {
					intfSatisfied = true
					if cmd := adapter.SelectDo(wn.widget); cmd != nil {
						return cmd
					}
				}
				if adapter, ok := any(wn.widget).(DetailsAdapter); ok && adapter.DetailsOnSelect() {
					wn.SetModeMust(NavigatorModeDetails)
					return adapter.DetailsDo(wn.widget)
				}
				if !intfSatisfied {
					panic("WidgetNavigator: widget does not implement SelectAdapter or DetailsAdapter interface")
				}

			case "ctrl+d":
				// Switch to delete mode with confirmation
				logging.Debug("Switching to delete mode via 'ctrl+d' key")
				wn.setModeIfSupported(NavigatorModeDelete)
			case "x":
				if extraWidget, ok := wn.widget.(ExtraWidget); ok {
					if extraWidget.CanUseExtra() {
						wn.setModeIfSupported(NavigatorModeExtra)
						return msg_types.ProcessWithSpinner(extraWidget.Init)

					} else {
						logging.Debug("WidgetNavigator: extra widget is not available, ignoring 'x' key")
					}
				} else {
					panic("WidgetNavigator: widget does not implement ExtraWidget interface")
				}
			default:

				// Check for extra widget shortcuts
				if extraWidget, ok := wn.widget.(ExtraWidget); ok && extraWidget.CanUseExtra() {
					return wn.handleExtraWidgetShortcuts(msg.String())
				}
			}
		}

	case NavigatorModeCreate:
		logging.Debug("Navigate: in create mode")
		// Safety check - ensure inputs exist
		hasInputs := wn.widget.HasInputs()
		logging.Debug("Navigate: hasInputs check", zap.Bool("hasInputs", hasInputs))
		if !hasInputs {
			logging.Debug("Navigate: no inputs, returning nil")
			return nil
		}

		switch msg := msg.(type) {
		case tea.KeyMsg:
			logging.Debug("Navigate: processing key", zap.String("key", msg.String()))

			// Check if we're editing JSON
			if adapter, ok := any(wn.widget).(interface{ IsEditingJSON() bool }); ok && adapter.IsEditingJSON() {
				// In JSON editing mode, handle special keys
				switch msg.String() {
				case "ctrl+t":
					// Toggle back to form mode
					logging.Debug("Toggle back to form mode from JSON")
					if toggleAdapter, ok := any(wn.widget).(FormJSONToggleAdapter); ok {
						toggleAdapter.ToggleFormJSONMode()
					}
					return nil
				case "ctrl+s":
					// Submit from JSON mode - first save JSON to form, then submit
					logging.Debug("Submit from JSON mode")
					// Save JSON edits to form inputs first
					if saveAdapter, ok := any(wn.widget).(interface{ SaveJSONEdits() error }); ok {
						if err := saveAdapter.SaveJSONEdits(); err != nil {
							logging.Error("Failed to save JSON edits", zap.Error(err))
							return func() tea.Msg {
								return msg_types.ErrorMsg{Err: err}
							}
						}
					}
					// Now submit the form
					if adapter, ok := any(wn.widget).(CreateFromInputsAdapter); ok {
						return adapter.CreateFromInputsDo(wn.widget)
					}
					return nil
				default:
					// Forward all other keys to the textarea
					if textareaAdapter, ok := any(wn.widget).(interface{ UpdateJSONTextarea(tea.Msg) tea.Cmd }); ok {
						return textareaAdapter.UpdateJSONTextarea(msg)
					}
					return nil
				}
			}

			if _, ok := wn.NotAllowedCreateKeys[msg.String()]; ok {
				logging.Debug("Ignoring key in create mode", zap.String("key", msg.String()))
				return nil // Ignore keys that are not allowed in create mode
			}

			switch msg.String() {
			case "tab", "down":
				logging.Debug("Navigate: %s key pressed, calling NextInput", zap.String("key", msg.String()))
				// Move to next input
				if adapter, ok := any(wn.widget).(FormNavigateAdaptor); ok {
					logging.Debug("Navigate: FormNavigateAdaptor interface OK, calling NextInput")
					adapter.NextInput()
				} else {
					logging.Debug("Navigate: FormNavigateAdaptor interface FAILED")
					panic("WidgetNavigator: widget does not implement FormNavigateAdaptor interface")
				}
			case "shift+tab", "up":
				logging.Debug("Navigate: %s key pressed, calling PrevInput", zap.String("key", msg.String()))
				// Move to previous input
				if adapter, ok := any(wn.widget).(FormNavigateAdaptor); ok {
					logging.Debug("Navigate: FormNavigateAdaptor interface OK, calling PrevInput")
					adapter.PrevInput()
				} else {
					logging.Debug("Navigate: FormNavigateAdaptor interface FAILED")
					panic("WidgetNavigator: widget does not implement FormNavigateAdaptor interface")
				}
			case "ctrl+s":
				// Submit form using public GetInputs method
				if adapter, ok := any(wn.widget).(CreateFromInputsAdapter); ok {
					return adapter.CreateFromInputsDo(wn.widget)
				} else {
					panic("WidgetNavigator: widget does not implement CreateFromInputsAdapter interface")
				}
			case "esc":
				// Cancel form and return to list mode
				logging.Debug("Canceling create mode, returning to list mode")
				wn.widget.ResetCreateForm()
				wn.setModeIfSupported(NavigatorModeList)
			case "ctrl+t":
				// Toggle between form and JSON mode (auto-enters editing in JSON mode)
				logging.Debug("Toggle form/JSON mode")
				if adapter, ok := any(wn.widget).(FormJSONToggleAdapter); ok {
					adapter.ToggleFormJSONMode()
				} else {
					logging.Debug("Widget does not implement FormJSONToggleAdapter interface")
				}
			default:
				// Handle input for the currently focused field using public method
				if adapter, ok := any(wn.widget).(UpdateInputAdapter); ok {
					// Update the current input field with the message
					adapter.UpdateCurrentInput(msg)
				} else {
					panic("WidgetNavigator: widget does not implement UpdateInputAdapter interface")
				}
			}
		}

	case NavigatorModeDelete:
		// Handle delete confirmation
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if _, ok := wn.NotAllowedDeleteKeys[msg.String()]; ok {
				logging.Debug("Ignoring key in delete mode", zap.String("key", msg.String()))
				return nil // Ignore keys that are not allowed in delete mode
			}

			switch msg.String() {
			case "y", "Y", "enter":
				// Confirm deletion - call Delete method if widget supports it
				logging.Debug("Delete confirmed")
				if adapter, ok := any(wn.widget).(DeleteAdapter); ok {
					return msg_types.ProcessWithClearError(adapter.DeleteDo(wn.widget))
				} else {
					panic("WidgetNavigator: widget does not implement DeleteAdapter interface")
				}
			case "n", "N", "esc":
				// Cancel deletion and return to list mode
				logging.Debug("Delete canceled, returning to list mode")
				wn.widget.SetMode(NavigatorModeList)
				return msg_types.ProcessWithClearError(nil)
			}
		}

	case NavigatorModeDetails:
		// Handle details view navigation
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if _, ok := wn.NotAllowedDetailsKeys[msg.String()]; ok {
				logging.Debug("Ignoring key in details mode", zap.String("key", msg.String()))
				return nil // Ignore keys that are not allowed in details mode
			}

			switch msg.String() {
			case "n":
				// Switch to create mode and initialize inputs
				wn.setModeIfSupported(NavigatorModeCreate)
			case "ctrl+s":
				if adapter, ok := any(wn.widget).(CopyToClipboardAdapter); ok {
					return adapter.CopyToClipboard
				} else {
					panic("WidgetNavigator: widget does not implement CopyToClipboardAdapter interface")
				}
			case "esc":
				// Return to list mode
				logging.Debug("Details mode canceled, returning to list mode")
				wn.setModeIfSupported(NavigatorModeList)
				return func() tea.Msg {
					return msg_types.SetDataMsg{}
				}
			case "ctrl+r":
				// Refresh details view
				wn.widget.ClearFuzzyDetailsSearch()
				if adapter, ok := any(wn.widget).(DetailsAdapter); ok {
					return adapter.DetailsDo(wn.widget)
				}

			default:
				// Pass other keys (arrows, pgup/pgdn, etc.) to the details adapter for scrolling
				if adapter, ok := any(wn.widget).(ViewPortAdapter); ok {
					return adapter.UpdateViewPort(msg)
				} else {
					panic("WidgetNavigator: widget does not implement ViewPortAdapter interface")
				}
			}
		default:
			// Pass other messages to the details adapter
			if adapter, ok := any(wn.widget).(ViewPortAdapter); ok {
				return adapter.UpdateViewPort(msg)
			} else {
				panic("WidgetNavigator: widget does not implement ViewPortAdapter interface")
			}
		}

	case NavigatorModeExtra:
		// Handle extra navigator mode
		if extraNavigator, ok := wn.widget.(ExtraNavigator); ok {

			if cmd, accepted := extraNavigator.ExtraNavigate(msg); accepted {
				return cmd // Extra navigation handled, return command
			}

			// Check if this is ESC key that bubbled up from ExtraNavigator
			if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
				logging.Debug("ESC key bubbled up from ExtraNavigator, exiting extra mode")

				// Switch back to list mode
				wn.setModeIfSupported(NavigatorModeList)
				logging.Debug("Switched back to NavigatorModeList from extra mode")

				// Return command to refresh list data
				return func() tea.Msg {
					return msg_types.SetDataMsg{}
				}
			}
		} else {
			panic("WidgetNavigator: widget does not implement ExtraNavigator interface")
		}

	}

	return nil
}

// handleExtraWidgetShortcuts checks if the pressed key matches any extra widget shortcuts
// and activates the corresponding widget if found
func (wn *WidgetNavigator) handleExtraWidgetShortcuts(key string) tea.Cmd {
	shortcuts := wn.widget.ShortCuts()

	if widget, found := shortcuts[key]; found {
		logging.Debug("Extra widget shortcut matched",
			zap.String("key", key),
			zap.String("widget", widget.GetExtraResourceType()))

		// Switch to extra mode
		wn.SetModeMust(NavigatorModeExtra)

		extraResourceType := widget.GetExtraResourceType()
		// Use NewRowData instead of direct struct initialization for proper key handling
		rowData := NewRowData([]string{"action"}, []string{extraResourceType})

		// Call Select on the main widget (which implements DetailsWidget), not on the extra widget
		if detailsWidget, ok := any(wn.widget).(DetailsWidget); ok {
			if cmd, err := detailsWidget.Select(rowData); err != nil {
				logging.Debug("Failed to select extra widget",
					zap.String("resourceType", extraResourceType),
					zap.Error(err))
				return nil
			} else {
				// Return the command from Select and also initialize the specific extra widget
				return tea.Batch(cmd, msg_types.ProcessWithSpinner(widget.Init))
			}
		}

		// Fallback: just initialize the widget if Select is not available
		return msg_types.ProcessWithSpinner(widget.Init)
	}

	return nil
}
