package vastparser

import (
	"fmt"
	"strings"

	"github.com/vast-data/go-vast-client/codegen/apibuilder"
	"github.com/vast-data/go-vast-client/codegen/markers"
)

// UntypedResource represents a parsed untyped resource with its extra methods
type UntypedResource struct {
	Name         string                    `json:"name"`
	ExtraMethods []apibuilder.ExtraMethod  `json:"extraMethods,omitempty"`
	AllMarkers   []markers.MarkerValue     `json:"allMarkers,omitempty"`
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
		if !isAPIUntypedMarker(markerName) {
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
		if strings.HasPrefix(markerName, "apiuntyped:extraMethod:") {
			// Extract HTTP method from marker name (e.g., "apiuntyped:extraMethod:PATCH" -> "PATCH")
			method := strings.TrimPrefix(markerName, "apiuntyped:extraMethod:")
			p.addExtraMethod(resource, method, values)
		}
	}

	// Only return resources that have APIUntyped markers
	if !hasAPIUntypedMarkers {
		return nil
	}

	return resource
}

// addExtraMethod adds an extra method to the resource
func (p *UntypedResourceParser) addExtraMethod(resource *UntypedResource, method string, values []interface{}) {
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

// isAPIUntypedMarker checks if a marker name is an APIUntyped marker
func isAPIUntypedMarker(markerName string) bool {
	// Check if it starts with the apiuntyped prefix
	return strings.HasPrefix(markerName, "apiuntyped:")
}

// HasExtraMethod checks if the resource has any extra methods
func (r *UntypedResource) HasExtraMethod() bool {
	return len(r.ExtraMethods) > 0
}

// GetExtraMethods returns all extra methods for this resource
func (r *UntypedResource) GetExtraMethods() []apibuilder.ExtraMethod {
	return r.ExtraMethods
}
