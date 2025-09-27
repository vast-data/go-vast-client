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
		Host:     "l101",
		Username: "admin",
		Password: "123456",
	}

	// Create typed client directly from config
	typedClient, err := typed.NewTypedVMSRest(config)
	if err != nil {
		log.Fatalf("Failed to create typed client: %v", err)
	}
	typedClient.SetCtx(ctx)

	// Use typed quota resource
	quotaClient := typedClient.Quotas

	// Example 1: List quotas with typed search parameters
	searchParams := &typed.QuotaSearchParams{
		TenantId: 1,
		// Note: Common pagination parameters like page, page_size are excluded from search params
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

	// Example 3: Create a new quota with typed create body
	createBody := &typed.QuotaCreateBody{
		Name:      "typed-example-quota",
		Path:      "/example",
		TenantId:  1,
		HardLimit: 1024 * 1024 * 1024, // 1GB
	}

	newQuota, err := quotaClient.Create(createBody)
	if err != nil {
		log.Printf("Failed to create quota: %v", err)
	} else {
		fmt.Printf("Created quota: ID=%d, Name=%s\n", newQuota.Id, newQuota.Name)

		// Example 4: Update the quota
		updateBody := &typed.QuotaCreateBody{
			Name:      "typed-example-quota-updated",
			Path:      "/example",
			TenantId:  1,
			HardLimit: 2 * 1024 * 1024 * 1024, // 2GB
			CreateDir: true,
		}

		updatedQuota, err := quotaClient.Update(newQuota.Id, updateBody)
		if err != nil {
			log.Printf("Failed to update quota: %v", err)
		} else {
			fmt.Printf("Updated quota: ID=%d, Name=%s, HardLimit=%d\n",
				updatedQuota.Id, updatedQuota.Name, updatedQuota.HardLimit)
		}

		// Example 5: Check if quota exists
		exists, err := quotaClient.Exists(&typed.QuotaSearchParams{
			Name: "typed-example-quota-updated",
		})
		if err != nil {
			log.Printf("Failed to check quota existence: %v", err)
		} else {
			fmt.Printf("Quota exists: %t\n", exists)
		}

		// Clean up: delete the created quota
		deleteParams := &typed.QuotaSearchParams{
			Name: "typed-example-quota-updated",
		}
		if err := quotaClient.Delete(deleteParams); err != nil {
			log.Printf("Failed to delete quota: %v", err)
		} else {
			fmt.Println("Successfully deleted the example quota")
		}
	}
}
