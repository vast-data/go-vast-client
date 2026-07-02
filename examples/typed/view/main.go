package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/resources/typed"
	"github.com/vast-data/go-vast-client/resources/typed/expr"
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
	fmt.Printf("View created successfully: %s (ID: %d)\n", view.Name, view.Id)

	// --- DELETE ---
	err = rest.Views.Delete(&typed.ViewSearchParams{
		Path: expr.S("/go-client-testview"),
	}, true)
	if err != nil {
		panic(fmt.Errorf("failed to delete view: %w", err))
	}
	fmt.Println("View deleted successfully.")
}
