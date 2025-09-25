# Schema Consistency Testing for Kolumn Providers

This document explains how to implement and use the schema consistency testing framework for Kolumn providers. This testing framework ensures that your provider's `Schema()` method returns schemas that exactly match your implementation, preventing schema drift and ensuring reliability.

## Overview

The schema consistency testing framework validates:

1. **Configuration Schema Consistency** - All configuration fields accepted by `Configure()` are documented in the schema
2. **Resource Schema Consistency** - All resource types and their properties are correctly exposed
3. **Function Coverage Consistency** - All implemented functions are listed in `SupportedFunctions`
4. **Handler Routing Consistency** - All supported functions can be successfully routed by `CallFunction()`

## Quick Start

### 1. Copy the Test Template

Copy `/templates/schema_consistency_test.go` to your provider repository and customize it:

```go
package main

import (
	"testing"
	"github.com/stretchr/testify/require"
	"github.com/schemabounce/kolumn-sdk/testing"
)

func TestSchemaImplementationConsistency(t *testing.T) {
	provider := NewMyProvider() // Replace with your provider constructor
	require.NotNil(t, provider)

	config := &testing.SchemaTestConfig{
		Provider: provider,
		ExpectedConfigFields: map[string]testing.ConfigFieldExpectation{
			"endpoint": {Required: false, FieldType: "string", Description: "API endpoint"},
			"api_key":  {Required: false, FieldType: "string", Description: "API key"},
		},
		ExpectedResources: map[string]testing.ResourceExpectation{
			"my_table": {
				Operations:  []string{"create", "read", "update", "delete"},
				Description: "Database table resource",
			},
		},
		ExpectedFunctions: []string{
			"create_table", "read_table", "update_table", "delete_table",
		},
	}

	testing.RunSchemaConsistencyTests(t, config)
}

func TestSchemaConsistencyPreCommitHook(t *testing.T) {
	TestSchemaImplementationConsistency(t)
}
```

### 2. Set Up Pre-commit Hooks

Copy `/templates/pre-commit-config.yaml` to `.pre-commit-config.yaml` in your provider repository:

```yaml
repos:
  # ... other hooks ...
  - repo: local
    hooks:
      - id: schema-consistency-check
        name: Validate provider schema matches implementation
        entry: sh -c 'go test . -run TestSchemaConsistencyPreCommitHook -v'
        language: system
        files: .*\.go$
        pass_filenames: false
```

### 3. Install Pre-commit

```bash
pip install pre-commit
pre-commit install
```

## Interface Requirements

Your provider must implement the `testing.SchemaProvider` interface:

```go
type SchemaProvider interface {
	Schema() (*ProviderSchema, error)
	Configure(ctx context.Context, config map[string]interface{}) error
	CallFunction(ctx context.Context, function string, input json.RawMessage) (json.RawMessage, error)
}
```

## Configuration

### ExpectedConfigFields

Define all configuration fields that your `Configure()` method accepts:

```go
ExpectedConfigFields: map[string]testing.ConfigFieldExpectation{
	"endpoint": {
		Required:    false,
		FieldType:   "string",
		Description: "API endpoint",
	},
	"timeout": {
		Required:    false,
		FieldType:   "string",
		Description: "Request timeout",
	},
	"retries": {
		Required:    false,
		FieldType:   "number",
		Description: "Number of retries",
	},
}
```

### ExpectedResources

Define all resource types your provider supports:

```go
ExpectedResources: map[string]testing.ResourceExpectation{
	"postgres_table": {
		Operations:  []string{"create", "read", "update", "delete"},
		Description: "PostgreSQL table",
	},
	"postgres_user": {
		Operations:  []string{"create", "read", "update", "delete"},
		Description: "PostgreSQL user",
	},
}
```

### ExpectedFunctions

List all functions your provider's `CallFunction()` method can handle:

```go
ExpectedFunctions: []string{
	"create_table", "read_table", "update_table", "delete_table",
	"create_user", "read_user", "update_user", "delete_user",
	"create_index", "read_index", "update_index", "delete_index",
}
```

## State Adapter Integration (Optional)

If your provider has a state adapter for detailed schema management, you can include it:

```go
// Your state adapter must implement this interface
type StateAdapter interface {
	GetResourceSchema(resourceType string) (*ResourceStateSchema, error)
}

// In your test configuration
config := &testing.SchemaTestConfig{
	Provider:     provider,
	StateAdapter: stateAdapter, // Optional
	ExpectedResources: map[string]testing.ResourceExpectation{
		"my_table": {
			StateAdapterMethod: "getTableSchema", // Method name in state adapter
			Operations:         []string{"create", "read", "update", "delete"},
			Description:        "Database table",
		},
	},
	// ... other config
}
```

## Common Issues and Solutions

### Issue: Configuration Field Missing from Schema

```
Configuration field 'timeout' is expected but missing from schema
```

**Solution**: Add the field to your provider's `ConfigSchema`:

```go
ConfigSchema: json.RawMessage(`{
	"type": "object",
	"properties": {
		"timeout": {"type": "string", "description": "Request timeout"},
		"endpoint": {"type": "string", "description": "API endpoint"}
	}
}`),
```

### Issue: Function Routing Error

```
Function 'create_table' failed with routing error: unsupported resource type
```

**Solution**: Check your `CallFunction()` implementation's routing logic. Make sure it can handle the function name format.

### Issue: Resource Schema Mismatch

```
Resource 'my_table': Field 'created_at' is in state adapter but missing from exposed schema
```

**Solution**: Update your resource's `StateSchema` to include all fields from your state adapter.

## Best Practices

### 1. Run Tests Frequently

Run the schema consistency tests regularly during development:

```bash
go test . -run TestSchemaImplementationConsistency -v
```

### 2. Update Tests When Adding Features

When you add new configuration fields, resources, or functions, update the test expectations immediately.

### 3. Use Descriptive Names

Use clear, descriptive names for your configuration fields and resources that match your documentation.

### 4. Validate Real Usage

The tests use minimal test inputs. Consider adding integration tests that use realistic data.

### 5. Version Your Schemas

When making breaking changes to your schemas, consider versioning:

```go
schema := &testing.ProviderSchema{
	Name:    "my-provider",
	Version: "2.0.0", // Increment for breaking changes
	// ...
}
```

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Schema Consistency Check
on: [push, pull_request]

jobs:
  schema-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      - name: Run schema consistency tests
        run: go test . -run TestSchemaConsistencyPreCommitHook -v
```

### Pre-commit Integration

The pre-commit hook will automatically run on every commit and block commits that have schema inconsistencies.

## Troubleshooting

### Debug Test Failures

Run with verbose output to see detailed failure information:

```bash
go test . -run TestSchemaConsistencyPreCommitHook -v
```

### Skip State Adapter Tests

If you don't have a state adapter, set it to `nil`:

```go
config := &testing.SchemaTestConfig{
	Provider:     provider,
	StateAdapter: nil, // No state adapter
	// ... other config
}
```

### Custom Validation

For complex validation needs, you can extend the framework by implementing additional test functions following the same patterns used in the SDK.

## Examples

See the Kolumn provider implementation at `/mnt/c/git/Kolumn/providers/kolumn/schema_consistency_test.go` for a complete example of how the framework is used in practice.

## Support

For questions or issues with the schema testing framework:

1. Check the examples in the Kolumn repository
2. Review this documentation
3. File an issue in the Kolumn SDK repository