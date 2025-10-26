package common

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// BorderPosition represents different border positions for embedded text
type BorderPosition int

const (
	TopLeftBorder BorderPosition = iota
	TopMiddleBorder
	TopRightBorder
	BottomLeftBorder
	BottomMiddleBorder
	BottomRightBorder
)

// Colors - copied from main TUI package
var (
	Blue                  = lipgloss.Color("63")
	InactivePreviewBorder = lipgloss.AdaptiveColor{
		Dark:  "244",
		Light: "250",
	}
)

// GlobalSpinnerState interface for accessing spinner state
// This will be set by the main application
var GlobalSpinnerAccessor func() bool

// SetGlobalSpinnerAccessor sets the function to access global spinner state
func SetGlobalSpinnerAccessor(accessor func() bool) {
	GlobalSpinnerAccessor = accessor
}

// isGlobalSpinnerActive returns true if the global spinner is currently active
func isGlobalSpinnerActive() bool {
	if GlobalSpinnerAccessor != nil {
		return GlobalSpinnerAccessor()
	}
	return false
}

// Borderize wraps content with a border and embedded text
// It automatically detects if the global spinner is active
func Borderize(content string, active bool, embeddedText map[BorderPosition]string) string {
	if embeddedText == nil {
		embeddedText = make(map[BorderPosition]string)
	}

	var (
		// Always use thick border for working area main border
		border = lipgloss.Border(lipgloss.ThickBorder())

		// Choose color based on active state
		borderColor lipgloss.TerminalColor
		style       lipgloss.Style
		width       = lipgloss.Width(content)
	)

	if active {
		// Active: use blue color with thick border
		borderColor = Blue
	} else {
		// Inactive: use gray color with thick border
		borderColor = InactivePreviewBorder
	}

	style = lipgloss.NewStyle().Foreground(borderColor)

	encloseInSquareBrackets := func(text string) string {
		if text != "" {
			return fmt.Sprintf("%s%s%s",
				style.Render(border.TopRight),
				text,
				style.Render(border.TopLeft),
			)
		}
		return text
	}

	buildHorizontalBorder := func(leftText, middleText, rightText, leftCorner, inbetween, rightCorner string) string {
		leftText = encloseInSquareBrackets(leftText)
		middleText = encloseInSquareBrackets(middleText)
		rightText = encloseInSquareBrackets(rightText)
		// Calculate length of border between embedded texts
		remaining := max(0, width-lipgloss.Width(leftText)-lipgloss.Width(middleText)-lipgloss.Width(rightText))
		leftBorderLen := max(0, (width/2)-lipgloss.Width(leftText)-(lipgloss.Width(middleText)/2))
		rightBorderLen := max(0, remaining-leftBorderLen)
		// Then construct border string
		s := leftText +
			style.Render(strings.Repeat(inbetween, leftBorderLen)) +
			middleText +
			style.Render(strings.Repeat(inbetween, rightBorderLen)) +
			rightText
		// Make it fit in the space available between the two corners.
		s = lipgloss.NewStyle().
			Inline(true).
			MaxWidth(width).
			Render(s)
		// Add the corners
		return style.Render(leftCorner) + s + style.Render(rightCorner)
	}

	// Stack top border, content and horizontal borders, and bottom border.
	return strings.Join([]string{
		buildHorizontalBorder(
			embeddedText[TopLeftBorder],
			embeddedText[TopMiddleBorder],
			embeddedText[TopRightBorder],
			border.TopLeft,
			border.Top,
			border.TopRight,
		),
		lipgloss.NewStyle().
			BorderForeground(borderColor).
			Border(border, false, true, false, true).Render(content),
		buildHorizontalBorder(
			embeddedText[BottomLeftBorder],
			embeddedText[BottomMiddleBorder],
			embeddedText[BottomRightBorder],
			border.BottomLeft,
			border.Bottom,
			border.BottomRight,
		),
	}, "\n")
}

// BorderizeWithSpinnerCheck wraps content with a border, automatically checking spinner state
func BorderizeWithSpinnerCheck(content string, active bool, embeddedText map[BorderPosition]string) string {
	// If global spinner is active, make border inactive to indicate disabled state
	if isGlobalSpinnerActive() {
		active = false
	}
	return Borderize(content, active, embeddedText)
}

// Helper function for max (for Go versions < 1.21)
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
