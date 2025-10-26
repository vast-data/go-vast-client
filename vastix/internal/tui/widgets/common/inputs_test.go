package common

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
)

func TestInputs_ToParams_BasicTypes(t *testing.T) {
	tests := []struct {
		name     string
		setupFn  func() Inputs
		expected map[string]any
	}{
		{
			name: "dirty string field should be included",
			setupFn: func() Inputs {
				var inputs Inputs
				textInput := textinput.New()
				textInput.SetValue("new_value")
				wrapper := NewTextInputWrapper("name", &textInput, true, "initial_value")
				inputs.Append(wrapper)
				return inputs
			},
			expected: map[string]any{"name": "new_value"},
		},
		{
			name: "unchanged string field should be excluded",
			setupFn: func() Inputs {
				var inputs Inputs
				textInput := textinput.New()
				textInput.SetValue("unchanged")
				wrapper := NewTextInputWrapper("name", &textInput, false, "unchanged")
				inputs.Append(wrapper)
				return inputs
			},
			expected: map[string]any{},
		},
		{
			name: "empty optional field should be excluded",
			setupFn: func() Inputs {
				var inputs Inputs
				textInput := textinput.New()
				textInput.SetValue("")
				wrapper := NewTextInputWrapper("optional", &textInput, false, "")
				inputs.Append(wrapper)
				return inputs
			},
			expected: map[string]any{},
		},
		{
			name: "dirty boolean field should be included",
			setupFn: func() Inputs {
				var inputs Inputs
				boolInput := NewBoolInput(true, "Enable feature")
				wrapper := NewBoolInputWrapper("enabled", boolInput, true, "false")
				inputs.Append(wrapper)
				return inputs
			},
			expected: map[string]any{"enabled": true},
		},
		{
			name: "dirty int64 field should be included",
			setupFn: func() Inputs {
				var inputs Inputs
				int64Input := NewInt64Input("42")
				wrapper := NewInt64InputWrapper("port", int64Input, false, "0")
				inputs.Append(wrapper)
				return inputs
			},
			expected: map[string]any{"port": int64(42)},
		},
		{
			name: "dirty float64 field should be included",
			setupFn: func() Inputs {
				var inputs Inputs
				float64Input := NewFloat64Input("3.14")
				wrapper := NewFloat64InputWrapper("ratio", float64Input, false, "0.0")
				inputs.Append(wrapper)
				return inputs
			},
			expected: map[string]any{"ratio": 3.14},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputs := tt.setupFn()
			result := inputs.ToParams()

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ToParams() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestInputs_ToParams_Arrays(t *testing.T) {
	tests := []struct {
		name      string
		arrayType string
		values    []string
		initial   string
		expected  interface{}
	}{
		{
			name:      "string array should convert to []string",
			arrayType: "array[string]",
			values:    []string{"item1", "item2", "item3"},
			initial:   "[]",
			expected:  []string{"item1", "item2", "item3"},
		},
		{
			name:      "int64 array should convert to []int64",
			arrayType: "array[int64]",
			values:    []string{"10", "20", "30"},
			initial:   "[]",
			expected:  []int64{10, 20, 30},
		},
		{
			name:      "float64 array should convert to []float64",
			arrayType: "array[float64]",
			values:    []string{"1.1", "2.2", "3.3"},
			initial:   "[]",
			expected:  []float64{1.1, 2.2, 3.3},
		},
		{
			name:      "bool array should convert to []bool",
			arrayType: "array[bool]",
			values:    []string{"true", "false", "true"},
			initial:   "[]",
			expected:  []bool{true, false, true},
		},
		{
			name:      "mixed case bool array should convert correctly",
			arrayType: "array[bool]",
			values:    []string{"True", "FALSE", "yes", "no", "1", "0"},
			initial:   "[]",
			expected:  []bool{true, false, true, false, true, false},
		},
		{
			name:      "int array with invalid values should skip invalid ones",
			arrayType: "array[int64]",
			values:    []string{"10", "invalid", "30"},
			initial:   "[]",
			expected:  []int64{10, 30},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var inputs Inputs
			arrayInput := NewPrimitivesArrayInputWithType(tt.values, tt.arrayType)
			wrapper := NewPrimitivesArrayInputWrapper("test_array", arrayInput, false, tt.initial)
			inputs.Append(wrapper)

			result := inputs.ToParams()

			if len(result) != 1 {
				t.Fatalf("Expected 1 field, got %d", len(result))
			}

			actualValue := result["test_array"]
			if !reflect.DeepEqual(actualValue, tt.expected) {
				t.Errorf("Array conversion failed. Got %v (%T), expected %v (%T)",
					actualValue, actualValue, tt.expected, tt.expected)
			}
		})
	}
}

func TestInputs_ToParams_NestedStructures(t *testing.T) {
	t.Run("simple nested structure", func(t *testing.T) {
		// Create child inputs for nested structure
		var childInputs []InputWrapper

		// Child 1: dirty string field
		textInput1 := textinput.New()
		textInput1.SetValue("nested_value")
		childWrapper1 := NewTextInputWrapper("nested_name", &textInput1, true, "initial")
		childInputs = append(childInputs, childWrapper1)

		// Child 2: unchanged optional field (should be excluded)
		textInput2 := textinput.New()
		textInput2.SetValue("unchanged")
		childWrapper2 := NewTextInputWrapper("nested_desc", &textInput2, false, "unchanged")
		childInputs = append(childInputs, childWrapper2)

		// Child 3: dirty boolean field
		boolInput := NewBoolInput(true, "Nested flag")
		childWrapper3 := NewBoolInputWrapper("nested_enabled", boolInput, false, "false")
		childInputs = append(childInputs, childWrapper3)

		// Create nested input
		nestedInput := NewNestedInput(childInputs)
		nestedWrapper := NewNestedInputWrapper("config", nestedInput, false, "{}")

		var inputs Inputs
		inputs.Append(nestedWrapper)

		result := inputs.ToParams()

		expected := map[string]any{
			"config": map[string]interface{}{
				"nested_name":    "nested_value",
				"nested_enabled": true,
				// nested_desc should be excluded because it's unchanged
			},
		}

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Nested structure conversion failed.\nGot: %v\nExpected: %v", result, expected)
		}
	})

	t.Run("deeply nested structure", func(t *testing.T) {
		// Create deep nested structure: config.database.connection
		var connectionInputs []InputWrapper

		// Connection host
		hostInput := textinput.New()
		hostInput.SetValue("localhost")
		hostWrapper := NewTextInputWrapper("host", &hostInput, true, "")
		connectionInputs = append(connectionInputs, hostWrapper)

		// Connection port
		portInput := NewInt64Input("5432")
		portWrapper := NewInt64InputWrapper("port", portInput, true, "0")
		connectionInputs = append(connectionInputs, portWrapper)

		// Create connection nested input
		connectionNested := NewNestedInput(connectionInputs)
		connectionWrapper := NewNestedInputWrapper("connection", connectionNested, true, "{}")

		// Create database inputs containing connection
		var databaseInputs []InputWrapper
		databaseInputs = append(databaseInputs, connectionWrapper)

		// Database name
		dbNameInput := textinput.New()
		dbNameInput.SetValue("mydb")
		dbNameWrapper := NewTextInputWrapper("name", &dbNameInput, true, "")
		databaseInputs = append(databaseInputs, dbNameWrapper)

		// Create database nested input
		databaseNested := NewNestedInput(databaseInputs)
		databaseWrapper := NewNestedInputWrapper("database", databaseNested, true, "{}")

		// Create top-level config
		var configInputs []InputWrapper
		configInputs = append(configInputs, databaseWrapper)

		// Add a top-level field
		appNameInput := textinput.New()
		appNameInput.SetValue("MyApp")
		appNameWrapper := NewTextInputWrapper("app_name", &appNameInput, true, "")
		configInputs = append(configInputs, appNameWrapper)

		configNested := NewNestedInput(configInputs)
		configWrapper := NewNestedInputWrapper("config", configNested, true, "{}")

		var inputs Inputs
		inputs.Append(configWrapper)

		result := inputs.ToParams()

		expected := map[string]any{
			"config": map[string]interface{}{
				"database": map[string]interface{}{
					"connection": map[string]interface{}{
						"host": "localhost",
						"port": int64(5432),
					},
					"name": "mydb",
				},
				"app_name": "MyApp",
			},
		}

		if !reflect.DeepEqual(result, expected) {
			resultJSON, _ := json.MarshalIndent(result, "", "  ")
			expectedJSON, _ := json.MarshalIndent(expected, "", "  ")
			t.Errorf("Deep nested structure conversion failed.\nGot:\n%s\nExpected:\n%s", resultJSON, expectedJSON)
		}
	})

	t.Run("nested structure with arrays", func(t *testing.T) {
		// Create nested structure containing arrays
		var serverInputs []InputWrapper

		// Server name
		nameInput := textinput.New()
		nameInput.SetValue("web-server")
		nameWrapper := NewTextInputWrapper("name", &nameInput, true, "")
		serverInputs = append(serverInputs, nameWrapper)

		// Server tags (string array)
		tagsArray := NewPrimitivesArrayInputWithType([]string{"web", "production"}, "array[string]")
		tagsWrapper := NewPrimitivesArrayInputWrapper("tags", tagsArray, false, "[]")
		serverInputs = append(serverInputs, tagsWrapper)

		// Server ports (int array)
		portsArray := NewPrimitivesArrayInputWithType([]string{"80", "443"}, "array[int64]")
		portsWrapper := NewPrimitivesArrayInputWrapper("ports", portsArray, false, "[]")
		serverInputs = append(serverInputs, portsWrapper)

		// Create server nested input
		serverNested := NewNestedInput(serverInputs)
		serverWrapper := NewNestedInputWrapper("server", serverNested, true, "{}")

		var inputs Inputs
		inputs.Append(serverWrapper)

		result := inputs.ToParams()

		expected := map[string]any{
			"server": map[string]interface{}{
				"name":  "web-server",
				"tags":  []string{"web", "production"},
				"ports": []int64{80, 443},
			},
		}

		if !reflect.DeepEqual(result, expected) {
			resultJSON, _ := json.MarshalIndent(result, "", "  ")
			expectedJSON, _ := json.MarshalIndent(expected, "", "  ")
			t.Errorf("Nested structure with arrays conversion failed.\nGot:\n%s\nExpected:\n%s", resultJSON, expectedJSON)
		}
	})
}

func TestInputs_ToParams_EdgeCases(t *testing.T) {
	t.Run("empty inputs should return empty map", func(t *testing.T) {
		var inputs Inputs
		result := inputs.ToParams()

		if len(result) != 0 {
			t.Errorf("Expected empty map, got %v", result)
		}
	})

	t.Run("all unchanged fields should return empty map", func(t *testing.T) {
		var inputs Inputs

		textInput1 := textinput.New()
		textInput1.SetValue("unchanged1")
		wrapper1 := NewTextInputWrapper("field1", &textInput1, false, "unchanged1")
		inputs.Append(wrapper1)

		textInput2 := textinput.New()
		textInput2.SetValue("unchanged2")
		wrapper2 := NewTextInputWrapper("field2", &textInput2, false, "unchanged2")
		inputs.Append(wrapper2)

		result := inputs.ToParams()

		if len(result) != 0 {
			t.Errorf("Expected empty map for unchanged fields, got %v", result)
		}
	})

	t.Run("required fields with default values should be included", func(t *testing.T) {
		var inputs Inputs

		// Required boolean field with false value (should be included even if "default")
		boolInput := NewBoolInput(false, "Required flag")
		wrapper := NewBoolInputWrapper("required_flag", boolInput, true, "false")
		inputs.Append(wrapper)

		result := inputs.ToParams()

		expected := map[string]any{"required_flag": false}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Required field with default value should be included. Got %v, expected %v", result, expected)
		}
	})

	t.Run("empty arrays should be excluded", func(t *testing.T) {
		var inputs Inputs

		emptyArray := NewPrimitivesArrayInputWithType([]string{}, "array[string]")
		wrapper := NewPrimitivesArrayInputWrapper("empty_tags", emptyArray, false, "[]")
		inputs.Append(wrapper)

		result := inputs.ToParams()

		if len(result) != 0 {
			t.Errorf("Empty array should be excluded, got %v", result)
		}
	})

	t.Run("nested structure with no dirty fields should be excluded", func(t *testing.T) {
		// Create nested structure with only unchanged fields
		var childInputs []InputWrapper

		textInput := textinput.New()
		textInput.SetValue("unchanged")
		childWrapper := NewTextInputWrapper("nested_field", &textInput, false, "unchanged")
		childInputs = append(childInputs, childWrapper)

		nestedInput := NewNestedInput(childInputs)
		nestedWrapper := NewNestedInputWrapper("config", nestedInput, false, "{}")

		var inputs Inputs
		inputs.Append(nestedWrapper)

		result := inputs.ToParams()

		if len(result) != 0 {
			t.Errorf("Nested structure with no dirty fields should be excluded, got %v", result)
		}
	})
}

func TestInputs_ToParams_ComplexScenario(t *testing.T) {
	t.Run("comprehensive test with mixed types and nesting", func(t *testing.T) {
		var inputs Inputs

		// Top-level string field (dirty)
		nameInput := textinput.New()
		nameInput.SetValue("MyApplication")
		nameWrapper := NewTextInputWrapper("app_name", &nameInput, true, "")
		inputs.Append(nameWrapper)

		// Top-level unchanged field (should be excluded)
		versionInput := textinput.New()
		versionInput.SetValue("1.0.0")
		versionWrapper := NewTextInputWrapper("version", &versionInput, false, "1.0.0")
		inputs.Append(versionWrapper)

		// Top-level array
		envArray := NewPrimitivesArrayInputWithType([]string{"production", "secure"}, "array[string]")
		envWrapper := NewPrimitivesArrayInputWrapper("environments", envArray, false, "[]")
		inputs.Append(envWrapper)

		// Nested configuration
		var configInputs []InputWrapper

		// Config: enabled flag
		enabledInput := NewBoolInput(true, "Enable config")
		enabledWrapper := NewBoolInputWrapper("enabled", enabledInput, true, "false")
		configInputs = append(configInputs, enabledWrapper)

		// Config: timeout
		timeoutInput := NewInt64Input("30")
		timeoutWrapper := NewInt64InputWrapper("timeout", timeoutInput, false, "0")
		configInputs = append(configInputs, timeoutWrapper)

		// Config: nested database settings
		var dbInputs []InputWrapper

		hostInput := textinput.New()
		hostInput.SetValue("db.example.com")
		hostWrapper := NewTextInputWrapper("host", &hostInput, true, "localhost")
		dbInputs = append(dbInputs, hostWrapper)

		portInput := NewInt64Input("5432")
		portWrapper := NewInt64InputWrapper("port", portInput, true, "3306")
		dbInputs = append(dbInputs, portWrapper)

		// Unchanged database field (should be excluded)
		dbNameInput := textinput.New()
		dbNameInput.SetValue("mydb")
		dbNameWrapper := NewTextInputWrapper("database", &dbNameInput, false, "mydb")
		dbInputs = append(dbInputs, dbNameWrapper)

		dbNested := NewNestedInput(dbInputs)
		dbWrapper := NewNestedInputWrapper("database", dbNested, true, "{}")
		configInputs = append(configInputs, dbWrapper)

		configNested := NewNestedInput(configInputs)
		configWrapper := NewNestedInputWrapper("config", configNested, true, "{}")
		inputs.Append(configWrapper)

		result := inputs.ToParams()

		expected := map[string]any{
			"app_name":     "MyApplication",
			"environments": []string{"production", "secure"},
			"config": map[string]interface{}{
				"enabled": true,
				"timeout": int64(30),
				"database": map[string]interface{}{
					"host": "db.example.com",
					"port": int64(5432),
					// "database" field excluded because unchanged
				},
			},
			// "version" field excluded because unchanged
		}

		if !reflect.DeepEqual(result, expected) {
			resultJSON, _ := json.MarshalIndent(result, "", "  ")
			expectedJSON, _ := json.MarshalIndent(expected, "", "  ")
			t.Errorf("Comprehensive test failed.\nGot:\n%s\nExpected:\n%s", resultJSON, expectedJSON)
		}
	})
}

// Benchmark tests
func BenchmarkInputs_ToParams_SimpleFields(b *testing.B) {
	var inputs Inputs

	for i := 0; i < 10; i++ {
		textInput := textinput.New()
		textInput.SetValue("value")
		wrapper := NewTextInputWrapper("field", &textInput, true, "")
		inputs.Append(wrapper)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = inputs.ToParams()
	}
}

func BenchmarkInputs_ToParams_NestedStructure(b *testing.B) {
	var inputs Inputs

	// Create a complex nested structure
	var childInputs []InputWrapper
	for i := 0; i < 5; i++ {
		textInput := textinput.New()
		textInput.SetValue("value")
		wrapper := NewTextInputWrapper("field", &textInput, true, "")
		childInputs = append(childInputs, wrapper)
	}

	nestedInput := NewNestedInput(childInputs)
	nestedWrapper := NewNestedInputWrapper("config", nestedInput, true, "{}")
	inputs.Append(nestedWrapper)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = inputs.ToParams()
	}
}
