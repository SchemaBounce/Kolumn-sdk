// Package pdk provides CRUD helper abstractions for Kolumn Provider SDK
// This eliminates the need for provider developers to manually handle JSON marshaling
// and function dispatch for standard CRUD operations
package pdk

import (
	"context"
	"time"
)

// =============================================================================
// CRUD RESOURCE HANDLER INTERFACE
// =============================================================================

// ResourceHandler defines the interface that provider developers implement
// for each resource type. The SDK handles all the RPC plumbing.
// This interface supports both the original simplified operations and Kolumn-compatible operations.
type ResourceHandler interface {
	// =====================================================
	// ORIGINAL SIMPLIFIED CRUD OPERATIONS
	// =====================================================

	// Core CRUD operations
	Create(ctx context.Context, req *CreateRequest) (*CreateResponse, error)
	Read(ctx context.Context, req *ReadRequest) (*ReadResponse, error)
	Update(ctx context.Context, req *UpdateRequest) (*UpdateResponse, error)
	Delete(ctx context.Context, req *DeleteRequest) (*DeleteResponse, error)

	// Enhanced destroy operation (with safety checks, rollback, etc.)
	Destroy(ctx context.Context, req *DestroyRequest) (*DestroyResponse, error)

	// Optional enhanced operations
	Import(ctx context.Context, req *ImportRequest) (*ImportResponse, error)
	Plan(ctx context.Context, req *PlanRequest) (*PlanResponse, error)
	Validate(ctx context.Context, req *ValidateRequest) (*ValidateResponse, error)
}

// KolumnCompatibleResourceHandler extends ResourceHandler with Kolumn-compatible methods
// Providers can optionally implement this interface for enhanced Kolumn compatibility
type KolumnCompatibleResourceHandler interface {
	ResourceHandler // Embed the original interface

	// =====================================================
	// KOLUMN-COMPATIBLE OPERATIONS (OPTIONAL)
	// =====================================================

	// ValidateConfig validates resource configuration (Kolumn-style)
	ValidateConfig(ctx context.Context, req *KolumnValidateConfigRequest) (*KolumnValidateConfigResponse, error)

	// PlanChange compares desired vs current state and returns planned changes
	PlanChange(ctx context.Context, req *KolumnPlanChangeRequest) (*KolumnPlanChangeResponse, error)

	// ApplyChange executes planned changes and returns new state
	ApplyChange(ctx context.Context, req *KolumnApplyChangeRequest) (*KolumnApplyChangeResponse, error)

	// RefreshState refreshes resource state from the actual system
	RefreshState(ctx context.Context, req *KolumnRefreshStateRequest) (*KolumnRefreshStateResponse, error)

	// ImportState converts existing resources into managed state
	ImportState(ctx context.Context, req *KolumnImportStateRequest) (*KolumnImportStateResponse, error)

	// UpgradeState handles schema version migrations
	UpgradeState(ctx context.Context, req *KolumnUpgradeStateRequest) (*KolumnUpgradeStateResponse, error)
}

// OptionalResourceHandler defines optional operations that providers can implement
type OptionalResourceHandler interface {
	// State management
	GetState(ctx context.Context, req *GetStateRequest) (*GetStateResponse, error)
	SetState(ctx context.Context, req *SetStateRequest) (*SetStateResponse, error)
	DetectDrift(ctx context.Context, req *DetectDriftRequest) (*DetectDriftResponse, error)

	// Discovery and introspection
	Discover(ctx context.Context, req *DiscoverRequest) (*DiscoverResponse, error)
	Introspect(ctx context.Context, req *IntrospectRequest) (*IntrospectResponse, error)

	// Performance and monitoring
	GetMetrics(ctx context.Context, req *GetMetricsRequest) (*GetMetricsResponse, error)

	// Maintenance operations
	Backup(ctx context.Context, req *BackupRequest) (*BackupResponse, error)
	Restore(ctx context.Context, req *RestoreRequest) (*RestoreResponse, error)
}

// =============================================================================
// SIMPLIFIED REQUEST/RESPONSE TYPES FOR SDK
// =============================================================================

// CreateRequest represents a simplified resource creation request
type CreateRequest struct {
	ResourceType string                 `json:"resource_type"`
	Name         string                 `json:"name"`
	Config       map[string]interface{} `json:"config"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

// CreateResponse represents a simplified resource creation response
type CreateResponse struct {
	ResourceID string                 `json:"resource_id"`
	State      map[string]interface{} `json:"state,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Warnings   []string               `json:"warnings,omitempty"`
}

// ReadRequest represents a simplified resource read request
type ReadRequest struct {
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	Name         string                 `json:"name"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

// ReadResponse represents a simplified resource read response
type ReadResponse struct {
	State    map[string]interface{} `json:"state,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	NotFound bool                   `json:"not_found"`
	Warnings []string               `json:"warnings,omitempty"`
}

// UpdateRequest represents a simplified resource update request
type UpdateRequest struct {
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	Name         string                 `json:"name"`
	Config       map[string]interface{} `json:"config"`
	CurrentState map[string]interface{} `json:"current_state,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

// UpdateResponse represents a simplified resource update response
type UpdateResponse struct {
	NewState map[string]interface{} `json:"new_state,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Warnings []string               `json:"warnings,omitempty"`
}

// DeleteRequest represents a simplified resource deletion request
type DeleteRequest struct {
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	Name         string                 `json:"name"`
	State        map[string]interface{} `json:"state,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

// DeleteResponse represents a simplified resource deletion response
type DeleteResponse struct {
	Warnings []string `json:"warnings,omitempty"`
}

// DestroyRequest represents an enhanced destroy request with safety features
type DestroyRequest struct {
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

// DestroyResponse represents an enhanced destroy response with safety information
type DestroyResponse struct {
	Success        bool                 `json:"success"`
	BackupID       string               `json:"backup_id,omitempty"`
	RollbackPlan   *RollbackPlan        `json:"rollback_plan,omitempty"`
	SafetyChecks   []SafetyCheck        `json:"safety_checks,omitempty"`
	Dependencies   []ResourceDependency `json:"dependencies,omitempty"`
	EstimatedTime  time.Duration        `json:"estimated_time,omitempty"`
	RiskAssessment *RiskAssessment      `json:"risk_assessment,omitempty"`
	Warnings       []string             `json:"warnings,omitempty"`
}

// Additional simplified request/response types for optional operations

// ImportRequest represents a simplified import request
type ImportRequest struct {
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	Name         string                 `json:"name,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

// ImportResponse represents a simplified import response
type ImportResponse struct {
	State        map[string]interface{} `json:"state"`
	Config       map[string]interface{} `json:"config"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
}

// PlanRequest represents a simplified plan request
type PlanRequest struct {
	ResourceType  string                 `json:"resource_type"`
	Name          string                 `json:"name"`
	DesiredConfig map[string]interface{} `json:"desired_config"`
	CurrentState  map[string]interface{} `json:"current_state,omitempty"`
	Options       map[string]interface{} `json:"options,omitempty"`
}

// PlanResponse represents a simplified plan response
type PlanResponse struct {
	Changes         []PlannedChange `json:"changes"`
	RequiresDestroy bool            `json:"requires_destroy"`
	RiskLevel       string          `json:"risk_level"` // low, medium, high, critical
	EstimatedTime   time.Duration   `json:"estimated_time,omitempty"`
	Warnings        []string        `json:"warnings,omitempty"`
}

// ValidateRequest represents a simplified validation request
type ValidateRequest struct {
	ResourceType string                 `json:"resource_type"`
	Config       map[string]interface{} `json:"config"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

// ValidateResponse represents a simplified validation response
type ValidateResponse struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationError `json:"warnings,omitempty"`
}

// =============================================================================
// SUPPORTING TYPES FOR ENHANCED OPERATIONS
// =============================================================================

// PlannedChange represents a planned change to a resource
type PlannedChange struct {
	Action          string        `json:"action"` // create, update, delete, destroy
	Field           string        `json:"field,omitempty"`
	OldValue        interface{}   `json:"old_value,omitempty"`
	NewValue        interface{}   `json:"new_value,omitempty"`
	RequiresDestroy bool          `json:"requires_destroy"`
	RiskLevel       string        `json:"risk_level"`
	Description     string        `json:"description"`
	EstimatedTime   time.Duration `json:"estimated_time,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Field      string                 `json:"field,omitempty"`
	Severity   string                 `json:"severity"` // error, warning, info
	Suggestion string                 `json:"suggestion,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// RollbackPlan contains information for rolling back a destroy operation
type RollbackPlan struct {
	ID            string              `json:"id"`
	Operations    []RollbackOperation `json:"operations"`
	EstimatedTime time.Duration       `json:"estimated_time"`
	ValidUntil    time.Time           `json:"valid_until"`
}

// RollbackOperation represents a single rollback operation
type RollbackOperation struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Command     string                 `json:"command,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Order       int                    `json:"order"`
}

// SafetyCheck represents a safety validation performed before destroy
type SafetyCheck struct {
	Name     string                 `json:"name"`
	Passed   bool                   `json:"passed"`
	Message  string                 `json:"message"`
	Severity string                 `json:"severity"` // info, warning, error, critical
	Details  map[string]interface{} `json:"details,omitempty"`
}

// ResourceDependency represents a dependency relationship
type ResourceDependency struct {
	Type         string `json:"type"` // hard, soft, circular
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	ResourceName string `json:"resource_name"`
	Relationship string `json:"relationship"` // depends_on, references, blocks
	Impact       string `json:"impact"`       // none, warning, blocking
}

// RiskAssessment provides risk analysis for destroy operations
type RiskAssessment struct {
	OverallRisk     string        `json:"overall_risk"` // low, medium, high, critical
	RiskFactors     []RiskFactor  `json:"risk_factors"`
	DataLossRisk    bool          `json:"data_loss_risk"`
	DowntimeRisk    bool          `json:"downtime_risk"`
	RecoveryTime    time.Duration `json:"recovery_time,omitempty"`
	Recommendations []string      `json:"recommendations,omitempty"`
}

// RiskFactor represents a specific risk factor
type RiskFactor struct {
	Type        string `json:"type"`     // data_loss, downtime, dependency, security
	Severity    string `json:"severity"` // low, medium, high, critical
	Description string `json:"description"`
	Mitigation  string `json:"mitigation,omitempty"`
}

// Additional simplified request/response types for optional operations

// GetStateRequest represents a state retrieval request
type GetStateRequest struct {
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
}

// GetStateResponse represents a state retrieval response
type GetStateResponse struct {
	State    map[string]interface{} `json:"state"`
	Version  int64                  `json:"version,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SetStateRequest represents a state setting request
type SetStateRequest struct {
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	State        map[string]interface{} `json:"state"`
	Version      int64                  `json:"version,omitempty"`
}

// SetStateResponse represents a state setting response
type SetStateResponse struct {
	NewVersion int64 `json:"new_version,omitempty"`
}

// DetectDriftRequest represents a drift detection request
type DetectDriftRequest struct {
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	ManagedState map[string]interface{} `json:"managed_state"`
}

// DetectDriftResponse represents a drift detection response
type DetectDriftResponse struct {
	HasDrift    bool                   `json:"has_drift"`
	ActualState map[string]interface{} `json:"actual_state,omitempty"`
	Differences []StateDifference      `json:"differences,omitempty"`
}

// StateDifference represents a difference in state
type StateDifference struct {
	Field        string      `json:"field"`
	ManagedValue interface{} `json:"managed_value"`
	ActualValue  interface{} `json:"actual_value"`
	DriftType    string      `json:"drift_type"` // added, removed, changed
	Impact       string      `json:"impact"`     // low, medium, high
}

// DiscoverRequest represents a resource discovery request
type DiscoverRequest struct {
	ResourceTypes []string               `json:"resource_types,omitempty"`
	Filters       map[string]interface{} `json:"filters,omitempty"`
	MaxResults    int                    `json:"max_results,omitempty"`
}

// DiscoverResponse represents a resource discovery response
type DiscoverResponse struct {
	Resources []DiscoveredResource `json:"resources"`
	HasMore   bool                 `json:"has_more"`
	NextToken string               `json:"next_token,omitempty"`
}

// DiscoveredResource represents a discovered resource
type DiscoveredResource struct {
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	Name         string                 `json:"name"`
	State        map[string]interface{} `json:"state,omitempty"`
	Managed      bool                   `json:"managed"`
	Importable   bool                   `json:"importable"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// IntrospectRequest represents a resource introspection request
type IntrospectRequest struct {
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
}

// IntrospectResponse represents a resource introspection response
type IntrospectResponse struct {
	Schema       map[string]interface{} `json:"schema,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
	Capabilities []string               `json:"capabilities,omitempty"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
}

// GetMetricsRequest represents a metrics request
type GetMetricsRequest struct {
	ResourceType string   `json:"resource_type"`
	ResourceID   string   `json:"resource_id,omitempty"`
	MetricNames  []string `json:"metric_names,omitempty"`
}

// GetMetricsResponse represents a metrics response
type GetMetricsResponse struct {
	Metrics map[string]interface{} `json:"metrics"`
}

// BackupRequest represents a backup request
type BackupRequest struct {
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	BackupType   string                 `json:"backup_type"` // full, incremental
	Options      map[string]interface{} `json:"options,omitempty"`
}

// BackupResponse represents a backup response
type BackupResponse struct {
	BackupID string        `json:"backup_id"`
	Size     int64         `json:"size,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`
}

// RestoreRequest represents a restore request
type RestoreRequest struct {
	BackupID     string                 `json:"backup_id"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

// RestoreResponse represents a restore response
type RestoreResponse struct {
	Success  bool          `json:"success"`
	Duration time.Duration `json:"duration,omitempty"`
}

// =============================================================================
// KOLUMN-COMPATIBLE REQUEST/RESPONSE TYPES FOR SDK
// =============================================================================

// KolumnValidateConfigRequest represents Kolumn-style validation request
type KolumnValidateConfigRequest struct {
	ResourceType string                 `json:"resource_type"`
	Config       map[string]interface{} `json:"config"`
}

// KolumnValidateConfigResponse represents Kolumn-style validation response
type KolumnValidateConfigResponse struct {
	Valid       bool                `json:"valid"`
	Diagnostics []KolumnDiagnostic `json:"diagnostics,omitempty"`
}

// KolumnPlanChangeRequest represents Kolumn-style plan request
type KolumnPlanChangeRequest struct {
	ResourceType  string                 `json:"resource_type"`
	PriorState    map[string]interface{} `json:"prior_state,omitempty"`
	ProposedState map[string]interface{} `json:"proposed_state"`
	Config        map[string]interface{} `json:"config"`
	PriorPrivate  []byte                 `json:"prior_private,omitempty"`
}

// KolumnPlanChangeResponse represents Kolumn-style plan response
type KolumnPlanChangeResponse struct {
	PlannedState    map[string]interface{} `json:"planned_state"`
	RequiresReplace []string               `json:"requires_replace,omitempty"`
	PlannedPrivate  []byte                 `json:"planned_private,omitempty"`
	Diagnostics     []KolumnDiagnostic     `json:"diagnostics,omitempty"`
}

// KolumnApplyChangeRequest represents Kolumn-style apply request
type KolumnApplyChangeRequest struct {
	ResourceType   string                 `json:"resource_type"`
	PriorState     map[string]interface{} `json:"prior_state,omitempty"`
	PlannedState   map[string]interface{} `json:"planned_state"`
	Config         map[string]interface{} `json:"config"`
	PlannedPrivate []byte                 `json:"planned_private,omitempty"`
}

// KolumnApplyChangeResponse represents Kolumn-style apply response
type KolumnApplyChangeResponse struct {
	NewState    map[string]interface{} `json:"new_state"`
	Private     []byte                 `json:"private,omitempty"`
	Diagnostics []KolumnDiagnostic     `json:"diagnostics,omitempty"`
}

// KolumnRefreshStateRequest represents Kolumn-style refresh request
type KolumnRefreshStateRequest struct {
	ResourceType string                 `json:"resource_type"`
	CurrentState map[string]interface{} `json:"current_state"`
	Private      []byte                 `json:"private,omitempty"`
}

// KolumnRefreshStateResponse represents Kolumn-style refresh response
type KolumnRefreshStateResponse struct {
	NewState    map[string]interface{} `json:"new_state"`
	Private     []byte                 `json:"private,omitempty"`
	Diagnostics []KolumnDiagnostic     `json:"diagnostics,omitempty"`
}

// KolumnImportStateRequest represents Kolumn-style import request
type KolumnImportStateRequest struct {
	ResourceType string `json:"resource_type"`
	ID           string `json:"id"`
}

// KolumnImportStateResponse represents Kolumn-style import response
type KolumnImportStateResponse struct {
	ImportedResources []KolumnImportedResource `json:"imported_resources"`
	Diagnostics       []KolumnDiagnostic       `json:"diagnostics,omitempty"`
}

// KolumnImportedResource represents an imported resource
type KolumnImportedResource struct {
	ResourceType string                 `json:"resource_type"`
	State        map[string]interface{} `json:"state"`
	Private      []byte                 `json:"private,omitempty"`
}

// KolumnUpgradeStateRequest represents Kolumn-style upgrade request
type KolumnUpgradeStateRequest struct {
	ResourceType string                 `json:"resource_type"`
	Version      int64                  `json:"version"`
	RawState     map[string]interface{} `json:"raw_state"`
}

// KolumnUpgradeStateResponse represents Kolumn-style upgrade response
type KolumnUpgradeStateResponse struct {
	UpgradedState map[string]interface{} `json:"upgraded_state"`
	Diagnostics   []KolumnDiagnostic     `json:"diagnostics,omitempty"`
}

// KolumnDiagnostic represents a Kolumn-style diagnostic message
type KolumnDiagnostic struct {
	Severity  string                 `json:"severity"` // "error", "warning", "info"
	Summary   string                 `json:"summary"`
	Detail    string                 `json:"detail,omitempty"`
	Attribute map[string]interface{} `json:"attribute,omitempty"`
}
