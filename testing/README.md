# Kolumn SDK Testing Framework

The Kolumn SDK Testing Framework provides comprehensive schema consistency testing for Kolumn providers. This framework ensures that your provider's schema exactly matches its implementation, preventing schema drift and ensuring reliability across the Kolumn ecosystem.

## Features

- **Configuration Schema Validation** - Ensures all `Configure()` parameters are documented
- **Resource Schema Consistency** - Validates resource type definitions match implementation
- **Function Coverage Testing** - Verifies all implemented functions are exposed
- **Handler Routing Validation** - Tests that `CallFunction()` can route all supported functions
- **State Adapter Integration** - Optional deep validation with state adapters
- **Pre-commit Hook Support** - Automatic validation on every commit

## Quick Start

1. **Add the dependency**:
   ```go
   import "github.com/schemabounce/kolumn-sdk/testing"
   ```

2. **Create a test file**:
   ```go
   func TestSchemaConsistency(t *testing.T) {
       provider := NewMyProvider()
       config := &testing.SchemaTestConfig{
           Provider: provider,
           ExpectedConfigFields: map[string]testing.ConfigFieldExpectation{
               "endpoint": {FieldType: "string", Description: "API endpoint"},
           },
           ExpectedResources: map[string]testing.ResourceExpectation{
               "my_resource": {Operations: []string{"create", "read", "update", "delete"}},
           },
           ExpectedFunctions: []string{"create_resource", "read_resource"},
       }
       testing.RunSchemaConsistencyTests(t, config)
   }
   ```

3. **Set up pre-commit hooks** using the template in `/templates/`

## Documentation

- [Complete Documentation](../docs/SCHEMA_TESTING.md) - Comprehensive guide with examples
- [Templates](../templates/) - Ready-to-use templates for providers
- [Examples](../../examples/) - Working examples from real providers

## Interface Requirements

Your provider must implement `testing.SchemaProvider`:

```go
type SchemaProvider interface {
    Schema() (*ProviderSchema, error)
    Configure(ctx context.Context, config map[string]interface{}) error
    CallFunction(ctx context.Context, function string, input json.RawMessage) (json.RawMessage, error)
}
```

## Benefits

- **Prevents Schema Drift** - Catches when implementation changes but schema doesn't
- **Improves Reliability** - Ensures users see accurate schema information
- **Enforces Standards** - Maintains consistency across all Kolumn providers
- **Catches Issues Early** - Pre-commit hooks prevent problematic commits
- **Reduces Debugging** - Clear test failures guide developers to fixes

## Integration

The framework integrates with:
- **Pre-commit hooks** - Automatic validation
- **CI/CD pipelines** - Continuous integration testing
- **Development workflow** - Fast feedback during development
- **SDK ecosystem** - Consistent patterns across all providers

For complete documentation and examples, see [SCHEMA_TESTING.md](../docs/SCHEMA_TESTING.md).