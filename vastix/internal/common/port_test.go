package common

import (
	"math"
	"testing"
)

// TestToPort is table-driven and covers four categories: positive/valid,
// negative, boundary (the off-by-one edges of the range), and corner/overflow
// (values a bare uint16(n) conversion would silently truncate or wrap).
func TestToPort(t *testing.T) {
	tests := []struct {
		name    string
		in      int64
		want    uint16
		wantErr bool
	}{
		// ── positive / valid ────────────────────────────────────────────
		{"zero", 0, 0, false},
		{"one", 1, 1, false},
		{"http", 80, 80, false},
		{"https", 443, 443, false},
		{"alt-http", 8080, 8080, false},
		{"wireguard-base", 51820, 51820, false},
		{"max-valid", 65535, 65535, false},

		// ── negative ────────────────────────────────────────────────────
		{"negative-one", -1, 0, true},
		{"negative-large", -51820, 0, true},
		{"min-int64", math.MinInt64, 0, true},

		// ── boundary (edges of [0, 65535]) ──────────────────────────────
		{"lower-edge-valid", 0, 0, false},
		{"lower-edge-invalid", -1, 0, true},
		{"upper-edge-valid", 65535, 65535, false},
		{"upper-edge-invalid", 65536, 0, true},

		// ── corner / overflow (bare uint16(n) would truncate/wrap) ──────
		{"wraps-to-zero", 65536, 0, true}, // uint16(65536) == 0
		{"seventy-thousand", 70000, 0, true},
		{"hundred-thousand", 100000, 0, true},
		{"max-uint16-plus-one", math.MaxUint16 + 1, 0, true},
		{"max-int64", math.MaxInt64, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToPort(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ToPort(%d): expected an error, got nil (result %d)", tt.in, got)
				}
				if got != 0 {
					t.Errorf("ToPort(%d): on error want result 0, got %d", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ToPort(%d): unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("ToPort(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

// TestToPort_NoAllocs enforces that the success path allocates nothing, so the
// zero-alloc property is a guarantee rather than an observation.
func TestToPort_NoAllocs(t *testing.T) {
	if n := testing.AllocsPerRun(1000, func() {
		_, _ = ToPort(443)
	}); n != 0 {
		t.Errorf("ToPort success path allocates %v times/op, want 0", n)
	}
}

// BenchmarkToPort measures the hot (valid) path. Expect sub-nanosecond,
// 0 B/op, 0 allocs/op — the function is trivially inlinable.
func BenchmarkToPort(b *testing.B) {
	for b.Loop() {
		_, _ = ToPort(443)
	}
}

// BenchmarkToPort_Invalid measures the error path, confirming the only cost
// (the fmt.Errorf allocation) is confined to failures.
func BenchmarkToPort_Invalid(b *testing.B) {
	for b.Loop() {
		_, _ = ToPort(70000)
	}
}
