package expr_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/vast-data/go-vast-client/core"
	"github.com/vast-data/go-vast-client/resources/typed/expr"
)

// searchParams mirrors a typical generated *SearchParams struct.
type searchParams struct {
	Name     expr.StrField `json:"name,omitempty"`
	Guid     expr.StrField `json:"guid,omitempty"`
	ID       expr.IntField `json:"id,omitempty"`
	TenantID int64         `json:"tenant_id,omitempty"`
	RawData  core.Params   `json:"-"`
}

// ---- helpers ----

func toParams(t *testing.T, v any) core.Params {
	t.Helper()
	p, err := core.NewParamsFromStruct(v)
	if err != nil {
		t.Fatalf("NewParamsFromStruct: %v", err)
	}
	return p
}

func has(t *testing.T, p core.Params, key, want string) {
	t.Helper()
	raw, ok := p[key]
	if !ok {
		t.Errorf("missing %q in %v", key, p)
		return
	}
	if got := fmt.Sprintf("%v", raw); got != want {
		t.Errorf("%q: got %q want %q", key, got, want)
	}
}

func absent(t *testing.T, p core.Params, key string) {
	t.Helper()
	if _, ok := p[key]; ok {
		t.Errorf("unexpected key %q in %v", key, p)
	}
}

// ---- expr.Str tests ----

func TestStrCallExact(t *testing.T) {
	has(t, toParams(t, searchParams{Name: expr.Str("admin")}), "name", "admin")
}
func TestStrExact(t *testing.T) {
	has(t, toParams(t, searchParams{Name: expr.Str.Exact("admin")}), "name", "admin")
}
func TestStrIExact(t *testing.T) {
	has(t, toParams(t, searchParams{Name: expr.Str.IExact("Admin")}), "name__iexact", "Admin")
}
func TestStrContains(t *testing.T) {
	has(t, toParams(t, searchParams{Name: expr.Str.Contains("adm")}), "name__contains", "adm")
}
func TestStrIContains(t *testing.T) {
	has(t, toParams(t, searchParams{Name: expr.Str.IContains("adm")}), "name__icontains", "adm")
}
func TestStrStartsWith(t *testing.T) {
	has(t, toParams(t, searchParams{Name: expr.Str.StartsWith("sys")}), "name__startswith", "sys")
}
func TestStrEndsWith(t *testing.T) {
	has(t, toParams(t, searchParams{Name: expr.Str.EndsWith("vol")}), "name__endswith", "vol")
}
func TestStrRegex(t *testing.T) {
	has(t, toParams(t, searchParams{Name: expr.Str.Regex(`^sys`)}), "name__regex", "^sys")
}
func TestStrIRegex(t *testing.T) {
	has(t, toParams(t, searchParams{Name: expr.Str.IRegex(`^sys`)}), "name__iregex", "^sys")
}
func TestStrIn(t *testing.T) {
	has(t, toParams(t, searchParams{Name: expr.Str.In("alice", "bob")}), "name__in", "alice,bob")
}

func TestStrNotExact(t *testing.T) {
	has(t, toParams(t, searchParams{Name: expr.Str.NotExact("root")}), "name__not_exact", "root")
}
func TestStrNotContains(t *testing.T) {
	has(t, toParams(t, searchParams{Name: expr.Str.NotContains("test")}), "name__not_contains", "test")
}
func TestStrNotIContains(t *testing.T) {
	has(t, toParams(t, searchParams{Name: expr.Str.NotIContains("test")}), "name__not_icontains", "test")
}
func TestStrNotStartsWith(t *testing.T) {
	has(t, toParams(t, searchParams{Name: expr.Str.NotStartsWith("tmp")}), "name__not_startswith", "tmp")
}
func TestStrNotIn(t *testing.T) {
	has(t, toParams(t, searchParams{Name: expr.Str.NotIn("root", "nobody")}), "name__not_in", "root,nobody")
}
func TestStrNotRegex(t *testing.T) {
	has(t, toParams(t, searchParams{Name: expr.Str.NotRegex(`^tmp`)}), "name__not_regex", "^tmp")
}

// ---- expr.Int tests ----

func TestIntCallExact(t *testing.T) {
	has(t, toParams(t, searchParams{ID: expr.Int(42)}), "id", "42")
}
func TestIntExact(t *testing.T) {
	has(t, toParams(t, searchParams{ID: expr.Int.Exact(42)}), "id", "42")
}
func TestIntGT(t *testing.T) { has(t, toParams(t, searchParams{ID: expr.Int.GT(5)}), "id__gt", "5") }
func TestIntGTE(t *testing.T) {
	has(t, toParams(t, searchParams{ID: expr.Int.GTE(1000)}), "id__gte", "1000")
}
func TestIntLT(t *testing.T) {
	has(t, toParams(t, searchParams{ID: expr.Int.LT(100)}), "id__lt", "100")
}
func TestIntLTE(t *testing.T) {
	has(t, toParams(t, searchParams{ID: expr.Int.LTE(100)}), "id__lte", "100")
}
func TestIntIn(t *testing.T) {
	has(t, toParams(t, searchParams{ID: expr.Int.In(1, 2, 3)}), "id__in", "1,2,3")
}
func TestIntNotExact(t *testing.T) {
	has(t, toParams(t, searchParams{ID: expr.Int.NotExact(0)}), "id__not_exact", "0")
}
func TestIntNotGTE(t *testing.T) {
	has(t, toParams(t, searchParams{ID: expr.Int.NotGTE(100)}), "id__not_gte", "100")
}
func TestIntNotIn(t *testing.T) {
	has(t, toParams(t, searchParams{ID: expr.Int.NotIn(1, 2)}), "id__not_in", "1,2")
}
func TestIntNotGT(t *testing.T) {
	has(t, toParams(t, searchParams{ID: expr.Int.NotGT(5)}), "id__not_gt", "5")
}
func TestIntNotLT(t *testing.T) {
	has(t, toParams(t, searchParams{ID: expr.Int.NotLT(5)}), "id__not_lt", "5")
}
func TestIntNotLTE(t *testing.T) {
	has(t, toParams(t, searchParams{ID: expr.Int.NotLTE(5)}), "id__not_lte", "5")
}

func TestStrNotIExact(t *testing.T) {
	has(t, toParams(t, searchParams{Name: expr.Str.NotIExact("Admin")}), "name__not_iexact", "Admin")
}
func TestStrNotEndsWith(t *testing.T) {
	has(t, toParams(t, searchParams{Name: expr.Str.NotEndsWith("tmp")}), "name__not_endswith", "tmp")
}
func TestStrNotIRegex(t *testing.T) {
	has(t, toParams(t, searchParams{Name: expr.Str.NotIRegex(`^tmp`)}), "name__not_iregex", "^tmp")
}

func TestNotWrapperExactFallback(t *testing.T) {
	// not_ on an exact-match expression falls back to __not_exact
	field := expr.Str.NotExact("root")
	p := toParams(t, searchParams{Name: field})
	has(t, p, "name__not_exact", "root")
}


// ---- unset / mixed / query string ----

func TestUnsetFieldsOmitted(t *testing.T) {
	p := toParams(t, searchParams{TenantID: 7})
	absent(t, p, "name")
	absent(t, p, "id")
	has(t, p, "tenant_id", "7")
}

func TestMixedFields(t *testing.T) {
	p := toParams(t, searchParams{
		Name:     expr.Str.StartsWith("sys"),
		ID:       expr.Int.GT(100),
		TenantID: 3,
	})
	has(t, p, "name__startswith", "sys")
	has(t, p, "id__gt", "100")
	has(t, p, "tenant_id", "3")
	absent(t, p, "guid")
}

func TestQueryStringEncoding(t *testing.T) {
	p, _ := core.NewParamsFromStruct(searchParams{
		Name: expr.Str.IContains("foo"),
		ID:   expr.Int.In(1, 2),
	})
	parsed, err := url.ParseQuery(p.ToQuery())
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Get("name__icontains"); got != "foo" {
		t.Errorf("name__icontains: %q", got)
	}
	if got := parsed.Get("id__in"); got != "1,2" {
		t.Errorf("id__in: %q", got)
	}
}

// TestRawDataMergesWithTypedFields verifies that RawData entries (e.g. "fields"
// projection) are merged alongside typed expr fields, not replacing them.
// This covers the real-world pattern:
//
//	ViewSearchParams{Name: expr.Str("clusterA"), RawData: Params{"fields": "id,path"}}
//	=> ?name=clusterA&fields=id,path
func TestRawDataMergesWithTypedFields(t *testing.T) {
	p := toParams(t, searchParams{
		Name:    expr.Str("clusterA-source"),
		ID:      expr.Int(42),
		RawData: core.Params{"fields": "id,name,path"},
	})

	has(t, p, "name", "clusterA-source")
	has(t, p, "id", "42")
	has(t, p, "fields", "id,name,path")
}

// TestRawDataOverridesTypedField verifies that RawData wins on key conflict.
func TestRawDataOverridesTypedField(t *testing.T) {
	p := toParams(t, searchParams{
		Name:    expr.Str("original"),
		RawData: core.Params{"name": "override"},
	})

	has(t, p, "name", "override")
}

// TestRawDataOnlyNoTypedFields verifies backward-compat: RawData-only still works.
func TestRawDataOnlyNoTypedFields(t *testing.T) {
	p := toParams(t, searchParams{
		RawData: core.Params{"subsystem_name": "foo", "fields": "id"},
	})

	has(t, p, "subsystem_name", "foo")
	has(t, p, "fields", "id")
	absent(t, p, "name")
}

// TestTypedFieldsOnlyNoRawData verifies typed-only still works (regression guard).
func TestTypedFieldsOnlyNoRawData(t *testing.T) {
	p := toParams(t, searchParams{
		Name: expr.Str.StartsWith("prefix"),
		ID:   expr.Int.GT(100),
	})

	has(t, p, "name__startswith", "prefix")
	has(t, p, "id__gt", "100")
	absent(t, p, "fields")
}
