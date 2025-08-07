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

	createParams := client.Params{
		"name":             "mytopic",
		"database_name":    "mydb",
		"topic_partitions": 3,
	}
	res, err := rest.Topics.Create(createParams)
	if err != nil {
		panic(err)
	}

	fmt.Println(res)
}
