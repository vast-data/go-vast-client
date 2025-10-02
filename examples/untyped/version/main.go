package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
)

func main() {
	config := &client.VMSConfig{
		Host:     "l101",
		Username: "admin",
		Password: "123456",
	}
	rest, err := client.NewUntypedVMSRest(config)
	if err != nil {
		panic(err)
	}

	version, err := rest.Versions.GetVersion()
	if err != nil {
		panic(err)
	}
	fmt.Println(version)
}
