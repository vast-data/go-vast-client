package adapters

import (
	"vastix/internal/colors"
	"strings"

	"vastix/internal/database"

	"github.com/charmbracelet/lipgloss"
)

// Define colors locally to avoid import cycle
var (
	colorOrange    = colors.Orange
	colorBlack     = colors.Black
	colorLightGrey = colors.LightGrey
	colorWhite     = colors.White
	colorYellow    = colors.Yellow // Match key binding color
	colorBlue      = colors.Blue
	colorDarkGrey  = colors.DarkGrey
)

type PromptAdapter struct {
	db              *database.Service
	resourceType    string
	predefinedMsg   string
	predefinedTitle string
	selectedNo      bool // true if "No" button is selected, false if "Yes" is selected (default is Yes)
}

func NewPromptAdapter(db *database.Service, resourceType string) *PromptAdapter {
	return &PromptAdapter{
		db:           db,
		resourceType: resourceType,
		selectedNo:   false, // Default to "Yes" (selectedNo = false means Yes is selected)
	}
}

func NewPromptAdapterWithPredefined(db *database.Service, resourceType, msg, title string) *PromptAdapter {
	return &PromptAdapter{
		db:              db,
		resourceType:    resourceType,
		predefinedMsg:   msg,
		predefinedTitle: title,
		selectedNo:      false, // Default to "Yes" (selectedNo = false means Yes is selected)
	}
}

func (pa *PromptAdapter) PromptDo(msg, title string, width, height int) string {
	// Use predefined message and title if available, otherwise use provided ones
	promptMsg := msg
	promptTitle := title

	if pa.predefinedMsg != "" {
		promptMsg = pa.predefinedMsg
	}
	if pa.predefinedTitle != "" {
		promptTitle = pa.predefinedTitle
	}

	return pa.ViewPrompt(promptMsg, promptTitle, width, height)
}

// SetPredefinedText allows setting the predefined message and title dynamically
func (pa *PromptAdapter) SetPredefinedText(msg, title string) {
	pa.predefinedMsg = msg
	pa.predefinedTitle = title
}

// ToggleSelection toggles between Yes and No buttons
func (pa *PromptAdapter) ToggleSelection() {
	pa.selectedNo = !pa.selectedNo
}

// IsNoSelected returns true if "No" is currently selected
func (pa *PromptAdapter) IsNoSelected() bool {
	return pa.selectedNo
}

// SetSelection sets the button selection (true for No, false for Yes)
func (pa *PromptAdapter) SetSelection(selectNo bool) {
	pa.selectedNo = selectNo
}

func (pa *PromptAdapter) ViewPrompt(msg, title string, width, height int) string {
	// Base styles
	baseStyle := lipgloss.NewStyle()

	// Title style
	titleStyle := lipgloss.NewStyle().
		Background(colorOrange).
		Foreground(colorBlack).
		Padding(0, 1).
		Bold(true)

	// Message style
	msgStyle := lipgloss.NewStyle().
		Foreground(colorLightGrey).
		Bold(false)

	// Button styles - base style with padding
	buttonBaseStyle := lipgloss.NewStyle().
		Padding(0, 2)

	// Text color for unselected buttons
	textColor := colorLightGrey // Light grey for the rest (like descriptions)

	// Apply selection highlighting and create buttons
	// Selected button uses Yellow background (like navigation keys), unselected is darker
	var yesButton, noButton string

	if pa.selectedNo {
		// No is selected (highlighted with yellow background like navigation keys)
		noButtonStyle := buttonBaseStyle.
			Background(colorYellow).
			Foreground(colorBlack).
			Bold(true).
			Underline(true)
		noButton = noButtonStyle.Render("No")

		// Yes is not selected (dimmed)
		yesButtonStyle := buttonBaseStyle.
			Background(colorDarkGrey).
			Foreground(textColor).
			Underline(true)
		yesButton = yesButtonStyle.Render("Yes")
	} else {
		// Yes is selected (highlighted with yellow background like navigation keys)
		yesButtonStyle := buttonBaseStyle.
			Background(colorYellow).
			Foreground(colorBlack).
			Bold(true).
			Underline(true)
		yesButton = yesButtonStyle.Render("Yes")

		// No is not selected (dimmed)
		noButtonStyle := buttonBaseStyle.
			Background(colorDarkGrey).
			Foreground(textColor).
			Underline(true)
		noButton = noButtonStyle.Render("No")
	}

	// Split message into lines for proper rendering
	msgLines := strings.Split(msg, "\n")
	var renderedMsg []string
	for _, line := range msgLines {
		if strings.TrimSpace(line) != "" {
			renderedMsg = append(renderedMsg, msgStyle.Render(line))
		} else {
			renderedMsg = append(renderedMsg, "")
		}
	}
	messageContent := strings.Join(renderedMsg, "\n")

	// Calculate button container width based on message
	msgWidth := 0
	for _, line := range msgLines {
		if w := lipgloss.Width(line); w > msgWidth {
			msgWidth = w
		}
	}

	// Ensure minimum width for buttons
	minWidth := lipgloss.Width(yesButton) + lipgloss.Width(noButton) + 4 // 4 for spacing
	if msgWidth < minWidth {
		msgWidth = minWidth
	}

	// Create buttons row - No on left, Yes on right
	buttons := baseStyle.
		Width(msgWidth).
		Align(lipgloss.Right).
		Render(lipgloss.JoinHorizontal(lipgloss.Center, noButton, "  ", yesButton))

	// Combine title, message, and buttons
	titleRendered := titleStyle.Render(title)

	content := baseStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			titleRendered,
			"",
			messageContent,
			"",
			buttons,
		),
	)

	// Create dialog window with rounded border
	dialogStyle := baseStyle.
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBlue)

	dialog := dialogStyle.Render(content)

	// Center the dialog in the available space
	dialogWidth := lipgloss.Width(dialog)
	dialogHeight := lipgloss.Height(dialog)

	// Calculate centering
	verticalPadding := (height - dialogHeight) / 2
	horizontalCenterPadding := (width - dialogWidth) / 2

	if verticalPadding < 0 {
		verticalPadding = 0
	}
	if horizontalCenterPadding < 0 {
		horizontalCenterPadding = 0
	}

	// Create vertical padding
	verticalPad := strings.Repeat("\n", verticalPadding)

	// Create horizontal padding
	leftPad := strings.Repeat(" ", horizontalCenterPadding)

	// Add padding to each line of the dialog
	dialogLines := strings.Split(dialog, "\n")
	for i := range dialogLines {
		dialogLines[i] = leftPad + dialogLines[i]
	}

	// Combine everything
	centeredDialog := verticalPad + strings.Join(dialogLines, "\n")

	// Fill the rest with empty lines to take up full height
	currentHeight := verticalPadding + dialogHeight
	if currentHeight < height {
		bottomPadding := strings.Repeat("\n", height-currentHeight)
		centeredDialog += bottomPadding
	}

	return centeredDialog
}
