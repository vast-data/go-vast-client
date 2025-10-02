package core

import (
	"context"
	"fmt"
	"net/http"
	"strings"
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
}

func NewVastResource(resourcePath string, resourceType string, rest VastRest) *VastResource {
	return &VastResource{
		resourcePath: resourcePath,
		resourceType: resourceType,
		Rest:         rest,
		mu:           NewKeyLocker(),
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
	return Request[RecordSet](ctx, e, http.MethodGet, e.resourcePath, params, nil)
}

// CreateWithContext creates a new resource using the provided parameters and context.
func (e *VastResource) CreateWithContext(ctx context.Context, body Params) (Record, error) {
	return Request[Record](ctx, e, http.MethodPost, e.resourcePath, nil, body)
}

// UpdateWithContext updates an existing resource by its ID using the provided parameters and context.
func (e *VastResource) UpdateWithContext(ctx context.Context, id any, body Params) (Record, error) {
	path := BuildResourcePathWithID(e.resourcePath, id)
	return Request[Record](ctx, e, http.MethodPatch, path, nil, body)
}

// UpdateNonIdWithContext updates a resource that does not use a numeric ID for identification.
// The resource is identified using unique fields within the provided parameters (e.g., SID, UID).
func (e *VastResource) UpdateNonIdWithContext(ctx context.Context, body Params) (Record, error) {
	return Request[Record](ctx, e, http.MethodPatch, e.resourcePath, nil, body)
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

// DeleteNonIdWithContext deletes a resource that does not use a numeric ID for identification.
// The resource is identified using unique fields within the provided parameters (e.g., SID, UID).
func (e *VastResource) DeleteNonIdWithContext(ctx context.Context, queryParams, deleteParams Params) (EmptyRecord, error) {
	return Request[EmptyRecord](ctx, e, http.MethodDelete, e.resourcePath, queryParams, deleteParams)
}

// DeleteByIdWithContext deletes a resource by its unique ID using the provided context and delete parameters.
func (e *VastResource) DeleteByIdWithContext(ctx context.Context, id any, queryParams, deleteParams Params) (EmptyRecord, error) {
	path := BuildResourcePathWithID(e.resourcePath, id)
	return Request[EmptyRecord](ctx, e, http.MethodDelete, path, queryParams, deleteParams)
}

// EnsureWithContext ensures a resource matching the search parameters exists. If not, it creates it using the body.
// All operations are performed within the given context.
func (e *VastResource) EnsureWithContext(ctx context.Context, searchParams Params, body Params) (Record, error) {
	result, err := e.GetWithContext(ctx, searchParams)
	if IsNotFoundErr(err) {
		return e.CreateWithContext(ctx, body)
	} else if err != nil {
		return nil, err
	}
	return result, nil
}

// EnsureByNameWithContext ensures a resource with the given name exists. If not, it creates one using the provided body.
// All operations are performed within the provided context.
func (e *VastResource) EnsureByNameWithContext(ctx context.Context, name string, body Params) (Record, error) {
	result, err := e.GetWithContext(ctx, Params{"name": name})
	if IsNotFoundErr(err) {
		body["name"] = name
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
	return Request[Record](ctx, e, http.MethodGet, path, nil, nil)
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

// UpdateNonId updates a resource that does not use a numeric ID for identification.
// The resource is identified using unique fields within the provided parameters (e.g., SID, UID).
// This method delegates to UpdateNonIdWithContext using the default context.
func (e *VastResource) UpdateNonId(params Params) (Record, error) {
	return e.UpdateNonIdWithContext(e.Rest.GetCtx(), params)
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

// DeleteNonId deletes a resource that does not use a numeric ID for identification.
// The resource is identified using unique fields within the provided parameters (e.g., SID, UID).
// This method delegates to DeleteNonIdWithContext using the default context.
func (e *VastResource) DeleteNonId(queryParams, deleteParams Params) (EmptyRecord, error) {
	return e.DeleteNonIdWithContext(e.Rest.GetCtx(), queryParams, deleteParams)
}

// Ensure ensures a resource exists matching the searchParams. Creates it with body if not found.
// Uses the bound REST context.
func (e *VastResource) Ensure(searchParams, body Params) (Record, error) {
	return e.EnsureWithContext(e.Rest.GetCtx(), searchParams, body)
}

// EnsureByName ensures a resource with the given name exists using the bound REST context.
// Creates it with the provided body if not found.
func (e *VastResource) EnsureByName(name string, body Params) (Record, error) {
	return e.EnsureByNameWithContext(e.Rest.GetCtx(), name, body)
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
