package tui

import (
	"strings"
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

	// Styles for generic keybindings (different color)
	genericKeyStyle := lipgloss.NewStyle().
		Foreground(LightBlue).
		Bold(true)

	// Styles for widget-specific keybindings (current colors)
	widgetKeyStyle := lipgloss.NewStyle().
		Foreground(Yellow).
		Bold(true)

	widgetDescStyle := lipgloss.NewStyle().
		Foreground(LightGrey)

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

	// Build columns with max 5 items per column
	const itemsPerColumn = 5
	var columns []string

	for i := 0; i < len(allBindings); i += itemsPerColumn {
		end := i + itemsPerColumn
		if end > len(allBindings) {
			end = len(allBindings)
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

			// Use different styles based on whether it's generic or widget-specific
			var item string
			if kb.Generic {
				item = lipgloss.JoinHorizontal(lipgloss.Left,
					genericKeyStyle.Render(keyText),
					widgetDescStyle.Render(" "+kb.Desc),
				)
			} else {
				item = lipgloss.JoinHorizontal(lipgloss.Left,
					widgetKeyStyle.Render(keyText),
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
	} else if len(columns) == 1 {
		return columns[0]
	} else {
		// Join with "    " separator between each column
		result := columns[0]
		for i := 1; i < len(columns); i++ {
			result = lipgloss.JoinHorizontal(lipgloss.Top, result, "    ", columns[i])
		}
		return result
	}
}

// Ready returns whether the keybindings zone is ready to be displayed
func (k *KeybindingsZone) Ready() bool {
	// Keybindings zone is always ready since it's static
	return true
}
