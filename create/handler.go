// Package create provides utilities for implementing CREATE object handlers
//
// CREATE objects are resources that providers can create, update, and manage.
// Examples: tables, indexes, users, buckets, topics, clusters
package create

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/schemabounce/kolumn/sdk/core"
	"github.com/schemabounce/kolumn/sdk/helpers/security"
)

// ObjectHandler defines the interface for handling CREATE objects
type ObjectHandler interface {
	// Create creates a new instance of this object type
	Create(ctx context.Context, req *CreateRequest) (*CreateResponse, error)

	// Read retrieves the current state of an object instance
	Read(ctx context.Context, req *ReadRequest) (*ReadResponse, error)

	// Update modifies an existing object instance
	Update(ctx context.Context, req *UpdateRequest) (*UpdateResponse, error)

	// Delete removes an object instance
	Delete(ctx context.Context, req *DeleteRequest) (*DeleteResponse, error)

	// Plan calculates what changes would be made (optional)
	Plan(ctx context.Context, req *PlanRequest) (*PlanResponse, error)
}

// EnhancedObjectHandler extends ObjectHandler with advanced features
type EnhancedObjectHandler interface {
	ObjectHandler

	// Validate validates object configuration before creation
	Validate(ctx context.Context, req *ValidateRequest) (*ValidateResponse, error)

	// Import imports existing infrastructure as managed objects
	Import(ctx context.Context, req *ImportRequest) (*ImportResponse, error)

	// GetState returns the raw state of an object instance
	GetState(ctx context.Context, req *GetStateRequest) (*GetStateResponse, error)
}

// Use types from core package to avoid duplication and ensure consistency
type (
	CreateRequest  = core.CreateRequest
	CreateResponse = core.CreateResponse
	ReadRequest    = core.ReadRequest
	ReadResponse   = core.ReadResponse
	UpdateRequest  = core.UpdateRequest
	UpdateResponse = core.UpdateResponse
	DeleteRequest  = core.DeleteRequest
	DeleteResponse = core.DeleteResponse
	PlanRequest    = core.PlanRequest
	PlanResponse   = core.PlanResponse
)

// ValidateRequest contains configuration to validate
type ValidateRequest struct {
	ObjectType string                 `json:"object_type"`
	Config     map[string]interface{} `json:"config"`
}

// ValidateResponse contains validation results
type ValidateResponse struct {
	Valid    bool               `json:"valid"`
	Errors   []*ValidationIssue `json:"errors,omitempty"`
	Warnings []*ValidationIssue `json:"warnings,omitempty"`
}

// ImportRequest specifies existing infrastructure to import
type ImportRequest struct {
	ObjectType   string                 `json:"object_type"`
	ID           string                 `json:"id"`
	Name         string                 `json:"name,omitempty"`
	ImportConfig map[string]interface{} `json:"import_config,omitempty"`
}

// ImportResponse contains the imported object state
type ImportResponse struct {
	State        map[string]interface{} `json:"state"`
	Config       map[string]interface{} `json:"config"`
	Dependencies []string               `json:"dependencies,omitempty"`
}

// GetStateRequest specifies which object state to retrieve
type GetStateRequest struct {
	ObjectType string `json:"object_type"`
	ID         string `json:"id"`
}

// GetStateResponse contains raw object state
type GetStateResponse struct {
	State   map[string]interface{} `json:"state"`
	Version string                 `json:"version,omitempty"`
}

// Change represents a modification made to an object
type Change struct {
	Action      string      `json:"action"` // "create", "update", "delete"
	Field       string      `json:"field"`
	OldValue    interface{} `json:"old_value,omitempty"`
	NewValue    interface{} `json:"new_value,omitempty"`
	Description string      `json:"description"`
}

// Use PlannedChange from core package
type PlannedChange = core.PlannedChange

// ValidationIssue represents a validation error or warning
type ValidationIssue struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// Registry manages CREATE object handlers
type Registry struct {
	handlers map[string]ObjectHandler
	schemas  map[string]*core.ObjectType
}

// NewRegistry creates a new CREATE object registry
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]ObjectHandler),
		schemas:  make(map[string]*core.ObjectType),
	}
}

// RegisterHandler registers a handler for a CREATE object type
func (r *Registry) RegisterHandler(objectType string, handler ObjectHandler, schema *core.ObjectType) error {
	if schema.Type != core.CREATE {
		return fmt.Errorf("schema type must be CREATE for object type %s", objectType)
	}

	r.handlers[objectType] = handler
	r.schemas[objectType] = schema
	return nil
}

// GetHandler returns the handler for an object type
func (r *Registry) GetHandler(objectType string) (ObjectHandler, bool) {
	handler, exists := r.handlers[objectType]
	return handler, exists
}

// GetSchema returns the schema for an object type
func (r *Registry) GetSchema(objectType string) (*core.ObjectType, bool) {
	schema, exists := r.schemas[objectType]
	return schema, exists
}

// GetObjectTypes returns all registered CREATE object types
func (r *Registry) GetObjectTypes() map[string]*core.ObjectType {
	result := make(map[string]*core.ObjectType)
	for k, v := range r.schemas {
		result[k] = v
	}
	return result
}

// CallHandler executes a handler method by name with comprehensive security validation
func (r *Registry) CallHandler(ctx context.Context, objectType, method string, input []byte) ([]byte, error) {
	// SECURITY: Validate object type to prevent injection
	if err := security.ValidateObjectType(objectType); err != nil {
		secErr := security.NewSecureError(
			"invalid object type",
			fmt.Sprintf("object type validation failed: %v", err),
			"INVALID_OBJECT_TYPE",
		)
		return nil, secErr
	}

	// SECURITY: Validate method name against whitelist
	if err := security.ValidateMethod(method); err != nil {
		secErr := security.NewSecureError(
			"operation not supported",
			fmt.Sprintf("method validation failed: %s for object type %s", method, objectType),
			"INVALID_METHOD",
		)
		return nil, secErr
	}

	// SECURITY: Check if handler exists before proceeding
	handler, exists := r.GetHandler(objectType)
	if !exists {
		secErr := security.NewSecureError(
			"object type not supported",
			fmt.Sprintf("no handler registered for object type: %s", objectType),
			"HANDLER_NOT_FOUND",
		)
		return nil, secErr
	}

	// SECURITY: Use secure JSON unmarshaling with size and depth limits
	switch method {
	case "create":
		var req CreateRequest
		if err := security.SafeUnmarshal(input, &req); err != nil {
			secErr := security.NewSecureError(
				"invalid request format",
				fmt.Sprintf("create request unmarshal failed: %v", err),
				"INVALID_REQUEST",
			)
			return nil, secErr
		}

		// SECURITY: Validate request configuration size
		validator := &security.InputSizeValidator{}
		if err := validator.ValidateConfigSize(req.Config); err != nil {
			secErr := security.NewSecureError(
				"request too large",
				fmt.Sprintf("create request config validation failed: %v", err),
				"REQUEST_TOO_LARGE",
			)
			return nil, secErr
		}

		resp, err := handler.Create(ctx, &req)
		if err != nil {
			secErr := security.NewSecureError(
				"operation failed",
				fmt.Sprintf("create operation failed: %v", err),
				"OPERATION_FAILED",
			)
			return nil, secErr
		}
		return json.Marshal(resp)

	case "read":
		var req ReadRequest
		if err := security.SafeUnmarshal(input, &req); err != nil {
			secErr := security.NewSecureError(
				"invalid request format",
				fmt.Sprintf("read request unmarshal failed: %v", err),
				"INVALID_REQUEST",
			)
			return nil, secErr
		}

		resp, err := handler.Read(ctx, &req)
		if err != nil {
			secErr := security.NewSecureError(
				"operation failed",
				fmt.Sprintf("read operation failed: %v", err),
				"OPERATION_FAILED",
			)
			return nil, secErr
		}
		return json.Marshal(resp)

	case "update":
		var req UpdateRequest
		if err := security.SafeUnmarshal(input, &req); err != nil {
			secErr := security.NewSecureError(
				"invalid request format",
				fmt.Sprintf("update request unmarshal failed: %v", err),
				"INVALID_REQUEST",
			)
			return nil, secErr
		}

		// SECURITY: Validate request configuration size
		validator := &security.InputSizeValidator{}
		if err := validator.ValidateConfigSize(req.Config); err != nil {
			secErr := security.NewSecureError(
				"request too large",
				fmt.Sprintf("update request config validation failed: %v", err),
				"REQUEST_TOO_LARGE",
			)
			return nil, secErr
		}

		resp, err := handler.Update(ctx, &req)
		if err != nil {
			secErr := security.NewSecureError(
				"operation failed",
				fmt.Sprintf("update operation failed: %v", err),
				"OPERATION_FAILED",
			)
			return nil, secErr
		}
		return json.Marshal(resp)

	case "delete":
		var req DeleteRequest
		if err := security.SafeUnmarshal(input, &req); err != nil {
			secErr := security.NewSecureError(
				"invalid request format",
				fmt.Sprintf("delete request unmarshal failed: %v", err),
				"INVALID_REQUEST",
			)
			return nil, secErr
		}

		resp, err := handler.Delete(ctx, &req)
		if err != nil {
			secErr := security.NewSecureError(
				"operation failed",
				fmt.Sprintf("delete operation failed: %v", err),
				"OPERATION_FAILED",
			)
			return nil, secErr
		}
		return json.Marshal(resp)

	case "plan":
		var req PlanRequest
		if err := security.SafeUnmarshal(input, &req); err != nil {
			secErr := security.NewSecureError(
				"invalid request format",
				fmt.Sprintf("plan request unmarshal failed: %v", err),
				"INVALID_REQUEST",
			)
			return nil, secErr
		}

		// SECURITY: Validate request configuration size
		validator := &security.InputSizeValidator{}
		if err := validator.ValidateConfigSize(req.DesiredConfig); err != nil {
			secErr := security.NewSecureError(
				"request too large",
				fmt.Sprintf("plan request config validation failed: %v", err),
				"REQUEST_TOO_LARGE",
			)
			return nil, secErr
		}

		resp, err := handler.Plan(ctx, &req)
		if err != nil {
			secErr := security.NewSecureError(
				"operation failed",
				fmt.Sprintf("plan operation failed: %v", err),
				"OPERATION_FAILED",
			)
			return nil, secErr
		}
		return json.Marshal(resp)

	default:
		// This should never be reached due to method validation above
		secErr := security.NewSecureError(
			"operation not supported",
			fmt.Sprintf("unexpected method %s for object type %s", method, objectType),
			"UNEXPECTED_METHOD",
		)
		return nil, secErr
	}
}

// =============================================================================
// ADVANCED HANDLER IMPLEMENTATION
// =============================================================================

// AdvancedHandler provides an advanced implementation of ObjectHandler with extensible components
type AdvancedHandler struct {
	objectType     string
	schema         *core.ObjectType
	validators     []Validator
	planners       []Planner
	importers      []Importer
	driftDetectors []DriftDetector
}

// NewAdvancedHandler creates a new AdvancedHandler for the specified object type
func NewAdvancedHandler(objectType string) *AdvancedHandler {
	return &AdvancedHandler{
		objectType:     objectType,
		validators:     make([]Validator, 0),
		planners:       make([]Planner, 0),
		importers:      make([]Importer, 0),
		driftDetectors: make([]DriftDetector, 0),
		schema: &core.ObjectType{
			Name:       objectType,
			Type:       core.CREATE,
			Properties: make(map[string]*core.Property),
			Required:   make([]string, 0),
			Optional:   make([]string, 0),
		},
	}
}

// Schema returns the object type schema
func (h *AdvancedHandler) Schema() *core.ObjectType {
	return h.schema
}

// AddValidator adds a validator to the handler
func (h *AdvancedHandler) AddValidator(validator Validator) {
	h.validators = append(h.validators, validator)
}

// AddPlanner adds a planner to the handler
func (h *AdvancedHandler) AddPlanner(planner Planner) {
	h.planners = append(h.planners, planner)
}

// AddImporter adds an importer to the handler
func (h *AdvancedHandler) AddImporter(importer Importer) {
	h.importers = append(h.importers, importer)
}

// AddDriftDetector adds a drift detector to the handler
func (h *AdvancedHandler) AddDriftDetector(detector DriftDetector) {
	h.driftDetectors = append(h.driftDetectors, detector)
}

// ValidateConfig validates configuration using all registered validators
func (h *AdvancedHandler) ValidateConfig(config map[string]interface{}) error {
	for _, validator := range h.validators {
		if err := validator.Validate(config); err != nil {
			return fmt.Errorf("validation failed for %s: %w", validator.Name(), err)
		}
	}
	return nil
}

// Default implementations for ObjectHandler interface
// These should be overridden by specific handler implementations

func (h *AdvancedHandler) Create(ctx context.Context, req *CreateRequest) (*CreateResponse, error) {
	return nil, fmt.Errorf("Create method not implemented for object type: %s", h.objectType)
}

func (h *AdvancedHandler) Read(ctx context.Context, req *ReadRequest) (*ReadResponse, error) {
	return nil, fmt.Errorf("Read method not implemented for object type: %s", h.objectType)
}

func (h *AdvancedHandler) Update(ctx context.Context, req *UpdateRequest) (*UpdateResponse, error) {
	return nil, fmt.Errorf("Update method not implemented for object type: %s", h.objectType)
}

func (h *AdvancedHandler) Delete(ctx context.Context, req *DeleteRequest) (*DeleteResponse, error) {
	return nil, fmt.Errorf("Delete method not implemented for object type: %s", h.objectType)
}

func (h *AdvancedHandler) Plan(ctx context.Context, req *PlanRequest) (*PlanResponse, error) {
	// Use registered planners
	for _, planner := range h.planners {
		coreResp, err := planner.Plan(ctx, &core.PlanRequest{
			ObjectType:    req.ObjectType,
			Name:          req.Name,
			DesiredConfig: req.DesiredConfig,
			CurrentState:  req.CurrentState,
		})
		if err != nil {
			return nil, err
		}

		// Convert core.PlanResponse to create.PlanResponse
		changes := make([]PlannedChange, len(coreResp.Changes))
		for i, change := range coreResp.Changes {
			changes[i] = PlannedChange{
				Action:          change.Action,
				Property:        change.Property,
				OldValue:        change.OldValue,
				NewValue:        change.NewValue,
				RequiresReplace: change.RequiresReplace,
				RiskLevel:       change.RiskLevel,
				Description:     change.Description,
			}
		}

		return &PlanResponse{
			Summary: coreResp.Summary,
			Changes: changes,
		}, nil
	}

	// Default implementation
	return &PlanResponse{
		Summary: &core.PlanSummary{
			RequiresReplace: false,
			RiskLevel:       "low",
			TotalChanges:    0,
		},
		Changes: []PlannedChange{},
	}, nil
}

// =============================================================================
// COMPONENT INTERFACES AND IMPLEMENTATIONS
// =============================================================================

// Validator validates object configuration
type Validator interface {
	Validate(config map[string]interface{}) error
	Name() string
}

// Planner generates execution plans
type Planner interface {
	Plan(ctx context.Context, req *core.PlanRequest) (*core.PlanResponse, error)
	Name() string
}

// Importer imports existing infrastructure
type Importer interface {
	Import(ctx context.Context, req *core.ImportRequest) (*core.ImportResponse, error)
	CanImport(ctx context.Context, resourceID string) (bool, error)
	Name() string
}

// DriftDetector detects configuration drift
type DriftDetector interface {
	DetectDrift(ctx context.Context, req *core.DriftRequest) (*core.DriftResponse, error)
	Name() string
}

// =============================================================================
// BASIC HANDLER IMPLEMENTATION
// =============================================================================

// BasicHandler provides a simple handler implementation
type BasicHandler struct {
	objectType string
	schema     *core.ObjectType
}

// NewHandler creates a new BasicHandler for the specified object type
func NewHandler(objectType string) ObjectHandler {
	return &BasicHandler{
		objectType: objectType,
		schema: &core.ObjectType{
			Name:       objectType,
			Type:       core.CREATE,
			Properties: make(map[string]*core.Property),
			Required:   make([]string, 0),
			Optional:   make([]string, 0),
		},
	}
}

func (h *BasicHandler) Schema() *core.ObjectType {
	return h.schema
}

func (h *BasicHandler) Create(ctx context.Context, req *CreateRequest) (*CreateResponse, error) {
	return nil, fmt.Errorf("Create method not implemented for object type: %s", h.objectType)
}

func (h *BasicHandler) Read(ctx context.Context, req *ReadRequest) (*ReadResponse, error) {
	return nil, fmt.Errorf("Read method not implemented for object type: %s", h.objectType)
}

func (h *BasicHandler) Update(ctx context.Context, req *UpdateRequest) (*UpdateResponse, error) {
	return nil, fmt.Errorf("Update method not implemented for object type: %s", h.objectType)
}

func (h *BasicHandler) Delete(ctx context.Context, req *DeleteRequest) (*DeleteResponse, error) {
	return nil, fmt.Errorf("Delete method not implemented for object type: %s", h.objectType)
}

func (h *BasicHandler) Plan(ctx context.Context, req *PlanRequest) (*PlanResponse, error) {
	return &PlanResponse{
		Summary: &core.PlanSummary{
			RequiresReplace: false,
			RiskLevel:       "low",
			TotalChanges:    0,
		},
		Changes: []PlannedChange{},
	}, nil
}

// =============================================================================
// BUILT-IN VALIDATORS
// =============================================================================

// RequiredValidator validates that required fields are present
type RequiredValidator struct {
	fields []string
}

// NewRequiredValidator creates a validator that checks required fields
func NewRequiredValidator(fields ...string) Validator {
	return &RequiredValidator{fields: fields}
}

func (v *RequiredValidator) Validate(config map[string]interface{}) error {
	for _, field := range v.fields {
		if _, exists := config[field]; !exists {
			return fmt.Errorf("required field '%s' is missing", field)
		}
	}
	return nil
}

func (v *RequiredValidator) Name() string {
	return "required_validator"
}
