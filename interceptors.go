package vast_client

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// RequestInterceptor defines a middleware-style interface for intercepting API requests
// and responses in client-server interactions. It allows implementing logic that runs
// before sending a request and after receiving a response.
// Typical use cases include logging, request mutation, authentication, and response transformation.
type RequestInterceptor interface {
	// beforeRequest is invoked prior to sending the API request.
	//
	// Parameters:
	//   - ctx: The request context, useful for deadlines, tracing, or cancellation.
	//   - req: Request object
	//   - verb: The HTTP method (e.g., GET, POST, PUT).
	//   - url: The URL path being accessed (including query params)
	//   - body: The request body as an io.Reader, typically containing JSON data.
	beforeRequest(context.Context, *http.Request, string, string, io.Reader) error

	// afterRequest is invoked after the API response is received.
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
	afterRequest(context.Context, Renderable) (Renderable, error)

	// doBeforeRequest No need to implement on VAST API Resources. For internal usage only
	doBeforeRequest(context.Context, *http.Request, string, string, io.Reader) error

	// doAfterRequest No need to implement on VAST API Resources. For internal usage only
	doAfterRequest(context.Context, Renderable) (Renderable, error)
}

// ######################################################
//
//	REQUEST/RESPONSE INTERCEPTORS
//
// ######################################################

// beforeRequest No op in current implementation. You have to shadow this method on particular VastResource
// IOW declare the same method with the same signature for Users or Quotas or Views etc.
func (e *VastResource) beforeRequest(ctx context.Context, r *http.Request, verb, url string, body io.Reader) error {
	return nil
}

// afterRequest No op in current implementation. You have to shadow this method on particular VastResource
// IOW declare the same method with the same signature for Users or Quotas or Views etc.
func (e *VastResource) afterRequest(ctx context.Context, response Renderable) (Renderable, error) {
	return response, nil
}

// doBeforeRequest Do not override this method in VastResource implementations. For internal use only
func (e *VastResource) doBeforeRequest(ctx context.Context, r *http.Request, verb, url string, body io.Reader) error {
	var err error
	session := e.Session()
	config := session.GetConfig()
	resourceType := e.GetResourceType()
	resourceCaller, ok := e.rest.resourceMap[resourceType]
	if !ok {
		panic(fmt.Sprintf("resource not found in resourceMap for %s", resourceType))
	}
	if interceptor, ok := resourceCaller.(RequestInterceptor); ok {
		if err = interceptor.beforeRequest(ctx, r, verb, url, body); err != nil {
			return err
		}
	}
	// User-defined callback
	if config.BeforeRequestFn != nil {
		return config.BeforeRequestFn(ctx, r, verb, url, body)
	}
	return nil
}

// doAfterRequest Do not override this method in VastResource implementations. For internal use only
func (e *VastResource) doAfterRequest(ctx context.Context, response Renderable) (Renderable, error) {
	var err error
	session := e.Session()
	config := session.GetConfig()
	resourceType := e.GetResourceType()
	isDummyResource := resourceType == "Dummy"
	resourceCaller, ok := e.rest.resourceMap[resourceType]
	if !ok {
		panic(fmt.Sprintf("resource not found in resourceMap for %s", e.GetResourceType()))
	}
	if !isDummyResource {
		// Set resource key only if type of response is the same as declared type.
		if err = setResourceKey(response, resourceType); err != nil {
			return nil, err
		}
	}
	if interceptor, ok := resourceCaller.(RequestInterceptor); ok {
		response, err = interceptor.afterRequest(ctx, response)
		if err != nil {
			return nil, err
		}
	}
	// User-defined callback
	if config.AfterRequestFn != nil {
		response, err = config.AfterRequestFn(ctx, response)
		if err != nil {
			return nil, err
		}
	}
	// Common VAST Response mutations.
	return defaultResponseMutations(response)
}

// defaultResponseMutations A set of common response transformations in the VAST REST API
// that can be universally applied across all resource types.
func defaultResponseMutations(response Renderable) (Renderable, error) {
	switch typed := response.(type) {
	case Record:
		// Case when VAST Response returns Async Task instead of actual response is very common in VAST API so can be applied here
		// NOTE: This mutation just normalizes response so you can retrieve id and wait for task to be completed.
		//       Waiting is not accomplished here.
		if raw, ok := response.(Record)["async_task"]; ok {
			var m map[string]any
			if m, ok = raw.(map[string]any); ok {
				m[resourceTypeKey] = "VTask"
				return toRecord(m)
			}
			return nil, fmt.Errorf("expected map[string]any under 'async_task', got %T", raw)
		}
		return response, nil
	case RecordSet:
		// Add mutation for each Record in RecordSet if needed
		return typed, nil
	case EmptyRecord:
		// No op.
		return typed, nil
	}
	return nil, fmt.Errorf("unsupported type %T for result", response)
}
