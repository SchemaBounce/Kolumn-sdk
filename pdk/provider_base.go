// Package pdk provides a base provider implementation that handles RPC plumbing
// Provider developers extend BaseProvider and register resource handlers
package pdk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/schemabounce/kolumn/sdk/rpc"
	"github.com/schemabounce/kolumn/sdk/types"
)

// =============================================================================
// BASE PROVIDER IMPLEMENTATION
// =============================================================================

// BaseProvider provides automatic RPC handling for standard CRUD operations
// Provider developers extend this and register ResourceHandlers for each resource type
type BaseProvider struct {
	name             string
	version          string
	resourceHandlers map[string]ResourceHandler
	optionalHandlers map[string]OptionalResourceHandler
	configured       bool
	config           map[string]interface{}
	capabilities     []string
}

// NewBaseProvider creates a new base provider
func NewBaseProvider(name, version string) *BaseProvider {
	return &BaseProvider{
		name:             name,
		version:          version,
		resourceHandlers: make(map[string]ResourceHandler),
		optionalHandlers: make(map[string]OptionalResourceHandler),
		capabilities:     []string{},
	}
}

// RegisterResourceHandler registers a handler for a specific resource type
func (p *BaseProvider) RegisterResourceHandler(resourceType string, handler ResourceHandler) {
	p.resourceHandlers[resourceType] = handler
}

// RegisterOptionalHandler registers an optional handler for a resource type
func (p *BaseProvider) RegisterOptionalHandler(resourceType string, handler OptionalResourceHandler) {
	p.optionalHandlers[resourceType] = handler
}

// AddCapability adds a capability to the provider
func (p *BaseProvider) AddCapability(capability string) {
	p.capabilities = append(p.capabilities, capability)
}

// =============================================================================
// UNIVERSAL PROVIDER INTERFACE IMPLEMENTATION
// =============================================================================

// Configure implements rpc.UniversalProvider
func (p *BaseProvider) Configure(ctx context.Context, config map[string]interface{}) error {
	p.config = config
	p.configured = true
	return nil
}

// GetSchema implements rpc.UniversalProvider
func (p *BaseProvider) GetSchema() (*types.ProviderSchema, error) {
	if !p.configured {
		return nil, fmt.Errorf("provider not configured")
	}

	// Build resource types from registered handlers
	var resourceTypes []string
	for resourceType := range p.resourceHandlers {
		resourceTypes = append(resourceTypes, resourceType)
	}

	// Build supported functions based on registered handlers and capabilities
	functions := p.getSupportedFunctions()

	return &types.ProviderSchema{
		Provider: types.ProviderSpec{
			Name:        p.name,
			Version:     p.version,
			Description: fmt.Sprintf("%s provider with SDK CRUD helpers", p.name),
		},
		Functions:     functions,
		ResourceTypes: resourceTypes,
		Capabilities:  p.capabilities,
	}, nil
}

// CallFunction implements rpc.UniversalProvider with automatic CRUD dispatch
func (p *BaseProvider) CallFunction(ctx context.Context, function string, input json.RawMessage) (json.RawMessage, error) {
	if !p.configured {
		return nil, fmt.Errorf("provider not configured")
	}

	// Parse the function and resource type
	// Format: "CreateResource", "ReadResource_tablename", etc.
	parts := strings.Split(function, "_")
	operation := parts[0]

	// Determine resource type from request or function name
	resourceType := ""
	if len(parts) > 1 {
		resourceType = parts[1]
	} else {
		// Try to extract resource type from request
		var baseReq struct {
			ResourceType string `json:"resource_type"`
		}
		if err := json.Unmarshal(input, &baseReq); err == nil {
			resourceType = baseReq.ResourceType
		}
	}

	// Dispatch to appropriate handler based on operation
	switch operation {
	case "CreateResource":
		return p.handleCreate(ctx, resourceType, input)
	case "ReadResource":
		return p.handleRead(ctx, resourceType, input)
	case "UpdateResource":
		return p.handleUpdate(ctx, resourceType, input)
	case "DeleteResource":
		return p.handleDelete(ctx, resourceType, input)
	case "DestroyResource":
		return p.handleDestroy(ctx, resourceType, input)
	case "ImportResource":
		return p.handleImport(ctx, resourceType, input)
	case "PlanResource":
		return p.handlePlan(ctx, resourceType, input)
	case "ValidateResource":
		return p.handleValidate(ctx, resourceType, input)
	case "GetState":
		return p.handleGetState(ctx, resourceType, input)
	case "SetState":
		return p.handleSetState(ctx, resourceType, input)
	case "DetectDrift":
		return p.handleDetectDrift(ctx, resourceType, input)
	case "DiscoverResources":
		return p.handleDiscover(ctx, resourceType, input)
	case "IntrospectResource":
		return p.handleIntrospect(ctx, resourceType, input)
	case "GetMetrics":
		return p.handleGetMetrics(ctx, resourceType, input)
	case "Backup":
		return p.handleBackup(ctx, resourceType, input)
	case "Restore":
		return p.handleRestore(ctx, resourceType, input)
	case "Ping":
		return p.handlePing(ctx, input)
	default:
		return nil, fmt.Errorf("unsupported function: %s", function)
	}
}

// Close implements rpc.UniversalProvider
func (p *BaseProvider) Close() error {
	p.configured = false
	return nil
}

// =============================================================================
// TERRAFORM-COMPATIBLE METHOD IMPLEMENTATIONS
// =============================================================================

// ValidateProviderConfig implements rpc.UniversalProvider
func (p *BaseProvider) ValidateProviderConfig(ctx context.Context, req *rpc.ValidateProviderConfigRequest) (*rpc.ValidateProviderConfigResponse, error) {
	// Basic validation - check for required fields
	var diagnostics []rpc.Diagnostic

	// Provider-specific validation can be added by overriding this method
	// For now, just accept any configuration

	return &rpc.ValidateProviderConfigResponse{
		Success:     true,
		Diagnostics: diagnostics,
	}, nil
}

// ValidateResourceConfig implements rpc.UniversalProvider
func (p *BaseProvider) ValidateResourceConfig(ctx context.Context, req *rpc.ValidateResourceConfigRequest) (*rpc.ValidateResourceConfigResponse, error) {
	handler, exists := p.resourceHandlers[req.ResourceType]
	if !exists {
		return &rpc.ValidateResourceConfigResponse{
			Success: false,
			Diagnostics: []rpc.Diagnostic{
				{
					Severity: "error",
					Summary:  "Unknown resource type",
					Detail:   fmt.Sprintf("Resource type %s is not supported by this provider", req.ResourceType),
				},
			},
		}, nil
	}

	// Check if handler supports Terraform-compatible validation
	if terraformHandler, ok := handler.(TerraformCompatibleResourceHandler); ok {
		// Use Terraform-compatible validation
		tfReq := &TerraformValidateConfigRequest{
			ResourceType: req.ResourceType,
			Config:       req.Config,
		}

		tfResp, err := terraformHandler.ValidateConfig(ctx, tfReq)
		if err != nil {
			return &rpc.ValidateResourceConfigResponse{
				Success: false,
				Error: &rpc.RPCError{
					Code:    "VALIDATION_ERROR",
					Message: err.Error(),
				},
			}, nil
		}

		// Convert Terraform diagnostics to RPC format
		var diagnostics []rpc.Diagnostic
		for _, d := range tfResp.Diagnostics {
			diagnostics = append(diagnostics, rpc.Diagnostic{
				Severity:  d.Severity,
				Summary:   d.Summary,
				Detail:    d.Detail,
				Attribute: d.Attribute,
			})
		}

		return &rpc.ValidateResourceConfigResponse{
			Success:     tfResp.Valid,
			Diagnostics: diagnostics,
		}, nil
	}

	// Fall back to existing validation handler
	sdkReq := &ValidateRequest{
		ResourceType: req.ResourceType,
		Config:       req.Config,
	}

	sdkResp, err := handler.Validate(ctx, sdkReq)
	if err != nil {
		return &rpc.ValidateResourceConfigResponse{
			Success: false,
			Error: &rpc.RPCError{
				Code:    "VALIDATION_ERROR",
				Message: err.Error(),
			},
		}, nil
	}

	// Convert SDK validation result to RPC format
	var diagnostics []rpc.Diagnostic
	for _, e := range sdkResp.Errors {
		diagnostics = append(diagnostics, rpc.Diagnostic{
			Severity: "error",
			Summary:  e.Message,
			Detail:   e.Suggestion,
		})
	}
	for _, w := range sdkResp.Warnings {
		diagnostics = append(diagnostics, rpc.Diagnostic{
			Severity: "warning",
			Summary:  w.Message,
			Detail:   w.Suggestion,
		})
	}

	return &rpc.ValidateResourceConfigResponse{
		Success:     sdkResp.Valid,
		Diagnostics: diagnostics,
	}, nil
}

// PlanResourceChange implements rpc.UniversalProvider
func (p *BaseProvider) PlanResourceChange(ctx context.Context, req *rpc.PlanResourceChangeRequest) (*rpc.PlanResourceChangeResponse, error) {
	handler, exists := p.resourceHandlers[req.ResourceType]
	if !exists {
		return &rpc.PlanResourceChangeResponse{
			Success: false,
			Error: &rpc.RPCError{
				Code:    "UNKNOWN_RESOURCE_TYPE",
				Message: fmt.Sprintf("Resource type %s is not supported", req.ResourceType),
			},
		}, nil
	}

	// Check if handler supports Terraform-compatible planning
	if terraformHandler, ok := handler.(TerraformCompatibleResourceHandler); ok {
		// Use Terraform-compatible planning
		tfReq := &TerraformPlanChangeRequest{
			ResourceType:  req.ResourceType,
			PriorState:    req.PriorState,
			ProposedState: req.ProposedNewState,
			Config:        req.Config,
			PriorPrivate:  req.PriorPrivate,
		}

		tfResp, err := terraformHandler.PlanChange(ctx, tfReq)
		if err != nil {
			return &rpc.PlanResourceChangeResponse{
				Success: false,
				Error: &rpc.RPCError{
					Code:    "PLAN_ERROR",
					Message: err.Error(),
				},
			}, nil
		}

		// Convert Terraform diagnostics to RPC format
		var diagnostics []rpc.Diagnostic
		for _, d := range tfResp.Diagnostics {
			diagnostics = append(diagnostics, rpc.Diagnostic{
				Severity:  d.Severity,
				Summary:   d.Summary,
				Detail:    d.Detail,
				Attribute: d.Attribute,
			})
		}

		return &rpc.PlanResourceChangeResponse{
			Success:          true,
			PlannedState:     tfResp.PlannedState,
			RequiresReplace:  tfResp.RequiresReplace,
			PlannedPrivate:   tfResp.PlannedPrivate,
			Diagnostics:      diagnostics,
			LegacyTypeSystem: false,
		}, nil
	}

	// Fall back to existing plan handler
	sdkReq := &PlanRequest{
		ResourceType:  req.ResourceType,
		DesiredConfig: req.ProposedNewState,
		CurrentState:  req.PriorState,
	}

	sdkResp, err := handler.Plan(ctx, sdkReq)
	if err != nil {
		return &rpc.PlanResourceChangeResponse{
			Success: false,
			Error: &rpc.RPCError{
				Code:    "PLAN_ERROR",
				Message: err.Error(),
			},
		}, nil
	}

	// Determine requires replace based on planned changes
	var requiresReplace []string
	for _, change := range sdkResp.Changes {
		if change.RequiresDestroy {
			requiresReplace = append(requiresReplace, change.Field)
		}
	}

	// Build planned state from desired config
	plannedState := req.ProposedNewState
	if plannedState == nil {
		plannedState = make(map[string]interface{})
	}

	return &rpc.PlanResourceChangeResponse{
		Success:          true,
		PlannedState:     plannedState,
		RequiresReplace:  requiresReplace,
		LegacyTypeSystem: false,
	}, nil
}

// ApplyResourceChange implements rpc.UniversalProvider
func (p *BaseProvider) ApplyResourceChange(ctx context.Context, req *rpc.ApplyResourceChangeRequest) (*rpc.ApplyResourceChangeResponse, error) {
	handler, exists := p.resourceHandlers[req.ResourceType]
	if !exists {
		return &rpc.ApplyResourceChangeResponse{
			Success: false,
			Error: &rpc.RPCError{
				Code:    "UNKNOWN_RESOURCE_TYPE",
				Message: fmt.Sprintf("Resource type %s is not supported", req.ResourceType),
			},
		}, nil
	}

	// Check if handler supports Terraform-compatible apply
	if terraformHandler, ok := handler.(TerraformCompatibleResourceHandler); ok {
		// Use Terraform-compatible apply
		tfReq := &TerraformApplyChangeRequest{
			ResourceType:   req.ResourceType,
			PriorState:     req.PriorState,
			PlannedState:   req.PlannedState,
			Config:         req.Config,
			PlannedPrivate: req.PlannedPrivate,
		}

		tfResp, err := terraformHandler.ApplyChange(ctx, tfReq)
		if err != nil {
			return &rpc.ApplyResourceChangeResponse{
				Success: false,
				Error: &rpc.RPCError{
					Code:    "APPLY_ERROR",
					Message: err.Error(),
				},
			}, nil
		}

		// Convert Terraform diagnostics to RPC format
		var diagnostics []rpc.Diagnostic
		for _, d := range tfResp.Diagnostics {
			diagnostics = append(diagnostics, rpc.Diagnostic{
				Severity:  d.Severity,
				Summary:   d.Summary,
				Detail:    d.Detail,
				Attribute: d.Attribute,
			})
		}

		return &rpc.ApplyResourceChangeResponse{
			Success:          true,
			NewState:         tfResp.NewState,
			Private:          tfResp.Private,
			Diagnostics:      diagnostics,
			LegacyTypeSystem: false,
		}, nil
	}

	// Fall back to existing handlers - determine operation type based on state
	var operation string
	if req.PriorState == nil || len(req.PriorState) == 0 {
		operation = "create"
	} else if req.PlannedState == nil || len(req.PlannedState) == 0 {
		operation = "delete"
	} else {
		operation = "update"
	}

	var newState map[string]interface{}

	switch operation {
	case "create":
		// Extract name and resource ID from planned state
		name, _ := req.PlannedState["name"].(string)
		if name == "" {
			name = "unnamed-resource"
		}

		sdkReq := &CreateRequest{
			ResourceType: req.ResourceType,
			Name:         name,
			Config:       req.PlannedState,
		}

		sdkResp, err := handler.Create(ctx, sdkReq)
		if err != nil {
			return &rpc.ApplyResourceChangeResponse{
				Success: false,
				Error: &rpc.RPCError{
					Code:    "CREATE_ERROR",
					Message: err.Error(),
				},
			}, nil
		}

		newState = sdkResp.State

	case "update":
		// Extract name and resource ID from prior state
		name, _ := req.PriorState["name"].(string)
		resourceID, _ := req.PriorState["id"].(string)

		sdkReq := &UpdateRequest{
			ResourceType: req.ResourceType,
			ResourceID:   resourceID,
			Name:         name,
			Config:       req.PlannedState,
			CurrentState: req.PriorState,
		}

		sdkResp, err := handler.Update(ctx, sdkReq)
		if err != nil {
			return &rpc.ApplyResourceChangeResponse{
				Success: false,
				Error: &rpc.RPCError{
					Code:    "UPDATE_ERROR",
					Message: err.Error(),
				},
			}, nil
		}

		newState = sdkResp.NewState

	case "delete":
		// Extract name and resource ID from prior state
		name, _ := req.PriorState["name"].(string)
		resourceID, _ := req.PriorState["id"].(string)

		sdkReq := &DeleteRequest{
			ResourceType: req.ResourceType,
			ResourceID:   resourceID,
			Name:         name,
			State:        req.PriorState,
		}

		_, err := handler.Delete(ctx, sdkReq)
		if err != nil {
			return &rpc.ApplyResourceChangeResponse{
				Success: false,
				Error: &rpc.RPCError{
					Code:    "DELETE_ERROR",
					Message: err.Error(),
				},
			}, nil
		}

		// For delete, new state should be empty
		newState = make(map[string]interface{})
	}

	return &rpc.ApplyResourceChangeResponse{
		Success:          true,
		NewState:         newState,
		LegacyTypeSystem: false,
	}, nil
}

// ReadResource implements rpc.UniversalProvider
func (p *BaseProvider) ReadResource(ctx context.Context, req *rpc.TerraformReadResourceRequest) (*rpc.TerraformReadResourceResponse, error) {
	handler, exists := p.resourceHandlers[req.ResourceType]
	if !exists {
		return &rpc.TerraformReadResourceResponse{
			Success: false,
			Error: &rpc.RPCError{
				Code:    "UNKNOWN_RESOURCE_TYPE",
				Message: fmt.Sprintf("Resource type %s is not supported", req.ResourceType),
			},
		}, nil
	}

	// Check if handler supports Terraform-compatible refresh
	if terraformHandler, ok := handler.(TerraformCompatibleResourceHandler); ok {
		// Use Terraform-compatible refresh
		tfReq := &TerraformRefreshStateRequest{
			ResourceType: req.ResourceType,
			CurrentState: req.CurrentState,
			Private:      req.Private,
		}

		tfResp, err := terraformHandler.RefreshState(ctx, tfReq)
		if err != nil {
			return &rpc.TerraformReadResourceResponse{
				Success: false,
				Error: &rpc.RPCError{
					Code:    "READ_ERROR",
					Message: err.Error(),
				},
			}, nil
		}

		// Convert Terraform diagnostics to RPC format
		var diagnostics []rpc.Diagnostic
		for _, d := range tfResp.Diagnostics {
			diagnostics = append(diagnostics, rpc.Diagnostic{
				Severity:  d.Severity,
				Summary:   d.Summary,
				Detail:    d.Detail,
				Attribute: d.Attribute,
			})
		}

		return &rpc.TerraformReadResourceResponse{
			Success:     true,
			NewState:    tfResp.NewState,
			Private:     tfResp.Private,
			Diagnostics: diagnostics,
		}, nil
	}

	// Fall back to existing read handler
	// Extract name and resource ID from current state
	name, _ := req.CurrentState["name"].(string)
	resourceID, _ := req.CurrentState["id"].(string)

	sdkReq := &ReadRequest{
		ResourceType: req.ResourceType,
		ResourceID:   resourceID,
		Name:         name,
	}

	sdkResp, err := handler.Read(ctx, sdkReq)
	if err != nil {
		return &rpc.TerraformReadResourceResponse{
			Success: false,
			Error: &rpc.RPCError{
				Code:    "READ_ERROR",
				Message: err.Error(),
			},
		}, nil
	}

	if sdkResp.NotFound {
		// Resource no longer exists
		return &rpc.TerraformReadResourceResponse{
			Success:  true,
			NewState: make(map[string]interface{}), // Empty state indicates resource is gone
		}, nil
	}

	return &rpc.TerraformReadResourceResponse{
		Success:  true,
		NewState: sdkResp.State,
	}, nil
}

// ImportResourceState implements rpc.UniversalProvider
func (p *BaseProvider) ImportResourceState(ctx context.Context, req *rpc.ImportResourceStateRequest) (*rpc.ImportResourceStateResponse, error) {
	handler, exists := p.resourceHandlers[req.ResourceType]
	if !exists {
		return &rpc.ImportResourceStateResponse{
			Success: false,
			Error: &rpc.RPCError{
				Code:    "UNKNOWN_RESOURCE_TYPE",
				Message: fmt.Sprintf("Resource type %s is not supported", req.ResourceType),
			},
		}, nil
	}

	// Check if handler supports Terraform-compatible import
	if terraformHandler, ok := handler.(TerraformCompatibleResourceHandler); ok {
		// Use Terraform-compatible import
		tfReq := &TerraformImportStateRequest{
			ResourceType: req.ResourceType,
			ID:           req.ID,
		}

		tfResp, err := terraformHandler.ImportState(ctx, tfReq)
		if err != nil {
			return &rpc.ImportResourceStateResponse{
				Success: false,
				Error: &rpc.RPCError{
					Code:    "IMPORT_ERROR",
					Message: err.Error(),
				},
			}, nil
		}

		// Convert Terraform imported resources to RPC format
		var importedResources []rpc.ImportedResource
		for _, res := range tfResp.ImportedResources {
			importedResources = append(importedResources, rpc.ImportedResource{
				ResourceType: res.ResourceType,
				State:        res.State,
				Private:      res.Private,
			})
		}

		// Convert Terraform diagnostics to RPC format
		var diagnostics []rpc.Diagnostic
		for _, d := range tfResp.Diagnostics {
			diagnostics = append(diagnostics, rpc.Diagnostic{
				Severity:  d.Severity,
				Summary:   d.Summary,
				Detail:    d.Detail,
				Attribute: d.Attribute,
			})
		}

		return &rpc.ImportResourceStateResponse{
			Success:           true,
			ImportedResources: importedResources,
			Diagnostics:       diagnostics,
		}, nil
	}

	// Fall back to existing import handler
	sdkReq := &ImportRequest{
		ResourceType: req.ResourceType,
		ResourceID:   req.ID,
	}

	sdkResp, err := handler.Import(ctx, sdkReq)
	if err != nil {
		return &rpc.ImportResourceStateResponse{
			Success: false,
			Error: &rpc.RPCError{
				Code:    "IMPORT_ERROR",
				Message: err.Error(),
			},
		}, nil
	}

	importedResources := []rpc.ImportedResource{
		{
			ResourceType: req.ResourceType,
			State:        sdkResp.State,
		},
	}

	return &rpc.ImportResourceStateResponse{
		Success:           true,
		ImportedResources: importedResources,
	}, nil
}

// UpgradeResourceState implements rpc.UniversalProvider
func (p *BaseProvider) UpgradeResourceState(ctx context.Context, req *rpc.UpgradeResourceStateRequest) (*rpc.UpgradeResourceStateResponse, error) {
	// Check if handler supports Terraform-compatible upgrade
	if handler, exists := p.resourceHandlers[req.ResourceType]; exists {
		if terraformHandler, ok := handler.(TerraformCompatibleResourceHandler); ok {
			// Use Terraform-compatible upgrade
			tfReq := &TerraformUpgradeStateRequest{
				ResourceType: req.ResourceType,
				Version:      req.Version,
				RawState:     req.RawState,
			}

			tfResp, err := terraformHandler.UpgradeState(ctx, tfReq)
			if err != nil {
				return &rpc.UpgradeResourceStateResponse{
					Success: false,
					Error: &rpc.RPCError{
						Code:    "UPGRADE_ERROR",
						Message: err.Error(),
					},
				}, nil
			}

			// Convert Terraform diagnostics to RPC format
			var diagnostics []rpc.Diagnostic
			for _, d := range tfResp.Diagnostics {
				diagnostics = append(diagnostics, rpc.Diagnostic{
					Severity:  d.Severity,
					Summary:   d.Summary,
					Detail:    d.Detail,
					Attribute: d.Attribute,
				})
			}

			return &rpc.UpgradeResourceStateResponse{
				Success:       true,
				UpgradedState: tfResp.UpgradedState,
				Diagnostics:   diagnostics,
			}, nil
		}
	}

	// Basic implementation - in most cases, no upgrade is needed
	// Provider implementations can override this for complex schema migrations
	return &rpc.UpgradeResourceStateResponse{
		Success:       true,
		UpgradedState: req.RawState, // No changes by default
	}, nil
}

// =============================================================================
// CRUD OPERATION HANDLERS
// =============================================================================

// handleCreate handles resource creation with automatic marshaling
func (p *BaseProvider) handleCreate(ctx context.Context, resourceType string, input json.RawMessage) (json.RawMessage, error) {
	handler, exists := p.resourceHandlers[resourceType]
	if !exists {
		return nil, fmt.Errorf("no handler registered for resource type: %s", resourceType)
	}

	// Convert RPC request to simplified SDK request
	var rpcReq rpc.CreateResourceRequest
	if err := json.Unmarshal(input, &rpcReq); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create request: %w", err)
	}

	sdkReq := &CreateRequest{
		ResourceType: rpcReq.ResourceType,
		Name:         rpcReq.Name,
		Config:       rpcReq.Config,
		Dependencies: rpcReq.Dependencies,
		Options:      rpcReq.Metadata, // Use metadata as options
	}

	// Call the handler
	sdkResp, err := handler.Create(ctx, sdkReq)
	if err != nil {
		return nil, err
	}

	// Convert SDK response back to RPC response
	rpcResp := &rpc.CreateResourceResponse{
		Success:    true,
		ResourceID: sdkResp.ResourceID,
		State:      sdkResp.State,
		Warnings:   sdkResp.Warnings,
		Metadata:   sdkResp.Metadata,
	}

	return json.Marshal(rpcResp)
}

// handleRead handles resource reading with automatic marshaling
func (p *BaseProvider) handleRead(ctx context.Context, resourceType string, input json.RawMessage) (json.RawMessage, error) {
	handler, exists := p.resourceHandlers[resourceType]
	if !exists {
		return nil, fmt.Errorf("no handler registered for resource type: %s", resourceType)
	}

	var rpcReq rpc.ReadResourceRequest
	if err := json.Unmarshal(input, &rpcReq); err != nil {
		return nil, fmt.Errorf("failed to unmarshal read request: %w", err)
	}

	sdkReq := &ReadRequest{
		ResourceType: rpcReq.ResourceType,
		ResourceID:   rpcReq.ResourceID,
		Name:         rpcReq.Name,
	}

	sdkResp, err := handler.Read(ctx, sdkReq)
	if err != nil {
		return nil, err
	}

	rpcResp := &rpc.ReadResourceResponse{
		Success:  !sdkResp.NotFound,
		State:    sdkResp.State,
		NotFound: sdkResp.NotFound,
		Metadata: sdkResp.Metadata,
	}

	return json.Marshal(rpcResp)
}

// handleUpdate handles resource updates with automatic marshaling
func (p *BaseProvider) handleUpdate(ctx context.Context, resourceType string, input json.RawMessage) (json.RawMessage, error) {
	handler, exists := p.resourceHandlers[resourceType]
	if !exists {
		return nil, fmt.Errorf("no handler registered for resource type: %s", resourceType)
	}

	var rpcReq rpc.UpdateResourceRequest
	if err := json.Unmarshal(input, &rpcReq); err != nil {
		return nil, fmt.Errorf("failed to unmarshal update request: %w", err)
	}

	sdkReq := &UpdateRequest{
		ResourceType: rpcReq.ResourceType,
		ResourceID:   rpcReq.ResourceID,
		Name:         rpcReq.Name,
		Config:       rpcReq.Config,
		CurrentState: rpcReq.CurrentState,
	}

	sdkResp, err := handler.Update(ctx, sdkReq)
	if err != nil {
		return nil, err
	}

	rpcResp := &rpc.UpdateResourceResponse{
		Success:  true,
		NewState: sdkResp.NewState,
		Warnings: sdkResp.Warnings,
		Metadata: sdkResp.Metadata,
	}

	return json.Marshal(rpcResp)
}

// handleDelete handles resource deletion with automatic marshaling
func (p *BaseProvider) handleDelete(ctx context.Context, resourceType string, input json.RawMessage) (json.RawMessage, error) {
	handler, exists := p.resourceHandlers[resourceType]
	if !exists {
		return nil, fmt.Errorf("no handler registered for resource type: %s", resourceType)
	}

	var rpcReq rpc.DeleteResourceRequest
	if err := json.Unmarshal(input, &rpcReq); err != nil {
		return nil, fmt.Errorf("failed to unmarshal delete request: %w", err)
	}

	sdkReq := &DeleteRequest{
		ResourceType: rpcReq.ResourceType,
		ResourceID:   rpcReq.ResourceID,
		Name:         rpcReq.Name,
		State:        rpcReq.State,
	}

	sdkResp, err := handler.Delete(ctx, sdkReq)
	if err != nil {
		return nil, err
	}

	rpcResp := &rpc.DeleteResourceResponse{
		Success:  true,
		Warnings: sdkResp.Warnings,
	}

	return json.Marshal(rpcResp)
}

// handleDestroy handles enhanced destroy operations with safety features
func (p *BaseProvider) handleDestroy(ctx context.Context, resourceType string, input json.RawMessage) (json.RawMessage, error) {
	handler, exists := p.resourceHandlers[resourceType]
	if !exists {
		return nil, fmt.Errorf("no handler registered for resource type: %s", resourceType)
	}

	// Try to unmarshal as enhanced destroy request first, fall back to delete
	var destroyReq struct {
		ResourceType         string                 `json:"resource_type"`
		ResourceID           string                 `json:"resource_id"`
		Name                 string                 `json:"name"`
		State                map[string]interface{} `json:"state,omitempty"`
		Force                bool                   `json:"force"`
		CreateBackup         bool                   `json:"create_backup"`
		ValidateDependencies bool                   `json:"validate_dependencies"`
		DryRun               bool                   `json:"dry_run"`
		Options              map[string]interface{} `json:"options,omitempty"`
	}

	if err := json.Unmarshal(input, &destroyReq); err != nil {
		return nil, fmt.Errorf("failed to unmarshal destroy request: %w", err)
	}

	sdkReq := &DestroyRequest{
		ResourceType:         destroyReq.ResourceType,
		ResourceID:           destroyReq.ResourceID,
		Name:                 destroyReq.Name,
		State:                destroyReq.State,
		Force:                destroyReq.Force,
		CreateBackup:         destroyReq.CreateBackup,
		ValidateDependencies: destroyReq.ValidateDependencies,
		DryRun:               destroyReq.DryRun,
		Options:              destroyReq.Options,
	}

	sdkResp, err := handler.Destroy(ctx, sdkReq)
	if err != nil {
		return nil, err
	}

	// Return enhanced destroy response
	return json.Marshal(sdkResp)
}

// handleImport handles resource imports
func (p *BaseProvider) handleImport(ctx context.Context, resourceType string, input json.RawMessage) (json.RawMessage, error) {
	handler, exists := p.resourceHandlers[resourceType]
	if !exists {
		return nil, fmt.Errorf("no handler registered for resource type: %s", resourceType)
	}

	var rpcReq rpc.ImportRequest
	if err := json.Unmarshal(input, &rpcReq); err != nil {
		return nil, fmt.Errorf("failed to unmarshal import request: %w", err)
	}

	sdkReq := &ImportRequest{
		ResourceType: rpcReq.ResourceType,
		ResourceID:   rpcReq.ResourceID,
		Name:         rpcReq.ResourceName,
		Config:       rpcReq.Config,
		Options:      rpcReq.Options.ProviderOptions,
	}

	sdkResp, err := handler.Import(ctx, sdkReq)
	if err != nil {
		return nil, err
	}

	rpcResp := &rpc.ImportResponse{
		Success: true,
		ImportResult: &rpc.ImportResult{
			ResourceType: rpcReq.ResourceType,
			ResourceID:   rpcReq.ResourceID,
			ResourceName: rpcReq.ResourceName,
			State:        sdkResp.State,
			Config:       sdkResp.Config,
			Dependencies: sdkResp.Dependencies,
			Warnings:     sdkResp.Warnings,
		},
	}

	return json.Marshal(rpcResp)
}

// handlePlan handles resource planning
func (p *BaseProvider) handlePlan(ctx context.Context, resourceType string, input json.RawMessage) (json.RawMessage, error) {
	handler, exists := p.resourceHandlers[resourceType]
	if !exists {
		return nil, fmt.Errorf("no handler registered for resource type: %s", resourceType)
	}

	var rpcReq rpc.PlanRequest
	if err := json.Unmarshal(input, &rpcReq); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan request: %w", err)
	}

	// Extract resource-specific config from desired state
	var resourceConfig map[string]interface{}
	if rpcReq.DesiredState != nil {
		resourceConfig = rpcReq.DesiredState
	}

	sdkReq := &PlanRequest{
		ResourceType:  resourceType,
		DesiredConfig: resourceConfig,
		CurrentState:  rpcReq.CurrentState,
		Options:       rpcReq.Options.ProviderOptions,
	}

	sdkResp, err := handler.Plan(ctx, sdkReq)
	if err != nil {
		return nil, err
	}

	// Convert SDK response to RPC format
	var rpcChanges []rpc.PlannedChange
	for _, change := range sdkResp.Changes {
		rpcChanges = append(rpcChanges, rpc.PlannedChange{
			Action:            change.Action,
			ResourceType:      resourceType,
			RequiresDestroy:   change.RequiresDestroy,
			RiskLevel:         rpc.RiskLevel(change.RiskLevel),
			EstimatedDuration: change.EstimatedTime,
		})
	}

	rpcResp := &rpc.PlanResponse{
		Success: true,
		Plan: &rpc.ExecutionPlan{
			Changes:       rpcChanges,
			EstimatedTime: sdkResp.EstimatedTime,
			RiskLevel:     rpc.RiskLevel(sdkResp.RiskLevel),
		},
		Warnings: sdkResp.Warnings,
	}

	return json.Marshal(rpcResp)
}

// handleValidate handles resource validation
func (p *BaseProvider) handleValidate(ctx context.Context, resourceType string, input json.RawMessage) (json.RawMessage, error) {
	handler, exists := p.resourceHandlers[resourceType]
	if !exists {
		return nil, fmt.Errorf("no handler registered for resource type: %s", resourceType)
	}

	var rpcReq rpc.ValidatePlanRequest
	if err := json.Unmarshal(input, &rpcReq); err != nil {
		// Try simpler validation request format
		var simpleReq struct {
			ResourceType string                 `json:"resource_type"`
			Config       map[string]interface{} `json:"config"`
		}
		if err := json.Unmarshal(input, &simpleReq); err != nil {
			return nil, fmt.Errorf("failed to unmarshal validate request: %w", err)
		}

		sdkReq := &ValidateRequest{
			ResourceType: simpleReq.ResourceType,
			Config:       simpleReq.Config,
		}

		sdkResp, err := handler.Validate(ctx, sdkReq)
		if err != nil {
			return nil, err
		}

		// Convert to RPC format
		var rpcErrors []rpc.ValidationError
		for _, e := range sdkResp.Errors {
			rpcErrors = append(rpcErrors, rpc.ValidationError{
				Code:       e.Code,
				Message:    e.Message,
				Field:      e.Field,
				Severity:   e.Severity,
				Suggestion: e.Suggestion,
			})
		}

		var rpcWarnings []rpc.ValidationError
		for _, w := range sdkResp.Warnings {
			rpcWarnings = append(rpcWarnings, rpc.ValidationError{
				Code:       w.Code,
				Message:    w.Message,
				Field:      w.Field,
				Severity:   w.Severity,
				Suggestion: w.Suggestion,
			})
		}

		rpcResp := &rpc.ValidatePlanResponse{
			Success: true,
			Valid:   sdkResp.Valid,
			ValidationResult: &rpc.ValidationResult{
				Valid:    sdkResp.Valid,
				Errors:   rpcErrors,
				Warnings: rpcWarnings,
			},
		}

		return json.Marshal(rpcResp)
	}

	return nil, fmt.Errorf("plan-based validation not yet implemented")
}

// handlePing handles health check requests
func (p *BaseProvider) handlePing(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var rpcReq rpc.PingRequest
	if err := json.Unmarshal(input, &rpcReq); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ping request: %w", err)
	}

	// Basic health check - provider is healthy if it's configured
	rpcResp := &rpc.PingResponse{
		Success: p.configured,
		Healthy: p.configured,
		Details: fmt.Sprintf("%s provider is %s", p.name, map[bool]string{true: "healthy", false: "not configured"}[p.configured]),
	}

	return json.Marshal(rpcResp)
}

// =============================================================================
// OPTIONAL OPERATION HANDLERS
// =============================================================================

// handleGetState handles state retrieval for resources with optional handlers
func (p *BaseProvider) handleGetState(ctx context.Context, resourceType string, input json.RawMessage) (json.RawMessage, error) {
	handler, exists := p.optionalHandlers[resourceType]
	if !exists {
		return nil, fmt.Errorf("no optional handler registered for resource type: %s", resourceType)
	}

	var req GetStateRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal get state request: %w", err)
	}

	resp, err := handler.GetState(ctx, &req)
	if err != nil {
		return nil, err
	}

	return json.Marshal(resp)
}

// handleSetState handles state setting for resources with optional handlers
func (p *BaseProvider) handleSetState(ctx context.Context, resourceType string, input json.RawMessage) (json.RawMessage, error) {
	handler, exists := p.optionalHandlers[resourceType]
	if !exists {
		return nil, fmt.Errorf("no optional handler registered for resource type: %s", resourceType)
	}

	var req SetStateRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal set state request: %w", err)
	}

	resp, err := handler.SetState(ctx, &req)
	if err != nil {
		return nil, err
	}

	return json.Marshal(resp)
}

// handleDetectDrift handles drift detection for resources with optional handlers
func (p *BaseProvider) handleDetectDrift(ctx context.Context, resourceType string, input json.RawMessage) (json.RawMessage, error) {
	handler, exists := p.optionalHandlers[resourceType]
	if !exists {
		return nil, fmt.Errorf("no optional handler registered for resource type: %s", resourceType)
	}

	var req DetectDriftRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal detect drift request: %w", err)
	}

	resp, err := handler.DetectDrift(ctx, &req)
	if err != nil {
		return nil, err
	}

	return json.Marshal(resp)
}

// handleDiscover handles resource discovery
func (p *BaseProvider) handleDiscover(ctx context.Context, resourceType string, input json.RawMessage) (json.RawMessage, error) {
	handler, exists := p.optionalHandlers[resourceType]
	if !exists {
		return nil, fmt.Errorf("no optional handler registered for resource type: %s", resourceType)
	}

	var req DiscoverRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal discover request: %w", err)
	}

	resp, err := handler.Discover(ctx, &req)
	if err != nil {
		return nil, err
	}

	return json.Marshal(resp)
}

// handleIntrospect handles resource introspection
func (p *BaseProvider) handleIntrospect(ctx context.Context, resourceType string, input json.RawMessage) (json.RawMessage, error) {
	handler, exists := p.optionalHandlers[resourceType]
	if !exists {
		return nil, fmt.Errorf("no optional handler registered for resource type: %s", resourceType)
	}

	var req IntrospectRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal introspect request: %w", err)
	}

	resp, err := handler.Introspect(ctx, &req)
	if err != nil {
		return nil, err
	}

	return json.Marshal(resp)
}

// handleGetMetrics handles metrics retrieval
func (p *BaseProvider) handleGetMetrics(ctx context.Context, resourceType string, input json.RawMessage) (json.RawMessage, error) {
	handler, exists := p.optionalHandlers[resourceType]
	if !exists {
		return nil, fmt.Errorf("no optional handler registered for resource type: %s", resourceType)
	}

	var req GetMetricsRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal get metrics request: %w", err)
	}

	resp, err := handler.GetMetrics(ctx, &req)
	if err != nil {
		return nil, err
	}

	return json.Marshal(resp)
}

// handleBackup handles backup operations
func (p *BaseProvider) handleBackup(ctx context.Context, resourceType string, input json.RawMessage) (json.RawMessage, error) {
	handler, exists := p.optionalHandlers[resourceType]
	if !exists {
		return nil, fmt.Errorf("no optional handler registered for resource type: %s", resourceType)
	}

	var req BackupRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backup request: %w", err)
	}

	resp, err := handler.Backup(ctx, &req)
	if err != nil {
		return nil, err
	}

	return json.Marshal(resp)
}

// handleRestore handles restore operations
func (p *BaseProvider) handleRestore(ctx context.Context, resourceType string, input json.RawMessage) (json.RawMessage, error) {
	handler, exists := p.optionalHandlers[resourceType]
	if !exists {
		return nil, fmt.Errorf("no optional handler registered for resource type: %s", resourceType)
	}

	var req RestoreRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal restore request: %w", err)
	}

	resp, err := handler.Restore(ctx, &req)
	if err != nil {
		return nil, err
	}

	return json.Marshal(resp)
}

// =============================================================================
// HELPER METHODS
// =============================================================================

// getSupportedFunctions returns the list of functions this provider supports
func (p *BaseProvider) getSupportedFunctions() map[string]types.FunctionSpec {
	functions := make(map[string]types.FunctionSpec)

	// Add core CRUD functions for each registered resource type
	for resourceType := range p.resourceHandlers {
		functions[fmt.Sprintf("CreateResource_%s", resourceType)] = types.FunctionSpec{
			Description: fmt.Sprintf("Create a %s resource", resourceType),
			Idempotent:  false,
		}
		functions[fmt.Sprintf("ReadResource_%s", resourceType)] = types.FunctionSpec{
			Description: fmt.Sprintf("Read a %s resource", resourceType),
			Idempotent:  true,
		}
		functions[fmt.Sprintf("UpdateResource_%s", resourceType)] = types.FunctionSpec{
			Description: fmt.Sprintf("Update a %s resource", resourceType),
			Idempotent:  false,
		}
		functions[fmt.Sprintf("DeleteResource_%s", resourceType)] = types.FunctionSpec{
			Description: fmt.Sprintf("Delete a %s resource", resourceType),
			Idempotent:  true,
		}
		functions[fmt.Sprintf("DestroyResource_%s", resourceType)] = types.FunctionSpec{
			Description: fmt.Sprintf("Destroy a %s resource with safety checks", resourceType),
			Idempotent:  true,
		}
	}

	// Add optional functions for resource types with optional handlers
	for resourceType := range p.optionalHandlers {
		functions[fmt.Sprintf("GetState_%s", resourceType)] = types.FunctionSpec{
			Description: fmt.Sprintf("Get state for %s resource", resourceType),
			Idempotent:  true,
		}
		functions[fmt.Sprintf("DetectDrift_%s", resourceType)] = types.FunctionSpec{
			Description: fmt.Sprintf("Detect drift for %s resource", resourceType),
			Idempotent:  true,
		}
		functions[fmt.Sprintf("DiscoverResources_%s", resourceType)] = types.FunctionSpec{
			Description: fmt.Sprintf("Discover %s resources", resourceType),
			Idempotent:  true,
		}
	}

	// Add universal functions
	functions["Ping"] = types.FunctionSpec{
		Description: "Health check",
		Idempotent:  true,
	}

	return functions
}

// Ensure BaseProvider implements rpc.UniversalProvider
var _ rpc.UniversalProvider = (*BaseProvider)(nil)
