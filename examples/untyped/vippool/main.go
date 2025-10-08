package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/resources/typed"
)

func main() {
	var pool typed.ViewUpsertModel

	config := &client.VMSConfig{
		Host:     "l101", // replace with your VAST IP
		Username: "admin",
		Password: "123456",
	}

	rest, err := client.NewVMSRest(config)
	if err != nil {
		panic(err)
	}

	// --- CREATE ---
	createParams := client.Params{
		"name":        "myvippool",
		"start_ip":    "20.0.0.1",
		"end_ip":      "20.0.0.16",
		"subnet_cidr": 24,
	}
	result, err := rest.VipPools.Create(createParams)
	if err != nil {
		panic(fmt.Errorf("failed to create VIP pool: %w", err))
	}
	fmt.Println("VIP Pool created.")
	if err = result.Fill(&pool); err != nil {
		panic(fmt.Errorf("failed to fill VippoolContainer: %w", err))
	}

	// --- UPDATE ---
	updateParams := client.Params{
		"subnet_cidr": 22,
	}
	_, err = rest.VipPools.Update(pool.Id, updateParams)
	if err != nil {
		panic(fmt.Errorf("failed to update VIP pool: %w", err))
	}
	fmt.Println("VIP Pool updated.")

	// --- DELETE ---
	_, err = rest.VipPools.Delete(client.Params{"name": "myvippool"}, nil)
	if err != nil {
		panic(fmt.Errorf("failed to delete VIP pool: %w", err))
	}
	fmt.Println("VIP Pool deleted.")
}
