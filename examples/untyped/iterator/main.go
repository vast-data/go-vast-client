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

	iter := rest.Views.GetIterator(client.Params{"name__contains": "view-1"}, 5)
	result, err := iter.All()

	fmt.Println(len(result))
}
