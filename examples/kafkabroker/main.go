package main

import (
	"fmt"
	client "github.com/vast-data/go-vast-client"
	"log"
)

func main() {
	config := &client.VMSConfig{
		Host:     "v99", // replace with your VAST IP
		Username: "admin",
		Password: "123456",
	}

	rest, err := client.NewVMSRest(config)
	if err != nil {
		panic(err)
	}

	// --- CREATE ---
	kafkaBrokerName := "my-kafka-broker"
	createParams := client.Params{
		"name": kafkaBrokerName,
		"addresses": []client.Params{
			{"host": "10.131.21.121", "port": 31485},
			{"host": "10.131.21.121", "port": 31486},
		},
	}
	result, err := rest.KafkaBrokers.Ensure(client.Params{"name": kafkaBrokerName}, createParams)
	if err != nil {
		log.Fatalf("Error creating Kafka Broker: %v", err)
	}
	fmt.Println(result.PrettyTable())

}
