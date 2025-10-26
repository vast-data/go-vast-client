package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	log "vastix/internal/logging"

	vastclient "github.com/vast-data/go-vast-client"
	"go.uber.org/zap"
)

const recordTypeKey = "@resourceType"

// BeforeRequestFnCallback logs the HTTP request being sent.
// It reads and optionally compacts the body (if present) for structured logging.
// For more details see: https://github.com/vast-data/go-vast-client
func BeforeRequestFnCallback(_ context.Context, _ *http.Request, verb, url string, body io.Reader) error {
	requestInfo := fmt.Sprintf("HTTP request start: [%s] %s", verb, url)
	var bodyMsg string

	if body != nil {
		bodyBytes, err := io.ReadAll(body)
		if err != nil {
			log.Error("failed to read request body", zap.Error(err))
			return err
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
		log.Info(requestInfo)
	} else {
		log.Info(requestInfo, zap.Any("body", json.RawMessage(bodyMsg)))
	}
	return nil
}

// AfterRequestFnCallback logs the response received from the HTTP request.
// For single records, it shows the @resourceType if available, otherwise just "Record received".
// For record sets, it shows the count and @resourceType from the first record if available.
// For more details see: https://github.com/vast-data/go-vast-client
func AfterRequestFnCallback(_ context.Context, response vastclient.Renderable) (vastclient.Renderable, error) {
	var responseStr string
	switch resp := response.(type) {
	case vastclient.Record:
		// For single records, try to extract @resourceType for concise logging
		if resourceType, ok := resp[recordTypeKey].(string); ok && resourceType != "" {
			responseStr = fmt.Sprintf("Record of type: %s", resourceType)
		} else {
			responseStr = "Record received"
		}
	case vastclient.RecordSet:
		// For record sets, show count and resource type from first record if available
		count := len(resp)
		if count > 0 {
			// Try to extract @resourceType from the first record
			firstRecord := resp[0]
			if resourceType, ok := firstRecord[recordTypeKey].(string); ok && resourceType != "" {
				responseStr = fmt.Sprintf("RecordSet with %d record(s) of type: %s", count, resourceType)
			} else {
				responseStr = fmt.Sprintf("RecordSet with %d record(s)", count)
			}
		} else {
			responseStr = "RecordSet with 0 record(s)"
		}
	default:
		// Fallback - just indicate response received
		responseStr = "Response received"
	}

	log.Info("response", zap.String("response", responseStr))
	return response, nil
}
