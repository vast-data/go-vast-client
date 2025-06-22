package main

import (
	"context"
	"fmt"
	client "github.com/vast-data/go-vast-client"
)

func main() {

	ctx := context.Background()

	config := &client.VMSConfig{
		Host:     "v95",
		Username: "admin",
		Password: "123456",
	}

	rest, err := client.NewVMSRest(config)
	if err != nil {
		panic(err)
	}

	schema, err := rest.OpenAPI.FetchSchemaV2(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to get swagger schema: %v", err))
	}

	for def, _ := range schema.Definitions {
		fmt.Println(def)
	}

}
