package untyped

import (
	"context"
	"fmt"
	"time"

	"github.com/vast-data/go-vast-client/core"
)

type VTask struct {
	*core.VastResource
}

type (
	AsyncResult = core.AsyncResult
)

// WaitTaskWithContext polls the task status until it completes, fails, or times out.
//
// This method creates an AsyncResult and uses its Wait() method to poll the task status
// with exponential backoff. The AsyncResult.Success and AsyncResult.Err fields are updated
// based on the task outcome.
//
// Task states:
//   - "completed" → returns the task Record with AsyncResult.Success=true.
//   - "running"   → continues polling with exponential backoff.
//   - any other state → considered failure, returns error with AsyncResult.Success=false.
//
// Parameters:
//   - ctx: context for cancellation control.
//   - taskId: unique identifier of the task to wait for.
//   - timeout: maximum duration to wait for task completion (0 uses default of 10 minutes).
//
// Returns:
//   - Record: the completed task record, if successful.
//   - error: if the task failed, timeout occurred, or an API error occurred.
func (t *VTask) WaitTaskWithContext(ctx context.Context, taskId int64, timeout time.Duration) (core.Record, error) {
	if t == nil {
		return nil, fmt.Errorf("VTask is nil")
	}

	// Create AsyncResult and reuse its Wait() logic
	asyncResult := &core.AsyncResult{
		TaskId: taskId,
		Rest:   t.Rest,
		Ctx:    ctx,
	}

	return asyncResult.Wait(timeout)
}

// WaitTask polls the task status until it completes, fails, or times out.
//
// This is a convenience wrapper around WaitTaskWithContext that uses the bound REST context.
//
// Parameters:
//   - taskId: unique identifier of the task to wait for.
//   - timeout: maximum duration to wait for task completion (0 uses default of 10 minutes).
//
// Returns:
//   - Record: the completed task record, if successful.
//   - error: if the task failed, timeout occurred, or an API error occurred.
func (t *VTask) WaitTask(taskId int64, timeout time.Duration) (core.Record, error) {
	return t.WaitTaskWithContext(t.Rest.GetCtx(), taskId, timeout)
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
func MaybeWaitAsyncResult(record core.Record, rest core.VastRest, timeout time.Duration) (*AsyncResult, error) {
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
//   - ctx: The context to use for waiting (cancellation and timeout control)
//   - record: The record that may contain an async task response
//   - rest: The REST client to use for waiting on the task
//   - timeout: If 0, returns immediately without waiting. Otherwise, waits for task completion with the specified timeout.
//
// Returns:
//   - *AsyncResult: The async result if one was found, nil otherwise (Success and Err fields populated after wait)
//   - core.Record: The completed task record if timeout > 0 and task completed successfully, nil otherwise
//   - error: Any error that occurred during waiting (same as AsyncResult.Err)
func MaybeWaitAsyncResultWithContext(ctx context.Context, record core.Record, rest core.VastRest, timeout time.Duration) (*AsyncResult, error) {
	asyncResult := core.MaybeAsyncResultFromRecord(ctx, record, rest)
	if asyncResult == nil {
		return nil, nil
	}

	// If timeout is 0, return immediately (fire and forget)
	if timeout == 0 {
		return asyncResult, nil
	}

	// Wait for task completion using AsyncResult.Wait()
	// This will populate asyncResult.Success and asyncResult.Err
	_, err := asyncResult.Wait(timeout)

	return asyncResult, err
}
