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

	result, err := rest.UserKeys.CreateKey(1)
	if err != nil {
		panic(err)
	}
	accessKey := result["access_key"].(string)

	fmt.Printf("access key: %s\n", accessKey)

	if _, err = rest.UserKeys.DeleteKey(1, accessKey); err != nil {
		panic(err)
	}
}
