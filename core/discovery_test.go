package core

import (
	"testing"
	"time"
)

// TestDiscoveryRequest validates DiscoveryRequest structure
func TestDiscoveryRequest(t *testing.T) {
	req := &DiscoveryRequest{
		Schemas:       []string{"public", "analytics"},
		ObjectTypes:   []string{"table", "view"},
		IncludeSystem: false,
		SampleData:    true,
		MaxObjects:    100,
	}

	if len(req.Schemas) != 2 {
		t.Errorf("Expected 2 schemas, got %d", len(req.Schemas))
	}

	if req.MaxObjects != 100 {
		t.Errorf("Expected MaxObjects=100, got %d", req.MaxObjects)
	}
}

// TestDiscoveredObject validates DiscoveredObject structure
func TestDiscoveredObject(t *testing.T) {
	now := time.Now()
	obj := DiscoveredObject{
		Type:     "table",
		Schema:   "public",
		Name:     "users",
		FullName: "public.users",
		Definition: map[string]interface{}{
			"columns": []string{"id", "name", "email"},
		},
		Dependencies: []string{"public.roles"},
		Statistics: &ObjectStatistics{
			RowCount:       1000,
			SizeBytes:      524288,
			IndexSizeBytes: 131072,
			LastAnalyzed:   &now,
			LastModified:   &now,
		},
		Metadata: map[string]string{
			"owner": "postgres",
		},
	}

	if obj.Type != "table" {
		t.Errorf("Expected type=table, got %s", obj.Type)
	}

	if obj.Statistics.RowCount != 1000 {
		t.Errorf("Expected RowCount=1000, got %d", obj.Statistics.RowCount)
	}

	if obj.Statistics.SizeBytes != 524288 {
		t.Errorf("Expected SizeBytes=524288, got %d", obj.Statistics.SizeBytes)
	}
}

// TestDiscoveryResult validates DiscoveryResult structure
func TestDiscoveryResult(t *testing.T) {
	result := &DiscoveryResult{
		ProviderType: "postgres",
		DatabaseName: "mydb",
		Objects: []DiscoveredObject{
			{
				Type:     "table",
				Schema:   "public",
				Name:     "users",
				FullName: "public.users",
			},
			{
				Type:     "view",
				Schema:   "public",
				Name:     "user_stats",
				FullName: "public.user_stats",
			},
		},
		Statistics: DiscoveryStatistics{
			TotalObjects: 2,
			ObjectCounts: map[string]int{
				"table": 1,
				"view":  1,
			},
			TotalSizeBytes: 1048576,
			SchemasCovered: []string{"public"},
		},
		Timestamp: time.Now(),
		Duration:  5 * time.Second,
	}

	if result.ProviderType != "postgres" {
		t.Errorf("Expected ProviderType=postgres, got %s", result.ProviderType)
	}

	if len(result.Objects) != 2 {
		t.Errorf("Expected 2 objects, got %d", len(result.Objects))
	}

	if result.Statistics.TotalObjects != 2 {
		t.Errorf("Expected TotalObjects=2, got %d", result.Statistics.TotalObjects)
	}

	if result.Statistics.ObjectCounts["table"] != 1 {
		t.Errorf("Expected 1 table, got %d", result.Statistics.ObjectCounts["table"])
	}
}

// TestDiscoveryHelper validates DiscoveryHelper functionality
func TestDiscoveryHelper(t *testing.T) {
	helper := NewDiscoveryHelper()

	objects := []DiscoveredObject{
		{Type: "table", Schema: "public", Name: "users", Statistics: &ObjectStatistics{SizeBytes: 1024}},
		{Type: "table", Schema: "analytics", Name: "events", Statistics: &ObjectStatistics{SizeBytes: 2048}},
		{Type: "view", Schema: "public", Name: "user_stats"},
		{Type: "table", Schema: "pg_catalog", Name: "pg_tables"}, // System object
	}

	// Test filtering by object type (includes system tables by default since IncludeSystem is false by default)
	req := &DiscoveryRequest{
		ObjectTypes:   []string{"table"},
		IncludeSystem: true, // Explicitly include system objects
	}
	filtered := helper.FilterObjects(objects, req)
	if len(filtered) != 3 { // 3 tables (including system)
		t.Errorf("Expected 3 tables, got %d", len(filtered))
	}

	// Test filtering out system objects
	req = &DiscoveryRequest{
		IncludeSystem: false,
	}
	filtered = helper.FilterObjects(objects, req)
	if len(filtered) != 3 { // Excludes pg_catalog.pg_tables
		t.Errorf("Expected 3 non-system objects, got %d", len(filtered))
	}

	// Test filtering by schema
	req = &DiscoveryRequest{
		Schemas: []string{"public"},
	}
	filtered = helper.FilterObjects(objects, req)
	if len(filtered) != 2 { // users table and user_stats view
		t.Errorf("Expected 2 public schema objects, got %d", len(filtered))
	}

	// Test max objects limit
	req = &DiscoveryRequest{
		MaxObjects: 2,
	}
	filtered = helper.FilterObjects(objects, req)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 objects (max limit), got %d", len(filtered))
	}

	// Test statistics building
	stats := helper.BuildStatistics(objects)
	if stats.TotalObjects != 4 {
		t.Errorf("Expected TotalObjects=4, got %d", stats.TotalObjects)
	}
	if stats.ObjectCounts["table"] != 3 {
		t.Errorf("Expected 3 tables, got %d", stats.ObjectCounts["table"])
	}
	if stats.ObjectCounts["view"] != 1 {
		t.Errorf("Expected 1 view, got %d", stats.ObjectCounts["view"])
	}
	if stats.TotalSizeBytes != 3072 { // 1024 + 2048
		t.Errorf("Expected TotalSizeBytes=3072, got %d", stats.TotalSizeBytes)
	}
	if len(stats.SchemasCovered) != 3 { // public, analytics, pg_catalog
		t.Errorf("Expected 3 schemas, got %d", len(stats.SchemasCovered))
	}
}

// TestIsSystemObject validates system object detection
func TestIsSystemObject(t *testing.T) {
	helper := NewDiscoveryHelper()

	tests := []struct {
		obj      DiscoveredObject
		expected bool
		name     string
	}{
		{
			obj:      DiscoveredObject{Schema: "pg_catalog", Name: "pg_tables"},
			expected: true,
			name:     "PostgreSQL system schema",
		},
		{
			obj:      DiscoveredObject{Schema: "information_schema", Name: "tables"},
			expected: true,
			name:     "Information schema",
		},
		{
			obj:      DiscoveredObject{Schema: "mysql", Name: "user"},
			expected: true,
			name:     "MySQL system schema",
		},
		{
			obj:      DiscoveredObject{Schema: "public", Name: "pg_stat_statements"},
			expected: true,
			name:     "System object by name prefix",
		},
		{
			obj:      DiscoveredObject{Schema: "public", Name: "users"},
			expected: false,
			name:     "Regular user table",
		},
		{
			obj:      DiscoveredObject{Schema: "analytics", Name: "events"},
			expected: false,
			name:     "Regular analytics table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helper.isSystemObject(tt.obj)
			if result != tt.expected {
				t.Errorf("isSystemObject(%v) = %v, expected %v", tt.obj, result, tt.expected)
			}
		})
	}
}
