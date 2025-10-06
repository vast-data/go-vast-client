//go:build examples

package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
)

func main() {
	// Example 1: Using Basic Authentication
	configBasicAuth := &client.VMSConfig{
		Host:         "10.27.40.1", // replace with your VAST address
		Username:     "admin",
		Password:     "123456",
		UseBasicAuth: true, // Enable Basic Authentication
	}

	restBasic, err := client.NewTypedVMSRest(configBasicAuth)
	if err != nil {
		panic(fmt.Errorf("failed to create basic auth client: %w", err))
	}

	// List versions using Basic Auth
	versions, err := restBasic.Versions.List(nil)
	if err != nil {
		panic(fmt.Errorf("failed to list versions: %w", err))
	}
	fmt.Printf("Basic Auth: Found %d version(s)\n", len(versions))

	// Example 2: Default JWT Authentication (for comparison)
	configJWT := &client.VMSConfig{
		Host:     "10.27.40.1",
		Username: "admin",
		Password: "123456",
		// UseBasicAuth: false (or omitted) - uses JWT by default
	}

	restJWT, err := client.NewTypedVMSRest(configJWT)
	if err != nil {
		panic(fmt.Errorf("failed to create JWT client: %w", err))
	}

	// List versions using JWT Auth
	versionsJWT, err := restJWT.Versions.List(nil)
	if err != nil {
		panic(fmt.Errorf("failed to list versions: %w", err))
	}
	fmt.Printf("JWT Auth: Found %d version(s)\n", len(versionsJWT))

	// Example 3: API Token Authentication (highest priority)
	configAPIToken := &client.VMSConfig{
		Host:     "10.27.40.1",
		ApiToken: "your-api-token-here", // API Token has highest priority
	}

	restAPIToken, err := client.NewTypedVMSRest(configAPIToken)
	if err != nil {
		panic(fmt.Errorf("failed to create API token client: %w", err))
	}

	versionsAPI, err := restAPIToken.Versions.List(nil)
	if err != nil {
		panic(fmt.Errorf("failed to list versions: %w", err))
	}
	fmt.Printf("API Token Auth: Found %d version(s)\n", len(versionsAPI))
}
