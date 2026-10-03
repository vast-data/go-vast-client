package dataengine

import (
	"context"
	"fmt"
	"net/http"

	"github.com/vast-data/go-vast-client/core"
)

type Pipeline struct {
	*core.VastResource
}

// DeployWithContext POSTs /pipeline/{guid}/deploy (OpenAPI uses singular "pipeline").
func (r *Pipeline) DeployWithContext(ctx context.Context, id any, query core.Params) (core.Record, error) {
	path := fmt.Sprintf("pipeline/%v/deploy", id)
	return core.Request[core.Record](ctx, r, http.MethodPost, path, query, nil)
}

// Deploy POSTs /pipeline/{guid}/deploy.
func (r *Pipeline) Deploy(id any, query core.Params) (core.Record, error) {
	return r.DeployWithContext(r.Rest.GetCtx(), id, query)
}
