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

	userId := 5
	tenantId := int64(1)

	// Get tenant data for a user
	getParams := core.Params{
		"tenant_id": tenantId,
	}

	result, err := rest.Users.UserTenantData_GET(userId, getParams)
	if err != nil {
		panic(err)
	}

	fmt.Println("User tenant data:")
	fmt.Println(result.PrettyTable())

	// Update tenant data for a user
	updateBody := core.Params{
		"tenant_id":           tenantId,
		"s3_superuser":        true,
		"allow_create_bucket": true,
		"allow_delete_bucket": true,
	}

	_, err = rest.Users.UserTenantData_PATCH(userId, updateBody)
	if err != nil {
		panic(err)
	}

	fmt.Println("\nUser tenant data updated successfully!")

	// Verify the update
	result2, err := rest.Users.UserTenantData_GET(userId, getParams)
	if err != nil {
		panic(err)
	}

	fmt.Println("\nUpdated tenant data:")
	fmt.Println(result2.PrettyTable())
}
