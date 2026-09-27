package dataengine

import (
	"context"
	"net/http"

	"github.com/vast-data/go-vast-client/core"
	unde "github.com/vast-data/go-vast-client/resources/untyped/dataengine"
)

// ApiRoot is the nested-rest path segment after /api/{version}/ for DataEngine
// (gateway: /api/{version}/serverless/...).
const ApiRoot = "serverless"

const (
	C = core.C
	L = core.L
	R = core.R
	U = core.U
	D = core.D
)

// UntypedRest is the nested VastRest for DataEngine (serverless) APIs.
// It shares the parent VMS session/auth and uses ApiRoot "serverless".
//
// Usage: vms.DataEngine.Functions.List(...)
// URL shape: /api/{version}/serverless/{resource}/
type UntypedRest struct {
	ctx         context.Context
	Session     core.RESTSession
	resourceMap map[string]core.VastResourceAPIWithContext
	apiRoot     string

	ContainerRegistries           *unde.ContainerRegistry
	KubernetesClusters            *unde.KubernetesCluster
	KubernetesSecrets             *unde.KubernetesSecret
	MtlsAuthenticationCredentials *unde.MtlsAuthenticationCredential
	Functions                     *unde.Function
	ScheduleTriggers              *unde.ScheduleTrigger
	ElementTriggers               *unde.ElementTrigger
	Pipelines                     *unde.Pipeline
	DataEngine                    *unde.DataEngine
}

var _ core.VastRest = (*UntypedRest)(nil)

// NewUntyped builds a DataEngine nested rest sharing parent session and context.
func NewUntyped(session core.RESTSession, ctx context.Context) *UntypedRest {
	if ctx == nil {
		ctx = context.Background()
	}
	de := &UntypedRest{
		ctx:         ctx,
		Session:     session,
		resourceMap: make(map[string]core.VastResourceAPIWithContext),
		apiRoot:     ApiRoot,
	}

	// ResourceOps match PackDataEngine OpenAPI (collection vs {guid} verbs).
	de.ContainerRegistries = core.NewUntypedResource[unde.ContainerRegistry](de, "container-registries", C, L, R, U, D)
	de.KubernetesClusters = core.NewUntypedResource[unde.KubernetesCluster](de, "kubernetes-clusters", C, L, R, U, D)
	de.KubernetesSecrets = core.NewUntypedResource[unde.KubernetesSecret](de, "kubernetes-secrets", C, D)
	de.MtlsAuthenticationCredentials = core.NewUntypedResource[unde.MtlsAuthenticationCredential](de, "mtls-authentication-credentials", C, L, R, U, D)
	de.Functions = core.NewUntypedResource[unde.Function](de, "functions", C, L, R, U, D)
	// functions/{guid} is PUT-only (no PATCH)
	setUpdateMethod(de.Functions.VastResource, http.MethodPut)
	de.ScheduleTriggers = core.NewUntypedResource[unde.ScheduleTrigger](de, "triggers/schedule", C, L, R, U, D)
	setUpdateMethod(de.ScheduleTriggers.VastResource, http.MethodPut)
	de.ElementTriggers = core.NewUntypedResource[unde.ElementTrigger](de, "triggers/element", C, L, R, U, D)
	setUpdateMethod(de.ElementTriggers.VastResource, http.MethodPut)
	de.Pipelines = core.NewUntypedResource[unde.Pipeline](de, "pipelines", C, L, R, U, D)
	// data-engine is a singleton collection (GET/POST/DELETE); no {guid} read.
	de.DataEngine = core.NewUntypedResource[unde.DataEngine](de, "data-engine", C, L, D)

	return de
}

func (rest *UntypedRest) GetSession() core.RESTSession { return rest.Session }
func (rest *UntypedRest) GetResourceMap() map[string]core.VastResourceAPIWithContext {
	return rest.resourceMap
}
func (rest *UntypedRest) GetCtx() context.Context    { return rest.ctx }
func (rest *UntypedRest) SetCtx(ctx context.Context) { rest.ctx = ctx }
func (rest *UntypedRest) GetApiRoot() string         { return rest.apiRoot }
