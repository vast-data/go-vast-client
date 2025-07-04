Resources can be queried using session if you need more control over request endpoints, headers, or other parameters. 
Or if endpoint is not present in the VMSRest object.

Session implements 5 methods

- **Get**
- **Post**
- **Put**
- **Patch**
- **Delete**

Example:

```go
ctx := context.Background()
config := &client.VMSConfig{
    Host:     "10.27.40.1",
    Username: "admin",
    Password: "123456",
    BeforeRequestFn: func(ctx context.Context, r *http.Request, verb, url string, body io.Reader) error {
        log.Printf("Sending request: verb=%s, url=%s", verb, url)
        return nil
    },
    AfterRequestFn: func(ctx context.Context, response client.Renderable) (client.Renderable, error) {
        log.Printf("Result:\n%s", response.PrettyTable())
        return response, nil
    },
}

rest, err := client.NewVMSRest(config)
if err != nil {
    panic(err)
}

// Get View by name
path := "views?name=myview"
result, err := rest.Session.Get(ctx, path, nil)
if err != nil {
    log.Fatal(err)
}

recordSet := result.(client.RecordSet)
if !recordSet.Empty() {
    firstRecord := recordSet[0]
    // Get View by id
    path = fmt.Sprintf("views/%d", firstRecord.RecordID())
    result, err = rest.Session.Get(ctx, path, nil)
    if err != nil {
        panic(err)
    }
} else {
    log.Println("No records found")
}
```
