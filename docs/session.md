# Session API

The Session API provides low-level HTTP access when you need more control over requests or need to access endpoints not exposed through the typed/untyped resource clients.

## When to Use Session

- Custom or undocumented API endpoints
- Direct control over HTTP methods and paths
- Custom request headers or parameters
- Endpoints not yet available in resource clients

## Available Methods

Session implements 5 HTTP methods:

- **Get** - Retrieve resources
- **Post** - Create resources
- **Put** - Replace resources
- **Patch** - Update resources
- **Delete** - Remove resources

## Accessing Session

Both typed and untyped clients provide access to the Session API:

```go
// From typed client
typedRest, _ := client.NewTypedVMSRest(config)
session := typedRest.GetSession()

// From untyped client
untypedRest, _ := client.NewUntypedVMSRest(config)
session := untypedRest.GetSession()
```

> **Note:** Session methods always return untyped data (`core.Record`, `core.RecordSet`, or `core.EmptyRecord`), even when accessed from a typed client.

## Example Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    client "github.com/vast-data/go-vast-client"
    "github.com/vast-data/go-vast-client/core"
)

func main() {
    config := &client.VMSConfig{
        Host:     "10.27.40.1",
        Username: "admin",
        Password: "123456",
    }

    // Can use either typed or untyped client
    rest, err := client.NewUntypedVMSRest(config)
    if err != nil {
        panic(err)
    }

    ctx := context.Background()

    // Get view by name using query parameter
    path := "views?name=myview"
    result, err := rest.Session.Get(ctx, path, nil, nil)
    if err != nil {
        log.Fatal(err)
    }

    // Session returns untyped data
    recordSet := result.(core.RecordSet)
    if !recordSet.Empty() {
        firstRecord := recordSet[0]
        viewID := firstRecord.RecordID()
        
        // Get view by ID
        path = fmt.Sprintf("views/%d", viewID)
        result, err = rest.Session.Get(ctx, path, nil, nil)
        if err != nil {
            panic(err)
        }
        
        record := result.(core.Record)
        fmt.Printf("View name: %s\n", record["name"])
    } else {
        log.Println("No records found")
    }
}
```

## Custom Headers and Parameters

You can pass custom parameters and headers to Session methods:

```go
// Using query parameters (note: for GET, params go in the URL, not as separate arg)
result, err := session.Get(ctx, "views?name=myview&limit=10&offset=0", nil, nil)

// POST with request body
body := client.Params{
    "name": "newview",
    "path": "/newview",
}
result, err := session.Post(ctx, "views", body, nil)

// PATCH to update
updateBody := client.Params{
    "protocols": []string{"NFS", "SMB"},
}
result, err := session.Patch(ctx, "views/123", updateBody, nil)

// DELETE
result, err := session.Delete(ctx, "views/123", nil, nil)

// Using custom headers
customHeaders := []http.Header{
    {"X-Custom-Header": []string{"value"}},
}
result, err := session.Get(ctx, "views", nil, customHeaders)
```
