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

	// Create typed client directly from config
	typedClient, err := typed.NewTypedVMSRest(config)
	if err != nil {
		log.Fatalf("Failed to create typed client: %v", err)
	}
	typedClient.Untyped.SetCtx(ctx)

	// Use typed quota resource
	quotaClient := typedClient.Quotas

	// Example 1: List quotas with typed search parameters
	searchParams := &typed.QuotaSearchParams{
		TenantId: 1,
		// Note: PageSize is excluded from search params as it's a common pagination parameter
	}

	quotas, err := quotaClient.List(searchParams)
	if err != nil {
		log.Fatalf("Failed to list quotas: %v", err)
	}

	fmt.Printf("Found %d quotas\n", len(quotas))
	for _, quota := range quotas {
		if quota.Name != "" && quota.Id != 0 {
			fmt.Printf("Quota: ID=%d, Name=%s\n", quota.Id, quota.Name)
		}
	}

	// Example 2: Get a specific quota by ID (if any exist)
	if len(quotas) > 0 && quotas[0].Id != 0 {
		quota, err := quotaClient.GetById(quotas[0].Id)
		if err != nil {
			log.Printf("Failed to get quota: %v", err)
		} else {
			fmt.Printf("Retrieved quota: %+v\n", quota)
		}
	}

	// Example 3: Create a new quota with typed request body
	requestBody := &typed.QuotaRequestBody{
		Name:      "typed-example-quota",
		Path:      "/example",
		TenantId:  1,
		HardLimit: 1024 * 1024 * 1024, // 1GB
	}

	newQuota, err := quotaClient.Create(requestBody)
	if err != nil {
		log.Printf("Failed to create quota: %v", err)
	} else {
		fmt.Printf("Created quota: ID=%d, Name=%s\n", newQuota.Id, newQuota.Name)

		// Clean up: delete the created quota
		if err := quotaClient.DeleteById(newQuota.Id); err != nil {
			log.Printf("Failed to delete quota: %v", err)
		} else {
			fmt.Println("Successfully deleted the example quota")
		}
	}
}
