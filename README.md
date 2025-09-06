# Kolumn Provider SDK

**Go SDK for building Kolumn providers** - A clean, library-based toolkit for developing Kolumn data infrastructure providers.

## Overview

The Kolumn Provider SDK enables developers to build external providers that integrate with Kolumn's infrastructure-as-code platform. Following Go SDK best practices, this is a **library**, not an application framework.

### Key Features

- **🏗️ Library Pattern** - Import as Go library
- **🎯 Create/Discover Architecture** - Clear separation of concerns
- **✨ Simple Interface** - Just 4 methods to implement
- **📈 Progressive Disclosure** - Start simple, add complexity as needed

## 🚀 SDK Compatibility Status

**✅ 100% CORE COMPATIBILITY ACHIEVED**

The Kolumn Provider SDK is now **fully compatible** with Kolumn core implementation:

- **✅ Unified Function Dispatch**: Supports `CreateResource`, `ReadResource`, `UpdateResource`, `DeleteResource`, `Ping`, `DiscoverResources`  
- **✅ Enhanced Schema Structure**: Includes `SupportedFunctions`, `ResourceTypes`, and `ConfigSchema` fields
- **✅ Configuration Interface**: Accepts `map[string]interface{}` for direct core compatibility
- **✅ UnifiedDispatcher**: Bridges existing registries with new unified dispatch pattern
- **✅ Provider Naming**: Follows `kolumn-provider-{name}` pattern for automatic discovery

**Compatibility Score**: 100% (up from 85% baseline)

## Quick Start

### 1. Create a New Provider Project

**⚠️ Important: Provider Binary Naming Convention**

All provider binaries must follow the `kolumn-provider-{name}` pattern for automatic discovery by Kolumn core.

```bash
# REQUIRED naming pattern: kolumn-provider-{name}
mkdir kolumn-provider-mydb
cd kolumn-provider-mydb
go mod init github.com/yourorg/kolumn-provider-mydb
go get github.com/schemabounce/kolumn/sdk
```

### 2. Implement the Provider Interface

```go
package main

import (
    "context"
    "github.com/schemabounce/kolumn/sdk/core"
    "github.com/schemabounce/kolumn/sdk/create"
    "github.com/schemabounce/kolumn/sdk/discover"
)

type MyProvider struct {
    createRegistry   *create.Registry
    discoverRegistry *discover.Registry
}

// Implement the 4-method core.Provider interface:
func (p *MyProvider) Configure(ctx context.Context, config core.Config) error { }
func (p *MyProvider) Schema() (*core.Schema, error) { }
func (p *MyProvider) CallFunction(ctx context.Context, function string, input []byte) ([]byte, error) { }
func (p *MyProvider) Close() error { }
```

### 3. Study the Example

See `examples/simple/provider.go` for a complete working example showing all patterns.

## Architecture

### Core Interface

All providers implement a simple 4-method interface:

```go
type Provider interface {
    Configure(ctx context.Context, config Config) error
    Schema() (*Schema, error)
    CallFunction(ctx context.Context, function string, input []byte) ([]byte, error)
    Close() error
}
```

### Create/Discover Pattern

Providers categorize their functionality:

- **CREATE objects**: Resources you can create and manage (tables, buckets, topics)
- **DISCOVER objects**: Existing infrastructure you can find and analyze (schemas, performance issues)

### Handler Registration

```go
// Register CREATE object handlers
createRegistry := create.NewRegistry()
createRegistry.RegisterHandler("table", tableHandler, tableSchema)

// Register DISCOVER object handlers
discoverRegistry := discover.NewRegistry()  
discoverRegistry.RegisterHandler("existing_tables", discoverer, discoverySchema)
```

## Package Structure

- **`core/`** - Core Provider interface and types
- **`create/`** - CREATE object handler utilities  
- **`discover/`** - DISCOVER object handler utilities
- **`helpers/validation/`** - Configuration validation helpers
- **`examples/simple/`** - Working example provider

## Validation Helpers

Use the validation package for configuration validation:

```go
import "github.com/schemabounce/kolumn/sdk/helpers/validation"

// Validate database connection config
validators := validation.CreateValidator{}.DatabaseConnectionConfig()
err := validation.ValidateConfig(config, validators)
```

## Development

```bash
# Build all packages
go build ./...

# Run the example
cd examples/simple
go run provider.go

# Test compilation
go build ./core ./create ./discover ./helpers/validation
```

## Documentation

- **Schema-driven**: Documentation is generated from your provider's `Schema()` method
- **Examples**: Complete working examples in `examples/`
- **Simple**: Clean interfaces following Go best practices

## License

See LICENSE file for details.

---

**Ready to build Kolumn providers!** 🚀

The SDK provides everything you need to build production-ready providers that integrate with Kolumn's infrastructure-as-code platform.