//go:build examples

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
	rest, err := client.NewTypedVMSRest(config)
	if err != nil {
		panic(err)
	}

	versions, err := rest.Versions.List(nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(versions[0].PrettyTable())
}
