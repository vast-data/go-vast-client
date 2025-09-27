package main

import (
	"fmt"
	"log"

	"github.com/vast-data/go-vast-client/api"
)

func main() {
	fmt.Println("=== Testing bucket_logging Schema Differences ===")
	
	// Test 1: POST=views approach
	fmt.Println("\n1. POST=views approach:")
	postSchema, err := api.GetSchema_POST_StatusOk("views")
	if err != nil {
		log.Printf("Error getting POST schema: %v", err)
		return
	}
	
	if bucketLoggingProp, ok := postSchema.Value.Properties["bucket_logging"]; ok {
		if bucketLoggingProp.Value != nil {
			fmt.Printf("POST bucket_logging required fields: %v\n", bucketLoggingProp.Value.Required)
			if bucketLoggingProp.Value.Properties != nil {
				if destProp, ok := bucketLoggingProp.Value.Properties["destination_id"]; ok {
					fmt.Printf("POST bucket_logging.destination_id type: %v\n", destProp.Value.Type)
					fmt.Printf("POST bucket_logging.destination_id description: %s\n", destProp.Value.Description)
				}
			}
		}
	}
	
	// Test 2: SCHEMA=View approach
	fmt.Println("\n2. SCHEMA=View approach:")
	schemaFromComponents, err := api.GetSchema_FromComponents("View")
	if err != nil {
		log.Printf("Error getting schema from components: %v", err)
		return
	}
	
	if bucketLoggingProp, ok := schemaFromComponents.Value.Properties["bucket_logging"]; ok {
		if bucketLoggingProp.Value != nil {
			fmt.Printf("SCHEMA bucket_logging required fields: %v\n", bucketLoggingProp.Value.Required)
			if bucketLoggingProp.Value.Properties != nil {
				if destProp, ok := bucketLoggingProp.Value.Properties["destination_id"]; ok {
					fmt.Printf("SCHEMA bucket_logging.destination_id type: %v\n", destProp.Value.Type)
					fmt.Printf("SCHEMA bucket_logging.destination_id description: %s\n", destProp.Value.Description)
				}
			}
		}
	}
}
