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
func FlexibleUnmarshal(data []byte, target any) error {
	// First unmarshal into a generic map
	var rawData map[string]any
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
func convertMapToStruct(data map[string]any, structType reflect.Type) map[string]any {
	result := make(map[string]any)

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
func convertValue(value any, targetType reflect.Type) any {
	if value == nil {
		return nil
	}

	// If target is string, convert non-strings to strings
	if targetType.Kind() == reflect.String {
		return convertToString(value)
	}

	// If target is bool, convert non-bools to bools
	if targetType.Kind() == reflect.Bool {
		if boolVal, err := ToBool(value); err == nil {
			return boolVal
		}
		// If conversion fails, return the original value and let JSON unmarshaling handle the error
		return value
	}

	// If target is a numeric type, handle type mismatches
	if isNumericKind(targetType.Kind()) {
		// If value is a string, try to parse it
		if strVal, ok := value.(string); ok {
			if convertedNum := convertStringToNumeric(strVal, targetType.Kind()); convertedNum != nil {
				return convertedNum
			}
			// If parsing fails, return zero value for the numeric type
			return getZeroValueForNumericKind(targetType.Kind())
		}
		// If target is an integer kind but value is any numeric type, truncate to int.
		// JSON always produces float64 in map[string]any, but handle all numeric
		// source types for robustness (float32, int variants, etc.).
		if isIntegerKind(targetType.Kind()) {
			if isNumericKind(reflect.TypeOf(value).Kind()) {
				return convertFloatToInteger(toFloat64(value), targetType.Kind())
			}
		}
	}

	// If target is a slice, recursively convert elements
	if targetType.Kind() == reflect.Slice {
		if arr, ok := value.([]any); ok {
			result := make([]any, len(arr))
			elemType := targetType.Elem()
			for i, item := range arr {
				result[i] = convertValue(item, elemType)
			}
			return result
		}
	}

	// If target is a pointer to slice, recursively convert
	if targetType.Kind() == reflect.Ptr && targetType.Elem().Kind() == reflect.Slice {
		if arr, ok := value.([]any); ok {
			result := make([]any, len(arr))
			elemType := targetType.Elem().Elem()
			for i, item := range arr {
				result[i] = convertValue(item, elemType)
			}
			return result
		}
	}

	// If target is a struct, recursively convert
	if targetType.Kind() == reflect.Struct {
		if m, ok := value.(map[string]any); ok {
			return convertMapToStruct(m, targetType)
		}
	}

	return value
}

// convertToString converts any value to a string
func convertToString(value any) string {
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

// isNumericKind returns true if the kind is a numeric type
func isNumericKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

// convertStringToNumeric tries to parse a string into the target numeric type
// Returns nil if parsing fails
func convertStringToNumeric(strVal string, kind reflect.Kind) any {
	switch kind {
	case reflect.Int:
		if val, err := strconv.ParseInt(strVal, 10, 0); err == nil {
			return int(val)
		}
	case reflect.Int8:
		if val, err := strconv.ParseInt(strVal, 10, 8); err == nil {
			return int8(val)
		}
	case reflect.Int16:
		if val, err := strconv.ParseInt(strVal, 10, 16); err == nil {
			return int16(val)
		}
	case reflect.Int32:
		if val, err := strconv.ParseInt(strVal, 10, 32); err == nil {
			return int32(val)
		}
	case reflect.Int64:
		if val, err := strconv.ParseInt(strVal, 10, 64); err == nil {
			return val
		}
	case reflect.Uint:
		if val, err := strconv.ParseUint(strVal, 10, 0); err == nil {
			return uint(val)
		}
	case reflect.Uint8:
		if val, err := strconv.ParseUint(strVal, 10, 8); err == nil {
			return uint8(val)
		}
	case reflect.Uint16:
		if val, err := strconv.ParseUint(strVal, 10, 16); err == nil {
			return uint16(val)
		}
	case reflect.Uint32:
		if val, err := strconv.ParseUint(strVal, 10, 32); err == nil {
			return uint32(val)
		}
	case reflect.Uint64:
		if val, err := strconv.ParseUint(strVal, 10, 64); err == nil {
			return val
		}
	case reflect.Float32:
		if val, err := strconv.ParseFloat(strVal, 32); err == nil {
			return float32(val)
		}
	case reflect.Float64:
		if val, err := strconv.ParseFloat(strVal, 64); err == nil {
			return val
		}
	}
	return nil
}

// isIntegerKind returns true if the kind is a signed or unsigned integer type
func isIntegerKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	}
	return false
}

// toFloat64 converts any numeric value to float64
func toFloat64(value any) float64 {
	switch v := value.(type) {
	case float32:
		return float64(v)
	case float64:
		return v
	case int:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	}
	return 0
}

// convertFloatToInteger truncates a float64 to the target integer kind
func convertFloatToInteger(f float64, kind reflect.Kind) any {
	switch kind {
	case reflect.Int:
		return int(f)
	case reflect.Int8:
		return int8(f)
	case reflect.Int16:
		return int16(f)
	case reflect.Int32:
		return int32(f)
	case reflect.Int64:
		return int64(f)
	case reflect.Uint:
		return uint(f)
	case reflect.Uint8:
		return uint8(f)
	case reflect.Uint16:
		return uint16(f)
	case reflect.Uint32:
		return uint32(f)
	case reflect.Uint64:
		return uint64(f)
	}
	return int64(f)
}

// getZeroValueForNumericKind returns the zero value for a numeric kind
func getZeroValueForNumericKind(kind reflect.Kind) any {
	switch kind {
	case reflect.Int:
		return int(0)
	case reflect.Int8:
		return int8(0)
	case reflect.Int16:
		return int16(0)
	case reflect.Int32:
		return int32(0)
	case reflect.Int64:
		return int64(0)
	case reflect.Uint:
		return uint(0)
	case reflect.Uint8:
		return uint8(0)
	case reflect.Uint16:
		return uint16(0)
	case reflect.Uint32:
		return uint32(0)
	case reflect.Uint64:
		return uint64(0)
	case reflect.Float32:
		return float32(0)
	case reflect.Float64:
		return float64(0)
	}
	return nil
}
