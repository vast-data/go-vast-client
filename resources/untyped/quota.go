package untyped

import (
	"github.com/vast-data/go-vast-client/core"
)

// +apityped:details:GET=quotas
// +apityped:upsert:POST=quotas
type Quota struct {
	*core.VastResource
}
