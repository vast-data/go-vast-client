package dataengine

import (
	"context"
	"net/http"

	"github.com/vast-data/go-vast-client/core"
)

// ScheduleTrigger is a hand-maintained untyped resource.
//
// OpenAPI splits paths by verb:
//   - POST / PUT / PATCH → /triggers/schedule[/ {guid}]
//   - GET list / GET+DELETE by id → /triggers[/{guid}] (?type=Schedule)
//
// GetResourcePath stays triggers/schedule so UIs (vx) show a distinct resource
// key and create schemas resolve correctly. List iterators use listPathResource.
//
// Typed autogen still keys off the create path in rest/dataengine/untyped.go;
// do not replace these overrides with generated stubs.
type ScheduleTrigger struct {
	*core.VastResource
}

const scheduleTriggerType = "Schedule"

func (r *ScheduleTrigger) List(params core.Params) (core.RecordSet, error) {
	return r.ListWithContext(r.Rest.GetCtx(), params)
}

func (r *ScheduleTrigger) ListWithContext(ctx context.Context, params core.Params) (core.RecordSet, error) {
	iter := r.GetIteratorWithContext(ctx, params, 0)
	return iter.All()
}

func (r *ScheduleTrigger) GetIterator(params core.Params, pageSize int) core.Iterator {
	return r.GetIteratorWithContext(r.Rest.GetCtx(), params, pageSize)
}

func (r *ScheduleTrigger) GetIteratorWithContext(ctx context.Context, params core.Params, pageSize int) core.Iterator {
	return core.NewDataEngineIterator(
		ctx,
		&listPathResource{inner: r, path: "/" + triggersCollectionPath + "/"},
		withDefaultTriggerType(params, scheduleTriggerType),
		pageSize,
	)
}

func (r *ScheduleTrigger) Get(params core.Params) (core.Record, error) {
	return r.GetWithContext(r.Rest.GetCtx(), params)
}

func (r *ScheduleTrigger) GetWithContext(ctx context.Context, params core.Params) (core.Record, error) {
	result, err := r.ListWithContext(ctx, params)
	if err != nil {
		return nil, err
	}
	switch len(result) {
	case 0:
		return nil, &core.NotFoundError{Resource: triggersCollectionPath, Query: params.ToQuery()}
	case 1:
		if result[0].Empty() {
			return nil, &core.NotFoundError{Resource: triggersCollectionPath, Query: params.ToQuery()}
		}
		return result[0], nil
	default:
		return nil, &core.TooManyRecordsError{ResourcePath: triggersCollectionPath, Params: params}
	}
}

func (r *ScheduleTrigger) GetById(id any, params ...core.Params) (core.Record, error) {
	return r.GetByIdWithContext(r.Rest.GetCtx(), id, params...)
}

func (r *ScheduleTrigger) GetByIdWithContext(ctx context.Context, id any, params ...core.Params) (core.Record, error) {
	query := firstParams(params)
	path := core.BuildResourcePathWithID(triggersCollectionPath, id)
	return core.Request[core.Record](ctx, r, http.MethodGet, path, query, nil)
}

func (r *ScheduleTrigger) Delete(searchParams core.Params, deleteParams ...core.Params) (core.Record, error) {
	return r.DeleteWithContext(r.Rest.GetCtx(), searchParams, deleteParams...)
}

func (r *ScheduleTrigger) DeleteWithContext(ctx context.Context, searchParams core.Params, deleteParams ...core.Params) (core.Record, error) {
	result, err := r.GetWithContext(ctx, searchParams)
	if err != nil {
		if core.IsNotFoundErr(err) {
			return core.Record{}, nil
		}
		return nil, err
	}
	idVal, ok := result["guid"]
	if !ok {
		idVal, ok = result["id"]
	}
	if !ok {
		return nil, &core.NotFoundError{Resource: triggersCollectionPath, Query: searchParams.ToQuery()}
	}
	return r.DeleteByIdWithContext(ctx, idVal, deleteParams...)
}

func (r *ScheduleTrigger) DeleteById(id any, params ...core.Params) (core.Record, error) {
	return r.DeleteByIdWithContext(r.Rest.GetCtx(), id, params...)
}

func (r *ScheduleTrigger) DeleteByIdWithContext(ctx context.Context, id any, params ...core.Params) (core.Record, error) {
	query, body, err := core.SplitQueryBody(params)
	if err != nil {
		return nil, err
	}
	path := core.BuildResourcePathWithID(triggersCollectionPath, id)
	return core.Request[core.Record](ctx, r, http.MethodDelete, path, query, body)
}

func (r *ScheduleTrigger) Ensure(searchParams core.Params, createParams ...core.Params) (core.Record, error) {
	return r.EnsureWithContext(r.Rest.GetCtx(), searchParams, createParams...)
}

func (r *ScheduleTrigger) EnsureWithContext(ctx context.Context, searchParams core.Params, createParams ...core.Params) (core.Record, error) {
	result, err := r.GetWithContext(ctx, searchParams)
	if core.IsNotFoundErr(err) {
		return r.CreateWithContext(ctx, createParams...)
	} else if err != nil {
		return nil, err
	}
	return result, nil
}
