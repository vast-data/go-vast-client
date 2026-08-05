package tui

import (
	"fmt"
	"os"

	"vastix/internal/colors"
)

// ApplyTerminalDefaultBackground sets the terminal emulator's default background
// color (OSC 11). Unstyled cells use this color — same idea as crush's
// tea.View.BackgroundColor without per-widget lipgloss fills.
func ApplyTerminalDefaultBackground() {
	r, g, b, _ := colors.AppBackgroundRGBA().RGBA()
	// OSC 11: set terminal default background (xterm, kitty, gnome-terminal, iTerm, etc.)
	fmt.Fprintf(os.Stdout, "\033]11;rgb:%02x/%02x/%02x\033\\", r>>8, g>>8, b>>8)
}

// RestoreTerminalDefaultBackground resets the terminal default background to the
// emulator's configured default (OSC 111), instead of forcing a specific color.
func RestoreTerminalDefaultBackground() {
	fmt.Fprintf(os.Stdout, "\033]111\033\\")
}
