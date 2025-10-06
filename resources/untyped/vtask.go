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

// AsyncResult represents the result of an asynchronous task.
// It contains the task's ID and necessary context for waiting on the task to complete.
type AsyncResult struct {
	TaskId int64
	Rest   core.VastRest
	ctx    context.Context
}

// NewAsyncResult creates a new AsyncResult from a task ID and REST client
func NewAsyncResult(ctx context.Context, taskId int64, rest core.VastRest) *AsyncResult {
	return &AsyncResult{
		ctx:    ctx,
		TaskId: taskId,
		Rest:   rest,
	}
}

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

// Wait blocks until the asynchronous task completes and returns the resulting Record.
// If the context (ar.ctx) is not set, it falls back to the context from the associated rest client.
func (ar *AsyncResult) Wait(timeout time.Duration) (core.Record, error) {
	ctx := ar.ctx
	if ctx == nil {
		ctx = ar.Rest.GetCtx()
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return ar.WaitWithContext(ctx)
}

// WaitWithContext blocks until the asynchronous task completes or the provided context is canceled.
// It delegates to the VTasks.WaitTaskWithContext method of the rest client to poll for task completion.
func (ar *AsyncResult) WaitWithContext(ctx context.Context) (core.Record, error) {
	return ar.Rest.GetResourceMap()["VTask"].(*VTask).WaitTaskWithContext(ctx, ar.TaskId)
}

func asyncResultFromRecord(ctx context.Context, r core.Record, rest core.VastRest) *AsyncResult {
	taskId := r.RecordID()
	return &AsyncResult{
		ctx:    ctx,
		TaskId: taskId,
		Rest:   rest,
	}
}
