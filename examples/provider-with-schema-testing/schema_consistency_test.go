package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/schemabounce/kolumn-sdk/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchemaImplementationConsistency demonstrates how to use the Kolumn SDK
// schema testing framework to ensure your provider's schema matches its implementation.
//
// This test MUST pass for pre-commit hooks and prevents schema drift.
func TestSchemaImplementationConsistency(t *testing.T) {
	// Create provider instance
	provider := NewExampleProvider()
	require.NotNil(t, provider)

	// Configure the test using the SDK testing framework
	config := &testing.SchemaTestConfig{
		Provider:     provider,
		StateAdapter: nil, // This example doesn't use a state adapter

		// Expected configuration fields from the Configure method
		// These MUST match exactly what your Configure() method accepts
		ExpectedConfigFields: map[string]testing.ConfigFieldExpectation{
			"endpoint": {
				Required:    false,
				FieldType:   "string",
				Description: "Database endpoint URL",
			},
			"api_key": {
				Required:    false,
				FieldType:   "string",
				Description: "API authentication key",
			},
			"timeout": {
				Required:    false,
				FieldType:   "string",
				Description: "Connection timeout duration",
			},
		},

		// Expected resource types that your provider implements
		// These MUST match the ResourceTypes in your Schema() method
		ExpectedResources: map[string]testing.ResourceExpectation{
			"example_table": {
				StateAdapterMethod: "", // No state adapter in this example
				Operations:         []string{"create", "read", "update", "delete"},
				Description:        "Database table resource",
			},
			"example_user": {
				StateAdapterMethod: "", // No state adapter in this example
				Operations:         []string{"create", "read", "update", "delete"},
				Description:        "Database user resource",
			},
		},

		// Expected functions that your CallFunction method can handle
		// These MUST match the SupportedFunctions in your Schema() method
		ExpectedFunctions: []string{
			"create_table", "read_table", "update_table", "delete_table",
			"create_user", "read_user", "update_user", "delete_user",
		},
	}

	// Run the comprehensive schema consistency tests
	testing.RunSchemaConsistencyTests(t, config)
}

// TestSchemaConsistencyPreCommitHook is the entry point for pre-commit hook testing.
// This test MUST pass for commits to be allowed.
func TestSchemaConsistencyPreCommitHook(t *testing.T) {
	// This is the same test as TestSchemaImplementationConsistency but with
	// a clear name for pre-commit hook usage
	TestSchemaImplementationConsistency(t)
}

// TestProviderBasicFunctionality demonstrates basic provider testing
// beyond just schema consistency
func TestProviderBasicFunctionality(t *testing.T) {
	provider := NewExampleProvider()
	require.NotNil(t, provider)

	t.Run("Schema", func(t *testing.T) {
		schema, err := provider.Schema()
		require.NoError(t, err)
		require.NotNil(t, schema)

		assert.Equal(t, "example", schema.Name)
		assert.Equal(t, "1.0.0", schema.Version)
		assert.Equal(t, "rpc", schema.Protocol)
		assert.Equal(t, "database", schema.Type)

		// Verify we have the expected number of functions
		assert.Len(t, schema.SupportedFunctions, 8)

		// Verify we have the expected number of resource types
		assert.Len(t, schema.ResourceTypes, 2)
	})

	t.Run("Configure", func(t *testing.T) {
		config := map[string]interface{}{
			"endpoint": "localhost:5432",
			"api_key":  "test-key",
			"timeout":  "30s",
		}

		err := provider.Configure(context.Background(), config)
		assert.NoError(t, err)

		// Verify configuration was applied
		assert.Equal(t, "localhost:5432", provider.endpoint)
		assert.Equal(t, "test-key", provider.apiKey)
		assert.Equal(t, "30s", provider.timeout)
	})

	t.Run("CallFunction", func(t *testing.T) {
		ctx := context.Background()

		// Test create_table function
		input := `{"name": "test_table", "schema": "public"}`
		result, err := provider.CallFunction(ctx, "create_table", json.RawMessage(input))
		require.NoError(t, err)
		require.NotNil(t, result)

		// Parse result
		var response map[string]interface{}
		err = json.Unmarshal(result, &response)
		require.NoError(t, err)

		assert.Equal(t, "table_test_table", response["id"])
		assert.Equal(t, "test_table", response["name"])
		assert.NotEmpty(t, response["created_at"])
	})

	t.Run("UnsupportedFunction", func(t *testing.T) {
		ctx := context.Background()
		input := `{"name": "test"}`

		_, err := provider.CallFunction(ctx, "unsupported_function", json.RawMessage(input))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported function")
	})
}

// TestSchemaValidation demonstrates additional schema validation tests
func TestSchemaValidation(t *testing.T) {
	provider := NewExampleProvider()
	schema, err := provider.Schema()
	require.NoError(t, err)

	t.Run("ConfigSchemaValidation", func(t *testing.T) {
		// Validate that ConfigSchema is valid JSON
		var configSchema map[string]interface{}
		err := json.Unmarshal(schema.ConfigSchema, &configSchema)
		require.NoError(t, err)

		// Verify properties exist
		properties, ok := configSchema["properties"].(map[string]interface{})
		require.True(t, ok)
		require.NotEmpty(t, properties)

		// Verify specific fields
		endpoint, ok := properties["endpoint"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "string", endpoint["type"])
		assert.Contains(t, endpoint["description"], "endpoint")
	})

	t.Run("ResourceSchemaValidation", func(t *testing.T) {
		for _, resourceType := range schema.ResourceTypes {
			t.Run(resourceType.Name, func(t *testing.T) {
				// Validate ConfigSchema is valid JSON
				var configSchema map[string]interface{}
				err := json.Unmarshal(resourceType.ConfigSchema, &configSchema)
				require.NoError(t, err)

				// Validate StateSchema is valid JSON
				var stateSchema map[string]interface{}
				err = json.Unmarshal(resourceType.StateSchema, &stateSchema)
				require.NoError(t, err)

				// Verify basic structure
				assert.NotEmpty(t, resourceType.Name)
				assert.NotEmpty(t, resourceType.Description)
				assert.NotEmpty(t, resourceType.Operations)
			})
		}
	})
}
