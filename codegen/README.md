# Code Generation

This directory contains the code generation tools for the VAST Data Go Client.

## Available Markers

### Basic Resource Markers

- `+apityped:details:GET=<path>` - Generates `SearchParams` and `DetailsModel` for typed resources
- `+apityped:upsert:POST=<path>` - Generates `RequestBody` and `UpsertModel` for typed resources
- `+apityped:upsert:PUT=<path>` - Same as POST but for PUT operations
- `+apityped:upsert:PATCH=<path>` - Same as POST but for PATCH operations

### Extra Method Markers

Extra methods allow you to define custom API endpoints beyond the standard CRUD operations.

#### Typed Extra Methods
```go
// +apityped:extraMethod:GET=/users/{id}/tenant_data/
// +apityped:extraMethod:PATCH=/users/{id}/tenant_data/
```

Generates typed methods with strong types for request body and response:
- `UserGetTenantData(id any) (*UserGetTenantDataModel, error)`
- `UserUpdateTenantData(id any, body *UserUpdateTenantDataBody) (core.Record, error)`

#### Untyped Extra Methods
```go
// +apiuntyped:extraMethod:GET=/users/{id}/tenant_data/
// +apiuntyped:extraMethod:PATCH=/users/{id}/tenant_data/
```

Generates untyped methods that accept and return generic `core.Params` and `core.Record`:
- `UserGetTenantData(id any, params core.Params) (core.Record, error)`
- `UserUpdateTenantData(id any, body core.Params) (core.Record, error)`

#### Combined Markers (NEW!)

The `+apiall:extraMethod` marker generates **both typed and untyped** methods from a single marker:

```go
// +apiall:extraMethod:GET=/users/{id}/tenant_data/
// +apiall:extraMethod:PATCH=/users/{id}/tenant_data/
```

This is equivalent to having both `+apityped:extraMethod` and `+apiuntyped:extraMethod` markers, but cleaner and less repetitive.

**Example:**
```go
// resources/untyped/user.go
// +apityped:details:GET=users
// +apityped:upsert:POST=users
// +apiall:extraMethod:PATCH=/users/{id}/tenant_data/
// +apiall:extraMethod:GET=/users/{id}/tenant_data/
type User struct {
	*core.VastResource
}
```

This will generate:
- **Typed methods** in `resources/typed/user_autogen.go` with strong types
- **Untyped methods** in `resources/untyped/user_autogen.go` with generic types

## Running Code Generation

```bash
# Generate all resources (typed + untyped)
make autogen

# Or individually:
make generate-typed    # Generate typed resources only
make generate-untyped  # Generate untyped resources only
```

## SearchParams and RawData

All generated `SearchParams` structs include a special `RawData` field:

```go
type UserSearchParams struct {
    Name string `json:"name,omitempty"`
    Uid  int64  `json:"uid,omitempty"`
    // ... other fields ...
    
    // RawData allows arbitrary search parameters as key/value pairs.
    // Use this when you need fuzzy search or parameters not covered by typed fields.
    // For example: path__contains, name__icontains, etc.
    // This field is excluded from JSON/YAML serialization.
    RawData core.Params `json:"-" yaml:"-"`
}
```

**Usage:**
```go
// Option 1: Use typed fields
params := &UserSearchParams{Name: "john"}

// Option 2: Use RawData for fuzzy search
params := &UserSearchParams{
    RawData: core.Params{
        "name__contains": "john",
        "path__contains": "/foo",
    },
}
```

## Simplified Body Parameters (Untyped Extra Methods)

For untyped extra methods with request bodies containing 1-3 simple fields, the generator automatically creates type-safe inline parameters instead of requiring a generic `core.Params` map.

### Requirements for Simplification

A request body is simplified when ALL conditions are met:
- The body has 1-3 properties
- All properties are simple types (`string`, `int`, `int64`, `float64`, `bool`)
- No complex nested objects or arrays

### Example

**OpenAPI DELETE with single field:**
```yaml
delete:
  parameters:
  - in: body
    schema:
      type: object
      properties:
        access_key:
          type: string
      required:
      - access_key
```

**Generated (Simplified):**
```go
func (u *UserKey) UserKeyDeleteAccessKeys(id any, accessKey string) (core.EmptyRecord, error) {
    // Body automatically constructed: {"access_key": accessKey}
}
```

**Generated (4+ fields - not simplified):**
```go
func (r *Resource) MethodName(id any, body core.Params) (core.Record, error) {
    // User constructs body manually
}
```

### Benefits

- **Type Safety**: Compile-time checking
- **Better IDE Support**: Autocomplete and documentation
- **Cleaner API**: `DeleteAccessKeys(id, "key123")` vs `DeleteAccessKeys(id, core.Params{"access_key": "key123"})`

**Parameter Naming**: `access_key` → `accessKey`, `tenant_id` → `tenantId`

## Model Generation

The code generator creates two types of models:

1. **DetailsModel** - Used for GET/List responses (read-only view)
2. **UpsertModel** - Used for POST/PUT/PATCH responses (after create/update)

These are often different because creation/update operations may return additional fields (e.g., generated tokens, IDs) that aren't part of the standard detail view.

## Deprecation Notice

The following markers are deprecated but still supported for backward compatibility:
- `+apityped:searchQuery:GET` → Use `+apityped:details:GET`
- `+apityped:createQuery:POST` → Use `+apityped:upsert:POST`
- `+apityped:detailsQuery:GET` → Use `+apityped:details:GET`
- `+apityped:upsertQuery:POST` → Use `+apityped:upsert:POST`
- `+apityped:requestBody:POST` → Use `+apityped:upsert:POST`
- `+apityped:model:SCHEMA` → **Not supported**, use details/upsert markers instead

