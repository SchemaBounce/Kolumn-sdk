// Package metadata provides metadata collection interfaces for the Kolumn SDK
package metadata

import (
	"context"

	"github.com/schemabounce/kolumn/sdk/types"
)

// MetadataCollector defines the interface for collecting metadata from providers
type MetadataCollector interface {
	// Collect collects metadata from the provider
	Collect(ctx context.Context) (*types.UniversalMetadata, error)

	// CollectEntity collects metadata for a specific entity
	CollectEntity(ctx context.Context, entityID string) (*types.UniversalMetadata, error)

	// CollectIncremental collects metadata changes since the last collection
	CollectIncremental(ctx context.Context, since string) ([]*types.UniversalMetadata, error)

	// GetSupportedEntityTypes returns the entity types this collector supports
	GetSupportedEntityTypes() []types.EntityType

	// ValidateConnection validates the connection to the metadata source
	ValidateConnection(ctx context.Context) error
}

// MetadataRegistry manages metadata collections across providers
type MetadataRegistry interface {
	// RegisterCollector registers a metadata collector for a provider
	RegisterCollector(providerType string, collector MetadataCollector) error

	// GetCollector returns the metadata collector for a provider
	GetCollector(providerType string) (MetadataCollector, error)

	// CollectAll collects metadata from all registered collectors
	CollectAll(ctx context.Context) (map[string]*types.UniversalMetadata, error)

	// Search searches metadata across all collectors
	Search(ctx context.Context, query *MetadataQuery) ([]*types.UniversalMetadata, error)

	// Index indexes metadata for fast searching
	Index(ctx context.Context, metadata *types.UniversalMetadata) error

	// GetLineage gets lineage information for an entity
	GetLineage(ctx context.Context, entityID string) (*types.LineageGraph, error)
}

// MetadataQuery represents a metadata search query
type MetadataQuery struct {
	// Text search
	Query string `json:"query,omitempty"`

	// Filters
	ProviderTypes   []string           `json:"provider_types,omitempty"`
	EntityTypes     []types.EntityType `json:"entity_types,omitempty"`
	Classifications []string           `json:"classifications,omitempty"`
	Tags            []string           `json:"tags,omitempty"`

	// PII filtering
	HasPII   *bool    `json:"has_pii,omitempty"`
	PIITypes []string `json:"pii_types,omitempty"`

	// Time-based filtering
	CreatedAfter  string `json:"created_after,omitempty"`
	CreatedBefore string `json:"created_before,omitempty"`
	UpdatedAfter  string `json:"updated_after,omitempty"`
	UpdatedBefore string `json:"updated_before,omitempty"`

	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`

	// Sorting
	SortBy    string    `json:"sort_by,omitempty"`
	SortOrder SortOrder `json:"sort_order,omitempty"`

	// Advanced filters
	CustomFilters map[string]interface{} `json:"custom_filters,omitempty"`
}

// SortOrder represents sort order
type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// SchemaRegistry manages schema information across providers
type SchemaRegistry interface {
	// RegisterSchema registers a schema for a provider entity
	RegisterSchema(ctx context.Context, providerType, entityName string, schema *types.UniversalSchema) error

	// GetSchema gets the schema for a provider entity
	GetSchema(ctx context.Context, providerType, entityName string) (*types.UniversalSchema, error)

	// GetLatestSchema gets the latest version of a schema
	GetLatestSchema(ctx context.Context, providerType, entityName string) (*types.UniversalSchema, error)

	// ListSchemas lists all schemas for a provider
	ListSchemas(ctx context.Context, providerType string) ([]SchemaInfo, error)

	// ValidateSchema validates a schema against the registry
	ValidateSchema(ctx context.Context, schema *types.UniversalSchema) (*ValidationResult, error)

	// CheckCompatibility checks if two schemas are compatible
	CheckCompatibility(ctx context.Context, oldSchema, newSchema *types.UniversalSchema) (*CompatibilityResult, error)

	// GetSchemaEvolution gets the evolution history of a schema
	GetSchemaEvolution(ctx context.Context, providerType, entityName string) (*SchemaEvolution, error)
}

// SchemaInfo contains information about a schema
type SchemaInfo struct {
	ProviderType string `json:"provider_type"`
	EntityName   string `json:"entity_name"`
	Version      string `json:"version"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// ValidationResult contains schema validation results
type ValidationResult struct {
	Valid    bool                `json:"valid"`
	Errors   []ValidationError   `json:"errors,omitempty"`
	Warnings []ValidationWarning `json:"warnings,omitempty"`
}

// ValidationError represents a schema validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ValidationWarning represents a schema validation warning
type ValidationWarning struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// CompatibilityResult contains schema compatibility check results
type CompatibilityResult struct {
	Compatible      bool                 `json:"compatible"`
	Issues          []CompatibilityIssue `json:"issues,omitempty"`
	Changes         []SchemaChange       `json:"changes,omitempty"`
	BreakingChanges bool                 `json:"breaking_changes"`
}

// CompatibilityIssue represents a schema compatibility issue
type CompatibilityIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Field       string `json:"field,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// SchemaChange represents a change between schemas
type SchemaChange struct {
	Type        ChangeType  `json:"type"`
	Field       string      `json:"field"`
	OldValue    interface{} `json:"old_value,omitempty"`
	NewValue    interface{} `json:"new_value,omitempty"`
	Breaking    bool        `json:"breaking"`
	Description string      `json:"description"`
}

// ChangeType represents the type of schema change
type ChangeType string

const (
	ChangeTypeAdd    ChangeType = "add"
	ChangeTypeRemove ChangeType = "remove"
	ChangeTypeModify ChangeType = "modify"
	ChangeTypeRename ChangeType = "rename"
)

// SchemaEvolution contains schema evolution history
type SchemaEvolution struct {
	ProviderType string                 `json:"provider_type"`
	EntityName   string                 `json:"entity_name"`
	Versions     []SchemaVersion        `json:"versions"`
	Timeline     []EvolutionEvent       `json:"timeline"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// SchemaVersion represents a version in schema evolution
type SchemaVersion struct {
	Version   string                 `json:"version"`
	Schema    *types.UniversalSchema `json:"schema"`
	CreatedAt string                 `json:"created_at"`
	CreatedBy string                 `json:"created_by,omitempty"`
	Changes   []SchemaChange         `json:"changes,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// EvolutionEvent represents an event in schema evolution
type EvolutionEvent struct {
	Type        string                 `json:"type"`
	Version     string                 `json:"version"`
	Timestamp   string                 `json:"timestamp"`
	Actor       string                 `json:"actor,omitempty"`
	Description string                 `json:"description"`
	Changes     []SchemaChange         `json:"changes,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// LineageTracker tracks data lineage across providers
type LineageTracker interface {
	// TrackLineage tracks lineage between entities
	TrackLineage(ctx context.Context, source, target *types.UniversalMetadata, relationship types.RelationshipType) error

	// GetDownstreamLineage gets downstream lineage for an entity
	GetDownstreamLineage(ctx context.Context, entityID string) (*types.LineageGraph, error)

	// GetUpstreamLineage gets upstream lineage for an entity
	GetUpstreamLineage(ctx context.Context, entityID string) (*types.LineageGraph, error)

	// GetFullLineage gets complete lineage graph for an entity
	GetFullLineage(ctx context.Context, entityID string) (*types.LineageGraph, error)

	// AnalyzeImpact analyzes the impact of changes to an entity
	AnalyzeImpact(ctx context.Context, entityID string, changeType string) (*ImpactAnalysis, error)

	// UpdateLineage updates lineage information
	UpdateLineage(ctx context.Context, lineage *types.LineageGraph) error
}

// ImpactAnalysis contains impact analysis results
type ImpactAnalysis struct {
	EntityID         string                 `json:"entity_id"`
	ChangeType       string                 `json:"change_type"`
	ImpactedEntities []ImpactedEntity       `json:"impacted_entities"`
	Risk             RiskLevel              `json:"risk"`
	Recommendations  []string               `json:"recommendations"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ImpactedEntity represents an entity impacted by a change
type ImpactedEntity struct {
	EntityID    string  `json:"entity_id"`
	EntityType  string  `json:"entity_type"`
	ImpactType  string  `json:"impact_type"`
	Severity    string  `json:"severity"`
	Probability float64 `json:"probability"`
	Description string  `json:"description"`
}

// RiskLevel represents the risk level of a change
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)
