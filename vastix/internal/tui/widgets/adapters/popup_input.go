package adapters

import (
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PopupInput provides a centered modal input field for adapters
type PopupInput struct {
	textInput   textinput.Model
	title       string
	placeholder string
	hidden      bool   // Controls visibility (true = hidden)
	content     string // Stores the input result
	isSecret    bool   // Whether to use password mode
}

// NewPopupInput creates a new popup input
func NewPopupInput() *PopupInput {
	ti := textinput.New()
	ti.CharLimit = 100
	ti.Width = 40
	ti.Prompt = ""

	return &PopupInput{
		textInput: ti,
		hidden:    true,
		content:   "",
		isSecret:  false,
	}
}

// Show displays the popup with the given title and placeholder
func (p *PopupInput) Show(title, placeholder string, isSecret bool) {
	log.Printf("DEBUG PopupInput.Show: called with title=%q, clearing content", title)
	p.title = title
	p.placeholder = placeholder
	p.isSecret = isSecret
	p.hidden = false
	p.content = ""

	p.textInput.SetValue("")
	p.textInput.Placeholder = placeholder
	if isSecret {
		p.textInput.EchoMode = textinput.EchoPassword
	} else {
		p.textInput.EchoMode = textinput.EchoNormal
	}
	p.textInput.Focus()
	log.Printf("DEBUG PopupInput.Show: popup is now visible (hidden=%v)", p.hidden)
}

// Hide hides the popup
func (p *PopupInput) Hide() {
	p.hidden = true
	p.textInput.Blur()
}

// IsHidden returns whether the popup is hidden
func (p *PopupInput) IsHidden() bool {
	return p.hidden
}

// GetContent returns the stored input content
func (p *PopupInput) GetContent() string {
	return p.content
}

// ClearContent clears the stored content
func (p *PopupInput) ClearContent() {
	p.content = ""
}

// Submit stores the current input value and hides the popup
func (p *PopupInput) Submit() {
	p.content = p.textInput.Value()
	log.Printf("DEBUG PopupInput.Submit: storing content (length=%d), hiding popup", len(p.content))
	p.Hide()
	log.Printf("DEBUG PopupInput.Submit: popup hidden=%v", p.hidden)
}

// Cancel hides the popup without storing the value
func (p *PopupInput) Cancel() {
	p.Hide()
}

// Update handles input updates
func (p *PopupInput) Update(msg tea.Msg) tea.Cmd {
	if p.hidden {
		return nil
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.Type {
		case tea.KeyEnter:
			p.Submit()
			return nil
		case tea.KeyEsc:
			p.Cancel()
			return nil
		}
	}

	var cmd tea.Cmd
	p.textInput, cmd = p.textInput.Update(msg)
	return cmd
}

// View renders the popup centered on screen
func (p *PopupInput) View(width, height int) string {
	if p.hidden {
		return ""
	}

	// Base styles
	baseStyle := lipgloss.NewStyle()

	// Title style
	titleStyle := lipgloss.NewStyle().
		Background(colorOrange).
		Foreground(colorBlack).
		Padding(0, 1).
		Bold(true)

	// Instruction style
	instructionStyle := lipgloss.NewStyle().
		Foreground(colorLightGrey).
		Bold(false)

	// Render title
	titleRendered := titleStyle.Render(p.title)

	// Render instructions
	instructions := instructionStyle.Render("Press Enter to submit, Esc to cancel")

	// Render input field
	inputView := p.textInput.View()

	// Combine title, input, and instructions
	content := baseStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			titleRendered,
			"",
			inputView,
			"",
			instructions,
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
