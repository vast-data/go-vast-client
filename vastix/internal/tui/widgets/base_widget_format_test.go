package widgets

import (
	"reflect"
	"testing"
)

func TestFormatSliceForDisplayPrimitives(t *testing.T) {
	got := formatSliceForDisplay(reflect.ValueOf([]interface{}{"NFS", "S3"}))
	if got != "[NFS, S3]" {
		t.Fatalf("got %q, want [NFS, S3]", got)
	}
}

func TestFormatSliceForDisplayEmpty(t *testing.T) {
	got := formatSliceForDisplay(reflect.ValueOf([]string{}))
	if got != "[]" {
		t.Fatalf("got %q, want []", got)
	}
}

func TestFormatSliceForDisplayIPRanges(t *testing.T) {
	// API JSON shape: nested []interface{} with start/end pairs
	got := formatSliceForDisplay(reflect.ValueOf([]interface{}{
		[]interface{}{"10.0.0.1", "10.0.0.10"},
	}))
	if got != "[10.0.0.1 - 10.0.0.10]" {
		t.Fatalf("got %q, want [10.0.0.1 - 10.0.0.10]", got)
	}

	got = formatSliceForDisplay(reflect.ValueOf([][]string{
		{"192.168.1.1", "192.168.1.50"},
		{"1.1.1.1"},
	}))
	if got != "[192.168.1.1 - 192.168.1.50, 1.1.1.1]" {
		t.Fatalf("got %q, want [192.168.1.1 - 192.168.1.50, 1.1.1.1]", got)
	}
}

func TestFormatSliceForDisplayComplex(t *testing.T) {
	got := formatSliceForDisplay(reflect.ValueOf([]map[string]interface{}{{"a": 1}}))
	if got != "[1 items]" {
		t.Fatalf("got %q, want [1 items]", got)
	}
}
