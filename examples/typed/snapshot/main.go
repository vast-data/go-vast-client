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
	createParams := &typed.SnapshotRequestBody{
		Name: "go-client-test-snapshot",
		Path: "/",
	}

	snapshot, err := rest.Snapshots.Create(createParams)
	if err != nil {
		panic(fmt.Errorf("failed to create snapshot: %w", err))
	}
	fmt.Printf("Snapshot created successfully: %s (ID: %d)\n", snapshot.Name, snapshot.Id)

	// --- LIST ---
	snapshots, err := rest.Snapshots.List(&typed.SnapshotSearchParams{})
	if err != nil {
		panic(fmt.Errorf("failed to list snapshots: %w", err))
	}
	fmt.Printf("Found %d snapshot(s)\n", len(snapshots))

	// --- GET ---
	fetchedSnapshot, err := rest.Snapshots.Get(&typed.SnapshotSearchParams{
		Name: "go-client-test-snapshot",
	})
	if err != nil {
		panic(fmt.Errorf("failed to get snapshot: %w", err))
	}
	fmt.Printf("Fetched snapshot: %s (Path: %s)\n", fetchedSnapshot.Name, fetchedSnapshot.Path)

	// --- DELETE ---
	searchParams := &typed.SnapshotSearchParams{
		Name: "go-client-test-snapshot",
	}

	err = rest.Snapshots.Delete(searchParams)
	if err != nil {
		panic(fmt.Errorf("failed to delete snapshot: %w", err))
	}
	fmt.Println("Snapshot deleted successfully.")
}
