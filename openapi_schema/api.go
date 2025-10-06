package openapi_schema

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
)

var (
	//go:embed api.tar.gz
	FS             embed.FS
	openApiDocOnce sync.Once
	openApiDoc     *openapi3.T
	openApiDocErr  error
	shemaRelPath   = "api.tar.gz"
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

// GetSchemaFromComponent retrieves a schema by component name (e.g., "ActiveDirectory")
// Returns the RESOLVED schema (after resolving refs and compositions)
func GetSchemaFromComponent(componentName string) (*openapi3.SchemaRef, error) {
	doc, err := loadOpenAPIDocOnce()
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI document: %w", err)
	}

	content, ok := doc.Components.Schemas[componentName]
	if !ok {
		return nil, fmt.Errorf("component schema %q not found in OpenAPI document", componentName)
	}

	final := resolveComposedSchema(resolveAllRefs(content))
	return &openapi3.SchemaRef{Value: final}, nil
}

// ComponentSchema represents a component schema with its name and reference
type ComponentSchema struct {
	Name      string // e.g., "ActiveDirectory"
	Reference string // e.g., "#/components/schemas/ActiveDirectory"
	Schema    *openapi3.Schema
}

// GetAllComponentSchemas retrieves all schemas from the OpenAPI components section
func GetAllComponentSchemas() ([]ComponentSchema, error) {
	doc, err := loadOpenAPIDocOnce()
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI document: %w", err)
	}

	var components []ComponentSchema
	for name, schemaRef := range doc.Components.Schemas {
		if schemaRef == nil || schemaRef.Value == nil {
			continue
		}

		// Resolve all refs and compositions
		resolved := resolveComposedSchema(resolveAllRefs(schemaRef))

		components = append(components, ComponentSchema{
			Name:      name,
			Reference: fmt.Sprintf("#/components/schemas/%s", name),
			Schema:    resolved,
		})
	}

	// Sort by name for consistent output
	sort.Slice(components, func(i, j int) bool {
		return components[i].Name < components[j].Name
	})

	return components, nil
}

// IsDirectComponentReference checks if a SchemaRef is a direct $ref to a component
// (not inline, not composed with allOf/oneOf/anyOf)
// Returns the component name if it's a direct reference, empty string otherwise
func IsDirectComponentReference(schemaRef *openapi3.SchemaRef) string {
	if schemaRef == nil {
		return ""
	}

	// Check if it has a $ref
	if schemaRef.Ref == "" {
		return ""
	}

	// Check if it's a component reference (#/components/schemas/X or #/definitions/X)
	ref := schemaRef.Ref
	if strings.HasPrefix(ref, "#/components/schemas/") {
		componentName := strings.TrimPrefix(ref, "#/components/schemas/")
		return componentName
	}
	if strings.HasPrefix(ref, "#/definitions/") {
		componentName := strings.TrimPrefix(ref, "#/definitions/")
		return componentName
	}

	return ""
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

	result := make([]string, 0)
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

// ######################################################
// UNIFIED API METHODS
// ######################################################

// GetRequestBodySchema extracts the request body schema for a given HTTP method and resource path.
// It supports POST, PATCH, PUT, and DELETE methods.
//
// Parameters:
//   - httpMethod: HTTP method (e.g., "POST", "PATCH", "PUT", "DELETE")
//   - resourcePath: The API resource path (e.g., "apitokens")
//
// Returns:
//   - *openapi3.SchemaRef: The request body schema, or an empty schema if not found
//   - error: If the resource cannot be loaded or the method is not supported
//
// Example:
//
//	schema, err := GetRequestBodySchema("POST", "apitokens")
func GetRequestBodySchema(httpMethod, resourcePath string) (*openapi3.SchemaRef, error) {
	resource, err := GetOpenApiResource(resourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get OpenAPI resource %q: %w", resourcePath, err)
	}

	if resource == nil {
		return &openapi3.SchemaRef{Value: &openapi3.Schema{}}, nil
	}

	// Get the operation based on HTTP method
	var operation *openapi3.Operation
	switch httpMethod {
	case "POST":
		operation = resource.Post
	case "PATCH":
		operation = resource.Patch
	case "PUT":
		operation = resource.Put
	case "DELETE":
		operation = resource.Delete
	default:
		return nil, fmt.Errorf("unsupported HTTP method for request body: %s (use POST, PATCH, PUT, or DELETE)", httpMethod)
	}

	if operation == nil || operation.RequestBody == nil || operation.RequestBody.Value == nil {
		return &openapi3.SchemaRef{Value: &openapi3.Schema{}}, nil
	}

	// Try application/json, then fallback to */*
	content := operation.RequestBody.Value.Content["application/json"]
	if content == nil {
		content = operation.RequestBody.Value.Content["*/*"]
	}
	if content == nil || content.Schema == nil {
		return &openapi3.SchemaRef{Value: &openapi3.Schema{}}, nil
	}

	// Resolve and compose if necessary
	final := resolveComposedSchema(resolveAllRefs(content.Schema))
	return &openapi3.SchemaRef{Value: final}, nil
}

// GetResponseModelSchemaUnresolved extracts the RAW response model schema (BEFORE resolving $refs)
// This is useful for detecting if the schema is a direct component reference
func GetResponseModelSchemaUnresolved(httpMethod, resourcePath string) (*openapi3.SchemaRef, error) {
	resource, err := GetOpenApiResource(resourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get OpenAPI resource %q: %w", resourcePath, err)
	}

	if resource == nil {
		return nil, nil
	}

	// Get the operation based on HTTP method
	var operation *openapi3.Operation
	switch httpMethod {
	case "GET":
		operation = resource.Get
	case "POST":
		operation = resource.Post
	case "PATCH":
		operation = resource.Patch
	case "PUT":
		operation = resource.Put
	case "DELETE":
		operation = resource.Delete
	default:
		return nil, fmt.Errorf("unsupported HTTP method for response: %s", httpMethod)
	}

	if operation == nil {
		return nil, nil
	}

	// For GET, extract from 200 response
	if httpMethod == "GET" {
		resp := operation.Responses.Status(200)
		if resp == nil || resp.Value == nil {
			return nil, nil
		}
		content := resp.Value.Content["application/json"]
		if content == nil || content.Schema == nil {
			return nil, nil
		}
		// Return unresolved schema - check for paginated first
		rootSchema := content.Schema
		if rootSchema.Value != nil && rootSchema.Value.Properties != nil {
			if resultsRef, ok := rootSchema.Value.Properties["results"]; ok {
				// Paginated response - return the items schema
				if resultsRef.Value != nil && resultsRef.Value.Items != nil {
					return resultsRef.Value.Items, nil
				}
			}
		}
		// Check if it's a direct array
		if rootSchema.Value != nil && rootSchema.Value.Items != nil {
			return rootSchema.Value.Items, nil
		}
		return rootSchema, nil
	}

	// For non-GET methods, check status codes 200, 201, 202
	for _, code := range []int{200, 201, 202} {
		resp := operation.Responses.Status(code)
		schemaRef := extractSchemaFromResponse(resp)
		if schemaRef != nil {
			// Return UNRESOLVED schema (before resolveAllRefs)
			return schemaRef, nil
		}
	}

	return nil, nil
}

// GetResponseModelSchema extracts the response model schema for a given HTTP method and resource path.
// It checks for successful status codes (200, 201, 202) and returns the schema from the response body.
//
// Parameters:
//   - httpMethod: HTTP method (e.g., "GET", "POST", "PATCH", "PUT", "DELETE")
//   - resourcePath: The API resource path (e.g., "apitokens")
//
// Returns:
//   - *openapi3.SchemaRef: The response model schema
//   - error: If the resource cannot be loaded, the method is not supported, or no valid schema is found
//
// Example:
//
//	schema, err := GetResponseModelSchema("GET", "apitokens")
//
// Notes:
//   - For GET requests, it automatically handles paginated responses, arrays, and single objects
//   - For other methods, it checks status codes 200, 201, 202 in order
func GetResponseModelSchema(httpMethod, resourcePath string) (*openapi3.SchemaRef, error) {
	resource, err := GetOpenApiResource(resourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get OpenAPI resource %q: %w", resourcePath, err)
	}

	if resource == nil {
		return &openapi3.SchemaRef{Value: &openapi3.Schema{}}, nil
	}

	// Get the operation based on HTTP method
	var operation *openapi3.Operation
	switch httpMethod {
	case "GET":
		operation = resource.Get
	case "POST":
		operation = resource.Post
	case "PATCH":
		operation = resource.Patch
	case "PUT":
		operation = resource.Put
	case "DELETE":
		operation = resource.Delete
	default:
		return nil, fmt.Errorf("unsupported HTTP method for response: %s", httpMethod)
	}

	if operation == nil {
		return &openapi3.SchemaRef{Value: &openapi3.Schema{}}, nil
	}

	// Special handling for GET to support paginated/array responses
	if httpMethod == "GET" {
		return getResponseModelSchemaForGET(resource, resourcePath)
	}

	// For non-GET methods, check status codes 200, 201, 202
	for _, code := range []int{200, 201, 202} {
		resp := operation.Responses.Status(code)
		schemaRef := extractSchemaFromResponse(resp)
		if schemaRef != nil {
			final := resolveComposedSchema(resolveAllRefs(schemaRef))
			return &openapi3.SchemaRef{Value: final}, nil
		}
	}

	return nil, fmt.Errorf(
		"no valid schema found in %s response (200/201/202) for resource %s",
		httpMethod, resourcePath,
	)
}

// getResponseModelSchemaForGET handles GET-specific logic for extracting response schemas.
// It supports paginated (results[]), flat list ([]), and single-object responses.
func getResponseModelSchemaForGET(resource *openapi3.PathItem, resourcePath string) (*openapi3.SchemaRef, error) {
	if resource.Get == nil {
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

	// 1. Check if response is paginated with "results" field
	if rootSchema.Type != nil && (*rootSchema.Type).Is("object") && rootSchema.Properties != nil {
		if resultsRef, ok := rootSchema.Properties["results"]; ok {
			resultsSchema := resolveComposedSchema(resolveAllRefs(resultsRef))
			if resultsSchema.Type != nil && (*resultsSchema.Type).Is("array") && resultsSchema.Items != nil {
				itemSchema := resolveComposedSchema(resolveAllRefs(resultsSchema.Items))
				return &openapi3.SchemaRef{Value: itemSchema}, nil
			}
		}
	}

	// 2. Check if response is a flat array
	if rootSchema.Type != nil && (*rootSchema.Type).Is("array") && rootSchema.Items != nil {
		itemSchema := resolveComposedSchema(resolveAllRefs(rootSchema.Items))
		return &openapi3.SchemaRef{Value: itemSchema}, nil
	}

	// 3. Single object response
	if rootSchema.Type != nil && (*rootSchema.Type).Is("object") {
		return &openapi3.SchemaRef{Value: rootSchema}, nil
	}

	return nil, fmt.Errorf("unsupported GET response schema structure for resource %s", resourcePath)
}

// GetOperationSummary returns the summary description for a specific HTTP method and resource path
// from the OpenAPI specification.
//
// Parameters:
//   - httpMethod: HTTP method (GET, POST, PUT, PATCH, DELETE, etc.)
//   - resourcePath: API path (e.g., "/users/{id}/access_keys/")
//
// Returns:
//   - string: The operation summary, or empty string if not found
//   - error: if the OpenAPI document cannot be loaded or path is not found
//
// DeleteParams holds the DELETE operation parameters
type DeleteParams struct {
	QueryParams   []*openapi3.ParameterRef // Query parameters (excluding id in path)
	BodySchema    *openapi3.SchemaRef      // Body schema if present
	IdDescription string                   // Description of the id path parameter
}

func GetOperationSummary(httpMethod, resourcePath string) (string, error) {
	doc, err := loadOpenAPIDocOnce()
	if err != nil {
		return "", fmt.Errorf("failed to load OpenAPI document: %w", err)
	}

	// Normalize the path
	normalizedPath := "/" + strings.Trim(resourcePath, "/")
	if !strings.HasSuffix(normalizedPath, "/") {
		normalizedPath += "/"
	}

	// Get the path item
	pathItem := doc.Paths.Find(normalizedPath)
	if pathItem == nil {
		// Try without trailing slash
		normalizedPath = strings.TrimSuffix(normalizedPath, "/")
		pathItem = doc.Paths.Find(normalizedPath)
		if pathItem == nil {
			return "", fmt.Errorf("path not found: %s", resourcePath)
		}
	}

	// Get the operation for the specified method
	var operation *openapi3.Operation
	switch strings.ToUpper(httpMethod) {
	case "GET":
		operation = pathItem.Get
	case "POST":
		operation = pathItem.Post
	case "PUT":
		operation = pathItem.Put
	case "PATCH":
		operation = pathItem.Patch
	case "DELETE":
		operation = pathItem.Delete
	case "HEAD":
		operation = pathItem.Head
	case "OPTIONS":
		operation = pathItem.Options
	default:
		return "", fmt.Errorf("unsupported HTTP method: %s", httpMethod)
	}

	if operation == nil {
		return "", fmt.Errorf("operation not found for %s %s", httpMethod, resourcePath)
	}

	return operation.Summary, nil
}

// ValidateOperationExists checks if a specific HTTP method exists for a given path in the OpenAPI spec
// Returns an error if the path or method doesn't exist
func ValidateOperationExists(httpMethod, resourcePath string) error {
	doc, err := loadOpenAPIDocOnce()
	if err != nil {
		return fmt.Errorf("failed to load OpenAPI document: %w", err)
	}

	// Normalize the path - trim spaces and slashes
	normalizedPath := "/" + strings.Trim(strings.TrimSpace(resourcePath), "/")
	pathWithSlash := normalizedPath + "/"
	pathWithoutSlash := normalizedPath

	// Get all paths and try to find a match
	paths := doc.Paths.Map()
	var pathItem *openapi3.PathItem

	// Try exact matches first (case-sensitive)
	if item := paths[pathWithSlash]; item != nil {
		pathItem = item
	} else if item := paths[pathWithoutSlash]; item != nil {
		pathItem = item
	}

	// If not found, try case-insensitive matching
	if pathItem == nil {
		normalizedLower := strings.ToLower(normalizedPath)
		for path, item := range paths {
			pathLower := strings.ToLower(path)
			if pathLower == normalizedLower || pathLower == normalizedLower+"/" || pathLower+"/" == normalizedLower {
				pathItem = item
				fmt.Printf("  ℹ️  Note: Found path with different casing: %s (you specified: %s)\n", path, resourcePath)
				break
			}
		}
	}

	if pathItem == nil {
		// Collect all available paths for debugging
		var availablePaths []string
		for path := range paths {
			availablePaths = append(availablePaths, path)
		}
		sort.Strings(availablePaths)

		// Extract a keyword from the path for debugging (e.g., "rpc" from "/clusters/{id}/rpc/")
		pathParts := strings.Split(strings.Trim(strings.TrimSpace(resourcePath), "/"), "/")
		keyword := ""
		for _, part := range pathParts {
			if part != "" && !strings.HasPrefix(part, "{") {
				keyword = part
				break
			}
		}

		errorMsg := fmt.Sprintf("path not found in OpenAPI spec: %s (tried both %s and %s)",
			resourcePath, pathWithSlash, pathWithoutSlash)

		if keyword != "" {
			// Try case-insensitive search for similar paths
			similarPaths := filterPathsCaseInsensitive(availablePaths, keyword)
			errorMsg += fmt.Sprintf("\nPaths containing '%s': %v", keyword, similarPaths)

			// If no matches, show first 10 paths to help debug
			if len(similarPaths) == 1 && similarPaths[0] == "none" && len(availablePaths) > 0 {
				limit := 10
				if len(availablePaths) < limit {
					limit = len(availablePaths)
				}
				errorMsg += fmt.Sprintf("\nFirst %d paths in spec: %v", limit, availablePaths[:limit])
			}
		}

		return fmt.Errorf("%s", errorMsg)
	}

	// Get the operation for the specified method
	var operation *openapi3.Operation
	switch strings.ToUpper(httpMethod) {
	case "GET":
		operation = pathItem.Get
	case "POST":
		operation = pathItem.Post
	case "PUT":
		operation = pathItem.Put
	case "PATCH":
		operation = pathItem.Patch
	case "DELETE":
		operation = pathItem.Delete
	case "HEAD":
		operation = pathItem.Head
	case "OPTIONS":
		operation = pathItem.Options
	default:
		return fmt.Errorf("unsupported HTTP method: %s", httpMethod)
	}

	if operation == nil {
		// Collect available methods for this path to help debugging
		var availableMethods []string
		if pathItem.Get != nil {
			availableMethods = append(availableMethods, "GET")
		}
		if pathItem.Post != nil {
			availableMethods = append(availableMethods, "POST")
		}
		if pathItem.Put != nil {
			availableMethods = append(availableMethods, "PUT")
		}
		if pathItem.Patch != nil {
			availableMethods = append(availableMethods, "PATCH")
		}
		if pathItem.Delete != nil {
			availableMethods = append(availableMethods, "DELETE")
		}
		if pathItem.Head != nil {
			availableMethods = append(availableMethods, "HEAD")
		}
		if pathItem.Options != nil {
			availableMethods = append(availableMethods, "OPTIONS")
		}

		return fmt.Errorf("method %s not found for path %s (available methods: %v)", httpMethod, resourcePath, availableMethods)
	}

	return nil
}

// filterPaths returns paths that contain the given substring (helper for debugging)
func filterPaths(paths []string, substring string) []string {
	var filtered []string
	for _, path := range paths {
		if strings.Contains(path, substring) {
			filtered = append(filtered, path)
		}
	}
	if len(filtered) == 0 {
		return []string{"none"}
	}
	return filtered
}

// filterPathsCaseInsensitive returns paths that contain the given substring (case-insensitive)
func filterPathsCaseInsensitive(paths []string, substring string) []string {
	var filtered []string
	substringLower := strings.ToLower(substring)
	for _, path := range paths {
		if strings.Contains(strings.ToLower(path), substringLower) {
			filtered = append(filtered, path)
		}
	}
	if len(filtered) == 0 {
		return []string{"none"}
	}
	return filtered
}

// GetDeleteParams extracts DELETE operation parameters (query params and body schema)
func GetDeleteParams(resourcePath string) (*DeleteParams, error) {
	doc, err := loadOpenAPIDocOnce()
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI document: %w", err)
	}

	// Normalize the path - DELETE operations are typically on /{resource}/{id}/
	normalizedPath := "/" + strings.Trim(resourcePath, "/") + "/{id}/"

	// Get the path item
	pathItem := doc.Paths.Find(normalizedPath)
	if pathItem == nil {
		// Try without trailing slash
		normalizedPath = strings.TrimSuffix(normalizedPath, "/")
		pathItem = doc.Paths.Find(normalizedPath)
		if pathItem == nil {
			return nil, fmt.Errorf("path not found: %s", normalizedPath)
		}
	}

	if pathItem.Delete == nil {
		return nil, fmt.Errorf("DELETE operation not found for path: %s", normalizedPath)
	}

	params := &DeleteParams{
		QueryParams: []*openapi3.ParameterRef{},
	}

	// Extract parameters
	for _, paramRef := range pathItem.Delete.Parameters {
		if paramRef.Value != nil {
			if paramRef.Value.In == "query" {
				// Query parameters
				params.QueryParams = append(params.QueryParams, paramRef)
			} else if paramRef.Value.In == "path" && paramRef.Value.Name == "id" {
				// Path parameter 'id' - extract description
				params.IdDescription = paramRef.Value.Description
			}
		}
	}

	// Extract body schema
	if pathItem.Delete.RequestBody != nil && pathItem.Delete.RequestBody.Value != nil {
		content := pathItem.Delete.RequestBody.Value.Content
		if content != nil {
			if jsonContent := content.Get("application/json"); jsonContent != nil {
				params.BodySchema = jsonContent.Schema
			}
		}
	}

	return params, nil
}

// ReturnsTextPlain checks if an operation returns text/plain content type
//
// Parameters:
//   - httpMethod: HTTP method (GET, POST, PUT, PATCH, DELETE, etc.)
//   - resourcePath: API path (e.g., "/prometheusmetrics/")
//
// Returns:
//   - bool: true if the operation returns text/plain, false otherwise
//   - error: if the OpenAPI document cannot be loaded or path/operation is not found
func ReturnsTextPlain(httpMethod, resourcePath string) (bool, error) {
	doc, err := loadOpenAPIDocOnce()
	if err != nil {
		return false, fmt.Errorf("failed to load OpenAPI document: %w", err)
	}

	// Normalize the path
	normalizedPath := "/" + strings.Trim(resourcePath, "/")
	if !strings.HasSuffix(normalizedPath, "/") {
		normalizedPath += "/"
	}

	// Get the path item
	pathItem := doc.Paths.Find(normalizedPath)
	if pathItem == nil {
		// Try without trailing slash
		normalizedPath = strings.TrimSuffix(normalizedPath, "/")
		pathItem = doc.Paths.Find(normalizedPath)
		if pathItem == nil {
			return false, fmt.Errorf("path not found: %s", resourcePath)
		}
	}

	// Get the operation for the specified method
	var operation *openapi3.Operation
	switch strings.ToUpper(httpMethod) {
	case "GET":
		operation = pathItem.Get
	case "POST":
		operation = pathItem.Post
	case "PUT":
		operation = pathItem.Put
	case "PATCH":
		operation = pathItem.Patch
	case "DELETE":
		operation = pathItem.Delete
	case "HEAD":
		operation = pathItem.Head
	case "OPTIONS":
		operation = pathItem.Options
	default:
		return false, fmt.Errorf("unsupported HTTP method: %s", httpMethod)
	}

	if operation == nil {
		return false, fmt.Errorf("operation not found for %s %s", httpMethod, resourcePath)
	}

	// Check if response contains text/plain in any 2xx status code
	if operation.Responses != nil {
		for code := 200; code < 300; code++ {
			if response := operation.Responses.Status(code); response != nil && response.Value != nil {
				// Skip 204 No Content responses
				if code == 204 {
					continue
				}

				if response.Value.Content != nil {
					// Check all content types in the response
					for contentType := range response.Value.Content {
						if strings.Contains(contentType, "text/plain") {
							return true, nil
						}
					}
				} else {
					// If there's no Content field at all, check if this is a known text/plain endpoint
					// This happens when OpenAPI v2 "produces: - text/plain" wasn't properly converted to v3
					// We use heuristics to detect these cases:
					// 1. Prometheus metrics endpoints (contain "prometheus" in path or description)
					if strings.Contains(strings.ToLower(resourcePath), "prometheus") ||
						(response.Value.Description != nil && strings.Contains(strings.ToLower(*response.Value.Description), "prometheus")) {
						return true, nil
					}
				}
			}
		}
	}

	// Also check if the default response has text/plain
	if operation.Responses != nil && operation.Responses.Default() != nil {
		if response := operation.Responses.Default(); response.Value != nil {
			if response.Value.Content != nil {
				for contentType := range response.Value.Content {
					if strings.Contains(contentType, "text/plain") {
						return true, nil
					}
				}
			} else {
				// Check for known text/plain endpoints using heuristics
				if strings.Contains(strings.ToLower(resourcePath), "prometheus") ||
					(response.Value.Description != nil && strings.Contains(strings.ToLower(*response.Value.Description), "prometheus")) {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// Returns204NoContent checks if an operation returns HTTP 204 No Content status
//
// Parameters:
//   - httpMethod: HTTP method (GET, POST, PUT, PATCH, DELETE, etc.)
//   - resourcePath: API path (e.g., "/users/{id}/access_keys/")
//
// Returns:
//   - bool: true if the operation returns 204 No Content, false otherwise
//   - error: if the OpenAPI document cannot be loaded or path/operation is not found
func Returns204NoContent(httpMethod, resourcePath string) (bool, error) {
	doc, err := loadOpenAPIDocOnce()
	if err != nil {
		return false, fmt.Errorf("failed to load OpenAPI document: %w", err)
	}

	// Normalize the path
	normalizedPath := "/" + strings.Trim(resourcePath, "/")
	if !strings.HasSuffix(normalizedPath, "/") {
		normalizedPath += "/"
	}

	// Get the path item
	pathItem := doc.Paths.Find(normalizedPath)
	if pathItem == nil {
		// Try without trailing slash
		normalizedPath = strings.TrimSuffix(normalizedPath, "/")
		pathItem = doc.Paths.Find(normalizedPath)
		if pathItem == nil {
			return false, fmt.Errorf("path not found: %s", resourcePath)
		}
	}

	// Get the operation for the specified method
	var operation *openapi3.Operation
	switch strings.ToUpper(httpMethod) {
	case "GET":
		operation = pathItem.Get
	case "POST":
		operation = pathItem.Post
	case "PUT":
		operation = pathItem.Put
	case "PATCH":
		operation = pathItem.Patch
	case "DELETE":
		operation = pathItem.Delete
	case "HEAD":
		operation = pathItem.Head
	case "OPTIONS":
		operation = pathItem.Options
	default:
		return false, fmt.Errorf("unsupported HTTP method: %s", httpMethod)
	}

	if operation == nil {
		return false, fmt.Errorf("operation not found for %s %s", httpMethod, resourcePath)
	}

	// Check if response contains 204 status code
	if operation.Responses != nil {
		if response := operation.Responses.Status(204); response != nil {
			return true, nil
		}
	}

	return false, nil
}
