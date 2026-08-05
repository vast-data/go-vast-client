package common

import (
	"strings"

	"vastix/internal/colors"

	"github.com/charmbracelet/lipgloss"
)

// JSONSyntaxStyles carries foreground colors for JSON detail views.
// Background comes from the terminal default (OSC 11 / crush BackgroundColor).
type JSONSyntaxStyles struct {
	Key     lipgloss.Style
	String  lipgloss.Style
	Number  lipgloss.Style
	Bool    lipgloss.Style
	Null    lipgloss.Style
	Bracket lipgloss.Style
	Indent2 string
}

// NewJSONSyntaxStyles returns syntax-highlight styles (foreground only).
func NewJSONSyntaxStyles() JSONSyntaxStyles {
	style := func(fg lipgloss.TerminalColor) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(fg)
	}
	return JSONSyntaxStyles{
		Key:     style(colors.MediumCyan),
		String:  style(colors.MediumGreen),
		Number:  style(colors.MutedOrange),
		Bool:    style(colors.MediumPurple),
		Null:    style(colors.MediumGrey),
		Bracket: style(colors.VeryLightGrey),
		Indent2: "  ",
	}
}

// IndentSpaces returns plain indentation (terminal bg shows through).
func IndentSpaces(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(" ", n)
}

// ContentBase returns default content foreground style.
func ContentBase() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colors.OffWhite)
}

// ContentMuted returns subdued content foreground style.
func ContentMuted() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colors.LightGrey)
}

// ContentLine renders a line with optional width padding (no lipgloss background).
func ContentLine(width int, text string) string {
	if width < 1 {
		return text
	}
	w := lipgloss.Width(text)
	if w >= width {
		return text
	}
	return text + strings.Repeat(" ", width-w)
}

// VisibleLines returns the scrolled slice of content lines, padded to height with blank lines.
func VisibleLines(content string, yOffset, width, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	allLines := strings.Split(content, "\n")
	if yOffset > len(allLines) {
		yOffset = len(allLines)
	}
	end := yOffset + height
	if end > len(allLines) {
		end = len(allLines)
	}

	visible := allLines[yOffset:end]
	for len(visible) < height {
		visible = append(visible, "")
	}
	if len(visible) > height {
		visible = visible[:height]
	}

	for i, line := range visible {
		visible[i] = ContentLine(width, line)
	}

	return strings.Join(visible, "\n")
}

// FillOpaqueLines pads lines to width/height without lipgloss backgrounds.
func FillOpaqueLines(lines []string, width, height int) []string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}

	for i, line := range lines {
		lines[i] = ContentLine(width, line)
	}

	return lines
}
