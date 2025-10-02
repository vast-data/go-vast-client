package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/resources/typed"
)

func main() {
	config := &client.VMSConfig{
		Host:     "l101", // replace with your VAST address
		Username: "admin",
		Password: "123456",
	}

	rest, err := client.NewTypedVMSRest(config)
	if err != nil {
		panic(err)
	}

	userId := 5
	tenantId := int64(1)

	// Get tenant data for a user
	getParams := &typed.UserTenantData_GET_Body{
		TenantId: tenantId,
	}

	result, err := rest.Users.UserTenantData_GET(userId, getParams)
	if err != nil {
		panic(err)
	}

	fmt.Println("User tenant data:")
	fmt.Printf("S3 Superuser: %t\n", result.S3Superuser)
	fmt.Printf("Allow Create Bucket: %t\n", result.AllowCreateBucket)
	fmt.Printf("Allow Delete Bucket: %t\n", result.AllowDeleteBucket)
	if result.S3PoliciesIds != nil {
		fmt.Printf("S3 Policies IDs: %v\n", *result.S3PoliciesIds)
	}

	// Update tenant data for a user
	updateBody := &typed.UserTenantData_PATCH_Body{
		TenantId:          tenantId,
		S3Superuser:       true,
		AllowCreateBucket: true,
		AllowDeleteBucket: true,
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
	fmt.Printf("S3 Superuser: %t\n", result2.S3Superuser)
	fmt.Printf("Allow Create Bucket: %t\n", result2.AllowCreateBucket)
	fmt.Printf("Allow Delete Bucket: %t\n", result2.AllowDeleteBucket)
}
