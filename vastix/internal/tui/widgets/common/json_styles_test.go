package common

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"vastix/internal/colors"
)

func TestContentLineFillsWidth(t *testing.T) {
	line := ContentLine(12, "hi")
	if lipgloss.Width(line) != 12 {
		t.Fatalf("line width = %d, want 12", lipgloss.Width(line))
	}
}

func TestIndentSpacesHasWidth(t *testing.T) {
	indent := IndentSpaces(4)
	if lipgloss.Width(indent) != 4 {
		t.Fatalf("indent width = %d, want 4", lipgloss.Width(indent))
	}
}

func TestContentLinePadsStyledText(t *testing.T) {
	styled := lipgloss.NewStyle().Foreground(colors.MediumGreen).Render("key")
	line := ContentLine(20, styled)
	if lipgloss.Width(line) != 20 {
		t.Fatalf("width = %d, want 20", lipgloss.Width(line))
	}
}

func TestFillOpaqueLinesPadsWithoutRewrapping(t *testing.T) {
	s := NewJSONSyntaxStyles()
	line := s.String.Render("value")
	lines := FillOpaqueLines([]string{line}, 20, 2)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lipgloss.Width(lines[0]) != 20 {
		t.Fatalf("line width = %d, want 20", lipgloss.Width(lines[0]))
	}
	if lipgloss.Width(lines[1]) != 20 {
		t.Fatalf("empty line width = %d, want 20", lipgloss.Width(lines[1]))
	}
}

func TestVisibleLinesScrollsContent(t *testing.T) {
	content := strings.Join([]string{"line1", "line2", "line3"}, "\n")
	out := VisibleLines(content, 1, 10, 2)
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 visible lines, got %d", len(lines))
	}
	if lipgloss.Width(lines[0]) != 10 {
		t.Fatalf("first line width = %d", lipgloss.Width(lines[0]))
	}
}
