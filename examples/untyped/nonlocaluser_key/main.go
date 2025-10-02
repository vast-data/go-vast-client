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

	rest, err := client.NewUntypedVMSRest(config)
	if err != nil {
		panic(err)
	}

	// Create access keys for a non-local user (e.g., LDAP, AD, NIS)
	createParams := core.Params{
		"uid":       888,
		"tenant_id": int64(1),
	}

	record, err := rest.Users.UserNonLocalKeys_POST(createParams)
	if err != nil {
		panic(err)
	}

	fmt.Println("Created access keys:")
	fmt.Println(record.PrettyTable())

	accessKey := record["access_key"].(string)

	// Update (disable) the access key
	updateParams := core.Params{
		"access_key": accessKey,
		"enabled":    false,
		"uid":        888,
		"tenant_id":  int64(1),
	}

	if err := rest.Users.UserNonLocalKeys_PATCH(updateParams); err != nil {
		panic(err)
	}
	fmt.Println("Access key disabled")

	// Delete the access key
	deleteParams := core.Params{
		"access_key": accessKey,
		"uid":        888,
	}

	if err := rest.Users.UserNonLocalKeys_DELETE(deleteParams); err != nil {
		panic(err)
	}
	fmt.Println("Access key deleted")
}
