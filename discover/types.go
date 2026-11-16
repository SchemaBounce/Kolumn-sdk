// Package discover provides types and interfaces for discover object handlers
package discover

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/schemabounce/kolumn/sdk/core"
)

// =============================================================================
// EXTENSION INTERFACES FOR DISCOVER OPERATIONS
// =============================================================================

// SpecializedScanner extends Scanner with specialized scanning capabilities
type SpecializedScanner interface {
	Scanner

	// SupportsScanType checks if this scanner supports a specific scan type
	SupportsScanType(scanType string) bool

	// SpecializedScan performs specialized scans (security, performance, etc.)
	SpecializedScan(ctx context.Context, req *core.ScanRequest) ([]core.ScanResult, error)
}

// MetricsCollector provides resource metrics collection
type MetricsCollector interface {
	// CollectMetrics gathers performance and usage metrics
	CollectMetrics(ctx context.Context, req *core.MetricsRequest) (map[string]interface{}, error)

	// Name returns the collector name for debugging
	Name() string
}

// =============================================================================
// ADDITIONAL REQUEST/RESPONSE TYPES FOR DISCOVER OPERATIONS
// =============================================================================

// RelationsRequest represents a request to analyze resource relationships
type RelationsRequest struct {
	ObjectType string           `json:"object_type"`
	ResourceID string           `json:"resource_id"`
	Options    *RelationOptions `json:"options,omitempty"`
}

// RelationOptions provides options for relationship analysis
type RelationOptions struct {
	MaxDepth      int      `json:"max_depth,omitempty"`      // maximum relationship depth to explore
	RelationTypes []string `json:"relation_types,omitempty"` // specific types of relationships to find
	IncludeWeak   bool     `json:"include_weak"`             // include weak/indirect relationships
	Direction     string   `json:"direction,omitempty"`      // incoming, outgoing, both
}

// RelationsResponse represents the result of relationship analysis
type RelationsResponse struct {
	Relations    []core.ResourceReference `json:"relations"`
	RelationTree *RelationTree            `json:"relation_tree,omitempty"` // hierarchical view
	Summary      *RelationSummary         `json:"summary,omitempty"`
}

// RelationTree represents hierarchical resource relationships
type RelationTree struct {
	Resource *core.ResourceReference `json:"resource"`
	Children []*RelationTree         `json:"children,omitempty"`
	Parents  []*RelationTree         `json:"parents,omitempty"`
	Metadata map[string]interface{}  `json:"metadata,omitempty"`
}

// RelationSummary provides statistics about discovered relationships
type RelationSummary struct {
	TotalRelations int            `json:"total_relations"`
	ByType         map[string]int `json:"by_type"`
	ByDirection    map[string]int `json:"by_direction"`
	MaxDepth       int            `json:"max_depth"`
	CircularRefs   int            `json:"circular_refs"`
}

// MetadataRequest represents a request for additional resource metadata
type MetadataRequest struct {
	ObjectType    string   `json:"object_type"`
	ResourceID    string   `json:"resource_id"`
	MetadataTypes []string `json:"metadata_types,omitempty"` // specific types of metadata to retrieve
}

// MetadataResponse represents the result of metadata retrieval
type MetadataResponse struct {
	Metadata    map[string]interface{} `json:"metadata"`
	LastUpdated time.Time              `json:"last_updated,omitempty"`
	Source      string                 `json:"source,omitempty"` // source of metadata
}

// MetricsRequest represents a request for resource metrics
type MetricsRequest struct {
	ObjectType  string     `json:"object_type"`
	ResourceID  string     `json:"resource_id"`
	MetricTypes []string   `json:"metric_types,omitempty"` // specific metrics to collect
	TimeRange   *TimeRange `json:"time_range,omitempty"`   // time range for historical metrics
}

// TimeRange represents a time range for metrics collection
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// MetricsResponse represents the result of metrics collection
type MetricsResponse struct {
	Metrics     map[string]interface{} `json:"metrics"`
	CollectedAt time.Time              `json:"collected_at"`
	TimeRange   *TimeRange             `json:"time_range,omitempty"`
}

// =============================================================================
// BUILT-IN SCANNERS
// =============================================================================

// PatternScanner scans for resources matching specific patterns
type PatternScanner struct {
	ObjectType string
	Patterns   []string // patterns to match (simple glob patterns)
	ScanFunc   func(ctx context.Context, patterns []string) ([]core.DiscoveredResource, error)
}

// NewPatternScanner creates a pattern-based scanner
func NewPatternScanner(objectType string, patterns []string, scanFunc func(ctx context.Context, patterns []string) ([]core.DiscoveredResource, error)) *PatternScanner {
	return &PatternScanner{
		ObjectType: objectType,
		Patterns:   patterns,
		ScanFunc:   scanFunc,
	}
}

// Scan performs pattern-based scanning
func (s *PatternScanner) Scan(ctx context.Context, req *core.DiscoverRequest) ([]core.DiscoveredResource, error) {
	if s.ScanFunc != nil {
		return s.ScanFunc(ctx, s.Patterns)
	}
	return []core.DiscoveredResource{}, nil
}

// Name returns the scanner name
func (s *PatternScanner) Name() string {
	return fmt.Sprintf("pattern_%s_scanner", s.ObjectType)
}

// TagScanner scans for resources with specific tags or labels
type TagScanner struct {
	ObjectType string
	Tags       map[string]string // tags to match
	ScanFunc   func(ctx context.Context, tags map[string]string) ([]core.DiscoveredResource, error)
}

// NewTagScanner creates a tag-based scanner
func NewTagScanner(objectType string, tags map[string]string, scanFunc func(ctx context.Context, tags map[string]string) ([]core.DiscoveredResource, error)) *TagScanner {
	return &TagScanner{
		ObjectType: objectType,
		Tags:       tags,
		ScanFunc:   scanFunc,
	}
}

// Scan performs tag-based scanning
func (s *TagScanner) Scan(ctx context.Context, req *core.DiscoverRequest) ([]core.DiscoveredResource, error) {
	if s.ScanFunc != nil {
		return s.ScanFunc(ctx, s.Tags)
	}
	return []core.DiscoveredResource{}, nil
}

// Name returns the scanner name
func (s *TagScanner) Name() string {
	return fmt.Sprintf("tag_%s_scanner", s.ObjectType)
}

// =============================================================================
// BUILT-IN FILTERS
// =============================================================================

// TypeFilter filters resources by object type
type TypeFilter struct {
	ObjectTypes []string
	Include     bool // true to include matches, false to exclude
}

// NewTypeFilter creates a type-based filter
func NewTypeFilter(objectTypes []string, include bool) *TypeFilter {
	return &TypeFilter{
		ObjectTypes: objectTypes,
		Include:     include,
	}
}

// Filter applies type filtering to resources
func (f *TypeFilter) Filter(resources []core.DiscoveredResource) []core.DiscoveredResource {
	var filtered []core.DiscoveredResource

	for _, resource := range resources {
		matches := false

		for _, objectType := range f.ObjectTypes {
			if resource.ObjectType == objectType {
				matches = true
				break
			}
		}

		if (f.Include && matches) || (!f.Include && !matches) {
			filtered = append(filtered, resource)
		}
	}

	return filtered
}

// Name returns the filter name
func (f *TypeFilter) Name() string {
	return "type_filter"
}

// =============================================================================
// BUILT-IN ENRICHERS
// =============================================================================

// RelationEnricher adds relationship information to discovered resources
type RelationEnricher struct {
	RelationAnalyzers []RelationAnalyzer
}

// NewRelationEnricher creates an enricher that analyzes relationships
func NewRelationEnricher(analyzers ...RelationAnalyzer) *RelationEnricher {
	return &RelationEnricher{
		RelationAnalyzers: analyzers,
	}
}

// Enrich adds relationship information to resources
func (e *RelationEnricher) Enrich(ctx context.Context, resources []core.DiscoveredResource) ([]core.DiscoveredResource, error) {
	enriched := make([]core.DiscoveredResource, len(resources))
	copy(enriched, resources)

	for i, resource := range enriched {
		var allRelations []core.ResourceReference

		// Analyze relationships using all analyzers
		for _, analyzer := range e.RelationAnalyzers {
			req := &core.RelationsRequest{
				ResourceType: resource.ObjectType,
				ResourceID:   resource.ResourceID,
			}

			relations, err := analyzer.AnalyzeRelations(ctx, req)
			if err != nil {
				continue // Skip failed analyzers
			}

			allRelations = append(allRelations, relations...)
		}

		// Update resource with discovered relationships
		resource.Dependencies = []core.ResourceReference{}
		resource.Dependents = []core.ResourceReference{}

		for _, relation := range allRelations {
			switch relation.RelationType {
			case "depends_on", "references":
				resource.Dependencies = append(resource.Dependencies, relation)
			case "used_by", "referenced_by":
				resource.Dependents = append(resource.Dependents, relation)
			}
		}

		enriched[i] = resource
	}

	return enriched, nil
}

// Name returns the enricher name
func (e *RelationEnricher) Name() string {
	return "relation_enricher"
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

// BuildScanExample creates a scan example for a discover object
func BuildScanExample(objectType, scanType string) *core.ObjectExample {
	hcl := fmt.Sprintf(`discover "%s" "scan" {
  scan_type = "%s"
  
  # Scan options
  options = {
    include_metrics = true
    include_security = true
    depth = 2
  }
  
  # Filters
  filters = {
    name_pattern = "*prod*"
    severity = ["high", "critical"]
  }
}`, objectType, scanType)

	return &core.ObjectExample{
		Name:        fmt.Sprintf("%s_%s_scan", objectType, scanType),
		Title:       fmt.Sprintf("%s %s Scan", strings.Title(scanType), strings.Title(objectType)),
		Description: fmt.Sprintf("Example of scanning %s resources for %s issues", objectType, scanType),
		Category:    "advanced",
		UseCase:     fmt.Sprintf("Security and compliance scanning of %s resources", objectType),
		HCL:         hcl,
		Config: map[string]interface{}{
			"scan_type": scanType,
			"options": map[string]interface{}{
				"include_metrics":  true,
				"include_security": true,
				"depth":            2,
			},
			"filters": map[string]interface{}{
				"name_pattern": "*prod*",
				"severity":     []string{"high", "critical"},
			},
		},
	}
}

// NewDiscoveryContext creates a context with discovery-specific values
func NewDiscoveryContext(ctx context.Context, maxDuration time.Duration) (context.Context, context.CancelFunc) {
	if maxDuration > 0 {
		return context.WithTimeout(ctx, maxDuration)
	}
	return ctx, func() {} // Return no-op cancel function
}
