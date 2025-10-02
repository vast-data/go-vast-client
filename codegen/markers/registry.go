package markers

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

// Registry holds all registered marker definitions and provides lookup functionality.
type Registry struct {
	definitions map[string]*Definition
}

// NewRegistry creates a new marker registry.
func NewRegistry() *Registry {
	return &Registry{
		definitions: make(map[string]*Definition),
	}
}

// Register adds a marker definition to the registry.
// name: the marker name (e.g., "vast:generate")
// target: what the marker can be applied to (package, type, field)
// outputType: the Go type that the marker parses into
// description: help text for the marker
func (r *Registry) Register(name string, target TargetType, outputType interface{}, description string) error {
	def := &Definition{
		Name:        name,
		Target:      target,
		OutputType:  reflect.TypeOf(outputType),
		Fields:      make(map[string]Argument),
		FieldNames:  make(map[string]string),
		Description: description,
	}

	if err := r.analyzeOutputType(def); err != nil {
		return fmt.Errorf("failed to analyze output type for marker %s: %w", name, err)
	}

	r.definitions[name] = def
	return nil
}

// MustRegister is like Register but panics on error.
func (r *Registry) MustRegister(name string, target TargetType, outputType interface{}, description string) {
	if err := r.Register(name, target, outputType, description); err != nil {
		panic(err)
	}
}

// Lookup finds a marker definition by name and target type.
func (r *Registry) Lookup(markerText string, target TargetType) *Definition {
	name := r.extractMarkerName(markerText)
	def, exists := r.definitions[name]
	if !exists || def.Target != target {
		return nil
	}
	return def
}

// GetDefinition returns a marker definition by name, regardless of target type.
func (r *Registry) GetDefinition(name string) *Definition {
	return r.definitions[name]
}

// ListDefinitions returns all registered marker definitions.
func (r *Registry) ListDefinitions() map[string]*Definition {
	result := make(map[string]*Definition)
	for name, def := range r.definitions {
		result[name] = def
	}
	return result
}

// extractMarkerName extracts the marker name from marker text.
// For example, "+vast:generate=config" -> "vast:generate"
func (r *Registry) extractMarkerName(markerText string) string {
	// Remove leading "+" if present
	if strings.HasPrefix(markerText, "+") {
		markerText = markerText[1:]
	}

	// Split on "=" to separate name from arguments
	parts := strings.SplitN(markerText, "=", 2)
	return strings.TrimSpace(parts[0])
}

// analyzeOutputType analyzes the output type to determine field information.
func (r *Registry) analyzeOutputType(def *Definition) error {
	if def.OutputType.Kind() != reflect.Struct {
		// For non-struct types, create a single anonymous field
		argType, err := r.argumentFromType(def.OutputType)
		if err != nil {
			return err
		}
		def.Fields[""] = argType
		def.FieldNames[""] = ""
		return nil
	}

	// For struct types, analyze each field
	for i := 0; i < def.OutputType.NumField(); i++ {
		field := def.OutputType.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		// Get the argument name (convert PascalCase to camelCase)
		argName := r.fieldToArgName(field.Name)

		// Check for marker tag overrides
		if markerTag := field.Tag.Get("marker"); markerTag != "" {
			parts := strings.Split(markerTag, ",")
			if parts[0] != "" {
				argName = parts[0]
			}

			// Check for optional flag
			for _, part := range parts[1:] {
				if part == "optional" {
					// Will be handled when creating the argument
				}
			}
		}

		argType, err := r.argumentFromType(field.Type)
		if err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}

		// Check if field is optional (pointer type or has optional tag)
		if field.Type.Kind() == reflect.Ptr {
			argType.Optional = true
		}
		if markerTag := field.Tag.Get("marker"); markerTag != "" && strings.Contains(markerTag, "optional") {
			argType.Optional = true
		}

		def.Fields[argName] = argType
		def.FieldNames[argName] = field.Name
	}

	return nil
}

// argumentFromType creates an Argument from a reflect.Type.
func (r *Registry) argumentFromType(typ reflect.Type) (Argument, error) {
	arg := Argument{}

	// Handle pointer types
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		arg.Optional = true
	}

	switch typ.Kind() {
	case reflect.String:
		arg.Type = StringType
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		arg.Type = IntType
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		arg.Type = IntType
	case reflect.Bool:
		arg.Type = BoolType
	case reflect.Slice:
		arg.Type = SliceType
		itemType, err := r.argumentFromType(typ.Elem())
		if err != nil {
			return Argument{}, fmt.Errorf("slice element type: %w", err)
		}
		arg.ItemType = &itemType
	case reflect.Map:
		if typ.Key().Kind() != reflect.String {
			return Argument{}, fmt.Errorf("map keys must be strings, got %s", typ.Key().Kind())
		}
		arg.Type = MapType
		valueType, err := r.argumentFromType(typ.Elem())
		if err != nil {
			return Argument{}, fmt.Errorf("map value type: %w", err)
		}
		arg.ItemType = &valueType
	case reflect.Interface:
		if typ == reflect.TypeOf((*interface{})(nil)).Elem() {
			arg.Type = AnyType
		} else {
			return Argument{}, fmt.Errorf("unsupported interface type: %s", typ)
		}
	default:
		return Argument{}, fmt.Errorf("unsupported type: %s", typ.Kind())
	}

	return arg, nil
}

// fieldToArgName converts a struct field name to an argument name.
// Converts PascalCase to camelCase (e.g., "MaxLength" -> "maxLength").
func (r *Registry) fieldToArgName(fieldName string) string {
	if fieldName == "" {
		return ""
	}

	runes := []rune(fieldName)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}
