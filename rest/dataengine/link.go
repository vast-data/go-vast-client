package dataengine

import (
	_ "unsafe"

	"github.com/vast-data/go-vast-client/core"
)

// setUpdateMethod aliases core.setUpdateMethod via go:linkname so this package
// can configure update verbs without exporting a public VastResource API.
//
//go:linkname setUpdateMethod github.com/vast-data/go-vast-client/core.setUpdateMethod
func setUpdateMethod(e *core.VastResource, method string)
