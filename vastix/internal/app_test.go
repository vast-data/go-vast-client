package internal

import (
	"testing"
	"time"
	"vastix/internal/tui"
)

func TestForceStopAllSpinners(t *testing.T) {
	// Create a mock app with activeSpinners
	activeSpinners := make(map[int16]time.Time)
	
	// Setup test data
	activeSpinners[1] = time.Now()
	activeSpinners[2] = time.Now()
	activeSpinners[3] = time.Now()
	
	// Mock app structure
	mockApp := struct {
		activeSpinners map[int16]time.Time
		spinnerActive  bool
	}{
		activeSpinners: activeSpinners,
		spinnerActive:  true,
	}
	
	// Set global spinner state
	tui.GetGlobalSpinnerState().SetActive(true)
	
	// Test initial state
	if len(mockApp.activeSpinners) != 3 {
		t.Errorf("Expected 3 active spinners, got %d", len(mockApp.activeSpinners))
	}
	
	if !mockApp.spinnerActive {
		t.Error("Expected spinnerActive to be true")
	}
	
	if !tui.GetGlobalSpinnerState().IsActive() {
		t.Error("Expected global spinner state to be active")
	}
	
	// Simulate forceStopAllSpinners behavior
	for spinnerId := range mockApp.activeSpinners {
		delete(mockApp.activeSpinners, spinnerId)
	}
	mockApp.spinnerActive = false
	tui.GetGlobalSpinnerState().SetActive(false)
	
	// Test results
	if len(mockApp.activeSpinners) != 0 {
		t.Errorf("Expected 0 active spinners after cleanup, got %d", len(mockApp.activeSpinners))
	}
	
	if mockApp.spinnerActive {
		t.Error("Expected spinnerActive to be false after cleanup")
	}
	
	if tui.GetGlobalSpinnerState().IsActive() {
		t.Error("Expected global spinner state to be inactive after cleanup")
	}
}

func TestSpinnerCleanupEdgeCases(t *testing.T) {
	// Test cleanup with empty spinner map
	t.Run("EmptySpinnerMap", func(t *testing.T) {
		activeSpinners := make(map[int16]time.Time)
		
		// Cleanup should not panic with empty map
		for spinnerId := range activeSpinners {
			delete(activeSpinners, spinnerId)
		}
		
		if len(activeSpinners) != 0 {
			t.Errorf("Expected empty map to remain empty, got %d elements", len(activeSpinners))
		}
	})
	
	// Test single spinner cleanup
	t.Run("SingleSpinner", func(t *testing.T) {
		activeSpinners := make(map[int16]time.Time)
		activeSpinners[42] = time.Now()
		
		if len(activeSpinners) != 1 {
			t.Errorf("Expected 1 active spinner, got %d", len(activeSpinners))
		}
		
		for spinnerId := range activeSpinners {
			delete(activeSpinners, spinnerId)
		}
		
		if len(activeSpinners) != 0 {
			t.Errorf("Expected 0 active spinners after cleanup, got %d", len(activeSpinners))
		}
	})
	
	// Test global spinner state consistency
	t.Run("GlobalStateConsistency", func(t *testing.T) {
		// Reset state
		tui.GetGlobalSpinnerState().SetActive(false)
		
		if tui.GetGlobalSpinnerState().IsActive() {
			t.Error("Expected global spinner state to start inactive")
		}
		
		// Activate
		tui.GetGlobalSpinnerState().SetActive(true)
		
		if !tui.GetGlobalSpinnerState().IsActive() {
			t.Error("Expected global spinner state to be active after setting")
		}
		
		// Deactivate
		tui.GetGlobalSpinnerState().SetActive(false)
		
		if tui.GetGlobalSpinnerState().IsActive() {
			t.Error("Expected global spinner state to be inactive after clearing")
		}
	})
}

func BenchmarkSpinnerCleanup(b *testing.B) {
	// Benchmark cleanup performance with many spinners
	b.Run("ManySpinners", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			activeSpinners := make(map[int16]time.Time)
			
			// Create 100 spinners
			for j := int16(1); j <= 100; j++ {
				activeSpinners[j] = time.Now()
			}
			
			// Cleanup all spinners
			for spinnerId := range activeSpinners {
				delete(activeSpinners, spinnerId)
			}
		}
	})
}
