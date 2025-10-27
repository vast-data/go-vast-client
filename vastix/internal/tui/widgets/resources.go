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
	"github.com/vast-data/go-vast-client/rest"
	"go.uber.org/zap"
)

var customResources = shared.NewSet[string](
	[]string{
		"profiles",
	},
)

// Global widgets map to store auto-generated widgets
var generatedWidgets map[string]common.Widget
var globalFactory *WidgetFactory

type Resources struct {
	*BaseWidget
	factory *WidgetFactory
}

// InitializeWidgets initializes specific widgets one by one and returns the factory
func InitializeWidgets(db *database.Service, restClient *rest.UntypedVMSRest) (*WidgetFactory, error) {
	if restClient == nil {
		return nil, fmt.Errorf("rest client is nil")
	}

	factory := NewWidgetFactory(db, restClient)
	generatedWidgets = make(map[string]common.Widget)

	// Create widgets one by one - add more as needed
	if restClient.Users != nil {
		usersWidget, err := factory.CreateWidget(restClient.Users, []string{"id", "name", "uid", "sid"})
		if err != nil {
			return nil, fmt.Errorf("failed to create Users widget: %w", err)
		}
		// Use the widget's resource type as the key (already cleaned by factory)
		resourceKey := usersWidget.GetResourceType()
		generatedWidgets[resourceKey] = usersWidget
		factory.addSupportedResource(resourceKey)
	}

	// TODO: Add more widgets one by one as needed:
	// if restClient.Groups != nil {
	// 	// Pass custom headers or nil for defaults
	// 	groupsWidget, err := factory.CreateWidget(restClient.Groups, nil)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to create Groups widget: %w", err)
	// 	}
	// 	resourceKey := groupsWidget.GetResourceType()
	// 	generatedWidgets[resourceKey] = groupsWidget
	// 	factory.addSupportedResource(resourceKey)
	// }

	// Store globally for Resources widget to access
	globalFactory = factory

	return factory, nil
}

// GetGeneratedWidget returns a generated widget by name
func GetGeneratedWidget(name string) (common.Widget, bool) {
	widget, ok := generatedWidgets[name]
	if ok {
		return widget, true
	}

	for key, widget := range generatedWidgets {
		if strings.EqualFold(key, name) {
			return widget, true
		}
	}

	return nil, false
}

// NewResources creates a new resources widget
func NewResources(db *database.Service) common.Widget {
	resourceType := "resources"
	listHeaders := []string{"type"}

	keyRestrictions := &common.NavigatorKeyRestrictions{
		Main: common.KeyRestrictions{
			NotAllowedListKeys: []string{"d"}, // Block extra actions in main list mode
		},
		Extra: common.NewDefaultKeyRestrictions(), // No restrictions for extra widgets
	}

	widget := &Resources{
		BaseWidget: NewBaseWidget(db, listHeaders, nil, resourceType, nil, keyRestrictions),
		factory:    globalFactory,
	}

	widget.SetParentForBaseWidget(widget, false)
	return widget
}

// GetAllWidgets returns all available widgets (custom + generated)
func (r *Resources) GetAllWidgets() map[string]common.Widget {
	allWidgets := make(map[string]common.Widget)

	// Add Profile widget (always available)
	profileWidget := NewProfile(r.db)
	allWidgets[profileWidget.GetResourceType()] = profileWidget

	// Add Resources widget itself
	allWidgets["resources"] = r

	// Add all generated widgets from factory (already using clean keys)
	for key, widget := range generatedWidgets {
		allWidgets[key] = widget
	}

	return allWidgets
}

func (r *Resources) SetListData() tea.Msg {
	var supportedResources []string

	// Add custom resources first (always available)
	supportedResources = append(supportedResources, customResources.ToOrderedSlice()...)

	// Add resources from factory if available
	// Check if globalFactory has been initialized since widget creation
	if globalFactory != nil {
		r.factory = globalFactory // Update reference to current factory
		supportedResources = append(supportedResources, r.factory.GetSupportedResources()...)
	}

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
