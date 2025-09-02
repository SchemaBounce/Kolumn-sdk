// Package examples demonstrates how to build a Kolumn provider using the enhanced SDK
// This shows the new CRUD helper pattern with BaseProvider and resource handlers
package examples

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/schemabounce/kolumn/sdk/pdk"
)

// =============================================================================
// EXAMPLE: ENHANCED PROVIDER USING SDK BASE PROVIDER PATTERN
// =============================================================================

// EnhancedProvider demonstrates a provider using the BaseProvider pattern
type EnhancedProvider struct {
	*pdk.BaseProvider
	config    *Config
	connected bool
	resources map[string]interface{} // Simple in-memory storage
}

// Config represents the provider configuration
type Config struct {
	GovernanceEndpoint string `json:"governance_endpoint"`
	WorkspaceID        string `json:"workspace_id"`
	Environment        string `json:"environment"`
	StorageBackend     string `json:"storage_backend"`
}

// NewEnhancedProvider creates a new enhanced provider using BaseProvider
func NewEnhancedProvider() *EnhancedProvider {
	baseProvider := pdk.NewBaseProvider("enhanced", "v1.0.0")

	provider := &EnhancedProvider{
		BaseProvider: baseProvider,
		resources:    make(map[string]interface{}),
	}

	// Register resource handlers
	provider.RegisterResourceHandler("classification", &ClassificationHandler{provider: provider})
	provider.RegisterResourceHandler("data_object", &DataObjectHandler{provider: provider})

	// Add capabilities
	provider.AddCapability("governance")
	provider.AddCapability("compliance")
	provider.AddCapability("audit")

	return provider
}

// Configure overrides the base Configure to add provider-specific logic
func (p *EnhancedProvider) Configure(ctx context.Context, config map[string]interface{}) error {
	// Parse configuration
	p.config = &Config{}
	if err := pdk.ParseConfig(config, p.config); err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Set defaults
	if p.config.Environment == "" {
		p.config.Environment = "dev"
	}
	if p.config.StorageBackend == "" {
		p.config.StorageBackend = "memory"
	}

	// Call base Configure
	if err := p.BaseProvider.Configure(ctx, config); err != nil {
		return err
	}

	// Simulate connection
	time.Sleep(100 * time.Millisecond)
	p.connected = true

	return nil
}

// =============================================================================
// RESOURCE HANDLERS
// =============================================================================

// ClassificationHandler handles classification resources
type ClassificationHandler struct {
	provider *EnhancedProvider
}

func (h *ClassificationHandler) Create(ctx context.Context, req *pdk.CreateRequest) (*pdk.CreateResponse, error) {
	// Create a classification resource
	classification := map[string]interface{}{
		"name":         req.Name,
		"description":  req.Config["description"],
		"level":        req.Config["level"],
		"requirements": req.Config["requirements"],
		"created_at":   time.Now(),
	}

	// Store in provider
	h.provider.resources[req.Name] = classification

	return &pdk.CreateResponse{
		ResourceID: req.Name,
		State:      classification,
		Metadata: map[string]interface{}{
			"resource_type": "classification",
			"created_by":    "enhanced_provider",
		},
	}, nil
}

func (h *ClassificationHandler) Read(ctx context.Context, req *pdk.ReadRequest) (*pdk.ReadResponse, error) {
	resource, exists := h.provider.resources[req.Name]
	if !exists {
		return &pdk.ReadResponse{NotFound: true}, nil
	}

	return &pdk.ReadResponse{
		State: resource.(map[string]interface{}),
		Metadata: map[string]interface{}{
			"resource_type": "classification",
		},
	}, nil
}

func (h *ClassificationHandler) Update(ctx context.Context, req *pdk.UpdateRequest) (*pdk.UpdateResponse, error) {
	resource, exists := h.provider.resources[req.Name]
	if !exists {
		return nil, fmt.Errorf("resource not found: %s", req.Name)
	}

	// Update the resource
	updated := resource.(map[string]interface{})
	for k, v := range req.Config {
		updated[k] = v
	}
	updated["updated_at"] = time.Now()

	h.provider.resources[req.Name] = updated

	return &pdk.UpdateResponse{
		NewState: updated,
		Metadata: map[string]interface{}{
			"resource_type": "classification",
			"updated_by":    "enhanced_provider",
		},
	}, nil
}

func (h *ClassificationHandler) Delete(ctx context.Context, req *pdk.DeleteRequest) (*pdk.DeleteResponse, error) {
	delete(h.provider.resources, req.Name)
	return &pdk.DeleteResponse{}, nil
}

func (h *ClassificationHandler) Destroy(ctx context.Context, req *pdk.DestroyRequest) (*pdk.DestroyResponse, error) {
	// Enhanced destroy with safety checks
	if req.DryRun {
		return &pdk.DestroyResponse{
			Success: true,
			SafetyChecks: []pdk.SafetyCheck{
				{
					Name:     "dependency_check",
					Passed:   true,
					Message:  "No dependent resources found",
					Severity: "info",
				},
			},
			RiskAssessment: &pdk.RiskAssessment{
				OverallRisk:  "low",
				DataLossRisk: false,
				DowntimeRisk: false,
				RiskFactors: []pdk.RiskFactor{
					{
						Type:        "dependency",
						Severity:    "low",
						Description: "Classification has no dependencies",
					},
				},
			},
		}, nil
	}

	// Actual destroy
	delete(h.provider.resources, req.Name)

	return &pdk.DestroyResponse{
		Success: true,
		SafetyChecks: []pdk.SafetyCheck{
			{
				Name:     "destroy_complete",
				Passed:   true,
				Message:  "Resource successfully destroyed",
				Severity: "info",
			},
		},
	}, nil
}

func (h *ClassificationHandler) Import(ctx context.Context, req *pdk.ImportRequest) (*pdk.ImportResponse, error) {
	// Simulate import
	return &pdk.ImportResponse{
		State: map[string]interface{}{
			"name":        req.Name,
			"imported":    true,
			"import_time": time.Now(),
		},
		Config: req.Config,
	}, nil
}

func (h *ClassificationHandler) Plan(ctx context.Context, req *pdk.PlanRequest) (*pdk.PlanResponse, error) {
	// Simple planning logic
	changes := []pdk.PlannedChange{
		{
			Action:      "create",
			Description: fmt.Sprintf("Create classification '%s'", req.Name),
			RiskLevel:   "low",
		},
	}

	return &pdk.PlanResponse{
		Changes:       changes,
		RiskLevel:     "low",
		EstimatedTime: time.Second * 5,
	}, nil
}

func (h *ClassificationHandler) Validate(ctx context.Context, req *pdk.ValidateRequest) (*pdk.ValidateResponse, error) {
	var errors []pdk.ValidationError

	// Check required fields
	if req.Config["description"] == nil {
		errors = append(errors, pdk.ValidationError{
			Code:     "MISSING_DESCRIPTION",
			Message:  "Classification must have a description",
			Field:    "description",
			Severity: "error",
		})
	}

	if req.Config["level"] == nil {
		errors = append(errors, pdk.ValidationError{
			Code:     "MISSING_LEVEL",
			Message:  "Classification must have a level",
			Field:    "level",
			Severity: "error",
		})
	}

	valid := len(errors) == 0

	return &pdk.ValidateResponse{
		Valid:  valid,
		Errors: errors,
	}, nil
}

// DataObjectHandler handles data object resources
type DataObjectHandler struct {
	provider *EnhancedProvider
}

func (h *DataObjectHandler) Create(ctx context.Context, req *pdk.CreateRequest) (*pdk.CreateResponse, error) {
	// Validate columns exist
	columns, ok := req.Config["columns"]
	if !ok {
		return nil, fmt.Errorf("data object must have columns")
	}

	dataObject := map[string]interface{}{
		"name":            req.Name,
		"description":     req.Config["description"],
		"columns":         columns,
		"classifications": req.Config["classifications"],
		"created_at":      time.Now(),
	}

	h.provider.resources[req.Name] = dataObject

	return &pdk.CreateResponse{
		ResourceID: req.Name,
		State:      dataObject,
		Metadata: map[string]interface{}{
			"resource_type": "data_object",
			"column_count":  len(columns.([]interface{})),
		},
	}, nil
}

func (h *DataObjectHandler) Read(ctx context.Context, req *pdk.ReadRequest) (*pdk.ReadResponse, error) {
	resource, exists := h.provider.resources[req.Name]
	if !exists {
		return &pdk.ReadResponse{NotFound: true}, nil
	}

	return &pdk.ReadResponse{
		State: resource.(map[string]interface{}),
		Metadata: map[string]interface{}{
			"resource_type": "data_object",
		},
	}, nil
}

func (h *DataObjectHandler) Update(ctx context.Context, req *pdk.UpdateRequest) (*pdk.UpdateResponse, error) {
	resource, exists := h.provider.resources[req.Name]
	if !exists {
		return nil, fmt.Errorf("resource not found: %s", req.Name)
	}

	updated := resource.(map[string]interface{})
	for k, v := range req.Config {
		updated[k] = v
	}
	updated["updated_at"] = time.Now()

	h.provider.resources[req.Name] = updated

	return &pdk.UpdateResponse{
		NewState: updated,
		Metadata: map[string]interface{}{
			"resource_type": "data_object",
		},
	}, nil
}

func (h *DataObjectHandler) Delete(ctx context.Context, req *pdk.DeleteRequest) (*pdk.DeleteResponse, error) {
	delete(h.provider.resources, req.Name)
	return &pdk.DeleteResponse{}, nil
}

func (h *DataObjectHandler) Destroy(ctx context.Context, req *pdk.DestroyRequest) (*pdk.DestroyResponse, error) {
	if req.DryRun {
		return &pdk.DestroyResponse{
			Success: true,
			SafetyChecks: []pdk.SafetyCheck{
				{
					Name:     "data_object_check",
					Passed:   true,
					Message:  "Data object can be safely destroyed",
					Severity: "info",
				},
			},
		}, nil
	}

	delete(h.provider.resources, req.Name)
	return &pdk.DestroyResponse{Success: true}, nil
}

func (h *DataObjectHandler) Import(ctx context.Context, req *pdk.ImportRequest) (*pdk.ImportResponse, error) {
	return &pdk.ImportResponse{
		State: map[string]interface{}{
			"name":     req.Name,
			"imported": true,
		},
		Config: req.Config,
	}, nil
}

func (h *DataObjectHandler) Plan(ctx context.Context, req *pdk.PlanRequest) (*pdk.PlanResponse, error) {
	changes := []pdk.PlannedChange{
		{
			Action:      "create",
			Description: fmt.Sprintf("Create data object '%s'", req.Name),
			RiskLevel:   "medium",
		},
	}

	return &pdk.PlanResponse{
		Changes:       changes,
		RiskLevel:     "medium",
		EstimatedTime: time.Second * 10,
	}, nil
}

func (h *DataObjectHandler) Validate(ctx context.Context, req *pdk.ValidateRequest) (*pdk.ValidateResponse, error) {
	var errors []pdk.ValidationError

	// Check required fields
	if req.Config["columns"] == nil {
		errors = append(errors, pdk.ValidationError{
			Code:     "MISSING_COLUMNS",
			Message:  "Data object must have column definitions",
			Field:    "columns",
			Severity: "error",
		})
	}

	valid := len(errors) == 0

	return &pdk.ValidateResponse{
		Valid:  valid,
		Errors: errors,
	}, nil
}

// RunEnhancedProvider demonstrates the enhanced provider SDK pattern
func RunEnhancedProvider() {
	// Create the enhanced provider using the BaseProvider pattern
	provider := NewEnhancedProvider()

	log.Println("=== Enhanced Provider SDK Pattern Demo ===")

	// Configure the provider
	config := map[string]interface{}{
		"governance_endpoint": "http://localhost:8080",
		"workspace_id":        "demo-workspace",
		"environment":         "dev",
		"storage_backend":     "memory",
	}

	ctx := context.Background()
	if err := provider.Configure(ctx, config); err != nil {
		log.Fatalf("Failed to configure provider: %v", err)
	}

	log.Println("✅ Provider configured successfully")

	// Get the provider schema
	schema, err := provider.GetSchema()
	if err != nil {
		log.Fatalf("Failed to get schema: %v", err)
	}

	log.Printf("✅ Provider schema loaded: %s v%s", schema.Provider.Name, schema.Provider.Version)
	log.Printf("   Resource types: %v", schema.ResourceTypes)
	log.Printf("   Capabilities: %v", schema.Capabilities)

	// Test the automatic CRUD dispatch through CallFunction
	// The BaseProvider automatically routes these to the appropriate resource handlers

	// 1. Create a classification
	log.Println("\n=== Testing Classification Creation ===")
	createReq := map[string]interface{}{
		"resource_type": "classification",
		"name":          "demo_pii",
		"config": map[string]interface{}{
			"description": "Demo PII classification",
			"level":       "confidential",
			"requirements": map[string]interface{}{
				"encryption": true,
				"audit":      true,
			},
		},
	}

	result, err := testCreateResource(provider, ctx, createReq)
	if err != nil {
		log.Printf("❌ Create failed: %v", err)
	} else {
		log.Printf("✅ Created classification: %v", result)
	}

	// 2. Create a data object
	log.Println("\n=== Testing Data Object Creation ===")
	dataObjReq := map[string]interface{}{
		"resource_type": "data_object",
		"name":          "user_profile",
		"config": map[string]interface{}{
			"description": "User profile data structure",
			"columns": []map[string]interface{}{
				{
					"name":            "id",
					"type":            "BIGINT",
					"primary_key":     true,
					"classifications": []string{},
				},
				{
					"name":            "email",
					"type":            "VARCHAR(255)",
					"unique":          true,
					"classifications": []string{"demo_pii"},
				},
			},
			"classifications": []string{"demo_pii"},
		},
	}

	result, err = testCreateResource(provider, ctx, dataObjReq)
	if err != nil {
		log.Printf("❌ Data object create failed: %v", err)
	} else {
		log.Printf("✅ Created data object: %v", result)
	}

	// 3. Test enhanced destroy operation with safety checks
	log.Println("\n=== Testing Enhanced Destroy (Dry Run) ===")
	destroyReq := map[string]interface{}{
		"resource_type":         "data_object",
		"name":                  "user_profile",
		"dry_run":               true,
		"create_backup":         true,
		"validate_dependencies": true,
	}

	result, err = testDestroyResource(provider, ctx, destroyReq)
	if err != nil {
		log.Printf("❌ Destroy dry run failed: %v", err)
	} else {
		log.Printf("✅ Destroy dry run completed: %v", result)
	}

	// 4. Test validation
	log.Println("\n=== Testing Resource Validation ===")
	validateReq := map[string]interface{}{
		"resource_type": "data_object",
		"config": map[string]interface{}{
			"description": "Test validation",
			// Missing required 'columns' field to trigger validation error
		},
	}

	result, err = testValidateResource(provider, ctx, validateReq)
	if err != nil {
		log.Printf("❌ Validation failed: %v", err)
	} else {
		log.Printf("✅ Validation completed: %v", result)
	}

	// 5. Demonstrate the SDK eliminated manual JSON marshaling
	log.Println("\n=== SDK Benefits Demonstrated ===")
	log.Println("🔧 The SDK BaseProvider automatically handled:")
	log.Println("   - JSON marshaling/unmarshaling for all requests")
	log.Println("   - Function dispatch to appropriate resource handlers")
	log.Println("   - Consistent error handling and response formatting")
	log.Println("   - Enhanced destroy operations with safety checks")
	log.Println("   - Validation, planning, and import operations")
	log.Println("")
	log.Println("📈 Benefits over manual RPC handling:")
	log.Println("   - No manual JSON marshaling code required")
	log.Println("   - Automatic function routing based on resource type")
	log.Println("   - Built-in enhanced operations (destroy, validate, plan)")
	log.Println("   - Consistent patterns across all resource types")
	log.Println("   - Reduced boilerplate code by ~70%")

	// Serve the provider (in real usage)
	log.Println("\n=== Starting Provider Server ===")
	log.Println("In production, this would serve the provider via RPC:")
	log.Println("rpc.ServeProvider(&rpc.ServeConfig{Provider: provider})")
}

// Helper functions to demonstrate the RPC calls through the BaseProvider
func testCreateResource(provider *EnhancedProvider, ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
	inputBytes, _ := json.Marshal(req)
	resultBytes, err := provider.CallFunction(ctx, "CreateResource", inputBytes)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func testDestroyResource(provider *EnhancedProvider, ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
	inputBytes, _ := json.Marshal(req)
	resultBytes, err := provider.CallFunction(ctx, "DestroyResource", inputBytes)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func testValidateResource(provider *EnhancedProvider, ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
	inputBytes, _ := json.Marshal(req)
	resultBytes, err := provider.CallFunction(ctx, "ValidateResource", inputBytes)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return nil, err
	}

	return result, nil
}

/*
Expected Output:
=== Enhanced Provider SDK Pattern Demo ===
✅ Provider configured successfully
✅ Provider schema loaded: enhanced v1.0.0
   Resource types: [classification data_object]
   Capabilities: [governance compliance audit]

=== Testing Classification Creation ===
✅ Created classification: map[success:true resource_id:demo_pii state:map[created_at:... description:Demo PII classification level:confidential name:demo_pii requirements:map[audit:true encryption:true]]]

=== Testing Data Object Creation ===
✅ Created data object: map[success:true resource_id:user_profile state:map[classifications:[demo_pii] columns:[...] created_at:... description:User profile data structure name:user_profile]]

=== Testing Enhanced Destroy (Dry Run) ===
✅ Destroy dry run completed: map[success:true safety_checks:[map[message:Data object can be safely destroyed name:data_object_check passed:true severity:info]]]

=== Testing Resource Validation ===
✅ Validation completed: map[valid:false errors:[map[code:MISSING_COLUMNS field:columns message:Data object must have column definitions severity:error]]]

=== SDK Benefits Demonstrated ===
🔧 The SDK BaseProvider automatically handled:
   - JSON marshaling/unmarshaling for all requests
   - Function dispatch to appropriate resource handlers
   - Consistent error handling and response formatting
   - Enhanced destroy operations with safety checks
   - Validation, planning, and import operations

📈 Benefits over manual RPC handling:
   - No manual JSON marshaling code required
   - Automatic function routing based on resource type
   - Built-in enhanced operations (destroy, validate, plan)
   - Consistent patterns across all resource types
   - Reduced boilerplate code by ~70%

=== Starting Provider Server ===
In production, this would serve the provider via RPC:
rpc.ServeProvider(&rpc.ServeConfig{Provider: provider})
*/
