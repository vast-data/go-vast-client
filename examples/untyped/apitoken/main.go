package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
)

func main() {
	config := &client.VMSConfig{
		Host:     "l101", // replace with your VAST address
		Username: "admin",
		Password: "123456",
	}

	rest, err := client.NewVMSRest(config)
	if err != nil {
		panic(err)
	}

	tokenParams := client.Params{
		"expiry_date": "1Y",
		"name":        "my_new_token",
		"owner":       "admin",
	}

	result, err := rest.ApiTokens.Create(tokenParams)
	if err != nil {
		panic(err)
	}
	fmt.Println(result.PrettyTable())
}
