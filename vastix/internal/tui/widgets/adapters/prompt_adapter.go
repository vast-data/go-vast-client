package adapters

import (
	"strings"

	"vastix/internal/database"
	"vastix/internal/tui/widgets/common"

	"github.com/charmbracelet/lipgloss"
)

type PromptAdapter struct {
	db              *database.Service
	resourceType    string
	predefinedMsg   string
	predefinedTitle string
}

func NewPromptAdapter(db *database.Service, resourceType string) *PromptAdapter {
	return &PromptAdapter{
		db:           db,
		resourceType: resourceType,
	}
}

func NewPromptAdapterWithPredefined(db *database.Service, resourceType, msg, title string) *PromptAdapter {
	return &PromptAdapter{
		db:              db,
		resourceType:    resourceType,
		predefinedMsg:   msg,
		predefinedTitle: title,
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

func (pa *PromptAdapter) ViewPrompt(msg, title string, width, height int) string {
	// Main prompt message styling
	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")). // LightGrey
		Bold(true)

	// Hint styling for (y/n)
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#DBBD70")). // Yellow
		Bold(true)

	styledMsg := promptStyle.Render(msg)
	styledHint := hintStyle.Render("(y/n)")

	// Sizing and layout
	innerWidth := width - 2
	innerHeight := height - 3

	if innerWidth < 1 {
		innerWidth = width
	}
	if innerHeight < 1 {
		innerHeight = 5 // Minimum height
	}

	lines := make([]string, innerHeight)
	for i := range lines {
		lines[i] = strings.Repeat(" ", innerWidth)
	}

	// Centering the main message
	msgLines := strings.Split(styledMsg, "\n")
	msgHeight := len(msgLines)
	topPadding := (innerHeight - msgHeight - 2) / 2 // -2 to account for hint and spacing
	if topPadding < 0 {
		topPadding = 0
	}

	for i, line := range msgLines {
		lineIndex := topPadding + i
		if lineIndex < len(lines) {
			lineWidth := lipgloss.Width(line)
			leftPadding := (innerWidth - lineWidth) / 2
			if leftPadding < 0 {
				leftPadding = 0
			}
			padding := strings.Repeat(" ", leftPadding)
			lines[lineIndex] = padding + line
		}
	}

	// Centering the (y/n) hint
	hintIndex := topPadding + msgHeight + 1 // Place it below the message
	if hintIndex < len(lines) {
		hintWidth := lipgloss.Width(styledHint)
		leftPadding := (innerWidth - hintWidth) / 2
		if leftPadding < 0 {
			leftPadding = 0
		}
		padding := strings.Repeat(" ", leftPadding)
		lines[hintIndex] = padding + styledHint
	}

	content := strings.Join(lines, "\n")

	// Border and title
	resourceNameStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("214")).
		Foreground(lipgloss.Color("0"))

	embeddedText := map[common.BorderPosition]string{
		common.TopMiddleBorder: resourceNameStyle.Render(title),
	}

	return common.Borderize(content, true, embeddedText)
}
