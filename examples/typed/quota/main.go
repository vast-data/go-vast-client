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
	createParams := &typed.QuotaRequestBody{
		Name:      "test-quota",
		Path:      "/go-client-test-quota",
		CreateDir: true,
		HardLimit: 1073741824, // 1GB
	}

	quota, err := rest.Quotas.Create(createParams)
	if err != nil {
		panic(fmt.Errorf("failed to create quota: %w", err))
	}
	fmt.Printf("Quota created successfully: %s (ID: %d)\n", quota.Path, quota.Id)

	// --- LIST ---
	quotas, err := rest.Quotas.List(&typed.QuotaSearchParams{})
	if err != nil {
		panic(fmt.Errorf("failed to list quotas: %w", err))
	}
	fmt.Printf("Found %d quota(s)\n", len(quotas))

	// --- GET ---
	fetchedQuota, err := rest.Quotas.Get(&typed.QuotaSearchParams{
		Path: "/go-client-test-quota",
	})
	if err != nil {
		panic(fmt.Errorf("failed to get quota: %w", err))
	}
	fmt.Printf("Fetched quota: %s (Hard Limit: %d)\n", fetchedQuota.Path, fetchedQuota.HardLimit)

	// --- DELETE ---
	searchParams := &typed.QuotaSearchParams{
		Path: "/go-client-test-quota",
	}

	err = rest.Quotas.Delete(searchParams)
	if err != nil {
		panic(fmt.Errorf("failed to delete quota: %w", err))
	}
	fmt.Println("Quota deleted successfully.")
}
