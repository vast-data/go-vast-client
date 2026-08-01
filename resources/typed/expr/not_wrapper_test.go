package expr

import "testing"

type badQueryExpr struct{}

func (badQueryExpr) queryParam(string) (string, string) { return "unexpected-key", "value" }

func TestNotWrapperQueryParam_WithSuffix(t *testing.T) {
	wrapped := not_{expr: suffixExpr{suffix: "contains", value: "foo"}}
	key, val := wrapped.queryParam("name")
	if key != "name__not_contains" || val != "foo" {
		t.Fatalf("got %q=%q", key, val)
	}
}

func TestNotWrapperQueryParam_Fallback(t *testing.T) {
	wrapped := not_{expr: badQueryExpr{}}
	key, val := wrapped.queryParam("name")
	if key != "name__not_exact" || val != "value" {
		t.Fatalf("got %q=%q", key, val)
	}
}
