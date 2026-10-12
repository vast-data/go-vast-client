package dataengine

import "github.com/vast-data/go-vast-client/core"

// DataEngineTopic is named this way because the VMS client already has a Topic type.
// The parser keys resources by type name, so the two cannot share the name Topic.
type DataEngineTopic struct {
	*core.VastResource
}
