package vast_client

import (
	"context"
	"errors"
	"fmt"
	version "github.com/hashicorp/go-version"
	"net/http"
	"strings"
	"sync"
)

//  ######################################################
//              VAST RESOURCES BASE CRUD OPS
//  ######################################################

type NotFoundError struct {
	Resource string
	Query    string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("resource '%s' not found for params '%s'", e.Resource, e.Query)
}

type TooManyRecordsError struct {
	ResourcePath string
	Params       Params
}

// Implement the Error method to satisfy the error interface
func (e *TooManyRecordsError) Error() string {
	return fmt.Sprintf("too many records found for resource '%s' with params '%v'", e.ResourcePath, e.Params)
}

func IsNotFoundErr(err error) bool {
	var nfErr *NotFoundError
	if errors.As(err, &nfErr) {
		return true
	}
	return false
}

func IgnoreNotFound(val Record, err error) (Record, error) {
	if IsNotFoundErr(err) {
		return val, nil
	}
	return val, err
}

func isTooManyRecordsErr(err error) bool {
	var tooManyRecordsErr *TooManyRecordsError
	if errors.As(err, &tooManyRecordsErr) {
		return true
	}
	return false
}

// VastResourceAPI defines the interface for standard CRUD operations on a VAST resource.
type VastResourceAPI interface {
	Session() RESTSession
	GetResourceType() string
	GetResourcePath() string // normalized path to the resource in OpenAPI format

	List(Params) (RecordSet, error)
	Create(Params) (Record, error)
	Update(int64, Params) (Record, error)
	Delete(Params, Params) (EmptyRecord, error)
	DeleteById(int64, Params) (EmptyRecord, error)
	Ensure(Params, Params) (Record, error)
	EnsureByName(string, Params) (Record, error)
	Get(Params) (Record, error)
	GetById(int64) (Record, error)
	Exists(Params) (bool, error)
	MustExists(Params) bool

	// Internal methods
	sync.Locker
	getRest() *VMSRest
	getAvailableFromVersion() *version.Version
}

type VastResourceAPIWithContext interface {
	VastResourceAPI
	ListWithContext(context.Context, Params) (RecordSet, error)
	CreateWithContext(context.Context, Params) (Record, error)
	UpdateWithContext(context.Context, int64, Params) (Record, error)
	DeleteWithContext(context.Context, Params, Params) (EmptyRecord, error)
	DeleteByIdWithContext(context.Context, int64, Params) (EmptyRecord, error)
	EnsureWithContext(context.Context, Params, Params) (Record, error)
	EnsureByNameWithContext(context.Context, string, Params) (Record, error)
	GetWithContext(context.Context, Params) (Record, error)
	GetByIdWithContext(context.Context, int64) (Record, error)
	ExistsWithContext(context.Context, Params) (bool, error)
	MustExistsWithContext(context.Context, Params) bool
}

// InterceptableVastResourceAPI combines request interception with vast resource behavior.
type InterceptableVastResourceAPI interface {
	RequestInterceptor
	VastResourceAPIWithContext
}

type Awaitable interface {
	WaitWithContext(context.Context) (Record, error)
	Wait() (Record, error)
}

// VastResource implements VastResource and provides common behavior for managing VAST resources.
type VastResource struct {
	resourcePath         string
	resourceType         string
	apiVersion           string
	availableFromVersion *version.Version
	rest                 *VMSRest
	mu                   sync.Mutex
}

// AsyncResult represents the result of an asynchronous task.
// It contains the task's ID and necessary context for waiting on the task to complete.
type AsyncResult struct {
	TaskId int64
	rest   VMSRest
	ctx    context.Context
}

// Session returns the current VMSSession associated with the resource.
func (e *VastResource) Session() RESTSession {
	return e.rest.Session
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
	return request[RecordSet](ctx, e, http.MethodGet, e.resourcePath, e.apiVersion, params, nil)
}

// CreateWithContext creates a new resource using the provided parameters and context.
func (e *VastResource) CreateWithContext(ctx context.Context, body Params) (Record, error) {
	return request[Record](ctx, e, http.MethodPost, e.resourcePath, e.apiVersion, nil, body)
}

// UpdateWithContext updates an existing resource by its ID using the provided parameters and context.
func (e *VastResource) UpdateWithContext(ctx context.Context, id int64, body Params) (Record, error) {
	path := fmt.Sprintf("%s/%d", e.resourcePath, id)
	return request[Record](ctx, e, http.MethodPatch, path, e.apiVersion, nil, body)
}

// DeleteWithContext deletes a resource found using searchParams, using the provided deleteParams, within the given context.
// If the resource is not found, it returns success without error.
func (e *VastResource) DeleteWithContext(ctx context.Context, searchParams, deleteParams Params) (EmptyRecord, error) {
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
	idInt, err := toInt(idVal)
	if err != nil {
		return nil, err
	}
	return e.DeleteByIdWithContext(ctx, idInt, deleteParams)
}

// DeleteByIdWithContext deletes a resource by its unique ID using the provided context and delete parameters.
func (e *VastResource) DeleteByIdWithContext(ctx context.Context, id int64, deleteParams Params) (EmptyRecord, error) {
	path := fmt.Sprintf("%s/%d", e.resourcePath, id)
	return request[EmptyRecord](ctx, e, http.MethodDelete, path, e.apiVersion, nil, deleteParams)
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
	result, err := request[RecordSet](ctx, e, http.MethodGet, e.resourcePath, e.apiVersion, params, nil)
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
		if singleResult := result[0]; singleResult.empty() {
			return nil, &NotFoundError{
				Resource: e.resourcePath,
				Query:    params.ToQuery(),
			}
		} else {
			return singleResult, nil
		}
	default:
		return nil, &TooManyRecordsError{
			ResourcePath: e.resourcePath,
			Params:       params,
		}
	}
}

// GetByIdWithContext retrieves a resource by its unique ID using the provided context.
func (e *VastResource) GetByIdWithContext(ctx context.Context, id int64) (Record, error) {
	path := fmt.Sprintf("%s/%d", e.resourcePath, id)
	return request[Record](ctx, e, http.MethodGet, path, e.apiVersion, nil, nil)
}

// ExistsWithContext checks if any resource matches the provided parameters within the given context.
// Returns true if a match is found. Returns false if not found. Returns an error only if an unexpected failure occurs.
func (e *VastResource) ExistsWithContext(ctx context.Context, params Params) (bool, error) {
	if _, err := e.GetWithContext(ctx, params); err != nil && !isTooManyRecordsErr(err) {
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
	return must(e.ExistsWithContext(ctx, params))
}

// List retrieves all resources matching the given parameters using the bound REST context.
func (e *VastResource) List(params Params) (RecordSet, error) {
	return e.ListWithContext(e.rest.ctx, params)
}

// Create creates a new resource using the provided parameters and the bound REST context.
func (e *VastResource) Create(params Params) (Record, error) {
	return e.CreateWithContext(e.rest.ctx, params)
}

// Update updates a resource by its ID using the provided parameters and the bound REST context.
func (e *VastResource) Update(id int64, params Params) (Record, error) {
	return e.UpdateWithContext(e.rest.ctx, id, params)
}

// Delete deletes a resource found with searchParams using deleteParams and the bound REST context.
// Returns success even if the resource is not found.
func (e *VastResource) Delete(searchParams, deleteParams Params) (EmptyRecord, error) {
	return e.DeleteWithContext(e.rest.ctx, searchParams, deleteParams)
}

// DeleteById deletes a resource by its ID using the bound REST context and provided deleteParams.
func (e *VastResource) DeleteById(id int64, deleteParams Params) (EmptyRecord, error) {
	return e.DeleteByIdWithContext(e.rest.ctx, id, deleteParams)
}

// Ensure ensures a resource exists matching the searchParams. Creates it with body if not found.
// Uses the bound REST context.
func (e *VastResource) Ensure(searchParams, body Params) (Record, error) {
	return e.EnsureWithContext(e.rest.ctx, searchParams, body)
}

// EnsureByName ensures a resource with the given name exists using the bound REST context.
// Creates it with the provided body if not found.
func (e *VastResource) EnsureByName(name string, body Params) (Record, error) {
	return e.EnsureByNameWithContext(e.rest.ctx, name, body)
}

// Get retrieves a single resource matching the given parameters using the bound REST context.
// Returns NotFoundError if the resource does not exist.
func (e *VastResource) Get(params Params) (Record, error) {
	return e.GetWithContext(e.rest.ctx, params)
}

// GetById retrieves a resource by its ID using the bound REST context.
func (e *VastResource) GetById(id int64) (Record, error) {
	return e.GetByIdWithContext(e.rest.ctx, id)
}

// Exists checks if any resource matches the given parameters using the bound REST context.
// Returns true if a match is found, false if not. Returns error only for unexpected issues.
func (e *VastResource) Exists(params Params) (bool, error) {
	return e.ExistsWithContext(e.rest.ctx, params)
}

// MustExists performs an existence check for a resource using the given parameters.
// It returns true if the resource exists, or false if it does not.
// If an error occurs during the check (other than not-found), the method panics.
// This is a convenience method intended for use in control paths where failures are not expected or tolerated.
func (e *VastResource) MustExists(params Params) bool {
	return e.MustExistsWithContext(e.rest.ctx, params)
}

func (e *VastResource) Lock() {
	e.mu.Lock()
}

func (e *VastResource) Unlock() {
	e.mu.Lock()
}

// internal methods
// getRest Rest returns Rest object
func (e *VastResource) getRest() *VMSRest {
	return e.rest
}

// getAvailableFromVersion Get minimal VAST version resource is available from.
func (e *VastResource) getAvailableFromVersion() *version.Version {
	return e.availableFromVersion
}

// Wait blocks until the asynchronous task completes and returns the resulting Record.
// If the context (ar.ctx) is not set, it falls back to the context from the associated rest client.
func (ar *AsyncResult) Wait() (Record, error) {
	ctx := ar.ctx
	if ctx == nil {
		ctx = ar.rest.ctx
	}
	return ar.WaitWithContext(ctx)
}

// WaitWithContext blocks until the asynchronous task completes or the provided context is cancelled.
// It delegates to the VTasks.WaitTaskWithContext method of the rest client to poll for task completion.
func (ar *AsyncResult) WaitWithContext(ctx context.Context) (Record, error) {
	return ar.rest.VTasks.WaitTaskWithContext(ctx, ar.TaskId)
}

func asyncResultFromRecord(ctx context.Context, r Record) *AsyncResult {
	taskId := r.RecordID()
	return &AsyncResult{
		TaskId: taskId,
		ctx:    ctx,
	}
}
