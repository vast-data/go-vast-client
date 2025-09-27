package main

import (
	"context"
	"fmt"
	"log"

	client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/typed"
)

func main() {
	ctx := context.Background()
	config := &client.VMSConfig{
		Host:     "10.27.40.1",
		Username: "admin",
		Password: "123456",
	}

	fmt.Println("=== Typed VAST Client Demo ===")

	// Method 1: Create typed client directly from config
	fmt.Println("\n1. Creating typed client from config...")
	typedClient, err := typed.NewTypedVMSRest(config)
	if err != nil {
		log.Fatalf("Failed to create typed client: %v", err)
	}
	typedClient.SetCtx(ctx)

	fmt.Printf("✓ Typed client created successfully\n")
	fmt.Printf("✓ Untyped client available at: typedClient.Untyped\n")
	fmt.Printf("✓ Typed resources available: Quotas\n")

	// Method 2: Wrap existing raw client
	fmt.Println("\n2. Alternative: Wrapping existing raw client...")
	rawClient, err := client.NewVMSRest(config)
	if err != nil {
		log.Fatalf("Failed to create raw client: %v", err)
	}
	rawClient.SetCtx(ctx)

	wrappedClient := typed.NewTypedVMSRest(rawClient)
	fmt.Printf("✓ Raw client wrapped successfully\n")
	fmt.Printf("✓ Wrapped client also has Quotas: %v\n", wrappedClient.Quotas != nil)

	// Demonstrate typed operations
	fmt.Println("\n3. Demonstrating typed operations...")

	// List quotas with typed request
	listRequest := &typed.QuotaRequest{
		PageSize: stringPtr("5"),
		TenantID: int64Ptr(1),
	}

	fmt.Printf("Making typed List request with: PageSize=%s, TenantID=%d\n",
		*listRequest.PageSize, *listRequest.TenantID)

	quotas, err := typedClient.Quotas.List(ctx, listRequest)
	if err != nil {
		log.Printf("Note: List operation failed (expected if no VAST cluster): %v", err)
	} else {
		fmt.Printf("✓ Retrieved %d quotas with typed response\n", len(quotas))
		for i, quota := range quotas {
			if i >= 3 { // Show only first 3
				break
			}
			if quota.Name != nil && quota.ID != nil {
				fmt.Printf("  - Quota %d: ID=%d, Name=%s\n", i+1, *quota.ID, *quota.Name)
			}
		}
	}

	// Demonstrate access to untyped client when needed
	fmt.Println("\n4. Accessing untyped client when needed...")
	fmt.Printf("Untyped client version info available at: typedClient.Untyped.Versions\n")
	fmt.Printf("Untyped client session available at: typedClient.Untyped.Session\n")

	// Show type safety benefits
	fmt.Println("\n5. Type safety demonstration...")
	fmt.Println("✓ Compile-time validation of field names")
	fmt.Println("✓ IDE autocomplete for all request/response fields")
	fmt.Println("✓ No more string-based parameter keys")
	fmt.Println("✓ Clear method signatures with context support")

	fmt.Println("\n=== Demo completed successfully! ===")
}

// Helper functions for pointer conversion
func stringPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}
