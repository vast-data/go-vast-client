package expr

import "strings"

// Str is a callable factory for string search field expressions.
// Calling it directly produces an exact-match field; methods produce lookups.
//
//	type UserSearchParams struct {
//	    Name expr.StrField `json:"name,omitempty"`
//	}
//
//	UserSearchParams{Name: expr.Str("admin")}              // ?name=admin
//	UserSearchParams{Name: expr.Str.StartsWith("sys")}     // ?name__startswith=sys
//	UserSearchParams{Name: expr.Str.IContains("adm")}      // ?name__icontains=adm
//	UserSearchParams{Name: expr.Str.Regex(`^sys\d+`)}      // ?name__regex=^sys\d+
//	UserSearchParams{Name: expr.Str.In("alice", "bob")}    // ?name__in=alice,bob
//	UserSearchParams{Name: expr.Str.NotContains("test")}   // ?name__not_contains=test
var Str = strOp(func(v string) StrField {
	return exactField(v)
})

// strOp is a named function type so it can both be called (exact match)
// and carry lookup methods.
type strOp func(string) StrField

func (strOp) Exact(v string) StrField      { return exactField(v) }
func (strOp) IExact(v string) StrField     { return sf("iexact", v) }
func (strOp) Contains(v string) StrField   { return sf("contains", v) }
func (strOp) IContains(v string) StrField  { return sf("icontains", v) }
func (strOp) StartsWith(v string) StrField { return sf("startswith", v) }
func (strOp) EndsWith(v string) StrField   { return sf("endswith", v) }
func (strOp) Regex(v string) StrField      { return sf("regex", v) }
func (strOp) IRegex(v string) StrField     { return sf("iregex", v) }
func (strOp) In(vs ...string) StrField     { return sf("in", strings.Join(vs, ",")) }

// Negated variants (not_ prefix)
func (strOp) NotExact(v string) StrField      { return sf("not_exact", v) }
func (strOp) NotIExact(v string) StrField     { return sf("not_iexact", v) }
func (strOp) NotContains(v string) StrField   { return sf("not_contains", v) }
func (strOp) NotIContains(v string) StrField  { return sf("not_icontains", v) }
func (strOp) NotStartsWith(v string) StrField { return sf("not_startswith", v) }
func (strOp) NotEndsWith(v string) StrField   { return sf("not_endswith", v) }
func (strOp) NotRegex(v string) StrField      { return sf("not_regex", v) }
func (strOp) NotIRegex(v string) StrField     { return sf("not_iregex", v) }
func (strOp) NotIn(vs ...string) StrField     { return sf("not_in", strings.Join(vs, ",")) }

func sf(suffix, value string) StrField {
	return exprField[string](suffixExpr{suffix: suffix, value: value})
}
