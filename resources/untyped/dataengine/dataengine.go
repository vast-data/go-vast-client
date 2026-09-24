package dataengine

import "github.com/vast-data/go-vast-client/core"

// DataEngine maps to /data-engine (provision / status / delete).
type DataEngine struct {
	*core.VastResource
}
