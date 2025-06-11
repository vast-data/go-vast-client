package main

import (
	"fmt"
	client "github.com/vast-data/go-vast-client"
)

func main() {
	config := &client.VMSConfig{
		Host:     "10.27.40.1",
		Username: "admin",
		Password: "123456",
	}
	rest, err := client.NewVMSRest(config)
	if err != nil {
		panic(err)
	}

	version, err := rest.Versions.GetVersion()
	if err != nil {
		panic(err)
	}
	fmt.Println(version)
}
