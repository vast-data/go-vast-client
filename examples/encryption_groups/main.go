package main

import (
	"fmt"
	client "github.com/vast-data/go-vast-client"
)

func main() {
	config := &client.VMSConfig{
		Host:     "v95", // replace with your VAST address
		Username: "admin",
		Password: "123456",
	}

	rest, err := client.NewVMSRest(config)
	if err != nil {
		panic(err)
	}

	groups, err := rest.EncryptionGroups.List(nil)
	fmt.Println(groups)
}
