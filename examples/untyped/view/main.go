package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/resources/typed"
)

func main() {
	var view typed.ViewUpsertModel

	config := &client.VMSConfig{
		Host:     "l101", // replace with your VAST address
		Username: "admin",
		Password: "123456",
	}

	rest, err := client.NewUntypedVMSRest(config)
	if err != nil {
		panic(err)
	}

	// --- CREATE ---
	createParams := client.Params{
		"name":       "go-client-testview",
		"path":       "/go-client-testview",
		"create_dir": true,
		"policy_id":  1,
		"protocols":  []string{"NFS"},
	}
	result, err := rest.Views.Create(createParams)
	if err != nil {
		panic(fmt.Errorf("failed to create view: %w", err))
	}
	fmt.Println("View created successfully.")
	if err := result.Fill(&view); err != nil {
		panic(fmt.Errorf("failed to fill ViewContainer: %w", err))
	}

	// --- UPDATE ---
	updateParams := client.Params{
		"protocols": []string{"NFS", "NFS4"},
	}
	_, err = rest.Views.Update(view.Id, updateParams)
	if err != nil {
		panic(fmt.Errorf("failed to update view: %w", err))
	}
	fmt.Println("View updated successfully.")

	// --- DELETE ---
	_, err = rest.Views.Delete(client.Params{
		"path__endswith": "go-client-testview",
	}, nil)
	if err != nil {
		panic(fmt.Errorf("failed to delete view: %w", err))
	}
	fmt.Println("View deleted successfully.")
}
