package vastix

import (
	_ "embed"
	"strings"
)

//go:embed version
var clientVersion string

func AppVersion() string {
	return strings.TrimSpace(clientVersion)
}
