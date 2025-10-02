package core

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestParams_FromStruct tests the FromStruct method with various struct types
func TestParams_FromStruct(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected Params
		wantErr  bool
	}{
		{
			name: "simple struct with basic types",
			input: struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{Name: "John", Age: 30},
			expected: Params{"name": "John", "age": 30}, // Direct conversion, no JSON marshaling
			wantErr:  false,
		},
		{
			name: "struct with omitempty and zero values",
			input: struct {
				Name  string `json:"name,omitempty"`
				Age   int    `json:"age,omitempty"`
				Empty string `json:"empty,omitempty"`
			}{Name: "Jane", Age: 0, Empty: ""},
			expected: Params{"name": "Jane"},
			wantErr:  false,
		},
		{
			name: "struct with pointer fields",
			input: struct {
				Name *string `json:"name,omitempty"`
				Age  *int    `json:"age,omitempty"`
			}{Name: stringPtr("Alice"), Age: intPtr(25)},
			expected: Params{"name": "Alice", "age": 25},
			wantErr:  false,
		},
		{
			name: "struct with nested struct",
			input: struct {
				Name    string `json:"name"`
				Address struct {
					City string `json:"city"`
				} `json:"address"`
			}{
				Name: "Bob",
				Address: struct {
					City string `json:"city"`
				}{City: "NYC"},
			},
			expected: Params{"name": "Bob", "address": map[string]interface{}{"city": "NYC"}},
			wantErr:  false,
		},
		{
			name: "struct with slice",
			input: struct {
				Tags []string `json:"tags"`
			}{Tags: []string{"tag1", "tag2"}},
			expected: Params{"tags": []string{"tag1", "tag2"}},
			wantErr:  false,
		},
		{
			name: "struct with json dash (ignored field)",
			input: struct {
				Public  string `json:"public"`
				Private string `json:"-"`
			}{Public: "visible", Private: "hidden"},
			expected: Params{"public": "visible"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := make(Params)
			err := params.FromStruct(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("FromStruct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(params, tt.expected) {
				t.Errorf("FromStruct() got = %v, want %v", params, tt.expected)
			}
		})
	}
}

// TestNewParamsFromStruct tests the NewParamsFromStruct function
func TestNewParamsFromStruct(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected Params
		wantErr  bool
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: Params{},
			wantErr:  false,
		},
		{
			name: "simple struct",
			input: struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{Name: "Test", Age: 20},
			expected: Params{"name": "Test", "age": 20},
			wantErr:  false,
		},
		{
			name: "pointer to struct",
			input: &struct {
				Name string `json:"name"`
			}{Name: "Test"},
			expected: Params{"name": "Test"},
			wantErr:  false,
		},
		{
			name: "nil pointer",
			input: (*struct {
				Name string `json:"name"`
			})(nil),
			expected: Params{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewParamsFromStruct(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewParamsFromStruct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("NewParamsFromStruct() got = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestNewParamsFromStruct_RawData tests the RawData feature
func TestNewParamsFromStruct_RawData(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected Params
		wantErr  bool
	}{
		{
			name: "struct with empty RawData - uses typed fields",
			input: struct {
				Name    string `json:"name"`
				Age     int    `json:"age"`
				RawData Params `json:"-"`
			}{Name: "John", Age: 30, RawData: Params{}},
			expected: Params{"name": "John", "age": 30},
			wantErr:  false,
		},
		{
			name: "struct with nil RawData - uses typed fields",
			input: struct {
				Name    string `json:"name"`
				Age     int    `json:"age"`
				RawData Params `json:"-"`
			}{Name: "Jane", Age: 25, RawData: nil},
			expected: Params{"name": "Jane", "age": 25},
			wantErr:  false,
		},
		{
			name: "struct with RawData - ignores typed fields",
			input: struct {
				Name    string `json:"name"`
				Age     int    `json:"age"`
				RawData Params `json:"-"`
			}{
				Name:    "Ignored",
				Age:     999,
				RawData: Params{"custom__filter": "value", "path__contains": "/foo"},
			},
			expected: Params{"custom__filter": "value", "path__contains": "/foo"},
			wantErr:  false,
		},
		{
			name: "pointer to struct with RawData",
			input: &struct {
				Name    string `json:"name"`
				RawData Params `json:"-"`
			}{
				Name:    "Ignored",
				RawData: Params{"django__filter": "test"},
			},
			expected: Params{"django__filter": "test"},
			wantErr:  false,
		},
		{
			name: "struct with RawData containing complex values",
			input: struct {
				Name    string `json:"name"`
				RawData Params `json:"-"`
			}{
				Name: "Ignored",
				RawData: Params{
					"filter":   "value",
					"count":    10,
					"enabled":  true,
					"tags":     []string{"a", "b"},
					"metadata": map[string]string{"key": "value"},
				},
			},
			expected: Params{
				"filter":   "value",
				"count":    10,
				"enabled":  true,
				"tags":     []string{"a", "b"},
				"metadata": map[string]string{"key": "value"},
			},
			wantErr: false,
		},
		{
			name: "struct without RawData field - normal behavior",
			input: struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{Name: "Normal", Age: 40},
			expected: Params{"name": "Normal", "age": 40},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewParamsFromStruct(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewParamsFromStruct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("NewParamsFromStruct() got = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestRecord_Fill tests the Fill method on Record type
func TestRecord_Fill(t *testing.T) {
	tests := []struct {
		name     string
		record   Record
		target   interface{}
		validate func(interface{}) error
		wantErr  bool
	}{
		{
			name:   "fill simple struct",
			record: Record{"name": "John", "age": float64(30)},
			target: &struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{},
			validate: func(target interface{}) error {
				s := target.(*struct {
					Name string `json:"name"`
					Age  int    `json:"age"`
				})
				if s.Name != "John" || s.Age != 30 {
					t := &testing.T{}
					t.Errorf("expected Name='John', Age=30, got Name='%s', Age=%d", s.Name, s.Age)
					return nil
				}
				return nil
			},
			wantErr: false,
		},
		{
			name:   "fill struct with missing fields",
			record: Record{"name": "Jane"},
			target: &struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{},
			validate: func(target interface{}) error {
				s := target.(*struct {
					Name string `json:"name"`
					Age  int    `json:"age"`
				})
				if s.Name != "Jane" || s.Age != 0 {
					t := &testing.T{}
					t.Errorf("expected Name='Jane', Age=0, got Name='%s', Age=%d", s.Name, s.Age)
					return nil
				}
				return nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.record.Fill(tt.target)

			if (err != nil) != tt.wantErr {
				t.Errorf("Fill() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(tt.target)
			}
		})
	}
}

// TestParams_JSONSerialization tests JSON marshaling and unmarshaling
func TestParams_JSONSerialization(t *testing.T) {
	original := Params{
		"name":    "Test",
		"age":     30,
		"enabled": true,
		"tags":    []string{"a", "b", "c"},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal Params: %v", err)
	}

	// Unmarshal back to Params
	var result Params
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal Params: %v", err)
	}

	// Compare (note: numbers will be float64 after JSON roundtrip)
	expected := Params{
		"name":    "Test",
		"age":     float64(30),
		"enabled": true,
		"tags":    []interface{}{"a", "b", "c"},
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("JSON roundtrip failed, got = %v, want %v", result, expected)
	}
}

// TestRawDataExclusionFromJSON tests that RawData is excluded from JSON serialization
func TestRawDataExclusionFromJSON(t *testing.T) {
	type TestStruct struct {
		Name    string `json:"name"`
		Age     int    `json:"age"`
		RawData Params `json:"-" yaml:"-"`
	}

	input := TestStruct{
		Name:    "John",
		Age:     30,
		RawData: Params{"should": "not", "appear": "in", "json": "output"},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal struct: %v", err)
	}

	// Unmarshal to map to check contents
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify RawData fields are not present
	if _, exists := result["should"]; exists {
		t.Error("RawData content leaked into JSON output")
	}

	// Verify normal fields are present
	if result["name"] != "John" || result["age"] != float64(30) {
		t.Errorf("Expected name='John', age=30, got %v", result)
	}

	// Verify RawData field itself is not present
	if _, exists := result["RawData"]; exists {
		t.Error("RawData field itself appeared in JSON output")
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
