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
	createParams := &typed.GroupRequestBody{
		Name: "go-client-test-group",
		Gid:  5000,
	}

	group, err := rest.Groups.Create(createParams)
	if err != nil {
		panic(fmt.Errorf("failed to create group: %w", err))
	}
	fmt.Printf("Group created successfully: %s (ID: %d, GID: %d)\n", group.Name, group.Id, group.Gid)

	// --- LIST ---
	groups, err := rest.Groups.List(&typed.GroupSearchParams{})
	if err != nil {
		panic(fmt.Errorf("failed to list groups: %w", err))
	}
	fmt.Printf("Found %d group(s)\n", len(groups))

	// --- GET ---
	fetchedGroup, err := rest.Groups.Get(&typed.GroupSearchParams{
		Name: "go-client-test-group",
	})
	if err != nil {
		panic(fmt.Errorf("failed to get group: %w", err))
	}
	fmt.Printf("Fetched group: %s (GID: %d)\n", fetchedGroup.Name, fetchedGroup.Gid)

	// --- UPDATE ---
	updateParams := &typed.GroupRequestBody{
		Gid: 5001,
	}

	_, err = rest.Groups.Update(group.Id, updateParams)
	if err != nil {
		panic(fmt.Errorf("failed to update group: %w", err))
	}
	fmt.Println("Group updated successfully.")

	// --- DELETE ---
	searchParams := &typed.GroupSearchParams{
		Name: "go-client-test-group",
	}

	err = rest.Groups.Delete(searchParams)
	if err != nil {
		panic(fmt.Errorf("failed to delete group: %w", err))
	}
	fmt.Println("Group deleted successfully.")
}
