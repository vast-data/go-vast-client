package main

import (
	"context"
	"fmt"
	"log"

	client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/typed"
)

func main() {
	ctx := context.Background()
	config := &client.VMSConfig{
		Host:     "10.27.40.1",
		Username: "admin",
		Password: "123456",
	}

	// Create typed client
	typedClient, err := typed.NewTypedVMSRest(config)
	if err != nil {
		log.Fatalf("Failed to create typed client: %v", err)
	}
	typedClient.Untyped.SetCtx(ctx)

	// Use typed version resource (read-only)
	versionClient := typedClient.Versions

	// Example 1: List all versions with typed search parameters
	searchParams := &typed.VersionSearchParams{
		// Version is a read-only resource, so we can only search and retrieve
	}

	versions, err := versionClient.List(searchParams)
	if err != nil {
		log.Fatalf("Failed to list versions: %v", err)
	}

	fmt.Printf("Found %d versions\n", len(versions))
	for _, version := range versions {
		if version.Name != "" {
			fmt.Printf("Version: Name=%s, Created=%s\n", version.Name, version.Created)
		}
	}

	// Example 2: Get a specific version by ID (if any exist)
	if len(versions) > 0 && versions[0].Id != 0 {
		version, err := versionClient.GetById(versions[0].Id)
		if err != nil {
			log.Printf("Failed to get version: %v", err)
		} else {
			fmt.Printf("Retrieved version: %+v\n", version)
		}
	}

	// Example 3: Check if a version exists
	exists, err := versionClient.Exists(&typed.VersionSearchParams{
		Name: "4.7.0", // Example version name
	})
	if err != nil {
		log.Printf("Failed to check version existence: %v", err)
	} else {
		fmt.Printf("Version '4.7.0' exists: %t\n", exists)
	}

	// Example 4: Search for versions by name pattern
	if len(versions) > 0 {
		firstVersionName := versions[0].Name
		namedVersions, err := versionClient.List(&typed.VersionSearchParams{
			Name: firstVersionName,
		})
		if err != nil {
			log.Printf("Failed to search versions by name: %v", err)
		} else {
			fmt.Printf("Found %d versions matching name '%s'\n", len(namedVersions), firstVersionName)
		}
	}

	// Note: Version is a read-only resource, so no Create/Update/Delete methods are available
	// This demonstrates the power of typed resources - the compiler prevents you from
	// accidentally trying to modify read-only resources
}
