package tui

import (
	"regexp"
	"strings"
	"vastix/internal/database"
	log "vastix/internal/logging"

	"github.com/charmbracelet/lipgloss"
	"go.uber.org/zap"
)

// StatusZone represents the status display zone for errors, info messages and spinner
type StatusZone struct {
	width, height int
	errorMsg      string
	infoMsg       string
	spinnerMsg    string
	db            *database.Service // Database service
}

// NewStatusZone creates a new status zone
func NewStatusZone(db *database.Service) *StatusZone {
	log.Debug("StatusZone initializing")

	statusZone := &StatusZone{
		db: db,
	}

	log.Debug("StatusZone initialized successfully")

	return statusZone
}

func (s *StatusZone) Init() {}

// SetSize sets the dimensions of the status zone
func (s *StatusZone) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// SetError sets the error message to display
func (s *StatusZone) SetError(msg string) {
	s.errorMsg = msg
	log.Debug("Error message set", zap.String("message", msg))
}

// SetInfo sets the info message to display
func (s *StatusZone) SetInfo(msg string) {
	s.infoMsg = msg
	log.Debug("Info message set", zap.String("message", msg))
}

// SetSpinner sets the spinner message to display
func (s *StatusZone) SetSpinner(msg string) {
	s.spinnerMsg = msg
	log.Debug("Spinner message set", zap.String("message", msg))
}

// Clear clears all messages (error, info, and spinner)
func (s *StatusZone) Clear() {
	s.errorMsg = ""
	s.infoMsg = ""
	s.spinnerMsg = ""
	log.Debug("Status messages cleared")
}

// ClearError clears only the error message
func (s *StatusZone) ClearError() {
	s.errorMsg = ""
	log.Debug("Error message cleared")
}

// ClearInfo clears only the info message
func (s *StatusZone) ClearInfo() {
	s.infoMsg = ""
	log.Debug("Info message cleared")
}

// ClearSpinner clears only the spinner message
func (s *StatusZone) ClearSpinner() {
	s.spinnerMsg = ""
	log.Debug("Spinner message cleared")
}

// HasError returns true if there's an error message to display
func (s *StatusZone) HasError() bool {
	return s.errorMsg != ""
}

// HasInfo returns true if there's an info message to display
func (s *StatusZone) HasInfo() bool {
	return s.infoMsg != ""
}

// HasSpinner returns true if there's a spinner message to display
func (s *StatusZone) HasSpinner() bool {
	return s.spinnerMsg != ""
}

// GetHeight returns the height the status zone needs when visible
func (s *StatusZone) GetHeight() int {
	// Priority: error first, then info, then spinner
	if s.errorMsg != "" {
		// For errors, calculate height based on text wrapping
		message := s.errorMsg
		contentWidth := s.width - 2 // Leave space for borders
		if contentWidth < 1 {
			contentWidth = 1
		}

		// Split the message into words and calculate required lines
		words := strings.Fields(message)
		if len(words) == 0 {
			return 4 // Minimum height for empty message (corners + content + padding)
		}

		lines := 1
		currentLineLength := 0

		for _, word := range words {
			wordLength := len(word)
			// If adding this word would exceed the line, start a new line
			if currentLineLength > 0 && currentLineLength+1+wordLength > contentWidth {
				lines++
				currentLineLength = wordLength
			} else {
				if currentLineLength > 0 {
					currentLineLength++ // Add space
				}
				currentLineLength += wordLength
			}
		}

		return lines + 3 // +3 for top border, bottom border, and padding line

	} else if s.infoMsg != "" {
		// For info messages, calculate height based on text wrapping
		message := s.infoMsg
		contentWidth := s.width - 2 // Leave space for borders
		if contentWidth < 1 {
			contentWidth = 1
		}

		// Split the message into words and calculate required lines
		words := strings.Fields(message)
		if len(words) == 0 {
			return 4 // Minimum height for empty message (corners + content + padding)
		}

		lines := 1
		currentLineLength := 0

		for _, word := range words {
			wordLength := len(word)
			// If adding this word would exceed the line, start a new line
			if currentLineLength > 0 && currentLineLength+1+wordLength > contentWidth {
				lines++
				currentLineLength = wordLength
			} else {
				if currentLineLength > 0 {
					currentLineLength++ // Add space
				}
				currentLineLength += wordLength
			}
		}

		return lines + 3 // +3 for top border, bottom border, and padding line

	} else if s.spinnerMsg != "" {
		// For spinner, height is fixed since it's typically a single line
		return 4 // Top border + spinner line + padding line + bottom border
	}

	return 0 // Nothing to display
}

// Ready returns whether the status zone is ready to be displayed
func (s *StatusZone) Ready() bool {
	return true // Status zone is always ready
}

// View renders the status zone
func (s *StatusZone) View() string {
	// Priority: error first, then info, then spinner
	if s.errorMsg != "" {
		return s.renderErrorMsg()
	} else if s.infoMsg != "" {
		return s.renderInfoMsg()
	} else if s.spinnerMsg != "" {
		return s.renderSpinnerMsg()
	}
	return ""
}

// renderErrorMsg renders an error message with corner borders and spaces.
// The border and error message are colored red.
func (s *StatusZone) renderErrorMsg() string {
	if s.errorMsg == "" {
		return ""
	}

	// Create red style for both border and message
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5353"))

	return s.renderMessage(s.errorMsg, redStyle)
}

// renderInfoMsg renders an info message with corner borders and spaces.
// The border and info message are colored green.
func (s *StatusZone) renderInfoMsg() string {
	if s.infoMsg == "" {
		return ""
	}

	// Create green style for both border and message
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))

	return s.renderMessage(s.infoMsg, greenStyle)
}

// renderSpinnerMsg renders a spinner message with corner borders and gradient colors.
func (s *StatusZone) renderSpinnerMsg() string {
	if s.spinnerMsg == "" {
		return ""
	}

	log.Debug("Rendering spinner in status zone", zap.String("spinner_content", s.spinnerMsg))

	// Calculate content width - leave space for vertical borders on each side
	contentWidth := s.width - 2 // 1 char for ┃ + 1 char for ┃

	if contentWidth < 1 {
		contentWidth = 1
	}

	// Create gradient style for borders only (spinner content already has gradient)
	gradientColors := []string{
		"#00FFFF", // Cyan
		"#40E0D0", // Turquoise
		"#8A2BE2", // Blue Violet
		"#FF69B4", // Hot Pink
		"#FF1493", // Deep Pink
	}

	// Use a color based on the spinner message content to create variation
	colorIndex := len(s.spinnerMsg) % len(gradientColors)
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(gradientColors[colorIndex]))

	// For spinner, we don't need text wrapping since it's typically a single short string
	// But we still need to handle it like renderMessage for consistency
	msgLines := []string{s.spinnerMsg} // Spinner is typically one line

	// Build the block with corner borders (spaces between corners)
	var lines []string

	// Add top corner line with spaces (thick border style)
	topLine := "┏" + strings.Repeat(" ", contentWidth) + "┓"
	lines = append(lines, borderStyle.Render(topLine))

	// Add spinner lines with centered text (thick border style)
	for _, line := range msgLines {
		// For spinner content, we need to handle ANSI escape codes in length calculation
		centeredLine := s.centerTextWithANSI(line, contentWidth)
		// Only style the borders, not the spinner content itself
		styledBorders := borderStyle.Render("┃") + centeredLine + borderStyle.Render("┃")
		lines = append(lines, styledBorders)
	}

	// Add one empty line for padding (thick border style)
	emptyLine := "┃" + strings.Repeat(" ", contentWidth) + "┃"
	lines = append(lines, borderStyle.Render(emptyLine))

	// Add bottom corner line with spaces (thick border style)
	bottomLine := "┗" + strings.Repeat(" ", contentWidth) + "┛"
	lines = append(lines, borderStyle.Render(bottomLine))

	// Join all lines
	block := strings.Join(lines, "\n")

	// Return the block - no additional width styling to avoid alignment issues
	return block
}

// renderMessage renders a message with the given style using corner borders
func (s *StatusZone) renderMessage(message string, style lipgloss.Style) string {
	// Calculate content width - leave space for vertical borders on each side
	contentWidth := s.width - 2 // 1 char for ┃ + 1 char for ┃

	if contentWidth < 1 {
		contentWidth = 1
	}

	// Wrap the message to fit within content width
	wrappedMsg := s.wrapText(message, contentWidth)
	msgLines := strings.Split(wrappedMsg, "\n")

	// Build the block with corner borders (spaces between corners)
	var lines []string

	// Add top corner line with spaces (thick border style)
	topLine := "┏" + strings.Repeat(" ", contentWidth) + "┓"
	lines = append(lines, style.Render(topLine))

	// Add message lines with centered text (thick border style)
	for _, line := range msgLines {
		// Center the text within the content width
		centeredLine := s.centerText(line, contentWidth)
		msgLine := "┃" + centeredLine + "┃"
		lines = append(lines, style.Render(msgLine))
	}

	// Add one empty line for padding (thick border style)
	emptyLine := "┃" + strings.Repeat(" ", contentWidth) + "┃"
	lines = append(lines, style.Render(emptyLine))

	// Add bottom corner line with spaces (thick border style)
	bottomLine := "┗" + strings.Repeat(" ", contentWidth) + "┛"
	lines = append(lines, style.Render(bottomLine))

	// Join all lines
	block := strings.Join(lines, "\n")

	// Return the block - no additional width styling to avoid alignment issues
	return block
}

// centerText centers text within the given width
func (s *StatusZone) centerText(text string, width int) string {
	textLen := lipgloss.Width(text)
	if textLen >= width {
		if width <= 3 {
			return strings.Repeat(".", width)
		}
		// Truncate and preserve balance (approximate, since Width may differ from byte indices)
		leftWidth := (width - 1) / 2
		rightWidth := width - 1 - leftWidth

		// This part is tricky because slicing by visual width requires rune-based logic,
		// not just slicing by bytes or characters — for full correctness, you'd need
		// to walk runes by visual width.
		runes := []rune(text)
		return string(runes[:leftWidth]) + "…" + string(runes[len(runes)-rightWidth:])
	}

	leftPadding := (width - textLen) / 2
	rightPadding := width - textLen - leftPadding

	return strings.Repeat(" ", leftPadding) + text + strings.Repeat(" ", rightPadding)
}

// centerTextWithANSI centers text within the given width, handling ANSI escape codes.
func (s *StatusZone) centerTextWithANSI(text string, width int) string {
	// Strip ANSI escape codes for accurate length calculation
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	cleanText := ansiRegex.ReplaceAllString(text, "")

	cleanTextLen := len(cleanText)
	if cleanTextLen >= width {
		// If text is too long, truncate from the middle to keep it centered
		if width <= 3 {
			return strings.Repeat(".", width) // Just dots if too narrow
		}
		// For ANSI text, truncation is complex, so let's just use the original text
		// This might cause alignment issues but preserves colors
		return text
	}

	leftPadding := (width - cleanTextLen) / 2
	rightPadding := width - cleanTextLen - leftPadding

	return strings.Repeat(" ", leftPadding) + text + strings.Repeat(" ", rightPadding)
}

// wrapText wraps text to fit within the specified width
func (s *StatusZone) wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var lines []string
	var currentLine []string
	currentLength := 0

	for _, word := range words {
		wordLength := len(word)

		// If adding this word would exceed the width, start a new line
		if currentLength > 0 && currentLength+1+wordLength > width {
			lines = append(lines, strings.Join(currentLine, " "))
			currentLine = []string{word}
			currentLength = wordLength
		} else {
			currentLine = append(currentLine, word)
			if currentLength > 0 {
				currentLength++ // Add space
			}
			currentLength += wordLength
		}
	}

	// Add the last line
	if len(currentLine) > 0 {
		lines = append(lines, strings.Join(currentLine, " "))
	}

	return strings.Join(lines, "\n")
}
