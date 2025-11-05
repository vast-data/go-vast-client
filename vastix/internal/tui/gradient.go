package tui

// This file re-exports gradient functionality from the centralized colors package
// for backwards compatibility with existing TUI code.
// New code should import "vastix/internal/colors" directly.

import "vastix/internal/colors"

// ApplyGradient applies a gradient color to the given text lines
// Deprecated: Use colors.ApplyGradient directly
func ApplyGradient(lines []string) []string {
	return colors.ApplyGradient(lines)
}
