# Kolumn Provider SDK

**Go SDK for building Kolumn providers** - A clean, library-based toolkit for developing Kolumn data infrastructure providers.

## Overview

The Kolumn Provider SDK enables developers to build external providers that integrate with Kolumn's infrastructure-as-code platform. Following Go SDK best practices, this is a **library**, not an application framework.

### Key Features

- **🏗️ Library Pattern** - Import as Go library
- **🎯 Create/Discover Architecture** - Clear separation of concerns
- **✨ Simple 4-Method Interface** - Core RPC pattern with exactly 4 methods ([see details](./4-METHOD-PATTERN.md))
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
func (p *MyProvider) Configure(ctx context.Context, config map[string]interface{}) error { }
func (p *MyProvider) Schema() (*core.Schema, error) { }
func (p *MyProvider) CallFunction(ctx context.Context, function string, input []byte) ([]byte, error) { }
func (p *MyProvider) Close() error { }
```

### 2.1 (Optional) Read Auth Claims from Context

Kolumn core can pass validated authentication details into `ctx`. Use SDK helpers to read them:

```go
import sdkauth "github.com/schemabounce/kolumn/sdk/core/auth"

func (p *MyProvider) CallFunction(ctx context.Context, function string, input []byte) ([]byte, error) {
    if info, ok := sdkauth.FromAuth(ctx); ok {
        // info.Claims.Tier: community|pro|enterprise
        // info.Claims.Entitlements: e.g., ["governance"]
        // info.RawToken: bearer token (if provided)
        // Use for feature gating or audit only; core validates tokens.
    }
    // ...
}
```

### 3. Study the Example

See `examples/simple/provider.go` for a complete working example showing all patterns.

## Architecture

### Core Interface - The 4-Method RPC Pattern

**⚠️ CRITICAL: Providers must implement EXACTLY 4 methods - no more, no less.**

All providers implement this precise 4-method interface:

```go
type Provider interface {
    // Configure sets up the provider with configuration (RPC call 1)
    Configure(ctx context.Context, config map[string]interface{}) error

    // Schema returns provider capabilities and documentation (RPC call 2)
    Schema() (*Schema, error)

    // CallFunction handles unified resource operations (RPC call 3)
    CallFunction(ctx context.Context, function string, input []byte) ([]byte, error)

    // Close cleans up provider resources (RPC call 4)
    Close() error
}
```

**Why exactly 4 methods?**
- **Clean RPC interface** - Simple, predictable communication protocol
- **Core compatibility** - Matches Kolumn core's provider contract exactly
- **No method bloat** - Avoids interface pollution and complexity
- **Unified dispatch** - All operations route through `CallFunction` for consistency

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

## Configuration Validation

### ⚠️ Important: ValidateConfig Method Removed

**The ValidateConfig method was removed from the Provider interface to maintain the 4-method RPC pattern.**

**Why was it removed?**
- **Interface purity** - Keeps Provider interface to exactly 4 RPC methods
- **Internal concern** - Configuration validation should happen within `Configure()`
- **No RPC overhead** - Validation doesn't need separate network calls

### ✅ Validation Alternatives

**1. Validate within Configure() method:**
```go
func (p *MyProvider) Configure(ctx context.Context, config map[string]interface{}) error {
    // Use helper methods for validation
    if err := p.validateConfiguration(ctx, config); err != nil {
        return fmt.Errorf("configuration validation failed: %w", err)
    }

    // Apply configuration
    return p.applyConfig(config)
}

// Helper method for internal validation
func (p *MyProvider) validateConfiguration(ctx context.Context, config map[string]interface{}) error {
    validators := validation.CreateValidator{}.DatabaseConnectionConfig()
    return validation.ValidateConfig(config, validators)
}
```

**2. Use BaseProvider helper (optional):**
```go
type MyProvider struct {
    *core.BaseProvider
}

func (p *MyProvider) Configure(ctx context.Context, config map[string]interface{}) error {
    // Use BaseProvider validation helper
    if result := p.ValidateConfiguration(ctx, config); !result.IsValid() {
        return fmt.Errorf("configuration invalid: %v", result.Errors)
    }

    return p.applyConfig(config)
}
```

### Validation Helpers

Use the validation package for robust configuration validation:

```go
import "github.com/schemabounce/kolumn/sdk/helpers/validation"

// Validate database connection config
validators := validation.CreateValidator{}.DatabaseConnectionConfig()
err := validation.ValidateConfig(config, validators)
```

### 📚 Complete Validation Guide

For comprehensive validation patterns, migration guides, and best practices, see:
**[docs/VALIDATION_GUIDE.md](./docs/VALIDATION_GUIDE.md)**

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
