// Package rpc provides RPC interface definitions for the Kolumn Provider SDK
package rpc

import (
	"context"
	"encoding/json"

	"github.com/schemabounce/kolumn/sdk/types"
)

// UniversalProvider defines the interface that all providers must implement
// This combines both the simplified 4-method interface and Terraform-compatible methods
type UniversalProvider interface {
	// =====================================================
	// ORIGINAL SIMPLIFIED INTERFACE (4 methods)
	// =====================================================

	// Configure the provider with the given configuration
	Configure(ctx context.Context, config map[string]interface{}) error

	// GetSchema returns the provider's schema including supported functions and resource types
	GetSchema() (*types.ProviderSchema, error)

	// CallFunction dispatches function calls to the provider
	CallFunction(ctx context.Context, function string, input json.RawMessage) (json.RawMessage, error)

	// Close cleans up provider resources
	Close() error

	// =====================================================
	// TERRAFORM-COMPATIBLE METHODS (7 new methods)
	// =====================================================

	// ValidateProviderConfig validates provider configuration before Configure
	ValidateProviderConfig(ctx context.Context, req *ValidateProviderConfigRequest) (*ValidateProviderConfigResponse, error)

	// ValidateResourceConfig validates resource configuration
	ValidateResourceConfig(ctx context.Context, req *ValidateResourceConfigRequest) (*ValidateResourceConfigResponse, error)

	// PlanResourceChange compares desired vs current state and returns planned changes
	PlanResourceChange(ctx context.Context, req *PlanResourceChangeRequest) (*PlanResourceChangeResponse, error)

	// ApplyResourceChange executes planned changes and returns new state
	ApplyResourceChange(ctx context.Context, req *ApplyResourceChangeRequest) (*ApplyResourceChangeResponse, error)

	// ReadResource refreshes resource state from the actual system
	ReadResource(ctx context.Context, req *TerraformReadResourceRequest) (*TerraformReadResourceResponse, error)

	// ImportResourceState converts existing resources into Kolumn state
	ImportResourceState(ctx context.Context, req *ImportResourceStateRequest) (*ImportResourceStateResponse, error)

	// UpgradeResourceState handles schema version migrations
	UpgradeResourceState(ctx context.Context, req *UpgradeResourceStateRequest) (*UpgradeResourceStateResponse, error)
}

// ProviderInfo contains basic provider information
type ProviderInfo struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	SDKVersion string `json:"sdk_version"`
	APIVersion string `json:"api_version"`
	Source     string `json:"source,omitempty"`
}

// ServeConfig contains configuration for serving a provider plugin
type ServeConfig struct {
	Provider UniversalProvider
	Logger   interface{} // Compatible with hclog.Logger
	Debug    bool
}
