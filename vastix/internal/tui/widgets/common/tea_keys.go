package common

import tea "github.com/charmbracelet/bubbletea"

// RemapRightArrowToTab converts right-arrow to tab for suggestion acceptance.
func RemapRightArrowToTab(msg tea.Msg, acceptSuggestion bool) tea.Msg {
	if !acceptSuggestion {
		return msg
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyRight {
		return tea.KeyMsg{Type: tea.KeyTab}
	}
	return msg
}

// KeyCode returns the key type for a key message, or 0 when msg is not a key event.
func KeyCode(msg tea.Msg) tea.KeyType {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return keyMsg.Type
	}
	return 0
}
