package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
)

func main() {
	config := &client.VMSConfig{
		Host:     "l3118", // replace with your VAST address
		Username: "de-lab-admin",
		Password: "123456aA#",
		Tenant:   "de-lab", // DataEngine tenant
	}

	rest, err := client.NewVMSRest(config)
	if err != nil {
		panic(err)
	}

	pipeline, err := rest.DataEngine.Pipelines.Get(client.Params{
		"name": "my-pipeline", // replace with an existing pipeline name
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Pipeline: %v (guid=%v status=%v)\n", pipeline["name"], pipeline["guid"], pipeline["status"])
}
