package openapi_schema

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

// IsObject returns true if the given OpenAPI schema represents an object type
// (possibly as one alternative in a multi-type oneOf/anyOf merge).
func IsObject(prop *openapi3.Schema) bool {
	if prop == nil {
		return false
	}
	if len(prop.Properties) > 0 {
		return true
	}
	return schemaHasType(prop, openapi3.TypeObject)
}

func schemaHasType(prop *openapi3.Schema, want string) bool {
	if prop == nil || prop.Type == nil {
		return false
	}
	for _, t := range *prop.Type {
		if t == want {
			return true
		}
	}
	return false
}

// IsAmbiguousObject returns true if the schema is an object with no properties and no additionalProperties.
// Objects with additionalProperties (like maps) are valid and should not be considered ambiguous.
func IsAmbiguousObject(prop *openapi3.Schema) bool {
	return IsObject(prop) && len(prop.Properties) == 0 && prop.AdditionalProperties.Schema == nil
}

// IsPrimitive returns true if the given OpenAPI schema represents only primitive
// type(s) (string, integer, number, or boolean) — not object/array.
func IsPrimitive(prop *openapi3.Schema) bool {
	if prop == nil || prop.Type == nil || len(*prop.Type) == 0 {
		return false
	}
	if len(prop.Properties) > 0 || schemaHasType(prop, openapi3.TypeObject) || schemaHasType(prop, openapi3.TypeArray) {
		return false
	}
	for _, t := range *prop.Type {
		switch t {
		case openapi3.TypeString, openapi3.TypeInteger, openapi3.TypeNumber, openapi3.TypeBoolean:
			continue
		default:
			return false
		}
	}
	return true
}

// IsStringOrInteger returns true if the given OpenAPI schema includes string or integer
// (and no object/array). Multi-type unions like oneOf[string, integer] qualify.
func IsStringOrInteger(prop *openapi3.Schema) bool {
	if prop == nil || prop.Type == nil || len(*prop.Type) == 0 {
		return false
	}
	if len(prop.Properties) > 0 || schemaHasType(prop, openapi3.TypeObject) || schemaHasType(prop, openapi3.TypeArray) {
		return false
	}
	return schemaHasType(prop, openapi3.TypeString) || schemaHasType(prop, openapi3.TypeInteger)
}

// IsEmptySchema returns true if the schema reference is nil or represents an empty schema
func IsEmptySchema(ref *openapi3.SchemaRef) bool {
	if ref == nil || ref.Value == nil {
		return true
	}
	schema := ref.Value
	return (schema.Type == nil || len(*schema.Type) == 0) &&
		len(schema.Properties) == 0 &&
		schema.Items == nil &&
		len(schema.AllOf) == 0 &&
		len(schema.OneOf) == 0 &&
		len(schema.AnyOf) == 0 &&
		len(schema.Required) == 0
}

// GetSchemaType returns the primary type string of the given OpenAPI schema.
// For multi-type schemas produced by merging oneOf/anyOf branches, prefers
// object (especially when properties are present), then array, otherwise the
// first listed type. Callers that need the full union should inspect Type.
func GetSchemaType(s *openapi3.Schema) string {
	if s == nil {
		return ""
	}
	if s.Type == nil || len(*s.Type) == 0 {
		if len(s.Properties) > 0 {
			return openapi3.TypeObject
		}
		return ""
	}
	if len(*s.Type) == 1 {
		return (*s.Type)[0]
	}
	if schemaHasType(s, openapi3.TypeObject) || len(s.Properties) > 0 {
		return openapi3.TypeObject
	}
	if schemaHasType(s, openapi3.TypeArray) {
		return openapi3.TypeArray
	}
	return (*s.Type)[0]
}

// CompareSchemaValues compares two OpenAPI schemas and returns a description of any differences.
// Returns an empty string and true if schemas match, or an error message and false if they differ.
func CompareSchemaValues(a, b *openapi3.Schema) (string, bool) {
	if a == nil || b == nil {
		if a == b {
			return "", true
		}
		return "One schema is nil while the other is not", false
	}

	typeA := GetSchemaType(a)
	typeB := GetSchemaType(b)
	if typeA != typeB {
		return fmt.Sprintf("Type mismatch: %q vs %q", typeA, typeB), false
	}

	// Compare array items
	if typeA == "array" {
		if a.Items == nil || b.Items == nil {
			if a.Items == b.Items {
				return "", true
			}
			return "Array item schema is nil in one but not the other", false
		}
		msg, ok := CompareSchemaValues(a.Items.Value, b.Items.Value)
		if !ok {
			return fmt.Sprintf("Array item mismatch: %s", msg), false
		}
		return "", true
	}

	// Compare object properties
	if typeA == "object" {
		if len(a.Properties) != len(b.Properties) {
			return fmt.Sprintf("Object property count mismatch: %d vs %d", len(a.Properties), len(b.Properties)), false
		}
		for key, valA := range a.Properties {
			valB, ok := b.Properties[key]
			if !ok {
				return fmt.Sprintf("Property %q missing in one schema", key), false
			}
			msg, ok := CompareSchemaValues(valA.Value, valB.Value)
			if !ok {
				return fmt.Sprintf("Property %q mismatch: %s", key, msg), false
			}
		}
	}

	return "", true
}
