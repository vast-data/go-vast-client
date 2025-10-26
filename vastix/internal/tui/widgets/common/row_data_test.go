package common

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestRowData_BasicOperations(t *testing.T) {
	t.Run("create new row data", func(t *testing.T) {
		data := NewRowData([]string{}, []string{})
		if data.Len() != 0 {
			t.Errorf("Expected empty row data, got length %d", data.Len())
		}
	})

	t.Run("create with headers and values", func(t *testing.T) {
		headers := []string{"id", "name", "status"}
		values := []string{"1", "test", "active"}
		data := NewRowData(headers, values)

		if data.Len() != 3 {
			t.Errorf("Expected length 3, got %d", data.Len())
		}

		if data.GetString("id") != "1" {
			t.Errorf("Expected 'id' to be '1', got '%s'", data.GetString("id"))
		}

		if data.GetString("name") != "test" {
			t.Errorf("Expected 'name' to be 'test', got '%s'", data.GetString("name"))
		}
	})

	t.Run("get non-existent key returns empty string", func(t *testing.T) {
		data := NewRowData([]string{"id"}, []string{"1"})
		value := data.GetString("non-existent")
		if value != "" {
			t.Errorf("Expected empty string for non-existent key, got '%s'", value)
		}
	})

	t.Run("set and get values", func(t *testing.T) {
		data := NewRowData([]string{"id"}, []string{"1"})
		data.Set("key1", "value1")
		data.Set("key2", "value2")

		value1 := data.Get("key1")
		if value1 != "value1" {
			t.Errorf("Expected 'value1', got '%v'", value1)
		}

		value2 := data.Get("key2")
		if value2 != "value2" {
			t.Errorf("Expected 'value2', got '%v'", value2)
		}
	})
}

func TestRowData_GetID(t *testing.T) {
	tests := []struct {
		name     string
		setupFn  func() RowData
		expected string
	}{
		{
			name: "get ID from lowercase id field",
			setupFn: func() RowData {
				return NewRowData([]string{"id", "name"}, []string{"123", "test"})
			},
			expected: "123",
		},
		{
			name: "get ID from uppercase ID field",
			setupFn: func() RowData {
				return NewRowData([]string{"ID", "name"}, []string{"456", "test"})
			},
			expected: "456",
		},
		{
			name: "get ID from first column when no explicit ID",
			setupFn: func() RowData {
				return NewRowData([]string{"uid", "name"}, []string{"789", "test"})
			},
			expected: "789",
		},
		{
			name: "return empty string when no headers",
			setupFn: func() RowData {
				return NewRowData([]string{}, []string{})
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.setupFn()
			result := data.GetID()
			if result != tt.expected {
				t.Errorf("GetID() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestRowData_ToSlice(t *testing.T) {
	t.Run("convert to slice maintains header order", func(t *testing.T) {
		headers := []string{"first", "second", "third"}
		values := []string{"value1", "value2", "value3"}
		data := NewRowData(headers, values)

		slice := data.ToSlice()
		expected := []string{"value1", "value2", "value3"}

		if !reflect.DeepEqual(slice, expected) {
			t.Errorf("ToSlice() = %v, expected %v", slice, expected)
		}
	})

	t.Run("empty row data returns empty slice", func(t *testing.T) {
		data := NewRowData([]string{}, []string{})
		slice := data.ToSlice()

		if len(slice) != 0 {
			t.Errorf("Expected empty slice, got %v", slice)
		}
	})

	t.Run("handles missing values with empty strings", func(t *testing.T) {
		headers := []string{"first", "second", "third"}
		values := []string{"value1"} // Only one value for three headers
		data := NewRowData(headers, values)

		slice := data.ToSlice()
		expected := []string{"value1", "", ""}

		if !reflect.DeepEqual(slice, expected) {
			t.Errorf("ToSlice() = %v, expected %v", slice, expected)
		}
	})
}

func TestRowData_EdgeCases(t *testing.T) {
	t.Run("set additional data beyond headers", func(t *testing.T) {
		data := NewRowData([]string{"id"}, []string{"1"})
		data.Set("extra", "extra_value")

		value := data.Get("extra")
		if value != "extra_value" {
			t.Errorf("Expected 'extra_value', got '%v'", value)
		}

		// Length should still be based on headers
		if data.Len() != 1 {
			t.Errorf("Expected length 1 (based on headers), got %d", data.Len())
		}
	})

	t.Run("GetString handles non-string values", func(t *testing.T) {
		data := NewRowData([]string{"id"}, []string{"1"})
		data.Set("number", 42)

		stringValue := data.GetString("number")
		if stringValue != "" {
			t.Errorf("Expected empty string for non-string value, got '%s'", stringValue)
		}

		// But Get should return the actual value
		actualValue := data.Get("number")
		if actualValue != 42 {
			t.Errorf("Expected 42, got %v", actualValue)
		}
	})

	t.Run("IntIdMust works correctly", func(t *testing.T) {
		data := NewRowData([]string{"id"}, []string{"123"})

		id := data.IntIdMust()
		if id != 123 {
			t.Errorf("Expected 123, got %d", id)
		}
	})

	t.Run("IntIdMust panics on invalid ID", func(t *testing.T) {
		data := NewRowData([]string{"id"}, []string{"invalid"})

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic for invalid ID")
			}
		}()

		data.IntIdMust()
	})
}

func TestRowData_GetStringMust(t *testing.T) {
	t.Run("finds key in original case", func(t *testing.T) {
		data := NewRowData([]string{"name", "ID"}, []string{"john", "123"})

		name := data.GetStringMust("name")
		if name != "john" {
			t.Errorf("Expected 'john', got '%s'", name)
		}
	})

	t.Run("finds key in uppercase when data has uppercase", func(t *testing.T) {
		data := NewRowData([]string{"NAME", "ID"}, []string{"jane", "456"})

		name := data.GetStringMust("name") // lowercase lookup should find NAME
		if name != "jane" {
			t.Errorf("Expected 'jane', got '%s'", name)
		}
	})

	t.Run("finds key in lowercase when data has lowercase", func(t *testing.T) {
		data := NewRowData([]string{"name", "id"}, []string{"bob", "789"})

		name := data.GetStringMust("NAME") // uppercase lookup should find name
		if name != "bob" {
			t.Errorf("Expected 'bob', got '%s'", name)
		}
	})

	t.Run("panics when key not found", func(t *testing.T) {
		data := NewRowData([]string{"id", "email"}, []string{"123", "test@example.com"})

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic for missing key")
			} else {
				panicMsg := fmt.Sprintf("%v", r)
				if !strings.Contains(panicMsg, "name") {
					t.Errorf("Expected panic message to contain 'name', got: %s", panicMsg)
				}
				if !strings.Contains(panicMsg, "Available keys") {
					t.Errorf("Expected panic message to show available keys, got: %s", panicMsg)
				}
			}
		}()

		data.GetStringMust("name")
	})

	t.Run("panics when key exists but not string", func(t *testing.T) {
		data := NewRowData([]string{"id"}, []string{"123"})
		data.Set("number", 42) // Set non-string value

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic for non-string value")
			} else {
				panicMsg := fmt.Sprintf("%v", r)
				if !strings.Contains(panicMsg, "not a string") {
					t.Errorf("Expected panic message about type mismatch, got: %s", panicMsg)
				}
			}
		}()

		data.GetStringMust("number")
	})
}

func TestRowData_GetInt64Must(t *testing.T) {
	t.Run("finds key in original case and converts string to int64", func(t *testing.T) {
		data := NewRowData([]string{"age", "score"}, []string{"25", "1000"})

		age := data.GetInt64Must("age")
		if age != 25 {
			t.Errorf("Expected 25, got %d", age)
		}

		score := data.GetInt64Must("score")
		if score != 1000 {
			t.Errorf("Expected 1000, got %d", score)
		}
	})

	t.Run("finds key with case insensitive lookup", func(t *testing.T) {
		data := NewRowData([]string{"AGE", "Score"}, []string{"30", "2000"})

		age := data.GetInt64Must("age") // lowercase lookup should find AGE
		if age != 30 {
			t.Errorf("Expected 30, got %d", age)
		}

		score := data.GetInt64Must("SCORE") // uppercase lookup should find Score
		if score != 2000 {
			t.Errorf("Expected 2000, got %d", score)
		}
	})

	t.Run("handles negative numbers", func(t *testing.T) {
		data := NewRowData([]string{"temperature", "balance"}, []string{"-10", "-500"})

		temp := data.GetInt64Must("temperature")
		if temp != -10 {
			t.Errorf("Expected -10, got %d", temp)
		}

		balance := data.GetInt64Must("balance")
		if balance != -500 {
			t.Errorf("Expected -500, got %d", balance)
		}
	})

	t.Run("panics when key not found", func(t *testing.T) {
		data := NewRowData([]string{"name", "age"}, []string{"john", "25"})

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic when key not found")
			}
		}()

		data.GetInt64Must("nonexistent")
	})

	t.Run("panics when value cannot be converted to int64", func(t *testing.T) {
		data := NewRowData([]string{"name", "score"}, []string{"john", "not_a_number"})

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic when value cannot be converted to int64")
			}
		}()

		data.GetInt64Must("score")
	})

	t.Run("panics when value is empty string", func(t *testing.T) {
		data := NewRowData([]string{"name", "age"}, []string{"john", ""})

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic when value is empty string")
			}
		}()

		data.GetInt64Must("age")
	})
}

func TestRowData_GetStringOptional(t *testing.T) {
	t.Run("finds key in original case", func(t *testing.T) {
		data := NewRowData([]string{"name", "ID"}, []string{"john", "123"})

		name := data.GetStringOptional("name")
		if name != "john" {
			t.Errorf("Expected 'john', got '%s'", name)
		}
	})

	t.Run("finds key case-insensitively", func(t *testing.T) {
		data := NewRowData([]string{"NAME", "ID"}, []string{"jane", "456"})

		name := data.GetStringOptional("name") // lowercase lookup should find NAME
		if name != "jane" {
			t.Errorf("Expected 'jane', got '%s'", name)
		}
	})

	t.Run("returns empty string when key not found", func(t *testing.T) {
		data := NewRowData([]string{"id", "email"}, []string{"123", "test@example.com"})

		name := data.GetStringOptional("name")
		if name != "" {
			t.Errorf("Expected empty string for missing key, got '%s'", name)
		}
	})

	t.Run("returns empty string when key exists but not string", func(t *testing.T) {
		data := NewRowData([]string{"id"}, []string{"123"})
		data.Set("number", 42) // Set non-string value

		result := data.GetStringOptional("number")
		if result != "" {
			t.Errorf("Expected empty string for non-string value, got '%s'", result)
		}
	})
}

func BenchmarkRowData_Set(b *testing.B) {
	data := NewRowData([]string{"id"}, []string{"1"})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		data.Set("key", "value")
	}
}

func BenchmarkRowData_Get(b *testing.B) {
	data := NewRowData([]string{"id"}, []string{"1"})
	data.Set("key", "value")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = data.Get("key")
	}
}

func BenchmarkRowData_ToSlice(b *testing.B) {
	headers := make([]string, 10)
	values := make([]string, 10)
	for i := 0; i < 10; i++ {
		headers[i] = fmt.Sprintf("header%d", i)
		values[i] = fmt.Sprintf("value%d", i)
	}
	data := NewRowData(headers, values)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = data.ToSlice()
	}
}
