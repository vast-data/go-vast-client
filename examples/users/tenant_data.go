package main

import (
	"context"
	"fmt"
	"log"

	client "github.com/vast-data/go-vast-client"
)

func main() {
	config := &client.VMSConfig{
		Host:     "10.27.40.1", // replace with your VAST address
		Username: "admin",
		Password: "123456",
	}

	rest, err := client.NewVMSRest(config)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	userId := int64(123) // Replace with actual user ID

	fmt.Println("=== USER TENANT DATA EXAMPLE ===")

	// Example 1: Get current tenant data for a user
	fmt.Println("\n1. Getting current tenant data...")
	tenantData, err := rest.Users.GetTenantDataWithContext(ctx, userId)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Current tenant data retrieved successfully\n")
	fmt.Printf("  Result: %s\n", tenantData.PrettyTable())

	// Example 2: Update tenant data for a user
	fmt.Println("\n2. Updating tenant data...")
	updateParams := client.Params{
		"allow_create_bucket": true,
		"allow_delete_bucket": false,
		"s3_superuser":        true,
		"s3_policies_ids":     []int64{1, 2, 3}, // S3 policy IDs to attach
	}

	updatedData, err := rest.Users.UpdateTenantDataWithContext(ctx, userId, updateParams)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Tenant data updated successfully\n")
	fmt.Printf("  Result: %s\n", updatedData.PrettyTable())

	// Example 3: Update only specific fields (partial update)
	fmt.Println("\n3. Partial update - only changing S3 superuser permission...")
	partialUpdateParams := client.Params{
		"s3_superuser": false,
	}

	finalData, err := rest.Users.UpdateTenantData(userId, partialUpdateParams)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Partial update completed successfully\n")
	fmt.Printf("  Result: %s\n", finalData.PrettyTable())

	fmt.Println("\n=== EXAMPLE COMPLETED ===")
} 