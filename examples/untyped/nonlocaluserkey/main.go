package main

import (
	"fmt"
	client "github.com/vast-data/go-vast-client"
)

func main() {
	config := &client.VMSConfig{
		Host:     "v95",
		Username: "admin",
		Password: "123456",
	}
	rest, err := client.NewVMSRest(config)
	if err != nil {
		panic(err)
	}

	result, err := rest.NonLocalUsers.Get(client.Params{"uid": 26565})
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", result.PrettyTable())
}
