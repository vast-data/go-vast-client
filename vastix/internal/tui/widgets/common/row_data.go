package common

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// RowData represents a row of data with ordered columns
type RowData struct {
	headers []string
	data    map[string]any
}

// NewRowData creates a new RowData with the given headers and values
// Headers are stored as provided for display, but data keys are stored as lowercase for consistent access
func NewRowData(headers []string, values []string) RowData {
	data := make(map[string]any)
	for i, header := range headers {
		// Store data with lowercase keys for consistent access
		lowerKey := strings.ToLower(header)
		if i < len(values) {
			data[lowerKey] = values[i]
		} else {
			data[lowerKey] = ""
		}
	}
	return RowData{
		headers: headers, // Keep original headers for display
		data:    data,    // Data keys are lowercase
	}
}

// GetString safely retrieves a string value from RowData
func (rd RowData) GetString(key string) string {
	// Convert key to lowercase since all data keys are stored lowercase
	lowerKey := strings.ToLower(key)
	if val, exists := rd.data[lowerKey]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// GetID is a convenience method to get the ID field (commonly the first column)
func (rd RowData) GetID() string {
	// Try common ID field names
	for _, idField := range []string{"id", "ID", "Id"} {
		if id := rd.GetString(idField); id != "" {
			return id
		}
	}
	// If no explicit ID field found, return the first column value
	if len(rd.headers) > 0 {
		return rd.GetString(rd.headers[0])
	}
	return ""
}

// IntIdMust extracts the ID and converts it to int64, panics if conversion fails
func (rd RowData) IntIdMust() int64 {
	id, err := rd.GetIntID()
	if err != nil {
		panic(err)
	}
	return id

}

func (rd RowData) GetIntID() (int64, error) {
	idStr := rd.GetID()
	if idStr == "" {
		return 0, errors.New("RowData: ID field is empty or missing")
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("RowData: failed to convert ID '%s' to int64: %v", idStr, err)
	}

	return id, nil
}

// ToSlice returns the values in the same order as headers
func (rd RowData) ToSlice() []string {
	result := make([]string, len(rd.headers))
	for i, header := range rd.headers {
		// Convert header to lowercase to access data
		lowerKey := strings.ToLower(header)
		if val, exists := rd.data[lowerKey]; exists {
			if str, ok := val.(string); ok {
				result[i] = str
			} else {
				result[i] = fmt.Sprintf("%v", val)
			}
		} else {
			result[i] = ""
		}
	}
	return result
}

// Len returns the number of columns
func (rd RowData) Len() int {
	return len(rd.headers)
}

// Get returns the value for the given key
func (rd RowData) Get(key string) any {
	// Convert key to lowercase since all data keys are stored lowercase
	lowerKey := strings.ToLower(key)
	return rd.data[lowerKey]
}

// Set sets a value for the given key
func (rd RowData) Set(key string, value any) {
	// Convert key to lowercase for consistent storage
	lowerKey := strings.ToLower(key)
	rd.data[lowerKey] = value
}

// GetStringMust retrieves a string value from RowData with case-insensitive lookup
// Panics if the key is not found (keys are internally stored as lowercase)
func (rd RowData) GetStringMust(key string) string {
	// Convert key to lowercase since all data keys are stored lowercase
	lowerKey := strings.ToLower(key)

	if val, exists := rd.data[lowerKey]; exists {
		if str, ok := val.(string); ok {
			return str
		}
		panic(fmt.Sprintf("RowData: key '%s' exists but is not a string (got %T)", key, val))
	}

	// Key not found - panic with available keys for debugging
	availableKeys := make([]string, 0, len(rd.data))
	for k := range rd.data {
		availableKeys = append(availableKeys, k)
	}
	panic(fmt.Sprintf("RowData: key '%s' (lowercase: '%s') not found. Available keys: %v",
		key, lowerKey, availableKeys))
}

// GetInt64 retrieves an int64 value from RowData with case-insensitive lookup
// Returns 0 if key is not found or cannot be converted (non-panicking version)
func (rd RowData) GetInt64(key string) int64 {
	// Convert key to lowercase since all data keys are stored lowercase
	lowerKey := strings.ToLower(key)

	if val, exists := rd.data[lowerKey]; exists {
		// Handle string values that need to be parsed
		if str, ok := val.(string); ok {
			if intVal, err := strconv.ParseInt(str, 10, 64); err == nil {
				return intVal
			}
		}
		// Handle direct int64 values
		if intVal, ok := val.(int64); ok {
			return intVal
		}
		// Handle other numeric types that can be converted
		if intVal, ok := val.(int); ok {
			return int64(intVal)
		}
		// Handle float64 (common in JSON unmarshaling)
		if floatVal, ok := val.(float64); ok {
			return int64(floatVal)
		}
	}
	return 0
}

// GetInt64Must retrieves an int64 value from RowData with case-insensitive lookup
// Panics if the key is not found or cannot be converted to int64
func (rd RowData) GetInt64Must(key string) int64 {
	// Convert key to lowercase since all data keys are stored lowercase
	lowerKey := strings.ToLower(key)

	if val, exists := rd.data[lowerKey]; exists {
		// Handle string values that need to be parsed
		if str, ok := val.(string); ok {
			if intVal, err := strconv.ParseInt(str, 10, 64); err == nil {
				return intVal
			} else {
				panic(fmt.Sprintf("RowData: key '%s' exists but cannot be converted to int64: '%s' (error: %v)", key, str, err))
			}
		}
		// Handle direct int64 values
		if intVal, ok := val.(int64); ok {
			return intVal
		}
		// Handle other numeric types that can be converted
		if intVal, ok := val.(int); ok {
			return int64(intVal)
		}
		panic(fmt.Sprintf("RowData: key '%s' exists but is not convertible to int64 (got %T)", key, val))
	}

	// Key not found - panic with available keys for debugging
	availableKeys := make([]string, 0, len(rd.data))
	for k := range rd.data {
		availableKeys = append(availableKeys, k)
	}
	panic(fmt.Sprintf("RowData: key '%s' (lowercase: '%s') not found. Available keys: %v",
		key, lowerKey, availableKeys))
}

// GetStringOptional retrieves a string value from RowData with case-insensitive lookup
// Returns empty string if key is not found (non-panicking version)
func (rd RowData) GetStringOptional(key string) string {
	// Convert key to lowercase since all data keys are stored lowercase
	lowerKey := strings.ToLower(key)

	if val, exists := rd.data[lowerKey]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}

	return ""
}
