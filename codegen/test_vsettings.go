package main

import (
	"fmt"
	"log"
	
	"github.com/vast-data/go-vast-client/codegen/apibuilder"
)

func main() {
	api, err := apibuilder.NewAPIBuilder("../openapi_schema/api.json")
	if err != nil {
		log.Fatal(err)
	}
	
	// Check PATCH /clusters/{id}/vsettings/
	schema, err := api.GetRequestBodySchema("PATCH", "/clusters/{id}/vsettings/")
	if err != nil {
		log.Fatalf("Failed to get request body schema: %v", err)
	}
	
	fmt.Printf("Request body schema for PATCH /clusters/{id}/vsettings/:\n")
	fmt.Printf("Type: %v\n", schema.Value.Type)
	fmt.Printf("Properties: %v\n", len(schema.Value.Properties))
	for name, prop := range schema.Value.Properties {
		fmt.Printf("  - %s: %+v\n", name, prop.Value)
	}
}
