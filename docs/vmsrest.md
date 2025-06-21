After configuration, you use VMSRest to access API groups:

```go
rest.Views.Create(...)
rest.Quotas.DeleteById(...)
rest.Volumes.Ensure(...)
```

#### Standard Resource Methods

Each subresource supports the following standard methods:

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


##### Example Usage
```go
user, err := rest.User.GetWithContext(ctx, client.Params{"name": "admin"})
```
