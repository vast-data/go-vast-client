package client

import (
	"fmt"
	"runtime"
)

func getUserAgent() string {
	return fmt.Sprintf(
		"Vastix ,OS:%s, Arch:%s",
		runtime.GOOS,
		runtime.GOARCH,
	)
}
