## Configuration

This struct configures how the client connects to the VAST API.

Example with all fields:

```go
config := &client.VMSConfig{
    Host:           "10.27.40.1",
    Port:           443,
    Username:       "admin",
    Password:       "123456",
    SslVerify:      true,
    Timeout:        &timeout,
    MaxConnections: 10,
    UserAgent:      "vast-go-client/1.0",
    ApiVersion:     "v5",
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

| Field           | Type                                                                                 | Description                                                                        | Required | Default |
|-----------------|--------------------------------------------------------------------------------------|------------------------------------------------------------------------------------|--------|----|
| `Host`          | `string`                                                                             | Hostname or IP of the VMS API server.                                              | ✅      | —  |
| `Port`          | `uint64`                                                                             | Port for the API server.                                                           | ❌      | `443` |
| `Username`      | `string`                                                                             | Username for basic auth (used with `Password`).                                    | ⚠️     | —  |
| `Password`      | `string`                                                                             | Password for basic auth (used with `Username`).                                    | ⚠️     | —  |
| `ApiToken`      | `string`                                                                             | Optional bearer token (alternative to username/password).                          | ⚠️     | —  |
| `SslVerify`     | `bool`                                                                               | Verify SSL certificates when `true`.                                               | ❌      | `false` |
| `Timeout`       | `*time.Duration`                                                                     | HTTP timeout for API requests. If `nil`, a default is used.                        | ❌      | `30s` |
| `MaxConnections`| `int`                                                                                | Max concurrent HTTP connections.                                                   | ❌      | `10` |
| `UserAgent`     | `string`                                                                             | Optional custom `User-Agent` string for HTTP requests.                             | ❌      | `vast-go-client` |
| `BeforeRequestFn`    | `func(ctx context.Context, r *http.Request, verb, url string, body io.Reader) error` | Optional hook executed before each request. Useful for logging or mutation.        | ❌      | —  |
| `AfterRequestFn`    | `func(ctx context.Context, response Renderable) (Renderable, error)`                 | Optional hook executed after receiving a response. Useful for logging or mutation. | ❌   | —  |
