package untyped

import (
	"github.com/vast-data/go-vast-client/core"
)

// +apityped:details:GET=apitokens
// +apityped:upsert:POST=apitokens
type ApiToken struct {
	*core.VastResource
}
