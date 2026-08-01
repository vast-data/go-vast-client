package openapi_schema

import (
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestFilterPaths(t *testing.T) {
	paths := []string{"/users/", "/groups/", "/tenants/"}
	if got := filterPaths(paths, "user"); len(got) != 1 || got[0] != "/users/" {
		t.Fatalf("filterPaths = %v", got)
	}
	if got := filterPaths(paths, "missing"); len(got) != 1 || got[0] != "none" {
		t.Fatalf("filterPaths missing = %v", got)
	}
}

func TestFilterPathsCaseInsensitive(t *testing.T) {
	paths := []string{"/Users/", "/groups/"}
	if got := filterPathsCaseInsensitive(paths, "users"); len(got) != 1 {
		t.Fatalf("filterPathsCaseInsensitive = %v", got)
	}
}

func TestValidateOperationExists_ErrorPaths(t *testing.T) {
	if err := ValidateOperationExists("GET", "/this/path/absolutely/does/not/exist/"); err == nil {
		t.Fatal("expected missing path error")
	}

	path := findPathWithOperation(t, "GET")
	if path == "" {
		t.Skip("no GET path")
	}
	getOnly := findGETOnlyPath(t)
	if getOnly == "" {
		t.Skip("no GET-only path")
	}
	if err := ValidateOperationExists("PATCH", getOnly); err == nil {
		t.Fatal("expected missing method error")
	}
	if err := ValidateOperationExists("INVALID", path); err == nil {
		t.Fatal("expected unsupported method error")
	}
}

func TestGetQueryParameters_AllMethods(t *testing.T) {
	path := findPathWithOperation(t, "GET")
	if path == "" {
		t.Skip("no GET path")
	}
	for _, method := range []string{"GET", "POST", "PATCH", "PUT", "DELETE", "HEAD", "OPTIONS"} {
		params, err := GetQueryParameters(method, path)
		if err != nil {
			t.Fatalf("%s: %v", method, err)
		}
		if params == nil {
			t.Fatalf("%s: nil params", method)
		}
	}
	if _, err := GetQueryParameters("TRACE", path); err == nil {
		t.Fatal("expected unsupported method error")
	}
}

func TestIsDirectComponentReference_AllCases(t *testing.T) {
	if IsDirectComponentReference(nil) != "" {
		t.Fatal("nil ref")
	}
	if IsDirectComponentReference(&openapi3.SchemaRef{}) != "" {
		t.Fatal("empty ref")
	}
	if got := IsDirectComponentReference(&openapi3.SchemaRef{Ref: "#/components/schemas/User"}); got != "User" {
		t.Fatalf("got %q", got)
	}
	if got := IsDirectComponentReference(&openapi3.SchemaRef{Ref: "#/definitions/Legacy"}); got != "Legacy" {
		t.Fatalf("got %q", got)
	}
	if got := IsDirectComponentReference(&openapi3.SchemaRef{Ref: "#/other/Thing"}); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveComposedSchema_OneOfAnyOf(t *testing.T) {
	stringType := openapi3.TypeString
	intType := openapi3.TypeInteger
	schema := &openapi3.Schema{
		OneOf: []*openapi3.SchemaRef{
			{Value: &openapi3.Schema{Type: &openapi3.Types{stringType}}},
			{Value: &openapi3.Schema{Type: &openapi3.Types{intType}}},
		},
	}
	resolved := ResolveComposedSchema(schema)
	if resolved == nil || resolved.Type == nil || (*resolved.Type)[0] != stringType {
		t.Fatalf("unexpected resolved schema: %+v", resolved)
	}
}

func TestGetSchema_GET_QueryParams_InvalidPath(t *testing.T) {
	if _, err := GetSchema_GET_QueryParams("/missing/path/"); err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestGetRequestBodySchema_AllMethods(t *testing.T) {
	for _, method := range []string{"POST", "PATCH", "PUT", "DELETE"} {
		path := findPathWithOperation(t, method)
		if path == "" {
			continue
		}
		if _, err := GetRequestBodySchema(method, path); err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
	}
	if _, err := GetRequestBodySchema("TRACE", "/users/"); err == nil {
		t.Fatal("expected unsupported method")
	}
}

func TestGetResponseModelSchema_AllMethods(t *testing.T) {
	for _, method := range []string{"GET", "POST", "PATCH", "PUT", "DELETE"} {
		path := findPathWithOperation(t, method)
		if path == "" {
			continue
		}
		if _, err := GetResponseModelSchema(method, path); err != nil {
			t.Logf("%s %s: %v", method, path, err)
		}
	}
}

func TestGetResponseModelSchemaUnresolved_GET(t *testing.T) {
	path := findPathWithOperation(t, "GET")
	if path == "" {
		t.Skip("no GET path")
	}
	if _, err := GetResponseModelSchemaUnresolved("GET", path); err != nil {
		t.Fatalf("GetResponseModelSchemaUnresolved: %v", err)
	}
}

func TestReturnsTextPlain_And204(t *testing.T) {
	doc := mustLoadDoc(t)
	var plainPath, noContentPath string
	for p, item := range doc.Paths.Map() {
		if item.Get != nil && item.Get.Responses != nil {
			if resp := item.Get.Responses.Status(200); resp != nil && resp.Value != nil && resp.Value.Content != nil {
				for ct := range resp.Value.Content {
					if strings.Contains(ct, "text/plain") {
						plainPath = p
						break
					}
				}
			}
			if item.Get.Responses.Status(204) != nil {
				noContentPath = p
			}
		}
		if plainPath != "" && noContentPath != "" {
			break
		}
	}

	if plainPath != "" {
		ok, err := ReturnsTextPlain("GET", plainPath)
		if err != nil {
			t.Fatalf("ReturnsTextPlain: %v", err)
		}
		if !ok {
			t.Fatal("expected text/plain endpoint")
		}
	} else if path := findPrometheusLikePath(doc); path != "" {
		ok, err := ReturnsTextPlain("GET", path)
		if err != nil {
			t.Fatalf("ReturnsTextPlain heuristic: %v", err)
		}
		if !ok {
			t.Log("prometheus heuristic path did not match")
		}
	}

	if noContentPath != "" {
		ok, err := Returns204NoContent("GET", noContentPath)
		if err != nil {
			t.Fatalf("Returns204NoContent: %v", err)
		}
		_ = ok
	}
}

func findPrometheusLikePath(doc *openapi3.T) string {
	for p := range doc.Paths.Map() {
		if strings.Contains(strings.ToLower(p), "prometheus") {
			return p
		}
	}
	return ""
}

func TestGetOperationSummary_AllMethods(t *testing.T) {
	for _, method := range []string{"GET", "POST", "PATCH", "PUT", "DELETE"} {
		path := findPathWithOperation(t, method)
		if path == "" {
			continue
		}
		if _, err := GetOperationSummary(method, path); err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
	}
}

func TestGetOpenApiComponentSchema_InvalidRef(t *testing.T) {
	if ref, err := GetOpenApiComponentSchema("DefinitelyMissingComponentXYZ"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if ref != nil {
		t.Fatal("expected nil schema ref for missing component")
	}
}

func findGETOnlyPath(t *testing.T) string {
	t.Helper()
	doc := mustLoadDoc(t)
	for p, item := range doc.Paths.Map() {
		if item.Get == nil {
			continue
		}
		if item.Post == nil && item.Patch == nil && item.Put == nil && item.Delete == nil {
			return p
		}
	}
	return ""
}

func TestSearchableQueryParams_WithMatches(t *testing.T) {
	doc := mustLoadDoc(t)
	for p, item := range doc.Paths.Map() {
		if item.Get == nil {
			continue
		}
		names, err := SearchableQueryParams(p)
		if err != nil {
			t.Fatalf("SearchableQueryParams(%q): %v", p, err)
		}
		if len(names) > 0 {
			schema, err := GetSchema_GET_QueryParams(p)
			if err != nil {
				t.Fatalf("GetSchema_GET_QueryParams(%q): %v", p, err)
			}
			if schema == nil || schema.Value == nil {
				t.Fatalf("nil schema for %q", p)
			}
			return
		}
	}
	t.Log("no searchable query params found in schema sample")
}

func TestReturns204NoContent_MissingPath(t *testing.T) {
	if _, err := Returns204NoContent("GET", "/missing/path/"); err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestReturnsTextPlain_MissingPath(t *testing.T) {
	if _, err := ReturnsTextPlain("GET", "/missing/path/"); err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestGetOperationSummary_MissingPath(t *testing.T) {
	if _, err := GetOperationSummary("GET", "/missing/path/"); err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestGetDeleteParams_WithDetails(t *testing.T) {
	doc := mustLoadDoc(t)
	for p, item := range doc.Paths.Map() {
		if item.Delete == nil {
			continue
		}
		base := strings.TrimSuffix(strings.Trim(p, "/"), "/{id}")
		if base == "" {
			continue
		}
		params, err := GetDeleteParams(base)
		if err != nil {
			continue
		}
		if params == nil {
			t.Fatal("expected delete params")
		}
		return
	}
	t.Skip("no suitable DELETE params path found")
}

func TestGetResponseModelSchemaUnresolved_AllMethods(t *testing.T) {
	for _, method := range []string{"POST", "PATCH", "PUT", "DELETE"} {
		path := findPathWithOperation(t, method)
		if path == "" {
			continue
		}
		if _, err := GetResponseModelSchemaUnresolved(method, path); err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
	}
}

func TestGetOpenApiResource_InvalidPathListsAvailable(t *testing.T) {
	if _, err := GetOpenApiResource("/this/path/does/not/exist/"); err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestGetQueryParameters_InvalidPath(t *testing.T) {
	if _, err := GetQueryParameters("GET", "/this/path/does/not/exist/"); err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestIsStringOrInteger_Internal(t *testing.T) {
	if !isStringOrInteger(&openapi3.Schema{Type: &openapi3.Types{openapi3.TypeString}}) {
		t.Fatal("string should match")
	}
	if isStringOrInteger(&openapi3.Schema{Type: &openapi3.Types{openapi3.TypeBoolean}}) {
		t.Fatal("boolean should not match")
	}
}

func TestCompareSchemaValues_ObjectMismatch(t *testing.T) {
	objType := openapi3.TypeObject
	a := &openapi3.Schema{
		Type: &openapi3.Types{objType},
		Properties: map[string]*openapi3.SchemaRef{
			"a": {Value: &openapi3.Schema{Type: &openapi3.Types{openapi3.TypeString}}},
		},
	}
	b := &openapi3.Schema{
		Type: &openapi3.Types{objType},
		Properties: map[string]*openapi3.SchemaRef{
			"b": {Value: &openapi3.Schema{Type: &openapi3.Types{openapi3.TypeString}}},
		},
	}
	if _, ok := CompareSchemaValues(a, b); ok {
		t.Fatal("expected property mismatch")
	}
}

func TestGetDeleteParams_ErrorPaths(t *testing.T) {
	if _, err := GetDeleteParams("/missing/resource/"); err == nil {
		t.Fatal("expected error for missing path")
	}

	doc := mustLoadDoc(t)
	for p, item := range doc.Paths.Map() {
		if item.Delete != nil {
			continue
		}
		trimmed := strings.Trim(p, "/")
		if !strings.HasSuffix(trimmed, "/{id}") {
			continue
		}
		base := strings.TrimSuffix(trimmed, "/{id}")
		if base == "" {
			continue
		}
		_, err := GetDeleteParams(base)
		if err == nil {
			continue
		}
		if strings.Contains(err.Error(), "DELETE operation not found") {
			return
		}
	}
	t.Skip("no /{resource}/{id}/ path without DELETE in schema")
}

func TestGetSchemaFromComponent_Invalid(t *testing.T) {
	if _, err := GetSchemaFromComponent("DefinitelyMissingComponentXYZ"); err == nil {
		t.Fatal("expected error for missing component")
	}
}

func TestGetSchema_FromComponents_Invalid(t *testing.T) {
	if _, err := GetSchema_FromComponents("/DefinitelyMissingComponentXYZ/"); err == nil {
		t.Fatal("expected error for missing component path")
	}
}

func TestValidateOperationExists_CaseInsensitivePath(t *testing.T) {
	path := findPathWithOperation(t, "GET")
	if path == "" {
		t.Skip("no GET path")
	}
	upper := strings.ToUpper(strings.Trim(path, "/"))
	if err := ValidateOperationExists("GET", "/"+upper+"/"); err != nil {
		t.Fatalf("case-insensitive path should match: %v", err)
	}
}

func TestReturnsTextPlain_UnsupportedMethod(t *testing.T) {
	path := findPathWithOperation(t, "GET")
	if path == "" {
		t.Skip("no GET path")
	}
	if _, err := ReturnsTextPlain("TRACE", path); err == nil {
		t.Fatal("expected unsupported method error")
	}
}

func TestReturnsTextPlain_OperationMissing(t *testing.T) {
	path := findGETOnlyPath(t)
	if path == "" {
		t.Skip("no GET-only path")
	}
	if _, err := ReturnsTextPlain("POST", path); err == nil {
		t.Fatal("expected missing operation error")
	}
}

func TestReturnsTextPlain_PathWithoutTrailingSlash(t *testing.T) {
	path := findPathWithOperation(t, "GET")
	if path == "" {
		t.Skip("no GET path")
	}
	trimmed := strings.TrimSuffix(path, "/")
	ok, err := ReturnsTextPlain("GET", trimmed)
	if err != nil {
		t.Fatalf("ReturnsTextPlain: %v", err)
	}
	_ = ok
}

func TestReturns204NoContent_UnsupportedMethod(t *testing.T) {
	path := findPathWithOperation(t, "GET")
	if path == "" {
		t.Skip("no GET path")
	}
	if _, err := Returns204NoContent("INVALID", path); err == nil {
		t.Fatal("expected unsupported method error")
	}
}

func TestGetResponseModelSchema_GETVariants(t *testing.T) {
	doc := mustLoadDoc(t)
	for p, item := range doc.Paths.Map() {
		if item.Get == nil {
			continue
		}
		if _, err := GetResponseModelSchema("GET", p); err != nil {
			t.Logf("GET %s: %v", p, err)
			continue
		}
		return
	}
	t.Skip("no GET response schema found")
}

func TestGetOpenApiComponentSchema_WithSlashRef(t *testing.T) {
	doc := mustLoadDoc(t)
	if doc.Components == nil || len(doc.Components.Schemas) == 0 {
		t.Skip("no component schemas")
	}
	for name := range doc.Components.Schemas {
		ref, err := GetOpenApiComponentSchema("components/schemas/" + name)
		if err != nil {
			t.Fatalf("GetOpenApiComponentSchema: %v", err)
		}
		if ref == nil {
			t.Fatal("expected schema ref")
		}
		return
	}
}

func TestGetDeleteParams_AllDeletePaths(t *testing.T) {
	doc := mustLoadDoc(t)
	found := false
	for p, item := range doc.Paths.Map() {
		if item.Delete == nil {
			continue
		}
		base := strings.TrimSuffix(strings.Trim(p, "/"), "/{id}")
		if base == "" {
			continue
		}
		params, err := GetDeleteParams(base)
		if err != nil {
			continue
		}
		found = true
		if params == nil {
			t.Fatal("expected delete params")
		}
		_ = params.QueryParams
		_ = params.BodySchema
		_ = params.IdDescription
	}
	if !found {
		t.Skip("no DELETE params found in schema")
	}
}

func TestSearchableQueryParams_ReadOnlySkipped(t *testing.T) {
	doc := mustLoadDoc(t)
	for p, item := range doc.Paths.Map() {
		if item.Get == nil {
			continue
		}
		names, err := SearchableQueryParams(p)
		if err != nil {
			t.Fatalf("SearchableQueryParams(%q): %v", p, err)
		}
		for _, name := range names {
			params, err := GetQueryParameters("GET", p)
			if err != nil {
				t.Fatalf("GetQueryParameters: %v", err)
			}
			for _, param := range params {
				if param.Name == name && param.Schema != nil && param.Schema.Value != nil && param.Schema.Value.ReadOnly {
					t.Fatalf("read-only param %q should be skipped", name)
				}
			}
		}
		return
	}
	t.Skip("no GET paths in schema")
}

func TestGetResponseModelSchemaUnresolved_AllGETPaths(t *testing.T) {
	doc := mustLoadDoc(t)
	for p, item := range doc.Paths.Map() {
		if item.Get == nil {
			continue
		}
		if _, err := GetResponseModelSchemaUnresolved("GET", p); err != nil {
			t.Fatalf("GET %s: %v", p, err)
		}
	}
}

func TestReturnsTextPlain_PrometheusHeuristic(t *testing.T) {
	doc := mustLoadDoc(t)
	for p, item := range doc.Paths.Map() {
		if item.Get == nil || item.Get.Responses == nil {
			continue
		}
		resp := item.Get.Responses.Status(200)
		if resp == nil || resp.Value == nil {
			continue
		}
		if resp.Value.Content != nil {
			continue
		}
		desc := ""
		if resp.Value.Description != nil {
			desc = *resp.Value.Description
		}
		if strings.Contains(strings.ToLower(p), "prometheus") || strings.Contains(strings.ToLower(desc), "prometheus") {
			ok, err := ReturnsTextPlain("GET", p)
			if err != nil {
				t.Fatalf("ReturnsTextPlain(%q): %v", p, err)
			}
			if !ok {
				t.Fatalf("expected prometheus heuristic match for %q", p)
			}
			return
		}
	}
	t.Skip("no prometheus heuristic endpoint in schema")
}
