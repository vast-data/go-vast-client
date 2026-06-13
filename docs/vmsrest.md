# REST Clients

After configuration, you can use one of two REST client types:

## Client Types

### Typed REST Client

The typed client provides strongly-typed structs for all requests and responses:

```go
import (
    client "github.com/vast-data/go-vast-client"
    "github.com/vast-data/go-vast-client/resources/typed"
    "github.com/vast-data/go-vast-client/resources/typed/expr"
)

// Initialize typed client
rest, err := client.NewTypedVMSRest(config)
if err != nil {
    log.Fatal(err)
}

// client.S() creates an exact-match string field
searchParams := &typed.QuotaSearchParams{
    Name: client.S("my-quota"),
}

body := &typed.QuotaRequestBody{
    Name:      "my-quota",
    Path:      "/data",
    HardLimit: 1099511627776, // 1TB
}

quota, err := rest.Quotas.Ensure(searchParams, body)
if err != nil {
    log.Fatal(err)
}
```

**Benefits:**
- **Type Safety**: Compile-time checking of request/response structures
- **IDE Support**: Better autocomplete and documentation
- **Clear Contracts**: Explicit field types and requirements
- **Reduced Errors**: Invalid field names caught at compile time

### Untyped REST Client

The untyped client uses flexible `map[string]any` for parameters and responses:

```go
import client "github.com/vast-data/go-vast-client"

// Initialize untyped client
rest, err := client.NewVMSRest(config)
if err != nil {
    log.Fatal(err)
}

// Use Params maps for requests
result, err := rest.Quotas.Create(client.Params{
    "name":       "my-quota",
    "path":       "/data",
    "hard_limit": 1099511627776, // 1TB
})
if err != nil {
    log.Fatal(err)
}
```

**Use Cases:**
- Dynamic scenarios where field names are not known at compile time
- Prototyping and experimentation
- Working with custom or undocumented API fields

### Accessing Untyped Client from Typed

If you're using the typed client but need untyped access for specific operations, you can access the underlying untyped client:

```go
rest, err := client.NewTypedVMSRest(config)

// Access untyped client when needed
untypedRest := rest.Untyped
record, err := untypedRest.Quotas.GetWithContext(ctx, client.Params{"name": "my-quota"})
```

## Standard Resource Methods

Both typed and untyped clients support standard CRUD methods for each resource (subject to API permissions):

### Basic Methods

- `List` / `ListWithContext` - List all resources
- `Get` / `GetWithContext` - Get a resource by search parameters
- `Create` / `CreateWithContext` - Create a new resource
- `Update` / `UpdateWithContext` - Update an existing resource
- `Delete` / `DeleteWithContext` - Delete a resource
- `Ensure` / `EnsureWithContext` - Create if doesn't exist, return if exists

### Context-Aware Methods

All methods have `WithContext` variants that accept a `context.Context` as the first parameter. These are useful when you need to:

- Set custom timeouts per request
- Cancel long-running operations
- Propagate request-scoped values (tracing, logging, etc.)

```go
// Example: Request with timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

view, err := rest.Views.GetWithContext(ctx, searchParams)
```


## Example Usage Comparison

### Creating a View

**Typed Client:**
```go
import (
    client "github.com/vast-data/go-vast-client"
    "github.com/vast-data/go-vast-client/resources/typed"
)

rest, _ := client.NewTypedVMSRest(config)

body := &typed.ViewRequestBody{
    Name:      "myview",
    Path:      "/myview",
    Protocols: &[]string{"NFS"},
    PolicyId:  1,
    CreateDir: true,
}

view, err := rest.Views.Create(body)
```

**Untyped Client:**
```go
import client "github.com/vast-data/go-vast-client"

rest, _ := client.NewVMSRest(config)

result, err := rest.Views.Create(client.Params{
    "name":       "myview",
    "path":       "/myview",
    "protocols":  []string{"NFS"},
    "policy_id":  1,
    "create_dir": true,
})
```

### Getting a User

**Typed Client:**
```go
// Exact match
user, err := rest.Users.Get(&typed.UserSearchParams{Name: client.S("admin")})

// Expression: name starts with "svc-"
users, err := rest.Users.List(&typed.UserSearchParams{Name: client.Str.StartsWith("svc-")})

// Expression: uid greater than 1000
users, err := rest.Users.List(&typed.UserSearchParams{Uid: client.Int.GT(1000)})
```

**Untyped Client:**
```go
user, err := rest.Users.Get(client.Params{"name": "admin"})
```

---

## Typed Search Expressions

`SearchParams` string and integer fields are typed as `expr.StrField` / `expr.IntField`.
This lets you pass Django-style lookup expressions directly — no `RawData` needed for common
filters:

```go
// ?name__startswith=prod
rest.Views.List(&typed.ViewSearchParams{
    Name: client.Str.StartsWith("prod"),
})

// ?name__in=alice,bob
rest.Users.List(&typed.UserSearchParams{
    Name: client.Str.In("alice", "bob"),
})

// ?uid__gte=500
rest.Users.List(&typed.UserSearchParams{
    Uid: client.Int.GTE(500),
})

// ?name__not_contains=test&tenant_id__gt=0
rest.Snapshots.List(&typed.SnapshotSearchParams{
    Name:     client.Str.NotContains("test"),
    TenantId: client.Int.GT(0),
})
```

See [Typed Search Expressions](typed-expressions.md) for the full reference.
