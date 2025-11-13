package colors

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/muesli/termenv"
)

// Base colors - Named colors for better readability
const (
	// Grayscale
	Black           = lipgloss.Color("#000000")
	DarkGrey        = lipgloss.Color("#606362")
	Grey            = lipgloss.Color("#737373")
	MediumGrey      = lipgloss.Color("243")
	LightGrey       = lipgloss.Color("245")
	LighterGrey     = lipgloss.Color("250")
	VeryLightGrey   = lipgloss.Color("252")
	EvenLighterGrey = lipgloss.Color("253")
	OffWhite        = lipgloss.Color("#a8a7a5")
	White           = lipgloss.Color("#ffffff")

	// Numbered greys for specific uses
	Grey240 = lipgloss.Color("240") // Dim/disabled text
	Grey243 = lipgloss.Color("243") // Medium dim
	Grey250 = lipgloss.Color("250") // Light borders

	// Reds
	DarkRed   = lipgloss.Color("1") // Dark red
	BrightRed = lipgloss.Color("#FF0000")
	Red       = lipgloss.Color("#FF5353")

	// Oranges & Yellows
	BurntOrange = lipgloss.Color("214")
	Orange      = lipgloss.Color("214")
	MutedOrange = lipgloss.Color("179")
	Yellow      = lipgloss.Color("#DBBD70")

	// Greens
	DarkGreen   = lipgloss.Color("#325451")
	Green       = lipgloss.Color("34")
	GreenTerm   = lipgloss.Color("2") // Terminal green
	BrightGreen = lipgloss.Color("42")
	MediumGreen = lipgloss.Color("77")
	LightGreen  = lipgloss.Color("47")
	NeonGreen   = lipgloss.Color("#00FF00")
	GreenBlue   = lipgloss.Color("#00A095")
	BrightCyan  = lipgloss.Color("#00D787") // Bright green/cyan for extra actions
	Turquoise   = lipgloss.Color("86")

	// Blues
	DarkBlue      = lipgloss.Color("18") // Dark blue background
	VeryDarkBlue  = lipgloss.Color("25") // Darker blue for spinner
	DarkGreenBlue = lipgloss.Color("22") // Dark green-blue
	DeepBlue      = lipgloss.Color("39")
	MediumCyan    = lipgloss.Color("51") // Medium cyan for JSON keys
	Blue          = lipgloss.Color("63")
	LightishBlue  = lipgloss.Color("75")
	LightBlue     = lipgloss.Color("81")

	// Purples
	MediumPurple = lipgloss.Color("105")
	SelectedBg   = lipgloss.Color("110") // Selected item background
	Purple       = lipgloss.Color("135")
	Violet       = lipgloss.Color("13")

	// Pinks
	CurrentSelectedBg = lipgloss.Color("117") // Current + selected background
	HotPink           = lipgloss.Color("200")

	// Numbered colors for terminal compatibility
	BlackTerm    = lipgloss.Color("0")  // Terminal black
	WhiteTerm    = lipgloss.Color("15") // Terminal white
	BlackishTerm = lipgloss.Color("8")  // Terminal dark grey
)

// Semantic color names - Use these for specific UI elements
var (
	// Log levels
	DebugLogLevel = Blue
	InfoLogLevel  = lipgloss.AdaptiveColor{Dark: string(Turquoise), Light: string(Green)}
	ErrorLogLevel = Red
	WarnLogLevel  = Yellow

	// Log attributes
	LogRecordAttributeKey = lipgloss.AdaptiveColor{Dark: string(LightGrey), Light: string(LightGrey)}

	// Help/Keybindings
	HelpKey = lipgloss.AdaptiveColor{
		Dark:  "ff",
		Light: "",
	}
	HelpDesc = lipgloss.AdaptiveColor{
		Dark:  "248",
		Light: "246",
	}
	HelpKeyExtra = BrightCyan // For extra action numbered shortcuts

	// Borders
	InactivePreviewBorder = lipgloss.AdaptiveColor{
		Dark:  "244",
		Light: "250",
	}
	BorderBlue   = DeepBlue    // For focused borders
	BorderNormal = Grey240     // For normal borders
	BorderLight  = LighterGrey // For light borders

	// List selection states
	CurrentBackground            = Grey
	CurrentForeground            = White
	SelectedBackground           = SelectedBg
	SelectedForeground           = Black
	CurrentAndSelectedBackground = CurrentSelectedBg
	CurrentAndSelectedForeground = Black

	// Title colors
	TitleColor = lipgloss.AdaptiveColor{
		Dark:  "",
		Light: "",
	}

	// Background colors
	GroupReportBackgroundColor = EvenLighterGrey
	TaskSummaryBackgroundColor = EvenLighterGrey
	ScrollPercentageBackground = lipgloss.AdaptiveColor{
		Dark:  string(DarkGrey),
		Light: string(EvenLighterGrey),
	}

	// Input field colors
	InputFocusedBg     = DarkBlue    // Focused input background
	InputFocusedFg     = WhiteTerm   // Focused input text
	InputFocusedBorder = DeepBlue    // Focused input border
	InputNormalBorder  = Grey240     // Normal input border
	InputNormalFg      = LighterGrey // Normal input text
	InputLabelFg       = Orange      // Input label color
	InputRequiredStar  = Red         // Required field asterisk
	InputTypeFg        = Grey240     // Input type annotation

	// Boolean toggle colors
	BooleanEnabledBg  = BrightGreen // Green for enabled/true
	BooleanDisabledBg = LighterGrey // Grey for disabled/false
	BooleanTextFg     = WhiteTerm   // Text on boolean badges

	// JSON syntax highlighting
	JSONKeyColor    = MediumCyan   // JSON object keys
	JSONStringColor = MediumGreen  // JSON string values
	JSONNumberColor = MutedOrange  // JSON number values
	JSONBoolColor   = MediumPurple // JSON boolean values

	// Status colors
	SuccessColor = BrightGreen // Success states, enabled badges
	ErrorColor   = Red         // Error states
	WarningColor = Orange      // Warning states
	InfoColor    = DeepBlue    // Info states
	DimColor     = Grey240     // Dimmed/disabled text

	// Profile status colors
	ProfileActiveColor = GreenTerm // Active profile indicator
)

// Gradient colors for logo and decorative elements
var (
	GradientStart = "#5A56E0" // Purple
	GradientEnd   = "#EE6FF8" // Pink
)

// ApplyGradient applies a gradient color to the given text lines
// Used for the application logo and decorative text
func ApplyGradient(lines []string) []string {
	colorA, _ := colorful.Hex(GradientStart)
	colorB, _ := colorful.Hex(GradientEnd)

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
