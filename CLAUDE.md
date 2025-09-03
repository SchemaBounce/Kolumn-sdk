# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the **Kolumn Provider SDK** - a Go SDK for building external providers that integrate with Kolumn's infrastructure-as-code platform. The SDK enables developers to create RPC-based plugins for database, storage, streaming, and other data stack technologies.

## Development Commands

### Building and Testing
```bash
# Build the SDK
make build

# Run all tests
make test

# Run tests with coverage
make test-coverage

# Format code
make fmt

# Run linting
make lint

# Run go vet
make vet

# Full validation (clean, deps, build, test, lint, vet, examples)
make validate

# Build examples
make examples
```

### Development Workflow
```bash
# Setup development environment (installs required tools)
make dev-setup

# Development build and test
make dev

# Watch for changes and run tests automatically
make watch

# CI validation pipeline
make ci
```

### Dependencies
```bash
# Download and verify dependencies
make deps
```

## Architecture

### Core Components

1. **RPC Plugin System** (`rpc/`) - ✅ Correct for provider SDK
   - `UniversalProvider` interface - Universal 4-method + 7 Kolumn-compatible methods
   - Plugin serving infrastructure via `rpc.ServeProvider()`
   - Both simplified and Kolumn-compatible method sets

2. **Provider Development Kit** (`pdk/`) - ✅ Correct for provider SDK
   - `BaseProvider` - Handles RPC plumbing and automatic CRUD dispatch
   - Resource handler registration system
   - Helper functions for validation, configuration parsing
   - Automatic function-to-handler routing

3. **Type System** (`types/`) - Shared types across providers
   - `ProviderSchema` - Provider capability declarations
   - `UniversalState` - Cross-provider state format
   - Configuration schema definitions

4. **Metadata System** (`metadata/`) - Data discovery and lineage
   - Metadata collection interfaces
   - Schema introspection support

### Provider Interface Pattern

Providers implement the `UniversalProvider` interface with 11 total methods:

**Core 4-method interface:**
- `Configure()` - Provider setup
- `GetSchema()` - Capability declaration
- `CallFunction()` - Function dispatch
- `Close()` - Cleanup

**Kolumn-compatible methods (7):**
- `ValidateProviderConfig()`, `ValidateResourceConfig()`
- `PlanResourceChange()`, `ApplyResourceChange()`
- `ReadResource()`, `ImportResourceState()`, `UpgradeResourceState()`

### Resource Handler Architecture

Providers register resource handlers with `BaseProvider`:

```go
provider.RegisterResourceHandler("table", &TableHandler{})
```

Handlers implement the `ResourceHandler` interface for CRUD operations.

## Key Files

- `pdk/provider_base.go` - Core provider implementation with automatic RPC handling
- `rpc/provider.go` - Universal provider interface definitions
- `examples/simple_provider.go` - Complete working provider example
- `cmd/simple_example/main.go` - Example provider binary
- `types/schema.go` - Schema and configuration type definitions

## Provider Development Pattern

1. Extend `BaseProvider` from `pdk` package
2. Register resource handlers for each resource type
3. Implement CRUD operations in handlers
4. Use `rpc.ServeProvider()` to serve as plugin
5. Build as binary: `go build -o kolumn-provider-name`

## Testing

- Use `make test` for full test suite
- Test files follow `*_test.go` convention
- Coverage reports generated in `build/coverage.html`
- Examples are validated during CI via `make examples`

## Dependencies

- **HashiCorp go-plugin** - RPC plugin framework
- **Testify** - Testing framework  
- **Go 1.24+** - Required Go version