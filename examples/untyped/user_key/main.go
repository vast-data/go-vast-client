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

	rest, err := client.NewUntypedVMSRest(config)
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

	fmt.Println(record.PrettyTable())

	accessKey := record["access_key"].(string)

	// Modify
	if err := rest.Users.UserAccessKeys_PATCH(userId, accessKey, false); err != nil {
		panic(err)
	}

	// Delete
	if err := rest.Users.UserAccessKeys_DELETE(userId, accessKey); err != nil {
		panic(err)
	}

}
