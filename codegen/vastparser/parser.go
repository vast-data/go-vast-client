package vastparser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vast-data/go-vast-client/codegen/apibuilder"
	"github.com/vast-data/go-vast-client/codegen/markers"
)

// VastResource represents a parsed VastData resource with its APITyped markers
type VastResource struct {
	Name           string                     `json:"name"`
	RequestURLs    []apibuilder.RequestURL    `json:"requestUrls,omitempty"`
	ResponseURLs   []apibuilder.ResponseURL   `json:"responseUrls,omitempty"`
	Details        []apibuilder.Details       `json:"details,omitempty"`          // NEW: for details markers
	Upserts        []apibuilder.Upsert        `json:"upserts,omitempty"`          // NEW: for upsert markers
	ExtraMethods   []apibuilder.ExtraMethod   `json:"extraMethods,omitempty"`     // NEW: for extraMethod markers
	DetailsQueries []apibuilder.DetailsQuery  `json:"detailsQueries,omitempty"`   // DEPRECATED: kept for backward compatibility
	UpsertQueries  []apibuilder.UpsertQuery   `json:"upsertQueries,omitempty"`    // DEPRECATED: kept for backward compatibility
	SearchQueries  []apibuilder.SearchQuery   `json:"searchQueries,omitempty"`    // DEPRECATED: kept for backward compatibility
	CreateQueries  []apibuilder.CreateQuery   `json:"createQueries,omitempty"`    // DEPRECATED: kept for backward compatibility
	RequestBodies  []apibuilder.RequestBody   `json:"requestBodies,omitempty"`    // DEPRECATED: kept for backward compatibility
	Models         []apibuilder.ResponseBody  `json:"models,omitempty"`           // DEPRECATED: kept for backward compatibility
	RequestModel   string                     `json:"requestModel,omitempty"`
	ResponseModel  string                     `json:"responseModel,omitempty"`
	AllMarkers     []markers.MarkerValue      `json:"allMarkers,omitempty"`
}

// VastResourceParser parses VastData resource files
type VastResourceParser struct {
	registry  *markers.Registry
	collector *markers.Collector
}

// NewVastResourceParser creates a new parser for VastData resources
func NewVastResourceParser() *VastResourceParser {
	registry := markers.NewRegistry()
	apibuilder.MustRegisterAPITypedMarkers(registry)

	return &VastResourceParser{
		registry:  registry,
		collector: markers.NewCollector(registry),
	}
}

// ParseFile parses a Go file and returns all VastData resources with APITyped markers
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

// ParseDirectory parses all Go files in a directory and returns all VastData resources with APITyped markers
func (p *VastResourceParser) ParseDirectory(dirname string) ([]VastResource, error) {
	var allResources []VastResource

	// Read all files in the directory
	entries, err := os.ReadDir(dirname)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dirname, err)
	}

	// Parse each .go file
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		filename := entry.Name()
		if filepath.Ext(filename) != ".go" {
			continue
		}

		fullPath := filepath.Join(dirname, filename)
		
		// Parse each file and collect resources
		err := p.collector.EachType(fullPath, func(typeInfo *markers.TypeInfo) {
			resource := p.parseTypeInfo(typeInfo)
			if resource != nil {
				allResources = append(allResources, *resource)
			}
		})

		if err != nil {
			return nil, fmt.Errorf("failed to parse file %s: %w", fullPath, err)
		}
	}

	return allResources, nil
}

// parseTypeInfo converts a TypeInfo into a VastResource if it has APITyped markers
func (p *VastResourceParser) parseTypeInfo(typeInfo *markers.TypeInfo) *VastResource {
	resource := &VastResource{
		Name: typeInfo.Name,
	}

	hasAPITypedMarkers := false

	// Process all markers
	for markerName, values := range typeInfo.Markers {
		if !isAPITypedMarker(markerName) {
			continue
		}

		hasAPITypedMarkers = true

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
		case "apityped:requestUrl:GET":
			p.addRequestURL(resource, "GET", values)
		case "apityped:requestUrl:POST":
			p.addRequestURL(resource, "POST", values)
		case "apityped:requestUrl:PUT":
			p.addRequestURL(resource, "PUT", values)
		case "apityped:requestUrl:DELETE":
			p.addRequestURL(resource, "DELETE", values)
		case "apityped:requestUrl:PATCH":
			p.addRequestURL(resource, "PATCH", values)

		case "apityped:responseUrl:GET":
			p.addResponseURL(resource, "GET", values)
		case "apityped:responseUrl:POST":
			p.addResponseURL(resource, "POST", values)
		case "apityped:responseUrl:PUT":
			p.addResponseURL(resource, "PUT", values)
		case "apityped:responseUrl:DELETE":
			p.addResponseURL(resource, "DELETE", values)
		case "apityped:responseUrl:PATCH":
			p.addResponseURL(resource, "PATCH", values)

		case "apityped:details:GET":
			p.addDetails(resource, "GET", values)
		case "apityped:details:PATCH":
			p.addDetails(resource, "PATCH", values)

		case "apityped:upsert:POST":
			p.addUpsert(resource, "POST", values)
		case "apityped:upsert:PUT":
			p.addUpsert(resource, "PUT", values)
		case "apityped:upsert:PATCH":
			p.addUpsert(resource, "PATCH", values)

		// Handle apityped:extraMethod:METHOD markers
		default:
			if strings.HasPrefix(markerName, "apityped:extraMethod:") {
				method := strings.TrimPrefix(markerName, "apityped:extraMethod:")
				p.addTypedExtraMethod(resource, method, values)
			}

		// DEPRECATED: old marker names (kept for backward compatibility)
		case "apityped:detailsQuery:GET":
			p.addDetailsQuery(resource, "GET", values)
		case "apityped:detailsQuery:PATCH":
			p.addDetailsQuery(resource, "PATCH", values)

		case "apityped:upsertQuery:POST":
			p.addUpsertQuery(resource, "POST", values)
		case "apityped:upsertQuery:PUT":
			p.addUpsertQuery(resource, "PUT", values)
		case "apityped:upsertQuery:PATCH":
			p.addUpsertQuery(resource, "PATCH", values)

		case "apityped:searchQuery:GET":
			p.addSearchQuery(resource, "GET", values)
		case "apityped:searchQuery:PATCH":
			p.addSearchQuery(resource, "PATCH", values)

		case "apityped:createQuery:POST":
			p.addCreateQuery(resource, "POST", values)
		case "apityped:createQuery:PUT":
			p.addCreateQuery(resource, "PUT", values)
		case "apityped:createQuery:PATCH":
			p.addCreateQuery(resource, "PATCH", values)

		case "apityped:responseBody:GET":
			p.addResponseBody(resource, "GET", values)
		case "apityped:responseBody:POST":
			p.addResponseBody(resource, "POST", values)
		case "apityped:responseBody:PUT":
			p.addResponseBody(resource, "PUT", values)
		case "apityped:responseBody:DELETE":
			p.addResponseBody(resource, "DELETE", values)
		case "apityped:responseBody:PATCH":
			p.addResponseBody(resource, "PATCH", values)
		case "apityped:responseBody:SCHEMA":
			p.addResponseBody(resource, "SCHEMA", values)

		case "apityped:requestModel":
			if len(values) > 0 {
				if model, ok := values[0].(string); ok {
					resource.RequestModel = model
				}
			}

		case "apityped:responseModel":
			if len(values) > 0 {
				if model, ok := values[0].(string); ok {
					resource.ResponseModel = model
				}
			}

		// Legacy markers (DEPRECATED - kept for backward compatibility only)
		case "apityped:requestBody:POST":
			p.addRequestBody(resource, "POST", values)
		case "apityped:requestBody:PUT":
			p.addRequestBody(resource, "PUT", values)
		case "apityped:requestBody:PATCH":
			p.addRequestBody(resource, "PATCH", values)
		// Note: model:SCHEMA is intentionally NOT supported - use searchQuery or createQuery instead
		}
	}

	// Only return resources that have APITyped markers
	if !hasAPITypedMarkers {
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

// addDetails adds a details marker to the resource
func (p *VastResourceParser) addDetails(resource *VastResource, method string, values []interface{}) {
	for _, value := range values {
		if url, ok := value.(string); ok {
			resource.Details = append(resource.Details, apibuilder.Details{
				Method: method,
				URL:    url,
			})
		}
	}
}

// addUpsert adds an upsert marker to the resource
func (p *VastResourceParser) addUpsert(resource *VastResource, method string, values []interface{}) {
	for _, value := range values {
		if url, ok := value.(string); ok {
			resource.Upserts = append(resource.Upserts, apibuilder.Upsert{
				Method: method,
				URL:    url,
			})
		}
	}
}

// addTypedExtraMethod adds an extra method marker to the resource
func (p *VastResourceParser) addTypedExtraMethod(resource *VastResource, method string, values []interface{}) {
	for _, value := range values {
		if path, ok := value.(string); ok {
			resource.ExtraMethods = append(resource.ExtraMethods, apibuilder.ExtraMethod{
				Method: method,
				Path:   path,
			})
		}
	}
}

// addDetailsQuery adds a details query to the resource (DEPRECATED - use addDetails)
func (p *VastResourceParser) addDetailsQuery(resource *VastResource, method string, values []interface{}) {
	for _, value := range values {
		if url, ok := value.(string); ok {
			resource.DetailsQueries = append(resource.DetailsQueries, apibuilder.DetailsQuery{
				Method: method,
				URL:    url,
			})
		}
	}
}

// addUpsertQuery adds an upsert query to the resource (DEPRECATED - use addUpsert)
func (p *VastResourceParser) addUpsertQuery(resource *VastResource, method string, values []interface{}) {
	for _, value := range values {
		if url, ok := value.(string); ok {
			resource.UpsertQueries = append(resource.UpsertQueries, apibuilder.UpsertQuery{
				Method: method,
				URL:    url,
			})
		}
	}
}

// addSearchQuery adds a search query to the resource (DEPRECATED - use addDetailsQuery)
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

// addCreateQuery adds a create query to the resource (DEPRECATED - use addUpsertQuery)
func (p *VastResourceParser) addCreateQuery(resource *VastResource, method string, values []interface{}) {
	for _, value := range values {
		if url, ok := value.(string); ok {
			resource.CreateQueries = append(resource.CreateQueries, apibuilder.CreateQuery{
				Method: method,
				URL:    url,
			})
		}
	}
}

// addRequestBody adds a request body to the resource (DEPRECATED - kept for backward compatibility)
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

// addResponseBody adds a response body to the resource (now creates model)
func (p *VastResourceParser) addResponseBody(resource *VastResource, method string, values []interface{}) {
	for _, value := range values {
		if url, ok := value.(string); ok {
			resource.Models = append(resource.Models, apibuilder.ResponseBody{
				Method: method,
				URL:    url,
			})
		}
	}
}

// isAPITypedMarker checks if a marker name is an APITyped marker
func isAPITypedMarker(markerName string) bool {
	apiTypedMarkers := []string{
		"apityped:requestUrl:GET",
		"apityped:requestUrl:POST",
		"apityped:requestUrl:PUT",
		"apityped:requestUrl:DELETE",
		"apityped:requestUrl:PATCH",
		"apityped:responseUrl:GET",
		"apityped:responseUrl:POST",
		"apityped:responseUrl:PUT",
		"apityped:responseUrl:DELETE",
		"apityped:responseUrl:PATCH",
		"apityped:details:GET",
		"apityped:details:PATCH",
		"apityped:upsert:POST",
		"apityped:upsert:PUT",
		"apityped:upsert:PATCH",
		"apityped:extraMethod:GET",
		"apityped:extraMethod:POST",
		"apityped:extraMethod:PUT",
		"apityped:extraMethod:PATCH",
		"apityped:extraMethod:DELETE",
		"apityped:extraMethod:HEAD",
		"apityped:extraMethod:OPTIONS",
		"apityped:detailsQuery:GET",      // DEPRECATED
		"apityped:detailsQuery:PATCH",    // DEPRECATED
		"apityped:upsertQuery:POST",      // DEPRECATED
		"apityped:upsertQuery:PUT",       // DEPRECATED
		"apityped:upsertQuery:PATCH",     // DEPRECATED
		"apityped:searchQuery:GET",       // DEPRECATED
		"apityped:searchQuery:PATCH",     // DEPRECATED
		"apityped:createQuery:POST",      // DEPRECATED
		"apityped:createQuery:PUT",       // DEPRECATED
		"apityped:createQuery:PATCH",     // DEPRECATED
		"apityped:requestBody:POST",      // DEPRECATED
		"apityped:requestBody:PUT",       // DEPRECATED
		"apityped:requestBody:PATCH",     // DEPRECATED
		"apityped:responseBody:GET",
		"apityped:responseBody:POST",
		"apityped:responseBody:PUT",
		"apityped:responseBody:DELETE",
		"apityped:responseBody:PATCH",
		"apityped:responseBody:SCHEMA",
		"apityped:requestModel",
		"apityped:responseModel",
	}

	for _, marker := range apiTypedMarkers {
		if markerName == marker {
			return true
		}
	}
	return false
}

// GetAllMarkerNames returns all possible APITyped marker names
func GetAllMarkerNames() []string {
	return []string{
		"apityped:requestUrl:GET",
		"apityped:requestUrl:POST",
		"apityped:requestUrl:PUT",
		"apityped:requestUrl:DELETE",
		"apityped:requestUrl:PATCH",
		"apityped:responseUrl:GET",
		"apityped:responseUrl:POST",
		"apityped:responseUrl:PUT",
		"apityped:responseUrl:DELETE",
		"apityped:responseUrl:PATCH",
		"apityped:searchQuery:GET",
		"apityped:searchQuery:SCHEMA",
		"apityped:requestBody:POST",
		"apityped:requestBody:PUT",
		"apityped:requestBody:PATCH",
		"apityped:requestBody:SCHEMA",
		"apityped:responseBody:GET",
		"apityped:responseBody:POST",
		"apityped:responseBody:PUT",
		"apityped:responseBody:DELETE",
		"apityped:responseBody:PATCH",
		"apityped:responseBody:SCHEMA",
		"apityped:requestModel",
		"apityped:responseModel",
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

// HasDetails checks if the resource has a details marker for the given method
func (r *VastResource) HasDetails(method string) bool {
	for _, query := range r.Details {
		if query.Method == method {
			return true
		}
	}
	// Fallback to deprecated markers for backward compatibility
	for _, query := range r.DetailsQueries {
		if query.Method == method {
			return true
		}
	}
	for _, query := range r.SearchQueries {
		if query.Method == method {
			return true
		}
	}
	return false
}

// HasUpsert checks if the resource has an upsert marker for the given method
func (r *VastResource) HasUpsert(method string) bool {
	for _, query := range r.Upserts {
		if query.Method == method {
			return true
		}
	}
	// Fallback to deprecated markers for backward compatibility
	for _, query := range r.UpsertQueries {
		if query.Method == method {
			return true
		}
	}
	for _, query := range r.CreateQueries {
		if query.Method == method {
			return true
		}
	}
	return false
}

// HasDetailsQuery checks if the resource has a details query for the given method (DEPRECATED - use HasDetails)
func (r *VastResource) HasDetailsQuery(method string) bool {
	return r.HasDetails(method)
}

// HasUpsertQuery checks if the resource has an upsert query for the given method (DEPRECATED - use HasUpsert)
func (r *VastResource) HasUpsertQuery(method string) bool {
	return r.HasUpsert(method)
}

// HasSearchQuery checks if the resource has a search query for the given method (DEPRECATED - use HasDetails)
func (r *VastResource) HasSearchQuery(method string) bool {
	return r.HasDetails(method)
}

// HasCreateQuery checks if the resource has a create query for the given method (DEPRECATED - use HasUpsert)
func (r *VastResource) HasCreateQuery(method string) bool {
	return r.HasUpsert(method)
}

// HasRequestBody checks if the resource has a request body for the given method (DEPRECATED - use HasCreateQuery)
func (r *VastResource) HasRequestBody(method string) bool {
	for _, body := range r.RequestBodies {
		if body.Method == method {
			return true
		}
	}
	return false
}

// HasModel checks if the resource has a model for the given method
func (r *VastResource) HasModel(method string) bool {
	for _, body := range r.Models {
		if body.Method == method {
			return true
		}
	}
	return false
}

// HasCreateBody checks if the resource has a create body for the given method (deprecated, use HasRequestBody)
func (r *VastResource) HasCreateBody(method string) bool {
	return r.HasRequestBody(method)
}

// HasResponseBody checks if the resource has a response body for the given method (deprecated, use HasModel)
func (r *VastResource) HasResponseBody(method string) bool {
	return r.HasModel(method)
}

// GetDetails returns the details marker for the given method, or empty string if not found
func (r *VastResource) GetDetails(method string) string {
	for _, query := range r.Details {
		if query.Method == method {
			return query.URL
		}
	}
	// Fallback to deprecated markers for backward compatibility
	for _, query := range r.DetailsQueries {
		if query.Method == method {
			return query.URL
		}
	}
	for _, query := range r.SearchQueries {
		if query.Method == method {
			return query.URL
		}
	}
	return ""
}

// GetUpsert returns the upsert marker for the given method, or empty string if not found
func (r *VastResource) GetUpsert(method string) string {
	for _, query := range r.Upserts {
		if query.Method == method {
			return query.URL
		}
	}
	// Fallback to deprecated markers for backward compatibility
	for _, query := range r.UpsertQueries {
		if query.Method == method {
			return query.URL
		}
	}
	for _, query := range r.CreateQueries {
		if query.Method == method {
			return query.URL
		}
	}
	return ""
}

// GetDetailsQuery returns the details query for the given method, or empty string if not found (DEPRECATED - use GetDetails)
func (r *VastResource) GetDetailsQuery(method string) string {
	return r.GetDetails(method)
}

// GetUpsertQuery returns the upsert query for the given method, or empty string if not found (DEPRECATED - use GetUpsert)
func (r *VastResource) GetUpsertQuery(method string) string {
	return r.GetUpsert(method)
}

// GetSearchQuery returns the search query for the given method, or empty string if not found (DEPRECATED - use GetDetails)
func (r *VastResource) GetSearchQuery(method string) string {
	return r.GetDetails(method)
}

// GetCreateQuery returns the create query for the given method, or empty string if not found (DEPRECATED - use GetUpsert)
func (r *VastResource) GetCreateQuery(method string) string {
	return r.GetUpsert(method)
}

// GetRequestBody returns the request body for the given method, or empty string if not found (DEPRECATED - use GetCreateQuery)
func (r *VastResource) GetRequestBody(method string) string {
	for _, body := range r.RequestBodies {
		if body.Method == method {
			return body.URL
		}
	}
	return ""
}

// GetModel returns the model for the given method, or empty string if not found
func (r *VastResource) GetModel(method string) string {
	for _, body := range r.Models {
		if body.Method == method {
			return body.URL
		}
	}
	return ""
}

// GetCreateBody returns the create body for the given method, or empty string if not found (deprecated, use GetRequestBody)
func (r *VastResource) GetCreateBody(method string) string {
	return r.GetRequestBody(method)
}

// GetResponseBody returns the response body for the given method, or empty string if not found (deprecated, use GetModel)
func (r *VastResource) GetResponseBody(method string) string {
	return r.GetModel(method)
}

// IsReadOnly returns true if the resource is marked as read-only
// IsReadOnly returns true if the resource has no upsert markers (POST, PUT, or PATCH)
// A resource is considered read-only if it only has details markers but no upsert markers
func (r *VastResource) IsReadOnly() bool {
	// Check if any upsert marker exists
	return !r.HasUpsert("POST") && !r.HasUpsert("PUT") && !r.HasUpsert("PATCH")
}
