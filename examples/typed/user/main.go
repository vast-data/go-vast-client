package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	client "github.com/vast-data/go-vast-client"
	"github.com/vast-data/go-vast-client/typed"
)

// BeforeRequestFnCallback logs the HTTP request being sent.
// It reads and optionally compacts the body (if present) for structured logging,
// and includes the request ID from context (if available).
func BeforeRequestFnCallback(_ context.Context, _ *http.Request, verb, url string, body io.Reader) error {
	var logMsg strings.Builder
	logMsg.WriteString(fmt.Sprintf("➤ START: [%s] %s", verb, url))

	if body != nil {
		bodyBytes, err := io.ReadAll(body)
		if err != nil {
			log.Printf("ERROR: failed to read request body: %v", err)
			return err
		}

		trimmed := bytes.TrimSpace(bodyBytes)
		if len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null")) {
			var compact bytes.Buffer
			if err := json.Compact(&compact, trimmed); err == nil {
				logMsg.WriteString(fmt.Sprintf(" - body: %s", compact.String()))
			} else {
				logMsg.WriteString(fmt.Sprintf(" - body (raw): %s", string(trimmed)))
			}
		}
	}

	log.Println(logMsg.String())
	return nil
}

// AfterRequestFnCallback logs the response received from the HTTP request.
// It uses the response's PrettyTable method to render a formatted table,
// and includes the request ID from context.
func AfterRequestFnCallback(ctx context.Context, response client.Renderable) (client.Renderable, error) {
	log.Printf("✓ END: \n%s", response.PrettyTable())
	return response, nil
}

func main() {
	ctx := context.Background()

	config := &client.VMSConfig{
		Host:            "l101", // replace with your VAST IP
		Username:        "admin",
		Password:        "123456",
		BeforeRequestFn: BeforeRequestFnCallback,
		AfterRequestFn:  AfterRequestFnCallback,
	}

	// Create typed client
	typedClient, err := typed.NewTypedVMSRest(config)
	if err != nil {
		log.Fatalf("Failed to create typed client: %v", err)
	}

	typedClient.SetCtx(ctx)

	searchParams := &typed.UserSearchParams{
		Name: "vb",
	}

	user, err := typedClient.Users.Get(searchParams)
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}

	fmt.Printf("User found: ID=%d, Name=%s, Email=%s\n", user.Id, user.Name, user)

	gids := &[]int64{}

	_, err = typedClient.Users.Update(user.Id, &typed.UserRequestBody{
		Gids: gids,
	})
	if err != nil {
		log.Fatalf("Failed to update user: %v", err)
	}
}
