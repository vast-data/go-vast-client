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
func (e *VastResource) beforeRequest(_ context.Context, r *http.Request, verb, url string, body io.Reader) error {
	return nil
}

// afterRequest No op in current implementation. You have to shadow this method on particular VastResource
// IOW declare the same method with the same signature for Users or Quotas or Views etc.
func (e *VastResource) afterRequest(_ context.Context, response Renderable) (Renderable, error) {
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
		// Pre-normalization: attach @resourceType so resource hooks and user AfterRequestFn
		// can rely on it for formatting/logging/branching even if later mutations change shape.
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
	// Common VAST Response mutations (may unwrap pagination or otherwise replace the value)
	mutated, err := defaultResponseMutations(response)
	if err != nil {
		return nil, err
	}
	// Post-normalization: re-attach @resourceType, because mutations can produce new
	// Record/RecordSet instances which won't carry the earlier key.
	if !isDummyResource {
		if err = setResourceKey(mutated, resourceType); err != nil {
			return nil, err
		}
	}
	return mutated, nil
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
		// Normalize pagination envelope when response is a single Record
		if rs, matched, err := unwrapPaginationEnvelopeFromRecord(typed); matched {
			if err != nil {
				return nil, err
			}
			if rs != nil {
				return rs, nil
			}
		}
		return response, nil
	case RecordSet:
		// Normalize list responses that wrap data under a pagination envelope
		// Expected keys: "results", "count", "next", "previous"
		// Example payload returned as a single Record inside RecordSet: {"results": [...], "count": N, "next": ..., "previous": ...}
		if len(typed) == 1 {
			// Accept both Record and raw map[string]any as the envelope
			if rec, ok := any(typed[0]).(Record); ok {
				if rs, matched, err := unwrapPaginationEnvelopeFromRecord(rec); matched {
					if err != nil {
						return nil, err
					}
					if rs != nil {
						return rs, nil
					}
				}
			} else if raw, ok := any(typed[0]).(map[string]any); ok {
				if rs, matched, err := unwrapPaginationEnvelopeFromRecord(Record(raw)); matched {
					if err != nil {
						return nil, err
					}
					if rs != nil {
						return rs, nil
					}
				}
			}
		}
		return typed, nil
	case EmptyRecord:
		// No op.
		return typed, nil
	}
	return nil, fmt.Errorf("unsupported type %T for result", response)
}

// unwrapPaginationEnvelopeFromRecord attempts to detect and unwrap a standard pagination envelope
// of the form {"results": [...], "count": N, "next": ..., "previous": ...} into a RecordSet.
//
// Returns:
//   - (RecordSet, true, nil) when envelope matched and conversion succeeded
//   - (nil, true, nil) when envelope matched but results are of unsupported type
//   - (nil, false, nil) when envelope did not match
//   - (nil, true, err) when envelope matched but conversion failed
func unwrapPaginationEnvelopeFromRecord(rec Record) (RecordSet, bool, error) {
	_, hasResults := rec["results"]
	_, hasCount := rec["count"]
	_, hasNext := rec["next"]
	_, hasPrev := rec["previous"]
	if !(hasResults && hasCount && hasNext && hasPrev) {
		return nil, false, nil
	}
	inner := rec["results"]
	// Prefer []map[string]any, but also handle []any of maps
	if list, ok := inner.([]map[string]any); ok {
		recordSet, err := toRecordSet(list)
		if err != nil {
			return nil, true, err
		}
		return recordSet, true, nil
	}
	if anyList, ok := inner.([]any); ok {
		converted := make([]map[string]any, 0, len(anyList))
		for _, it := range anyList {
			if m, ok := it.(map[string]any); ok {
				converted = append(converted, m)
			} else {
				return nil, true, nil
			}
		}
		recordSet, err := toRecordSet(converted)
		if err != nil {
			return nil, true, err
		}
		return recordSet, true, nil
	}
	return nil, true, nil
}
