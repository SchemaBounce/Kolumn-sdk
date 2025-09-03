// Package rpc provides request/response types for the Kolumn Provider SDK RPC interface
package rpc

import (
	"time"
)

// =============================================================================
// RESOURCE MANAGEMENT REQUEST/RESPONSE TYPES
// =============================================================================

// CreateResourceRequest represents a request to create a new resource
type CreateResourceRequest struct {
	ResourceType string                 `json:"resource_type"`
	Name         string                 `json:"name"`
	Config       map[string]interface{} `json:"config"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Options      RequestOptions         `json:"options,omitempty"`
}

// CreateResourceResponse represents a response to a resource creation request
type CreateResourceResponse struct {
	Success    bool                   `json:"success"`
	ResourceID string                 `json:"resource_id"`
	State      map[string]interface{} `json:"state,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Warnings   []string               `json:"warnings,omitempty"`
	Error      *RPCError              `json:"error,omitempty"`
}

// ReadResourceRequest represents a request to read a resource
type ReadResourceRequest struct {
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id"`
	Name         string         `json:"name"`
	Options      RequestOptions `json:"options,omitempty"`
}

// ReadResourceResponse represents a response to a resource read request
type ReadResourceResponse struct {
	Success  bool                   `json:"success"`
	State    map[string]interface{} `json:"state,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	NotFound bool                   `json:"not_found"`
	Error    *RPCError              `json:"error,omitempty"`
}

// UpdateResourceRequest represents a request to update a resource
type UpdateResourceRequest struct {
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	Name         string                 `json:"name"`
	Config       map[string]interface{} `json:"config"`
	CurrentState map[string]interface{} `json:"current_state,omitempty"`
	Options      RequestOptions         `json:"options,omitempty"`
}

// UpdateResourceResponse represents a response to a resource update request
type UpdateResourceResponse struct {
	Success  bool                   `json:"success"`
	NewState map[string]interface{} `json:"new_state,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Warnings []string               `json:"warnings,omitempty"`
	Error    *RPCError              `json:"error,omitempty"`
}

// DeleteResourceRequest represents a request to delete a resource
type DeleteResourceRequest struct {
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	Name         string                 `json:"name"`
	State        map[string]interface{} `json:"state,omitempty"`
	Options      RequestOptions         `json:"options,omitempty"`
}

// DeleteResourceResponse represents a response to a resource deletion request
type DeleteResourceResponse struct {
	Success  bool      `json:"success"`
	Warnings []string  `json:"warnings,omitempty"`
	Error    *RPCError `json:"error,omitempty"`
}

// =============================================================================
// IMPORT/EXPORT REQUEST/RESPONSE TYPES
// =============================================================================

// ImportRequest represents a request to import an existing resource
type ImportRequest struct {
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	ResourceName string                 `json:"resource_name"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Options      ImportOptions          `json:"options,omitempty"`
}

// ImportResponse represents a response to an import request
type ImportResponse struct {
	Success      bool          `json:"success"`
	ImportResult *ImportResult `json:"import_result,omitempty"`
	Error        *RPCError     `json:"error,omitempty"`
}

// ImportResult contains the result of importing a resource
type ImportResult struct {
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	ResourceName string                 `json:"resource_name"`
	State        map[string]interface{} `json:"state"`
	Config       map[string]interface{} `json:"config"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
}

// ImportOptions contains options for import operations
type ImportOptions struct {
	ProviderOptions map[string]interface{} `json:"provider_options,omitempty"`
	Overwrite       bool                   `json:"overwrite"`
	SkipValidation  bool                   `json:"skip_validation"`
}

// =============================================================================
// PLANNING REQUEST/RESPONSE TYPES
// =============================================================================

// PlanRequest represents a request to create an execution plan
type PlanRequest struct {
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	DesiredState map[string]interface{} `json:"desired_state"`
	CurrentState map[string]interface{} `json:"current_state,omitempty"`
	Options      PlanOptions            `json:"options,omitempty"`
}

// PlanResponse represents a response to a planning request
type PlanResponse struct {
	Success  bool           `json:"success"`
	Plan     *ExecutionPlan `json:"plan,omitempty"`
	Warnings []string       `json:"warnings,omitempty"`
	Error    *RPCError      `json:"error,omitempty"`
}

// ExecutionPlan contains the details of a planned execution
type ExecutionPlan struct {
	Changes       []PlannedChange `json:"changes"`
	RiskLevel     RiskLevel       `json:"risk_level"`
	EstimatedTime time.Duration   `json:"estimated_time,omitempty"`
	Summary       string          `json:"summary,omitempty"`
}

// PlannedChange represents a planned change to a resource
type PlannedChange struct {
	Action            string        `json:"action"` // create, update, delete, destroy
	ResourceType      string        `json:"resource_type"`
	ResourceID        string        `json:"resource_id,omitempty"`
	Field             string        `json:"field,omitempty"`
	OldValue          interface{}   `json:"old_value,omitempty"`
	NewValue          interface{}   `json:"new_value,omitempty"`
	RequiresDestroy   bool          `json:"requires_destroy"`
	RiskLevel         RiskLevel     `json:"risk_level"`
	EstimatedDuration time.Duration `json:"estimated_duration,omitempty"`
	Description       string        `json:"description"`
}

// PlanOptions contains options for planning operations
type PlanOptions struct {
	ProviderOptions map[string]interface{} `json:"provider_options,omitempty"`
	DetailLevel     string                 `json:"detail_level"` // summary, detailed, verbose
	IncludeSecrets  bool                   `json:"include_secrets"`
}

// RiskLevel represents the risk level of an operation
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// =============================================================================
// VALIDATION REQUEST/RESPONSE TYPES
// =============================================================================

// ValidatePlanRequest represents a request to validate a plan or configuration
type ValidatePlanRequest struct {
	ResourceType   string                 `json:"resource_type"`
	Config         map[string]interface{} `json:"config,omitempty"`
	Plan           *ExecutionPlan         `json:"plan,omitempty"`
	ValidationMode string                 `json:"validation_mode"` // config, plan, full
	Options        ValidationOptions      `json:"options,omitempty"`
}

// ValidatePlanResponse represents a response to a validation request
type ValidatePlanResponse struct {
	Success          bool              `json:"success"`
	Valid            bool              `json:"valid"`
	ValidationResult *ValidationResult `json:"validation_result,omitempty"`
	Error            *RPCError         `json:"error,omitempty"`
}

// ValidationResult contains the results of validation
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationError `json:"warnings,omitempty"`
	Info     []ValidationError `json:"info,omitempty"`
}

// ValidationError represents a validation error or warning
type ValidationError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Field      string                 `json:"field,omitempty"`
	Severity   string                 `json:"severity"` // error, warning, info
	Suggestion string                 `json:"suggestion,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// ValidationOptions contains options for validation operations
type ValidationOptions struct {
	ProviderOptions   map[string]interface{} `json:"provider_options,omitempty"`
	StrictMode        bool                   `json:"strict_mode"`
	CheckDependencies bool                   `json:"check_dependencies"`
}

// =============================================================================
// HEALTH CHECK REQUEST/RESPONSE TYPES
// =============================================================================

// PingRequest represents a health check request
type PingRequest struct {
	IncludeDetails bool                   `json:"include_details"`
	CheckServices  []string               `json:"check_services,omitempty"`
	Options        map[string]interface{} `json:"options,omitempty"`
}

// PingResponse represents a health check response
type PingResponse struct {
	Success   bool                   `json:"success"`
	Healthy   bool                   `json:"healthy"`
	Details   string                 `json:"details,omitempty"`
	Latency   time.Duration          `json:"latency,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Error     *RPCError              `json:"error,omitempty"`
}

// =============================================================================
// ERROR TYPES
// =============================================================================

// RPCError represents an error in RPC communication
type RPCError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// Error implements the error interface
func (e *RPCError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}

// =============================================================================
// COMMON OPTIONS TYPES
// =============================================================================

// RequestOptions contains common options for requests
type RequestOptions struct {
	ProviderOptions map[string]interface{} `json:"provider_options,omitempty"`
	Timeout         time.Duration          `json:"timeout,omitempty"`
	RetryPolicy     *RetryPolicy           `json:"retry_policy,omitempty"`
	DryRun          bool                   `json:"dry_run"`
	Async           bool                   `json:"async"`
}

// RetryPolicy defines retry behavior for operations
type RetryPolicy struct {
	MaxRetries    int           `json:"max_retries"`
	InitialDelay  time.Duration `json:"initial_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
	MaxDelay      time.Duration `json:"max_delay"`
}

// =============================================================================
// KOLUMN-COMPATIBLE REQUEST/RESPONSE TYPES
// =============================================================================

// ValidateProviderConfigRequest validates provider configuration
type ValidateProviderConfigRequest struct {
	Config map[string]interface{} `json:"config"`
}

// ValidateProviderConfigResponse returns provider configuration validation results
type ValidateProviderConfigResponse struct {
	Success     bool         `json:"success"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
	Error       *RPCError    `json:"error,omitempty"`
}

// ValidateResourceConfigRequest validates resource configuration
type ValidateResourceConfigRequest struct {
	ResourceType string                 `json:"resource_type"`
	Config       map[string]interface{} `json:"config"`
}

// ValidateResourceConfigResponse returns resource configuration validation results
type ValidateResourceConfigResponse struct {
	Success     bool         `json:"success"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
	Error       *RPCError    `json:"error,omitempty"`
}

// PlanResourceChangeRequest requests planning for resource changes
type PlanResourceChangeRequest struct {
	ResourceType     string                 `json:"resource_type"`
	PriorState       map[string]interface{} `json:"prior_state,omitempty"`
	ProposedNewState map[string]interface{} `json:"proposed_new_state"`
	Config           map[string]interface{} `json:"config"`
	PriorPrivate     []byte                 `json:"prior_private,omitempty"`
}

// PlanResourceChangeResponse returns planned changes for a resource
type PlanResourceChangeResponse struct {
	Success          bool                   `json:"success"`
	PlannedState     map[string]interface{} `json:"planned_state"`
	RequiresReplace  []string               `json:"requires_replace,omitempty"`
	PlannedPrivate   []byte                 `json:"planned_private,omitempty"`
	Diagnostics      []Diagnostic           `json:"diagnostics,omitempty"`
	LegacyTypeSystem bool                   `json:"legacy_type_system"`
	Error            *RPCError              `json:"error,omitempty"`
}

// ApplyResourceChangeRequest requests applying planned changes
type ApplyResourceChangeRequest struct {
	ResourceType   string                 `json:"resource_type"`
	PriorState     map[string]interface{} `json:"prior_state,omitempty"`
	PlannedState   map[string]interface{} `json:"planned_state"`
	Config         map[string]interface{} `json:"config"`
	PlannedPrivate []byte                 `json:"planned_private,omitempty"`
}

// ApplyResourceChangeResponse returns the result of applying changes
type ApplyResourceChangeResponse struct {
	Success          bool                   `json:"success"`
	NewState         map[string]interface{} `json:"new_state"`
	Private          []byte                 `json:"private,omitempty"`
	Diagnostics      []Diagnostic           `json:"diagnostics,omitempty"`
	LegacyTypeSystem bool                   `json:"legacy_type_system"`
	Error            *RPCError              `json:"error,omitempty"`
}

// KolumnReadResourceRequest requests reading resource state (Kolumn-compatible)
type KolumnReadResourceRequest struct {
	ResourceType string                 `json:"resource_type"`
	CurrentState map[string]interface{} `json:"current_state"`
	Private      []byte                 `json:"private,omitempty"`
}

// KolumnReadResourceResponse returns current resource state (Kolumn-compatible)
type KolumnReadResourceResponse struct {
	Success     bool                   `json:"success"`
	NewState    map[string]interface{} `json:"new_state"`
	Private     []byte                 `json:"private,omitempty"`
	Diagnostics []Diagnostic           `json:"diagnostics,omitempty"`
	Error       *RPCError              `json:"error,omitempty"`
}

// ImportResourceStateRequest requests importing existing resource
type ImportResourceStateRequest struct {
	ResourceType string `json:"resource_type"`
	ID           string `json:"id"`
}

// ImportResourceStateResponse returns imported resource state
type ImportResourceStateResponse struct {
	Success           bool               `json:"success"`
	ImportedResources []ImportedResource `json:"imported_resources"`
	Diagnostics       []Diagnostic       `json:"diagnostics,omitempty"`
	Error             *RPCError          `json:"error,omitempty"`
}

// ImportedResource represents an imported resource
type ImportedResource struct {
	ResourceType string                 `json:"resource_type"`
	State        map[string]interface{} `json:"state"`
	Private      []byte                 `json:"private,omitempty"`
}

// UpgradeResourceStateRequest requests upgrading resource state schema
type UpgradeResourceStateRequest struct {
	ResourceType string                 `json:"resource_type"`
	Version      int64                  `json:"version"`
	RawState     map[string]interface{} `json:"raw_state"`
}

// UpgradeResourceStateResponse returns upgraded resource state
type UpgradeResourceStateResponse struct {
	Success       bool                   `json:"success"`
	UpgradedState map[string]interface{} `json:"upgraded_state"`
	Diagnostics   []Diagnostic           `json:"diagnostics,omitempty"`
	Error         *RPCError              `json:"error,omitempty"`
}

// Diagnostic represents a diagnostic message (error, warning, or info)
type Diagnostic struct {
	Severity  string                 `json:"severity"` // "error", "warning", "info"
	Summary   string                 `json:"summary"`
	Detail    string                 `json:"detail,omitempty"`
	Attribute map[string]interface{} `json:"attribute,omitempty"`
}
