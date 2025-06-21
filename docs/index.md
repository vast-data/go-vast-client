# VAST Go Client

[![License: Apache2](https://img.shields.io/badge/License-Apache2-yellow.svg)](https://opensource.org/licenses/MIT)

The VAST Go client provides a typed interface to the VAST Data REST API. It wraps low-level HTTP calls in structured methods, allowing you to interact with volumes, views, quotas, and other resources easily.

---

## Installation

```bash
go get github.com/vast-data/go-vast-client
```

Import it in your Go code:

```go
import client "github.com/vast-data/go-vast-client"
```

## Quick Start

```go
package main

import (
    "log"
    client "github.com/vast-data/go-vast-client"
)

func main() {
    config := &client.VMSConfig{
        Host:     "10.27.40.1",
        Username: "admin",
        Password: "123456",
    }

    rest, err := client.NewVMSRest(config)
    if err != nil {
        panic(err)
    }

    result, err := rest.Views.EnsureByName("myview", client.Params{
        "path":      "/myblock",
        "protocols": []string{"BLOCK"},
        "policy_id": 1,
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Println(result.PrettyTable())
}
```
