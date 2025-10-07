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

	rest, err := client.NewUntypedVMSRest(config)
	if err != nil {
		panic(err)
	}

	metrics := []string{
		"Capacity,drr",
		"Capacity,logical_space",
		"Capacity,logical_space_in_use",
		"Capacity,physical_space",
		"Capacity,physical_space_in_use",
	}

	// Build params for the ad_hoc_query extra method
	params := client.Params{
		"object_type": "cluster",
		"time_frame":  "5m",
		"prop_list":   metrics,
	}

	res, err := rest.Monitors.MonitorAdHocQuery_GET(params)
	if err != nil {
		panic(err)
	}

	fmt.Println(res)

}
