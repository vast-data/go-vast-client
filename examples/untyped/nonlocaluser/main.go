package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/core"
)

func main() {
	config := &client.VMSConfig{
		Host:     "l101", // replace with your VAST address
		Username: "admin",
		Password: "123456",
	}

	rest, err := client.NewVMSRest(config)
	if err != nil {
		panic(err)
	}

	// Query a non-local user by UID
	queryParams := core.Params{
		"uid":       888,
		"tenant_id": int64(1),
	}

	result, err := rest.Users.UserQuery_GET(queryParams)
	if err != nil {
		panic(err)
	}

	fmt.Println("User details:")
	fmt.Println(result.PrettyTable())

	// Update user S3 permissions
	updateParams := core.Params{
		"uid":                 888,
		"s3_superuser":        true,
		"allow_create_bucket": true,
		"allow_delete_bucket": true,
	}

	_, err = rest.Users.UserQuery_PATCH(updateParams)
	if err != nil {
		panic(err)
	}

	fmt.Println("\nUser S3 permissions updated successfully!")
}
