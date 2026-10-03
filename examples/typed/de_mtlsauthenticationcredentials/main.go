package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/core"
	de "github.com/vast-data/go-vast-client/resources/typed/dataengine"
)

func main() {
	config := &client.VMSConfig{
		Host:     "l1158",
		Username: "de-admin",
		Password: "123456aA#",
		Tenant:   "de-lab",
	}

	rest, err := client.NewTypedVMSRest(config)
	if err != nil {
		panic(err)
	}

	params := &de.MtlsAuthenticationCredentialSearchParams{
		RawData: core.Params{},
	}

	list, err := rest.DataEngine.MtlsAuthenticationCredentials.List(params)
	if err != nil {
		panic(err)
	}

	if len(list) == 0 {
		fmt.Println("no data")
	} else {
		fmt.Printf("%#v\n", list[0])
	}
}
