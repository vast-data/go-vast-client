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

	tokenParams := &typed.ApiTokenRequestBody{
		ExpiryDate: "1Y",
		Name:       "my_new_token",
		Owner:      "admin",
	}

	result, err := rest.ApiTokens.Create(tokenParams)
	if err != nil {
		panic(err)
	}
	fmt.Println(result.PrettyTable())
}
