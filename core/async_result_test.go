package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// Mock implementations for testing
type mockVastRest struct {
	resourceMap map[string]VastResourceAPIWithContext
}

func (m *mockVastRest) GetSession() RESTSession {
	return nil
}

func (m *mockVastRest) GetResourceMap() map[string]VastResourceAPIWithContext {
	return m.resourceMap
}

func (m *mockVastRest) GetCtx() context.Context {
	return context.Background()
}

func (m *mockVastRest) SetCtx(ctx context.Context) {}

type mockVastResourceAPI struct {
	getByIdFunc func(ctx context.Context, id any) (Record, error)
	getFunc     func(ctx context.Context, params Params) (Record, error)
}

func (m *mockVastResourceAPI) Session() RESTSession                           { return nil }
func (m *mockVastResourceAPI) GetResourceType() string                        { return VTaskKey }
func (m *mockVastResourceAPI) GetResourcePath() string                        { return "/vtasks/" }
func (m *mockVastResourceAPI) List(Params) (RecordSet, error)                 { return nil, nil }
func (m *mockVastResourceAPI) Create(Params) (Record, error)                  { return nil, nil }
func (m *mockVastResourceAPI) Update(any, Params) (Record, error)             { return nil, nil }
func (m *mockVastResourceAPI) Delete(Params, Params) (Record, error)          { return nil, nil }
func (m *mockVastResourceAPI) DeleteById(any, Params, Params) (Record, error) { return nil, nil }
func (m *mockVastResourceAPI) Ensure(Params, Params) (Record, error)          { return nil, nil }
func (m *mockVastResourceAPI) Get(Params) (Record, error)                     { return nil, nil }
func (m *mockVastResourceAPI) GetById(any) (Record, error)                    { return nil, nil }
func (m *mockVastResourceAPI) Exists(Params) (bool, error)                    { return false, nil }
func (m *mockVastResourceAPI) MustExists(Params) bool                         { return false }
func (m *mockVastResourceAPI) GetIterator(Params, int) Iterator               { return nil }
func (m *mockVastResourceAPI) Lock(...any) func()                             { return func() {} }

func (m *mockVastResourceAPI) ListWithContext(context.Context, Params) (RecordSet, error) {
	return nil, nil
}
func (m *mockVastResourceAPI) CreateWithContext(context.Context, Params) (Record, error) {
	return nil, nil
}
func (m *mockVastResourceAPI) UpdateWithContext(context.Context, any, Params) (Record, error) {
	return nil, nil
}
func (m *mockVastResourceAPI) DeleteWithContext(context.Context, Params, Params, Params) (Record, error) {
	return nil, nil
}
func (m *mockVastResourceAPI) DeleteByIdWithContext(context.Context, any, Params, Params) (Record, error) {
	return nil, nil
}
func (m *mockVastResourceAPI) EnsureWithContext(context.Context, Params, Params) (Record, error) {
	return nil, nil
}
func (m *mockVastResourceAPI) GetWithContext(ctx context.Context, params Params) (Record, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, params)
	}
	return nil, nil
}
func (m *mockVastResourceAPI) GetByIdWithContext(ctx context.Context, id any) (Record, error) {
	if m.getByIdFunc != nil {
		return m.getByIdFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockVastResourceAPI) ExistsWithContext(context.Context, Params) (bool, error) {
	return false, nil
}
func (m *mockVastResourceAPI) MustExistsWithContext(context.Context, Params) bool { return false }
func (m *mockVastResourceAPI) GetIteratorWithContext(context.Context, Params, int) Iterator {
	return nil
}

// Tests for AsyncResult

func TestNewAsyncResult(t *testing.T) {
	ctx := context.Background()
	taskId := int64(12345)
	rest := &mockVastRest{}

	result := NewAsyncResult(ctx, taskId, rest)

	if result.TaskId != taskId {
		t.Errorf("Expected TaskId %d, got %d", taskId, result.TaskId)
	}
	if result.Ctx != ctx {
		t.Error("Context not set correctly")
	}
	if result.Rest != rest {
		t.Error("Rest not set correctly")
	}
	if result.Success {
		t.Error("Success should be false by default")
	}
	if result.Err != nil {
		t.Error("Err should be nil by default")
	}
}

func TestAsyncResult_IsFailed(t *testing.T) {
	tests := []struct {
		name    string
		success bool
		want    bool
	}{
		{"Failed task", false, true},
		{"Successful task", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ar := &AsyncResult{Success: tt.success}
			if got := ar.IsFailed(); got != tt.want {
				t.Errorf("IsFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAsyncResult_IsSuccess(t *testing.T) {
	tests := []struct {
		name    string
		success bool
		want    bool
	}{
		{"Successful task", true, true},
		{"Failed task", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ar := &AsyncResult{Success: tt.success}
			if got := ar.IsSuccess(); got != tt.want {
				t.Errorf("IsSuccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAsyncResult_Wait_Completed(t *testing.T) {
	ctx := context.Background()
	taskId := int64(123)

	mockAPI := &mockVastResourceAPI{
		getByIdFunc: func(ctx context.Context, id any) (Record, error) {
			return Record{
				"id":            123,
				"state":         "completed",
				ResourceTypeKey: VTaskKey,
			}, nil
		},
	}

	rest := &mockVastRest{
		resourceMap: map[string]VastResourceAPIWithContext{
			VTaskKey: mockAPI,
		},
	}

	ar := NewAsyncResult(ctx, taskId, rest)
	record, err := ar.Wait(100 * time.Millisecond)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !ar.Success {
		t.Error("Expected Success to be true")
	}
	if ar.Err != nil {
		t.Errorf("Expected Err to be nil, got %v", ar.Err)
	}
	if record["state"] != "completed" {
		t.Error("Expected completed state in record")
	}
}

func TestAsyncResult_Wait_FailedTask(t *testing.T) {
	ctx := context.Background()
	taskId := int64(123)

	mockAPI := &mockVastResourceAPI{
		getByIdFunc: func(ctx context.Context, id any) (Record, error) {
			return Record{
				"id":            123,
				"name":          "test-task",
				"state":         "failed",
				"messages":      []any{"Task execution error"},
				ResourceTypeKey: VTaskKey,
			}, nil
		},
	}

	rest := &mockVastRest{
		resourceMap: map[string]VastResourceAPIWithContext{
			VTaskKey: mockAPI,
		},
	}

	ar := NewAsyncResult(ctx, taskId, rest)
	_, err := ar.Wait(100 * time.Millisecond)

	if err == nil {
		t.Error("Expected error for failed task")
	}
	if ar.Success {
		t.Error("Expected Success to be false")
	}
	if ar.Err == nil {
		t.Error("Expected Err to be set")
	}
}

func TestAsyncResult_Wait_FailedTaskNoMessages(t *testing.T) {
	ctx := context.Background()
	taskId := int64(123)

	mockAPI := &mockVastResourceAPI{
		getByIdFunc: func(ctx context.Context, id any) (Record, error) {
			return Record{
				"id":            123,
				"name":          "test-task",
				"state":         "error",
				ResourceTypeKey: VTaskKey,
			}, nil
		},
	}

	rest := &mockVastRest{
		resourceMap: map[string]VastResourceAPIWithContext{
			VTaskKey: mockAPI,
		},
	}

	ar := NewAsyncResult(ctx, taskId, rest)
	_, err := ar.Wait(100 * time.Millisecond)

	if err == nil {
		t.Error("Expected error for failed task")
	}
	if !ar.IsFailed() {
		t.Error("Expected task to be failed")
	}
	if ar.Err == nil {
		t.Error("Expected Err to be set")
	}
	// Check error message contains expected text
	if err != nil && !strings.Contains(err.Error(), "no messages or unexpected format") {
		t.Errorf("Expected 'no messages' in error, got: %v", err)
	}
}

func TestAsyncResult_Wait_RunningThenCompleted(t *testing.T) {
	ctx := context.Background()
	taskId := int64(123)

	callCount := 0
	mockAPI := &mockVastResourceAPI{
		getByIdFunc: func(ctx context.Context, id any) (Record, error) {
			callCount++
			if callCount < 3 {
				return Record{
					"id":            123,
					"state":         "running",
					ResourceTypeKey: VTaskKey,
				}, nil
			}
			return Record{
				"id":            123,
				"state":         "completed",
				ResourceTypeKey: VTaskKey,
			}, nil
		},
	}

	rest := &mockVastRest{
		resourceMap: map[string]VastResourceAPIWithContext{
			VTaskKey: mockAPI,
		},
	}

	ar := NewAsyncResult(ctx, taskId, rest)

	// Use very short timeout and intervals for faster test
	config := &WaitAPIConditionConfig{
		Timeout:       5 * time.Second,
		Interval:      10 * time.Millisecond,
		MaxInterval:   50 * time.Millisecond,
		BackoffFactor: 0.25,
	}

	record, err := WaitAPICondition(
		ar.Ctx,
		ar.Rest.GetResourceMap()[VTaskKey],
		Params{"id": ar.TaskId},
		config,
		func(record Record) (bool, error) {
			state := fmt.Sprintf("%v", record["state"])
			if state == "completed" {
				return true, nil
			}
			return false, nil
		},
	)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if callCount < 3 {
		t.Errorf("Expected at least 3 calls, got %d", callCount)
	}
	if record["state"] != "completed" {
		t.Error("Expected completed state")
	}
}

// Tests for MaybeAsyncResultFromRecord

func TestMaybeAsyncResultFromRecord_EmptyRecord(t *testing.T) {
	ctx := context.Background()
	rest := &mockVastRest{}

	result := MaybeAsyncResultFromRecord(ctx, Record{}, rest)

	if result != nil {
		t.Error("Expected nil for empty record")
	}
}

func TestMaybeAsyncResultFromRecord_DirectTask(t *testing.T) {
	ctx := context.Background()
	rest := &mockVastRest{}

	record := Record{
		"id":            int64(456),
		ResourceTypeKey: VTaskKey,
	}

	result := MaybeAsyncResultFromRecord(ctx, record, rest)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.TaskId != 456 {
		t.Errorf("Expected TaskId 456, got %d", result.TaskId)
	}
}

func TestMaybeAsyncResultFromRecord_DirectTaskNoID(t *testing.T) {
	ctx := context.Background()
	rest := &mockVastRest{}

	record := Record{
		ResourceTypeKey: VTaskKey,
	}

	result := MaybeAsyncResultFromRecord(ctx, record, rest)

	if result != nil {
		t.Error("Expected nil for task without ID")
	}
}

func TestMaybeAsyncResultFromRecord_WrongResourceType(t *testing.T) {
	ctx := context.Background()
	rest := &mockVastRest{}

	record := Record{
		"id":            int64(456),
		ResourceTypeKey: "SomeOtherResource",
	}

	result := MaybeAsyncResultFromRecord(ctx, record, rest)

	if result != nil {
		t.Error("Expected nil for non-task resource")
	}
}

func TestMaybeAsyncResultFromRecord_NestedAsyncTask(t *testing.T) {
	ctx := context.Background()
	rest := &mockVastRest{}

	record := Record{
		"async_task": map[string]any{
			"id": int64(789),
		},
	}

	result := MaybeAsyncResultFromRecord(ctx, record, rest)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.TaskId != 789 {
		t.Errorf("Expected TaskId 789, got %d", result.TaskId)
	}
}

func TestMaybeAsyncResultFromRecord_NestedAsyncTaskNoID(t *testing.T) {
	ctx := context.Background()
	rest := &mockVastRest{}

	record := Record{
		"async_task": map[string]any{
			"name": "task",
		},
	}

	result := MaybeAsyncResultFromRecord(ctx, record, rest)

	if result != nil {
		t.Error("Expected nil for nested task without ID")
	}
}

func TestMaybeAsyncResultFromRecord_InvalidAsyncTaskType(t *testing.T) {
	ctx := context.Background()
	rest := &mockVastRest{}

	record := Record{
		"async_task": "not a map",
	}

	result := MaybeAsyncResultFromRecord(ctx, record, rest)

	if result != nil {
		t.Error("Expected nil for invalid async_task type")
	}
}

// Tests for WaitAPIConditionConfig

func TestWaitAPIConditionConfig_Normalize_AllDefaults(t *testing.T) {
	config := &WaitAPIConditionConfig{}
	config.normalize()

	if config.Timeout != 10*time.Minute {
		t.Errorf("Expected Timeout 10m, got %v", config.Timeout)
	}
	if config.Interval != 500*time.Millisecond {
		t.Errorf("Expected Interval 500ms, got %v", config.Interval)
	}
	if config.MaxInterval != 30*time.Second {
		t.Errorf("Expected MaxInterval 30s, got %v", config.MaxInterval)
	}
	if config.BackoffFactor != 0.25 {
		t.Errorf("Expected BackoffFactor 0.25, got %v", config.BackoffFactor)
	}
}

func TestWaitAPIConditionConfig_Normalize_PartialDefaults(t *testing.T) {
	config := &WaitAPIConditionConfig{
		Timeout:  5 * time.Minute,
		Interval: 1 * time.Second,
	}
	config.normalize()

	if config.Timeout != 5*time.Minute {
		t.Errorf("Expected Timeout 5m, got %v", config.Timeout)
	}
	if config.Interval != 1*time.Second {
		t.Errorf("Expected Interval 1s, got %v", config.Interval)
	}
	if config.MaxInterval != 30*time.Second {
		t.Errorf("Expected MaxInterval 30s (default), got %v", config.MaxInterval)
	}
	if config.BackoffFactor != 0.25 {
		t.Errorf("Expected BackoffFactor 0.25 (default), got %v", config.BackoffFactor)
	}
}

func TestWaitAPIConditionConfig_NextInterval(t *testing.T) {
	config := &WaitAPIConditionConfig{
		Interval:      100 * time.Millisecond,
		MaxInterval:   500 * time.Millisecond,
		BackoffFactor: 0.5,
	}

	// First call: returns 100ms, sets next to 150ms (100 * 1.5)
	interval1 := config.NextInterval()
	if interval1 != 100*time.Millisecond {
		t.Errorf("First interval: expected 100ms, got %v", interval1)
	}
	if config.Interval != 150*time.Millisecond {
		t.Errorf("After first call: expected interval 150ms, got %v", config.Interval)
	}

	// Second call: returns 150ms, sets next to 225ms (150 * 1.5)
	interval2 := config.NextInterval()
	if interval2 != 150*time.Millisecond {
		t.Errorf("Second interval: expected 150ms, got %v", interval2)
	}
	if config.Interval != 225*time.Millisecond {
		t.Errorf("After second call: expected interval 225ms, got %v", config.Interval)
	}
}

func TestWaitAPIConditionConfig_NextInterval_CapsAtMax(t *testing.T) {
	config := &WaitAPIConditionConfig{
		Interval:      400 * time.Millisecond,
		MaxInterval:   500 * time.Millisecond,
		BackoffFactor: 0.5,
	}

	// First call: returns 400ms, tries to set 600ms but caps at 500ms
	interval1 := config.NextInterval()
	if interval1 != 400*time.Millisecond {
		t.Errorf("Expected 400ms, got %v", interval1)
	}
	if config.Interval != 500*time.Millisecond {
		t.Errorf("Expected interval capped at 500ms, got %v", config.Interval)
	}

	// Second call: returns 500ms (at max), stays at 500ms
	interval2 := config.NextInterval()
	if interval2 != 500*time.Millisecond {
		t.Errorf("Expected 500ms, got %v", interval2)
	}
	if config.Interval != 500*time.Millisecond {
		t.Errorf("Expected interval to stay at 500ms, got %v", config.Interval)
	}
}

// Tests for WaitAPICondition

func TestWaitAPICondition_ImmediateSuccess(t *testing.T) {
	ctx := context.Background()

	mockAPI := &mockVastResourceAPI{
		getByIdFunc: func(ctx context.Context, id any) (Record, error) {
			return Record{"status": "ready"}, nil
		},
	}

	config := &WaitAPIConditionConfig{
		Timeout:  1 * time.Second,
		Interval: 100 * time.Millisecond,
	}

	record, err := WaitAPICondition(
		ctx,
		mockAPI,
		Params{"id": 1},
		config,
		func(r Record) (bool, error) {
			return r["status"] == "ready", nil
		},
	)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if record["status"] != "ready" {
		t.Error("Expected status to be ready")
	}
}

func TestWaitAPICondition_VerificationError(t *testing.T) {
	ctx := context.Background()

	mockAPI := &mockVastResourceAPI{
		getByIdFunc: func(ctx context.Context, id any) (Record, error) {
			return Record{"status": "error"}, nil
		},
	}

	config := &WaitAPIConditionConfig{
		Timeout:  1 * time.Second,
		Interval: 100 * time.Millisecond,
	}

	expectedErr := errors.New("verification failed")

	_, err := WaitAPICondition(
		ctx,
		mockAPI,
		Params{"id": 1},
		config,
		func(r Record) (bool, error) {
			return false, expectedErr
		},
	)

	if err == nil {
		t.Error("Expected error from verification function")
	}
	if !strings.Contains(err.Error(), "verification failed") {
		t.Errorf("Expected 'verification failed' in error, got: %v", err)
	}
}

func TestWaitAPICondition_Timeout(t *testing.T) {
	ctx := context.Background()

	mockAPI := &mockVastResourceAPI{
		getByIdFunc: func(ctx context.Context, id any) (Record, error) {
			return Record{"status": "pending"}, nil
		},
	}

	config := &WaitAPIConditionConfig{
		Timeout:  200 * time.Millisecond,
		Interval: 50 * time.Millisecond,
	}

	_, err := WaitAPICondition(
		ctx,
		mockAPI,
		Params{"id": 1},
		config,
		func(r Record) (bool, error) {
			// Never completes
			return false, nil
		},
	)

	if err == nil {
		t.Error("Expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected 'timeout' in error, got: %v", err)
	}
}

func TestWaitAPICondition_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	mockAPI := &mockVastResourceAPI{
		getByIdFunc: func(ctx context.Context, id any) (Record, error) {
			return Record{"status": "pending"}, nil
		},
	}

	config := &WaitAPIConditionConfig{
		Timeout:  5 * time.Second,
		Interval: 100 * time.Millisecond,
	}

	// Cancel context after a short delay
	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()

	_, err := WaitAPICondition(
		ctx,
		mockAPI,
		Params{"id": 1},
		config,
		func(r Record) (bool, error) {
			return false, nil
		},
	)

	if err == nil {
		t.Error("Expected cancellation error")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("Expected 'cancelled' in error, got: %v", err)
	}
}

func TestWaitAPICondition_APIError(t *testing.T) {
	ctx := context.Background()

	expectedErr := errors.New("API call failed")
	mockAPI := &mockVastResourceAPI{
		getByIdFunc: func(ctx context.Context, id any) (Record, error) {
			return nil, expectedErr
		},
	}

	config := &WaitAPIConditionConfig{
		Timeout:  1 * time.Second,
		Interval: 100 * time.Millisecond,
	}

	_, err := WaitAPICondition(
		ctx,
		mockAPI,
		Params{"id": 1},
		config,
		func(r Record) (bool, error) {
			return true, nil
		},
	)

	if err == nil {
		t.Error("Expected API error")
	}
	if !strings.Contains(err.Error(), "API call failed") {
		t.Errorf("Expected 'API call failed' in error, got: %v", err)
	}
}

func TestWaitAPICondition_NilConfig(t *testing.T) {
	ctx := context.Background()

	mockAPI := &mockVastResourceAPI{
		getByIdFunc: func(ctx context.Context, id any) (Record, error) {
			return Record{"status": "ready"}, nil
		},
	}

	record, err := WaitAPICondition(
		ctx,
		mockAPI,
		Params{"id": 1},
		nil, // nil config should use defaults
		func(r Record) (bool, error) {
			return r["status"] == "ready", nil
		},
	)

	if err != nil {
		t.Errorf("Expected no error with nil config, got %v", err)
	}
	if record["status"] != "ready" {
		t.Error("Expected status to be ready")
	}
}

func TestWaitAPICondition_UseGetInsteadOfGetById(t *testing.T) {
	ctx := context.Background()

	getCalled := false
	mockAPI := &mockVastResourceAPI{
		getFunc: func(ctx context.Context, params Params) (Record, error) {
			getCalled = true
			return Record{"status": "ready"}, nil
		},
	}

	config := &WaitAPIConditionConfig{
		Timeout:  1 * time.Second,
		Interval: 100 * time.Millisecond,
	}

	record, err := WaitAPICondition(
		ctx,
		mockAPI,
		Params{"name": "test"}, // No "id" param
		config,
		func(r Record) (bool, error) {
			return r["status"] == "ready", nil
		},
	)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !getCalled {
		t.Error("Expected Get to be called instead of GetById")
	}
	if record["status"] != "ready" {
		t.Error("Expected status to be ready")
	}
}
