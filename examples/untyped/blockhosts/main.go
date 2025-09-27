package main

import (
	"fmt"
	client "github.com/vast-data/go-vast-client"
)

func main() {
	config := &client.VMSConfig{
		Host:       "10.141.200.101", // replace with your VAST IP
		Username:   "admin",          // Authentication user
		Password:   "123456",
		ApiVersion: "latest",
	}

	rest, err := client.NewVMSRest(config)
	if err != nil {
		panic(err)
	}

	//var params client.Params

	records, err := rest.BlockHosts.List(nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(records.PrettyTable())
}
