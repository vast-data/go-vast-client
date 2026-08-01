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

	// --- GET (exact match) ---
	fetchedSnapshot, err := rest.Snapshots.Get(&typed.SnapshotSearchParams{
		Name: expr.Str("go-client-test-snapshot"),
	})
	if err != nil {
		panic(fmt.Errorf("failed to get snapshot: %w", err))
	}
	fmt.Printf("Fetched snapshot: %s (Path: %s)\n", fetchedSnapshot.Name, fetchedSnapshot.Path)

	// --- GET (expression: name starts with prefix) ---
	filtered, err := rest.Snapshots.List(&typed.SnapshotSearchParams{
		Name: expr.Str.StartsWith("go-client"),
	})
	if err != nil {
		panic(fmt.Errorf("failed to list filtered snapshots: %w", err))
	}
	fmt.Printf("Snapshots starting with 'go-client': %d\n", len(filtered))

	// --- DELETE ---
	err = rest.Snapshots.Delete(&typed.SnapshotSearchParams{
		Name: expr.Str("go-client-test-snapshot"),
	})
	if err != nil {
		panic(fmt.Errorf("failed to delete snapshot: %w", err))
	}
	fmt.Println("Snapshot deleted successfully.")
}
