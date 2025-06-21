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
rest.Quotas = newResource[Quota](rest, "quotas", dummyClusterVersion)
rest.Views = newResource[View](rest, "views", dummyClusterVersion)
rest.VipPools = newResource[VipPool](rest, "vippools", dummyClusterVersion)
rest.Users = newResource[User](rest, "users", dummyClusterVersion) // <- Here
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

func (uk *UserKey) CreateKey(context.Context, userId int64) (Record, error) {
	path := fmt.Sprintf(uk.resourcePath, userId)
	return request[Record](ctx, uk, http.MethodPost, path, uk.apiVersion, nil, nil)
}

func (uk *UserKey) DeleteKey(context.Context, userId int64, accessKey string) (EmptyRecord, error) {
	path := fmt.Sprintf(uk.resourcePath, userId)
	return request[EmptyRecord](ctx, uk, http.MethodDelete, path, uk.apiVersion, nil, Params{"access_key": accessKey})
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
- response: Resources that implement Renderable interface (Record, RecordSet, EmptyRecord)


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
