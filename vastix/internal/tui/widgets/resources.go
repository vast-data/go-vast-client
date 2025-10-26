package widgets

import (
	"fmt"
	"sort"
	"strings"
	shared "vastix/internal/common"
	"vastix/internal/database"
	"vastix/internal/msg_types"
	"vastix/internal/tui/widgets/common"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.uber.org/zap"
)

var SupportedResources = []string{}
var customResources = shared.NewSet[string](
	[]string{
		"profiles",
		"ssh_connections",
		"user_keys [local store]",
		"api_tokens [local store]",
	},
)

type Resources struct {
	*BaseWidget
}

// NewResources creates a new resources widget
func NewResources(db *database.Service) common.Widget {
	resourceType := "resources"
	listHeaders := []string{"type"}

	extraNav := []common.ExtraWidget{}

	keyRestrictions := &common.NavigatorKeyRestrictions{
		Main: common.KeyRestrictions{
			NotAllowedListKeys: []string{"d"}, // Block extra actions in main list mode
		},
		Extra: common.NewDefaultKeyRestrictions(), // No restrictions for extra widgets
	}

	widget := &Resources{
		NewBaseWidget(db, listHeaders, nil, resourceType, extraNav, keyRestrictions),
	}

	widget.SetParentForBaseWidget(widget, false)
	return widget
}

func (r *Resources) SetListData() tea.Msg {
	// Get all available resource types from the registered widgets
	supportedResources := SupportedResources

	// Sort alphabetically, but keep "profiles" first and "ssh_connections" second (both not sortable)
	var sortedResources []string
	var otherResources []string

	for _, resource := range supportedResources {
		if customResources.Contains(resource) {
			continue
		}
		otherResources = append(otherResources, resource)
	}

	sort.Strings(otherResources)

	sortedResources = append(sortedResources, customResources.ToOrderedSlice()...)
	sortedResources = append(sortedResources, otherResources...)

	// Convert to data format
	data := make([][]string, 0, len(sortedResources))
	for _, resource := range sortedResources {
		data = append(data, []string{resource})
	}

	r.ListAdapter.SetListData(data, r.GetFuzzyListSearchString())
	return msg_types.MockMsg{}
}

func (Resources) GetNotAllowedNavigatorModes() []common.NavigatorMode {
	return []common.NavigatorMode{
		common.NavigatorModeCreate,
		common.NavigatorModeDetails,
	}
}

func (r *Resources) GetKeyBindings() []common.KeyBinding {
	// Only list mode keybindings, no create/details/delete
	keyBindings := []common.KeyBinding{
		{Key: "</>", Desc: "search", Generic: true},
		{Key: "<↑/↓>", Desc: "navigate"},
		{Key: "<enter>", Desc: "select"},
	}
	return keyBindings
}

// Select implements the Selectable interface for resource type switching
func (r *Resources) Select(selectedRowData common.RowData) (tea.Cmd, error) {
	if selectedRowData.Len() == 0 {
		return nil, fmt.Errorf("no resource selected")
	}

	r.log.Debug("Resource selection requested",
		zap.Any("rowData", selectedRowData),
		zap.Int("dataLength", selectedRowData.Len()))

	// Extract resource type from the row data
	resourceType := strings.TrimSpace(strings.ToLower(selectedRowData.GetID()))
	if resourceType == "" {
		return nil, fmt.Errorf("invalid resource data: missing type")
	}

	r.log.Info("Switching to resource type", zap.String("resourceType", resourceType))

	// Return a command to switch resource type
	return func() tea.Msg {
		return msg_types.SetResourceTypeMsg{ResourceType: resourceType}
	}, nil
}

// RenderRow implements the RenderRow interface for custom resource row styling
func (r *Resources) RenderRow(rowData common.RowData, isSelected bool, colWidth int) []string {
	if rowData.Len() == 0 {
		return []string{}
	}

	// Get ordered slice from RowData
	styledRow := rowData.ToSlice()

	// Apply styling to "profiles" and "ssh_connections" entries
	for i, cell := range styledRow {
		lowerCell := strings.TrimSpace(strings.ToLower(cell))
		if customResources.Contains(lowerCell) && !isSelected {
			greenStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("2")) // Green color (same as [active])
			styledRow[i] = greenStyle.Render(cell)
		}
	}

	return styledRow
}
