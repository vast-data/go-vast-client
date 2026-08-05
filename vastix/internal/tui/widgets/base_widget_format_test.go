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

func TestFormatSliceForDisplayComplex(t *testing.T) {
	got := formatSliceForDisplay(reflect.ValueOf([]map[string]interface{}{{"a": 1}}))
	if got != "[1 items]" {
		t.Fatalf("got %q, want [1 items]", got)
	}
}
