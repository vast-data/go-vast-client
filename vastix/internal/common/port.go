package common

// ToPort converts an integer to a uint16 TCP/UDP port number, returning an
// error if it is outside the valid range [0, math.MaxUint16].
//
// A bare uint16(n) conversion silently truncates or wraps out-of-range values
// (e.g. uint16(65536) == 0), which gosec flags as G115. ToPort is the single
// checked conversion used wherever the app turns an integer into a port.
func ToPort(n int64) (uint16, error) {
	// NOTE: range validation is added in the follow-up fix commit. This naive
	// body intentionally reproduces the shipped bug so that TestToPort fails
	// first (TDD red), then passes once the check is in place (green).
	return uint16(n), nil
}
