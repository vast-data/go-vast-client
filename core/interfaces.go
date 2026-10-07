package core

import (
	"context"
	"io"
	"net/http"
	"time"
)

// VastResourceAPI defines the interface for standard CRUD operations on a VAST resource.
type VastResourceAPI interface {
	Session() RESTSession
	GetResourceType() string
	GetResourcePath() string // normalized path to the resource in OpenAPI format
	// GetApiRoot returns the path segment after /api/{version}/ (empty for VMS; e.g. "serverless").
	GetApiRoot() string

	List(Params) (RecordSet, error)
	// Create accepts optional query via variadic Params:
	//   Create(body) or Create(query, body).
	Create(...Params) (Record, error)
	// Update accepts optional query via variadic Params:
	//   Update(id, body) or Update(id, query, body).
	Update(id any, params ...Params) (Record, error)
	// Delete accepts searchParams plus optional query/body via variadic deleteParams:
	//   Delete(search), Delete(search, body), or Delete(search, query, body).
	Delete(searchParams Params, deleteParams ...Params) (Record, error)
	// DeleteById accepts optional query/body via variadic Params:
	//   DeleteById(id), DeleteById(id, body), or DeleteById(id, query, body).
	DeleteById(id any, params ...Params) (Record, error)
	// Ensure accepts searchParams plus createParams (body or query+body):
	//   Ensure(search, body) or Ensure(search, query, body).
	Ensure(Params, ...Params) (Record, error)
	Get(Params) (Record, error)
	// GetById accepts an optional query Params: GetById(id) or GetById(id, query).
	GetById(any, ...Params) (Record, error)
	Exists(Params) (bool, error)
	MustExists(Params) bool
	GetIterator(Params, int) Iterator
	// Lock Resource-level mutex lock for concurrent access control
	Lock(...any) func()
	// Internal methods
}

type VastResourceAPIWithContext interface {
	VastResourceAPI
	ListWithContext(context.Context, Params) (RecordSet, error)
	// CreateWithContext accepts optional query via variadic Params:
	//   CreateWithContext(ctx, body) or CreateWithContext(ctx, query, body).
	CreateWithContext(context.Context, ...Params) (Record, error)
	// UpdateWithContext accepts optional query via variadic Params:
	//   UpdateWithContext(ctx, id, body) or UpdateWithContext(ctx, id, query, body).
	UpdateWithContext(ctx context.Context, id any, params ...Params) (Record, error)
	// DeleteWithContext accepts searchParams plus optional query/body via variadic deleteParams:
	//   DeleteWithContext(ctx, search), DeleteWithContext(ctx, search, body),
	//   or DeleteWithContext(ctx, search, query, body).
	DeleteWithContext(ctx context.Context, searchParams Params, deleteParams ...Params) (Record, error)
	// DeleteByIdWithContext accepts optional query/body via variadic Params:
	//   DeleteByIdWithContext(ctx, id), DeleteByIdWithContext(ctx, id, body),
	//   or DeleteByIdWithContext(ctx, id, query, body).
	DeleteByIdWithContext(ctx context.Context, id any, params ...Params) (Record, error)
	// EnsureWithContext accepts searchParams plus createParams (body or query+body):
	//   EnsureWithContext(ctx, search, body) or EnsureWithContext(ctx, search, query, body).
	EnsureWithContext(context.Context, Params, ...Params) (Record, error)
	GetWithContext(context.Context, Params) (Record, error)
	// GetByIdWithContext accepts an optional query Params:
	//   GetByIdWithContext(ctx, id) or GetByIdWithContext(ctx, id, query).
	GetByIdWithContext(context.Context, any, ...Params) (Record, error)
	ExistsWithContext(context.Context, Params) (bool, error)
	MustExistsWithContext(context.Context, Params) bool
	GetIteratorWithContext(context.Context, Params, int) Iterator
}

// InterceptableVastResourceAPI combines request interception with vast resource behavior.
type InterceptableVastResourceAPI interface {
	RequestInterceptor
	VastResourceAPIWithContext
}

type Awaitable interface {
	WaitWithContext(context.Context) (Record, error)
	Wait(time.Duration) (Record, error)
}

// TaskWaiter defines an interface for resources that can wait on asynchronous tasks.
// This interface is primarily implemented by the VTask resource to provide task polling capabilities.
// It allows the core package to call task waiting functionality without creating a circular dependency
// with the untyped package.
type TaskWaiter interface {
	WaitTaskWithContext(ctx context.Context, taskId int64, timeout time.Duration) (Record, error)
	WaitTask(taskId int64, timeout time.Duration) (Record, error)
}

// RequestInterceptor defines a middleware-style interface for intercepting API requests
// and responses in client-server interactions. It allows implementing logic that runs
// before sending a request and after receiving a response.
// Typical use cases include logging, request mutation, authentication, and response transformation.
type RequestInterceptor interface {
	// BeforeRequest is invoked prior to sending the API request.
	//
	// Parameters:
	//   - ctx: The request context, useful for deadlines, tracing, or cancellation.
	//   - req: Request object
	//   - verb: The HTTP method (e.g., GET, POST, PUT).
	//   - url: The URL path being accessed (including query params)
	//   - body: The request body as an io.Reader, typically containing JSON data.
	BeforeRequest(context.Context, *http.Request, string, string, io.Reader) error

	// AfterRequest is invoked after the API response is received.
	//
	// The input and output are of type Renderable, which includes types like:
	//   - Record: a single key-value response object
	//   - RecordSet: a list of Record objects
	//   - Record: a map response (may be empty Record{} for DELETE operations)
	//
	// This method can inspect, mutate, or log the response data.
	//
	// Returns:
	//   - A (possibly modified) Renderable
	//   - An error if the interceptor encounters issues processing the response
	AfterRequest(context.Context, Renderable) (Renderable, error)

	// doBeforeRequest No need to implement on VAST API Resources. For internal usage only
	doBeforeRequest(context.Context, *http.Request, string, string, io.Reader) error

	// doAfterRequest No need to implement on VAST API Resources. For internal usage only
	doAfterRequest(context.Context, Renderable, int) (Renderable, error)
}

type VastRest interface {
	GetSession() RESTSession
	GetResourceMap() map[string]ResourceEntry
	GetCtx() context.Context
	SetCtx(context.Context)
	// GetApiRoot returns the rest-level path segment after /api/{version}/.
	// Empty for the main VMS rest; nested rests set their own (e.g. "serverless").
	GetApiRoot() string
}

// ResourceEntry is the value stored in GetResourceMap.
// The embedded API keeps method call sites working; IdentityField is "id", "guid",
// or "" when the resource has no OpenAPI item path ({id}|{guid}).
type ResourceEntry struct {
	VastResourceAPIWithContext
	IdentityField string
}

// Iterator provides an interface for iterating over paginated or non-paginated API results.
// It abstracts away the differences between pagination models (VMS DRF links, DataEngine
// cursors, or future strategies). Implementations are internal; callers use GetIterator.
type Iterator interface {
	// Next advances to the next page and returns the records and any error.
	// Returns empty RecordSet when there are no more pages.
	Next() (RecordSet, error)

	// Previous moves to the previous page and returns the records and any error.
	// Returns empty RecordSet when there is no previous page.
	Previous() (RecordSet, error)

	// HasNext returns true if there is a next page available.
	HasNext() bool

	// HasPrevious returns true if there is a previous page available.
	HasPrevious() bool

	// Count returns the total count of items (if available from pagination metadata).
	// Returns -1 if count information is not available.
	Count() int

	// PageSize returns the current page size.
	PageSize() int

	// Reset resets the iterator to the first page and returns the first page records.
	Reset() (RecordSet, error)

	// All fetches all remaining pages and returns all records as a single RecordSet.
	// This should be used with caution for large datasets.
	All() (RecordSet, error)
}
