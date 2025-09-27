package api

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
)

var (
	//go:embed schema/api.tar.gz
	FS             embed.FS
	openApiDocOnce sync.Once
	openApiDoc     *openapi3.T
	openApiDocErr  error
	shemaRelPath   = "schema/api.tar.gz"
)

// loadOpenAPIDocOnce loads and parses the OpenAPI v3 document from a .tar.gz archive exactly once.
// It looks for a file named "openapi-v3.json" inside the archive located at "api/openapi-v3.tar.gz".
// The document is parsed using the kin-openapi loader and cached for future calls.
//
// Returns:
//   - *openapi3.T: the parsed OpenAPI document.
//   - error: if the archive cannot be read, the JSON file is not found, or the document fails to parse.
//
// Notes:
//   - This function is thread-safe and memoized via sync.Once to ensure the document is only loaded once.
//   - Errors encountered during the initial load are also cached and returned on subsequent calls.
func loadOpenAPIDocOnce() (*openapi3.T, error) {
	openApiDocOnce.Do(func() {
		data, err := FS.ReadFile(shemaRelPath)
		if err != nil {
			openApiDocErr = fmt.Errorf("read embedded tar.gz: %w", err)
			return
		}

		gzr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			openApiDocErr = fmt.Errorf("gzip reader: %w", err)
			return
		}
		defer gzr.Close()

		tr := tar.NewReader(gzr)

		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				openApiDocErr = fmt.Errorf("api.json not found in embedded archive")
				return
			}
			if err != nil {
				openApiDocErr = fmt.Errorf("tar read error: %w", err)
				return
			}

			if strings.HasSuffix(hdr.Name, "api.json") {
				var buf bytes.Buffer
				if _, err := io.Copy(&buf, tr); err != nil {
					openApiDocErr = fmt.Errorf("copy api.json from tar: %w", err)
					return
				}

				loader := openapi3.NewLoader()
				openApiDoc, openApiDocErr = loader.LoadFromData(buf.Bytes())
				return
			}
		}
	})

	return openApiDoc, openApiDocErr
}

func GetOpenApiResource(resourcePath string) (*openapi3.PathItem, error) {
	// Accept both forms: with and without trailing slash
	base := "/" + strings.Trim(resourcePath, "/")
	withSlash := base + "/"

	doc, err := loadOpenAPIDocOnce()
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI document: %w", err)
	}

	paths := doc.Paths.Map()
	if item := paths[withSlash]; item != nil {
		return item, nil
	}
	if item := paths[base]; item != nil {
		return item, nil
	}

	// Collect all available paths for diagnostics
	var available []string
	for path := range paths {
		available = append(available, path)
	}
	return nil, fmt.Errorf(
		"path %q not found in OpenAPI schema. Available paths:\n  - %s",
		resourcePath,
		strings.Join(available, "\n  - "),
	)
}

func GetOpenApiComponents() (*openapi3.Components, error) {
	doc, err := loadOpenAPIDocOnce()

	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI document: %w", err)
	}

	if doc.Components == nil {
		return nil, fmt.Errorf("OpenAPI document has no components defined")
	}

	return doc.Components, nil
}

func GetOpenApiComponentSchema(ref string) (*openapi3.SchemaRef, error) {
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		ref = parts[len(parts)-1]
	} else {
		panic("invalid schema reference: " + ref)
	}
	components, err := GetOpenApiComponents()
	if err != nil {
		return nil, fmt.Errorf("failed to get OpenAPI components: %w", err)
	}
	schemaRef := components.Schemas[ref]
	return schemaRef, nil
}

// GetSchema_POST_RequestBody extracts the request body schema from a POST operation.
// It expects the request body to be defined with content type "application/json".
// Returns the schema reference for the POST body payload.
// Returns an error if the POST operation or its schema is not properly defined.
func GetSchema_POST_RequestBody(resourcePath string) (*openapi3.SchemaRef, error) {
	resource, err := GetOpenApiResource(resourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get OpenAPI resource %q: %w", resourcePath, err)
	}

	if resource == nil || resource.Post == nil || resource.Post.RequestBody == nil ||
		resource.Post.RequestBody.Value == nil {
		return &openapi3.SchemaRef{Value: &openapi3.Schema{}}, nil
	}

	// Try application/json, then fallback to */*
	content := resource.Post.RequestBody.Value.Content["application/json"]
	if content == nil {
		content = resource.Post.RequestBody.Value.Content["*/*"]
	}
	if content == nil || content.Schema == nil {
		return &openapi3.SchemaRef{Value: &openapi3.Schema{}}, nil
	}

	// Resolve and compose if necessary
	final := resolveComposedSchema(resolveAllRefs(content.Schema))
	return &openapi3.SchemaRef{Value: final}, nil
}

// GetSchema_PATCH_RequestBody extracts the request body schema from a PATCH operation.
// Returns an empty schema if PATCH or application/json content is missing.
func GetSchema_PATCH_RequestBody(resourcePath string) (*openapi3.SchemaRef, error) {
	resource, err := GetOpenApiResource(resourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get OpenAPI resource %q: %w", resourcePath, err)
	}

	if resource == nil || resource.Patch == nil || resource.Patch.RequestBody == nil ||
		resource.Patch.RequestBody.Value == nil {
		return &openapi3.SchemaRef{Value: &openapi3.Schema{}}, nil
	}

	content := resource.Patch.RequestBody.Value.Content["application/json"]
	if content == nil {
		content = resource.Patch.RequestBody.Value.Content["*/*"]
	}
	if content == nil || content.Schema == nil {
		return &openapi3.SchemaRef{Value: &openapi3.Schema{}}, nil
	}

	final := resolveComposedSchema(resolveAllRefs(content.Schema))
	return &openapi3.SchemaRef{Value: final}, nil
}

// GetSchema_POST_StatusOk extracts the schema from a POST operation's response,
// checking status codes 200, 201, 202 (in that order of preference).
// It returns the schema if available under "application/json" content type.
func GetSchema_POST_StatusOk(resourcePath string) (*openapi3.SchemaRef, error) {
	resource, err := GetOpenApiResource(resourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get OpenAPI resource %q: %w", resourcePath, err)
	}

	if resource == nil || resource.Post == nil {
		return &openapi3.SchemaRef{Value: &openapi3.Schema{}}, nil
	}

	for _, code := range []int{200, 201, 202} {
		resp := resource.Post.Responses.Status(code)
		schemaRef := extractSchemaFromResponse(resp)
		if schemaRef != nil {
			final := resolveComposedSchema(resolveAllRefs(schemaRef))
			return &openapi3.SchemaRef{Value: final}, nil
		}
	}

	return nil, fmt.Errorf(
		"no valid schema found in POST response (200/201/202) for resource %s", resourcePath,
	)
}

// GetSchema_GET_StatusOk attempts to extract the item schema from a GET 200 response.
// It supports paginated (results[]), flat list ([]), and single-object responses.
func GetSchema_GET_StatusOk(resourcePath string) (*openapi3.SchemaRef, error) {
	resource, err := GetOpenApiResource(resourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get OpenAPI resource %q: %w", resourcePath, err)
	}

	if resource == nil || resource.Get == nil {
		return &openapi3.SchemaRef{Value: &openapi3.Schema{}}, nil
	}

	resp := resource.Get.Responses.Status(200)
	if resp == nil || resp.Value == nil {
		return nil, fmt.Errorf("GET missing 200 response for resource %s", resourcePath)
	}

	content := resp.Value.Content["application/json"]
	if content == nil || content.Schema == nil {
		return nil, fmt.Errorf("GET response missing or malformed schema")
	}

	rootSchema := resolveComposedSchema(resolveAllRefs(content.Schema))

	// Case 1: paginated response { "results": [...] }
	if results, ok := rootSchema.Properties["results"]; ok && results != nil {
		resolvedResults := resolveComposedSchema(resolveAllRefs(results))
		if resolvedResults.Type != nil && len(*resolvedResults.Type) > 0 && (*resolvedResults.Type)[0] == "array" {
			if resolvedResults.Items != nil {
				item := resolveComposedSchema(resolveAllRefs(resolvedResults.Items))
				return &openapi3.SchemaRef{Value: item}, nil
			}
			return nil, fmt.Errorf("GET response 'results' array has no items schema")
		}
		return nil, fmt.Errorf("GET response 'results' is not an array")
	}

	// Case 2: root is array itself
	if rootSchema.Type != nil && len(*rootSchema.Type) > 0 && (*rootSchema.Type)[0] == "array" {
		if rootSchema.Items != nil {
			item := resolveComposedSchema(resolveAllRefs(rootSchema.Items))
			return &openapi3.SchemaRef{Value: item}, nil
		}
		return nil, fmt.Errorf("GET root array has no items schema")
	}

	// Case 3: single object
	return &openapi3.SchemaRef{Value: rootSchema}, nil
}

// GetSchema_FromComponents retrieves a schema from the OpenAPI components section
// based on the provided resource path. It extracts the last part of the path as the component
func GetSchema_FromComponents(resourcePath string) (*openapi3.SchemaRef, error) {
	parts := strings.Split(resourcePath, "/")
	component := parts[len(parts)-1]

	doc, err := loadOpenAPIDocOnce()

	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI document: %w", err)
	}

	content, ok := doc.Components.Schemas[component]
	if !ok {
		return nil, fmt.Errorf("component schema %q not found in OpenAPI document", component)
	}

	final := resolveComposedSchema(resolveAllRefs(content))
	return &openapi3.SchemaRef{Value: final}, nil
}

// QueryParametersGET extracts all query parameters from the GET operation of a given OpenAPI path item.
// It returns a slice of openapi3.Parameter objects whose "in" field is "query".
// These typically represent optional or required query string inputs accepted by the endpoint.
//
// Parameters:
//   - resource: an *openapi3.PathItem representing a specific OpenAPI path (e.g., "/users/").
//
// Returns:
//   - []*openapi3.Parameter: a slice of query parameter definitions.
//   - error: if the GET operation is not defined.
func QueryParametersGET(resourcePath string) ([]*openapi3.Parameter, error) {
	resource, err := GetOpenApiResource(resourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get OpenAPI resource %q: %w", resourcePath, err)
	}

	if resource == nil || resource.Get == nil {
		// No GET operation — treat as no query parameters
		return []*openapi3.Parameter{}, nil
	}

	queryParams := make([]*openapi3.Parameter, 0)
	for _, paramRef := range resource.Get.Parameters {
		if paramRef == nil || paramRef.Value == nil {
			continue
		}
		if strings.EqualFold(paramRef.Value.In, "query") {
			queryParams = append(queryParams, paramRef.Value)
		}
	}

	return queryParams, nil
}

// GetSchema_GET_QueryParams converts query parameters from a GET operation into a SchemaRef.
// It creates an object schema where each query parameter becomes a property.
// The schema includes parameter names, types, descriptions, and required status.
//
// Parameters:
//   - resourcePath: the OpenAPI path to extract query parameters from.
//
// Returns:
//   - *openapi3.SchemaRef: a schema representing all query parameters as an object.
//   - error: if the resource cannot be found or parameters cannot be processed.
func GetSchema_GET_QueryParams(resourcePath string) (*openapi3.SchemaRef, error) {
	queryParams, err := QueryParametersGET(resourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get query parameters for %q: %w", resourcePath, err)
	}

	// Create a schema representing query parameters as object properties
	schema := &openapi3.Schema{
		Type:       &openapi3.Types{openapi3.TypeObject},
		Properties: make(map[string]*openapi3.SchemaRef),
		Required:   []string{},
	}

	for _, param := range queryParams {
		if param == nil || param.Schema == nil || param.Schema.Value == nil {
			continue
		}

		// Convert parameter schema to property schema
		propSchema := param.Schema.Value
		schema.Properties[param.Name] = &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type:        propSchema.Type,
				Description: param.Description,
				Default:     propSchema.Default,
				Enum:        propSchema.Enum,
				Format:      propSchema.Format,
				Min:         propSchema.Min,
				Max:         propSchema.Max,
				ReadOnly:    propSchema.ReadOnly,
			},
		}

		// Add to required fields if parameter is required
		if param.Required {
			schema.Required = append(schema.Required, param.Name)
		}
	}

	return &openapi3.SchemaRef{Value: schema}, nil
}

// extractSchemaFromResponse attempts to extract an application/json schema from a response.
func extractSchemaFromResponse(resp *openapi3.ResponseRef) *openapi3.SchemaRef {
	if resp == nil || resp.Value == nil {
		return nil
	}
	content := resp.Value.Content["application/json"]
	if content == nil || content.Schema == nil {
		return nil
	}
	return content.Schema
}

// SearchableQueryParams returns only query parameters that are primitive types
// (string, integer) from the GET operation of the given resource path.
func SearchableQueryParams(resourcePath string) ([]string, error) {
	params, err := QueryParametersGET(resourcePath)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, p := range params {
		if p == nil || p.Schema == nil || p.Schema.Value == nil {
			continue
		}
		schema := p.Schema.Value

		// Skip primitive or read-only fields
		if !isStringOrInteger(schema) || schema.ReadOnly {
			continue
		}

		result = append(result, p.Name)
	}

	return result, nil
}

// isStringOrInteger returns true if the given OpenAPI schema represents string or integer
func isStringOrInteger(prop *openapi3.Schema) bool {
	if prop == nil || prop.Type == nil || len(*prop.Type) == 0 {
		return false
	}
	switch (*prop.Type)[0] {
	case openapi3.TypeString, openapi3.TypeInteger:
		return true
	default:
		return false
	}
}

func resolveComposedSchema(schema *openapi3.Schema) *openapi3.Schema {
	if schema == nil {
		return nil
	}
	// Resolve allOf first, regardless of whether Type is set on the current schema.
	if len(schema.AllOf) > 0 {
		merged := &openapi3.Schema{
			Properties:   map[string]*openapi3.SchemaRef{},
			Required:     []string{},
			Title:        schema.Title,
			Description:  schema.Description,
			ExternalDocs: schema.ExternalDocs,
		}

		// First, copy properties from the original schema itself
		for name, prop := range schema.Properties {
			merged.Properties[name] = prop
		}
		merged.Required = append(merged.Required, schema.Required...)
		if schema.Type != nil && len(*schema.Type) > 0 {
			merged.Type = schema.Type
		}

		// Then, merge properties from allOf sub-schemas
		for _, subRef := range schema.AllOf {
			// Resolve refs and also compose nested allOf/anyOf/oneOf
			sub := resolveComposedSchema(resolveAllRefs(subRef))
			if sub == nil {
				continue
			}
			for name, prop := range sub.Properties {
				merged.Properties[name] = prop
			}
			merged.Required = append(merged.Required, sub.Required...)
			if sub.Type != nil && len(*sub.Type) > 0 {
				merged.Type = sub.Type
			}
		}
		return merged
	}

	// If there is no composition to resolve, return as-is.
	if schema.Type != nil && len(*schema.Type) > 0 {
		return schema
	}

	// Resolve oneOf or anyOf by picking the first resolvable schema with a type
	for _, refList := range [][]*openapi3.SchemaRef{schema.OneOf, schema.AnyOf} {
		for _, subRef := range refList {
			sub := resolveAllRefs(subRef)
			if sub != nil && sub.Type != nil && len(*sub.Type) > 0 {
				return sub
			}
		}
	}
	return schema
}

func resolveAllRefs(ref *openapi3.SchemaRef) *openapi3.Schema {
	seen := map[string]bool{}
	for ref != nil && ref.Ref != "" && !seen[ref.Ref] {
		seen[ref.Ref] = true
		ref, _ = GetOpenApiComponentSchema(ref.Ref)
	}
	if ref == nil || ref.Value == nil {
		panic(fmt.Sprintf("cannot resolve final schema from ref: %+v", ref))
	}
	return ref.Value
}
