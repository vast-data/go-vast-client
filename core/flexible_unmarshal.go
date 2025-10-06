package core

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
)

// FlexibleUnmarshal unmarshals JSON with flexible type conversion for string fields.
// When a string field in the target struct receives a non-string value (number, boolean),
// it automatically converts it to a string.
func FlexibleUnmarshal(data []byte, target interface{}) error {
	// First unmarshal into a generic map
	var rawData map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return err
	}

	// Get the target struct type
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}
	targetElem := targetValue.Elem()
	if targetElem.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct")
	}

	// Convert values based on target field types
	convertedData := convertMapToStruct(rawData, targetElem.Type())

	// Marshal and unmarshal again with the converted data
	convertedJSON, err := json.Marshal(convertedData)
	if err != nil {
		return err
	}

	return json.Unmarshal(convertedJSON, target)
}

// convertMapToStruct recursively converts map values to match struct field types
func convertMapToStruct(data map[string]interface{}, structType reflect.Type) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range data {
		// Find the corresponding struct field
		field, found := findFieldByJSONTag(structType, key)
		if !found {
			result[key] = value
			continue
		}

		result[key] = convertValue(value, field.Type)
	}

	return result
}

// convertValue converts a value to match the target type
func convertValue(value interface{}, targetType reflect.Type) interface{} {
	if value == nil {
		return nil
	}

	// If target is string, convert non-strings to strings
	if targetType.Kind() == reflect.String {
		return convertToString(value)
	}

	// If target is a slice, recursively convert elements
	if targetType.Kind() == reflect.Slice {
		if arr, ok := value.([]interface{}); ok {
			result := make([]interface{}, len(arr))
			elemType := targetType.Elem()
			for i, item := range arr {
				result[i] = convertValue(item, elemType)
			}
			return result
		}
	}

	// If target is a pointer to slice, recursively convert
	if targetType.Kind() == reflect.Ptr && targetType.Elem().Kind() == reflect.Slice {
		if arr, ok := value.([]interface{}); ok {
			result := make([]interface{}, len(arr))
			elemType := targetType.Elem().Elem()
			for i, item := range arr {
				result[i] = convertValue(item, elemType)
			}
			return result
		}
	}

	// If target is a struct, recursively convert
	if targetType.Kind() == reflect.Struct {
		if m, ok := value.(map[string]interface{}); ok {
			return convertMapToStruct(m, targetType)
		}
	}

	return value
}

// convertToString converts any value to a string
func convertToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		// Check if it's an integer value
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

// findFieldByJSONTag finds a struct field by its JSON tag
func findFieldByJSONTag(structType reflect.Type, jsonTag string) (reflect.StructField, bool) {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		tag := field.Tag.Get("json")
		if tag == "" {
			continue
		}

		// Remove options like ",omitempty"
		tagName := tag
		for i, c := range tag {
			if c == ',' {
				tagName = tag[:i]
				break
			}
		}

		if tagName == jsonTag {
			return field, true
		}
	}
	return reflect.StructField{}, false
}
