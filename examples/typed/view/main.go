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
	createParams := &typed.ViewRequestBody{
		Name:      "go-client-testview",
		Path:      "/go-client-testview",
		CreateDir: true,
		PolicyId:  1,
		Protocols: &[]string{"NFS"},
	}

	view, err := rest.Views.Create(createParams)
	if err != nil {
		panic(fmt.Errorf("failed to create view: %w", err))
	}
	fmt.Println("View created successfully.")
	fmt.Println(view.PrettyTable())

	// --- UPDATE ---
	updateParams := &typed.ViewRequestBody{
		Protocols: &[]string{"NFS", "NFS4"},
	}

	_, err = rest.Views.Update(view.Id, updateParams)
	if err != nil {
		panic(fmt.Errorf("failed to update view: %w", err))
	}
	fmt.Println("View updated successfully.")

	// --- DELETE ---
	searchParams := &typed.ViewSearchParams{
		Path: "/go-client-testview",
	}

	err = rest.Views.Delete(searchParams)
	if err != nil {
		panic(fmt.Errorf("failed to delete view: %w", err))
	}
	fmt.Println("View deleted successfully.")
}
