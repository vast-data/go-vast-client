package main

import (
	"fmt"
	"log"

	vast_client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/typed"
)

func main() {
	fmt.Println("=== FromStruct Demo ===")

	// Example 1: Using FromStruct with a typed request struct
	fmt.Println("\n1. Converting typed request to Params:")

	quotaRequest := &typed.QuotaRequest{
		Name:     stringPtr("test-quota"),
		TenantId: int64Ptr(1),
		PageSize: stringPtr("10"),
	}

	// Method 1: Using FromStruct on existing Params
	params1 := make(vast_client.Params)
	err := params1.FromStruct(quotaRequest)
	if err != nil {
		log.Fatalf("FromStruct failed: %v", err)
	}
	fmt.Printf("  FromStruct result: %+v\n", params1)

	// Method 2: Using NewParamsFromStruct convenience function
	params2, err := vast_client.NewParamsFromStruct(quotaRequest)
	if err != nil {
		log.Fatalf("NewParamsFromStruct failed: %v", err)
	}
	fmt.Printf("  NewParamsFromStruct result: %+v\n", params2)

	// Example 2: Custom struct conversion
	fmt.Println("\n2. Converting custom struct to Params:")

	type CustomRequest struct {
		Username string  `json:"username"`
		Email    string  `json:"email"`
		Age      int     `json:"age"`
		Active   *bool   `json:"active,omitempty"`
		Score    float64 `json:"score"`
	}

	customReq := CustomRequest{
		Username: "john.doe",
		Email:    "john@example.com",
		Age:      30,
		Active:   boolPtr(true),
		Score:    95.5,
	}

	customParams, err := vast_client.NewParamsFromStruct(customReq)
	if err != nil {
		log.Fatalf("Custom struct conversion failed: %v", err)
	}
	fmt.Printf("  Custom struct result: %+v\n", customParams)

	// Example 3: Demonstrating omitempty behavior
	fmt.Println("\n3. Demonstrating omitempty behavior:")

	customReqWithNil := CustomRequest{
		Username: "jane.doe",
		Email:    "jane@example.com",
		Age:      25,
		Active:   nil, // This should be omitted
		Score:    88.0,
	}

	nilParams, err := vast_client.NewParamsFromStruct(customReqWithNil)
	if err != nil {
		log.Fatalf("Nil field conversion failed: %v", err)
	}
	fmt.Printf("  With nil field (omitted): %+v\n", nilParams)

	// Example 4: Converting to query string and JSON body
	fmt.Println("\n4. Using Params for HTTP requests:")

	queryString := params1.ToQuery()
	fmt.Printf("  Query string: %s\n", queryString)

	body, err := params1.ToBody()
	if err != nil {
		log.Fatalf("ToBody failed: %v", err)
	}
	fmt.Printf("  JSON body ready for HTTP request: %v\n", body != nil)

	fmt.Println("\n=== Demo Complete ===")
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}
