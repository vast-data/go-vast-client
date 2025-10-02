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

	List(Params) (RecordSet, error)
	Create(Params) (Record, error)
	Update(any, Params) (Record, error)
	Delete(Params, Params) (EmptyRecord, error)
	DeleteById(any, Params, Params) (EmptyRecord, error)
	Ensure(Params, Params) (Record, error)
	Get(Params) (Record, error)
	GetById(any) (Record, error)
	Exists(Params) (bool, error)
	MustExists(Params) bool
	// Resource-level mutex lock for concurrent access control
	Lock(...any) func()
	// Internal methods
}

type VastResourceAPIWithContext interface {
	VastResourceAPI
	ListWithContext(context.Context, Params) (RecordSet, error)
	CreateWithContext(context.Context, Params) (Record, error)
	UpdateWithContext(context.Context, any, Params) (Record, error)
	DeleteWithContext(context.Context, Params, Params, Params) (EmptyRecord, error)
	DeleteByIdWithContext(context.Context, any, Params, Params) (EmptyRecord, error)
	EnsureWithContext(context.Context, Params, Params) (Record, error)
	GetWithContext(context.Context, Params) (Record, error)
	GetByIdWithContext(context.Context, any) (Record, error)
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
	Wait(time.Duration) (Record, error)
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
	//   - EmptyRecord: an empty object used for operations like DELETE
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
	doAfterRequest(context.Context, Renderable) (Renderable, error)
}

type VastRest interface {
	GetSession() RESTSession
	GetResourceMap() map[string]VastResourceAPIWithContext
	GetCtx() context.Context
	SetCtx(context.Context)
}
