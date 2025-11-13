//go:build tools

package main

import (
	"fmt"
	"log"
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

// MethodInfo represents information about a generated method
type MethodInfo struct {
	ResourceName     string
	Name             string
	HTTPMethod       string
	GoHTTPMethod     string
	Path             string
	ResourcePath     string
	SubPath          string
	Summary          string
	HasID            bool
	HasParams        bool
	HasBody          bool
	ReceiverName     string
	ReturnsNoContent bool // True if returns 204 No Content (use core.Record)
	ReturnsArray     bool // True if returns an array (use core.RecordSet)
	// For async task methods (returns AsyncTaskInResponse)
	IsAsyncTask bool
	// Body field documentation (for comment generation)
	BodyFields []BodyFieldInfo
	// Query parameters documentation (for comment generation)
	ParamsFields []BodyFieldInfo
}

// BodyFieldInfo represents a body or params field with its documentation
type BodyFieldInfo struct {
	Name        string // Field name in JSON (e.g., "column_type")
	Description string // Field description from OpenAPI
}

// UntypedResourceData represents data for untyped resource template generation
type UntypedResourceData struct {
	Name            string
	LowerName       string
	ReceiverName    string
	Methods         []MethodInfo
	Resource        *vastparser.UntypedResource
	HasAsyncMethods bool // True if any method is an async task
}

func main() {
	// Paths
	outputDir := "../resources/untyped"
	restFilePath := "../rest/untyped_rest.go"

	// Parse rest/untyped_rest.go to get resource configurations and extra methods
	fmt.Println("Parsing rest/untyped_rest.go for resource configurations...")
	restParser := vastparser.NewRestParser()
	if err := restParser.ParseRestFile(restFilePath); err != nil {
		log.Fatalf("Failed to parse rest file: %v", err)
	}

	configs := restParser.GetAllConfigs()
	fmt.Printf("Found %d resource configurations\n", len(configs))

	// Convert configs to UntypedResource format
	var allResources []vastparser.UntypedResource
	for _, config := range configs {
		// Skip resources without extra methods
		if len(config.ExtraMethods) == 0 {
			continue
		}

		resource := vastparser.UntypedResource{
			Name:         config.Name,
			ExtraMethods: config.ExtraMethods,
		}
		allResources = append(allResources, resource)
	}

	if len(allResources) == 0 {
		fmt.Println("No resources with extra methods found")
		return
	}

	// Sort resources by name for consistent generation order
	sort.Slice(allResources, func(i, j int) bool {
		return allResources[i].Name < allResources[j].Name
	})

	fmt.Printf("Found %d resources with apiuntyped markers:\n", len(allResources))
	for _, resource := range allResources {
		fmt.Printf("  - %s\n", resource.Name)
	}

	// Generate files for each resource
	var generatedFiles []string
	for _, resource := range allResources {
		fmt.Printf("\n%s:\n", resource.Name)

		resourceData := UntypedResourceData{
			Name:         resource.Name,
			LowerName:    strings.ToLower(resource.Name),
			ReceiverName: strings.ToLower(string(resource.Name[0])),
			Resource:     &resource,
		}

		// Generate method information for each extra method
		for _, extraMethod := range resource.ExtraMethods {
			methodInfo := generateMethodInfo(resource.Name, extraMethod, resourceData.ReceiverName)
			resourceData.Methods = append(resourceData.Methods, methodInfo)

			// Check if this is an async method
			if methodInfo.IsAsyncTask {
				resourceData.HasAsyncMethods = true
			}

			fmt.Printf("  ✅ Generated method: %s\n", methodInfo.Name)
		}

		// Sort methods by name+method for deterministic output
		sort.Slice(resourceData.Methods, func(i, j int) bool {
			iKey := resourceData.Methods[i].Name + "_" + resourceData.Methods[i].HTTPMethod
			jKey := resourceData.Methods[j].Name + "_" + resourceData.Methods[j].HTTPMethod
			return iKey < jKey
		})

		// Generate the autogen file (always regenerate)
		autogenFile := filepath.Join(outputDir, strings.ToLower(toSnakeCase(resource.Name))+"_autogen.go")
		if err := generateAutogenFile(autogenFile, resourceData); err != nil {
			log.Fatalf("Failed to generate %s: %v", autogenFile, err)
		}
		generatedFiles = append(generatedFiles, filepath.Base(autogenFile))

		// Generate the metadata file (always regenerate)
		metadataFile := filepath.Join(outputDir, strings.ToLower(toSnakeCase(resource.Name))+"_metadata.go")
		if err := generateMetadataFile(metadataFile, resourceData, configs[resource.Name]); err != nil {
			log.Fatalf("Failed to generate %s: %v", metadataFile, err)
		}
		generatedFiles = append(generatedFiles, filepath.Base(metadataFile))
	}

	fmt.Printf("\nGenerated untyped extra methods for %d resources in %s/\n", len(allResources), outputDir)
	for _, file := range generatedFiles {
		fmt.Printf("  - %s: Extra methods implementation\n", file)
	}

	// Format all generated Go files
	if err := formatGeneratedFiles(generatedFiles); err != nil {
		log.Printf("Warning: Failed to format generated files: %v", err)
	} else {
		fmt.Printf("Formatted all generated Go files with go fmt\n")
	}
}

// generateMethodInfo generates method information from an extra method configuration
func generateMethodInfo(resourceName string, extraMethod apibuilder.ExtraMethod, receiverName string) MethodInfo {
	methodInfo := MethodInfo{
		ResourceName: resourceName,
		HTTPMethod:   extraMethod.Method,
		Path:         extraMethod.Path,
		ReceiverName: receiverName,
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

	// Convert HTTP method to Go constant (e.g., "PATCH" -> "MethodPatch")
	methodInfo.GoHTTPMethod = httpMethodToGoConstant(extraMethod.Method)

	// Extract summary from OpenAPI spec
	summary, err := api.GetOperationSummary(extraMethod.Method, extraMethod.Path)
	if err != nil {
		// If summary not found, just log and continue
		fmt.Printf("  ℹ️  No summary found for %s %s\n", extraMethod.Method, extraMethod.Path)
	} else {
		methodInfo.Summary = summary
	}

	// Check if operation returns 204 No Content
	returns204, err := api.Returns204NoContent(extraMethod.Method, extraMethod.Path)
	if err == nil && returns204 {
		methodInfo.ReturnsNoContent = true
		fmt.Printf("  ℹ️  Method returns 204 No Content, using core.Record\n")
	}

	// Check if this is a bare array response BEFORE schema unwrapping
	// (GetResponseModelSchema unwraps arrays for GET, so we need to check the raw schema first)
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
						schema := content.Schema.Value
						if schema.Type != nil && (*schema.Type).Is("array") {
							// Check if the array contains objects or primitives
							isArrayOfObjects := false
							if schema.Items != nil {
								// Check for $ref (reference to another schema definition)
								if schema.Items.Ref != "" {
									// It's a reference - assume it's an object
									isArrayOfObjects = true
								} else if schema.Items.Value != nil {
									itemType := schema.Items.Value.Type
									// If items type is "object" or has properties, it's an array of objects
									if itemType != nil && (*itemType).Is("object") {
										isArrayOfObjects = true
									} else if schema.Items.Value.Properties != nil && len(schema.Items.Value.Properties) > 0 {
										isArrayOfObjects = true
									}
								}
							}

							if isArrayOfObjects {
								methodInfo.ReturnsArray = true
								fmt.Printf("  ℹ️  Detected array of objects response, using core.RecordSet\n")
							} else {
								// Array of primitives - use core.Record with @raw
								fmt.Printf("  ℹ️  Detected array of primitives response, using core.Record\n")
							}
						}
					}
				}
			}
		}
	}

	// Check if path contains {id} parameter
	methodInfo.HasID = strings.Contains(extraMethod.Path, "{id}")

	// Parse the path to extract resource path and sub-path
	// Example: /users/{id}/tenant_data/ -> resource: "users", subPath: "tenant_data"
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

	// Generate method name: ResourceName + HTTPMethodAction + LastPathPart
	// Example: PATCH /users/{id}/tenant_data/ -> UserUpdateTenantData
	lastPart := pathParts[len(pathParts)-1]
	if lastPart == "" && len(pathParts) > 1 {
		lastPart = pathParts[len(pathParts)-2]
	}

	// Clean and capitalize the last part
	lastPart = cleanPathPart(lastPart)
	action := toCamelCase(lastPart)

	// Store base name without HTTP method suffix
	methodInfo.Name = resourceName + action

	// Determine if method has body based on HTTP method (match main branch logic)
	methodInfo.HasBody = extraMethod.Method == "POST" || extraMethod.Method == "PUT" || extraMethod.Method == "PATCH"

	// Extract query parameters from OpenAPI schema for ALL HTTP methods
	// Only set HasParams=true if there are actual query parameters defined
	methodInfo.ParamsFields = extractQueryParams(extraMethod.Method, extraMethod.Path)
	methodInfo.HasParams = len(methodInfo.ParamsFields) > 0

	// Check if DELETE has a request body with properties in OpenAPI spec
	if extraMethod.Method == "DELETE" {
		schema, err := api.GetRequestBodySchema(extraMethod.Method, extraMethod.Path)
		if err == nil && schema != nil && schema.Value != nil {
			// Check if body has actual properties
			if schema.Value.Properties != nil && len(schema.Value.Properties) > 0 {
				// DELETE has a body with actual content
				methodInfo.HasBody = true
				methodInfo.HasParams = false // Body takes precedence over query params
			}
		}
	}

	// Extract body field documentation from OpenAPI schema if method has body
	if methodInfo.HasBody {
		methodInfo.BodyFields = extractBodyFields(extraMethod.Method, extraMethod.Path)
	}

	return methodInfo
}

// extractBodyFields extracts body field names and descriptions from OpenAPI schema
func extractBodyFields(httpMethod, path string) []BodyFieldInfo {
	var bodyFields []BodyFieldInfo

	// Get the request body schema from OpenAPI
	schema, err := api.GetRequestBodySchema(httpMethod, path)
	if err != nil || schema == nil || schema.Value == nil {
		return bodyFields
	}

	// Check if it's an object with properties
	if schema.Value.Type == nil || !(*schema.Value.Type).Is("object") {
		return bodyFields
	}

	properties := schema.Value.Properties
	if properties == nil || len(properties) == 0 {
		return bodyFields
	}

	// Extract field names and descriptions
	for fieldName, propRef := range properties {
		if propRef == nil || propRef.Value == nil {
			continue
		}

		prop := propRef.Value
		description := ""
		if prop.Description != "" {
			// Replace newlines with spaces to prevent multi-line descriptions from breaking comment syntax
			description = strings.ReplaceAll(prop.Description, "\n", " ")
			// Collapse multiple spaces into one
			description = strings.Join(strings.Fields(description), " ")
		}

		bodyFields = append(bodyFields, BodyFieldInfo{
			Name:        fieldName,
			Description: description,
		})
	}

	// Sort by field name for consistent output
	sort.Slice(bodyFields, func(i, j int) bool {
		return bodyFields[i].Name < bodyFields[j].Name
	})

	return bodyFields
}

// extractQueryParams extracts query parameter names and descriptions from OpenAPI schema
func extractQueryParams(httpMethod, path string) []BodyFieldInfo {
	var paramsFields []BodyFieldInfo

	// Get the query parameters from OpenAPI
	params, err := api.GetQueryParameters(httpMethod, path)
	if err != nil || len(params) == 0 {
		return paramsFields
	}

	// Extract parameter names and descriptions
	// Preserve the original order from OpenAPI schema (do NOT sort)
	for _, param := range params {
		if param == nil {
			continue
		}

		description := ""
		if param.Description != "" {
			// Replace newlines with spaces to prevent multi-line descriptions from breaking comment syntax
			description = strings.ReplaceAll(param.Description, "\n", " ")
			// Collapse multiple spaces into one
			description = strings.Join(strings.Fields(description), " ")
		}

		paramsFields = append(paramsFields, BodyFieldInfo{
			Name:        param.Name,
			Description: description,
		})
	}

	return paramsFields
}

// httpMethodToGoConstant converts HTTP method string to Go http.Method constant
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

// toCamelCase converts snake_case or kebab-case to CamelCase
func toCamelCase(s string) string {
	// Replace underscores and hyphens with spaces
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")

	// Split into words and capitalize each
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, "")
}

// generateAutogenFile generates the autogen file with all extra methods
func generateAutogenFile(filename string, data UntypedResourceData) error {
	tmpl := `// Code generated by generate-untyped-resources. DO NOT EDIT.

package untyped

import (
	"context"
	"net/http"
{{if .HasAsyncMethods}}	"time"
{{end}}
	"github.com/vast-data/go-vast-client/core"
)

{{range .Methods}}
{{$method := .}}
// {{.Name}}WithContext_{{.HTTPMethod}}
// method: {{.HTTPMethod}}
// url: {{.Path}}{{if .Summary}}
// summary: {{.Summary}}{{end}}{{if or .HasParams .HasBody .IsAsyncTask}}
//{{if .HasParams}}{{if .ParamsFields}}
// Params:{{range .ParamsFields}}
//   - {{.Name}}{{if .Description}}: {{.Description}}{{end}}{{end}}{{end}}{{end}}{{if .HasBody}}
// Body:{{if .BodyFields}}{{range .BodyFields}}
//   - {{.Name}}{{if .Description}}: {{.Description}}{{end}}{{end}}{{else}}
//
//	< not declared in schema >{{end}}{{end}}{{if .IsAsyncTask}}
//
// Parameters:
//   - waitTimeout: If 0, returns immediately without waiting (async). Otherwise, waits for task completion with the specified timeout.{{end}}{{end}}
func ({{$.ReceiverName}} *{{$.Name}}) {{.Name}}WithContext_{{.HTTPMethod}}(ctx context.Context{{if .HasID}}, id any{{end}}{{if .HasParams}}, params core.Params{{end}}{{if .HasBody}}, body core.Params{{end}}{{if .IsAsyncTask}}, waitTimeout time.Duration{{end}}) ({{if .IsAsyncTask}}*AsyncResult, error{{else}}{{if .ReturnsNoContent}}error{{else}}{{if .ReturnsArray}}core.RecordSet, error{{else}}core.Record, error{{end}}{{end}}{{end}}) {
	{{if .HasID}}resourcePath := core.BuildResourcePathWithID("{{.ResourcePath}}", id{{if .SubPath}}, "{{.SubPath}}"{{end}})
	{{else}}resourcePath := "{{.Path}}"
	{{end}}{{if .IsAsyncTask}}result, err := core.Request[core.Record](ctx, {{$.ReceiverName}}, http.{{.GoHTTPMethod}}, resourcePath, {{if .HasParams}}params{{else}}nil{{end}}, {{if .HasBody}}body{{else}}nil{{end}})
	if err != nil {
		return nil, err
	}

	return MaybeWaitAsyncResultWithContext(ctx, result, {{$.ReceiverName}}.Rest, waitTimeout)
	{{else}}{{if .ReturnsNoContent}}_, err := core.Request[core.Record](ctx, {{$.ReceiverName}}, http.{{.GoHTTPMethod}}, resourcePath, {{if .HasParams}}params{{else}}nil{{end}}, {{if .HasBody}}body{{else}}nil{{end}})
	return err
	{{else}}{{if .ReturnsArray}}result, err := core.Request[core.RecordSet](ctx, {{$.ReceiverName}}, http.{{.GoHTTPMethod}}, resourcePath, {{if .HasParams}}params{{else}}nil{{end}}, {{if .HasBody}}body{{else}}nil{{end}})
	if err != nil {
		return nil, err
	}
	return result, nil
	{{else}}result, err := core.Request[core.Record](ctx, {{$.ReceiverName}}, http.{{.GoHTTPMethod}}, resourcePath, {{if .HasParams}}params{{else}}nil{{end}}, {{if .HasBody}}body{{else}}nil{{end}})
	if err != nil {
		return nil, err
	}
	return result, nil{{end}}{{end}}{{end}}
}

// {{.Name}}_{{.HTTPMethod}}
// method: {{.HTTPMethod}}
// url: {{.Path}}{{if .Summary}}
// summary: {{.Summary}}{{end}}{{if or .HasParams .HasBody .IsAsyncTask}}
//{{if .HasParams}}{{if .ParamsFields}}
// Params:{{range .ParamsFields}}
//   - {{.Name}}{{if .Description}}: {{.Description}}{{end}}{{end}}{{end}}{{end}}{{if .HasBody}}
// Body:{{if .BodyFields}}{{range .BodyFields}}
//   - {{.Name}}{{if .Description}}: {{.Description}}{{end}}{{end}}{{else}}
//
//	< not declared in schema >{{end}}{{end}}{{if .IsAsyncTask}}
//
// Parameters:
//   - waitTimeout: If 0, returns immediately without waiting (async). Otherwise, waits for task completion with the specified timeout.{{end}}{{end}}
func ({{$.ReceiverName}} *{{$.Name}}) {{.Name}}_{{.HTTPMethod}}({{if .HasID}}id any, {{end}}{{if .HasParams}}params core.Params, {{end}}{{if .HasBody}}body core.Params{{if .IsAsyncTask}}, {{end}}{{end}}{{if .IsAsyncTask}}waitTimeout time.Duration{{end}}) ({{if .IsAsyncTask}}*AsyncResult, error{{else}}{{if .ReturnsNoContent}}error{{else}}{{if .ReturnsArray}}core.RecordSet, error{{else}}core.Record, error{{end}}{{end}}{{end}}) {
	return {{$.ReceiverName}}.{{.Name}}WithContext_{{.HTTPMethod}}({{$.ReceiverName}}.Rest.GetCtx(){{if .HasID}}, id{{end}}{{if .HasParams}}, params{{end}}{{if .HasBody}}, body{{end}}{{if .IsAsyncTask}}, waitTimeout{{end}})
}

{{end}}`

	t, err := template.New("autogen").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse autogen template: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create autogen file: %w", err)
	}
	defer file.Close()

	if err := t.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute autogen template: %w", err)
	}

	return nil
}

// generateMetadataFile generates the metadata file with init() function for method registration
func generateMetadataFile(filename string, data UntypedResourceData, config *vastparser.RestResourceConfig) error {
	tmpl := `// Code generated by generate-untyped-resources. DO NOT EDIT.

package untyped

import "github.com/vast-data/go-vast-client/core"

// This file registers metadata for all extra methods on the {{.Name}} resource
// This information comes from the OpenAPI schema during code generation

func init() {
	{{if .ResourcePath}}// Register metadata for {{.Name}} extra methods
	// Resource path: {{.ResourcePath}}
	{{range .Methods}}
	core.RegisterExtraMethod(
		"{{$.Name}}",                   // resource type (Go struct name)
		"{{.Name}}_{{.HTTPMethod}}",    // method name
		"{{.HTTPMethod}}",              // HTTP verb
		"{{.Path}}",                    // URL path with placeholders{{if .Summary}}
		"{{.Summary}}",                 // summary from OpenAPI{{else}}
		"",                             // no summary available{{end}}
	)
	{{end}}{{else}}// No resource path found for {{.Name}}
	{{end}}
}
`

	t, err := template.New("metadata").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse metadata template: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create metadata file: %w", err)
	}
	defer file.Close()

	// Create template data with ResourcePath
	type TemplateData struct {
		UntypedResourceData
		ResourcePath string
	}

	templateData := TemplateData{
		UntypedResourceData: data,
		ResourcePath:        "",
	}

	// Get resource path from config
	if config != nil {
		templateData.ResourcePath = config.ResourcePath
	}

	if err := t.Execute(file, templateData); err != nil {
		return fmt.Errorf("failed to execute metadata template: %w", err)
	}

	return nil
}

// toSnakeCase converts CamelCase to snake_case
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

// formatGeneratedFiles runs go fmt on generated files
func formatGeneratedFiles(files []string) error {
	if len(files) == 0 {
		return nil
	}

	// Run go fmt on all generated files
	args := append([]string{"fmt"}, files...)
	cmd := exec.Command("go", args...)
	cmd.Dir = "../resources/untyped"

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go fmt failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}
