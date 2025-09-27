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
		Host:     "10.27.40.1", // replace with your VAST IP
		Username: "admin",
		Password: "123456",
	}

	// Create typed client
	typedClient, err := typed.NewTypedVMSRest(config)
	if err != nil {
		log.Fatalf("Failed to create typed client: %v", err)
	}
	typedClient.SetCtx(ctx)

	// Use typed VipPool resource
	vipPoolClient := typedClient.VipPools

	// Example 1: List VIP pools with typed search parameters
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

	// Example 2: Create a new VIP pool with typed create body
	createBody := &typed.VipPoolCreateBody{
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

		// Example 3: Update the VIP pool
		updateBody := &typed.VipPoolCreateBody{
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

		// Example 4: Get VIP pool by ID
		retrievedPool, err := vipPoolClient.GetById(newVipPool.Id)
		if err != nil {
			log.Printf("Failed to get VIP pool by ID: %v", err)
		} else {
			fmt.Printf("Retrieved VIP pool: ID=%d, Name=%s\n",
				retrievedPool.Id, retrievedPool.Name)
		}

		// Example 5: Check if VIP pool exists
		exists, err := vipPoolClient.Exists(&typed.VipPoolSearchParams{
			Name: "typed-example-vippool-updated",
		})
		if err != nil {
			log.Printf("Failed to check VIP pool existence: %v", err)
		} else {
			fmt.Printf("VIP pool exists: %t\n", exists)
		}

		// Clean up: delete the created VIP pool
		deleteParams := &typed.VipPoolSearchParams{
			Name: "typed-example-vippool-updated",
		}
		if err := vipPoolClient.Delete(deleteParams); err != nil {
			log.Printf("Failed to delete VIP pool: %v", err)
		} else {
			fmt.Println("Successfully deleted the example VIP pool")
		}
	}
}
