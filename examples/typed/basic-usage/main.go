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
		Host:     "10.27.40.1", // replace with your VAST address
		Username: "admin",
		Password: "123456",
	}

	// Create typed client
	typedClient, err := typed.NewTypedVMSRest(config)
	if err != nil {
		log.Fatalf("Failed to create typed client: %v", err)
	}
	typedClient.Untyped.SetCtx(ctx)

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
	versions, err := versionClient.List(&typed.VersionSearchParams{})
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

	// Example 5: Accessing untyped client when needed
	fmt.Println("\n5. Accessing untyped client when needed:")
	fmt.Println("   You can always access the underlying untyped client:")
	fmt.Printf("   Untyped client available at: typedClient.Untyped\n")
	fmt.Printf("   This gives you access to all resources, even those without typed support\n")

	// Example: Use untyped client for resources without typed support
	users, err := typedClient.Untyped.Users.List(client.Params{})
	if err != nil {
		log.Printf("Failed to list users via untyped client: %v", err)
	} else {
		fmt.Printf("   Found %d users via untyped client\n", len(users.Data))
	}

	fmt.Println("\n=== Demo completed successfully! ===")
	fmt.Println("\nKey benefits of typed resources:")
	fmt.Println("• Type safety: Compile-time checking of request/response structures")
	fmt.Println("• IntelliSense: Better IDE support with auto-completion")
	fmt.Println("• Documentation: Self-documenting code with typed structs")
	fmt.Println("• Read-only protection: Compiler prevents modification of read-only resources")
	fmt.Println("• Validation: Automatic validation of required fields")
}
