package rest

import (
	"context"
	"fmt"
	"reflect"

	"github.com/vast-data/go-vast-client/core"
	"github.com/vast-data/go-vast-client/resources/typed"
)

type TypedVMSRest struct {
	Untyped *UntypedVMSRest

	Quotas    *typed.Quota
	ApiTokens *typed.ApiToken
	Versions  *typed.Version
	Views     *typed.View
	VipPools  *typed.VipPool
	Users     *typed.User
}

func NewTypedVMSRest(config *core.VMSConfig) (*TypedVMSRest, error) {
	untyped, err := NewUntypedVMSRest(config)
	if err != nil {
		return nil, err
	}

	rest := &TypedVMSRest{
		Untyped: untyped,
	}

	rest.Quotas = newTypedResource[typed.Quota](rest)
	rest.ApiTokens = newTypedResource[typed.ApiToken](rest)
	rest.Versions = newTypedResource[typed.Version](rest)
	rest.Views = newTypedResource[typed.View](rest)
	rest.VipPools = newTypedResource[typed.VipPool](rest)
	rest.Users = newTypedResource[typed.User](rest)

	return rest, nil
}

func (rest *TypedVMSRest) GetSession() core.RESTSession {
	return rest.Untyped.Session
}

func (rest *TypedVMSRest) GetResourceMap() map[string]core.VastResourceAPIWithContext {
	return rest.Untyped.resourceMap
}

func (rest *TypedVMSRest) GetCtx() context.Context {
	return rest.Untyped.ctx
}

func (rest *TypedVMSRest) SetCtx(ctx context.Context) {
	rest.Untyped.ctx = ctx
}

func newTypedResource[T TypedVastResourceType](rest *TypedVMSRest) *T {
	resourceType := reflect.TypeOf(T{}).Name()
	resource := &T{
		core.NewTypedVastResource(resourceType, rest.Untyped),
	}
	if _, ok := rest.Untyped.resourceMap[resourceType]; !ok {
		panic(fmt.Errorf("untyped resource type %s not found in REST", resourceType))
	}
	return resource
}
