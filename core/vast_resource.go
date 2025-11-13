package core

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

// Dummy resource is used to support Request interceptors for "low level" session methods like GET, POST etc.
type Dummy struct {
	*VastResource
}

type DummyRest struct {
	ctx         context.Context
	Session     RESTSession
	resourceMap map[string]VastResourceAPIWithContext
}

func (rest *DummyRest) GetSession() RESTSession {
	return rest.Session
}

func (rest *DummyRest) GetResourceMap() map[string]VastResourceAPIWithContext {
	return rest.resourceMap
}

func (rest *DummyRest) GetCtx() context.Context {
	return rest.ctx
}

func (rest *DummyRest) SetCtx(ctx context.Context) {
	rest.ctx = ctx
}

func NewDummy(ctx context.Context, session RESTSession) *Dummy {
	dummy := &Dummy{
		VastResource: &VastResource{
			resourceType: "Dummy",
			resourcePath: "",
		},
	}
	rest := &DummyRest{
		ctx:         ctx,
		Session:     session,
		resourceMap: map[string]VastResourceAPIWithContext{"Dummy": dummy},
	}
	dummy.Rest = rest
	return dummy
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
	parent       any // Reference to the parent resource that embeds this VastResource
}

func NewVastResource(resourcePath string, resourceType string, rest VastRest, resourceOps ResourceOps, parent any) *VastResource {
	return &VastResource{
		resourcePath: resourcePath,
		resourceType: resourceType,
		Rest:         rest,
		mu:           NewKeyLocker(),
		resourceOps:  resourceOps,
		parent:       parent,
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
// This method uses GetIteratorWithContext internally and fetches all pages.
func (e *VastResource) ListWithContext(ctx context.Context, params Params) (RecordSet, error) {
	// Use Iterator as base abstraction - fetch all pages
	pageSize := e.Session().GetConfig().PageSize
	iter := e.GetIteratorWithContext(ctx, params, pageSize)
	result, err := iter.All()
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
func (e *VastResource) DeleteWithContext(ctx context.Context, searchParams, queryParams, deleteParams Params) (Record, error) {
	result, err := e.GetWithContext(ctx, searchParams)
	if err != nil {
		if IsNotFoundErr(err) {
			// Resource not found. For "Delete" it is not error condition.
			// If you want custom logic you can implement your own Get logic and then ue "DeleteById"
			return Record{}, nil
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
func (e *VastResource) DeleteByIdWithContext(ctx context.Context, id any, queryParams, deleteParams Params) (Record, error) {
	path := BuildResourcePathWithID(e.resourcePath, id)
	result, err := Request[Record](ctx, e, http.MethodDelete, path, queryParams, deleteParams)
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
	result, err := e.ListWithContext(ctx, params)

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
func (e *VastResource) Delete(searchParams, deleteParams Params) (Record, error) {
	return e.DeleteWithContext(e.Rest.GetCtx(), searchParams, nil, deleteParams)
}

// DeleteById deletes a resource by its ID using the bound REST context and provided deleteParams.
func (e *VastResource) DeleteById(id any, queryParams, deleteParams Params) (Record, error) {
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

// GetIteratorWithContext creates a new iterator for paginated results using the provided context.
// The iterator abstracts away the differences between paginated and non-paginated API responses.
//
// Parameters:
//   - ctx: The context for the iterator (used for all subsequent requests)
//   - params: Query parameters to filter results
//   - pageSize: Number of items per page (if <= 0, uses session's configured PageSize)
//
// Returns an Iterator that can be used to navigate through pages of results.
//
// Example usage:
//
//	iter := resource.GetIteratorWithContext(ctx, Params{"name__contains": "test"}, 50)
//	for {
//	    records, err := iter.Next()
//	    if err != nil || len(records) == 0 {
//	        break
//	    }
//	    // Process records
//	}
func (e *VastResource) GetIteratorWithContext(ctx context.Context, params Params, pageSize int) Iterator {
	return NewResourceIterator(ctx, e, params, pageSize)
}

// GetIterator creates a new iterator for paginated results using the bound REST context.
//
// Parameters:
//   - params: Query parameters to filter results
//   - pageSize: Number of items per page (if <= 0, uses session's configured PageSize; 0 means no page_size param)
//
// Returns an Iterator that can be used to navigate through pages of results.
//
// Example usage:
//
//	iter := resource.GetIterator(Params{"tenant_id": 1}, 25)
//	for {
//	    records, err := iter.Next()
//	    if err != nil || len(records) == 0 {
//	        break
//	    }
//	    fmt.Printf("Page has %d records\n", len(records))
//	}
func (e *VastResource) GetIterator(params Params, pageSize int) Iterator {
	return e.GetIteratorWithContext(e.Rest.GetCtx(), params, pageSize)
}

// Lock acquires the resource-level mutex and returns a function to release it.
// This allows for convenient deferring of unlock operations:
//
//	defer resource.Lock()()
func (e *VastResource) Lock(keys ...any) func() {
	return e.mu.Lock(keys...)
}

// ExtraMethodInfo contains information about an extra method discovered on a resource.
// Extra methods are non-CRUD operations that follow the pattern <MethodName>_<HTTPVerb>.
type ExtraMethodInfo struct {
	Name     string // Method name (e.g., "ViewCheckPermissionsTemplates_POST")
	HTTPVerb string // HTTP method (GET, POST, PATCH, DELETE)
	Path     string // Full path (e.g., "/views/{id}/check_permissions_templates/")
}

func (e *VastResource) String() string {
	// Use parent if available, otherwise fallback to self
	target := e.parent
	if target == nil {
		target = e
	}
	return e.describeResourceFrom(target)
}

// describeResourceFrom returns a comprehensive description of all endpoints for a resource
// parentResource should be the struct that embeds this VastResource (e.g., *Host)
func (e *VastResource) describeResourceFrom(parentResource any) string {
	var sb strings.Builder

	// Build header with resource name only (no operation flags)
	sb.WriteString(fmt.Sprintf("| %s\n", e.resourceType))

	// Check if resource has any operations
	hasAnyOps := e.resourceOps.isListable() || e.resourceOps.isReadable() ||
		e.resourceOps.isCreatable() || e.resourceOps.isUpdatable() || e.resourceOps.isDeletable()

	if !hasAnyOps {
		sb.WriteString("| supported hints:\n")
		sb.WriteString("|    [-]\n")
	} else {
		// Print supported hints section with operations grouped by type
		sb.WriteString("| supported hints:\n")

		// LIST operations
		if e.resourceOps.isListable() {
			sb.WriteString("|    [LIST]\n")
			sb.WriteString("|      - List / ListWithContext\n")
			sb.WriteString("|      - Get / GetWithContext\n")
			sb.WriteString("|      - Exists / ExistsWithContext\n")
		}

		// DETAILS operations
		if e.resourceOps.isReadable() {
			sb.WriteString("|    [DETAILS]\n")
			sb.WriteString("|      - GetById / GetByIdWithContext\n")
		}

		// CREATE operations
		if e.resourceOps.isCreatable() {
			sb.WriteString("|    [CREATE]\n")
			sb.WriteString("|      - Create / CreateWithContext\n")
			if e.resourceOps.isListable() {
				sb.WriteString("|      - Ensure / EnsureWithContext\n")
			}
		}

		// UPDATE operations
		if e.resourceOps.isUpdatable() {
			sb.WriteString("|    [UPDATE]\n")
			sb.WriteString("|      - Update / UpdateWithContext\n")
		}

		// DELETE operations
		if e.resourceOps.isDeletable() {
			sb.WriteString("|    [DELETE]\n")
			sb.WriteString("|      - Delete / DeleteWithContext\n")
			sb.WriteString("|      - DeleteById / DeleteByIdWithContext\n")
		}
	}

	// Discover and print extra methods
	extraMethods := DiscoverExtraMethodsFromResource(parentResource)
	if len(extraMethods) > 0 {
		sb.WriteString("| extra methods:\n")
		for _, extra := range extraMethods {
			if extra.Path != "" {
				sb.WriteString(fmt.Sprintf("|    - %s [%s]\n", extra.Name, extra.Path))
			}
		}
	}

	return sb.String()
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

func (e *TypedVastResource) String() string {
	return fmt.Sprintf("%s", e.getUntypedVastResource())
}

// GetIteratorWithContext creates a new iterator for paginated results using the provided context.
func (e *TypedVastResource) GetIteratorWithContext(ctx context.Context, params Params, pageSize int) Iterator {
	return e.getUntypedVastResource().(VastResourceAPIWithContext).GetIteratorWithContext(ctx, params, pageSize)
}

// GetIterator creates a new iterator for paginated results using the bound REST context.
func (e *TypedVastResource) GetIterator(params Params, pageSize int) Iterator {
	return e.getUntypedVastResource().GetIterator(params, pageSize)
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

// GetCRUDHintsFromResource is a helper function to extract CRUD operation hints from a resource.
// This is useful for introspection and tooling purposes (e.g., auto-generating widgets).
//
// Example:
//
//	hints := core.GetCRUDHintsFromResource(rest.Users)
//	canCreate := hints & core.C != 0
//	canList := hints & core.L != 0
func GetCRUDHintsFromResource(resource any) ResourceOps {
	// Try to type assert to *VastResource
	if vr, ok := resource.(*VastResource); ok {
		return vr.resourceOps
	}

	// Try VastResourceAPI interface which might have a *VastResource embedded
	if _, ok := resource.(VastResourceAPI); ok {
		// Use reflection to access the embedded *VastResource field
		val := reflect.ValueOf(resource)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		// Look for an embedded *VastResource field
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			if field.Type() == reflect.TypeOf((*VastResource)(nil)) {
				if vr, ok := field.Interface().(*VastResource); ok {
					return vr.resourceOps
				}
			}
		}
	}

	// Default: no operations supported
	return ResourceOps(0)
}

// DiscoverExtraMethodsFromResource discovers all extra methods for a resource using the metadata registry.
func DiscoverExtraMethodsFromResource(resource any) []ExtraMethodInfo {
	var resourceType string

	if api, ok := resource.(VastResourceAPI); ok {
		resourceType = api.GetResourceType()
	} else {
		if method := reflect.ValueOf(resource).MethodByName("GetResourceType"); method.IsValid() {
			if results := method.Call(nil); len(results) > 0 {
				resourceType = results[0].String()
			}
		}
	}

	// If we couldn't get the resource type, return empty
	if resourceType == "" {
		return []ExtraMethodInfo{}
	}

	metadataList := GetAllExtraMethodsForResource(resourceType)

	if len(metadataList) == 0 {
		// No metadata found - this means either:
		// 1. This resource has no extra methods
		// 2. The metadata files haven't been generated yet (run `make autogen`)
		return []ExtraMethodInfo{}
	}

	// Convert metadata to ExtraMethodInfo
	var discovered []ExtraMethodInfo
	for _, metadata := range metadataList {
		discovered = append(discovered, ExtraMethodInfo{
			Name:     metadata.MethodName,
			HTTPVerb: metadata.HTTPVerb,
			Path:     metadata.URLPath,
		})
	}

	return discovered
}
