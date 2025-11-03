This struct configures how the client connects to the VAST API.

Example with all fields:

```go
config := &client.VMSConfig{
    Host:           "10.27.40.1",
    Port:           443,
    Username:       "admin",
    Password:       "123456",
    ApiToken:       "",        // Alternative to Username/Password
    UseBasicAuth:   false,     // Use HTTP Basic Auth instead of JWT
    Tenant:         "mytenant", // Optional tenant for scoped authentication
    SslVerify:      true,
    RespectProxy:   true,      // Respect proxy environment variables (HTTP_PROXY, HTTPS_PROXY, NO_PROXY)
    Timeout:        &timeout,
    MaxConnections: 10,
    UserAgent:      "vast-go-client/1.0",
    ApiVersion:     "v5",
    Context:        ctx,       // Optional external context for request control
    BeforeRequestFn: func(ctx context.Context, r *http.Request, verb, url string, body io.Reader) error {
        log.Printf("Request: %s %s", verb, url)
        return nil
    },
    AfterRequestFn: func(ctx context.Context, response client.Renderable) (client.Renderable, error) {
        log.Println(response.PrettyTable())
        return response, nil
    },
}
```

| Field           | Type                                                                                 | Description                                                                       | Required | Default          |
|-----------------|--------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------|--------|------------------|
| `Host`          | `string`                                                                             | Hostname or IP of the VMS API server.                                             | ✅      | —                |
| `Port`          | `uint64`                                                                             | Port for the API server.                                                          | ❌      | `443`            |
| `Username`      | `string`                                                                             | Username for authentication (used with `Password`).                               | ⚠️     | —                |
| `Password`      | `string`                                                                             | Password for authentication (used with `Username`).                               | ⚠️     | —                |
| `ApiToken`      | `string`                                                                             | Optional API token (alternative to username/password). Takes priority over other auth methods. | ⚠️     | —                |
| `UseBasicAuth`  | `bool`                                                                               | Use HTTP Basic Authentication instead of JWT (requires `Username`/`Password`).    | ❌      | `false`          |
| `Tenant`        | `string`                                                                             | Optional tenant name for tenant scoped authentication (tenant admin).             | ❌      | —                |
| `SslVerify`     | `bool`                                                                               | Verify SSL certificates when `true`.                                              | ❌      | `false`          |
| `RespectProxy`  | `bool`                                                                               | Respect proxy environment variables (`HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`).   | ❌      | `false`          |
| `Timeout`       | `*time.Duration`                                                                     | HTTP timeout for API requests. If `nil`, a default is used.                       | ❌      | `30s`            |
| `MaxConnections`| `int`                                                                                | Max concurrent HTTP connections.                                                  | ❌      | `10`             |
| `UserAgent`     | `string`                                                                             | Optional custom `User-Agent` string for HTTP requests.                            | ❌      | `vast-go-client` |
| `ApiVersion`    | `string`                                                                             | Optional API version to use for requests.                                         | ❌      | `v5`             |
| `Context`       | `context.Context`                                                                    | Optional external context for controlling HTTP request lifecycle. Used as parent context for all requests. | ❌ | `nil` |
| `BeforeRequestFn`    | `func(ctx context.Context, r *http.Request, verb, url string, body io.Reader) error` | Optional hook executed before each request. Useful for logging or mutation.       | ❌      | —                |
| `AfterRequestFn`    | `func(ctx context.Context, response Renderable) (Renderable, error)`                 | Optional hook executed after receiving a response. Useful for logging or mutation. | ❌   | —                |

## Authentication Methods

The client supports three authentication methods with the following priority:

1. **API Token** (highest priority) - if `ApiToken` is provided
2. **HTTP Basic Authentication** - if `UseBasicAuth=true` AND `Username/Password` are provided
3. **JWT Authentication** (default) - if `Username/Password` are provided

### JWT Authentication (Default)
```go
config := &client.VMSConfig{
    Host:     "10.27.40.1",
    Username: "admin",
    Password: "secret",
    // UseBasicAuth: false (or omitted) - uses JWT by default
}
```

### HTTP Basic Authentication
```go
config := &client.VMSConfig{
    Host:         "10.27.40.1",
    Username:     "admin",
    Password:     "secret",
    UseBasicAuth: true,  // Enable Basic Auth
}
```

### API Token Authentication
```go
config := &client.VMSConfig{
    Host:     "10.27.40.1",
    ApiToken: "your-api-token-here",
    // ApiToken always takes precedence
}
```

## Context Usage

The `Context` field allows you to control the lifecycle of all HTTP requests:

### Request Timeout
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

config := &client.VMSConfig{
    Host:    "10.27.40.1",
    Context: ctx,  // All requests will respect this 30s timeout
}
```

### Request Cancellation
```go
ctx, cancel := context.WithCancel(context.Background())

config := &client.VMSConfig{
    Host:    "10.27.40.1",
    Context: ctx,
}

// Later: cancel all in-flight requests
cancel()
```

## Proxy Configuration

By default, the client does not use proxy servers. To enable proxy support through environment variables, set `RespectProxy` to `true`:

```go
config := &client.VMSConfig{
    Host:         "10.27.40.1",
    Username:     "admin",
    Password:     "secret",
    RespectProxy: true,  // Enable proxy environment variables
}
```

When `RespectProxy` is `true`, the client will respect these standard environment variables:

- `HTTP_PROXY` or `http_proxy` - Proxy URL for HTTP requests
- `HTTPS_PROXY` or `https_proxy` - Proxy URL for HTTPS requests
- `NO_PROXY` or `no_proxy` - Comma-separated list of hosts to bypass the proxy

### Example with Proxy

```bash
# Set proxy environment variables
export HTTPS_PROXY="http://proxy.example.com:8080"
export NO_PROXY="localhost,127.0.0.1"

# Run your Go application
go run main.go
```

```go
config := &client.VMSConfig{
    Host:         "vast-api.example.com",
    Username:     "admin",
    Password:     "secret",
    RespectProxy: true,  // Client will use proxy.example.com:8080
}
```

### Disable Proxy

To explicitly disable proxy support (default behavior):

```go
config := &client.VMSConfig{
    Host:         "10.27.40.1",
    Username:     "admin",
    Password:     "secret",
    RespectProxy: false,  // Ignore proxy environment variables (default)
}
```
