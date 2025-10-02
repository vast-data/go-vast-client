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
	// Note: VAST requires at least one QoS static limit to be set
	createParams := &typed.TenantRequestBody{
		Name: "go-client-test-tenant",
		Qos: typed.TenantRequestBody_Qos{
			StaticLimits: typed.TenantRequestBody_Qos_StaticLimits{
				MaxReadsBwMbps:  1000,  // 1 GB/s max read bandwidth
				MaxReadsIops:    10000, // 10k IOPS max reads
				MaxWritesBwMbps: 1000,  // 1 GB/s max write bandwidth
				MaxWritesIops:   10000, // 10k IOPS max writes
			},
		},
	}

	tenant, err := rest.Tenants.Create(createParams)
	if err != nil {
		panic(fmt.Errorf("failed to create tenant: %w", err))
	}
	fmt.Printf("Tenant created successfully: %s (ID: %d)\n", tenant.Name, tenant.Id)

	// --- LIST ---
	tenants, err := rest.Tenants.List(nil)
	if err != nil {
		panic(fmt.Errorf("failed to list tenants: %w", err))
	}
	fmt.Printf("Found %d tenant(s)\n", len(tenants))

	// --- GET ---
	fetchedTenant, err := rest.Tenants.Get(&typed.TenantSearchParams{
		Name: "go-client-test-tenant",
	})
	if err != nil {
		panic(fmt.Errorf("failed to get tenant: %w", err))
	}
	fmt.Printf("Fetched tenant: %s (GUID: %s)\n", fetchedTenant.Name, fetchedTenant.Guid)

	// --- UPDATE ---
	accessRanges := []string{"10.0.0.0/8"}
	updateParams := &typed.TenantRequestBody{
		AccessIpRanges: &accessRanges,
	}

	_, err = rest.Tenants.Update(tenant.Id, updateParams)
	if err != nil {
		panic(fmt.Errorf("failed to update tenant: %w", err))
	}
	fmt.Println("Tenant updated successfully.")

	// --- DELETE ---
	searchParams := &typed.TenantSearchParams{
		Name: "go-client-test-tenant",
	}

	err = rest.Tenants.Delete(searchParams, false) // forceRemove=false
	if err != nil {
		panic(fmt.Errorf("failed to delete tenant: %w", err))
	}
	fmt.Println("Tenant deleted successfully.")
}
