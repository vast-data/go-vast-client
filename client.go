package vast_client

import (
	"github.com/vast-data/go-vast-client/core"
	"github.com/vast-data/go-vast-client/rest"
)

type (
	VMSConfig                    = core.VMSConfig
	Params                       = core.Params
	Record                       = core.Record
	RecordSet                    = core.RecordSet
	Renderable                   = core.Renderable
	TypedVMSRest                 = rest.TypedVMSRest
	UntypedVMSRest               = rest.UntypedVMSRest
	VastResourceAPI              = core.VastResourceAPI
	VastResourceAPIWithContext   = core.VastResourceAPIWithContext
	InterceptableVastResourceAPI = core.InterceptableVastResourceAPI
)

func NewTypedVMSRest(config *VMSConfig) (*TypedVMSRest, error) {
	return rest.NewTypedVMSRest(config)
}

func NewUntypedVMSRest(config *VMSConfig) (*UntypedVMSRest, error) {
	return rest.NewUntypedVMSRest(config)
}
