package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	client "github.com/vast-data/go-vast-client"
)

func main() {
	ctx := context.Background()
	config := &client.VMSConfig{
		Host:     "v117",
		Username: "admin",
		Password: "123456",
	}

	rest, err := client.NewVMSRest(config)
	if err != nil {
		panic(err)
	}
	session := rest.Session

	metrics := []string{
		"Capacity,drr",
		"Capacity,logical_space",
		"Capacity,logical_space_in_use",
		"Capacity,physical_space",
		"Capacity,physical_space_in_use",
	}

	// Construct query string using `url.Values`
	values := url.Values{}
	values.Set("object_type", "cluster")
	values.Set("time_frame", "5m")
	for _, m := range metrics {
		values.Add("prop_list", m)
	}

	// Proper URL with manually encoded multi-param `prop_list`
	urlWithQuery := "monitors/ad_hoc_query?" + values.Encode()

	res, err := session.Get(ctx, urlWithQuery, nil)
	if err != nil {
		panic(err)
	}

	// Parse response
	var parsed struct {
		Data     [][]interface{} `json:"data"`
		PropList []string        `json:"prop_list"`
	}
	raw, _ := json.Marshal(res)
	if err := json.Unmarshal(raw, &parsed); err != nil {
		panic(err)
	}

	if len(parsed.Data) == 0 {
		fmt.Println("no samples")
		return
	}

	last := parsed.Data[len(parsed.Data)-1]
	metricsMap := map[string]interface{}{}
	for i, name := range parsed.PropList {
		refined := name
		if idx := findComma(name); idx != -1 {
			refined = name[idx+1:]
		}
		if i < len(last) {
			metricsMap[refined] = last[i]
		}
	}

	const GiB = 1 << 30
	total := float64OrZero(metricsMap["logical_space"]) / GiB
	used := float64OrZero(metricsMap["logical_space_in_use"]) / GiB
	free := total - used

	fmt.Printf("Total Capacity: %.2f GiB\n", total)
	fmt.Printf("Free Capacity:  %.2f GiB\n", free)
}

func findComma(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			return i
		}
	}
	return -1
}

func float64OrZero(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case json.Number:
		f, _ := x.Float64()
		return f
	default:
		return 0
	}
}
