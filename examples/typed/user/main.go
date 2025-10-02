//go:build examples

package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/resources/typed"
)

func main() {
	config := &client.VMSConfig{
		Host:     "10.27.40.1", // replace with your VAST address
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
	result, err := rest.Users.Create(createParams)
	if err != nil {
		panic(fmt.Errorf("failed to create user: %w", err))
	}
	fmt.Println("User created successfully.")

	// --- UPDATE ---
	updateParams := &typed.UserRequestBody{
		Uid: 10000,
	}
	_, err = rest.Users.Update(result.Id, updateParams)
	if err != nil {
		panic(fmt.Errorf("failed to update user: %w", err))
	}
	fmt.Println("User updated successfully.")

	// --- GET ---
	user, err := rest.Users.Get(&typed.UserSearchParams{
		Name: "myUser",
	})
	if err != nil {
		panic(fmt.Errorf("failed to get user: %w", err))
	}
	fmt.Printf("Fetched user: %+v\n", user)

	// --- DELETE ---
	err = rest.Users.Delete(&typed.UserSearchParams{
		Name: "myUser",
	})
	if err != nil {
		panic(fmt.Errorf("failed to delete user: %w", err))
	}
	fmt.Println("User deleted successfully.")
}
