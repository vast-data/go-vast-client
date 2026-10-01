package main

import (
	"fmt"

	client "github.com/vast-data/go-vast-client"
	de "github.com/vast-data/go-vast-client/resources/typed/dataengine"
	"github.com/vast-data/go-vast-client/resources/typed/expr"
)

func main() {
	config := &client.VMSConfig{
		Host:     "l3118", // replace with your VAST address
		Username: "de-lab-admin",
		Password: "123456aA#",
		Tenant:   "de-lab", // DataEngine tenant
	}

	rest, err := client.NewTypedVMSRest(config)
	if err != nil {
		panic(err)
	}

	pipeline, err := rest.DataEngine.Pipelines.Get(&de.PipelineSearchParams{
		Name: expr.Str("my-pipeline"), // replace with an existing pipeline name
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Pipeline: %s (guid=%s status=%s)\n", pipeline.Name, pipeline.Guid, pipeline.Status)
}
