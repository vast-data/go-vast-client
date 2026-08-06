package openapi_schema

import (
	"testing"
)

func TestGetResponseModelSchema_AllMethodsOnAllPaths(t *testing.T) {
	doc := mustLoadDoc(t)
	methods := []string{"GET", "POST", "PATCH", "PUT", "DELETE"}
	for p := range doc.Paths.Map() {
		for _, method := range methods {
			schema, err := GetResponseModelSchema(method, p)
			if err != nil {
				continue
			}
			if schema == nil {
				t.Fatalf("%s %s: nil schema without error", method, p)
			}
		}
	}
}

func TestGetResponseModelSchema_UnsupportedMethod(t *testing.T) {
	path := findPathWithOperation(t, "GET")
	if path == "" {
		t.Skip("no GET path")
	}
	if _, err := GetResponseModelSchema("TRACE", path); err == nil {
		t.Fatal("expected unsupported method error")
	}
}

func TestGetResponseModelSchema_InvalidResourcePath(t *testing.T) {
	if _, err := GetResponseModelSchema("GET", "/this/path/absolutely/does/not/exist/"); err == nil {
		t.Fatal("expected error for invalid resource path")
	}
}

func TestGetResponseModelSchemaUnresolved_AllMethodsOnAllPaths(t *testing.T) {
	doc := mustLoadDoc(t)
	methods := []string{"GET", "POST", "PATCH", "PUT", "DELETE"}
	for p := range doc.Paths.Map() {
		for _, method := range methods {
			_, _ = GetResponseModelSchemaUnresolved(method, p)
		}
	}
}

func TestGetResponseModelSchemaUnresolved_UnsupportedMethod(t *testing.T) {
	path := findPathWithOperation(t, "GET")
	if path == "" {
		t.Skip("no GET path")
	}
	if _, err := GetResponseModelSchemaUnresolved("TRACE", path); err == nil {
		t.Fatal("expected unsupported method error")
	}
}
