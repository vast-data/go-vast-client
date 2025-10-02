package markers

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Parse parses a marker text into a typed value using the definition.
func (d *Definition) Parse(markerText string) (interface{}, error) {
	// Remove leading "+" if present
	if strings.HasPrefix(markerText, "+") {
		markerText = markerText[1:]
	}

	// Split marker into name and arguments
	name, args := d.splitMarker(markerText)
	if name != d.Name {
		return nil, fmt.Errorf("marker name mismatch: expected %s, got %s", d.Name, name)
	}

	// Create output value
	outPtr := reflect.New(d.OutputType)
	out := reflect.Indirect(outPtr)

	// If no arguments, return zero value
	if args == "" {
		return out.Interface(), nil
	}

	// Parse arguments
	if err := d.parseArguments(args, out); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	return out.Interface(), nil
}

// splitMarker splits a marker into name and arguments.
// Example: "vast:generate=config" -> ("vast:generate", "config")
func (d *Definition) splitMarker(markerText string) (string, string) {
	parts := strings.SplitN(markerText, "=", 2)
	if len(parts) == 1 {
		return strings.TrimSpace(parts[0]), ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

// parseArguments parses the argument string and sets values in the output struct.
func (d *Definition) parseArguments(args string, out reflect.Value) error {
	if d.OutputType.Kind() != reflect.Struct {
		// For non-struct types, parse as single value
		field := d.Fields[""]
		return d.parseValue(args, field, out)
	}

	// Special handling for apiuntyped:extraMethod format: METHOD=PATH
	if d.Name == "apiuntyped:extraMethod" && strings.Contains(args, "=") {
		parts := strings.SplitN(args, "=", 2)
		if len(parts) == 2 {
			method := strings.TrimSpace(parts[0])
			path := strings.TrimSpace(parts[1])
			
			// Set Method field
			if methodField := out.FieldByName("Method"); methodField.IsValid() {
				methodField.SetString(method)
			}
			
			// Set Path field
			if pathField := out.FieldByName("Path"); pathField.IsValid() {
				pathField.SetString(path)
			}
			
			return nil
		}
	}

	// For struct types, parse named arguments
	if !strings.Contains(args, "=") {
		// If no "=" found, treat as anonymous argument for first field
		if len(d.Fields) == 1 {
			for _, field := range d.Fields {
				return d.parseValue(args, field, out)
			}
		}
		return fmt.Errorf("expected named arguments for struct type")
	}

	// Parse key=value pairs
	pairs, err := d.parseKeyValuePairs(args)
	if err != nil {
		return err
	}

	for key, value := range pairs {
		fieldType, exists := d.Fields[key]
		if !exists {
			return fmt.Errorf("unknown argument: %s", key)
		}

		fieldName, exists := d.FieldNames[key]
		if !exists {
			return fmt.Errorf("no field mapping for argument: %s", key)
		}

		var fieldValue reflect.Value
		if fieldName == "" {
			fieldValue = out
		} else {
			fieldValue = out.FieldByName(fieldName)
			if !fieldValue.IsValid() {
				return fmt.Errorf("field %s not found", fieldName)
			}
		}

		if err := d.parseValue(value, fieldType, fieldValue); err != nil {
			return fmt.Errorf("argument %s: %w", key, err)
		}
	}

	return nil
}

// parseKeyValuePairs parses a string like "key1=value1,key2=value2" into a map.
func (d *Definition) parseKeyValuePairs(args string) (map[string]string, error) {
	result := make(map[string]string)

	// Handle complex parsing with nested structures
	i := 0
	for i < len(args) {
		// Skip whitespace
		for i < len(args) && args[i] == ' ' {
			i++
		}
		if i >= len(args) {
			break
		}

		// Find key
		keyStart := i
		for i < len(args) && args[i] != '=' {
			i++
		}
		if i >= len(args) {
			return nil, fmt.Errorf("missing '=' in key=value pair")
		}
		key := strings.TrimSpace(args[keyStart:i])
		i++ // skip '='

		// Find value
		valueStart := i
		braceCount := 0
		inQuotes := false

		for i < len(args) {
			char := args[i]
			if char == '"' && (i == 0 || args[i-1] != '\\') {
				inQuotes = !inQuotes
			} else if !inQuotes {
				if char == '{' {
					braceCount++
				} else if char == '}' {
					braceCount--
				} else if char == ',' && braceCount == 0 {
					break
				}
			}
			i++
		}

		value := strings.TrimSpace(args[valueStart:i])

		// Remove quotes if present
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			var err error
			value, err = strconv.Unquote(value)
			if err != nil {
				return nil, fmt.Errorf("invalid quoted string: %s", value)
			}
		}

		result[key] = value

		// Skip comma
		if i < len(args) && args[i] == ',' {
			i++
		}
	}

	return result, nil
}

// parseValue parses a string value according to the argument type and sets it in the output.
func (d *Definition) parseValue(value string, argType Argument, out reflect.Value) error {
	// Handle pointer types
	if argType.Optional && out.Type().Kind() == reflect.Ptr {
		if value == "" {
			// Leave as nil for empty optional values
			return nil
		}
		// Create new value and set it
		newVal := reflect.New(out.Type().Elem())
		if err := d.parseValue(value, Argument{Type: argType.Type, ItemType: argType.ItemType}, reflect.Indirect(newVal)); err != nil {
			return err
		}
		out.Set(newVal)
		return nil
	}

	switch argType.Type {
	case StringType:
		out.SetString(value)
		return nil

	case IntType:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer: %s", value)
		}
		out.SetInt(intVal)
		return nil

	case BoolType:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean: %s", value)
		}
		out.SetBool(boolVal)
		return nil

	case SliceType:
		return d.parseSlice(value, argType, out)

	case MapType:
		return d.parseMap(value, argType, out)

	case AnyType:
		// For AnyType, try to guess the type and parse accordingly
		return d.parseAnyType(value, out)

	default:
		return fmt.Errorf("unsupported argument type: %v", argType.Type)
	}
}

// parseSlice parses a slice value like "{val1,val2,val3}" or "val1;val2;val3".
func (d *Definition) parseSlice(value string, argType Argument, out reflect.Value) error {
	if argType.ItemType == nil {
		return fmt.Errorf("slice type missing item type")
	}

	var items []string

	// Handle curly brace format: {val1,val2,val3}
	if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
		inner := value[1 : len(value)-1]
		if inner == "" {
			// Empty slice
			out.Set(reflect.MakeSlice(out.Type(), 0, 0))
			return nil
		}
		items = strings.Split(inner, ",")
	} else {
		// Handle semicolon format: val1;val2;val3
		items = strings.Split(value, ";")
	}

	// Create slice
	slice := reflect.MakeSlice(out.Type(), 0, len(items))
	elemType := out.Type().Elem()

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		// Create new element
		elem := reflect.New(elemType).Elem()
		if err := d.parseValue(item, *argType.ItemType, elem); err != nil {
			return fmt.Errorf("slice item: %w", err)
		}
		slice = reflect.Append(slice, elem)
	}

	out.Set(slice)
	return nil
}

// parseMap parses a map value like "{key1:val1,key2:val2}".
func (d *Definition) parseMap(value string, argType Argument, out reflect.Value) error {
	if argType.ItemType == nil {
		return fmt.Errorf("map type missing value type")
	}

	// Must be in curly brace format
	if !strings.HasPrefix(value, "{") || !strings.HasSuffix(value, "}") {
		return fmt.Errorf("map values must be in {key:value,key:value} format")
	}

	inner := value[1 : len(value)-1]
	if inner == "" {
		// Empty map
		out.Set(reflect.MakeMap(out.Type()))
		return nil
	}

	// Create map
	mapVal := reflect.MakeMap(out.Type())
	keyType := out.Type().Key()
	valueType := out.Type().Elem()

	// Parse key:value pairs
	pairs := strings.Split(inner, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		kv := strings.SplitN(pair, ":", 2)
		if len(kv) != 2 {
			return fmt.Errorf("invalid map pair: %s (expected key:value)", pair)
		}

		keyStr := strings.TrimSpace(kv[0])
		valueStr := strings.TrimSpace(kv[1])

		// Parse key (always string for now)
		key := reflect.New(keyType).Elem()
		key.SetString(keyStr)

		// Parse value
		val := reflect.New(valueType).Elem()
		if err := d.parseValue(valueStr, *argType.ItemType, val); err != nil {
			return fmt.Errorf("map value for key %s: %w", keyStr, err)
		}

		mapVal.SetMapIndex(key, val)
	}

	out.Set(mapVal)
	return nil
}

// parseAnyType tries to guess the type and parse accordingly.
func (d *Definition) parseAnyType(value string, out reflect.Value) error {
	// Try different types in order of preference

	// Try boolean
	if boolVal, err := strconv.ParseBool(value); err == nil {
		out.Set(reflect.ValueOf(boolVal))
		return nil
	}

	// Try integer
	if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
		out.Set(reflect.ValueOf(intVal))
		return nil
	}

	// Try float
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		out.Set(reflect.ValueOf(floatVal))
		return nil
	}

	// Default to string
	out.Set(reflect.ValueOf(value))
	return nil
}

// isMarkerComment checks if a comment text is a marker (starts with +).
func isMarkerComment(comment string) bool {
	if len(comment) < 3 {
		return false
	}
	// Remove "//" prefix
	text := strings.TrimSpace(comment[2:])
	return len(text) > 0 && text[0] == '+'
}

// extractMarkerText extracts the marker text from a comment, removing "//" and whitespace.
func extractMarkerText(comment string) string {
	if len(comment) < 2 {
		return ""
	}
	return strings.TrimSpace(comment[2:])
}
