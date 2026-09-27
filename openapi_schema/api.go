package openapi_schema

import (
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

var (
	//go:embed api.tar.gz dataengine.tar.gz
	FS embed.FS
)

// GetOpenApiResource returns the PathItem for resourcePath in the given pack.
// Accepts paths with or without a trailing slash.
func GetOpenApiResource(pack Pack, resourcePath string) (*openapi3.PathItem, error) {
	// Accept both forms: with and without trailing slash
	base := "/" + strings.Trim(resourcePath, "/")
	withSlash := base + "/"

	doc, err := loadOpenAPIDoc(pack)
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

// ResolveCollectionItemPath finds the single-resource path for a collection
// (e.g. /functions/{guid} or /users/{id}/). VMS uses {id}; DataEngine uses {guid}.
func ResolveCollectionItemPath(pack Pack, collectionPath string) (string, error) {
	base := "/" + strings.Trim(collectionPath, "/")
	candidates := []string{
		base + "/{id}",
		base + "/{id}/",
		base + "/{guid}",
		base + "/{guid}/",
	}
	for _, c := range candidates {
		if _, err := GetOpenApiResource(pack, c); err == nil {
			return strings.TrimSuffix(c, "/"), nil
		}
	}
	return "", fmt.Errorf("no item path ({id}|{guid}) for collection %q in pack %s", collectionPath, pack)
}

func GetOpenApiComponents(pack Pack) (*openapi3.Components, error) {
	doc, err := loadOpenAPIDoc(pack)

	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI document: %w", err)
	}

	if doc.Components == nil {
		return nil, fmt.Errorf("OpenAPI document has no components defined")
	}

	return doc.Components, nil
}

func GetOpenApiComponentSchema(pack Pack, ref string) (*openapi3.SchemaRef, error) {
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		ref = parts[len(parts)-1]
	} else {
		panic("invalid schema reference: " + ref)
	}
	components, err := GetOpenApiComponents(pack)
	if err != nil {
		return nil, fmt.Errorf("failed to get OpenAPI components: %w", err)
	}
	schemaRef := components.Schemas[ref]
	return schemaRef, nil
}

// GetSchema_FromComponents retrieves a schema from the OpenAPI components section
// based on the provided resource path. It extracts the last part of the path as the component
func GetSchema_FromComponents(pack Pack, resourcePath string) (*openapi3.SchemaRef, error) {
	parts := strings.Split(resourcePath, "/")
	component := parts[len(parts)-1]

	doc, err := loadOpenAPIDoc(pack)

	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI document: %w", err)
	}

	content, ok := doc.Components.Schemas[component]
	if !ok {
		return nil, fmt.Errorf("component schema %q not found in OpenAPI document", component)
	}

	final := ResolveComposedSchema(pack, ResolveAllRefs(pack, content))
	return &openapi3.SchemaRef{Value: final}, nil
}

// GetSchemaFromComponent retrieves a schema by component name (e.g., "ActiveDirectory")
// Returns the RESOLVED schema (after resolving refs and compositions)
func GetSchemaFromComponent(pack Pack, componentName string) (*openapi3.SchemaRef, error) {
	doc, err := loadOpenAPIDoc(pack)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI document: %w", err)
	}

	content, ok := doc.Components.Schemas[componentName]
	if !ok {
		return nil, fmt.Errorf("component schema %q not found in OpenAPI document", componentName)
	}

	final := ResolveComposedSchema(pack, ResolveAllRefs(pack, content))
	return &openapi3.SchemaRef{Value: final}, nil
}

// ComponentSchema represents a component schema with its name and reference
type ComponentSchema struct {
	Name      string // e.g., "ActiveDirectory"
	Reference string // e.g., "#/components/schemas/ActiveDirectory"
	Schema    *openapi3.Schema
}

// GetAllComponentSchemas retrieves all schemas from the OpenAPI components section
func GetAllComponentSchemas(pack Pack) ([]ComponentSchema, error) {
	doc, err := loadOpenAPIDoc(pack)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI document: %w", err)
	}

	var components []ComponentSchema
	for name, schemaRef := range doc.Components.Schemas {
		if schemaRef == nil || schemaRef.Value == nil {
			continue
		}

		// Resolve all refs and compositions
		resolved := ResolveComposedSchema(pack, ResolveAllRefs(pack, schemaRef))

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

// GetQueryParameters extracts all query parameters from the specified HTTP method operation of a given OpenAPI path item.
// It returns a slice of openapi3.Parameter objects whose "in" field is "query".
// These typically represent optional or required query string inputs accepted by the endpoint.
//
// Parameters:
//   - httpMethod: HTTP method (GET, POST, PATCH, PUT, DELETE, etc.)
//   - resourcePath: an OpenAPI path (e.g., "/users/")
//
// Returns:
//   - []*openapi3.Parameter: a slice of query parameter definitions.
//   - error: if the resource cannot be retrieved.
func GetQueryParameters(pack Pack, httpMethod, resourcePath string) ([]*openapi3.Parameter, error) {
	resource, err := GetOpenApiResource(pack, resourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get OpenAPI resource %q: %w", resourcePath, err)
	}

	if resource == nil {
		return []*openapi3.Parameter{}, nil
	}

	var operation *openapi3.Operation
	switch strings.ToUpper(httpMethod) {
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
	case "HEAD":
		operation = resource.Head
	case "OPTIONS":
		operation = resource.Options
	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", httpMethod)
	}

	if operation == nil {
		// No operation for this method — treat as no query parameters
		return []*openapi3.Parameter{}, nil
	}

	queryParams := make([]*openapi3.Parameter, 0)
	for _, paramRef := range operation.Parameters {
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
func GetSchema_GET_QueryParams(pack Pack, resourcePath string) (*openapi3.SchemaRef, error) {
	queryParams, err := GetQueryParameters(pack, "GET", resourcePath)
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
func SearchableQueryParams(pack Pack, resourcePath string) ([]string, error) {
	params, err := GetQueryParameters(pack, "GET", resourcePath)
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

// isStringOrInteger returns true if the given OpenAPI schema includes string or integer.
func isStringOrInteger(prop *openapi3.Schema) bool {
	return IsStringOrInteger(prop)
}

// ResolveComposedSchema resolves allOf/oneOf/anyOf compositions in an OpenAPI schema
func ResolveComposedSchema(pack Pack, schema *openapi3.Schema) *openapi3.Schema {
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
			sub := ResolveComposedSchema(pack, ResolveAllRefs(pack, subRef))
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

	// Resolve oneOf / anyOf by merging ALL branches (not picking the first).
	// Each branch is fully composed first so allOf object bodies get properties + type.
	// Types are unioned (including string), properties/required are merged.
	// Common DataEngine pattern: oneOf[PipelineCreateInfo(object), raw YAML/JSON string].
	for _, refList := range [][]*openapi3.SchemaRef{schema.OneOf, schema.AnyOf} {
		if len(refList) == 0 {
			continue
		}
		merged := &openapi3.Schema{
			Properties:   map[string]*openapi3.SchemaRef{},
			Required:     []string{},
			Title:        schema.Title,
			Description:  schema.Description,
			ExternalDocs: schema.ExternalDocs,
		}
		var typeSet openapi3.Types
		sawBranch := false
		for _, subRef := range refList {
			sub := ResolveComposedSchema(pack, ResolveAllRefs(pack, subRef))
			if sub == nil {
				continue
			}
			hasType := sub.Type != nil && len(*sub.Type) > 0
			hasProps := len(sub.Properties) > 0
			if !hasType && !hasProps && sub.Items == nil {
				continue
			}
			sawBranch = true
			for name, prop := range sub.Properties {
				merged.Properties[name] = prop
			}
			merged.Required = append(merged.Required, sub.Required...)
			if hasType {
				for _, t := range *sub.Type {
					typeSet = appendTypeUnique(typeSet, t)
				}
			} else if hasProps {
				// Composed object without an explicit type (e.g. allOf-only branch).
				typeSet = appendTypeUnique(typeSet, openapi3.TypeObject)
			}
			if sub.Items != nil && merged.Items == nil {
				merged.Items = sub.Items
			}
			if merged.Description == "" && sub.Description != "" {
				merged.Description = sub.Description
			}
		}
		if !sawBranch {
			continue
		}
		if len(typeSet) > 0 {
			merged.Type = &typeSet
		}
		return merged
	}
	return schema
}

func appendTypeUnique(types openapi3.Types, t string) openapi3.Types {
	for _, existing := range types {
		if existing == t {
			return types
		}
	}
	return append(types, t)
}

// ResolveAllRefs resolves all $ref references in an OpenAPI schema.
// Returns nil if the reference chain cannot be resolved.
func ResolveAllRefs(pack Pack, ref *openapi3.SchemaRef) *openapi3.Schema {
	seen := map[string]bool{}
	for ref != nil && ref.Ref != "" && !seen[ref.Ref] {
		seen[ref.Ref] = true
		next, err := GetOpenApiComponentSchema(pack, ref.Ref)
		if err != nil || next == nil {
			return nil
		}
		ref = next
	}
	if ref == nil || ref.Value == nil {
		return nil
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
//	schema, err := GetRequestBodySchema(PackVMS, "POST", "apitokens")
func GetRequestBodySchema(pack Pack, httpMethod, resourcePath string) (*openapi3.SchemaRef, error) {
	resource, err := GetOpenApiResource(pack, resourcePath)
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
	final := ResolveComposedSchema(pack, ResolveAllRefs(pack, content.Schema))
	return &openapi3.SchemaRef{Value: final}, nil
}

// GetResponseModelSchemaUnresolved extracts the RAW response model schema (BEFORE resolving $refs)
// This is useful for detecting if the schema is a direct component reference
func GetResponseModelSchemaUnresolved(pack Pack, httpMethod, resourcePath string) (*openapi3.SchemaRef, error) {
	resource, err := GetOpenApiResource(pack, resourcePath)
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
				// VMS DRF paginated response - return the items schema
				if resultsRef.Value != nil && resultsRef.Value.Items != nil {
					return resultsRef.Value.Items, nil
				}
			}
			if dataRef, ok := rootSchema.Value.Properties["data"]; ok {
				// DataEngine cursor-paginated response - return the items schema
				if dataRef.Value != nil && dataRef.Value.Items != nil {
					return dataRef.Value.Items, nil
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
// It checks for successful status codes (200, 201, 202, 204) and returns the schema from the response body.
//
// Parameters:
//   - httpMethod: HTTP method (e.g., "GET", "POST", "PATCH", "PUT", "DELETE")
//   - resourcePath: The API resource path (e.g., "apitokens")
//
// Returns:
//   - *openapi3.SchemaRef: The response model schema (empty schema for 204 No Content responses)
//   - error: If the resource cannot be loaded, the method is not supported, or no valid schema is found
//
// Example:
//
//	schema, err := GetResponseModelSchema(PackVMS, "GET", "apitokens")
//
// Notes:
//   - For GET requests, it automatically handles paginated responses, arrays, and single objects
//   - For other methods, it checks status codes 200, 201, 202 for content, and 204 for No Content
//   - 204 No Content responses return an empty schema since there's no response body
func GetResponseModelSchema(pack Pack, httpMethod, resourcePath string) (*openapi3.SchemaRef, error) {
	resource, err := GetOpenApiResource(pack, resourcePath)
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
		return getResponseModelSchemaForGET(pack, resource, resourcePath)
	}

	// For non-GET methods, check status codes 200, 201, 202
	for _, code := range []int{200, 201, 202} {
		resp := operation.Responses.Status(code)
		schemaRef := extractSchemaFromResponse(resp)
		if schemaRef != nil {
			final := ResolveComposedSchema(pack, ResolveAllRefs(pack, schemaRef))
			return &openapi3.SchemaRef{Value: final}, nil
		}
	}

	// Check for 204 No Content response (operation succeeded but no content to return)
	resp204 := operation.Responses.Status(204)
	if resp204 != nil && resp204.Value != nil {
		// 204 responses have no content, return empty schema
		return &openapi3.SchemaRef{Value: &openapi3.Schema{}}, nil
	}

	return nil, fmt.Errorf(
		"no valid schema found in %s response (200/201/202/204) for resource %s",
		httpMethod, resourcePath,
	)
}

// getResponseModelSchemaForGET handles GET-specific logic for extracting response schemas.
// It supports paginated (results[]), flat list ([]), and single-object responses.
func getResponseModelSchemaForGET(pack Pack, resource *openapi3.PathItem, resourcePath string) (*openapi3.SchemaRef, error) {
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

	rootSchema := ResolveComposedSchema(pack, ResolveAllRefs(pack, content.Schema))

	// 1. Check if response is paginated with "results" field (VMS DRF)
	if rootSchema.Type != nil && (*rootSchema.Type).Is("object") && rootSchema.Properties != nil {
		if resultsRef, ok := rootSchema.Properties["results"]; ok {
			resultsSchema := ResolveComposedSchema(pack, ResolveAllRefs(pack, resultsRef))
			if resultsSchema.Type != nil && (*resultsSchema.Type).Is("array") && resultsSchema.Items != nil {
				itemSchema := ResolveComposedSchema(pack, ResolveAllRefs(pack, resultsSchema.Items))
				return &openapi3.SchemaRef{Value: itemSchema}, nil
			}
		}
		// 1b. DataEngine cursor-paginated response with "data" field
		if dataRef, ok := rootSchema.Properties["data"]; ok {
			dataSchema := ResolveComposedSchema(pack, ResolveAllRefs(pack, dataRef))
			if dataSchema.Type != nil && (*dataSchema.Type).Is("array") && dataSchema.Items != nil {
				itemSchema := ResolveComposedSchema(pack, ResolveAllRefs(pack, dataSchema.Items))
				return &openapi3.SchemaRef{Value: itemSchema}, nil
			}
		}
	}

	// 2. Check if response is a flat array
	if rootSchema.Type != nil && (*rootSchema.Type).Is("array") && rootSchema.Items != nil {
		itemSchema := ResolveComposedSchema(pack, ResolveAllRefs(pack, rootSchema.Items))
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

func GetOperationSummary(pack Pack, httpMethod, resourcePath string) (string, error) {
	doc, err := loadOpenAPIDoc(pack)
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

// GetAllPaths returns every path defined in the OpenAPI document together with the
// HTTP methods that are declared for it.
//
// The returned map key is the raw path string from the spec (e.g. "/activedirectory/{id}/refresh/").
// The value is a sorted slice of uppercase HTTP method strings (e.g. ["GET", "PATCH"]).
// This is used by the code generator to auto-discover extra (non-CRUD) methods.
func GetAllPaths(pack Pack) (map[string][]string, error) {
	doc, err := loadOpenAPIDoc(pack)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI document: %w", err)
	}

	result := make(map[string][]string)
	for path, item := range doc.Paths.Map() {
		var methods []string
		if item.Get != nil {
			methods = append(methods, "GET")
		}
		if item.Post != nil {
			methods = append(methods, "POST")
		}
		if item.Put != nil {
			methods = append(methods, "PUT")
		}
		if item.Patch != nil {
			methods = append(methods, "PATCH")
		}
		if item.Delete != nil {
			methods = append(methods, "DELETE")
		}
		if item.Head != nil {
			methods = append(methods, "HEAD")
		}
		if item.Options != nil {
			methods = append(methods, "OPTIONS")
		}
		sort.Strings(methods)
		result[path] = methods
	}
	return result, nil
}

// ValidateOperationExists checks if a specific HTTP method exists for a given path in the OpenAPI spec
// Returns an error if the path or method doesn't exist
func ValidateOperationExists(pack Pack, httpMethod, resourcePath string) error {
	doc, err := loadOpenAPIDoc(pack)
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
func GetDeleteParams(pack Pack, resourcePath string) (*DeleteParams, error) {
	doc, err := loadOpenAPIDoc(pack)
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
	}
	// DataEngine and other packs may use {guid} (or only the unsuffixed form).
	if pathItem == nil {
		if itemPath, rerr := ResolveCollectionItemPath(pack, resourcePath); rerr == nil {
			if got, gerr := GetOpenApiResource(pack, itemPath); gerr == nil {
				pathItem = got
				normalizedPath = itemPath
			}
		}
	}
	if pathItem == nil {
		return nil, fmt.Errorf("path not found: %s", normalizedPath)
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
			} else if paramRef.Value.In == "path" && (paramRef.Value.Name == "id" || paramRef.Value.Name == "guid") {
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
func ReturnsTextPlain(pack Pack, httpMethod, resourcePath string) (bool, error) {
	doc, err := loadOpenAPIDoc(pack)
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
func Returns204NoContent(pack Pack, httpMethod, resourcePath string) (bool, error) {
	doc, err := loadOpenAPIDoc(pack)
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
