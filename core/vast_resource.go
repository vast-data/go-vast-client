package core

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"unicode"
)

// Dummy resource is used to support Request interceptors for "low level" session methods like GET, POST etc.
type Dummy struct {
	*VastResource
}

//  ######################################################
//              VAST RESOURCES BASE CRUD OPS
//  ######################################################

// VastResource implements VastResourceAPI and provides common behavior for managing VAST resources.
type VastResource struct {
	resourcePath string
	resourceType string
	Rest         VastRest
	mu           *KeyLocker
	resourceOps  ResourceOps
}

func NewVastResource(resourcePath string, resourceType string, rest VastRest, resourceOps ResourceOps) *VastResource {
	return &VastResource{
		resourcePath: resourcePath,
		resourceType: resourceType,
		Rest:         rest,
		mu:           NewKeyLocker(),
		resourceOps:  resourceOps,
	}
}

// Session returns the current VMSSession associated with the resource.
func (e *VastResource) Session() RESTSession {
	return e.Rest.GetSession()
}

func (e *VastResource) GetResourceType() string {
	return e.resourceType
}

func (e *VastResource) GetResourcePath() string {
	path := e.resourcePath
	trimmed := strings.Trim(path, "/")
	return "/" + trimmed + "/"
}

// ListWithContext retrieves all resources matching the given parameters using the provided context.
func (e *VastResource) ListWithContext(ctx context.Context, params Params) (RecordSet, error) {
	result, err := Request[RecordSet](ctx, e, http.MethodGet, e.resourcePath, params, nil)
	if !e.resourceOps.has(L) && ExpectStatusCodes(err, http.StatusNotFound) {
		err.(*ApiError).hints = e.describeResourceFrom(e)
	}
	return result, err
}

// CreateWithContext creates a new resource using the provided parameters and context.
func (e *VastResource) CreateWithContext(ctx context.Context, body Params) (Record, error) {
	result, err := Request[Record](ctx, e, http.MethodPost, e.resourcePath, nil, body)
	if !e.resourceOps.has(C) && ExpectStatusCodes(err, http.StatusNotFound) {
		err.(*ApiError).hints = e.describeResourceFrom(e)
	}
	return result, err
}

// UpdateWithContext updates an existing resource by its ID using the provided parameters and context.
func (e *VastResource) UpdateWithContext(ctx context.Context, id any, body Params) (Record, error) {
	path := BuildResourcePathWithID(e.resourcePath, id)
	result, err := Request[Record](ctx, e, http.MethodPatch, path, nil, body)
	if !e.resourceOps.has(U) && ExpectStatusCodes(err, http.StatusNotFound) {
		err.(*ApiError).hints = e.describeResourceFrom(e)
	}
	return result, err
}

// DeleteWithContext deletes a resource found using searchParams, using the provided deleteParams, within the given context.
// If the resource is not found, it returns success without error.
func (e *VastResource) DeleteWithContext(ctx context.Context, searchParams, queryParams, deleteParams Params) (EmptyRecord, error) {
	result, err := e.GetWithContext(ctx, searchParams)
	if err != nil {
		if IsNotFoundErr(err) {
			// Resource not found. For "Delete" it is not error condition.
			// If you want custom logic you can implement your own Get logic and then ue "DeleteById"
			return EmptyRecord{}, nil
		}
		return nil, err
	}
	idVal, ok := result["id"]
	if !ok {
		return nil, fmt.Errorf(
			"resource '%s' does not have id field in body"+
				" and thereby cannot be deleted by id", e.GetResourceType(),
		)
	}
	return e.DeleteByIdWithContext(ctx, idVal, queryParams, deleteParams)
}

// DeleteByIdWithContext deletes a resource by its unique ID using the provided context and delete parameters.
func (e *VastResource) DeleteByIdWithContext(ctx context.Context, id any, queryParams, deleteParams Params) (EmptyRecord, error) {
	path := BuildResourcePathWithID(e.resourcePath, id)
	result, err := Request[EmptyRecord](ctx, e, http.MethodDelete, path, queryParams, deleteParams)
	if !e.resourceOps.has(D) && ExpectStatusCodes(err, http.StatusNotFound) {
		err.(*ApiError).hints = e.describeResourceFrom(e)
	}
	return result, err
}

// EnsureWithContext ensures a resource matching the search parameters exists. If not, it creates it using the body.
// All operations are performed within the given context.
// Note: This method calls GetWithContext (requires R) and CreateWithContext (requires C) internally,
// which will validate permissions automatically.
func (e *VastResource) EnsureWithContext(ctx context.Context, searchParams Params, body Params) (Record, error) {
	result, err := e.GetWithContext(ctx, searchParams)
	if IsNotFoundErr(err) {
		return e.CreateWithContext(ctx, body)
	} else if err != nil {
		return nil, err
	}
	return result, nil
}

// GetWithContext retrieves a single resource that matches the given parameters using the provided context.
// Returns a NotFoundError if no resource is found.
func (e *VastResource) GetWithContext(ctx context.Context, params Params) (Record, error) {
	result, err := Request[RecordSet](ctx, e, http.MethodGet, e.resourcePath, params, nil)
	if !e.resourceOps.has(L) && ExpectStatusCodes(err, http.StatusNotFound) {
		err.(*ApiError).hints = e.describeResourceFrom(e)
	}
	if err != nil {
		return nil, err
	}
	switch len(result) {
	case 0:
		return nil, &NotFoundError{
			Resource: e.resourcePath,
			Query:    params.ToQuery(),
		}
	case 1:
		singleResult := result[0]
		if singleResult.Empty() {
			return nil, &NotFoundError{
				Resource: e.resourcePath,
				Query:    params.ToQuery(),
			}
		}
		return singleResult, nil
	default:
		return nil, &TooManyRecordsError{
			ResourcePath: e.resourcePath,
			Params:       params,
		}
	}
}

// GetByIdWithContext retrieves a resource by its unique ID using the provided context.
//
// Not all VAST resources have strictly numeric IDs; some may use UUIDs, names, or other formats.
// Therefore, this method accepts a generic 'id' parameter and dynamically formats the request path
// to handle both numeric and non-numeric identifiers.
func (e *VastResource) GetByIdWithContext(ctx context.Context, id any) (Record, error) {
	path := BuildResourcePathWithID(e.resourcePath, id)
	record, err := Request[Record](ctx, e, http.MethodGet, path, nil, nil)
	if !e.resourceOps.has(R) && ExpectStatusCodes(err, http.StatusNotFound) {
		err.(*ApiError).hints = e.describeResourceFrom(e)
	}
	return record, err
}

// ExistsWithContext checks if any resource matches the provided parameters within the given context.
// Returns true if a match is found. Returns false if not found. Returns an error only if an unexpected failure occurs.
func (e *VastResource) ExistsWithContext(ctx context.Context, params Params) (bool, error) {
	if _, err := e.GetWithContext(ctx, params); err != nil && !IsTooManyRecordsErr(err) {
		if !IsNotFoundErr(err) {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

// MustExistsWithContext checks if a resource exists using the provided context and parameters.
// It returns true if the resource exists, and false otherwise.
// This method panics if an unexpected error occurs during the check.
// It is intended for use in scenarios where failure to access the resource is considered fatal.
func (e *VastResource) MustExistsWithContext(ctx context.Context, params Params) bool {
	return Must(e.ExistsWithContext(ctx, params))
}

// List retrieves all resources matching the given parameters using the bound REST context.
func (e *VastResource) List(params Params) (RecordSet, error) {
	return e.ListWithContext(e.Rest.GetCtx(), params)
}

// Create creates a new resource using the provided parameters and the bound REST context.
func (e *VastResource) Create(params Params) (Record, error) {
	return e.CreateWithContext(e.Rest.GetCtx(), params)
}

// Update updates a resource by its ID using the provided parameters and the bound REST context.
func (e *VastResource) Update(id any, params Params) (Record, error) {
	return e.UpdateWithContext(e.Rest.GetCtx(), id, params)
}

// Delete deletes a resource found with searchParams using deleteParams and the bound REST context.
// Returns success even if the resource is not found.
func (e *VastResource) Delete(searchParams, deleteParams Params) (EmptyRecord, error) {
	return e.DeleteWithContext(e.Rest.GetCtx(), searchParams, nil, deleteParams)
}

// DeleteById deletes a resource by its ID using the bound REST context and provided deleteParams.
func (e *VastResource) DeleteById(id any, queryParams, deleteParams Params) (EmptyRecord, error) {
	return e.DeleteByIdWithContext(e.Rest.GetCtx(), id, queryParams, deleteParams)
}

// Ensure ensures a resource exists matching the searchParams. Creates it with body if not found.
// Uses the bound REST context.
func (e *VastResource) Ensure(searchParams, body Params) (Record, error) {
	return e.EnsureWithContext(e.Rest.GetCtx(), searchParams, body)
}

// Get retrieves a single resource matching the given parameters using the bound REST context.
// Returns NotFoundError if the resource does not exist.
func (e *VastResource) Get(params Params) (Record, error) {
	return e.GetWithContext(e.Rest.GetCtx(), params)
}

// GetById retrieves a resource by its ID using the bound REST context.
func (e *VastResource) GetById(id any) (Record, error) {
	return e.GetByIdWithContext(e.Rest.GetCtx(), id)
}

// Exists checks if any resource matches the given parameters using the bound REST context.
// Returns true if a match is found, false if not. Returns error only for unexpected issues.
func (e *VastResource) Exists(params Params) (bool, error) {
	return e.ExistsWithContext(e.Rest.GetCtx(), params)
}

// MustExists performs an existence check for a resource using the given parameters.
// It returns true if the resource exists, or false if it does not.
// If an error occurs during the check (other than not-found), the method panics.
// This is a convenience method intended for use in control paths where failures are not expected or tolerated.
func (e *VastResource) MustExists(params Params) bool {
	return e.MustExistsWithContext(e.Rest.GetCtx(), params)
}

// Lock acquires the resource-level mutex and returns a function to release it.
// This allows for convenient deferring of unlock operations:
//
//	defer resource.Lock()()
func (e *VastResource) Lock(keys ...any) func() {
	return e.mu.Lock(keys...)
}

//  ######################################################
//              TYPED VAST RESOURCE
//  ######################################################

type TypedVastResource struct {
	resourceType string
	Untyped      VastRest
}

func NewTypedVastResource(resourceType string, rest VastRest) *TypedVastResource {
	return &TypedVastResource{
		resourceType: resourceType,
		Untyped:      rest,
	}
}

// Session returns the current VMSSession associated with the resource.
func (e *TypedVastResource) getUntypedVastResource() VastResourceAPI {
	return e.Untyped.GetResourceMap()[e.resourceType]
}

// Session returns the current VMSSession associated with the resource.
func (e *TypedVastResource) Session() RESTSession {
	return e.getUntypedVastResource().Session()
}

func (e *TypedVastResource) GetResourceType() string {
	return e.resourceType
}

// Lock acquires the resource-level mutex and returns a function to release it.
// This allows for convenient deferring of unlock operations:
//
//	defer resource.Lock()()
func (e *TypedVastResource) Lock(keys ...any) func() {
	return e.getUntypedVastResource().Lock(keys...)
}

// ExtraMethodInfo holds metadata about extra methods
type ExtraMethodInfo struct {
	Name     string // Method name (e.g., "ViewCheckPermissionsTemplates_POST")
	HTTPVerb string // HTTP method (GET, POST, PATCH, DELETE)
	Path     string // Full path (e.g., "/views/{id}/check_permissions_templates/")
}

// discoverExtraMethods uses reflection to find all extra methods on the parent resource
// It looks for methods matching the pattern *_GET, *_POST, *_PATCH, *_DELETE
// The parent resource is the struct that embeds this VastResource
func (e *VastResource) discoverExtraMethods(parentResource interface{}) []ExtraMethodInfo {
	// Use reflection to find all methods on the parent resource (e.g., *Host)
	resourceValue := reflect.ValueOf(parentResource)
	resourceType := resourceValue.Type()

	var discovered []ExtraMethodInfo
	httpVerbs := []string{"GET", "POST", "PATCH", "PUT", "DELETE"}

	for i := 0; i < resourceType.NumMethod(); i++ {
		method := resourceType.Method(i)
		methodName := method.Name

		// Check if method ends with _<HTTP_VERB>
		for _, verb := range httpVerbs {
			suffix := "_" + verb
			if strings.HasSuffix(methodName, suffix) {
				// Extract base name (remove _GET, _POST, etc.)
				baseName := strings.TrimSuffix(methodName, suffix)

				// Try to infer the URL path from the method name
				// Method names follow pattern: <ResourceName><PathParts>_<VERB>
				// e.g., HostDiscoveredHosts_GET -> /hosts/discovered_hosts/
				path := e.inferPathFromMethodName(baseName)

				discovered = append(discovered, ExtraMethodInfo{
					Name:     methodName,
					HTTPVerb: verb,
					Path:     path,
				})
				break
			}
		}
	}

	return discovered
}

// inferPathFromMethodName attempts to infer the URL path from a method name
// e.g., HostDiscoveredHosts -> /hosts/discovered_hosts/
func (e *VastResource) inferPathFromMethodName(methodName string) string {
	// Remove "WithContext" suffix if present
	methodName = strings.TrimSuffix(methodName, "WithContext")

	// Remove the resource name prefix
	// e.g., HostDiscoveredHosts -> DiscoveredHosts
	// Capitalize first letter of resource type (replaces deprecated strings.Title)
	var resourceNameTitle string
	if len(e.resourceType) > 0 {
		resourceNameTitle = strings.ToUpper(e.resourceType[:1]) + e.resourceType[1:]
	}
	if strings.HasPrefix(methodName, resourceNameTitle) {
		methodName = strings.TrimPrefix(methodName, resourceNameTitle)
	}

	if methodName == "" {
		// This is a standard CRUD method, not an extra method
		return ""
	}

	// Convert CamelCase to snake_case with slashes
	// e.g., DiscoveredHosts -> discovered_hosts
	var result strings.Builder
	for i, r := range methodName {
		if i > 0 && unicode.IsUpper(r) {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToLower(r))
	}

	pathPart := result.String()

	// Build the full path
	// Check if it starts with resource path or is standalone
	basePath := e.GetResourcePath()

	// Try both patterns: /resource/extra/ and /resource/{id}/extra/
	// For now, assume non-id pattern (most common for extra methods)
	return strings.TrimSuffix(basePath, "/") + "/" + pathPart + "/"
}

// describeResourceFrom returns a comprehensive description of all endpoints for a resource
// parentResource should be the struct that embeds this VastResource (e.g., *Host)
func (e *VastResource) describeResourceFrom(parentResource any) string {
	var sb strings.Builder

	// Group endpoints by path
	endpointMap := make(map[string][]string)

	// Add standard CRUD endpoints
	basePath := e.GetResourcePath()
	detailPath := strings.TrimSuffix(basePath, "/") + "/{id}/"

	if e.resourceOps.isListable() {
		endpointMap[basePath] = append(endpointMap[basePath],
			"List() / ListWithContext()",
			"Get() / GetWithContext()",
			"Exists() / ExistsWithContext()",
		)
	}

	if e.resourceOps.isReadable() {
		endpointMap[detailPath] = append(endpointMap[detailPath],
			"GetById() / GetByIdWithContext()",
		)
	}

	if e.resourceOps.isCreatable() {
		endpointMap[basePath] = append(endpointMap[basePath],
			"Create() / CreateWithContext()",
		)
		if e.resourceOps.isListable() {
			endpointMap[basePath] = append(endpointMap[basePath],
				"Ensure() / EnsureWithContext()",
			)
		}
	}

	if e.resourceOps.isUpdatable() {
		endpointMap[detailPath] = append(endpointMap[detailPath],
			"Update() / UpdateWithContext()",
		)
	}

	if e.resourceOps.isDeletable() {
		endpointMap[detailPath] = append(endpointMap[detailPath],
			"Delete() / DeleteWithContext()",
			"DeleteById() / DeleteByIdWithContext()",
		)
	}

	// Add extra methods (discovered via reflection)
	extraMethods := e.discoverExtraMethods(parentResource)
	for _, extra := range extraMethods {
		if extra.Path != "" { // Skip if path inference failed
			methodDesc := fmt.Sprintf("%s [%s]", extra.Name, extra.HTTPVerb)
			endpointMap[extra.Path] = append(endpointMap[extra.Path], methodDesc)
		}
	}

	// Sort and format output (deterministic ordering)
	// First, collect all paths and sort them
	var paths []string
	for path := range endpointMap {
		paths = append(paths, path)
	}
	// Sort paths: base path first, then detail path, then alphabetically
	sortPaths := func(paths []string) {
		for i := 0; i < len(paths); i++ {
			for j := i + 1; j < len(paths); j++ {
				// Base path comes first
				if paths[i] == detailPath && paths[j] == basePath {
					paths[i], paths[j] = paths[j], paths[i]
				} else if paths[i] != basePath && paths[i] != detailPath && (paths[j] == basePath || paths[j] == detailPath) {
					paths[i], paths[j] = paths[j], paths[i]
				} else if paths[i] > paths[j] && paths[i] != basePath && paths[j] != basePath && paths[i] != detailPath && paths[j] != detailPath {
					paths[i], paths[j] = paths[j], paths[i]
				}
			}
		}
	}
	sortPaths(paths)

	for _, path := range paths {
		methods := endpointMap[path]
		sb.WriteString(fmt.Sprintf("  %s\n", path))
		for _, method := range methods {
			sb.WriteString(fmt.Sprintf("    • %s\n", method))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("Supported Operations: %s\n", e.resourceOps.String()))

	return sb.String()
}

//  ######################################################
//              CRUD FLAGS
//  ######################################################

// ResourceOps is a bitmask representing which CRUD operations are supported
// by a given resource (Create, Read, Update, Delete).
type ResourceOps int

const (
	C ResourceOps = 1 << iota // Create permission
	L                         // Read (List) permissions
	R                         // Read (<entry>/<id>) permission
	U                         // Update permission
	D                         // Delete permission
)

// NewResourceOps creates a new bitmask from the provided flags.
// Example: NewResourceOps(R, U) -> Read+Update.
func NewResourceOps(flags ...ResourceOps) ResourceOps {
	var f ResourceOps
	for _, fl := range flags {
		f |= fl
	}
	return f
}

// has reports whether all given flags are present in the bitmask.
func (ops ResourceOps) has(flag ResourceOps) bool {
	return ops&flag == flag
}

// Convenience methods for checking specific operations
func (ops ResourceOps) isCreatable() bool { return ops&C != 0 }
func (ops ResourceOps) isListable() bool  { return ops&L != 0 }
func (ops ResourceOps) isReadable() bool  { return ops&R != 0 }
func (ops ResourceOps) isUpdatable() bool { return ops&U != 0 }
func (ops ResourceOps) isDeletable() bool { return ops&D != 0 }

// set returns a new bitmask with the given flag(s) enabled.
func (ops ResourceOps) set(flag ResourceOps) ResourceOps {
	return ops | flag
}

// clear returns a new bitmask with the given flag(s) disabled.
func (ops ResourceOps) clear(flag ResourceOps) ResourceOps {
	return ops &^ flag
}

// String returns a compact string representation of the active flags.
// Example: "CLRU", "LR", "CD", or "-" if no flags are set.
func (ops ResourceOps) String() string {
	if ops == ResourceOps(0) {
		return "-"
	}
	var b strings.Builder
	if ops&C != 0 {
		b.WriteByte('C')
	}
	if ops&L != 0 {
		b.WriteByte('L')
	}
	if ops&R != 0 {
		b.WriteByte('R')
	}
	if ops&U != 0 {
		b.WriteByte('U')
	}
	if ops&D != 0 {
		b.WriteByte('D')
	}
	return b.String()
}
