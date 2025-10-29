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
	// For simplified bodies (up to 3 fields become inline parameters)
	SimplifiedBody   bool              // True if body has 1-3 fields
	SimplifiedParams []SimplifiedParam // The inline parameters
	// For async task methods (returns AsyncTaskInResponse)
	IsAsyncTask bool
}

// SimplifiedParam represents a simplified inline parameter
type SimplifiedParam struct {
	Name        string // Go parameter name (e.g., "accessKey")
	Type        string // Go type (e.g., "string", "int64", "bool")
	BodyField   string // JSON field name (e.g., "access_key")
	Required    bool   // Whether the field is required
	Description string // Parameter description from OpenAPI spec
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
						if content.Schema.Value.Type != nil && (*content.Schema.Value.Type).Is("array") {
							methodInfo.ReturnsArray = true
							fmt.Printf("  ℹ️  Detected bare array response, using core.RecordSet\n")
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

	// Determine if method has params and body based on HTTP method
	methodInfo.HasParams = extraMethod.Method == "GET" || extraMethod.Method == "DELETE"
	methodInfo.HasBody = extraMethod.Method == "POST" || extraMethod.Method == "PUT" || extraMethod.Method == "PATCH"

	// Check if method body can be simplified (1-3 fields become inline parameters)
	// For DELETE, check if it has a request body in the OpenAPI spec
	if methodInfo.HasBody || extraMethod.Method == "DELETE" {
		simplified, params := checkSimplifiedBody(extraMethod.Method, extraMethod.Path)
		if simplified && len(params) > 0 && len(params) <= 3 {
			methodInfo.SimplifiedBody = true
			methodInfo.SimplifiedParams = params
			// If DELETE has a body, switch from query params to body
			if extraMethod.Method == "DELETE" {
				methodInfo.HasParams = false
				methodInfo.HasBody = true
			}
			fmt.Printf("  ℹ️  Simplified body: %d inline parameters\n", len(params))
		}
	}

	return methodInfo
}

// checkSimplifiedBody checks if a request body can be simplified to inline parameters
// Returns true and the list of parameters if the body has 1-3 fields, false otherwise
func checkSimplifiedBody(httpMethod, path string) (bool, []SimplifiedParam) {
	// Get the request body schema from OpenAPI
	schema, err := api.GetRequestBodySchema(httpMethod, path)
	if err != nil || schema == nil || schema.Value == nil {
		return false, nil
	}

	// Check if it's an object with properties
	if schema.Value.Type == nil || !(*schema.Value.Type).Is("object") {
		return false, nil
	}

	properties := schema.Value.Properties
	if properties == nil || len(properties) == 0 || len(properties) > 3 {
		return false, nil
	}

	// Extract parameters
	var params []SimplifiedParam
	for fieldName, propRef := range properties {
		if propRef == nil || propRef.Value == nil {
			continue
		}

		prop := propRef.Value

		// Determine Go type from OpenAPI type
		goType := "string" // default
		if prop.Type != nil {
			switch {
			case (*prop.Type).Is("string"):
				goType = "string"
			case (*prop.Type).Is("integer"):
				// Always use int64 for integers to match typed generator
				goType = "int64"
			case (*prop.Type).Is("number"):
				goType = "float64"
			case (*prop.Type).Is("boolean"):
				goType = "bool"
			default:
				// Skip complex types
				return false, nil
			}
		}

		// Convert snake_case to camelCase for parameter name
		paramName := toCamelCase(fieldName)
		// Make first letter lowercase for parameter
		if len(paramName) > 0 {
			paramName = strings.ToLower(string(paramName[0])) + paramName[1:]
		}

		// Check if required
		isRequired := false
		for _, req := range schema.Value.Required {
			if req == fieldName {
				isRequired = true
				break
			}
		}

		// Get description from OpenAPI schema
		description := ""
		if prop != nil && prop.Description != "" {
			description = prop.Description
		}

		params = append(params, SimplifiedParam{
			Name:        paramName,
			Type:        goType,
			BodyField:   fieldName,
			Required:    isRequired,
			Description: description,
		})
	}

	// Sort params to ensure consistent ordering (required first, then alphabetically)
	sort.Slice(params, func(i, j int) bool {
		if params[i].Required != params[j].Required {
			return params[i].Required // required params first
		}
		return params[i].Name < params[j].Name
	})

	return true, params
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
// summary: {{.Summary}}{{end}}{{if or .SimplifiedBody .IsAsyncTask}}
//
// Parameters:{{range .SimplifiedParams}}
//   - {{.Name}} (body): {{if .Description}}{{.Description}}{{else}}Request parameter{{end}}{{end}}{{if .IsAsyncTask}}
//   - waitTimeout: If 0, returns immediately without waiting (async). Otherwise, waits for task completion with the specified timeout.{{end}}{{end}}
func ({{$.ReceiverName}} *{{$.Name}}) {{.Name}}WithContext_{{.HTTPMethod}}(ctx context.Context{{if .HasID}}, id any{{end}}{{if .HasParams}}, params core.Params{{end}}{{if .SimplifiedBody}}{{range $i, $p := .SimplifiedParams}}, {{$p.Name}} {{$p.Type}}{{end}}{{else}}{{if .HasBody}}, body core.Params{{end}}{{end}}{{if .IsAsyncTask}}, waitTimeout time.Duration{{end}}) ({{if .IsAsyncTask}}*AsyncResult, error{{else}}{{if .ReturnsNoContent}}error{{else}}{{if .ReturnsArray}}core.RecordSet, error{{else}}core.Record, error{{end}}{{end}}{{end}}) {
	{{if .HasID}}resourcePath := core.BuildResourcePathWithID("{{.ResourcePath}}", id{{if .SubPath}}, "{{.SubPath}}"{{end}})
	{{else}}resourcePath := "{{.Path}}"
	{{end}}{{if .SimplifiedBody}}body := core.Params{}
	{{range .SimplifiedParams}}{{if .Required}}body["{{.BodyField}}"] = {{.Name}}
	{{else}}if {{.Name}} != {{if eq .Type "string"}}""{{else if eq .Type "bool"}}false{{else}}0{{end}} {
		body["{{.BodyField}}"] = {{.Name}}
	}
	{{end}}{{end}}{{end}}{{if .IsAsyncTask}}result, err := core.Request[core.Record](ctx, {{$.ReceiverName}}, http.{{.GoHTTPMethod}}, resourcePath, {{if .HasParams}}params{{else}}nil{{end}}, {{if or .HasBody .SimplifiedBody}}body{{else}}nil{{end}})
	if err != nil {
		return nil, err
	}

	return MaybeWaitAsyncResultWithContext(ctx, result, {{$.ReceiverName}}.Rest, waitTimeout)
	{{else}}{{if .ReturnsNoContent}}_, err := core.Request[core.Record](ctx, {{$.ReceiverName}}, http.{{.GoHTTPMethod}}, resourcePath, {{if .HasParams}}params{{else}}nil{{end}}, {{if or .HasBody .SimplifiedBody}}body{{else}}nil{{end}})
	return err
	{{else}}{{if .ReturnsArray}}result, err := core.Request[core.RecordSet](ctx, {{$.ReceiverName}}, http.{{.GoHTTPMethod}}, resourcePath, {{if .HasParams}}params{{else}}nil{{end}}, {{if or .HasBody .SimplifiedBody}}body{{else}}nil{{end}})
	if err != nil {
		return nil, err
	}
	return result, nil
	{{else}}result, err := core.Request[core.Record](ctx, {{$.ReceiverName}}, http.{{.GoHTTPMethod}}, resourcePath, {{if .HasParams}}params{{else}}nil{{end}}, {{if or .HasBody .SimplifiedBody}}body{{else}}nil{{end}})
	if err != nil {
		return nil, err
	}
	return result, nil{{end}}{{end}}{{end}}
}

// {{.Name}}_{{.HTTPMethod}}
// method: {{.HTTPMethod}}
// url: {{.Path}}{{if .Summary}}
// summary: {{.Summary}}{{end}}{{if or .SimplifiedBody .IsAsyncTask}}
//
// Parameters:{{range .SimplifiedParams}}
//   - {{.Name}} (body): {{if .Description}}{{.Description}}{{else}}Request parameter{{end}}{{end}}{{if .IsAsyncTask}}
//   - waitTimeout: If 0, returns immediately without waiting (async). Otherwise, waits for task completion with the specified timeout.{{end}}{{end}}
func ({{$.ReceiverName}} *{{$.Name}}) {{.Name}}_{{.HTTPMethod}}({{if .HasID}}id any, {{end}}{{if .HasParams}}params core.Params, {{end}}{{if .SimplifiedBody}}{{range $i, $p := .SimplifiedParams}}{{if gt $i 0}}, {{end}}{{$p.Name}} {{$p.Type}}{{end}}{{if .IsAsyncTask}}, {{end}}{{else}}{{if .HasBody}}body core.Params{{if .IsAsyncTask}}, {{end}}{{else}}{{if .IsAsyncTask}}{{end}}{{end}}{{end}}{{if .IsAsyncTask}}waitTimeout time.Duration{{end}}) ({{if .IsAsyncTask}}*AsyncResult, error{{else}}{{if .ReturnsNoContent}}error{{else}}{{if .ReturnsArray}}core.RecordSet, error{{else}}core.Record, error{{end}}{{end}}{{end}}) {
	return {{$.ReceiverName}}.{{.Name}}WithContext_{{.HTTPMethod}}({{$.ReceiverName}}.Rest.GetCtx(){{if .HasID}}, id{{end}}{{if .HasParams}}, params{{end}}{{if .SimplifiedBody}}{{range .SimplifiedParams}}, {{.Name}}{{end}}{{else}}{{if .HasBody}}, body{{end}}{{end}}{{if .IsAsyncTask}}, waitTimeout{{end}})
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
