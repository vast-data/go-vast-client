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
	resourceCaller, ok := e.Rest.GetResourceMap()[resourceType]
	if !ok {
		panic(fmt.Sprintf("resource not found in resourceMap for %s", e.GetResourceType()))
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
	return response, nil
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
		if displayName := recordDisplayName(resp); displayName != "" {
			responseStr = fmt.Sprintf("Record of type: %s", displayName)
		} else {
			responseStr = "Record received"
		}
	case RecordSet:
		count := len(resp)
		if count > 0 {
			if displayName := recordDisplayName(resp[0]); displayName != "" {
				responseStr = fmt.Sprintf("RecordSet with %d record(s) of type: %s", count, displayName)
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
		if displayName := recordDisplayName(resp); displayName != "" {
			header = "response |"
		} else {
			header = "response | Record received"
		}
		body = resp.PrettyJson("  ")
	case RecordSet:
		count := len(resp)
		if count > 0 {
			if displayName := recordDisplayName(resp[0]); displayName != "" {
				header = fmt.Sprintf("response | RecordSet with %d record(s) of type: %s", count, displayName)
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
