# Development Guide

## Prerequisites

- Go 1.20 or later
- golangci-lint (for linting)
- git

## Building

```bash
# Build the library
make build

# Run all checks (format, lint, test)
make all
```

## Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# View coverage report
make coverage-report

# Run short tests only
make test-short
```

## Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Run static analysis
make vet

# Run security checks
make security
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests and linting (`make all`)
5. Commit your changes (`git commit -m 'Add some amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

Please ensure your code follows the existing style and includes appropriate tests.

## Continuous Integration

This project uses GitHub Actions for CI/CD:

- **Tests**: Run on Go 1.20, 1.21, and 1.22
- **Linting**: golangci-lint with comprehensive rules
- **Security**: gosec and govulncheck
- **Coverage**: Reports sent to Coveralls
- **Dependencies**: Automated updates via Dependabot

---

## Adding New API Resources

### Do you want to add new API resource?

Suppose you want to add a User resource so that it can be queried
using the endpoints `<base url>/users` for listing and `<base url>/users/<id>` for retrieving details.

- Start by defining a new VastResource named User in the `vast_resource.go` file.

```go
type User struct {
	*VastResource
}
```

- Add new `User` type to `VastResourceType` generic type in `the vast_resource.go` file.

```go
type VastResourceType interface {
	Version | Quota | View | User // <- Here
}
```

- Declare new `Users` sub-resource in `rest.go` file and add resource initialization.

```go
type VMSRest struct {
	Session     RESTSession
	resourceMap map[string]VastResource

	Versions          *Version
	VTasks            *VTask
	Quotas            *Quota
	Views             *View
	VipPools          *VipPool
	Users             *User  // <- Here
}
```

```go
....
rest.Quotas = newResource[Quota](rest, "quotas")
rest.Views = newResource[View](rest, "views")
rest.VipPools = newResource[VipPool](rest, "vippools")
rest.Users = newResource[User](rest, "users") // <- Here
....
```

At this point methods:
`List`, `Get`, `Delete`, `Update`, `Create`, `Ensure`, `EnsureByName`, `GetById`, `DeleteById` 
and also variants of forementioned methods with context:
`ListWithContext`, `GetWithContext`, `DeleteWithContext`, `UpdateWithContext`, `CreateWithContext`, `EnsureWithContext`, `EnsureByNameWithContext`, `GetByIdWithContext`, `DeleteByIdWithContext`
are available for `User` resource.

Examples:

Create `User`:
```go
result, err := rest.Users.Create(client.Params{
    "name": "myUser",
    "uid":  9999,
})
```

Ensure `User` by name (Get by name or Create with provided name and additional params):
```go
result, err := rest.Users.EnsureByName("myUser", client.Params{"uid": 9999})
```

Ensure `User` by search params (Get by search params or Create with body params):
```go
searchParams := client.Params{"name": "test", "tenant_id": 1}
result, err := rest.Users.EnsureByName(searchParams, client.Params{"uid": 9999})
```

Update `User`:
```go
result, err := rest.Users.Update(1, client.Params{"uid": 10000})
```

Get `User`:
```go
result, err := rest.User.Get(client.Params{"name": "myUser"})
```

Get `User` by id:
```go
result, err := rest.Users.GetById(1)
```

Delete `User` (Get user by search params and if found delete it. Not found is not error condition):
```go
result, err := rest.Users.Delete(client.Params{"name": "myUser"}, nil)
```

Delete `User` by id:
```go
result, err := rest.Users.DeleteById(1, nil)
```

!!! note
Aforementioned flow covers "classic" API resources of form `/<resource name>/<id>`.
For non-standard APIs you have to define custom methods or use "Low level API" (see Overview section)


### Define non-standard method for API Resource

You can define custom methods for API Resource. Good example is `UserKey` Resource for generating S3 keys.
It has 2 custom methods `CreateKey` and `DeleteKey`

```go
type UserKey struct {
	*VastResource
}

func (r *UserKey) CreateKey(context.Context, userId int64) (Record, error) {
	path := fmt.Sprintf(r.resourcePath, userId)
	return request[Record](ctx, r, http.MethodPost, path, nil, nil)
}

func (r *UserKey) DeleteKey(context.Context, userId int64, accessKey string) (Record, error) {
	path := fmt.Sprintf(r.resourcePath, userId)
	return request[Record](ctx, r, http.MethodDelete, path, nil, Params{"access_key": accessKey})
}
```

!!! warning
Main rule: **Do not override standard methods Ensure, Get, List, Create, DeleteById etc**. Create your own methods like CreateUser, DeleteKey etc.


Another good example is `BlockHostMapping` Resource where specific methods are used to map BlockHosts to Volumes.


### Request/Response interceptors

API Resources can implement `RequestInterceptor` interface

You can define method `beforeRequest` for particular resource:
```go
beforeRequest(context.Context, r *http.Request, verb string, url string, body io.Reader) error
```

Parameters:

- ctx: The request context, useful for deadlines, tracing, or cancellation.
- verb: The HTTP method (e.g., GET, POST, PUT).
- url: The URL path being accessed (including query params)
- body: The request body as an io.Reader, typically containing JSON data.


Or you can define method `afterRequest`:

```go
afterRequest(ctx context.Context, response Renderable) (Renderable, error)
```

Parameters:

- ctx: The request context, useful for deadlines, tracing, or cancellation.
- response: Resources that implement Renderable interface (Record, RecordSet)


At this moment I don't have practical example for `beforeRequest`. Probably it can be used for logging etc.

For `afterRequest` good example is `Snapshot` resource:

```go
type Snapshot struct {
	*VastResource
}

func (s *Snapshot) afterRequest(ctx context.Context, response Renderable) (Renderable, error) {
	// List of snapshots is returned under "results" key
	return applyCallbackForRecordUnion[RecordSet](response, func(r Renderable) (Renderable, error) {
		// This callback is only invoked if response is a RecordSet
		if rawMap, ok := any(r).(map[string]interface{}); ok {
			if inner, found := rawMap["results"]; found {
				if list, ok := inner.([]map[string]any); ok {
					return toRecordSet(list)
				}
			}
		}
		return r, nil
	})
}
```

Here in case of `RecordSet` (List endpoint) list of snapshot records are returned under `results` key. IOW smth like:
```json
{
  "results": [
    {
      "name": "snapshot1",  .. other fields
    },
    {
      "name": "snapshot2" .. other fields
    }
  ]
}
```

So make sense to get `results` value and return only it to avoid additional parsing of returned Record.

---

## Typed Resources Auto-Generation

The go-vast-client supports automatic generation of typed resources that provide compile-time type safety and better IDE support. This system uses APIBuilder markers to define how to generate typed structs and methods.

### Generating Typed Resources

To generate all typed resources:

```bash
make generate-typed
```

This command:
1. Scans `vast_resource.go` for resources with APIBuilder markers
2. Generates typed structs based on OpenAPI schema
3. Creates typed methods with proper request/response types
4. Formats the generated code with `go fmt`

Generated files are placed in the `typed/` directory:
- `rest.go` - Typed VMSRest client
- `<resource>.go` - Individual typed resource implementations (e.g., `quota.go`, `user.go`)

### APIBuilder Markers

APIBuilder markers are Go comments that define how to generate typed resources. They must be placed directly above the resource struct definition.

#### Required Markers

Every resource must have:

1. **Search Query Marker** - Defines how to search/list resources
2. **Model Marker** - Defines the response structure

Non-read-only resources also need:

3. **Request Body Marker** - Defines the structure for create/update operations

#### Search Query Markers

Define how to generate search parameters for listing and filtering resources:

```go
// +apityped:searchQuery:GET=<endpoint>
// +apityped:searchQuery:SCHEMA=<SchemaName>
```

**Examples:**
```go
// Use GET endpoint parameters
// +apityped:searchQuery:GET=quotas

// Use specific OpenAPI schema
// +apityped:searchQuery:SCHEMA=QuotaSearchParams
```

**Generated:** `<Resource>SearchParams` struct with fields for filtering.

#### Request Body Markers

Define the structure for create and update operations:

```go
// +apityped:requestBody:POST=<endpoint>
// +apityped:requestBody:PUT=<endpoint>  
// +apityped:requestBody:PATCH=<endpoint>
// +apityped:requestBody:SCHEMA=<SchemaName>
```

**Examples:**
```go
// Use POST endpoint request body
// +apityped:requestBody:POST=quotas

// Use specific OpenAPI schema
// +apityped:requestBody:SCHEMA=QuotaCreateRequest
```

**Generated:** `<Resource>RequestBody` struct for create/update operations.

#### Model Markers

Define the response structure for API operations:

```go
// +apityped:model:GET=<endpoint>
// +apityped:model:POST=<endpoint>
// +apityped:model:PUT=<endpoint>
// +apityped:model:DELETE=<endpoint>
// +apityped:model:PATCH=<endpoint>
// +apityped:model:SCHEMA=<SchemaName>
```

**Examples:**
```go
// Use POST endpoint response
// +apityped:model:POST=quotas

// Use specific OpenAPI schema  
// +apityped:model:SCHEMA=Quota
```

**Generated:** `<Resource>Model` struct for API responses.

#### Read-Only Marker

For resources that don't support create/update/delete operations:

```go
// +apityped:readOnly
```

Read-only resources only generate `Get*`, `List*`, `Exists*`, and `MustExists*` methods.

### Complete Resource Examples

#### Full CRUD Resource

```go
// +apityped:searchQuery:GET=quotas
// +apityped:requestBody:POST=quotas  
// +apityped:model:SCHEMA=Quota
type Quota struct {
    *VastResource
}
```

**Generates:**
- `QuotaSearchParams` - for filtering/searching
- `QuotaRequestBody` - for create/update operations
- `QuotaModel` - for API responses
- Full CRUD methods: `Create`, `Update`, `Delete`, `Get`, `List`, `Ensure`, etc.

#### Read-Only Resource

```go
// +apityped:readOnly
// +apityped:searchQuery:GET=versions
// +apityped:model:SCHEMA=Version
type Version struct {
    *VastResource  
}
```

**Generates:**
- `VersionSearchParams` - for filtering/searching
- `VersionModel` - for API responses  
- Read-only methods: `Get`, `List`, `Exists`, `MustExists` (no Create/Update/Delete)

#### Using Schema References

```go
// +apityped:searchQuery:SCHEMA=UserSearchCriteria
// +apityped:requestBody:SCHEMA=UserCreateRequest
// +apityped:model:SCHEMA=User
type User struct {
    *VastResource
}
```

This approach uses specific OpenAPI schema definitions instead of endpoint-derived schemas.

### Generated Struct Features

All generated structs include:

- **JSON Tags**: For serialization (`json:"field_name,omitempty"`)
- **YAML Tags**: For YAML support (`yaml:"field_name,omitempty"`)  
- **Required Tags**: Field requirement info (`required:"true/false"`)
- **Doc Tags**: Field documentation (`doc:"Field description"`)
- **Type Safety**: Proper Go types (string, int64, bool, etc.)
- **Nested Structs**: For complex object fields
- **Pointer Fields**: For arrays and objects to handle `omitempty` correctly

### Generated Methods

Typed resources provide the same methods as untyped resources but with typed parameters:

**Standard Methods:**
- `Get(req *SearchParams) (*Model, error)`
- `List(req *SearchParams) ([]*Model, error)`  
- `Create(req *RequestBody) (*Model, error)`
- `Update(id any, req *RequestBody) (*Model, error)`
- `Delete(req *SearchParams) error`
- `Ensure(searchParams *SearchParams, body *RequestBody) (*Model, error)`

**Context Methods:**
- `GetWithContext(ctx context.Context, req *SearchParams) (*Model, error)`
- `ListWithContext(ctx context.Context, req *SearchParams) ([]*Model, error)`
- And so on...

### Accessing Untyped Methods

From any typed resource, you can access the underlying untyped client:

```go
// Access untyped methods when needed
record, err := typedRest.Untyped.Quotas.GetWithContext(ctx, params)
```

### Troubleshooting Generation

**Common Issues:**

1. **Missing Markers**: Ensure all required markers are present
2. **Invalid Schema Names**: Verify schema names exist in OpenAPI spec
3. **Compilation Errors**: Check generated code for syntax issues
4. **Missing Fields**: Verify OpenAPI schema completeness

**Debug Generation:**
```bash
# Verbose generation output
cd autogen && go run ./cmd/generate-typed-resources
```

### Adding New Resources for Typed Generation

1. Add APIBuilder markers to your resource in `vast_resource.go`
2. Ensure the resource exists in the untyped `VMSRest` struct
3. Run `make generate-typed`
4. Verify generated code compiles: `go build ./typed/...`

### Release new version of the client

To release a new version of the client:

1. Update the version in `version` file.
2. Push changes to the `main` branch.