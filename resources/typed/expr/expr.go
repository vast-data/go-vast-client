// Package expr provides typed query field expressions for the VMS typed REST API.
//
//	expr.Str  — singleton factory for string search fields
//	expr.Int  — singleton factory for integer search fields
//	expr.StrField — field type to use in *SearchParams structs for string params
//	expr.IntField — field type to use in *SearchParams structs for integer params
//
// Usage:
//
//	type UserSearchParams struct {
//	    Name expr.StrField `json:"name,omitempty"`
//	    Uid  expr.IntField `json:"uid,omitempty"`
//	}
//
//	UserSearchParams{Name: expr.Str.StartsWith("sys")}   // ?name__startswith=sys
//	UserSearchParams{Uid:  expr.Int.GT(1000)}            // ?uid__gt=1000
package expr

import "strings"

// queryExpr is the private interface for all expression types.
type queryExpr interface {
	queryParam(field string) (key string, value string)
}

// suffixExpr is the internal implementation used by all Str / Int factory methods.
type suffixExpr struct {
	suffix string
	value  string
}

func (e suffixExpr) queryParam(field string) (string, string) {
	return field + "__" + e.suffix, e.value
}

// not_ wraps any queryExpr and negates it by prepending "not_" to the lookup suffix.
type not_ struct{ expr queryExpr }

func (e not_) queryParam(field string) (string, string) {
	key, val := e.expr.queryParam(field)
	prefix := field + "__"
	if after, ok := strings.CutPrefix(key, prefix); ok {
		return prefix + "not_" + after, val
	}
	return field + "__not_exact", val
}
