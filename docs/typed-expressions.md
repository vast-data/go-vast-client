# Typed Search Expressions

The typed client supports **Django-style query expressions** on `SearchParams` fields.
String and integer fields in every `*SearchParams` struct are typed as `client.StrField` or
`client.IntField` instead of plain `string` / `int64`. This allows you to pass either an
exact value or a rich lookup expression without falling back to `RawData`.

All expression helpers are exported directly from the top-level `client` package — no extra
import needed.

---

## Quick Reference

| Factory | Type | Generated query parameter |
|---|---|---|
| `expr.Str("foo")` | `StrField` | `name=foo` |
| `expr.Str.Exact("foo")` | `StrField` | `name=foo` |
| `expr.Str.Contains("foo")` | `StrField` | `name__contains=foo` |
| `expr.Str.IContains("foo")` | `StrField` | `name__icontains=foo` |
| `expr.Str.StartsWith("foo")` | `StrField` | `name__startswith=foo` |
| `expr.Str.EndsWith("foo")` | `StrField` | `name__endswith=foo` |
| `expr.Str.Regex(` `` `^sys\d+` `` `)` | `StrField` | `name__regex=^sys\d+` |
| `expr.Str.IRegex(` `` `^sys\d+` `` `)` | `StrField` | `name__iregex=^sys\d+` |
| `expr.Str.In("a", "b")` | `StrField` | `name__in=a,b` |
| `expr.Str.NotContains("foo")` | `StrField` | `name__not_contains=foo` |
| `expr.Str.NotIn("a", "b")` | `StrField` | `name__not_in=a,b` |
| `expr.Int(42)` | `IntField` | `uid=42` |
| `expr.Int.Exact(42)` | `IntField` | `uid=42` |
| `expr.Int.GT(100)` | `IntField` | `uid__gt=100` |
| `expr.Int.GTE(100)` | `IntField` | `uid__gte=100` |
| `expr.Int.LT(100)` | `IntField` | `uid__lt=100` |
| `expr.Int.LTE(100)` | `IntField` | `uid__lte=100` |
| `expr.Int.In(1, 2, 3)` | `IntField` | `uid__in=1,2,3` |
| `expr.Int.NotGTE(9999)` | `IntField` | `uid__not_gte=9999` |

> Negated variants exist for all expressions: `NotExact`, `NotContains`, `NotStartsWith`, etc.

---

## Exact Match

Call `expr.Str` / `expr.Int` directly for an exact-match filter:

```go
// GET /users/?name=admin
user, err := rest.Users.Get(&typed.UserSearchParams{
    Name: expr.Str("admin"),
})

// GET /users/?uid=9999
user, err := rest.Users.Get(&typed.UserSearchParams{
    Uid: expr.Int(9999),
})
```

---

## String Expressions

```go
// GET /snapshots/?name__startswith=go-client
snapshots, err := rest.Snapshots.List(&typed.SnapshotSearchParams{
    Name: expr.Str.StartsWith("go-client"),
})

// GET /views/?path__contains=production
views, err := rest.Views.List(&typed.ViewSearchParams{
    Path: expr.Str.Contains("production"),
})

// GET /users/?name__in=alice,bob,carol
users, err := rest.Users.List(&typed.UserSearchParams{
    Name: expr.Str.In("alice", "bob", "carol"),
})

// GET /protectedpaths/?name__endswith=b816a408a6
path, err := rest.ProtectedPaths.Get(&typed.ProtectedPathSearchParams{
    Name: expr.Str.EndsWith("b816a408a6"),
})

// GET /users/?name__icontains=admin  (case-insensitive)
users, err := rest.Users.List(&typed.UserSearchParams{
    Name: expr.Str.IContains("admin"),
})

// GET /views/?name__regex=^prod-\d+
views, err := rest.Views.List(&typed.ViewSearchParams{
    Name: expr.Str.Regex(`^prod-\d+`),
})
```

---

## Integer Expressions

```go
// GET /users/?uid__gt=1000
users, err := rest.Users.List(&typed.UserSearchParams{
    Uid: expr.Int.GT(1000),
})

// GET /users/?uid__gte=500&uid__lte=999  (range)
users, err := rest.Users.List(&typed.UserSearchParams{
    Uid: expr.Int.GTE(500),
    // combine multiple fields freely
})

// GET /users/?uid__in=1,2,3  (primary-key lookup)
users, err := rest.Users.List(&typed.UserSearchParams{
    Uid: expr.Int.In(1, 2, 3),
})

// GET /quotas/?tenant_id__not_in=5,6
quotas, err := rest.Quotas.List(&typed.QuotaSearchParams{
    TenantId: expr.Int.NotIn(5, 6),
})
```

---

## Combining Multiple Fields

All `SearchParams` fields are independent — set as many as needed:

```go
// GET /vippools/?name__startswith=prod&tenant_id__gte=10
pools, err := rest.VipPools.List(&typed.VipPoolSearchParams{
    Name:     expr.Str.StartsWith("prod"),
    TenantId: expr.Int.GTE(10),
})
```

---

## Negation

Every expression has a `Not*` counterpart:

```go
// GET /users/?name__not_contains=test
users, err := rest.Users.List(&typed.UserSearchParams{
    Name: expr.Str.NotContains("test"),
})

// GET /users/?uid__not_in=0,1
users, err := rest.Users.List(&typed.UserSearchParams{
    Uid: expr.Int.NotIn(0, 1),
})
```

---

## RawData Fallback

If a backend filter has no corresponding typed field (e.g., a newly added parameter not yet
regenerated), use `RawData`:

```go
users, err := rest.Users.List(&typed.UserSearchParams{
    Name:    expr.Str.StartsWith("svc"),
    RawData: client.Params{"some_new_filter": "value"},
})
```

Typed fields and `RawData` are merged before the request is sent.
