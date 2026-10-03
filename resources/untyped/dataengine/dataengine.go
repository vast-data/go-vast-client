package dataengine

import (
	"context"
	"net/http"

	"github.com/vast-data/go-vast-client/core"
)

// DataEngine maps to /data-engine (provision / status / delete).
// It is a singleton collection: DELETE has no {guid} path segment.
type DataEngine struct {
	*core.VastResource
}

func (r *DataEngine) Delete(searchParams core.Params, deleteParams ...core.Params) (core.Record, error) {
	return r.DeleteWithContext(r.Rest.GetCtx(), searchParams, deleteParams...)
}

// DeleteWithContext DELETEs /data-engine (no id). Optional query carries tenant_name etc.
func (r *DataEngine) DeleteWithContext(ctx context.Context, searchParams core.Params, deleteParams ...core.Params) (core.Record, error) {
	query, body, err := core.SplitQueryBody(deleteParams)
	if err != nil {
		return nil, err
	}
	return core.Request[core.Record](ctx, r, http.MethodDelete, "data-engine", query, body)
}
