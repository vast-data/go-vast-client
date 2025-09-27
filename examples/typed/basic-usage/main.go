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
		Host:     "l101", // replace with your VAST address
		Username: "admin",
		Password: "123456",
	}

	// Create typed client
	typedClient, err := typed.NewTypedVMSRest(config)
	if err != nil {
		log.Fatalf("Failed to create typed client: %v", err)
	}
	typedClient.SetCtx(ctx)

	fmt.Println("=== Typed VAST Client Basic Usage Demo ===")
	fmt.Println()

	// Example 1: Working with Quotas (Full CRUD resource)
	fmt.Println("1. Working with Quotas (Full CRUD resource):")
	quotaClient := typedClient.Quotas

	// List quotas
	quotas, err := quotaClient.List(&typed.QuotaSearchParams{
		TenantId: 1,
	})
	if err != nil {
		log.Printf("Failed to list quotas: %v", err)
	} else {
		fmt.Printf("   Found %d quotas\n", len(quotas))
	}

	// Example 2: Working with Versions (Read-only resource)
	fmt.Println("\n2. Working with Versions (Read-only resource):")
	versionClient := typedClient.Versions

	// List versions (read-only resource)
	versions, err := versionClient.List(nil)
	if err != nil {
		log.Printf("Failed to list versions: %v", err)
	} else {
		fmt.Printf("   Found %d versions\n", len(versions))
		if len(versions) > 0 {
			fmt.Printf("   Latest version: %s\n", versions[0].Name)
		}
	}

	// Note: Version is read-only, so no Create/Update/Delete methods are available
	// The compiler will prevent you from accidentally trying to modify read-only resources

	// Example 3: Working with Views (Full CRUD resource)
	fmt.Println("\n3. Working with Views (Full CRUD resource):")
	viewClient := typedClient.Views

	// List views
	views, err := viewClient.List(&typed.ViewSearchParams{
		TenantId: 1,
	})
	if err != nil {
		log.Printf("Failed to list views: %v", err)
	} else {
		fmt.Printf("   Found %d views\n", len(views))
	}

	// Example 4: Working with VIP Pools (Full CRUD resource)
	fmt.Println("\n4. Working with VIP Pools (Full CRUD resource):")
	vipPoolClient := typedClient.VipPools

	// List VIP pools
	vipPools, err := vipPoolClient.List(&typed.VipPoolSearchParams{
		TenantId: 1,
	})
	if err != nil {
		log.Printf("Failed to list VIP pools: %v", err)
	} else {
		fmt.Printf("   Found %d VIP pools\n", len(vipPools))
	}
}
