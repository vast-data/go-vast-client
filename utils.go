package vast_client

import (
	"fmt"
	"net"
	"reflect"
	"strings"
)


func toInt(val any) (int64, error) {
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

func toRecord(m map[string]interface{}) (Record, error) {
	converted := Record{}
	for k, v := range m {
		converted[k] = v
	}
	return converted, nil
}

func toRecordSet(list []map[string]any) (RecordSet, error) {
	records := make(RecordSet, 0, len(list))
	for _, item := range list {
		rec, err := toRecord(item)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, nil
}

// contains checks if a string is present in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func generateIPRange(ipRanges [][2]string) ([]string, error) {
	ips := []string{}
	for _, r := range ipRanges {
		start := net.ParseIP(r[0]).To4()
		end := net.ParseIP(r[1]).To4()
		if start == nil || end == nil {
			return nil, fmt.Errorf("invalid IP in range: %v", r)
		}
		for ip := start; !ipGreaterThan(ip, end); ip = nextIP(ip) {
			ips = append(ips, ip.String())
		}
	}
	return ips, nil
}

func nextIP(ip net.IP) net.IP {
	newIP := make(net.IP, len(ip))
	copy(newIP, ip)
	for j := len(newIP) - 1; j >= 0; j-- {
		newIP[j]++
		if newIP[j] != 0 {
			break
		}
	}
	return newIP
}

func ipGreaterThan(a, b net.IP) bool {
	for i := 0; i < len(a); i++ {
		if a[i] > b[i] {
			return true
		} else if a[i] < b[i] {
			return false
		}
	}
	return false
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("must: %v", err))
	}
	return v
}

// buildResourcePathWithID builds a complete resource path with an ID parameter and optional additional segments.
// It takes a resource path (e.g., "/users"), an ID of any type, and optional additional path segments.
// Returns the complete path (e.g., "/users/123/tenant_data" or "/users/uuid/tenant_data").
func buildResourcePathWithID(resourcePath string, id any, additionalSegments ...string) string {
	var path string
	if intId, err := toInt(id); err == nil {
		path = fmt.Sprintf("%s/%d", resourcePath, intId)
	} else {
		path = fmt.Sprintf("%s/%v", resourcePath, id)
	}

	// Append additional segments if provided
	for _, segment := range additionalSegments {
		path += "/" + segment
	}

	return path
}

// structToMap converts a struct to a map[string]interface{} using reflection,
// respecting json tags and handling nested structs recursively.
// This avoids the overhead of JSON marshaling/unmarshaling.
func structToMap(item interface{}) map[string]interface{} {
	res := map[string]interface{}{}
	if item == nil {
		return res
	}

	v := reflect.TypeOf(item)
	reflectValue := reflect.ValueOf(item)
	reflectValue = reflect.Indirect(reflectValue)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Only process structs
	if v.Kind() != reflect.Struct {
		return res
	}

	for i := 0; i < v.NumField(); i++ {
		jsonTag := v.Field(i).Tag.Get("json")
		field := reflectValue.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		// Parse JSON tag properly
		tagName, omitEmpty := parseJSONTag(jsonTag)

		// Skip if no tag or tag is "-"
		if tagName == "" || tagName == "-" {
			continue
		}

		fieldValue := field.Interface()

		// Handle different field types
		switch {
		case field.Kind() == reflect.Ptr:
			if field.IsNil() {
				// Nil pointers are always omitted with omitempty
				if omitEmpty {
					continue
				}
				res[tagName] = nil
			} else if field.Elem().Kind() == reflect.Struct {
				// Pointer to struct - recursively process
				nestedMap := structToMap(field.Interface())
				// Note: JSON marshaling includes empty structs as {} even with omitempty
				// So we always include nested structs, never omit them
				res[tagName] = nestedMap
			} else {
				// Pointer to primitive - dereference it
				derefValue := field.Elem().Interface()
				// Note: For pointers with omitempty, JSON marshaling only omits nil pointers,
				// not pointers to zero values. So we always include non-nil pointers.
				res[tagName] = derefValue
			}

		case v.Field(i).Type.Kind() == reflect.Struct:
			// Direct struct (not pointer)
			nestedMap := structToMap(fieldValue)
			// Note: JSON marshaling includes empty structs as {} even with omitempty
			// So we always include nested structs, never omit them
			res[tagName] = nestedMap

		case field.Kind() == reflect.Slice || field.Kind() == reflect.Array:
			if field.IsNil() {
				// Nil slices are omitted with omitempty
				if omitEmpty {
					continue
				}
				res[tagName] = nil
			} else if field.Len() == 0 {
				// Empty slices are omitted with omitempty
				if omitEmpty {
					continue
				}
				res[tagName] = fieldValue
			} else {
				// Non-empty slices are always included
				res[tagName] = fieldValue
			}

		default:
			// Primitive types
			if omitEmpty && isZeroValue(field) {
				continue
			}
			res[tagName] = fieldValue
		}
	}
	return res
}

// parseJSONTag parses a JSON struct tag and returns the field name and whether omitempty is specified.
// Examples:
//   - `json:"name"` returns ("name", false)
//   - `json:"name,omitempty"` returns ("name", true)
//   - `json:",omitempty"` returns ("", true)
//   - `json:"-"` returns ("-", false)
//   - `json:""` returns ("", false)
func parseJSONTag(tag string) (name string, omitEmpty bool) {
	if tag == "" {
		return "", false
	}

	// Split by comma to separate name from options
	parts := strings.Split(tag, ",")
	name = parts[0]

	// Check for omitempty option
	for i := 1; i < len(parts); i++ {
		if strings.TrimSpace(parts[i]) == "omitempty" {
			omitEmpty = true
			break
		}
	}

	return name, omitEmpty
}

// isZeroValue reports whether v is the zero value for its type.
// This is used to implement omitempty behavior for non-pointer types.
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() == 0
	case reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		return v.IsNil()
	case reflect.String:
		return v.Len() == 0
	case reflect.Struct:
		// For structs, check if all fields are zero
		return v.IsZero() // Available in Go 1.13+
	default:
		// For other types, use the built-in IsZero if available
		return v.IsZero()
	}
}

// isZeroValueInterface reports whether the interface{} value is a zero value.
// This is used for dereferenced pointer values.
func isZeroValueInterface(val interface{}) bool {
	if val == nil {
		return true
	}

	v := reflect.ValueOf(val)
	return isZeroValue(v)
}
