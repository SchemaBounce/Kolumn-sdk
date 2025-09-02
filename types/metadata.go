// Package types provides metadata type definitions for the Kolumn SDK
package types

import (
	"time"
)

// UniversalMetadata represents the universal metadata format for all providers
type UniversalMetadata struct {
	// Core Entity Information
	EntityID     string     `json:"entity_id"`
	EntityType   EntityType `json:"entity_type"`
	ProviderType string     `json:"provider_type"`
	ProviderName string     `json:"provider_name"`

	// Schema Information
	Schema        UniversalSchema   `json:"schema"`
	Columns       []UniversalColumn `json:"columns,omitempty"`
	Relationships []Relationship    `json:"relationships,omitempty"`

	// AI-Powered Intelligence
	Classifications []Classification `json:"classifications"`
	PIIDetection    PIIAnalysis      `json:"pii_detection"`
	Patterns        []PatternMatch   `json:"patterns"`
	Insights        AIInsights       `json:"insights"`

	// Lineage and Dependencies
	Lineage      LineageGraph `json:"lineage"`
	Dependencies []Dependency `json:"dependencies"`

	// Governance and Compliance
	Governance GovernanceMetadata `json:"governance"`
	Compliance ComplianceStatus   `json:"compliance"`

	// Performance and Usage
	Performance PerformanceMetrics `json:"performance"`
	Usage       UsageStatistics    `json:"usage"`

	// Temporal Information
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastAccessed time.Time `json:"last_accessed"`
}

// EntityType represents the type of entity
type EntityType string

const (
	EntityTypeTable      EntityType = "table"
	EntityTypeView       EntityType = "view"
	EntityTypeIndex      EntityType = "index"
	EntityTypeTopic      EntityType = "topic"
	EntityTypeBucket     EntityType = "bucket"
	EntityTypeWorkflow   EntityType = "workflow"
	EntityTypeJob        EntityType = "job"
	EntityTypeDataset    EntityType = "dataset"
	EntityTypeConnection EntityType = "connection"
	EntityTypeFunction   EntityType = "function"
	EntityTypeProcedure  EntityType = "procedure"
	EntityTypeStream     EntityType = "stream"
	EntityTypeQueue      EntityType = "queue"
	EntityTypeCache      EntityType = "cache"
	EntityTypeCollection EntityType = "collection"
)

// UniversalSchema represents a schema that works across all provider types
type UniversalSchema struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version,omitempty"`
	Fields      []UniversalField       `json:"fields"`
	Constraints []SchemaConstraint     `json:"constraints,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UniversalField represents a field/column that works across providers
type UniversalField struct {
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	Nullable        bool                   `json:"nullable"`
	Default         interface{}            `json:"default,omitempty"`
	Description     string                 `json:"description,omitempty"`
	Constraints     []FieldConstraint      `json:"constraints,omitempty"`
	Classifications []string               `json:"classifications,omitempty"`
	PIIAnalysis     PIIAnalysis            `json:"pii_analysis"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// UniversalColumn represents a column in universal format (alias for UniversalField)
type UniversalColumn = UniversalField

// SchemaConstraint represents schema-level constraints
type SchemaConstraint struct {
	Type       string                 `json:"type"`
	Name       string                 `json:"name,omitempty"`
	Fields     []string               `json:"fields"`
	Reference  *Reference             `json:"reference,omitempty"`
	Expression string                 `json:"expression,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// FieldConstraint represents field-level constraints
type FieldConstraint struct {
	Type       string      `json:"type"`
	Value      interface{} `json:"value,omitempty"`
	Expression string      `json:"expression,omitempty"`
}

// Reference represents a reference to another entity
type Reference struct {
	Schema string   `json:"schema,omitempty"`
	Table  string   `json:"table"`
	Fields []string `json:"fields"`
}

// Relationship represents a relationship between entities
type Relationship struct {
	Type        RelationshipType       `json:"type"`
	Source      EntityReference        `json:"source"`
	Target      EntityReference        `json:"target"`
	Cardinality string                 `json:"cardinality,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RelationshipType represents the type of relationship
type RelationshipType string

const (
	RelationshipTypeFK         RelationshipType = "foreign_key"
	RelationshipTypeReference  RelationshipType = "reference"
	RelationshipTypeLineage    RelationshipType = "lineage"
	RelationshipTypeDerivation RelationshipType = "derivation"
	RelationshipTypeDependency RelationshipType = "dependency"
)

// EntityReference represents a reference to an entity
type EntityReference struct {
	EntityID   string   `json:"entity_id"`
	EntityType string   `json:"entity_type"`
	Fields     []string `json:"fields,omitempty"`
}

// Classification represents a data classification
type Classification struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Level      string                 `json:"level"`
	Category   string                 `json:"category"`
	Confidence float64                `json:"confidence"`
	Source     string                 `json:"source"`
	AppliedAt  time.Time              `json:"applied_at"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// PIIAnalysis represents PII detection analysis
type PIIAnalysis struct {
	HasPII     bool                   `json:"has_pii"`
	Confidence float64                `json:"confidence"`
	PIITypes   []PIIType              `json:"pii_types,omitempty"`
	DetectedAt time.Time              `json:"detected_at"`
	Method     string                 `json:"method"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// PIIType represents a type of PII
type PIIType struct {
	Type       string  `json:"type"`
	Confidence float64 `json:"confidence"`
	Pattern    string  `json:"pattern,omitempty"`
}

// PatternMatch represents a detected pattern
type PatternMatch struct {
	Pattern    string                 `json:"pattern"`
	Type       string                 `json:"type"`
	Confidence float64                `json:"confidence"`
	Examples   []string               `json:"examples,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// AIInsights represents AI-powered insights
type AIInsights struct {
	Quality      QualityInsights      `json:"quality"`
	Usage        UsageInsights        `json:"usage"`
	Optimization OptimizationInsights `json:"optimization"`
	Anomalies    []AnomalyInsight     `json:"anomalies,omitempty"`
	GeneratedAt  time.Time            `json:"generated_at"`
}

// QualityInsights represents data quality insights
type QualityInsights struct {
	Score       float64                `json:"score"`
	Issues      []QualityIssue         `json:"issues,omitempty"`
	Suggestions []QualitySuggestion    `json:"suggestions,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// QualityIssue represents a data quality issue
type QualityIssue struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Field       string  `json:"field,omitempty"`
	Description string  `json:"description"`
	Count       int64   `json:"count,omitempty"`
	Percentage  float64 `json:"percentage,omitempty"`
}

// QualitySuggestion represents a data quality improvement suggestion
type QualitySuggestion struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Effort      string `json:"effort"`
	Priority    int    `json:"priority"`
}

// UsageInsights represents usage pattern insights
type UsageInsights struct {
	AccessPatterns []AccessPattern        `json:"access_patterns"`
	HotSpots       []HotSpot              `json:"hot_spots"`
	Trends         []UsageTrend           `json:"trends"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// AccessPattern represents an access pattern
type AccessPattern struct {
	Pattern   string    `json:"pattern"`
	Frequency int64     `json:"frequency"`
	Users     []string  `json:"users,omitempty"`
	TimeRange TimeRange `json:"time_range"`
}

// HotSpot represents a frequently accessed area
type HotSpot struct {
	Entity      string    `json:"entity"`
	Field       string    `json:"field,omitempty"`
	AccessCount int64     `json:"access_count"`
	LastAccess  time.Time `json:"last_access"`
}

// UsageTrend represents a usage trend
type UsageTrend struct {
	Metric    string                 `json:"metric"`
	Direction string                 `json:"direction"`
	Change    float64                `json:"change"`
	Period    string                 `json:"period"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// OptimizationInsights represents optimization recommendations
type OptimizationInsights struct {
	Recommendations  []OptimizationRecommendation `json:"recommendations"`
	PotentialSavings float64                      `json:"potential_savings"`
	Metadata         map[string]interface{}       `json:"metadata,omitempty"`
}

// OptimizationRecommendation represents an optimization recommendation
type OptimizationRecommendation struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"`
	Effort      string  `json:"effort"`
	Savings     float64 `json:"savings,omitempty"`
	Priority    int     `json:"priority"`
}

// AnomalyInsight represents an anomaly detection result
type AnomalyInsight struct {
	Type        string      `json:"type"`
	Severity    string      `json:"severity"`
	Description string      `json:"description"`
	Field       string      `json:"field,omitempty"`
	Value       interface{} `json:"value,omitempty"`
	Threshold   interface{} `json:"threshold,omitempty"`
	DetectedAt  time.Time   `json:"detected_at"`
	Confidence  float64     `json:"confidence"`
}

// LineageGraph represents data lineage information
type LineageGraph struct {
	Nodes    []LineageNode          `json:"nodes"`
	Edges    []LineageEdge          `json:"edges"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// LineageNode represents a node in the lineage graph
type LineageNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Name     string                 `json:"name"`
	Provider string                 `json:"provider"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// LineageEdge represents an edge in the lineage graph
type LineageEdge struct {
	Source       string                 `json:"source"`
	Target       string                 `json:"target"`
	Relationship string                 `json:"relationship"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// GovernanceMetadata represents governance-related metadata
type GovernanceMetadata struct {
	Owner            string                 `json:"owner,omitempty"`
	Steward          string                 `json:"steward,omitempty"`
	BusinessGlossary []GlossaryTerm         `json:"business_glossary,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	Policies         []string               `json:"policies,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// GlossaryTerm represents a business glossary term
type GlossaryTerm struct {
	Term       string `json:"term"`
	Definition string `json:"definition"`
	Category   string `json:"category,omitempty"`
	Source     string `json:"source,omitempty"`
}

// PerformanceMetrics represents performance-related metrics
type PerformanceMetrics struct {
	ResponseTime  time.Duration          `json:"response_time,omitempty"`
	Throughput    float64                `json:"throughput,omitempty"`
	ErrorRate     float64                `json:"error_rate,omitempty"`
	ResourceUsage ResourceUsage          `json:"resource_usage,omitempty"`
	LastMeasured  time.Time              `json:"last_measured"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceUsage represents resource usage metrics
type ResourceUsage struct {
	CPU     float64            `json:"cpu,omitempty"`
	Memory  float64            `json:"memory,omitempty"`
	Storage float64            `json:"storage,omitempty"`
	Network float64            `json:"network,omitempty"`
	Custom  map[string]float64 `json:"custom,omitempty"`
}

// UsageStatistics represents usage statistics
type UsageStatistics struct {
	AccessCount      int64                  `json:"access_count"`
	UniqueUsers      int64                  `json:"unique_users"`
	LastAccessed     time.Time              `json:"last_accessed"`
	AccessFrequency  string                 `json:"access_frequency"`
	TopUsers         []UserUsage            `json:"top_users,omitempty"`
	TimeDistribution map[string]int64       `json:"time_distribution,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// UserUsage represents usage by a specific user
type UserUsage struct {
	User        string    `json:"user"`
	AccessCount int64     `json:"access_count"`
	LastAccess  time.Time `json:"last_access"`
}

// TimeRange represents a time range
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}
