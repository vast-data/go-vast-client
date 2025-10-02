//go:build tools

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/vast-data/go-vast-client/codegen/apibuilder"
	"github.com/vast-data/go-vast-client/codegen/vastparser"
	api "github.com/vast-data/go-vast-client/openapi_schema"
)

// isAmbiguousArray checks if a schema represents an array of ambiguous objects
func isAmbiguousArray(schema *openapi3.Schema) bool {
	if schema == nil || schema.Type == nil || len(*schema.Type) == 0 {
		return false
	}

	// Check if it's an array type
	for _, t := range *schema.Type {
		if t == "array" {
			// Check if the array items are ambiguous objects
			if schema.Items != nil && schema.Items.Value != nil {
				return isAmbiguousObject(schema.Items.Value)
			}
		}
	}

	return false
}

// pluralize converts a resource name to its plural form used in the untyped client
func pluralize(name string) string {
	// Handle special cases that don't follow simple "s" pluralization
	specialCases := map[string]string{
		"ActiveDirectory":    "ActiveDirectories",
		"Dns":                "Dns",                // DNS is already plural-like (Domain Name System)
		"Nis":                "Nis",                // NIS is already plural-like (Network Information Service)
		"ProtectionPolicy":   "ProtectionPolicies", // Policy -> Policies
		"QosPolicy":          "QosPolicies",        // Policy -> Policies
		"ViewPolicy":         "ViewPolies",         // Note: ViewPolies appears to be a typo in rest.go
		"S3Policy":           "S3Policies",         // Policy -> Policies
		"ReplicationPeers":   "ReplicationPeers",   // Already plural
		"S3replicationPeers": "S3replicationPeers", // Already plural
		"UserKey":            "UserKeys",           // Key -> Keys
		"NonLocalUserKey":    "NonLocalUserKeys",   // Key -> Keys
		"LocalS3Key":         "LocalS3Keys",        // Key -> Keys
		"Vms":                "Vms",                // VMS is already plural-like (Virtual Machine System)
		// Add other irregular plurals as needed
	}

	if plural, exists := specialCases[name]; exists {
		return plural
	}

	// Default case: simple "s" addition
	return name + "s"
}

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
	Name    string
	Fields  []Field
	Section string // Section where this type belongs (e.g., "SEARCH PARAMS", "CREATE BODY", "MODEL")
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
func (tr *TypeRegistry) RegisterType(name string, fields []Field, section string) string {
	if existing, exists := tr.types[name]; exists {
		// Type already exists, return existing name
		return existing.Name
	}

	nestedType := &NestedType{
		Name:    name,
		Fields:  fields,
		Section: section,
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

// generateExtraMethodInfo generates type information for an extra method
func generateExtraMethodInfo(resourceName string, extraMethod apibuilder.ExtraMethod) (ExtraMethodInfo, []*NestedType, error) {
	// CRITICAL: Validate that the method exists in the OpenAPI schema FIRST
	// This catches typos in markers (e.g., GET instead of PATCH, misspelled paths)
	if err := api.ValidateOperationExists(extraMethod.Method, extraMethod.Path); err != nil {
		// This is a fatal error - the marker is incorrect and must be fixed
		return ExtraMethodInfo{}, nil, fmt.Errorf("🔴 FATAL: Invalid marker for %s - %v. Please check the marker declaration and fix the method or path", resourceName, err)
	}

	methodInfo := ExtraMethodInfo{
		HTTPMethod: extraMethod.Method,
		Path:       extraMethod.Path,
	}

	// Convert HTTP method to Go constant
	methodInfo.GoHTTPMethod = httpMethodToGoConstant(extraMethod.Method)

	// Extract summary from OpenAPI spec
	summary, err := api.GetOperationSummary(extraMethod.Method, extraMethod.Path)
	if err != nil {
		// If summary not found, just log and continue
		fmt.Printf("  ℹ️  No summary found for %s %s\n", extraMethod.Method, extraMethod.Path)
	} else {
		methodInfo.Summary = summary
	}

	// Check if path contains {id}
	methodInfo.HasID = strings.Contains(extraMethod.Path, "{id}")

	// Parse path to extract resource path and sub-path
	pathParts := strings.Split(strings.Trim(extraMethod.Path, "/"), "/")
	if len(pathParts) > 0 {
		methodInfo.ResourcePath = pathParts[0]
	}

	// Extract sub-path (everything after {id})
	if methodInfo.HasID {
		idIndex := -1
		for i, part := range pathParts {
			if part == "{id}" {
				idIndex = i
				break
			}
		}
		if idIndex >= 0 && idIndex < len(pathParts)-1 {
			methodInfo.SubPath = strings.Join(pathParts[idIndex+1:], "/")
			methodInfo.SubPath = strings.TrimSuffix(methodInfo.SubPath, "/")
		}
	}

	// Generate method name from path
	lastPart := pathParts[len(pathParts)-1]
	if lastPart == "" && len(pathParts) > 1 {
		lastPart = pathParts[len(pathParts)-2]
	}
	lastPart = cleanPathPart(lastPart)
	action := toCamelCase(lastPart)

	// Store base name without HTTP method suffix
	methodInfo.Name = resourceName + action
	methodInfo.BaseName = methodInfo.Name
	methodInfo.BodyTypeName = methodInfo.Name + "_" + extraMethod.Method + "_Body"
	methodInfo.ResponseTypeName = methodInfo.Name + "_" + extraMethod.Method + "_Model"

	// Don't assume HasBody - will be determined by checking OpenAPI spec for actual request body
	methodInfo.HasBody = false
	methodInfo.HasParams = extraMethod.Method == "GET"

	var allNestedTypes []*NestedType

	// Check if method returns 204 No Content
	returns204, err := api.Returns204NoContent(extraMethod.Method, extraMethod.Path)
	if err != nil {
		fmt.Printf("  ℹ️  Could not check 204 status for %s %s\n", extraMethod.Method, extraMethod.Path)
	} else if returns204 {
		methodInfo.ReturnsNoContent = true
		fmt.Printf("  ℹ️  Method returns 204 No Content\n")
	}

	// Check if this is a bare array response BEFORE schema unwrapping
	// (GetResponseModelSchema unwraps arrays for GET, so we need to check the raw schema first)
	isBareArray := false
	if extraMethod.Method == "GET" || extraMethod.Method == "POST" {
		if rawResp, err := api.GetOpenApiResource(extraMethod.Path); err == nil && rawResp != nil {
			var op *openapi3.Operation
			if extraMethod.Method == "GET" {
				op = rawResp.Get
			} else if extraMethod.Method == "POST" {
				op = rawResp.Post
			}
			if op != nil {
				if resp := op.Responses.Status(200); resp != nil && resp.Value != nil {
					if content := resp.Value.Content["application/json"]; content != nil && content.Schema != nil && content.Schema.Value != nil {
						if content.Schema.Value.Type != nil && (*content.Schema.Value.Type).Is("array") {
							isBareArray = true
							methodInfo.ReturnsArray = true
							fmt.Printf("  ℹ️  Detected bare array response\n")
						}
					}
				}
			}
		}
	}

	// Check if method returns text/plain
	returnsTextPlain, err := api.ReturnsTextPlain(extraMethod.Method, extraMethod.Path)
	if err != nil {
		fmt.Printf("  ℹ️  Could not check text/plain for %s %s\n", extraMethod.Method, extraMethod.Path)
	} else if returnsTextPlain {
		methodInfo.ReturnsTextPlain = true
		fmt.Printf("  ℹ️  Extra method returns text/plain\n")
	}

	// Check if response is AsyncTaskInResponse - these are async methods that need timeout parameter
	if rawResp, err := api.GetOpenApiResource(extraMethod.Path); err == nil && rawResp != nil {
		var op *openapi3.Operation
		switch extraMethod.Method {
		case "GET":
			op = rawResp.Get
		case "POST":
			op = rawResp.Post
		case "PATCH":
			op = rawResp.Patch
		case "PUT":
			op = rawResp.Put
		case "DELETE":
			op = rawResp.Delete
		}

		if op != nil {
			if resp := op.Responses.Status(200); resp != nil && resp.Value != nil {
				if content := resp.Value.Content["application/json"]; content != nil && content.Schema != nil {
					// Check if response references AsyncTaskInResponse
					if content.Schema.Ref == "#/components/schemas/AsyncTaskInResponse" {
						// This is an async method - add timeout parameter
						methodInfo.IsAsyncTask = true
						fmt.Printf("  ℹ️  Async task method detected (will add timeout parameter)\n")
					}
				}
			}
		}
	}

	// Generate query parameter fields for GET extra methods
	if methodInfo.HasParams && extraMethod.Method == "GET" {
		paramsRegistry := NewTypeRegistry()
		// Get query parameters from OpenAPI
		params, err := api.QueryParametersGET(extraMethod.Path)
		if err != nil {
			fmt.Printf("  ℹ️  No query parameters found for GET %s\n", extraMethod.Path)
		} else if len(params) > 0 {
			bodyFields, err := generateSearchParamsFromParameters(params, extraMethod.Path, paramsRegistry)
			if err != nil {
				fmt.Printf("  ℹ️  Failed to generate query params for GET %s: %v\n", extraMethod.Path, err)
			} else {
				methodInfo.BodyFields = bodyFields // Reuse BodyFields for query params
				fmt.Printf("  ℹ️  Generated %d query parameters\n", len(bodyFields))

				// Tag nested types
				paramsNestedTypes := paramsRegistry.GetTypes()
				for _, nt := range paramsNestedTypes {
					nt.Section = "EXTRA_METHOD_BODY"
					oldName := nt.Name
					nt.Name = sanitizeNestedTypeName(nt.Name, methodInfo.Name)
					for i := range methodInfo.BodyFields {
						methodInfo.BodyFields[i].Type = strings.ReplaceAll(methodInfo.BodyFields[i].Type, oldName, nt.Name)
					}
				}
				methodInfo.NestedTypes = append(methodInfo.NestedTypes, paramsNestedTypes...)
				allNestedTypes = append(allNestedTypes, paramsNestedTypes...)
			}
		}
	}

	// Check OpenAPI spec for request body schema for POST/PUT/PATCH/DELETE methods
	// Only set HasBody = true if an actual request body schema exists
	if extraMethod.Method == "POST" || extraMethod.Method == "PUT" || extraMethod.Method == "PATCH" || extraMethod.Method == "DELETE" {
		bodyRegistry := NewTypeRegistry()
		bodyFields, err := generateRequestBodyFields(extraMethod.Path, extraMethod.Method, bodyRegistry)
		if err != nil {
			// If schema not found, method has no body
			fmt.Printf("  ℹ️  No request body schema found for %s %s\n", extraMethod.Method, extraMethod.Path)
			methodInfo.HasBody = false
		} else if len(bodyFields) > 0 {
			// Body schema exists - set HasBody = true
			methodInfo.HasBody = true
			methodInfo.BodyFields = bodyFields

			// If DELETE has a body, switch from query params to body
			if extraMethod.Method == "DELETE" {
				methodInfo.HasParams = false
			}

			// Check if body can be simplified (1-3 simple fields)
			if canSimplifyBody(bodyFields) {
				methodInfo.SimplifiedBody = true
				methodInfo.SimplifiedParams = convertFieldsToSimplifiedParams(bodyFields)
				fmt.Printf("  ℹ️  Simplified body: %d inline parameters\n", len(methodInfo.SimplifiedParams))
			}

			// Tag nested types with EXTRA_METHOD_BODY section and fix names
			bodyNestedTypes := bodyRegistry.GetTypes()

			// First pass: sanitize all nested type names
			oldToNew := make(map[string]string)
			for _, nt := range bodyNestedTypes {
				nt.Section = "EXTRA_METHOD_BODY"
				oldName := nt.Name
				nt.Name = sanitizeNestedTypeName(nt.Name, methodInfo.Name)
				oldToNew[oldName] = nt.Name
			}

			// Second pass: update all field type references (including nested-nested types)
			for _, nt := range bodyNestedTypes {
				for i := range nt.Fields {
					for oldName, newName := range oldToNew {
						nt.Fields[i].Type = strings.ReplaceAll(nt.Fields[i].Type, oldName, newName)
					}
				}
			}

			// Third pass: update body field type references
			for i := range methodInfo.BodyFields {
				for oldName, newName := range oldToNew {
					methodInfo.BodyFields[i].Type = strings.ReplaceAll(methodInfo.BodyFields[i].Type, oldName, newName)
				}
			}

			methodInfo.NestedTypes = append(methodInfo.NestedTypes, bodyNestedTypes...)
			allNestedTypes = append(allNestedTypes, bodyNestedTypes...)
		}
	}

	// Check response schema FIRST for ambiguous objects or missing schemas before attempting to generate fields
	// This prevents generating methods that would return core.Record
	// Skip validation if:
	//   - This is an async task method (returns AsyncTaskInResponse, we only care about task completion)
	//   - This returns no content (204)
	//   - This returns text/plain
	//   - This is a bare array response (validation happens during field generation)
	//   - This is a DELETE method (DELETE often returns success status without response body)
	if !methodInfo.ReturnsNoContent && !methodInfo.ReturnsTextPlain && !methodInfo.IsAsyncTask && !isBareArray {
		schema, schemaErr := api.GetResponseModelSchema(extraMethod.Method, extraMethod.Path)

		// CRITICAL: Skip if no response schema is defined at all
		// EXCEPTION: DELETE methods are allowed without response schema (they return only error)
		if schemaErr != nil {
			if extraMethod.Method == "DELETE" {
				// DELETE without response schema is OK - treat it like 204 No Content
				methodInfo.ReturnsNoContent = true
				fmt.Printf("  ℹ️  DELETE method has no response schema - treating as No Content\n")
			} else {
				return ExtraMethodInfo{}, nil, fmt.Errorf("❌ SKIPPED: %s %s - No response schema defined in OpenAPI spec. Method would return core.Record which is not allowed for typed methods. Error: %v", extraMethod.Method, extraMethod.Path, schemaErr)
			}
		}

		// Skip if schema is ambiguous (type: object with no properties or array of ambiguous objects)
		if schema != nil && schema.Value != nil {
			// Check for primitive response types (string, number, boolean, integer)
			if isPrimitive(schema.Value) {
				// Primitive responses are supported - map to Go type
				goType := mapOpenAPITypeToGo(schema.Value)
				methodInfo.ReturnsPrimitive = true
				methodInfo.PrimitiveType = goType
				fmt.Printf("  ℹ️  Primitive response detected: %s\n", goType)
			} else if isMapObject(schema.Value) {
				// Skip map-like responses (objects with additionalProperties but no properties)
				// These would return map[string]T in Go, which is not currently supported for typed methods
				// TODO: Add support for map[string]string, map[string]int, etc. return types
				mapValueType := "any"
				if schema.Value.AdditionalProperties.Schema != nil && schema.Value.AdditionalProperties.Schema.Value != nil {
					if schema.Value.AdditionalProperties.Schema.Value.Type != nil && len(*schema.Value.AdditionalProperties.Schema.Value.Type) > 0 {
						mapValueType = string((*schema.Value.AdditionalProperties.Schema.Value.Type)[0])
					}
				}
				return ExtraMethodInfo{}, nil, fmt.Errorf("❌ SKIPPED: %s %s - Response is a map-like object (additionalProperties: %s). Typed methods do not support map return types", extraMethod.Method, extraMethod.Path, mapValueType)
			} else if schema.Value.Type != nil && len(*schema.Value.Type) > 0 && (*schema.Value.Type)[0] == openapi3.TypeArray {
				// Check for bare array responses (arrays not wrapped in an object)
				// We can handle these if the array items are well-defined (e.g., []Lock)
				// Check if array items are defined and valid
				if schema.Value.Items == nil || schema.Value.Items.Value == nil {
					return ExtraMethodInfo{}, nil, fmt.Errorf("❌ SKIPPED: %s %s - Response is a bare array with undefined items. Cannot generate typed methods", extraMethod.Method, extraMethod.Path)
				}

				items := schema.Value.Items.Value

				// Skip if items are ambiguous objects
				if isAmbiguousObject(items) {
					return ExtraMethodInfo{}, nil, fmt.Errorf("❌ SKIPPED: %s %s - Response is array of ambiguous objects (objects with no properties)", extraMethod.Method, extraMethod.Path)
				}

				// Skip if items are primitives
				if isPrimitive(items) {
					typeName := "unknown"
					if items.Type != nil && len(*items.Type) > 0 {
						typeName = string((*items.Type)[0])
					}
					return ExtraMethodInfo{}, nil, fmt.Errorf("❌ SKIPPED: %s %s - Response is array of primitives (%s). Cannot generate typed methods for primitive arrays", extraMethod.Method, extraMethod.Path, typeName)
				}

				// Note: We don't check hasAmbiguousNestedObjects for array items
				// because generateFieldsFromSchema already handles skipping ambiguous fields.
				// If SOME fields are ambiguous, they'll be skipped, but the rest will be generated.

				// Valid bare array with well-defined items - mark for special handling
				methodInfo.ReturnsArray = true
				// We'll generate the array item type later
			} else {
				// Only check for ambiguous objects/arrays if it's NOT a bare array response
				// (we already validated array items above)

				// CRITICAL: Check recursively for ambiguous objects in the schema tree
				// This catches cases where a valid schema has nested ambiguous objects (e.g., AsyncTaskInResponse with async_task: {type: object})
				if hasAmbiguousNestedObjects(schema.Value) {
					return ExtraMethodInfo{}, nil, fmt.Errorf("❌ SKIPPED: %s %s - Response schema contains ambiguous nested objects (objects with no properties). Method would return fields with core.Record type which is not allowed for typed methods", extraMethod.Method, extraMethod.Path)
				}

				if isAmbiguousArray(schema.Value) {
					return ExtraMethodInfo{}, nil, fmt.Errorf("❌ SKIPPED: %s %s - Response schema is ambiguous (array of objects with no properties). Method would return []core.Record which is not allowed for typed methods", extraMethod.Method, extraMethod.Path)
				}
			}
		}
	}

	// Generate response model fields from OpenAPI schema
	responseRegistry := NewTypeRegistry()

	// Handle bare array responses specially
	if methodInfo.ReturnsArray {
		// For GET requests, GetResponseModelSchema unwraps arrays, so we need the unwrapped item schema
		// For POST/other methods, we need to extract items from the array schema
		itemSchema, err := api.GetResponseModelSchema(extraMethod.Method, extraMethod.Path)
		if err == nil && itemSchema != nil && itemSchema.Value != nil {
			// Check if array item is a primitive type
			if isPrimitive(itemSchema.Value) {
				// For primitive types, use the Go type directly
				goType := mapOpenAPITypeToGo(itemSchema.Value)
				methodInfo.ArrayItemType = goType
				fmt.Printf("  ✅ Array of primitives detected: []%s\n", goType)
			} else {
				// Generate type name for array items
				// Don't use sanitizeNestedTypeName here - it's designed for schema paths, not clean method names
				itemTypeName := methodInfo.Name + "Item"

				// Generate fields for the array item type
				itemFields, err := generateFieldsFromSchema(itemSchema.Value, itemTypeName, responseRegistry, false, "EXTRA_METHOD_RESPONSE")
				if err == nil && len(itemFields) > 0 {
					// Register the item type as a nested type
					responseRegistry.RegisterType(itemTypeName, itemFields, "EXTRA_METHOD_RESPONSE")
					methodInfo.ArrayItemType = itemTypeName
					fmt.Printf("  ✅ Generated array response type: []%s\n", itemTypeName)
				} else {
					// If no fields generated, skip this method
					return ExtraMethodInfo{}, nil, fmt.Errorf("❌ SKIPPED: %s %s - Array item schema is ambiguous or empty", extraMethod.Method, extraMethod.Path)
				}
			}
		}
	} else if !methodInfo.ReturnsPrimitive {
		// Normal response (not an array, not a primitive)
		responseFields, err := generateModelFields(extraMethod.Path, extraMethod.Method, responseRegistry)
		if err != nil {
			// If schema not found, just skip response generation
			fmt.Printf("  ℹ️  No response schema found for %s %s\n", extraMethod.Method, extraMethod.Path)
		} else {
			methodInfo.ResponseFields = responseFields
		}
	}
	// For primitives, we don't need to generate response fields

	// Process nested types for both array and non-array responses
	responseNestedTypes := responseRegistry.GetTypes()
	if len(responseNestedTypes) > 0 {
		// Mark all response nested types with EXTRA_METHOD_RESPONSE section
		for _, nt := range responseNestedTypes {
			nt.Section = "EXTRA_METHOD_RESPONSE"
		}

		// Already sanitized for array responses, only sanitize for non-array
		if !methodInfo.ReturnsArray {
			// First pass: sanitize all nested type names
			oldToNew := make(map[string]string)
			for _, nt := range responseNestedTypes {
				oldName := nt.Name
				nt.Name = sanitizeNestedTypeName(nt.Name, methodInfo.Name)
				oldToNew[oldName] = nt.Name
			}

			// Second pass: update all field type references (including nested-nested types)
			for _, nt := range responseNestedTypes {
				for i := range nt.Fields {
					for oldName, newName := range oldToNew {
						nt.Fields[i].Type = strings.ReplaceAll(nt.Fields[i].Type, oldName, newName)
					}
				}
			}

			// Third pass: update response field type references
			for i := range methodInfo.ResponseFields {
				for oldName, newName := range oldToNew {
					methodInfo.ResponseFields[i].Type = strings.ReplaceAll(methodInfo.ResponseFields[i].Type, oldName, newName)
				}
			}
		}

		methodInfo.NestedTypes = append(methodInfo.NestedTypes, responseNestedTypes...)
		allNestedTypes = append(allNestedTypes, responseNestedTypes...)
	}

	return methodInfo, allNestedTypes, nil
}

// canSimplifyBody checks if body fields can be simplified to inline parameters
// Returns true if there are 1-3 fields and all are simple types
func canSimplifyBody(fields []Field) bool {
	if len(fields) == 0 || len(fields) > 3 {
		return false
	}

	for _, field := range fields {
		// Check if it's a simple type (not pointer, not slice, not map, not complex struct)
		t := strings.TrimPrefix(field.Type, "*")
		if strings.HasPrefix(t, "[]") || strings.HasPrefix(t, "map[") || strings.Contains(t, ".") {
			return false
		}
		// Allow only basic Go types
		simpleTypes := map[string]bool{
			"string": true, "int": true, "int32": true, "int64": true,
			"float32": true, "float64": true, "bool": true,
		}
		if !simpleTypes[t] {
			return false
		}
	}

	return true
}

// convertFieldsToSimplifiedParams converts Field structs to SimplifiedParam structs
func convertFieldsToSimplifiedParams(fields []Field) []SimplifiedParam {
	params := make([]SimplifiedParam, len(fields))

	for i, field := range fields {
		// Make parameter name camelCase (first letter lowercase)
		paramName := field.Name
		if len(paramName) > 0 {
			paramName = strings.ToLower(string(paramName[0])) + paramName[1:]
		}

		params[i] = SimplifiedParam{
			Name:        paramName,
			Type:        strings.TrimPrefix(field.Type, "*"), // Remove pointer if present
			BodyField:   field.JSONTag,
			Required:    field.RequiredTag == "true",
			Description: field.DocTag, // Copy description from OpenAPI spec
		}
	}

	// Sort: required first, then alphabetically
	sort.Slice(params, func(i, j int) bool {
		if params[i].Required != params[j].Required {
			return params[i].Required
		}
		return params[i].Name < params[j].Name
	})

	return params
}

// httpMethodToGoConstant converts HTTP method to Go http.Method constant
func httpMethodToGoConstant(method string) string {
	switch method {
	case "GET":
		return "MethodGet"
	case "POST":
		return "MethodPost"
	case "PUT":
		return "MethodPut"
	case "PATCH":
		return "MethodPatch"
	case "DELETE":
		return "MethodDelete"
	case "HEAD":
		return "MethodHead"
	case "OPTIONS":
		return "MethodOptions"
	default:
		// Capitalize first letter (replaces deprecated strings.Title)
		lower := strings.ToLower(method)
		if len(lower) > 0 {
			return "Method" + strings.ToUpper(lower[:1]) + lower[1:]
		}
		return "Method"
	}
}

// cleanPathPart removes {id} and other template variables from path part
func cleanPathPart(part string) string {
	// Remove template variables like {id}, {name}, etc.
	re := regexp.MustCompile(`\{[^}]+\}`)
	part = re.ReplaceAllString(part, "")
	// Remove trailing slashes
	part = strings.TrimSuffix(part, "/")
	return part
}

// sanitizeNestedTypeName removes path separators and brackets from nested type names
func sanitizeNestedTypeName(originalName, cleanBaseName string) string {
	// Extract the suffix after the path (e.g., "_S3PoliciesItem" from "/users/{id}/tenantData/Model_S3PoliciesItem")
	parts := strings.Split(originalName, "_")
	if len(parts) < 2 {
		return cleanBaseName + originalName
	}

	// Find where the actual field name starts (after the path parts)
	// The name format is usually: /path/{id}/subpath/Model_FieldName or /path/RequestBody_FieldName
	suffix := ""
	if strings.Contains(originalName, "Model_") {
		idx := strings.Index(originalName, "Model_")
		suffix = originalName[idx:]
	} else if strings.Contains(originalName, "RequestBody_") {
		idx := strings.Index(originalName, "RequestBody_")
		suffix = strings.Replace(originalName[idx:], "RequestBody_", "Body_", 1)
	} else if strings.Contains(originalName, "Body_") {
		idx := strings.Index(originalName, "Body_")
		suffix = originalName[idx:]
	} else {
		// If no Model_ or Body_ prefix, just use everything after last underscore
		suffix = "_" + parts[len(parts)-1]
	}

	return cleanBaseName + suffix
}

// deduplicateNestedTypes removes duplicate nested types by name
func deduplicateNestedTypes(types []*NestedType) []*NestedType {
	seen := make(map[string]bool)
	var result []*NestedType

	for _, t := range types {
		if !seen[t.Name] {
			seen[t.Name] = true
			result = append(result, t)
		}
	}

	// Sort by name for consistent generation order
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// ExtraMethodInfo represents information about an extra method
type ExtraMethodInfo struct {
	Name             string // Full name without HTTP method (e.g., "CnodeSetTenants")
	BaseName         string // Base name for templates (same as Name, kept for compatibility)
	HTTPMethod       string // HTTP method (e.g., "POST")
	GoHTTPMethod     string
	Path             string
	ResourcePath     string
	SubPath          string
	Summary          string
	HasID            bool
	HasParams        bool
	HasBody          bool
	BodyFields       []Field
	ResponseFields   []Field
	BodyTypeName     string
	ResponseTypeName string
	NestedTypes      []*NestedType
	// For simplified bodies (1-3 simple fields become inline parameters)
	SimplifiedBody   bool
	SimplifiedParams []SimplifiedParam
	// For 204 No Content responses
	ReturnsNoContent bool
	// For text/plain responses
	ReturnsTextPlain bool
	// For bare array responses (e.g., []Lock)
	ReturnsArray  bool
	ArrayItemType string // e.g., "Lock" for []Lock
	// For async task methods (returns AsyncTaskInResponse)
	IsAsyncTask bool
	// For primitive responses (e.g., string, int64, bool)
	ReturnsPrimitive bool
	PrimitiveType    string // Go type (e.g., "string", "int64", "bool", "float64")
}

// SimplifiedParam represents a simplified inline parameter for typed extra methods
type SimplifiedParam struct {
	Name        string // Go parameter name (e.g., "accessKey")
	Type        string // Go type (e.g., "string", "int64", "bool")
	BodyField   string // JSON field name (e.g., "access_key")
	Required    bool   // Whether the field is required
	Description string // Parameter description from OpenAPI spec
}

// returnsAsyncTaskInResponse checks if an operation returns AsyncTaskInResponse
func returnsAsyncTaskInResponse(method, resourcePath string) bool {
	rawResp, err := api.GetOpenApiResource(resourcePath)
	if err != nil || rawResp == nil {
		return false
	}

	var op *openapi3.Operation
	switch method {
	case "PATCH":
		op = rawResp.Patch
	case "PUT":
		op = rawResp.Put
	case "DELETE":
		op = rawResp.Delete
	case "POST":
		op = rawResp.Post
	default:
		return false
	}

	if op == nil {
		return false
	}

	// Only check 200 OK responses - async tasks return 200 with AsyncTaskInResponse
	// Methods returning 204 No Content are NOT async tasks
	if resp := op.Responses.Status(200); resp != nil && resp.Value != nil {
		// Must have application/json content
		if content := resp.Value.Content["application/json"]; content != nil && content.Schema != nil {
			// Check if response references AsyncTaskInResponse
			if content.Schema.Ref == "#/components/schemas/AsyncTaskInResponse" {
				return true
			}
		}
	}

	return false
}

// cleanIssueMessage removes emoji icons and verbose text from issue messages for generated file comments
func cleanIssueMessage(msg string) string {
	// Remove common emojis/icons used in error messages
	msg = strings.ReplaceAll(msg, "❌ SKIPPED: ", "")
	msg = strings.ReplaceAll(msg, "🔴 FATAL: ", "")
	msg = strings.ReplaceAll(msg, "⚠️  ", "")
	msg = strings.ReplaceAll(msg, "✅ ", "")
	msg = strings.ReplaceAll(msg, "ℹ️  ", "")

	// Remove verbose explanations that aren't needed in the generated file
	msg = strings.ReplaceAll(msg, ". Method would return core.Record which is not allowed for typed methods", "")
	msg = strings.ReplaceAll(msg, "Method would return core.Record which is not allowed for typed methods. ", "")
	msg = strings.ReplaceAll(msg, ". Method would return fields with core.Record type which is not allowed for typed methods", "")
	msg = strings.ReplaceAll(msg, "Method would return fields with core.Record type which is not allowed for typed methods. ", "")

	return msg
}

// ResourceData represents data for template generation
type ResourceData struct {
	Name                string
	LowerName           string
	PluralName          string
	SearchParamsFields  []Field
	DetailsModelFields  []Field           // From searchQuery response (GET/PATCH response)
	RequestBodyFields   []Field           // From createQuery request body
	UpsertModelFields   []Field           // From createQuery response (POST/PUT/PATCH response)
	DeleteQueryParams   []SimplifiedParam // DELETE query parameters as simplified function args
	DeleteBodyParams    []SimplifiedParam // DELETE body parameters as simplified function args
	DeleteIdDescription string            // DELETE id parameter description from path
	ExtraMethods        []ExtraMethodInfo
	NestedTypes         []*NestedType
	Resource            *vastparser.VastResource
	HasSearchParams     bool // True if SearchParamsFields is not empty
	HasAsyncMethods     bool // True if any extra method is an async task
	ReturnsTextPlain    bool // True if GET operation returns text/plain instead of JSON
	HasTextPlainMethods bool // True if any method (main or extra) returns text/plain
	HasPrimitiveMethods bool // True if any extra method returns a primitive type
	HasArrayMethods     bool // True if any extra method returns an array
	// CRUD operation flags - determine which methods to generate
	HasCreate   bool   // True if C is in ops marker (or legacy upsert:POST)
	HasList     bool   // True if L is in ops marker - enables List, Get, Exists, MustExists methods
	HasRead     bool   // True if R is in ops marker (or legacy details:GET) - enables GetById method
	HasUpdate   bool   // True if U is in ops marker (or legacy upsert:PUT/PATCH)
	HasDelete   bool   // True if D is in ops marker
	TemplateOps string // CRUD operations string for template header (e.g., "CREATE|LIST|READ|UPDATE|DELETE")
	// Type alias optimization - use aliases for direct component references
	DetailsModelIsAlias      bool   // True if DetailsModel is a type alias to a component
	DetailsModelAlias        string // Component name to alias (e.g., "Component_ActiveDirectory")
	DetailsModelComponentRef string // OpenAPI component reference (e.g., "#/components/schemas/ActiveDirectory")
	UpsertModelIsAlias       bool   // True if UpsertModel/EditModel is a type alias to a component
	UpsertModelAlias         string // Component name to alias (e.g., "Component_ActiveDirectory")
	UpsertModelComponentRef  string // OpenAPI component reference (e.g., "#/components/schemas/ActiveDirectory")
	// Edit model for UPDATE-only resources (no CREATE)
	UseEditModel          bool              // True if we should generate EditModel instead of UpsertModel (UPDATE without CREATE)
	UpdateInlineParams    []SimplifiedParam // Inline parameters for Update when body has ≤4 params
	HasUpdateInlineParams bool              // True if Update should use inline params instead of RequestBody
	HasRequestBodyContent bool              // True if REQUEST BODY section has any content (nested types or RequestBody struct)
	GetSummary            string            // Summary for GET/List operations
	GetByIdSummary        string            // Summary for GetById operation
	CreateSummary         string            // Summary for Create operation
	UpdateSummary         string            // Summary for Update operation
	DeleteSummary         string            // Summary for Delete operation
	// Async task support for CRUD operations
	UpdateIsAsync     bool     // True if Update operation returns AsyncTaskInResponse
	DeleteIsAsync     bool     // True if Delete operation returns AsyncTaskInResponse
	DeleteByIdIsAsync bool     // True if DeleteById operation returns AsyncTaskInResponse
	GenerationIssues  []string // List of issues encountered during generation
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
	inputDir := "../resources/untyped"
	outputDir := "../resources/typed"
	restConfigFile := "../rest/untyped_rest.go"

	// STEP 1: Parse rest/untyped_rest.go to get CRUD configurations
	fmt.Println("Parsing rest/untyped_rest.go for CRUD configurations...")
	restParser := vastparser.NewRestParser()
	if err := restParser.ParseRestFile(restConfigFile); err != nil {
		log.Fatalf("Failed to parse rest configuration file: %v", err)
	}

	configs := restParser.GetAllConfigs()
	fmt.Printf("Found %d resource configurations in rest/untyped_rest.go\n", len(configs))

	// STEP 2: Parse the input directory to find resources with APITyped markers
	parser := vastparser.NewVastResourceParser()
	resources, err := parser.ParseDirectory(inputDir)
	if err != nil {
		log.Fatalf("Failed to parse file: %v", err)
	}

	// STEP 3: Augment resources with CRUD operations from rest config
	// Also track which configs have been used
	usedConfigs := make(map[string]bool)

	for i := range resources {
		resource := &resources[i]

		// If resource doesn't have operations marker, try to get from rest config
		if resource.Operations == nil {
			if config, exists := configs[resource.Name]; exists {
				resource.Operations = config.ConvertToOperations()
				usedConfigs[resource.Name] = true
				fmt.Printf("  ✅ %s: Using CRUD config from rest/untyped_rest.go: %s\n", resource.Name, config.Operations)

				// Also merge extra methods from rest config
				if len(config.ExtraMethods) > 0 {
					resource.ExtraMethods = append(resource.ExtraMethods, config.ExtraMethods...)
					fmt.Printf("  ✅ %s: Merged %d extra methods from rest/untyped_rest.go\n", resource.Name, len(config.ExtraMethods))
				}
			} else {
				// Fallback to legacy markers if present
				if len(resource.Details) > 0 && len(resource.Upserts) > 0 {
					resource.Operations = &apibuilder.Operations{
						Operations: "CRUD",
						URL:        resource.Details[0].URL,
					}
					fmt.Printf("  ⚠️  %s: Using legacy markers (CRUD assumed)\n", resource.Name)
				} else if len(resource.Details) > 0 {
					resource.Operations = &apibuilder.Operations{
						Operations: "R",
						URL:        resource.Details[0].URL,
					}
					fmt.Printf("  ⚠️  %s: Using legacy markers (R only)\n", resource.Name)
				} else {
					fmt.Printf("  ❌ %s: No CRUD configuration found - skipping\n", resource.Name)
					continue
				}
			}
		} else {
			usedConfigs[resource.Name] = true
			fmt.Printf("  ✅ %s: Using ops marker: %s\n", resource.Name, resource.Operations.Operations)
		}
	}

	// STEP 4: Add resources from rest config that weren't found by parser
	// This handles resources that don't have any markers in their files
	for resourceName, config := range configs {
		if !usedConfigs[resourceName] {
			// Skip resources with empty resource paths (e.g., Dummy, test resources)
			if config.ResourcePath == "" {
				fmt.Printf("  ⏭️  %s: Skipping (empty resource path)\n", resourceName)
				continue
			}

			// Create a new resource from the rest config
			newResource := vastparser.VastResource{
				Name:         resourceName,
				Operations:   config.ConvertToOperations(),
				ExtraMethods: config.ExtraMethods, // Include extra methods from rest config
			}
			resources = append(resources, newResource)
			fmt.Printf("  ✅ %s: Auto-generated from rest/untyped_rest.go (no markers found): %s\n", resourceName, config.Operations)
			if len(config.ExtraMethods) > 0 {
				fmt.Printf("  ✅ %s: Including %d extra methods from rest/untyped_rest.go\n", resourceName, len(config.ExtraMethods))
			}
		}
	}

	// Filter out resources without operations
	var validResources []vastparser.VastResource
	for _, resource := range resources {
		if resource.Operations != nil {
			validResources = append(validResources, resource)
		}
	}
	resources = validResources

	if len(resources) == 0 {
		log.Println("No resources with CRUD configuration found")
		return
	}

	// STEP 5: Validate CRUD operations against OpenAPI schemas
	// Exclude operations if they don't have valid response schemas (unless 204 NO CONTENT)
	// Track excluded operations as generation issues for each resource
	excludedOpsMap := make(map[string][]string) // resourceName -> []excludedOps

	for i := range resources {
		resource := &resources[i]
		if resource.Operations == nil {
			continue
		}

		originalOps := resource.Operations.Operations
		excludedOps := []string{}

		// Check CREATE operation (C)
		if resource.Operations.HasCreate() {
			createURL := resource.GetOperationsURL()
			if createURL != "" {
				// Check if POST has valid response schema or returns 204
				if !isCreateResponseValid(createURL) {
					// Remove C from operations
					resource.Operations.Operations = strings.ReplaceAll(resource.Operations.Operations, "C", "")
					issue := fmt.Sprintf("CREATE operation excluded: POST %s has no response schema and doesn't return 204 NO CONTENT", createURL)
					excludedOps = append(excludedOps, "C (CREATE has no response schema and doesn't return 204 NO CONTENT)")
					excludedOpsMap[resource.Name] = append(excludedOpsMap[resource.Name], issue)
				}
			}
		}

		// Check LIST operation (L)
		if resource.Operations.HasList() {
			listURL := resource.GetOperationsURL()
			if listURL != "" {
				// Check if LIST returns array of ambiguous objects
				if isListResponseAmbiguous(listURL) {
					// Remove L from operations
					resource.Operations.Operations = strings.ReplaceAll(resource.Operations.Operations, "L", "")
					issue := fmt.Sprintf("LIST operation excluded: GET %s returns array of objects without properties", listURL)
					excludedOps = append(excludedOps, "L (LIST returns array of objects without properties)")
					excludedOpsMap[resource.Name] = append(excludedOpsMap[resource.Name], issue)
				}
			}
		}

		// Check READ operation (R)
		if resource.Operations.HasRead() {
			readURL := resource.GetOperationsURL()
			if readURL != "" {
				// Add /{id}/ for the read by ID endpoint
				readByIdURL := "/" + strings.Trim(readURL, "/") + "/{id}/"
				// Check if READ returns ambiguous object
				if isReadResponseAmbiguous(readByIdURL) {
					// Remove R from operations
					resource.Operations.Operations = strings.ReplaceAll(resource.Operations.Operations, "R", "")
					issue := fmt.Sprintf("READ operation excluded: GET %s returns object without properties", readByIdURL)
					excludedOps = append(excludedOps, "R (READ returns object without properties)")
					excludedOpsMap[resource.Name] = append(excludedOpsMap[resource.Name], issue)
				}
			}
		}

		// Check UPDATE operation (U)
		if resource.Operations.HasUpdate() {
			updateURL := resource.GetOperationsURL()
			if updateURL != "" {
				// Add /{id}/ for the update endpoint
				updateByIdURL := "/" + strings.Trim(updateURL, "/") + "/{id}/"
				// Check if PATCH/PUT has valid response schema or returns 204
				if !isUpdateResponseValid(updateByIdURL) {
					// Remove U from operations
					resource.Operations.Operations = strings.ReplaceAll(resource.Operations.Operations, "U", "")
					issue := fmt.Sprintf("UPDATE operation excluded: PATCH/PUT %s has no response schema and doesn't return 204 NO CONTENT", updateByIdURL)
					excludedOps = append(excludedOps, "U (UPDATE has no response schema and doesn't return 204 NO CONTENT)")
					excludedOpsMap[resource.Name] = append(excludedOpsMap[resource.Name], issue)
				}
			}
		}

		// Log any excluded operations
		if len(excludedOps) > 0 {
			fmt.Printf("  ⚠️  %s: Excluded operations: %s (schema validation failed)\n", resource.Name, strings.Join(excludedOps, ", "))
			fmt.Printf("      Operations changed from %s to %s\n", originalOps, resource.Operations.Operations)
		}
	}

	// Sort resources by name for consistent generation order
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Name < resources[j].Name
	})

	fmt.Printf("\nGenerating %d typed resources:\n", len(resources))
	for _, resource := range resources {
		fmt.Printf("  - %s (%s)\n", resource.Name, resource.Operations.Operations)
	}

	// Generate template data
	templateData := TemplateData{}
	for _, resource := range resources {
		// Print resource header
		fmt.Printf("\n%s:\n", resource.Name)

		// Validate required markers
		if err := validateResourceMarkers(&resource); err != nil {
			fmt.Printf("  ❌ Error: Resource validation failed: %v\n", err)
			continue
		}

		resourceData := ResourceData{
			Name:       resource.Name,
			LowerName:  strings.ToLower(resource.Name),
			PluralName: pluralize(resource.Name),
			Resource:   &resource,
		}

		// Add excluded operations as generation issues
		if excludedIssues, exists := excludedOpsMap[resource.Name]; exists {
			resourceData.GenerationIssues = append(resourceData.GenerationIssues, excludedIssues...)
		}

		// Create separate registries for each generation phase to avoid type name conflicts
		searchRegistry := NewTypeRegistry()
		requestRegistry := NewTypeRegistry()

		// Generate search params fields
		var searchFields []Field

		// Determine search method and URL from Operations marker or legacy details marker
		var searchMethod string
		var searchURL string

		if resource.HasOperations() {
			// Use Operations marker URL for read operations (R flag) or list operations (L flag)
			if resource.Operations.HasRead() || resource.Operations.HasList() {
				searchMethod = "GET"
				searchURL = resource.GetOperationsURL()
			}
		} else {
			// Fall back to legacy details markers
			if resource.HasDetails("GET") {
				searchMethod = "GET"
				searchURL = resource.GetDetails("GET")
			} else if resource.HasDetails("PATCH") {
				searchMethod = "PATCH"
				searchURL = resource.GetDetails("PATCH")
			}
		}

		// Generate SearchParams from OpenAPI query parameters
		if searchURL != "" {
			fields, err := generateSearchParamsFields(searchURL, searchMethod, searchRegistry)
			if err != nil {
				fmt.Printf("  ⚠️  Warning: Failed to generate search params fields: %v\n", err)
			} else {
				searchFields = fields
			}
		}

		// Add common searchable fields from response body if they exist
		commonFields, err := extractCommonSearchableFields(&resource, searchRegistry)
		if err != nil {
			fmt.Printf("  ⚠️  Warning: Failed to extract common searchable fields: %v\n", err)
		} else {
			searchFields = mergeSearchFields(searchFields, commonFields)
		}

		// Collect all nested types from all registries
		var allNestedTypes []*NestedType
		allNestedTypes = append(allNestedTypes, searchRegistry.GetTypes()...)

		if searchURL != "" {
			// Check if the response is text/plain instead of JSON
			isTextPlain, err := api.ReturnsTextPlain(searchMethod, searchURL)
			if err != nil {
				fmt.Printf("  ⚠️  Warning: Failed to check if response is text/plain for %s %s: %v\n", searchMethod, searchURL, err)
			}

			if isTextPlain {
				resourceData.ReturnsTextPlain = true
				resourceData.HasTextPlainMethods = true
				fmt.Printf("  ℹ️  Response is text/plain, will extract from @raw key\n")
			} else if err == nil {
				// Generate DetailsModel from details response schema (only for JSON responses)
				// Check if response is a direct component reference (for alias optimization)
				detailsSchemaRef, detailsSchemaErr := api.GetResponseModelSchemaUnresolved(searchMethod, searchURL)
				if detailsSchemaErr == nil && detailsSchemaRef != nil {
					// Check for direct component reference
					if componentName := api.IsDirectComponentReference(detailsSchemaRef); componentName != "" {
						// Verify the component is not ambiguous before aliasing
						componentSchema, compErr := api.GetSchemaFromComponent(componentName)
						if compErr == nil && componentSchema != nil && componentSchema.Value != nil {
							if !isAmbiguousObject(componentSchema.Value) && !isPrimitive(componentSchema.Value) {
								// Direct component reference to a valid (non-ambiguous) component - use type alias
								resourceData.DetailsModelIsAlias = true
								resourceData.DetailsModelAlias = "Component_" + componentName
								resourceData.DetailsModelComponentRef = "#/components/schemas/" + componentName
								fmt.Printf("  ✅ DetailsModel is alias to %s\n", resourceData.DetailsModelAlias)
							}
						}
					} else if detailsSchemaRef.Value != nil && detailsSchemaRef.Value.Type != nil && len(*detailsSchemaRef.Value.Type) > 0 {
						// Check for array of component references (e.g., type: array, items: {$ref: #/components/schemas/...})
						if (*detailsSchemaRef.Value.Type)[0] == "array" && detailsSchemaRef.Value.Items != nil {
							if componentName := api.IsDirectComponentReference(detailsSchemaRef.Value.Items); componentName != "" {
								// Verify the component is not ambiguous before aliasing
								componentSchema, compErr := api.GetSchemaFromComponent(componentName)
								if compErr == nil && componentSchema != nil && componentSchema.Value != nil {
									if !isAmbiguousObject(componentSchema.Value) && !isPrimitive(componentSchema.Value) {
										// Array of component references - use type alias for the array item
										resourceData.DetailsModelIsAlias = true
										resourceData.DetailsModelAlias = "Component_" + componentName
										resourceData.DetailsModelComponentRef = "#/components/schemas/" + componentName
										fmt.Printf("  ✅ DetailsModel is alias to %s (from array items)\n", resourceData.DetailsModelAlias)
									}
								}
							}
						}
					}
				}

				// Only generate full struct if NOT an alias
				if !resourceData.DetailsModelIsAlias {
					detailsRegistry := NewTypeRegistry()
					detailsFields, err := generateModelFields(searchURL, searchMethod, detailsRegistry)
					if err != nil {
						fmt.Printf("  ⚠️  Warning: Failed to generate details model fields from details marker: %v\n", err)
					} else if len(detailsFields) == 0 {
						// Skip DetailsModel if no fields were generated (likely ambiguous object with no properties)
						// Also exclude L and R operations since typed methods require typed models
						fmt.Printf("  ⚠️  Warning: DetailsModel has no fields (ambiguous schema), skipping model generation and excluding L/R operations\n")
						resourceData.GenerationIssues = append(resourceData.GenerationIssues,
							fmt.Sprintf("DetailsModel skipped: Response schema is ambiguous (object with no properties) - %s %s. LIST and READ operations excluded", searchMethod, searchURL))
						// Force L and R to be excluded
						if resource.HasOperations() {
							resource.Operations.Operations = strings.ReplaceAll(resource.Operations.Operations, "L", "")
							resource.Operations.Operations = strings.ReplaceAll(resource.Operations.Operations, "R", "")
						}
						// Don't set DetailsModelFields, which will prevent the model from being generated
					} else {
						resourceData.DetailsModelFields = detailsFields
						// Fallback: If no search params were found, use DetailsModel fields as SearchParams
						if len(searchFields) == 0 && len(detailsFields) > 0 {
							fmt.Printf("  ℹ️  No search params found, using DetailsModel fields as SearchParams\n")
							searchFields = detailsFields
						}
						// Add details model nested types
						allNestedTypes = append(allNestedTypes, detailsRegistry.GetTypes()...)
					}
				}
			}
		}

		resourceData.SearchParamsFields = searchFields
		resourceData.HasSearchParams = len(searchFields) > 0

		// Declare createURL/createMethod outside the block so we can use it for summaries
		var createMethod string
		var createURL string

		// Generate RequestBody and UpsertModel from Operations marker or legacy upsert marker
		if !resource.IsReadOnly() {
			if resource.HasOperations() {
				// Use Operations marker URL for create/update operations
				if resource.Operations.HasCreate() || resource.Operations.HasUpdate() {
					// Prefer POST for create, otherwise use PATCH for update
					if resource.Operations.HasCreate() {
						createMethod = "POST"
						createURL = resource.GetOperationsURL()
					} else {
						// UPDATE-only (PATCH without CREATE) - use /{id}/ path
						createMethod = "PATCH"
						baseURL := resource.GetOperationsURL()
						createURL = "/" + strings.Trim(baseURL, "/") + "/{id}/"
					}
				}
			} else {
				// Fall back to legacy upsert markers
				if resource.HasUpsert("POST") {
					createMethod = "POST"
					createURL = resource.GetUpsert("POST")
				} else if resource.HasUpsert("PUT") {
					createMethod = "PUT"
					createURL = resource.GetUpsert("PUT")
				} else if resource.HasUpsert("PATCH") {
					createMethod = "PATCH"
					createURL = resource.GetUpsert("PATCH")
				}
			}

			if createURL != "" {
				// Generate RequestBody from upsert request body schema
				requestFields, err := generateRequestBodyFields(createURL, createMethod, requestRegistry)
				if err != nil {
					fmt.Printf("  ⚠️  Warning: Failed to generate request body fields from upsert marker: %v\n", err)
				} else if len(requestFields) == 0 {
					// Skip RequestBody if no fields were generated (likely ambiguous object with no properties)
					// Also exclude C/U operations since typed methods require typed models
					fmt.Printf("  ⚠️  Warning: RequestBody has no fields (ambiguous schema), skipping model generation and excluding C/U operations\n")
					resourceData.GenerationIssues = append(resourceData.GenerationIssues,
						fmt.Sprintf("RequestBody skipped: Request schema is ambiguous (object with no properties) - %s %s. CREATE and UPDATE operations excluded", createMethod, createURL))
					// Force C and U to be excluded
					if resource.HasOperations() {
						resource.Operations.Operations = strings.ReplaceAll(resource.Operations.Operations, "C", "")
						resource.Operations.Operations = strings.ReplaceAll(resource.Operations.Operations, "U", "")
					}
				} else {
					// Check if we should use inline params for UPDATE (≤4 params and PATCH/PUT method)
					if (createMethod == "PATCH" || createMethod == "PUT") && len(requestFields) > 0 && len(requestFields) <= 4 {
						// Convert fields to inline params for Update methods
						resourceData.HasUpdateInlineParams = true
						resourceData.UpdateInlineParams = make([]SimplifiedParam, 0, len(requestFields))
						for _, field := range requestFields {
							resourceData.UpdateInlineParams = append(resourceData.UpdateInlineParams, SimplifiedParam{
								Name:        strings.ToLower(field.Name[:1]) + field.Name[1:], // camelCase
								Type:        field.Type,
								BodyField:   field.JSONTag,
								Required:    field.RequiredTag == "true",
								Description: field.DocTag,
							})
						}
						fmt.Printf("  ℹ️  Update will use inline parameters (%d params)\n", len(requestFields))
						// Don't set RequestBodyFields - we'll use inline params instead
					} else {
						resourceData.RequestBodyFields = requestFields
					}
				}
				// Always add request body nested types, even if using inline params or empty fields
				allNestedTypes = append(allNestedTypes, requestRegistry.GetTypes()...)

				// Generate UpsertModel (or EditModel for UPDATE-only) from upsert response schema
				// Check if response is a direct component reference (for alias optimization)
				upsertSchemaRef, upsertSchemaErr := api.GetResponseModelSchemaUnresolved(createMethod, createURL)
				if upsertSchemaErr == nil && upsertSchemaRef != nil {
					if componentName := api.IsDirectComponentReference(upsertSchemaRef); componentName != "" {
						// Verify the component is not ambiguous before aliasing
						componentSchema, compErr := api.GetSchemaFromComponent(componentName)
						if compErr == nil && componentSchema != nil && componentSchema.Value != nil {
							if !isAmbiguousObject(componentSchema.Value) && !isPrimitive(componentSchema.Value) {
								// Direct component reference to a valid (non-ambiguous) component - use type alias
								resourceData.UpsertModelIsAlias = true
								resourceData.UpsertModelAlias = "Component_" + componentName
								resourceData.UpsertModelComponentRef = "#/components/schemas/" + componentName
								fmt.Printf("  ✅ UpsertModel is alias to %s\n", resourceData.UpsertModelAlias)
							}
						}
					}
				}

				// Only generate full struct if NOT an alias
				if !resourceData.UpsertModelIsAlias {
					upsertRegistry := NewTypeRegistry()
					upsertFields, err := generateModelFields(createURL, createMethod, upsertRegistry)
					if err != nil {
						// CRITICAL: Check if this is an UPDATE operation with missing response schema
						// Missing schemas are only acceptable for 204 NO CONTENT responses
						if createMethod == "PATCH" || createMethod == "PUT" {
							returns204, checkErr := api.Returns204NoContent(createMethod, createURL)
							if checkErr != nil || !returns204 {
								// This is a broken UPDATE operation - response should have a schema but doesn't
								issueMsg := fmt.Sprintf("UPDATE operation is broken for %s %s: Response schema is missing but status is not 204 NO CONTENT. UPDATE operations MUST return a valid response schema or 204 status. Error: %v", createMethod, createURL, err)
								resourceData.GenerationIssues = append(resourceData.GenerationIssues, cleanIssueMessage(issueMsg))

								fmt.Printf("  🔴 FATAL: UPDATE operation is broken for %s %s\n", createMethod, createURL)
								fmt.Printf("          Response schema is missing but status is not 204 NO CONTENT.\n")
								fmt.Printf("          UPDATE operations MUST return a valid response schema or 204 status.\n")
								fmt.Printf("          Error: %v\n", err)
								fmt.Printf("  🔧 FIX: Excluding UPDATE from template. Effective operations: ops:%s%s%s\n",
									func() string {
										if resource.HasOperations() && resource.Operations.HasCreate() {
											return "C"
										}
										return ""
									}(),
									func() string {
										if resource.HasOperations() && resource.Operations.HasRead() {
											return "R"
										}
										return ""
									}(),
									func() string {
										if resource.HasOperations() && resource.Operations.HasDelete() {
											return "D"
										}
										return ""
									}())
								// Force UPDATE to be excluded
								if resource.HasOperations() {
									resource.Operations.Operations = strings.ReplaceAll(resource.Operations.Operations, "U", "")
								}
							}
						} else {
							fmt.Printf("  ⚠️  Warning: Failed to generate upsert model fields from upsert marker: %v\n", err)
						}
					} else if len(upsertFields) == 0 {
						// Skip UpsertModel if no fields were generated (likely ambiguous object with no properties)
						fmt.Printf("  ⚠️  Warning: UpsertModel/EditModel has no fields (ambiguous schema), skipping model generation\n")
						resourceData.GenerationIssues = append(resourceData.GenerationIssues,
							fmt.Sprintf("UpsertModel/EditModel skipped: Response schema is ambiguous (object with no properties) - %s %s", createMethod, createURL))
					} else {
						resourceData.UpsertModelFields = upsertFields
						// Add upsert model nested types
						allNestedTypes = append(allNestedTypes, upsertRegistry.GetTypes()...)
					}
				}
			}
		}

		// Generate extra methods from apityped:extraMethod markers
		for _, extraMethod := range resource.ExtraMethods {
			extraMethodInfo, extraMethodNested, err := generateExtraMethodInfo(resource.Name, extraMethod)
			if err != nil {
				// Check for FATAL errors (invalid markers) - these must be fixed
				if strings.Contains(err.Error(), "🔴 FATAL") {
					fmt.Printf("\n%v\n\n", err)
					log.Fatalf("Generation stopped due to fatal error. Please fix the marker and try again.")
				}
				// Collect issue for display in generated file
				issueMsg := fmt.Sprintf("Extra method %s %s skipped: %v", extraMethod.Method, extraMethod.Path, err)
				resourceData.GenerationIssues = append(resourceData.GenerationIssues, cleanIssueMessage(issueMsg))

				// Check if this is an ambiguous schema error (requires special handling)
				if strings.Contains(err.Error(), "❌ SKIPPED") {
					fmt.Printf("  %v\n", err)
				} else {
					fmt.Printf("  ⚠️  Warning: Failed to generate extra method %s %s: %v\n", extraMethod.Method, extraMethod.Path, err)
				}
				continue
			}
			resourceData.ExtraMethods = append(resourceData.ExtraMethods, extraMethodInfo)
			allNestedTypes = append(allNestedTypes, extraMethodNested...)

			// Check if this extra method returns text/plain
			if extraMethodInfo.ReturnsTextPlain {
				resourceData.HasTextPlainMethods = true
			}
			// Check if this extra method returns a primitive type
			if extraMethodInfo.ReturnsPrimitive {
				resourceData.HasPrimitiveMethods = true
			}

			// Check if this extra method returns an array
			if extraMethodInfo.ReturnsArray {
				resourceData.HasArrayMethods = true
			}

			// Check if this is an async method
			if extraMethodInfo.IsAsyncTask {
				resourceData.HasAsyncMethods = true
			}

			fmt.Printf("  ✅ Generated extra method: %s\n", extraMethodInfo.Name)
		}

		// Sort extra methods by name+method for deterministic output
		sort.Slice(resourceData.ExtraMethods, func(i, j int) bool {
			iKey := resourceData.ExtraMethods[i].Name + "_" + resourceData.ExtraMethods[i].HTTPMethod
			jKey := resourceData.ExtraMethods[j].Name + "_" + resourceData.ExtraMethods[j].HTTPMethod
			return iKey < jKey
		})

		// Extract summaries for main CRUD operations
		if searchURL != "" {
			// GET/List summary
			if summary, err := api.GetOperationSummary("GET", searchURL); err == nil {
				resourceData.GetSummary = summary
			}
			// GET by ID summary (use searchURL + /{id}/)
			getByIdPath := "/" + strings.Trim(searchURL, "/") + "/{id}/"
			if summary, err := api.GetOperationSummary("GET", getByIdPath); err == nil {
				resourceData.GetByIdSummary = summary
			}
		}
		if createURL != "" {
			// Create summary (POST)
			if summary, err := api.GetOperationSummary("POST", createURL); err == nil {
				resourceData.CreateSummary = summary
			}
			// Update summary (PATCH or PUT)
			if resource.HasUpsert("PATCH") {
				// For PATCH, use /{id}/ path
				patchPath := "/" + strings.Trim(createURL, "/") + "/{id}/"
				if summary, err := api.GetOperationSummary("PATCH", patchPath); err == nil {
					resourceData.UpdateSummary = summary
				}
			} else if resource.HasUpsert("PUT") {
				putPath := "/" + strings.Trim(createURL, "/") + "/{id}/"
				if summary, err := api.GetOperationSummary("PUT", putPath); err == nil {
					resourceData.UpdateSummary = summary
				}
			}
			// Delete summary and parameters
			deletePath := "/" + strings.Trim(createURL, "/") + "/{id}/"
			if summary, err := api.GetOperationSummary("DELETE", deletePath); err == nil {
				resourceData.DeleteSummary = summary
			}

			// Extract DELETE parameters (query params and body params)
			if deleteParams, err := api.GetDeleteParams(createURL); err == nil {
				// Store id parameter description
				resourceData.DeleteIdDescription = deleteParams.IdDescription
				if deleteParams.IdDescription != "" {
					fmt.Printf("  ✅ Added DELETE id description: %s\n", deleteParams.IdDescription)
				}

				// Convert query parameters to simplified params
				for _, paramRef := range deleteParams.QueryParams {
					if paramRef.Value == nil {
						continue
					}
					param := paramRef.Value
					// Convert to camelCase with lowercase first letter for function parameters
					paramName := toCamelCase(param.Name)
					if len(paramName) > 0 {
						paramName = strings.ToLower(string(paramName[0])) + paramName[1:]
					}
					simplified := SimplifiedParam{
						Name:        paramName, // e.g., "force" → "force", "skip_ldap" → "skipLdap"
						Type:        getGoTypeFromOpenAPI(param.Schema.Value, param.Required),
						BodyField:   param.Name, // Original field name for JSON
						Required:    param.Required,
						Description: param.Description,
					}
					resourceData.DeleteQueryParams = append(resourceData.DeleteQueryParams, simplified)
					fmt.Printf("  ✅ Added DELETE query param: %s (%s) - %s\n", simplified.Name, simplified.Type, simplified.Description)
				}

				// Convert body schema properties to simplified params
				if deleteParams.BodySchema != nil && deleteParams.BodySchema.Value != nil {
					schema := deleteParams.BodySchema.Value
					if schema.Properties != nil {
						for propName, propRef := range schema.Properties {
							if propRef == nil || propRef.Value == nil {
								continue
							}

							// Check if required
							isRequired := false
							for _, req := range schema.Required {
								if req == propName {
									isRequired = true
									break
								}
							}

							// Convert to camelCase with lowercase first letter for function parameters
							paramName := toCamelCase(propName)
							if len(paramName) > 0 {
								paramName = strings.ToLower(string(paramName[0])) + paramName[1:]
							}

							simplified := SimplifiedParam{
								Name:        paramName, // e.g., "skip_ldap" → "skipLdap"
								Type:        getGoTypeFromOpenAPI(propRef.Value, isRequired),
								BodyField:   propName, // Original field name for JSON
								Required:    isRequired,
								Description: propRef.Value.Description,
							}
							resourceData.DeleteBodyParams = append(resourceData.DeleteBodyParams, simplified)
							fmt.Printf("  ✅ Added DELETE body param: %s (%s) - %s\n", simplified.Name, simplified.Type, simplified.Description)
						}
					}
				}
			} else {
				fmt.Printf("  ℹ️  No special DELETE parameters found\n")
			}
		}

		// Deduplicate nested types (same type may appear in multiple models)
		resourceData.NestedTypes = deduplicateNestedTypes(allNestedTypes)

		// Sort generation issues for deterministic output (avoid unnecessary git diffs)
		sort.Strings(resourceData.GenerationIssues)

		templateData.Resources = append(templateData.Resources, resourceData)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Generate separate files for each resource
	var generatedFiles []string
	for _, resourceData := range templateData.Resources {
		// Check if REQUEST BODY section has any content
		hasRequestBodyNestedTypes := false
		for _, nt := range resourceData.NestedTypes {
			if nt.Section == "REQUEST BODY" {
				hasRequestBodyNestedTypes = true
				break
			}
		}
		resourceData.HasRequestBodyContent = hasRequestBodyNestedTypes || (len(resourceData.RequestBodyFields) > 0 && !resourceData.HasUpdateInlineParams)

		resourceFile := filepath.Join(outputDir, strings.ToLower(resourceData.Name)+"_autogen.go")
		if err := generateResourceFile(resourceFile, resourceData); err != nil {
			log.Fatalf("Failed to generate %s: %v", resourceFile, err)
		}
		generatedFiles = append(generatedFiles, strings.ToLower(resourceData.Name)+"_autogen.go")
	}

	fmt.Printf("Generated typed resources for %d resources in %s/\n", len(resources), outputDir)
	for _, file := range generatedFiles {
		fmt.Printf("  - %s: Typed resource implementation\n", file)
	}

	// Generate common_components.go with all OpenAPI component schemas
	fmt.Printf("\nGenerating common components from OpenAPI spec...\n")
	componentsFile := filepath.Join(outputDir, "common_components.go")
	if err := generateCommonComponentsFile(componentsFile); err != nil {
		log.Fatalf("Failed to generate common_components.go: %v", err)
	}
	fmt.Printf("✅ Generated common_components.go with reusable OpenAPI schemas\n")

	// Format all generated Go files
	if err := formatGeneratedFiles(outputDir); err != nil {
		log.Printf("Warning: Failed to format generated files: %v", err)
	} else {
		fmt.Printf("Formatted all generated Go files with go fmt\n")
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
	// Choose template based on resource type
	// Support both new Operations marker and legacy markers
	var hasList, hasRead, hasCreate, hasUpdate, hasDelete bool

	if data.Resource.HasOperations() {
		// Use new Operations marker
		hasList = data.Resource.Operations.HasList()
		hasRead = data.Resource.Operations.HasRead()
		hasCreate = data.Resource.Operations.HasCreate()
		hasUpdate = data.Resource.Operations.HasUpdate()
		hasDelete = data.Resource.Operations.HasDelete()
	} else {
		// Fall back to legacy markers
		// Legacy: R meant both List and GetById
		hasList = data.Resource.HasDetails("GET") || data.Resource.HasDetails("PATCH")
		hasRead = data.Resource.HasDetails("GET") || data.Resource.HasDetails("PATCH")
		hasCreate = data.Resource.HasUpsert("POST")
		hasUpdate = data.Resource.HasUpsert("PUT") || data.Resource.HasUpsert("PATCH")
		hasDelete = true // Legacy resources always had delete
	}

	// CRITICAL: If DetailsModel is missing (no alias, no fields), exclude List and Read operations
	// Typed methods require typed models
	if hasList || hasRead {
		if !data.DetailsModelIsAlias && len(data.DetailsModelFields) == 0 {
			fmt.Printf("  🔴 Excluding LIST and READ operations: DetailsModel is missing (ambiguous schema)\n")
			hasList = false
			hasRead = false
		}
	}

	// CRITICAL: If RequestBody or UpsertModel is missing, exclude Create and Update operations
	// Typed methods require both typed request models (input) and response models (output)
	hasRequestBody := len(data.RequestBodyFields) > 0 || data.HasUpdateInlineParams
	hasResponseModel := data.UpsertModelIsAlias || len(data.UpsertModelFields) > 0

	if hasCreate && (!hasRequestBody || !hasResponseModel) {
		fmt.Printf("  🔴 Excluding CREATE operation: Missing RequestBody=%v or UpsertModel=%v (ambiguous schema)\n", hasRequestBody, hasResponseModel)
		hasCreate = false
	}

	if hasUpdate && (!hasRequestBody || !hasResponseModel) {
		fmt.Printf("  🔴 Excluding UPDATE operation: Missing RequestBody=%v or UpsertModel=%v (ambiguous schema)\n", hasRequestBody, hasResponseModel)
		hasUpdate = false
	}

	// CRITICAL: Pass CRUD flags to template so it can conditionally generate methods
	data.HasCreate = hasCreate
	data.HasList = hasList
	data.HasRead = hasRead
	data.HasUpdate = hasUpdate
	data.HasDelete = hasDelete

	// Check if Update/Delete operations return AsyncTaskInResponse
	if hasUpdate {
		updateURL := data.Resource.GetOperationsURL()
		// Try PATCH first, then PUT
		if returnsAsyncTaskInResponse("PATCH", updateURL) {
			data.UpdateIsAsync = true
			data.HasAsyncMethods = true
			fmt.Printf("  ℹ️  Update operation returns AsyncTaskInResponse (async method with timeout parameter)\n")
		} else if returnsAsyncTaskInResponse("PUT", updateURL) {
			data.UpdateIsAsync = true
			data.HasAsyncMethods = true
			fmt.Printf("  ℹ️  Update operation returns AsyncTaskInResponse (async method with timeout parameter)\n")
		}
	}

	if hasDelete {
		deleteURL := data.Resource.GetOperationsURL()
		// Check both DELETE /{id}/ and DELETE / endpoints
		if returnsAsyncTaskInResponse("DELETE", deleteURL+"/{id}/") {
			data.DeleteByIdIsAsync = true
			data.HasAsyncMethods = true
			fmt.Printf("  ℹ️  Delete by ID operation (DELETE %s/{id}/) returns AsyncTaskInResponse (async method with timeout parameter)\n", deleteURL)
		}
		if returnsAsyncTaskInResponse("DELETE", deleteURL+"/") {
			data.DeleteIsAsync = true
			data.HasAsyncMethods = true
			fmt.Printf("  ℹ️  Delete operation (DELETE %s/) returns AsyncTaskInResponse (async method with timeout parameter)\n", deleteURL)
		}
	}

	// Determine if we should use EditModel (UPDATE without CREATE)
	data.UseEditModel = hasUpdate && !hasCreate

	// Build CRUD operations string for template header comment
	var ops []string
	if hasCreate {
		ops = append(ops, "CREATE")
	}
	if hasList {
		ops = append(ops, "LIST")
	}
	if hasRead {
		ops = append(ops, "READ")
	}
	if hasUpdate {
		ops = append(ops, "UPDATE")
	}
	if hasDelete {
		ops = append(ops, "DELETE")
	}

	hasValidExtraMethods := len(data.ExtraMethods) > 0
	hasGenerationIssues := len(data.GenerationIssues) > 0

	if len(ops) > 0 {
		data.TemplateOps = strings.Join(ops, "|")
	} else if hasValidExtraMethods {
		data.TemplateOps = "EXTRA_METHODS_ONLY"
	} else if hasGenerationIssues {
		data.TemplateOps = "EMPTY (All methods skipped)"
	} else {
		data.TemplateOps = "NONE"
	}

	var templateFile string
	if !hasList && !hasRead && !hasCreate && !hasUpdate && !hasDelete {
		if hasValidExtraMethods {
			// Resource with ONLY extra methods (no CRUD)
			templateFile = "templates/extramethods-only.tpl"
		} else if hasGenerationIssues {
			// Resource with generation issues but no valid methods
			templateFile = "templates/empty.tpl"
		} else {
			// Truly empty resource (shouldn't happen)
			templateFile = "templates/empty.tpl"
		}
	} else {
		// Resource with any combination of CRUD operations (including R-only)
		// The template uses conditionals ({{if .HasCreate}}, {{if .HasRead}}, etc.) to generate only the requested methods
		templateFile = "templates/resource.tpl"
	}

	tmpl, err := template.ParseFiles(templateFile)
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

// generateCommonComponentsFile generates common_components.go with all OpenAPI component schemas
func generateCommonComponentsFile(filename string) error {
	components, err := api.GetAllComponentSchemas()
	if err != nil {
		return fmt.Errorf("failed to get component schemas: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create common components file: %w", err)
	}
	defer file.Close()

	// Write header
	fmt.Fprintln(file, "// Code generated by generate-typed-resources. DO NOT EDIT.")
	fmt.Fprintln(file, "// This file contains all OpenAPI component schemas for reuse across resources.")
	fmt.Fprintln(file, "")
	fmt.Fprintln(file, "package typed")
	fmt.Fprintln(file, "")

	// Generate each component schema
	registry := NewTypeRegistry()
	for _, component := range components {
		// Skip if it's an ambiguous object or primitive
		if isAmbiguousObject(component.Schema) {
			continue
		}
		if isPrimitive(component.Schema) {
			continue
		}

		// Generate fields for this component
		fields, err := generateFieldsFromSchema(component.Schema, component.Name, registry, false, "COMPONENT")
		if err != nil {
			fmt.Printf("  ⚠️  Warning: Skipping component %s: %v\n", component.Name, err)
			continue
		}

		// Write component documentation
		componentTypeName := "Component_" + component.Name
		fmt.Fprintf(file, "// %s represents the OpenAPI component schema\n", componentTypeName)
		fmt.Fprintf(file, "// Component: %s\n", component.Reference)

		// Generate empty struct if no fields (e.g., all fields were ambiguous objects)
		// This is needed for type aliases to work even when the component has no usable fields
		if len(fields) == 0 {
			// Check if the schema has properties (but they were all skipped)
			if component.Schema.Properties != nil && len(component.Schema.Properties) > 0 {
				fmt.Fprintf(file, "// Note: All fields in this schema are ambiguous objects (objects without properties) and were skipped\n")
			}
			fmt.Fprintf(file, "type %s struct {}\n\n", componentTypeName)
			continue
		}

		fmt.Fprintf(file, "type %s struct {\n", componentTypeName)

		// Write fields
		for _, field := range fields {
			fmt.Fprintf(file, "\t%s %s `json:\"%s,omitempty\" yaml:\"%s,omitempty\" required:\"%s\"",
				field.Name, field.Type, field.JSONTag, field.YAMLTag, field.RequiredTag)
			if field.DocTag != "" {
				fmt.Fprintf(file, " doc:\"%s\"", field.DocTag)
			} else {
				fmt.Fprintf(file, " doc:\"\"")
			}
			fmt.Fprintln(file, "`")
		}

		fmt.Fprintln(file, "}")
		fmt.Fprintln(file, "")
	}

	// Generate nested types
	nestedTypes := registry.GetTypes()
	if len(nestedTypes) > 0 {
		fmt.Fprintln(file, "// -----------------------------------------------------")
		fmt.Fprintln(file, "// NESTED COMPONENT TYPES")
		fmt.Fprintln(file, "// -----------------------------------------------------")
		fmt.Fprintln(file, "")

		for _, nested := range nestedTypes {
			fmt.Fprintf(file, "// %s represents a nested type within components\n", nested.Name)
			fmt.Fprintf(file, "type %s struct {\n", nested.Name)
			for _, field := range nested.Fields {
				fmt.Fprintf(file, "\t%s %s `json:\"%s,omitempty\" yaml:\"%s,omitempty\" required:\"%s\"",
					field.Name, field.Type, field.JSONTag, field.YAMLTag, field.RequiredTag)
				if field.DocTag != "" {
					fmt.Fprintf(file, " doc:\"%s\"", field.DocTag)
				} else {
					fmt.Fprintf(file, " doc:\"\"")
				}
				fmt.Fprintln(file, "`")
			}
			fmt.Fprintln(file, "}")
			fmt.Fprintln(file, "")
		}
	}

	fmt.Printf("  ✅ Generated %d component schemas\n", len(components))
	return nil
}

// generateRequestFields generates struct fields from GET query parameters
func generateRequestFields(resourcePath string, registry *TypeRegistry) ([]Field, error) {
	// Get query parameters schema from OpenAPI
	schema, err := api.GetSchema_GET_QueryParams(resourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get query params schema: %w", err)
	}

	return generateFieldsFromSchema(schema.Value, resourcePath+"Request", registry, false, "REQUEST")
}

// toCamelCase converts snake_case to CamelCase
func toCamelCase(s string) string {
	// Replace hyphens with underscores first, then split on underscores
	s = strings.ReplaceAll(s, "-", "_")
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// toSingularCamelCase converts plural resource paths to singular CamelCase
// e.g., "quotas" -> "Quota", "views" -> "View"
func toSingularCamelCase(resourcePath string) string {
	// Convert to CamelCase first
	camelCase := toCamelCase(resourcePath)

	// Simple pluralization rules (can be extended as needed)
	if strings.HasSuffix(camelCase, "s") && len(camelCase) > 1 {
		// Remove trailing 's' for simple plurals
		return camelCase[:len(camelCase)-1]
	}

	return camelCase
}

// escapeQuotes escapes double quotes in strings to prevent breaking struct tags
func escapeQuotes(s string) string {
	// Escape quotes, backticks, and newlines for Go struct tags
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "`", "'") // Replace backticks with single quotes
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	// Collapse multiple spaces into single space
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}

// validateResourceMarkers validates that a resource has all required markers
func validateResourceMarkers(resource *vastparser.VastResource) error {
	// If resource has ONLY extra methods (no details/upsert), skip validation
	hasExtraMethods := len(resource.ExtraMethods) > 0
	hasDetails := resource.HasDetails("GET") || resource.HasDetails("PATCH")
	hasUpsert := resource.HasUpsert("POST") || resource.HasUpsert("PUT") || resource.HasUpsert("PATCH")

	// Allow resources with only extra methods (no CRUD operations)
	if hasExtraMethods && !hasDetails && !hasUpsert {
		return nil
	}

	// All resources with CRUD operations must have details marker
	if !hasDetails && hasUpsert {
		return fmt.Errorf("missing required details marker (GET or PATCH)")
	}

	// Non-read-only resources must have upsert marker
	if hasDetails && !resource.IsReadOnly() {
		if !hasUpsert {
			return fmt.Errorf("non-read-only resource missing required upsert marker (POST, PUT, or PATCH)")
		}
	}

	return nil
}

// formatGeneratedFiles runs go fmt on all Go files in the specified directory
func formatGeneratedFiles(dir string) error {
	// Find all .go files in the directory
	goFiles, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return fmt.Errorf("failed to find Go files: %w", err)
	}

	if len(goFiles) == 0 {
		return nil // No Go files to format
	}

	// Run go fmt on all Go files
	args := append([]string{"fmt"}, goFiles...)
	cmd := exec.Command("go", args...)

	// Set the working directory to the current directory (where the files are)
	// This ensures go fmt can find the files correctly
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go fmt failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// isObject returns true if the schema represents an object type
func isObject(prop *openapi3.Schema) bool {
	return prop.Type != nil && len(*prop.Type) > 0 && (*prop.Type)[0] == openapi3.TypeObject
}

// isAmbiguousObject returns true if the schema is an object without properties (ambiguous)
func isAmbiguousObject(prop *openapi3.Schema) bool {
	// An object is ambiguous if it has no properties AND no additionalProperties defined
	// Objects with additionalProperties are maps (e.g., map[string]string) and are NOT ambiguous
	return isObject(prop) && len(prop.Properties) == 0 && prop.AdditionalProperties.Schema == nil
}

// isMapObject checks if a schema represents a map-like object (additionalProperties with no properties)
// These are objects like {type: object, additionalProperties: {type: string}} which represent map[string]T
func isMapObject(prop *openapi3.Schema) bool {
	return isObject(prop) && len(prop.Properties) == 0 && prop.AdditionalProperties.Schema != nil
}

// isCreateResponseValid checks if POST operation has valid response schema or returns 204
func isCreateResponseValid(url string) bool {
	// Get OpenAPI resource
	resource, err := api.GetOpenApiResource(url)
	if err != nil || resource == nil || resource.Post == nil {
		return false
	}

	// Check if it returns 204 NO CONTENT - this is acceptable
	if resp204 := resource.Post.Responses.Status(204); resp204 != nil {
		return true
	}

	// Check for 200/201/202 responses with valid schema
	for _, statusCode := range []int{200, 201, 202} {
		resp := resource.Post.Responses.Status(statusCode)
		if resp == nil || resp.Value == nil {
			continue
		}

		// Get JSON content
		content := resp.Value.Content["application/json"]
		if content == nil || content.Schema == nil || content.Schema.Value == nil {
			continue
		}

		schema := content.Schema.Value

		// If schema is not ambiguous, it's valid
		if !isAmbiguousObject(schema) && !IsEmptySchema(&openapi3.SchemaRef{Value: schema}) {
			return true
		}
	}

	// No valid response found
	return false
}

// isListResponseAmbiguous checks if GET list operation returns array of ambiguous objects
func isListResponseAmbiguous(url string) bool {
	// Get OpenAPI resource
	resource, err := api.GetOpenApiResource(url)
	if err != nil || resource == nil || resource.Get == nil {
		return false
	}

	// Get 200 response
	resp := resource.Get.Responses.Status(200)
	if resp == nil || resp.Value == nil {
		return false
	}

	// Get JSON content
	content := resp.Value.Content["application/json"]
	if content == nil || content.Schema == nil || content.Schema.Value == nil {
		return false
	}

	schema := content.Schema.Value

	// Check if it's an array
	if schema.Type == nil || !(*schema.Type).Is("array") {
		return false
	}

	// Check if array items are ambiguous objects
	if schema.Items == nil || schema.Items.Value == nil {
		return false
	}

	itemSchema := schema.Items.Value
	return isAmbiguousObject(itemSchema)
}

// isReadResponseAmbiguous checks if GET by ID operation returns ambiguous object
func isReadResponseAmbiguous(url string) bool {
	// Get OpenAPI resource
	resource, err := api.GetOpenApiResource(url)
	if err != nil || resource == nil || resource.Get == nil {
		return false
	}

	// Get 200 response
	resp := resource.Get.Responses.Status(200)
	if resp == nil || resp.Value == nil {
		return false
	}

	// Get JSON content
	content := resp.Value.Content["application/json"]
	if content == nil || content.Schema == nil || content.Schema.Value == nil {
		return false
	}

	schema := content.Schema.Value
	return isAmbiguousObject(schema)
}

// isUpdateResponseValid checks if PATCH/PUT operation has valid response schema or returns 204
func isUpdateResponseValid(url string) bool {
	// Get OpenAPI resource
	resource, err := api.GetOpenApiResource(url)
	if err != nil || resource == nil {
		return false
	}

	// Check both PATCH and PUT operations
	operations := []*openapi3.Operation{resource.Patch, resource.Put}

	for _, op := range operations {
		if op == nil {
			continue
		}

		// Check if it returns 204 NO CONTENT - this is acceptable
		if resp204 := op.Responses.Status(204); resp204 != nil {
			return true
		}

		// Check for 200/201/202 responses with valid schema
		for _, statusCode := range []int{200, 201, 202} {
			resp := op.Responses.Status(statusCode)
			if resp == nil || resp.Value == nil {
				continue
			}

			// Get JSON content
			content := resp.Value.Content["application/json"]
			if content == nil || content.Schema == nil || content.Schema.Value == nil {
				continue
			}

			schema := content.Schema.Value

			// If schema is not ambiguous, it's valid
			if !isAmbiguousObject(schema) && !IsEmptySchema(&openapi3.SchemaRef{Value: schema}) {
				return true
			}
		}
	}

	// No valid response found
	return false
}

// hasAmbiguousNestedObjects recursively checks if a schema or any of its nested properties contain ambiguous objects
func hasAmbiguousNestedObjects(schema *openapi3.Schema) bool {
	if schema == nil {
		return false
	}

	// Check if this schema itself is ambiguous
	if isAmbiguousObject(schema) {
		return true
	}

	// Recursively check all properties
	for _, propRef := range schema.Properties {
		if propRef == nil || propRef.Value == nil {
			continue
		}
		if hasAmbiguousNestedObjects(propRef.Value) {
			return true
		}
	}

	// Check array items
	if schema.Items != nil && schema.Items.Value != nil {
		if hasAmbiguousNestedObjects(schema.Items.Value) {
			return true
		}
	}

	// Check allOf, oneOf, anyOf compositions
	for _, schemaRef := range schema.AllOf {
		if schemaRef != nil && schemaRef.Value != nil && hasAmbiguousNestedObjects(schemaRef.Value) {
			return true
		}
	}
	for _, schemaRef := range schema.OneOf {
		if schemaRef != nil && schemaRef.Value != nil && hasAmbiguousNestedObjects(schemaRef.Value) {
			return true
		}
	}
	for _, schemaRef := range schema.AnyOf {
		if schemaRef != nil && schemaRef.Value != nil && hasAmbiguousNestedObjects(schemaRef.Value) {
			return true
		}
	}

	return false
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

// mapOpenAPITypeToGo maps OpenAPI primitive types to Go types
func mapOpenAPITypeToGo(schema *openapi3.Schema) string {
	if schema == nil || schema.Type == nil || len(*schema.Type) == 0 {
		return "any"
	}

	switch (*schema.Type)[0] {
	case openapi3.TypeString:
		return "string"
	case openapi3.TypeInteger:
		if schema.Format == "int64" {
			return "int64"
		}
		return "int"
	case openapi3.TypeNumber:
		if schema.Format == "double" {
			return "float64"
		}
		return "float32"
	case openapi3.TypeBoolean:
		return "bool"
	default:
		return "any"
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
		if usePointers {
			goType = "*[]interface{}" // pointer to slice for proper omitempty handling
		} else {
			goType = "[]interface{}"
		}
	case "object":
		goType = "map[string]interface{}" // generic object type
	default:
		goType = "interface{}" // fallback for unknown types
	}

	// Only use pointers for objects and arrays (at top level), primitives stay as-is
	if usePointers && baseType == "object" {
		return "*" + goType
	}
	return goType
}

// generateResponseFields generates struct fields from POST response schema
func generateResponseFields(resourcePath string, registry *TypeRegistry) ([]Field, error) {
	// Get response schema from OpenAPI
	schema, err := api.GetResponseModelSchema("POST", resourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get response schema: %w", err)
	}

	return generateFieldsFromSchema(schema.Value, resourcePath+"Response", registry, false, "RESPONSE")
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
			fmt.Printf("    ⏭️  Skipping non-primitive search param '%s'\n", p.Name)
			continue
		}

		name := p.Name
		if contains(excludeSearchParams, name) {
			fmt.Printf("    ⏭️  Skipping excluded search param '%s'\n", name)
			continue
		}

		// Skip fields with double underscores (Django-style query filters)
		if strings.Contains(name, "__") {
			fmt.Printf("    ⏭️  Skipping Django-style query filter '%s'\n", name)
			continue
		}

		if p.Schema == nil || p.Schema.Value == nil || p.Schema.Value.Type == nil || len(*p.Schema.Value.Type) == 0 {
			fmt.Printf("    ⏭️  Skipping search param '%s' with invalid schema\n", name)
			continue
		}

		// Generate field for this parameter
		field := Field{
			Name:        toCamelCase(name),
			JSONTag:     name,
			YAMLTag:     name,
			RequiredTag: "false", // Search parameters are typically optional
			DocTag:      escapeQuotes(p.Description),
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
		schema, err = api.GetRequestBodySchema("POST", resourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get POST request body schema for resource %q: %w", resourcePath, err)
		}
	case http.MethodPatch:
		schema, err = api.GetRequestBodySchema("PATCH", resourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get PATCH request body schema for resource %q: %w", resourcePath, err)
		}
	case http.MethodPut:
		schema, err = api.GetRequestBodySchema("PUT", resourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get PUT request body schema for resource %q: %w", resourcePath, err)
		}
	case http.MethodDelete:
		schema, err = api.GetRequestBodySchema("DELETE", resourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get DELETE request body schema for resource %q: %w", resourcePath, err)
		}
	default:
		return nil, fmt.Errorf("unsupported method %q for request body generation", method)
	}

	if IsEmptySchema(schema) {
		return nil, fmt.Errorf("request body schema is empty for resource %q", resourcePath)
	}

	// Convert resource path to singular Go type name (e.g., "quotas" -> "Quota")
	typeName := toSingularCamelCase(resourcePath) + "RequestBody"
	return generateFieldsFromSchema(schema.Value, typeName, registry, false, "REQUEST BODY")
}

// generateModelFields generates model fields using method-based resolution
func generateModelFields(resourcePath, method string, registry *TypeRegistry) ([]Field, error) {
	var schema *openapi3.SchemaRef
	var err error

	// Use method-based switch like terraform provider (modelSchemaRef pattern)
	switch method {
	case http.MethodPost:
		schema, err = api.GetResponseModelSchema("POST", resourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get POST response schema for resource %q: %w", resourcePath, err)
		}
	case http.MethodGet:
		schema, err = api.GetResponseModelSchema("GET", resourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get GET response schema for resource %q: %w", resourcePath, err)
		}
	case http.MethodPatch:
		schema, err = api.GetResponseModelSchema("PATCH", resourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get PATCH response schema for resource %q: %w", resourcePath, err)
		}
	case http.MethodPut:
		schema, err = api.GetResponseModelSchema("PUT", resourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get PUT response schema for resource %q: %w", resourcePath, err)
		}
	case http.MethodDelete:
		schema, err = api.GetResponseModelSchema("DELETE", resourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get DELETE response schema for resource %q: %w", resourcePath, err)
		}
	default:
		return nil, fmt.Errorf("unsupported method %q for response body generation", method)
	}

	if IsEmptySchema(schema) {
		return nil, fmt.Errorf("response body schema is empty for resource %q", resourcePath)
	}

	// Convert resource path to singular Go type name (e.g., "quotas" -> "Quota")
	typeName := toSingularCamelCase(resourcePath) + "Model"
	return generateFieldsFromSchema(schema.Value, typeName, registry, false, "MODEL")
}

// extractCommonSearchableFields extracts common searchable fields from response body schema
func extractCommonSearchableFields(resource *vastparser.VastResource, registry *TypeRegistry) ([]Field, error) {
	// Common searchable field names
	commonSearchableFields := []string{
		"name", "path", "bucket", "gid", "uid", "guid", "tenant_id",
	}

	var responseSchema *openapi3.SchemaRef
	var err error

	// Get response body schema from Operations marker or legacy details marker
	if resource.HasOperations() && resource.Operations.HasRead() {
		responseURL := resource.GetOperationsURL()
		responseSchema, err = api.GetResponseModelSchema("GET", responseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to get GET response schema: %w", err)
		}
	} else if resource.HasDetails("GET") {
		responseURL := resource.GetDetails("GET")
		responseSchema, err = api.GetResponseModelSchema("GET", responseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to get GET response schema: %w", err)
		}
	} else if resource.HasDetails("PATCH") {
		responseURL := resource.GetDetails("PATCH")
		responseSchema, err = api.GetResponseModelSchema("PATCH", responseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to get PATCH response schema: %w", err)
		}
	} else {
		// No response body schema available
		return nil, nil
	}

	if responseSchema == nil || responseSchema.Value == nil || responseSchema.Value.Properties == nil {
		return nil, nil
	}

	var fields []Field

	// Check each common searchable field
	for _, fieldName := range commonSearchableFields {
		if propRef, exists := responseSchema.Value.Properties[fieldName]; exists {
			if propRef == nil || propRef.Value == nil {
				continue
			}

			// Only include primitive types for search params
			if !isPrimitive(propRef.Value) {
				fmt.Printf("    ⏭️  Skipping non-primitive common searchable field '%s'\n", fieldName)
				continue
			}

			// Determine if field is required
			isRequired := "false"
			for _, requiredField := range responseSchema.Value.Required {
				if requiredField == fieldName {
					isRequired = "true"
					break
				}
			}

			// Get Go type for the field
			goType := getGoTypeFromOpenAPI(propRef.Value, false)

			field := Field{
				Name:        toCamelCase(fieldName),
				Type:        goType,
				JSONTag:     fieldName,
				YAMLTag:     fieldName,
				RequiredTag: isRequired,
				DocTag:      escapeQuotes(propRef.Value.Description),
			}

			fields = append(fields, field)
			fmt.Printf("    ✅ Added common searchable field '%s'\n", fieldName)
		}
	}

	// Sort fields: required first, then non-required
	sortFieldsByRequired(fields)

	return fields, nil
}

// mergeSearchFields merges search fields from different sources, avoiding duplicates
func mergeSearchFields(existing, additional []Field) []Field {
	// Create a map to track existing field names
	existingNames := make(map[string]bool)
	for _, field := range existing {
		existingNames[field.JSONTag] = true
	}

	// Add additional fields that don't already exist
	result := existing
	for _, field := range additional {
		if !existingNames[field.JSONTag] {
			result = append(result, field)
		}
	}

	// Sort the final result: required first, then non-required
	sortFieldsByRequired(result)

	return result
}

// generateSearchParamsFromSchema generates search params fields from a schema component
func generateSearchParamsFromSchema(schemaName string, registry *TypeRegistry) ([]Field, error) {
	// Get schema from components
	schema, err := api.GetSchema_FromComponents(schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema from components: %w", err)
	}

	return generateFieldsFromSchema(schema.Value, schemaName+"SearchParams", registry, true, "SEARCH PARAMS")
}

// generateRequestBodyFromSchema generates request body fields from a schema component
func generateRequestBodyFromSchema(schemaName string, registry *TypeRegistry) ([]Field, error) {
	// Get schema from components
	schema, err := api.GetSchema_FromComponents(schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema from components: %w", err)
	}

	return generateFieldsFromSchema(schema.Value, schemaName+"RequestBody", registry, false, "REQUEST BODY")
}

// generateModelFromSchema generates model fields from a schema component
func generateModelFromSchema(schemaName string, registry *TypeRegistry) ([]Field, error) {
	// Get schema from components
	schema, err := api.GetSchema_FromComponents(schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema from components: %w", err)
	}

	return generateFieldsFromSchema(schema.Value, schemaName+"Model", registry, false, "MODEL")
}

// generateFieldsFromSchema recursively generates fields from an OpenAPI schema
func generateFieldsFromSchema(schema *openapi3.Schema, parentTypeName string, registry *TypeRegistry, usePointers bool, section string) ([]Field, error) {
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
			fmt.Printf("    ⚠️  Skipping ambiguous object field '%s' (object without properties)\n", propName)
			continue
		}

		// Skip ambiguous arrays (arrays of objects without properties)
		if isAmbiguousArray(propRef.Value) {
			fmt.Printf("    ⚠️  Skipping ambiguous array field '%s' (array of objects without properties)\n", propName)
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
			DocTag:      escapeQuotes(propRef.Value.Description),
		}

		// Recursively determine Go type - pass false for isNestedArray since this is a top-level field
		goType, err := getGoTypeFromOpenAPIRecursive(propRef.Value, parentTypeName+"_"+toCamelCase(propName), registry, usePointers, false, section)
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
// isNestedArray indicates if we're processing an item inside an array (to prevent nested pointers like *[]*[]string)
func getGoTypeFromOpenAPIRecursive(schema *openapi3.Schema, typeName string, registry *TypeRegistry, usePointers bool, isNestedArray bool, section string) (string, error) {
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
			// Arrays should have pointer at top level (for omitempty), but not when nested
			if !isNestedArray {
				goType = "*[]interface{}" // pointer to slice for proper omitempty handling
			} else {
				goType = "[]interface{}"
			}
		} else {
			// Recurse into array items with isNestedArray=true to prevent nested pointers
			itemType, err := getGoTypeFromOpenAPIRecursive(schema.Items.Value, typeName+"Item", registry, false, true, section)
			if err != nil {
				return "", fmt.Errorf("failed to generate array item type: %w", err)
			}
			// Arrays should have pointer at top level (for omitempty), but not when nested
			if !isNestedArray {
				goType = "*[]" + itemType // pointer to slice for proper omitempty handling
			} else {
				goType = "[]" + itemType
			}
		}
	case "object":
		if schema.Properties == nil || len(schema.Properties) == 0 {
			// Empty object or map with additionalProperties
			if schema.AdditionalProperties.Schema != nil {
				valueType, err := getGoTypeFromOpenAPIRecursive(schema.AdditionalProperties.Schema.Value, typeName+"Value", registry, false, isNestedArray, section)
				if err != nil {
					return "", fmt.Errorf("failed to generate map value type: %w", err)
				}
				goType = "map[string]" + valueType
			} else {
				goType = "map[string]interface{}" // generic object type
			}
		} else {
			// Object with defined properties - generate a nested struct
			nestedFields, err := generateFieldsFromSchema(schema, typeName, registry, false, section)
			if err != nil {
				return "", fmt.Errorf("failed to generate nested fields: %w", err)
			}

			// Register the nested type
			registry.RegisterType(typeName, nestedFields, section)
			goType = typeName
		}
	default:
		goType = "interface{}" // fallback for unknown types
	}

	// Only use pointers for objects (nested structs), arrays are already pointers
	if usePointers && baseType == "object" {
		return "*" + goType, nil
	}
	return goType, nil
}
