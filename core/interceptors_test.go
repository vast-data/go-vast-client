package core

import (
	"testing"
)

// Test that defaultResponseMutations no longer unwraps pagination envelopes
func TestDefaultResponseMutations_PaginationNotUnwrapped(t *testing.T) {
	// Create a pagination envelope
	paginationEnvelope := Record{
		"results": []any{
			map[string]any{"id": float64(1), "name": "item1"},
			map[string]any{"id": float64(2), "name": "item2"},
		},
		"count":    float64(2),
		"next":     "https://example.com/api/v1/resources/?page=2",
		"previous": nil,
	}

	// Apply defaultResponseMutations
	result, err := defaultResponseMutations(paginationEnvelope)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify that the pagination envelope is NOT unwrapped
	resultRecord, ok := result.(Record)
	if !ok {
		t.Fatalf("Expected Record, got %T", result)
	}

	// Check that the envelope structure is preserved
	if _, hasResults := resultRecord["results"]; !hasResults {
		t.Error("Expected results field to be preserved")
	}

	if _, hasCount := resultRecord["count"]; !hasCount {
		t.Error("Expected count field to be preserved")
	}

	if _, hasNext := resultRecord["next"]; !hasNext {
		t.Error("Expected next field to be preserved")
	}

	if _, hasPrevious := resultRecord["previous"]; !hasPrevious {
		t.Error("Expected previous field to be preserved")
	}
}

// Test that RecordSet is not altered by defaultResponseMutations
func TestDefaultResponseMutations_RecordSetUnchanged(t *testing.T) {
	recordSet := RecordSet{
		{"id": float64(1), "name": "item1"},
		{"id": float64(2), "name": "item2"},
	}

	// Apply defaultResponseMutations
	result, err := defaultResponseMutations(recordSet)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify that RecordSet is unchanged
	resultRecordSet, ok := result.(RecordSet)
	if !ok {
		t.Fatalf("Expected RecordSet, got %T", result)
	}

	if len(resultRecordSet) != 2 {
		t.Errorf("Expected 2 records, got %d", len(resultRecordSet))
	}
}

// Test that async_task response mutation still works
func TestDefaultResponseMutations_AsyncTask(t *testing.T) {
	asyncTaskResponse := Record{
		"async_task": map[string]any{
			"id":    float64(12345),
			"state": "RUNNING",
		},
	}

	// Apply defaultResponseMutations
	result, err := defaultResponseMutations(asyncTaskResponse)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify that async_task is normalized
	resultRecord, ok := result.(Record)
	if !ok {
		t.Fatalf("Expected Record, got %T", result)
	}

	// Check that ResourceTypeKey was added
	if resourceType, ok := resultRecord[ResourceTypeKey]; !ok || resourceType != "VTask" {
		t.Errorf("Expected ResourceTypeKey to be 'VTask', got: %v", resourceType)
	}
}
