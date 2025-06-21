## Errors

The VAST client can return **two categories of errors**:

---

#### 1. API Errors (from VAST backend)

These are errors returned by the VAST server, such as:

They are represented by the `ApiError` type.

```go
type ApiError struct {
    Method     string
    URL        string
    StatusCode int
    Body       string
}
```

Helpers:


| Helper Function                          | Description                                        |
| ---------------------------------------- | -------------------------------------------------- |
| `IsApiError(err error) bool`             | Checks if the error is of type `*ApiError`         |
| `IgnoreStatusCodes(err, codes...) error` | Ignores the error if its HTTP status is in `codes` |


#### 2. Validation Errors (from client-side logic)

These are raised after an HTTP request is sent, such as:

Common validation errors include:

- `NotFoundError`: Resource not found - for methods which expect at least one record to be returned.
- `TooManyRecordsError`: More than one record found - for methods which expect a single record to be returned.

```go
type NotFoundError struct {
	Resource string
	Query    string
}

type TooManyRecordsError struct {
	ResourcePath string
	Params       Params
}

```

Helpers:

| Helper Function                               | Description                                       |
| --------------------------------------------- | ------------------------------------------------- |
| `IsNotFoundErr(err error) bool`               | Checks if error is `*NotFoundError`               |
| `IgnoreNotFound(record, err) (record, error)` | Returns `record, nil` if error is `NotFoundError` |
