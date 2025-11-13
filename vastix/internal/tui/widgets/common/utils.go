package common

import (
	"fmt"
	"strconv"
	"strings"

	vast_client "github.com/vast-data/go-vast-client"
)

func ToUpperSlice(input []string) []string {
	result := make([]string, len(input))
	for i, s := range input {
		result[i] = strings.ToUpper(s)
	}
	return result
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func ToInt(val any) (int64, error) {
	var idInt int64
	switch v := val.(type) {
	case int64:
		idInt = v
	case float64:
		idInt = int64(v)
	case int:
		idInt = int64(v)
	default:
		return 0, fmt.Errorf("unexpected type for id field: %T", v)
	}
	return idInt, nil
}

func ToIntMust(val any) int64 {
	idInt, err := ToInt(val)
	if err != nil {
		panic(fmt.Sprintf("ToIntMust: %v", err))
	}
	return idInt
}

// SplitServerParams splits server parameters intelligently
// Properly handles cases like "path__contains=test id__in=1,2,3" and "key1=val1,key2=val2"
func SplitServerParams(input string) []string {
	if input == "" {
		return []string{}
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return []string{}
	}

	// Use a regex-based approach to find key=value patterns
	var params []string

	// First, try to detect if this looks like space-separated key=value pairs
	if strings.Contains(input, " ") {
		params = parseSpaceSeparatedParams(input)
	} else {
		// Check for comma-separated parameters using heuristic
		eqCount := strings.Count(input, "=")
		commaCount := strings.Count(input, ",")

		if eqCount > 1 && commaCount > 0 {
			// Multiple parameters likely separated by comma
			params = strings.Split(input, ",")
		} else {
			// Single parameter (possibly with commas in value like id__in=1,2,3)
			params = []string{input}
		}
	}

	// Clean up each parameter
	var result []string
	for _, param := range params {
		param = strings.TrimSpace(param)
		// Remove trailing commas that might be left over from comma-separated parsing
		for strings.HasSuffix(param, ",") {
			param = strings.TrimSuffix(param, ",")
			param = strings.TrimSpace(param)
		}
		if param != "" {
			result = append(result, param)
		}
	}

	return result
}

// parseSpaceSeparatedParams parses space-separated key=value pairs
// Handles cases where values might contain commas
func parseSpaceSeparatedParams(input string) []string {
	var params []string
	var current strings.Builder
	inValue := false

	parts := strings.Fields(input)

	for i, part := range parts {
		if strings.Contains(part, "=") {
			// If we were building a previous parameter, finish it
			if current.Len() > 0 {
				params = append(params, current.String())
				current.Reset()
			}

			// Start new parameter
			current.WriteString(part)

			// Check if this completes the parameter or if we need more parts
			if i == len(parts)-1 {
				// Last part, must be complete
				inValue = false
			} else {
				// Check next part to see if it looks like a new key=value
				nextPart := parts[i+1]
				if strings.Contains(nextPart, "=") {
					// Next part is a new parameter, current one is complete
					inValue = false
				} else {
					// Next part might be continuation of current value
					inValue = true
				}
			}
		} else {
			// This part doesn't contain =
			if inValue {
				// Part of current parameter value
				current.WriteString(",")
				current.WriteString(part)
			} else {
				// Treat as separate parameter (might be malformed)
				if current.Len() > 0 {
					params = append(params, current.String())
					current.Reset()
				}
				current.WriteString(part)
				inValue = false
			}
		}
	}

	// Add final parameter if any
	if current.Len() > 0 {
		params = append(params, current.String())
	}

	return params
}

// ConvertServerParamsToVastParams converts a server search string to vast_client.Params for API calls
// Special handling for __in parameters: converts comma-separated values to slices
func ConvertServerParamsToVastParams(serverSearchStr string) (vast_client.Params, error) {
	params := make(vast_client.Params)

	if serverSearchStr == "" {
		return params, nil
	}

	// Split by comma or space to handle multiple parameters
	paramStrings := SplitServerParams(serverSearchStr)

	for _, paramStr := range paramStrings {
		paramStr = strings.TrimSpace(paramStr)
		if paramStr == "" {
			continue
		}

		// Split by = to get key and value
		parts := strings.SplitN(paramStr, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid parameter format: %s (expected key=value)", paramStr)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			return nil, fmt.Errorf("empty key in parameter: %s", paramStr)
		}

		// Special handling for __in parameters: convert value to slice
		if strings.HasSuffix(key, "__in") {
			// Handle bracket notation: [1,2,3] -> 1,2,3
			if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
				value = strings.TrimPrefix(value, "[")
				value = strings.TrimSuffix(value, "]")
				value = strings.TrimSpace(value)
			}

			// Split the value by comma to create a slice
			valueSlice := strings.Split(value, ",")
			var cleanedSlice []string
			for _, v := range valueSlice {
				v = strings.TrimSpace(v)
				if v != "" {
					cleanedSlice = append(cleanedSlice, v)
				}
			}

			// Try to convert to appropriate type based on content
			if len(cleanedSlice) > 0 {
				// Check if all values are integers
				var intSlice []int64
				allInts := true
				for _, v := range cleanedSlice {
					if intVal, err := strconv.ParseInt(v, 10, 64); err == nil {
						intSlice = append(intSlice, intVal)
					} else {
						allInts = false
						break
					}
				}

				if allInts {
					params[key] = intSlice
				} else {
					params[key] = cleanedSlice
				}
			} else {
				params[key] = []string{}
			}
		} else {
			// Regular parameter: store as string
			params[key] = value
		}
	}

	return params, nil
}
