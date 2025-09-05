// Package core provides context and request/response types for Kolumn Provider SDK
package core

import (
	"fmt"
	"time"
)

// =============================================================================
// VALIDATION TYPES
// =============================================================================

// ValidationError represents a validation error or warning
type ValidationError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Field      string                 `json:"field,omitempty"`
	Severity   string                 `json:"severity"` // error, warning, info
	Suggestion string                 `json:"suggestion,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// =============================================================================
// SCHEMA TYPES
// =============================================================================

// ObjectSchema defines the schema for a managed object type
type ObjectSchema struct {
	Type        string                   `json:"type"`
	Category    string                   `json:"category"`
	Description string                   `json:"description"`
	Properties  map[string]*PropertySpec `json:"properties,omitempty"`
	Examples    []*ObjectExample         `json:"examples,omitempty"`
	Tags        []string                 `json:"tags,omitempty"`
}

// PropertySpec defines a property within an object schema
type PropertySpec struct {
	Type        string        `json:"type"`
	Description string        `json:"description"`
	Required    bool          `json:"required"`
	Pattern     *string       `json:"pattern,omitempty"`
	Example     interface{}   `json:"example,omitempty"`
	Default     interface{}   `json:"default,omitempty"`
	Items       *PropertySpec `json:"items,omitempty"` // for list/array types
}

// ObjectExample provides an example of using the object
type ObjectExample struct {
	Name        string                 `json:"name"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Category    string                 `json:"category,omitempty"`
	UseCase     string                 `json:"use_case,omitempty"`
	HCL         string                 `json:"hcl"`
	JSON        string                 `json:"json,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// =============================================================================
// DRIFT DETECTION REQUEST/RESPONSE TYPES
// =============================================================================

// DriftRequest represents a request to detect configuration drift
type DriftRequest struct {
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	ManagedState map[string]interface{} `json:"managed_state"`
	Options      *DriftOptions          `json:"options,omitempty"`
}

// DriftOptions provides configuration for drift detection
type DriftOptions struct {
	IgnoreFields   []string `json:"ignore_fields,omitempty"`
	DeepInspection bool     `json:"deep_inspection"`
	IncludeMetrics bool     `json:"include_metrics"`
}

// DriftResponse represents the result of drift detection
type DriftResponse struct {
	HasDrift        bool                   `json:"has_drift"`
	Changes         []DriftChange          `json:"changes,omitempty"`
	ActualState     map[string]interface{} `json:"actual_state,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	Recommendations []string               `json:"recommendations,omitempty"`
}

// DriftChange represents a detected configuration drift
type DriftChange struct {
	Field         string      `json:"field"`
	ExpectedValue interface{} `json:"expected_value"`
	ActualValue   interface{} `json:"actual_value"`
	ChangeType    string      `json:"change_type"` // added, removed, modified
	Severity      string      `json:"severity"`    // low, medium, high, critical
}

// =============================================================================
// DISCOVER REQUEST/RESPONSE TYPES
// =============================================================================

// RelationsRequest represents a request to analyze resource relationships
type RelationsRequest struct {
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	State        map[string]interface{} `json:"state,omitempty"`
	Options      *RelationsOptions      `json:"options,omitempty"`
}

// RelationsOptions provides configuration for relation analysis
type RelationsOptions struct {
	MaxDepth        int      `json:"max_depth,omitempty"`
	IncludeInbound  bool     `json:"include_inbound"`
	IncludeOutbound bool     `json:"include_outbound"`
	RelationTypes   []string `json:"relation_types,omitempty"`
}

// MetadataRequest represents a request for additional resource metadata
type MetadataRequest struct {
	ResourceID   string           `json:"resource_id"`
	ResourceType string           `json:"resource_type"`
	Options      *MetadataOptions `json:"options,omitempty"`
}

// MetadataOptions provides configuration for metadata collection
type MetadataOptions struct {
	IncludeSchema  bool     `json:"include_schema"`
	IncludeMetrics bool     `json:"include_metrics"`
	IncludeTags    bool     `json:"include_tags"`
	MetadataTypes  []string `json:"metadata_types,omitempty"`
}

// MetricsRequest represents a request for resource metrics
type MetricsRequest struct {
	ResourceID   string          `json:"resource_id"`
	ResourceType string          `json:"resource_type"`
	TimeRange    *TimeRange      `json:"time_range,omitempty"`
	Options      *MetricsOptions `json:"options,omitempty"`
}

// TimeRange specifies a time period for metrics collection
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// MetricsOptions provides configuration for metrics collection
type MetricsOptions struct {
	MetricTypes []string `json:"metric_types,omitempty"`
	Aggregation string   `json:"aggregation,omitempty"` // sum, avg, min, max
	Granularity string   `json:"granularity,omitempty"` // 1m, 5m, 1h, 1d
}

// ResourceReference represents a reference to another resource
type ResourceReference struct {
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	RelationType string                 `json:"relation_type"` // depends_on, references, contains
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

// =============================================================================
// CREATE OBJECT REQUEST/RESPONSE TYPES
// =============================================================================

// CreateRequest represents a request to create a new managed resource
type CreateRequest struct {
	// Resource identity
	ObjectType string `json:"object_type"`
	Name       string `json:"name"`

	// Configuration
	Config       map[string]interface{} `json:"config"`
	Dependencies []string               `json:"dependencies,omitempty"`

	// Options
	Options  *CreateOptions         `json:"options,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// CreateOptions provides optional settings for create operations
type CreateOptions struct {
	DryRun       bool          `json:"dry_run"`           // preview changes without applying
	Timeout      time.Duration `json:"timeout,omitempty"` // operation timeout
	RetryCount   int           `json:"retry_count,omitempty"`
	ValidateOnly bool          `json:"validate_only"` // only validate, don't create
}

// CreateResponse represents the result of a create operation
type CreateResponse struct {
	// Resource identity
	ResourceID string                 `json:"resource_id"`
	State      map[string]interface{} `json:"state"`

	// Operation metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Warnings []string               `json:"warnings,omitempty"`
	Duration time.Duration          `json:"duration,omitempty"`

	// Status
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// ReadRequest represents a request to read a managed resource
type ReadRequest struct {
	ObjectType string `json:"object_type"`
	ResourceID string `json:"resource_id"`
	Name       string `json:"name"`
}

// ReadResponse represents the result of a read operation
type ReadResponse struct {
	State        map[string]interface{} `json:"state"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	NotFound     bool                   `json:"not_found"`
	LastModified time.Time              `json:"last_modified,omitempty"`
}

// UpdateRequest represents a request to update a managed resource
type UpdateRequest struct {
	ObjectType   string                 `json:"object_type"`
	ResourceID   string                 `json:"resource_id"`
	Name         string                 `json:"name"`
	Config       map[string]interface{} `json:"config"`
	CurrentState map[string]interface{} `json:"current_state,omitempty"`
	Options      *UpdateOptions         `json:"options,omitempty"`
}

// UpdateOptions provides optional settings for update operations
type UpdateOptions struct {
	DryRun        bool          `json:"dry_run"`
	Timeout       time.Duration `json:"timeout,omitempty"`
	ForceReplace  bool          `json:"force_replace"`            // force recreation if needed
	IgnoreChanges []string      `json:"ignore_changes,omitempty"` // properties to ignore
}

// UpdateResponse represents the result of an update operation
type UpdateResponse struct {
	NewState map[string]interface{} `json:"new_state"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Warnings []string               `json:"warnings,omitempty"`
	Changes  []PropertyChange       `json:"changes,omitempty"`
	Duration time.Duration          `json:"duration,omitempty"`
	Replaced bool                   `json:"replaced"` // true if resource was recreated
}

// PropertyChange represents a change to a specific property
type PropertyChange struct {
	Property        string      `json:"property"`
	OldValue        interface{} `json:"old_value"`
	NewValue        interface{} `json:"new_value"`
	Action          string      `json:"action"` // create, update, delete
	RequiresReplace bool        `json:"requires_replace"`
}

// DeleteRequest represents a request to delete a managed resource
type DeleteRequest struct {
	ObjectType string                 `json:"object_type"`
	ResourceID string                 `json:"resource_id"`
	Name       string                 `json:"name"`
	State      map[string]interface{} `json:"state,omitempty"`
	Options    *DeleteOptions         `json:"options,omitempty"`
}

// DeleteOptions provides optional settings for delete operations
type DeleteOptions struct {
	DryRun       bool          `json:"dry_run"`
	Timeout      time.Duration `json:"timeout,omitempty"`
	Force        bool          `json:"force"`         // force delete even with dependencies
	CreateBackup bool          `json:"create_backup"` // create backup before deletion
}

// DeleteResponse represents the result of a delete operation
type DeleteResponse struct {
	Warnings []string      `json:"warnings,omitempty"`
	BackupID string        `json:"backup_id,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`
	Success  bool          `json:"success"`
	Message  string        `json:"message,omitempty"`
}

// =============================================================================
// DISCOVER OBJECT REQUEST/RESPONSE TYPES
// =============================================================================

// DiscoverRequest represents a request to discover existing infrastructure
type DiscoverRequest struct {
	// Discovery scope
	ObjectTypes []string               `json:"object_types,omitempty"` // specific types to discover
	Filters     map[string]interface{} `json:"filters,omitempty"`      // discovery filters

	// Discovery options
	Options  *DiscoverOptions       `json:"options,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// DiscoverOptions provides optional settings for discover operations
type DiscoverOptions struct {
	MaxResults   int           `json:"max_results,omitempty"` // limit results
	Timeout      time.Duration `json:"timeout,omitempty"`     // discovery timeout
	Deep         bool          `json:"deep"`                  // deep discovery with relationships
	IncludeState bool          `json:"include_state"`         // include current state
	Concurrent   bool          `json:"concurrent"`            // concurrent discovery
}

// DiscoverResponse represents the result of a discover operation
type DiscoverResponse struct {
	Resources  []DiscoveredResource   `json:"resources"`
	HasMore    bool                   `json:"has_more"`
	NextToken  string                 `json:"next_token,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Duration   time.Duration          `json:"duration,omitempty"`
	TotalFound int                    `json:"total_found"`
}

// DiscoveredResource represents a discovered infrastructure resource
type DiscoveredResource struct {
	// Identity
	ObjectType string `json:"object_type"`
	ResourceID string `json:"resource_id"`
	Name       string `json:"name"`

	// Current state
	State    map[string]interface{} `json:"state,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Management status
	Managed    bool `json:"managed"`    // already managed by Kolumn
	Importable bool `json:"importable"` // can be imported
	ReadOnly   bool `json:"read_only"`  // read-only resource

	// Relationships
	Dependencies []ResourceReference `json:"dependencies,omitempty"`
	Dependents   []ResourceReference `json:"dependents,omitempty"`

	// Discovery metadata
	DiscoveredAt time.Time `json:"discovered_at"`
	Source       string    `json:"source,omitempty"` // discovery source
}

// ScanRequest represents a request to scan infrastructure for specific patterns
type ScanRequest struct {
	// Scan parameters
	ScanType    string                 `json:"scan_type"` // security, performance, compliance, etc.
	ObjectTypes []string               `json:"object_types,omitempty"`
	Filters     map[string]interface{} `json:"filters,omitempty"`

	// Scan options
	Options *ScanOptions `json:"options,omitempty"`
}

// ScanOptions provides optional settings for scan operations
type ScanOptions struct {
	Depth           int           `json:"depth,omitempty"` // scan depth
	Timeout         time.Duration `json:"timeout,omitempty"`
	IncludeMetrics  bool          `json:"include_metrics"`  // include performance metrics
	IncludeSecurity bool          `json:"include_security"` // include security analysis
}

// ScanResponse represents the result of a scan operation
type ScanResponse struct {
	Results  []ScanResult           `json:"results"`
	Summary  *ScanSummary           `json:"summary"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Duration time.Duration          `json:"duration,omitempty"`
}

// ScanResult represents a single scan finding
type ScanResult struct {
	// Finding details
	Type        string `json:"type"`     // issue, recommendation, info
	Category    string `json:"category"` // security, performance, compliance
	Severity    string `json:"severity"` // low, medium, high, critical
	Title       string `json:"title"`
	Description string `json:"description"`

	// Affected resource
	Resource *ResourceReference     `json:"resource,omitempty"`
	Location map[string]interface{} `json:"location,omitempty"`

	// Recommendations
	Recommendation string   `json:"recommendation,omitempty"`
	References     []string `json:"references,omitempty"`
}

// ScanSummary provides high-level scan statistics
type ScanSummary struct {
	TotalScanned int            `json:"total_scanned"`
	IssuesFound  int            `json:"issues_found"`
	BySeverity   map[string]int `json:"by_severity"`
	ByCategory   map[string]int `json:"by_category"`
	ByObjectType map[string]int `json:"by_object_type"`
}

// =============================================================================
// ADVANCED OPERATION REQUEST/RESPONSE TYPES
// =============================================================================

// PlanRequest represents a request to plan changes
type PlanRequest struct {
	ObjectType    string                 `json:"object_type"`
	Name          string                 `json:"name"`
	DesiredConfig map[string]interface{} `json:"desired_config"`
	CurrentState  map[string]interface{} `json:"current_state,omitempty"`
	Options       *PlanOptions           `json:"options,omitempty"`
}

// PlanOptions provides optional settings for plan operations
type PlanOptions struct {
	Detailed          bool `json:"detailed"`           // include detailed change analysis
	ValidateOnly      bool `json:"validate_only"`      // only validate, don't generate plan
	CheckDependencies bool `json:"check_dependencies"` // analyze dependency impact
}

// PlanResponse represents the result of a plan operation
type PlanResponse struct {
	Changes  []PlannedChange   `json:"changes"`
	Valid    bool              `json:"valid"`
	Summary  *PlanSummary      `json:"summary"`
	Warnings []string          `json:"warnings,omitempty"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Duration time.Duration     `json:"duration,omitempty"`
}

// PlannedChange represents a planned change to a resource
type PlannedChange struct {
	Action          string        `json:"action"` // create, update, delete, replace
	Property        string        `json:"property,omitempty"`
	OldValue        interface{}   `json:"old_value,omitempty"`
	NewValue        interface{}   `json:"new_value,omitempty"`
	RequiresReplace bool          `json:"requires_replace"`
	RiskLevel       string        `json:"risk_level"` // low, medium, high, critical
	Description     string        `json:"description"`
	EstimatedTime   time.Duration `json:"estimated_time,omitempty"`
}

// PlanSummary provides high-level plan statistics
type PlanSummary struct {
	TotalChanges    int            `json:"total_changes"`
	ByAction        map[string]int `json:"by_action"`
	RequiresReplace bool           `json:"requires_replace"`
	EstimatedTime   time.Duration  `json:"estimated_time"`
	RiskLevel       string         `json:"risk_level"`
}

// ImportRequest represents a request to import existing resources
type ImportRequest struct {
	ObjectType string                 `json:"object_type"`
	ResourceID string                 `json:"resource_id"`
	Name       string                 `json:"name,omitempty"`
	Config     map[string]interface{} `json:"config,omitempty"`
	Options    *ImportOptions         `json:"options,omitempty"`
}

// ImportOptions provides optional settings for import operations
type ImportOptions struct {
	DryRun         bool `json:"dry_run"`
	GenerateConfig bool `json:"generate_config"` // generate configuration from current state
	ValidateConfig bool `json:"validate_config"` // validate generated/provided config
}

// ImportResponse represents the result of an import operation
type ImportResponse struct {
	State        map[string]interface{} `json:"state"`
	Config       map[string]interface{} `json:"config,omitempty"` // generated config
	Dependencies []string               `json:"dependencies,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
	Success      bool                   `json:"success"`
	Message      string                 `json:"message,omitempty"`
}

// ValidateRequest represents a request to validate configuration
type ValidateRequest struct {
	ObjectType string                 `json:"object_type"`
	Config     map[string]interface{} `json:"config"`
	Options    *ValidateOptions       `json:"options,omitempty"`
}

// ValidateOptions provides optional settings for validation
type ValidateOptions struct {
	Strict            bool `json:"strict"`             // strict validation mode
	CheckDependencies bool `json:"check_dependencies"` // validate dependencies
}

// ValidateResponse represents the result of a validation operation
type ValidateResponse struct {
	Valid       bool              `json:"valid"`
	Errors      []ValidationError `json:"errors,omitempty"`
	Warnings    []ValidationError `json:"warnings,omitempty"`
	Suggestions []string          `json:"suggestions,omitempty"`
}

// =============================================================================
// INTROSPECTION REQUEST/RESPONSE TYPES
// =============================================================================

// IntrospectRequest represents a request for deep resource introspection
type IntrospectRequest struct {
	ObjectType string             `json:"object_type"`
	ResourceID string             `json:"resource_id"`
	Options    *IntrospectOptions `json:"options,omitempty"`
}

// IntrospectOptions provides optional settings for introspection
type IntrospectOptions struct {
	IncludeSchema    bool `json:"include_schema"`    // include schema information
	IncludeMetrics   bool `json:"include_metrics"`   // include performance metrics
	IncludeRelations bool `json:"include_relations"` // include relationship data
	IncludeSecurity  bool `json:"include_security"`  // include security analysis
}

// IntrospectResponse represents the result of introspection
type IntrospectResponse struct {
	Schema       *ObjectSchema          `json:"schema,omitempty"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	Relations    []ResourceReference    `json:"relations,omitempty"`
	Security     *SecurityAnalysis      `json:"security,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
	Capabilities []string               `json:"capabilities,omitempty"`
}

// SecurityAnalysis provides security analysis results
type SecurityAnalysis struct {
	Threats         []SecurityThreat   `json:"threats,omitempty"`
	Controls        []SecurityControl  `json:"controls,omitempty"`
	Compliance      []ComplianceResult `json:"compliance,omitempty"`
	Recommendations []string           `json:"recommendations,omitempty"`
}

// SecurityThreat represents a security threat
type SecurityThreat struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Mitigation  string `json:"mitigation,omitempty"`
}

// SecurityControl represents a security control
type SecurityControl struct {
	Name          string `json:"name"`
	Status        string `json:"status"` // enabled, disabled, not_applicable
	Description   string `json:"description"`
	Effectiveness string `json:"effectiveness,omitempty"` // high, medium, low
}

// ComplianceResult represents compliance check results
type ComplianceResult struct {
	Standard     string   `json:"standard"` // GDPR, SOX, HIPAA, etc.
	Status       string   `json:"status"`   // compliant, non_compliant, not_applicable
	Details      string   `json:"details,omitempty"`
	Requirements []string `json:"requirements,omitempty"`
}

// =============================================================================
// BACKUP/RESTORE REQUEST/RESPONSE TYPES
// =============================================================================

// BackupRequest represents a request to backup resources
type BackupRequest struct {
	ObjectTypes []string       `json:"object_types,omitempty"`
	ResourceIDs []string       `json:"resource_ids,omitempty"`
	Options     *BackupOptions `json:"options,omitempty"`
}

// BackupOptions provides optional settings for backup operations
type BackupOptions struct {
	BackupType  string        `json:"backup_type"` // full, incremental, differential
	Compression bool          `json:"compression"`
	Encryption  bool          `json:"encryption"`
	Timeout     time.Duration `json:"timeout,omitempty"`
}

// BackupResponse represents the result of a backup operation
type BackupResponse struct {
	BackupID      string        `json:"backup_id"`
	Size          int64         `json:"size,omitempty"` // backup size in bytes
	Duration      time.Duration `json:"duration,omitempty"`
	ResourceCount int           `json:"resource_count"` // number of resources backed up
	Success       bool          `json:"success"`
	Message       string        `json:"message,omitempty"`
}

// RestoreRequest represents a request to restore from backup
type RestoreRequest struct {
	BackupID string          `json:"backup_id"`
	Options  *RestoreOptions `json:"options,omitempty"`
}

// RestoreOptions provides optional settings for restore operations
type RestoreOptions struct {
	DryRun           bool          `json:"dry_run"`
	Timeout          time.Duration `json:"timeout,omitempty"`
	ForceReplace     bool          `json:"force_replace"`               // replace existing resources
	SelectiveRestore []string      `json:"selective_restore,omitempty"` // specific resources to restore
}

// RestoreResponse represents the result of a restore operation
type RestoreResponse struct {
	RestoredCount int           `json:"restored_count"`
	SkippedCount  int           `json:"skipped_count"`
	Duration      time.Duration `json:"duration,omitempty"`
	Warnings      []string      `json:"warnings,omitempty"`
	Success       bool          `json:"success"`
	Message       string        `json:"message,omitempty"`
}
