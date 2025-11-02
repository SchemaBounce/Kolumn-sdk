# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the **Kolumn Provider SDK** - a Go SDK library for building external providers that integrate with Kolumn's infrastructure-as-code platform. The SDK follows Go best practices (like AWS SDK and other provider SDKs) as a **library**, not an application framework.

**🚀 100% CORE COMPATIBILITY ACHIEVED** - The SDK is now fully compatible with Kolumn core implementation, supporting unified function dispatch, enhanced schema structure, and standardized configuration interface.

## Key Architecture Principles

### 1. **Library Pattern** (Not RPC Plugins)
- This SDK is imported as a Go library
- Provider developers create their own binaries and main.go files
- No cmd/ directories or main.go files in the SDK itself
- Follows standard Go SDK patterns (AWS SDK, HashiCorp Provider SDK, etc.)

### 2. **4-Method RPC Interface Design**
- **EXACTLY 4 methods** - Configure, Schema, CallFunction, Close (enforced architectural constraint)
- **No method bloat** - ValidateConfig was intentionally removed to maintain interface purity
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
    // Updated to accept map[string]interface{} for core compatibility
    Configure(ctx context.Context, config map[string]interface{}) error
    Schema() (*Schema, error)
    // Updated to support unified dispatch functions: CreateResource, ReadResource, etc.
    CallFunction(ctx context.Context, function string, input []byte) ([]byte, error)
    Close() error
}
```

**Key Updates for Core Compatibility:**
- `Configure()` now accepts `map[string]interface{}` directly (no longer uses Config interface)
- `CallFunction()` supports unified dispatch: `CreateResource`, `ReadResource`, `UpdateResource`, `DeleteResource`, `DiscoverResources`, `Ping`
- `Schema()` returns enhanced schema with `SupportedFunctions`, `ResourceTypes`, and `ConfigSchema` fields

The `Schema()` method returns all information needed for documentation generation over RPC.

## ⚠️ CRITICAL: ValidateConfig Method Removal

**ARCHITECTURAL DECISION: ValidateConfig was intentionally removed from the Provider interface.**

### Why ValidateConfig Was Removed

**Problem:** ValidateConfig violated the 4-method RPC pattern by adding a 5th method:
```go
// ❌ INCORRECT - This was the 5-method interface (REMOVED)
type Provider interface {
    Configure(ctx context.Context, config map[string]interface{}) error    // RPC 1
    Schema() (*Schema, error)                                              // RPC 2
    ValidateConfig(ctx context.Context, config map[string]interface{}) ... // RPC 3 (REMOVED)
    CallFunction(ctx context.Context, function string, input []byte) ...   // RPC 4
    Close() error                                                          // RPC 5
}
```

**Solution:** Validation moved to internal helpers, maintaining 4-method pattern:
```go
// ✅ CORRECT - Clean 4-method RPC interface
type Provider interface {
    Configure(ctx context.Context, config map[string]interface{}) error  // RPC 1
    Schema() (*Schema, error)                                            // RPC 2
    CallFunction(ctx context.Context, function string, input []byte) ... // RPC 3
    Close() error                                                        // RPC 4
}
```

### Architecture Benefits

1. **Interface Purity** - Exactly 4 RPC methods, no bloat
2. **Core Compatibility** - Matches Kolumn core's provider contract
3. **Performance** - No separate RPC call overhead for validation
4. **Separation of Concerns** - Validation is internal implementation detail

### Migration Guide for Providers

**Old Pattern (DEPRECATED):**
```go
func (p *Provider) ValidateConfig(ctx context.Context, config map[string]interface{}) *ConfigValidationResult {
    // This method no longer exists in interface
}
```

**New Pattern (RECOMMENDED):**
```go
func (p *Provider) Configure(ctx context.Context, config map[string]interface{}) error {
    // Validate internally within Configure
    if err := p.validateConfiguration(ctx, config); err != nil {
        return fmt.Errorf("configuration validation failed: %w", err)
    }

    // Apply configuration
    return p.applyConfig(config)
}

// Helper method (not part of interface)
func (p *Provider) validateConfiguration(ctx context.Context, config map[string]interface{}) error {
    // Use SDK validation helpers
    return validation.ValidateConfig(config, p.getValidators())
}
```

### Available Validation Helpers

1. **BaseProvider.ValidateConfiguration()** - Helper method for internal use
2. **Schema.ValidateConfig()** - Schema-based validation
3. **validation.ValidateConfig()** - Flexible validation framework

**Note:** These are helper methods, NOT part of the Provider interface.

## Unified Function Dispatch (Core Compatibility)

### UnifiedDispatcher Pattern
The SDK now includes `UnifiedDispatcher` to bridge existing registry patterns with core's unified function dispatch:

```go
// Create unified dispatcher from existing registries
dispatcher := core.NewUnifiedDispatcher(createRegistry, discoverRegistry)

// Handle core function calls
response, err := dispatcher.Dispatch(ctx, "CreateResource", input)

// Build core-compatible schema
schema := dispatcher.BuildCompatibleSchema(name, version, providerType, description)
```

**Supported Unified Functions:**
- `CreateResource` - Routes to CREATE registry's create method
- `ReadResource` - Routes to CREATE registry's read method  
- `UpdateResource` - Routes to CREATE registry's update method
- `DeleteResource` - Routes to CREATE registry's delete method
- `DiscoverResources` - Routes to DISCOVER registry's scan method
- `Ping` - Returns health status

**Request Format Transformation:**
- Unified format uses `resource_type` field
- Registry format uses `object_type` field
- UnifiedDispatcher automatically transforms between formats

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

## Security Features

The SDK includes comprehensive security hardening across all operations:

### Security Measures
- **SafeUnmarshal**: All JSON unmarshaling uses `security.SafeUnmarshal` with size and depth limits
- **ValidateObjectType**: Resource types validated against security criteria before processing
- **InputSizeValidator**: Configuration size limits enforced to prevent DoS attacks
- **SecureError**: All errors use `security.NewSecureError` for consistent, safe error handling
- **Request Validation**: All unified dispatch handlers validate requests before processing

### Security by Handler
Each unified dispatch handler includes:
```go
// Example: CreateResource handler security
func (d *UnifiedDispatcher) handleCreateResource(ctx context.Context, input []byte) ([]byte, error) {
    // 1. Safe unmarshaling with limits
    var unifiedReq map[string]interface{}
    if err := security.SafeUnmarshal(input, &unifiedReq); err != nil {
        return nil, security.NewSecureError("invalid request format", ..., "INVALID_REQUEST")
    }
    
    // 2. Resource type validation
    if err := security.ValidateObjectType(resourceType); err != nil {
        return nil, security.NewSecureError("invalid resource type", ..., "INVALID_RESOURCE_TYPE")
    }
    
    // 3. Configuration size validation
    validator := &security.InputSizeValidator{}
    if err := validator.ValidateConfigSize(config); err != nil {
        return nil, security.NewSecureError("request too large", ..., "REQUEST_TOO_LARGE")
    }
}
```

### Secure Configuration
- `SecureConfig` automatically marks sensitive fields (password, secret, token, key, credential)
- `GetSanitized()` method for safe logging without exposing secrets
- Enhanced validation for sensitive field requirements

## Provider Development Pattern

### Required Binary Naming Convention
**⚠️ CRITICAL**: All provider binaries must follow the `kolumn-provider-{name}` pattern for automatic discovery by Kolumn core.

1. **Create provider project**: `mkdir kolumn-provider-mydb` (note the required naming)
2. **Import SDK**: `go get github.com/schemabounce/kolumn/sdk`
3. **Implement Provider interface**: Start with 4-method interface (new signature for Configure)
4. **Register object handlers**: Use create/discover registries
5. **Add unified dispatch**: Use UnifiedDispatcher for core compatibility
6. **Build provider binary**: `go build -o kolumn-provider-mydb` (matching the directory name)

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