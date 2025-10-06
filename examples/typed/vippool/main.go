package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/resources/typed"
)

func main() {
	config := &client.VMSConfig{
		Host:     "l101", // replace with your VAST address
		Username: "admin",
		Password: "123456",
	}

	rest, err := client.NewTypedVMSRest(config)
	if err != nil {
		panic(err)
	}

	// --- CREATE ---
	ipRanges := [][]string{
		{"192.168.1.100", "192.168.1.110"},
		{"192.168.1.200", "192.168.1.210"},
	}

	createParams := &typed.VipPoolRequestBody{
		Name:           "go-client-testvippool",
		SubnetCidr:     24,
		IpRanges:       &ipRanges,
		Role:           "PROTOCOLS",
		PortMembership: "ALL",
		Vlan:           100,
	}

	vippool, err := rest.VipPools.Create(createParams)
	if err != nil {
		panic(fmt.Errorf("failed to create vippool: %w", err))
	}
	fmt.Printf("VipPool created successfully: %s (ID: %d)\n", vippool.Name, vippool.Id)

	// --- LIST ---
	fmt.Println("\n--- Listing VipPools ---")
	searchParams := &typed.VipPoolSearchParams{
		// You can filter by various fields
		// StartIp: "192.168.1.100",
		// PortMembership: 1,
	}

	vippools, err := rest.VipPools.List(searchParams)
	if err != nil {
		panic(fmt.Errorf("failed to list vippools: %w", err))
	}
	fmt.Printf("Found %d vippools\n", len(vippools))
	for _, vp := range vippools {
		fmt.Printf("  - %s (ID: %d, Role: %s)\n", vp.Name, vp.Id, vp.Role)
	}

	// --- GET BY ID ---
	fmt.Println("\n--- Getting VipPool by ID ---")
	retrievedVipPool, err := rest.VipPools.GetById(vippool.Id)
	if err != nil {
		panic(fmt.Errorf("failed to get vippool by ID: %w", err))
	}
	fmt.Printf("Retrieved VipPool: %s\n", retrievedVipPool.Name)

	// --- UPDATE ---
	fmt.Println("\n--- Updating VipPool ---")
	newIpRanges := [][]string{
		{"192.168.1.100", "192.168.1.120"}, // Extended range
	}

	updateParams := &typed.VipPoolRequestBody{
		IpRanges: &newIpRanges,
		Vlan:     200, // Change VLAN
	}

	_, err = rest.VipPools.Update(vippool.Id, updateParams)
	if err != nil {
		panic(fmt.Errorf("failed to update vippool: %w", err))
	}
	fmt.Println("VipPool updated successfully.")

	// --- CHECK IF EXISTS ---
	fmt.Println("\n--- Checking if VipPool exists ---")
	exists, err := rest.VipPools.Exists(&typed.VipPoolSearchParams{
		StartIp: "192.168.1.100",
	})
	if err != nil {
		panic(fmt.Errorf("failed to check vippool existence: %w", err))
	}
	fmt.Printf("VipPool with start IP 192.168.1.100 exists: %t\n", exists)

	// --- DELETE ---
	fmt.Println("\n--- Deleting VipPool ---")
	deleteParams := &typed.VipPoolSearchParams{
		// Delete by name
		// Note: You can also use rest.VipPools.DeleteById(vippool.Id)
	}

	err = rest.VipPools.Delete(deleteParams)
	if err != nil {
		// Try by ID if search deletion fails
		err = rest.VipPools.DeleteById(vippool.Id)
		if err != nil {
			panic(fmt.Errorf("failed to delete vippool: %w", err))
		}
	}
	fmt.Println("VipPool deleted successfully.")
}
