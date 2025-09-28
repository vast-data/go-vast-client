After configuration, you can use VMSRest in two ways: **untyped** (traditional) or **typed** (new).

## Untyped REST Client

The traditional approach uses `vast_client.VMSRest` with untyped parameters and responses:

```go
rest.Views.Create(...)
rest.Quotas.DeleteById(...)
rest.Volumes.Ensure(...)
```

## Typed REST Client

> **Note**: The typed REST client is still evolving and new features are being added regularly.

The typed approach uses `typed.VMSRest` with strongly-typed request and response structs:

```go
import "github.com/vast-data/go-vast-client/typed"

// Initialize typed client
typedRest, err := typed.NewTypedVMSRest(config)
if err != nil {
    log.Fatal(err)
}

// Use typed structs for requests and responses
quota := &typed.QuotaRequestBody{
    Name:      "my-quota",
    Path:      "/data",
    HardLimit: 1099511627776, // 1TB
}

result, err := typedRest.Quotas.Create(quota)
if err != nil {
    log.Fatal(err)
}
```

### Accessing Untyped Client from Typed

You can always access the underlying untyped client via the `Untyped` property:

```go
// Access untyped methods when needed
record, err := typedRest.Untyped.Quotas.GetWithContext(ctx, params)
```

### Benefits of Typed Client

- **Type Safety**: Compile-time checking of request/response structures
- **IDE Support**: Better autocomplete and documentation
- **Validation**: Automatic validation of required fields
- **Convenience**: No need to manually construct `Params` maps

## Standard Resource Methods

Both typed and untyped clients support the following standard methods for each resource:

- `List`
- `Get`
- `Delete`
- `Update`
- `Create`
- `Ensure`
- `EnsureByName`
- `GetById`
- `DeleteById`

For context-aware usage, the following variants are available:

- `ListWithContext`
- `GetWithContext`
- `DeleteWithContext`
- `UpdateWithContext`
- `CreateWithContext`
- `EnsureWithContext`
- `EnsureByNameWithContext`
- `GetByIdWithContext`
- `DeleteByIdWithContext`

These variants are especially useful when you need to:

- Set custom timeouts
- Cancel long-running requests
- Propagate tracing or logging via `context.Context`


## Example Usage

### Untyped Client
```go
// Using untyped client with Params
user, err := rest.Users.GetWithContext(ctx, vast_client.Params{"name": "admin"})
```

### Typed Client
```go
// Using typed client with structs
searchParams := &typed.UserSearchParams{Name: "admin"}
user, err := typedRest.Users.GetWithContext(ctx, searchParams)
```
