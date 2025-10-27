package common

import (
	"fmt"
	stdlog "log"
	"vastix/internal/logging"
	"vastix/internal/msg_types"

	tea "github.com/charmbracelet/bubbletea"
)

// Removed duplicate NavigatorMode definitions to avoid conflicts with main navigator

// ExtraWidgetNavigator uses empty interfaces to avoid type coupling
// The actual types will be asserted at runtime
type ExtraWidgetNavigator struct {
	mode                  ExtraNavigatorMode
	widget                ExtraWidget
	parentNavigator       *WidgetNavigator // Reference to the navigator or main widget.
	NotAllowedListKeys    map[string]struct{}
	NotAllowedCreateKeys  map[string]struct{}
	NotAllowedDeleteKeys  map[string]struct{}
	NotAllowedDetailsKeys map[string]struct{}
	auxlog                *stdlog.Logger
}

func NewExtraWidgetNavigator(
	widget ExtraWidget,
	parentNavigator *WidgetNavigator,
	notAllowedListKeys []string,
	notAllowedCreateKeys []string,
	notAllowedDeleteKeys []string,
	notAllowedDetailsKeys []string,
) *ExtraWidgetNavigator {
	wn := &ExtraWidgetNavigator{
		widget:                widget,
		parentNavigator:       parentNavigator,
		mode:                  ExtraNavigatorModeList,
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

func (wn *ExtraWidgetNavigator) SetParentNavigatorMode(mode NavigatorMode) {
	wn.parentNavigator.SetModeMust(mode)
}

func (wn *ExtraWidgetNavigator) GetParentNavigatorMode() NavigatorMode {
	return wn.parentNavigator.GetMode()
}

func (wn *ExtraWidgetNavigator) SetExtraWidget(widget ExtraWidget) {
	wn.widget = widget
}

func (wn *ExtraWidgetNavigator) GetExtraWidget() ExtraWidget {
	return wn.widget
}

func (wn *ExtraWidgetNavigator) SetExtraMode(mode ExtraNavigatorMode) {
	if wn.mode == mode {
		wn.auxlog.Printf("EXTRA WIDGET NAVIGATOR mode already set to %s", mode.String())
		return
	}
	wn.mode = mode

	// Initialize CreateAdapter inputs when switching to create mode (similar to main widget's SetMode)
	if mode == ExtraNavigatorModeCreate {
		wn.auxlog.Printf("🔄 Checking if we can initialize inputs for extra create mode")
		// Check if there's an active widget first (to avoid the panic)
		if hasActiveWidget, ok := wn.widget.(interface{ HasActiveWidget() bool }); ok && hasActiveWidget.HasActiveWidget() {
			wn.auxlog.Printf("Active widget available - initializing inputs")
			// Always refresh inputs to ensure fresh state every time
			if inputsWidget, ok := wn.widget.(interface{ GetInputs() (Inputs, error) }); ok {
				if createWidget, ok := wn.widget.(interface{ SetInputs(Inputs) }); ok {
					inputs, err := inputsWidget.GetInputs()
					if err != nil {
						wn.auxlog.Printf("Failed to get inputs for extra create mode: %v", err)
						return
					}
					createWidget.SetInputs(inputs)

					// Reset the form to clear any previous values
					if resetWidget, ok := wn.widget.(interface{ ResetCreate() }); ok {
						resetWidget.ResetCreate()
						wn.auxlog.Printf("Extra create form reset for fresh state")
					}

					wn.auxlog.Printf("Extra create inputs initialized and reset for navigation")
				}
			}
		} else {
			wn.auxlog.Printf("No active widget yet - inputs will be initialized later")
		}
	}
}

func (wn *ExtraWidgetNavigator) SetExtraModeMust(m ExtraNavigatorMode) {
	allowedModes := wn.widget.GetAllowedExtraNavigatorModes()
	if allowedModes == nil {
		// If widget does not restrict modes, allow all
		wn.widget.SetExtraMode(m)
		return
	}
	for _, allowedMode := range allowedModes {
		if allowedMode == m {
			wn.widget.SetExtraMode(m)
			return
		}
	}

	// Mode not supported → panic
	panic(fmt.Sprintf("ExtraWidgetNavigator: mode %s not supported by widget %T", m.String(), wn.widget))
}

func (wn *ExtraWidgetNavigator) GetExtraMode() ExtraNavigatorMode {
	return wn.mode
}

func (wn *ExtraWidgetNavigator) setModeIfSupported(m ExtraNavigatorMode) {
	allowedModes := wn.widget.GetAllowedExtraNavigatorModes()
	if allowedModes == nil {
		wn.widget.SetExtraMode(m)
		return
	}
	for _, allowedMode := range allowedModes {
		if allowedMode == m {
			wn.widget.SetExtraMode(m)
			return
		}
	}
}

// handleEscKey provides common ESC handling for all extra modes
func (wn *ExtraWidgetNavigator) handleEscKey() (tea.Cmd, bool) {
	wn.auxlog.Println("ESC pressed in extra widget, letting it bubble up to WidgetNavigator")
	wn.widget.Reset()
	return nil, false // Don't handle it here, let it bubble up
}

func (wn *ExtraWidgetNavigator) ExtraNavigate(msg tea.Msg) (tea.Cmd, bool) {
	currentMode := wn.widget.GetExtraMode()

	switch currentMode {
	case ExtraNavigatorModeList:

		switch msg := msg.(type) {
		case tea.KeyMsg:

			if _, ok := wn.NotAllowedListKeys[msg.String()]; ok {
				wn.auxlog.Printf("Ignoring key in list mode: %s", msg.String())
				return nil, false // Ignore keys that are not allowed in list mode
			}

			adapter, ok := any(wn.widget).(ListAdapter)
			if !ok {
				panic("ExtraWidgetNavigator: widget does not implement ListAdapter interface")
			}

			switch msg.String() {
			case "up", "k":
				adapter.MoveUp()
				return nil, true
			case "down", "j":
				adapter.MoveDown()
				return nil, true
			case "home":
				adapter.MoveHome()
				return nil, true
			case "end":
				adapter.MoveEnd()
				return nil, true
			case "pgup":
				adapter.PageUp()
				return nil, true
			case "pgdn":
				adapter.PageDown()
				return nil, true
			case "n":
				// Switch to create mode and initialize inputs
				wn.setModeIfSupported(ExtraNavigatorModeCreate)
				return nil, true

			case "enter":
				wn.widget.ClearFuzzyDetailsSearch()
				if adapter, ok := any(wn.widget).(SelectAdapter); ok {
					// Store the current mode before selection
					currentMode := wn.widget.GetExtraMode()
					if cmd := adapter.SelectDo(wn.widget); cmd != nil {
						return cmd, true
					}
					// Check if the mode changed after selection (widget wants specific mode)
					newMode := wn.widget.GetExtraMode()
					if newMode != currentMode {
						wn.auxlog.Printf("Mode changed after selection from %s to %s, respecting widget preference", currentMode.String(), newMode.String())
						return nil, true
					}
				}
				// Switch to details mode and trigger async details loading
				wn.auxlog.Println("Switching to details mode via 'enter' key")
				wn.setModeIfSupported(ExtraNavigatorModeDetails)
				if wn.GetExtraMode() != ExtraNavigatorModeDetails {
					// Switching to sub-action. No-op
					return nil, true
				}

				if adapter, ok := any(wn.widget).(DetailsAdapter); ok {
					return adapter.DetailsDo(wn.widget), true
				} else {
					panic("WidgetNavigator: widget does not implement DetailsAdapter interface")
				}

			case "ctrl+d":
				// Switch to delete mode with confirmation
				wn.auxlog.Println("Switching to delete mode via 'ctrl+d' key")
				wn.setModeIfSupported(ExtraNavigatorModeDelete)
				return nil, true
			case "esc":
				return wn.handleEscKey()
			}
		}

	case ExtraNavigatorModeCreate:
		hasInputs := wn.widget.HasInputs()
		if !hasInputs {
			return nil, false
		}

		switch msg := msg.(type) {
		case tea.KeyMsg:
			// Check if we're editing JSON
			if adapter, ok := any(wn.widget).(interface{ IsEditingJSON() bool }); ok && adapter.IsEditingJSON() {
				// In JSON editing mode, handle special keys
				switch msg.String() {
				case "ctrl+t":
					// Toggle back to form mode
					wn.auxlog.Println("Toggle back to form mode from JSON")
					if toggleAdapter, ok := any(wn.widget).(interface{ ToggleFormJSONMode() }); ok {
						toggleAdapter.ToggleFormJSONMode()
					}
					return nil, true
				case "ctrl+s":
					// Submit from JSON mode - first save JSON to form, then submit
					wn.auxlog.Println("Submit from JSON mode")
					// Save JSON edits to form inputs first
					if saveAdapter, ok := any(wn.widget).(interface{ SaveJSONEdits() error }); ok {
						if err := saveAdapter.SaveJSONEdits(); err != nil {
							wn.auxlog.Printf("Failed to save JSON edits: %v", err)
							return func() tea.Msg {
								return msg_types.ErrorMsg{Err: err}
							}, true
						}
					}
					// Now submit the form
					if adapter, ok := any(wn.widget).(CreateFromInputsAdapter); ok {
						wn.widget.ClearFuzzyDetailsSearch()
						cmd := adapter.CreateFromInputsDo(wn.widget)
						return cmd, true
					}
					return nil, true
				default:
					// Forward all other keys to the textarea
					if textareaAdapter, ok := any(wn.widget).(interface{ UpdateJSONTextarea(tea.Msg) tea.Cmd }); ok {
						return textareaAdapter.UpdateJSONTextarea(msg), true
					}
					return nil, true
				}
			}

			if _, ok := wn.NotAllowedCreateKeys[msg.String()]; ok {
				return nil, false // Ignore keys that are not allowed in create mode
			}

			switch msg.String() {
			case "tab", "down":
				// Move to next input
				if adapter, ok := any(wn.widget).(FormNavigateAdaptor); ok {
					adapter.NextInput()
					return nil, true
				} else {
					panic("ExtraWidgetNavigator: widget does not implement FormNavigateAdaptor interface")
				}
			case "shift+tab", "up":
				// Move to previous input
				if adapter, ok := any(wn.widget).(FormNavigateAdaptor); ok {
					adapter.PrevInput()
					return nil, true
				} else {
					panic("ExtraWidgetNavigator: widget does not implement FormNavigateAdaptor interface")
				}
			case "ctrl+s":
				// Submit form using public GetInputs method (matches normal widget behavior)
				if adapter, ok := any(wn.widget).(CreateFromInputsAdapter); ok {
					wn.widget.ClearFuzzyDetailsSearch()
					cmd := adapter.CreateFromInputsDo(wn.widget)
					return cmd, true
				} else {
					panic("ExtraWidgetNavigator: widget does not implement CreateFromInputsAdapter interface")
				}
			case "esc":
				return wn.handleEscKey()
			case "ctrl+t":
				// Toggle between form and JSON mode
				wn.auxlog.Printf("Toggle form/JSON mode")
				if adapter, ok := any(wn.widget).(FormJSONToggleAdapter); ok {
					adapter.ToggleFormJSONMode()
					return nil, true
				} else {
					wn.auxlog.Printf("Widget does not implement FormJSONToggleAdapter interface")
					return nil, true
				}
			default:
				// Check for system keys that should bubble up to the main app
				switch msg.String() {
				case "ctrl+c":
					return nil, false // Let system keys bubble up to main app
				default:
					// Handle input for the currently focused field using public method
					if adapter, ok := any(wn.widget).(UpdateInputAdapter); ok {
						// Update the current input field with the message
						adapter.UpdateCurrentInput(msg)
						return nil, true
					} else {
						panic("ExtraWidgetNavigator: widget does not implement UpdateInputAdapter interface")
					}
				}
			}
		default:
			wn.auxlog.Printf("DEBUG ExtraNavigate: non-KeyMsg received in create mode: %T", msg)
		}

	case ExtraNavigatorModeDelete:
		// Handle delete confirmation
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if _, ok := wn.NotAllowedDeleteKeys[msg.String()]; ok {
				wn.auxlog.Printf("Ignoring key in delete mode: %s", msg.String())
				return nil, false // Ignore keys that are not allowed in delete mode
			}

			switch msg.String() {
			case "left", "right", "tab":
				// Toggle between Yes and No buttons
				wn.auxlog.Printf("Toggle button selection in delete mode: %s", msg.String())
				if adapter, ok := any(wn.widget).(PromptToggleAdapter); ok {
					adapter.TogglePromptSelection()
				}
				return nil, true
			case "y", "Y":
				// Always confirm deletion when Y is pressed
				wn.auxlog.Println("Delete confirmed via Y key")
				if adapter, ok := any(wn.widget).(DeleteAdapter); ok {
					cmd := msg_types.ProcessWithClearError(adapter.DeleteDo(wn.widget))
					wn.setModeIfSupported(ExtraNavigatorModeList)
					return cmd, true
				} else {
					panic("ExtraWidgetNavigator: widget does not implement DeleteAdapter interface")
				}
			case "n", "N":
				// Always cancel deletion when N is pressed
				wn.auxlog.Println("Delete canceled via N key, returning to list mode")
				wn.setModeIfSupported(ExtraNavigatorModeList)
				return msg_types.ProcessWithClearError(nil), true
			case "esc":
				return wn.handleEscKey()
			case "enter":
				// Respect button selection when Enter is pressed
				wn.auxlog.Println("Delete prompt: enter key pressed, checking button selection")
				if adapter, ok := any(wn.widget).(PromptSelectionAdapter); ok {
					if adapter.IsPromptNoSelected() {
						// No is selected, cancel
						wn.auxlog.Println("Delete canceled via Enter (No selected), returning to list mode")
						wn.setModeIfSupported(ExtraNavigatorModeList)
						return msg_types.ProcessWithClearError(nil), true
					} else {
						// Yes is selected, confirm deletion
						wn.auxlog.Println("Delete confirmed via Enter (Yes selected)")
						if deleteAdapter, ok := any(wn.widget).(DeleteAdapter); ok {
							cmd := msg_types.ProcessWithClearError(deleteAdapter.DeleteDo(wn.widget))
							wn.setModeIfSupported(ExtraNavigatorModeList)
							return cmd, true
						} else {
							panic("ExtraWidgetNavigator: widget does not implement DeleteAdapter interface")
						}
					}
				} else {
					// Fallback to old behavior if adapter not implemented (confirm deletion)
					wn.auxlog.Println("Delete confirmed via Enter (fallback)")
					if deleteAdapter, ok := any(wn.widget).(DeleteAdapter); ok {
						cmd := msg_types.ProcessWithClearError(deleteAdapter.DeleteDo(wn.widget))
						wn.setModeIfSupported(ExtraNavigatorModeList)
						return cmd, true
					} else {
						panic("ExtraWidgetNavigator: widget does not implement DeleteAdapter interface")
					}
				}
			}
		}

	case ExtraNavigatorModeDetails:
		// Handle details view navigation
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if _, ok := wn.NotAllowedDetailsKeys[msg.String()]; ok {
				wn.auxlog.Printf("Ignoring key in details mode: %s", msg.String())
				return nil, false // Ignore keys that are not allowed in details mode
			}

			switch msg.String() {
			case "ctrl+s":
				if adapter, ok := any(wn.widget).(CopyToClipboardAdapter); ok {
					return adapter.CopyToClipboard, true
				} else {
					panic("ExtraWidgetNavigator: widget does not implement CopyToClipboardAdapter interface")
				}
			case "ctrl+e":
				// Go back to create mode to allow editing and resubmitting
				wn.auxlog.Println("Switching back to create mode to edit parameters")
				wn.setModeIfSupported(ExtraNavigatorModeCreate)
				return nil, true
			case "esc":
				return wn.handleEscKey()
			default:
				// Pass other keys (arrows, pgup/pgdn, etc.) to the details adapter for scrolling
				if adapter, ok := any(wn.widget).(ViewPortAdapter); ok {
					return adapter.UpdateViewPort(msg), true
				} else {
					panic("ExtraWidgetNavigator: widget does not implement ViewPortAdapter interface")
				}
			}
		default:
			// Pass other messages to the details adapter
			if adapter, ok := any(wn.widget).(ViewPortAdapter); ok {
				return adapter.UpdateViewPort(msg), true
			} else {
				panic("ExtraWidgetNavigator: widget does not implement ViewPortAdapter interface")
			}
		}

	case ExtraNavigatorModePrompt:
		// Handle prompt confirmation
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "y", "Y", "enter":
				// Confirm prompt action - call custom action handler
				wn.auxlog.Println("Prompt confirmed")
				if adapter, ok := any(wn.widget).(CreateFromInputsAdapter); ok {
					// Use CreateFromInputsAdapter to handle the confirmed action
					cmd := adapter.CreateFromInputsDo(wn.widget)

					wn.setModeIfSupported(ExtraNavigatorModeDetails)
					return cmd, true
				} else {
					panic("ExtraWidgetNavigator: widget does not implement CreateFromInputsAdapter interface")
				}
			case "n", "N":
				// Cancel prompt and return to list mode
				wn.auxlog.Println("Prompt canceled, returning to list mode")
				wn.setModeIfSupported(ExtraNavigatorModeList)
				return nil, true
			case "esc":
				return wn.handleEscKey()
			}
		}
	}

	return nil, false
}
