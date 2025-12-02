// Package core provides discovery types for database introspection
package core

import "time"

// =============================================================================
// DATABASE DISCOVERY REQUEST/RESPONSE TYPES
// =============================================================================

// DiscoveryRequest contains parameters for database discovery
type DiscoveryRequest struct {
	Schemas       []string `json:"schemas,omitempty"`      // Optional: limit to specific schemas
	ObjectTypes   []string `json:"object_types,omitempty"` // Optional: limit to specific types (table, view, index, etc.)
	IncludeSystem bool     `json:"include_system"`         // Include system objects (pg_*, information_schema, etc.)
	SampleData    bool     `json:"sample_data"`            // Include sample data for analysis
	MaxObjects    int      `json:"max_objects,omitempty"`  // Limit number of objects (0 = no limit)
}

// ObjectStatistics contains size and usage statistics for an object
type ObjectStatistics struct {
	RowCount       int64      `json:"row_count,omitempty"`
	SizeBytes      int64      `json:"size_bytes,omitempty"`
	IndexSizeBytes int64      `json:"index_size_bytes,omitempty"`
	LastAnalyzed   *time.Time `json:"last_analyzed,omitempty"`
	LastModified   *time.Time `json:"last_modified,omitempty"`
}

// DiscoveredObject represents a single discovered database object
type DiscoveredObject struct {
	Type         string                 `json:"type"`                   // "table", "view", "index", "function", etc.
	Schema       string                 `json:"schema"`                 // Schema/database name
	Name         string                 `json:"name"`                   // Object name
	FullName     string                 `json:"full_name"`              // schema.name
	Definition   map[string]interface{} `json:"definition"`             // Full object definition
	Dependencies []string               `json:"dependencies,omitempty"` // Object dependencies (foreign keys, etc.)
	Statistics   *ObjectStatistics      `json:"statistics,omitempty"`
	Metadata     map[string]string      `json:"metadata,omitempty"`
}

// DiscoveryStatistics contains summary statistics from discovery
type DiscoveryStatistics struct {
	TotalObjects   int            `json:"total_objects"`
	ObjectCounts   map[string]int `json:"object_counts"` // By type
	TotalSizeBytes int64          `json:"total_size_bytes"`
	SchemasCovered []string       `json:"schemas_covered"`
}

// DiscoveryResult contains full database discovery results
type DiscoveryResult struct {
	ProviderType string              `json:"provider_type"`
	DatabaseName string              `json:"database_name"`
	Objects      []DiscoveredObject  `json:"objects"`
	Statistics   DiscoveryStatistics `json:"statistics"`
	Timestamp    time.Time           `json:"timestamp"`
	Duration     time.Duration       `json:"duration"`
}
