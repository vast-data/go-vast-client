package common

import (
	"fmt"
	"math"
)

// ToUint64 converts a non-negative int64 to uint64.
func ToUint64(n int64) (uint64, error) {
	if n < 0 {
		return 0, fmt.Errorf("value %d out of range for uint64", n)
	}
	return uint64(n), nil
}

// ToUint converts a non-negative int64 to uint.
func ToUint(n int64) (uint, error) {
	if n < 0 {
		return 0, fmt.Errorf("value %d out of range for uint", n)
	}
	return uint(n), nil
}

// Int64FromUint converts uint to int64 when the value fits.
func Int64FromUint(n uint) (int64, error) {
	if uint64(n) > uint64(math.MaxInt64) {
		return 0, fmt.Errorf("value %d out of range for int64", n)
	}
	return int64(n), nil
}

// ToUintFromUint64 converts uint64 to uint when the value fits.
func ToUintFromUint64(n uint64) (uint, error) {
	if n > uint64(math.MaxUint) {
		return 0, fmt.Errorf("value %d out of range for uint", n)
	}
	return uint(n), nil
}

func UintFromFloat64(n float64) (uint, error) {
	if n < 0 || n > float64(math.MaxUint) || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, fmt.Errorf("value %v out of range for uint", n)
	}
	if n != math.Trunc(n) {
		return 0, fmt.Errorf("value %v is not an integer", n)
	}
	return uint(n), nil
}
