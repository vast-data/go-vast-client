package main

import (
	"fmt"
	client "github.com/vast-data/go-vast-client"
)

type ViewContainer struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	TenantID int64  `json:"tenant_id"`
}

func main() {
	var view ViewContainer
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
		"name":       "myview",
		"path":       "/myview",
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
	_, err = rest.Views.Update(view.ID, updateParams)
	if err != nil {
		panic(fmt.Errorf("failed to update view: %w", err))
	}
	fmt.Println("View updated successfully.")

	// --- DELETE ---
	_, err = rest.Views.Delete(client.Params{
		"path__endswith": "view",
	}, nil)
	if err != nil {
		panic(fmt.Errorf("failed to delete view: %w", err))
	}
	fmt.Println("View deleted successfully.")
}
