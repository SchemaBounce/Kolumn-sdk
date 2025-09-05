# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the **Kolumn Provider SDK** - a Go SDK library for building external providers that integrate with Kolumn's infrastructure-as-code platform. The SDK follows Go best practices (like AWS SDK and other provider SDKs) as a **library**, not an application framework.

## Key Architecture Principles

### 1. **Library Pattern** (Not RPC Plugins)
- This SDK is imported as a Go library
- Provider developers create their own binaries and main.go files
- No cmd/ directories or main.go files in the SDK itself
- Follows standard Go SDK patterns (AWS SDK, HashiCorp Provider SDK, etc.)

### 2. **Simple Interface Design**
- Single 4-method `Provider` interface (clean and minimal)
- Documentation generated from `Schema()` method over RPC

### 3. **Create/Discover Object Categorization**
- **CREATE objects**: Resources providers can create and manage (tables, buckets, topics)
- **DISCOVER objects**: Existing infrastructure providers can find and analyze (schemas, performance issues)

### 4. **Clean Package Structure**
- `core/` - Core interfaces and types
- `create/` - CREATE object handler utilities
- `discover/` - DISCOVER object handler utilities  
- `examples/` - Usage examples and patterns

## Core Interface

### Provider Interface
```go
type Provider interface {
    Configure(ctx context.Context, config Config) error
    Schema() (*Schema, error)
    CallFunction(ctx context.Context, function string, input []byte) ([]byte, error)
    Close() error
}
```

The `Schema()` method returns all information needed for documentation generation over RPC.

## Handler Registry Pattern

### CREATE Object Handlers
```go
// CREATE objects implement CRUD operations
type ObjectHandler interface {
    Create(ctx context.Context, req *CreateRequest) (*CreateResponse, error)
    Read(ctx context.Context, req *ReadRequest) (*ReadResponse, error)
    Update(ctx context.Context, req *UpdateRequest) (*UpdateResponse, error)
    Delete(ctx context.Context, req *DeleteRequest) (*DeleteResponse, error)
    Plan(ctx context.Context, req *PlanRequest) (*PlanResponse, error)
}

// Register with CREATE registry
createRegistry := create.NewRegistry()
createRegistry.RegisterHandler("table", tableHandler, tableSchema)
```

### DISCOVER Object Handlers
```go
// DISCOVER objects implement discovery operations
type ObjectHandler interface {
    Scan(ctx context.Context, req *ScanRequest) (*ScanResponse, error)
    Analyze(ctx context.Context, req *AnalyzeRequest) (*AnalyzeResponse, error)
    Query(ctx context.Context, req *QueryRequest) (*QueryResponse, error)
}

// Register with DISCOVER registry  
discoverRegistry := discover.NewRegistry()
discoverRegistry.RegisterHandler("existing_tables", discoverer, discoverySchema)
```

## File Organization

### Core SDK Files
- `core/provider.go` (229 lines) - Core interfaces and progressive disclosure pattern
- `create/handler.go` (305 lines) - CREATE object handler interface and registry
- `discover/handler.go` (414 lines) - DISCOVER object handler interface and registry  

### Examples
- `examples/simple/provider.go` (510 lines) - Complete minimal provider demonstrating all SDK concepts
- `examples/simple/README.md` - Detailed explanation of patterns

### Documentation  
- `README.md` - Main SDK documentation for provider developers
- `CLAUDE.md` - This file (development guidance)

## Development Workflow

### Building the SDK
```bash
go build ./...
go test ./...
go mod tidy
```

### Testing Examples
```bash
cd examples/simple
go run provider.go
```

### Schema for Documentation
```go
// The Schema() method provides all documentation information
schema, err := provider.Schema()
// Schema contains CreateObjects, DiscoverObjects, examples, etc.
// Kolumn CLI calls this over RPC for documentation generation
```

## Provider Development Pattern

1. **Create provider project**: `mkdir kolumn-provider-name`
2. **Import SDK**: `go get github.com/schemabounce/kolumn/sdk`
3. **Implement Provider interface**: Start with 4-method interface
4. **Register object handlers**: Use create/discover registries
5. **Build provider binary**: `go build -o kolumn-provider-name`

## Important Notes for Development

### ✅ **Correct Patterns** (Follow These)
- SDK as Go library (import with `go get`)
- Simple 4-method Provider interface
- Create/discover object categorization
- Handler registration pattern
- Schema() method for documentation over RPC
- Provider creates own main.go in separate project

### ❌ **Incorrect Patterns** (Avoid These)
- RPC plugin architecture with HashiCorp go-plugin
- cmd/ directories in SDK with main.go files
- Function-based dispatch instead of object handlers
- SDKs that contain application binaries
- Monolithic interfaces with many required methods

## Documentation Philosophy

Documentation is generated from the `Schema()` method over RPC:

- **Schema-driven** - All documentation comes from provider's Schema() method
- **Create/discover categorization** of all object types  
- **Rich object definitions** with properties, examples, and validation
- **RPC-based generation** - Kolumn CLI calls Schema() to generate docs

## Testing Strategy

- Unit tests for all core interfaces
- Integration tests for registries and handlers
- Example validation to ensure patterns work
- Schema() method testing for documentation completeness

## Dependencies

- **Standard library only** for core functionality
- **Minimal external dependencies** following Go best practices
- **No RPC frameworks** (HashiCorp go-plugin removed)
- **Clean module structure** with clear dependency boundaries

This SDK enables clean, maintainable provider development following Go SDK best practices while providing powerful abstractions for the create/discover pattern and schema-driven documentation.

## IMPORTANT RESTRICTIONS

⚠️ **TERRAFORM REFERENCES PROHIBITED** ⚠️
- The word "terraform" or "Terraform" is **BANNED** from all code, documentation, comments, and examples
- This includes variable names, function names, package names, and any text content
- Use alternative terms: "infrastructure-as-code", "IaC", "configuration management", "provider SDK"
- When referencing similar tools, use "HashiCorp Provider SDK" or "infrastructure tools"

This restriction is critical to maintain product independence and avoid trademark/brand conflicts.