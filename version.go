package vast_client

import (
	_ "embed"
	"strings"
)

//go:embed version
var clientVersion string

func ClientVersion() string {
	return strings.TrimSpace(clientVersion)
}
