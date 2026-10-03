package tui

import (
	"strings"
	"vastix/internal/colors"
	"vastix/internal/tui/widgets/common"

	"go.uber.org/zap"

	"vastix/internal/database"
	log "vastix/internal/logging"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// KeybindingsZone represents the keyboard shortcuts information zone
type KeybindingsZone struct {
	width, height  int
	keyBindings    []common.KeyBinding
	db             *database.Service
	getKeyBindings func() []common.KeyBinding // Getter function for dynamic keybindings
}

// NewKeybindingsZone creates a new keybindings zone with logging support
func NewKeybindingsZone(db *database.Service) *KeybindingsZone {
	log.Debug("KeybindingsZone initializing")

	keybindings := &KeybindingsZone{
		db: db,
	}

	log.Debug("KeybindingsZone initialized successfully")

	return keybindings
}

func (k *KeybindingsZone) Init() {}

// SetSize sets the dimensions of the keybindings zone
func (k *KeybindingsZone) SetSize(width, height int) {
	k.width = width
	k.height = height
}

// Update handles messages for the keybindings zone
func (k *KeybindingsZone) Update(msg tea.Msg) (*KeybindingsZone, tea.Cmd) {
	return k, nil
}

func (k *KeybindingsZone) SetKeyBindings(kb []common.KeyBinding) {
	k.keyBindings = kb
	if len(k.keyBindings) == 0 {
		log.Warn("No keybindings available to display")
	} else {
		log.Debug("Keybindings set successfully", zap.Int("count", len(k.keyBindings)))
	}
}

// SetKeyBindingsGetter sets the getter function for dynamic keybindings
func (k *KeybindingsZone) SetKeyBindingsGetter(getter func() []common.KeyBinding) {
	k.getKeyBindings = getter
	log.Debug("Keybindings getter function set for dynamic updates")
}

// View renders the keybindings zone
func (k *KeybindingsZone) View() string {
	if k.width == 0 {
		return ""
	}

	// All key hints use the same yellow. Generic and extra-action markers
	// are kept for grouping / future styling, but they no longer change color.
	keyStyle := lipgloss.NewStyle().
		Foreground(colors.Yellow).
		Bold(true)

	widgetDescStyle := lipgloss.NewStyle().
		Foreground(colors.White)

	extraActionVerbStyle := lipgloss.NewStyle().
		Foreground(colors.OffWhite)

	// Get current keybindings dynamically using getter function if available
	currentKeyBindings := k.keyBindings
	if k.getKeyBindings != nil {
		currentKeyBindings = k.getKeyBindings()
	}

	// Separate keybindings into generic and widget-specific
	var genericBindings []common.KeyBinding
	var widgetBindings []common.KeyBinding

	for _, kb := range currentKeyBindings {
		if kb.Generic {
			genericBindings = append(genericBindings, kb)
		} else {
			widgetBindings = append(widgetBindings, kb)
		}
	}

	// Combine all keybindings: generic first, then widget-specific
	allBindings := append(genericBindings, widgetBindings...)

	if len(allBindings) == 0 {
		return ""
	}

	// Never more than 2 columns. Split items evenly across those columns
	// (ceil(n/2) rows) so a third column is never created. Cap by available
	// height when known so the hints stay within the profile header band.
	const maxColumns = 2
	maxRows := 7
	if k.height > 0 {
		maxRows = k.height
	}
	capacity := maxColumns * maxRows
	if len(allBindings) > capacity {
		allBindings = allBindings[:capacity]
	}
	itemsPerColumn := (len(allBindings) + maxColumns - 1) / maxColumns
	if itemsPerColumn < 1 {
		itemsPerColumn = 1
	}

	var columns []string
	for i := 0; i < len(allBindings); i += itemsPerColumn {
		end := i + itemsPerColumn
		if end > len(allBindings) {
			end = len(allBindings)
		}
		// Hard stop at 2 columns even if math drifts.
		if len(columns) >= maxColumns {
			break
		}

		// Find the longest key in THIS specific column
		var maxKeyLenInColumn int
		for j := i; j < end; j++ {
			kb := allBindings[j]
			if len(kb.Key) > maxKeyLenInColumn {
				maxKeyLenInColumn = len(kb.Key)
			}
		}

		var columnItems []string
		for j := i; j < end; j++ {
			kb := allBindings[j]
			// Left-align the key with padding to match the longest key width in THIS column
			keyText := lipgloss.NewStyle().
				Width(maxKeyLenInColumn).
				Align(lipgloss.Left).
				Render(kb.Key)

			var item string
			if kb.IsExtraAction {
				item = lipgloss.JoinHorizontal(lipgloss.Left,
					keyStyle.Render(keyText),
					renderExtraActionDesc(kb.Desc, widgetDescStyle, extraActionVerbStyle),
				)
			} else {
				item = lipgloss.JoinHorizontal(lipgloss.Left,
					keyStyle.Render(keyText),
					widgetDescStyle.Render(" "+kb.Desc),
				)
			}
			columnItems = append(columnItems, item)
		}

		columns = append(columns, strings.Join(columnItems, "\n"))
	}

	// Join all columns horizontally with separators
	if len(columns) == 0 {
		return ""
	}
	if len(columns) == 1 {
		return columns[0]
	}
	// Join with "    " separator between each column
	result := columns[0]
	for i := 1; i < len(columns); i++ {
		result = lipgloss.JoinHorizontal(lipgloss.Top, result, "    ", columns[i])
	}
	return result
}

// renderExtraActionDesc styles extra-action hints as "verb:path" with a lighter verb prefix.
func renderExtraActionDesc(desc string, pathStyle, verbStyle lipgloss.Style) string {
	idx := strings.Index(desc, ":")
	if idx < 0 {
		return pathStyle.Render(" " + desc)
	}
	return lipgloss.JoinHorizontal(lipgloss.Left,
		pathStyle.Render(" "),
		verbStyle.Render(desc[:idx+1]),
		pathStyle.Render(desc[idx+1:]),
	)
}

// Ready returns whether the keybindings zone is ready to be displayed
func (k *KeybindingsZone) Ready() bool {
	// Keybindings zone is always ready since it's static
	return true
}
