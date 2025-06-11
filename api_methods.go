package vast_client

import (
	"context"
	"errors"
	"fmt"
	version "github.com/hashicorp/go-version"
	"net/http"
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

func isTooManyRecordsErr(err error) bool {
	var tooManyRecordsErr *TooManyRecordsError
	if errors.As(err, &tooManyRecordsErr) {
		return true
	}
	return false
}

// VastResource defines the interface for standard CRUD operations on a VAST resource.
type VastResource interface {
	Session() RESTSession
	GetResourceType() string

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

	// Internal methods
	sync.Locker
	getRest() *VMSRest
	getAvailableFromVersion() *version.Version
}

// VastResourceWithContext VastResource method with pre-bound Rest context
type VastResourceWithContext interface {
	VastResource
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
}

// InterceptableVastResource combines request interception with vast resource behavior.
type InterceptableVastResource interface {
	RequestInterceptor
	VastResourceWithContext
}

type Awaitable interface {
	WaitWithContext(context.Context) (Record, error)
	Wait() (Record, error)
}

// VastResourceEntry implements VastResource and provides common behavior for managing VAST resources.
type VastResourceEntry struct {
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
func (e *VastResourceEntry) Session() RESTSession {
	return e.rest.Session
}

func (e *VastResourceEntry) GetResourceType() string {
	return e.resourceType
}

// ListWithContext retrieves all resources matching the given parameters using the provided context.
func (e *VastResourceEntry) ListWithContext(ctx context.Context, params Params) (RecordSet, error) {
	return request[RecordSet](ctx, e, http.MethodGet, e.resourcePath, e.apiVersion, params, nil)
}

// CreateWithContext creates a new resource using the provided parameters and context.
func (e *VastResourceEntry) CreateWithContext(ctx context.Context, body Params) (Record, error) {
	return request[Record](ctx, e, http.MethodPost, e.resourcePath, e.apiVersion, nil, body)
}

// UpdateWithContext updates an existing resource by its ID using the provided parameters and context.
func (e *VastResourceEntry) UpdateWithContext(ctx context.Context, id int64, body Params) (Record, error) {
	path := fmt.Sprintf("%s/%d", e.resourcePath, id)
	return request[Record](ctx, e, http.MethodPatch, path, e.apiVersion, nil, body)
}

// DeleteWithContext deletes a resource found using searchParams, using the provided deleteParams, within the given context.
// If the resource is not found, it returns success without error.
func (e *VastResourceEntry) DeleteWithContext(ctx context.Context, searchParams, deleteParams Params) (EmptyRecord, error) {
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
func (e *VastResourceEntry) DeleteByIdWithContext(ctx context.Context, id int64, deleteParams Params) (EmptyRecord, error) {
	path := fmt.Sprintf("%s/%d", e.resourcePath, id)
	return request[EmptyRecord](ctx, e, http.MethodDelete, path, e.apiVersion, nil, deleteParams)
}

// EnsureWithContext ensures a resource matching the search parameters exists. If not, it creates it using the body.
// All operations are performed within the given context.
func (e *VastResourceEntry) EnsureWithContext(ctx context.Context, searchParams Params, body Params) (Record, error) {
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
func (e *VastResourceEntry) EnsureByNameWithContext(ctx context.Context, name string, body Params) (Record, error) {
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
func (e *VastResourceEntry) GetWithContext(ctx context.Context, params Params) (Record, error) {
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
func (e *VastResourceEntry) GetByIdWithContext(ctx context.Context, id int64) (Record, error) {
	path := fmt.Sprintf("%s/%d", e.resourcePath, id)
	return request[Record](ctx, e, http.MethodGet, path, e.apiVersion, nil, nil)
}

// ExistsWithContext checks if any resource matches the provided parameters within the given context.
// Returns true if a match is found. Returns false if not found. Returns an error only if an unexpected failure occurs.
func (e *VastResourceEntry) ExistsWithContext(ctx context.Context, params Params) (bool, error) {
	if _, err := e.GetWithContext(ctx, params); err != nil && !isTooManyRecordsErr(err) {
		if !IsNotFoundErr(err) {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

// List retrieves all resources matching the given parameters using the bound REST context.
func (e *VastResourceEntry) List(params Params) (RecordSet, error) {
	return e.ListWithContext(e.rest.ctx, params)
}

// Create creates a new resource using the provided parameters and the bound REST context.
func (e *VastResourceEntry) Create(params Params) (Record, error) {
	return e.CreateWithContext(e.rest.ctx, params)
}

// Update updates a resource by its ID using the provided parameters and the bound REST context.
func (e *VastResourceEntry) Update(id int64, params Params) (Record, error) {
	return e.UpdateWithContext(e.rest.ctx, id, params)
}

// Delete deletes a resource found with searchParams using deleteParams and the bound REST context.
// Returns success even if the resource is not found.
func (e *VastResourceEntry) Delete(searchParams, deleteParams Params) (EmptyRecord, error) {
	return e.DeleteWithContext(e.rest.ctx, searchParams, deleteParams)
}

// DeleteById deletes a resource by its ID using the bound REST context and provided deleteParams.
func (e *VastResourceEntry) DeleteById(id int64, deleteParams Params) (EmptyRecord, error) {
	return e.DeleteByIdWithContext(e.rest.ctx, id, deleteParams)
}

// Ensure ensures a resource exists matching the searchParams. Creates it with body if not found.
// Uses the bound REST context.
func (e *VastResourceEntry) Ensure(searchParams, body Params) (Record, error) {
	return e.EnsureWithContext(e.rest.ctx, searchParams, body)
}

// EnsureByName ensures a resource with the given name exists using the bound REST context.
// Creates it with the provided body if not found.
func (e *VastResourceEntry) EnsureByName(name string, body Params) (Record, error) {
	return e.EnsureByNameWithContext(e.rest.ctx, name, body)
}

// Get retrieves a single resource matching the given parameters using the bound REST context.
// Returns NotFoundError if the resource does not exist.
func (e *VastResourceEntry) Get(params Params) (Record, error) {
	return e.GetWithContext(e.rest.ctx, params)
}

// GetById retrieves a resource by its ID using the bound REST context.
func (e *VastResourceEntry) GetById(id int64) (Record, error) {
	return e.GetByIdWithContext(e.rest.ctx, id)
}

// Exists checks if any resource matches the given parameters using the bound REST context.
// Returns true if a match is found, false if not. Returns error only for unexpected issues.
func (e *VastResourceEntry) Exists(params Params) (bool, error) {
	return e.ExistsWithContext(e.rest.ctx, params)
}

func (e *VastResourceEntry) Lock() {
	e.mu.Lock()
}

func (e *VastResourceEntry) Unlock() {
	e.mu.Lock()
}

// internal methods
// getRest Rest returns Rest object
func (e *VastResourceEntry) getRest() *VMSRest {
	return e.rest
}

// getAvailableFromVersion Get minimal VAST version resource is available from.
func (e *VastResourceEntry) getAvailableFromVersion() *version.Version {
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
