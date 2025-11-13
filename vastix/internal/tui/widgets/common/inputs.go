package common

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"vastix/internal/colors"
	log "vastix/internal/logging"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.uber.org/zap"
)

type SkipDefinition struct {
	defName string
	reason  string
}

func (s SkipDefinition) Error() string {
	if s.reason != "" {
		return fmt.Sprintf("skip definition: %s (reason: %s)", s.defName, s.reason)
	}
	return fmt.Sprintf("skip definition: %s", s.defName)
}

// InputType represents the different types of inputs supported
type InputType int

const (
	InputTypeText InputType = iota
	InputTypeBool
	InputTypeInt64
	InputTypeFloat64
	InputTypeComplexArray
	InputTypePrimitivesArray
	InputTypeNested
)

// String returns the string representation of InputType
func (it InputType) String() string {
	switch it {
	case InputTypeText:
		return "text"
	case InputTypeBool:
		return "bool"
	case InputTypeInt64:
		return "int64"
	case InputTypeFloat64:
		return "float64"
	case InputTypeComplexArray:
		return "complex_array"
	case InputTypePrimitivesArray:
		return "primitives_array"
	case InputTypeNested:
		return "nested"
	default:
		return "unknown"
	}
}

// TextInputWrapper wraps textinput.Model to implement InputField interface
type TextInputWrapper struct {
	TextInput *textinput.Model
	oneOf     []string // Valid options for enum-like validation
	required  bool
	label     string
	arrayType string // Store array type info (e.g., "array[string]", "array[int]", or "" for non-arrays)
}

// NewTextInputWrapperFromModel creates a wrapper around textinput.Model
func NewTextInputWrapperFromModel(ti *textinput.Model) *TextInputWrapper {
	ti.Prompt = "" // Remove any prompt
	return &TextInputWrapper{
		TextInput: ti,
		oneOf:     []string{},
		required:  false,
		label:     "",
		arrayType: "",
	}
}

// NewTextInputWrapperWithOneOf creates a wrapper with OneOf validation
func NewTextInputWrapperWithOneOf(ti *textinput.Model, oneOf []string) *TextInputWrapper {
	ti.Prompt = "" // Remove any prompt
	return &TextInputWrapper{TextInput: ti, oneOf: oneOf}
}

// Update handles updates for text input
func (t *TextInputWrapper) Update(msg tea.Msg) tea.Cmd {
	// Intercept right arrow key to accept suggestions (instead of tab)
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyRight && t.TextInput.ShowSuggestions && len(t.TextInput.AvailableSuggestions()) > 0 {
			// Convert right arrow to tab to accept suggestion
			msg = tea.KeyMsg{Type: tea.KeyTab}
		}
	}

	var cmd tea.Cmd
	*t.TextInput, cmd = t.TextInput.Update(msg)
	return cmd
}

// View renders the text input
func (t *TextInputWrapper) View() string {
	return t.TextInput.View()
}

// Focus sets focus to the text input
func (t *TextInputWrapper) Focus() {
	t.TextInput.Focus()
}

// Blur removes focus from the text input
func (t *TextInputWrapper) Blur() {
	t.TextInput.Blur()
}

// Value returns the current text value
func (t *TextInputWrapper) Value() string {
	return t.TextInput.Value()
}

// SetValue sets the text value
func (t *TextInputWrapper) SetValue(val string) {
	t.TextInput.SetValue(val)
}

// Validate validates the text input
func (t *TextInputWrapper) Validate() error {
	value := strings.TrimSpace(t.TextInput.Value())

	// Check OneOf validation if specified and value is not empty
	if len(t.oneOf) > 0 && value != "" {
		for _, option := range t.oneOf {
			if value == option {
				return nil // Valid option found
			}
		}
		return fmt.Errorf("value '%s' is not one of the allowed options: %v", value, t.oneOf)
	}

	return nil
}

// SetOneOf sets the valid options for enum-like validation
func (t *TextInputWrapper) SetOneOf(options []string) {
	t.oneOf = options
}

// GetOneOf returns the valid options
func (t *TextInputWrapper) GetOneOf() []string {
	return t.oneOf
}

// TypedValue returns the text value as string
func (t *TextInputWrapper) TypedValue() interface{} {
	return t.TextInput.Value()
}

// IsDefaultValue returns true if this is the default value (empty string)
func (t *TextInputWrapper) IsDefaultValue() bool {
	return strings.TrimSpace(t.TextInput.Value()) == ""
}

// SetPrompt sets the prompt for the text input
func (t *TextInputWrapper) SetPrompt(prompt string) {
	t.TextInput.Prompt = prompt
}

// GetTextInput returns the underlying textinput.Model for direct access when needed
func (t *TextInputWrapper) GetTextInput() *textinput.Model {
	return t.TextInput
}

// BoolInput represents a boolean input that toggles with y/n keys
type BoolInput struct {
	value       bool
	focused     bool
	label       string
	description string
}

// NewBoolInput creates a new boolean input
func NewBoolInput(defaultValue bool, description string) *BoolInput {
	return &BoolInput{
		value:       defaultValue,
		focused:     false,
		description: description,
	}
}

// Update handles key messages for the boolean input
func (b *BoolInput) Update(msg tea.Msg) tea.Cmd {
	if !b.focused {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			b.value = true
		case "n", "N":
			b.value = false
		case " ":
			b.value = !b.value // Toggle on space
		}
	}
	return nil
}

// View renders the boolean input as a modern pill-shaped toggle switch
func (b *BoolInput) View() string {
	var toggleSwitch string

	if b.value {
		// ON state: Green pill with white circle on the right
		pillStyle := lipgloss.NewStyle().
			Background(colors.BrightGreen). // Bright green
			Foreground(colors.BrightGreen). // Hidden text
			Bold(true)
		switchStyle := lipgloss.NewStyle().
			Background(colors.WhiteTerm). // White circle
			Foreground(colors.WhiteTerm). // White text
			Bold(true)

		// Create rounded pill shape: ●●●○
		leftPill := pillStyle.Render("●●●")
		switchCircle := switchStyle.Render("○")
		toggleSwitch = leftPill + switchCircle

	} else {
		// OFF state: Light gray pill with white circle on the left
		switchStyle := lipgloss.NewStyle().
			Background(colors.WhiteTerm). // White circle
			Foreground(colors.WhiteTerm). // White text
			Bold(true)
		pillStyle := lipgloss.NewStyle().
			Background(colors.LighterGrey). // Light gray
			Foreground(colors.LighterGrey). // Hidden text
			Bold(true)

		// Create rounded pill shape: ○●●●
		switchCircle := switchStyle.Render("○")
		rightPill := pillStyle.Render("●●●")
		toggleSwitch = switchCircle + rightPill
	}

	// No visual focus indicator needed - the toggle is clear enough

	// Show description with dimmed styling if available
	if b.description != "" {
		// Truncate description to 64 characters
		description := b.description
		if len(description) > 64 {
			description = description[:61] + "..."
		}

		dimStyle := lipgloss.NewStyle().Foreground(colors.Grey240)
		dimmedDescription := dimStyle.Render(description)
		return toggleSwitch + "  " + dimmedDescription
	}

	return toggleSwitch
}

// Focus sets focus to the boolean input
func (b *BoolInput) Focus() {
	b.focused = true
}

// Blur removes focus from the boolean input
func (b *BoolInput) Blur() {
	b.focused = false
}

// Value returns the current boolean value as a string
func (b *BoolInput) Value() string {
	if b.value {
		return "true"
	}
	return "false"
}

// SetValue sets the boolean value from a string
func (b *BoolInput) SetValue(val string) {
	val = strings.ToLower(val)
	b.value = val == "true" || val == "yes" || val == "y" || val == "1"
}

// Validate validates the boolean input (always valid)
func (b *BoolInput) Validate() error {
	return nil
}

// TypedValue returns the boolean value
func (b *BoolInput) TypedValue() interface{} {
	return b.value
}

// IsDefaultValue returns true if this is the default value (false)
func (b *BoolInput) IsDefaultValue() bool {
	// Default value for boolean is false
	return !b.value
}

// Int64Input represents an integer input field
type Int64Input struct {
	TextInput *textinput.Model
	value     int64
	focused   bool
}

// NewInt64Input creates a new int64 input
func NewInt64Input(defaultValue string) *Int64Input {
	var (
		defaultValueInt int64
		err             error
	)
	if defaultValue == "" {
		defaultValueInt = 0 // Default to 0 if no value provided
	} else {
		defaultValueInt, err = strconv.ParseInt(defaultValue, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("Invalid default value for Int64Input: %s", defaultValue))
		}
	}

	ti := textinput.New()
	ti.SetValue(defaultValue)
	ti.CharLimit = 20
	ti.Width = 30
	ti.Prompt = ""

	return &Int64Input{
		TextInput: &ti,
		value:     defaultValueInt,
		focused:   false,
	}
}

// Update handles updates for int64 input
func (i *Int64Input) Update(msg tea.Msg) tea.Cmd {
	// Intercept right arrow key to accept suggestions (instead of tab)
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyRight && i.TextInput.ShowSuggestions && len(i.TextInput.AvailableSuggestions()) > 0 {
			// Convert right arrow to tab to accept suggestion
			msg = tea.KeyMsg{Type: tea.KeyTab}
		}
	}

	// Let the underlying textinput handle focus state like TextInputWrapper does
	var cmd tea.Cmd
	*i.TextInput, cmd = i.TextInput.Update(msg)

	// Try to parse the value
	if val, err := strconv.ParseInt(i.TextInput.Value(), 10, 64); err == nil {
		i.value = val
	}

	return cmd
}

// View renders the int64 input
func (i *Int64Input) View() string {
	return i.TextInput.View()
}

// Focus sets focus to the int64 input
func (i *Int64Input) Focus() {
	i.focused = true
	i.TextInput.Focus()
}

// Blur removes focus from the int64 input
func (i *Int64Input) Blur() {
	i.focused = false
	i.TextInput.Blur()
}

// Value returns the current int64 value as a string
func (i *Int64Input) Value() string {
	return i.TextInput.Value()
}

// SetValue sets the int64 value from a string
func (i *Int64Input) SetValue(val string) {
	i.TextInput.SetValue(val)
	if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
		i.value = parsed
	}
}

// Validate validates the int64 input
func (i *Int64Input) Validate() error {
	value := i.TextInput.Value()
	if value == "" {
		return fmt.Errorf("integer value cannot be empty")
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid integer value: %s", value)
	}

	// Update the stored value if parsing succeeded
	i.value = parsed
	return nil
}

// TypedValue returns the int64 value
func (i *Int64Input) TypedValue() interface{} {
	// Ensure we return the parsed value, attempt to parse if needed
	if val, err := strconv.ParseInt(i.TextInput.Value(), 10, 64); err == nil {
		i.value = val
		return val
	}
	return i.value
}

// IsDefaultValue returns true if this is the default value (0)
func (i *Int64Input) IsDefaultValue() bool {
	// Default is when value is 0 and text input is either empty or "0"
	textValue := i.TextInput.Value()
	return i.value == 0 && (textValue == "" || textValue == "0")
}

// Float64Input represents a float input field
type Float64Input struct {
	TextInput *textinput.Model
	value     float64
	focused   bool
}

// NewFloat64Input creates a new float64 input
func NewFloat64Input(defaultValue string) *Float64Input {
	var (
		defaultValueFloat float64
		err               error
	)
	if defaultValue == "" {
		defaultValueFloat = 0.0 // Default to 0.0 if no value provided
	} else {
		defaultValueFloat, err = strconv.ParseFloat(defaultValue, 64)
		if err != nil {
			panic(fmt.Sprintf("Invalid default value for Float64Input: %s", defaultValue))
		}
		defaultValue = fmt.Sprintf("%.2f", defaultValueFloat)           // Format to 2 decimal places
		defaultValueFloat = float64(int(defaultValueFloat*100)) / 100.0 // Ensure 2 decimal precision
	}

	ti := textinput.New()
	ti.SetValue(defaultValue)
	ti.CharLimit = 30
	ti.Prompt = "" // Remove any default prompt

	return &Float64Input{
		TextInput: &ti,
		value:     defaultValueFloat,
		focused:   false,
	}
}

// Update handles updates for float64 input
func (f *Float64Input) Update(msg tea.Msg) tea.Cmd {
	// Let the underlying textinput handle focus state like TextInputWrapper does
	var cmd tea.Cmd
	*f.TextInput, cmd = f.TextInput.Update(msg)

	// Try to parse the value
	if val, err := strconv.ParseFloat(f.TextInput.Value(), 64); err == nil {
		f.value = val
	}

	return cmd
}

// View renders the float64 input
func (f *Float64Input) View() string {
	return f.TextInput.View()
}

// Focus sets focus to the float64 input
func (f *Float64Input) Focus() {
	f.focused = true
	f.TextInput.Focus()
}

// Blur removes focus from the float64 input
func (f *Float64Input) Blur() {
	f.focused = false
	f.TextInput.Blur()
}

// Value returns the current float64 value as a string
func (f *Float64Input) Value() string {
	return f.TextInput.Value()
}

// SetValue sets the float64 value from a string
func (f *Float64Input) SetValue(val string) {
	f.TextInput.SetValue(val)
	if parsed, err := strconv.ParseFloat(val, 64); err == nil {
		f.value = parsed
	}
}

// Validate validates the float64 input
func (f *Float64Input) Validate() error {
	value := f.TextInput.Value()
	if value == "" {
		return fmt.Errorf("float value cannot be empty")
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("invalid float value: %s", value)
	}

	// Update the stored value if parsing succeeded
	f.value = parsed
	return nil
}

// TypedValue returns the float64 value
func (f *Float64Input) TypedValue() interface{} {
	// Ensure we return the parsed value, attempt to parse if needed
	if val, err := strconv.ParseFloat(f.TextInput.Value(), 64); err == nil {
		f.value = val
		return val
	}
	return f.value
}

// IsDefaultValue returns true if this is the default value (0.0)
func (f *Float64Input) IsDefaultValue() bool {
	return f.value == 0.0 && f.TextInput.Value() == "0.00"
}

// PrimitivesArrayInput represents a simple inline array input field for primitive types
type PrimitivesArrayInput struct {
	TextInput   *textinput.Model
	values      []string // All completed values (chips)
	currentText string   // Text currently being typed (before comma)
	focused     bool
	label       string
	arrayType   string // Store array type info (e.g., "array[string]", "array[int]")
	placeholder string // Placeholder text to show when empty
}

// NewPrimitivesArrayInput creates a new primitives array input
func NewPrimitivesArrayInput(defaultValues []string) *PrimitivesArrayInput {
	return NewPrimitivesArrayInputWithType(defaultValues, "array[string]")
}

// NewPrimitivesArrayInputWithType creates a new primitives array input with type information
func NewPrimitivesArrayInputWithType(defaultValues []string, arrayType string) *PrimitivesArrayInput {
	return NewPrimitivesArrayInputWithTypePlaceholder(defaultValues, arrayType, "")
}

// NewPrimitivesArrayInputWithTypePlaceholder creates a new primitives array input with type and placeholder
func NewPrimitivesArrayInputWithTypePlaceholder(defaultValues []string, arrayType, placeholder string) *PrimitivesArrayInput {
	ti := textinput.New()
	ti.CharLimit = 500
	ti.Prompt = ""
	ti.SetValue("")              // Start with empty current text
	ti.Placeholder = placeholder // Set placeholder on underlying textinput for standard styling

	// Auto-size width based on placeholder length, with reasonable bounds
	width := 30 // Default minimum width
	if placeholder != "" {
		placeholderLen := len(placeholder)
		if placeholderLen > 30 {
			width = placeholderLen + 5 // Add some padding
		}
		// Cap at reasonable maximum to avoid extremely wide inputs
		if width > 80 {
			width = 80
		}
	}
	ti.Width = width

	return &PrimitivesArrayInput{
		TextInput:   &ti,
		values:      defaultValues, // These are completed chips
		currentText: "",            // No text being typed initially
		focused:     false,
		label:       "",
		arrayType:   arrayType,
		placeholder: placeholder,
	}
}

// Update handles updates for primitives array input with chip creation on comma
func (a *PrimitivesArrayInput) Update(msg tea.Msg) tea.Cmd {
	if !a.focused {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case ",":
			// Convert current text to a chip and clear input for next entry
			currentText := strings.TrimSpace(a.TextInput.Value())
			if currentText != "" {
				// Add current text as a completed value (chip)
				a.values = append(a.values, currentText)
			}
			// Clear the text input for the next entry
			a.TextInput.SetValue("")
			a.currentText = ""
			return nil
		case "backspace":
			// If text input is empty and we have chips, remove the last chip
			if a.TextInput.Value() == "" && len(a.values) > 0 {
				// Remove last chip and put its text back in the input
				lastIndex := len(a.values) - 1
				lastValue := a.values[lastIndex]
				a.values = a.values[:lastIndex]
				a.TextInput.SetValue(lastValue)
				a.TextInput.SetCursor(len(lastValue))
				return nil
			}
			// Otherwise let normal backspace handling occur
		}
	}

	// Let the underlying textinput handle other keys
	var cmd tea.Cmd
	*a.TextInput, cmd = a.TextInput.Update(msg)
	a.currentText = a.TextInput.Value()
	return cmd
}

// getAllValues returns all values including current text being typed
func (a *PrimitivesArrayInput) getAllValues() []string {
	allValues := make([]string, len(a.values))
	copy(allValues, a.values)

	// Add current text if it's not empty
	currentText := strings.TrimSpace(a.TextInput.Value())
	if currentText != "" {
		allValues = append(allValues, currentText)
	}

	return allValues
}

// View renders the primitives array input with chips and current text input
func (a *PrimitivesArrayInput) View() string {
	// Style for completed chips (green background like enabled checkbox) with black text
	chipStyle := lipgloss.NewStyle().
		Background(colors.BrightGreen). // Bright green
		Foreground(colors.BlackTerm).   // Black text
		Padding(0, 1).                  // Small padding around text
		MarginRight(1)                  // Space between chips

	var result strings.Builder

	// Render completed chips
	for _, value := range a.values {
		chip := chipStyle.Render(value)
		result.WriteString(chip)
	}

	// Render current text input
	currentText := a.TextInput.Value()
	if a.focused {
		// When focused, show cursor
		cursor := a.TextInput.Position()
		if cursor > len(currentText) {
			cursor = len(currentText)
		}

		beforeCursor := currentText[:cursor]
		afterCursor := currentText[cursor:]
		cursorChar := "|"

		result.WriteString(beforeCursor + cursorChar + afterCursor)
	} else {
		// When not focused, just show the current text
		result.WriteString(currentText)
	}

	// If no chips and no current text, show the underlying textinput with its standard placeholder styling
	if len(a.values) == 0 && currentText == "" {
		return a.TextInput.View()
	}

	return result.String()
}

// Focus sets focus to the array input
func (a *PrimitivesArrayInput) Focus() {
	a.focused = true
	a.TextInput.Focus()
}

// Blur removes focus from the array input
func (a *PrimitivesArrayInput) Blur() {
	a.focused = false
	// Finalize current text into a chip on blur so it remains visible
	current := strings.TrimSpace(a.TextInput.Value())
	if current != "" {
		a.values = append(a.values, current)
		a.TextInput.SetValue("")
		a.currentText = ""
	}
	a.TextInput.Blur()
}

// Value returns the current array value as a JSON string
func (a *PrimitivesArrayInput) Value() string {
	allValues := a.getAllValues()
	if len(allValues) == 0 {
		return "[]"
	}

	// Convert to JSON format
	jsonBytes, err := json.Marshal(allValues)
	if err != nil {
		// Fallback to simple comma-separated format
		return "[" + strings.Join(allValues, ", ") + "]"
	}
	return string(jsonBytes)
}

// SetValue sets the array value from a string (expects JSON array or comma-separated)
func (a *PrimitivesArrayInput) SetValue(val string) {
	val = strings.TrimSpace(val)

	// Try to parse as JSON array first
	var jsonValues []string
	if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
		if err := json.Unmarshal([]byte(val), &jsonValues); err == nil {
			a.values = jsonValues
			a.TextInput.SetValue("") // Clear current text input
			a.currentText = ""
			return
		}
	}

	// Fallback to comma-separated parsing
	if val == "" || val == "[]" {
		a.values = []string{}
		a.TextInput.SetValue("")
		a.currentText = ""
	} else {
		// Remove brackets if present
		val = strings.TrimPrefix(val, "[")
		val = strings.TrimSuffix(val, "]")

		parts := strings.Split(val, ",")
		a.values = make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				a.values = append(a.values, trimmed)
			}
		}
		a.TextInput.SetValue("") // Clear current text input
		a.currentText = ""
	}
}

// Validate validates the array input
func (a *PrimitivesArrayInput) Validate() error {
	// Array inputs are generally always valid
	return nil
}

// TypedValue returns the string slice
func (a *PrimitivesArrayInput) TypedValue() interface{} {
	return a.getAllValues()
}

// IsDefaultValue returns true if this is the default value (empty array)
func (a *PrimitivesArrayInput) IsDefaultValue() bool {
	return len(a.values) == 0 && strings.TrimSpace(a.TextInput.Value()) == ""
}

// ComplexArrayInput represents an array input field with tab-based navigation for complex types
type ComplexArrayInput struct {
	TextInput  *textinput.Model   // Fallback for simple comma-separated mode
	ItemInputs []*textinput.Model // Individual inputs for simple types
	ItemForms  [][]InputWrapper   // Sub-forms for complex types (objects/arrays)
	Values     []interface{}      // Changed from []string to []interface{} for complex types
	Focused    bool

	// Tab-based navigation
	ActiveTab  int  // Currently active tab (0-based)
	IsExpanded bool // Whether showing tabs or collapsed view

	// Navigation state
	ShowControls bool // Whether to show +/- controls

	// Item type definition
	ItemDef      *InputDefinition // Schema for array items
	IsSimpleType bool             // Whether items are simple strings or complex types
}

// NewComplexArrayInput creates a new complex array input
func NewComplexArrayInput(defaultValues []string) *ComplexArrayInput {
	return NewComplexArrayInputWithItemDef(defaultValues, nil)
}

// NewComplexArrayInputWithItemDef creates a new complex array input with item definition for complex types
func NewComplexArrayInputWithItemDef(defaultValues []string, itemDef *InputDefinition) *ComplexArrayInput {
	ti := textinput.New()
	ti.SetValue(strings.Join(defaultValues, ", "))
	ti.CharLimit = 500

	// Convert string values to interface{}
	values := make([]interface{}, len(defaultValues))
	for i, v := range defaultValues {
		values[i] = v
	}

	// Determine if this is a simple type or complex type
	isSimpleType := itemDef == nil || itemDef.Type == "string" || itemDef.Type == "integer" || itemDef.Type == "number" || itemDef.Type == "boolean"

	var itemInputs []*textinput.Model
	var itemForms [][]InputWrapper

	if isSimpleType {
		// Create individual inputs for each default value (simple types)
		for i, value := range defaultValues {
			input := textinput.New()
			input.SetValue(value)
			input.CharLimit = 200
			input.Width = 30
			if i == 0 {
				input.Focus() // Focus first tab by default
			}
			itemInputs = append(itemInputs, &input)
		}

		// If no default values, create one empty input
		if len(defaultValues) == 0 {
			input := textinput.New()
			input.CharLimit = 200
			input.Width = 30
			input.Focus()
			itemInputs = append(itemInputs, &input)
			values = append(values, "")
		}
	} else {
		// Create sub-forms for complex types (objects/arrays)
		for range defaultValues {
			// Create sub-form inputs based on item definition
			subForm, err := createSubFormFromDefinition(itemDef)
			if err == nil {
				itemForms = append(itemForms, subForm)
			}
		}

		// If no default values, create one empty sub-form
		if len(defaultValues) == 0 {
			subForm, err := createSubFormFromDefinition(itemDef)
			if err == nil {
				itemForms = append(itemForms, subForm)
				values = append(values, make(map[string]interface{}))
			}
		}
	}

	return &ComplexArrayInput{
		TextInput:    &ti,
		ItemInputs:   itemInputs,
		ItemForms:    itemForms,
		Values:       values,
		Focused:      false,
		ActiveTab:    0,
		IsExpanded:   false, // Start collapsed
		ShowControls: true,
		ItemDef:      itemDef,
		IsSimpleType: isSimpleType,
	}
}

// Update handles updates for complex array input
func (l *ComplexArrayInput) Update(msg tea.Msg) tea.Cmd {
	if !l.Focused {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			// Toggle expansion when tab is pressed on collapsed array
			if !l.IsExpanded {
				l.IsExpanded = true
				l.focusActiveTab()
				return nil
			}
			// If expanded, let normal tab navigation handle it
			return nil

		case "shift+tab":
			// Collapse array when shift+tab is pressed on expanded array
			if l.IsExpanded {
				l.IsExpanded = false
				l.blurAllTabs()
				return nil
			}
			return nil

		case ">", "right":
			// Navigate to next tab
			if l.IsExpanded && len(l.ItemInputs) > 0 {
				l.blurActiveTab()
				l.ActiveTab = (l.ActiveTab + 1) % len(l.ItemInputs)
				l.focusActiveTab()
				return nil
			}

		case "<", "left":
			// Navigate to previous tab
			if l.IsExpanded && len(l.ItemInputs) > 0 {
				l.blurActiveTab()
				l.ActiveTab = (l.ActiveTab - 1 + len(l.ItemInputs)) % len(l.ItemInputs)
				l.focusActiveTab()
				return nil
			}

		case "+":
			// Add new array item
			if l.IsExpanded {
				l.addNewItem()
				return nil
			}

		case "-":
			// Remove current array item
			if l.IsExpanded && len(l.ItemInputs) > 1 {
				l.removeCurrentItem()
				return nil
			}
		}
	}

	// If expanded, forward to active tab input
	if l.IsExpanded && l.ActiveTab >= 0 && l.ActiveTab < len(l.ItemInputs) {
		*l.ItemInputs[l.ActiveTab], _ = l.ItemInputs[l.ActiveTab].Update(msg)

		// Update values array
		if l.ActiveTab < len(l.Values) {
			l.Values[l.ActiveTab] = l.ItemInputs[l.ActiveTab].Value()
		}

		return nil
	}

	// Fallback to simple text input mode
	*l.TextInput, _ = l.TextInput.Update(msg)

	// Parse comma-separated values
	value := l.TextInput.Value()
	if value == "" {
		l.Values = make([]interface{}, 0)
	} else {
		parts := strings.Split(value, ",")
		l.Values = make([]interface{}, len(parts))
		for i, part := range parts {
			l.Values[i] = strings.TrimSpace(part)
		}
	}

	return nil
}

// Helper methods for tab management

// focusActiveTab focuses the currently active tab
func (l *ComplexArrayInput) focusActiveTab() {
	if l.IsSimpleType {
		if l.ActiveTab >= 0 && l.ActiveTab < len(l.ItemInputs) {
			l.ItemInputs[l.ActiveTab].Focus()
		}
	} else {
		if l.ActiveTab >= 0 && l.ActiveTab < len(l.ItemForms) {
			// Focus first input in the sub-form
			if len(l.ItemForms[l.ActiveTab]) > 0 {
				l.ItemForms[l.ActiveTab][0].Focus()
			}
		}
	}
}

// blurActiveTab blurs the currently active tab
func (l *ComplexArrayInput) blurActiveTab() {
	if l.IsSimpleType {
		if l.ActiveTab >= 0 && l.ActiveTab < len(l.ItemInputs) {
			l.ItemInputs[l.ActiveTab].Blur()
		}
	} else {
		if l.ActiveTab >= 0 && l.ActiveTab < len(l.ItemForms) {
			// Blur all inputs in the sub-form
			for _, wrapper := range l.ItemForms[l.ActiveTab] {
				wrapper.Blur()
			}
		}
	}
}

// blurAllTabs blurs all tab inputs
func (l *ComplexArrayInput) blurAllTabs() {
	if l.IsSimpleType {
		for _, input := range l.ItemInputs {
			input.Blur()
		}
	} else {
		for _, subForm := range l.ItemForms {
			for _, wrapper := range subForm {
				wrapper.Blur()
			}
		}
	}
}

// addNewItem adds a new array item as a new tab
func (l *ComplexArrayInput) addNewItem() {
	if l.IsSimpleType {
		// Add simple text input
		newInput := textinput.New()
		newInput.CharLimit = 200
		newInput.Width = 30

		// Add to inputs and values
		l.ItemInputs = append(l.ItemInputs, &newInput)
		l.Values = append(l.Values, "")
	} else {
		// Add complex sub-form
		if l.ItemDef != nil {
			subForm, err := createSubFormFromDefinition(l.ItemDef)
			if err == nil {
				l.ItemForms = append(l.ItemForms, subForm)
				l.Values = append(l.Values, make(map[string]interface{}))
			}
		}
	}

	// Focus new tab
	l.blurActiveTab()
	if l.IsSimpleType {
		l.ActiveTab = len(l.ItemInputs) - 1
	} else {
		l.ActiveTab = len(l.ItemForms) - 1
	}
	l.focusActiveTab()
}

// removeCurrentItem removes the currently active array item
func (l *ComplexArrayInput) removeCurrentItem() {
	if l.IsSimpleType {
		if len(l.ItemInputs) <= 1 {
			return // Don't remove if only one item left
		}

		// Remove from inputs and values
		l.ItemInputs = append(l.ItemInputs[:l.ActiveTab], l.ItemInputs[l.ActiveTab+1:]...)
		l.Values = append(l.Values[:l.ActiveTab], l.Values[l.ActiveTab+1:]...)

		// Adjust active tab
		if l.ActiveTab >= len(l.ItemInputs) {
			l.ActiveTab = len(l.ItemInputs) - 1
		}
	} else {
		if len(l.ItemForms) <= 1 {
			return // Don't remove if only one item left
		}

		// Remove from forms and values
		l.ItemForms = append(l.ItemForms[:l.ActiveTab], l.ItemForms[l.ActiveTab+1:]...)
		l.Values = append(l.Values[:l.ActiveTab], l.Values[l.ActiveTab+1:]...)

		// Adjust active tab
		if l.ActiveTab >= len(l.ItemForms) {
			l.ActiveTab = len(l.ItemForms) - 1
		}
	}

	// Focus adjusted tab
	l.focusActiveTab()
}

// View renders the list input with tabs when expanded or hint when collapsed
func (l *ComplexArrayInput) View() string {
	if !l.IsExpanded {
		// Show collapsed view with hint
		hint := " (collapsed array)"
		dimStyle := lipgloss.NewStyle().Foreground(colors.Grey240)
		dimmedHint := dimStyle.Render(hint)

		// Convert interface{} values to strings for display
		var stringValues []string
		for _, val := range l.Values {
			if str, ok := val.(string); ok {
				stringValues = append(stringValues, str)
			} else {
				stringValues = append(stringValues, fmt.Sprintf("%v", val))
			}
		}

		return "[" + strings.Join(stringValues, ", ") + "]" + dimmedHint
	}

	// Render tab-based interface when expanded
	var renderedTabs []string

	// Tab styles similar to bubbletea example
	inactiveTabStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, true, false, true).
		BorderForeground(colors.Grey240).
		Padding(0, 1).
		Foreground(colors.LighterGrey)

	activeTabStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, true, false, true).
		BorderForeground(colors.DeepBlue). // Blue
		Padding(0, 1).
		Foreground(colors.WhiteTerm). // White
		Background(colors.DarkBlue)   // Dark blue

	// Render each tab
	tabCount := len(l.ItemInputs)
	if !l.IsSimpleType {
		tabCount = len(l.ItemForms)
	}

	for i := 0; i < tabCount; i++ {
		tabLabel := fmt.Sprintf("%d", i+1)
		var style lipgloss.Style

		if i == l.ActiveTab {
			style = activeTabStyle
		} else {
			style = inactiveTabStyle
		}

		renderedTabs = append(renderedTabs, style.Render(tabLabel))
	}

	// Join tabs horizontally
	tabRow := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	// Add controls hint with array type information
	controlsStyle := lipgloss.NewStyle().Foreground(colors.Grey240)

	// Get array type information
	var itemType string = "unknown"
	if l.ItemDef != nil {
		switch l.ItemDef.Type {
		case "string":
			itemType = "string"
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
			itemType = l.ItemDef.Type
		}
	} else {
		// Default to string for simple arrays without item definition
		itemType = "string"
	}

	arrayTypeStyle := lipgloss.NewStyle().Foreground(colors.Grey240)
	arrayTypeText := arrayTypeStyle.Render("[array[" + itemType + "]]")

	controls := controlsStyle.Render("  [+] add  [-] remove  [</>] navigate") + "  " + arrayTypeText

	// Content area for active tab
	var content string
	if l.ActiveTab >= 0 && l.ActiveTab < len(l.ItemInputs) && l.IsSimpleType {
		// Simple type: show text input
		content = l.ItemInputs[l.ActiveTab].View()
	} else if l.ActiveTab >= 0 && l.ActiveTab < len(l.ItemForms) && !l.IsSimpleType {
		// Complex type: render sub-form
		var formRows []string
		for _, wrapper := range l.ItemForms[l.ActiveTab] {
			// Render each input in the sub-form with proper styling
			labelText := wrapper.Label
			var typeText string
			if !wrapper.IsNestedInput() {
				switch wrapper.GetType() {
				case InputTypeText:
					// Check if text input has enum values
					if wrapper.TextInput != nil && len(wrapper.TextInput.GetOneOf()) > 0 {
						enumValues := strings.Join(wrapper.TextInput.GetOneOf(), ",")
						typeText = " [string|enum:" + enumValues + "]"
					} else {
						typeText = " [string]"
					}
				case InputTypeInt64:
					typeText = " [int]"
				case InputTypeFloat64:
					typeText = " [float]"
				case InputTypeBool:
					typeText = " [bool]"
				case InputTypeComplexArray:
					typeText = " [array]"
				default:
					panic(fmt.Sprintf("Unknown input type: %s", wrapper.GetType()))
				}
			}

			// Style the type text in gray
			typeStyle := lipgloss.NewStyle().Foreground(colors.Grey240)
			styledType := typeStyle.Render(typeText)

			var label string
			if wrapper.Required {
				redStar := lipgloss.NewStyle().Foreground(colors.Red).Render("*")
				yellowColon := lipgloss.NewStyle().Foreground(colors.Orange).Render(":")
				label = labelText + redStar + styledType + yellowColon
			} else {
				label = labelText + styledType + ":"
			}

			// Create label row
			labelRow := lipgloss.NewStyle().Foreground(colors.Orange).Render(label)
			formRows = append(formRows, labelRow)

			// Create input row
			inputView := wrapper.View()
			formRows = append(formRows, "  "+inputView)
		}
		content = strings.Join(formRows, "\n")
	}

	// Content box style
	contentStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, true, true).
		BorderForeground(colors.DeepBlue).
		Padding(0, 1).
		Width(40) // Fixed width for consistency

	contentBox := contentStyle.Render(content)

	// Combine tabs, content, and controls
	return tabRow + "\n" + contentBox + controls
}

// Focus sets focus to the list input
func (l *ComplexArrayInput) Focus() {
	l.Focused = true
	if l.IsExpanded {
		l.focusActiveTab()
	} else {
		l.TextInput.Focus()
	}
}

// Blur removes focus from the list input
func (l *ComplexArrayInput) Blur() {
	l.Focused = false
	l.IsExpanded = false // Collapse when losing focus
	l.blurAllTabs()
	l.TextInput.Blur()
}

// Value returns the current list value as a string
func (l *ComplexArrayInput) Value() string {
	return l.TextInput.Value()
}

// SetValue sets the list value from a string
func (l *ComplexArrayInput) SetValue(val string) {
	l.TextInput.SetValue(val)
	if val == "" {
		l.Values = []interface{}{}
		l.ItemInputs = []*textinput.Model{}
		// Add one empty input
		input := textinput.New()
		input.CharLimit = 200
		input.Width = 30
		l.ItemInputs = append(l.ItemInputs, &input)
		l.Values = append(l.Values, "")
		l.ActiveTab = 0
	} else {
		parts := strings.Split(val, ",")
		l.Values = make([]interface{}, len(parts))
		l.ItemInputs = make([]*textinput.Model, len(parts))

		for i, part := range parts {
			trimmedPart := strings.TrimSpace(part)
			l.Values[i] = trimmedPart

			// Create individual input for this value
			input := textinput.New()
			input.SetValue(trimmedPart)
			input.CharLimit = 200
			input.Width = 30
			l.ItemInputs[i] = &input
		}

		if len(l.ItemInputs) > 0 {
			l.ActiveTab = 0
		}
	}
}

// Validate validates the list input
func (l *ComplexArrayInput) Validate() error {
	// Lists are generally always valid, but you could add specific validation here
	return nil
}

// GetValues returns the parsed list values as strings
func (l *ComplexArrayInput) GetValues() []string {
	var stringValues []string
	for _, val := range l.Values {
		if str, ok := val.(string); ok {
			stringValues = append(stringValues, str)
		} else {
			stringValues = append(stringValues, fmt.Sprintf("%v", val))
		}
	}
	return stringValues
}

// TypedValue returns the slice of values (strings for simple types, complex objects for complex types)
func (l *ComplexArrayInput) TypedValue() interface{} {
	// Sync values from individual tab inputs first
	for i, input := range l.ItemInputs {
		if i < len(l.Values) {
			l.Values[i] = input.Value()
		}
	}

	if l.IsSimpleType {
		// Filter out empty values for simple types
		var nonEmptyValues []string
		for _, value := range l.Values {
			if str, ok := value.(string); ok {
				trimmed := strings.TrimSpace(str)
				if trimmed != "" {
					nonEmptyValues = append(nonEmptyValues, trimmed)
				}
			}
		}
		return nonEmptyValues
	} else {
		// For complex types, return the values as-is (objects/arrays)
		var nonEmptyValues []interface{}
		for _, value := range l.Values {
			if value != nil {
				nonEmptyValues = append(nonEmptyValues, value)
			}
		}
		return nonEmptyValues
	}
}

// IsDefaultValue returns true if this is the default value (empty list)
func (l *ComplexArrayInput) IsDefaultValue() bool {
	return len(l.Values) == 0 || (len(l.Values) == 1 && l.Values[0] == "")
}

// NestedInput represents a nested JSON structure input
type NestedInput struct {
	inputs  []InputWrapper
	focused bool
	current int // currently focused nested input
}

// NewNestedInput creates a new nested input with child inputs
func NewNestedInput(childInputs []InputWrapper) *NestedInput {
	return &NestedInput{
		inputs:  childInputs,
		focused: false,
		current: 0,
	}
}

// Update handles updates for nested input
func (n *NestedInput) Update(msg tea.Msg) tea.Cmd {
	if !n.focused || len(n.inputs) == 0 {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			if n.current < len(n.inputs)-1 {
				n.inputs[n.current].Blur()
				n.current++
				n.inputs[n.current].Focus()
			}
			return nil
		case "shift+tab", "up":
			if n.current > 0 {
				n.inputs[n.current].Blur()
				n.current--
				n.inputs[n.current].Focus()
			}
			return nil
		}
	}

	// Forward the message to the currently focused input
	return n.inputs[n.current].Update(msg)
}

// View renders the nested input
func (n *NestedInput) View() string {
	if len(n.inputs) == 0 {
		return "No nested inputs"
	}

	var parts []string
	parts = append(parts, "{\n")

	for i, input := range n.inputs {
		prefix := "  "
		if i == n.current && n.focused {
			prefix = "> "
		}
		parts = append(parts, fmt.Sprintf("%s%s: %s\n", prefix, input.GetLabel(), input.View()))
	}

	parts = append(parts, "}")
	return strings.Join(parts, "")
}

// Focus sets focus to the nested input
func (n *NestedInput) Focus() {
	n.focused = true
	if len(n.inputs) > 0 {
		n.inputs[n.current].Focus()
	}
}

// Blur removes focus from the nested input
func (n *NestedInput) Blur() {
	n.focused = false
	if len(n.inputs) > 0 {
		n.inputs[n.current].Blur()
	}
}

// Value returns the nested input as JSON string
func (n *NestedInput) Value() string {
	if len(n.inputs) == 0 {
		return "{}"
	}

	result := make(map[string]interface{})
	for _, input := range n.inputs {
		result[input.GetLabel()] = input.GetTypedValue()
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return "{}"
	}

	return string(jsonBytes)
}

// SetValue sets the nested input from a JSON string
func (n *NestedInput) SetValue(val string) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return
	}

	for _, input := range n.inputs {
		if value, exists := data[input.GetLabel()]; exists {
			input.SetValue(fmt.Sprintf("%v", value))
		}
	}
}

// Validate validates all nested inputs
func (n *NestedInput) Validate() error {
	for _, input := range n.inputs {
		if err := input.Validate(); err != nil {
			return fmt.Errorf("nested input '%s': %w", input.GetLabel(), err)
		}
	}
	return nil
}

// GetInputs returns the child inputs
func (n *NestedInput) GetInputs() []InputWrapper {
	return n.inputs
}

// TypedValue returns the nested structure as a map
func (n *NestedInput) TypedValue() interface{} {
	result := make(map[string]interface{})
	for _, input := range n.inputs {
		result[input.GetLabel()] = input.GetTypedValue()
	}
	return result
}

// IsDefaultValue returns true if all nested inputs have default values
func (n *NestedInput) IsDefaultValue() bool {
	for _, input := range n.inputs {
		if !input.IsDefaultValue() {
			return false
		}
	}
	return len(n.inputs) == 0
}

type Inputs []InputWrapper

func (i *Inputs) Append(input InputWrapper) {
	*i = append(*i, input)
}

// GetValues returns the current values of all inputs as a map
func (i *Inputs) GetValues() map[string]string {
	values := make(map[string]string)
	for _, wrapper := range *i {
		values[wrapper.GetLabel()] = wrapper.Value()
	}
	return values
}

// Validate validates all inputs in the collection
func (i *Inputs) Validate() error {
	var errors []string

	for _, wrapper := range *i {
		// Debug logging to see field status
		log.Debug("Validation check",
			zap.String("field", wrapper.GetLabel()),
			zap.String("type", wrapper.GetType().String()),
			zap.Bool("required", wrapper.IsRequired()),
			zap.Bool("is_default", wrapper.IsDefaultValue()),
			zap.String("value", wrapper.Value()))

		// Determine if we should validate this field
		shouldValidate := false
		if wrapper.IsRequired() {
			// For required boolean fields, false is a valid value, so don't validate
			if wrapper.GetType() == InputTypeBool {
				shouldValidate = false
			} else {
				// For all other required fields, always validate
				shouldValidate = true
			}
		} else {
			// For optional fields, only validate if they have non-default values
			shouldValidate = !wrapper.IsDefaultValue()
		}

		// Special case: always validate fields with validation constraints (like OneOf) if they have any value
		hasValue := !wrapper.IsDefaultValue()
		hasConstraints := false
		if wrapper.IsTextInput() && len(wrapper.TextInput.GetOneOf()) > 0 && hasValue {
			hasConstraints = true
		}
		if hasConstraints {
			shouldValidate = true
		}

		log.Debug("Validation decision",
			zap.String("field", wrapper.GetLabel()),
			zap.Bool("should_validate", shouldValidate),
			zap.Bool("has_value", hasValue),
			zap.Bool("has_constraints", hasConstraints))

		if !shouldValidate {
			continue // Skip validation for this field
		}

		// Validate the input itself (parsing, OneOf, etc.)
		if err := wrapper.Validate(); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s", wrapper.GetLabel(), err.Error()))
			continue
		}

		// Then check required field validation for non-nested fields
		if wrapper.IsRequired() && wrapper.GetType() != InputTypeNested && wrapper.GetType() != InputTypeBool {
			if wrapper.IsDefaultValue() {
				var defaultDesc string
				switch wrapper.GetType() {
				case InputTypeText:
					defaultDesc = "empty string"
				case InputTypeInt64:
					defaultDesc = "empty (required)"
				case InputTypeFloat64:
					defaultDesc = "empty (required)"
				case InputTypeComplexArray:
					defaultDesc = "empty list"
				case InputTypePrimitivesArray:
					defaultDesc = "empty array"
				default:
					defaultDesc = "default value"
				}
				errors = append(errors, fmt.Sprintf("%s: required field cannot be %s", wrapper.GetLabel(), defaultDesc))
			}
		}

		// Handle nested input validation
		if wrapper.GetType() == InputTypeNested && shouldValidate {
			if err := i.validateNestedChildren(wrapper); err != nil {
				errors = append(errors, err.Error())
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed:\n- %s", strings.Join(errors, "\n- "))
	}

	return nil
}

// validateNestedChildren validates children of a nested input with proper logic
func (i *Inputs) validateNestedChildren(wrapper InputWrapper) error {
	if !wrapper.IsNestedInput() || wrapper.NestedInput == nil {
		return nil
	}

	var errors []string
	nestedInputs := wrapper.NestedInput.GetInputs()

	// Check if the nested object is "dirty" (has any non-default values)
	// EXCLUDE boolean fields from dirty check - they can't reliably determine default state
	isNestedObjectDirty := false
	for _, childWrapper := range nestedInputs {
		// Skip boolean fields entirely - they don't count towards "dirty" state
		if childWrapper.GetType() == InputTypeBool {
			continue
		}
		if !childWrapper.IsDefaultValue() {
			isNestedObjectDirty = true
			break
		}
	}

	for _, childWrapper := range nestedInputs {
		// Apply the conditional validation logic to children
		shouldValidateChild := false
		if childWrapper.IsRequired() {
			// For required boolean fields, false is a valid value, so don't validate
			if childWrapper.GetType() == InputTypeBool {
				shouldValidateChild = false
			} else {
				// For required fields, only validate if the nested object is dirty
				shouldValidateChild = isNestedObjectDirty
			}
		} else {
			// For optional fields, only validate if they have non-default values
			shouldValidateChild = !childWrapper.IsDefaultValue()
		}

		// Special case: always validate fields with validation constraints (like OneOf) if they have any value
		hasValue := !childWrapper.IsDefaultValue()
		hasConstraints := false
		if childWrapper.IsTextInput() && len(childWrapper.TextInput.GetOneOf()) > 0 && hasValue {
			hasConstraints = true
		}
		if hasConstraints {
			shouldValidateChild = true
		}

		if !shouldValidateChild {
			continue // Skip validation for this child
		}

		// Validate the child input
		if err := childWrapper.Validate(); err != nil {
			errors = append(errors, fmt.Sprintf("%s: nested input '%s': %s", wrapper.GetLabel(), childWrapper.GetLabel(), err.Error()))
			continue
		}

		// Check required field validation for child
		if childWrapper.IsRequired() && childWrapper.GetType() != InputTypeNested && childWrapper.GetType() != InputTypeBool {
			if childWrapper.IsDefaultValue() {
				var defaultDesc string
				switch childWrapper.GetType() {
				case InputTypeText:
					defaultDesc = "empty string"
				case InputTypeInt64:
					defaultDesc = "integer value cannot be empty"
				case InputTypeFloat64:
					defaultDesc = "float value cannot be empty"
				case InputTypeComplexArray:
					defaultDesc = "empty list"
				case InputTypePrimitivesArray:
					defaultDesc = "empty array"
				default:
					defaultDesc = "default value"
				}
				errors = append(errors, fmt.Sprintf("%s: nested input '%s': %s", wrapper.GetLabel(), childWrapper.GetLabel(), defaultDesc))
			}
		}

		// Recursively validate nested children
		if childWrapper.GetType() == InputTypeNested && shouldValidateChild {
			if err := i.validateNestedChildren(childWrapper); err != nil {
				errors = append(errors, err.Error())
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "\n- "))
	}

	return nil
}

// Helper methods for convenient input creation with standard defaults
// CharLimit: 100, Width: 50, Prompt: always empty

// NewTextInput creates and appends a text input with standard defaults
func (i *Inputs) NewTextInput(label, placeholder string, required bool, initialValue string) {
	input := textinput.New()
	input.Placeholder = placeholder
	input.CharLimit = 100 // Standard default
	input.Width = 50      // Standard default
	input.Prompt = ""     // Always empty
	if initialValue != "" {
		input.SetValue(initialValue)
	}
	wrapper := NewTextInputWrapper(label, &input, required, initialValue)
	i.Append(wrapper)
}

// NewSecretTextInput creates and appends a password text input with EchoMode and standard defaults
func (i *Inputs) NewSecretTextInput(label, placeholder string, required bool, initialValue string) {
	input := textinput.New()
	input.Placeholder = placeholder
	input.CharLimit = 100                   // Standard default
	input.Width = 50                        // Standard default
	input.Prompt = ""                       // Always empty
	input.EchoMode = textinput.EchoPassword // Secret mode for passwords
	if initialValue != "" {
		input.SetValue(initialValue)
	}
	wrapper := NewTextInputWrapper(label, &input, required, initialValue)
	i.Append(wrapper)
}

// NewBoolInput creates and appends a boolean input with standard defaults
func (i *Inputs) NewBoolInput(label, description string, required bool, defaultValue bool) {
	input := NewBoolInput(defaultValue, description)
	initialValue := ""
	if defaultValue {
		initialValue = "true"
	} else {
		initialValue = "false"
	}
	wrapper := NewBoolInputWrapper(label, input, required, initialValue)
	i.Append(wrapper)
}

// NewInt64Input creates and appends an int64 input with standard defaults
func (i *Inputs) NewInt64Input(label, placeholder string, required bool, defaultValue int64) {
	input := NewInt64Input("") // Create with empty initial value
	input.TextInput.Placeholder = placeholder
	input.TextInput.CharLimit = 100 // Standard default
	input.TextInput.Width = 50      // Standard default
	input.TextInput.Prompt = ""     // Always empty

	initialValue := ""
	if defaultValue != 0 {
		initialValue = fmt.Sprintf("%d", defaultValue)
		input.TextInput.SetValue(initialValue)
	}
	wrapper := NewInt64InputWrapper(label, input, required, initialValue)
	i.Append(wrapper)
}

// InputWrapper represents any type of input with enhanced type system
type InputWrapper struct {
	TextInput            *TextInputWrapper
	BoolInput            *BoolInput
	Int64Input           *Int64Input
	Float64Input         *Float64Input
	ComplexArrayInput    *ComplexArrayInput
	PrimitivesArrayInput *PrimitivesArrayInput
	NestedInput          *NestedInput
	Label                string
	Required             bool
	InputType            InputType
	InitialValue         string // Store the initial/default value for dirty tracking
	Touched              bool   // Track if user has modified this field
}

// Constructor functions for each input type

// NewTextInputWrapper creates a wrapper for text input
func NewTextInputWrapper(label string, input *textinput.Model, required bool, initialValue string) InputWrapper {
	// If no initial value provided, use current input value
	if initialValue == "" {
		initialValue = input.Value()
	}
	return InputWrapper{
		TextInput:    NewTextInputWrapperFromModel(input),
		Label:        label,
		Required:     required,
		InputType:    InputTypeText,
		InitialValue: initialValue,
	}
}

// NewTextInputWrapperArray creates a text input wrapper specifically for arrays
func NewTextInputWrapperArray(label string, textInput *textinput.Model, required bool, arrayType string, initialValue string) InputWrapper {
	wrapper := NewTextInputWrapperFromModel(textInput)
	wrapper.arrayType = arrayType

	// If no initial value provided, use current input value
	if initialValue == "" {
		initialValue = textInput.Value()
	}

	return InputWrapper{
		TextInput:    wrapper,
		Label:        label,
		Required:     required,
		InputType:    InputTypeText,
		InitialValue: initialValue,
	}
}

// NewTextInputWrapperEnum creates a wrapper for text input with OneOf validation
func NewTextInputWrapperEnum(label string, input *textinput.Model, required bool, oneOf []string, initialValue string) InputWrapper {
	// If no initial value provided, use current input value
	if initialValue == "" {
		initialValue = input.Value()
	}
	return InputWrapper{
		TextInput:    NewTextInputWrapperWithOneOf(input, oneOf),
		Label:        label,
		Required:     required,
		InputType:    InputTypeText,
		InitialValue: initialValue,
	}
}

// NewBoolInputWrapper creates a wrapper for boolean input
func NewBoolInputWrapper(label string, input *BoolInput, required bool, initialValue string) InputWrapper {
	// If no initial value provided, use current input value
	if initialValue == "" {
		initialValue = input.Value()
	}
	return InputWrapper{
		BoolInput:    input,
		Label:        label,
		Required:     required,
		InputType:    InputTypeBool,
		InitialValue: initialValue,
	}
}

// NewInt64InputWrapper creates a wrapper for int64 input
func NewInt64InputWrapper(label string, input *Int64Input, required bool, initialValue string) InputWrapper {
	// If no initial value provided, use current input value
	if initialValue == "" {
		initialValue = input.Value()
	}
	return InputWrapper{
		Int64Input:   input,
		Label:        label,
		Required:     required,
		InputType:    InputTypeInt64,
		InitialValue: initialValue,
	}
}

// NewFloat64InputWrapper creates a wrapper for float64 input
func NewFloat64InputWrapper(label string, input *Float64Input, required bool, initialValue string) InputWrapper {
	// If no initial value provided, use current input value
	if initialValue == "" {
		initialValue = input.Value()
	}
	return InputWrapper{
		Float64Input: input,
		Label:        label,
		Required:     required,
		InputType:    InputTypeFloat64,
		InitialValue: initialValue,
	}
}

// NewComplexArrayInputWrapper creates a wrapper for complex array input
func NewComplexArrayInputWrapper(label string, input *ComplexArrayInput, required bool, initialValue string) InputWrapper {
	// If no initial value provided, use current input value
	if initialValue == "" {
		initialValue = input.Value()
	}
	return InputWrapper{
		ComplexArrayInput: input,
		Label:             label,
		Required:          required,
		InputType:         InputTypeComplexArray,
		InitialValue:      initialValue,
	}
}

// NewPrimitivesArrayInputWrapper creates a wrapper for primitives array input
func NewPrimitivesArrayInputWrapper(label string, input *PrimitivesArrayInput, required bool, initialValue string) InputWrapper {
	// If no initial value provided, use current input value
	if initialValue == "" {
		initialValue = input.Value()
	}
	return InputWrapper{
		PrimitivesArrayInput: input,
		Label:                label,
		Required:             required,
		InputType:            InputTypePrimitivesArray,
		InitialValue:         initialValue,
	}
}

// NewNestedInputWrapper creates a wrapper for nested input
func NewNestedInputWrapper(label string, input *NestedInput, required bool, initialValue string) InputWrapper {
	// If no initial value provided, use current input value
	if initialValue == "" {
		initialValue = input.Value()
	}
	return InputWrapper{
		NestedInput:  input,
		Label:        label,
		Required:     required,
		InputType:    InputTypeNested,
		InitialValue: initialValue,
	}
}

// GetInput returns the appropriate input interface
func (iw *InputWrapper) GetInput() InputField {
	switch iw.InputType {
	case InputTypeText:
		return iw.TextInput
	case InputTypeBool:
		return iw.BoolInput
	case InputTypeInt64:
		return iw.Int64Input
	case InputTypeFloat64:
		return iw.Float64Input
	case InputTypeComplexArray:
		return iw.ComplexArrayInput
	case InputTypePrimitivesArray:
		return iw.PrimitivesArrayInput
	case InputTypeNested:
		return iw.NestedInput
	default:
		return nil
	}
}

// Update forwards the update to the appropriate input
func (iw *InputWrapper) Update(msg tea.Msg) tea.Cmd {
	if input := iw.GetInput(); input != nil {
		// Mark as touched when receiving user input
		if _, ok := msg.(tea.KeyMsg); ok {
			iw.Touched = true
		}
		return input.Update(msg)
	}
	return nil
}

// View renders the appropriate input
func (iw *InputWrapper) View() string {
	if input := iw.GetInput(); input != nil {
		return input.View()
	}
	return ""
}

// Focus sets focus to the input
func (iw *InputWrapper) Focus() {
	if input := iw.GetInput(); input != nil {
		input.Focus()
	}
}

// Blur removes focus from the input
func (iw *InputWrapper) Blur() {
	if input := iw.GetInput(); input != nil {
		input.Blur()
	}
}

// Value returns the current value as a string
func (iw *InputWrapper) Value() string {
	if input := iw.GetInput(); input != nil {
		return input.Value()
	}
	return ""
}

// SetValue sets the value from a string
func (iw *InputWrapper) SetValue(val string) {
	if input := iw.GetInput(); input != nil {
		input.SetValue(val)
	}
}

// Validate validates the input
func (iw *InputWrapper) Validate() error {
	if !iw.IsRequired() && !iw.IsDirty() {
		// value hasn't changed, no need to validate
		return nil
	}

	if input := iw.GetInput(); input != nil {
		return input.Validate()
	}
	return nil
}

// GetLabel returns the input label
func (iw *InputWrapper) GetLabel() string {
	return iw.Label
}

// GetType returns the input type
func (iw *InputWrapper) GetType() InputType {
	return iw.InputType
}

// IsRequired returns whether the input is required
func (iw *InputWrapper) IsRequired() bool {
	return iw.Required
}

// GetTypedValue returns the value in its appropriate Go type
func (iw *InputWrapper) GetTypedValue() interface{} {
	switch iw.InputType {
	case InputTypeText:
		return iw.TextInput.TypedValue()
	case InputTypeBool:
		return iw.BoolInput.TypedValue()
	case InputTypeInt64:
		if iw.Int64Input != nil {
			return iw.Int64Input.TypedValue()
		}
		return int64(0)
	case InputTypeFloat64:
		if iw.Float64Input != nil {
			return iw.Float64Input.TypedValue()
		}
		return float64(0)
	case InputTypeComplexArray:
		if iw.ComplexArrayInput != nil {
			return iw.ComplexArrayInput.TypedValue()
		}
		return []string{}
	case InputTypePrimitivesArray:
		if iw.PrimitivesArrayInput != nil {
			return iw.PrimitivesArrayInput.TypedValue()
		}
		return []string{}
	case InputTypeNested:
		if iw.NestedInput != nil {
			return iw.NestedInput.TypedValue()
		}
		return make(map[string]interface{})
	default:
		return iw.Value()
	}
}

// IsNestedInput returns true if this is a nested input
func (iw *InputWrapper) IsNestedInput() bool {
	return iw.InputType == InputTypeNested
}

// Type checking helper methods for rendering
func (iw *InputWrapper) IsTextInput() bool {
	return iw.InputType == InputTypeText
}

func (iw *InputWrapper) IsBoolInput() bool {
	return iw.InputType == InputTypeBool
}

func (iw *InputWrapper) IsInt64Input() bool {
	return iw.InputType == InputTypeInt64
}

func (iw *InputWrapper) IsFloat64Input() bool {
	return iw.InputType == InputTypeFloat64
}

func (iw *InputWrapper) IsComplexArrayInput() bool {
	return iw.InputType == InputTypeComplexArray
}

func (iw *InputWrapper) IsPrimitivesArrayInput() bool {
	return iw.InputType == InputTypePrimitivesArray
}

// IsDefaultValue returns true if the input has its default value
func (iw *InputWrapper) IsDefaultValue() bool {
	switch iw.InputType {
	case InputTypeText:
		return iw.TextInput.IsDefaultValue()
	case InputTypeBool:
		return iw.BoolInput.IsDefaultValue()
	case InputTypeInt64:
		return iw.Int64Input.IsDefaultValue()
	case InputTypeFloat64:
		return iw.Float64Input.IsDefaultValue()
	case InputTypeComplexArray:
		return iw.ComplexArrayInput.IsDefaultValue()
	case InputTypePrimitivesArray:
		return iw.PrimitivesArrayInput.IsDefaultValue()
	case InputTypeNested:
		return iw.NestedInput.IsDefaultValue()
	default:
		return true
	}
}

// GetInitialValue returns the initial/default value of the input
func (iw *InputWrapper) GetInitialValue() string {
	return iw.InitialValue
}

// SetInitialValue sets the initial/default value of the input
func (iw *InputWrapper) SetInitialValue(value string) {
	iw.InitialValue = value
}

// IsDirty returns true if the current value differs from the initial value
func (iw *InputWrapper) IsDirty() bool {
	return iw.Value() != iw.InitialValue
}

// SetSuggestions sets suggestions for supported input types (text and int64)
// Panics if called on unsupported input types
func (iw *InputWrapper) SetSuggestions(suggestions []string) {
	if len(suggestions) == 0 {
		return
	}

	if iw.IsTextInput() {
		if iw.TextInput != nil && iw.TextInput.TextInput != nil {
			iw.TextInput.TextInput.SetSuggestions(suggestions)
			iw.TextInput.TextInput.ShowSuggestions = true
		}
	} else if iw.IsInt64Input() {
		if iw.Int64Input != nil && iw.Int64Input.TextInput != nil {
			iw.Int64Input.TextInput.SetSuggestions(suggestions)
			iw.Int64Input.TextInput.ShowSuggestions = true
		}
	} else {
		panic(fmt.Sprintf("SetSuggestions is not supported for input type: %v", iw.InputType))
	}
}

// Field returns an InputWrapper by label for chaining
func (i *Inputs) Field(label string) *InputWrapper {
	for _, wrapper := range *i {
		if wrapper.GetLabel() == label {
			return &wrapper
		}
	}
	return nil // Return nil if not found
}

// MustField returns an InputWrapper by label, panics if not found
func (i *Inputs) MustField(label string) *InputWrapper {
	for _, wrapper := range *i {
		if wrapper.GetLabel() == label {
			return &wrapper
		}
	}
	panic(fmt.Sprintf("InputWrapper with label '%s' not found", label))
}

// RemoveField removes an InputWrapper by label, returns true if found and removed
func (i *Inputs) RemoveField(label string) bool {
	for index, wrapper := range *i {
		if wrapper.GetLabel() == label {
			// Remove the element at index
			*i = append((*i)[:index], (*i)[index+1:]...)
			return true
		}
	}
	return false // Field not found
}

// Typed value access methods for InputWrapper

// String returns the value as a string
func (iw *InputWrapper) String() string {
	// For string types, use GetTypedValue, otherwise fallback to Value()
	if iw.GetType() == InputTypeText {
		if typed := iw.GetTypedValue(); typed != nil {
			if str, ok := typed.(string); ok {
				return str
			}
		}
	}
	return iw.Value()
}

// Bool returns the value as a boolean
func (iw *InputWrapper) Bool() bool {
	// First try GetTypedValue for bool inputs
	if iw.GetType() == InputTypeBool {
		if typed := iw.GetTypedValue(); typed != nil {
			if b, ok := typed.(bool); ok {
				return b
			}
		}
	}

	// Try to parse as boolean from string
	val := strings.ToLower(iw.Value())
	return val == "true" || val == "yes" || val == "y" || val == "1"
}

// Int64 returns the value as an int64
func (iw *InputWrapper) Int64() int64 {
	// First try GetTypedValue for int64 inputs
	if iw.GetType() == InputTypeInt64 {
		if typed := iw.GetTypedValue(); typed != nil {
			if i, ok := typed.(int64); ok {
				return i
			}
		}
	}

	// Try to parse from string
	if val, err := strconv.ParseInt(iw.Value(), 10, 64); err == nil {
		return val
	}
	return 0
}

// Float64 returns the value as a float64
func (iw *InputWrapper) Float64() float64 {
	// First try GetTypedValue for float64 inputs
	if iw.GetType() == InputTypeFloat64 {
		if typed := iw.GetTypedValue(); typed != nil {
			if f, ok := typed.(float64); ok {
				return f
			}
		}
	}

	// Try to parse from string
	if val, err := strconv.ParseFloat(iw.Value(), 64); err == nil {
		return val
	}
	return 0.0
}

// List returns the value as a slice of strings (comma-separated)
func (iw *InputWrapper) List() []string {
	// First try GetTypedValue for array inputs
	if iw.GetType() == InputTypeComplexArray || iw.GetType() == InputTypePrimitivesArray {
		if typed := iw.GetTypedValue(); typed != nil {
			if list, ok := typed.([]string); ok {
				return list
			}
		}
	}

	// Parse from string as comma-separated values
	value := strings.TrimSpace(iw.Value())
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, len(parts))
	for i, part := range parts {
		result[i] = strings.TrimSpace(part)
	}
	return result
}

// createInputFromDefinition converts a single InputDefinition to an InputWrapper
func createInputFromDefinition(def InputDefinition, onlyPrimitives bool) (InputWrapper, error) {
	switch def.Type {
	case "string":
		return createStringInput(def), nil
	case "integer":
		return createIntegerInput(def), nil
	case "number":
		return createNumberInput(def), nil
	case "boolean":
		return createBooleanInput(def), nil
	case "array":
		if onlyPrimitives {
			if def.Required {
				panic(fmt.Sprintf("cannot skip required array field '%s' when onlyPrimitives is true", def.Name))
			}
			return InputWrapper{}, SkipDefinition{defName: def.Name, reason: "Only primitive arrays allowed"}
		}

		// Debug logging to understand what's happening
		log.Debug("Processing array field", zap.String("field_name", def.Name))
		if def.Items != nil {
			log.Debug("Array items details",
				zap.String("field_name", def.Name),
				zap.String("items_type", def.Items.Type),
				zap.Bool("is_complex", def.Items.Type == "object" || def.Items.Type == "array"))
		} else {
			log.Debug("Array items is nil", zap.String("field_name", def.Name))
		}

		// Check if this is a primitive array or complex array
		if def.Items != nil && (def.Items.Type == "object" || def.Items.Type == "array") {
			log.Debug("Skipping array of objects/arrays", zap.String("field_name", def.Name))
			// Skip arrays of objects or nested arrays entirely

			if def.Required {
				log.Warn("Skipping required array field (not supported yet)",
					zap.String("field_name", def.Name),
					zap.String("reason", "Arrays of objects or nested arrays are not supported"))
			}

			return InputWrapper{}, SkipDefinition{
				defName: def.Name,
				reason:  "Arrays of objects or nested arrays are not supported (temporary)",
			}
		} else {
			log.Debug("Creating primitive array input", zap.String("field_name", def.Name))
			// Primitive array (strings, ints, bools, etc.) - use ArrayInput for simple inline editing

			// Determine array type
			var arrayType string = "array[string]"
			if def.Items != nil {
				switch def.Items.Type {
				case "string":
					arrayType = "array[string]"
				case "integer":
					arrayType = "array[int]"
				case "number":
					arrayType = "array[float]"
				case "boolean":
					arrayType = "array[bool]"
				default:
					arrayType = "array[string]" // Default fallback
				}
			}

			// Get default values
			var defaultValues []string
			if def.Default != nil {
				if defaultArray, ok := def.Default.([]interface{}); ok {
					for _, item := range defaultArray {
						defaultValues = append(defaultValues, fmt.Sprintf("%v", item))
					}
				}
			}

			// Determine placeholder text
			var placeholder string
			if def.Placeholder != "" {
				placeholder = def.Placeholder
			} else if def.Description != "" {
				placeholder = def.Description
			} else {
				typ := strings.TrimPrefix(arrayType, "array[")
				typ = strings.TrimSuffix(typ, "]")
				placeholder = fmt.Sprintf("Enter %s values separated by commas", typ)
			}

			// Create PrimitivesArrayInput with type information and placeholder
			arrayInput := NewPrimitivesArrayInputWithTypePlaceholder(defaultValues, arrayType, placeholder)
			log.Debug("Successfully created PrimitivesArrayInput",
				zap.String("field_name", def.Name),
				zap.String("array_type", arrayType),
				zap.Int("default_values_count", len(defaultValues)))

			// Use the joined default values as initial value
			initialValue := strings.Join(defaultValues, ", ")
			return NewPrimitivesArrayInputWrapper(def.Name, arrayInput, def.Required, initialValue), nil
		}
	case "object":
		if onlyPrimitives {
			if def.Required {
				panic(fmt.Sprintf("cannot skip required object field '%s' when onlyPrimitives is true", def.Name))
			}
			return InputWrapper{}, SkipDefinition{
				defName: def.Name,
				reason:  "Only primitive objects allowed"}
		}
		// Create nested input with proper object structure and recursion
		return createObjectInput(def), nil
	default:
		panic(fmt.Sprintf("unsupported input type '%s' for field '%s'", def.Type, def.Name))
	}
}

// createStringInput creates a text input wrapper for string properties
func createStringInput(def InputDefinition) InputWrapper {
	textInput := textinput.New()

	// Set placeholder with priority: custom placeholder > description > field name
	var placeholder string
	if def.Placeholder != "" {
		placeholder = def.Placeholder
	} else if def.Description != "" {
		placeholder = def.Description
	} else {
		placeholder = def.Name
	}

	// Truncate placeholder to 64 characters
	if len(placeholder) > 64 {
		placeholder = placeholder[:61] + "..."
	}

	textInput.Placeholder = placeholder

	// Don't set default values - keep fields empty with placeholder visible

	// Configure input based on format
	switch def.Format {
	case "password":
		textInput.EchoMode = textinput.EchoPassword
		textInput.CharLimit = 200
	case "email":
		textInput.CharLimit = 100
	case "uri", "url":
		textInput.CharLimit = 500
	default:
		textInput.CharLimit = 255 // Default limit
	}

	// Auto-size width based on expected content
	if textInput.CharLimit <= 30 {
		textInput.Width = 25
	} else if textInput.CharLimit <= 100 {
		textInput.Width = 40
	} else {
		textInput.Width = 60
	}

	// Get initial value from definition default or current input value
	initialValue := ""
	if def.Default != nil {
		initialValue = fmt.Sprintf("%v", def.Default)
	} else {
		initialValue = textInput.Value()
	}

	// Handle enum validation
	if len(def.Enum) > 0 {
		return NewTextInputWrapperEnum(def.Name, &textInput, def.Required, def.Enum, initialValue)
	}

	return NewTextInputWrapper(def.Name, &textInput, def.Required, initialValue)
}

// createIntegerInput creates an int64 input wrapper for integer properties
func createIntegerInput(def InputDefinition) InputWrapper {
	// Don't set default values - use 0 internally but don't display it
	int64Input := NewInt64Input("")

	// Clear the display so no default shows
	int64Input.SetValue("")

	// Set placeholder with priority: custom placeholder > description > field name
	var placeholder string
	if def.Placeholder != "" {
		placeholder = def.Placeholder
	} else if def.Description != "" {
		placeholder = def.Description
	} else {
		placeholder = def.Name
	}

	// Truncate placeholder to 64 characters
	if len(placeholder) > 64 {
		placeholder = placeholder[:61] + "..."
	}

	int64Input.TextInput.Placeholder = placeholder

	// TODO: Add constraint validation when SetMinimum/SetMaximum methods are implemented
	// For now, constraints are not enforced for integer inputs

	// Get initial value from definition default or current input value
	initialValue := ""
	if def.Default != nil {
		initialValue = fmt.Sprintf("%v", def.Default)
	} else {
		initialValue = int64Input.Value()
	}

	return NewInt64InputWrapper(def.Name, int64Input, def.Required, initialValue)
}

// createNumberInput creates a float64 input wrapper for number properties
func createNumberInput(def InputDefinition) InputWrapper {
	// Don't set default values - use 0.0 internally but don't display it
	float64Input := NewFloat64Input("")

	// Clear the display so no default shows
	float64Input.SetValue("")

	// Set placeholder with priority: custom placeholder > description > field name
	var placeholder string
	if def.Placeholder != "" {
		placeholder = def.Placeholder
	} else if def.Description != "" {
		placeholder = def.Description
	} else {
		placeholder = def.Name
	}

	// Truncate placeholder to 64 characters
	if len(placeholder) > 64 {
		placeholder = placeholder[:61] + "..."
	}

	float64Input.TextInput.Placeholder = placeholder

	// TODO: Add constraint validation when SetMinimum/SetMaximum methods are implemented
	// For now, constraints are not enforced for float inputs

	// Get initial value from definition default or current input value
	initialValue := ""
	if def.Default != nil {
		initialValue = fmt.Sprintf("%v", def.Default)
	} else {
		initialValue = float64Input.Value()
	}

	return NewFloat64InputWrapper(def.Name, float64Input, def.Required, initialValue)
}

// createBooleanInput creates a boolean input wrapper for boolean properties
func createBooleanInput(def InputDefinition) InputWrapper {
	// Truncate description to 64 characters
	description := def.Description
	if len(description) > 64 {
		description = description[:61] + "..."
	}

	// Don't set default values - use false internally but clear display
	boolInput := NewBoolInput(false, description)
	boolInput.SetValue("")

	// Get initial value from definition default or current input value
	initialValue := ""
	if def.Default != nil {
		initialValue = fmt.Sprintf("%v", def.Default)
	} else {
		initialValue = boolInput.Value()
	}

	return NewBoolInputWrapper(def.Name, boolInput, def.Required, initialValue)
}

// createObjectInput creates a nested input for object properties with recursion
func createObjectInput(def InputDefinition) InputWrapper {
	// If the object has no properties, this is an ambiguous object that should have been skipped
	// This should not happen if the OpenAPI parsing is working correctly, but we handle it gracefully
	if len(def.Properties) == 0 {
		// Create a simple text input as fallback for edge cases
		textInput := textinput.New()
		textInput.Placeholder = "No properties defined"
		textInput.CharLimit = 100
		textInput.Width = 30

		// Use empty string as initial value for fallback case
		return NewTextInputWrapper(def.Name, &textInput, def.Required, "")
	}

	// Create nested inputs for each property
	var childInputs []InputWrapper

	// Sort properties for consistent ordering (required first, then alphabetical)
	var propNames []string
	for propName := range def.Properties {
		propNames = append(propNames, propName)
	}

	// Sort: required fields first, then alphabetical
	sort.Slice(propNames, func(i, j int) bool {
		propI := def.Properties[propNames[i]]
		propJ := def.Properties[propNames[j]]

		// Required fields come first
		if propI.Required && !propJ.Required {
			return true
		}
		if !propI.Required && propJ.Required {
			return false
		}

		// Then alphabetical
		return propNames[i] < propNames[j]
	})

	// Create child inputs recursively
	for _, propName := range propNames {
		propDef := def.Properties[propName]
		if propDef == nil {
			continue
		}

		childInput, err := createInputFromDefinition(*propDef, false)
		if err != nil {
			// Skip properties that can't be created
			continue
		}

		childInputs = append(childInputs, childInput)
	}

	// Create the nested input
	nestedInput := NewNestedInput(childInputs)
	// Get initial value from definition default or current input value
	initialValue := ""
	if def.Default != nil {
		initialValue = fmt.Sprintf("%v", def.Default)
	} else {
		initialValue = nestedInput.Value()
	}
	return NewNestedInputWrapper(def.Name, nestedInput, def.Required, initialValue)
}

// createSubFormFromDefinition creates a new ListInput with item definition for complex types
func createSubFormFromDefinition(itemDef *InputDefinition) ([]InputWrapper, error) {
	// If the item definition is nil or a simple type, create a simple ListInput
	if itemDef == nil || (itemDef.Type == "string" || itemDef.Type == "integer" || itemDef.Type == "number" || itemDef.Type == "boolean") {
		var defaultValues []string
		if itemDef != nil && itemDef.Default != nil {
			if defaultArray, ok := itemDef.Default.([]interface{}); ok {
				for _, item := range defaultArray {
					if str, ok := item.(string); ok {
						defaultValues = append(defaultValues, str)
					} else {
						defaultValues = append(defaultValues, fmt.Sprintf("%v", item))
					}
				}
			}
		}
		complexArrayInput := NewComplexArrayInputWithItemDef(defaultValues, itemDef)
		// Use the joined default values as initial value
		initialValue := strings.Join(defaultValues, ", ")
		return []InputWrapper{NewComplexArrayInputWrapper(itemDef.Name, complexArrayInput, itemDef.Required, initialValue)}, nil
	}

	// For complex types (objects/arrays), create a new NestedInput
	var childInputs []InputWrapper

	// Check for ambiguous objects (objects with no properties) - these should be skipped
	if itemDef.Type == "object" && len(itemDef.Properties) == 0 {
		// This is an ambiguous object that should have been filtered out earlier
		// Return an error to indicate this should be skipped
		return nil, fmt.Errorf("ambiguous object with no properties should be skipped: %s", itemDef.Name)
	}

	// Sort properties for consistent ordering (required first, then alphabetical)
	var propNames []string
	for propName := range itemDef.Properties {
		propNames = append(propNames, propName)
	}

	// Sort: required fields first, then alphabetical
	sort.Slice(propNames, func(i, j int) bool {
		propI := itemDef.Properties[propNames[i]]
		propJ := itemDef.Properties[propNames[j]]

		// Required fields come first
		if propI.Required && !propJ.Required {
			return true
		}
		if !propI.Required && propJ.Required {
			return false
		}

		// Then alphabetical
		return propNames[i] < propNames[j]
	})

	// Create child inputs recursively
	for _, propName := range propNames {
		propDef := itemDef.Properties[propName]
		if propDef == nil {
			continue
		}

		childInput, err := createInputFromDefinition(*propDef, false)
		if err != nil {
			// Skip properties that can't be created
			continue
		}

		childInputs = append(childInputs, childInput)
	}

	// Create the nested input
	nestedInput := NewNestedInput(childInputs)
	// Get initial value from definition default or current input value
	initialValue := ""
	if itemDef.Default != nil {
		initialValue = fmt.Sprintf("%v", itemDef.Default)
	} else {
		initialValue = nestedInput.Value()
	}
	return []InputWrapper{NewNestedInputWrapper(itemDef.Name, nestedInput, itemDef.Required, initialValue)}, nil
}

// Ensure Inputs implements sort.Interface

func (i Inputs) Len() int {
	return len(i)
}

func (i Inputs) Swap(a, b int) {
	i[a], i[b] = i[b], i[a]
}

func (i Inputs) Less(a, b int) bool {
	// Required fields come first
	if i[a].IsRequired() && !i[b].IsRequired() {
		return true
	}
	if !i[a].IsRequired() && i[b].IsRequired() {
		return false
	}

	return i[a].GetLabel() < i[b].GetLabel()
}

// ToParams converts the Inputs to vast_client.Params (map[string]any)
// Only includes dirty fields (where current value differs from initial value) and non-empty values
// Handles nested structures and proper slice type conversion
func (i *Inputs) ToParams() map[string]any {
	params := make(map[string]any)

	for _, wrapper := range *i {
		// Skip if field is not dirty (current value equals initial value)
		if !wrapper.IsDirty() && !wrapper.IsRequired() {
			continue
		}

		// Skip empty optional fields
		if wrapper.IsDefaultValue() && !wrapper.IsRequired() {
			continue
		}

		fieldName := wrapper.GetLabel()
		value := wrapper.convertToTypedValue()

		// Only add non-nil values
		if value != nil {
			params[fieldName] = value
		}
	}

	return params
}

// convertToTypedValue converts the InputWrapper value to the appropriate Go type
// with proper handling of nested structures and typed arrays
func (iw *InputWrapper) convertToTypedValue() interface{} {
	switch iw.InputType {
	case InputTypeText:
		val := iw.String()
		if val == "" {
			return nil
		}
		return val

	case InputTypeBool:
		return iw.Bool()

	case InputTypeInt64:
		if iw.IsDefaultValue() && !iw.IsRequired() {
			return nil
		}
		return iw.Int64()

	case InputTypeFloat64:
		if iw.IsDefaultValue() && !iw.IsRequired() {
			return nil
		}
		return iw.Float64()

	case InputTypePrimitivesArray:
		if iw.PrimitivesArrayInput == nil {
			return nil
		}

		// Get the array values
		values := iw.PrimitivesArrayInput.getAllValues()
		if len(values) == 0 {
			return nil
		}

		// Convert based on array type
		arrayType := iw.PrimitivesArrayInput.arrayType
		switch arrayType {
		case "array[int]", "array[int64]":
			result := make([]int64, 0, len(values))
			for _, val := range values {
				if parsed, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64); err == nil {
					result = append(result, parsed)
				}
			}
			return result

		case "array[float]", "array[float64]":
			result := make([]float64, 0, len(values))
			for _, val := range values {
				if parsed, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
					result = append(result, parsed)
				}
			}
			return result

		case "array[bool]":
			result := make([]bool, 0, len(values))
			for _, val := range values {
				trimmed := strings.ToLower(strings.TrimSpace(val))
				result = append(result, trimmed == "true" || trimmed == "yes" || trimmed == "y" || trimmed == "1")
			}
			return result

		default: // "array[string]" or unspecified
			// Filter out empty strings
			result := make([]string, 0, len(values))
			for _, val := range values {
				if trimmed := strings.TrimSpace(val); trimmed != "" {
					result = append(result, trimmed)
				}
			}
			return result
		}

	case InputTypeComplexArray:
		if iw.ComplexArrayInput == nil {
			return nil
		}

		// Get typed values and filter out empty ones
		if typedValues := iw.ComplexArrayInput.TypedValue(); typedValues != nil {
			if slice, ok := typedValues.([]string); ok {
				// Filter out empty strings
				result := make([]string, 0, len(slice))
				for _, val := range slice {
					if trimmed := strings.TrimSpace(val); trimmed != "" {
						result = append(result, trimmed)
					}
				}
				if len(result) == 0 {
					return nil
				}
				return result
			}
			if slice, ok := typedValues.([]interface{}); ok {
				// For complex objects, convert each one
				result := make([]map[string]interface{}, 0, len(slice))
				for _, val := range slice {
					if converted := convertComplexValue(val); converted != nil {
						result = append(result, converted)
					}
				}
				if len(result) == 0 {
					return nil
				}
				return result
			}
		}
		return nil

	case InputTypeNested:
		if iw.NestedInput == nil {
			return nil
		}

		// Recursively convert nested inputs
		nestedParams := make(map[string]interface{})
		hasValues := false

		for _, childWrapper := range iw.NestedInput.GetInputs() {
			// Only include dirty or required fields in nested structures
			if childWrapper.IsDirty() || (childWrapper.IsRequired() && !childWrapper.IsDefaultValue()) {
				if value := childWrapper.convertToTypedValue(); value != nil {
					nestedParams[childWrapper.GetLabel()] = value
					hasValues = true
				}
			}
		}

		if !hasValues {
			return nil
		}
		return nestedParams

	default:
		// Fallback to string value
		val := iw.String()
		if val == "" {
			return nil
		}
		return val
	}
}

// convertComplexValue converts a complex interface{} value to map[string]interface{}
func convertComplexValue(value interface{}) map[string]interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case map[string]interface{}:
		// Filter out empty values
		result := make(map[string]interface{})
		for key, val := range v {
			if val != nil && val != "" {
				result[key] = val
			}
		}
		if len(result) == 0 {
			return nil
		}
		return result

	case string:
		// Try to parse as JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(v), &parsed); err == nil {
			return convertComplexValue(parsed)
		}
		// If not JSON, return as single string value
		if v != "" {
			return map[string]interface{}{"value": v}
		}
		return nil

	default:
		// Convert other types to string representation
		if str := fmt.Sprintf("%v", v); str != "" && str != "<nil>" {
			return map[string]interface{}{"value": str}
		}
		return nil
	}
}

// ------------------------
// JSON Conversion Methods for Form/JSON Toggle
// ------------------------

// ToJSON converts the Inputs to a JSON string
// Includes all fields with their current values (not just dirty fields)
func (i *Inputs) ToJSON() (string, error) {
	data := i.toJSONMap(false)
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal inputs to JSON: %w", err)
	}
	return string(jsonBytes), nil
}

// ToJSONIndented converts the Inputs to a pretty-printed JSON string
// Includes all fields with their current values (not just dirty fields)
func (i *Inputs) ToJSONIndented() (string, error) {
	data := i.toJSONMap(false)
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal inputs to JSON: %w", err)
	}
	return string(jsonBytes), nil
}

// toJSONMap converts Inputs to a map suitable for JSON marshaling
// If skipDefaults is true, only includes non-default values
func (i *Inputs) toJSONMap(skipDefaults bool) map[string]interface{} {
	result := make(map[string]interface{})

	for _, wrapper := range *i {
		// Skip default values if requested
		if skipDefaults && wrapper.IsDefaultValue() && !wrapper.IsRequired() {
			continue
		}

		fieldName := wrapper.GetLabel()
		value := wrapper.convertToJSONValue()

		// Always include the field, even if nil (for JSON structure visibility)
		result[fieldName] = value
	}

	return result
}

// convertToJSONValue converts an InputWrapper value to a JSON-compatible type
func (iw *InputWrapper) convertToJSONValue() interface{} {
	// If field is not touched, return null (especially important for bools)
	if !iw.Touched {
		return nil
	}

	switch iw.InputType {
	case InputTypeText:
		val := iw.String()
		if val == "" {
			return nil
		}
		return val

	case InputTypeBool:
		return iw.Bool()

	case InputTypeInt64:
		if iw.IsDefaultValue() && !iw.IsRequired() {
			return nil
		}
		return iw.Int64()

	case InputTypeFloat64:
		if iw.IsDefaultValue() && !iw.IsRequired() {
			return nil
		}
		return iw.Float64()

	case InputTypePrimitivesArray:
		if iw.PrimitivesArrayInput == nil {
			return []interface{}{}
		}

		// Get the array values
		values := iw.PrimitivesArrayInput.getAllValues()
		if len(values) == 0 {
			return []interface{}{}
		}

		// Convert based on array type
		arrayType := iw.PrimitivesArrayInput.arrayType
		switch arrayType {
		case "array[int]", "array[int64]":
			result := make([]interface{}, 0, len(values))
			for _, val := range values {
				if parsed, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64); err == nil {
					result = append(result, parsed)
				}
			}
			return result

		case "array[float]", "array[float64]":
			result := make([]interface{}, 0, len(values))
			for _, val := range values {
				if parsed, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
					result = append(result, parsed)
				}
			}
			return result

		case "array[bool]":
			result := make([]interface{}, 0, len(values))
			for _, val := range values {
				trimmed := strings.ToLower(strings.TrimSpace(val))
				result = append(result, trimmed == "true" || trimmed == "yes" || trimmed == "y" || trimmed == "1")
			}
			return result

		default: // "array[string]" or unspecified
			result := make([]interface{}, 0, len(values))
			for _, val := range values {
				if trimmed := strings.TrimSpace(val); trimmed != "" {
					result = append(result, trimmed)
				}
			}
			return result
		}

	case InputTypeComplexArray:
		if iw.ComplexArrayInput == nil {
			return []interface{}{}
		}

		// Get typed values
		if typedValues := iw.ComplexArrayInput.TypedValue(); typedValues != nil {
			if slice, ok := typedValues.([]string); ok {
				result := make([]interface{}, 0, len(slice))
				for _, val := range slice {
					if trimmed := strings.TrimSpace(val); trimmed != "" {
						result = append(result, trimmed)
					}
				}
				return result
			}
			if slice, ok := typedValues.([]interface{}); ok {
				result := make([]interface{}, 0, len(slice))
				for _, val := range slice {
					if converted := convertComplexValueToJSON(val); converted != nil {
						result = append(result, converted)
					}
				}
				return result
			}
		}
		return []interface{}{}

	case InputTypeNested:
		if iw.NestedInput == nil {
			return make(map[string]interface{})
		}

		// Recursively convert nested inputs
		nestedData := make(map[string]interface{})
		for _, childWrapper := range iw.NestedInput.GetInputs() {
			nestedData[childWrapper.GetLabel()] = childWrapper.convertToJSONValue()
		}
		return nestedData

	default:
		// Fallback to string value
		val := iw.String()
		if val == "" {
			return nil
		}
		return val
	}
}

// convertComplexValueToJSON converts a complex interface{} value to JSON-compatible format
func convertComplexValueToJSON(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case map[string]interface{}:
		// Filter out empty values
		result := make(map[string]interface{})
		for key, val := range v {
			if val != nil && val != "" {
				result[key] = val
			}
		}
		if len(result) == 0 {
			return nil
		}
		return result

	case string:
		// Try to parse as JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(v), &parsed); err == nil {
			return convertComplexValueToJSON(parsed)
		}
		// If not JSON, return as string
		if v != "" {
			return v
		}
		return nil

	default:
		// Return other types as-is
		if str := fmt.Sprintf("%v", v); str != "" && str != "<nil>" {
			return v
		}
		return nil
	}
}

// FromJSON populates the Inputs from a JSON string
// Updates field values based on the JSON data
func (i *Inputs) FromJSON(jsonStr string) error {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return i.fromJSONMap(data)
}

// fromJSONMap populates Inputs from a map
func (i *Inputs) fromJSONMap(data map[string]interface{}) error {
	for _, wrapper := range *i {
		fieldName := wrapper.GetLabel()
		value, exists := data[fieldName]
		if !exists {
			continue
		}

		// Skip null values - they represent untouched fields
		if value == nil {
			wrapper.Touched = false
			continue
		}

		if err := wrapper.setFromJSONValue(value); err != nil {
			log.Debug("Failed to set field from JSON",
				zap.String("field", fieldName),
				zap.Error(err))
			// Continue processing other fields instead of failing completely
		} else {
			// Mark field as touched if it has a non-null value
			wrapper.Touched = true
		}
	}

	return nil
}

// setFromJSONValue sets an InputWrapper value from a JSON-compatible type
func (iw *InputWrapper) setFromJSONValue(value interface{}) error {
	if value == nil {
		// Set to empty/default value
		iw.SetValue("")
		return nil
	}

	switch iw.InputType {
	case InputTypeText:
		if str, ok := value.(string); ok {
			iw.SetValue(str)
			return nil
		}
		// Try to convert to string
		iw.SetValue(fmt.Sprintf("%v", value))
		return nil

	case InputTypeBool:
		if b, ok := value.(bool); ok {
			iw.SetValue(fmt.Sprintf("%v", b))
			return nil
		}
		return fmt.Errorf("expected bool, got %T", value)

	case InputTypeInt64:
		switch v := value.(type) {
		case float64:
			// JSON numbers are decoded as float64
			iw.SetValue(fmt.Sprintf("%d", int64(v)))
			return nil
		case int64:
			iw.SetValue(fmt.Sprintf("%d", v))
			return nil
		case int:
			iw.SetValue(fmt.Sprintf("%d", v))
			return nil
		case string:
			// Try to parse string as int
			if _, err := strconv.ParseInt(v, 10, 64); err == nil {
				iw.SetValue(v)
				return nil
			}
			return fmt.Errorf("invalid int64 string: %s", v)
		default:
			return fmt.Errorf("expected number, got %T", value)
		}

	case InputTypeFloat64:
		switch v := value.(type) {
		case float64:
			iw.SetValue(fmt.Sprintf("%f", v))
			return nil
		case int64:
			iw.SetValue(fmt.Sprintf("%f", float64(v)))
			return nil
		case int:
			iw.SetValue(fmt.Sprintf("%f", float64(v)))
			return nil
		case string:
			// Try to parse string as float
			if _, err := strconv.ParseFloat(v, 64); err == nil {
				iw.SetValue(v)
				return nil
			}
			return fmt.Errorf("invalid float64 string: %s", v)
		default:
			return fmt.Errorf("expected number, got %T", value)
		}

	case InputTypePrimitivesArray, InputTypeComplexArray:
		// Handle array types
		if arr, ok := value.([]interface{}); ok {
			var strValues []string
			for _, item := range arr {
				strValues = append(strValues, fmt.Sprintf("%v", item))
			}
			jsonBytes, _ := json.Marshal(strValues)
			iw.SetValue(string(jsonBytes))
			return nil
		}
		return fmt.Errorf("expected array, got %T", value)

	case InputTypeNested:
		// Handle nested objects
		if obj, ok := value.(map[string]interface{}); ok {
			if iw.NestedInput != nil {
				for _, childWrapper := range iw.NestedInput.GetInputs() {
					childValue, exists := obj[childWrapper.GetLabel()]
					if exists {
						if err := childWrapper.setFromJSONValue(childValue); err != nil {
							log.Debug("Failed to set nested field from JSON",
								zap.String("field", childWrapper.GetLabel()),
								zap.Error(err))
						}
					}
				}
			}
			return nil
		}
		return fmt.Errorf("expected object, got %T", value)

	default:
		// Fallback to string conversion
		iw.SetValue(fmt.Sprintf("%v", value))
		return nil
	}
}
