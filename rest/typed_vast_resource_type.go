package rest

import (
	"github.com/vast-data/go-vast-client/resources/typed"
)

type TypedVastResourceType interface {
	typed.Quota |
		typed.ApiToken |
		typed.Version |
		typed.View |
		typed.VipPool |
		typed.User
}
