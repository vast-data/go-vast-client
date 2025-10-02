package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/resources/typed"
)

func main() {
	var tenant typed.TenantUpsertModel

	config := &client.VMSConfig{
		Host:     "l101", // replace with your VAST address
		Username: "admin",
		Password: "123456",
	}

	rest, err := client.NewUntypedVMSRest(config)
	if err != nil {
		panic(err)
	}

	// --- CREATE ---
	// Note: VAST requires at least one QoS static limit to be set
	createParams := client.Params{
		"name": "go-client-test-tenant",
		"qos": map[string]any{
			"static_limits": map[string]any{
				"max_reads_bw_mbps":  1000,  // 1 GB/s max read bandwidth
				"max_reads_iops":     10000, // 10k IOPS max reads
				"max_writes_bw_mbps": 1000,  // 1 GB/s max write bandwidth
				"max_writes_iops":    10000, // 10k IOPS max writes
			},
		},
	}
	result, err := rest.Tenants.Create(createParams)
	if err != nil {
		panic(fmt.Errorf("failed to create tenant: %w", err))
	}
	fmt.Println("Tenant created successfully.")
	if err := result.Fill(&tenant); err != nil {
		panic(fmt.Errorf("failed to fill TenantUpsertModel: %w", err))
	}

	// --- LIST ---
	tenants, err := rest.Tenants.List(client.Params{})
	if err != nil {
		panic(fmt.Errorf("failed to list tenants: %w", err))
	}
	fmt.Printf("Found %d tenant(s)\n", len(tenants))

	// --- GET ---
	fetchedTenant, err := rest.Tenants.Get(client.Params{
		"name": "go-client-test-tenant",
	})
	if err != nil {
		panic(fmt.Errorf("failed to get tenant: %w", err))
	}
	fmt.Printf("Fetched tenant: %v (GUID: %v)\n", fetchedTenant["name"], fetchedTenant["guid"])

	// --- UPDATE ---
	updateParams := client.Params{
		"access_ip_ranges": []string{"10.0.0.0/8"},
	}
	_, err = rest.Tenants.Update(tenant.Id, updateParams)
	if err != nil {
		panic(fmt.Errorf("failed to update tenant: %w", err))
	}
	fmt.Println("Tenant updated successfully.")

	// --- DELETE ---
	_, err = rest.Tenants.Delete(client.Params{
		"name":         "go-client-test-tenant",
		"force_remove": false,
	}, nil)
	if err != nil {
		panic(fmt.Errorf("failed to delete tenant: %w", err))
	}
	fmt.Println("Tenant deleted successfully.")
}
