package tui

import (
	"sync"
	log "vastix/internal/logging"

	"go.uber.org/zap"
)

// Global spinner state manager
var (
	globalSpinnerState *SpinnerState
	spinnerOnce        sync.Once
)

// SpinnerState manages the global spinner state
type SpinnerState struct {
	mu     sync.RWMutex
	active bool
}

// GetGlobalSpinnerState returns the singleton spinner state instance
func GetGlobalSpinnerState() *SpinnerState {
	spinnerOnce.Do(func() {
		globalSpinnerState = &SpinnerState{
			active: false,
		}
	})
	return globalSpinnerState
}

// SetActive sets the spinner active state
func (s *SpinnerState) SetActive(active bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active = active
	log.Debug("Global spinner state changed", zap.Bool("active", active))
}

// IsActive returns whether the spinner is currently active
func (s *SpinnerState) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}
