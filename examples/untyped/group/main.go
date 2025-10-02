package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/resources/typed"
)

func main() {
	var group typed.GroupUpsertModel

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
		"name": "go-client-test-group",
		"gid":  5000,
	}
	result, err := rest.Groups.Create(createParams)
	if err != nil {
		panic(fmt.Errorf("failed to create group: %w", err))
	}
	fmt.Println("Group created successfully.")
	if err := result.Fill(&group); err != nil {
		panic(fmt.Errorf("failed to fill GroupUpsertModel: %w", err))
	}

	// --- LIST ---
	groups, err := rest.Groups.List(client.Params{})
	if err != nil {
		panic(fmt.Errorf("failed to list groups: %w", err))
	}
	fmt.Printf("Found %d group(s)\n", len(groups))

	// --- GET ---
	fetchedGroup, err := rest.Groups.Get(client.Params{
		"name": "go-client-test-group",
	})
	if err != nil {
		panic(fmt.Errorf("failed to get group: %w", err))
	}
	fmt.Printf("Fetched group: %v (GID: %v)\n", fetchedGroup["name"], fetchedGroup["gid"])

	// --- UPDATE ---
	updateParams := client.Params{
		"gid": 5001,
	}
	_, err = rest.Groups.Update(group.Id, updateParams)
	if err != nil {
		panic(fmt.Errorf("failed to update group: %w", err))
	}
	fmt.Println("Group updated successfully.")

	// --- DELETE ---
	_, err = rest.Groups.Delete(client.Params{
		"name": "go-client-test-group",
	}, nil)
	if err != nil {
		panic(fmt.Errorf("failed to delete group: %w", err))
	}
	fmt.Println("Group deleted successfully.")
}
