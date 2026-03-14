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

func TestFlexibleUnmarshal_StringToBool(t *testing.T) {
	type TestStruct struct {
		Enabled bool `json:"enabled"`
	}

	tests := []struct {
		name     string
		json     string
		expected bool
	}{
		{"string true", `{"enabled": "true"}`, true},
		{"string false", `{"enabled": "false"}`, false},
		{"string 1", `{"enabled": "1"}`, true},
		{"string 0", `{"enabled": "0"}`, false},
		{"bool true", `{"enabled": true}`, true},
		{"bool false", `{"enabled": false}`, false},
		{"int 1", `{"enabled": 1}`, true},
		{"int 0", `{"enabled": 0}`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result TestStruct
			err := FlexibleUnmarshal([]byte(tt.json), &result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Enabled != tt.expected {
				t.Errorf("expected Enabled to be %v, got %v", tt.expected, result.Enabled)
			}
		})
	}
}

func TestFlexibleUnmarshal_NestedBoolConversion(t *testing.T) {
	type Nested struct {
		Active bool `json:"active"`
	}

	type TestStruct struct {
		Config Nested `json:"config"`
	}

	jsonData := []byte(`{
		"config": {
			"active": "true"
		}
	}`)

	var result TestStruct
	err := FlexibleUnmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Config.Active {
		t.Errorf("expected Config.Active to be true, got false")
	}
}

func TestFlexibleUnmarshal_ArrayOfBools(t *testing.T) {
	type TestStruct struct {
		Flags []bool `json:"flags"`
	}

	jsonData := []byte(`{
		"flags": ["true", "false", "1", "0", true, false]
	}`)

	var result TestStruct
	err := FlexibleUnmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []bool{true, false, true, false, true, false}
	if len(result.Flags) != len(expected) {
		t.Fatalf("expected %d flags, got %d", len(expected), len(result.Flags))
	}

	for i, exp := range expected {
		if result.Flags[i] != exp {
			t.Errorf("expected Flags[%d] to be %v, got %v", i, exp, result.Flags[i])
		}
	}
}

func TestFlexibleUnmarshal_UnparseableStringToNumeric(t *testing.T) {
	type TestStruct struct {
		EstimatedTime float32 `json:"estimated_read_only_time"`
		Count         int     `json:"count"`
		Size          int64   `json:"size"`
		Ratio         float64 `json:"ratio"`
	}

	// JSON with unparseable string values in numeric fields
	jsonData := []byte(`{
		"estimated_read_only_time": "UNKNOWN",
		"count": "INVALID",
		"size": "N/A",
		"ratio": "undefined"
	}`)

	var result TestStruct
	err := FlexibleUnmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All unparseable strings should become zero values
	if result.EstimatedTime != 0 {
		t.Errorf("expected EstimatedTime to be 0, got %f", result.EstimatedTime)
	}
	if result.Count != 0 {
		t.Errorf("expected Count to be 0, got %d", result.Count)
	}
	if result.Size != 0 {
		t.Errorf("expected Size to be 0, got %d", result.Size)
	}
	if result.Ratio != 0 {
		t.Errorf("expected Ratio to be 0, got %f", result.Ratio)
	}
}

func TestFlexibleUnmarshal_ParseableStringToNumeric(t *testing.T) {
	type TestStruct struct {
		Count   int     `json:"count"`
		Size    int64   `json:"size"`
		Ratio   float32 `json:"ratio"`
		Percent float64 `json:"percent"`
	}

	// JSON with parseable string values in numeric fields
	jsonData := []byte(`{
		"count": "42",
		"size": "9876543210",
		"ratio": "3.14",
		"percent": "99.99"
	}`)

	var result TestStruct
	err := FlexibleUnmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parseable strings should be converted to their numeric values
	if result.Count != 42 {
		t.Errorf("expected Count to be 42, got %d", result.Count)
	}
	if result.Size != 9876543210 {
		t.Errorf("expected Size to be 9876543210, got %d", result.Size)
	}
	if result.Ratio != 3.14 {
		t.Errorf("expected Ratio to be 3.14, got %f", result.Ratio)
	}
	if result.Percent != 99.99 {
		t.Errorf("expected Percent to be 99.99, got %f", result.Percent)
	}
}

func TestFlexibleUnmarshal_MixedNumericTypes(t *testing.T) {
	type TestStruct struct {
		Int8Val   int8    `json:"int8_val"`
		Int16Val  int16   `json:"int16_val"`
		Int32Val  int32   `json:"int32_val"`
		Uint8Val  uint8   `json:"uint8_val"`
		Uint16Val uint16  `json:"uint16_val"`
		Uint32Val uint32  `json:"uint32_val"`
		Uint64Val uint64  `json:"uint64_val"`
		Float32   float32 `json:"float32_val"`
	}

	// Mix of valid numbers, valid strings, and invalid strings
	jsonData := []byte(`{
		"int8_val": 127,
		"int16_val": "32000",
		"int32_val": "INVALID",
		"uint8_val": 255,
		"uint16_val": "65000",
		"uint32_val": "N/A",
		"uint64_val": "18446744073709551615",
		"float32_val": "UNKNOWN"
	}`)

	var result TestStruct
	err := FlexibleUnmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Int8Val != 127 {
		t.Errorf("expected Int8Val to be 127, got %d", result.Int8Val)
	}
	if result.Int16Val != 32000 {
		t.Errorf("expected Int16Val to be 32000, got %d", result.Int16Val)
	}
	if result.Int32Val != 0 {
		t.Errorf("expected Int32Val to be 0 (unparseable), got %d", result.Int32Val)
	}
	if result.Uint8Val != 255 {
		t.Errorf("expected Uint8Val to be 255, got %d", result.Uint8Val)
	}
	if result.Uint16Val != 65000 {
		t.Errorf("expected Uint16Val to be 65000, got %d", result.Uint16Val)
	}
	if result.Uint32Val != 0 {
		t.Errorf("expected Uint32Val to be 0 (unparseable), got %d", result.Uint32Val)
	}
	if result.Uint64Val != 18446744073709551615 {
		t.Errorf("expected Uint64Val to be 18446744073709551615, got %d", result.Uint64Val)
	}
	if result.Float32 != 0 {
		t.Errorf("expected Float32 to be 0 (unparseable), got %f", result.Float32)
	}
}

func TestFlexibleUnmarshal_ProtectedPathExample(t *testing.T) {
	// Simulate the real-world ProtectedPath case from the issue
	type ProtectedPath struct {
		ID                   int     `json:"id"`
		Name                 string  `json:"name"`
		EstimatedReadOnlyTime float32 `json:"estimated_read_only_time"`
		State                string  `json:"state"`
	}

	jsonData := []byte(`{
		"id": 6,
		"name": "vo-b816a408a6",
		"estimated_read_only_time": "UNKNOWN",
		"state": "Active"
	}`)

	var result ProtectedPath
	err := FlexibleUnmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != 6 {
		t.Errorf("expected ID to be 6, got %d", result.ID)
	}
	if result.Name != "vo-b816a408a6" {
		t.Errorf("expected Name to be 'vo-b816a408a6', got %q", result.Name)
	}
	if result.EstimatedReadOnlyTime != 0 {
		t.Errorf("expected EstimatedReadOnlyTime to be 0 (UNKNOWN string), got %f", result.EstimatedReadOnlyTime)
	}
	if result.State != "Active" {
		t.Errorf("expected State to be 'Active', got %q", result.State)
	}
}
