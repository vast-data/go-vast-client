package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/resources/typed"
)

func main() {
	var snapshot typed.SnapshotUpsertModel

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
		"name": "go-client-test-snapshot",
		"path": "/",
	}
	result, err := rest.Snapshots.Create(createParams)
	if err != nil {
		panic(fmt.Errorf("failed to create snapshot: %w", err))
	}
	fmt.Println("Snapshot created successfully.")
	if err := result.Fill(&snapshot); err != nil {
		panic(fmt.Errorf("failed to fill SnapshotUpsertModel: %w", err))
	}

	// --- LIST ---
	snapshots, err := rest.Snapshots.List(client.Params{})
	if err != nil {
		panic(fmt.Errorf("failed to list snapshots: %w", err))
	}
	fmt.Printf("Found %d snapshot(s)\n", len(snapshots))

	// --- GET ---
	fetchedSnapshot, err := rest.Snapshots.Get(client.Params{
		"name": "go-client-test-snapshot",
	})
	if err != nil {
		panic(fmt.Errorf("failed to get snapshot: %w", err))
	}
	fmt.Printf("Fetched snapshot: %v (Path: %v)\n", fetchedSnapshot["name"], fetchedSnapshot["path"])

	// --- DELETE ---
	_, err = rest.Snapshots.Delete(client.Params{
		"name": "go-client-test-snapshot",
	}, nil)
	if err != nil {
		panic(fmt.Errorf("failed to delete snapshot: %w", err))
	}
	fmt.Println("Snapshot deleted successfully.")
}
