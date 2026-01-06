package logging

import (
	"testing"

	"github.com/schemabounce/kolumn/sdk/core"
)

func TestStripTemplateContext(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
		{
			name: "no template context",
			input: map[string]interface{}{
				"table":  "users",
				"values": map[string]interface{}{"id": 1},
			},
			expected: map[string]interface{}{
				"table":  "users",
				"values": map[string]interface{}{"id": 1},
			},
		},
		{
			name: "with template context",
			input: map[string]interface{}{
				"table":             "users",
				"values":            map[string]interface{}{"id": 1},
				"_template_context": map[string]interface{}{"source": "seeds/users.csv"},
			},
			expected: map[string]interface{}{
				"table":  "users",
				"values": map[string]interface{}{"id": 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripTemplateContext(tt.input)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d keys, got %d", len(tt.expected), len(result))
			}
			if _, hasTemplateContext := result["_template_context"]; hasTemplateContext {
				t.Error("_template_context should be stripped")
			}
		})
	}
}

func TestBuildResourceSummary(t *testing.T) {
	resource := core.PlanResource{
		ResourceType: "postgres_table",
		Name:         "users",
		Config: map[string]interface{}{
			"schema":            "public",
			"_template_context": map[string]interface{}{"source": "main.kl"},
		},
		Action: "create",
	}

	summary := BuildResourceSummary(resource)

	if summary.ResourceType != "postgres_table" {
		t.Errorf("expected resource_type 'postgres_table', got '%s'", summary.ResourceType)
	}
	if summary.Name != "users" {
		t.Errorf("expected name 'users', got '%s'", summary.Name)
	}
	if summary.Action != "create" {
		t.Errorf("expected action 'create', got '%s'", summary.Action)
	}
	if _, hasTemplateContext := summary.ConfigSnapshot["_template_context"]; hasTemplateContext {
		t.Error("_template_context should be stripped from config_snapshot")
	}
}

func TestBuildResourceSummaryWithNOOP(t *testing.T) {
	resource := core.PlanResource{
		ResourceType: "postgres_insert",
		Name:         "seed_admin",
		Config: map[string]interface{}{
			"table":  "users",
			"values": map[string]interface{}{"id": 1, "email": "admin@example.com"},
		},
		Action: "create",
	}

	// Test when row does NOT exist
	summary := BuildResourceSummaryWithNOOP(resource, false, "")
	if summary.Action != "create" {
		t.Errorf("expected action 'create' when row doesn't exist, got '%s'", summary.Action)
	}

	// Test when row DOES exist
	summary = BuildResourceSummaryWithNOOP(resource, true, "row already exists")
	if summary.Action != "noop" {
		t.Errorf("expected action 'noop' when row exists, got '%s'", summary.Action)
	}
	if summary.Reason != "row already exists" {
		t.Errorf("expected reason 'row already exists', got '%s'", summary.Reason)
	}
}

func TestIsInsertResourceType(t *testing.T) {
	tests := []struct {
		resourceType string
		expected     bool
	}{
		{"postgres_table", false},
		{"postgres_insert", true},
		{"mysql_insert", true},
		{"mongodb_collection", false},
		{"dynamodb_item_insert", true},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			result := IsInsertResourceType(tt.resourceType)
			if result != tt.expected {
				t.Errorf("IsInsertResourceType(%s) = %v, expected %v", tt.resourceType, result, tt.expected)
			}
		})
	}
}

func TestExtractInsertConfig(t *testing.T) {
	config := map[string]interface{}{
		"table": "users",
		"values": map[string]interface{}{
			"id":    1,
			"email": "test@example.com",
		},
		"unique_keys": []interface{}{"id", "email"},
	}

	tableName, values, uniqueKeys := ExtractInsertConfig(config)

	if tableName != "users" {
		t.Errorf("expected table 'users', got '%s'", tableName)
	}
	if values == nil {
		t.Error("expected values to not be nil")
	}
	if len(uniqueKeys) != 2 {
		t.Errorf("expected 2 unique keys, got %d", len(uniqueKeys))
	}
}

func TestBuildUniqueKeyFilter(t *testing.T) {
	values := map[string]interface{}{
		"id":         1,
		"email":      "test@example.com",
		"created_at": "2024-01-01",
	}
	uniqueKeys := []string{"id", "email"}

	filter := BuildUniqueKeyFilter(values, uniqueKeys)

	if len(filter) != 2 {
		t.Errorf("expected 2 filter keys, got %d", len(filter))
	}
	if filter["id"] != 1 {
		t.Errorf("expected filter[id] = 1, got %v", filter["id"])
	}
	if filter["email"] != "test@example.com" {
		t.Errorf("expected filter[email] = 'test@example.com', got %v", filter["email"])
	}
	if _, hasCreatedAt := filter["created_at"]; hasCreatedAt {
		t.Error("created_at should not be in filter")
	}
}

func TestBuildUniqueKeyFilterEmptyKeys(t *testing.T) {
	values := map[string]interface{}{"id": 1}

	filter := BuildUniqueKeyFilter(values, nil)
	if filter != nil {
		t.Error("expected nil filter for empty unique keys")
	}

	filter = BuildUniqueKeyFilter(values, []string{})
	if filter != nil {
		t.Error("expected nil filter for empty unique keys slice")
	}
}
