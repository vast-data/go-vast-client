//go:build tools

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/vast-data/go-vast-client/api"
	"github.com/vast-data/go-vast-client/autogen/vastparser"
)

// Field represents a struct field
type Field struct {
	Name        string
	Type        string
	JSONTag     string
	YAMLTag     string
	RequiredTag string
	DocTag      string
}

// NestedType represents a nested struct type that needs to be generated
type NestedType struct {
	Name   string
	Fields []Field
}

// TypeRegistry keeps track of generated types to avoid duplicates
type TypeRegistry struct {
	types map[string]*NestedType
}

func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		types: make(map[string]*NestedType),
	}
}

// RegisterType adds a new nested type to the registry
func (tr *TypeRegistry) RegisterType(name string, fields []Field) string {
	if existing, exists := tr.types[name]; exists {
		// Type already exists, return existing name
		return existing.Name
	}

	nestedType := &NestedType{
		Name:   name,
		Fields: fields,
	}
	tr.types[name] = nestedType
	return name
}

// GetTypes returns all registered types sorted by name for consistent generation
func (tr *TypeRegistry) GetTypes() []*NestedType {
	var types []*NestedType
	for _, t := range tr.types {
		types = append(types, t)
	}
	// Sort by name for consistent generation order
	sort.Slice(types, func(i, j int) bool {
		return types[i].Name < types[j].Name
	})
	return types
}

// ResourceData represents data for template generation
type ResourceData struct {
	Name               string
	LowerName          string
	PluralName         string
	SearchParamsFields []Field
	RequestBodyFields  []Field
	ResponseBodyFields []Field
	NestedTypes        []*NestedType
	Resource           *vastparser.VastResource
}

// GetRequestURL returns the request URL for the given method
func (r *ResourceData) GetRequestURL(method string) string {
	return r.Resource.GetRequestURL(method)
}

// GetResponseURL returns the response URL for the given method
func (r *ResourceData) GetResponseURL(method string) string {
	return r.Resource.GetResponseURL(method)
}

// TemplateData represents the data passed to the template
type TemplateData struct {
	Resources []ResourceData
}

func main() {
	// Hardcoded paths - this tool has one specific purpose
	inputFile := "../vast_resource.go"
	outputDir := "../typed"

	// Parse the input file to find resources with APIBuilder markers
	parser := vastparser.NewVastResourceParser()
	resources, err := parser.ParseFile(inputFile)
	if err != nil {
		log.Fatalf("Failed to parse file: %v", err)
	}

	if len(resources) == 0 {
		log.Println("No resources with APIBuilder markers found")
		return
	}

	// Sort resources by name for consistent generation order
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Name < resources[j].Name
	})

	fmt.Printf("Found %d resources with APIBuilder markers:\n", len(resources))
	for _, resource := range resources {
		fmt.Printf("  - %s\n", resource.Name)
	}

	// Generate template data
	templateData := TemplateData{}
	for _, resource := range resources {
		resourceData := ResourceData{
			Name:       resource.Name,
			LowerName:  strings.ToLower(resource.Name),
			PluralName: resource.Name + "s", // Simple pluralization
			Resource:   &resource,
		}

		// Create separate registries for each generation phase to avoid type name conflicts
		searchRegistry := NewTypeRegistry()
		requestRegistry := NewTypeRegistry()
		responseRegistry := NewTypeRegistry()

		// Generate search params fields
		if resource.HasSearchQuery("GET") {
			searchURL := resource.GetSearchQuery("GET")
			searchFields, err := generateSearchParamsFields(searchURL, "GET", searchRegistry)
			if err != nil {
				log.Printf("Warning: Failed to generate search params fields for %s: %v", resource.Name, err)
			} else {
				resourceData.SearchParamsFields = searchFields
			}
		} else if resource.HasSearchQuery("SCHEMA") {
			schemaName := resource.GetSearchQuery("SCHEMA")
			searchFields, err := generateSearchParamsFromSchema(schemaName, searchRegistry)
			if err != nil {
				log.Printf("Warning: Failed to generate search params from schema for %s: %v", resource.Name, err)
			} else {
				resourceData.SearchParamsFields = searchFields
			}
		}

		// Generate request body fields
		if resource.HasRequestBody("POST") {
			requestURL := resource.GetRequestBody("POST")
			requestFields, err := generateRequestBodyFields(requestURL, "POST", requestRegistry)
			if err != nil {
				log.Printf("Warning: Failed to generate request body fields for %s: %v", resource.Name, err)
			} else {
				resourceData.RequestBodyFields = requestFields
			}
		} else if resource.HasRequestBody("SCHEMA") {
			schemaName := resource.GetRequestBody("SCHEMA")
			requestFields, err := generateRequestBodyFromSchema(schemaName, requestRegistry)
			if err != nil {
				log.Printf("Warning: Failed to generate request body from schema for %s: %v", resource.Name, err)
			} else {
				resourceData.RequestBodyFields = requestFields
			}
		}

		// Generate response body fields
		if resource.HasResponseBody("POST") {
			responseURL := resource.GetResponseBody("POST")
			responseFields, err := generateResponseBodyFields(responseURL, "POST", responseRegistry)
			if err != nil {
				log.Printf("Warning: Failed to generate response body fields for %s: %v", resource.Name, err)
			} else {
				resourceData.ResponseBodyFields = responseFields
			}
		} else if resource.HasResponseBody("SCHEMA") {
			schemaName := resource.GetResponseBody("SCHEMA")
			responseFields, err := generateResponseBodyFromSchema(schemaName, responseRegistry)
			if err != nil {
				log.Printf("Warning: Failed to generate response body from schema for %s: %v", resource.Name, err)
			} else {
				resourceData.ResponseBodyFields = responseFields
			}
		}

		// Combine all nested types from all registries
		var allNestedTypes []*NestedType
		allNestedTypes = append(allNestedTypes, searchRegistry.GetTypes()...)
		allNestedTypes = append(allNestedTypes, requestRegistry.GetTypes()...)
		allNestedTypes = append(allNestedTypes, responseRegistry.GetTypes()...)

		// Sort combined nested types for consistent generation
		sort.Slice(allNestedTypes, func(i, j int) bool {
			return allNestedTypes[i].Name < allNestedTypes[j].Name
		})

		resourceData.NestedTypes = allNestedTypes

		templateData.Resources = append(templateData.Resources, resourceData)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Generate the rest.go file
	restFile := filepath.Join(outputDir, "rest.go")
	if err := generateRestFile(restFile, templateData); err != nil {
		log.Fatalf("Failed to generate rest.go: %v", err)
	}

	// Generate separate files for each resource
	var generatedFiles []string
	for _, resourceData := range templateData.Resources {
		resourceFile := filepath.Join(outputDir, strings.ToLower(resourceData.Name)+".go")
		if err := generateResourceFile(resourceFile, resourceData); err != nil {
			log.Fatalf("Failed to generate %s: %v", resourceFile, err)
		}
		generatedFiles = append(generatedFiles, strings.ToLower(resourceData.Name)+".go")
	}

	fmt.Printf("Generated typed resources for %d resources in %s/\n", len(resources), outputDir)
	fmt.Printf("  - rest.go: Typed VMSRest client\n")
	for _, file := range generatedFiles {
		fmt.Printf("  - %s: Typed resource implementation\n", file)
	}
}

// generateRestFile generates the rest.go file with typed VMSRest client
func generateRestFile(filename string, data TemplateData) error {
	tmpl, err := template.ParseFiles("templates/rest.tpl")
	if err != nil {
		return fmt.Errorf("failed to parse rest template: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create rest file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute rest template: %w", err)
	}

	return nil
}

// generateResourceFile generates a single resource file with typed resource implementation
func generateResourceFile(filename string, data ResourceData) error {
	tmpl, err := template.ParseFiles("templates/resource.tpl")
	if err != nil {
		return fmt.Errorf("failed to parse resource template: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create resource file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute resource template: %w", err)
	}

	return nil
}

// generateRequestFields generates struct fields from GET query parameters
func generateRequestFields(resourcePath string, registry *TypeRegistry) ([]Field, error) {
	// Get query parameters schema from OpenAPI
	schema, err := api.GetSchema_GET_QueryParams(resourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get query params schema: %w", err)
	}

	return generateFieldsFromSchema(schema.Value, resourcePath+"Request", registry, false)
}

// toCamelCase converts snake_case to CamelCase
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// isObject returns true if the schema represents an object type
func isObject(prop *openapi3.Schema) bool {
	return prop.Type != nil && len(*prop.Type) > 0 && (*prop.Type)[0] == openapi3.TypeObject
}

// isAmbiguousObject returns true if the schema is an object without properties (ambiguous)
func isAmbiguousObject(prop *openapi3.Schema) bool {
	return isObject(prop) && len(prop.Properties) == 0
}

// excludeSearchParams contains common search parameters that should be excluded from typed search params
var excludeSearchParams = []string{"page", "page_size", "sync", "created", "sync_time"}

// isPrimitive returns true if the given OpenAPI schema represents a primitive type
// supported by search parameters (string, integer, number, or boolean).
func isPrimitive(prop *openapi3.Schema) bool {
	if prop == nil || prop.Type == nil || len(*prop.Type) == 0 {
		return false
	}

	switch (*prop.Type)[0] {
	case openapi3.TypeString, openapi3.TypeInteger, openapi3.TypeNumber, openapi3.TypeBoolean:
		return true
	default:
		return false
	}
}

// contains checks if a slice contains a specific value
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// sortFieldsByRequired sorts fields so required fields come first, then non-required
func sortFieldsByRequired(fields []Field) {
	sort.Slice(fields, func(i, j int) bool {
		// Required fields come first
		if fields[i].RequiredTag == "true" && fields[j].RequiredTag == "false" {
			return true
		}
		if fields[i].RequiredTag == "false" && fields[j].RequiredTag == "true" {
			return false
		}
		// If both have same required status, sort alphabetically by name
		return fields[i].Name < fields[j].Name
	})
}

// IsEmptySchema returns true if the schema reference is empty or has no meaningful content
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

// getGoTypeFromOpenAPI converts OpenAPI schema type to Go type
func getGoTypeFromOpenAPI(schema *openapi3.Schema, usePointers bool) string {
	if schema == nil || schema.Type == nil || len(*schema.Type) == 0 {
		if usePointers {
			return "*string" // default fallback
		}
		return "string"
	}

	baseType := (*schema.Type)[0]
	var goType string

	switch baseType {
	case "string":
		goType = "string"
	case "integer":
		if schema.Format == "int64" {
			goType = "int64"
		} else {
			goType = "int64" // default to int64 for integers
		}
	case "number":
		if schema.Format == "float" {
			goType = "float32"
		} else {
			goType = "float64" // default to float64 for numbers
		}
	case "boolean":
		goType = "bool"
	case "array":
		goType = "[]interface{}" // generic array type
	case "object":
		goType = "map[string]interface{}" // generic object type
	default:
		goType = "interface{}" // fallback for unknown types
	}

	if usePointers && baseType != "array" && baseType != "object" {
		return "*" + goType
	}
	return goType
}

// generateResponseFields generates struct fields from POST response schema
func generateResponseFields(resourcePath string, registry *TypeRegistry) ([]Field, error) {
	// Get response schema from OpenAPI
	schema, err := api.GetSchema_POST_StatusOk(resourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get response schema: %w", err)
	}

	return generateFieldsFromSchema(schema.Value, resourcePath+"Response", registry, false)
}

// generateSearchParamsFields generates search params fields using method-based resolution
func generateSearchParamsFields(resourcePath, method string, registry *TypeRegistry) ([]Field, error) {
	// Use method-based switch like terraform provider
	switch method {
	case http.MethodGet:
		// For GET requests, get individual query parameters
		params, err := api.QueryParametersGET(resourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get GET query params for resource %q: %w", resourcePath, err)
		}

		return generateSearchParamsFromParameters(params, resourcePath, registry)
	default:
		return nil, fmt.Errorf("unsupported method %q for search params generation", method)
	}
}

// generateSearchParamsFromParameters generates search params fields from individual parameters
func generateSearchParamsFromParameters(params []*openapi3.Parameter, resourcePath string, registry *TypeRegistry) ([]Field, error) {
	var fields []Field

	for _, p := range params {
		if p == nil || p.Schema == nil || p.Schema.Value == nil {
			continue
		}

		// Apply filtering logic from terraform provider
		if !isPrimitive(p.Schema.Value) {
			// We search only for primitive types
			log.Printf("Skipping non-primitive search param '%s' for resource %s", p.Name, resourcePath)
			continue
		}

		name := p.Name
		if contains(excludeSearchParams, name) {
			log.Printf("Skipping excluded search param '%s' for resource %s", name, resourcePath)
			continue
		}

		if p.Schema == nil || p.Schema.Value == nil || p.Schema.Value.Type == nil || len(*p.Schema.Value.Type) == 0 {
			log.Printf("Skipping search param '%s' with invalid schema for resource %s", name, resourcePath)
			continue
		}

		// Generate field for this parameter
		field := Field{
			Name:        toCamelCase(name),
			JSONTag:     name,
			YAMLTag:     name,
			RequiredTag: "false", // Search parameters are typically optional
			DocTag:      p.Description,
		}

		// Convert OpenAPI type to Go type (no pointers for search params - omitempty works with zero values)
		goType := getGoTypeFromOpenAPI(p.Schema.Value, false)
		field.Type = goType

		fields = append(fields, field)
	}

	// Sort fields: required first, then non-required
	sortFieldsByRequired(fields)

	return fields, nil
}

// generateRequestBodyFields generates request body fields using method-based resolution
func generateRequestBodyFields(resourcePath, method string, registry *TypeRegistry) ([]Field, error) {
	var schema *openapi3.SchemaRef
	var err error

	// Use method-based switch like terraform provider (createSchemaRef pattern)
	switch method {
	case http.MethodPost:
		schema, err = api.GetSchema_POST_RequestBody(resourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get POST request body schema for resource %q: %w", resourcePath, err)
		}
	case http.MethodPatch:
		schema, err = api.GetSchema_PATCH_RequestBody(resourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get PATCH request body schema for resource %q: %w", resourcePath, err)
		}
	default:
		return nil, fmt.Errorf("unsupported method %q for request body generation", method)
	}

	if IsEmptySchema(schema) {
		return nil, fmt.Errorf("request body schema is empty for resource %q", resourcePath)
	}

	// Convert resource path to proper Go type name (e.g., "quotas" -> "Quotas")
	typeName := toCamelCase(resourcePath) + "RequestBody"
	return generateFieldsFromSchema(schema.Value, typeName, registry, false)
}

// generateResponseBodyFields generates response body fields using method-based resolution
func generateResponseBodyFields(resourcePath, method string, registry *TypeRegistry) ([]Field, error) {
	var schema *openapi3.SchemaRef
	var err error

	// Use method-based switch like terraform provider (modelSchemaRef pattern)
	switch method {
	case http.MethodPost:
		schema, err = api.GetSchema_POST_StatusOk(resourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get POST response schema for resource %q: %w", resourcePath, err)
		}
	case http.MethodGet:
		schema, err = api.GetSchema_GET_StatusOk(resourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get GET response schema for resource %q: %w", resourcePath, err)
		}
	default:
		return nil, fmt.Errorf("unsupported method %q for response body generation", method)
	}

	if IsEmptySchema(schema) {
		return nil, fmt.Errorf("response body schema is empty for resource %q", resourcePath)
	}

	// Convert resource path to proper Go type name (e.g., "quotas" -> "Quotas")
	typeName := toCamelCase(resourcePath) + "ResponseBody"
	return generateFieldsFromSchema(schema.Value, typeName, registry, false)
}

// generateSearchParamsFromSchema generates search params fields from a schema component
func generateSearchParamsFromSchema(schemaName string, registry *TypeRegistry) ([]Field, error) {
	// Get schema from components
	schema, err := api.GetSchema_FromComponents(schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema from components: %w", err)
	}

	return generateFieldsFromSchema(schema.Value, schemaName+"SearchParams", registry, true)
}

// generateRequestBodyFromSchema generates request body fields from a schema component
func generateRequestBodyFromSchema(schemaName string, registry *TypeRegistry) ([]Field, error) {
	// Get schema from components
	schema, err := api.GetSchema_FromComponents(schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema from components: %w", err)
	}

	return generateFieldsFromSchema(schema.Value, schemaName+"RequestBody", registry, false)
}

// generateResponseBodyFromSchema generates response body fields from a schema component
func generateResponseBodyFromSchema(schemaName string, registry *TypeRegistry) ([]Field, error) {
	// Get schema from components
	schema, err := api.GetSchema_FromComponents(schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema from components: %w", err)
	}

	return generateFieldsFromSchema(schema.Value, schemaName+"ResponseBody", registry, false)
}

// generateFieldsFromSchema recursively generates fields from an OpenAPI schema
func generateFieldsFromSchema(schema *openapi3.Schema, parentTypeName string, registry *TypeRegistry, usePointers bool) ([]Field, error) {
	if schema == nil || schema.Properties == nil {
		return nil, nil
	}

	var fields []Field

	// Get property names and sort them for consistent generation order
	var propNames []string
	for propName := range schema.Properties {
		propNames = append(propNames, propName)
	}
	sort.Strings(propNames)

	for _, propName := range propNames {
		propRef := schema.Properties[propName]
		if propRef == nil || propRef.Value == nil {
			continue
		}

		// Skip ambiguous objects (objects without properties) like terraform provider
		if isAmbiguousObject(propRef.Value) {
			log.Printf("Warning: Skipping ambiguous object field '%s' (object without properties)", propName)
			continue
		}

		// Check if this field is required
		isRequired := "false"
		for _, requiredField := range schema.Required {
			// Check exact match first
			if requiredField == propName {
				isRequired = "true"
				break
			}
			// Handle OpenAPI schema inconsistencies where required field names
			// might use different underscore patterns than property names
			// e.g., required: "policy__id" but property: "policy_id"
			normalizedRequired := strings.ReplaceAll(requiredField, "__", "_")
			normalizedProp := strings.ReplaceAll(propName, "__", "_")
			if normalizedRequired == normalizedProp {
				isRequired = "true"
				break
			}
		}

		field := Field{
			Name:        toCamelCase(propName),
			JSONTag:     propName,
			YAMLTag:     propName,
			RequiredTag: isRequired,
			DocTag:      propRef.Value.Description,
		}

		// Recursively determine Go type
		goType, err := getGoTypeFromOpenAPIRecursive(propRef.Value, parentTypeName+"_"+toCamelCase(propName), registry, usePointers)
		if err != nil {
			return nil, fmt.Errorf("failed to generate type for field %s: %w", propName, err)
		}
		field.Type = goType

		fields = append(fields, field)
	}

	// Sort fields: required first, then non-required
	sortFieldsByRequired(fields)

	return fields, nil
}

// getGoTypeFromOpenAPIRecursive recursively converts OpenAPI schema to Go type, generating nested structs as needed
func getGoTypeFromOpenAPIRecursive(schema *openapi3.Schema, typeName string, registry *TypeRegistry, usePointers bool) (string, error) {
	if schema == nil || schema.Type == nil || len(*schema.Type) == 0 {
		if usePointers {
			return "*string", nil // default fallback
		}
		return "string", nil
	}

	baseType := (*schema.Type)[0]
	var goType string

	switch baseType {
	case "string":
		goType = "string"
	case "integer":
		if schema.Format == "int64" {
			goType = "int64"
		} else {
			goType = "int64" // default to int64 for integers
		}
	case "number":
		if schema.Format == "float" {
			goType = "float32"
		} else {
			goType = "float64" // default to float64 for numbers
		}
	case "boolean":
		goType = "bool"
	case "array":
		if schema.Items == nil || schema.Items.Value == nil {
			goType = "[]interface{}" // fallback for arrays without items
		} else {
			itemType, err := getGoTypeFromOpenAPIRecursive(schema.Items.Value, typeName+"Item", registry, false)
			if err != nil {
				return "", fmt.Errorf("failed to generate array item type: %w", err)
			}
			goType = "[]" + itemType
		}
	case "object":
		if schema.Properties == nil || len(schema.Properties) == 0 {
			// Empty object or map with additionalProperties
			if schema.AdditionalProperties.Schema != nil {
				valueType, err := getGoTypeFromOpenAPIRecursive(schema.AdditionalProperties.Schema.Value, typeName+"Value", registry, false)
				if err != nil {
					return "", fmt.Errorf("failed to generate map value type: %w", err)
				}
				goType = "map[string]" + valueType
			} else {
				goType = "map[string]interface{}" // generic object type
			}
		} else {
			// Object with defined properties - generate a nested struct
			nestedFields, err := generateFieldsFromSchema(schema, typeName, registry, false)
			if err != nil {
				return "", fmt.Errorf("failed to generate nested fields: %w", err)
			}

			// Register the nested type
			registry.RegisterType(typeName, nestedFields)
			goType = typeName
		}
	default:
		goType = "interface{}" // fallback for unknown types
	}

	if usePointers && baseType != "array" && baseType != "object" {
		return "*" + goType, nil
	}
	return goType, nil
}
