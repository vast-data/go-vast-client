package main

import (
	"fmt"
	"os"

	"vastix/internal/database"
	"vastix/internal/tui/widgets"
	"vastix/internal/tui/widgets/adapters"

	tea "github.com/charmbracelet/bubbletea"
)

// Simple demo showing how to use the terminal widget
// Run with: go run terminal_demo.go

type model struct {
	vpnWidget     *widgets.VPNTerminalWidget
	width, height int
	commandIndex  int
	commands      []commandInfo
}

type commandInfo struct {
	name        string
	displayName string
	args        []string
}

func initialModel() model {
	// Create database (pass nil for demo purposes, or initialize properly)
	var db *database.Service // nil is acceptable for this demo

	// Initialize VPN terminal widget
	vpnWidget := widgets.NewVPNTerminalWidget(db)

	// Define available commands to demo
	commands := []commandInfo{
		{name: "echo", displayName: "Echo Test", args: []string{"Hello from terminal widget!"}},
		{name: "ls", displayName: "List Files", args: []string{"-lah"}},
		{name: "date", displayName: "Show Date", args: []string{}},
		{name: "ps", displayName: "Process List", args: []string{"aux"}},
		{name: "ping", displayName: "Ping Test (5 pings)", args: []string{"-c", "5", "8.8.8.8"}},
		// Example wg-quick command (commented out - requires sudo)
		// {name: "sudo", displayName: "WireGuard Up", args: []string{"wg-quick", "up", "wg0"}},
	}

	return model{
		vpnWidget:    vpnWidget,
		commandIndex: 0,
		commands:     commands,
	}
}

func (m model) Init() tea.Cmd {
	// Run first command automatically
	if len(m.commands) > 0 {
		cmd := m.commands[m.commandIndex]
		m.vpnWidget.AddLine(fmt.Sprintf("🚀 Demo: %s", cmd.displayName))
		m.vpnWidget.AddLine("")
		return m.runCurrentCommand()
	}
	return nil
}

func (m model) runCurrentCommand() tea.Cmd {
	if m.commandIndex >= len(m.commands) {
		return nil
	}

	cmd := m.commands[m.commandIndex]
	return m.vpnWidget.RunCustomCommand(cmd.name, cmd.args...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.vpnWidget.SetSize(m.width, m.height)
		return m, nil

	case tea.KeyMsg:
		key := msg.String()

		// Navigation keys
		switch key {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			m.vpnWidget.ScrollUp()
			return m, nil

		case "down", "j":
			m.vpnWidget.ScrollDown()
			return m, nil

		case "G":
			m.vpnWidget.ScrollToBottom()
			return m, nil

		case "ctrl+l":
			m.vpnWidget.Clear()
			return m, nil

		case "n": // Next command
			if !m.vpnWidget.IsRunning() && m.commandIndex < len(m.commands)-1 {
				m.commandIndex++
				m.vpnWidget.Clear()
				cmd := m.commands[m.commandIndex]
				m.vpnWidget.AddLine(fmt.Sprintf("🚀 Demo: %s", cmd.displayName))
				m.vpnWidget.AddLine("")
				return m, m.runCurrentCommand()
			}
			return m, nil

		case "p": // Previous command
			if !m.vpnWidget.IsRunning() && m.commandIndex > 0 {
				m.commandIndex--
				m.vpnWidget.Clear()
				cmd := m.commands[m.commandIndex]
				m.vpnWidget.AddLine(fmt.Sprintf("🚀 Demo: %s", cmd.displayName))
				m.vpnWidget.AddLine("")
				return m, m.runCurrentCommand()
			}
			return m, nil

		case "r": // Restart current command
			if !m.vpnWidget.IsRunning() {
				m.vpnWidget.Clear()
				cmd := m.commands[m.commandIndex]
				m.vpnWidget.AddLine(fmt.Sprintf("🔄 Rerunning: %s", cmd.displayName))
				m.vpnWidget.AddLine("")
				return m, m.runCurrentCommand()
			}
			return m, nil

		case "?", "h": // Show help
			m.vpnWidget.Clear()
			m.vpnWidget.AddLine("📚 Terminal Widget Demo - Help")
			m.vpnWidget.AddLine("")
			m.vpnWidget.AddLine("Commands:")
			m.vpnWidget.AddLine("  n      - Next command")
			m.vpnWidget.AddLine("  p      - Previous command")
			m.vpnWidget.AddLine("  r      - Restart current command")
			m.vpnWidget.AddLine("  ↑/k    - Scroll up")
			m.vpnWidget.AddLine("  ↓/j    - Scroll down")
			m.vpnWidget.AddLine("  G      - Jump to bottom")
			m.vpnWidget.AddLine("  Ctrl+L - Clear screen")
			m.vpnWidget.AddLine("  ?/h    - Show this help")
			m.vpnWidget.AddLine("  q      - Quit")
			m.vpnWidget.AddLine("")
			m.vpnWidget.AddLine(fmt.Sprintf("Available commands (%d total):", len(m.commands)))
			for i, cmd := range m.commands {
				prefix := "  "
				if i == m.commandIndex {
					prefix = "► "
				}
				m.vpnWidget.AddLine(fmt.Sprintf("%s%d. %s", prefix, i+1, cmd.displayName))
			}
			m.vpnWidget.AddLine("")
			m.vpnWidget.AddLine("Press 'n' to run a command")
			return m, nil
		}

	case adapters.TerminalOutputMsg:
		// Command is producing output - continue ticking to update display
		if m.vpnWidget.IsRunning() {
			return m, m.vpnWidget.Navigate(msg)
		}
		return m, nil

	case adapters.TerminalExitMsg:
		// Command finished
		exitCode := msg.ExitCode
		m.vpnWidget.AddLine("")
		if exitCode == 0 {
			m.vpnWidget.AddLine("✅ Command completed successfully!")
		} else {
			m.vpnWidget.AddLine(fmt.Sprintf("❌ Command failed with exit code %d", exitCode))
		}
		m.vpnWidget.AddLine("")
		m.vpnWidget.AddLine("Press 'n' for next command, 'r' to rerun, 'h' for help")
		m.vpnWidget.ScrollToBottom()
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	return m.vpnWidget.View()
}

func main() {
	fmt.Println("Terminal Widget Demo")
	fmt.Println("====================")
	fmt.Println()
	fmt.Println("This demo shows how to use the terminal widget to execute commands")
	fmt.Println("and display their output in real-time within your TUI.")
	fmt.Println()
	fmt.Println("Controls:")
	fmt.Println("  n      - Next command")
	fmt.Println("  p      - Previous command")
	fmt.Println("  r      - Restart current command")
	fmt.Println("  ↑/k    - Scroll up")
	fmt.Println("  ↓/j    - Scroll down")
	fmt.Println("  ?/h    - Show help")
	fmt.Println("  q      - Quit")
	fmt.Println()
	fmt.Println("Starting demo...")
	fmt.Println()

	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
