# Example Kolumn Provider with Schema Testing

This example demonstrates how to build a Kolumn provider with comprehensive schema consistency testing using the Kolumn SDK testing framework.

## Features Demonstrated

- **Complete Provider Implementation** - Shows all required interfaces
- **Schema Consistency Testing** - Validates schema matches implementation
- **Pre-commit Hook Integration** - Automatic validation on commits
- **Function Routing** - Proper `CallFunction` implementation
- **Configuration Handling** - Standard provider configuration patterns

## Files

- `main.go` - Complete provider implementation
- `schema_consistency_test.go` - Schema testing demonstration
- `.pre-commit-config.yaml` - Pre-commit hook configuration
- `go.mod` - Go module definition

## Provider Features

### Resources
- `example_table` - Database table management
- `example_user` - User management

### Operations
- Full CRUD operations for all resources
- Proper error handling and validation
- JSON-based configuration and state

### Configuration
- `endpoint` - Database endpoint URL
- `api_key` - API authentication key
- `timeout` - Connection timeout duration

## Running the Example

### 1. Install Dependencies
```bash
go mod tidy
```

### 2. Run the Provider
```bash
go run main.go
```

### 3. Run Schema Consistency Tests
```bash
go test . -run TestSchemaImplementationConsistency -v
```

### 4. Run All Tests
```bash
go test . -v
```

### 5. Set Up Pre-commit Hooks
```bash
pip install pre-commit
pre-commit install
pre-commit run --all-files
```

## Expected Output

### Schema Consistency Test
```
=== RUN   TestSchemaImplementationConsistency
=== RUN   TestSchemaImplementationConsistency/ConfigurationSchemaConsistency
=== RUN   TestSchemaImplementationConsistency/ResourceSchemaConsistency
=== RUN   TestSchemaImplementationConsistency/FunctionCoverageConsistency
=== RUN   TestSchemaImplementationConsistency/HandlerRoutingConsistency
--- PASS: TestSchemaImplementationConsistency (0.00s)
```

### Provider Execution
```
Example provider created: &{endpoint: apiKey: timeout:}
2024/01/01 00:00:00 Configured example provider: endpoint=localhost:5432, timeout=30s
Provider schema: example v1.0.0
Supported functions: [create_table read_table update_table delete_table create_user read_user update_user delete_user]
```

## Schema Testing Benefits

This example shows how the schema testing framework:

1. **Prevents Schema Drift** - Catches when implementation changes but schema doesn't
2. **Validates Configuration** - Ensures all `Configure()` parameters are documented
3. **Tests Function Routing** - Verifies `CallFunction()` can handle all supported functions
4. **Enforces Standards** - Maintains consistency across provider implementations

## Customization

To adapt this example for your provider:

1. **Update Resource Types** - Replace `example_table` and `example_user` with your resources
2. **Modify Configuration** - Change configuration fields to match your needs
3. **Implement Business Logic** - Add real implementation in the handler methods
4. **Update Tests** - Modify test expectations to match your implementation

## Integration

This pattern can be used in any Kolumn provider by:

1. Copying the test structure from `schema_consistency_test.go`
2. Using the pre-commit configuration from `.pre-commit-config.yaml`
3. Following the provider interface patterns from `main.go`
4. Adapting the schema definitions to match your resources

## Next Steps

- See the [full documentation](../../docs/SCHEMA_TESTING.md) for advanced usage
- Review the [template files](../../templates/) for quick setup
- Explore the [Kolumn provider](../../../Kolumn/providers/kolumn/) for a production example