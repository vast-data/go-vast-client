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
	"testing"
	"time"
)

// Mock implementations for testing
type mockVastRest struct {
	resourceMap map[string]ResourceEntry
}

func (m *mockVastRest) GetSession() RESTSession {
	return nil
}

func (m *mockVastRest) GetResourceMap() map[string]ResourceEntry {
	return m.resourceMap
}

func (m *mockVastRest) GetCtx() context.Context {
	return context.Background()
}

func (m *mockVastRest) SetCtx(ctx context.Context) {}

func (m *mockVastRest) GetApiRoot() string { return "" }

type mockVastResourceAPI struct {
	getByIdFunc func(ctx context.Context, id any) (Record, error)
	getFunc     func(ctx context.Context, params Params) (Record, error)
}

func (m *mockVastResourceAPI) Session() RESTSession                           { return nil }
func (m *mockVastResourceAPI) GetResourceType() string                        { return VTaskKey }
func (m *mockVastResourceAPI) GetResourcePath() string                        { return "/vtasks/" }
func (m *mockVastResourceAPI) GetApiRoot() string                             { return "" }
func (m *mockVastResourceAPI) List(Params) (RecordSet, error)                 { return nil, nil }
func (m *mockVastResourceAPI) Create(...Params) (Record, error)               { return nil, nil }
func (m *mockVastResourceAPI) Update(any, ...Params) (Record, error)          { return nil, nil }
func (m *mockVastResourceAPI) Delete(Params, ...Params) (Record, error)          { return nil, nil }
func (m *mockVastResourceAPI) DeleteById(any, ...Params) (Record, error) { return nil, nil }
func (m *mockVastResourceAPI) Ensure(Params, ...Params) (Record, error)       { return nil, nil }
func (m *mockVastResourceAPI) Get(Params) (Record, error)                     { return nil, nil }
func (m *mockVastResourceAPI) GetById(any, ...Params) (Record, error)         { return nil, nil }
func (m *mockVastResourceAPI) Exists(Params) (bool, error)                    { return false, nil }
func (m *mockVastResourceAPI) MustExists(Params) bool                         { return false }
func (m *mockVastResourceAPI) GetIterator(Params, int) Iterator               { return nil }
func (m *mockVastResourceAPI) Lock(...any) func()                             { return func() {} }

func (m *mockVastResourceAPI) ListWithContext(context.Context, Params) (RecordSet, error) {
	return nil, nil
}
func (m *mockVastResourceAPI) CreateWithContext(context.Context, ...Params) (Record, error) {
	return nil, nil
}
func (m *mockVastResourceAPI) UpdateWithContext(context.Context, any, ...Params) (Record, error) {
	return nil, nil
}
func (m *mockVastResourceAPI) DeleteWithContext(context.Context, Params, ...Params) (Record, error) {
	return nil, nil
}
func (m *mockVastResourceAPI) DeleteByIdWithContext(context.Context, any, ...Params) (Record, error) {
	return nil, nil
}
func (m *mockVastResourceAPI) EnsureWithContext(context.Context, Params, ...Params) (Record, error) {
	return nil, nil
}
func (m *mockVastResourceAPI) GetWithContext(ctx context.Context, params Params) (Record, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, params)
	}
	return nil, nil
}
func (m *mockVastResourceAPI) GetByIdWithContext(ctx context.Context, id any, _ ...Params) (Record, error) {
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
				"id":    123,
				"state": "completed",
				"url":   "https://l101/api/v5/vtasks/123/",
			}, nil
		},
	}

	rest := &mockVastRest{
		resourceMap: map[string]ResourceEntry{
			VTaskKey: ResourceEntry{VastResourceAPIWithContext: mockAPI},
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
				"id":       123,
				"name":     "test-task",
				"state":    "failed",
				"messages": []any{"Task execution error"},
				"url":      "https://l101/api/v5/vtasks/123/",
			}, nil
		},
	}

	rest := &mockVastRest{
		resourceMap: map[string]ResourceEntry{
			VTaskKey: ResourceEntry{VastResourceAPIWithContext: mockAPI},
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
				"id":    123,
				"name":  "test-task",
				"state": "error",
				"url":   "https://l101/api/v5/vtasks/123/",
			}, nil
		},
	}

	rest := &mockVastRest{
		resourceMap: map[string]ResourceEntry{
			VTaskKey: ResourceEntry{VastResourceAPIWithContext: mockAPI},
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
	if err != nil && !strings.Contains(err.Error(), "async task") {
		t.Errorf("Expected 'async task' in error, got: %v", err)
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
					"id":    123,
					"state": "running",
					"url":   "https://l101/api/v5/vtasks/123/",
				}, nil
			}
			return Record{
				"id":    123,
				"state": "completed",
				"url":   "https://l101/api/v5/vtasks/123/",
			}, nil
		},
	}

	rest := &mockVastRest{
		resourceMap: map[string]ResourceEntry{
			VTaskKey: ResourceEntry{VastResourceAPIWithContext: mockAPI},
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

func TestMaybeAsyncResultFromRecord_DirectTaskWithURL(t *testing.T) {
	ctx := context.Background()
	rest := &mockVastRest{}

	record := Record{
		"id":  int64(456),
		"url": "https://l101/api/v5/vtasks/456/",
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
		"url": "https://l101/api/v5/vtasks/456/",
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
		"id":  int64(456),
		"url": "https://l101/api/v5/views/456/",
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
	if config.TransientInterval != 5*time.Second {
		t.Errorf("Expected TransientInterval 5s, got %v", config.TransientInterval)
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
	if config.TransientInterval != 5*time.Second {
		t.Errorf("Expected TransientInterval 5s (default), got %v", config.TransientInterval)
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

func TestIsTransientPollError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"canceled", context.Canceled, false},
		{"deadline", context.DeadlineExceeded, false},
		{"verification failed", fmt.Errorf("WaitAPICondition verification failed: boom"), false},
		{"timeout", fmt.Errorf("WaitAPICondition timeout after 10m0s"), false},
		{"cancelled", fmt.Errorf("WaitAPICondition cancelled: %w", context.Canceled), false},
		{"generic", errors.New("task failed"), false},
		{"api 400", &ApiError{StatusCode: 400, Body: "bad"}, false},
		{"api 503", &ApiError{StatusCode: 503, Body: "unavailable"}, true},
		{"api unreachable", &ApiError{StatusCode: 0, Body: "unreachable"}, true},
		{
			"wrapped connection reset",
			fmt.Errorf("WaitAPICondition API call failed: %w",
				fmt.Errorf(`failed to perform GET request to https://v162:443/api/latest/vtasks/20/, error Get "https://v162:443/api/latest/vtasks/20/": read tcp 10.241.12.3:62794->10.141.200.162:443: connection reset by peer`)),
			true,
		},
		{
			"network_settings connection reset by peer",
			fmt.Errorf("WaitAPICondition API call failed: %w",
				fmt.Errorf(`failed to perform GET request to https://10.131.200.144:443/api/latest/vtasks/22/, error Get "https://10.131.200.144:443/api/latest/vtasks/22/": read tcp 10.241.166.85:65299->10.131.200.144:443: read: connection reset by peer`)),
			true,
		},
		{"eof", io.EOF, true},
		{"unexpected eof", io.ErrUnexpectedEOF, true},
		{"api 502", &ApiError{StatusCode: http.StatusBadGateway, Body: "bad gateway"}, true},
		{"api 504", &ApiError{StatusCode: http.StatusGatewayTimeout, Body: "timeout"}, true},
		{"net error", &net.DNSError{Err: "no such host", Name: "vms", IsNotFound: true}, true},
		{"op error", &net.OpError{Op: "read", Net: "tcp", Err: errors.New("reset")}, true},
		{"url error wraps eof", &url.Error{Op: "Get", URL: "https://vms/api", Err: io.EOF}, true},
		{"syscall econnreset", syscall.ECONNRESET, true},
		{"syscall econnrefused", syscall.ECONNREFUSED, true},
		{"syscall epipe", syscall.EPIPE, true},
		{"syscall etimedout", syscall.ETIMEDOUT, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTransientPollError(tt.err); got != tt.want {
				t.Errorf("IsTransientPollError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAsyncTaskFailureMessage_NilRecord(t *testing.T) {
	if got := asyncTaskFailureMessage(nil); got != "" {
		t.Fatalf("asyncTaskFailureMessage(nil) = %q, want empty", got)
	}
}

func TestVerifyAsyncTaskRecord(t *testing.T) {
	tests := []struct {
		name       string
		record     Record
		done       bool
		wantErr    bool
		errContain string
	}{
		{
			name:   "completed",
			record: Record{"id": int64(1), "state": "COMPLETED"},
			done:   true,
		},
		{
			name:   "success",
			record: Record{"id": int64(1), "state": "SUCCESS"},
			done:   true,
		},
		{
			name:   "running keeps polling",
			record: Record{"id": int64(1), "state": "RUNNING"},
			done:   false,
		},
		{
			name:   "pending keeps polling",
			record: Record{"id": int64(1), "state": "PENDING"},
			done:   false,
		},
		{
			name:   "empty state keeps polling",
			record: Record{"id": int64(1)},
			done:   false,
		},
		{
			name:   "nil record keeps polling",
			record: nil,
			done:   false,
		},
		{
			name:   "status fallback success",
			record: Record{"id": int64(1), "status": "success"},
			done:   true,
		},
		{
			name:       "failed with messages",
			record:     Record{"id": int64(1), "state": "FAILED", "messages": []any{"boom"}},
			done:       false,
			wantErr:    true,
			errContain: "boom",
		},
		{
			name:       "error with failure_reason",
			record:     Record{"id": int64(1), "state": "ERROR", "failure_reason": "nope"},
			done:       false,
			wantErr:    true,
			errContain: "nope",
		},
		{
			name:       "failed with error field",
			record:     Record{"id": int64(2), "state": "failed", "error": "disk full"},
			done:       false,
			wantErr:    true,
			errContain: "disk full",
		},
		{
			name:       "failed with message field",
			record:     Record{"id": int64(3), "state": "failed", "message": "quota exceeded"},
			done:       false,
			wantErr:    true,
			errContain: "quota exceeded",
		},
		{
			name:       "cancelled",
			record:     Record{"id": int64(1), "state": "CANCELLED", "messages": []any{"stopped"}},
			done:       false,
			wantErr:    true,
			errContain: "stopped",
		},
		{
			name:       "canceled us spelling",
			record:     Record{"id": int64(1), "state": "canceled"},
			done:       false,
			wantErr:    true,
			errContain: "state=canceled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done, err := VerifyAsyncTaskRecord(tt.record)
			if done != tt.done {
				t.Fatalf("done = %v, want %v", done, tt.done)
			}
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.errContain != "" && (err == nil || !strings.Contains(err.Error(), tt.errContain)) {
				t.Fatalf("err = %v, want substring %q", err, tt.errContain)
			}
		})
	}
}

func TestAsyncResult_Wait_SuccessState(t *testing.T) {
	ctx := context.Background()
	mockAPI := &mockVastResourceAPI{
		getByIdFunc: func(ctx context.Context, id any) (Record, error) {
			return Record{
				"id":    123,
				"state": "SUCCESS",
				"url":   "https://l101/api/v5/vtasks/123/",
			}, nil
		},
	}
	rest := &mockVastRest{
		resourceMap: map[string]ResourceEntry{
			VTaskKey: {VastResourceAPIWithContext: mockAPI},
		},
	}

	ar := NewAsyncResult(ctx, 123, rest)
	record, err := ar.Wait(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !ar.Success {
		t.Error("Expected Success to be true")
	}
	if record["state"] != "SUCCESS" {
		t.Errorf("Expected SUCCESS state, got %v", record["state"])
	}
}

func TestAsyncResult_Wait_PendingThenCompleted(t *testing.T) {
	ctx := context.Background()
	callCount := 0
	mockAPI := &mockVastResourceAPI{
		getByIdFunc: func(ctx context.Context, id any) (Record, error) {
			callCount++
			if callCount < 3 {
				return Record{"id": 123, "state": "pending"}, nil
			}
			return Record{"id": 123, "state": "completed"}, nil
		},
	}
	rest := &mockVastRest{
		resourceMap: map[string]ResourceEntry{
			VTaskKey: {VastResourceAPIWithContext: mockAPI},
		},
	}

	ar := NewAsyncResult(ctx, 123, rest)
	cfg := &WaitAPIConditionConfig{
		Timeout:       2 * time.Second,
		Interval:      10 * time.Millisecond,
		MaxInterval:   20 * time.Millisecond,
		BackoffFactor: 0.25,
	}
	record, err := WaitAPICondition(
		ar.Ctx,
		ar.Rest.GetResourceMap()[VTaskKey],
		Params{"id": ar.TaskId},
		cfg,
		VerifyAsyncTaskRecord,
	)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if callCount < 3 {
		t.Fatalf("Expected at least 3 calls, got %d", callCount)
	}
	if record["state"] != "completed" {
		t.Fatalf("Expected completed, got %v", record["state"])
	}
}

func TestWaitAPICondition_RetriesTransientError(t *testing.T) {
	ctx := context.Background()
	callCount := 0
	mockAPI := &mockVastResourceAPI{
		getByIdFunc: func(ctx context.Context, id any) (Record, error) {
			callCount++
			if callCount < 3 {
				return nil, errors.New("read: connection reset by peer")
			}
			return Record{"id": 123, "state": "completed"}, nil
		},
	}

	cfg := &WaitAPIConditionConfig{
		Timeout:           2 * time.Second,
		Interval:          10 * time.Millisecond,
		MaxInterval:       20 * time.Millisecond,
		TransientInterval: 10 * time.Millisecond,
		BackoffFactor:     0.25,
	}
	record, err := WaitAPICondition(
		ctx,
		mockAPI,
		Params{"id": int64(123)},
		cfg,
		VerifyAsyncTaskRecord,
	)
	if err != nil {
		t.Fatalf("Expected no error after transient retries, got %v", err)
	}
	if callCount < 3 {
		t.Fatalf("Expected at least 3 calls, got %d", callCount)
	}
	if record["state"] != "completed" {
		t.Fatalf("Expected completed, got %v", record["state"])
	}
}

func TestWaitAPICondition_AlreadyCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockAPI := &mockVastResourceAPI{
		getByIdFunc: func(ctx context.Context, id any) (Record, error) {
			t.Fatal("GetById should not be called when context is already cancelled")
			return nil, nil
		},
	}
	_, err := WaitAPICondition(
		ctx,
		mockAPI,
		Params{"id": 1},
		&WaitAPIConditionConfig{Timeout: time.Second, Interval: 10 * time.Millisecond},
		VerifyAsyncTaskRecord,
	)
	if err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("expected cancelled error, got %v", err)
	}
}

func TestWaitAPICondition_TransientErrorThenTimeout(t *testing.T) {
	ctx := context.Background()
	mockAPI := &mockVastResourceAPI{
		getByIdFunc: func(ctx context.Context, id any) (Record, error) {
			return nil, errors.New("connection reset by peer")
		},
	}
	_, err := WaitAPICondition(
		ctx,
		mockAPI,
		Params{"id": int64(1)},
		&WaitAPIConditionConfig{
			Timeout:           40 * time.Millisecond,
			TransientInterval: 200 * time.Millisecond,
			Interval:          10 * time.Millisecond,
		},
		VerifyAsyncTaskRecord,
	)
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected timeout while retrying transient errors, got %v", err)
	}
}

func TestWaitAPICondition_NonTransientAPIErrorDoesNotRetry(t *testing.T) {
	ctx := context.Background()
	callCount := 0
	mockAPI := &mockVastResourceAPI{
		getByIdFunc: func(ctx context.Context, id any) (Record, error) {
			callCount++
			return nil, &ApiError{StatusCode: http.StatusForbidden, Body: "denied"}
		},
	}

	cfg := &WaitAPIConditionConfig{
		Timeout:           500 * time.Millisecond,
		Interval:          10 * time.Millisecond,
		TransientInterval: 10 * time.Millisecond,
	}
	_, err := WaitAPICondition(ctx, mockAPI, Params{"id": int64(1)}, cfg, VerifyAsyncTaskRecord)
	if err == nil {
		t.Fatal("expected non-transient API error")
	}
	if callCount != 1 {
		t.Fatalf("expected exactly 1 call (no retry), got %d", callCount)
	}
	if !strings.Contains(err.Error(), "WaitAPICondition API call failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
