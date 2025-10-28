package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

var logLevel string

func init() {
	logLevel = strings.ToLower(os.Getenv("VAST_LOG"))
}

// ######################################################
//
//	REQUEST/RESPONSE INTERCEPTORS
//
// ######################################################

// BeforeRequest No op in current implementation. You have to shadow this method on particular VastResource
// IOW declare the same method with the same signature for Users or Quotas or Views etc.
func (e *VastResource) BeforeRequest(_ context.Context, r *http.Request, verb, url string, body io.Reader) error {
	return nil
}

// AfterRequest No op in current implementation. You have to shadow this method on particular VastResource
// IOW declare the same method with the same signature for Users or Quotas or Views etc.
func (e *VastResource) AfterRequest(_ context.Context, response Renderable) (Renderable, error) {
	return response, nil
}

// DoBeforeRequest Do not override this method in VastResource implementations. For internal use only
func (e *VastResource) doBeforeRequest(ctx context.Context, r *http.Request, verb, url string, body io.Reader) error {
	var err error
	session := e.Session()
	config := session.GetConfig()
	resourceType := e.GetResourceType()
	resourceCaller, ok := e.Rest.GetResourceMap()[resourceType]
	if !ok {
		panic(fmt.Sprintf("resource not found in resourceMap for %s", resourceType))
	}
	if logLevel != "" {
		beforeRequestLog(verb, url, body)
	}
	if interceptor, ok := resourceCaller.(RequestInterceptor); ok {
		if err = interceptor.BeforeRequest(ctx, r, verb, url, body); err != nil {
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
	resourceCaller, ok := e.Rest.GetResourceMap()[resourceType]
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
	if logLevel != "" {
		afterRequestLog(response)
	}
	if interceptor, ok := resourceCaller.(RequestInterceptor); ok {
		response, err = interceptor.AfterRequest(ctx, response)
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
				m[ResourceTypeKey] = "VTask"
				return ToRecord(m), nil
			}
			return nil, fmt.Errorf("expected map[string]any under 'async_task', got %T", raw)
		}
		return response, nil
	case RecordSet:
		return typed, nil
	}
	return nil, fmt.Errorf("unsupported type %T for result", response)
}

// ######################################################
//
//	REQUEST/RESPONSE LOGGING
//
// ######################################################

// beforeRequestLog logs HTTP request details before sending the request.
// In debug mode, it includes the request body (if present).
// In info mode, it only logs the HTTP method and URL.
//
// Parameters:
//   - verb: HTTP method (GET, POST, PUT, DELETE, etc.)
//   - url: The request URL
//   - body: Optional request body reader
func beforeRequestLog(verb, url string, body io.Reader) {
	requestInfo := fmt.Sprintf("http request start: [%s] %s", verb, url)
	var bodyMsg string

	// In debug mode, read and format the request body
	if body != nil && logLevel == "debug" {
		bodyBytes, err := io.ReadAll(body)
		if err != nil {
			log.Printf("ERROR: failed to read request body: %v", err)
			return
		}

		trimmed := bytes.TrimSpace(bodyBytes)
		if len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null")) {
			var compact bytes.Buffer
			if err := json.Compact(&compact, trimmed); err == nil {
				bodyMsg = compact.String()
			} else {
				bodyMsg = string(trimmed)
			}
		}
	}

	if bodyMsg == "" {
		log.Printf("INFO: %s", requestInfo)
	} else {
		log.Printf("DEBUG: %s | body: %s", requestInfo, bodyMsg)
	}
}

// afterRequestLog logs HTTP response details after receiving the response.
// In debug mode, it pretty-prints the full response data using PrettyJson.
// In info mode, it only logs a summary (record count, resource type, etc.).
//
// Parameters:
//   - response: The response object (Record or RecordSet)
func afterRequestLog(response Renderable) {
	if logLevel == "debug" {
		// Debug mode: print full response using PrettyJson
		afterRequestLogDebug(response)
	} else {
		// Info mode: print summary only
		afterRequestLogInfo(response)
	}
}

// afterRequestLogInfo logs a summary of the response (info level).
// It includes the count and type of records returned.
func afterRequestLogInfo(response Renderable) {
	var responseStr string

	switch resp := response.(type) {
	case Record:
		if resourceType, ok := resp[ResourceTypeKey].(string); ok && resourceType != "" {
			responseStr = fmt.Sprintf("Record of type: %s", resourceType)
		} else {
			responseStr = "Record received"
		}
	case RecordSet:
		count := len(resp)
		if count > 0 {
			firstRecord := resp[0]
			if resourceType, ok := firstRecord[ResourceTypeKey].(string); ok && resourceType != "" {
				responseStr = fmt.Sprintf("RecordSet with %d record(s) of type: %s", count, resourceType)
			} else {
				responseStr = fmt.Sprintf("RecordSet with %d record(s)", count)
			}
		} else {
			responseStr = "RecordSet with 0 record(s)"
		}
	default:
		responseStr = "Response received"
	}

	log.Printf("INFO: response | %s", responseStr)
}

// afterRequestLogDebug logs the full response data (debug level).
// It uses PrettyJson to format the response for better readability.
func afterRequestLogDebug(response Renderable) {
	var header string
	var body string

	switch resp := response.(type) {
	case Record:
		if resourceType, ok := resp[ResourceTypeKey].(string); ok && resourceType != "" {
			header = fmt.Sprintf("response |")
		} else {
			header = "response | Record received"
		}
		body = resp.PrettyJson("  ")
	case RecordSet:
		count := len(resp)
		if count > 0 {
			firstRecord := resp[0]
			if resourceType, ok := firstRecord[ResourceTypeKey].(string); ok && resourceType != "" {
				header = fmt.Sprintf("response | RecordSet with %d record(s) of type: %s", count, resourceType)
			} else {
				header = fmt.Sprintf("response | RecordSet with %d record(s)", count)
			}
		} else {
			header = "response | RecordSet with 0 record(s)"
		}
		body = resp.PrettyJson("  ")
	default:
		header = "response | Response received"
		body = fmt.Sprintf("%v", response)
	}

	log.Printf("DEBUG: %s\n%s", header, body)
}
