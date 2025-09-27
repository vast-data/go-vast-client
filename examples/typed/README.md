# Typed Examples

This directory contains examples using the new typed VAST Go client API.

## Features

- Strongly-typed structs for requests and responses
- Compile-time type checking and validation
- Auto-completion and IntelliSense support
- Automatic validation of required fields
- Read-only resource protection
- Self-documenting code

## Example Structure

Each example typically follows this pattern:

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    client "github.com/vast-data/go-vast-client"
    "github.com/vast-data/go-vast-client/typed"
)

func main() {
    ctx := context.Background()
    config := &client.VMSConfig{
        Host:     "your-vast-ip",
        Username: "admin",
        Password: "password", 
    }
    
    // Create typed client
    typedClient, err := typed.NewTypedVMSRest(config)
    if err != nil {
        log.Fatalf("Failed to create typed client: %v", err)
    }
    typedClient.SetCtx(ctx)
    
    // Use typed resource
    resourceClient := typedClient.ResourceName
    
    // List with typed search parameters
    searchParams := &typed.ResourceSearchParams{
        Field1: "value1",
        Field2: 123,
    }
    items, err := resourceClient.List(searchParams)
    
    // Create with typed create body
    createBody := &typed.ResourceCreateBody{
        Name:  "example",
        Field: "value",
    }
    newItem, err := resourceClient.Create(createBody)
    
    // Update with typed create body
    updateBody := &typed.ResourceCreateBody{
        Name:  "updated-example", 
        Field: "new-value",
    }
    updatedItem, err := resourceClient.Update(newItem.Id, updateBody)
    
    // Delete with typed search parameters
    deleteParams := &typed.ResourceSearchParams{
        Name: "updated-example",
    }
    err = resourceClient.Delete(deleteParams)
}
```

## Available Examples

### Full CRUD Resources
- **quota/** - Type-safe quota operations with create, update, delete
- **view/** - Type-safe view management with NFS protocols  
- **vippool/** - Type-safe VIP pool configuration

### Read-Only Resources
- **version/** - Read-only version resource (demonstrates read-only protection)

### General Examples
- **basic-usage/** - Overview of typed client features and capabilities
- **rest-demo/** - General typed client demonstration

## Resource Types

### Full CRUD Resources
These resources support all operations (Create, Read, Update, Delete):
- `Quota` - Quota management
- `View` - View management
- `VipPool` - VIP pool management

Methods available:
- `Get(searchParams)` / `GetWithContext(ctx, searchParams)`
- `GetById(id)` / `GetByIdWithContext(ctx, id)`
- `List(searchParams)` / `ListWithContext(ctx, searchParams)`
- `Create(createBody)` / `CreateWithContext(ctx, createBody)`
- `Update(id, createBody)` / `UpdateWithContext(ctx, id, createBody)`
- `Delete(searchParams)` / `DeleteWithContext(ctx, searchParams)`
- `DeleteById(id)` / `DeleteByIdWithContext(ctx, id)`
- `Exists(searchParams)` / `ExistsWithContext(ctx, searchParams)`
- `MustExists(searchParams)` / `MustExistsWithContext(ctx, searchParams)`

### Read-Only Resources
These resources only support read operations:
- `Version` - Version information

Methods available:
- `Get(searchParams)` / `GetWithContext(ctx, searchParams)`
- `GetById(id)` / `GetByIdWithContext(ctx, id)`
- `List(searchParams)` / `ListWithContext(ctx, searchParams)`
- `Exists(searchParams)` / `ExistsWithContext(ctx, searchParams)`
- `MustExists(searchParams)` / `MustExistsWithContext(ctx, searchParams)`

## Type Structure

### Search Parameters
Used for filtering and searching resources:
```go
type QuotaSearchParams struct {
    Name     string `json:"name,omitempty"`
    Path     string `json:"path,omitempty"`
    TenantId int64  `json:"tenant_id,omitempty"`
    Guid     string `json:"guid,omitempty"`
    // ... other searchable fields
}
```

### Create Body
Used for creating and updating resources:
```go
type QuotaCreateBody struct {
    Name      string `json:"name,omitempty" required:"true"`
    Path      string `json:"path,omitempty" required:"true"`
    TenantId  int64  `json:"tenant_id,omitempty" required:"true"`
    HardLimit int64  `json:"hard_limit,omitempty" required:"false"`
    // ... other fields
}
```

### Model (Response)
Represents the complete resource data:
```go
type QuotaModel struct {
    Id        int64  `json:"id,omitempty"`
    Name      string `json:"name,omitempty"`
    Path      string `json:"path,omitempty"`
    TenantId  int64  `json:"tenant_id,omitempty"`
    HardLimit int64  `json:"hard_limit,omitempty"`
    Created   string `json:"created,omitempty"`
    // ... other fields including nested structs
}
```

## Key Benefits

### 1. Type Safety
```go
// Compile-time error if field doesn't exist
createBody.NonExistentField = "value" // ❌ Compile error

// Correct usage
createBody.Name = "quota-name" // ✅ Type-safe
```

### 2. IDE Support
- Auto-completion for all fields
- Inline documentation
- Go to definition support
- Refactoring support

### 3. Read-Only Protection
```go
versionClient := typedClient.Versions

// These methods don't exist for read-only resources
versionClient.Create(...)  // ❌ Compile error
versionClient.Update(...)  // ❌ Compile error  
versionClient.Delete(...)  // ❌ Compile error

// Only read operations are available
versions, err := versionClient.List(...) // ✅ Works
```

### 4. Validation
Required fields are automatically validated:
```go
type QuotaCreateBody struct {
    Name string `json:"name,omitempty" required:"true"`  // Required
    Path string `json:"path,omitempty" required:"false"` // Optional
}
```

### 5. Self-Documenting
```go
type QuotaModel struct {
    HardLimit int64 `json:"hard_limit,omitempty" doc:"Maximum storage limit in bytes"`
}
```

## Error Handling

```go
quota, err := quotaClient.Create(createBody)
if err != nil {
    log.Printf("Failed to create quota: %v", err)
    return
}
```

## Convenience Methods

### Setting Context
The typed client provides a shorthand method for setting context:

```go
// Shorthand method (recommended)
typedClient.SetCtx(ctx)

// Equivalent to:
typedClient.Untyped.SetCtx(ctx)
```

### Accessing Session
The typed client provides a shorthand method for accessing the REST session:

```go
// Shorthand method (recommended)
session := typedClient.GetSession()

// Equivalent to:
session := typedClient.Untyped.Session
```

## Context Support

All methods have context variants for timeout and cancellation:
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

quota, err := quotaClient.CreateWithContext(ctx, createBody)
```

## Accessing Untyped Client

For resources without typed support:
```go
// Access untyped client for any resource
users, err := typedClient.Untyped.Users.List(client.Params{})
```

## Best Practices

1. **Use context variants** for production code
2. **Handle errors appropriately** - don't ignore them
3. **Use typed search parameters** instead of empty structs
4. **Leverage IDE auto-completion** for field discovery
5. **Check required fields** using struct tags
6. **Use read-only resources** when you only need to read data
