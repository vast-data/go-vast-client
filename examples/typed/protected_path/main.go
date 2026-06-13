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

	// Search using a typed expression (name ends with a suffix)
	resp, err := rest.ProtectedPaths.Get(&typed.ProtectedPathSearchParams{
		Name: expr.Str.EndsWith("b816a408a6"),
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(resp)
}
