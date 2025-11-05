package tui

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
	"vastix/internal/msg_types"

	"github.com/charmbracelet/lipgloss"
)

var (
	allCars          = []rune{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z', '@', '#', '$', '%', '&', '*', '+', '=', '~', '^', '|', '\\', '/', '<', '>', '?'}
	vastCharsHidden  = []string{"*", "*", "*", "*"}
	vastCharsVisible = []string{"V", "A", "S", "T"}
)

type Spinner struct {
	counter    int
	leftChars  []rune
	rightChars []rune
	vastChars  []string
}

// NewSpinner creates a new spinner instance
func NewSpinner() *Spinner {
	return &Spinner{
		counter:    0,
		leftChars:  generateRandomChars(8),
		rightChars: generateRandomChars(8),
		vastChars:  vastCharsHidden[:],
	}
}

func (s *Spinner) Reset() {
	s.counter = 0
	s.leftChars = generateRandomChars(8)
	s.rightChars = generateRandomChars(8)
	s.vastChars = vastCharsHidden[:]
}

// generateRandomChars creates random ASCII characters
func generateRandomChars(count int) []rune {
	// ASCII characters for visual variety

	result := make([]rune, count)
	for i := 0; i < count; i++ {
		result[i] = allCars[rand.Intn(len(allCars))]
	}
	return result
}

// Update updates the spinner state
func (s *Spinner) Update() {
	s.counter++

	// Generate new random characters on each iteration
	s.leftChars = generateRandomChars(8)
	s.rightChars = generateRandomChars(8)

	// Every 30 iterations, reveal one letter of "VAST"
	letterIndex := (s.counter / 30) % 5

	// Fill in letters up to current position
	for i := 0; i <= letterIndex && i < 4; i++ {
		if s.counter >= (i+1)*30 {
			s.vastChars[i] = vastCharsVisible[i]
		}
	}

	// Reset after completing "VAST" (120 iterations)
	if s.counter >= 150 {
		s.counter = 0
		s.vastChars = []string{"*", "*", "*", "*"}
	}
}

// View renders the spinner
func (s *Spinner) View() string {
	// Build the spinner line
	leftStr := string(s.leftChars)
	rightStr := string(s.rightChars)
	middleStr := strings.Join(s.vastChars, "")

	spinnerLine := fmt.Sprintf("%s%s%s", leftStr, middleStr, rightStr)

	// Apply gradient to the spinner line
	lines := []string{spinnerLine}
	gradientLines := ApplyGradient(lines)
	spinnerContent := gradientLines[0]

	// Add gradient-colored brackets around the spinner content
	return s.addBracketsToSpinner(spinnerContent)
}

// StartSpinner creates a new spinner and returns a channel for tick messages.
// It starts a goroutine that sends new spinner tick messages every 80 milliseconds.
// The goroutine stops when the provided context is cancelled.
func StartSpinner(ctx context.Context) (*Spinner, <-chan msg_types.SpinnerTickMsg) {
	spinner := NewSpinner()
	tickChan := make(chan msg_types.SpinnerTickMsg)
	spinnerCounter := 0

	// Start goroutine that sends ticks every 80 milliseconds
	go func() {

		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		defer close(tickChan)

		for {
			spinnerCounter = (spinnerCounter + 1) % 100
			select {
			case <-ticker.C:
				spinner.Update()
				// Send the updated view as the tick message
				select {
				case tickChan <- msg_types.SpinnerTickMsg(spinner.View()):
					// Successfully sent
				case <-ctx.Done():
					return // Exit if context cancelled while trying to send
				default:
					// Channel blocked, skip this tick to avoid blocking the spinner
				}
			case <-ctx.Done():
				return // Exit goroutine when context is cancelled
			}
		}
	}()

	return spinner, tickChan
}

// addBracketsToSpinner adds colored brackets around spinner using gradient colors
func (s *Spinner) addBracketsToSpinner(spinnerText string) string {
	if spinnerText == "" {
		return ""
	}

	// Use the same gradient colors as working zone used
	gradientColors := []string{
		"#00FFFF", // Cyan
		"#40E0D0", // Turquoise
		"#8A2BE2", // Blue Violet
		"#FF69B4", // Hot Pink
		"#FF1493", // Deep Pink
	}

	// Use a color based on the spinner content to create variation
	colorIndex := len(spinnerText) % len(gradientColors)
	bracketStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(gradientColors[colorIndex]))

	// Create colored brackets with spaces
	leftBracket := bracketStyle.Render("[  ")
	rightBracket := bracketStyle.Render("  ]")

	return leftBracket + spinnerText + rightBracket
}
