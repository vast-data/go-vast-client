package untyped

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vast-data/go-vast-client/core"
)

type VTask struct {
	*core.VastResource
}

type (
	AsyncResult = core.AsyncResult
)

// nextBackoff returns the next polling interval using additive backoff strategy.
//
// It increases the current interval by 250ms up to a given max value.
//
// Parameters:
//   - current: the current polling interval.
//   - max: the maximum allowed interval.
//
// Returns:
//   - time.Duration: the next interval to wait before polling again.
func nextBackoff(current, max time.Duration) time.Duration {
	next := current + 250*time.Millisecond
	if next > max {
		return max
	}
	return next
}

// WaitTaskWithContext polls the task status until it completes, fails, or the context expires.
//
// It starts with a 500ms polling interval and increases it slightly after each attempt,
// using exponential-style backoff (capped at 5 seconds). This reduces the load on the API
// during long-running tasks.
//
// Task states:
//   - "completed" → returns the task Record.
//   - "running"   → continues polling.
//   - any other state → considered failure, and returns the last message from the task.
//
// If the context deadline is exceeded or canceled, the method returns an error with context cause.
//
// Parameters:
//   - ctx: context with optional timeout or cancellation.
//   - taskId: unique identifier of the task to wait for.
//
// Returns:
//   - Record: the completed task record, if successful.
//   - error: if the task failed, context expired, or an API error occurred.
func (t *VTask) WaitTaskWithContext(ctx context.Context, taskId int64) (core.Record, error) {
	if t == nil {
		return nil, fmt.Errorf("VTask is nil")
	}

	baseInterval := 500 * time.Millisecond
	maxInterval := 5 * time.Second
	currentInterval := baseInterval

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for task %d: %w", taskId, ctx.Err())

		default:
			task, err := t.GetByIdWithContext(ctx, taskId)
			if err != nil {
				return nil, err
			}

			state := strings.ToLower(fmt.Sprintf("%v", task["state"]))
			switch state {
			case "completed":
				return task, nil
			case "running":
				// backoff
				time.Sleep(currentInterval)
				currentInterval = nextBackoff(currentInterval, maxInterval)
			default:
				rawMessages := task["messages"]
				messages, ok := rawMessages.([]interface{})
				if !ok || len(messages) == 0 {
					return nil, fmt.Errorf("task %s failed with ID %d: no messages or unexpected format", task.RecordName(), task.RecordID())
				}
				lastMsg := fmt.Sprintf("%v", messages[len(messages)-1])
				return nil, fmt.Errorf("task %s failed with ID %d: %s", task.RecordName(), task.RecordID(), lastMsg)
			}
		}
	}
}

func (t *VTask) WaitTask(taskId int64, timeout time.Duration) (core.Record, error) {
	ctx, cancel := context.WithTimeout(t.Rest.GetCtx(), timeout)
	defer cancel()
	return t.WaitTaskWithContext(ctx, taskId)
}

// MaybeWaitAsyncResult checks if the record represents an async task result and optionally waits for it to complete.
// It uses the context from the rest client.
//
// Parameters:
//   - record: The record that may contain an async task response
//   - rest: The REST client to use for waiting on the task
//   - timeout: If 0, returns immediately without waiting. Otherwise, waits for task completion with the specified timeout.
//
// Returns:
//   - *AsyncResult: The async result if one was found, nil otherwise
//   - core.Record: The completed task record if timeout > 0 and task completed successfully, nil otherwise
//   - error: Any error that occurred during waiting
func MaybeWaitAsyncResult(record core.Record, rest core.VastRest, timeout time.Duration) (*AsyncResult, core.Record, error) {
	return MaybeWaitAsyncResultWithContext(rest.GetCtx(), record, rest, timeout)
}

// MaybeWaitAsyncResultWithContext checks if the record represents an async task result and optionally waits for it to complete.
// This is a utility function that combines async task detection and optional waiting into a single call.
//
// Behavior:
//   - If record is empty, returns (nil, nil, nil)
//   - If record does not contain an async task, returns (nil, nil, nil)
//   - If timeout is 0, returns immediately with (*AsyncResult, nil, nil) - async background operation
//   - If timeout > 0, waits for task completion and returns (*AsyncResult, taskRecord, error)
//
// Parameters:
//   - ctx: The context to use for waiting (will be wrapped with timeout if timeout > 0)
//   - record: The record that may contain an async task response
//   - rest: The REST client to use for waiting on the task
//   - timeout: If 0, returns immediately without waiting. Otherwise, waits for task completion with the specified timeout.
//
// Returns:
//   - *AsyncResult: The async result if one was found, nil otherwise
//   - core.Record: The completed task record if timeout > 0 and task completed successfully, nil otherwise
//   - error: Any error that occurred during waiting
func MaybeWaitAsyncResultWithContext(ctx context.Context, record core.Record, rest core.VastRest, timeout time.Duration) (*AsyncResult, core.Record, error) {
	var (
		asyncResult  *AsyncResult
		taskResponse core.Record
		err          error
	)
	asyncResult = core.MaybeAsyncResultFromRecord(ctx, record, rest)
	if asyncResult != nil && timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		taskResponse, err = rest.GetResourceMap()["VTask"].(*VTask).WaitTaskWithContext(ctx, asyncResult.TaskId)
		if err == nil && taskResponse != nil {
			if state, ok := taskResponse["state"]; ok {
				asyncResult.Status = state.(string)
			}
		}
	}

	return asyncResult, taskResponse, err

}
