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
	createParams := &typed.ProtectionPolicyRequestBody{
		Name:      "go-client-test-policy",
		CloneType: "LOCAL",
		Prefix:    "snap",
	}

	policy, err := rest.ProtectionPolicies.Create(createParams)
	if err != nil {
		panic(fmt.Errorf("failed to create protection policy: %w", err))
	}
	fmt.Printf("Protection policy created successfully: %s (ID: %d)\n", policy.Name, policy.Id)

	// --- LIST ---
	policies, err := rest.ProtectionPolicies.List(&typed.ProtectionPolicySearchParams{})
	if err != nil {
		panic(fmt.Errorf("failed to list protection policies: %w", err))
	}
	fmt.Printf("Found %d protection policy(s)\n", len(policies))

	// --- GET ---
	fetchedPolicy, err := rest.ProtectionPolicies.Get(&typed.ProtectionPolicySearchParams{
		Name: "go-client-test-policy",
	})
	if err != nil {
		panic(fmt.Errorf("failed to get protection policy: %w", err))
	}
	fmt.Printf("Fetched protection policy: %s (Clone Type: %s)\n", fetchedPolicy.Name, fetchedPolicy.CloneType)

	// --- DELETE ---
	searchParams := &typed.ProtectionPolicySearchParams{
		Name: "go-client-test-policy",
	}

	err = rest.ProtectionPolicies.Delete(searchParams)
	if err != nil {
		panic(fmt.Errorf("failed to delete protection policy: %w", err))
	}
	fmt.Println("Protection policy deleted successfully.")
}
