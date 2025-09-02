// Package state - enhanced_types.go provides enhanced multi-provider state management types for the SDK
package state

import (
	"context"
	"time"

	"github.com/schemabounce/kolumn/sdk/types"
)

// EnhancedResourceState represents a resource with multi-provider support and enhanced metadata
type EnhancedResourceState struct {
	// Core identification
	Type     string `json:"type"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Category string `json:"category"` // streaming, database, etl, orchestration, storage

	// Multi-provider support
	Dependencies []ResourceDependency `json:"dependencies,omitempty"`
	Collections  []string             `json:"collections,omitempty"` // Resource groups

	// Enhanced state management
	LifecycleState string    `json:"lifecycle_state"` // Provider-specific state
	DesiredState   string    `json:"desired_state"`   // Target state
	DriftDetected  bool      `json:"drift_detected"`
	LastDriftCheck time.Time `json:"last_drift_check"`

	// Flexible attributes with schema validation
	Attributes     map[string]interface{} `json:"attributes"`
	ComputedAttrs  map[string]interface{} `json:"computed_attributes"`
	SensitiveAttrs []string               `json:"sensitive_attributes"`

	// Enhanced metadata
	Metadata       ResourceMetadata       `json:"metadata"`
	ProviderConfig map[string]interface{} `json:"provider_config,omitempty"`

	// State tracking (maintains compatibility with UniversalResource)
	Instances []ResourceInstance `json:"instances"`
	Mode      string             `json:"mode"`
}

// ResourceDependency represents a dependency between resources across providers
type ResourceDependency struct {
	Provider     string                 `json:"provider"`
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	Relationship string                 `json:"relationship"` // depends_on, creates, configures, consumes, produces
	Optional     bool                   `json:"optional"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceMetadata provides universal metadata fields for all resource types
type ResourceMetadata struct {
	// Universal metadata
	Tags        map[string]string `json:"tags,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Owner       string            `json:"owner,omitempty"`
	Environment string            `json:"environment,omitempty"`

	// Provider-specific metadata
	ProviderMeta map[string]interface{} `json:"provider_metadata,omitempty"`

	// Operational metadata
	CostCenter   string `json:"cost_center,omitempty"`
	BusinessUnit string `json:"business_unit,omitempty"`
	CreatedBy    string `json:"created_by,omitempty"`
	Purpose      string `json:"purpose,omitempty"`
	Project      string `json:"project,omitempty"`

	// Data governance
	DataClassification   string      `json:"data_classification,omitempty"`   // public, internal, confidential, restricted
	ComplianceFrameworks []string    `json:"compliance_frameworks,omitempty"` // GDPR, HIPAA, SOX, etc.
	RetentionPolicy      string      `json:"retention_policy,omitempty"`
	ContactInfo          ContactInfo `json:"contact_info,omitempty"`
}

// ContactInfo provides contact information for resources
type ContactInfo struct {
	PrimaryOwner   string `json:"primary_owner,omitempty"`
	SecondaryOwner string `json:"secondary_owner,omitempty"`
	Team           string `json:"team,omitempty"`
	Email          string `json:"email,omitempty"`
	SlackChannel   string `json:"slack_channel,omitempty"`
	OnCallRotation string `json:"oncall_rotation,omitempty"`
}

// ResourceCollection represents a group of related multi-provider resources
type ResourceCollection struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"` // pipeline, stack, environment, service
	Description string `json:"description"`

	// Multi-provider resources
	Resources []ResourceReference `json:"resources"`

	// Collection-level metadata
	Metadata ResourceMetadata `json:"metadata"`

	// State management
	DesiredState string           `json:"desired_state"` // active, paused, destroyed
	Status       CollectionStatus `json:"status"`

	// Dependencies between collections
	Dependencies []CollectionDependency `json:"dependencies,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ResourceReference identifies a resource within a collection
type ResourceReference struct {
	Provider string                 `json:"provider"`
	Type     string                 `json:"type"`
	Name     string                 `json:"name"`
	Role     string                 `json:"role"`             // source, sink, processor, storage, compute
	Config   map[string]interface{} `json:"config,omitempty"` // Collection-specific config overrides
}

// CollectionStatus tracks the overall status of a resource collection
type CollectionStatus struct {
	State         string    `json:"state"` // active, starting, stopping, error, unknown
	Healthy       bool      `json:"healthy"`
	ResourceCount int       `json:"resource_count"`
	HealthyCount  int       `json:"healthy_count"`
	ErrorCount    int       `json:"error_count"`
	LastUpdated   time.Time `json:"last_updated"`
	Issues        []string  `json:"issues,omitempty"`
	LastCheck     time.Time `json:"last_check"`
}

// CollectionDependency represents dependencies between collections
type CollectionDependency struct {
	CollectionID string `json:"collection_id"`
	Type         string `json:"type"` // depends_on, provides_to, configures
	Optional     bool   `json:"optional"`
}

// EnhancedResourceDefinition extends resource definitions with enhanced schema support
type EnhancedResourceDefinition struct {
	// Core definition fields
	Type         string   `json:"type"`
	Provider     string   `json:"provider"`
	Category     string   `json:"category"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`

	// Enhanced schema support
	StateSchema     *ResourceStateSchema `json:"state_schema"`
	LifecycleStates []string             `json:"lifecycle_states"`
	MetadataSchema  *MetadataSchema      `json:"metadata_schema"`
	Dependencies    []DependencyPattern  `json:"dependencies"`

	// Migration and rollback support
	DefaultMigrationStrategy string `json:"default_migration_strategy"`
	DefaultRollbackStrategy  string `json:"default_rollback_strategy"`

	// Validation
	ValidationRules []ValidationRule `json:"validation_rules"`
}

// ResourceStateSchema defines the schema for resource state attributes
type ResourceStateSchema struct {
	Version    int                       `json:"version"`
	Required   []string                  `json:"required"`
	Properties map[string]PropertySchema `json:"properties"`
	Computed   []string                  `json:"computed"`  // Provider-computed fields
	Sensitive  []string                  `json:"sensitive"` // Sensitive fields (passwords, keys, etc.)
}

// PropertySchema defines the schema for individual resource properties
type PropertySchema struct {
	Type        string                    `json:"type"` // string, int, bool, array, object, number
	Description string                    `json:"description"`
	Default     interface{}               `json:"default,omitempty"`
	Enum        []string                  `json:"enum,omitempty"`
	Pattern     string                    `json:"pattern,omitempty"`
	MinLength   *int                      `json:"min_length,omitempty"`
	MaxLength   *int                      `json:"max_length,omitempty"`
	Minimum     *float64                  `json:"minimum,omitempty"`
	Maximum     *float64                  `json:"maximum,omitempty"`
	Sensitive   bool                      `json:"sensitive"`            // For secrets/passwords
	Computed    bool                      `json:"computed"`             // Provider-computed field
	ForceNew    bool                      `json:"force_new"`            // Changes require resource recreation
	Items       *PropertySchema           `json:"items,omitempty"`      // For array types
	Properties  map[string]PropertySchema `json:"properties,omitempty"` // For object types
}

// MetadataSchema defines required and optional metadata fields for a resource type
type MetadataSchema struct {
	RequiredTags   []string          `json:"required_tags,omitempty"`
	RequiredLabels []string          `json:"required_labels,omitempty"`
	RequiredFields []string          `json:"required_fields,omitempty"` // Required metadata fields
	TagPatterns    map[string]string `json:"tag_patterns,omitempty"`    // Regex patterns for tag values
}

// DependencyPattern defines common dependency patterns for a resource type
type DependencyPattern struct {
	Type         string `json:"type"`         // The dependency resource type
	Provider     string `json:"provider"`     // The dependency provider (empty for same provider)
	Relationship string `json:"relationship"` // The type of relationship
	Required     bool   `json:"required"`     // Whether this dependency is required
	Multiple     bool   `json:"multiple"`     // Whether multiple dependencies of this type are allowed
}

// ValidationRule defines custom validation rules for resources
type ValidationRule struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"` // field, cross_field, dependency, custom
	Field      string                 `json:"field,omitempty"`
	Condition  string                 `json:"condition"` // The validation condition
	Message    string                 `json:"message"`   // Error message if validation fails
	Severity   string                 `json:"severity"`  // error, warning, info
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// ResourceGraph represents the dependency graph of resources
type ResourceGraph struct {
	Nodes []ResourceNode `json:"nodes"`
	Edges []ResourceEdge `json:"edges"`
}

// ResourceNode represents a resource in the dependency graph
type ResourceNode struct {
	ID       string                 `json:"id"` // Format: provider.type.name
	Type     string                 `json:"type"`
	Provider string                 `json:"provider"`
	Category string                 `json:"category"`
	Name     string                 `json:"name"`
	State    string                 `json:"state"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ResourceEdge represents a dependency relationship in the graph
type ResourceEdge struct {
	From         string `json:"from"`         // Source resource ID
	To           string `json:"to"`           // Target resource ID
	Relationship string `json:"relationship"` // Type of dependency
	Optional     bool   `json:"optional"`
	Weight       int    `json:"weight"` // For dependency ordering
}

// MetadataQuery provides a flexible way to query resources by metadata
type MetadataQuery struct {
	Tags        map[string]string `json:"tags,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Owner       string            `json:"owner,omitempty"`
	Environment string            `json:"environment,omitempty"`
	Provider    string            `json:"provider,omitempty"`
	Category    string            `json:"category,omitempty"`
	Project     string            `json:"project,omitempty"`
	Team        string            `json:"team,omitempty"`

	// Advanced query options
	LifecycleState []string `json:"lifecycle_state,omitempty"`
	DriftDetected  *bool    `json:"drift_detected,omitempty"`
	Collections    []string `json:"collections,omitempty"`

	// Date range filters
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	UpdatedAfter  *time.Time `json:"updated_after,omitempty"`
	UpdatedBefore *time.Time `json:"updated_before,omitempty"`
}

// StateTransition represents a lifecycle state transition
type StateTransition struct {
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Action    string                 `json:"action"`
	Timestamp time.Time              `json:"timestamp"`
	Actor     string                 `json:"actor"`  // Who/what initiated the transition
	Reason    string                 `json:"reason"` // Why the transition occurred
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Additional types for provider state adapters and testing

// ResourceFieldSchema defines schema for individual resource fields
type ResourceFieldSchema struct {
	Type         string                 `json:"type"`
	Required     bool                   `json:"required"`
	Description  string                 `json:"description"`
	DefaultValue interface{}            `json:"default_value,omitempty"`
	Constraints  []string               `json:"constraints,omitempty"`
	Examples     []interface{}          `json:"examples,omitempty"`
	Deprecated   bool                   `json:"deprecated"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// LifecycleHook represents a function that executes during resource lifecycle events
type LifecycleHook func(ctx context.Context, state *EnhancedResourceState) error

// AdapterMetrics provides performance metrics for provider state adapters
type AdapterMetrics struct {
	TotalConversions      int64         `json:"total_conversions"`
	SuccessfulConversions int64         `json:"successful_conversions"`
	FailedConversions     int64         `json:"failed_conversions"`
	AverageConversionTime time.Duration `json:"average_conversion_time"`
	LastActivity          time.Time     `json:"last_activity"`
}

// AdapterHealth provides health status information for provider state adapters
type AdapterHealth struct {
	Status      string                 `json:"status"` // healthy, degraded, unhealthy
	LastCheck   time.Time              `json:"last_check"`
	Errors      []string               `json:"errors,omitempty"`
	Warnings    []string               `json:"warnings,omitempty"`
	Diagnostics map[string]interface{} `json:"diagnostics,omitempty"`
}

// ResourceInstance extends types.ResourceInstance with enhanced functionality
type ResourceInstance struct {
	IndexKey   interface{}            `json:"index_key,omitempty"`
	Status     types.ResourceStatus   `json:"status"`
	Attributes map[string]interface{} `json:"attributes"`
	Private    []byte                 `json:"private,omitempty"`
	Metadata   ResourceMetadata       `json:"metadata"`

	// Enhanced fields
	Tainted              bool                   `json:"tainted,omitempty"`
	Deposed              string                 `json:"deposed,omitempty"`
	CreateBeforeDestroy  bool                   `json:"create_before_destroy,omitempty"`
	ProviderMeta         map[string]interface{} `json:"provider_meta,omitempty"`
	Dependencies         []ResourceDependency   `json:"dependencies,omitempty"`
	DriftStatus          string                 `json:"drift_status,omitempty"`
	LastValidated        time.Time              `json:"last_validated,omitempty"`
	ValidationResults    []ValidationResult     `json:"validation_results,omitempty"`
	LifecycleTransitions []StateTransition      `json:"lifecycle_transitions,omitempty"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

// ValidationResult represents the result of a validation check
type ValidationResult struct {
	Rule      string                 `json:"rule"`
	Passed    bool                   `json:"passed"`
	Message   string                 `json:"message,omitempty"`
	Severity  string                 `json:"severity"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// ConversionOptions provides options for state conversion operations
type ConversionOptions struct {
	ValidateSchema      bool `json:"validate_schema"`
	PreserveSensitive   bool `json:"preserve_sensitive"`
	IncludeDependencies bool `json:"include_dependencies"`
	IncludeMetadata     bool `json:"include_metadata"`
}

// EnhancedStateAdapter extends StateAdapter with additional capabilities
type EnhancedStateAdapter interface {
	StateAdapter

	// Enhanced conversion methods
	ToEnhancedState(providerState interface{}, options *ConversionOptions) (*EnhancedResourceState, error)
	FromEnhancedState(enhancedState *EnhancedResourceState, options *ConversionOptions) (interface{}, error)

	// Schema and validation
	GetStateSchema() *ResourceStateSchema
	ValidateAttributes(attributes map[string]interface{}) []ValidationResult

	// Metadata handling
	ExtractMetadata(providerState interface{}) (ResourceMetadata, error)
	ApplyMetadata(providerState interface{}, metadata ResourceMetadata) (interface{}, error)

	// Health and metrics
	GetHealth() AdapterHealth
	GetMetrics() AdapterMetrics

	// Lifecycle management
	RegisterLifecycleHook(event string, hook LifecycleHook)
	ExecuteLifecycleHooks(ctx context.Context, event string, state *EnhancedResourceState) error
}
