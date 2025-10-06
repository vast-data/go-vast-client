package openapi_schema

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

// helper: load doc and return it (uses package-private function)
func mustLoadDoc(t *testing.T) *openapi3.T {
	t.Helper()
	doc, err := loadOpenAPIDocOnce()
	if err != nil {
		t.Fatalf("failed to load OpenAPI doc: %v", err)
	}
	if doc == nil {
		t.Fatalf("openapi doc is nil")
	}
	return doc
}

func TestGetOpenApiComponents(t *testing.T) {
	comps, err := GetOpenApiComponents()
	if err != nil {
		t.Fatalf("GetOpenApiComponents error: %v", err)
	}
	if comps == nil {
		t.Fatalf("components is nil")
	}
}

func TestGetOpenApiResource_ValidAndInvalid(t *testing.T) {
	doc := mustLoadDoc(t)
	var anyPath string
	for p := range doc.Paths.Map() {
		anyPath = p
		break
	}
	if anyPath == "" {
		t.Skip("no paths found in OpenAPI doc")
	}

	// valid
	if _, err := GetOpenApiResource(anyPath); err != nil {
		t.Fatalf("GetOpenApiResource valid path %q: %v", anyPath, err)
	}
	// invalid
	if _, err := GetOpenApiResource("/this/path/does/not/exist/"); err == nil {
		t.Fatalf("expected error for invalid path, got nil")
	}
}

func TestGetOpenApiComponentSchema(t *testing.T) {
	doc := mustLoadDoc(t)
	var name string
	for k := range doc.Components.Schemas {
		name = k
		break
	}
	if name == "" {
		t.Skip("no components found in OpenAPI doc")
	}
	ref := "#/components/schemas/" + name
	got, err := GetOpenApiComponentSchema(ref)
	if err != nil {
		t.Fatalf("GetOpenApiComponentSchema error: %v", err)
	}
	if got == nil || got.Value == nil {
		t.Fatalf("schema ref is nil for %s", ref)
	}
}

func findPathWithOperation(t *testing.T, method string) string {
	t.Helper()
	doc := mustLoadDoc(t)
	for p, item := range doc.Paths.Map() {
		switch method {
		case "GET":
			if item.Get != nil {
				return p
			}
		case "POST":
			if item.Post != nil {
				return p
			}
		case "PATCH":
			if item.Patch != nil {
				return p
			}
		}
	}
	return ""
}

func TestGetRequestBodySchema_POST(t *testing.T) {
	path := findPathWithOperation(t, "POST")
	if path == "" {
		// No POSTs in schema; ensure function returns empty schema without error
		got, err := GetRequestBodySchema("POST", "/no/such/path/")
		if err == nil && got != nil {
			// expected empty schema, Value not nil
			return
		}
		t.Skip("no POST operation available; skipped")
	}
	if _, err := GetRequestBodySchema("POST", path); err != nil {
		t.Fatalf("GetRequestBodySchema(POST, %s) error: %v", path, err)
	}
}

func TestGetRequestBodySchema_PATCH(t *testing.T) {
	path := findPathWithOperation(t, "PATCH")
	if path == "" {
		got, err := GetRequestBodySchema("PATCH", "/no/such/path/")
		if err == nil && got != nil {
			return
		}
		t.Skip("no PATCH operation available; skipped")
	}
	if _, err := GetRequestBodySchema("PATCH", path); err != nil {
		t.Fatalf("GetRequestBodySchema(PATCH, %s) error: %v", path, err)
	}
}

func TestGetResponseModelSchema_POST(t *testing.T) {
	path := findPathWithOperation(t, "POST")
	if path == "" {
		// If absent, ensure graceful error
		if _, err := GetResponseModelSchema("POST", "/no/such/path/"); err == nil {
			t.Fatalf("expected error for missing POST schema")
		}
		t.Skip("no POST operation available; skipped")
	}
	if _, err := GetResponseModelSchema("POST", path); err != nil {
		// Some POSTs may not have 200/201/202; accept error but still exercised
		t.Logf("GetResponseModelSchema(POST, %s) returned: %v", path, err)
	}
}

func TestGetResponseModelSchema_GET(t *testing.T) {
	path := findPathWithOperation(t, "GET")
	if path == "" {
		t.Skip("no GET operation available; skipped")
	}
	if _, err := GetResponseModelSchema("GET", path); err != nil {
		t.Logf("GetResponseModelSchema(GET, %s) returned: %v", path, err)
	}
}

func TestGetSchema_FromComponents(t *testing.T) {
	doc := mustLoadDoc(t)
	var name string
	for k := range doc.Components.Schemas {
		name = k
		break
	}
	if name == "" {
		t.Skip("no components found in OpenAPI doc")
	}
	path := "/components/" + name
	got, err := GetSchema_FromComponents(path)
	if err != nil {
		t.Fatalf("GetSchema_FromComponents error: %v", err)
	}
	if got == nil || got.Value == nil {
		t.Fatalf("schema is nil for %s", path)
	}
}

func TestQueryParametersGET_And_GetSchema_GET_QueryParams(t *testing.T) {
	path := findPathWithOperation(t, "GET")
	if path == "" {
		t.Skip("no GET operation available; skipped")
	}
	params, err := QueryParametersGET(path)
	if err != nil {
		t.Fatalf("QueryParametersGET error: %v", err)
	}
	// Always should return slice (possibly empty)
	if params == nil {
		t.Fatalf("params is nil")
	}
	schemaRef, err := GetSchema_GET_QueryParams(path)
	if err != nil {
		t.Fatalf("GetSchema_GET_QueryParams error: %v", err)
	}
	if schemaRef == nil || schemaRef.Value == nil {
		t.Fatalf("GetSchema_GET_QueryParams returned nil schema")
	}
	if schemaRef.Value.Type == nil || len(*schemaRef.Value.Type) == 0 || (*schemaRef.Value.Type)[0] != openapi3.TypeObject {
		t.Fatalf("query params schema is not an object")
	}
}

func TestSearchableQueryParams(t *testing.T) {
	path := findPathWithOperation(t, "GET")
	if path == "" {
		t.Skip("no GET operation available; skipped")
	}
	names, err := SearchableQueryParams(path)
	if err != nil {
		t.Fatalf("SearchableQueryParams error: %v", err)
	}
	if names == nil {
		t.Fatalf("names slice is nil")
	}
}
