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

	"github.com/vast-data/go-vast-client/codegen/apibuilder"
	"github.com/vast-data/go-vast-client/codegen/vastparser"
)

// MethodInfo represents information about a generated method
type MethodInfo struct {
	ResourceName   string
	Name           string
	HTTPMethod     string
	GoHTTPMethod   string
	Path           string
	ResourcePath   string
	SubPath        string
	HasID          bool
	HasParams      bool
	HasBody        bool
	ReceiverName   string
}

// UntypedResourceData represents data for untyped resource template generation
type UntypedResourceData struct {
	Name         string
	LowerName    string
	ReceiverName string
	Methods      []MethodInfo
	Resource     *vastparser.UntypedResource
}

func main() {
	// Paths
	inputDir := "../resources/untyped"
	outputDir := "../resources/untyped"

	// Parse all Go files in the untyped directory
	parser := vastparser.NewUntypedResourceParser()
	
	files, err := filepath.Glob(filepath.Join(inputDir, "*.go"))
	if err != nil {
		log.Fatalf("Failed to find Go files: %v", err)
	}

	var allResources []vastparser.UntypedResource
	fmt.Printf("Scanning %d files in %s\n", len(files), inputDir)
	for _, file := range files {
		// Skip autogen files
		if strings.HasSuffix(file, "_autogen.go") {
			continue
		}

		fmt.Printf("Parsing: %s\n", filepath.Base(file))
		resources, err := parser.ParseFile(file)
		if err != nil {
			log.Printf("Warning: Failed to parse file %s: %v", file, err)
			continue
		}
		if len(resources) > 0 {
			fmt.Printf("  Found %d resources with apiuntyped markers\n", len(resources))
		}
		allResources = append(allResources, resources...)
	}

	if len(allResources) == 0 {
		log.Println("No resources with apiuntyped markers found")
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
			fmt.Printf("  ✅ Generated method: %s\n", methodInfo.Name)
		}

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

	// Convert HTTP method to Go constant (e.g., "PATCH" -> "MethodPatch")
	methodInfo.GoHTTPMethod = httpMethodToGoConstant(extraMethod.Method)

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

	// Add HTTP method prefix based on method type
	var methodPrefix string
	switch extraMethod.Method {
	case "GET":
		methodPrefix = "Get"
	case "POST":
		methodPrefix = "Create"
	case "PUT":
		methodPrefix = "Update"
	case "PATCH":
		methodPrefix = "Update"
	case "DELETE":
		methodPrefix = "Delete"
	default:
		methodPrefix = extraMethod.Method
	}

	methodInfo.Name = resourceName + methodPrefix + action

	// Determine if method has params and body based on HTTP method
	methodInfo.HasParams = extraMethod.Method == "GET" || extraMethod.Method == "DELETE"
	methodInfo.HasBody = extraMethod.Method == "POST" || extraMethod.Method == "PUT" || extraMethod.Method == "PATCH"

	return methodInfo
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
		return "Method" + strings.Title(strings.ToLower(method))
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

	"github.com/vast-data/go-vast-client/core"
)

{{range .Methods}}
// {{.Name}}WithContext
// method: {{.HTTPMethod}}
// url: {{.Path}}
func ({{$.ReceiverName}} *{{$.Name}}) {{.Name}}WithContext(ctx context.Context{{if .HasID}}, id any{{end}}{{if .HasParams}}, params core.Params{{end}}{{if .HasBody}}, body core.Params{{end}}) (core.Record, error) {
	{{if .HasID}}path := core.BuildResourcePathWithID("{{.ResourcePath}}", id{{if .SubPath}}, "{{.SubPath}}"{{end}})
	{{else}}path := "{{.Path}}"
	{{end}}result, err := core.Request[core.Record](ctx, {{$.ReceiverName}}, http.{{.GoHTTPMethod}}, path, {{if .HasParams}}params{{else}}nil{{end}}, {{if .HasBody}}body{{else}}nil{{end}})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// {{.Name}}
// method: {{.HTTPMethod}}
// url: {{.Path}}
func ({{$.ReceiverName}} *{{$.Name}}) {{.Name}}({{if .HasID}}id any{{end}}{{if and .HasID (or .HasParams .HasBody)}}, {{end}}{{if .HasParams}}params core.Params{{end}}{{if and .HasParams .HasBody}}, {{end}}{{if .HasBody}}body core.Params{{end}}) (core.Record, error) {
	return {{$.ReceiverName}}.{{.Name}}WithContext({{$.ReceiverName}}.Rest.GetCtx(){{if .HasID}}, id{{end}}{{if .HasParams}}, params{{end}}{{if .HasBody}}, body{{end}})
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
