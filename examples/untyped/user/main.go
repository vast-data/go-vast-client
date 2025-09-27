package main

import (
	"fmt"
	client "github.com/vast-data/go-vast-client"
)

type UserContainer struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func main() {
	config := &client.VMSConfig{
		Host:     "10.27.40.1", // replace with your VAST address
		Username: "admin",
		Password: "123456",
	}

	rest, err := client.NewVMSRest(config)
	if err != nil {
		panic(err)
	}

	// --- CREATE ---
	createParams := client.Params{
		"name": "myUser",
		"uid":  9999,
	}
	_, err = rest.Users.Create(createParams)
	if err != nil {
		panic(fmt.Errorf("failed to create user: %w", err))
	}
	fmt.Println("User created successfully.")

	// --- UPDATE ---
	updateParams := client.Params{
		"uid": 10000,
	}
	_, err = rest.Users.Update(1, updateParams) // replace `1` with the correct user ID
	if err != nil {
		panic(fmt.Errorf("failed to update user: %w", err))
	}
	fmt.Println("User updated successfully.")

	// --- GET + DESERIALIZE ---
	result, err := rest.Users.Get(client.Params{
		"name": "myUser",
	})
	if err != nil {
		panic(fmt.Errorf("failed to get user: %w", err))
	}

	var user UserContainer
	if err := result.Fill(&user); err != nil {
		panic(fmt.Errorf("failed to fill UserContainer: %w", err))
	}
	fmt.Printf("Fetched user: %+v\n", user)

	// --- DELETE ---
	_, err = rest.Users.Delete(client.Params{
		"name": "myUser",
	}, nil)
	if err != nil {
		panic(fmt.Errorf("failed to delete user: %w", err))
	}
	fmt.Println("User deleted successfully.")
}
