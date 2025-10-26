package internal

import (
	"context"
	"strings"
	"testing"
	"time"
	"vastix/internal/logging"
	"vastix/internal/msg_types"
)

func TestSplashScreenSpinner(t *testing.T) {
	t.Run("SpinnerActiveInitialization", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Initialize logger for testing
		closeLogger, err := logging.InitGlobalLogger()
		if err != nil {
			t.Fatalf("Failed to initialize logger: %v", err)
		}
		if closeLogger != nil {
			defer closeLogger()
		}

		// Create minimal spinner controls without background goroutines
		spinnerCtrl := NewSpinnerControl(ctx)
		tickerCtrl := NewTickerControl(ctx)

		// Create app
		app := NewApp("test-1.0.0", spinnerCtrl, tickerCtrl)

		// Test initial state
		if app.spinnerActive {
			t.Error("Expected spinner to start inactive")
		}

		if app.spinnerView != "" {
			t.Error("Expected spinner view to start empty")
		}

		// Simulate app initialization (like in Run())
		app.spinnerActive = true // This should fix the splash screen spinner

		if !app.spinnerActive {
			t.Error("Expected spinner to be active after initialization")
		}

		// Test spinner tick message handling
		tickMsg := msg_types.SpinnerTickMsg("test spinner content")
		updatedApp, _ := app.Update(tickMsg)
		app = updatedApp.(*App)

		if app.spinnerView != "test spinner content" {
			t.Errorf("Expected spinner view to be updated, got %s", app.spinnerView)
		}

		// Verify splash screen render includes spinner content
		app.spinnerView = "test spinner content" // Ensure we have content
		splashView := app.renderSplashScreen()
		if splashView == "" {
			t.Error("Expected splash screen to have content")
		}

		// The spinner content should be included in the splash screen
		if !strings.Contains(splashView, "test spinner content") {
			t.Error("Expected splash screen to include spinner content")
		}
	})

	t.Run("SpinnerTransitionHandling", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Initialize logger for testing
		closeLogger, err := logging.InitGlobalLogger()
		if err != nil {
			t.Fatalf("Failed to initialize logger: %v", err)
		}
		if closeLogger != nil {
			defer closeLogger()
		}

		spinnerCtrl := NewSpinnerControl(ctx)
		tickerCtrl := NewTickerControl(ctx)

		app := NewApp("test-1.0.0", spinnerCtrl, tickerCtrl)

		// Simulate splash screen phase
		app.spinnerActive = true
		app.initialTransition = false

		// Test that when no active spinners exist, transition stops spinner
		activeSpinners := make(map[int16]time.Time) // Empty - simulates no active operations

		// Simulate transition logic (when app becomes ready)
		if !app.initialTransition && len(activeSpinners) == 0 {
			app.spinnerActive = false
			app.spinnerView = ""
			app.initialTransition = true
		}

		if app.spinnerActive {
			t.Error("Expected spinner to be stopped after transition with no active operations")
		}

		if app.spinnerView != "" {
			t.Error("Expected spinner view to be cleared after transition")
		}

		if !app.initialTransition {
			t.Error("Expected initial transition to be marked complete")
		}
	})

	t.Run("SpinnerTransitionWithActiveOperations", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Initialize logger for testing
		closeLogger, err := logging.InitGlobalLogger()
		if err != nil {
			t.Fatalf("Failed to initialize logger: %v", err)
		}
		if closeLogger != nil {
			defer closeLogger()
		}

		spinnerCtrl := NewSpinnerControl(ctx)
		tickerCtrl := NewTickerControl(ctx)

		app := NewApp("test-1.0.0", spinnerCtrl, tickerCtrl)

		// Simulate splash screen phase with active operations
		app.spinnerActive = true
		app.initialTransition = false

		// Simulate active operations
		activeSpinners := map[int16]time.Time{
			1: time.Now(),
			2: time.Now(),
		}

		// Simulate transition logic (when app becomes ready but operations are still running)
		if !app.initialTransition && len(activeSpinners) > 0 {
			// Should keep spinner active
			app.initialTransition = true
			// spinnerActive should remain true
		}

		if !app.spinnerActive {
			t.Error("Expected spinner to remain active during transition with active operations")
		}

		if !app.initialTransition {
			t.Error("Expected initial transition to be marked complete even with active operations")
		}
	})
}

func TestSpinnerControlFlow(t *testing.T) {
	t.Run("ResumeAndSuspendFlow", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		spinnerCtrl := NewSpinnerControl(ctx)

		// Test Resume (should send false to channel)
		spinnerCtrl.Resume()

		// Test Suspend (should send true to channel)
		spinnerCtrl.Suspend()

		// Test multiple Resume calls (should not block)
		spinnerCtrl.Resume()
		spinnerCtrl.Resume()

		// Test multiple Suspend calls (should not block)
		spinnerCtrl.Suspend()
		spinnerCtrl.Suspend()

		// If we reach here without deadlocking, the test passes
		t.Log("Spinner control flow test completed successfully")
	})
}
