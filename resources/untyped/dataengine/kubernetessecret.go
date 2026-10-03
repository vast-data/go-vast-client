package dataengine

import (
	"context"
	"net/http"

	"github.com/vast-data/go-vast-client/core"
)

// KubernetesSecret is a hand-maintained untyped resource.
//
// OpenAPI only exposes POST (upsert) and DELETE — there is no list/read API.
// List overrides return an empty set so UIs (vx) show "No content" instead of
// treating a 405 as a hard error. Typed autogen stays CREATE|DELETE-only.
//
// DELETE is collection-scoped with the identity in the request body
// (kubernetes_cluster_vrn, namespace, secret_name) — not DELETE .../{guid}.
type KubernetesSecret struct {
	*core.VastResource
}

func (r *KubernetesSecret) List(params core.Params) (core.RecordSet, error) {
	return r.ListWithContext(r.Rest.GetCtx(), params)
}

func (r *KubernetesSecret) ListWithContext(ctx context.Context, params core.Params) (core.RecordSet, error) {
	return core.RecordSet{}, nil
}

func (r *KubernetesSecret) GetIterator(params core.Params, pageSize int) core.Iterator {
	return r.GetIteratorWithContext(r.Rest.GetCtx(), params, pageSize)
}

func (r *KubernetesSecret) GetIteratorWithContext(ctx context.Context, params core.Params, pageSize int) core.Iterator {
	return &emptyIterator{}
}

func (r *KubernetesSecret) Get(params core.Params) (core.Record, error) {
	return r.GetWithContext(r.Rest.GetCtx(), params)
}

func (r *KubernetesSecret) GetWithContext(ctx context.Context, params core.Params) (core.Record, error) {
	return nil, &core.NotFoundError{Resource: "kubernetes-secrets", Query: params.ToQuery()}
}

func (r *KubernetesSecret) GetById(id any, params ...core.Params) (core.Record, error) {
	return r.GetByIdWithContext(r.Rest.GetCtx(), id, params...)
}

func (r *KubernetesSecret) GetByIdWithContext(ctx context.Context, id any, params ...core.Params) (core.Record, error) {
	return nil, &core.NotFoundError{Resource: "kubernetes-secrets", Query: ""}
}

func (r *KubernetesSecret) Delete(searchParams core.Params, deleteParams ...core.Params) (core.Record, error) {
	return r.DeleteWithContext(r.Rest.GetCtx(), searchParams, deleteParams...)
}

// DeleteWithContext DELETEs /kubernetes-secrets with identity in the body.
// Prefer delete body from variadic deleteParams; fall back to searchParams when body is empty.
func (r *KubernetesSecret) DeleteWithContext(ctx context.Context, searchParams core.Params, deleteParams ...core.Params) (core.Record, error) {
	query, body, err := core.SplitQueryBody(deleteParams)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		body = searchParams
	}
	return core.Request[core.Record](ctx, r, http.MethodDelete, "kubernetes-secrets", query, body)
}

// emptyIterator satisfies core.Iterator for resources with no list API.
type emptyIterator struct{}

func (it *emptyIterator) Next() (core.RecordSet, error)     { return nil, nil }
func (it *emptyIterator) Previous() (core.RecordSet, error) { return nil, nil }
func (it *emptyIterator) HasNext() bool                     { return false }
func (it *emptyIterator) HasPrevious() bool                 { return false }
func (it *emptyIterator) Count() int                        { return 0 }
func (it *emptyIterator) PageSize() int                     { return 0 }
func (it *emptyIterator) Reset() (core.RecordSet, error)    { return nil, nil }
func (it *emptyIterator) All() (core.RecordSet, error)      { return nil, nil }
