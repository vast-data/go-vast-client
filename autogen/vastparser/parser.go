package vastparser

import (
	"fmt"

	"github.com/vast-data/go-vast-client/autogen/apibuilder"
	"github.com/vast-data/go-vast-client/autogen/markers"
)

// VastResource represents a parsed VastData resource with its APIBuilder markers
type VastResource struct {
	Name           string                    `json:"name"`
	RequestURLs    []apibuilder.RequestURL   `json:"requestUrls,omitempty"`
	ResponseURLs   []apibuilder.ResponseURL  `json:"responseUrls,omitempty"`
	SearchQueries  []apibuilder.SearchQuery  `json:"searchQueries,omitempty"`
	RequestBodies  []apibuilder.RequestBody  `json:"requestBodies,omitempty"`
	ResponseBodies []apibuilder.ResponseBody `json:"responseBodies,omitempty"`
	RequestModel   string                    `json:"requestModel,omitempty"`
	ResponseModel  string                    `json:"responseModel,omitempty"`
	AllMarkers     []markers.MarkerValue     `json:"allMarkers,omitempty"`
}

// VastResourceParser parses VastData resource files
type VastResourceParser struct {
	registry  *markers.Registry
	collector *markers.Collector
}

// NewVastResourceParser creates a new parser for VastData resources
func NewVastResourceParser() *VastResourceParser {
	registry := markers.NewRegistry()
	apibuilder.MustRegisterAPIBuilderMarkers(registry)

	return &VastResourceParser{
		registry:  registry,
		collector: markers.NewCollector(registry),
	}
}

// ParseFile parses a Go file and returns all VastData resources with APIBuilder markers
func (p *VastResourceParser) ParseFile(filename string) ([]VastResource, error) {
	var resources []VastResource

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

// parseTypeInfo converts a TypeInfo into a VastResource if it has APIBuilder markers
func (p *VastResourceParser) parseTypeInfo(typeInfo *markers.TypeInfo) *VastResource {
	resource := &VastResource{
		Name: typeInfo.Name,
	}

	hasAPIBuilderMarkers := false

	// Process all markers
	for markerName, values := range typeInfo.Markers {
		if !isAPIBuilderMarker(markerName) {
			continue
		}

		hasAPIBuilderMarkers = true

		// Add to AllMarkers
		for _, value := range values {
			resource.AllMarkers = append(resource.AllMarkers, markers.MarkerValue{
				Name:   markerName,
				Value:  value,
				Target: markers.DescribesType,
			})
		}

		// Process specific marker types
		switch markerName {
		case "apibuilder:requestUrl:GET":
			p.addRequestURL(resource, "GET", values)
		case "apibuilder:requestUrl:POST":
			p.addRequestURL(resource, "POST", values)
		case "apibuilder:requestUrl:PUT":
			p.addRequestURL(resource, "PUT", values)
		case "apibuilder:requestUrl:DELETE":
			p.addRequestURL(resource, "DELETE", values)
		case "apibuilder:requestUrl:PATCH":
			p.addRequestURL(resource, "PATCH", values)

		case "apibuilder:responseUrl:GET":
			p.addResponseURL(resource, "GET", values)
		case "apibuilder:responseUrl:POST":
			p.addResponseURL(resource, "POST", values)
		case "apibuilder:responseUrl:PUT":
			p.addResponseURL(resource, "PUT", values)
		case "apibuilder:responseUrl:DELETE":
			p.addResponseURL(resource, "DELETE", values)
		case "apibuilder:responseUrl:PATCH":
			p.addResponseURL(resource, "PATCH", values)

		case "apibuilder:searchQuery:GET":
			p.addSearchQuery(resource, "GET", values)
		case "apibuilder:searchQuery:SCHEMA":
			p.addSearchQuery(resource, "SCHEMA", values)

		case "apibuilder:requestBody:POST":
			p.addRequestBody(resource, "POST", values)
		case "apibuilder:requestBody:PUT":
			p.addRequestBody(resource, "PUT", values)
		case "apibuilder:requestBody:PATCH":
			p.addRequestBody(resource, "PATCH", values)
		case "apibuilder:requestBody:SCHEMA":
			p.addRequestBody(resource, "SCHEMA", values)

		case "apibuilder:responseBody:GET":
			p.addResponseBody(resource, "GET", values)
		case "apibuilder:responseBody:POST":
			p.addResponseBody(resource, "POST", values)
		case "apibuilder:responseBody:PUT":
			p.addResponseBody(resource, "PUT", values)
		case "apibuilder:responseBody:DELETE":
			p.addResponseBody(resource, "DELETE", values)
		case "apibuilder:responseBody:PATCH":
			p.addResponseBody(resource, "PATCH", values)
		case "apibuilder:responseBody:SCHEMA":
			p.addResponseBody(resource, "SCHEMA", values)

		case "apibuilder:requestModel":
			if len(values) > 0 {
				if model, ok := values[0].(string); ok {
					resource.RequestModel = model
				}
			}

		case "apibuilder:responseModel":
			if len(values) > 0 {
				if model, ok := values[0].(string); ok {
					resource.ResponseModel = model
				}
			}
		}
	}

	// Only return resources that have APIBuilder markers
	if !hasAPIBuilderMarkers {
		return nil
	}

	return resource
}

// addRequestURL adds a request URL to the resource
func (p *VastResourceParser) addRequestURL(resource *VastResource, method string, values []interface{}) {
	for _, value := range values {
		if url, ok := value.(string); ok {
			resource.RequestURLs = append(resource.RequestURLs, apibuilder.RequestURL{
				Method: method,
				URL:    url,
			})
		}
	}
}

// addResponseURL adds a response URL to the resource
func (p *VastResourceParser) addResponseURL(resource *VastResource, method string, values []interface{}) {
	for _, value := range values {
		if url, ok := value.(string); ok {
			resource.ResponseURLs = append(resource.ResponseURLs, apibuilder.ResponseURL{
				Method: method,
				URL:    url,
			})
		}
	}
}

// addSearchQuery adds a search query to the resource
func (p *VastResourceParser) addSearchQuery(resource *VastResource, method string, values []interface{}) {
	for _, value := range values {
		if url, ok := value.(string); ok {
			resource.SearchQueries = append(resource.SearchQueries, apibuilder.SearchQuery{
				Method: method,
				URL:    url,
			})
		}
	}
}

// addRequestBody adds a request body to the resource
func (p *VastResourceParser) addRequestBody(resource *VastResource, method string, values []interface{}) {
	for _, value := range values {
		if url, ok := value.(string); ok {
			resource.RequestBodies = append(resource.RequestBodies, apibuilder.RequestBody{
				Method: method,
				URL:    url,
			})
		}
	}
}

// addResponseBody adds a response body to the resource
func (p *VastResourceParser) addResponseBody(resource *VastResource, method string, values []interface{}) {
	for _, value := range values {
		if url, ok := value.(string); ok {
			resource.ResponseBodies = append(resource.ResponseBodies, apibuilder.ResponseBody{
				Method: method,
				URL:    url,
			})
		}
	}
}

// isAPIBuilderMarker checks if a marker name is an APIBuilder marker
func isAPIBuilderMarker(markerName string) bool {
	apiBuilderMarkers := []string{
		"apibuilder:requestUrl:GET",
		"apibuilder:requestUrl:POST",
		"apibuilder:requestUrl:PUT",
		"apibuilder:requestUrl:DELETE",
		"apibuilder:requestUrl:PATCH",
		"apibuilder:responseUrl:GET",
		"apibuilder:responseUrl:POST",
		"apibuilder:responseUrl:PUT",
		"apibuilder:responseUrl:DELETE",
		"apibuilder:responseUrl:PATCH",
		"apibuilder:searchQuery:GET",
		"apibuilder:searchQuery:SCHEMA",
		"apibuilder:requestBody:POST",
		"apibuilder:requestBody:PUT",
		"apibuilder:requestBody:PATCH",
		"apibuilder:requestBody:SCHEMA",
		"apibuilder:responseBody:GET",
		"apibuilder:responseBody:POST",
		"apibuilder:responseBody:PUT",
		"apibuilder:responseBody:DELETE",
		"apibuilder:responseBody:PATCH",
		"apibuilder:responseBody:SCHEMA",
		"apibuilder:requestModel",
		"apibuilder:responseModel",
	}

	for _, marker := range apiBuilderMarkers {
		if markerName == marker {
			return true
		}
	}
	return false
}

// GetAllMarkerNames returns all possible APIBuilder marker names
func GetAllMarkerNames() []string {
	return []string{
		"apibuilder:requestUrl:GET",
		"apibuilder:requestUrl:POST",
		"apibuilder:requestUrl:PUT",
		"apibuilder:requestUrl:DELETE",
		"apibuilder:requestUrl:PATCH",
		"apibuilder:responseUrl:GET",
		"apibuilder:responseUrl:POST",
		"apibuilder:responseUrl:PUT",
		"apibuilder:responseUrl:DELETE",
		"apibuilder:responseUrl:PATCH",
		"apibuilder:searchQuery:GET",
		"apibuilder:searchQuery:SCHEMA",
		"apibuilder:requestBody:POST",
		"apibuilder:requestBody:PUT",
		"apibuilder:requestBody:PATCH",
		"apibuilder:requestBody:SCHEMA",
		"apibuilder:responseBody:GET",
		"apibuilder:responseBody:POST",
		"apibuilder:responseBody:PUT",
		"apibuilder:responseBody:DELETE",
		"apibuilder:responseBody:PATCH",
		"apibuilder:responseBody:SCHEMA",
		"apibuilder:requestModel",
		"apibuilder:responseModel",
	}
}

// HasRequestURL checks if the resource has a request URL for the given method
func (r *VastResource) HasRequestURL(method string) bool {
	for _, url := range r.RequestURLs {
		if url.Method == method {
			return true
		}
	}
	return false
}

// HasResponseURL checks if the resource has a response URL for the given method
func (r *VastResource) HasResponseURL(method string) bool {
	for _, url := range r.ResponseURLs {
		if url.Method == method {
			return true
		}
	}
	return false
}

// GetRequestURL returns the request URL for the given method, or empty string if not found
func (r *VastResource) GetRequestURL(method string) string {
	for _, url := range r.RequestURLs {
		if url.Method == method {
			return url.URL
		}
	}
	return ""
}

// GetResponseURL returns the response URL for the given method, or empty string if not found
func (r *VastResource) GetResponseURL(method string) string {
	for _, url := range r.ResponseURLs {
		if url.Method == method {
			return url.URL
		}
	}
	return ""
}

// HasSearchQuery checks if the resource has a search query for the given method
func (r *VastResource) HasSearchQuery(method string) bool {
	for _, query := range r.SearchQueries {
		if query.Method == method {
			return true
		}
	}
	return false
}

// HasRequestBody checks if the resource has a request body for the given method
func (r *VastResource) HasRequestBody(method string) bool {
	for _, body := range r.RequestBodies {
		if body.Method == method {
			return true
		}
	}
	return false
}

// HasResponseBody checks if the resource has a response body for the given method
func (r *VastResource) HasResponseBody(method string) bool {
	for _, body := range r.ResponseBodies {
		if body.Method == method {
			return true
		}
	}
	return false
}

// GetSearchQuery returns the search query for the given method, or empty string if not found
func (r *VastResource) GetSearchQuery(method string) string {
	for _, query := range r.SearchQueries {
		if query.Method == method {
			return query.URL
		}
	}
	return ""
}

// GetRequestBody returns the request body for the given method, or empty string if not found
func (r *VastResource) GetRequestBody(method string) string {
	for _, body := range r.RequestBodies {
		if body.Method == method {
			return body.URL
		}
	}
	return ""
}

// GetResponseBody returns the response body for the given method, or empty string if not found
func (r *VastResource) GetResponseBody(method string) string {
	for _, body := range r.ResponseBodies {
		if body.Method == method {
			return body.URL
		}
	}
	return ""
}
