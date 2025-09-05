// Package create provides types and interfaces for create object handlers
package create

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/schemabounce/kolumn/sdk/core"
)

// =============================================================================
// EXTENSION INTERFACES
// =============================================================================

// Interceptor provides pre/post operation hooks for create objects
type Interceptor interface {
	// Intercept is called before operations to allow modification or blocking
	Intercept(ctx context.Context, operation string, req interface{}) error

	// Name returns the interceptor name for debugging
	Name() string
}

// =============================================================================
// ADDITIONAL REQUEST/RESPONSE TYPES FOR CREATE OPERATIONS
// =============================================================================

// DriftRequest represents a request to detect drift in a managed resource
type DriftRequest struct {
	ObjectType   string                 `json:"object_type"`
	ResourceID   string                 `json:"resource_id"`
	ManagedState map[string]interface{} `json:"managed_state"` // state as managed by Kolumn
	Options      *DriftOptions          `json:"options,omitempty"`
}

// DriftOptions provides options for drift detection
type DriftOptions struct {
	IgnoreProperties []string `json:"ignore_properties,omitempty"` // properties to ignore in drift detection
	DeepCheck        bool     `json:"deep_check"`                  // perform deep drift analysis
	IncludeMetadata  bool     `json:"include_metadata"`            // include metadata in drift analysis
}

// DriftResponse represents the result of drift detection
type DriftResponse struct {
	HasDrift    bool                   `json:"has_drift"`
	ActualState map[string]interface{} `json:"actual_state,omitempty"` // current actual state
	Differences []StateDifference      `json:"differences,omitempty"`  // specific differences found
	Summary     *DriftSummary          `json:"summary,omitempty"`
}

// StateDifference represents a specific difference between managed and actual state
type StateDifference struct {
	Property     string      `json:"property"`
	ManagedValue interface{} `json:"managed_value"`
	ActualValue  interface{} `json:"actual_value"`
	DriftType    string      `json:"drift_type"` // added, removed, changed
	Impact       string      `json:"impact"`     // low, medium, high, critical
	Description  string      `json:"description,omitempty"`
}

// DriftSummary provides high-level drift statistics
type DriftSummary struct {
	TotalDifferences int            `json:"total_differences"`
	ByImpact         map[string]int `json:"by_impact"`
	ByDriftType      map[string]int `json:"by_drift_type"`
	RequiresAction   bool           `json:"requires_action"`
}

// RefreshRequest represents a request to refresh resource state
type RefreshRequest struct {
	ObjectType string `json:"object_type"`
	ResourceID string `json:"resource_id"`
	Name       string `json:"name"`
}

// RefreshResponse represents the result of a state refresh
type RefreshResponse struct {
	State        map[string]interface{} `json:"state"`
	LastModified time.Time              `json:"last_modified,omitempty"`
	Changed      bool                   `json:"changed"` // true if state changed during refresh
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// =============================================================================
// BUILT-IN VALIDATORS
// =============================================================================

// TypeValidator ensures properties have correct types
type TypeValidator struct {
	PropertyTypes map[string]string // property name -> expected type
}

// NewTypeValidator creates a validator for property types
func NewTypeValidator(types map[string]string) *TypeValidator {
	return &TypeValidator{
		PropertyTypes: types,
	}
}

// Validate checks that properties have correct types
func (v *TypeValidator) Validate(config map[string]interface{}) error {
	for prop, value := range config {
		expectedType, exists := v.PropertyTypes[prop]
		if !exists {
			continue // Property not in our type map, skip
		}

		if !isValidType(value, expectedType) {
			return fmt.Errorf("property '%s' must be of type %s", prop, expectedType)
		}
	}
	return nil
}

// Name returns the validator name
func (v *TypeValidator) Name() string {
	return "property_types"
}

// PatternValidator validates string properties against regular expressions
type PatternValidator struct {
	PropertyPatterns map[string]string // property name -> regex pattern
}

// NewPatternValidator creates a validator for string patterns
func NewPatternValidator(patterns map[string]string) *PatternValidator {
	return &PatternValidator{
		PropertyPatterns: patterns,
	}
}

// Validate checks string properties against patterns
func (v *PatternValidator) Validate(config map[string]interface{}) error {
	for prop, value := range config {
		pattern, exists := v.PropertyPatterns[prop]
		if !exists {
			continue // Property not in our pattern map, skip
		}

		strValue, ok := value.(string)
		if !ok {
			continue // Not a string, skip pattern validation
		}

		// In a real implementation, you would use regexp package
		// For this example, we'll do a simple check
		if pattern != "" && strValue == "" {
			return fmt.Errorf("property '%s' cannot be empty", prop)
		}
	}
	return nil
}

// Name returns the validator name
func (v *PatternValidator) Name() string {
	return "property_patterns"
}

// =============================================================================
// BUILT-IN INTERCEPTORS
// =============================================================================

// LoggingInterceptor logs operations for debugging
type LoggingInterceptor struct {
	Logger Logger // Logger interface for flexibility
}

// Logger interface for flexible logging
type Logger interface {
	Log(level string, message string, fields map[string]interface{})
}

// NewLoggingInterceptor creates a logging interceptor
func NewLoggingInterceptor(logger Logger) *LoggingInterceptor {
	return &LoggingInterceptor{
		Logger: logger,
	}
}

// Intercept logs the operation
func (i *LoggingInterceptor) Intercept(ctx context.Context, operation string, req interface{}) error {
	if i.Logger != nil {
		i.Logger.Log("info", "operation starting", map[string]interface{}{
			"operation":    operation,
			"request_type": fmt.Sprintf("%T", req),
		})
	}
	return nil
}

// Name returns the interceptor name
func (i *LoggingInterceptor) Name() string {
	return "logging"
}

// MetricsInterceptor collects metrics for operations
type MetricsInterceptor struct {
	MetricsCollector MetricsCollector
}

// MetricsCollector interface for flexible metrics collection
type MetricsCollector interface {
	Increment(metric string, tags map[string]string)
	Timing(metric string, duration time.Duration, tags map[string]string)
}

// NewMetricsInterceptor creates a metrics interceptor
func NewMetricsInterceptor(collector MetricsCollector) *MetricsInterceptor {
	return &MetricsInterceptor{
		MetricsCollector: collector,
	}
}

// Intercept collects metrics for the operation
func (i *MetricsInterceptor) Intercept(ctx context.Context, operation string, req interface{}) error {
	if i.MetricsCollector != nil {
		tags := map[string]string{
			"operation": operation,
		}
		i.MetricsCollector.Increment("operation.started", tags)
	}
	return nil
}

// Name returns the interceptor name
func (i *MetricsInterceptor) Name() string {
	return "metrics"
}

// =============================================================================
// BUILT-IN PLANNERS
// =============================================================================

// DefaultPlanner provides basic planning functionality
type DefaultPlanner struct {
	ObjectType string
}

// NewDefaultPlanner creates a default planner for an object type
func NewDefaultPlanner(objectType string) *DefaultPlanner {
	return &DefaultPlanner{
		ObjectType: objectType,
	}
}

// Plan provides basic planning logic
func (p *DefaultPlanner) Plan(ctx context.Context, req *core.PlanRequest) (*core.PlanResponse, error) {
	var changes []core.PlannedChange

	// Simple planning: compare desired vs current configuration
	if req.CurrentState == nil || len(req.CurrentState) == 0 {
		// No current state, this is a create operation
		changes = append(changes, core.PlannedChange{
			Action:      "create",
			Description: fmt.Sprintf("Create new %s resource", p.ObjectType),
			RiskLevel:   "medium",
		})
	} else {
		// Compare configurations for updates
		for key, newValue := range req.DesiredConfig {
			oldValue, exists := req.CurrentState[key]
			if !exists || oldValue != newValue {
				changes = append(changes, core.PlannedChange{
					Action:      "update",
					Property:    key,
					OldValue:    oldValue,
					NewValue:    newValue,
					Description: fmt.Sprintf("Update %s.%s", p.ObjectType, key),
					RiskLevel:   "low",
				})
			}
		}

		// Check for removed properties
		for key, oldValue := range req.CurrentState {
			if _, exists := req.DesiredConfig[key]; !exists {
				changes = append(changes, core.PlannedChange{
					Action:      "delete",
					Property:    key,
					OldValue:    oldValue,
					Description: fmt.Sprintf("Remove %s.%s", p.ObjectType, key),
					RiskLevel:   "medium",
				})
			}
		}
	}

	// Determine overall risk level
	overallRisk := "low"
	for _, change := range changes {
		if change.RiskLevel == "high" || change.RiskLevel == "critical" {
			overallRisk = change.RiskLevel
		} else if change.RiskLevel == "medium" && overallRisk == "low" {
			overallRisk = "medium"
		}
	}

	summary := &core.PlanSummary{
		TotalChanges: len(changes),
		ByAction:     make(map[string]int),
		RiskLevel:    overallRisk,
	}

	for _, change := range changes {
		summary.ByAction[change.Action]++
	}

	return &core.PlanResponse{
		Changes: changes,
		Valid:   true,
		Summary: summary,
	}, nil
}

// Name returns the planner name
func (p *DefaultPlanner) Name() string {
	return "default_planner"
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

// BuildCreateExample creates a basic example for a create object
func BuildCreateExample(objectType string, config map[string]interface{}) *core.ObjectExample {
	hcl := fmt.Sprintf(`create "%s" "example" {`, objectType)

	for key, value := range config {
		switch v := value.(type) {
		case string:
			hcl += fmt.Sprintf(`
  %s = "%s"`, key, v)
		case bool:
			hcl += fmt.Sprintf(`
  %s = %t`, key, v)
		case int, int64, float64:
			hcl += fmt.Sprintf(`
  %s = %v`, key, v)
		default:
			hcl += fmt.Sprintf(`
  %s = %v`, key, v)
		}
	}

	hcl += `
}`

	return &core.ObjectExample{
		Name:        fmt.Sprintf("%s_basic", objectType),
		Title:       fmt.Sprintf("Basic %s", strings.Title(objectType)),
		Description: fmt.Sprintf("Basic example of creating a %s resource", objectType),
		Category:    "basic",
		UseCase:     fmt.Sprintf("Create a simple %s", objectType),
		HCL:         hcl,
		Config:      config,
	}
}

// isValidType checks if a value matches the expected type string
func isValidType(value interface{}, expectedType string) bool {
	if value == nil {
		return true // nil is considered valid for any type
	}

	valueType := reflect.TypeOf(value)
	switch expectedType {
	case "string":
		return valueType.Kind() == reflect.String
	case "integer":
		return valueType.Kind() == reflect.Int || valueType.Kind() == reflect.Int64 || valueType.Kind() == reflect.Int32
	case "number":
		return valueType.Kind() == reflect.Float32 || valueType.Kind() == reflect.Float64 ||
			valueType.Kind() == reflect.Int || valueType.Kind() == reflect.Int64 || valueType.Kind() == reflect.Int32
	case "boolean":
		return valueType.Kind() == reflect.Bool
	case "list", "array":
		return valueType.Kind() == reflect.Slice || valueType.Kind() == reflect.Array
	case "object", "map":
		return valueType.Kind() == reflect.Map || valueType.Kind() == reflect.Struct
	default:
		return false
	}
}
