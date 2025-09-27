package apibuilder

import (
	"fmt"
	"strings"

	"github.com/vast-data/go-vast-client/autogen/markers"
)

// ParseAPIBuilderMarkers parses apibuilder markers and organizes them into API endpoints
func ParseAPIBuilderMarkers(markerValues []markers.MarkerValue) (*APIBuilder, error) {
	builder := &APIBuilder{
		Endpoints: make(map[string]*APIEndpoint),
	}

	// Group markers by type name (assuming they're applied to types)
	typeMarkers := make(map[string][]markers.MarkerValue)

	for _, marker := range markerValues {
		if strings.HasPrefix(marker.Name, "apibuilder:") {
			// Extract type name from the marker's context
			// For now, we'll use the marker name as a key, but in a real implementation
			// you'd extract this from the AST node
			typeName := extractTypeNameFromMarker(marker)
			typeMarkers[typeName] = append(typeMarkers[typeName], marker)
		}
	}

	// Process each type's markers
	for typeName, markers := range typeMarkers {
		endpoint := &APIEndpoint{}

		for _, marker := range markers {
			if err := applyMarkerToEndpoint(marker, endpoint); err != nil {
				return nil, fmt.Errorf("failed to apply marker %s to type %s: %w", marker.Name, typeName, err)
			}
		}

		builder.Endpoints[typeName] = endpoint
	}

	return builder, nil
}

// extractTypeNameFromMarker extracts the type name from a marker
// In a real implementation, this would use the AST node information
func extractTypeNameFromMarker(marker markers.MarkerValue) string {
	// For demo purposes, we'll generate a name based on the marker
	// In reality, you'd get this from marker.Node (the AST node)
	return fmt.Sprintf("Type_%d", marker.Position)
}

// applyMarkerToEndpoint applies a single marker to an API endpoint
func applyMarkerToEndpoint(marker markers.MarkerValue, endpoint *APIEndpoint) error {
	switch {
	case strings.HasPrefix(marker.Name, "apibuilder:requestUrl:"):
		// Extract method from marker name: "apibuilder:requestUrl:GET" -> "GET"
		parts := strings.Split(marker.Name, ":")
		if len(parts) < 3 {
			return fmt.Errorf("invalid requestUrl marker name: %s", marker.Name)
		}
		method := parts[2]

		// The marker value is the URL string
		if url, ok := marker.Value.(string); ok {
			endpoint.RequestURL = &RequestURL{
				Method: method,
				URL:    url,
			}
		} else {
			return fmt.Errorf("invalid RequestURL marker value: %+v (expected string)", marker.Value)
		}

	case strings.HasPrefix(marker.Name, "apibuilder:responseUrl:"):
		// Extract method from marker name: "apibuilder:responseUrl:POST" -> "POST"
		parts := strings.Split(marker.Name, ":")
		if len(parts) < 3 {
			return fmt.Errorf("invalid responseUrl marker name: %s", marker.Name)
		}
		method := parts[2]

		// The marker value is the URL string
		if url, ok := marker.Value.(string); ok {
			endpoint.ResponseURL = &ResponseURL{
				Method: method,
				URL:    url,
			}
		} else {
			return fmt.Errorf("invalid ResponseURL marker value: %+v (expected string)", marker.Value)
		}

	case marker.Name == "apibuilder:requestModel":
		// The marker value is the model name string
		if model, ok := marker.Value.(string); ok {
			endpoint.RequestModel = &RequestModel{
				Model: model,
			}
		} else {
			return fmt.Errorf("invalid RequestModel marker value: %+v (expected string)", marker.Value)
		}

	case marker.Name == "apibuilder:responseModel":
		// The marker value is the model name string
		if model, ok := marker.Value.(string); ok {
			endpoint.ResponseModel = &ResponseModel{
				Model: model,
			}
		} else {
			return fmt.Errorf("invalid ResponseModel marker value: %+v (expected string)", marker.Value)
		}

	default:
		return fmt.Errorf("unknown apibuilder marker: %s", marker.Name)
	}

	return nil
}

// GenerateAPISpec generates an API specification from the parsed markers
func (b *APIBuilder) GenerateAPISpec() map[string]interface{} {
	spec := make(map[string]interface{})

	endpoints := make([]map[string]interface{}, 0, len(b.Endpoints))

	for typeName, endpoint := range b.Endpoints {
		endpointSpec := map[string]interface{}{
			"type": typeName,
		}

		if endpoint.RequestURL != nil {
			endpointSpec["request"] = map[string]interface{}{
				"method": endpoint.RequestURL.Method,
				"url":    endpoint.RequestURL.URL,
			}
		}

		if endpoint.ResponseURL != nil {
			endpointSpec["response"] = map[string]interface{}{
				"method": endpoint.ResponseURL.Method,
				"url":    endpoint.ResponseURL.URL,
			}
		}

		if endpoint.RequestModel != nil {
			endpointSpec["requestModel"] = endpoint.RequestModel.Model
		}

		if endpoint.ResponseModel != nil {
			endpointSpec["responseModel"] = endpoint.ResponseModel.Model
		}

		endpoints = append(endpoints, endpointSpec)
	}

	spec["endpoints"] = endpoints
	spec["count"] = len(endpoints)

	return spec
}

// ValidateEndpoint validates that an endpoint has the required markers
func (endpoint *APIEndpoint) Validate() error {
	var errors []string

	if endpoint.RequestURL == nil && endpoint.ResponseURL == nil {
		errors = append(errors, "endpoint must have at least one URL (request or response)")
	}

	if endpoint.RequestURL != nil && endpoint.RequestURL.URL == "" {
		errors = append(errors, "request URL cannot be empty")
	}

	if endpoint.ResponseURL != nil && endpoint.ResponseURL.URL == "" {
		errors = append(errors, "response URL cannot be empty")
	}

	if len(errors) > 0 {
		return fmt.Errorf("endpoint validation failed: %s", strings.Join(errors, ", "))
	}

	return nil
}
