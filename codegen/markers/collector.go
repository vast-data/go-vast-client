package markers

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"sync"
)

// Collector collects and parses marker comments from Go source code.
type Collector struct {
	Registry *Registry

	// Cache parsed results by file path
	cache map[string][]MarkerValue
	mu    sync.RWMutex
}

// NewCollector creates a new marker collector with the given registry.
func NewCollector(registry *Registry) *Collector {
	return &Collector{
		Registry: registry,
		cache:    make(map[string][]MarkerValue),
	}
}

// ParseFile parses all markers in a Go source file.
func (c *Collector) ParseFile(filename string) ([]MarkerValue, error) {
	// Check cache first
	c.mu.RLock()
	if cached, exists := c.cache[filename]; exists {
		c.mu.RUnlock()
		return cached, nil
	}
	c.mu.RUnlock()

	// Parse the file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filename, err)
	}

	markers := c.parseFileAST(file, fset)

	// Cache the results
	c.mu.Lock()
	c.cache[filename] = markers
	c.mu.Unlock()

	return markers, nil
}

// ParseSource parses markers from Go source code provided as a string.
func (c *Collector) ParseSource(filename string, src string) ([]MarkerValue, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source: %w", err)
	}

	return c.parseFileAST(file, fset), nil
}

// ParseDirectory parses all Go files in a directory and returns markers grouped by file.
func (c *Collector) ParseDirectory(dir string) (map[string][]MarkerValue, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory %s: %w", dir, err)
	}

	result := make(map[string][]MarkerValue)

	for _, pkg := range pkgs {
		for filename, file := range pkg.Files {
			markers := c.parseFileAST(file, fset)
			if len(markers) > 0 {
				result[filename] = markers
			}
		}
	}

	return result, nil
}

// EachType calls the callback for each type found in the file with its markers.
func (c *Collector) EachType(filename string, callback TypeCallback) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file %s: %w", filename, err)
	}

	// Collect all markers by AST node
	nodeMarkers := c.collectNodeMarkers(file)

	// Walk through type declarations
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.GenDecl:
			if node.Tok == token.TYPE {
				for _, spec := range node.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						typeInfo := c.buildTypeInfo(typeSpec, node, file, nodeMarkers)
						callback(typeInfo)
					}
				}
			}
		}
		return true
	})

	return nil
}

// parseFileAST parses markers from an AST file.
func (c *Collector) parseFileAST(file *ast.File, fset *token.FileSet) []MarkerValue {
	var markers []MarkerValue

	// Collect markers by AST node
	nodeMarkers := c.collectNodeMarkers(file)

	// Convert to MarkerValue structs
	for node, comments := range nodeMarkers {
		target := c.getTargetType(node)

		for _, comment := range comments {
			markerText := extractMarkerText(comment.Text)
			if !strings.HasPrefix(markerText, "+") {
				continue
			}

			// Look up marker definition
			def := c.Registry.Lookup(markerText, target)
			if def == nil {
				continue // Unknown marker, skip
			}

			// Parse marker value
			value, err := def.Parse(markerText)
			if err != nil {
				continue // Parse error, skip
			}

			markers = append(markers, MarkerValue{
				Name:     def.Name,
				Value:    value,
				Node:     node,
				Target:   target,
				Position: comment.Pos(),
			})
		}
	}

	return markers
}

// collectNodeMarkers associates comments with AST nodes.
func (c *Collector) collectNodeMarkers(file *ast.File) map[ast.Node][]*ast.Comment {
	nodeMarkers := make(map[ast.Node][]*ast.Comment)

	// Create a map of positions to comments
	commentMap := make(map[token.Pos]*ast.Comment)
	for _, commentGroup := range file.Comments {
		for _, comment := range commentGroup.List {
			if isMarkerComment(comment.Text) {
				commentMap[comment.Pos()] = comment
			}
		}
	}

	// Walk the AST and associate comments with nodes
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		// Look for comments that precede this node
		var associatedComments []*ast.Comment

		// Check for doc comments
		switch node := n.(type) {
		case *ast.GenDecl:
			if node.Doc != nil {
				for _, comment := range node.Doc.List {
					if isMarkerComment(comment.Text) {
						associatedComments = append(associatedComments, comment)
					}
				}
			}
		case *ast.TypeSpec:
			if node.Doc != nil {
				for _, comment := range node.Doc.List {
					if isMarkerComment(comment.Text) {
						associatedComments = append(associatedComments, comment)
					}
				}
			}
		case *ast.Field:
			if node.Doc != nil {
				for _, comment := range node.Doc.List {
					if isMarkerComment(comment.Text) {
						associatedComments = append(associatedComments, comment)
					}
				}
			}
		}

		if len(associatedComments) > 0 {
			nodeMarkers[n] = associatedComments
		}

		return true
	})

	// Also associate file-level comments with the file node
	var fileComments []*ast.Comment
	if file.Doc != nil {
		for _, comment := range file.Doc.List {
			if isMarkerComment(comment.Text) {
				fileComments = append(fileComments, comment)
			}
		}
	}
	if len(fileComments) > 0 {
		nodeMarkers[file] = fileComments
	}

	return nodeMarkers
}

// getTargetType determines the target type for an AST node.
func (c *Collector) getTargetType(node ast.Node) TargetType {
	switch node.(type) {
	case *ast.File:
		return DescribesPackage
	case *ast.Field:
		return DescribesField
	case *ast.TypeSpec, *ast.GenDecl:
		return DescribesType
	default:
		return DescribesType // Default to type-level
	}
}

// buildTypeInfo creates a TypeInfo from an AST type spec.
func (c *Collector) buildTypeInfo(typeSpec *ast.TypeSpec, genDecl *ast.GenDecl, file *ast.File, nodeMarkers map[ast.Node][]*ast.Comment) *TypeInfo {
	// Get markers for this type
	typeMarkers := c.buildMarkerValues(typeSpec, nodeMarkers)

	// Also include markers from the GenDecl if it's a single type declaration
	if len(genDecl.Specs) == 1 {
		declMarkers := c.buildMarkerValues(genDecl, nodeMarkers)
		for name, values := range declMarkers {
			typeMarkers[name] = append(typeMarkers[name], values...)
		}
	}

	// Build field info if this is a struct
	var fields []FieldInfo
	if structType, ok := typeSpec.Type.(*ast.StructType); ok {
		for _, field := range structType.Fields.List {
			fieldMarkers := c.buildMarkerValues(field, nodeMarkers)

			// Handle both named and anonymous fields
			if len(field.Names) > 0 {
				for _, name := range field.Names {
					fields = append(fields, FieldInfo{
						Name:     name.Name,
						Markers:  fieldMarkers,
						Tag:      reflect.StructTag(c.parseStructTag(field.Tag)),
						Doc:      c.extractDoc(field.Doc),
						RawField: field,
					})
				}
			} else {
				// Anonymous field
				fields = append(fields, FieldInfo{
					Name:     "",
					Markers:  fieldMarkers,
					Tag:      reflect.StructTag(c.parseStructTag(field.Tag)),
					Doc:      c.extractDoc(field.Doc),
					RawField: field,
				})
			}
		}
	}

	return &TypeInfo{
		Name:    typeSpec.Name.Name,
		Markers: typeMarkers,
		Fields:  fields,
		Doc:     c.extractDoc(typeSpec.Doc),
		RawDecl: genDecl,
		RawSpec: typeSpec,
		RawFile: file,
	}
}

// buildMarkerValues builds MarkerValues from AST node comments.
func (c *Collector) buildMarkerValues(node ast.Node, nodeMarkers map[ast.Node][]*ast.Comment) MarkerValues {
	values := make(MarkerValues)

	comments, exists := nodeMarkers[node]
	if !exists {
		return values
	}

	target := c.getTargetType(node)

	for _, comment := range comments {
		markerText := extractMarkerText(comment.Text)
		if !strings.HasPrefix(markerText, "+") {
			continue
		}

		def := c.Registry.Lookup(markerText, target)
		if def == nil {
			continue
		}

		value, err := def.Parse(markerText)
		if err != nil {
			continue
		}

		values[def.Name] = append(values[def.Name], value)
	}

	return values
}

// parseStructTag parses a struct tag into a reflect.StructTag.
func (c *Collector) parseStructTag(tag *ast.BasicLit) string {
	if tag == nil {
		return ""
	}
	// Remove backticks
	tagStr := tag.Value
	if len(tagStr) >= 2 && tagStr[0] == '`' && tagStr[len(tagStr)-1] == '`' {
		return tagStr[1 : len(tagStr)-1]
	}
	return tagStr
}

// extractDoc extracts documentation text from a comment group.
func (c *Collector) extractDoc(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}

	var lines []string
	for _, comment := range doc.List {
		line := comment.Text
		if strings.HasPrefix(line, "//") {
			line = strings.TrimSpace(line[2:])
		} else if strings.HasPrefix(line, "/*") && strings.HasSuffix(line, "*/") {
			line = strings.TrimSpace(line[2 : len(line)-2])
		}
		if line != "" {
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

// ClearCache clears the internal cache of parsed files.
func (c *Collector) ClearCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string][]MarkerValue)
}
