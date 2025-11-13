package core

import (
	"bytes"
	"io"
	"net/http"
	"reflect"
	"testing"
)

// Helper function to create a mock HTTP response
func createMockResponse(body string, statusCode int, contentLength int64) *http.Response {
	return &http.Response{
		StatusCode:    statusCode,
		Body:          io.NopCloser(bytes.NewBufferString(body)),
		ContentLength: contentLength,
		Header:        make(http.Header),
	}
}

// TestUnmarshalToRecordUnion_ArrayOfObjects tests unmarshaling arrays of objects
func TestUnmarshalToRecordUnion_ArrayOfObjects(t *testing.T) {
	tests := []struct {
		name     string
		jsonBody string
		want     RecordSet
	}{
		{
			name:     "array of simple objects",
			jsonBody: `[{"id": 1, "name": "foo"}, {"id": 2, "name": "bar"}]`,
			want: RecordSet{
				{"id": float64(1), "name": "foo"},
				{"id": float64(2), "name": "bar"},
			},
		},
		{
			name:     "array of nested objects",
			jsonBody: `[{"id": 1, "data": {"nested": true}}, {"id": 2, "data": {"nested": false}}]`,
			want: RecordSet{
				{"id": float64(1), "data": map[string]interface{}{"nested": true}},
				{"id": float64(2), "data": map[string]interface{}{"nested": false}},
			},
		},
		{
			name:     "single object in array",
			jsonBody: `[{"id": 1, "name": "single"}]`,
			want: RecordSet{
				{"id": float64(1), "name": "single"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := createMockResponse(tt.jsonBody, 200, int64(len(tt.jsonBody)))
			result, err := unmarshalToRecordUnion(resp)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			recordSet, ok := result.(RecordSet)
			if !ok {
				t.Fatalf("expected RecordSet, got %T", result)
			}

			if !reflect.DeepEqual(recordSet, tt.want) {
				t.Errorf("result mismatch:\ngot:  %+v\nwant: %+v", recordSet, tt.want)
			}
		})
	}
}

// TestUnmarshalToRecordUnion_ArrayOfPrimitives tests unmarshaling arrays of primitives
func TestUnmarshalToRecordUnion_ArrayOfPrimitives(t *testing.T) {
	tests := []struct {
		name     string
		jsonBody string
		want     interface{} // The value that should be in @raw
	}{
		{
			name:     "array of integers",
			jsonBody: `[5, 8, 13, 21]`,
			want:     []interface{}{float64(5), float64(8), float64(13), float64(21)},
		},
		{
			name:     "array of strings",
			jsonBody: `["foo", "bar", "baz"]`,
			want:     []interface{}{"foo", "bar", "baz"},
		},
		{
			name:     "array of booleans",
			jsonBody: `[true, false, true]`,
			want:     []interface{}{true, false, true},
		},
		{
			name:     "array of mixed primitives",
			jsonBody: `[1, "two", true, 4.5]`,
			want:     []interface{}{float64(1), "two", true, float64(4.5)},
		},
		{
			name:     "single integer in array",
			jsonBody: `[42]`,
			want:     []interface{}{float64(42)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := createMockResponse(tt.jsonBody, 200, int64(len(tt.jsonBody)))
			result, err := unmarshalToRecordUnion(resp)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			record, ok := result.(Record)
			if !ok {
				t.Fatalf("expected Record, got %T", result)
			}

			raw, exists := record[customRawKey]
			if !exists {
				t.Fatalf("expected @raw key in record, got: %+v", record)
			}

			if !reflect.DeepEqual(raw, tt.want) {
				t.Errorf("@raw value mismatch:\ngot:  %+v (%T)\nwant: %+v (%T)", raw, raw, tt.want, tt.want)
			}
		})
	}
}

// TestUnmarshalToRecordUnion_EmptyArray tests the edge case of empty arrays
// Empty arrays are treated as RecordSet since we can't determine if they're arrays of objects or primitives
func TestUnmarshalToRecordUnion_EmptyArray(t *testing.T) {
	jsonBody := `[]`
	resp := createMockResponse(jsonBody, 200, int64(len(jsonBody)))
	result, err := unmarshalToRecordUnion(resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty array is treated as RecordSet (ambiguous case)
	recordSet, ok := result.(RecordSet)
	if !ok {
		t.Fatalf("expected RecordSet for empty array, got %T", result)
	}

	if len(recordSet) != 0 {
		t.Errorf("expected empty RecordSet, got length %d", len(recordSet))
	}
}

// TestUnmarshalToRecordUnion_SingleObject tests unmarshaling single objects
func TestUnmarshalToRecordUnion_SingleObject(t *testing.T) {
	tests := []struct {
		name     string
		jsonBody string
		want     Record
	}{
		{
			name:     "simple object",
			jsonBody: `{"id": 1, "name": "test"}`,
			want:     Record{"id": float64(1), "name": "test"},
		},
		{
			name:     "nested object",
			jsonBody: `{"id": 1, "data": {"nested": true, "count": 42}}`,
			want:     Record{"id": float64(1), "data": map[string]interface{}{"nested": true, "count": float64(42)}},
		},
		{
			name:     "object with null values",
			jsonBody: `{"id": 1, "nullable": null}`,
			want:     Record{"id": float64(1), "nullable": nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := createMockResponse(tt.jsonBody, 200, int64(len(tt.jsonBody)))
			result, err := unmarshalToRecordUnion(resp)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			record, ok := result.(Record)
			if !ok {
				t.Fatalf("expected Record, got %T", result)
			}

			if !reflect.DeepEqual(record, tt.want) {
				t.Errorf("result mismatch:\ngot:  %+v\nwant: %+v", record, tt.want)
			}
		})
	}
}

// TestUnmarshalToRecordUnion_EmptyResponse tests handling of empty responses
func TestUnmarshalToRecordUnion_EmptyResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		length     int64
	}{
		{
			name:       "204 No Content",
			statusCode: 204,
			body:       "",
			length:     0,
		},
		{
			name:       "empty body with 0 content length",
			statusCode: 200,
			body:       "",
			length:     0,
		},
		{
			name:       "whitespace-only body",
			statusCode: 200,
			body:       "   \n\t  ",
			length:     int64(len("   \n\t  ")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := createMockResponse(tt.body, tt.statusCode, tt.length)
			result, err := unmarshalToRecordUnion(resp)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			record, ok := result.(Record)
			if !ok {
				t.Fatalf("expected Record, got %T", result)
			}

			if len(record) != 0 {
				t.Errorf("expected empty Record, got: %+v", record)
			}
		})
	}
}

// TestUnmarshalToRecordUnion_StringResponse tests handling of string responses
func TestUnmarshalToRecordUnion_StringResponse(t *testing.T) {
	tests := []struct {
		name     string
		jsonBody string
		want     string
	}{
		{
			name:     "simple string",
			jsonBody: `"hello world"`,
			want:     `"hello world"`,
		},
		{
			name:     "string with special characters",
			jsonBody: `"line1\nline2\ttabbed"`,
			want:     `"line1\nline2\ttabbed"`,
		},
		{
			name:     "empty string",
			jsonBody: `""`,
			want:     `""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := createMockResponse(tt.jsonBody, 200, int64(len(tt.jsonBody)))
			result, err := unmarshalToRecordUnion(resp)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			record, ok := result.(Record)
			if !ok {
				t.Fatalf("expected Record, got %T", result)
			}

			raw, exists := record[customRawKey]
			if !exists {
				t.Fatalf("expected @raw key in record, got: %+v", record)
			}

			rawStr, ok := raw.(string)
			if !ok {
				t.Fatalf("expected string in @raw, got %T: %v", raw, raw)
			}

			if rawStr != tt.want {
				t.Errorf("@raw value mismatch:\ngot:  %q\nwant: %q", rawStr, tt.want)
			}
		})
	}
}

// TestUnmarshalToRecordUnion_InvalidJSON tests error handling for invalid JSON
func TestUnmarshalToRecordUnion_InvalidJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonBody string
	}{
		{
			name:     "invalid JSON syntax",
			jsonBody: `{invalid json}`,
		},
		{
			name:     "unclosed array",
			jsonBody: `[1, 2, 3`,
		},
		{
			name:     "unclosed object",
			jsonBody: `{"key": "value"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := createMockResponse(tt.jsonBody, 200, int64(len(tt.jsonBody)))
			_, err := unmarshalToRecordUnion(resp)
			if err == nil {
				t.Error("expected error for invalid JSON, got nil")
			}
		})
	}
}

// TestUnmarshalToRecordUnion_UnsupportedFormat tests error handling for unsupported formats
func TestUnmarshalToRecordUnion_UnsupportedFormat(t *testing.T) {
	tests := []struct {
		name     string
		jsonBody string
	}{
		{
			name:     "bare number",
			jsonBody: `42`,
		},
		{
			name:     "bare boolean",
			jsonBody: `true`,
		},
		{
			name:     "bare null",
			jsonBody: `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := createMockResponse(tt.jsonBody, 200, int64(len(tt.jsonBody)))
			_, err := unmarshalToRecordUnion(resp)
			if err == nil {
				t.Error("expected error for unsupported format, got nil")
			}
			if err != nil && err.Error() != "unsupported JSON format: must be object or array" {
				t.Errorf("unexpected error message: %v", err)
			}
		})
	}
}

// TestUnmarshalToRecordUnion_RealWorldScenarios tests real-world API response scenarios
func TestUnmarshalToRecordUnion_RealWorldScenarios(t *testing.T) {
	t.Run("NicPort related_nicports (array of integers)", func(t *testing.T) {
		jsonBody := `[5, 8]`
		resp := createMockResponse(jsonBody, 200, int64(len(jsonBody)))
		result, err := unmarshalToRecordUnion(resp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		record, ok := result.(Record)
		if !ok {
			t.Fatalf("expected Record, got %T", result)
		}

		raw, exists := record[customRawKey]
		if !exists {
			t.Fatalf("expected @raw key in record")
		}

		arr, ok := raw.([]interface{})
		if !ok {
			t.Fatalf("expected []interface{} in @raw, got %T", raw)
		}

		if len(arr) != 2 || arr[0] != float64(5) || arr[1] != float64(8) {
			t.Errorf("expected [5, 8], got %v", arr)
		}
	})

	t.Run("BigCatalogConfig columns (array of objects)", func(t *testing.T) {
		jsonBody := `[{"id": 1, "name": "col1", "type": "int"}, {"id": 2, "name": "col2", "type": "string"}]`
		resp := createMockResponse(jsonBody, 200, int64(len(jsonBody)))
		result, err := unmarshalToRecordUnion(resp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		recordSet, ok := result.(RecordSet)
		if !ok {
			t.Fatalf("expected RecordSet, got %T", result)
		}

		if len(recordSet) != 2 {
			t.Errorf("expected 2 records, got %d", len(recordSet))
		}

		if recordSet[0]["name"] != "col1" || recordSet[1]["name"] != "col2" {
			t.Errorf("unexpected column names: %v", recordSet)
		}
	})

	t.Run("User list (array of objects)", func(t *testing.T) {
		jsonBody := `[{"id": 1, "name": "alice"}, {"id": 2, "name": "bob"}]`
		resp := createMockResponse(jsonBody, 200, int64(len(jsonBody)))
		result, err := unmarshalToRecordUnion(resp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		recordSet, ok := result.(RecordSet)
		if !ok {
			t.Fatalf("expected RecordSet for array of objects, got %T", result)
		}

		if len(recordSet) != 2 {
			t.Errorf("expected 2 records, got %d", len(recordSet))
		}
	})

	t.Run("Single user (object)", func(t *testing.T) {
		jsonBody := `{"id": 1, "name": "alice", "email": "alice@example.com"}`
		resp := createMockResponse(jsonBody, 200, int64(len(jsonBody)))
		result, err := unmarshalToRecordUnion(resp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		record, ok := result.(Record)
		if !ok {
			t.Fatalf("expected Record for single object, got %T", result)
		}

		if record["name"] != "alice" {
			t.Errorf("expected name 'alice', got %v", record["name"])
		}
	})

	t.Run("DELETE operation returning 204 No Content", func(t *testing.T) {
		resp := createMockResponse("", 204, 0)
		result, err := unmarshalToRecordUnion(resp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		record, ok := result.(Record)
		if !ok {
			t.Fatalf("expected Record for 204 response, got %T", result)
		}

		if len(record) != 0 {
			t.Errorf("expected empty Record for 204, got: %+v", record)
		}
	})
}
