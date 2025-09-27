# Untyped Examples

This directory contains examples using the traditional untyped VAST Go client API.

## Features

- Uses `client.Params` for request parameters
- Returns `Record` and `RecordSet` objects
- Requires manual type conversion using `Fill()`
- String-based field access
- Runtime type checking

## Example Structure

Each example typically follows this pattern:

```go
package main

import (
    "fmt"
    client "github.com/vast-data/go-vast-client"
)

type MyContainer struct {
    ID   int64  `json:"id"`
    Name string `json:"name"`
    // ... other fields
}

func main() {
    config := &client.VMSConfig{
        Host:     "your-vast-ip",
        Username: "admin", 
        Password: "password",
    }
    
    rest, err := client.NewVMSRest(config)
    if err != nil {
        panic(err)
    }
    
    // Create operation
    result, err := rest.ResourceName.Create(client.Params{
        "field1": "value1",
        "field2": 123,
    })
    
    // Manual type conversion
    var container MyContainer
    err = result.Fill(&container)
    
    // Update operation
    rest.ResourceName.Update(container.ID, client.Params{
        "field1": "new_value",
    })
    
    // Delete operation
    rest.ResourceName.Delete(client.Params{
        "name": "resource_name",
    }, nil)
}
```

## Available Examples

- **apitoken/** - API token management
- **blockhostmapping/** - Block host mapping operations
- **blockhosts/** - Block host management
- **encryption_groups/** - Encryption group operations
- **eventdefinitions/** - Event definition management
- **folders/** - Folder operations
- **globalsnapshotstream/** - Global snapshot streaming
- **kafkabroker/** - Kafka broker configuration
- **metrics/** - Metrics collection
- **nonlocalgroup/** - Non-local group management
- **nonlocaluser/** - Non-local user management
- **nonlocaluserkey/** - Non-local user key management
- **openapi-schema/** - OpenAPI schema operations
- **quota/** - Quota management (create, update, delete)
- **raw-request/** - Raw HTTP request examples
- **request-interceptors/** - Request interceptor examples
- **samlconfig/** - SAML configuration
- **topics/** - Topic management
- **user/** - User management
- **userkeys/** - User key management
- **version/** - Version information
- **view/** - View management with NFS protocols
- **vippool/** - VIP pool configuration
- **vms/** - VMS operations

## Common Patterns

### Error Handling
```go
result, err := rest.Resource.Operation(params)
if err != nil {
    panic(fmt.Errorf("operation failed: %w", err))
}
```

### Type Conversion
```go
var container MyStruct
if err := result.Fill(&container); err != nil {
    panic(fmt.Errorf("failed to fill container: %w", err))
}
```

### Parameter Building
```go
params := client.Params{
    "string_field": "value",
    "int_field":    123,
    "bool_field":   true,
    "array_field":  []string{"item1", "item2"},
}
```

## Migration to Typed API

Consider migrating to the typed API (in `../typed/`) for:
- Better type safety
- IDE auto-completion
- Compile-time validation
- Self-documenting code
- Read-only resource protection

See the typed examples for equivalent functionality with improved developer experience.
