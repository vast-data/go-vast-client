package main

import (
	"fmt"
	"os"

	"vastix/internal/database"
	"vastix/internal/tui/widgets/adapters"

	tea "github.com/charmbracelet/bubbletea"
)

// Simple demo showing how to use the terminal adapter.
// Run with: go run terminal_demo.go

type model struct {
	term          *adapters.TerminalAdapter
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
	var db *database.Service
	term := adapters.NewTerminalAdapter(db, "Terminal Demo")

	commands := []commandInfo{
		{name: "echo", displayName: "Echo Test", args: []string{"Hello from terminal widget!"}},
		{name: "ls", displayName: "List Files", args: []string{"-lah"}},
		{name: "date", displayName: "Show Date", args: []string{}},
		{name: "ps", displayName: "Process List", args: []string{"aux"}},
		{name: "ping", displayName: "Ping Test (5 pings)", args: []string{"-c", "5", "8.8.8.8"}},
	}

	return model{
		term:         term,
		commandIndex: 0,
		commands:     commands,
	}
}

func (m model) Init() tea.Cmd {
	if len(m.commands) > 0 {
		m.term.AddLine(fmt.Sprintf("🚀 Demo: %s", m.commands[m.commandIndex].displayName))
		m.term.AddLine("")
		return m.runCurrentCommand()
	}
	return nil
}

func (m model) runCurrentCommand() tea.Cmd {
	if m.commandIndex >= len(m.commands) {
		return nil
	}
	cmd := m.commands[m.commandIndex]
	return m.term.RunCommand(cmd.name, cmd.args...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.term.SetSize(m.width, m.height)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			m.term.ScrollUp()
			return m, nil
		case "down", "j":
			m.term.ScrollDown()
			return m, nil
		case "G":
			m.term.ScrollToBottom()
			return m, nil
		case "ctrl+l":
			m.term.Clear()
			return m, nil
		case "n":
			if !m.term.IsRunning() && m.commandIndex < len(m.commands)-1 {
				m.commandIndex++
				m.term.Clear()
				cmd := m.commands[m.commandIndex]
				m.term.AddLine(fmt.Sprintf("🚀 Demo: %s", cmd.displayName))
				m.term.AddLine("")
				return m, m.runCurrentCommand()
			}
			return m, nil
		case "p":
			if !m.term.IsRunning() && m.commandIndex > 0 {
				m.commandIndex--
				m.term.Clear()
				cmd := m.commands[m.commandIndex]
				m.term.AddLine(fmt.Sprintf("🚀 Demo: %s", cmd.displayName))
				m.term.AddLine("")
				return m, m.runCurrentCommand()
			}
			return m, nil
		case "r":
			if !m.term.IsRunning() {
				m.term.Clear()
				cmd := m.commands[m.commandIndex]
				m.term.AddLine(fmt.Sprintf("🔄 Rerunning: %s", cmd.displayName))
				m.term.AddLine("")
				return m, m.runCurrentCommand()
			}
			return m, nil
		case "?", "h":
			m.term.Clear()
			m.term.AddLine("📚 Terminal Widget Demo - Help")
			m.term.AddLine("")
			m.term.AddLine("Commands:")
			m.term.AddLine("  n      - Next command")
			m.term.AddLine("  p      - Previous command")
			m.term.AddLine("  r      - Restart current command")
			m.term.AddLine("  ↑/k    - Scroll up")
			m.term.AddLine("  ↓/j    - Scroll down")
			m.term.AddLine("  G      - Jump to bottom")
			m.term.AddLine("  Ctrl+L - Clear screen")
			m.term.AddLine("  ?/h    - Show this help")
			m.term.AddLine("  q      - Quit")
			return m, nil
		}

	case adapters.TerminalOutputMsg:
		if m.term.IsRunning() {
			return m, m.term.TickForUpdate()
		}
		return m, nil

	case adapters.TerminalExitMsg:
		m.term.AddLine("")
		if msg.ExitCode == 0 {
			m.term.AddLine("✅ Command completed successfully!")
		} else {
			m.term.AddLine(fmt.Sprintf("❌ Command failed with exit code %d", msg.ExitCode))
		}
		m.term.ScrollToBottom()
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}
	return m.term.View()
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
