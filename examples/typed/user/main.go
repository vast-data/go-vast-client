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

	// --- CREATE ---
	createParams := &typed.UserRequestBody{
		Name: "myUser",
		Uid:  9999,
	}
	_, err = rest.Users.Create(createParams)
	if err != nil {
		panic(fmt.Errorf("failed to create user: %w", err))
	}
	fmt.Println("User created successfully.")

	// --- DELETE ---
	err = rest.Users.Delete(&typed.UserSearchParams{
		Name: "myUser",
	})
	if err != nil {
		panic(fmt.Errorf("failed to delete user: %w", err))
	}
	fmt.Println("User deleted successfully.")
}
