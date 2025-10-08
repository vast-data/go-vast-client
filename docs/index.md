# VAST Go Client

[![CI](https://github.com/vast-data/go-vast-client/workflows/CI/badge.svg)](https://github.com/vast-data/go-vast-client/actions/workflows/ci.yml)
[![License: Apache2](https://img.shields.io/badge/License-Apache2-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/vast-data/go-vast-client)](https://goreportcard.com/report/github.com/vast-data/go-vast-client)
[![Coverage Status](https://coveralls.io/repos/github/vast-data/go-vast-client/badge.svg?branch=main)](https://coveralls.io/github/vast-data/go-vast-client?branch=main)
[![Go Reference](https://pkg.go.dev/badge/github.com/vast-data/go-vast-client.svg)](https://pkg.go.dev/github.com/vast-data/go-vast-client)

The VAST Go client provides an interface to the VAST Data REST API. It wraps low-level HTTP calls in structured methods, allowing you to interact with volumes, views, quotas, and other resources easily.

---

> **NOTE:** Since version 0.100.0, the REST client has been split into two distinct client types:
>
> - **Typed Client** (`NewTypedVMSRest`): Provides strongly-typed structs for requests and responses. Offers compile-time type safety, IDE auto-completion, and clear API contracts. Recommended for most use cases.
> - **Untyped Client** (`NewVMSRest`): Uses flexible `map[string]any` for data handling. Useful for dynamic scenarios and prototyping. This is the default recommended client.

## Installation

```bash
go get github.com/vast-data/go-vast-client@v0.103.0  # Replace with the latest available tag
```

Import it in your Go code:

```go
import client "github.com/vast-data/go-vast-client"
```

## Quick Start

```go
package main

import (
    "fmt"
    client "github.com/vast-data/go-vast-client"
    "github.com/vast-data/go-vast-client/resources/typed"
)

func main() {
    config := &client.VMSConfig{
        Host:     "10.27.40.1",
        Username: "admin",
        Password: "123456",
    }

    rest, err := client.NewTypedVMSRest(config)
    if err != nil {
        panic(err)
    }

    searchParams := &typed.ViewSearchParams{
        Path: "/myview",
    }
    
    body := &typed.ViewRequestBody{
        Name:      "myview",
        Path:      "/myview",
        Protocols: &[]string{"NFS"},
        PolicyId:  1,
        CreateDir: true,
    }

    view, err := rest.Views.Ensure(searchParams, body)
    if err != nil {
        panic(err)
    }

    fmt.Printf("View: %s (ID: %d)\n", view.Name, view.Id)
}
```



