package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
)

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

	kerberosId := 1 // replace with your Kerberos provider ID

	// Example 1: Generate a keytab
	fmt.Println("=== Generating Keytab ===")
	generateParams := client.Params{
		"admin_username": "admin@EXAMPLE.COM",
		"admin_password": "admin_password",
	}

	result, err := rest.Kerberos.GenerateKeytab(kerberosId, generateParams)
	if err != nil {
		panic(fmt.Errorf("failed to generate keytab: %w", err))
	}
	fmt.Printf("Generated keytab: %s\n", result.PrettyTable())
}
