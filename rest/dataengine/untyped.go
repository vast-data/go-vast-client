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
	resourceMap map[string]core.ResourceEntry
	apiRoot     string

	ContainerRegistries           *unde.ContainerRegistry
	KubernetesClusters            *unde.KubernetesCluster
	KubernetesSecrets             *unde.KubernetesSecret
	MtlsAuthenticationCredentials *unde.MtlsAuthenticationCredential
	Functions                     *unde.Function
	ScheduleTriggers              *unde.ScheduleTrigger
	ElementTriggers               *unde.ElementTrigger
	Pipelines                     *unde.Pipeline
	PipelineRevisions             *unde.PipelineRevision
	DataEngine                    *unde.DataEngine
	Agents                        *unde.Agent
	AgentDeployments              *unde.AgentDeployment
	Services                      *unde.Service
	ServiceDeployments            *unde.ServiceDeployment
	Stacks                        *unde.Stack
	VastGateways                  *unde.VastGateway
	JwtProviders                  *unde.JwtProvider
	Topics                        *unde.DataEngineTopic
	// Schedule and element triggers stay on their own resources. Exclude those
	// paths so this generic collection does not generate a second copy.
	// +apiexclude:extraMethod:GET|POST=/triggers/schedule/
	// +apiexclude:extraMethod:GET|PUT|PATCH|DELETE=/triggers/schedule/{guid}/
	// +apiexclude:extraMethod:GET|POST=/triggers/element/
	// +apiexclude:extraMethod:GET|PUT|PATCH|DELETE=/triggers/element/{guid}/
	Triggers  *unde.Trigger
	Reports   *unde.Report
	Dashboard *unde.Dashboard
	Search    *unde.Search
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
		resourceMap: make(map[string]core.ResourceEntry),
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
	// pipeline-revisions/{guid} is PUT-only (no PATCH).
	de.PipelineRevisions = core.NewUntypedResource[unde.PipelineRevision](de, "pipeline-revisions", C, L, R, U, D)
	setUpdateMethod(de.PipelineRevisions.VastResource, http.MethodPut)
	// data-engine is a singleton collection (GET/POST/DELETE); no {guid} read.
	de.DataEngine = core.NewUntypedResource[unde.DataEngine](de, "data-engine", C, L, D)

	de.Agents = core.NewUntypedResource[unde.Agent](de, "agents", C, L, R, U, D)
	de.AgentDeployments = core.NewUntypedResource[unde.AgentDeployment](de, "agent-deployments", L)
	de.Services = core.NewUntypedResource[unde.Service](de, "services", C, L, R, U, D)
	de.ServiceDeployments = core.NewUntypedResource[unde.ServiceDeployment](de, "service-deployments", L)
	de.Stacks = core.NewUntypedResource[unde.Stack](de, "stacks", C, L, R, U, D)
	de.VastGateways = core.NewUntypedResource[unde.VastGateway](de, "vast-gateways", C, L, R, U, D)
	de.JwtProviders = core.NewUntypedResource[unde.JwtProvider](de, "jwt-providers", C, L, R, U, D)
	de.Topics = core.NewUntypedResource[unde.DataEngineTopic](de, "topics", C, L, R, U, D)
	de.Triggers = core.NewUntypedResource[unde.Trigger](de, "triggers", C, L, R, U, D)
	// triggers/{guid} is PUT-only (no PATCH).
	setUpdateMethod(de.Triggers.VastResource, http.MethodPut)
	// reports/{report_id} is not {guid}, so get/update/delete are extra methods.
	de.Reports = core.NewUntypedResource[unde.Report](de, "reports", C, L)
	// No /dashboard collection. The stats routes are extra methods.
	de.Dashboard = core.NewUntypedResource[unde.Dashboard](de, "dashboard")
	de.Search = core.NewUntypedResource[unde.Search](de, "search", L)

	return de
}

func (rest *UntypedRest) GetSession() core.RESTSession { return rest.Session }
func (rest *UntypedRest) GetResourceMap() map[string]core.ResourceEntry {
	return rest.resourceMap
}
func (rest *UntypedRest) GetCtx() context.Context    { return rest.ctx }
func (rest *UntypedRest) SetCtx(ctx context.Context) { rest.ctx = ctx }
func (rest *UntypedRest) GetApiRoot() string         { return rest.apiRoot }
