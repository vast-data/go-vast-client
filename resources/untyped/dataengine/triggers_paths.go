package dataengine

import (
	"context"

	"github.com/vast-data/go-vast-client/core"
)

// withDefaultTriggerType copies params and sets type when absent so typed
// trigger resources only return their own rows from GET /triggers.
func withDefaultTriggerType(params core.Params, triggerType string) core.Params {
	out := core.Params{}
	for k, v := range params {
		out[k] = v
	}
	if _, ok := out["type"]; !ok {
		out["type"] = triggerType
	}
	return out
}

func firstParams(params []core.Params) core.Params {
	if len(params) == 0 {
		return nil
	}
	return params[0]
}

// listPathResource wraps a VastResourceAPIWithContext so DataEngineIterator
// GETs listPath (e.g. /triggers/) while Create/Update keep the real resourcePath.
type listPathResource struct {
	inner core.VastResourceAPIWithContext
	path  string
}

func (r *listPathResource) GetResourcePath() string { return r.path }

func (r *listPathResource) Session() core.RESTSession { return r.inner.Session() }
func (r *listPathResource) GetResourceType() string   { return r.inner.GetResourceType() }
func (r *listPathResource) GetApiRoot() string        { return r.inner.GetApiRoot() }

func (r *listPathResource) List(p core.Params) (core.RecordSet, error) {
	return r.inner.List(p)
}
func (r *listPathResource) Create(p ...core.Params) (core.Record, error) { return r.inner.Create(p...) }
func (r *listPathResource) Update(id any, p ...core.Params) (core.Record, error) {
	return r.inner.Update(id, p...)
}
func (r *listPathResource) Delete(searchParams core.Params, deleteParams ...core.Params) (core.Record, error) {
	return r.inner.Delete(searchParams, deleteParams...)
}
func (r *listPathResource) DeleteById(id any, p ...core.Params) (core.Record, error) {
	return r.inner.DeleteById(id, p...)
}
func (r *listPathResource) Ensure(s core.Params, c ...core.Params) (core.Record, error) {
	return r.inner.Ensure(s, c...)
}
func (r *listPathResource) Get(p core.Params) (core.Record, error) { return r.inner.Get(p) }
func (r *listPathResource) GetById(id any, p ...core.Params) (core.Record, error) {
	return r.inner.GetById(id, p...)
}
func (r *listPathResource) Exists(p core.Params) (bool, error) { return r.inner.Exists(p) }
func (r *listPathResource) MustExists(p core.Params) bool      { return r.inner.MustExists(p) }
func (r *listPathResource) GetIterator(p core.Params, n int) core.Iterator {
	return r.inner.GetIterator(p, n)
}
func (r *listPathResource) Lock(keys ...any) func() { return r.inner.Lock(keys...) }

func (r *listPathResource) ListWithContext(ctx context.Context, p core.Params) (core.RecordSet, error) {
	return r.inner.ListWithContext(ctx, p)
}
func (r *listPathResource) CreateWithContext(ctx context.Context, p ...core.Params) (core.Record, error) {
	return r.inner.CreateWithContext(ctx, p...)
}
func (r *listPathResource) UpdateWithContext(ctx context.Context, id any, p ...core.Params) (core.Record, error) {
	return r.inner.UpdateWithContext(ctx, id, p...)
}
func (r *listPathResource) DeleteWithContext(ctx context.Context, searchParams core.Params, deleteParams ...core.Params) (core.Record, error) {
	return r.inner.DeleteWithContext(ctx, searchParams, deleteParams...)
}
func (r *listPathResource) DeleteByIdWithContext(ctx context.Context, id any, p ...core.Params) (core.Record, error) {
	return r.inner.DeleteByIdWithContext(ctx, id, p...)
}
func (r *listPathResource) EnsureWithContext(ctx context.Context, s core.Params, c ...core.Params) (core.Record, error) {
	return r.inner.EnsureWithContext(ctx, s, c...)
}
func (r *listPathResource) GetWithContext(ctx context.Context, p core.Params) (core.Record, error) {
	return r.inner.GetWithContext(ctx, p)
}
func (r *listPathResource) GetByIdWithContext(ctx context.Context, id any, p ...core.Params) (core.Record, error) {
	return r.inner.GetByIdWithContext(ctx, id, p...)
}
func (r *listPathResource) ExistsWithContext(ctx context.Context, p core.Params) (bool, error) {
	return r.inner.ExistsWithContext(ctx, p)
}
func (r *listPathResource) MustExistsWithContext(ctx context.Context, p core.Params) bool {
	return r.inner.MustExistsWithContext(ctx, p)
}
func (r *listPathResource) GetIteratorWithContext(ctx context.Context, p core.Params, n int) core.Iterator {
	return r.inner.GetIteratorWithContext(ctx, p, n)
}
