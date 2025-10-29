// Package state provides state adaptation interfaces for Kolumn providers
//
// This package defines interfaces that enable providers to convert between
// their native state format and the universal state format.
package state

import (
	"context"
	"time"
)

// StateAdapter defines the interface for converting between provider-specific
// state formats and the universal state format
type StateAdapter interface {
	// Provider Information
	GetProvider() string
	GetCategory() string
	GetSupportedTypes() []string

	// State Conversion
	ToUniversalState(ctx context.Context, providerState interface{}) (*UniversalState, error)
	FromUniversalState(ctx context.Context, universalState *UniversalState) (interface{}, error)

	// Schema and Validation
	GetResourceSchema(resourceType string) (*ResourceSchema, error)
	ValidateState(ctx context.Context, state *UniversalState) error
	ValidateTransition(ctx context.Context, oldState, newState *UniversalState) error

	// Dependency Management
	ExtractDependencies(ctx context.Context, resource *UniversalResource) ([]string, error)
	ResolveDependency(ctx context.Context, dependency string) (*UniversalResource, error)

	// Lifecycle Hooks
	GetLifecycleHooks() []LifecycleHook
	ExecuteLifecycleHook(ctx context.Context, hook LifecycleHook, resource *UniversalResource) error

	// Monitoring and Health
	GetMetrics(ctx context.Context) (*AdapterMetrics, error)
	GetHealth(ctx context.Context) (*AdapterHealth, error)
}

// ResourceSchema defines the schema for a specific resource type
type ResourceSchema struct {
	Type        string                       `json:"type"`
	Version     string                       `json:"version"`
	Properties  map[string]*PropertySchema   `json:"properties"`
	Required    []string                     `json:"required"`
	Constraints map[string]*ConstraintSchema `json:"constraints"`
	Metadata    map[string]interface{}       `json:"metadata"`
}

// PropertySchema defines the schema for a resource property
type PropertySchema struct {
	Type        string                     `json:"type"`
	Description string                     `json:"description"`
	Required    bool                       `json:"required"`
	Default     interface{}                `json:"default,omitempty"`
	Enum        []interface{}              `json:"enum,omitempty"`
	Format      string                     `json:"format,omitempty"`
	Pattern     string                     `json:"pattern,omitempty"`
	MinLength   *int                       `json:"min_length,omitempty"`
	MaxLength   *int                       `json:"max_length,omitempty"`
	Minimum     *float64                   `json:"minimum,omitempty"`
	Maximum     *float64                   `json:"maximum,omitempty"`
	Properties  map[string]*PropertySchema `json:"properties,omitempty"`
	Items       *PropertySchema            `json:"items,omitempty"`
}

// ConstraintSchema defines validation constraints for resources
type ConstraintSchema struct {
	Type       string                 `json:"type"`
	Expression string                 `json:"expression"`
	Message    string                 `json:"message"`
	Severity   string                 `json:"severity"` // "error", "warning", "info"
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// LifecycleHook defines a lifecycle hook for state operations
type LifecycleHook struct {
	Name      string                 `json:"name"`
	Stage     LifecycleStage         `json:"stage"`
	Priority  int                    `json:"priority"`
	Function  string                 `json:"function"`
	Async     bool                   `json:"async"`
	Retryable bool                   `json:"retryable"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// LifecycleStage represents the stage in the resource lifecycle
type LifecycleStage string

const (
	LifecycleStagePreCreate  LifecycleStage = "pre_create"
	LifecycleStagePostCreate LifecycleStage = "post_create"
	LifecycleStagePreUpdate  LifecycleStage = "pre_update"
	LifecycleStagePostUpdate LifecycleStage = "post_update"
	LifecycleStagePreDelete  LifecycleStage = "pre_delete"
	LifecycleStagePostDelete LifecycleStage = "post_delete"
	LifecycleStagePreRead    LifecycleStage = "pre_read"
	LifecycleStagePostRead   LifecycleStage = "post_read"
)

// AdapterMetrics provides metrics about the state adapter
type AdapterMetrics struct {
	ConversionsTotal   int64                  `json:"conversions_total"`
	ConversionErrors   int64                  `json:"conversion_errors"`
	ValidationErrors   int64                  `json:"validation_errors"`
	AvgConversionTime  float64                `json:"avg_conversion_time_ms"`
	LastConversion     *time.Time             `json:"last_conversion,omitempty"`
	ResourceTypeCounts map[string]int64       `json:"resource_type_counts"`
	DependencyCount    int64                  `json:"dependency_count"`
	LifecycleHookCount int64                  `json:"lifecycle_hook_count"`
	CacheHitRate       float64                `json:"cache_hit_rate"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// AdapterHealth provides health information about the state adapter
type AdapterHealth struct {
	Status           string                 `json:"status"` // "healthy", "degraded", "unhealthy"
	LastCheck        time.Time              `json:"last_check"`
	ResponseTime     time.Duration          `json:"response_time"`
	ErrorRate        float64                `json:"error_rate"`
	SupportedTypes   []string               `json:"supported_types"`
	SchemaVersion    string                 `json:"schema_version"`
	CanConvert       bool                   `json:"can_convert"`
	CanValidate      bool                   `json:"can_validate"`
	DependencyHealth map[string]string      `json:"dependency_health"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// StateConversionOptions provides options for state conversion
type StateConversionOptions struct {
	IncludeMetadata      bool                   `json:"include_metadata"`
	IncludeDependencies  bool                   `json:"include_dependencies"`
	ValidateOnConvert    bool                   `json:"validate_on_convert"`
	EnableTransformation bool                   `json:"enable_transformation"`
	TransformationRules  map[string]interface{} `json:"transformation_rules,omitempty"`
	Context              map[string]interface{} `json:"context,omitempty"`
}

// DefaultStateConversionOptions returns default options for state conversion
func DefaultStateConversionOptions() *StateConversionOptions {
	return &StateConversionOptions{
		IncludeMetadata:      true,
		IncludeDependencies:  true,
		ValidateOnConvert:    true,
		EnableTransformation: false,
		TransformationRules:  make(map[string]interface{}),
		Context:              make(map[string]interface{}),
	}
}

// AdapterError represents an error from state adapter operations
type AdapterError struct {
	Operation    string
	ResourceType string
	Cause        error
	Retryable    bool
}

func (e *AdapterError) Error() string {
	if e.Cause != nil {
		return e.Operation + " failed for resource type " + e.ResourceType + ": " + e.Cause.Error()
	}
	return e.Operation + " failed for resource type " + e.ResourceType
}

func (e *AdapterError) Unwrap() error {
	return e.Cause
}

// NewAdapterError creates a new AdapterError
func NewAdapterError(operation, resourceType string, cause error, retryable bool) *AdapterError {
	return &AdapterError{
		Operation:    operation,
		ResourceType: resourceType,
		Cause:        cause,
		Retryable:    retryable,
	}
}

// IsRetryable returns whether the error is retryable
func (e *AdapterError) IsRetryable() bool {
	return e.Retryable
}
