package main

import (
	"fmt"
	"log"

	"github.com/vast-data/go-vast-client/api"
)

func main() {
	fmt.Println("=== Testing Schema Resolution Differences ===")
	
	// Test 1: POST=views approach
	fmt.Println("\n1. POST=views approach:")
	postSchema, err := api.GetSchema_POST_StatusOk("views")
	if err != nil {
		log.Printf("Error getting POST schema: %v", err)
	} else {
		fmt.Printf("POST schema properties count: %d\n", len(postSchema.Value.Properties))
		fmt.Printf("POST schema required fields: %v\n", postSchema.Value.Required)
		
		// Show first few properties
		count := 0
		for propName := range postSchema.Value.Properties {
			if count >= 3 {
				break
			}
			fmt.Printf("  Property: %s\n", propName)
			count++
		}
	}
	
	// Test 2: SCHEMA=View approach
	fmt.Println("\n2. SCHEMA=View approach:")
	schemaFromComponents, err := api.GetSchema_FromComponents("View")
	if err != nil {
		log.Printf("Error getting schema from components: %v", err)
	} else {
		fmt.Printf("Components schema properties count: %d\n", len(schemaFromComponents.Value.Properties))
		fmt.Printf("Components schema required fields: %v\n", schemaFromComponents.Value.Required)
		
		// Show first few properties
		count := 0
		for propName := range schemaFromComponents.Value.Properties {
			if count >= 3 {
				break
			}
			fmt.Printf("  Property: %s\n", propName)
			count++
		}
	}
	
	// Test 3: Compare if they're the same
	fmt.Println("\n3. Comparison:")
	if postSchema != nil && schemaFromComponents != nil {
		postCount := len(postSchema.Value.Properties)
		componentCount := len(schemaFromComponents.Value.Properties)
		fmt.Printf("Same property count: %v (%d vs %d)\n", postCount == componentCount, postCount, componentCount)
		
		postReqCount := len(postSchema.Value.Required)
		componentReqCount := len(schemaFromComponents.Value.Required)
		fmt.Printf("Same required count: %v (%d vs %d)\n", postReqCount == componentReqCount, postReqCount, componentReqCount)
		
		// Check if they have the same properties
		fmt.Println("\n4. Property comparison:")
		
		// Check destination_id property specifically
		if postDestProp, ok := postSchema.Value.Properties["destination_id"]; ok {
			fmt.Printf("POST destination_id description: %s\n", postDestProp.Value.Description)
		}
		if compDestProp, ok := schemaFromComponents.Value.Properties["destination_id"]; ok {
			fmt.Printf("SCHEMA destination_id description: %s\n", compDestProp.Value.Description)
		}
		
		// Check bucket_logging nested object
		if postBucketProp, ok := postSchema.Value.Properties["bucket_logging"]; ok {
			if postBucketProp.Value != nil && postBucketProp.Value.Properties != nil {
				if destProp, ok := postBucketProp.Value.Properties["destination_id"]; ok {
					fmt.Printf("POST bucket_logging.destination_id description: %s\n", destProp.Value.Description)
				}
			}
		}
		if compBucketProp, ok := schemaFromComponents.Value.Properties["bucket_logging"]; ok {
			if compBucketProp.Value != nil && compBucketProp.Value.Properties != nil {
				if destProp, ok := compBucketProp.Value.Properties["destination_id"]; ok {
					fmt.Printf("SCHEMA bucket_logging.destination_id description: %s\n", destProp.Value.Description)
				}
			}
		}
	}
}
