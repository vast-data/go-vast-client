# VAST Data Go Client

A comprehensive Go client library for VAST Data systems, providing both typed and untyped access to VAST Data REST APIs.

## 🚀 Quick Start

### Installation

```bash
go get github.com/vast-data/go-vast-client
```

### Basic Usage

```go
package main

import (
    "log"
    vast_client "github.com/vast-data/go-vast-client"
)

func main() {
    // Configuration
    config := &vast_client.VMSConfig{
        Host:     "https://vast.example.com",
        User:     "admin",
        Password: "password",
    }

    // Create typed client (original)
    client, err := vast_client.NewVMSRest(config)
    if err != nil {
        log.Fatal(err)
    }

    // Access resources directly
    users := client.Users
    quotas := client.Quotas
    views := client.Views

    // Use the resources...
}
```

### Untyped Usage

```go
package main

import (
    "log"
    vast_client "github.com/vast-data/go-vast-client"
)

func main() {
    // Configuration
    config := &vast_client.VMSConfig{
        Host:     "https://vast.example.com",
        User:     "admin",
        Password: "password",
    }

    // Create untyped client
    client, err := vast_client.NewUntypedVMSRest(config)
    if err != nil {
        log.Fatal(err)
    }

    // Access resources via methods
    users := client.Users()
    quotas := client.Quotas()
    views := client.Views()

    // Use the resources...
}
```

## 📦 Library Structure

This library is organized into modular components:

### Core Module (`core/`)
Essential functionality that all other modules depend on:
- **Types**: `VastResource`, `VMSRest`, `VMSConfig`, `Params`, `Record`, etc.
- **Session Management**: HTTP client, request handling, authentication
- **Utilities**: Common helper functions and utilities

### Resources Module (`resources/`)
All VAST API resource definitions:
- **`untyped/`**: Untyped resources for dynamic access
- **`typed/`**: Typed resources for compile-time type safety

### Auth Module (`auth/`)
Authentication and authorization:
- Basic authentication
- Token-based authentication
- SAML authentication

### Codegen Module (`codegen/`)
Code generation tools and utilities:
- Marker parsing
- API builders
- Template engines

## 🔧 Client Types

### Typed Client (`VMSRest`)
The original client that provides strongly-typed access to VAST Data resources:

```go
client, err := vast_client.NewVMSRest(config)
if err != nil {
    log.Fatal(err)
}

// Direct access to typed resources
users := client.Users
quotas := client.Quotas
views := client.Views
```

### Untyped Client (`UntypedVMSRest`)
A new client that provides dynamic access to VAST Data resources:

```go
client, err := vast_client.NewUntypedVMSRest(config)
if err != nil {
    log.Fatal(err)
}

// Method-based access to untyped resources
users := client.Users()
quotas := client.Quotas()
views := client.Views()
```

## 📚 Available Resources

Both client types provide access to the same VAST Data resources:

- **Users** - User management
- **Quotas** - Quota management
- **Views** - View management
- **VipPools** - VIP pool management
- **UserKeys** - User key management
- **Versions** - Version information
- **Snapshots** - Snapshot management
- **Volumes** - Volume management
- **BlockHosts** - Block host management
- **And many more...**

## 🔄 Context Support

Both client types support context for request cancellation and timeouts:

```go
ctx := context.Background()

// Typed client with context
typedClient, err := vast_client.NewVMSRestWithContext(ctx, config)

// Untyped client with context
untypedClient, err := vast_client.NewUntypedVMSRestWithContext(ctx, config)
```

## 🏗️ Development

### Building
```bash
# Build all modules
go build ./...

# Build specific module
go build ./core
go build ./resources
go build ./auth
go build ./codegen
```

### Testing
```bash
# Test all modules
go test ./...

# Test specific module
go test ./core
go test ./resources
```

## 📖 Examples

See the `examples/` directory for comprehensive usage examples:
- `examples/typed/` - Typed client examples
- `examples/untyped/` - Untyped client examples

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

For support and questions:
- Create an issue on GitHub
- Check the documentation in the `docs/` directory
- Review the examples in the `examples/` directory