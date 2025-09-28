package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"sync/atomic"

	vast_client "github.com/vast-data/go-vast-client"
)

type contextKey string

const requestIDKey contextKey = "@request_id"

// reqIDCounter holds a globally shared atomic counter used to generate unique request IDs.
// It is initialized with a random value in the lower half of uint32 range to avoid predictable sequences.
var reqIDCounter = rand.Uint32() % (uint32(math.MaxUint32/2) + 1) // result in [0, max]

// BeforeRequestFnCallback logs the HTTP request being sent.
// It reads and optionally compacts the body (if present) for structured logging,
// and includes the request ID from context (if available).
// For more details see: https://github.com/vast-data/go-vast-client
func BeforeRequestFnCallback(ctx context.Context, _ *http.Request, verb, url string, body io.Reader) error {
	var logMsg strings.Builder
	uid, _ := ctx.Value(requestIDKey).(string)
	logMsg.WriteString(fmt.Sprintf("➤ START: req_id=%s - [%s] %s", uid, verb, url))

	if body != nil {
		bodyBytes, err := io.ReadAll(body)
		if err != nil {
			log.Printf("ERROR: failed to read request body: %v", err)
			return err
		}

		trimmed := bytes.TrimSpace(bodyBytes)
		if len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null")) {
			var compact bytes.Buffer
			if err := json.Compact(&compact, trimmed); err == nil {
				logMsg.WriteString(fmt.Sprintf(" - body: %s", compact.String()))
			} else {
				logMsg.WriteString(fmt.Sprintf(" - body (raw): %s", string(trimmed)))
			}
		}
	}

	log.Println(logMsg.String())
	return nil
}

// AfterRequestFnCallback logs the response received from the HTTP request.
// It uses the response's PrettyTable method to render a formatted table,
// and includes the request ID from context.
// For more details see: https://github.com/vast-data/go-vast-client
func AfterRequestFnCallback(ctx context.Context, response vast_client.Renderable) (vast_client.Renderable, error) {
	uid, _ := ctx.Value(requestIDKey).(string)
	log.Printf("✓ END: req_id=%s\n%s", uid, response.PrettyTable())
	return response, nil
}

// ContextWithRequestID returns a new context containing a generated request ID.
// The ID is a hex string based on a global atomic counter to ensure uniqueness
// across concurrent requests within the same process.
func ContextWithRequestID(ctx context.Context) context.Context {
	newID := atomic.AddUint32(&reqIDCounter, 1)
	return context.WithValue(ctx, requestIDKey, fmt.Sprintf("0x%08x", newID))
}

func main() {
	fmt.Println("=== VAST Client Request Interceptors with Request ID Demo ===")

	// Create client configuration with interceptors
	config := &vast_client.VMSConfig{
		Host:     "vms.example.com",
		Username: "admin",
		Password: "password",
		// Configure interceptors for request/response logging with request ID tracking
		BeforeRequestFn: BeforeRequestFnCallback,
		AfterRequestFn:  AfterRequestFnCallback,
	}

	// Create client
	client, err := vast_client.NewVMSRest(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	fmt.Println("\n1. Demonstrating GET request with request ID tracking:")

	// Create context with request ID for the first request
	ctx1 := ContextWithRequestID(context.Background())

	// Make a GET request - this will trigger the interceptors
	versions, err := client.Versions.ListWithContext(ctx1, vast_client.Params{})
	if err != nil {
		log.Printf("Error getting versions: %v", err)
	} else {
		fmt.Printf("Retrieved %d versions\n", len(versions))
	}

	fmt.Println("\n2. Demonstrating POST request with request ID tracking:")

	// Create context with request ID for the second request
	ctx2 := ContextWithRequestID(context.Background())

	// Make a POST request with body - this will show body logging
	quotaParams := vast_client.Params{
		"name":         "demo-quota",
		"hard_limit":   "1TB",
		"soft_limit":   "800GB",
		"grace_period": "7d",
		"path":         "/demo",
		"create_dir":   true,
	}

	quota, err := client.Quotas.CreateWithContext(ctx2, quotaParams)
	if err != nil {
		log.Printf("Error creating quota (expected if quota exists): %v", err)
	} else {
		fmt.Printf("Created quota: %v\n", quota["name"])
	}

	fmt.Println("\n3. Demonstrating multiple concurrent requests with unique IDs:")

	// Make multiple concurrent requests to show unique request IDs
	for i := 0; i < 3; i++ {
		go func(requestNum int) {
			ctx := ContextWithRequestID(context.Background())
			log.Printf("Starting concurrent request #%d", requestNum+1)

			_, err := client.Versions.ListWithContext(ctx, vast_client.Params{"page": 1})
			if err != nil {
				log.Printf("Concurrent request #%d failed: %v", requestNum+1, err)
			}
		}(i)
	}

	fmt.Println("\n=== Key Features Demonstrated ===")
	fmt.Println("✓ Unique request ID generation using atomic counter")
	fmt.Println("✓ Context-based request ID propagation")
	fmt.Println("✓ Smart body logging (skips 'null' and empty bodies)")
	fmt.Println("✓ JSON compacting for readable logs")
	fmt.Println("✓ Before/After request logging with timing correlation")
	fmt.Println("✓ Concurrent request tracking with unique IDs")

	fmt.Println("\n=== Usage Pattern ===")
	fmt.Println("1. Create context with request ID: ctx := ContextWithRequestID(context.Background())")
	fmt.Println("2. Use context in API calls: client.Resource.MethodWithContext(ctx, params)")
	fmt.Println("3. Interceptors automatically log with request ID for correlation")
}
