package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
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

	// Create
	record, err := rest.Users.UserAccessKeys_POST(userId, tenantId)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Created access keys:\n")
	fmt.Printf("Access Key: %s\n", record.AccessKey)
	fmt.Printf("Secret Key: %s\n", record.SecretKey)

	// Modify (disable)
	if err := rest.Users.UserAccessKeys_PATCH(userId, record.AccessKey, false); err != nil {
		panic(err)
	}
	fmt.Println("Access key disabled")

	// Delete
	if err := rest.Users.UserAccessKeys_DELETE(userId, record.AccessKey); err != nil {
		panic(err)
	}
	fmt.Println("Access key deleted")
}
