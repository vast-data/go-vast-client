package vastparser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/vast-data/go-vast-client/codegen/apibuilder"
)

// RestResourceConfig represents a resource configuration parsed from untyped_rest.go
type RestResourceConfig struct {
	Name         string                   // Type name (e.g., "Alarm")
	ResourcePath string                   // API path (e.g., "alarms")
	Operations   string                   // CRUD string (e.g., "RUD")
	ExtraMethods []apibuilder.ExtraMethod // Extra methods from field comments
}

// RestParser parses rest/untyped_rest.go to extract resource configurations
type RestParser struct {
	resourceConfigs map[string]*RestResourceConfig // Key: resource type name
}

// NewRestParser creates a new RestParser
func NewRestParser() *RestParser {
	return &RestParser{
		resourceConfigs: make(map[string]*RestResourceConfig),
	}
}

// ParseRestFile parses rest/untyped_rest.go and extracts all newUntypedResource calls
// and field markers from the UntypedVMSRest struct
func (p *RestParser) ParseRestFile(filename string) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	// First, parse field markers from UntypedVMSRest struct
	p.parseFieldMarkers(file)

	// Then, walk the AST to find newUntypedResource calls
	ast.Inspect(file, func(n ast.Node) bool {
		// Look for assignment statements like:
		// rest.Alarms = newUntypedResource[untyped.Alarm](rest, "alarms", R, U, D)
		assignStmt, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}

		// Check if RHS is a function call
		if len(assignStmt.Rhs) != 1 {
			return true
		}

		callExpr, ok := assignStmt.Rhs[0].(*ast.CallExpr)
		if !ok {
			return true
		}

		// Check if the function is newUntypedResource with type parameters
		indexExpr, ok := callExpr.Fun.(*ast.IndexExpr)
		if !ok {
			// Try IndexListExpr for Go 1.18+ with multiple type parameters
			indexListExpr, ok := callExpr.Fun.(*ast.IndexListExpr)
			if !ok {
				return true
			}
			indexExpr = &ast.IndexExpr{
				X:     indexListExpr.X,
				Index: indexListExpr.Indices[0],
			}
		}

		// Check if function name is "newUntypedResource"
		ident, ok := indexExpr.X.(*ast.Ident)
		if !ok || ident.Name != "newUntypedResource" {
			return true
		}

		// Extract type parameter (e.g., untyped.Alarm)
		var typeName string
		switch typeArg := indexExpr.Index.(type) {
		case *ast.SelectorExpr:
			// untyped.Alarm -> "Alarm"
			typeName = typeArg.Sel.Name
		case *ast.Ident:
			// Just Alarm (unlikely but handle it)
			typeName = typeArg.Name
		default:
			return true
		}

		// Extract function arguments
		if len(callExpr.Args) < 2 {
			return true // Need at least rest and resourcePath
		}

		// Second argument is resource path (string literal)
		resourcePath := ""
		if lit, ok := callExpr.Args[1].(*ast.BasicLit); ok && lit.Kind == token.STRING {
			// Remove quotes from string literal
			resourcePath = strings.Trim(lit.Value, `"`)
		}

		// Remaining arguments are CRUD flags (C, R, U, D)
		var operations []string
		for i := 2; i < len(callExpr.Args); i++ {
			if ident, ok := callExpr.Args[i].(*ast.Ident); ok {
				operations = append(operations, ident.Name)
			}
		}

		// Convert CRUD flags to operations string
		opsString := strings.Join(operations, "")

		// Check if config already exists (from field markers parsing)
		if existingConfig, exists := p.resourceConfigs[typeName]; exists {
			// Merge: keep ExtraMethods, update Name/ResourcePath/Operations
			existingConfig.Name = typeName
			existingConfig.ResourcePath = resourcePath
			existingConfig.Operations = opsString
		} else {
			// Create new configuration
			p.resourceConfigs[typeName] = &RestResourceConfig{
				Name:         typeName,
				ResourcePath: resourcePath,
				Operations:   opsString,
				ExtraMethods: []apibuilder.ExtraMethod{},
			}
		}

		return true
	})

	return nil
}

// GetResourceConfig returns the configuration for a given resource type name
func (p *RestParser) GetResourceConfig(resourceName string) (*RestResourceConfig, bool) {
	config, exists := p.resourceConfigs[resourceName]
	return config, exists
}

// GetAllConfigs returns all parsed resource configurations
func (p *RestParser) GetAllConfigs() map[string]*RestResourceConfig {
	return p.resourceConfigs
}

// parseFieldMarkers parses field comments from the UntypedVMSRest struct
func (p *RestParser) parseFieldMarkers(file *ast.File) {
	ast.Inspect(file, func(n ast.Node) bool {
		// Look for struct declarations
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok || typeSpec.Name.Name != "UntypedVMSRest" {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		// Iterate through fields
		for _, field := range structType.Fields.List {
			if field.Doc == nil || len(field.Names) == 0 {
				continue
			}

			fieldName := field.Names[0].Name

			// Skip internal fields (lowercase or starting with dummy)
			if fieldName == "ctx" || fieldName == "Session" || fieldName == "resourceMap" || fieldName == "dummy" {
				continue
			}

			// Extract the actual type name from field.Type (e.g., "*untyped.SupportBundles" -> "SupportBundles")
			typeName := p.extractTypeName(field.Type)
			if typeName == "" {
				continue
			}

			// Parse markers from field comments
			for _, comment := range field.Doc.List {
				text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))

				// Check for apiall:extraMethod markers
				if strings.HasPrefix(text, "+apiall:extraMethod:") {
					// Parse the marker (e.g., "GET=/hosts/discover/" or "GET|POST=/bigcatalogconfig/query_data/")
					marker := strings.TrimPrefix(text, "+apiall:extraMethod:")

					// Split by = to get methods and path
					parts := strings.SplitN(marker, "=", 2)
					if len(parts) != 2 {
						continue
					}

					methodsStr := parts[0]
					path := parts[1]

					// Split methods by | (e.g., "GET|POST" -> ["GET", "POST"])
					methods := strings.Split(methodsStr, "|")

					// Find or create config for this resource using the actual type name
					config := p.findOrCreateConfigByTypeName(typeName)

					// Add extra method for each HTTP method
					for _, method := range methods {
						method = strings.TrimSpace(method)
						if method != "" {
							config.ExtraMethods = append(config.ExtraMethods, apibuilder.ExtraMethod{
								Method: method,
								Path:   path,
							})
						}
					}
				}
			}
		}

		return false // Found the struct, no need to continue
	})
}

// extractTypeName extracts the type name from a field's type expression
// e.g., "*untyped.SupportBundles" -> "SupportBundles"
func (p *RestParser) extractTypeName(typeExpr ast.Expr) string {
	// Handle pointer types: *untyped.SupportBundles
	if starExpr, ok := typeExpr.(*ast.StarExpr); ok {
		typeExpr = starExpr.X
	}

	// Handle selector expressions: untyped.SupportBundles
	if selExpr, ok := typeExpr.(*ast.SelectorExpr); ok {
		return selExpr.Sel.Name
	}

	// Handle simple identifiers: SupportBundles (if no package qualifier)
	if ident, ok := typeExpr.(*ast.Ident); ok {
		return ident.Name
	}

	return ""
}

// findOrCreateConfigByTypeName finds or creates a config entry by type name
func (p *RestParser) findOrCreateConfigByTypeName(typeName string) *RestResourceConfig {
	// Check if config already exists
	if config, exists := p.resourceConfigs[typeName]; exists {
		return config
	}

	// Create new config (will be filled in by newUntypedResource parsing)
	config := &RestResourceConfig{
		Name:         typeName,
		ExtraMethods: []apibuilder.ExtraMethod{},
	}
	p.resourceConfigs[typeName] = config
	return config
}

// ConvertToOperations converts a RestResourceConfig to apibuilder.Operations
func (config *RestResourceConfig) ConvertToOperations() *apibuilder.Operations {
	if config == nil {
		return nil
	}

	return &apibuilder.Operations{
		Operations: config.Operations,
		URL:        config.ResourcePath,
	}
}
