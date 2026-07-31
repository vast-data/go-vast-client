package expr

import (
	"fmt"
	"strings"
)

// Int is a callable factory for integer search field expressions.
// Calling it directly produces an exact-match field; methods produce lookups.
//
//	type UserSearchParams struct {
//	    Uid expr.IntField `json:"uid,omitempty"`
//	}
//
//	UserSearchParams{Uid: expr.Int(42)}             // ?uid=42
//	UserSearchParams{Uid: expr.Int.GT(1000)}        // ?uid__gt=1000
//	UserSearchParams{Uid: expr.Int.GTE(1000)}       // ?uid__gte=1000
//	UserSearchParams{Uid: expr.Int.In(1, 2, 3)}     // ?uid__in=1,2,3  (PK fields only)
//	UserSearchParams{Uid: expr.Int.NotGTE(9999)}    // ?uid__not_gte=9999
var Int = intOp(func(v int64) IntField {
	return exactField(v)
})

// intOp is a named function type so it can both be called (exact match)
// and carry lookup methods.
type intOp func(int64) IntField

func (intOp) Exact(v int64) IntField { return exactField(v) }
func (intOp) GT(v int64) IntField    { return inf("gt", v) }
func (intOp) GTE(v int64) IntField   { return inf("gte", v) }
func (intOp) LT(v int64) IntField    { return inf("lt", v) }
func (intOp) LTE(v int64) IntField   { return inf("lte", v) }

// In is valid for primary-key columns (AutoField / BigAutoField) only.
func (intOp) In(vs ...int64) IntField { return infSlice("in", vs) }

// Negated variants (not_ prefix)
func (intOp) NotExact(v int64) IntField  { return inf("not_exact", v) }
func (intOp) NotGT(v int64) IntField     { return inf("not_gt", v) }
func (intOp) NotGTE(v int64) IntField    { return inf("not_gte", v) }
func (intOp) NotLT(v int64) IntField     { return inf("not_lt", v) }
func (intOp) NotLTE(v int64) IntField    { return inf("not_lte", v) }
func (intOp) NotIn(vs ...int64) IntField { return infSlice("not_in", vs) }

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
