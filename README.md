# GO Vast Client

[![CI](https://github.com/vast-data/go-vast-client/workflows/CI/badge.svg)](https://github.com/vast-data/go-vast-client/actions/workflows/ci.yml)
[![License: Apache2](https://img.shields.io/badge/License-Apache2-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/vast-data/go-vast-client)](https://goreportcard.com/report/github.com/vast-data/go-vast-client)
[![Coverage Status](https://coveralls.io/repos/github/vast-data/go-vast-client/badge.svg?branch=main)](https://coveralls.io/github/vast-data/go-vast-client?branch=main)
[![Go Reference](https://pkg.go.dev/badge/github.com/vast-data/go-vast-client.svg)](https://pkg.go.dev/github.com/vast-data/go-vast-client)

Go client for the [VAST VMS REST API](https://kb.vastdata.com/documentation/docs/the-vast-rest-api-3).

Documentation: [go-vast-client docs](https://vast-data.github.io/go-vast-client/)

## Quickstart

```bash
go get github.com/vast-data/go-vast-client
```

```go
package main

import (
    "fmt"

    client "github.com/vast-data/go-vast-client"
    "github.com/vast-data/go-vast-client/resources/typed"
    "github.com/vast-data/go-vast-client/resources/typed/expr"
)

func main() {
    rest, err := client.NewTypedVMSRest(&client.VMSConfig{
        Host:     "10.27.40.1",
        Username: "admin",
        Password: "123456",
    })
    if err != nil {
        panic(err)
    }

    view, err := rest.Views.Ensure(
        &typed.ViewSearchParams{Path: expr.S("/myview")},
        &typed.ViewRequestBody{
            Name:      "myview",
            Path:      "/myview",
            Protocols: &[]string{"NFS"},
            PolicyId:  1,
            CreateDir: true,
        },
    )
    if err != nil {
        panic(err)
    }

    fmt.Printf("View: %s (ID: %d)\n", view.Name, view.Id)
}
```

For installation, configuration, typed vs untyped clients, and API reference, see the
[documentation](https://vast-data.github.io/go-vast-client/).

For developers and contributors: [DEVELOPER.md](docs/DEVELOPER.md)
