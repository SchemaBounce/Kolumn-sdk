# CLAUDE.md — Kolumn Provider SDK

## Overview

Go SDK library for building external providers that integrate with Kolumn's infrastructure-as-code platform. Providers are standalone binaries following Go library patterns (like AWS SDK).

## Core Interface (4 Methods — No More, No Less)

```go
type Provider interface {
    Configure(ctx context.Context, config map[string]interface{}) error
    Schema() (*Schema, error)
    CallFunction(ctx context.Context, function string, input []byte) ([]byte, error)
    Close() error
}
```

All resource operations route through `CallFunction` via unified dispatch: `CreateResource`, `ReadResource`, `UpdateResource`, `DeleteResource`, `DiscoverResources`, `Ping`.

## Build & Test

```bash
go build ./...
go test -race ./...
go vet ./...
go mod tidy
```

## Project Structure

```
core/           Core interfaces and types (Provider, Schema)
create/         CREATE object handler registry (CRUD operations)
discover/       DISCOVER object handler registry (scan/analyze)
helpers/
  ui/           Human-readable output formatting and colors
  logging/      Structured logging for providers
  security/     SafeUnmarshal, SecureError, input validation
  validation/   Config validation helpers
  quarantine/   Safe destroy quarantine framework
enterprise_safety/  Backup, cascade, and safety frameworks
runtime/        Handler registry types
runtimehelpers/ SQL runner, telemetry, test harness
state/          State adapter and backend helpers
examples/       Working provider examples
docs/           Provider Specification and guides
```

## Key Patterns

### Handler Registration
```go
createRegistry := create.NewRegistry()
createRegistry.RegisterHandler("table", tableHandler, tableSchema)

dispatcher := core.NewUnifiedDispatcher(createRegistry, discoverRegistry)
response, err := dispatcher.Dispatch(ctx, "CreateResource", input)
```

### Create vs Discover
- **CREATE objects**: Resources the provider manages (tables, indexes, views)
- **DISCOVER objects**: Existing infrastructure the provider inspects (schemas, performance)

### ValidateConfig Removed
Validation is internal to `Configure()` — keeps the 4-method interface clean. Use `BaseProvider.ValidateConfiguration()` or `Schema.ValidateConfig()` helpers.

## Provider Binary Naming

All provider binaries must follow `kolumn-provider-{name}` for automatic discovery.

## Provider Specification

The authoritative spec for all providers: `docs/PROVIDER_SPECIFICATION.md`

## Important Rules

- **No simulation code** — all operations must work against real infrastructure
- **Context as first parameter** in all functions
- **Always check and wrap errors** — never ignore with `_`
- **Exported functions require GoDoc comments**
- **4-method interface is inviolable** — do not add methods to Provider interface
