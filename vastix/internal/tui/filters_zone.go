package tui

import (
	"vastix/internal/database"
	log "vastix/internal/logging"
	"vastix/internal/msg_types"
	"vastix/internal/tui/widgets/common"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.uber.org/zap"
)

// FiltersZone handles all search and filtering functionality
type FiltersZone struct {
	width, height int
	db            *database.Service // Database service
	currentWidget common.Widget

	// Search functionality
	searchMode  bool
	searchInput textinput.Model

	// Server-side filtering
	serverSearchMode  bool
	serverSearchInput textinput.Model

	// Callbacks to working zone
	updateSizes func() // Trigger size recalculation
}

// NewFiltersZone creates a new filters zone
func NewFiltersZone(db *database.Service, updateSizes func()) *FiltersZone {
	log.Debug("FiltersZone initializing")

	// Initialize fuzzy search input
	searchInput := textinput.New()
	searchInput.Placeholder = "Fuzzy Search..."
	searchInput.Focus()
	searchInput.CharLimit = 156
	searchInput.Width = 50

	// Initialize server search input with autocompletion
	serverSearchInput := textinput.New()
	serverSearchInput.Placeholder = "Query Params: guid=xxx, path__contains=/foobar, id__in=1,2,3 ... etc"
	serverSearchInput.CharLimit = 256
	serverSearchInput.Width = 50

	filtersZone := &FiltersZone{
		db:                db,
		searchMode:        false,
		searchInput:       searchInput,
		serverSearchMode:  false,
		serverSearchInput: serverSearchInput,
		updateSizes:       updateSizes,
	}

	log.Debug("FiltersZone initialized successfully")

	return filtersZone
}

func (*FiltersZone) Init() {}

// SetSize sets the dimensions of the filters zone
func (f *FiltersZone) SetSize(width, height int) {
	f.width = width
	f.height = height

	// Update input widths
	f.searchInput.Width = width
	f.serverSearchInput.Width = width
}

// IsActive returns true if any of search inputs are active
func (f *FiltersZone) IsActive() bool {
	return f.searchMode || f.serverSearchMode
}

// IsSearchable returns true if any search mode is active
func (f *FiltersZone) IsSearchable() bool {
	return f.currentWidget.IsServerSearchable() || f.currentWidget.IsFuzzySearchable()
}

// IsServerSearchable returns true if server-side search is supported
func (f *FiltersZone) IsServerSearchable() bool {
	return f.currentWidget.IsServerSearchable()
}

// IsFuzzySearchable returns true if fuzzy search is supported
func (f *FiltersZone) IsFuzzySearchable() bool {
	return f.currentWidget.IsFuzzySearchable()
}

// GetHeight returns the height needed when filters are visible
func (f *FiltersZone) GetHeight() int {
	if f.IsSearchable() {
		return 3 // Search input with border takes 3 lines
	}
	return 0
}

// Ready returns whether the filters zone is ready to be displayed
func (f *FiltersZone) Ready() bool {
	return true // Filters zone is always ready
}

// Update handles messages for the filters zone
func (f *FiltersZone) Update(msg tea.Msg) (*FiltersZone, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle fuzzy search mode
		if f.searchMode {
			switch msg.String() {
			case "ctrl+c":
				return f, tea.Quit
			case "esc":
				// Exit search mode
				f.searchMode = false
				f.searchInput.SetValue("")
				if f.updateSizes != nil {
					f.updateSizes()
				}
				return f, nil
			case "enter":
				// Apply search and exit search mode
				f.searchMode = false
				if f.updateSizes != nil {
					f.updateSizes()
				}
				return f, nil
			default:
				// Update search input
				f.searchInput, cmd = f.searchInput.Update(msg)
				cmds = append(cmds, cmd)

				// Apply filter to working zone
				searchValue := f.searchInput.Value()

				switch f.currentWidget.GetMode() {
				case common.NavigatorModeList:
					f.currentWidget.SetFuzzyListSearchString(searchValue)
				case common.NavigatorModeDetails:
					f.currentWidget.SetFuzzyDetailsSearchString(searchValue)
				case common.NavigatorModeExtra:
					switch f.currentWidget.(common.ExtraWidget).GetExtraMode() {
					case common.ExtraNavigatorModeList:
						f.currentWidget.SetFuzzyListSearchString(searchValue)
					case common.ExtraNavigatorModeDetails:
						f.currentWidget.SetFuzzyDetailsSearchString(searchValue)
					default:
						panic("Unsupported extra mode for search: " + f.currentWidget.(common.ExtraWidget).GetExtraMode().String())
					}
				default:
					panic("Unsupported mode for search: " + f.currentWidget.GetMode().String())
				}
				return f, tea.Batch(cmds...)
			}
		}

		// Handle server search mode
		if f.serverSearchMode {
			switch msg.String() {
			case "ctrl+c":
				return f, tea.Quit
			case "esc":
				// Exit server search mode
				f.serverSearchMode = false
				f.serverSearchInput.SetValue("")
				if f.updateSizes != nil {
					f.updateSizes()
				}
				return f, nil
			case "enter":
				// Apply server search and exit mode
				serverQuery := f.serverSearchInput.Value()

				if _, err := common.ConvertServerParamsToVastParams(serverQuery); err != nil {
					return f, func() tea.Msg {
						return msg_types.ErrorMsg{
							Err: err,
						}
					}
				}

				f.serverSearchMode = false
				// Get server query string and set it directly
				f.currentWidget.SetServerSearchParams(serverQuery)

				if f.updateSizes != nil {
					f.updateSizes()
				}

				setDataFn := msg_types.ProcessWithSpinner(f.currentWidget.SetListData)
				return f, msg_types.ProcessWithClearError(setDataFn)
			default:
				// Update server search input
				f.serverSearchInput, cmd = f.serverSearchInput.Update(msg)
				cmds = append(cmds, cmd)
				return f, tea.Batch(cmds...)
			}
		}

		// Handle key bindings to enter search modes
		switch msg.String() {
		case "/":
			// Enter fuzzy search mode
			f.searchMode = true
			switch f.currentWidget.GetMode() {
			case common.NavigatorModeList:
				f.searchInput.SetValue(f.currentWidget.GetFuzzyListSearchString())
			case common.NavigatorModeDetails:
				f.searchInput.SetValue(f.currentWidget.GetFuzzyDetailsSearchString())
			case common.NavigatorModeExtra:
				switch f.currentWidget.(common.ExtraWidget).GetExtraMode() {
				case common.ExtraNavigatorModeList:
					f.searchInput.SetValue(f.currentWidget.GetFuzzyListSearchString())
				case common.ExtraNavigatorModeDetails:
					f.searchInput.SetValue(f.currentWidget.GetFuzzyDetailsSearchString())
				default:
					panic("Unsupported extra mode for search: " + f.currentWidget.(common.ExtraWidget).GetExtraMode().String())
				}
			}

			f.searchInput.Focus()
			f.searchInput.CursorEnd()

			if f.updateSizes != nil {
				f.updateSizes()
			}
			return f, nil
		case "?":
			// Enter server search mode (shift+/)
			f.serverSearchMode = true
			f.serverSearchInput.Focus()
			f.serverSearchInput.SetValue(f.currentWidget.GetServerSearchParams())

			if f.updateSizes != nil {
				f.updateSizes()
			}
			return f, nil
		}
	}

	return f, tea.Batch(cmds...)
}

// View renders the filters zone
func (f *FiltersZone) View() string {
	if !f.IsSearchable() {
		return ""
	}

	prefix := "🔍 "
	// Render search input if in search mode
	if f.searchMode {
		searchStyle := lipgloss.NewStyle().
			Width(f.width-2).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("2")). // Green border
			Padding(0, 1)

		return searchStyle.Render(prefix + f.searchInput.View())
	}

	// Render server search input if in server search mode
	if f.serverSearchMode {
		serverSearchStyle := lipgloss.NewStyle().
			Width(f.width-2).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("214")). // Yellow border (same as resource name)
			Padding(0, 1)

		return serverSearchStyle.Render(prefix + f.serverSearchInput.View())
	}

	return ""
}

// SetCurrentWidget sets the current widget that the filters will operate on
func (f *FiltersZone) SetCurrentWidget(widget common.Widget) {
	f.currentWidget = widget
}

// ResetFilters clears all active filters and search state
func (f *FiltersZone) ResetFilters() {
	// Clear search modes
	f.searchMode = false
	f.serverSearchMode = false

	// Clear search input values
	f.searchInput.SetValue("")
	f.serverSearchInput.SetValue("")

	// Clear filters on current widget if it exists
	if f.currentWidget != nil {
		f.currentWidget.ClearFilters()
		log.Debug("Filters reset for current widget", zap.String("widget_type", f.currentWidget.GetResourceType()))
	}

	// Update sizes to hide search UI
	if f.updateSizes != nil {
		f.updateSizes()
	}

	log.Debug("All filters and search state reset")
}
