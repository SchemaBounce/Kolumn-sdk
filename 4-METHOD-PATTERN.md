# The 4-Method RPC Pattern

## Overview

The Kolumn Provider SDK enforces a **strict 4-method RPC interface** for all providers. This architectural constraint ensures:

- **Clean RPC communication** with Kolumn core
- **Interface consistency** across all providers
- **No method bloat** or unnecessary complexity
- **Performance optimization** by avoiding extra RPC calls

## The 4 Required Methods

```go
type Provider interface {
    // RPC Method 1: Configure the provider
    Configure(ctx context.Context, config map[string]interface{}) error

    // RPC Method 2: Return provider schema and capabilities
    Schema() (*Schema, error)

    // RPC Method 3: Handle all resource operations via unified dispatch
    CallFunction(ctx context.Context, function string, input []byte) ([]byte, error)

    // RPC Method 4: Clean up provider resources
    Close() error
}
```

## ⚠️ What Was Removed: ValidateConfig

**ValidateConfig was intentionally removed** to maintain the 4-method pattern.

### ❌ Old Pattern (REMOVED)
```go
// This 5th method violated the 4-method pattern and was removed
func (p *Provider) ValidateConfig(ctx context.Context, config map[string]interface{}) *ConfigValidationResult
```

### ✅ New Pattern (RECOMMENDED)
```go
// Validation now happens within Configure()
func (p *Provider) Configure(ctx context.Context, config map[string]interface{}) error {
    // Validate configuration first
    if err := p.validateConfiguration(ctx, config); err != nil {
        return fmt.Errorf("configuration validation failed: %w", err)
    }

    // Apply configuration after validation
    return p.applyConfig(config)
}
```

## Why This Matters

1. **Core Compatibility** - Kolumn core expects exactly 4 RPC methods
2. **Interface Purity** - Clean, minimal interface without bloat
3. **Performance** - No separate validation RPC calls
4. **Architectural Consistency** - All providers follow same pattern

## Resources

- **[README.md](./README.md)** - Full SDK documentation
- **[CLAUDE.md](./CLAUDE.md)** - Development guidance
- **[docs/VALIDATION_GUIDE.md](./docs/VALIDATION_GUIDE.md)** - Comprehensive validation patterns
- **[examples/simple/provider.go](./examples/simple/provider.go)** - Working example

## Quick Migration

If you have existing ValidateConfig methods:

1. **Move validation logic** into `Configure()` method
2. **Use validation helpers** from `helpers/validation/` package
3. **Test configuration** during Configure, not separately
4. **Return errors** directly from Configure for invalid config

See [docs/VALIDATION_GUIDE.md](./docs/VALIDATION_GUIDE.md) for detailed migration examples.