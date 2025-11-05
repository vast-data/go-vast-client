package internal

import (
	"context"
	"sync"
	"time"
	"vastix/internal/logging"
	"vastix/internal/msg_types"
	"vastix/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

// SpinnerControl provides methods to control spinner behavior
type SpinnerControl struct {
	suspended chan bool
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewSpinnerControl creates a new spinner control instance
func NewSpinnerControl(ctx context.Context) *SpinnerControl {
	return &SpinnerControl{
		suspended: make(chan bool, 1),
		ctx:       ctx,
	}
}

// Suspend pauses the spinner
func (sc *SpinnerControl) Suspend() {
	select {
	case sc.suspended <- true:
	default:
		// Already suspended
	}
}

// Resume starts/resumes the spinner
func (sc *SpinnerControl) Resume() {
	select {
	case sc.suspended <- false:
	default:
		// Already resumed
	}
}

// TickerControl provides methods to control both data and profile tickers
//
// The TickerControl allows you to enable or disable both tickers simultaneously:
// - dataTicker: sends TickerSetDataMsg every 30 seconds
// - profileTicker: sends TickerUpdateProfileMsg every 5 minutes
//
// Usage:
//
//	tickerCtrl.Enable()   // Start both tickers
//	tickerCtrl.Disable()  // Stop both tickers
//
// The tickers start enabled by default.
type TickerControl struct {
	enabled chan bool
	ctx     context.Context
}

// NewTickerControl creates a new ticker control instance
func NewTickerControl(ctx context.Context) *TickerControl {
	return &TickerControl{
		enabled: make(chan bool, 1),
		ctx:     ctx,
	}
}

// Enable starts both tickers
func (tc *TickerControl) Enable() {
	select {
	case tc.enabled <- true:
	default:
		// Already enabled
	}
}

// Disable stops both tickers
func (tc *TickerControl) Disable() {
	select {
	case tc.enabled <- false:
	default:
		// Already disabled
	}
}

func SetupSubscriptions(ctx context.Context, cancel context.CancelFunc, ch chan tea.Msg) (*SpinnerControl, *TickerControl, func()) {
	wg := sync.WaitGroup{}
	auxlog := logging.GetAuxLogger()

	spinnerCtrl := NewSpinnerControl(ctx)
	tickerCtrl := NewTickerControl(ctx)

	// Start spinner with suspend/resume capability
	{
		spinner, sub := tui.StartSpinner(spinnerCtrl.ctx)
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer auxlog.Println("✅ Spinner goroutine terminated successfully")
			suspended := true // Start suspended, will be resumed by Run()
			auxlog.Println("🌀 Spinner goroutine started (suspended by default)")

			for {
				if suspended {
					// When suspended, only listen for suspend/resume commands and context cancellation
					select {
					case suspend := <-spinnerCtrl.suspended:
						suspended = suspend
						if !suspend {
							spinner.Reset() // Reset spinner state when resuming
						}
					case <-spinnerCtrl.ctx.Done():
						auxlog.Println("Spinner goroutine received context cancellation")
						return
					}
				} else {
					// When active, listen for spinner messages, suspend commands, and context cancellation
					select {
					case ev, ok := <-sub:
						if !ok {
							auxlog.Println("Spinner channel closed")
							return // Spinner channel closed
						}
						// Forward spinner message (no need to check suspended state here)
						select {
						case ch <- ev:
						case <-spinnerCtrl.ctx.Done():
							auxlog.Println("Spinner goroutine received context cancellation while forwarding")
							return
						}
					case suspend := <-spinnerCtrl.suspended:
						suspended = suspend
						if suspend {
							spinner.Reset() // Reset spinner state when suspending
						}
					case <-spinnerCtrl.ctx.Done():
						auxlog.Println("Spinner goroutine received context cancellation")
						return
					}
				}
			}
		}()
	}

	// Start ticker goroutine with enable/disable capability
	{
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer auxlog.Println("Ticker goroutine terminated successfully")

			// Create tickers for different intervals
			dataTicker := time.NewTicker(tui.DataTickerInterval)
			profileTicker := time.NewTicker(tui.ProfileTickerInterval)
			defer dataTicker.Stop()
			defer profileTicker.Stop()

			auxlog.Println("🕐 Tickers initialized")
			enabled := true // Start enabled by default
			auxlog.Println("Tickers enabled by default")

			for {
				if enabled {
					// When enabled, listen for ticker events and enable/disable commands
					select {
					case <-dataTicker.C:
						// Send TickerSetDataMsg every 30 seconds
						select {
						case ch <- msg_types.TickerSetDataMsg{}:
						case <-ctx.Done():
							auxlog.Println("Ticker goroutine received context cancellation while sending TickerSetDataMsg")
							return
						}
					case <-profileTicker.C:
						// Send TickerUpdateProfileMsg every 5 minutes
						select {
						case ch <- msg_types.TickerUpdateProfileMsg{}:
						case <-ctx.Done():
							return
						}
					case enable := <-tickerCtrl.enabled:
						enabled = enable
					case <-ctx.Done():
						auxlog.Println("Ticker goroutine received context cancellation")
						return
					}
				} else {
					// When disabled, only listen for enable/disable commands and context cancellation
					select {
					case enable := <-tickerCtrl.enabled:
						enabled = enable
					case <-ctx.Done():
						auxlog.Println("Ticker goroutine received context cancellation")
						return
					}
				}
			}
		}()
	}

	return spinnerCtrl, tickerCtrl, func() {
		auxlog.Println("Starting cleanup of background goroutines (spinner + ticker)")
		// Cancel context first to signal goroutines to stop
		cancel()
		auxlog.Println("Context cancelled, waiting for goroutines to terminate...")
		wg.Wait()
		close(ch)
		auxlog.Println("All background goroutines stopped successfully")
	}
}
