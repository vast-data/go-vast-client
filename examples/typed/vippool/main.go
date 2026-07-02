package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/resources/typed"
	"github.com/vast-data/go-vast-client/resources/typed/expr"
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

	// --- LIST all ---
	vippools, err := rest.VipPools.List(&typed.VipPoolSearchParams{})
	if err != nil {
		panic(fmt.Errorf("failed to list vippools: %w", err))
	}
	fmt.Printf("Found %d vippool(s)\n", len(vippools))
	for _, vp := range vippools {
		fmt.Printf("  - %s (ID: %d, Role: %s)\n", vp.Name, vp.Id, vp.Role)
	}

	// --- LIST with expression filter ---
	filtered, err := rest.VipPools.List(&typed.VipPoolSearchParams{
		Name: expr.Str.StartsWith("go-client"),
	})
	if err != nil {
		panic(fmt.Errorf("failed to list filtered vippools: %w", err))
	}
	fmt.Printf("VipPools starting with 'go-client': %d\n", len(filtered))

	// --- GET BY ID ---
	retrievedVipPool, err := rest.VipPools.GetById(vippool.Id)
	if err != nil {
		panic(fmt.Errorf("failed to get vippool by ID: %w", err))
	}
	fmt.Printf("Retrieved VipPool: %s\n", retrievedVipPool.Name)

	// --- UPDATE ---
	newIpRanges := [][]string{
		{"192.168.1.100", "192.168.1.120"},
	}

	updateParams := &typed.VipPoolRequestBody{
		IpRanges: &newIpRanges,
		Vlan:     200,
	}

	_, err = rest.VipPools.Update(vippool.Id, updateParams)
	if err != nil {
		panic(fmt.Errorf("failed to update vippool: %w", err))
	}
	fmt.Println("VipPool updated successfully.")

	// --- CHECK EXISTS ---
	exists, err := rest.VipPools.Exists(&typed.VipPoolSearchParams{
		StartIp: expr.S("192.168.1.100"),
	})
	if err != nil {
		panic(fmt.Errorf("failed to check vippool existence: %w", err))
	}
	fmt.Printf("VipPool with start IP 192.168.1.100 exists: %t\n", exists)

	// --- DELETE ---
	_, err = rest.VipPools.DeleteById(vippool.Id, 0) // 0 = fire-and-forget (no wait)
	if err != nil {
		panic(fmt.Errorf("failed to delete vippool: %w", err))
	}
	fmt.Println("VipPool deleted successfully.")
}
