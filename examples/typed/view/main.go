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
		Host:     "10.27.40.1", // replace with your VAST address
		Username: "admin",
		Password: "123456",
	}

	// Create typed client
	typedClient, err := typed.NewTypedVMSRest(config)
	if err != nil {
		log.Fatalf("Failed to create typed client: %v", err)
	}
	typedClient.Untyped.SetCtx(ctx)

	// Use typed view resource
	viewClient := typedClient.Views

	// Example 1: List views with typed search parameters
	searchParams := &typed.ViewSearchParams{
		TenantId: 1,
		Name:     "myview", // Search for specific view name
	}

	views, err := viewClient.List(searchParams)
	if err != nil {
		log.Printf("Failed to list views: %v", err)
	} else {
		fmt.Printf("Found %d views\n", len(views))
		for _, view := range views {
			if view.Name != "" && view.Id != 0 {
				fmt.Printf("View: ID=%d, Name=%s, Path=%s\n", view.Id, view.Name, view.Path)
			}
		}
	}

	// Example 2: Create a new view with typed create body
	createBody := &typed.ViewCreateBody{
		Name:      "typed-example-view",
		Path:      "/typed-example-view",
		CreateDir: true,
		PolicyId:  1,
		Protocols: []string{"NFS"},
	}

	newView, err := viewClient.Create(createBody)
	if err != nil {
		log.Printf("Failed to create view: %v", err)
	} else {
		fmt.Printf("Created view: ID=%d, Name=%s, Path=%s\n", 
			newView.Id, newView.Name, newView.Path)

		// Example 3: Update the view
		updateBody := &typed.ViewCreateBody{
			Name:      "typed-example-view-updated",
			Path:      "/typed-example-view",
			CreateDir: true,
			PolicyId:  1,
			Protocols: []string{"NFS", "NFS4"}, // Add NFS4 protocol
		}

		updatedView, err := viewClient.Update(newView.Id, updateBody)
		if err != nil {
			log.Printf("Failed to update view: %v", err)
		} else {
			fmt.Printf("Updated view: ID=%d, Name=%s, Protocols=%v\n", 
				updatedView.Id, updatedView.Name, updatedView.Protocols)
		}

		// Example 4: Get view by ID
		retrievedView, err := viewClient.GetById(newView.Id)
		if err != nil {
			log.Printf("Failed to get view by ID: %v", err)
		} else {
			fmt.Printf("Retrieved view: ID=%d, Name=%s\n", 
				retrievedView.Id, retrievedView.Name)
		}

		// Example 5: Check if view exists
		exists, err := viewClient.Exists(&typed.ViewSearchParams{
			Name: "typed-example-view-updated",
		})
		if err != nil {
			log.Printf("Failed to check view existence: %v", err)
		} else {
			fmt.Printf("View exists: %t\n", exists)
		}

		// Clean up: delete the created view
		deleteParams := &typed.ViewSearchParams{
			Path: "/typed-example-view",
		}
		if err := viewClient.Delete(deleteParams); err != nil {
			log.Printf("Failed to delete view: %v", err)
		} else {
			fmt.Println("Successfully deleted the example view")
		}
	}
}
