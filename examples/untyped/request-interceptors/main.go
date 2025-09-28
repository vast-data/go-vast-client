package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	client "github.com/vast-data/go-vast-client"
)

func main() {
	config := &client.VMSConfig{
		Host:     "10.27.40.1", // replace with your VAST address
		Username: "admin",
		Password: "123456",
		BeforeRequestFn: func(ctx context.Context, r *http.Request, verb, url string, body io.Reader) error {
			// Improved BeforeRequest interceptor that handles null bodies properly
			log.Printf("➤ Sending request: [%s] %s", verb, url)

			if body != nil {
				bodyBytes, err := io.ReadAll(body)
				if err != nil {
					log.Printf("ERROR: failed to read request body: %v", err)
					return err
				}

				trimmed := bytes.TrimSpace(bodyBytes)
				// Skip logging if body is empty or just "null"
				if len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null")) {
					var compact bytes.Buffer
					if err := json.Compact(&compact, trimmed); err == nil {
						log.Printf("Request body: %s", compact.String())
					} else {
						log.Printf("Request body (raw): %s", string(trimmed))
					}
				}
			}
			return nil
		},
		AfterRequestFn: func(ctx context.Context, response client.Renderable) (client.Renderable, error) {
			log.Printf("✓ Response received:\n%s", response.PrettyTable())
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
