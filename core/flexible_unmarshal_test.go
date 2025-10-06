package core

import (
	"testing"
)

func TestFlexibleUnmarshal_NumberToString(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value string `json:"value"`
		Count int64  `json:"count"`
	}

	// JSON with a number in a string field
	jsonData := []byte(`{
		"name": "vb",
		"value": -1,
		"count": 888
	}`)

	var result TestStruct
	err := FlexibleUnmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name != "vb" {
		t.Errorf("expected Name to be 'vb', got %q", result.Name)
	}

	// This is the key test - number -1 should be converted to string "-1"
	if result.Value != "-1" {
		t.Errorf("expected Value to be '-1', got %q", result.Value)
	}

	if result.Count != 888 {
		t.Errorf("expected Count to be 888, got %d", result.Count)
	}
}

func TestFlexibleUnmarshal_BooleanToString(t *testing.T) {
	type TestStruct struct {
		Enabled string `json:"enabled"`
	}

	jsonData := []byte(`{"enabled": true}`)

	var result TestStruct
	err := FlexibleUnmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Enabled != "true" {
		t.Errorf("expected Enabled to be 'true', got %q", result.Enabled)
	}
}

func TestFlexibleUnmarshal_NestedStruct(t *testing.T) {
	type Nested struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}

	type TestStruct struct {
		Nested Nested `json:"nested"`
		Count  int64  `json:"count"`
	}

	// Nested struct with number in string field
	jsonData := []byte(`{
		"nested": {
			"name": "test",
			"value": 123
		},
		"count": 456
	}`)

	var result TestStruct
	err := FlexibleUnmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Nested.Name != "test" {
		t.Errorf("expected Nested.Name to be 'test', got %q", result.Nested.Name)
	}

	if result.Nested.Value != "123" {
		t.Errorf("expected Nested.Value to be '123', got %q", result.Nested.Value)
	}

	if result.Count != 456 {
		t.Errorf("expected Count to be 456, got %d", result.Count)
	}
}

func TestFlexibleUnmarshal_ArrayFields(t *testing.T) {
	type TestStruct struct {
		Names  []string `json:"names"`
		Values []string `json:"values"`
		Counts []int64  `json:"counts"`
	}

	// Array with mixed types in string array
	jsonData := []byte(`{
		"names": ["a", "b", "c"],
		"values": ["text", 123, true],
		"counts": [1, 2, 3]
	}`)

	var result TestStruct
	err := FlexibleUnmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Names) != 3 {
		t.Errorf("expected 3 names, got %d", len(result.Names))
	}

	if len(result.Values) != 3 {
		t.Errorf("expected 3 values, got %d", len(result.Values))
	}

	// Check that numbers and booleans were converted to strings
	if result.Values[0] != "text" {
		t.Errorf("expected Values[0] to be 'text', got %q", result.Values[0])
	}
	if result.Values[1] != "123" {
		t.Errorf("expected Values[1] to be '123', got %q", result.Values[1])
	}
	if result.Values[2] != "true" {
		t.Errorf("expected Values[2] to be 'true', got %q", result.Values[2])
	}

	// Numbers should remain numbers
	if len(result.Counts) != 3 || result.Counts[0] != 1 || result.Counts[1] != 2 || result.Counts[2] != 3 {
		t.Errorf("expected Counts to be [1, 2, 3], got %v", result.Counts)
	}
}

func TestFlexibleUnmarshal_NullValues(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}

	jsonData := []byte(`{
		"name": "test",
		"value": null
	}`)

	var result TestStruct
	err := FlexibleUnmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name != "test" {
		t.Errorf("expected Name to be 'test', got %q", result.Name)
	}

	// Null should become empty string
	if result.Value != "" {
		t.Errorf("expected Value to be empty, got %q", result.Value)
	}
}

func TestFlexibleUnmarshal_FloatToString(t *testing.T) {
	type TestStruct struct {
		Price string `json:"price"`
	}

	jsonData := []byte(`{"price": 123.456}`)

	var result TestStruct
	err := FlexibleUnmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Price != "123.456" {
		t.Errorf("expected Price to be '123.456', got %q", result.Price)
	}
}

func TestFlexibleUnmarshal_IntegerAsFloat(t *testing.T) {
	type TestStruct struct {
		Count string `json:"count"`
	}

	// JSON numbers are always floats in Go's json.Unmarshal
	jsonData := []byte(`{"count": 123.0}`)

	var result TestStruct
	err := FlexibleUnmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be "123" not "123.0"
	if result.Count != "123" {
		t.Errorf("expected Count to be '123', got %q", result.Count)
	}
}
