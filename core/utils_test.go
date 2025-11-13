package core

import (
	"testing"
)

func TestToBool(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
		hasError bool
	}{
		// Boolean inputs
		{"bool true", true, true, false},
		{"bool false", false, false, false},

		// String inputs - truthy
		{"string 'true'", "true", true, false},
		{"string 'TRUE'", "TRUE", true, false},
		{"string '1'", "1", true, false},
		{"string 'yes'", "yes", true, false},
		{"string 'YES'", "YES", true, false},
		{"string 'y'", "y", true, false},
		{"string 'Y'", "Y", true, false},
		{"string 'on'", "on", true, false},
		{"string 'ON'", "ON", true, false},
		{"string ' true '", " true ", true, false},

		// String inputs - falsy
		{"string 'false'", "false", false, false},
		{"string 'FALSE'", "FALSE", false, false},
		{"string '0'", "0", false, false},
		{"string 'no'", "no", false, false},
		{"string 'NO'", "NO", false, false},
		{"string 'n'", "n", false, false},
		{"string 'N'", "N", false, false},
		{"string 'off'", "off", false, false},
		{"string 'OFF'", "OFF", false, false},
		{"string empty", "", false, false},
		{"string ' false '", " false ", false, false},

		// Integer inputs
		{"int 0", 0, false, false},
		{"int 1", 1, true, false},
		{"int -1", -1, true, false},
		{"int64 0", int64(0), false, false},
		{"int64 1", int64(1), true, false},
		{"int32 42", int32(42), true, false},

		// Float inputs
		{"float64 0.0", 0.0, false, false},
		{"float64 1.0", 1.0, true, false},
		{"float64 0.1", 0.1, true, false},
		{"float32 0.0", float32(0.0), false, false},

		// Unsigned integer inputs
		{"uint 0", uint(0), false, false},
		{"uint 1", uint(1), true, false},
		{"uint64 5", uint64(5), true, false},

		// Error cases
		{"string invalid", "invalid", false, true},
		{"string maybe", "maybe", false, true},
		{"struct", struct{}{}, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ToBool(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

