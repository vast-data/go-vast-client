# API Responses

API operations return different response types depending on whether you're using the typed or untyped client.

## Response Types by Client

### Typed Client Responses

The typed client returns **strongly-typed structs** for all operations:

```go
rest, _ := client.NewTypedVMSRest(config)

// Returns *typed.ViewUpsertModel
view, err := rest.Views.Create(body)

// Returns []typed.ViewDetailsModel
views, err := rest.Views.List(nil)

// Returns *typed.ViewDetailsModel
view, err := rest.Views.Get(searchParams)
```

**Benefits:**
- Compile-time type safety
- IDE autocomplete for all fields
- Clear struct definitions
- No need for type assertions or manual unmarshaling

### Untyped Client Responses

The untyped client returns **flexible map-based types**:

- **`core.Record`**: Single record (key-value map: `map[string]any`)
- **`core.RecordSet`**: List of records (`[]map[string]any`)
- **`core.EmptyRecord`**: Empty result (used in operations like DELETE)

```go
rest, _ := client.NewVMSRest(config)

// Returns core.Record
record, err := rest.Views.Create(params)

// Returns core.RecordSet
recordSet, err := rest.Views.List(params)
```

These types implement the `core.Renderable` interface with formatting and data-extraction methods.

---

## Working with Typed Responses

Typed responses are Go structs with strongly-typed fields:

```go
rest, _ := client.NewTypedVMSRest(config)

view, err := rest.Views.Get(&typed.ViewSearchParams{Name: "myview"})
if err != nil {
    log.Fatal(err)
}

// Direct field access with type safety
fmt.Printf("View ID: %d\n", view.Id)
fmt.Printf("View Name: %s\n", view.Name)
fmt.Printf("View Path: %s\n", view.Path)

// Work with nested structs
if view.Protocols != nil {
    for _, protocol := range *view.Protocols {
        fmt.Printf("Protocol: %s\n", protocol)
    }
}
```

---

## Working with Untyped Responses

Untyped responses provide helper methods for display and data extraction.

### Display Output

| Method           | Description                              | Output Style           |
|------------------|------------------------------------------|------------------------|
| `PrettyTable()`  | Render response as a formatted table     | Grid-like CLI table    |
| `PrettyJson()`   | Render response as pretty-printed JSON   | Indented/compact JSON  |

**Example:**

```go
rest, _ := client.NewVMSRest(config)

record, err := rest.Views.Get(client.Params{"name": "myview"})
if err != nil {
    log.Fatal(err)
}

fmt.Println(record.PrettyTable())
fmt.Println(record.PrettyJson("  "))
```

### Common Attribute Access

You can extract frequently used fields directly from a response object:

| Method               | Description                     | Source Key      |
| -------------------- | ------------------------------- | --------------- |
| `RecordID()`         | Returns record ID as `int64`    | `"id"`          |
| `RecordName()`       | Returns name as `string`        | `"name"`        |
| `RecordTenantID()`   | Returns tenant ID as `int64`    | `"tenant_id"`   |
| `RecordTenantName()` | Returns tenant name as `string` | `"tenant_name"` |
| `RecordGUID()`       | Returns GUID as `string`        | `"guid"`        |



**Example:**

```go
rest, _ := client.NewVMSRest(config)

record, err := rest.Views.Get(client.Params{"name": "myview"})
if err != nil {
    log.Fatal(err)
}

// Direct map access
viewID := record.RecordID()
viewName := record.RecordName()
fmt.Printf("View: %s (ID: %d)\n", viewName, viewID)
```

### Converting Untyped to Typed with Fill()

Untyped responses can be converted to Go structs using the `Fill()` method:

```go
package main

import (
    "fmt"
    "log"
    client "github.com/vast-data/go-vast-client"
)

// Custom struct that mirrors the expected API response
type ViewData struct {
    ID       int64  `json:"id"`
    Name     string `json:"name"`
    Path     string `json:"path"`
    TenantID int64  `json:"tenant_id"`
}

func main() {
    config := &client.VMSConfig{
        Host:     "10.27.40.1",
        Username: "admin",
        Password: "123456",
    }

    // Use untyped client
    rest, err := client.NewVMSRest(config)
    if err != nil {
        log.Fatal(err)
    }

    // Get untyped response
    record, err := rest.Views.Get(client.Params{"name": "myview"})
    if err != nil {
        log.Fatal(err)
    }

    // Fill custom struct from untyped response
    var view ViewData
    if err := record.Fill(&view); err != nil {
        log.Fatalf("failed to fill struct: %v", err)
    }

    fmt.Printf("View: %s (ID: %d, Path: %s)\n", view.Name, view.ID, view.Path)
}
```
