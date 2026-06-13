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
	createParams := &typed.TenantRequestBody{
		Name: "go-client-test-tenant",
		Qos: typed.TenantRequestBody_Qos{
			StaticLimits: typed.TenantRequestBody_Qos_StaticLimits{
				MaxReadsBwMbps:  1000,
				MaxReadsIops:    10000,
				MaxWritesBwMbps: 1000,
				MaxWritesIops:   10000,
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
		Name: expr.S("go-client-test-tenant"),
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
	err = rest.Tenants.Delete(&typed.TenantSearchParams{
		Name: expr.S("go-client-test-tenant"),
	}, false)
	if err != nil {
		panic(fmt.Errorf("failed to delete tenant: %w", err))
	}
	fmt.Println("Tenant deleted successfully.")
}
