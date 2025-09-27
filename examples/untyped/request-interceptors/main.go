package main

import (
	"bytes"
	"context"
	"encoding/json"
	client "github.com/vast-data/go-vast-client"
	"io"
	"log"
	"net/http"
)

func main() {
	config := &client.VMSConfig{
		Host:     "10.27.40.1", // replace with your VAST address
		Username: "admin",
		Password: "123456",
		BeforeRequestFn: func(ctx context.Context, r *http.Request, verb, url string, body io.Reader) error {
			// Example of BeforeRequest interceptor.
			// Interceptor takes copy of body so you can use it safely.
			log.Printf("Sending request: verb=%s, url=%s", verb, url)
			if body != nil {
				bodyBytes, err := io.ReadAll(body)
				if err != nil {
					return err
				}
				var pretty bytes.Buffer
				if err = json.Indent(&pretty, bodyBytes, "", "  "); err == nil {
					log.Printf("Request JSON:\n%s", pretty.String())
				} else {
					log.Printf("Request Body:\n%s", string(bodyBytes))
				}
			}
			return nil
		},
		AfterRequestFn: func(ctx context.Context, response client.Renderable) (client.Renderable, error) {
			log.Printf("Result:\n%s", response.PrettyTable())
			return response, nil
		},
	}

	rest, err := client.NewVMSRest(config)
	if err != nil {
		panic(err)
	}

	_, err = rest.Tenants.Get(nil)
	if err != nil {
		panic(err)
	}
}
