package tui

import (
	"strings"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/muesli/termenv"
)

// ApplyGradient applies a gradient color to the given text lines
func ApplyGradient(lines []string) []string {
	// Same gradient colors as used in logo
	colorA, _ := colorful.Hex("#5A56E0") // Default gradient start color
	colorB, _ := colorful.Hex("#EE6FF8") // Default gradient end color

	var gradientLines []string

	for _, line := range lines {
		var gradientLine strings.Builder
		lineWidth := len(line)

		for i, char := range line {
			if char == ' ' {
				// Keep spaces as spaces
				gradientLine.WriteRune(char)
			} else {
				// Calculate gradient position (0 to 1)
				var p float64
				if lineWidth == 1 {
					p = 0.5
				} else {
					p = float64(i) / float64(lineWidth-1)
				}

				// Blend colors
				c := colorA.BlendLuv(colorB, p).Hex()

				// Apply color to character
				coloredChar := termenv.String(string(char)).
					Foreground(termenv.ColorProfile().Color(c)).
					String()
				gradientLine.WriteString(coloredChar)
			}
		}
		gradientLines = append(gradientLines, gradientLine.String())
	}

	return gradientLines
}
