package main

import (
	client "github.com/vast-data/go-vast-client"
	"log"
)

func main() {
	config := &client.VMSConfig{
		Host:     "v95", // replace with your VAST IP
		Username: "admin",
		Password: "123456",
	}

	rest, err := client.NewVMSRest(config)
	if err != nil {
		panic(err)
	}

	eventDefinition, err := rest.EventDefinitions.Update(571, client.Params{"severity": "MAJOR"})
	if err != nil {
		log.Fatal(err)
	}

	log.Println(eventDefinition)

}
