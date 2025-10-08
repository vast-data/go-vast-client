package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/resources/typed"
)

func main() {
	var policy typed.ProtectionPolicyUpsertModel

	config := &client.VMSConfig{
		Host:     "l101", // replace with your VAST address
		Username: "admin",
		Password: "123456",
	}

	rest, err := client.NewVMSRest(config)
	if err != nil {
		panic(err)
	}

	// --- CREATE ---
	createParams := client.Params{
		"name":       "go-client-test-policy",
		"clone_type": "LOCAL",
		"prefix":     "snap",
	}
	result, err := rest.ProtectionPolicies.Create(createParams)
	if err != nil {
		panic(fmt.Errorf("failed to create protection policy: %w", err))
	}
	fmt.Println("Protection policy created successfully.")
	if err := result.Fill(&policy); err != nil {
		panic(fmt.Errorf("failed to fill ProtectionPolicyUpsertModel: %w", err))
	}

	// --- LIST ---
	policies, err := rest.ProtectionPolicies.List(client.Params{})
	if err != nil {
		panic(fmt.Errorf("failed to list protection policies: %w", err))
	}
	fmt.Printf("Found %d protection policy(s)\n", len(policies))

	// --- GET ---
	fetchedPolicy, err := rest.ProtectionPolicies.Get(client.Params{
		"name": "go-client-test-policy",
	})
	if err != nil {
		panic(fmt.Errorf("failed to get protection policy: %w", err))
	}
	fmt.Printf("Fetched protection policy: %v (Clone Type: %v)\n", fetchedPolicy["name"], fetchedPolicy["clone_type"])

	// --- DELETE ---
	_, err = rest.ProtectionPolicies.Delete(client.Params{
		"name": "go-client-test-policy",
	}, nil)
	if err != nil {
		panic(fmt.Errorf("failed to delete protection policy: %w", err))
	}
	fmt.Println("Protection policy deleted successfully.")
}
