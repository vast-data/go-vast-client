package untyped

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/vast-data/go-vast-client/core"
)

// CreateAccessKeysWithContext generates S3 access key pair for a User using the provided context.
// This method calls the POST /users/{id}/access_keys/ endpoint.
//
// Parameters:
//   - ctx: context for the request
//   - id: the ID parameter
//   - body: request body parameters
//
// Returns:
//   - Record: the response data
//   - error: if the request fails
func (r *UserKey) CreateAccessKeysWithContext(ctx context.Context, id any, body core.Params) (core.Record, error) {
	path := "/users/{id}/access_keys/"
	path = strings.ReplaceAll(path, "{id}", fmt.Sprintf("%v", id))
	return core.Request[core.Record](ctx, r, http.MethodPost, path, nil, body)
}

// CreateAccessKeys generates S3 access key pair for a User.
// This is a convenience method that uses the resource's default context.
func (r *UserKey) CreateAccessKeys(id any, body core.Params) (core.Record, error) {
	return r.CreateAccessKeysWithContext(r.Rest.GetCtx(), id, body)
}
