package vast_client

import (
	"github.com/vast-data/go-vast-client/core"
	"github.com/vast-data/go-vast-client/rest"
)

type (
	VMSConfig = core.VMSConfig
	Params    = core.Params
)

func NewTypedVMSRest(config *VMSConfig) (*rest.TypedVMSRest, error) {
	return rest.NewTypedVMSRest(config)
}

func NewUntypedVMSRest(config *VMSConfig) (*rest.UntypedVMSRest, error) {
	return rest.NewUntypedVMSRest(config)
}
