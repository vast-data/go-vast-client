package typed

import (
	vast_client "github.com/vast-data/go-vast-client"
)

// VMSRest provides typed access to VAST resources
type VMSRest struct {
	// Untyped provides access to the underlying untyped VMSRest client when needed
	Untyped *vast_client.VMSRest

	// Typed resources - only resources with apibuilder markers are included
	Quotas *Quota
	Views *View

}

// NewTypedVMSRest creates a new typed VMSRest client from configuration
func NewTypedVMSRest(config *vast_client.VMSConfig) (*VMSRest, error) {
	rawClient, err := vast_client.NewVMSRest(config)
	if err != nil {
		return nil, err
	}

	typedRest := &VMSRest{
		Untyped: rawClient,
	}

	// Initialize typed resources
	typedRest.Quotas = &Quota{Untyped: rawClient}
	typedRest.Views = &View{Untyped: rawClient}


	return typedRest, nil
}
