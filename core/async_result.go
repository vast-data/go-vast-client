package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

const VTaskKey = "VTask"

// AsyncResult represents the result of an asynchronous task.
// It contains the task's ID, completion status, and any error that occurred.
//
// Fields:
//   - TaskId: The unique identifier of the task
//   - Rest: The REST client used to query task status
//   - Ctx: The context associated with the task operation
//   - Success: True if task completed successfully, false if failed
//   - Err: The error that occurred during task execution (nil if successful)
type AsyncResult struct {
	TaskId  int64
	Rest    VastRest
	Ctx     context.Context
	Success bool
	Err     error
}

// IsFailed returns true if the task failed during execution.
func (ar *AsyncResult) IsFailed() bool {
	return !ar.Success
}

// IsSuccess returns true if the task completed successfully.
func (ar *AsyncResult) IsSuccess() bool {
	return ar.Success
}

// NewAsyncResult creates a new AsyncResult from a task ID and REST client.
//
// This constructor is used to create an AsyncResult when you already have a task ID.
// The context is stored for potential future use with waiting operations.
//
// Parameters:
//   - ctx: The context associated with the task operation
//   - taskId: The ID of the asynchronous task
//   - rest: The REST client that can be used to query task status
//
// Returns:
//   - *AsyncResult: A new AsyncResult instance
func NewAsyncResult(ctx context.Context, taskId int64, rest VastRest) *AsyncResult {
	return &AsyncResult{
		Ctx:    ctx,
		TaskId: taskId,
		Rest:   rest,
	}
}

// asyncTaskSuccessStates are terminal success values returned by VMS vtasks.
var asyncTaskSuccessStates = map[string]struct{}{
	"completed": {},
	"success":   {},
}

// asyncTaskFailureStates are terminal failure values returned by VMS vtasks.
var asyncTaskFailureStates = map[string]struct{}{
	"failed":    {},
	"error":     {},
	"cancelled": {},
	"canceled":  {},
}

// asyncTaskState returns the lower-cased task state (or status) from a vtask record.
func asyncTaskState(record Record) string {
	if record == nil {
		return ""
	}
	if v, ok := record["state"]; ok && v != nil && fmt.Sprintf("%v", v) != "" {
		return strings.ToLower(fmt.Sprintf("%v", v))
	}
	if v, ok := record["status"]; ok && v != nil {
		return strings.ToLower(fmt.Sprintf("%v", v))
	}
	return ""
}

func asyncTaskFailureMessage(record Record) string {
	if record == nil {
		return ""
	}
	for _, key := range []string{"failure_reason", "error", "message"} {
		if v := record[key]; v != nil {
			if s := strings.TrimSpace(fmt.Sprintf("%v", v)); s != "" && s != "<nil>" {
				return s
			}
		}
	}
	if rawMessages, ok := record["messages"].([]any); ok && len(rawMessages) > 0 {
		return strings.TrimSpace(fmt.Sprintf("%v", rawMessages[len(rawMessages)-1]))
	}
	return ""
}

// VerifyAsyncTaskRecord classifies a vtask poll result:
//   - completed/success → done
//   - failed/error/cancelled → abort with error
//   - any other state (running, pending, …) → keep polling
func VerifyAsyncTaskRecord(record Record) (bool, error) {
	state := asyncTaskState(record)
	if _, ok := asyncTaskSuccessStates[state]; ok {
		return true, nil
	}
	if _, ok := asyncTaskFailureStates[state]; ok {
		taskID := int64(0)
		if _, hasID := record["id"]; hasID {
			taskID = record.RecordID()
		}
		msg := asyncTaskFailureMessage(record)
		if msg == "" {
			return false, fmt.Errorf("async task %d failed: state=%s", taskID, state)
		}
		return false, fmt.Errorf("async task %d failed: state=%s, message: %s", taskID, state, msg)
	}
	return false, nil
}

// Wait polls the task status until it completes, fails, or times out.
//
// This method continuously polls the VTask resource to check the task's state.
// It uses exponential backoff with configurable intervals to avoid overwhelming the API.
// Transient network / availability errors (connection reset, 502/503/504, etc.) are
// retried until the overall timeout — common when VMS restarts mid-task.
// The Success and Err fields are updated based on the task outcome.
//
// Parameters:
//   - timeout: Maximum duration to wait for task completion (uses ar.Ctx for cancellation)
//
// Returns:
//   - Record: The final task record when completed
//   - error: An error if the task fails, times out, or a non-transient API error occurs
//
// Task states handled:
//   - "completed" / "success": Sets Success=true, returns the task record
//   - "failed" / "error" / "cancelled" / "canceled": Sets Success=false, returns error
//   - Any other state (running, pending, …): Continues polling with backoff
//
// After calling Wait(), check ar.Success and ar.Err for task execution results.
func (ar *AsyncResult) Wait(timeout time.Duration) (Record, error) {
	searchParams := Params{"id": ar.TaskId}
	waitAPIConditionConfig := &WaitAPIConditionConfig{
		Timeout: timeout,
	}

	record, err := WaitAPICondition(
		ar.Ctx,
		ar.Rest.GetResourceMap()[VTaskKey],
		searchParams,
		waitAPIConditionConfig,
		VerifyAsyncTaskRecord,
	)

	// Update AsyncResult fields based on outcome
	if err != nil {
		ar.Success = false
		ar.Err = err
	} else {
		ar.Success = true
		ar.Err = nil
	}

	return record, err
}

// MaybeAsyncResultFromRecord attempts to extract an async task ID from a record and create an AsyncResult.
//
// This function handles two common patterns in VAST API responses:
//  1. Direct task response: conventional vtasks self URL on the record
//  2. Nested task response: the record has an "async_task" field containing the task information
//
// If the record doesn't contain any task information, or if the task ID cannot be extracted,
// this function returns nil.
//
// Parameters:
//   - ctx: The context to associate with the async result
//   - record: The record that may contain async task information
//   - rest: The REST client for task operations
//
// Returns:
//   - *AsyncResult: An AsyncResult if task information was found, nil otherwise
func MaybeAsyncResultFromRecord(ctx context.Context, record Record, rest VastRest) *AsyncResult {
	var (
		taskId      int64
		asyncResult *AsyncResult
	)

	if record.Empty() {
		return nil
	}

	// Check if the record itself is a task (conventional vtasks URL).
	if isVTaskRecord(record) {
		if _, hasId := record["id"]; hasId {
			taskId = record.RecordID()
		}
	} else if asyncTask, ok := record["async_task"]; ok {
		var m map[string]any
		if m, ok = asyncTask.(map[string]any); ok {
			if _, hasId := m["id"]; hasId {
				taskId = ToRecord(m).RecordID()
			}
		}
	}

	if taskId != 0 {
		asyncResult = NewAsyncResult(ctx, taskId, rest)
	}

	return asyncResult

}

// WaitAPIConditionConfig defines retry/backoff parameters for polling operations.
//
// This configuration controls how WaitAPICondition polls an API endpoint,
// including timeout, polling intervals, and exponential backoff behavior.
//
// Fields:
//   - Timeout: Maximum duration to wait before giving up (default: 10 minutes)
//   - Interval: Initial polling interval between API calls (default: 500ms)
//   - MaxInterval: Maximum polling interval after backoff (default: 30 seconds)
//   - BackoffFactor: Multiplier for exponential backoff (default: 0.25 = 25% increase per iteration)
//   - TransientInterval: Sleep between retries when a poll hits a transient network /
//     availability error (default: 5 seconds)
//
// Example with custom configuration:
//
//	cfg := &WaitAPIConditionConfig{
//	    Timeout:       5 * time.Minute,
//	    Interval:      1 * time.Second,
//	    MaxInterval:   10 * time.Second,
//	    BackoffFactor: 0.5,  // 50% increase per iteration
//	}
//
// Zero values will be replaced with defaults by the normalize() method.
type WaitAPIConditionConfig struct {
	Timeout           time.Duration // Maximum total wait time
	Interval          time.Duration // Current/initial polling interval (mutated by NextInterval)
	MaxInterval       time.Duration // Cap for exponential backoff
	BackoffFactor     float64       // Rate of interval increase (0.25 = 25% per iteration)
	TransientInterval time.Duration // Sleep after transient poll errors
}

// normalize fills in missing (zero) values with sensible defaults.
//
// Default values:
//   - Timeout: 10 minutes
//   - Interval: 500 milliseconds
//   - MaxInterval: 30 seconds
//   - BackoffFactor: 0.25 (25% increase per iteration)
//   - TransientInterval: 5 seconds
//
// This method modifies the config in-place.
func (c *WaitAPIConditionConfig) normalize() {
	if c.Timeout == 0 {
		c.Timeout = 10 * time.Minute
	}
	if c.Interval == 0 {
		c.Interval = 500 * time.Millisecond
	}
	if c.MaxInterval == 0 {
		c.MaxInterval = 30 * time.Second
	}
	if c.BackoffFactor == 0 {
		c.BackoffFactor = 0.25
	}
	if c.TransientInterval == 0 {
		c.TransientInterval = 5 * time.Second
	}
}

// NextInterval returns the current interval and updates the internal state for the next iteration.
//
// This method implements exponential backoff by:
//  1. Returning the current interval value (for immediate use)
//  2. Calculating the next interval as: current * (1.0 + BackoffFactor)
//  3. Capping the next interval at MaxInterval
//  4. Updating c.Interval for the next call
//
// Example progression with Interval=500ms, BackoffFactor=0.25, MaxInterval=30s:
//   - 1st call: returns 500ms,  sets next to 625ms  (500 * 1.25)
//   - 2nd call: returns 625ms,  sets next to 781ms  (625 * 1.25)
//   - 3rd call: returns 781ms,  sets next to 976ms  (781 * 1.25)
//   - ...continues until reaching MaxInterval (30s)
//
// WARNING: This method mutates the config's Interval field. Do not reuse the same
// config instance for multiple concurrent polling operations.
func (c *WaitAPIConditionConfig) NextInterval() time.Duration {
	current := c.Interval

	// Calculate next interval for future calls using exponential backoff
	next := time.Duration(float64(c.Interval) * (1.0 + c.BackoffFactor))
	if next > c.MaxInterval {
		next = c.MaxInterval
	}
	c.Interval = next

	return current
}

// WaitAPICondition polls an API endpoint until a condition is met or timeout occurs.
//
// This is a generic polling function that repeatedly calls an API endpoint,
// applies a verification function to the result, and waits with exponential backoff
// between attempts until the condition is satisfied or a timeout is reached.
//
// Transient network / availability errors from the poll GET (see IsTransientPollError)
// are retried until Timeout; verification failures and non-transient API errors abort
// immediately.
//
// Parameters:
//   - ctx: The context for the operation (can be used for cancellation)
//   - caller: The resource API to poll (must support GetByIdWithContext or GetWithContext)
//   - searchParams: Parameters to identify the resource (if contains "id", uses GetById, otherwise Get)
//   - waitAPIConditionConfig: Configuration for timeout, intervals, and backoff (nil uses defaults)
//   - verifyFn: Function that checks if the condition is met. Returns (true, nil) when complete,
//     (false, nil) to continue polling, or (false, error) to abort with error.
//
// Returns:
//   - Record: The final record when the condition is met
//   - error: An error if verification fails, timeout occurs, or a non-transient API call fails
//
// Default configuration (when waitAPIConditionConfig is nil):
//   - Timeout: 10 minutes
//   - Initial Interval: 500ms
//   - Max Interval: 30 seconds
//   - Backoff Factor: 0.25 (25% increase per iteration)
//   - Transient Interval: 5 seconds
func WaitAPICondition(
	ctx context.Context,
	caller VastResourceAPIWithContext,
	searchParams Params,
	waitAPIConditionConfig *WaitAPIConditionConfig,
	verifyFn func(Record) (bool, error),
) (Record, error) {
	// Normalize config - use defaults if nil or zero values
	if waitAPIConditionConfig == nil {
		waitAPIConditionConfig = &WaitAPIConditionConfig{}
	}
	waitAPIConditionConfig.normalize()

	// Create a timeout context using the configured timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, waitAPIConditionConfig.Timeout)
	defer cancel()

	doneErr := func() error {
		if ctx.Err() != nil {
			return fmt.Errorf("WaitAPICondition cancelled: %w", ctx.Err())
		}
		return fmt.Errorf("WaitAPICondition timeout after %v", waitAPIConditionConfig.Timeout)
	}

	sleepOrDone := func(sleepFor time.Duration) error {
		timer := time.NewTimer(sleepFor)
		defer timer.Stop()
		select {
		case <-timeoutCtx.Done():
			return doneErr()
		case <-timer.C:
			return nil
		}
	}

	// Polling loop with exponential backoff
	for {
		if timeoutCtx.Err() != nil {
			return nil, doneErr()
		}

		var (
			record Record
			err    error
		)

		// Use GetById if "id" parameter is present, otherwise use Get with search params
		if id, ok := searchParams["id"]; ok {
			record, err = caller.GetByIdWithContext(timeoutCtx, id)
		} else {
			record, err = caller.GetWithContext(timeoutCtx, searchParams)
		}
		if err != nil {
			if IsTransientPollError(err) {
				if sleepErr := sleepOrDone(waitAPIConditionConfig.TransientInterval); sleepErr != nil {
					return nil, sleepErr
				}
				continue
			}
			return nil, fmt.Errorf("WaitAPICondition API call failed: %w", err)
		}

		// Check if condition is met
		completed, err := verifyFn(record)
		if err != nil {
			return nil, fmt.Errorf("WaitAPICondition verification failed: %w", err)
		}
		if completed {
			return record, nil
		}

		// Sleep for current interval, then bump interval for next iteration
		if sleepErr := sleepOrDone(waitAPIConditionConfig.NextInterval()); sleepErr != nil {
			return nil, sleepErr
		}
	}
}

// IsTransientPollError reports whether err is a temporary network / availability
// failure while polling an API (e.g. async VTask status). These are expected when
// VMS restarts mid-task (network_settings, compute cluster create, etc.).
//
// Non-retryable: context cancel/deadline, WaitAPICondition verification/timeout/
// cancel outcomes, and ApiError with a non-transient HTTP status.
func IsTransientPollError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	msg := err.Error()
	if strings.Contains(msg, "WaitAPICondition verification failed") ||
		strings.Contains(msg, "WaitAPICondition timeout") ||
		strings.Contains(msg, "WaitAPICondition cancelled") {
		return false
	}

	var apiErr *ApiError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 0, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		}
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return IsTransientPollError(urlErr.Err)
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	if errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ETIMEDOUT) {
		return true
	}

	lower := strings.ToLower(msg)
	for _, s := range []string{
		"connection reset by peer",
		"connection refused",
		"broken pipe",
		"i/o timeout",
		"tls: handshake timeout",
		"server closed idle connection",
		"use of closed network connection",
		"http2: client connection force closed",
		"http2: client connection lost",
		"eof",
	} {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}
