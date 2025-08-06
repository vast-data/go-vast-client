package main

import (
	"context"
	"fmt"
	client "github.com/vast-data/go-vast-client"
	"log"
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

	fmt.Println("=== USERS COPY EXAMPLE ===")

	// Example 1: Copy users by tenant ID
	fmt.Println("\n1. Copying users by tenant ID...")
	paramsByTenant := client.UsersCopyParams{
		DestinationProviderID: 2,             // ID of the destination local provider
		TenantID:              []int64{1}[0], // Tenant ID (using pointer)
	}

	err = rest.Users.CopyWithContext(ctx, paramsByTenant)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Users copied successfully by tenant ID\n")

	// Example 2: Copy specific users by their IDs
	fmt.Println("\n2. Copying specific users by their IDs...")
	userIDs := []int64{101, 102, 103} // Specific user IDs to copy
	paramsByUserIDs := client.UsersCopyParams{
		DestinationProviderID: 2, // ID of the destination local provider
		UserIDs:               userIDs,
	}

	err = rest.Users.CopyWithContext(ctx, paramsByUserIDs)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Specific users copied successfully\n")

	fmt.Println("\n=== EXAMPLE COMPLETED ===")
}
