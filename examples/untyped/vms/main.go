package main

import (
	"fmt"
	client "github.com/vast-data/go-vast-client"
)

type UserContainer struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

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

	//vms, err := rest.Vms.List(nil)
	//if err != nil {
	//	panic(fmt.Sprintf("error listing VMS: %s", err))
	//}
	//
	//fmt.Printf("VMS: %+v\n", vms)

	res, err := rest.Vms.SetMaxApiTokensPerUser(1, 6)
	fmt.Printf("res: %+v\n", res)
}
