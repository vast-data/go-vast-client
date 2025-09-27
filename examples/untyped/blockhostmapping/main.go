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
	hostName := "v144"
	volumeId := int64(1)
	tenantId := 1
	hostNQN := fmt.Sprintf("nqn.2014-08.com.example:default:%s", hostName)

	volumeHost, err := rest.BlockHosts.EnsureBlockHost(hostName, tenantId, hostNQN)
	if err != nil {
		panic(err)
	}
	volumeHostId := volumeHost.RecordID()

	fmt.Println("Ensure mapping")
	if _, err = rest.BlockHostMappings.EnsureMap(volumeHostId, volumeId); err != nil {
		panic(err)
	}

	fmt.Println("Ensure unmapping")
	if _, err = rest.BlockHostMappings.EnsureUnmap(volumeHostId, volumeId); err != nil {
		panic(err)
	}
}
