package msg_types

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	vast_client "github.com/vast-data/go-vast-client"
)

var spinnerCounter = Counter{1}

// SpinnerTickMsg is a message for updating the spinner
type SpinnerTickMsg string
type SpinnerStartMsg struct {
	SpinnerId int16
	SpinnerTs time.Time
}
type SpinnerStopMsg struct {
	SpinnerId int16
}

type SetDataMsg struct {
	UseSpinner bool // If true, spinner will be shown during data loading
}

type UpdateProfileMsg struct{}

// Ticker-specific message types for better separation
type TickerSetDataMsg struct{}

type TickerUpdateProfileMsg struct{}

// ProfileDataMsg is sent when profile data (space metrics) is updated
type ProfileDataMsg struct {
	AvailableSpace string
	UsedSpace      string
	FreeSpace      string
}

// DetailsContentMsg is sent when details content is loaded asynchronously
type DetailsContentMsg struct {
	Content      any
	ResourceType string
	Error        error
}

// SetResourceTypeMsg is sent when switching resource types
type SetResourceTypeMsg struct {
	ResourceType string
}

func ProcessWithSpinner(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	spinnerId := spinnerCounter.Value()
	spinnerTs := time.Now()
	spinnerCounter.Inc()

	startSpinner := func() tea.Msg {
		return SpinnerStartMsg{SpinnerId: spinnerId, SpinnerTs: spinnerTs}
	}
	stopSpinner := func() tea.Msg {
		return SpinnerStopMsg{SpinnerId: spinnerId}
	}
	return tea.Sequence(startSpinner, cmd, stopSpinner)
}

func ProcessWithSpinnerMust(cmd tea.Cmd, err error) tea.Cmd {
	if err != nil {
		return ProcessWithClearError(func() tea.Msg {
			return ErrorMsg{
				Err: err,
			}
		})
	}
	if cmd == nil {
		return nil
	}
	return ProcessWithSpinner(cmd)
}

func ProcessWithClearError(cmd tea.Cmd) tea.Cmd {
	clearError := func() tea.Msg {
		return ClearErrorMsg{}
	}

	if cmd == nil {
		return clearError
	}
	return tea.Sequence(clearError, cmd)
}

// ErrorMsg is a message for displaying errors
type ErrorMsg struct {
	Err error
}

// InfoMsg is a message for displaying info messages
type InfoMsg struct {
	Message string
	Tag     int // Used for debouncing
}

// ClearInfoMsg is a message for clearing info messages from status zone
type ClearInfoMsg struct{}

// InfoDebounceMsg is sent after delay to auto-clear info messages
type InfoDebounceMsg struct {
	Tag int // If this matches current tag, clear the info message
}

type MockMsg struct{}

// ClearErrorMsg is a message for clearing errors from status zone
type ClearErrorMsg struct{}

type InitProfileMsg struct {
	Client *vast_client.VMSRest
}

type Counter struct {
	value int16
}

func (c *Counter) Inc() {
	if c.value == 32767 {
		c.value = -32768
	} else {
		c.value++
	}

	if c.value == 0 {
		c.value++
	}
}

func (c *Counter) Dec() {
	if c.value == -32768 {
		c.value = 32767
	} else {
		c.value--
	}

	if c.value == 0 {
		c.value--
	}
}

func (c *Counter) Value() int16 {
	return c.value
}

// JSONEditedMsg is sent when the user finishes editing JSON in external editor
type JSONEditedMsg struct {
	JSON string
	Err  error
}
