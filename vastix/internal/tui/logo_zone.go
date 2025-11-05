package tui

import (
	"strings"

	"vastix/internal/database"
	log "vastix/internal/logging"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LogoZone represents the logo zone
type LogoZone struct {
	width, height int
	db            *database.Service
}

// NewLogoZone creates a new logo zone with logging support
func NewLogoZone(db *database.Service) *LogoZone {
	// Initialize logging service (singleton pattern)

	log.Debug("LogoZone initializing")

	logo := &LogoZone{
		db: db,
	}

	log.Debug("LogoZone initialized successfully")

	return logo
}

func (l *LogoZone) Init() {}

// SetSize sets the dimensions of the logo zone
func (l *LogoZone) SetSize(width, height int) {
	l.width = width
	l.height = height
}

// Update handles messages for the logo zone
func (l *LogoZone) Update(msg tea.Msg) (*LogoZone, tea.Cmd) {
	return l, nil
}

// View renders the logo zone
func (l *LogoZone) View() string {
	if l.width == 0 {
		return ""
	}

	bigText := []string{
		"____   ____  ___   _____ _____ ",
		"\\   \\ /   / /   \\ /  ___/_   _|",
		" \\   V   / |  _  |\\___ \\  | |  ",
		"  \\     /  |  _  |/___  \\ | |  ",
		"   \\___/   |_| |_|\\____/  |_|  ",
		"                              ",
	}

	// Apply gradient using common function
	gradientLines := ApplyGradient(bigText)

	logoStyle := lipgloss.NewStyle().
		Bold(true).
		Align(lipgloss.Right)

	// Join gradient lines and render
	logoContent := strings.Join(gradientLines, "\n")
	styledLogo := logoStyle.Render(logoContent)

	return styledLogo
}

// Ready returns whether the logo zone is ready to be displayed
func (l *LogoZone) Ready() bool {
	// Logo zone is always ready since it's static
	return true
}
