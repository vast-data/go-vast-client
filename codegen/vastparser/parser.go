package vastparser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vast-data/go-vast-client/codegen/apibuilder"
	"github.com/vast-data/go-vast-client/codegen/markers"
)

// VastResource represents a parsed VastData resource with its APITyped markers
type VastResource struct {
	Name          string                   `json:"name"`
	Operations    *apibuilder.Operations   `json:"operations,omitempty"` // Unified ops marker (replaces Details + Upserts)
	RequestURLs   []apibuilder.RequestURL  `json:"requestUrls,omitempty"`
	ResponseURLs  []apibuilder.ResponseURL `json:"responseUrls,omitempty"`
	Details       []apibuilder.Details     `json:"details,omitempty"` // Legacy - deprecated
	Upserts       []apibuilder.Upsert      `json:"upserts,omitempty"` // Legacy - deprecated
	ExtraMethods  []apibuilder.ExtraMethod `json:"extraMethods,omitempty"`
	RequestModel  string                   `json:"requestModel,omitempty"`
	ResponseModel string                   `json:"responseModel,omitempty"`
	AllMarkers    []markers.MarkerValue    `json:"allMarkers,omitempty"`
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
		// Parse out any options from the marker name (e.g., [wait=3m])
		cleanMarkerName, waitTimeout := parseExtraMethodOptions(markerName)

		if !isAPITypedMarker(cleanMarkerName) {
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

		// Process specific marker types (use cleanMarkerName without options)
		switch cleanMarkerName {
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

		// Legacy markers (deprecated but still supported for backward compatibility)
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
			// Handle unified ops marker (apityped:ops:CRUD, apityped:ops:R, etc.)
			if strings.HasPrefix(cleanMarkerName, "apityped:ops:") {
				ops := strings.TrimPrefix(cleanMarkerName, "apityped:ops:")
				p.addOperations(resource, ops, values)
			}
			if strings.HasPrefix(cleanMarkerName, "apityped:extraMethod:") {
				methodsStr := strings.TrimPrefix(cleanMarkerName, "apityped:extraMethod:")
				// Support multiple methods separated by | (e.g., "POST|PATCH|DELETE")
				methods := strings.Split(methodsStr, "|")
				for _, method := range methods {
					method = strings.TrimSpace(method)
					if method != "" {
						p.addTypedExtraMethod(resource, method, waitTimeout, values)
					}
				}
			}

			// Handle apiall:extraMethod markers (generates both typed and untyped)
			if strings.HasPrefix(cleanMarkerName, "apiall:extraMethod:") {
				methodsStr := strings.TrimPrefix(cleanMarkerName, "apiall:extraMethod:")
				// Support multiple methods separated by | (e.g., "POST|PATCH|DELETE")
				methods := strings.Split(methodsStr, "|")
				for _, method := range methods {
					method = strings.TrimSpace(method)
					if method != "" {
						p.addTypedExtraMethod(resource, method, waitTimeout, values)
						p.addUntypedExtraMethod(resource, method, values)
					}
				}
			}

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

// addOperations adds a unified ops marker to the resource
func (p *VastResourceParser) addOperations(resource *VastResource, ops string, values []interface{}) {
	for _, value := range values {
		if url, ok := value.(string); ok {
			resource.Operations = &apibuilder.Operations{
				Operations: ops,
				URL:        url,
			}
			// Only one ops marker is allowed per resource
			return
		}
	}
}

// parseExtraMethodOptions extracts options from marker name (e.g., "[wait(3m)]")
// Returns the cleaned marker name and extracted options
func parseExtraMethodOptions(markerName string) (cleanName string, waitTimeout string) {
	// Check for bracket notation: extraMethod[wait(3m)]:POST
	re := regexp.MustCompile(`^(.*?)\[wait\(([^\)]+)\)\](.*)$`)
	matches := re.FindStringSubmatch(markerName)
	if len(matches) == 4 {
		// matches[1] = prefix (e.g., "apityped:extraMethod" or "apiall:extraMethod")
		// matches[2] = wait timeout value (e.g., "3m")
		// matches[3] = suffix (e.g., ":POST")
		cleanName = matches[1] + matches[3]
		waitTimeout = matches[2]
		return
	}
	// No options found, return original marker name
	return markerName, ""
}

// addTypedExtraMethod adds an extra method marker to the resource
func (p *VastResourceParser) addTypedExtraMethod(resource *VastResource, method string, waitTimeout string, values []interface{}) {
	for _, value := range values {
		if path, ok := value.(string); ok {
			resource.ExtraMethods = append(resource.ExtraMethods, apibuilder.ExtraMethod{
				Method: method,
				Path:   path,
			})
		}
	}
}

// addUntypedExtraMethod adds an untyped extra method marker to the resource
// This is used when apiall:extraMethod marker is specified
func (p *VastResourceParser) addUntypedExtraMethod(resource *VastResource, method string, values []interface{}) {
	// For now, we add to the same ExtraMethods slice
	// The untyped generator will parse files separately and use its own parser
	// This function is a placeholder for future enhancement if needed
	// The actual untyped generation happens via UntypedResourceParser
}

// isAPITypedMarker checks if a marker name is an APITyped marker or apiall marker
func isAPITypedMarker(markerName string) bool {
	// Strip out [wait=...] or other options before checking
	cleanName, _ := parseExtraMethodOptions(markerName)

	// Check for apiall: prefix (generates both typed and untyped)
	if strings.HasPrefix(cleanName, "apiall:") {
		return true
	}

	// Check for apityped: prefix with specific markers
	if strings.HasPrefix(cleanName, "apityped:ops:") ||
		strings.HasPrefix(cleanName, "apityped:details:") ||
		strings.HasPrefix(cleanName, "apityped:upsert:") ||
		strings.HasPrefix(cleanName, "apityped:extraMethod:") {
		return true
	}

	return false
}

// GetAllMarkerNames returns all possible APITyped marker names
func GetAllMarkerNames() []string {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
	markers := []string{}

	// Add details markers
	for _, method := range []string{"GET", "PATCH"} {
		markers = append(markers, "apityped:details:"+method)
	}

	// Add upsert markers
	for _, method := range []string{"POST", "PUT", "PATCH"} {
		markers = append(markers, "apityped:upsert:"+method)
	}

	// Add extraMethod markers
	for _, method := range append(methods, "HEAD", "OPTIONS") {
		markers = append(markers, "apityped:extraMethod:"+method)
		markers = append(markers, "apiall:extraMethod:"+method)
	}

	return markers
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
	return false
}

// HasUpsert checks if the resource has an upsert marker for the given method
func (r *VastResource) HasUpsert(method string) bool {
	for _, query := range r.Upserts {
		if query.Method == method {
			return true
		}
	}
	return false
}

// GetDetails returns the details marker for the given method, or empty string if not found
func (r *VastResource) GetDetails(method string) string {
	for _, query := range r.Details {
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
	return ""
}

// IsReadOnly returns true if the resource is marked as read-only
// IsReadOnly returns true if the resource has no create, update, or delete operations
// A resource is considered read-only if it only has read operations
func (r *VastResource) IsReadOnly() bool {
	// Check new Operations marker first
	if r.Operations != nil {
		return !r.Operations.HasCreate() && !r.Operations.HasUpdate() && !r.Operations.HasDelete()
	}

	// Fall back to legacy markers
	return !r.HasUpsert("POST") && !r.HasUpsert("PUT") && !r.HasUpsert("PATCH")
}

// GetOperationsURL returns the URL from the Operations marker, or empty string if not found
func (r *VastResource) GetOperationsURL() string {
	if r.Operations != nil {
		return r.Operations.URL
	}
	return ""
}

// HasOperations returns true if the resource has an Operations marker
func (r *VastResource) HasOperations() bool {
	return r.Operations != nil
}

// GetOperationsString returns the operations string (e.g., "CRUD", "R", "CU")
func (r *VastResource) GetOperationsString() string {
	if r.Operations != nil {
		return r.Operations.Operations
	}
	return ""
}
