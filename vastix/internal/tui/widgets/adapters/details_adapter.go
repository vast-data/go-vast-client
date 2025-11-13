package adapters

import (
	"vastix/internal/colors"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"vastix/internal/database"
	log "vastix/internal/logging"
	"vastix/internal/msg_types"
	"vastix/internal/tui/widgets/common"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	vast_client "github.com/vast-data/go-vast-client"
	"go.uber.org/zap"
)

// DetailsAdapter handles the details view for widgets
type DetailsAdapter struct {
	resourceType    string // Type of resource this adapter represents, e.g., "views", "quotas", "users" etc.
	db              *database.Service
	predefinedTitle string // Optional predefined title to override default

	viewport      viewport.Model
	rawContent    any
	content       string // Content to display in the viewport
	ready         bool
	width, height int

	// Popup input support
	popupInput *PopupInput
}

// NewDetailsAdapter creates a new details adapter
func NewDetailsAdapter(db *database.Service, resourceType string) *DetailsAdapter {
	adapter := &DetailsAdapter{
		db:           db,
		resourceType: resourceType,
		popupInput:   NewPopupInput(),
	}
	return adapter
}

// NewDetailsAdapterWithPredefinedTitle creates a new details adapter with a predefined title
func NewDetailsAdapterWithPredefinedTitle(db *database.Service, resourceType, title string) *DetailsAdapter {
	adapter := &DetailsAdapter{
		db:              db,
		resourceType:    resourceType,
		predefinedTitle: title,
		popupInput:      NewPopupInput(),
	}
	return adapter
}

// SetPredefinedTitle allows setting the predefined title dynamically
func (da *DetailsAdapter) SetPredefinedTitle(title string) {
	da.predefinedTitle = title
}

// SetSize sets the dimensions of the details adapter
func (da *DetailsAdapter) SetSize(width, height int) {
	da.width = width
	da.height = height

	if da.ready {
		// Account for borders and header space
		innerWidth := width - 2   // Left and right borders
		innerHeight := height - 2 // Top, bottom borders, header

		// Ensure minimum viable dimensions
		if innerWidth < 1 {
			innerWidth = 1
		}
		if innerHeight < 1 {
			innerHeight = 1
		}

		da.viewport.Width = innerWidth
		da.viewport.Height = innerHeight
	}
}

// SetContent sets the content to display in the viewport
func (da *DetailsAdapter) SetContent(rawContent any) {
	da.rawContent = rawContent
	content := da.contentToString()
	da.content = content

	if da.ready {
		// Reset viewport to top when setting new content
		da.viewport.GotoTop()
		da.viewport.SetContent(content)
	}
}

// Write implements io.Writer interface for streaming logs to the viewport
// Applies left padding and gray color styling to log lines
func (da *DetailsAdapter) Write(p []byte) (n int, err error) {
	text := string(p)

	// Apply styling: left padding and gray color
	leftPadding := "  "                                                // 2 spaces left padding
	grayStyle := lipgloss.NewStyle().Foreground(colors.Grey240) // Gray color

	lines := strings.Split(text, "\n")
	var styledLines []string

	for _, line := range lines {
		if line == "" {
			// Keep empty lines as-is
			styledLines = append(styledLines, line)
		} else {
			// Apply padding and gray color
			styledLine := leftPadding + grayStyle.Render(line)
			styledLines = append(styledLines, styledLine)
		}
	}

	styledText := strings.Join(styledLines, "\n")
	da.content += styledText

	if da.ready {
		da.viewport.SetContent(da.content)
		// Auto-scroll to bottom when new content is written
		da.viewport.GotoBottom()
	}

	return len(p), nil
}

// AppendContent appends a string to the current content
// Applies left padding and gray color styling to log lines
func (da *DetailsAdapter) AppendContent(text string) {
	// Apply styling: left padding and gray color
	leftPadding := "  "                                                // 2 spaces left padding
	grayStyle := lipgloss.NewStyle().Foreground(colors.Grey240) // Gray color

	lines := strings.Split(text, "\n")
	var styledLines []string

	for _, line := range lines {
		if line == "" {
			// Keep empty lines as-is
			styledLines = append(styledLines, line)
		} else {
			// Apply padding and gray color
			styledLine := leftPadding + grayStyle.Render(line)
			styledLines = append(styledLines, styledLine)
		}
	}

	styledText := strings.Join(styledLines, "\n")
	da.content += styledText

	if da.ready {
		da.viewport.SetContent(da.content)
		da.viewport.GotoBottom()
	}
}

// ClearContent clears all content
func (da *DetailsAdapter) ClearContent() {
	da.content = ""
	da.rawContent = nil

	if da.ready {
		da.viewport.SetContent("")
		da.viewport.GotoTop()
	}
}

// UpdateViewPort handles messages for the details adapter
// Also handles popup input updates if popup is visible
func (da *DetailsAdapter) UpdateViewPort(msg tea.Msg) tea.Cmd {
	// If popup is visible, route updates to it
	if !da.popupInput.IsHidden() {
		return da.popupInput.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		da.initializeViewport(msg.Width, msg.Height)
	}

	if da.ready {
		var cmd tea.Cmd
		da.viewport, cmd = da.viewport.Update(msg)
		return cmd
	}

	return nil
}

// initializeViewport initializes the viewport with proper dimensions
func (da *DetailsAdapter) initializeViewport(width, height int) {
	if !da.ready {
		// Account for borders and header/footer space
		innerWidth := width - 2   // Left and right borders
		innerHeight := height - 2 // Top, bottom borders, header

		// Ensure minimum viable dimensions
		if innerWidth < 1 {
			innerWidth = 1
		}
		if innerHeight < 1 {
			innerHeight = 1
		}

		da.viewport = viewport.New(innerWidth, innerHeight)
		da.viewport.SetContent(da.content)
		da.ready = true

		log.Debug("Viewport initialized",
			zap.Int("width", innerWidth),
			zap.Int("height", innerHeight),
			zap.Int("originalWidth", width),
			zap.Int("originalHeight", height))
	}
}

// ViewDetails renders the details view with viewport, accepting fuzzy search parameter
func (da *DetailsAdapter) ViewDetails(width, height int, fuzzyDetailsLocalSearch string) string {
	// Minimum dimensions required for proper display
	minWidth := 15 // Reduced from 20 to be more forgiving
	minHeight := 6 // Reduced from 8 to be more forgiving

	if width < minWidth || height < minHeight {
		// For very small screens, show a simple message
		message := "Screen too small\nfor details view"
		// Try to fit at least the border and message
		if width >= 10 && height >= 4 {
			return common.BorderizeWithSpinnerCheck(message, true, map[common.BorderPosition]string{
				common.TopMiddleBorder: " details ",
			})
		} else {
			// Extremely small - just return the message without border
			return message
		}
	}

	// Initialize viewport if not ready
	if !da.ready {
		da.initializeViewport(width, height)
	}

	resourceNameStyle := lipgloss.NewStyle().
		Background(colors.Orange). // Orange background
		Foreground(colors.BlackTerm)    // Black text

	// Apply fuzzy search to the FULL content first (not just visible content)
	var filteredContent string
	if fuzzyDetailsLocalSearch != "" && da.content != "" {
		var filteredLines []string
		for _, line := range strings.Split(da.content, "\n") {
			// If fuzzy search is active, use proper fuzzy matching (case-insensitive)
			if da.fuzzyMatch(strings.ToLower(line), strings.ToLower(fuzzyDetailsLocalSearch)) {
				filteredLines = append(filteredLines, line)
			}
		}
		filteredContent = strings.Join(filteredLines, "\n")
	} else {
		filteredContent = da.content
	}

	// Update viewport with filtered content and get the content to render
	var content string
	if da.ready {
		da.viewport.SetContent(filteredContent)
		content = da.viewport.View()
	} else {
		content = "Initializing details view..."
	}

	// Calculate dimensions with safety checks
	innerWidth := width - 2   // Account for left and right borders
	innerHeight := height - 3 // Account for top, bottom borders and header

	// Ensure minimum viable dimensions
	if innerWidth < 1 {
		innerWidth = 1
	}
	if innerHeight < 1 {
		innerHeight = 1
	}

	// Split content into lines for processing
	lines := strings.Split(content, "\n")

	// Create a style for opaque background to cover any content behind
	opaqueStyle := lipgloss.NewStyle().
		Width(innerWidth).
		Background(colors.BlackTerm) // Black background to ensure opacity

	// Ensure we have enough lines with opaque background
	for len(lines) < innerHeight {
		lines = append(lines, opaqueStyle.Render(strings.Repeat(" ", innerWidth)))
	}

	// Ensure each line fills the width with opaque background
	for i, line := range lines {
		lineWidth := lipgloss.Width(line)
		if lineWidth < innerWidth {
			// Pad with spaces and ensure opaque background
			paddedLine := line + strings.Repeat(" ", innerWidth-lineWidth)
			lines[i] = opaqueStyle.Render(paddedLine)
		} else if lineWidth > innerWidth {
			// Truncate lines that are too long, preserving styling with opaque background
			lines[i] = opaqueStyle.Render(lipgloss.NewStyle().Width(innerWidth).Render(line))
		} else {
			// Line is exact width, ensure opaque background
			lines[i] = opaqueStyle.Render(line)
		}
	}

	content = strings.Join(lines, "\n")

	// Use predefined title if available, otherwise use default format
	var titleText string
	if da.predefinedTitle != "" {
		titleText = da.predefinedTitle
	} else {
		titleText = "details: " + da.resourceType
	}
	resourceTypeLabel := resourceNameStyle.Render(" " + titleText + " ")

	// Add fuzzy search label if active
	if fuzzyDetailsLocalSearch != "" {
		labelStyle := lipgloss.NewStyle().
			Background(colors.DarkGreenBlue). // Muted green background
			Foreground(colors.BlackTerm)   // Black text

		label := labelStyle.Render(fmt.Sprintf(" fuzzy-search: %s ", fuzzyDetailsLocalSearch))
		resourceTypeLabel = fmt.Sprintf("%s %s", resourceTypeLabel, label)
	}

	// Create border with embedded text
	embeddedText := map[common.BorderPosition]string{
		common.TopMiddleBorder: resourceTypeLabel,
	}

	if da.ready && da.rawContent != nil {
		// Add scroll percentage to bottom right (only when there's content)
		scrollPercent := da.viewport.ScrollPercent() * 100
		embeddedText[common.BottomRightBorder] = lipgloss.NewStyle().
			Render(fmt.Sprintf("%.0f%%", scrollPercent))
	}

	rendered := common.BorderizeWithSpinnerCheck(content, true, embeddedText)

	// If popup is visible, just show the popup (no base content overlay)
	if !da.popupInput.IsHidden() {
		return da.popupInput.View(width, height)
	}

	return rendered
}

// ShowPopup shows the popup input with the given title and placeholder
func (da *DetailsAdapter) ShowPopup(title, placeholder string, isSecret bool) {
	da.popupInput.Show(title, placeholder, isSecret)
}

// HidePopup hides the popup input
func (da *DetailsAdapter) HidePopup() {
	da.popupInput.Hide()
}

// IsPopupHidden returns whether the popup is hidden
func (da *DetailsAdapter) IsPopupHidden() bool {
	return da.popupInput.IsHidden()
}

// GetPopupContent returns the popup input content
func (da *DetailsAdapter) GetPopupContent() string {
	return da.popupInput.GetContent()
}

// ClearPopupContent clears the popup content
func (da *DetailsAdapter) ClearPopupContent() {
	da.popupInput.ClearContent()
}

// CopyToClipboard copies the raw content to clipboard
func (da *DetailsAdapter) CopyToClipboard() tea.Msg {
	// Copy to clipboard
	if err := clipboard.WriteAll(da.contentToClipboardString()); err != nil {
		return msg_types.ErrorMsg{Err: err}
	}
	return msg_types.InfoMsg{
		Message: "Content copied to clipboard",
	}
}

// fuzzyMatch checks if text contains all characters of query in order (not necessarily consecutive)
func (da *DetailsAdapter) fuzzyMatch(text, query string) bool {
	if query == "" {
		return true
	}

	textIndex := 0
	for _, queryChar := range query {
		found := false
		for textIndex < len(text) {
			if rune(text[textIndex]) == queryChar {
				found = true
				textIndex++
				break
			}
			textIndex++
		}
		if !found {
			return false
		}
	}
	return true
}

// Reset resets the details adapter state
func (da *DetailsAdapter) Reset() {
	da.content = ""
	da.ready = false
	da.resourceType = ""
	da.rawContent = nil
	log.Debug("DetailsAdapter reset")
}

func (da *DetailsAdapter) contentToString() string {
	// Handle nil rawContent by showing gray "No content" with padding
	if da.rawContent == nil {
		grayStyle := lipgloss.NewStyle().Foreground(colors.LightGrey) // Light gray color
		paddingStyle := lipgloss.NewStyle().
			Padding(0, 0, 0, 2) // top: 0, right: 0, bottom: 0, left: 2

		noContentMessage := grayStyle.Render("No content")
		return paddingStyle.Render(noContentMessage)
	}

	switch v := da.rawContent.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case map[string]any:
		return formatRecordAsJSON(v)
	case vast_client.Record:
		return formatRecordAsJSON(v)
	case vast_client.RecordSet:
		// Format RecordSet as a JSON array of objects
		return formatRecordSetAsJSON(v)
	default:
		// Try to handle structs with JSON tags by converting to map
		if isStruct(v) {
			if jsonMap, err := structToMap(v); err == nil {
				return formatRecordAsJSON(jsonMap)
			}
		}
		return fmt.Sprintf("%v", v) // Fallback for other types
	}
}

// isStruct checks if the given value is a struct
func isStruct(v any) bool {
	if v == nil {
		return false
	}
	t := reflect.TypeOf(v)
	// Handle pointers to structs as well
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Kind() == reflect.Struct
}

// structToMap converts a struct to map[string]any using JSON tags
func structToMap(v any) (map[string]any, error) {
	// Use JSON marshaling/unmarshaling to convert struct to map
	// This respects JSON tags and handles nested structs properly
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal struct to JSON: %w", err)
	}

	var result map[string]any
	err = json.Unmarshal(jsonBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	return result, nil
}

func (da *DetailsAdapter) contentToClipboardString() string {
	switch v := da.rawContent.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case map[string]any:
		return vast_client.Record(v).PrettyJson("  ")
	case vast_client.Record:
		return v.PrettyJson("  ")
	case vast_client.RecordSet:
		// Format RecordSet as pretty JSON for clipboard
		return v.PrettyJson("  ")
	default:
		// Try to handle structs with JSON tags by converting to JSON
		if isStruct(v) {
			if bytes, err := json.MarshalIndent(v, "", "  "); err == nil {
				return string(bytes)
			}
		}
		return fmt.Sprintf("%v", v) // Fallback for other types
	}
}

// Helper function to check if a string is a Go map representation
func isGoMapString(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "map[") && strings.HasSuffix(s, "]")
}

// Helper function to parse Go map string representation
func parseGoMapString(mapStr string) (map[string]interface{}, error) {
	mapStr = strings.TrimSpace(mapStr)
	if !strings.HasPrefix(mapStr, "map[") || !strings.HasSuffix(mapStr, "]") {
		return nil, fmt.Errorf("not a Go map string")
	}

	// Remove "map[" prefix and "]" suffix
	content := mapStr[4 : len(mapStr)-1]

	result := make(map[string]interface{})

	// Simple parser for key:value pairs
	// This handles the specific format: "key1:value1 key2:value2 ..."
	if content == "" {
		return result, nil
	}

	// Split by spaces, but be careful with quoted values
	var pairs []string
	var currentPair strings.Builder
	inQuotes := false

	for i, r := range content {
		switch r {
		case '"':
			inQuotes = !inQuotes
			currentPair.WriteRune(r)
		case ' ':
			if !inQuotes {
				if currentPair.Len() > 0 {
					pairs = append(pairs, currentPair.String())
					currentPair.Reset()
				}
			} else {
				currentPair.WriteRune(r)
			}
		default:
			currentPair.WriteRune(r)
		}

		// Add the last pair if we're at the end
		if i == len(content)-1 && currentPair.Len() > 0 {
			pairs = append(pairs, currentPair.String())
		}
	}

	// Parse each key:value pair
	for _, pair := range pairs {
		colonIndex := strings.Index(pair, ":")
		if colonIndex == -1 {
			continue // Skip invalid pairs
		}

		key := strings.TrimSpace(pair[:colonIndex])
		value := strings.TrimSpace(pair[colonIndex+1:])

		// Remove quotes if present
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = value[1 : len(value)-1]
		}

		result[key] = value
	}

	return result, nil
}

// formatObjectRecursive formats a JSON object string with proper indentation and colors recursively
func formatObjectRecursive(objStr string, nestLevel int) string {
	// Define colors for syntax highlighting (balanced brightness)
	keyColor := lipgloss.NewStyle().Foreground(colors.MediumCyan)      // Medium cyan for keys
	stringColor := lipgloss.NewStyle().Foreground(colors.MediumGreen)   // Medium green for strings
	numberColor := lipgloss.NewStyle().Foreground(colors.MutedOrange)  // Muted orange for numbers
	boolColor := lipgloss.NewStyle().Foreground(colors.MediumPurple)    // Medium purple for booleans
	nullColor := lipgloss.NewStyle().Foreground(colors.MediumGrey)    // Gray for null values
	bracketColor := lipgloss.NewStyle().Foreground(colors.VeryLightGrey) // Light white for brackets/punctuation

	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(objStr), &obj); err != nil {
		// If parsing fails, treat as regular string
		return stringColor.Render(fmt.Sprintf("\"%s\"", objStr))
	}

	if len(obj) == 0 {
		return bracketColor.Render("{}")
	}

	// Calculate indentation for nested objects
	baseIndent := strings.Repeat("  ", nestLevel+1) // Base indentation
	fieldIndent := baseIndent + "  "                // Field indentation (extra 2 spaces)

	// Build nested object string with proper indentation and colors
	var details strings.Builder
	details.WriteString(bracketColor.Render("{\n"))

	// Get keys and sort them for consistent output
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Format each field
	for i, key := range keys {
		isLast := i == len(keys)-1
		keyPart := keyColor.Render(fmt.Sprintf("\"%s\"", key))
		var valuePart string

		switch v := obj[key].(type) {
		case string:
			if isGoMapString(v) {
				// Handle Go map string representation
				if parsedMap, err := parseGoMapString(v); err == nil {
					// Convert parsed map to JSON and format it recursively
					if jsonBytes, err := json.Marshal(parsedMap); err == nil {
						valuePart = formatObjectRecursive(string(jsonBytes), nestLevel+1)
					} else {
						valuePart = stringColor.Render(fmt.Sprintf("\"%s\"", v))
					}
				} else {
					valuePart = stringColor.Render(fmt.Sprintf("\"%s\"", v))
				}
			} else {
				valuePart = stringColor.Render(fmt.Sprintf("\"%s\"", v))
			}
		case float64:
			if math.Mod(v, 1) == 0 {
				valuePart = numberColor.Render(fmt.Sprintf("%.0f", v))
			} else {
				valuePart = numberColor.Render(fmt.Sprintf("%.2f", v))
			}
		case bool:
			valuePart = boolColor.Render(fmt.Sprintf("%t", v))
		case nil:
			valuePart = nullColor.Render("null")
		case map[string]interface{}:
			// Format nested objects with proper indentation recursively
			if jsonBytes, err := json.Marshal(v); err == nil {
				valuePart = formatObjectRecursive(string(jsonBytes), nestLevel+1)
			} else {
				valuePart = stringColor.Render(fmt.Sprintf("\"%v\"", v))
			}
		case []interface{}:
			// Handle nested arrays - check if items are complex enough for multiline format
			if len(v) == 0 {
				valuePart = bracketColor.Render("[]")
			} else {
				// Check if any items are Go map strings or complex objects
				hasComplexItems := false
				for _, item := range v {
					if str, ok := item.(string); ok && isGoMapString(str) {
						hasComplexItems = true
						break
					}
					if _, isMap := item.(map[string]interface{}); isMap {
						hasComplexItems = true
						break
					}
					if _, isArray := item.([]interface{}); isArray {
						hasComplexItems = true
						break
					}
				}

				if hasComplexItems || len(v) > 0 {
					// Use multiline format for complex items OR for better readability of all arrays
					result := bracketColor.Render("[\n")
					for i, item := range v {
						isLast := i == len(v)-1
						indent := fieldIndent + "  "

						var itemStr string
						switch av := item.(type) {
						case string:
							if isGoMapString(av) {
								// Handle Go map string representation
								if parsedMap, err := parseGoMapString(av); err == nil {
									// Convert parsed map to JSON and format it recursively
									if jsonBytes, err := json.Marshal(parsedMap); err == nil {
										itemStr = formatObjectRecursive(string(jsonBytes), nestLevel+2)
									} else {
										itemStr = stringColor.Render(fmt.Sprintf("\"%s\"", av))
									}
								} else {
									itemStr = stringColor.Render(fmt.Sprintf("\"%s\"", av))
								}
							} else {
								itemStr = stringColor.Render(fmt.Sprintf("\"%s\"", av))
							}
						case float64:
							if math.Mod(av, 1) == 0 {
								itemStr = numberColor.Render(fmt.Sprintf("%.0f", av))
							} else {
								itemStr = numberColor.Render(fmt.Sprintf("%.2f", av))
							}
						case bool:
							itemStr = boolColor.Render(fmt.Sprintf("%t", av))
						case nil:
							itemStr = nullColor.Render("null")
						case map[string]interface{}:
							// Format nested objects recursively
							if jsonBytes, err := json.Marshal(av); err == nil {
								itemStr = formatObjectRecursive(string(jsonBytes), nestLevel+2)
							} else {
								itemStr = stringColor.Render(fmt.Sprintf("\"%v\"", av))
							}
						default:
							itemStr = stringColor.Render(fmt.Sprintf("\"%v\"", av))
						}

						comma := ""
						if !isLast {
							comma = bracketColor.Render(",")
						}
						result += indent + itemStr + comma + "\n"
					}
					result += fieldIndent + bracketColor.Render("]")
					valuePart = result
				} else {
					// Use inline format for simple items
					var arrayItems []string
					for _, item := range v {
						switch av := item.(type) {
						case string:
							arrayItems = append(arrayItems, stringColor.Render(fmt.Sprintf("\"%s\"", av)))
						case float64:
							if math.Mod(av, 1) == 0 {
								arrayItems = append(arrayItems, numberColor.Render(fmt.Sprintf("%.0f", av)))
							} else {
								arrayItems = append(arrayItems, numberColor.Render(fmt.Sprintf("%.2f", av)))
							}
						case bool:
							arrayItems = append(arrayItems, boolColor.Render(fmt.Sprintf("%t", av)))
						case nil:
							arrayItems = append(arrayItems, nullColor.Render("null"))
						default:
							arrayItems = append(arrayItems, stringColor.Render(fmt.Sprintf("\"%v\"", av)))
						}
					}
					valuePart = bracketColor.Render("[") + strings.Join(arrayItems, bracketColor.Render(", ")) + bracketColor.Render("]")
				}
			}
		default:
			valuePart = stringColor.Render(fmt.Sprintf("\"%v\"", v))
		}

		comma := ""
		if !isLast {
			comma = bracketColor.Render(",")
		}

		details.WriteString(fieldIndent + keyPart + bracketColor.Render(": ") + valuePart + comma + "\n")
	}

	details.WriteString(baseIndent + bracketColor.Render("}"))
	return details.String()
}

// formatRecordAsJSON converts a map[string]any record into JSON-style formatted string with syntax highlighting
// Moved here from widgets/common_utils.go to restore original colored formatting
func formatRecordAsJSON(record map[string]any) string {
	delete(record, "@resourceType")
	var details strings.Builder

	// Define colors for syntax highlighting (balanced brightness)
	keyColor := lipgloss.NewStyle().Foreground(colors.MediumCyan)      // Medium cyan for keys
	stringColor := lipgloss.NewStyle().Foreground(colors.MediumGreen)   // Medium green for strings
	numberColor := lipgloss.NewStyle().Foreground(colors.MutedOrange)  // Muted orange for numbers
	boolColor := lipgloss.NewStyle().Foreground(colors.MediumPurple)    // Medium purple for booleans
	nullColor := lipgloss.NewStyle().Foreground(colors.MediumGrey)    // Gray for null values
	bracketColor := lipgloss.NewStyle().Foreground(colors.VeryLightGrey) // Light white for brackets/punctuation

	// Left margin (2 spaces)
	leftMargin := "  "

	// Start JSON object
	details.WriteString(leftMargin + bracketColor.Render("{\n"))

	// Helper function to check if a string is a JSON array
	isJSONArray := func(s string) bool {
		s = strings.TrimSpace(s)
		return strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")
	}

	// Helper function to check if a string is a JSON object
	isJSONObject := func(s string) bool {
		s = strings.TrimSpace(s)
		return strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")
	}

	// Helper function to check if a string is a Go map representation
	isGoMapString := func(s string) bool {
		s = strings.TrimSpace(s)
		return strings.HasPrefix(s, "map[") && strings.HasSuffix(s, "]")
	}

	// Helper function to parse Go map string representation
	parseGoMapString := func(mapStr string) (map[string]interface{}, error) {
		mapStr = strings.TrimSpace(mapStr)
		if !strings.HasPrefix(mapStr, "map[") || !strings.HasSuffix(mapStr, "]") {
			return nil, fmt.Errorf("not a Go map string")
		}

		// Remove "map[" prefix and "]" suffix
		content := mapStr[4 : len(mapStr)-1]

		result := make(map[string]interface{})

		// Simple parser for key:value pairs
		// This handles the specific format: "key1:value1 key2:value2 ..."
		if content == "" {
			return result, nil
		}

		// Split by spaces, but be careful with quoted values
		var pairs []string
		var currentPair strings.Builder
		inQuotes := false

		for i, r := range content {
			switch r {
			case '"':
				inQuotes = !inQuotes
				currentPair.WriteRune(r)
			case ' ':
				if !inQuotes {
					if currentPair.Len() > 0 {
						pairs = append(pairs, currentPair.String())
						currentPair.Reset()
					}
				} else {
					currentPair.WriteRune(r)
				}
			default:
				currentPair.WriteRune(r)
			}

			// Add the last pair if we're at the end
			if i == len(content)-1 && currentPair.Len() > 0 {
				pairs = append(pairs, currentPair.String())
			}
		}

		// Parse each key:value pair
		for _, pair := range pairs {
			colonIndex := strings.Index(pair, ":")
			if colonIndex == -1 {
				continue // Skip invalid pairs
			}

			key := strings.TrimSpace(pair[:colonIndex])
			value := strings.TrimSpace(pair[colonIndex+1:])

			// Remove quotes if present
			if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
				value = value[1 : len(value)-1]
			}

			result[key] = value
		}

		return result, nil
	}

	// Helper function to format JSON object with proper indentation (unused - replaced by formatObjectRecursive)
	_ = func(objStr string, nestLevel int) string {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(objStr), &obj); err != nil {
			// If parsing fails, treat as regular string
			return stringColor.Render(fmt.Sprintf("\"%s\"", objStr))
		}

		if len(obj) == 0 {
			return bracketColor.Render("{}")
		}

		// Calculate indentation for nested objects
		baseIndent := strings.Repeat("  ", nestLevel+1) // Base indentation
		fieldIndent := baseIndent + "  "                // Field indentation (extra 2 spaces)

		// Build nested object string with proper indentation and colors
		var details strings.Builder
		details.WriteString(bracketColor.Render("{\n"))

		// Get keys and sort them for consistent output
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// Format each field
		for i, key := range keys {
			isLast := i == len(keys)-1
			keyPart := keyColor.Render(fmt.Sprintf("\"%s\"", key))
			var valuePart string

			switch v := obj[key].(type) {
			case string:
				valuePart = stringColor.Render(fmt.Sprintf("\"%s\"", v))
			case float64:
				if math.Mod(v, 1) == 0 {
					valuePart = numberColor.Render(fmt.Sprintf("%.0f", v))
				} else {
					valuePart = numberColor.Render(fmt.Sprintf("%.2f", v))
				}
			case bool:
				valuePart = boolColor.Render(fmt.Sprintf("%t", v))
			case nil:
				valuePart = nullColor.Render("null")
			case map[string]interface{}:
				// Format nested objects with proper indentation
				// Get keys and sort them for consistent output
				nestedKeys := make([]string, 0, len(v))
				for k := range v {
					nestedKeys = append(nestedKeys, k)
				}
				sort.Strings(nestedKeys)

				nestedObj := bracketColor.Render("{\n")
				for i, nestedKey := range nestedKeys {
					nestedValue := v[nestedKey]
					nestedKeyPart := keyColor.Render(fmt.Sprintf("\"%s\"", nestedKey))
					var nestedValuePart string
					switch nv := nestedValue.(type) {
					case string:
						nestedValuePart = stringColor.Render(fmt.Sprintf("\"%s\"", nv))
					case float64:
						if math.Mod(nv, 1) == 0 {
							nestedValuePart = numberColor.Render(fmt.Sprintf("%.0f", nv))
						} else {
							nestedValuePart = numberColor.Render(fmt.Sprintf("%.2f", nv))
						}
					case bool:
						nestedValuePart = boolColor.Render(fmt.Sprintf("%t", nv))
					case nil:
						nestedValuePart = nullColor.Render("null")
					default:
						nestedValuePart = stringColor.Render(fmt.Sprintf("\"%v\"", nv))
					}

					comma := ""
					if i < len(nestedKeys)-1 {
						comma = bracketColor.Render(",")
					}

					nestedObj += fieldIndent + nestedKeyPart + bracketColor.Render(": ") + nestedValuePart + comma + "\n"
				}
				nestedObj += fieldIndent + bracketColor.Render("}")
				valuePart = nestedObj
			case []interface{}:
				// Handle nested arrays - check if items are complex enough for multiline format
				if len(v) == 0 {
					valuePart = bracketColor.Render("[]")
				} else {
					// Check if any items are Go map strings or complex objects
					hasComplexItems := false
					for _, item := range v {
						if str, ok := item.(string); ok && isGoMapString(str) {
							hasComplexItems = true
							break
						}
						if _, isMap := item.(map[string]interface{}); isMap {
							hasComplexItems = true
							break
						}
						if _, isArray := item.([]interface{}); isArray {
							hasComplexItems = true
							break
						}
					}

					if hasComplexItems {
						// Use multiline format for complex items
						result := bracketColor.Render("[\n")
						for i, item := range v {
							isLast := i == len(v)-1
							indent := fieldIndent + "  "

							var itemStr string
							switch av := item.(type) {
							case string:
								if isGoMapString(av) {
									// Handle Go map string representation
									if parsedMap, err := parseGoMapString(av); err == nil {
										// Convert parsed map to JSON and format it
										if jsonBytes, err := json.Marshal(parsedMap); err == nil {
											itemStr = formatObjectRecursive(string(jsonBytes), nestLevel+2)
										} else {
											itemStr = stringColor.Render(fmt.Sprintf("\"%s\"", av))
										}
									} else {
										itemStr = stringColor.Render(fmt.Sprintf("\"%s\"", av))
									}
								} else {
									itemStr = stringColor.Render(fmt.Sprintf("\"%s\"", av))
								}
							case float64:
								if math.Mod(av, 1) == 0 {
									itemStr = numberColor.Render(fmt.Sprintf("%.0f", av))
								} else {
									itemStr = numberColor.Render(fmt.Sprintf("%.2f", av))
								}
							case bool:
								itemStr = boolColor.Render(fmt.Sprintf("%t", av))
							case nil:
								itemStr = nullColor.Render("null")
							case map[string]interface{}:
								// Format nested objects
								if jsonBytes, err := json.Marshal(av); err == nil {
									itemStr = formatObjectRecursive(string(jsonBytes), nestLevel+2)
								} else {
									itemStr = stringColor.Render(fmt.Sprintf("\"%v\"", av))
								}
							default:
								itemStr = stringColor.Render(fmt.Sprintf("\"%v\"", av))
							}

							comma := ""
							if !isLast {
								comma = bracketColor.Render(",")
							}
							result += indent + itemStr + comma + "\n"
						}
						result += fieldIndent + bracketColor.Render("]")
						valuePart = result
					} else {
						// Use inline format for simple items
						var arrayItems []string
						for _, item := range v {
							switch av := item.(type) {
							case string:
								arrayItems = append(arrayItems, stringColor.Render(fmt.Sprintf("\"%s\"", av)))
							case float64:
								if math.Mod(av, 1) == 0 {
									arrayItems = append(arrayItems, numberColor.Render(fmt.Sprintf("%.0f", av)))
								} else {
									arrayItems = append(arrayItems, numberColor.Render(fmt.Sprintf("%.2f", av)))
								}
							case bool:
								arrayItems = append(arrayItems, boolColor.Render(fmt.Sprintf("%t", av)))
							case nil:
								arrayItems = append(arrayItems, nullColor.Render("null"))
							default:
								arrayItems = append(arrayItems, stringColor.Render(fmt.Sprintf("\"%v\"", av)))
							}
						}
						valuePart = bracketColor.Render("[") + strings.Join(arrayItems, bracketColor.Render(", ")) + bracketColor.Render("]")
					}
				}
			default:
				valuePart = stringColor.Render(fmt.Sprintf("\"%v\"", v))
			}

			comma := ""
			if !isLast {
				comma = bracketColor.Render(",")
			}

			details.WriteString(fieldIndent + keyPart + bracketColor.Render(": ") + valuePart + comma + "\n")
		}

		details.WriteString(baseIndent + bracketColor.Render("}"))
		return details.String()
	}

	// Helper function to format JSON array
	formatArray := func(arrayStr string) string {
		var items []interface{}
		if err := json.Unmarshal([]byte(arrayStr), &items); err != nil {
			// If parsing fails, treat as regular string
			return stringColor.Render(fmt.Sprintf("\"%s\"", arrayStr))
		}

		if len(items) == 0 {
			return bracketColor.Render("[]")
		}

		// Check if any items are complex objects (maps or arrays)
		hasComplexItems := false
		for _, item := range items {
			if _, isMap := item.(map[string]interface{}); isMap {
				hasComplexItems = true
				break
			}
			if _, isArray := item.([]interface{}); isArray {
				hasComplexItems = true
				break
			}
		}

		// Force multiline format for arrays containing complex objects, or for better readability of all arrays
		if hasComplexItems || len(items) > 0 {
			// Multiline format
			result := bracketColor.Render("[\n")
			for i, item := range items {
				isLast := i == len(items)-1
				indent := leftMargin + "    "

				var itemStr string
				switch v := item.(type) {
				case string:
					if isGoMapString(v) {
						// Handle Go map string representation in arrays
						if parsedMap, err := parseGoMapString(v); err == nil {
							// Convert parsed map to JSON and format it
							if jsonBytes, err := json.Marshal(parsedMap); err == nil {
								itemStr = formatObjectRecursive(string(jsonBytes), 2)
							} else {
								itemStr = stringColor.Render(fmt.Sprintf("\"%s\"", v))
							}
						} else {
							itemStr = stringColor.Render(fmt.Sprintf("\"%s\"", v))
						}
					} else {
						itemStr = stringColor.Render(fmt.Sprintf("\"%s\"", v))
					}
				case float64:
					if math.Mod(v, 1) == 0 {
						itemStr = numberColor.Render(fmt.Sprintf("%.0f", v))
					} else {
						itemStr = numberColor.Render(fmt.Sprintf("%.2f", v))
					}
				case bool:
					itemStr = boolColor.Render(fmt.Sprintf("%t", v))
				case nil:
					itemStr = nullColor.Render("null")
				case map[string]interface{}:
					// Format nested map as JSON object with proper indentation
					if len(v) == 0 {
						itemStr = bracketColor.Render("{}")
					} else {
						// Convert map to JSON string and format with increased nesting level
						if jsonBytes, err := json.Marshal(v); err == nil {
							// Use recursive formatObjectRecursive call with increased nesting level
							itemStr = formatObjectRecursive(string(jsonBytes), 2)
						} else {
							itemStr = stringColor.Render(fmt.Sprintf("\"%v\"", v))
						}
					}
				case []interface{}:
					// Format nested array inline for simplicity to avoid recursion
					if len(v) == 0 {
						itemStr = bracketColor.Render("[]")
					} else {
						var nestedArrayItems []string
						for _, nestedItem := range v {
							switch nav := nestedItem.(type) {
							case string:
								nestedArrayItems = append(nestedArrayItems, stringColor.Render(fmt.Sprintf("\"%s\"", nav)))
							case float64:
								if math.Mod(nav, 1) == 0 {
									nestedArrayItems = append(nestedArrayItems, numberColor.Render(fmt.Sprintf("%.0f", nav)))
								} else {
									nestedArrayItems = append(nestedArrayItems, numberColor.Render(fmt.Sprintf("%.2f", nav)))
								}
							case bool:
								nestedArrayItems = append(nestedArrayItems, boolColor.Render(fmt.Sprintf("%t", nav)))
							case nil:
								nestedArrayItems = append(nestedArrayItems, nullColor.Render("null"))
							default:
								nestedArrayItems = append(nestedArrayItems, stringColor.Render(fmt.Sprintf("\"%v\"", nav)))
							}
						}
						itemStr = bracketColor.Render("[") + strings.Join(nestedArrayItems, bracketColor.Render(", ")) + bracketColor.Render("]")
					}
				default:
					itemStr = stringColor.Render(fmt.Sprintf("\"%v\"", v))
				}

				comma := ""
				if !isLast {
					comma = bracketColor.Render(",")
				}
				result += indent + itemStr + comma + "\n"
			}
			result += leftMargin + "  " + bracketColor.Render("]")
			return result
		} else {
			// Inline format for simple short arrays
			var formattedItems []string
			for _, item := range items {
				switch v := item.(type) {
				case string:
					if isGoMapString(v) {
						// Handle Go map string representation in inline arrays
						if parsedMap, err := parseGoMapString(v); err == nil {
							// For inline format, just show a compact representation
							var compactFields []string
							for key, value := range parsedMap {
								compactFields = append(compactFields, fmt.Sprintf("%s: %v", key, value))
							}
							compactObj := fmt.Sprintf("{%s}", strings.Join(compactFields, ", "))
							formattedItems = append(formattedItems, stringColor.Render(compactObj))
						} else {
							formattedItems = append(formattedItems, stringColor.Render(fmt.Sprintf("\"%s\"", v)))
						}
					} else {
						formattedItems = append(formattedItems, stringColor.Render(fmt.Sprintf("\"%s\"", v)))
					}
				case float64:
					if math.Mod(v, 1) == 0 {
						formattedItems = append(formattedItems, numberColor.Render(fmt.Sprintf("%.0f", v)))
					} else {
						formattedItems = append(formattedItems, numberColor.Render(fmt.Sprintf("%.2f", v)))
					}
				case bool:
					formattedItems = append(formattedItems, boolColor.Render(fmt.Sprintf("%t", v)))
				case nil:
					formattedItems = append(formattedItems, nullColor.Render("null"))
				default:
					formattedItems = append(formattedItems, stringColor.Render(fmt.Sprintf("\"%v\"", v)))
				}
			}
			return bracketColor.Render("[") + strings.Join(formattedItems, bracketColor.Render(", ")) + bracketColor.Render("]")
		}
	}

	// Helper function to add a field
	addField := func(key string, value interface{}, isLast bool) {
		indent := leftMargin
		keyStr := keyColor.Render(fmt.Sprintf("\"%s\"", key))
		colon := bracketColor.Render(": ")

		var valueStr string
		switch v := value.(type) {
		case string:
			if v == "" {
				valueStr = nullColor.Render("null")
			} else if isJSONArray(v) {
				// Handle JSON array strings
				valueStr = formatArray(v)
			} else if isJSONObject(v) {
				// Handle JSON object strings
				valueStr = formatObjectRecursive(v, 1)
			} else if isGoMapString(v) {
				// Handle Go map string representation (e.g., "map[key1:value1 key2:value2]")
				if parsedMap, err := parseGoMapString(v); err == nil {
					// Convert parsed map to JSON and format it recursively
					if jsonBytes, err := json.Marshal(parsedMap); err == nil {
						valueStr = formatObjectRecursive(string(jsonBytes), 1)
					} else {
						valueStr = stringColor.Render(fmt.Sprintf("\"%s\"", v))
					}
				} else {
					valueStr = stringColor.Render(fmt.Sprintf("\"%s\"", v))
				}
			} else {
				valueStr = stringColor.Render(fmt.Sprintf("\"%s\"", v))
			}
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			valueStr = numberColor.Render(fmt.Sprintf("%v", v))
		case float32:
			// Check if it's actually an integer value
			if math.Mod(float64(v), 1) == 0 {
				valueStr = numberColor.Render(fmt.Sprintf("%.0f", v))
			} else {
				valueStr = numberColor.Render(fmt.Sprintf("%.2f", v))
			}
		case float64:
			// Check if it's actually an integer value
			if math.Mod(v, 1) == 0 {
				valueStr = numberColor.Render(fmt.Sprintf("%.0f", v))
			} else {
				valueStr = numberColor.Render(fmt.Sprintf("%.2f", v))
			}
		case bool:
			valueStr = boolColor.Render(fmt.Sprintf("%t", v))
		case nil:
			valueStr = nullColor.Render("null")
		case map[string]interface{}:
			// Handle nested maps properly
			if len(v) == 0 {
				valueStr = bracketColor.Render("{}")
			} else {
				// Convert map to JSON string and format it
				if jsonBytes, err := json.Marshal(v); err == nil {
					valueStr = formatObjectRecursive(string(jsonBytes), 1)
				} else {
					valueStr = stringColor.Render(fmt.Sprintf("\"%v\"", v))
				}
			}
		case []interface{}:
			// Handle nested arrays properly
			if len(v) == 0 {
				valueStr = bracketColor.Render("[]")
			} else {
				// Convert array to JSON string and format it
				if jsonBytes, err := json.Marshal(v); err == nil {
					valueStr = formatArray(string(jsonBytes))
				} else {
					valueStr = stringColor.Render(fmt.Sprintf("\"%v\"", v))
				}
			}
		default:
			// For any other type, convert to string
			valueStr = stringColor.Render(fmt.Sprintf("\"%v\"", v))
		}

		comma := ""
		if !isLast {
			comma = bracketColor.Render(",")
		}

		details.WriteString(fmt.Sprintf("%s%s%s%s%s\n", indent, keyStr, colon, valueStr, comma))
	}

	// Get all keys and sort them for consistent output
	keys := make([]string, 0, len(record))
	for k := range record {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Render all fields
	for i, key := range keys {
		isLast := i == len(keys)-1
		addField(key, record[key], isLast)
	}

	// Close JSON object with left margin
	details.WriteString(leftMargin + bracketColor.Render("}"))

	return details.String()
}

// Minimal formatting helpers to avoid import cycles
func adaptersFormatRecordAsJSON(record map[string]any) string {
	if record == nil {
		return "{}"
	}
	var b strings.Builder
	b.WriteString("{ ")
	i := 0
	for k, v := range record {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf("\"%s\": %v", k, v))
		i++
	}
	b.WriteString(" }")
	return b.String()
}

// formatRecordSetAsJSON formats a RecordSet (array of records) as a JSON array with syntax highlighting
func formatRecordSetAsJSON(recordSet vast_client.RecordSet) string {
	if len(recordSet) == 0 {
		return "[]"
	}

	var result strings.Builder

	// Define colors for syntax highlighting (same as formatRecordAsJSON)
	bracketColor := lipgloss.NewStyle().Foreground(colors.VeryLightGrey) // Light white for brackets/punctuation

	// Left margin (2 spaces)
	leftMargin := "  "

	// Start JSON array
	result.WriteString(leftMargin + bracketColor.Render("[\n"))

	// Format each record
	for i, record := range recordSet {
		// Convert Record to map[string]any
		recordMap := map[string]any(record)

		// Format the record (but we need to indent it properly)
		formattedRecord := formatRecordAsJSON(recordMap)

		// Add indentation to each line of the formatted record (except the first margin)
		lines := strings.Split(formattedRecord, "\n")
		for j, line := range lines {
			if j == 0 {
				// First line - already has left margin, just add array element indent
				result.WriteString("  " + strings.TrimPrefix(line, "  ") + "\n")
			} else {
				// Other lines - add array element indent
				result.WriteString("  " + line + "\n")
			}
		}

		// Add comma if not the last element
		if i < len(recordSet)-1 {
			// Remove the last closing brace line, add comma, then add it back
			resultStr := result.String()
			lastBraceIdx := strings.LastIndex(resultStr, "}")
			if lastBraceIdx != -1 {
				result.Reset()
				result.WriteString(resultStr[:lastBraceIdx])
				result.WriteString(bracketColor.Render("},\n"))
			}
		}
	}

	// End JSON array
	result.WriteString(leftMargin + bracketColor.Render("]"))

	return result.String()
}

func adaptersRecordPretty(record map[string]any) string {
	return adaptersFormatRecordAsJSON(record)
}
