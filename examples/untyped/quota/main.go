package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/resources/typed"
)

func main() {
	var quota typed.QuotaUpsertModel

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
	createParams := client.Params{
		"name":       "test-quota",
		"path":       "/go-client-test-quota",
		"create_dir": true,
		"hard_limit": 1073741824, // 1GB
	}
	result, err := rest.Quotas.Create(createParams)
	if err != nil {
		panic(fmt.Errorf("failed to create quota: %w", err))
	}
	fmt.Println("Quota created successfully.")
	if err := result.Fill(&quota); err != nil {
		panic(fmt.Errorf("failed to fill QuotaUpsertModel: %w", err))
	}

	// --- LIST ---
	quotas, err := rest.Quotas.List(client.Params{})
	if err != nil {
		panic(fmt.Errorf("failed to list quotas: %w", err))
	}
	fmt.Printf("Found %d quota(s)\n", len(quotas))

	// --- GET ---
	fetchedQuota, err := rest.Quotas.Get(client.Params{
		"path": "/go-client-test-quota",
	})
	if err != nil {
		panic(fmt.Errorf("failed to get quota: %w", err))
	}
	fmt.Printf("Fetched quota: %v (Hard Limit: %v)\n", fetchedQuota["path"], fetchedQuota["hard_limit"])

	// --- DELETE ---
	_, err = rest.Quotas.Delete(client.Params{
		"path": "/go-client-test-quota",
	}, nil)
	if err != nil {
		panic(fmt.Errorf("failed to delete quota: %w", err))
	}
	fmt.Println("Quota deleted successfully.")
}
