# VAST Go Client

[![CI](https://github.com/vast-data/go-vast-client/workflows/CI/badge.svg)](https://github.com/vast-data/go-vast-client/actions/workflows/ci.yml)
[![License: Apache2](https://img.shields.io/badge/License-Apache2-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/vast-data/go-vast-client)](https://goreportcard.com/report/github.com/vast-data/go-vast-client)
[![Coverage Status](https://coveralls.io/repos/github/vast-data/go-vast-client/badge.svg?branch=main)](https://coveralls.io/github/vast-data/go-vast-client?branch=main)
[![Go Reference](https://pkg.go.dev/badge/github.com/vast-data/go-vast-client.svg)](https://pkg.go.dev/github.com/vast-data/go-vast-client)

The VAST Go client provides a typed interface to the VAST Data REST API. It wraps low-level HTTP calls in structured methods, allowing you to interact with volumes, views, quotas, and other resources easily.

---

## Installation

```bash
go get github.com/vast-data/go-vast-client@v0.21.0  # Replace with the latest available tag
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

---

## Development

### Prerequisites

- Go 1.20 or later
- golangci-lint (for linting)
- git

### Building

```bash
# Build the library
make build

# Build all examples
make examples

# Run all checks (format, lint, test)
make all
```

### Testing

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

### Code Quality

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

### Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests and linting (`make all`)
5. Commit your changes (`git commit -m 'Add some amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

Please ensure your code follows the existing style and includes appropriate tests.

### Continuous Integration

This project uses GitHub Actions for CI/CD:

- **Tests**: Run on Go 1.20, 1.21, and 1.22
- **Linting**: golangci-lint with comprehensive rules
- **Security**: gosec and govulncheck
- **Coverage**: Reports sent to Coveralls
- **Dependencies**: Automated updates via Dependabot
