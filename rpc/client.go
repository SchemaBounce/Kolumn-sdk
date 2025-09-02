// Package rpc provides RPC client functionality for the Kolumn Provider SDK
package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/rpc"

	"github.com/hashicorp/go-hclog"
	"github.com/schemabounce/kolumn/sdk/types"
)

// ProviderClient implements the UniversalProvider interface as an RPC client
type ProviderClient struct {
	Client *rpc.Client
	Logger hclog.Logger
}

// ConfigureRequest represents the request for Configure RPC call
type ConfigureRequest struct {
	Config map[string]interface{} `json:"config"`
}

// ConfigureResponse represents the response for Configure RPC call
type ConfigureResponse struct {
	Error *RPCError `json:"error,omitempty"`
}

// GetSchemaRequest represents the request for GetSchema RPC call
type GetSchemaRequest struct{}

// GetSchemaResponse represents the response for GetSchema RPC call
type GetSchemaResponse struct {
	Schema *types.ProviderSchema `json:"schema,omitempty"`
	Error  *RPCError             `json:"error,omitempty"`
}

// CallFunctionRequest represents the request for CallFunction RPC call
type CallFunctionRequest struct {
	Function string          `json:"function"`
	Input    json.RawMessage `json:"input,omitempty"`
}

// CallFunctionResponse represents the response for CallFunction RPC call
type CallFunctionResponse struct {
	Output json.RawMessage `json:"output,omitempty"`
	Error  *RPCError       `json:"error,omitempty"`
}

// CloseRequest represents the request for Close RPC call
type CloseRequest struct{}

// CloseResponse represents the response for Close RPC call
type CloseResponse struct {
	Error *RPCError `json:"error,omitempty"`
}

// RPCError is now defined in types.go

// Configure calls the provider's Configure method via RPC
func (c *ProviderClient) Configure(ctx context.Context, config map[string]interface{}) error {
	req := &ConfigureRequest{Config: config}
	var resp ConfigureResponse

	err := c.Client.Call("Provider.Configure", req, &resp)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

// GetSchema calls the provider's GetSchema method via RPC
func (c *ProviderClient) GetSchema() (*types.ProviderSchema, error) {
	req := &GetSchemaRequest{}
	var resp GetSchemaResponse

	err := c.Client.Call("Provider.GetSchema", req, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	return resp.Schema, nil
}

// CallFunction calls the provider's CallFunction method via RPC
func (c *ProviderClient) CallFunction(ctx context.Context, function string, input json.RawMessage) (json.RawMessage, error) {
	req := &CallFunctionRequest{
		Function: function,
		Input:    input,
	}
	var resp CallFunctionResponse

	err := c.Client.Call("Provider.CallFunction", req, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	return resp.Output, nil
}

// Close calls the provider's Close method via RPC
func (c *ProviderClient) Close() error {
	req := &CloseRequest{}
	var resp CloseResponse

	err := c.Client.Call("Provider.Close", req, &resp)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

// =============================================================================
// TERRAFORM-COMPATIBLE METHOD IMPLEMENTATIONS
// =============================================================================

// ValidateProviderConfig implements UniversalProvider
func (c *ProviderClient) ValidateProviderConfig(ctx context.Context, req *ValidateProviderConfigRequest) (*ValidateProviderConfigResponse, error) {
	// Marshal request and call through CallFunction
	input, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	output, err := c.CallFunction(ctx, "ValidateProviderConfig", json.RawMessage(input))
	if err != nil {
		return nil, err
	}

	var resp ValidateProviderConfigResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// ValidateResourceConfig implements UniversalProvider
func (c *ProviderClient) ValidateResourceConfig(ctx context.Context, req *ValidateResourceConfigRequest) (*ValidateResourceConfigResponse, error) {
	input, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	output, err := c.CallFunction(ctx, "ValidateResourceConfig", json.RawMessage(input))
	if err != nil {
		return nil, err
	}

	var resp ValidateResourceConfigResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// PlanResourceChange implements UniversalProvider
func (c *ProviderClient) PlanResourceChange(ctx context.Context, req *PlanResourceChangeRequest) (*PlanResourceChangeResponse, error) {
	input, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	output, err := c.CallFunction(ctx, "PlanResourceChange", json.RawMessage(input))
	if err != nil {
		return nil, err
	}

	var resp PlanResourceChangeResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// ApplyResourceChange implements UniversalProvider
func (c *ProviderClient) ApplyResourceChange(ctx context.Context, req *ApplyResourceChangeRequest) (*ApplyResourceChangeResponse, error) {
	input, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	output, err := c.CallFunction(ctx, "ApplyResourceChange", json.RawMessage(input))
	if err != nil {
		return nil, err
	}

	var resp ApplyResourceChangeResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// ReadResource implements UniversalProvider
func (c *ProviderClient) ReadResource(ctx context.Context, req *TerraformReadResourceRequest) (*TerraformReadResourceResponse, error) {
	input, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	output, err := c.CallFunction(ctx, "ReadResource", json.RawMessage(input))
	if err != nil {
		return nil, err
	}

	var resp TerraformReadResourceResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// ImportResourceState implements UniversalProvider
func (c *ProviderClient) ImportResourceState(ctx context.Context, req *ImportResourceStateRequest) (*ImportResourceStateResponse, error) {
	input, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	output, err := c.CallFunction(ctx, "ImportResourceState", json.RawMessage(input))
	if err != nil {
		return nil, err
	}

	var resp ImportResourceStateResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// UpgradeResourceState implements UniversalProvider
func (c *ProviderClient) UpgradeResourceState(ctx context.Context, req *UpgradeResourceStateRequest) (*UpgradeResourceStateResponse, error) {
	input, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	output, err := c.CallFunction(ctx, "UpgradeResourceState", json.RawMessage(input))
	if err != nil {
		return nil, err
	}

	var resp UpgradeResourceStateResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// Verify that ProviderClient implements UniversalProvider
var _ UniversalProvider = (*ProviderClient)(nil)
