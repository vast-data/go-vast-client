package expr

import "strings"

// Str is the singleton factory for string search field expressions.
//
//	type UserSearchParams struct {
//	    Name expr.StrField `json:"name,omitempty"`
//	}
//
//	UserSearchParams{Name: expr.Str.Exact("admin")}        // ?name=admin
//	UserSearchParams{Name: expr.Str.StartsWith("sys")}     // ?name__startswith=sys
//	UserSearchParams{Name: expr.Str.IContains("adm")}      // ?name__icontains=adm
//	UserSearchParams{Name: expr.Str.Regex(`^sys\d+`)}      // ?name__regex=^sys\d+
//	UserSearchParams{Name: expr.Str.In("alice", "bob")}    // ?name__in=alice,bob
//	UserSearchParams{Name: expr.Str.NotContains("test")}   // ?name__not_contains=test
var Str strFactory

type strFactory struct{}

func (strFactory) Exact(v string) StrField      { return exactField(v) }
func (strFactory) IExact(v string) StrField     { return sf("iexact", v) }
func (strFactory) Contains(v string) StrField   { return sf("contains", v) }
func (strFactory) IContains(v string) StrField  { return sf("icontains", v) }
func (strFactory) StartsWith(v string) StrField { return sf("startswith", v) }
func (strFactory) EndsWith(v string) StrField   { return sf("endswith", v) }
func (strFactory) Regex(v string) StrField      { return sf("regex", v) }
func (strFactory) IRegex(v string) StrField     { return sf("iregex", v) }
func (strFactory) In(vs ...string) StrField     { return sf("in", strings.Join(vs, ",")) }

// Negated variants (not_ prefix)
func (strFactory) NotExact(v string) StrField      { return sf("not_exact", v) }
func (strFactory) NotIExact(v string) StrField     { return sf("not_iexact", v) }
func (strFactory) NotContains(v string) StrField   { return sf("not_contains", v) }
func (strFactory) NotIContains(v string) StrField  { return sf("not_icontains", v) }
func (strFactory) NotStartsWith(v string) StrField { return sf("not_startswith", v) }
func (strFactory) NotEndsWith(v string) StrField   { return sf("not_endswith", v) }
func (strFactory) NotRegex(v string) StrField      { return sf("not_regex", v) }
func (strFactory) NotIRegex(v string) StrField     { return sf("not_iregex", v) }
func (strFactory) NotIn(vs ...string) StrField     { return sf("not_in", strings.Join(vs, ",")) }

func sf(suffix, value string) StrField {
	return exprField[string](suffixExpr{suffix: suffix, value: value})
}
