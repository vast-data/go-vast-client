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

	// Create access keys for a non-local user (e.g., LDAP, AD, NIS)
	createBody := &typed.UserNonLocalKeys_POST_Body{
		Uid:      int64(888),
		TenantId: int64(1),
	}

	result, err := rest.Users.UserNonLocalKeys_POST(createBody)
	if err != nil {
		panic(err)
	}

	fmt.Println("Created access keys:")
	fmt.Printf("Access Key: %s\n", result.AccessKey)
	fmt.Printf("Secret Key: %s\n", result.SecretKey)

	accessKey := result.AccessKey

	// Update (disable) the access key
	updateBody := &typed.UserNonLocalKeys_PATCH_Body{
		AccessKey: accessKey,
		Enabled:   false,
		Uid:       int64(888),
		TenantId:  int64(1),
	}

	if err := rest.Users.UserNonLocalKeys_PATCH(updateBody); err != nil {
		panic(err)
	}
	fmt.Println("Access key disabled")

	// Delete the access key
	deleteBody := &typed.UserNonLocalKeys_DELETE_Body{
		AccessKey: accessKey,
		Uid:       int64(888),
	}

	if err := rest.Users.UserNonLocalKeys_DELETE(deleteBody); err != nil {
		panic(err)
	}
	fmt.Println("Access key deleted")
}
