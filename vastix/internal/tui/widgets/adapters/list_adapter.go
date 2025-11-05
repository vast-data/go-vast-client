package adapters

import (
	"vastix/internal/colors"
	"fmt"
	"sort"
	"strings"
	"vastix/internal/msg_types"
	"vastix/internal/tui/widgets/common"

	"vastix/internal/database"
	log "vastix/internal/logging"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/lipgloss"
)

// BorderPosition moved to common package

// BorderPosition and borderize functionality moved to common package

// Colors - use the same colors as the main TUI package
var (
	Blue      = colors.Blue      // Main TUI Blue for active state
	White     = colors.White // Same as main TUI White
	Black     = colors.Black // Same as main TUI Black
	LightGrey = colors.LightGrey     // Same as main TUI LightGrey
	DarkGrey  = colors.DarkGrey // Same as main TUI DarkGrey
	Yellow    = colors.Yellow // Same as main TUI Yellow
)

type ListAdapter struct {
	resourceType    string // Type of resource this adapter represents, e.g., "views", "quotas", "users" etc.
	predefinedTitle string // Optional predefined title to override default

	// Data management
	headers      []string // Column headers
	data         [][]string
	filteredData [][]string

	// Scrolling functionality
	visibleStartRow int
	maxVisibleRows  int
	selectedRow     int

	// Column width configuration
	narrowColumns map[string]bool // Columns that should have limited width (e.g., ID fields)

	// Database connection
	db *database.Service
}

// GetHeaders returns the configured column headers (uppercased)
func (lr *ListAdapter) GetHeaders() []string {
	return lr.headers
}

// GetData returns the current data
func (lr *ListAdapter) GetData() [][]string {
	return lr.data
}

// GetFilteredDataCount returns the number of items in the filtered data (visible after fuzzy search)
func (lr *ListAdapter) GetFilteredDataCount() int {
	return len(lr.filteredData)
}

// NewListAdapter creates a new list adapter
func NewListAdapter(db *database.Service, resourceType string, headers []string) *ListAdapter {
	log.Debug("ListAdapter initializing")

	adapter := &ListAdapter{
		resourceType:    resourceType,
		headers:         common.ToUpperSlice(headers),
		data:            make([][]string, 0),
		filteredData:    make([][]string, 0),
		visibleStartRow: 0,
		maxVisibleRows:  0,
		selectedRow:     0,
		narrowColumns:   make(map[string]bool),
		db:              db,
	}

	// Set default narrow columns for common ID fields
	adapter.SetNarrowColumns([]string{"ID", "TENANT_ID", "USER_ID", "VAST", "METHOD"})

	log.Debug("ListAdapter initialized successfully")

	return adapter
}

// NewListAdapterWithPredefinedTitle creates a new list adapter with a predefined title
func NewListAdapterWithPredefinedTitle(db *database.Service, resourceType string, headers []string, title string) *ListAdapter {
	adapter := NewListAdapter(db, resourceType, headers)
	adapter.predefinedTitle = title
	return adapter
}

// SetPredefinedTitle allows setting the predefined title dynamically
func (lr *ListAdapter) SetPredefinedTitle(title string) {
	lr.predefinedTitle = title
}

func (lr *ListAdapter) ClearListData() {
	lr.data = make([][]string, 0)
	lr.filteredData = make([][]string, 0)
}

// SetNarrowColumns configures which columns should have limited width
func (lr *ListAdapter) SetNarrowColumns(columnNames []string) {
	lr.narrowColumns = make(map[string]bool)
	for _, col := range columnNames {
		lr.narrowColumns[strings.ToUpper(col)] = true
	}
}

// calculateColumnWidths calculates individual column widths based on preferences
func (lr *ListAdapter) calculateColumnWidths(totalWidth int) []int {
	if len(lr.headers) == 0 {
		return []int{}
	}

	widths := make([]int, len(lr.headers))
	narrowWidth := 10 // Fixed width for narrow columns (ID fields)

	// Count narrow and regular columns
	narrowCount := 0
	for _, header := range lr.headers {
		if lr.narrowColumns[header] {
			narrowCount++
		}
	}

	regularCount := len(lr.headers) - narrowCount

	// Calculate remaining width for regular columns
	usedWidth := narrowCount * narrowWidth
	remainingWidth := totalWidth - usedWidth

	var regularWidth int
	if regularCount > 0 {
		regularWidth = remainingWidth / regularCount
	} else {
		regularWidth = narrowWidth // Fallback if all columns are narrow
	}

	// Assign widths
	for i, header := range lr.headers {
		if lr.narrowColumns[header] {
			widths[i] = narrowWidth
		} else {
			widths[i] = regularWidth
		}
	}

	return widths
}

func (lr *ListAdapter) SelectDo(w common.DetailsWidget) tea.Cmd {
	selectedRowData := lr.GetSelectedRowData()
	return msg_types.ProcessWithSpinnerMust(w.Select(selectedRowData))
}

func (lr *ListAdapter) DeleteDo(w common.DeleteWidget) tea.Cmd {
	selectedRowData := lr.GetSelectedRowData()
	return msg_types.ProcessWithSpinnerMust(w.Delete(selectedRowData))
}

// DetailsDo calls the Details method on widgets that support it
func (lr *ListAdapter) DetailsDo(w common.DetailsWidget) tea.Cmd {
	selectedRowData := lr.GetSelectedRowData()
	return msg_types.ProcessWithSpinnerMust(w.Details(selectedRowData))
}

// GetSelectedRowData returns the data of the currently selected row as RowData
func (lr *ListAdapter) GetSelectedRowData() common.RowData {
	if lr.selectedRow >= 0 && lr.selectedRow < len(lr.filteredData) {
		row := lr.filteredData[lr.selectedRow]
		return common.NewRowData(lr.headers, row)
	}
	return common.NewRowData(lr.headers, []string{})
}

// SetListData sets the data for the renderer and applies fuzzy search filter if provided
func (lr *ListAdapter) SetListData(data [][]string, fuzzySearchQuery string) {
	lr.data = data
	if fuzzySearchQuery != "" {
		// If a fuzzy search is active, filter the data immediately
		lr.filteredData = lr.fuzzyFilter(data, fuzzySearchQuery)
	} else {
		// No filter applied, use original data
		lr.filteredData = data
	}

	// Adjust selectedRow if it's out of bounds after data update
	if len(lr.filteredData) == 0 {
		// No data, reset to beginning
		lr.selectedRow = 0
		lr.visibleStartRow = 0
	} else if lr.selectedRow >= len(lr.filteredData) {
		// Selected row is out of bounds, move to last valid row
		lr.selectedRow = len(lr.filteredData) - 1
		// Ensure the selected row is visible
		if lr.maxVisibleRows > 0 && lr.selectedRow >= lr.visibleStartRow+lr.maxVisibleRows {
			lr.visibleStartRow = max(0, lr.selectedRow-lr.maxVisibleRows+1)
		}
	}
}

// Navigation methods
func (lr *ListAdapter) MoveUp() {
	if lr.selectedRow > 0 {
		lr.selectedRow--
		// Scroll up if selected row is above visible area
		if lr.selectedRow < lr.visibleStartRow {
			lr.visibleStartRow = lr.selectedRow
		}
	}
}

func (lr *ListAdapter) MoveDown() {
	if lr.selectedRow < len(lr.filteredData)-1 {
		lr.selectedRow++
		// Scroll down if selected row is below visible area
		if lr.selectedRow >= lr.visibleStartRow+lr.maxVisibleRows {
			lr.visibleStartRow = lr.selectedRow - lr.maxVisibleRows + 1
		}
	}
}

func (lr *ListAdapter) MoveHome() {
	lr.selectedRow = 0
	lr.visibleStartRow = 0
}

func (lr *ListAdapter) MoveEnd() {
	lr.selectedRow = len(lr.filteredData) - 1
	lr.visibleStartRow = max(0, len(lr.filteredData)-lr.maxVisibleRows)
}

func (lr *ListAdapter) PageUp() {
	lr.selectedRow = max(0, lr.selectedRow-lr.maxVisibleRows)
	lr.visibleStartRow = max(0, lr.visibleStartRow-lr.maxVisibleRows)
}

func (lr *ListAdapter) PageDown() {
	lr.selectedRow = min(len(lr.filteredData)-1, lr.selectedRow+lr.maxVisibleRows)
	lr.visibleStartRow = min(max(0, len(lr.filteredData)-lr.maxVisibleRows), lr.visibleStartRow+lr.maxVisibleRows)
}

// ViewList renders the list view using the provided widget
func (lr *ListAdapter) ViewList(widget common.Widget) string {
	width := widget.GetWidth()
	height := widget.GetHeight()

	if width == 0 || height == 0 {
		return ""
	}

	// Get headers from widget
	headers := lr.headers

	// Calculate available space inside the border
	innerWidth := width - 2
	if innerWidth < 0 {
		innerWidth = width
	}

	// Calculate custom column widths
	columnWidths := lr.calculateColumnWidths(innerWidth)

	// Create styles for each column width (we'll apply them per column)
	headerStyles := make([]lipgloss.Style, len(headers))
	normalRowStyles := make([]lipgloss.Style, len(headers))
	selectedRowStyles := make([]lipgloss.Style, len(headers))

	for i, width := range columnWidths {
		headerStyles[i] = lipgloss.NewStyle().
			Foreground(Blue).
			Bold(true).
			Width(width).
			Align(lipgloss.Left)

		normalRowStyles[i] = lipgloss.NewStyle().
			Foreground(White).
			Width(width).
			Align(lipgloss.Left)

		selectedRowStyles[i] = lipgloss.NewStyle().
			Foreground(Black).
			Background(Blue).
			Width(width).
			Align(lipgloss.Left)
	}

	// Calculate maximum visible rows based on available height
	lr.maxVisibleRows = height - 4 // header + separator + 2 borders
	if lr.maxVisibleRows < 1 {
		lr.maxVisibleRows = 1
	}

	// Ensure visibleStartRow is within bounds
	if lr.visibleStartRow > len(lr.filteredData)-1 {
		lr.visibleStartRow = max(0, len(lr.filteredData)-lr.maxVisibleRows)
	}

	// Build list content
	var rows []string

	// Header row
	var headerCells []string
	for i, header := range headers {
		headerCells = append(headerCells, headerStyles[i].Render(header))
	}
	headerRow := lipgloss.JoinHorizontal(lipgloss.Top, headerCells...)
	headerRow = lipgloss.NewStyle().Width(innerWidth).Render(headerRow)
	rows = append(rows, headerRow)

	// Separator
	separatorStyle := lipgloss.NewStyle().Foreground(LightGrey)
	separator := separatorStyle.Render(strings.Repeat("─", innerWidth))
	rows = append(rows, separator)

	// Check if there's no data to display
	if len(lr.filteredData) == 0 {
		// Show "No content" message with same styling as details adapter
		grayStyle := lipgloss.NewStyle().Foreground(colors.LightGrey) // Light gray color
		paddingStyle := lipgloss.NewStyle().
			Padding(0, 0, 0, 2) // top: 0, right: 0, bottom: 0, left: 2

		noContentMessage := grayStyle.Render("No content")
		noContentRow := paddingStyle.Render(noContentMessage)
		noContentRow = lipgloss.NewStyle().Width(innerWidth).Render(noContentRow)
		rows = append(rows, noContentRow)

		// Fill remaining space with empty rows
		emptyRowStyle := lipgloss.NewStyle().Width(innerWidth)
		for i := 1; i < lr.maxVisibleRows; i++ { // Start from 1 since we already added the "No content" row
			rows = append(rows, emptyRowStyle.Render(""))
		}
	} else {
		// Calculate visible row range
		visibleEndRow := min(lr.visibleStartRow+lr.maxVisibleRows, len(lr.filteredData))

		// Data rows (only visible ones)
		for i := lr.visibleStartRow; i < visibleEndRow; i++ {
			var cells []string
			isSelected := i == lr.selectedRow

			row := lr.filteredData[i]

			// Check if widget implements common.RenderRow interface for custom styling
			// Pass the average column width for backward compatibility
			avgColWidth := innerWidth / len(headers)
			if renderRow, ok := widget.(common.RenderRow); ok {
				// Convert []string row to RowData for RenderRow method
				rowData := common.NewRowData(lr.headers, row)
				row = renderRow.RenderRow(rowData, isSelected, avgColWidth)
			}

			for j, cell := range row {
				if j < len(headers) && j < len(columnWidths) {
					// Select appropriate style for this column
					var style lipgloss.Style
					if isSelected {
						style = selectedRowStyles[j]
					} else {
						style = normalRowStyles[j]
					}

					// Truncate cell content to fit column width
					truncatedCell := truncateText(cell, columnWidths[j]-2) // -2 for potential padding

					// Add minimal padding for selected rows for better readability
					if isSelected {
						paddedCell := " " + truncatedCell + " "
						cells = append(cells, style.Render(paddedCell))
					} else {
						cells = append(cells, style.Render(truncatedCell))
					}
				}
			}
			dataRow := lipgloss.JoinHorizontal(lipgloss.Top, cells...)
			dataRow = lipgloss.NewStyle().Width(innerWidth).Render(dataRow)
			rows = append(rows, dataRow)
		}

		// Fill remaining space with empty rows
		rowsRendered := visibleEndRow - lr.visibleStartRow
		emptyRowsNeeded := lr.maxVisibleRows - rowsRendered

		emptyRowStyle := lipgloss.NewStyle().Width(innerWidth)
		for i := 0; i < emptyRowsNeeded; i++ {
			rows = append(rows, emptyRowStyle.Render(""))
		}
	}

	// Join list content
	listContent := strings.Join(rows, "\n")

	// Create resource type label with styling - use same orange as original
	resourceNameStyle := lipgloss.NewStyle().
		Background(colors.Orange). // Orange background (same as original)
		Foreground(colors.BlackTerm)    // Black text

	// Use predefined title if available, otherwise use resourceType
	var titleText string
	if lr.predefinedTitle != "" {
		titleText = lr.predefinedTitle
	} else {
		titleText = lr.resourceType
	}
	resourceTypeLabel := resourceNameStyle.Render(fmt.Sprintf(" %s ", titleText))

	// Add fuzzy search label if active - get from widget and re-filter data in real-time
	if searchableWidget, ok := widget.(common.SearchableWidget); ok {
		fuzzySearch := searchableWidget.GetFuzzyListSearchString()
		if fuzzySearch != "" {
			// Re-filter data based on current fuzzy search string
			lr.filteredData = lr.fuzzyFilter(lr.data, fuzzySearch)

			// Adjust selectedRow if it's out of bounds after filtering
			if len(lr.filteredData) == 0 {
				lr.selectedRow = 0
				lr.visibleStartRow = 0
			} else if lr.selectedRow >= len(lr.filteredData) {
				lr.selectedRow = len(lr.filteredData) - 1
				// Ensure the selected row is visible
				if lr.maxVisibleRows > 0 && lr.selectedRow >= lr.visibleStartRow+lr.maxVisibleRows {
					lr.visibleStartRow = max(0, lr.selectedRow-lr.maxVisibleRows+1)
				}
			}

			labelStyle := lipgloss.NewStyle().
				Background(colors.DarkGreenBlue). // Muted green background
				Foreground(colors.BlackTerm)   // Black text

			label := labelStyle.Render(fmt.Sprintf(" fuzzy-search=%s ", fuzzySearch))
			resourceTypeLabel = fmt.Sprintf("%s %s", resourceTypeLabel, label)
		} else {
			// No fuzzy search active, use original data
			lr.filteredData = lr.data
		}

		// Add server filter label if active - get from widget
		serverSearchParamStr := searchableWidget.GetServerSearchParams()
		if serverSearchParamStr != "" {
			// Parse string into individual key=value pairs and sort them
			var paramStrings []string

			// Split by comma or space to handle multiple parameters
			parts := common.SplitServerParams(serverSearchParamStr)

			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" {
					paramStrings = append(paramStrings, part)
				}
			}

			// Sort the parameter strings
			sort.Strings(paramStrings)

			// Create server labels for each parameter
			serverLabelStyle := lipgloss.NewStyle().
				Background(colors.Orange). // Yellow background
				Foreground(colors.BlackTerm)    // Black text

			for _, paramStr := range paramStrings {
				serverLabel := serverLabelStyle.Render(fmt.Sprintf(" %s ", paramStr))
				resourceTypeLabel = fmt.Sprintf("%s %s", resourceTypeLabel, serverLabel)
			}
		}
	}

	embeddedText := map[common.BorderPosition]string{
		common.TopMiddleBorder: resourceTypeLabel,
	}

	// Only show line count when there's data
	if len(lr.filteredData) > 0 {
		embeddedText[common.BottomRightBorder] = fmt.Sprintf("%d/%d", lr.selectedRow+1, len(lr.filteredData))
	}

	return common.BorderizeWithSpinnerCheck(listContent, true, embeddedText)
}

// fuzzyFilter performs fuzzy matching on list data
func (lr *ListAdapter) fuzzyFilter(data [][]string, query string) [][]string {
	if query == "" {
		return data
	}

	var filtered [][]string
	query = strings.ToLower(query)

	for _, row := range data {
		// Check if any cell in the row matches the query (fuzzy)
		for _, cell := range row {
			if lr.fuzzyMatch(strings.ToLower(cell), query) {
				filtered = append(filtered, row)
				break // Found match in this row, move to next row
			}
		}
	}

	return filtered
}

// fuzzyMatch checks if text contains all characters of query in order (not necessarily consecutive)
func (lr *ListAdapter) fuzzyMatch(text, query string) bool {
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

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// truncateText truncates text to fit within maxWidth, adding ellipsis if needed
func truncateText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	// If text fits, return as-is
	if len(text) <= maxWidth {
		return text
	}

	// If maxWidth is too small for ellipsis, just truncate
	if maxWidth <= 3 {
		return text[:maxWidth]
	}

	// Truncate and add ellipsis
	return text[:maxWidth-3] + "..."
}
