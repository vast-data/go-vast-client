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

	rest, err := client.NewTypedVMSRest(config)
	if err != nil {
		panic(err)
	}

	hosts, err := rest.Hosts.List(nil)
	if err != nil {
		panic(err)
	}

	for _, host := range hosts {
		fmt.Println(*host)
	}

	discoveredHosts, err := rest.Hosts.HostDiscoveredHosts_GET(nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Discovered hosts: %#v\n", discoveredHosts)
}
