package dataengine

import (
	"context"

	"github.com/vast-data/go-vast-client/core"
	tde "github.com/vast-data/go-vast-client/resources/typed/dataengine"
)

// TypedRest is the typed nested VastRest for DataEngine, wrapping UntypedRest.
type TypedRest struct {
	Untyped *UntypedRest

	ContainerRegistries           *tde.ContainerRegistry
	KubernetesClusters            *tde.KubernetesCluster
	KubernetesSecrets             *tde.KubernetesSecret
	MtlsAuthenticationCredentials *tde.MtlsAuthenticationCredential
	Functions                     *tde.Function
	ScheduleTriggers              *tde.ScheduleTrigger
	ElementTriggers               *tde.ElementTrigger
	Pipelines                     *tde.Pipeline
	DataEngine                    *tde.DataEngine
}

var _ core.VastRest = (*TypedRest)(nil)

// NewTyped builds a typed DataEngine nested rest from an existing untyped nested rest.
func NewTyped(untyped *UntypedRest) *TypedRest {
	if untyped == nil {
		panic("dataengine.NewTyped: untyped rest is nil")
	}
	rest := &TypedRest{Untyped: untyped}
	rest.ContainerRegistries = core.NewTypedResource[tde.ContainerRegistry](rest.Untyped)
	rest.KubernetesClusters = core.NewTypedResource[tde.KubernetesCluster](rest.Untyped)
	rest.KubernetesSecrets = core.NewTypedResource[tde.KubernetesSecret](rest.Untyped)
	rest.MtlsAuthenticationCredentials = core.NewTypedResource[tde.MtlsAuthenticationCredential](rest.Untyped)
	rest.Functions = core.NewTypedResource[tde.Function](rest.Untyped)
	rest.ScheduleTriggers = core.NewTypedResource[tde.ScheduleTrigger](rest.Untyped)
	rest.ElementTriggers = core.NewTypedResource[tde.ElementTrigger](rest.Untyped)
	rest.Pipelines = core.NewTypedResource[tde.Pipeline](rest.Untyped)
	rest.DataEngine = core.NewTypedResource[tde.DataEngine](rest.Untyped)
	return rest
}

func (rest *TypedRest) GetSession() core.RESTSession { return rest.Untyped.GetSession() }
func (rest *TypedRest) GetResourceMap() map[string]core.VastResourceAPIWithContext {
	return rest.Untyped.GetResourceMap()
}
func (rest *TypedRest) GetCtx() context.Context {
	return rest.Untyped.GetCtx()
}
func (rest *TypedRest) SetCtx(ctx context.Context) { rest.Untyped.SetCtx(ctx) }
func (rest *TypedRest) GetApiRoot() string         { return rest.Untyped.GetApiRoot() }
