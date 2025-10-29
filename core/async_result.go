package core

import (
	"context"
	"fmt"
	"strings"
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

// Wait polls the task status until it completes, fails, or times out.
//
// This method continuously polls the VTask resource to check the task's state.
// It uses exponential backoff with configurable intervals to avoid overwhelming the API.
// The Success and Err fields are updated based on the task outcome.
//
// Parameters:
//   - timeout: Maximum duration to wait for task completion (uses ar.Ctx for cancellation)
//
// Returns:
//   - Record: The final task record when completed
//   - error: An error if the task fails, times out, or an API error occurs
//
// Task states handled:
//   - "completed": Sets Success=true, returns the task record
//   - "running": Continues polling with backoff
//   - Any other state: Sets Success=false, Err=error, returns error with task messages
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
		func(record Record) (bool, error) {
			state := strings.ToLower(fmt.Sprintf("%v", record["state"]))
			switch state {
			case "completed":
				return true, nil
			case "running":
				// Continue polling with backoff
				return false, nil
			default:
				// Task failed or in unexpected state
				rawMessages := record["messages"]
				messages, ok := rawMessages.([]any)
				if !ok || len(messages) == 0 {
					taskErr := fmt.Errorf("task %s failed with ID %d: state=%s, no messages or unexpected format",
						record.RecordName(), record.RecordID(), state)
					return false, taskErr
				}
				lastMsg := fmt.Sprintf("%v", messages[len(messages)-1])
				taskErr := fmt.Errorf("task %s failed with ID %d: state=%s, message: %s",
					record.RecordName(), record.RecordID(), state, lastMsg)
				return false, taskErr
			}
		},
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
//  1. Direct task response: The record itself has a ResourceTypeKey and represents the task
//  2. Nested task response: The record has an "async_task" field containing the task information
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

	// Check if the record itself is a task (has ResourceTypeKey)
	if resourceType, ok := record[ResourceTypeKey]; ok {
		if resourceType != VTaskKey {
			return nil
		}

		// Only call RecordID if "id" field exists to avoid panic
		if _, hasId := record["id"]; hasId {
			taskId = record.RecordID()
		}
	} else {
		// Check for nested async_task field
		if asyncTask, ok := record["async_task"]; ok {
			var m map[string]any
			if m, ok = asyncTask.(map[string]any); ok {
				if _, hasId := m["id"]; hasId {
					taskId = ToRecord(m).RecordID()
				}
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
	Timeout       time.Duration // Maximum total wait time
	Interval      time.Duration // Current/initial polling interval (mutated by NextInterval)
	MaxInterval   time.Duration // Cap for exponential backoff
	BackoffFactor float64       // Rate of interval increase (0.25 = 25% per iteration)
}

// normalize fills in missing (zero) values with sensible defaults.
//
// Default values:
//   - Timeout: 10 minutes
//   - Interval: 500 milliseconds
//   - MaxInterval: 30 seconds
//   - BackoffFactor: 0.25 (25% increase per iteration)
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
//   - error: An error if verification fails, timeout occurs, or API call fails
//
// Default configuration (when waitAPIConditionConfig is nil):
//   - Timeout: 10 minutes
//   - Initial Interval: 500ms
//   - Max Interval: 30 seconds
//   - Backoff Factor: 0.25 (25% increase per iteration)
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

	// Polling loop with exponential backoff
	for {
		select {
		case <-timeoutCtx.Done():
			// Check if it's a timeout or cancellation
			if ctx.Err() != nil {
				return nil, fmt.Errorf("WaitAPICondition cancelled: %w", ctx.Err())
			}
			return nil, fmt.Errorf("WaitAPICondition timeout after %v", waitAPIConditionConfig.Timeout)

		default:
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
			sleepFor := waitAPIConditionConfig.NextInterval()
			time.Sleep(sleepFor)
		}
	}
}
