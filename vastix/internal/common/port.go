package common

import (
	"fmt"
	"math"
)

// ToPort converts an integer to a uint16 TCP/UDP port number, returning an
// error if it is outside the valid range [0, math.MaxUint16].
//
// A bare uint16(n) conversion silently truncates or wraps out-of-range values
// (e.g. uint16(65536) == 0), which gosec flags as G115. ToPort is the single
// checked conversion used wherever the app turns an integer into a port.
func ToPort(n int64) (uint16, error) {
	if n < 0 || n > math.MaxUint16 {
		return 0, fmt.Errorf("port %d out of range [0, %d]", n, math.MaxUint16)
	}
	return uint16(n), nil
}
