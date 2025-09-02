// Package pdk provides testing utilities for the Kolumn SDK
package pdk

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/schemabounce/kolumn/sdk/rpc"
	"github.com/schemabounce/kolumn/sdk/types"
)

// ProviderTestSuite provides a standard test suite for providers
type ProviderTestSuite struct {
	Provider    rpc.UniversalProvider
	Config      map[string]interface{}
	TestTimeout time.Duration
	t           *testing.T
}

// NewProviderTestSuite creates a new provider test suite
func NewProviderTestSuite(t *testing.T, provider rpc.UniversalProvider, config map[string]interface{}) *ProviderTestSuite {
	return &ProviderTestSuite{
		Provider:    provider,
		Config:      config,
		TestTimeout: 30 * time.Second,
		t:           t,
	}
}

// RunBasicTests runs basic provider tests
func (pts *ProviderTestSuite) RunBasicTests() {
	pts.t.Run("Configure", pts.TestConfigure)
	pts.t.Run("GetSchema", pts.TestGetSchema)
	pts.t.Run("UniversalFunctions", pts.TestUniversalFunctions)
	pts.t.Run("Close", pts.TestClose)
}

// TestConfigure tests provider configuration
func (pts *ProviderTestSuite) TestConfigure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), pts.TestTimeout)
	defer cancel()

	err := pts.Provider.Configure(ctx, pts.Config)
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}
}

// TestGetSchema tests schema retrieval
func (pts *ProviderTestSuite) TestGetSchema(t *testing.T) {
	schema, err := pts.Provider.GetSchema()
	if err != nil {
		t.Fatalf("GetSchema failed: %v", err)
	}

	if schema == nil {
		t.Fatal("GetSchema returned nil schema")
	}

	if schema.Provider.Name == "" {
		t.Error("Provider name is empty")
	}

	if schema.Provider.Version == "" {
		t.Error("Provider version is empty")
	}

	// Verify universal functions are present
	universalFunctions := []string{"ping", "get_version", "health_check", "get_metrics"}
	for _, funcName := range universalFunctions {
		if _, exists := schema.Functions[funcName]; !exists {
			t.Errorf("Universal function '%s' not found in schema", funcName)
		}
	}
}

// TestUniversalFunctions tests universal functions
func (pts *ProviderTestSuite) TestUniversalFunctions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), pts.TestTimeout)
	defer cancel()

	// Configure provider first
	err := pts.Provider.Configure(ctx, pts.Config)
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}

	t.Run("Ping", func(t *testing.T) {
		result, err := pts.Provider.CallFunction(ctx, "ping", nil)
		if err != nil {
			t.Fatalf("ping function failed: %v", err)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(result, &response); err != nil {
			t.Fatalf("failed to parse ping response: %v", err)
		}

		if status, ok := response["status"].(string); !ok || status == "" {
			t.Error("ping response missing or invalid status")
		}
	})

	t.Run("GetVersion", func(t *testing.T) {
		result, err := pts.Provider.CallFunction(ctx, "get_version", nil)
		if err != nil {
			t.Fatalf("get_version function failed: %v", err)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(result, &response); err != nil {
			t.Fatalf("failed to parse get_version response: %v", err)
		}

		if version, ok := response["version"].(string); !ok || version == "" {
			t.Error("get_version response missing or invalid version")
		}
	})

	t.Run("HealthCheck", func(t *testing.T) {
		result, err := pts.Provider.CallFunction(ctx, "health_check", nil)
		if err != nil {
			t.Fatalf("health_check function failed: %v", err)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(result, &response); err != nil {
			t.Fatalf("failed to parse health_check response: %v", err)
		}

		if status, ok := response["status"].(string); !ok || status == "" {
			t.Error("health_check response missing or invalid status")
		}
	})

	t.Run("GetMetrics", func(t *testing.T) {
		result, err := pts.Provider.CallFunction(ctx, "get_metrics", nil)
		if err != nil {
			t.Fatalf("get_metrics function failed: %v", err)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(result, &response); err != nil {
			t.Fatalf("failed to parse get_metrics response: %v", err)
		}

		// Metrics response should have some kind of metrics data
		if len(response) == 0 {
			t.Error("get_metrics response is empty")
		}
	})
}

// TestClose tests provider cleanup
func (pts *ProviderTestSuite) TestClose(t *testing.T) {
	err := pts.Provider.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

// TestFunctionCall tests a specific function call
func (pts *ProviderTestSuite) TestFunctionCall(functionName string, input interface{}, expectedOutput interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), pts.TestTimeout)
	defer cancel()

	// Configure provider
	err := pts.Provider.Configure(ctx, pts.Config)
	if err != nil {
		pts.t.Fatalf("Configure failed: %v", err)
	}

	var inputBytes json.RawMessage
	if input != nil {
		inputBytes, err = json.Marshal(input)
		if err != nil {
			pts.t.Fatalf("Failed to marshal input: %v", err)
		}
	}

	result, err := pts.Provider.CallFunction(ctx, functionName, inputBytes)
	if err != nil {
		pts.t.Fatalf("Function '%s' failed: %v", functionName, err)
	}

	if expectedOutput != nil {
		var actualOutput interface{}
		if err := json.Unmarshal(result, &actualOutput); err != nil {
			pts.t.Fatalf("Failed to unmarshal output: %v", err)
		}

		if !reflect.DeepEqual(actualOutput, expectedOutput) {
			pts.t.Errorf("Function '%s' output mismatch:\nExpected: %+v\nActual: %+v",
				functionName, expectedOutput, actualOutput)
		}
	}
}

// MockProvider creates a mock provider for testing
type MockProvider struct {
	ConfigureFn    func(ctx context.Context, config map[string]interface{}) error
	GetSchemaFn    func() (*types.ProviderSchema, error)
	CallFunctionFn func(ctx context.Context, function string, input json.RawMessage) (json.RawMessage, error)
	CloseFn        func() error

	// Terraform-compatible methods
	ValidateProviderConfigFn func(ctx context.Context, req *rpc.ValidateProviderConfigRequest) (*rpc.ValidateProviderConfigResponse, error)
	ValidateResourceConfigFn func(ctx context.Context, req *rpc.ValidateResourceConfigRequest) (*rpc.ValidateResourceConfigResponse, error)
	PlanResourceChangeFn     func(ctx context.Context, req *rpc.PlanResourceChangeRequest) (*rpc.PlanResourceChangeResponse, error)
	ApplyResourceChangeFn    func(ctx context.Context, req *rpc.ApplyResourceChangeRequest) (*rpc.ApplyResourceChangeResponse, error)
	ReadResourceFn           func(ctx context.Context, req *rpc.TerraformReadResourceRequest) (*rpc.TerraformReadResourceResponse, error)
	ImportResourceStateFn    func(ctx context.Context, req *rpc.ImportResourceStateRequest) (*rpc.ImportResourceStateResponse, error)
	UpgradeResourceStateFn   func(ctx context.Context, req *rpc.UpgradeResourceStateRequest) (*rpc.UpgradeResourceStateResponse, error)
}

// Configure implements UniversalProvider
func (m *MockProvider) Configure(ctx context.Context, config map[string]interface{}) error {
	if m.ConfigureFn != nil {
		return m.ConfigureFn(ctx, config)
	}
	return nil
}

// GetSchema implements UniversalProvider
func (m *MockProvider) GetSchema() (*types.ProviderSchema, error) {
	if m.GetSchemaFn != nil {
		return m.GetSchemaFn()
	}

	schema := CreateProviderSchema("mock", "1.0.0")
	AddUniversalFunctions(schema)
	return schema, nil
}

// CallFunction implements UniversalProvider
func (m *MockProvider) CallFunction(ctx context.Context, function string, input json.RawMessage) (json.RawMessage, error) {
	if m.CallFunctionFn != nil {
		return m.CallFunctionFn(ctx, function, input)
	}

	// Default implementations for universal functions
	switch function {
	case "ping":
		return CreatePingResponse("ok", 0, nil)
	case "get_version":
		return CreateSuccessResponse(map[string]string{"version": "1.0.0"})
	case "health_check":
		return CreateSuccessResponse(map[string]interface{}{"status": "healthy"})
	case "get_metrics":
		return CreateSuccessResponse(map[string]interface{}{"requests": 0})
	default:
		return CreateErrorResponse(fmt.Sprintf("function '%s' not implemented", function))
	}
}

// Close implements UniversalProvider
func (m *MockProvider) Close() error {
	if m.CloseFn != nil {
		return m.CloseFn()
	}
	return nil
}

// ValidateProviderConfig implements UniversalProvider
func (m *MockProvider) ValidateProviderConfig(ctx context.Context, req *rpc.ValidateProviderConfigRequest) (*rpc.ValidateProviderConfigResponse, error) {
	if m.ValidateProviderConfigFn != nil {
		return m.ValidateProviderConfigFn(ctx, req)
	}
	return &rpc.ValidateProviderConfigResponse{Success: true}, nil
}

// ValidateResourceConfig implements UniversalProvider
func (m *MockProvider) ValidateResourceConfig(ctx context.Context, req *rpc.ValidateResourceConfigRequest) (*rpc.ValidateResourceConfigResponse, error) {
	if m.ValidateResourceConfigFn != nil {
		return m.ValidateResourceConfigFn(ctx, req)
	}
	return &rpc.ValidateResourceConfigResponse{Success: true}, nil
}

// PlanResourceChange implements UniversalProvider
func (m *MockProvider) PlanResourceChange(ctx context.Context, req *rpc.PlanResourceChangeRequest) (*rpc.PlanResourceChangeResponse, error) {
	if m.PlanResourceChangeFn != nil {
		return m.PlanResourceChangeFn(ctx, req)
	}
	return &rpc.PlanResourceChangeResponse{
		Success:      true,
		PlannedState: req.ProposedNewState,
	}, nil
}

// ApplyResourceChange implements UniversalProvider
func (m *MockProvider) ApplyResourceChange(ctx context.Context, req *rpc.ApplyResourceChangeRequest) (*rpc.ApplyResourceChangeResponse, error) {
	if m.ApplyResourceChangeFn != nil {
		return m.ApplyResourceChangeFn(ctx, req)
	}
	return &rpc.ApplyResourceChangeResponse{
		Success:  true,
		NewState: req.PlannedState,
	}, nil
}

// ReadResource implements UniversalProvider
func (m *MockProvider) ReadResource(ctx context.Context, req *rpc.TerraformReadResourceRequest) (*rpc.TerraformReadResourceResponse, error) {
	if m.ReadResourceFn != nil {
		return m.ReadResourceFn(ctx, req)
	}
	return &rpc.TerraformReadResourceResponse{
		Success:  true,
		NewState: req.CurrentState,
	}, nil
}

// ImportResourceState implements UniversalProvider
func (m *MockProvider) ImportResourceState(ctx context.Context, req *rpc.ImportResourceStateRequest) (*rpc.ImportResourceStateResponse, error) {
	if m.ImportResourceStateFn != nil {
		return m.ImportResourceStateFn(ctx, req)
	}
	return &rpc.ImportResourceStateResponse{
		Success: true,
		ImportedResources: []rpc.ImportedResource{
			{
				ResourceType: req.ResourceType,
				State:        map[string]interface{}{"id": req.ID},
			},
		},
	}, nil
}

// UpgradeResourceState implements UniversalProvider
func (m *MockProvider) UpgradeResourceState(ctx context.Context, req *rpc.UpgradeResourceStateRequest) (*rpc.UpgradeResourceStateResponse, error) {
	if m.UpgradeResourceStateFn != nil {
		return m.UpgradeResourceStateFn(ctx, req)
	}
	return &rpc.UpgradeResourceStateResponse{
		Success:       true,
		UpgradedState: req.RawState,
	}, nil
}

// Ensure MockProvider implements UniversalProvider
var _ rpc.UniversalProvider = (*MockProvider)(nil)

// AssertNoError asserts that an error is nil
func AssertNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	if err != nil {
		if len(msgAndArgs) > 0 {
			msg := fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
			t.Fatalf("%s: %v", msg, err)
		} else {
			t.Fatalf("Unexpected error: %v", err)
		}
	}
}

// AssertError asserts that an error is not nil
func AssertError(t *testing.T, err error, msgAndArgs ...interface{}) {
	if err == nil {
		if len(msgAndArgs) > 0 {
			msg := fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
			t.Fatalf("%s: expected error but got none", msg)
		} else {
			t.Fatal("Expected error but got none")
		}
	}
}

// AssertEqual asserts that two values are equal
func AssertEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		if len(msgAndArgs) > 0 {
			msg := fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
			t.Fatalf("%s:\nExpected: %+v\nActual: %+v", msg, expected, actual)
		} else {
			t.Fatalf("Values not equal:\nExpected: %+v\nActual: %+v", expected, actual)
		}
	}
}

// AssertNotEqual asserts that two values are not equal
func AssertNotEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	if reflect.DeepEqual(expected, actual) {
		if len(msgAndArgs) > 0 {
			msg := fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
			t.Fatalf("%s: values should not be equal: %+v", msg, actual)
		} else {
			t.Fatalf("Values should not be equal: %+v", actual)
		}
	}
}

// AssertContains asserts that a slice contains a value
func AssertContains(t *testing.T, slice interface{}, item interface{}, msgAndArgs ...interface{}) {
	sliceValue := reflect.ValueOf(slice)
	if sliceValue.Kind() != reflect.Slice {
		t.Fatalf("AssertContains expects a slice, got %T", slice)
	}

	for i := 0; i < sliceValue.Len(); i++ {
		if reflect.DeepEqual(sliceValue.Index(i).Interface(), item) {
			return // Found
		}
	}

	if len(msgAndArgs) > 0 {
		msg := fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
		t.Fatalf("%s: slice %+v does not contain %+v", msg, slice, item)
	} else {
		t.Fatalf("Slice %+v does not contain %+v", slice, item)
	}
}
