package openapi_schema

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func objectSchema(props map[string]*openapi3.SchemaRef) *openapi3.Schema {
	return &openapi3.Schema{
		Type:       &openapi3.Types{openapi3.TypeObject},
		Properties: props,
	}
}

func typeSchema(typ string) *openapi3.Schema {
	return &openapi3.Schema{Type: &openapi3.Types{typ}}
}

func TestIsObject(t *testing.T) {
	if !IsObject(objectSchema(nil)) {
		t.Fatal("expected object schema")
	}
	if IsObject(typeSchema(openapi3.TypeString)) {
		t.Fatal("string schema should not be object")
	}
	if IsObject(nil) {
		t.Fatal("nil schema should not be object")
	}
}

func TestIsAmbiguousObject(t *testing.T) {
	if !IsAmbiguousObject(objectSchema(nil)) {
		t.Fatal("empty object should be ambiguous")
	}
	schema := objectSchema(nil)
	schema.AdditionalProperties = openapi3.AdditionalProperties{Schema: &openapi3.SchemaRef{Value: typeSchema(openapi3.TypeString)}}
	if IsAmbiguousObject(schema) {
		t.Fatal("object with additionalProperties should not be ambiguous")
	}
}

func TestIsPrimitive(t *testing.T) {
	for _, typ := range []string{
		openapi3.TypeString,
		openapi3.TypeInteger,
		openapi3.TypeNumber,
		openapi3.TypeBoolean,
	} {
		if !IsPrimitive(typeSchema(typ)) {
			t.Fatalf("%s should be primitive", typ)
		}
	}
	if IsPrimitive(objectSchema(nil)) {
		t.Fatal("object should not be primitive")
	}
}

func TestIsStringOrInteger(t *testing.T) {
	if !IsStringOrInteger(typeSchema(openapi3.TypeString)) {
		t.Fatal("string should match")
	}
	if !IsStringOrInteger(typeSchema(openapi3.TypeInteger)) {
		t.Fatal("integer should match")
	}
	if IsStringOrInteger(typeSchema(openapi3.TypeBoolean)) {
		t.Fatal("boolean should not match")
	}
}

func TestIsEmptySchema(t *testing.T) {
	if !IsEmptySchema(nil) {
		t.Fatal("nil ref should be empty")
	}
	if !IsEmptySchema(&openapi3.SchemaRef{Value: &openapi3.Schema{}}) {
		t.Fatal("blank schema should be empty")
	}
	if IsEmptySchema(&openapi3.SchemaRef{Value: typeSchema(openapi3.TypeString)}) {
		t.Fatal("typed schema should not be empty")
	}
}

func TestGetSchemaType(t *testing.T) {
	if got := GetSchemaType(typeSchema(openapi3.TypeInteger)); got != openapi3.TypeInteger {
		t.Fatalf("GetSchemaType = %q", got)
	}
	if got := GetSchemaType(nil); got != "" {
		t.Fatalf("GetSchemaType(nil) = %q", got)
	}
}

func TestCompareSchemaValues(t *testing.T) {
	a := typeSchema(openapi3.TypeString)
	b := typeSchema(openapi3.TypeString)
	if msg, ok := CompareSchemaValues(a, b); !ok || msg != "" {
		t.Fatalf("identical schemas should match, got ok=%v msg=%q", ok, msg)
	}

	if _, ok := CompareSchemaValues(a, typeSchema(openapi3.TypeInteger)); ok {
		t.Fatal("type mismatch should not match")
	}

	arrayA := &openapi3.Schema{
		Type:  &openapi3.Types{openapi3.TypeArray},
		Items: &openapi3.SchemaRef{Value: typeSchema(openapi3.TypeString)},
	}
	arrayB := &openapi3.Schema{
		Type:  &openapi3.Types{openapi3.TypeArray},
		Items: &openapi3.SchemaRef{Value: typeSchema(openapi3.TypeInteger)},
	}
	if _, ok := CompareSchemaValues(arrayA, arrayB); ok {
		t.Fatal("array item mismatch should not match")
	}

	objA := objectSchema(map[string]*openapi3.SchemaRef{
		"name": {Value: typeSchema(openapi3.TypeString)},
	})
	objB := objectSchema(map[string]*openapi3.SchemaRef{
		"name": {Value: typeSchema(openapi3.TypeString)},
		"id":   {Value: typeSchema(openapi3.TypeInteger)},
	})
	if _, ok := CompareSchemaValues(objA, objB); ok {
		t.Fatal("property count mismatch should not match")
	}
}

func TestCompareSchemaValues_NilAndArrayNilItems(t *testing.T) {
	if msg, ok := CompareSchemaValues(nil, nil); !ok || msg != "" {
		t.Fatalf("both nil should match, got ok=%v msg=%q", ok, msg)
	}
	if _, ok := CompareSchemaValues(typeSchema(openapi3.TypeString), nil); ok {
		t.Fatal("nil mismatch should not match")
	}

	arrayNoItems := &openapi3.Schema{Type: &openapi3.Types{openapi3.TypeArray}}
	if _, ok := CompareSchemaValues(arrayNoItems, arrayNoItems); !ok {
		t.Fatal("both arrays without items should match")
	}
	if _, ok := CompareSchemaValues(arrayNoItems, &openapi3.Schema{
		Type:  &openapi3.Types{openapi3.TypeArray},
		Items: &openapi3.SchemaRef{Value: typeSchema(openapi3.TypeString)},
	}); ok {
		t.Fatal("array nil items mismatch should not match")
	}
}
