package vastparser

import (
	"fmt"
	"strings"

	"github.com/vast-data/go-vast-client/codegen/apibuilder"
	"github.com/vast-data/go-vast-client/codegen/markers"
)

// UntypedResource represents a parsed untyped resource with its extra methods
type UntypedResource struct {
	Name         string                   `json:"name"`
	ExtraMethods []apibuilder.ExtraMethod `json:"extraMethods,omitempty"`
	AllMarkers   []markers.MarkerValue    `json:"allMarkers,omitempty"`
}

// UntypedResourceParser parses untyped resource files
type UntypedResourceParser struct {
	registry  *markers.Registry
	collector *markers.Collector
}

// NewUntypedResourceParser creates a new parser for untyped resources
func NewUntypedResourceParser() *UntypedResourceParser {
	registry := markers.NewRegistry()
	apibuilder.MustRegisterAPIUntypedMarkers(registry)

	return &UntypedResourceParser{
		registry:  registry,
		collector: markers.NewCollector(registry),
	}
}

// ParseFile parses a Go file and returns all untyped resources with apiuntyped markers
func (p *UntypedResourceParser) ParseFile(filename string) ([]UntypedResource, error) {
	var resources []UntypedResource

	err := p.collector.EachType(filename, func(typeInfo *markers.TypeInfo) {
		resource := p.parseTypeInfo(typeInfo)
		if resource != nil {
			resources = append(resources, *resource)
		}
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filename, err)
	}

	return resources, nil
}

// parseTypeInfo converts a TypeInfo into an UntypedResource if it has apiuntyped markers
func (p *UntypedResourceParser) parseTypeInfo(typeInfo *markers.TypeInfo) *UntypedResource {
	resource := &UntypedResource{
		Name: typeInfo.Name,
	}

	hasAPIUntypedMarkers := false

	// Process all markers
	for markerName, values := range typeInfo.Markers {
		// Parse out any options from the marker name (e.g., [wait=3m])
		cleanMarkerName, waitTimeout := parseExtraMethodOptions(markerName)

		if !isAPIUntypedMarker(cleanMarkerName) {
			continue
		}

		hasAPIUntypedMarkers = true

		// Add to AllMarkers
		for _, value := range values {
			resource.AllMarkers = append(resource.AllMarkers, markers.MarkerValue{
				Name:   markerName,
				Value:  value,
				Target: markers.DescribesType,
			})
		}

		// Process specific marker types
		// Handle apiuntyped:extraMethod:METHOD markers
		if strings.HasPrefix(cleanMarkerName, "apiuntyped:extraMethod:") {
			// Extract HTTP method from marker name (e.g., "apiuntyped:extraMethod:PATCH" -> "PATCH")
			methodsStr := strings.TrimPrefix(cleanMarkerName, "apiuntyped:extraMethod:")
			// Support multiple methods separated by | (e.g., "POST|PATCH|DELETE")
			methods := strings.Split(methodsStr, "|")
			for _, method := range methods {
				method = strings.TrimSpace(method)
				if method != "" {
					p.addExtraMethod(resource, method, waitTimeout, values)
				}
			}
		}

		// Handle apiall:extraMethod:METHOD markers (generates both typed and untyped)
		if strings.HasPrefix(cleanMarkerName, "apiall:extraMethod:") {
			methodsStr := strings.TrimPrefix(cleanMarkerName, "apiall:extraMethod:")
			// Support multiple methods separated by | (e.g., "POST|PATCH|DELETE")
			methods := strings.Split(methodsStr, "|")
			for _, method := range methods {
				method = strings.TrimSpace(method)
				if method != "" {
					p.addExtraMethod(resource, method, waitTimeout, values)
				}
			}
		}
	}

	// Only return resources that have APIUntyped markers
	if !hasAPIUntypedMarkers {
		return nil
	}

	return resource
}

// addExtraMethod adds an extra method to the resource
func (p *UntypedResourceParser) addExtraMethod(resource *UntypedResource, method string, waitTimeout string, values []interface{}) {
	for _, value := range values {
		// The value should be a string path (e.g., "/users/{id}/tenant_data/")
		if path, ok := value.(string); ok {
			extraMethod := apibuilder.ExtraMethod{
				Method: method,
				Path:   path,
			}
			resource.ExtraMethods = append(resource.ExtraMethods, extraMethod)
		}
	}
}

// isAPIUntypedMarker checks if a marker name is an APIUntyped marker or apiall marker
func isAPIUntypedMarker(markerName string) bool {
	// Strip out [wait=...] or other options before checking
	cleanName, _ := parseExtraMethodOptions(markerName)

	// Check if it starts with the apiuntyped or apiall prefix
	return strings.HasPrefix(cleanName, "apiuntyped:") || strings.HasPrefix(cleanName, "apiall:")
}

// HasExtraMethod checks if the resource has any extra methods
func (r *UntypedResource) HasExtraMethod() bool {
	return len(r.ExtraMethods) > 0
}

// GetExtraMethods returns all extra methods for this resource
func (r *UntypedResource) GetExtraMethods() []apibuilder.ExtraMethod {
	return r.ExtraMethods
}
