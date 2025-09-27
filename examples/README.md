# VAST Go Client Examples

This directory contains examples demonstrating how to use the VAST Go client library.

## Directory Structure

### `untyped/`
Contains examples using the traditional untyped client API. These examples use:
- `client.Params` for request parameters
- `Record` and `RecordSet` for responses
- Manual type conversion and validation
- String-based field access

### `typed/`
Contains examples using the new typed client API. These examples use:
- Strongly-typed structs for requests and responses
- Compile-time type checking
- Auto-completion and IntelliSense support
- Automatic validation of required fields
- Read-only resource protection

## Getting Started

### Prerequisites
- Go 1.19 or later
- Access to a VAST cluster
- Valid credentials (username/password)

### Configuration
Update the connection details in each example:
```go
config := &client.VMSConfig{
    Host:     "your-vast-cluster-ip",
    Username: "your-username", 
    Password: "your-password",
}
```

## Available Examples

### Untyped Examples
- `quota/` - Quota management (create, update, delete)
- `view/` - View management with NFS protocols
- `vippool/` - VIP pool configuration
- `version/` - Version information retrieval
- `user/` - User management
- `apitoken/` - API token operations
- `blockhosts/` - Block host management
- And many more...

### Typed Examples
- `basic-usage/` - Overview of typed client features
- `quota/` - Type-safe quota operations
- `view/` - Type-safe view management
- `vippool/` - Type-safe VIP pool operations
- `version/` - Read-only version resource (demonstrates read-only protection)
- `rest-demo/` - General typed client demonstration

## Key Differences

### Untyped API
```go
// Create with untyped parameters
result, err := rest.Quotas.Create(client.Params{
    "name":       "myquota",
    "path":       "/myview", 
    "tenant_id":  1,
    "hard_limit": 1024,
})

// Manual type conversion
var quota QuotaContainer
err = result.Fill(&quota)
```

### Typed API
```go
// Create with typed struct
createBody := &typed.QuotaCreateBody{
    Name:      "myquota",
    Path:      "/myview",
    TenantId:  1,
    HardLimit: 1024,
}
quota, err := quotaClient.Create(createBody)
// quota is already properly typed!
```

## Benefits of Typed API

1. **Type Safety**: Compile-time checking prevents runtime errors
2. **IDE Support**: Auto-completion and IntelliSense 
3. **Documentation**: Self-documenting with typed structs
4. **Validation**: Automatic validation of required fields
5. **Read-only Protection**: Compiler prevents modification of read-only resources
6. **Maintainability**: Easier to refactor and maintain code

## Running Examples

```bash
# Run an untyped example
cd examples/untyped/quota
go run main.go

# Run a typed example  
cd examples/typed/quota
go run main.go
```

## Migration Guide

To migrate from untyped to typed API:

1. Replace `client.NewVMSRest()` with `typed.NewTypedVMSRest()`
2. Use typed structs instead of `client.Params`
3. Access resources via typed client (e.g., `typedClient.Quotas`)
4. Use typed response structs instead of manual `Fill()` operations
5. For resources without typed support, use `typedClient.Untyped`

## Contributing

When adding new examples:
1. Add untyped version to `untyped/` directory
2. If the resource has apibuilder markers, add typed version to `typed/` directory
3. Include comprehensive error handling
4. Add comments explaining the operations
5. Follow the established naming conventions
