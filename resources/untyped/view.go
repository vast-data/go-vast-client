package untyped

import (
	"github.com/vast-data/go-vast-client/core"
)

// +apityped:details:GET=views
// +apityped:upsert:POST=views
type View struct {
	*core.VastResource
}
