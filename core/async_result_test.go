package core

import (
	"context"
	"testing"
)

// mockVastRest is a mock implementation of VastRest for testing
type mockVastRest struct {
	resourceMap map[string]VastResourceAPIWithContext
	ctx         context.Context
}

func (m *mockVastRest) GetSession() RESTSession {
	return nil
}

func (m *mockVastRest) GetResourceMap() map[string]VastResourceAPIWithContext {
	return m.resourceMap
}

func (m *mockVastRest) GetCtx() context.Context {
	if m.ctx == nil {
		return context.Background()
	}
	return m.ctx
}

func (m *mockVastRest) SetCtx(ctx context.Context) {
	m.ctx = ctx
}

func TestNewAsyncResult(t *testing.T) {
	ctx := context.Background()
	taskId := int64(12345)
	rest := &mockVastRest{}

	result := NewAsyncResult(ctx, taskId, rest)

	if result == nil {
		t.Fatal("NewAsyncResult returned nil")
	}

	if result.TaskId != taskId {
		t.Errorf("Expected TaskId %d, got %d", taskId, result.TaskId)
	}

	if result.Rest != rest {
		t.Error("Expected Rest to match provided rest client")
	}

	if result.ctx != ctx {
		t.Error("Expected ctx to match provided context")
	}
}

func TestMaybeAsyncResultFromRecord_EmptyMap(t *testing.T) {
	ctx := context.Background()
	rest := &mockVastRest{}
	record := Record{}

	result := MaybeAsyncResultFromRecord(ctx, record, rest)

	if result != nil {
		t.Error("Expected nil for empty record, got non-nil")
	}
}

func TestMaybeAsyncResultFromRecord_DirectTaskResponse(t *testing.T) {
	ctx := context.Background()
	rest := &mockVastRest{}
	taskId := int64(67890)

	record := Record{
		ResourceTypeKey: "VTask",
		"id":            taskId,
	}

	result := MaybeAsyncResultFromRecord(ctx, record, rest)

	if result == nil {
		t.Fatal("Expected non-nil result for direct task response")
	}

	if result.TaskId != taskId {
		t.Errorf("Expected TaskId %d, got %d", taskId, result.TaskId)
	}

	if result.Rest != rest {
		t.Error("Expected Rest to match provided rest client")
	}
}

func TestMaybeAsyncResultFromRecord_NestedTaskResponse(t *testing.T) {
	ctx := context.Background()
	rest := &mockVastRest{}
	taskId := int64(99999)

	record := Record{
		"async_task": map[string]any{
			"id": taskId,
		},
	}

	result := MaybeAsyncResultFromRecord(ctx, record, rest)

	if result == nil {
		t.Fatal("Expected non-nil result for nested task response")
	}

	if result.TaskId != taskId {
		t.Errorf("Expected TaskId %d, got %d", taskId, result.TaskId)
	}

	if result.Rest != rest {
		t.Error("Expected Rest to match provided rest client")
	}
}

func TestMaybeAsyncResultFromRecord_NestedTaskResponse_FloatId(t *testing.T) {
	// JSON unmarshaling often produces float64 for numbers
	ctx := context.Background()
	rest := &mockVastRest{}
	taskId := float64(11111)

	record := Record{
		"async_task": map[string]any{
			"id": taskId,
		},
	}

	result := MaybeAsyncResultFromRecord(ctx, record, rest)

	if result == nil {
		t.Fatal("Expected non-nil result for nested task response with float ID")
	}

	expectedTaskId := int64(11111)
	if result.TaskId != expectedTaskId {
		t.Errorf("Expected TaskId %d, got %d", expectedTaskId, result.TaskId)
	}
}

func TestMaybeAsyncResultFromRecord_NoTaskInfo(t *testing.T) {
	ctx := context.Background()
	rest := &mockVastRest{}

	record := Record{
		"name":        "test",
		"description": "no task here",
	}

	result := MaybeAsyncResultFromRecord(ctx, record, rest)

	if result != nil {
		t.Error("Expected nil for record without task information")
	}
}

func TestMaybeAsyncResultFromRecord_InvalidAsyncTask(t *testing.T) {
	ctx := context.Background()
	rest := &mockVastRest{}

	// async_task is not a map
	record := Record{
		"async_task": "invalid",
	}

	result := MaybeAsyncResultFromRecord(ctx, record, rest)

	if result != nil {
		t.Error("Expected nil for record with invalid async_task format")
	}
}

func TestMaybeAsyncResultFromRecord_AsyncTaskWithoutId(t *testing.T) {
	ctx := context.Background()
	rest := &mockVastRest{}

	record := Record{
		"async_task": map[string]any{
			"status": "running",
			// no id field
		},
	}

	result := MaybeAsyncResultFromRecord(ctx, record, rest)

	if result != nil {
		t.Error("Expected nil for async_task without id")
	}
}

func TestMaybeAsyncResultFromRecord_ZeroTaskId(t *testing.T) {
	ctx := context.Background()
	rest := &mockVastRest{}

	record := Record{
		ResourceTypeKey: "VTask",
		"id":            int64(0),
	}

	result := MaybeAsyncResultFromRecord(ctx, record, rest)

	if result != nil {
		t.Error("Expected nil for zero task ID")
	}
}

func TestMaybeAsyncResultFromRecord_WithContext(t *testing.T) {
	type key string
	ctxKey := key("test")
	ctx := context.WithValue(context.Background(), ctxKey, "test-value")
	rest := &mockVastRest{}
	taskId := int64(12345)

	record := Record{
		ResourceTypeKey: "VTask",
		"id":            taskId,
	}

	result := MaybeAsyncResultFromRecord(ctx, record, rest)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Verify the context was preserved
	if result.ctx.Value(ctxKey) != "test-value" {
		t.Error("Expected context to be preserved in AsyncResult")
	}
}
