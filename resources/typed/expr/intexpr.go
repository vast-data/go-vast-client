package expr

import (
	"fmt"
	"strings"
)

// Int is the singleton factory for integer search field expressions.
//
//	type UserSearchParams struct {
//	    Uid expr.IntField `json:"uid,omitempty"`
//	}
//
//	UserSearchParams{Uid: expr.Int.Exact(42)}       // ?uid=42
//	UserSearchParams{Uid: expr.Int.GT(1000)}        // ?uid__gt=1000
//	UserSearchParams{Uid: expr.Int.GTE(1000)}       // ?uid__gte=1000
//	UserSearchParams{Uid: expr.Int.In(1, 2, 3)}     // ?uid__in=1,2,3  (PK fields only)
//	UserSearchParams{Uid: expr.Int.NotGTE(9999)}    // ?uid__not_gte=9999
var Int intFactory

type intFactory struct{}

func (intFactory) Exact(v int64) IntField { return exactField(v) }
func (intFactory) GT(v int64) IntField    { return inf("gt", v) }
func (intFactory) GTE(v int64) IntField   { return inf("gte", v) }
func (intFactory) LT(v int64) IntField    { return inf("lt", v) }
func (intFactory) LTE(v int64) IntField   { return inf("lte", v) }

// In is valid for primary-key columns (AutoField / BigAutoField) only.
func (intFactory) In(vs ...int64) IntField { return infSlice("in", vs) }

// Negated variants (not_ prefix)
func (intFactory) NotExact(v int64) IntField  { return inf("not_exact", v) }
func (intFactory) NotGT(v int64) IntField     { return inf("not_gt", v) }
func (intFactory) NotGTE(v int64) IntField    { return inf("not_gte", v) }
func (intFactory) NotLT(v int64) IntField     { return inf("not_lt", v) }
func (intFactory) NotLTE(v int64) IntField    { return inf("not_lte", v) }
func (intFactory) NotIn(vs ...int64) IntField { return infSlice("not_in", vs) }

func inf(suffix string, v int64) IntField {
	return exprField[int64](suffixExpr{suffix: suffix, value: fmt.Sprintf("%d", v)})
}

func infSlice(suffix string, vs []int64) IntField {
	parts := make([]string, len(vs))
	for i, v := range vs {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return exprField[int64](suffixExpr{suffix: suffix, value: strings.Join(parts, ",")})
}
