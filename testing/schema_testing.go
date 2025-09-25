package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ProviderSchema represents the schema structure returned by providers
type ProviderSchema struct {
	Name               string                   `json:"name"`
	Version            string                   `json:"version"`
	Protocol           string                   `json:"protocol"`
	Type               string                   `json:"type"`
	SupportedFunctions []string                 `json:"supported_functions"`
	ResourceTypes      []ResourceTypeDefinition `json:"resource_types"`
	ConfigSchema       json.RawMessage          `json:"config_schema"`
}

// ResourceTypeDefinition represents a resource type in the provider schema
type ResourceTypeDefinition struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Operations   []string        `json:"operations"`
	ConfigSchema json.RawMessage `json:"config_schema"`
	StateSchema  json.RawMessage `json:"state_schema"`
}

// ResourceStateSchema represents the detailed state schema structure
type ResourceStateSchema struct {
	Version    int                       `json:"version"`
	Required   []string                  `json:"required"`
	Properties map[string]PropertySchema `json:"properties"`
	Computed   []string                  `json:"computed"`
	Sensitive  []string                  `json:"sensitive"`
}

// PropertySchema represents a property in the resource state schema
type PropertySchema struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// SchemaProvider interface for schema testing - any provider that implements these methods
// can use the schema consistency testing framework
type SchemaProvider interface {
	Schema() (*ProviderSchema, error)
	Configure(ctx context.Context, config map[string]interface{}) error
	CallFunction(ctx context.Context, function string, input json.RawMessage) (json.RawMessage, error)
}

// StateAdapter interface for providers that have state adapters
type StateAdapter interface {
	GetResourceSchema(resourceType string) (*ResourceStateSchema, error)
}

// SchemaTestConfig defines the configuration for schema consistency testing
type SchemaTestConfig struct {
	// Provider instance to test
	Provider SchemaProvider

	// StateAdapter instance (optional, for more detailed schema validation)
	StateAdapter StateAdapter

	// Expected configuration fields that the Configure method accepts
	ExpectedConfigFields map[string]ConfigFieldExpectation

	// Expected resource types and their corresponding state adapter methods
	ExpectedResources map[string]ResourceExpectation

	// Expected functions that should be in SupportedFunctions
	ExpectedFunctions []string
}

// ConfigFieldExpectation defines expectations for a configuration field
type ConfigFieldExpectation struct {
	Required    bool   `json:"required"`
	FieldType   string `json:"type"`
	Description string `json:"description"`
}

// ResourceExpectation defines expectations for a resource type
type ResourceExpectation struct {
	StateAdapterMethod string   `json:"state_adapter_method"`
	Operations         []string `json:"operations"`
	Description        string   `json:"description"`
}

// RunSchemaConsistencyTests runs comprehensive schema consistency tests for a provider.
// This function should be called in provider test suites to ensure schema/implementation alignment.
//
// Usage in provider tests:
//
//	func TestProviderSchemaConsistency(t *testing.T) {
//		provider := NewMyProvider()
//		config := &testing.SchemaTestConfig{
//			Provider: provider,
//			ExpectedConfigFields: map[string]testing.ConfigFieldExpectation{
//				"endpoint": {Required: false, FieldType: "string", Description: "API endpoint"},
//			},
//			ExpectedResources: map[string]testing.ResourceExpectation{
//				"my_resource": {StateAdapterMethod: "getMyResourceSchema", Operations: []string{"create", "read", "update", "delete"}},
//			},
//			ExpectedFunctions: []string{"create_my_resource", "read_my_resource", "update_my_resource", "delete_my_resource"},
//		}
//		testing.RunSchemaConsistencyTests(t, config)
//	}
func RunSchemaConsistencyTests(t *testing.T, config *SchemaTestConfig) {
	require.NotNil(t, config, "SchemaTestConfig cannot be nil")
	require.NotNil(t, config.Provider, "Provider cannot be nil")

	// Get schema from provider
	schema, err := config.Provider.Schema()
	require.NoError(t, err, "Provider.Schema() must not return error")
	require.NotNil(t, schema, "Provider.Schema() must not return nil")

	t.Run("ConfigurationSchemaConsistency", func(t *testing.T) {
		validateConfigurationSchema(t, schema, config.ExpectedConfigFields)
	})

	t.Run("ResourceSchemaConsistency", func(t *testing.T) {
		validateResourceSchemas(t, schema, config.ExpectedResources, config.StateAdapter)
	})

	t.Run("FunctionCoverageConsistency", func(t *testing.T) {
		validateFunctionCoverage(t, schema, config.ExpectedFunctions)
	})

	t.Run("HandlerRoutingConsistency", func(t *testing.T) {
		validateHandlerRouting(t, config.Provider, schema)
	})
}

// validateConfigurationSchema checks that the provider's configuration schema
// matches the expected configuration fields
func validateConfigurationSchema(t *testing.T, schema *ProviderSchema, expectedFields map[string]ConfigFieldExpectation) {
	if len(expectedFields) == 0 {
		t.Skip("No expected configuration fields specified")
		return
	}

	// Parse the configuration schema
	var configSchema map[string]interface{}
	err := json.Unmarshal(schema.ConfigSchema, &configSchema)
	require.NoError(t, err, "ConfigSchema must be valid JSON")

	properties, ok := configSchema["properties"].(map[string]interface{})
	require.True(t, ok, "ConfigSchema must have properties field")

	// Check each expected field
	for fieldName, expected := range expectedFields {
		t.Run(fmt.Sprintf("ConfigField_%s", fieldName), func(t *testing.T) {
			field, exists := properties[fieldName]
			assert.True(t, exists, "Configuration field '%s' is expected but missing from schema", fieldName)

			if exists {
				fieldMap, ok := field.(map[string]interface{})
				require.True(t, ok, "Field '%s' must be an object", fieldName)

				// Verify field type
				if expected.FieldType != "" {
					actualType, hasType := fieldMap["type"].(string)
					assert.True(t, hasType, "Field '%s' must have type", fieldName)
					assert.Equal(t, expected.FieldType, actualType, "Field '%s' type mismatch", fieldName)
				}

				// Verify description if expected
				if expected.Description != "" {
					actualDesc, hasDesc := fieldMap["description"].(string)
					assert.True(t, hasDesc, "Field '%s' must have description", fieldName)
					if hasDesc {
						assert.Contains(t, actualDesc, expected.Description,
							"Field '%s' description should contain expected text", fieldName)
					}
				}
			}
		})
	}

	// Check for unexpected fields
	for fieldName := range properties {
		_, isExpected := expectedFields[fieldName]
		assert.True(t, isExpected, "Configuration field '%s' is in schema but not in expected fields", fieldName)
	}
}

// validateResourceSchemas checks that all expected resource types are present
// and have the correct structure
func validateResourceSchemas(t *testing.T, schema *ProviderSchema, expectedResources map[string]ResourceExpectation, stateAdapter StateAdapter) {
	if len(expectedResources) == 0 {
		t.Skip("No expected resources specified")
		return
	}

	// Check count
	assert.Len(t, schema.ResourceTypes, len(expectedResources),
		"Schema must include exactly %d resource types", len(expectedResources))

	// Check each expected resource
	for expectedName, expected := range expectedResources {
		t.Run(fmt.Sprintf("Resource_%s", expectedName), func(t *testing.T) {
			// Find the resource in schema
			var foundResource *ResourceTypeDefinition
			for i, resourceType := range schema.ResourceTypes {
				if resourceType.Name == expectedName {
					foundResource = &schema.ResourceTypes[i]
					break
				}
			}

			require.NotNil(t, foundResource, "Resource type '%s' must be in schema", expectedName)

			// Verify operations
			if len(expected.Operations) > 0 {
				assert.ElementsMatch(t, expected.Operations, foundResource.Operations,
					"Resource '%s' operations mismatch", expectedName)
			}

			// Verify description
			if expected.Description != "" {
				assert.Contains(t, foundResource.Description, expected.Description,
					"Resource '%s' description should contain expected text", expectedName)
			}

			// If state adapter is provided, validate schema consistency
			if stateAdapter != nil && expected.StateAdapterMethod != "" {
				validateResourceStateSchema(t, foundResource, stateAdapter, expected.StateAdapterMethod)
			}
		})
	}

	// Check for unexpected resources
	for _, resourceType := range schema.ResourceTypes {
		_, isExpected := expectedResources[resourceType.Name]
		assert.True(t, isExpected, "Resource type '%s' is in schema but not in expected resources", resourceType.Name)
	}
}

// validateResourceStateSchema validates that a resource's schema matches the state adapter schema
func validateResourceStateSchema(t *testing.T, resourceType *ResourceTypeDefinition, stateAdapter StateAdapter, methodName string) {
	// Use reflection to call the state adapter method
	stateAdapterValue := reflect.ValueOf(stateAdapter)
	methodValue := stateAdapterValue.MethodByName(methodName)

	if !methodValue.IsValid() {
		t.Logf("State adapter method '%s' not found, skipping detailed schema validation", methodName)
		return
	}

	results := methodValue.Call(nil)
	if len(results) != 1 {
		t.Logf("State adapter method '%s' returned %d values, expected 1", methodName, len(results))
		return
	}

	actualSchema := results[0].Interface()
	if actualSchema == nil {
		t.Logf("State adapter method '%s' returned nil", methodName)
		return
	}

	// Parse exposed state schema
	var exposedStateSchema map[string]interface{}
	err := json.Unmarshal(resourceType.StateSchema, &exposedStateSchema)
	require.NoError(t, err, "Resource state schema must be valid JSON")

	// Basic validation - check that the exposed schema has properties if the actual schema does
	exposedStateProps, hasStateProps := exposedStateSchema["properties"].(map[string]interface{})

	// Use reflection to check if actual schema has properties
	schemaValue := reflect.ValueOf(actualSchema)
	if schemaValue.Kind() == reflect.Ptr {
		schemaValue = schemaValue.Elem()
	}

	if schemaValue.Kind() == reflect.Struct {
		propertiesField := schemaValue.FieldByName("Properties")
		if propertiesField.IsValid() && propertiesField.Kind() == reflect.Map {
			actualProperties := propertiesField.Interface()

			if actualProperties != nil {
				// We have actual properties, so exposed schema should have them too
				assert.True(t, hasStateProps,
					"Resource '%s': Implementation has properties but exposed state schema has none",
					resourceType.Name)

				if hasStateProps {
					// Count check
					actualPropsMap := actualProperties.(map[string]interface{})
					assert.True(t, len(exposedStateProps) > 0,
						"Resource '%s': Implementation has %d properties but exposed schema has none",
						resourceType.Name, len(actualPropsMap))

					// Field presence check (basic)
					for actualFieldName := range actualPropsMap {
						_, exposedHasField := exposedStateProps[actualFieldName]
						assert.True(t, exposedHasField,
							"Resource '%s': Field '%s' is in state adapter but missing from exposed schema",
							resourceType.Name, actualFieldName)
					}
				}
			}
		}
	}
}

// validateFunctionCoverage checks that all expected functions are in SupportedFunctions
func validateFunctionCoverage(t *testing.T, schema *ProviderSchema, expectedFunctions []string) {
	if len(expectedFunctions) == 0 {
		t.Skip("No expected functions specified")
		return
	}

	// Check all expected functions are present
	for _, expectedFunc := range expectedFunctions {
		assert.Contains(t, schema.SupportedFunctions, expectedFunc,
			"Function '%s' is expected but missing from SupportedFunctions", expectedFunc)
	}

	// Check no extra functions
	for _, supportedFunc := range schema.SupportedFunctions {
		assert.Contains(t, expectedFunctions, supportedFunc,
			"Function '%s' is in SupportedFunctions but not in expected functions", supportedFunc)
	}

	// Check exact count
	assert.Len(t, schema.SupportedFunctions, len(expectedFunctions),
		"SupportedFunctions must contain exactly %d functions", len(expectedFunctions))
}

// validateHandlerRouting checks that CallFunction can route to all supported functions
func validateHandlerRouting(t *testing.T, provider SchemaProvider, schema *ProviderSchema) {
	ctx := context.Background()

	for _, functionName := range schema.SupportedFunctions {
		t.Run(fmt.Sprintf("Routing_%s", functionName), func(t *testing.T) {
			// Create minimal test input
			testInput := map[string]interface{}{
				"id":   "test-id-123",
				"name": "test-name",
			}
			inputJSON, err := json.Marshal(testInput)
			require.NoError(t, err)

			// Call the function - routing should work even if validation fails
			_, err = provider.CallFunction(ctx, functionName, inputJSON)

			// Check for routing errors (should not occur)
			if err != nil {
				errorMsg := err.Error()

				// These errors indicate routing problems
				assert.NotContains(t, errorMsg, "unsupported resource type",
					"Function '%s' failed with routing error: %v", functionName, err)
				assert.NotContains(t, errorMsg, "invalid function name format",
					"Function '%s' failed with format error: %v", functionName, err)

				// Other errors are acceptable (validation, business logic, etc.)
			}
		})
	}
}
