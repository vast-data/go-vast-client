package main

import (
	"context"
	"fmt"
	client "github.com/vast-data/go-vast-client"
)

const KiB = 1024

type QuotaContainer struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	TenantID  int64  `json:"tenant_id"`
	HardLimit int64  `json:"hard_limit"`
}

func main() {
	var cn QuotaContainer
	ctx := context.Background()
	config := &client.VMSConfig{
		Host:     "10.27.40.1",
		Username: "admin",
		Password: "123456",
	}
	rest, err := client.NewVMSRest(config)
	if err != nil {
		panic(err)
	}
	rest.SetCtx(ctx)

	result, err := rest.Quotas.Create(client.Params{
		"name":       "myquota",
		"path":       "/myview",
		"tenant_id":  1,
		"hard_limit": 10 * KiB,
	})
	if err != nil {
		panic(err)
	}

	if err = result.Fill(&cn); err != nil {
		panic(err)
	}
	fmt.Println(cn)

	if _, err = rest.Quotas.Update(cn.ID, client.Params{"hard_limit": 20 * KiB}); err != nil {
		panic(err)
	}

	if _, err = rest.Quotas.Delete(client.Params{"name": "myquota"}, nil); err != nil {
		panic(err)
	}
}
