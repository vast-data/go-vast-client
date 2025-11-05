package adapters

import (
	"vastix/internal/colors"
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"vastix/internal/database"
	log "vastix/internal/logging"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.uber.org/zap"
)

// TerminalOutputMsg is sent when command output is received
type TerminalOutputMsg struct {
	Line string
	Err  error
}

// TerminalExitMsg is sent when command exits
type TerminalExitMsg struct {
	ExitCode int
	Err      error
}

// TerminalAdapter displays real-time command output in the working zone
type TerminalAdapter struct {
	// Display properties
	width, height int
	title         string

	// Command state
	cmd        *exec.Cmd
	running    bool
	exitCode   int
	finished   bool
	finishTime time.Time

	// Output buffer
	outputLines []string
	mu          sync.Mutex // Protects outputLines

	// Scrolling
	scrollOffset int
	maxLines     int // Maximum lines to keep in buffer

	// Database
	db *database.Service

	// Styling
	titleStyle   lipgloss.Style
	borderStyle  lipgloss.Style
	textStyle    lipgloss.Style
	errorStyle   lipgloss.Style
	successStyle lipgloss.Style
}

// NewTerminalAdapter creates a new terminal output adapter
func NewTerminalAdapter(db *database.Service, title string) *TerminalAdapter {
	return &TerminalAdapter{
		title:       title,
		outputLines: make([]string, 0),
		maxLines:    1000, // Keep last 1000 lines
		db:          db,
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(Blue).
			Padding(0, 1),
		borderStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Blue).
			Padding(1, 2),
		textStyle: lipgloss.NewStyle().
			Foreground(White),
		errorStyle: lipgloss.NewStyle().
			Foreground(colors.BrightRed).
			Bold(true),
		successStyle: lipgloss.NewStyle().
			Foreground(colors.NeonGreen).
			Bold(true),
	}
}

// SetSize sets the dimensions for the terminal display
func (t *TerminalAdapter) SetSize(width, height int) {
	t.width = width
	t.height = height
}

// GetWidth returns the current width
func (t *TerminalAdapter) GetWidth() int {
	return t.width
}

// GetHeight returns the current height
func (t *TerminalAdapter) GetHeight() int {
	return t.height
}

// RunCommand starts executing a command and streaming its output
func (t *TerminalAdapter) RunCommand(name string, args ...string) tea.Cmd {
	return func() tea.Msg {
		t.mu.Lock()
		t.cmd = exec.Command(name, args...)
		t.running = true
		t.finished = false
		t.outputLines = []string{} // Clear previous output
		t.mu.Unlock()

		// Get stdout pipe
		stdout, err := t.cmd.StdoutPipe()
		if err != nil {
			return TerminalExitMsg{ExitCode: -1, Err: fmt.Errorf("failed to get stdout: %w", err)}
		}

		// Get stderr pipe
		stderr, err := t.cmd.StderrPipe()
		if err != nil {
			return TerminalExitMsg{ExitCode: -1, Err: fmt.Errorf("failed to get stderr: %w", err)}
		}

		// Start the command
		if err := t.cmd.Start(); err != nil {
			return TerminalExitMsg{ExitCode: -1, Err: fmt.Errorf("failed to start command: %w", err)}
		}

		log.Info("Terminal: command started",
			zap.String("command", name),
			zap.Strings("args", args))

		// Stream stdout and stderr
		go t.streamOutput(stdout, false)
		go t.streamOutput(stderr, true)

		// Wait for command to finish (in background)
		go func() {
			err := t.cmd.Wait()
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					exitCode = -1
				}
			}

			t.mu.Lock()
			t.running = false
			t.finished = true
			t.exitCode = exitCode
			t.finishTime = time.Now()
			t.mu.Unlock()

			log.Info("Terminal: command finished",
				zap.Int("exitCode", exitCode),
				zap.Error(err))
		}()

		return TerminalOutputMsg{Line: fmt.Sprintf("$ %s %s", name, strings.Join(args, " ")), Err: nil}
	}
}

// streamOutput reads from a pipe and sends output messages
func (t *TerminalAdapter) streamOutput(pipe io.Reader, isStderr bool) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()

		t.mu.Lock()
		// Prefix stderr with "[ERROR]" for visibility
		if isStderr {
			line = "[ERROR] " + line
		}

		t.outputLines = append(t.outputLines, line)

		// Trim buffer if it exceeds maxLines
		if len(t.outputLines) > t.maxLines {
			t.outputLines = t.outputLines[len(t.outputLines)-t.maxLines:]
		}
		t.mu.Unlock()

		log.Debug("Terminal output", zap.String("line", line), zap.Bool("stderr", isStderr))
	}

	if err := scanner.Err(); err != nil {
		log.Error("Terminal: error reading output", zap.Error(err))
	}
}

// AddLine adds a line to the output (useful for custom messages)
func (t *TerminalAdapter) AddLine(line string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.outputLines = append(t.outputLines, line)
	if len(t.outputLines) > t.maxLines {
		t.outputLines = t.outputLines[len(t.outputLines)-t.maxLines:]
	}
}

// Clear clears the output buffer
func (t *TerminalAdapter) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.outputLines = []string{}
	t.scrollOffset = 0
}

// ScrollUp scrolls the view up
func (t *TerminalAdapter) ScrollUp() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.scrollOffset > 0 {
		t.scrollOffset--
	}
}

// ScrollDown scrolls the view down
func (t *TerminalAdapter) ScrollDown() {
	t.mu.Lock()
	defer t.mu.Unlock()

	maxOffset := len(t.outputLines) - (t.height - 6) // Account for border and title
	if maxOffset < 0 {
		maxOffset = 0
	}

	if t.scrollOffset < maxOffset {
		t.scrollOffset++
	}
}

// ScrollToBottom scrolls to the bottom of the output
func (t *TerminalAdapter) ScrollToBottom() {
	t.mu.Lock()
	defer t.mu.Unlock()

	maxOffset := len(t.outputLines) - (t.height - 6)
	if maxOffset < 0 {
		maxOffset = 0
	}
	t.scrollOffset = maxOffset
}

// IsRunning returns true if command is still running
func (t *TerminalAdapter) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

// GetExitCode returns the exit code (only valid after command finishes)
func (t *TerminalAdapter) GetExitCode() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.exitCode
}

// View renders the terminal output
func (t *TerminalAdapter) View() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.width == 0 || t.height == 0 {
		return ""
	}

	var content strings.Builder

	// Title
	content.WriteString(t.titleStyle.Render(t.title))
	content.WriteString("\n\n")

	// Status line
	statusLine := ""
	if t.running {
		statusLine = t.textStyle.Render("⚙ Running...")
	} else if t.finished {
		if t.exitCode == 0 {
			statusLine = t.successStyle.Render(fmt.Sprintf("✓ Completed successfully (exit code: %d)", t.exitCode))
		} else {
			statusLine = t.errorStyle.Render(fmt.Sprintf("✗ Failed (exit code: %d)", t.exitCode))
		}
		if !t.finishTime.IsZero() {
			statusLine += t.textStyle.Render(fmt.Sprintf(" at %s", t.finishTime.Format("15:04:05")))
		}
	} else {
		statusLine = t.textStyle.Render("Ready")
	}
	content.WriteString(statusLine)
	content.WriteString("\n\n")

	// Output area
	visibleLines := t.height - 8 // Account for title, status, border padding
	if visibleLines < 1 {
		visibleLines = 1
	}

	totalLines := len(t.outputLines)
	startLine := t.scrollOffset
	endLine := startLine + visibleLines

	if endLine > totalLines {
		endLine = totalLines
	}

	if startLine < totalLines {
		for i := startLine; i < endLine; i++ {
			line := t.outputLines[i]

			// Color stderr lines in red
			if strings.HasPrefix(line, "[ERROR]") {
				content.WriteString(t.errorStyle.Render(line))
			} else {
				content.WriteString(t.textStyle.Render(line))
			}
			content.WriteString("\n")
		}
	}

	// Scroll indicator
	if totalLines > visibleLines {
		scrollInfo := fmt.Sprintf("\n[Lines %d-%d of %d | Use ↑/↓ to scroll]",
			startLine+1, endLine, totalLines)
		content.WriteString(t.textStyle.Faint(true).Render(scrollInfo))
	}

	// Wrap in border
	return t.borderStyle.Width(t.width - 4).Height(t.height - 2).Render(content.String())
}

// WaitForCompletion creates a command that polls until the process completes
func (t *TerminalAdapter) WaitForCompletion(pollInterval time.Duration) tea.Cmd {
	return func() tea.Msg {
		for {
			t.mu.Lock()
			running := t.running
			t.mu.Unlock()

			if !running {
				t.mu.Lock()
				exitCode := t.exitCode
				t.mu.Unlock()

				if exitCode != 0 {
					return TerminalExitMsg{ExitCode: exitCode, Err: fmt.Errorf("command failed with exit code %d", exitCode)}
				}
				return TerminalExitMsg{ExitCode: 0, Err: nil}
			}

			time.Sleep(pollInterval)
		}
	}
}

// TickForUpdate creates a command that periodically triggers view updates while running
func (t *TerminalAdapter) TickForUpdate() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return TerminalOutputMsg{} // Empty message just to trigger re-render
	})
}
