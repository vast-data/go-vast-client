package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestFillScreenBackground(t *testing.T) {
	content := "hello"
	result := FillScreenBackground(content, 10, 3)
	lines := strings.Split(result, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	for i, line := range lines {
		if lipgloss.Width(line) != 10 {
			t.Fatalf("line %d width = %d, want 10", i, lipgloss.Width(line))
		}
	}
}
