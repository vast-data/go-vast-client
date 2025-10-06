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
	rest, err := client.NewTypedVMSRest(config)
	if err != nil {
		panic(err)
	}

	versions, err := rest.Versions.List(nil)
	if err != nil {
		panic(err)
	}

	if len(versions) > 0 {
		v := versions[0]
		fmt.Printf("Version Name: %s\n", v.Name)
		fmt.Printf("System Version: %s\n", v.SysVersion)
		fmt.Printf("OS Version: %s\n", v.OsVersion)
		fmt.Printf("Build: %s\n", v.Build)
		fmt.Printf("Status: %s\n", v.Status)
	}
}
