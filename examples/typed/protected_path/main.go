package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/resources/typed"
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

	req := typed.ProtectedPathSearchParams{
		RawData: client.Params{"name__endswith": "b816a408a6"},
	}

	resp, err := rest.ProtectedPaths.Get(&req)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp)
}
