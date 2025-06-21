All API operations return a **Response** object that supports rendering, inspection, and optional conversion into Go structs.

A response is one of the following:

- A **single record**: key-value map (`Record`)
- A **list of records**: array of maps (`RecordSet`)
- An **empty result**: used in operations like `DELETE`

These types implement the `DisplayableRecord` interface, which includes formatting and data-extraction methods.

---

## Common Methods

### Display Output

| Method           | Description                              | Output Style           |
|------------------|------------------------------------------|------------------------|
| `PrettyTable()`  | Render response as a formatted table     | Grid-like CLI table    |
| `PrettyJson()`   | Render response as pretty-printed JSON   | Indented/compact JSON  |

#### Example

```go
fmt.Println(response.PrettyTable())
fmt.Println(response.PrettyJson("  "))
```

#### Common attributes access

You can extract frequently used fields directly from a response object:

| Method               | Description                     | Source Key      |
| -------------------- | ------------------------------- | --------------- |
| `RecordID()`         | Returns record ID as `int64`    | `"id"`          |
| `RecordName()`       | Returns name as `string`        | `"name"`        |
| `RecordTenantID()`   | Returns tenant ID as `int64`    | `"tenant_id"`   |
| `RecordTenantName()` | Returns tenant name as `string` | `"tenant_name"` |
| `RecordGUID()`       | Returns GUID as `string`        | `"guid"`        |



#### Fill Struct

Response objects can populate a typed Go struct using .Fill()

```go
package main

import (
	"fmt"
	"log"

	"vast_client"
)

// ViewContainer is a typed struct that mirrors the expected API response.
type ViewContainer struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	TenantID int64  `json:"tenant_id"`
}

func main() {
	// Prepare API client configuration
	config := &vast_client.VMSConfig{
		Host:     "10.27.40.1",
		Username: "admin",
		Password: "123456",
	}

	// Create REST client
	rest, err := vast_client.NewVMSRest(config)
	if err != nil {
		log.Fatalf("failed to create REST client: %v", err)
	}

	// Call EnsureByName or any API method returning a Record
	response, err := rest.Views.EnsureByName("myvolume", vast_client.Params{
		"path":      "/myblock",
		"protocols": []string{"BLOCK"},
		"policy_id": 1,
	})
	if err != nil {
		log.Fatalf("API error: %v", err)
	}

	// Define a variable of the target struct and fill it with response data
	var view ViewContainer
	if err := response.Fill(&view); err != nil {
		log.Fatalf("failed to fill struct: %v", err)
	}

	// Print structured output
	fmt.Println("✅ View created:")
	fmt.Printf("  ID:        %d\n", view.ID)
	fmt.Printf("  Name:      %s\n", view.Name)
	fmt.Printf("  Path:      %s\n", view.Path)
	fmt.Printf("  Tenant ID: %d\n", view.TenantID)
}
```

