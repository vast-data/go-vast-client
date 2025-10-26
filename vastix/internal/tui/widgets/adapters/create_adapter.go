package adapters

import (
	"fmt"
	"regexp"
	"strings"
	"vastix/internal/database"
	log "vastix/internal/logging"
	"vastix/internal/msg_types"
	"vastix/internal/tui/widgets/common"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.uber.org/zap"
)

// Color definitions matching main TUI package
var (
	InactivePreviewBorder = lipgloss.AdaptiveColor{
		Dark:  "244",
		Light: "250",
	}
)

// CreateAdapter handles dynamic input forms similar to ListAdapter
type CreateAdapter struct {
	resourceType    string // Type of resource this adapter represents, e.g., "views", "quotas", "users" etc.
	predefinedTitle string // Optional predefined title to override default
	// Input management
	inputs            []common.InputWrapper
	flatInputs        []FlatInput // Flattened list for navigation
	focused           int         // Index into flatInputs, not inputs
	visibleStartInput int
	maxVisibleInputs  int

	// Zone-based navigation
	leftZone  *LeftZone
	rightZone *RightZone

	// Current zone dimensions (updated in RenderCreate)
	currentLeftZoneWidth   int
	currentLeftZoneHeight  int
	currentRightZoneWidth  int
	currentRightZoneHeight int

	// Collapsible object state
	expandedObjects map[int]bool // Track which top-level objects are expanded

	isJSONMode    bool           // Toggle between form mode (false) and JSON mode (true)
	isEditingJSON bool           // Whether actively editing JSON in textarea
	jsonContent   string         // Cached JSON content for JSON mode
	jsonTextarea  textarea.Model // Textarea for editing JSON

	// Database connection
	db *database.Service

	// Form state
	err error
}

// FlatInput represents a flattened input for navigation purposes
type FlatInput struct {
	wrapper    *common.InputWrapper // Reference to the actual input wrapper
	parentPath []int                // Path to parent (for nested inputs)
	depth      int                  // Nesting depth for rendering
}

// LeftZone manages the left side navigation and content
type LeftZone struct {
	content       string // Frozen content when navigating objects
	focused       int    // Current focused input in left zone
	isFrozen      bool   // Whether the zone is frozen (showing objects on right)
	frozenFocused int    // The focused input when zone was frozen
	width         int    // Zone width for frozen content
	height        int    // Zone height for frozen content
}

// RightZone manages the right side object navigation and content
type RightZone struct {
	content       string               // Current object content
	objectWrapper *common.InputWrapper // The object being shown
	focused       int                  // Current focused field within object
	flatFields    []FlatInput          // Flattened fields of the current object
	isActive      bool                 // Whether right zone is active
}

// NewCreateAdapter creates a new create form adapter
func NewCreateAdapter(db *database.Service, resourceType string) *CreateAdapter {
	adapter := &CreateAdapter{
		resourceType:      resourceType,
		inputs:            make([]common.InputWrapper, 0),
		focused:           0,
		visibleStartInput: 0,
		maxVisibleInputs:  0,
		expandedObjects:   make(map[int]bool),

		// Initialize zones
		leftZone: &LeftZone{
			content:       "",
			focused:       0,
			isFrozen:      false,
			frozenFocused: 0,
			width:         0,
			height:        0,
		},
		rightZone: &RightZone{
			content:       "",
			objectWrapper: nil,
			focused:       0,
			flatFields:    make([]FlatInput, 0),
			isActive:      false,
		},

		db: db,
	}
	return adapter
}

// NewCreateAdapterWithPredefinedTitle creates a new create adapter with a predefined title
func NewCreateAdapterWithPredefinedTitle(db *database.Service, resourceType, title string) *CreateAdapter {
	adapter := NewCreateAdapter(db, resourceType)
	adapter.predefinedTitle = title
	return adapter
}

// SetPredefinedTitle allows setting the predefined title dynamically
func (ca *CreateAdapter) SetPredefinedTitle(title string) {
	ca.predefinedTitle = title
}

// GetPredefinedTitle returns the predefined title
func (ca *CreateAdapter) GetPredefinedTitle() string {
	return ca.predefinedTitle
}

func (ca *CreateAdapter) CreateFromInputsDo(w common.CreateWidget) tea.Cmd {
	inputs := ca.GetInputs()
	return msg_types.ProcessWithSpinnerMust(w.CreateFromInputs(inputs))
}

// GetInputs returns the current inputs with user data
func (ca *CreateAdapter) GetInputs() common.Inputs {
	return ca.inputs
}

// SetInputs sets the inputs for the form from a widget's GetInputs method
func (ca *CreateAdapter) SetInputs(inputWrappers []common.InputWrapper) {
	log.Debug("Setting inputs for CreateAdapter",
		zap.Int("input_count", len(inputWrappers)))

	// Flatten nested inputs for navigation while preserving structure for rendering
	ca.inputs = make([]common.InputWrapper, len(inputWrappers))
	copy(ca.inputs, inputWrappers)

	// Copy input wrappers and clear prompts recursively
	for i, wrapper := range ca.inputs {
		ca.clearPromptsRecursively(&wrapper)
		ca.inputs[i] = wrapper
	}

	// Build flattened navigation structure
	ca.buildFlatInputs()

	// Initialize zone-based navigation and focus first input
	if len(ca.inputs) > 0 {
		ca.focused = 0
		ca.visibleStartInput = 0 // Reset scroll position

		// Initialize left zone
		ca.leftZone.focused = 0
		ca.leftZone.isFrozen = false

		// Deactivate right zone
		ca.rightZone.isActive = false
		ca.rightZone.focused = 0

		// Blur all inputs first (including nested ones)
		ca.blurAllInputs()

		// Focus the first top-level input
		ca.focusInput(&ca.inputs[0])

		log.Debug("Zone-based navigation initialized",
			zap.String("first_input_label", ca.inputs[0].GetLabel()),
			zap.Int("total_inputs", len(ca.inputs)))
	}
}

// clearPromptsRecursively clears prompts from an input and all its nested children
func (ca *CreateAdapter) clearPromptsRecursively(wrapper *common.InputWrapper) {
	// Skip prompt clearing for now - this was a UI detail that's not critical
	// The interface doesn't expose direct field access to textInput.Prompt
	// This functionality can be implemented in the concrete types if needed
	_ = wrapper
}

// blurAllInputs recursively blurs all inputs including nested ones
func (ca *CreateAdapter) blurAllInputs() {
	for i := range ca.inputs {
		ca.blurInput(&ca.inputs[i])
	}
}

// blurInput recursively blurs an input and all its nested children
func (ca *CreateAdapter) blurInput(wrapper *common.InputWrapper) {
	if wrapper.IsTextInput() {
		wrapper.TextInput.Blur()
	} else if wrapper.IsBoolInput() {
		wrapper.BoolInput.Blur()
	} else if wrapper.IsInt64Input() {
		wrapper.Int64Input.Blur()
	} else if wrapper.IsFloat64Input() {
		wrapper.Float64Input.Blur()
	} else if wrapper.IsComplexArrayInput() {
		wrapper.ComplexArrayInput.Blur()
	} else if wrapper.IsPrimitivesArrayInput() {
		wrapper.PrimitivesArrayInput.Blur()
	} else if wrapper.IsNestedInput() {
		// Recursively blur all child inputs
		childInputs := wrapper.NestedInput.GetInputs()
		for i := range childInputs {
			ca.blurInput(&childInputs[i]) // Take address of the value
		}
	}
}

// focusInput recursively focuses an input, handling nested structures
func (ca *CreateAdapter) focusInput(wrapper *common.InputWrapper) {
	if wrapper.IsTextInput() {
		wrapper.TextInput.Focus()
	} else if wrapper.IsBoolInput() {
		wrapper.BoolInput.Focus()
	} else if wrapper.IsInt64Input() {
		wrapper.Int64Input.Focus()
	} else if wrapper.IsFloat64Input() {
		wrapper.Float64Input.Focus()
	} else if wrapper.IsComplexArrayInput() {
		wrapper.ComplexArrayInput.Focus()
	} else if wrapper.IsPrimitivesArrayInput() {
		wrapper.PrimitivesArrayInput.Focus()
	} else if wrapper.IsNestedInput() {
		wrapper.NestedInput.Focus()
	}
}

// styleInputWithBottomBorder applies bottom border styling to input
func (ca *CreateAdapter) styleInputWithBottomBorder(input textinput.Model) textinput.Model {
	// Remove any existing prompt styling and set to empty
	input.Prompt = ""

	// Set consistent styling for all inputs
	return input
}

// renderFormJSONToggle renders a toggle button with labels inside
// Form mode: [form|json] where "form" is highlighted
// JSON mode: [form|json] where "json" is highlighted
func (ca *CreateAdapter) renderFormJSONToggle() string {
	if ca.isJSONMode {
		// JSON mode: form is inactive (gray), json is active (green)
		inactiveStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("250")). // Light gray background
			Foreground(lipgloss.Color("240"))  // Dim gray text

		activeStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("42")). // Bright green background
			Foreground(lipgloss.Color("0"))   // Black text

		formPart := inactiveStyle.Render(" form ")
		jsonPart := activeStyle.Render(" json ")

		return formPart + jsonPart
	} else {
		// Form mode: form is active (green), json is inactive (gray)
		activeStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("42")). // Bright green background
			Foreground(lipgloss.Color("0"))   // Black text

		inactiveStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("250")). // Light gray background
			Foreground(lipgloss.Color("240"))  // Dim gray text

		formPart := activeStyle.Render(" form ")
		jsonPart := inactiveStyle.Render(" json ")

		return formPart + jsonPart
	}
}

// ToggleFormJSONMode toggles between form and JSON mode
func (ca *CreateAdapter) ToggleFormJSONMode() {
	ca.isJSONMode = !ca.isJSONMode

	if ca.isJSONMode {
		// Immediately start editing when entering JSON mode
		ca.StartJSONEditing()
	} else {
		// Exit editing mode when returning to form
		if ca.isEditingJSON {
			ca.CancelJSONEditing()
		}
	}

	log.Debug("Toggled form/JSON mode", zap.Bool("isJSONMode", ca.isJSONMode), zap.Bool("isEditingJSON", ca.isEditingJSON))
}

// IsJSONMode returns whether the adapter is in JSON mode
func (ca *CreateAdapter) IsJSONMode() bool {
	return ca.isJSONMode
}

// SetJSONMode sets the JSON mode state
func (ca *CreateAdapter) SetJSONMode(enabled bool) {
	ca.isJSONMode = enabled
	log.Debug("Set form/JSON mode", zap.Bool("isJSONMode", ca.isJSONMode))
}

// StartJSONEditing initializes the textarea for JSON editing
func (ca *CreateAdapter) StartJSONEditing() {
	// Convert current inputs to JSON
	inputs := common.Inputs(ca.inputs)
	jsonStr, err := inputs.ToJSONIndented()
	if err != nil {
		log.Error("Failed to convert to JSON", zap.Error(err))
		jsonStr = "{}"
	}

	// Initialize textarea if not already done
	if ca.jsonTextarea.Value() == "" {
		ca.jsonTextarea = textarea.New()
		ca.jsonTextarea.Placeholder = "Enter JSON here..."
		ca.jsonTextarea.ShowLineNumbers = true
		ca.jsonTextarea.CharLimit = 0 // No limit

		// Style line numbers with gray color (same as hint values)
		lineNumberStyle := lipgloss.NewStyle().Foreground(LightGrey)
		ca.jsonTextarea.FocusedStyle.LineNumber = lineNumberStyle
		ca.jsonTextarea.BlurredStyle.LineNumber = lineNumberStyle
	}

	// Set the JSON content in the textarea
	ca.jsonTextarea.SetValue(jsonStr)
	ca.jsonTextarea.Focus()
	ca.jsonTextarea.CursorStart() // Move cursor to the beginning

	// Enter editing mode
	ca.isEditingJSON = true

	log.Info("Started JSON editing mode")
}

// SaveJSONEdits saves the edited JSON back to form inputs
func (ca *CreateAdapter) SaveJSONEdits() error {
	// Get the edited JSON from textarea
	jsonStr := ca.jsonTextarea.Value()

	// Apply to form inputs
	if err := ca.ApplyJSONEdits(jsonStr); err != nil {
		return err
	}

	return nil
}

// CancelJSONEditing exits JSON editing mode without saving
func (ca *CreateAdapter) CancelJSONEditing() {
	ca.isEditingJSON = false
	ca.isJSONMode = false
	ca.jsonTextarea.Blur()
	log.Info("Cancelled JSON editing, returned to form mode")
}

// UpdateJSONTextarea updates the textarea with a message
func (ca *CreateAdapter) UpdateJSONTextarea(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	ca.jsonTextarea, cmd = ca.jsonTextarea.Update(msg)
	return cmd
}

// IsEditingJSON returns whether currently editing JSON
func (ca *CreateAdapter) IsEditingJSON() bool {
	return ca.isEditingJSON
}

// ApplyJSONEdits applies the edited JSON to the form inputs
func (ca *CreateAdapter) ApplyJSONEdits(jsonStr string) error {
	inputs := common.Inputs(ca.inputs)
	if err := inputs.FromJSON(jsonStr); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Update cached JSON content
	ca.jsonContent = jsonStr

	log.Info("Successfully applied JSON edits to form inputs")
	return nil
}

// renderJSONView renders the JSON representation of the form inputs
func (ca *CreateAdapter) renderJSONView(width, height int) string {
	// Calculate inner dimensions
	innerWidth := width - 2 // Account for left and right borders
	if innerWidth < 0 {
		innerWidth = width
	}

	innerHeight := height - 2 // Account for top and bottom borders
	if innerHeight < 0 {
		innerHeight = height
	}

	// Always show textarea editor in JSON mode
	ca.jsonTextarea.SetWidth(innerWidth)
	ca.jsonTextarea.SetHeight(innerHeight)
	jsonContent := ca.jsonTextarea.View()

	// Create resource type label
	resourceNameStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("214")). // Orange background
		Foreground(lipgloss.Color("0"))    // Black text

	// Use predefined title if available, otherwise use default format
	var titleText string
	if ca.predefinedTitle != "" {
		titleText = ca.predefinedTitle
	} else {
		titleText = "create: " + ca.resourceType
	}
	resourceTypeLabel := resourceNameStyle.Render(" " + titleText + " ")

	// Render the form/JSON toggle for top-left border
	toggleLabel := ca.renderFormJSONToggle()

	// Calculate line count for display
	lines := strings.Split(ca.jsonTextarea.Value(), "\n")
	lineCountText := fmt.Sprintf("%d lines", len(lines))

	// Use the common borderize function to add borders (no hint text)
	embeddedText := map[common.BorderPosition]string{
		common.TopLeftBorder:     toggleLabel,
		common.TopMiddleBorder:   resourceTypeLabel,
		common.BottomRightBorder: lineCountText,
	}

	return common.BorderizeWithSpinnerCheck(jsonContent, true, embeddedText)
}

// RenderCreate renders the form with zone-based split layout or JSON view
func (ca *CreateAdapter) RenderCreate(width, height int) string {
	if len(ca.inputs) == 0 {
		panic(
			"No inputs configured. It usually means it is bug in implementation. " +
				"CreateMode should not appear for resources without inputs",
		)
	}

	// Check if we're in JSON mode
	if ca.isJSONMode {
		return ca.renderJSONView(width, height)
	}

	log.Debug("Zone-based RenderCreate",
		zap.Bool("left_frozen", ca.leftZone.isFrozen),
		zap.Bool("right_active", ca.rightZone.isActive),
		zap.Int("left_focused", ca.leftZone.focused),
		zap.Int("right_focused", ca.rightZone.focused))

	// Calculate inner dimensions
	innerWidth := width - 2 // Account for left and right borders
	if innerWidth < 0 {
		innerWidth = width
	}

	innerHeight := height - 2 // Account for top and bottom borders
	if innerHeight < 0 {
		innerHeight = height
	}

	// Create two equal zones
	leftZoneWidth := innerWidth / 2
	rightZoneWidth := innerWidth - leftZoneWidth

	// Store current zone dimensions for use in navigation methods
	ca.currentLeftZoneWidth = leftZoneWidth
	ca.currentLeftZoneHeight = innerHeight
	ca.currentRightZoneWidth = rightZoneWidth
	ca.currentRightZoneHeight = innerHeight

	// Get zone content
	var leftZoneContent string
	if ca.leftZone.isFrozen && ca.leftZone.content != "" &&
		ca.leftZone.width == leftZoneWidth && ca.leftZone.height == innerHeight {
		// Use frozen content (dimensions match)
		log.Debug("Using frozen left zone content")
		leftZoneContent = ca.leftZone.content
	} else {
		// Generate fresh content (not frozen or dimensions changed)
		log.Debug("Generating fresh left zone content")
		leftZoneContent = ca.generateLeftZoneContent(leftZoneWidth, innerHeight)

		// If zone is frozen but dimensions changed, update the stored content
		if ca.leftZone.isFrozen {
			log.Debug("Updating frozen content due to dimension change")
			ca.leftZone.content = leftZoneContent
			ca.leftZone.width = leftZoneWidth
			ca.leftZone.height = innerHeight
		}
	}

	// Always generate fresh right zone content
	rightZoneContent := ca.generateRightZoneContent(rightZoneWidth, innerHeight)

	// Join horizontally
	formContent := lipgloss.JoinHorizontal(lipgloss.Top, leftZoneContent, rightZoneContent)

	// Calculate position text based on current zone
	var positionText string
	if ca.rightZone.isActive {
		positionText = fmt.Sprintf("R %d/%d", ca.rightZone.focused+1, len(ca.rightZone.flatFields))
	} else {
		positionText = fmt.Sprintf("L %d/%d", ca.leftZone.focused+1, len(ca.inputs))
	}

	// Create resource type label
	resourceNameStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("214")). // Orange background
		Foreground(lipgloss.Color("0"))    // Black text

	// Use predefined title if available, otherwise use default format
	var titleText string
	if ca.predefinedTitle != "" {
		titleText = ca.predefinedTitle
	} else {
		titleText = "create: " + ca.resourceType
	}
	resourceTypeLabel := resourceNameStyle.Render(" " + titleText + " ")

	// Render the form/JSON toggle for top-left border
	toggleLabel := ca.renderFormJSONToggle()

	// Use the common borderize function to add borders
	embeddedText := map[common.BorderPosition]string{
		common.TopLeftBorder:     toggleLabel,
		common.TopMiddleBorder:   resourceTypeLabel,
		common.BottomRightBorder: positionText,
	}

	return common.BorderizeWithSpinnerCheck(formContent, true, embeddedText)
}

// renderLeftZoneContent renders the left zone as a complete independent content block
func (ca *CreateAdapter) renderLeftZoneContent(zoneWidth, zoneHeight int, focusedWrapper *common.InputWrapper) string {
	var rows []string

	if focusedWrapper != nil {
		log.Debug("Rendering left zone with focused wrapper",
			zap.String("focused_wrapper_label", focusedWrapper.GetLabel()))
	} else {
		log.Debug("Rendering left zone with no focused wrapper")
	}

	// Calculate maximum visible inputs based on available height
	linesPerInput := 4
	reservedLines := 3
	maxVisibleInputs := (zoneHeight - reservedLines) / linesPerInput
	if maxVisibleInputs < 1 {
		maxVisibleInputs = 1
	}

	// Ensure visibleStartInput is within bounds
	if ca.visibleStartInput > len(ca.inputs)-1 {
		ca.visibleStartInput = max(0, len(ca.inputs)-maxVisibleInputs)
	}

	// Calculate visible input range
	visibleEndInput := min(ca.visibleStartInput+maxVisibleInputs, len(ca.inputs))

	// Render only visible inputs within the zone width constraints
	for i := ca.visibleStartInput; i < visibleEndInput; i++ {
		wrapper := &ca.inputs[i] // Get pointer to the actual wrapper in ca.inputs

		// Determine border color based on focus (check if this wrapper or any child is focused)
		var borderColor lipgloss.TerminalColor = InactivePreviewBorder
		if wrapper == focusedWrapper || ca.isWrapperOrChildFocused(wrapper, focusedWrapper) {
			borderColor = Blue
		}

		log.Debug("Rendering left zone input",
			zap.Int("input_index", i),
			zap.String("input_label", wrapper.GetLabel()),
			zap.Bool("is_focused", wrapper == focusedWrapper),
			zap.Bool("is_child_focused", ca.isWrapperOrChildFocused(wrapper, focusedWrapper)))

		// Always show objects as collapsed in the left zone
		var inputRows []string
		if wrapper.IsNestedInput() || wrapper.IsComplexArrayInput() {
			// Render collapsed object or complex array within zone width
			inputRows = ca.renderCollapsedObjectInZone(*wrapper, borderColor, zoneWidth)
		} else {
			// Render regular input within zone width
			inputRows = ca.renderInputInZone(*wrapper, 0, borderColor, false, zoneWidth)
		}

		rows = append(rows, inputRows...)

		// Add spacing between inputs
		if i < visibleEndInput-1 {
			rows = append(rows, "")
		}
	}

	// Create a fixed-size zone content
	zoneStyle := lipgloss.NewStyle().
		Width(zoneWidth).
		Height(zoneHeight).
		Align(lipgloss.Left, lipgloss.Top)

	content := strings.Join(rows, "\n")
	return zoneStyle.Render(content)
}

// renderRightZoneContent renders the right zone as a complete independent content block
func (ca *CreateAdapter) renderRightZoneContent(zoneWidth, zoneHeight int, objectIndex int, objectWrapper *common.InputWrapper, actualFocusedWrapper *common.InputWrapper) string {
	var rows []string

	if objectIndex >= 0 && objectWrapper != nil {
		// Render the expanded object within the zone width
		// Use the actual focused wrapper to highlight the correct field within the object
		var borderColor lipgloss.TerminalColor = Blue // Always highlight the expanded object
		objectRows := ca.renderInputInZoneWithFocus(*objectWrapper, 0, borderColor, true, zoneWidth, actualFocusedWrapper)
		rows = append(rows, objectRows...)
	}
	// If no object focused, leave the right zone empty (no placeholder text)

	// Create a fixed-size zone content
	zoneStyle := lipgloss.NewStyle().
		Width(zoneWidth).
		Height(zoneHeight).
		Align(lipgloss.Left, lipgloss.Top)

	content := strings.Join(rows, "\n")
	return zoneStyle.Render(content)
}

// truncateToWidth truncates a string to fit within the specified width
func (ca *CreateAdapter) truncateToWidth(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	if lipgloss.Width(text) <= maxWidth {
		return text
	}

	// Simple truncation - could be improved to handle ANSI codes better
	runes := []rune(text)
	for i := len(runes); i > 0; i-- {
		candidate := string(runes[:i])
		if lipgloss.Width(candidate) <= maxWidth {
			return candidate
		}
	}

	return ""
}

// renderInputInZone renders an input wrapper within a specific zone width
func (ca *CreateAdapter) renderInputInZone(wrapper common.InputWrapper, depth int, borderColor lipgloss.TerminalColor, isFocused bool, zoneWidth int) []string {
	// Use the existing renderInputWrapper but constrain to zone width
	return ca.renderInputWrapper(wrapper, depth, borderColor, isFocused, zoneWidth)
}

// renderInputInZoneWithFocus renders an input wrapper within a specific zone width with specific focus highlighting
func (ca *CreateAdapter) renderInputInZoneWithFocus(wrapper common.InputWrapper, depth int, borderColor lipgloss.TerminalColor, isFocused bool, zoneWidth int, actualFocusedWrapper *common.InputWrapper) []string {
	// Use the existing renderInputWrapper but with focus-aware rendering
	return ca.renderInputWrapperWithFocus(wrapper, depth, borderColor, isFocused, zoneWidth, actualFocusedWrapper)
}

// renderCollapsedObjectInZone renders a collapsed object within a specific zone width
func (ca *CreateAdapter) renderCollapsedObjectInZone(wrapper common.InputWrapper, borderColor lipgloss.TerminalColor, zoneWidth int) []string {
	// Use the existing renderCollapsedObject but constrain to zone width
	return ca.renderCollapsedObject(wrapper, borderColor, zoneWidth)
}

// styleTypeHintsInLabel applies gray styling to type hints like [str|enum:...] within a label
func (ca *CreateAdapter) styleTypeHintsInLabel(text string, grayStyle lipgloss.Style) string {
	// Find and style type hints in brackets - handle both complete and cropped hints
	// Pattern 1: Complete type hints [content]
	re1 := regexp.MustCompile(`\[([^\]]+)\]`)
	text = re1.ReplaceAllStringFunc(text, func(match string) string {
		return grayStyle.Render(match)
	})

	// Pattern 2: Cropped type hints [content... (missing closing bracket)
	re2 := regexp.MustCompile(`\[[^\]]*\.\.\.`)
	text = re2.ReplaceAllStringFunc(text, func(match string) string {
		return grayStyle.Render(match)
	})

	return text
}

// renderInputWrapperWithFocus recursively renders an input wrapper with focus-aware highlighting for the right zone
func (ca *CreateAdapter) renderInputWrapperWithFocus(wrapper common.InputWrapper, depth int, borderColor lipgloss.TerminalColor, isFocused bool, availableWidth int, actualFocusedWrapper *common.InputWrapper) []string {
	var rows []string

	// Calculate indentation based on depth
	indentation := strings.Repeat("  ", depth) // 2 spaces per level

	// Create label with type information
	labelText := wrapper.Label

	// Add type information in brackets for non-object types
	var typeText string
	if !wrapper.IsNestedInput() { // Don't show types for objects
		switch wrapper.GetType() {
		case common.InputTypeComplexArray:
			// Get detailed array type information for complex arrays
			var itemType string = "unknown"
			if wrapper.ComplexArrayInput != nil && wrapper.ComplexArrayInput.ItemDef != nil {
				switch wrapper.ComplexArrayInput.ItemDef.Type {
				case "string":
					itemType = "str"
				case "integer":
					itemType = "int"
				case "number":
					itemType = "float"
				case "boolean":
					itemType = "bool"
				case "object":
					itemType = "object"
				case "array":
					itemType = "array"
				default:
					itemType = wrapper.ComplexArrayInput.ItemDef.Type
				}
			} else {
				// Default to str for simple arrays without item definition
				itemType = "str"
			}
			typeText = " [array[" + itemType + "]]"
		case common.InputTypeText:
			// Check if text input has enum values
			if wrapper.TextInput != nil && len(wrapper.TextInput.GetOneOf()) > 0 {
				enumValues := strings.Join(wrapper.TextInput.GetOneOf(), ",")
				typeText = " [str|enum:" + enumValues + "]"
			} else {
				typeText = " [str]"
			}
		case common.InputTypeInt64:
			typeText = " [int]"
		case common.InputTypeFloat64:
			typeText = " [float]"
		case common.InputTypeBool:
			typeText = " [bool]"
		case common.InputTypePrimitivesArray:
			// Get array type from ArrayInput
			if wrapper.PrimitivesArrayInput != nil && "array[str]" != "" {
				typeText = " [" + "array[str]" + "]"
			} else {
				typeText = " [array[str]]" // Default fallback
			}
		}
	}

	// Style the type text in gray
	typeStyle := lipgloss.NewStyle().Foreground(LightGrey)
	styledType := typeStyle.Render(typeText)

	var label string
	yellowColon := lipgloss.NewStyle().Foreground(Yellow).Render(":")
	if wrapper.IsRequired() {
		redStar := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5353")).Render("*")
		label = labelText + redStar + styledType + yellowColon
	} else {
		label = labelText + styledType + yellowColon
	}

	// Crop label if it exceeds available width (be more aggressive for zone constraints)
	maxLabelWidth := availableWidth - 10 // Leave space for input field and borders
	if maxLabelWidth < 10 {
		maxLabelWidth = availableWidth / 3 // Minimum space allocation
	}

	if lipgloss.Width(label) > maxLabelWidth && maxLabelWidth > 5 {
		// Remove styled parts to get the raw text for cropping
		rawLabel := labelText + typeText
		if wrapper.IsRequired() {
			rawLabel += "*"
		}
		rawLabel += ":"

		// Crop the raw label more aggressively for zone constraints
		if len(rawLabel) > maxLabelWidth-3 && maxLabelWidth > 3 {
			rawLabel = rawLabel[:maxLabelWidth-3] + "..."
		}

		// Re-style the cropped label
		grayStyle := lipgloss.NewStyle().Foreground(LightGrey)
		yellowColon := lipgloss.NewStyle().Foreground(Yellow).Render(":")
		if wrapper.IsRequired() {
			parts := strings.Split(rawLabel, "*")
			if len(parts) == 2 {
				redStar := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5353")).Render("*")

				// Apply gray styling to type hints in the label parts
				styledPart0 := ca.styleTypeHintsInLabel(parts[0], grayStyle)
				styledPart1 := ca.styleTypeHintsInLabel(parts[1], grayStyle)

				label = styledPart0 + redStar + styledPart1
				if !strings.HasSuffix(label, ":") {
					label = strings.TrimSuffix(label, ":") + yellowColon
				}
			} else {
				rawLabelWithoutColon := strings.TrimSuffix(rawLabel, ":")
				label = ca.styleTypeHintsInLabel(rawLabelWithoutColon, grayStyle) + yellowColon
			}
		} else {
			rawLabelWithoutColon := strings.TrimSuffix(rawLabel, ":")
			label = ca.styleTypeHintsInLabel(rawLabelWithoutColon, grayStyle) + yellowColon
		}
	}

	// Handle nested inputs differently
	if wrapper.IsNestedInput() {
		// Render nested input with border and label in top-left corner
		return ca.renderNestedInputWithFocus(wrapper, depth, borderColor, isFocused, availableWidth, indentation, label, actualFocusedWrapper)
	}

	// Check if this specific input is the actually focused one
	var actualInputFocused bool
	if actualFocusedWrapper != nil {
		// Use label-based comparison since GetID method was reverted
		actualInputFocused = (actualFocusedWrapper.GetLabel() == wrapper.GetLabel() && actualFocusedWrapper.GetType() == wrapper.GetType())
	}

	// Render regular input
	var labelPrefix string
	if actualInputFocused {
		labelPrefix = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("▶ ")
	} else {
		labelPrefix = "  "
	}
	labelRow := " " + indentation + labelPrefix + lipgloss.NewStyle().Foreground(Yellow).Render(label)

	// Input field - allocate half of the screen (full zone width)
	baseIndentWidth := len(indentation) + 1 // +1 for left margin
	remainingWidth := availableWidth - baseIndentWidth
	inputWidth := max(10, remainingWidth-2) // Use nearly full zone width for input area

	var inputView string
	if wrapper.IsTextInput() {
		if ti := wrapper.TextInput.GetTextInput(); ti != nil {
			ti.Width = max(0, inputWidth-2)
		}
		inputView = wrapper.TextInput.View()
	} else if wrapper.IsBoolInput() {
		inputView = wrapper.BoolInput.View()
	} else if wrapper.IsInt64Input() {
		if wrapper.Int64Input != nil && wrapper.Int64Input.TextInput != nil {
			wrapper.Int64Input.TextInput.Width = max(0, inputWidth-2)
		}
		inputView = wrapper.Int64Input.View()
	} else if wrapper.IsFloat64Input() {
		if wrapper.Float64Input != nil && wrapper.Float64Input.TextInput != nil {
			wrapper.Float64Input.TextInput.Width = max(0, inputWidth-2)
		}
		inputView = wrapper.Float64Input.View()
	} else if wrapper.IsComplexArrayInput() {
		// Sync array expansion state with CreateAdapter's expansion state (only complex arrays now)
		if actualFocusedWrapper == &wrapper {
			wrapper.ComplexArrayInput.IsExpanded = true
		}
		inputView = wrapper.ComplexArrayInput.View()
	} else if wrapper.IsPrimitivesArrayInput() {
		if wrapper.PrimitivesArrayInput != nil && wrapper.PrimitivesArrayInput.TextInput != nil {
			wrapper.PrimitivesArrayInput.TextInput.Width = max(0, inputWidth-2)
		}
		inputView = wrapper.PrimitivesArrayInput.View()
	}

	// Use different border color if this specific input is actually focused
	inputBorderColor := borderColor
	if actualInputFocused {
		inputBorderColor = Blue
	}

	// Create input row with left border manually - use thick border for focused fields
	leftBorderStyle := lipgloss.NewStyle().Foreground(inputBorderColor)
	var leftBorderChar, cornerChar string
	var horizontalChar string

	if actualInputFocused {
		// Focused: use thick border
		leftBorderChar = leftBorderStyle.Render("┃")
		cornerChar = leftBorderStyle.Render("┗")
		horizontalChar = "━"
	} else {
		// Not focused: use thin border
		leftBorderChar = leftBorderStyle.Render("│")
		cornerChar = leftBorderStyle.Render("└")
		horizontalChar = "─"
	}

	inputRow := " " + indentation + leftBorderChar + " " + inputView

	// Bottom border - create L-shaped corner with appropriate thickness
	bottomLine := lipgloss.NewStyle().Foreground(inputBorderColor).Width(inputWidth - 1).Render(strings.Repeat(horizontalChar, inputWidth-1))
	bottomBorder := " " + indentation + cornerChar + bottomLine

	rows = append(rows, labelRow)
	rows = append(rows, inputRow)
	rows = append(rows, bottomBorder)

	return rows
}

// renderNestedInputWithFocus renders a nested input with proper border and child indentation with focus awareness
func (ca *CreateAdapter) renderNestedInputWithFocus(wrapper common.InputWrapper, depth int, borderColor lipgloss.TerminalColor, isFocused bool, availableWidth int, indentation string, label string, actualFocusedWrapper *common.InputWrapper) []string {
	var rows []string

	nestedInput := wrapper.NestedInput
	if nestedInput == nil {
		return rows
	}

	// Calculate dimensions for the nested border - be more conservative in zones
	nestedWidth := availableWidth - len(indentation) - 2 // Account for parent indentation and border
	if nestedWidth < 8 {
		nestedWidth = max(8, availableWidth-2) // Very minimum width for zones
	}
	// Cap the nested width to prevent overflow
	if nestedWidth > availableWidth-2 {
		nestedWidth = availableWidth - 2
	}

	// Top border with label in top-left corner - use same color as labels (Yellow)
	// Use thin border for nested objects (they don't have individual focus like input fields)
	topBorderStyle := lipgloss.NewStyle().Foreground(Yellow)
	topLeftCorner := topBorderStyle.Render("┌")

	// Create label with background
	labelStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("240")).
		Foreground(Yellow).
		Padding(0, 1)
	styledLabel := labelStyle.Render(label)

	// Top border: corner + label + remaining line
	topBorder := " " + indentation + topLeftCorner + styledLabel
	rows = append(rows, topBorder)

	// Render child inputs recursively
	childInputs := nestedInput.GetInputs() // Returns []InputWrapper
	for _, childInput := range childInputs {
		childRows := ca.renderInputWrapperWithFocus(childInput, depth+1, borderColor, false, nestedWidth, actualFocusedWrapper) // Use value directly

		// Add left border to each child row - use same Yellow color (thin border style)
		for _, childRow := range childRows {
			leftBorderChar := topBorderStyle.Render("│")
			borderedRow := " " + indentation + leftBorderChar + childRow[len(indentation)+1:] // Replace indentation part
			rows = append(rows, borderedRow)
		}
	}

	// Bottom border - use same Yellow color as top border (thin border style)
	bottomLeftCorner := topBorderStyle.Render("└")
	bottomLine := strings.Repeat(" ", nestedWidth-1)
	bottomBorder := " " + indentation + bottomLeftCorner + bottomLine
	rows = append(rows, bottomBorder)

	return rows
}

// isWrapperOrChildFocused checks if the given wrapper or any of its children is the focused wrapper
func (ca *CreateAdapter) isWrapperOrChildFocused(wrapper *common.InputWrapper, focusedWrapper *common.InputWrapper) bool {
	if wrapper == focusedWrapper {
		return true
	}

	// Check nested children recursively
	if wrapper.IsNestedInput() && wrapper.NestedInput != nil {
		childInputs := wrapper.NestedInput.GetInputs()
		for i := range childInputs {
			if ca.isWrapperOrChildFocused(&childInputs[i], focusedWrapper) { // Take address of the value
				return true
			}
		}
	}

	return false
}

// nextInput moves focus to the next input with zone-based navigation
func (ca *CreateAdapter) nextInput() {
	log.Debug("Zone-based nextInput",
		zap.Bool("left_frozen", ca.leftZone.isFrozen),
		zap.Bool("right_active", ca.rightZone.isActive))

	if ca.rightZone.isActive {
		// Navigate within right zone (object fields)
		ca.navigateRightZone()
	} else {
		// Navigate within left zone (top-level inputs)
		ca.navigateLeftZone()
	}
}

// navigateLeftZone handles navigation within the left zone (top-level inputs)
func (ca *CreateAdapter) navigateLeftZone() {
	log.Debug("Navigating left zone", zap.Int("current_focused", ca.leftZone.focused))

	// Check if we have any inputs to navigate
	if len(ca.inputs) == 0 {
		log.Debug("No top-level inputs available for left zone navigation")
		return
	}

	// Blur current input
	if ca.leftZone.focused >= 0 && ca.leftZone.focused < len(ca.inputs) {
		ca.blurInput(&ca.inputs[ca.leftZone.focused])
	}

	// Move to next top-level input
	ca.leftZone.focused++
	if ca.leftZone.focused >= len(ca.inputs) {
		ca.leftZone.focused = 0 // Wrap around
	}

	// Additional safety check before accessing array
	if ca.leftZone.focused < 0 || ca.leftZone.focused >= len(ca.inputs) {
		log.Debug("Invalid left zone focus index after increment, resetting to 0")
		ca.leftZone.focused = 0
		if len(ca.inputs) == 0 {
			return
		}
	}

	currentInput := &ca.inputs[ca.leftZone.focused]
	log.Debug("Left zone moved to",
		zap.Int("new_focused", ca.leftZone.focused),
		zap.String("label", currentInput.GetLabel()))

	// Check if this is an object that should show on right
	if currentInput.IsNestedInput() || currentInput.IsComplexArrayInput() {
		log.Debug("Entered object - freezing left zone and activating right zone")
		ca.freezeLeftZone(ca.currentLeftZoneWidth, ca.currentLeftZoneHeight)
		ca.activateRightZone(currentInput)
		// Focus the first field in the right zone if it exists
		if len(ca.rightZone.flatFields) > 0 {
			ca.focusInput(ca.rightZone.flatFields[0].wrapper)
		}
	} else {
		// Regular input - ensure right zone is inactive and focus this input
		ca.deactivateRightZone()
		ca.focusInput(currentInput)
	}

	// Update main focused index for compatibility
	ca.focused = ca.leftZone.focused
}

// navigateRightZone handles navigation within the right zone (object fields)
func (ca *CreateAdapter) navigateRightZone() {
	log.Debug("Navigating right zone",
		zap.Int("current_focused", ca.rightZone.focused),
		zap.Int("total_fields", len(ca.rightZone.flatFields)))

	// Blur current field
	if ca.rightZone.focused >= 0 && ca.rightZone.focused < len(ca.rightZone.flatFields) {
		ca.blurInput(ca.rightZone.flatFields[ca.rightZone.focused].wrapper)
	}

	// Move to next field in object
	ca.rightZone.focused++
	if ca.rightZone.focused >= len(ca.rightZone.flatFields) {
		log.Debug("Reached end of object - returning to left zone")
		ca.exitRightZone()
		return
	}

	// Additional safety check (should not be needed, but defensive programming)
	if ca.rightZone.focused < 0 || ca.rightZone.focused >= len(ca.rightZone.flatFields) {
		log.Debug("Invalid right zone focus index after increment")
		ca.exitRightZone()
		return
	}

	currentField := ca.rightZone.flatFields[ca.rightZone.focused]
	log.Debug("Right zone moved to",
		zap.Int("new_focused", ca.rightZone.focused),
		zap.String("field_label", currentField.wrapper.GetLabel()))

	// Focus the new field
	ca.focusInput(currentField.wrapper)
}

// freezeLeftZone freezes the left zone content with actual dimensions
func (ca *CreateAdapter) freezeLeftZone(width, height int) {
	ca.leftZone.isFrozen = true
	ca.leftZone.frozenFocused = ca.leftZone.focused
	ca.leftZone.width = width
	ca.leftZone.height = height
	// Generate and store left zone content with actual dimensions
	ca.leftZone.content = ca.generateLeftZoneContent(width, height)
	log.Debug("Left zone frozen",
		zap.Int("frozen_focused", ca.leftZone.frozenFocused),
		zap.Int("width", width),
		zap.Int("height", height))
}

// activateRightZone activates the right zone with the given object
func (ca *CreateAdapter) activateRightZone(objectWrapper *common.InputWrapper) {
	ca.rightZone.isActive = true
	ca.rightZone.objectWrapper = objectWrapper
	ca.rightZone.focused = 0

	// Flatten the object's fields for navigation
	ca.rightZone.flatFields = ca.flattenObjectFields(objectWrapper)

	log.Debug("Right zone activated",
		zap.String("object_label", objectWrapper.GetLabel()),
		zap.Int("field_count", len(ca.rightZone.flatFields)))
}

// deactivateRightZone deactivates the right zone
func (ca *CreateAdapter) deactivateRightZone() {
	if ca.rightZone.isActive {
		log.Debug("Deactivating right zone")

		// Blur the currently focused field in right zone before deactivating
		if ca.rightZone.focused >= 0 && ca.rightZone.focused < len(ca.rightZone.flatFields) {
			ca.blurInput(ca.rightZone.flatFields[ca.rightZone.focused].wrapper)
		}
	}
	ca.rightZone.isActive = false
	ca.rightZone.content = ""
	ca.rightZone.objectWrapper = nil
	ca.rightZone.flatFields = make([]FlatInput, 0)
	ca.rightZone.focused = 0

	// Unfreeze left zone
	ca.leftZone.isFrozen = false
}

// exitRightZone exits the right zone and continues left zone navigation
func (ca *CreateAdapter) exitRightZone() {
	log.Debug("Exiting right zone")
	ca.deactivateRightZone()

	// Continue left zone navigation from where we left off
	ca.navigateLeftZone()
}

// navigateLeftZonePrev handles backward navigation within the left zone
func (ca *CreateAdapter) navigateLeftZonePrev() {
	log.Debug("Navigating left zone backward", zap.Int("current_focused", ca.leftZone.focused))

	// Check if we have any inputs to navigate
	if len(ca.inputs) == 0 {
		log.Debug("No top-level inputs available for left zone navigation")
		return
	}

	// Blur current input
	if ca.leftZone.focused >= 0 && ca.leftZone.focused < len(ca.inputs) {
		ca.blurInput(&ca.inputs[ca.leftZone.focused])
	}

	// Move to previous top-level input
	ca.leftZone.focused--
	if ca.leftZone.focused < 0 {
		ca.leftZone.focused = len(ca.inputs) - 1 // Wrap around to last
	}

	// Additional safety check before accessing array
	if ca.leftZone.focused < 0 || ca.leftZone.focused >= len(ca.inputs) {
		log.Debug("Invalid left zone focus index after decrement, resetting to 0")
		ca.leftZone.focused = 0
		if len(ca.inputs) == 0 {
			return
		}
	}

	currentInput := &ca.inputs[ca.leftZone.focused]
	log.Debug("Left zone moved backward to",
		zap.Int("new_focused", ca.leftZone.focused),
		zap.String("label", currentInput.GetLabel()))

	// Check if this is an object that should show on right
	if currentInput.IsNestedInput() || currentInput.IsComplexArrayInput() {
		log.Debug("Entered object via reverse navigation - freezing left zone and activating right zone")
		ca.freezeLeftZone(ca.currentLeftZoneWidth, ca.currentLeftZoneHeight)
		ca.activateRightZone(currentInput)
		// Focus the first field in the right zone if it exists
		if len(ca.rightZone.flatFields) > 0 {
			ca.focusInput(ca.rightZone.flatFields[0].wrapper)
		}
	} else {
		// Regular input - ensure right zone is inactive and focus this input
		ca.deactivateRightZone()
		ca.focusInput(currentInput)
	}

	// Update main focused index for compatibility
	ca.focused = ca.leftZone.focused
}

// navigateRightZonePrev handles backward navigation within the right zone
func (ca *CreateAdapter) navigateRightZonePrev() {
	log.Debug("Navigating right zone backward",
		zap.Int("current_focused", ca.rightZone.focused),
		zap.Int("total_fields", len(ca.rightZone.flatFields)))

	// Blur current field
	if ca.rightZone.focused >= 0 && ca.rightZone.focused < len(ca.rightZone.flatFields) {
		ca.blurInput(ca.rightZone.flatFields[ca.rightZone.focused].wrapper)
	}

	// Move to previous field in object
	ca.rightZone.focused--
	if ca.rightZone.focused < 0 {
		log.Debug("Reached beginning of object - returning to left zone")
		ca.exitRightZonePrev()
		return
	}

	// Additional safety check (should not be needed, but defensive programming)
	if ca.rightZone.focused >= len(ca.rightZone.flatFields) {
		log.Debug("Invalid right zone focus index after decrement")
		ca.exitRightZonePrev()
		return
	}

	currentField := ca.rightZone.flatFields[ca.rightZone.focused]
	log.Debug("Right zone moved backward to",
		zap.Int("new_focused", ca.rightZone.focused),
		zap.String("field_label", currentField.wrapper.GetLabel()))

	// Focus the new field
	ca.focusInput(currentField.wrapper)
}

// exitRightZonePrev exits the right zone and continues left zone backward navigation
func (ca *CreateAdapter) exitRightZonePrev() {
	log.Debug("Exiting right zone backward")
	ca.deactivateRightZone()

	// Continue left zone backward navigation from where we left off
	ca.navigateLeftZonePrev()
}

// generateLeftZoneContent generates content for the left zone
func (ca *CreateAdapter) generateLeftZoneContent(zoneWidth, zoneHeight int) string {
	var rows []string

	// Calculate maximum visible inputs based on available height
	linesPerInput := 4
	reservedLines := 3
	maxVisibleInputs := (zoneHeight - reservedLines) / linesPerInput
	if maxVisibleInputs < 1 {
		maxVisibleInputs = 1
	}

	// Calculate visible input range
	visibleStart := max(0, ca.leftZone.focused-maxVisibleInputs/2)
	visibleEnd := min(visibleStart+maxVisibleInputs, len(ca.inputs))

	// Render only visible inputs within the zone width constraints
	for i := visibleStart; i < visibleEnd; i++ {
		wrapper := &ca.inputs[i]

		// Determine border color based on focus
		var borderColor lipgloss.TerminalColor = InactivePreviewBorder
		if i == ca.leftZone.focused {
			borderColor = Blue
		}

		// Always show objects as collapsed in the left zone
		var inputRows []string
		isFocused := (i == ca.leftZone.focused)
		if wrapper.IsNestedInput() || wrapper.IsComplexArrayInput() {
			// Use the same borderColor that reflects focus state (Blue for focused, InactivePreviewBorder for not)
			inputRows = ca.renderCollapsedObjectInZone(*wrapper, borderColor, zoneWidth)
		} else {
			inputRows = ca.renderInputInZone(*wrapper, 0, borderColor, isFocused, zoneWidth)
		}

		rows = append(rows, inputRows...)

		// Add spacing between inputs
		if i < visibleEnd-1 {
			rows = append(rows, "")
		}
	}

	// Create a fixed-size zone content
	zoneStyle := lipgloss.NewStyle().
		Width(zoneWidth).
		Height(zoneHeight).
		Align(lipgloss.Left, lipgloss.Top)

	content := strings.Join(rows, "\n")
	return zoneStyle.Render(content)
}

// generateRightZoneContent generates content for the right zone
func (ca *CreateAdapter) generateRightZoneContent(zoneWidth, zoneHeight int) string {
	if !ca.rightZone.isActive || ca.rightZone.objectWrapper == nil {
		// Create empty zone
		zoneStyle := lipgloss.NewStyle().
			Width(zoneWidth).
			Height(zoneHeight).
			Align(lipgloss.Left, lipgloss.Top)
		return zoneStyle.Render("")
	}

	var rows []string

	// Render the object fields with focus highlighting
	for i, fieldFlat := range ca.rightZone.flatFields {
		wrapper := fieldFlat.wrapper

		// Determine border color based on focus
		var borderColor lipgloss.TerminalColor = InactivePreviewBorder
		if i == ca.rightZone.focused {
			borderColor = Blue
		}

		// Render the field
		var inputRows []string
		isFocused := (i == ca.rightZone.focused)
		if wrapper.IsNestedInput() || wrapper.IsComplexArrayInput() {
			// Use the same borderColor that reflects focus state (Blue for focused, InactivePreviewBorder for not)
			inputRows = ca.renderCollapsedObjectInZone(*wrapper, borderColor, zoneWidth)
		} else {
			inputRows = ca.renderInputInZone(*wrapper, fieldFlat.depth, borderColor, isFocused, zoneWidth)
		}

		rows = append(rows, inputRows...)

		// Add spacing between inputs
		if i < len(ca.rightZone.flatFields)-1 {
			rows = append(rows, "")
		}
	}

	// Create a fixed-size zone content
	zoneStyle := lipgloss.NewStyle().
		Width(zoneWidth).
		Height(zoneHeight).
		Align(lipgloss.Left, lipgloss.Top)

	content := strings.Join(rows, "\n")
	return zoneStyle.Render(content)
}

// flattenObjectFields flattens an object's fields for right zone navigation
func (ca *CreateAdapter) flattenObjectFields(objectWrapper *common.InputWrapper) []FlatInput {
	var flatFields []FlatInput

	if objectWrapper.IsNestedInput() {
		// Handle nested input
		nestedInput := objectWrapper.NestedInput
		if nestedInput != nil {
			for i, field := range nestedInput.GetInputs() {
				flatField := FlatInput{
					wrapper:    &field,
					parentPath: []int{0}, // Simple parent path for right zone
					depth:      1,
				}
				flatFields = append(flatFields, flatField)

				// If this field is also nested, flatten it recursively
				if field.IsNestedInput() || field.IsComplexArrayInput() {
					subFields := ca.flattenObjectFieldsRecursive(&field, []int{0, i}, 2)
					flatFields = append(flatFields, subFields...)
				}
			}
		}
	} else if objectWrapper.IsComplexArrayInput() {
		// Handle complex array input
		complexArray := objectWrapper.ComplexArrayInput
		if complexArray != nil && len(complexArray.ItemForms) > 0 && len(complexArray.ItemForms[0]) > 0 {
			for i, field := range complexArray.ItemForms[0] {
				flatField := FlatInput{
					wrapper:    &field,
					parentPath: []int{0},
					depth:      1,
				}
				flatFields = append(flatFields, flatField)

				// If this field is also nested, flatten it recursively
				if field.IsNestedInput() || field.IsComplexArrayInput() {
					subFields := ca.flattenObjectFieldsRecursive(&field, []int{0, i}, 2)
					flatFields = append(flatFields, subFields...)
				}
			}
		}
	}

	return flatFields
}

// flattenObjectFieldsRecursive recursively flattens nested object fields
func (ca *CreateAdapter) flattenObjectFieldsRecursive(wrapper *common.InputWrapper, parentPath []int, depth int) []FlatInput {
	var flatFields []FlatInput

	if wrapper.IsNestedInput() {
		nestedInput := wrapper.NestedInput
		if nestedInput != nil {
			for i, field := range nestedInput.GetInputs() {
				newPath := append([]int{}, parentPath...)
				newPath = append(newPath, i)

				flatField := FlatInput{
					wrapper:    &field,
					parentPath: newPath,
					depth:      depth,
				}
				flatFields = append(flatFields, flatField)

				// Continue recursion if needed
				if field.IsNestedInput() || field.IsComplexArrayInput() {
					subFields := ca.flattenObjectFieldsRecursive(&field, newPath, depth+1)
					flatFields = append(flatFields, subFields...)
				}
			}
		}
	} else if wrapper.IsComplexArrayInput() {
		complexArray := wrapper.ComplexArrayInput
		if complexArray != nil && len(complexArray.ItemForms) > 0 && len(complexArray.ItemForms[0]) > 0 {
			for i, field := range complexArray.ItemForms[0] {
				newPath := append([]int{}, parentPath...)
				newPath = append(newPath, i)

				flatField := FlatInput{
					wrapper:    &field,
					parentPath: newPath,
					depth:      depth,
				}
				flatFields = append(flatFields, flatField)

				// Continue recursion if needed
				if field.IsNestedInput() || field.IsComplexArrayInput() {
					subFields := ca.flattenObjectFieldsRecursive(&field, newPath, depth+1)
					flatFields = append(flatFields, subFields...)
				}
			}
		}
	}

	return flatFields
}

// prevInput moves focus to the previous input with zone-based navigation
func (ca *CreateAdapter) prevInput() {
	log.Debug("Zone-based prevInput",
		zap.Bool("left_frozen", ca.leftZone.isFrozen),
		zap.Bool("right_active", ca.rightZone.isActive))

	if ca.rightZone.isActive {
		// Navigate backward within right zone (object fields)
		ca.navigateRightZonePrev()
	} else {
		// Navigate backward within left zone (top-level inputs)
		ca.navigateLeftZonePrev()
	}
}

// Helper functions for smart navigation

// findNextFieldInSameObject finds the next field within the same object
func (ca *CreateAdapter) findNextFieldInSameObject(currentIndex int) int {
	if currentIndex < 0 || currentIndex >= len(ca.flatInputs) {
		return -1
	}

	currentInput := ca.flatInputs[currentIndex]
	if currentInput.depth == 0 {
		return -1 // Not inside an object
	}

	// Look for next field with same parent path and depth
	for i := currentIndex + 1; i < len(ca.flatInputs); i++ {
		flatInput := ca.flatInputs[i]
		if flatInput.depth == currentInput.depth && len(flatInput.parentPath) == len(currentInput.parentPath) {
			// Check if it has the same parent
			if len(flatInput.parentPath) > 0 && len(currentInput.parentPath) > 0 {
				if flatInput.parentPath[0] == currentInput.parentPath[0] {
					return i
				}
			}
		}
		// If we hit a different object or went back to top level, stop
		if flatInput.depth <= currentInput.depth {
			if len(flatInput.parentPath) == 0 || len(currentInput.parentPath) == 0 {
				break
			}
			if flatInput.parentPath[0] != currentInput.parentPath[0] {
				break
			}
		}
	}

	return -1
}

// findPrevFieldInSameObject finds the previous field within the same object
func (ca *CreateAdapter) findPrevFieldInSameObject(currentIndex int) int {
	if currentIndex <= 0 || currentIndex >= len(ca.flatInputs) {
		return -1
	}

	currentInput := ca.flatInputs[currentIndex]
	if currentInput.depth == 0 {
		return -1 // Not inside an object
	}

	// Look for previous field with same parent path and depth
	for i := currentIndex - 1; i >= 0; i-- {
		flatInput := ca.flatInputs[i]
		if flatInput.depth == currentInput.depth && len(flatInput.parentPath) == len(currentInput.parentPath) {
			// Check if it has the same parent
			if len(flatInput.parentPath) > 0 && len(currentInput.parentPath) > 0 {
				if flatInput.parentPath[0] == currentInput.parentPath[0] {
					return i
				}
			}
		}
		// If we hit the parent object or a different object, stop
		if flatInput.depth < currentInput.depth {
			break
		}
	}

	return -1
}

// findTopLevelFlatIndex finds the flat index for a top-level input
func (ca *CreateAdapter) findTopLevelFlatIndex(topLevelIndex int) int {
	for i, flatInput := range ca.flatInputs {
		if flatInput.depth == 0 && len(flatInput.parentPath) > 0 && flatInput.parentPath[0] == topLevelIndex {
			return i
		}
	}
	// If not found, find any top-level input that corresponds to the input at topLevelIndex
	if topLevelIndex >= 0 && topLevelIndex < len(ca.inputs) {
		targetWrapper := &ca.inputs[topLevelIndex]
		for i, flatInput := range ca.flatInputs {
			if flatInput.depth == 0 && flatInput.wrapper == targetWrapper {
				return i
			}
		}
	}
	return -1
}

// findNextTopLevelInput finds the next top-level input
func (ca *CreateAdapter) findNextTopLevelInput(currentIndex int) int {
	if currentIndex < 0 || currentIndex >= len(ca.flatInputs) {
		return 0
	}

	// Find the next top-level input (depth == 0)
	for i := currentIndex + 1; i < len(ca.flatInputs); i++ {
		if ca.flatInputs[i].depth == 0 {
			return i
		}
	}

	// Wraparound to the first top-level input
	for i := 0; i <= currentIndex; i++ {
		if ca.flatInputs[i].depth == 0 {
			return i
		}
	}

	return currentIndex // Fallback
}

// findPrevTopLevelInput finds the previous top-level input
func (ca *CreateAdapter) findPrevTopLevelInput(currentIndex int) int {
	if currentIndex <= 0 || currentIndex >= len(ca.flatInputs) {
		// Wraparound to the last top-level input
		for i := len(ca.flatInputs) - 1; i >= 0; i-- {
			if ca.flatInputs[i].depth == 0 {
				return i
			}
		}
		return 0
	}

	// Find the previous top-level input (depth == 0)
	for i := currentIndex - 1; i >= 0; i-- {
		if ca.flatInputs[i].depth == 0 {
			return i
		}
	}

	// Wraparound to the last top-level input
	for i := len(ca.flatInputs) - 1; i > currentIndex; i-- {
		if ca.flatInputs[i].depth == 0 {
			return i
		}
	}

	return currentIndex // Fallback
}

// findFirstFieldInObject finds the first field within an object
func (ca *CreateAdapter) findFirstFieldInObject(objectIndex int) int {
	if objectIndex < 0 || objectIndex >= len(ca.flatInputs) {
		return -1
	}

	objectFlatInput := ca.flatInputs[objectIndex]
	if objectFlatInput.depth != 0 {
		return -1 // Not a top-level object
	}

	// Look for the first field after this object
	for i := objectIndex + 1; i < len(ca.flatInputs); i++ {
		flatInput := ca.flatInputs[i]
		if flatInput.depth > 0 && len(flatInput.parentPath) > 0 {
			if flatInput.parentPath[0] == objectFlatInput.parentPath[0] {
				return i
			}
		}
		// If we hit another top-level input, stop
		if flatInput.depth == 0 {
			break
		}
	}

	return -1
}

// focusCurrentFlatInput focuses the current input in the flat structure
func (ca *CreateAdapter) focusCurrentFlatInput() {
	if len(ca.flatInputs) > 0 && ca.focused >= 0 && ca.focused < len(ca.flatInputs) {
		ca.focusInput(ca.flatInputs[ca.focused].wrapper)
	}
}

// ResetCreateForm clears all inputs and resets form state
func (ca *CreateAdapter) ResetCreateForm() {
	for i := range ca.inputs {
		ca.resetInput(&ca.inputs[i])
	}
	ca.focused = 0
	ca.visibleStartInput = 0 // Reset scroll position
	ca.err = nil

	// Reset JSON mode state
	ca.isJSONMode = false
	ca.isEditingJSON = false
	if ca.jsonTextarea.Focused() {
		ca.jsonTextarea.Blur()
	}

	// Rebuild flat inputs and focus first one
	ca.buildFlatInputs()
	if len(ca.flatInputs) > 0 {
		ca.focusCurrentFlatInput()
	}
}

// resetInput recursively resets an input and all its nested children
func (ca *CreateAdapter) resetInput(wrapper *common.InputWrapper) {
	if wrapper.IsTextInput() {
		wrapper.TextInput.SetValue("")
		wrapper.TextInput.Blur()
	} else if wrapper.IsBoolInput() {
		wrapper.BoolInput.SetValue("false")
		wrapper.BoolInput.Blur()
	} else if wrapper.IsInt64Input() {
		wrapper.Int64Input.SetValue("")
		wrapper.Int64Input.Blur()
	} else if wrapper.IsFloat64Input() {
		wrapper.Float64Input.SetValue("")
		wrapper.Float64Input.Blur()
	} else if wrapper.IsComplexArrayInput() {
		wrapper.ComplexArrayInput.SetValue("")
		wrapper.ComplexArrayInput.Blur()
	} else if wrapper.IsPrimitivesArrayInput() {
		wrapper.PrimitivesArrayInput.SetValue("[]")
		wrapper.PrimitivesArrayInput.Blur()
	} else if wrapper.IsNestedInput() {
		// Recursively reset all child inputs
		childInputs := wrapper.NestedInput.GetInputs()
		for i := range childInputs {
			ca.resetInput(&childInputs[i]) // Take address of the value
		}
	}
}

// Init returns initial command for text input blinking
func (ca *CreateAdapter) Init() tea.Cmd {
	return textinput.Blink
}

// Public methods called by WidgetNavigator

// NextInput moves focus to the next input (public method)
func (ca *CreateAdapter) NextInput() {
	log.Debug("PUBLIC NextInput called - starting navigation")
	ca.nextInput()
}

// PrevInput moves focus to the previous input (public method)
func (ca *CreateAdapter) PrevInput() {
	log.Debug("PUBLIC PrevInput called - starting navigation")
	ca.prevInput()
}

// HasInputs returns true if there are inputs available for navigation
func (ca *CreateAdapter) HasInputs() bool {
	return len(ca.flatInputs) > 0
}

// GetFocusedIndex returns the currently focused input index
func (ca *CreateAdapter) GetFocusedIndex() int {
	return ca.focused
}

// UpdateCurrentInput updates the currently focused input with the given message
func (ca *CreateAdapter) UpdateCurrentInput(msg tea.Msg) {
	if ca.rightZone.isActive {
		// Right zone is active - update focused field in object
		if ca.rightZone.focused >= 0 && ca.rightZone.focused < len(ca.rightZone.flatFields) {
			focusedField := ca.rightZone.flatFields[ca.rightZone.focused].wrapper
			log.Debug("Updating right zone input",
				zap.String("label", focusedField.GetLabel()),
				zap.Int("focused_index", ca.rightZone.focused))
			ca.updateInput(focusedField, msg)
		}
	} else {
		// Left zone is active - update focused top-level input
		if ca.leftZone.focused >= 0 && ca.leftZone.focused < len(ca.inputs) {
			focusedInput := &ca.inputs[ca.leftZone.focused]
			log.Debug("Updating left zone input",
				zap.String("label", focusedInput.GetLabel()),
				zap.Int("focused_index", ca.leftZone.focused))
			ca.updateInput(focusedInput, msg)
		}
	}
}

// updateInput recursively updates an input, handling nested structures
func (ca *CreateAdapter) updateInput(wrapper *common.InputWrapper, msg tea.Msg) {

	if wrapper.IsTextInput() {
		// Update text input
		wrapper.TextInput.Update(msg)
		// Debug: log the current value after update
		log.Debug("Text input updated",
			zap.String("label", wrapper.GetLabel()),
			zap.String("value", wrapper.Value()))
	} else if wrapper.IsBoolInput() {
		// Update boolean input
		wrapper.BoolInput.Update(msg)
		// Debug: log the current value after update
		log.Debug("Bool input updated",
			zap.String("label", wrapper.GetLabel()),
			zap.String("value", wrapper.Value()))
	} else if wrapper.IsInt64Input() {
		// Update int64 input
		wrapper.Int64Input.Update(msg)
		log.Debug("Int64 input updated",
			zap.String("label", wrapper.GetLabel()),
			zap.String("value", wrapper.Value()))
	} else if wrapper.IsFloat64Input() {
		// Update float64 input
		wrapper.Float64Input.Update(msg)
		log.Debug("Float64 input updated",
			zap.String("label", wrapper.GetLabel()),
			zap.String("value", wrapper.Value()))
	} else if wrapper.IsComplexArrayInput() {
		// Update list input
		wrapper.ComplexArrayInput.Update(msg)
		log.Debug("List input updated",
			zap.String("label", wrapper.GetLabel()),
			zap.String("value", wrapper.Value()))
	} else if wrapper.IsPrimitivesArrayInput() {
		// Update array input
		wrapper.PrimitivesArrayInput.Update(msg)
		log.Debug("Array input updated",
			zap.String("label", wrapper.GetLabel()),
			zap.String("value", wrapper.Value()))
	} else if wrapper.IsNestedInput() {
		// Update nested input (it handles its own navigation internally)
		wrapper.NestedInput.Update(msg)
		log.Debug("Nested input updated",
			zap.String("label", wrapper.GetLabel()))
	}

	// Value tracking (for potential future use)
	currentValue := wrapper.Value()
	_ = currentValue // Avoid unused variable warning
}

// updateObjectExpansion is no longer needed with split layout - objects stay collapsed in left panel
// and are shown expanded in the right panel when focused
func (ca *CreateAdapter) updateObjectExpansion() {
	// With split layout, we don't need to track expanded objects for rendering
	// Objects are always collapsed in the left panel, and shown expanded in the right panel
	// We still need to maintain the flatInputs structure for navigation
	ca.buildFlatInputs()
}

// rebuildIfExpansionChanged rebuilds flatInputs if the expansion state has changed
func (ca *CreateAdapter) rebuildIfExpansionChanged() {
	// Save the current number of flat inputs
	previousCount := len(ca.flatInputs)

	// Rebuild flat inputs (this will pick up any expansion changes)
	ca.buildFlatInputs()

	// If the count changed, we know expansion state changed
	if len(ca.flatInputs) != previousCount {
		log.Debug("Flat input count changed during navigation",
			zap.Int("previous", previousCount),
			zap.Int("current", len(ca.flatInputs)))

		// Adjust focus index if needed to stay within bounds
		if ca.focused >= len(ca.flatInputs) && len(ca.flatInputs) > 0 {
			ca.focused = len(ca.flatInputs) - 1
		}
	}
}

// blurCurrentFlatInput blurs the currently focused flat input
func (ca *CreateAdapter) blurCurrentFlatInput() {
	if ca.focused >= 0 && ca.focused < len(ca.flatInputs) {
		ca.blurInput(ca.flatInputs[ca.focused].wrapper)
	}
}

// ensureFocusedInputVisible ensures the currently focused input is visible by adjusting scroll position
func (ca *CreateAdapter) ensureFocusedInputVisible() {
	if len(ca.flatInputs) == 0 || ca.focused < 0 || ca.focused >= len(ca.flatInputs) {
		return
	}

	// ca.focused is already the index into flatInputs
	focusedFlatIndex := ca.focused

	// Scroll down if focused input is below visible area
	if focusedFlatIndex >= ca.visibleStartInput+ca.maxVisibleInputs {
		ca.visibleStartInput = focusedFlatIndex - ca.maxVisibleInputs + 1
	}
	// Scroll up if focused input is above visible area (wraparound case)
	if focusedFlatIndex < ca.visibleStartInput {
		ca.visibleStartInput = focusedFlatIndex
	}
}

// renderInputWrapper recursively renders an input wrapper with proper indentation for nested structures
func (ca *CreateAdapter) renderInputWrapper(wrapper common.InputWrapper, depth int, borderColor lipgloss.TerminalColor, isFocused bool, availableWidth int) []string {
	var rows []string

	// Calculate indentation based on depth
	indentation := strings.Repeat("  ", depth) // 2 spaces per level

	// Create label with type information
	labelText := wrapper.Label

	// Add type information in brackets for non-object types
	var typeText string
	if !wrapper.IsNestedInput() { // Don't show types for objects
		switch wrapper.GetType() {
		case common.InputTypeComplexArray:
			// Get detailed array type information for complex arrays
			var itemType string = "unknown"
			if wrapper.ComplexArrayInput != nil && wrapper.ComplexArrayInput.ItemDef != nil {
				switch wrapper.ComplexArrayInput.ItemDef.Type {
				case "string":
					itemType = "str"
				case "integer":
					itemType = "int"
				case "number":
					itemType = "float"
				case "boolean":
					itemType = "bool"
				case "object":
					itemType = "object"
				case "array":
					itemType = "array"
				default:
					itemType = wrapper.ComplexArrayInput.ItemDef.Type
				}
			} else {
				// Default to str for simple arrays without item definition
				itemType = "str"
			}
			typeText = " [array[" + itemType + "]]"
		case common.InputTypeText:
			// Check if text input has enum values
			if wrapper.TextInput != nil && len(wrapper.TextInput.GetOneOf()) > 0 {
				enumValues := strings.Join(wrapper.TextInput.GetOneOf(), ",")
				typeText = " [str|enum:" + enumValues + "]"
			} else {
				typeText = " [str]"
			}
		case common.InputTypeInt64:
			typeText = " [int]"
		case common.InputTypeFloat64:
			typeText = " [float]"
		case common.InputTypeBool:
			typeText = " [bool]"
		case common.InputTypePrimitivesArray:
			// Get array type from ArrayInput
			if wrapper.PrimitivesArrayInput != nil && "array[str]" != "" {
				typeText = " [" + "array[str]" + "]"
			} else {
				typeText = " [array[str]]" // Default fallback
			}
		}
	}

	// Style the type text in gray
	typeStyle := lipgloss.NewStyle().Foreground(LightGrey)
	styledType := typeStyle.Render(typeText)

	var label string
	yellowColon := lipgloss.NewStyle().Foreground(Yellow).Render(":")
	if wrapper.IsRequired() {
		redStar := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5353")).Render("*")
		label = labelText + redStar + styledType + yellowColon
	} else {
		label = labelText + styledType + yellowColon
	}

	// Crop label if it exceeds available width (be more aggressive for zone constraints)
	maxLabelWidth := availableWidth - 10 // Leave space for input field and borders
	if maxLabelWidth < 10 {
		maxLabelWidth = availableWidth / 3 // Minimum space allocation
	}

	if lipgloss.Width(label) > maxLabelWidth && maxLabelWidth > 5 {
		// Remove styled parts to get the raw text for cropping
		rawLabel := labelText + typeText
		if wrapper.IsRequired() {
			rawLabel += "*"
		}
		rawLabel += ":"

		// Crop the raw label more aggressively for zone constraints
		if len(rawLabel) > maxLabelWidth-3 && maxLabelWidth > 3 {
			rawLabel = rawLabel[:maxLabelWidth-3] + "..."
		}

		// Re-style the cropped label
		grayStyle := lipgloss.NewStyle().Foreground(LightGrey)
		yellowColon := lipgloss.NewStyle().Foreground(Yellow).Render(":")
		if wrapper.IsRequired() {
			parts := strings.Split(rawLabel, "*")
			if len(parts) == 2 {
				redStar := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5353")).Render("*")

				// Apply gray styling to type hints in the label parts
				styledPart0 := ca.styleTypeHintsInLabel(parts[0], grayStyle)
				styledPart1 := ca.styleTypeHintsInLabel(parts[1], grayStyle)

				label = styledPart0 + redStar + styledPart1
				if !strings.HasSuffix(label, ":") {
					label = strings.TrimSuffix(label, ":") + yellowColon
				}
			} else {
				rawLabelWithoutColon := strings.TrimSuffix(rawLabel, ":")
				label = ca.styleTypeHintsInLabel(rawLabelWithoutColon, grayStyle) + yellowColon
			}
		} else {
			rawLabelWithoutColon := strings.TrimSuffix(rawLabel, ":")
			label = ca.styleTypeHintsInLabel(rawLabelWithoutColon, grayStyle) + yellowColon
		}
	}

	// Handle nested inputs differently
	if wrapper.IsNestedInput() {
		// Render nested input with border and label in top-left corner
		return ca.renderNestedInput(wrapper, depth, borderColor, isFocused, availableWidth, indentation, label)
	}

	// Use the isFocused parameter passed from zone-based navigation
	actualInputFocused := isFocused

	// Render regular input
	var labelPrefix string
	if isFocused {
		labelPrefix = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("▶ ")
	} else {
		labelPrefix = "  "
	}
	labelRow := " " + indentation + labelPrefix + lipgloss.NewStyle().Foreground(Yellow).Render(label)

	// Input field - adjust width based on indentation and zone constraints
	baseIndentWidth := len(indentation) + 1 // +1 for left margin
	remainingWidth := availableWidth - baseIndentWidth
	inputWidth := max(15, remainingWidth/2) // Minimum 15 chars, use half of remaining width
	if inputWidth > remainingWidth-5 {      // Leave some space for borders
		inputWidth = max(10, remainingWidth-5)
	}

	var inputView string
	if wrapper.IsTextInput() {
		inputView = wrapper.TextInput.View()
	} else if wrapper.IsBoolInput() {
		inputView = wrapper.BoolInput.View()
	} else if wrapper.IsInt64Input() {
		inputView = wrapper.Int64Input.View()
	} else if wrapper.IsFloat64Input() {
		inputView = wrapper.Float64Input.View()
	} else if wrapper.IsComplexArrayInput() {
		// Sync array expansion state with CreateAdapter's expansion state (only complex arrays now)
		if ca.focused >= 0 && ca.focused < len(ca.flatInputs) {
			flatInput := ca.flatInputs[ca.focused]
			if flatInput.wrapper == &wrapper && flatInput.depth == 0 {
				// This array is currently focused - check if it should be expanded
				topLevelIndex := flatInput.parentPath[0]
				if ca.expandedObjects[topLevelIndex] {
					wrapper.ComplexArrayInput.IsExpanded = true
				} else {
					wrapper.ComplexArrayInput.IsExpanded = false
				}
			}
		}
		inputView = wrapper.ComplexArrayInput.View()
	} else if wrapper.IsPrimitivesArrayInput() {
		inputView = wrapper.PrimitivesArrayInput.View()
	}

	// Use different border color if this specific input is focused
	inputBorderColor := borderColor
	if actualInputFocused {
		inputBorderColor = Blue
	}

	// Create input row with left border manually - use thick border for focused fields
	leftBorderStyle := lipgloss.NewStyle().Foreground(inputBorderColor)
	var leftBorderChar, cornerChar string
	var horizontalChar string

	if actualInputFocused {
		// Focused: use thick border
		leftBorderChar = leftBorderStyle.Render("┃")
		cornerChar = leftBorderStyle.Render("┗")
		horizontalChar = "━"
	} else {
		// Not focused: use thin border
		leftBorderChar = leftBorderStyle.Render("│")
		cornerChar = leftBorderStyle.Render("└")
		horizontalChar = "─"
	}

	inputRow := " " + indentation + leftBorderChar + " " + inputView

	// Bottom border - create L-shaped corner with appropriate thickness
	bottomLine := lipgloss.NewStyle().Foreground(inputBorderColor).Width(inputWidth - 1).Render(strings.Repeat(horizontalChar, inputWidth-1))
	bottomBorder := " " + indentation + cornerChar + bottomLine

	rows = append(rows, labelRow)
	rows = append(rows, inputRow)
	rows = append(rows, bottomBorder)

	return rows
}

// renderNestedInput renders a nested input with proper border and child indentation
func (ca *CreateAdapter) renderNestedInput(wrapper common.InputWrapper, depth int, borderColor lipgloss.TerminalColor, isFocused bool, availableWidth int, indentation string, label string) []string {
	var rows []string

	nestedInput := wrapper.NestedInput
	if nestedInput == nil {
		return rows
	}

	// Calculate dimensions for the nested border - be more conservative in zones
	nestedWidth := availableWidth - len(indentation) - 2 // Account for parent indentation and border
	if nestedWidth < 8 {
		nestedWidth = max(8, availableWidth-2) // Very minimum width for zones
	}
	// Cap the nested width to prevent overflow
	if nestedWidth > availableWidth-2 {
		nestedWidth = availableWidth - 2
	}

	// Top border with label in top-left corner - use same color as labels (Yellow)
	// Use thin border for nested objects (they don't have individual focus like input fields)
	topBorderStyle := lipgloss.NewStyle().Foreground(Yellow)
	topLeftCorner := topBorderStyle.Render("┌")

	// Create label with background
	labelStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("240")).
		Foreground(Yellow).
		Padding(0, 1)
	styledLabel := labelStyle.Render(label)

	// Top border: corner + label + remaining line
	topBorder := " " + indentation + topLeftCorner + styledLabel
	rows = append(rows, topBorder)

	// Render child inputs recursively
	childInputs := nestedInput.GetInputs() // Returns []InputWrapper
	for _, childInput := range childInputs {
		childRows := ca.renderInputWrapper(childInput, depth+1, borderColor, false, nestedWidth) // Use value directly

		// Add left border to each child row - use same Yellow color (thin border style)
		for _, childRow := range childRows {
			leftBorderChar := topBorderStyle.Render("│")
			borderedRow := " " + indentation + leftBorderChar + childRow[len(indentation)+1:] // Replace indentation part
			rows = append(rows, borderedRow)
		}
	}

	// Bottom border - use same Yellow color as top border (thin border style)
	bottomLeftCorner := topBorderStyle.Render("└")
	bottomLine := strings.Repeat(" ", nestedWidth-1)
	bottomBorder := " " + indentation + bottomLeftCorner + bottomLine
	rows = append(rows, bottomBorder)

	return rows
}

// renderCollapsedObject renders a collapsed nested object or array as just a border with label
func (ca *CreateAdapter) renderCollapsedObject(wrapper common.InputWrapper, borderColor lipgloss.TerminalColor, innerWidth int) []string {
	var rows []string

	if !wrapper.IsNestedInput() && !wrapper.IsComplexArrayInput() {
		// Not a nested object or array, render normally
		return ca.renderInputWrapper(wrapper, 0, borderColor, false, innerWidth)
	}

	// Create collapsed display
	labelText := wrapper.GetLabel()

	// Style the type text in gray
	typeStyle := lipgloss.NewStyle().Foreground(LightGrey)
	var styledType string
	if wrapper.IsNestedInput() {
		// Show large triangle arrow when this object is focused (right side visible)
		if borderColor == Blue {
			arrowStyle := lipgloss.NewStyle().Foreground(White)
			styledType = typeStyle.Render(" [object] ") + arrowStyle.Render("▶▶▶")
		} else {
			styledType = typeStyle.Render(" [object]")
		}
	} else if wrapper.IsComplexArrayInput() {
		// Get detailed array type information
		var itemType string = "unknown"
		if wrapper.ComplexArrayInput != nil && wrapper.ComplexArrayInput.ItemDef != nil {
			switch wrapper.ComplexArrayInput.ItemDef.Type {
			case "string":
				itemType = "str"
			case "integer":
				itemType = "int"
			case "number":
				itemType = "float"
			case "boolean":
				itemType = "bool"
			case "object":
				itemType = "object"
			case "array":
				itemType = "array"
			default:
				itemType = wrapper.ComplexArrayInput.ItemDef.Type
			}
		} else {
			// Default to str for simple arrays without item definition
			itemType = "str"
		}
		styledType = typeStyle.Render(" [array[" + itemType + "]]")
	}

	// Crop label if too long (similar to regular input labels)
	fullLabel := labelText + " " + styledType
	maxLabelWidth := innerWidth / 2
	if lipgloss.Width(fullLabel) > maxLabelWidth && maxLabelWidth > 10 {
		// Get the raw type text for cropping
		var rawTypeText string
		if wrapper.IsNestedInput() {
			if borderColor == Blue {
				rawTypeText = " [object] ▶▶▶"
			} else {
				rawTypeText = " [object]"
			}
		} else if wrapper.IsComplexArrayInput() {
			var itemType string = "str" // default
			if wrapper.ComplexArrayInput != nil && wrapper.ComplexArrayInput.ItemDef != nil {
				switch wrapper.ComplexArrayInput.ItemDef.Type {
				case "string":
					itemType = "str"
				case "integer":
					itemType = "int"
				case "number":
					itemType = "float"
				case "boolean":
					itemType = "bool"
				case "object":
					itemType = "object"
				case "array":
					itemType = "array"
				default:
					itemType = wrapper.ComplexArrayInput.ItemDef.Type
				}
			}
			rawTypeText = " [array[" + itemType + "]]"
		}

		rawLabel := labelText + rawTypeText
		if len(rawLabel) > maxLabelWidth-6 { // Account for padding and "..."
			rawLabel = rawLabel[:maxLabelWidth-6] + "..."
		}
		fullLabel = rawLabel
	}

	// Style the label with Yellow color (no background)
	labelStyle := lipgloss.NewStyle().
		Foreground(Yellow).
		Padding(0, 1)
	styledLabel := labelStyle.Render(fullLabel)

	// Create border with label - use borderColor for focused objects, Yellow for others
	var borderStyle lipgloss.Style
	var topLeftCorner, bottomLeftCorner string

	if borderColor == Blue {
		// Focused object: use thick blue border
		borderStyle = lipgloss.NewStyle().Foreground(borderColor)
		topLeftCorner = borderStyle.Render("┏")
		bottomLeftCorner = borderStyle.Render("┗")
	} else {
		// Not focused: use thin yellow border
		borderStyle = lipgloss.NewStyle().Foreground(Yellow)
		topLeftCorner = borderStyle.Render("┌")
		bottomLeftCorner = borderStyle.Render("└")
	}

	// Calculate input width for collapsed objects in zones
	baseIndentWidth := 1 // +1 for left margin
	remainingWidth := innerWidth - baseIndentWidth
	inputWidth := max(15, remainingWidth/2) // Conservative width for zones
	if inputWidth > remainingWidth-3 {      // Leave space for borders
		inputWidth = max(10, remainingWidth-3)
	}

	// Top border with label
	topBorder := " " + topLeftCorner + styledLabel

	// Bottom border - use consistent style and thickness with top border
	var horizontalChar string
	if borderColor == Blue {
		horizontalChar = "━" // thick horizontal line for focused
	} else {
		horizontalChar = "─" // thin horizontal line for not focused
	}
	bottomLine := borderStyle.Render(strings.Repeat(horizontalChar, inputWidth-1))
	bottomBorder := " " + bottomLeftCorner + bottomLine

	rows = append(rows, topBorder)
	rows = append(rows, bottomBorder)

	return rows
}

// buildFlatInputs creates a flattened list of all inputs for navigation
func (ca *CreateAdapter) buildFlatInputs() {
	ca.flatInputs = nil
	for i := range ca.inputs {
		ca.flattenInput(&ca.inputs[i], []int{i}, 0)
	}
}

// flattenInput recursively flattens an input and its children
func (ca *CreateAdapter) flattenInput(wrapper *common.InputWrapper, parentPath []int, depth int) {
	// With split layout, include nested inputs and complex arrays in navigation
	// so they can be focused and displayed in the right panel
	if wrapper.IsNestedInput() || wrapper.IsComplexArrayInput() {
		// Add the object/array itself to navigation so it can be focused
		ca.flatInputs = append(ca.flatInputs, FlatInput{
			wrapper:    wrapper,
			parentPath: parentPath,
			depth:      depth,
		})

		// Flatten children if this is a nested input (object)
		if wrapper.IsNestedInput() {
			if wrapper.NestedInput != nil {
				childInputs := wrapper.NestedInput.GetInputs() // Returns []InputWrapper
				for i := range childInputs {
					childPath := append(parentPath, i)
					ca.flattenInput(&childInputs[i], childPath, depth+1) // Take address of the value
				}
			}
		}

		// Flatten children if this is a complex array input
		if wrapper.IsComplexArrayInput() {
			if wrapper.ComplexArrayInput != nil && !wrapper.ComplexArrayInput.IsSimpleType {
				// For arrays of objects/arrays, flatten the active tab's sub-form inputs
				if len(wrapper.ComplexArrayInput.ItemForms) > 0 {
					activeTab := wrapper.ComplexArrayInput.ActiveTab
					if activeTab >= 0 && activeTab < len(wrapper.ComplexArrayInput.ItemForms) {
						subForm := wrapper.ComplexArrayInput.ItemForms[activeTab]
						for i := range subForm {
							childPath := append(parentPath, activeTab, i) // Include tab index in path
							ca.flattenInput(&subForm[i], childPath, depth+1)
						}
					}
				}
			}
		}
	} else {
		// Add regular inputs to navigation
		ca.flatInputs = append(ca.flatInputs, FlatInput{
			wrapper:    wrapper, // Store reference to the actual wrapper, not a copy
			parentPath: parentPath,
			depth:      depth,
		})
	}
}
