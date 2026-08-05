package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FillScreenBackground pads content to the terminal size.
// Primary background comes from ApplyTerminalDefaultBackground (OSC 11).
// Lines are only padded, never truncated — renderNormal already sizes zones to fit.
func FillScreenBackground(content string, width, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	lines := strings.Split(content, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}

	for i, line := range lines {
		lineWidth := lipgloss.Width(line)
		if lineWidth < width {
			lines[i] = line + strings.Repeat(" ", width-lineWidth)
		}
	}

	return strings.Join(lines, "\n")
}
