package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/typed"
)

type contextKey string

const requestIDKey contextKey = "@request_id"

// BeforeRequestFnCallback logs the HTTP request being sent.
// It reads and optionally compacts the body (if present) for structured logging,
// and includes the request ID from context (if available).
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
func AfterRequestFnCallback(ctx context.Context, response client.Renderable) (client.Renderable, error) {
	uid, _ := ctx.Value(requestIDKey).(string)
	log.Printf("✓ END: req_id=%s\n%s", uid, response.PrettyTable())
	return response, nil
}

// ContextWithRequestID returns a new context containing a hardcoded request ID.
// Simplified version for demo purposes.
func ContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func main() {

	config := &client.VMSConfig{
		Host:     "l101", // replace with your VAST IP
		Username: "admin",
		Password: "123456",
		// Use the improved interceptors with request ID tracking
		BeforeRequestFn: BeforeRequestFnCallback,
		AfterRequestFn:  AfterRequestFnCallback,
	}

	// Create typed client
	typedClient, err := typed.NewTypedVMSRest(config)
	if err != nil {
		log.Fatalf("Failed to create typed client: %v", err)
	}

	// Use typed VipPool resource
	vipPoolClient := typedClient.VipPools

	fmt.Println("\n1. Listing VIP pools with request ID tracking:")
	// Example 1: List VIP pools with typed search parameters and request ID
	ctx1 := ContextWithRequestID(context.Background(), "REQ-001")
	typedClient.SetCtx(ctx1)

	searchParams := &typed.VipPoolSearchParams{
		TenantId: 1,
	}

	vipPools, err := vipPoolClient.List(searchParams)
	if err != nil {
		log.Printf("Failed to list VIP pools: %v", err)
	} else {
		fmt.Printf("Found %d VIP pools\n", len(vipPools))
		for _, pool := range vipPools {
			if pool.Name != "" && pool.Id != 0 {
				fmt.Printf("VIP Pool: ID=%d, Name=%s, StartIp=%s, EndIp=%s\n",
					pool.Id, pool.Name, pool.StartIp, pool.EndIp)
			}
		}
	}

	fmt.Println("\n2. Creating VIP pool with request ID tracking:")
	// Example 2: Create a new VIP pool with typed create body and new request ID
	ctx2 := ContextWithRequestID(context.Background(), "REQ-002")
	typedClient.SetCtx(ctx2)

	createBody := &typed.VipPoolRequestBody{
		Name:       "typed-example-vippool",
		StartIp:    "20.0.0.1",
		EndIp:      "20.0.0.16",
		SubnetCidr: 24,
	}

	newVipPool, err := vipPoolClient.Create(createBody)
	if err != nil {
		log.Printf("Failed to create VIP pool: %v", err)
	} else {
		fmt.Printf("Created VIP pool: ID=%d, Name=%s, StartIp=%s, EndIp=%s\n",
			newVipPool.Id, newVipPool.Name, newVipPool.StartIp, newVipPool.EndIp)

		fmt.Println("\n3. Updating VIP pool with request ID tracking:")
		// Example 3: Update the VIP pool with new request ID
		ctx3 := ContextWithRequestID(context.Background(), "REQ-003")
		typedClient.SetCtx(ctx3)

		updateBody := &typed.VipPoolRequestBody{
			Name:       "typed-example-vippool-updated",
			StartIp:    "20.0.0.1",
			EndIp:      "20.0.0.32", // Expand the range
			SubnetCidr: 22,          // Change subnet
		}

		updatedVipPool, err := vipPoolClient.Update(newVipPool.Id, updateBody)
		if err != nil {
			log.Printf("Failed to update VIP pool: %v", err)
		} else {
			fmt.Printf("Updated VIP pool: ID=%d, Name=%s, EndIp=%s, SubnetCidr=%d\n",
				updatedVipPool.Id, updatedVipPool.Name, updatedVipPool.EndIp, updatedVipPool.SubnetCidr)
		}

		fmt.Println("\n4. Getting VIP pool by ID with request ID tracking:")
		// Example 4: Get VIP pool by ID with new request ID
		ctx4 := ContextWithRequestID(context.Background(), "REQ-004")
		typedClient.SetCtx(ctx4)

		retrievedPool, err := vipPoolClient.GetById(newVipPool.Id)
		if err != nil {
			log.Printf("Failed to get VIP pool by ID: %v", err)
		} else {
			fmt.Printf("Retrieved VIP pool: ID=%d, Name=%s\n",
				retrievedPool.Id, retrievedPool.Name)
		}

		fmt.Println("\n5. Checking VIP pool existence with request ID tracking:")
		// Example 5: Check if VIP pool exists with new request ID
		ctx5 := ContextWithRequestID(context.Background(), "REQ-005")
		typedClient.SetCtx(ctx5)

		exists, err := vipPoolClient.Exists(&typed.VipPoolSearchParams{
			Name: "typed-example-vippool-updated",
		})
		if err != nil {
			log.Printf("Failed to check VIP pool existence: %v", err)
		} else {
			fmt.Printf("VIP pool exists: %t\n", exists)
		}

		fmt.Println("\n6. Deleting VIP pool with request ID tracking:")
		// Clean up: delete the created VIP pool with new request ID
		ctx6 := ContextWithRequestID(context.Background(), "REQ-006")
		typedClient.SetCtx(ctx6)

		deleteParams := &typed.VipPoolSearchParams{
			Name: "typed-example-vippool-updated",
		}
		if err := vipPoolClient.Delete(deleteParams); err != nil {
			log.Printf("Failed to delete VIP pool: %v", err)
		} else {
			fmt.Println("Successfully deleted the example VIP pool")
		}
	}

	fmt.Println("\n=== Request ID Tracking Benefits Demonstrated ===")
	fmt.Println("✓ Each operation has a unique request ID for correlation")
	fmt.Println("✓ Smart body logging (skips null/empty bodies)")
	fmt.Println("✓ JSON compacting for readable request logs")
	fmt.Println("✓ Before/After request correlation with same ID")
	fmt.Println("✓ Simple hardcoded IDs for easy demo understanding")
}
