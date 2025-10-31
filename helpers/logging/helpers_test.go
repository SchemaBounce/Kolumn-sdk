package logging

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestJSONToHuman(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		context  string
		expected string
	}{
		{
			name:     "nil input",
			input:    nil,
			context:  "test",
			expected: "test: <nil>",
		},
		{
			name:     "simple string",
			input:    "hello world",
			context:  "message",
			expected: "message: hello world",
		},
		{
			name:     "simple map",
			input:    map[string]interface{}{"name": "test", "value": 123},
			context:  "data",
			expected: "data: name=test value=123",
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			context:  "empty",
			expected: "empty: <empty>",
		},
		{
			name:     "simple array",
			input:    []interface{}{"a", "b", "c"},
			context:  "list",
			expected: "list: [0]=a [1]=b [2]=c",
		},
		{
			name:     "empty array",
			input:    []interface{}{},
			context:  "empty_list",
			expected: "empty_list: <empty array>",
		},
		{
			name:     "large array",
			input:    []interface{}{"a", "b", "c", "d", "e"},
			context:  "big_list",
			expected: "big_list: [5 items]",
		},
		{
			name:     "nested map",
			input:    map[string]interface{}{"data": map[string]interface{}{"nested": "value"}},
			context:  "nested",
			expected: "nested: data={1 fields}",
		},
		{
			name:     "long string",
			input:    map[string]interface{}{"description": strings.Repeat("A", 60)},
			context:  "long",
			expected: "long: description=" + strings.Repeat("A", 47) + "...",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := JSONToHuman(test.input, test.context)
			if result != test.expected {
				t.Errorf("Expected %q, got %q", test.expected, result)
			}
		})
	}
}

func TestJSONToHumanWithSensitiveData(t *testing.T) {
	sensitiveData := map[string]interface{}{
		"username": "admin",
		"password": "secret123",
		"api_key":  "key_abc123",
		"token":    "bearer_xyz789",
		"database": "production",
	}

	result := JSONToHuman(sensitiveData, "config")

	// Should contain non-sensitive data
	if !strings.Contains(result, "username=admin") {
		t.Error("Expected username to be visible")
	}
	if !strings.Contains(result, "database=production") {
		t.Error("Expected database to be visible")
	}

	// Should redact sensitive data
	if strings.Contains(result, "secret123") {
		t.Error("Password should be redacted")
	}
	if strings.Contains(result, "key_abc123") {
		t.Error("API key should be redacted")
	}
	if strings.Contains(result, "bearer_xyz789") {
		t.Error("Token should be redacted")
	}

	// Should show redaction markers
	if !strings.Contains(result, "password=<redacted>") {
		t.Error("Expected password redaction marker")
	}
}

func TestJSONStringParsing(t *testing.T) {
	// Test JSON string that should be parsed
	jsonString := `{"name": "test", "value": 123}`
	result := JSONToHuman(jsonString, "json_string")

	if !strings.Contains(result, "name=test") {
		t.Error("Expected JSON string to be parsed and formatted")
	}

	// Test non-JSON string
	regularString := "just a regular string"
	result2 := JSONToHuman(regularString, "regular")

	expected := "regular: just a regular string"
	if result2 != expected {
		t.Errorf("Expected %q, got %q", expected, result2)
	}
}

func TestSummarizeRequest(t *testing.T) {
	tests := []struct {
		name     string
		request  interface{}
		expected RequestSummary
	}{
		{
			name:    "nil request",
			request: nil,
			expected: RequestSummary{
				Type:   "unknown",
				Fields: map[string]interface{}{},
			},
		},
		{
			name: "map request with resource_type",
			request: map[string]interface{}{
				"resource_type": "table",
				"name":          "users",
				"schema":        "public",
			},
			expected: RequestSummary{
				Type:         "unknown",
				ResourceType: "table",
				ResourceName: "users",
				Fields: map[string]interface{}{
					"resource_type": "table",
					"name":          "users",
					"schema":        "public",
				},
			},
		},
		{
			name: "map request with object_type",
			request: map[string]interface{}{
				"object_type": "view",
				"name":        "user_view",
			},
			expected: RequestSummary{
				Type:         "unknown",
				ResourceType: "view",
				ResourceName: "user_view",
				Fields: map[string]interface{}{
					"object_type": "view",
					"name":        "user_view",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := SummarizeRequest(test.request)

			if result.ResourceType != test.expected.ResourceType {
				t.Errorf("Expected ResourceType %q, got %q", test.expected.ResourceType, result.ResourceType)
			}
			if result.ResourceName != test.expected.ResourceName {
				t.Errorf("Expected ResourceName %q, got %q", test.expected.ResourceName, result.ResourceName)
			}
		})
	}
}

func TestSummarizeResponse(t *testing.T) {
	tests := []struct {
		name     string
		response interface{}
		expected ResponseSummary
	}{
		{
			name:     "nil response",
			response: nil,
			expected: ResponseSummary{
				Success: true,
				Summary: "no response data",
			},
		},
		{
			name: "success response",
			response: map[string]interface{}{
				"success": true,
				"id":      "table_123",
			},
			expected: ResponseSummary{
				Success: true,
				Summary: "operation completed",
			},
		},
		{
			name: "error response",
			response: map[string]interface{}{
				"error": "table not found",
			},
			expected: ResponseSummary{
				Success: false,
				Error:   "table not found",
				Summary: "table not found",
			},
		},
		{
			name: "list response",
			response: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"name": "table1"},
					map[string]interface{}{"name": "table2"},
				},
			},
			expected: ResponseSummary{
				Success: true,
				Count:   2,
				Summary: "returned 2 items",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := SummarizeResponse(test.response)

			if result.Success != test.expected.Success {
				t.Errorf("Expected Success %v, got %v", test.expected.Success, result.Success)
			}
			if result.Count != test.expected.Count {
				t.Errorf("Expected Count %d, got %d", test.expected.Count, result.Count)
			}
			if result.Summary != test.expected.Summary {
				t.Errorf("Expected Summary %q, got %q", test.expected.Summary, result.Summary)
			}
		})
	}
}

func TestSanitizeEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected string
	}{
		{
			name:     "no credentials",
			endpoint: "postgresql://localhost:5432/mydb",
			expected: "postgresql://localhost:5432/mydb",
		},
		{
			name:     "with username only",
			endpoint: "postgresql://user@localhost:5432/mydb",
			expected: "postgresql://user@localhost:5432/mydb",
		},
		{
			name:     "with username and password",
			endpoint: "postgresql://user:password123@localhost:5432/mydb",
			expected: "postgresql://user:***@localhost:5432/mydb",
		},
		{
			name:     "complex password",
			endpoint: "mysql://admin:P@ssw0rd!@db.example.com:3306/production",
			expected: "mysql://admin:***@db.example.com:3306/production",
		},
		{
			name:     "redis with auth",
			endpoint: "redis://user:secret@cache.example.com:6379/0",
			expected: "redis://user:***@cache.example.com:6379/0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := SanitizeEndpoint(test.endpoint)
			if result != test.expected {
				t.Errorf("Expected %q, got %q", test.expected, result)
			}

			// Ensure original endpoint is not modified
			if test.endpoint != test.endpoint {
				t.Error("Original endpoint was modified")
			}
		})
	}
}

func TestLogRequest(t *testing.T) {
	logger, capture := NewTestLogger(t, "request", false)

	request := map[string]interface{}{
		"resource_type": "table",
		"name":          "users",
		"schema":        "public",
	}

	LogRequest(logger.Logger, "CreateTable", request)

	output := capture.GetOutput()
	if !strings.Contains(output, "CreateTable request for table 'users'") {
		t.Errorf("Expected human-readable request summary, got: %s", output)
	}

	// Test debug mode shows JSON
	logger2, capture2 := NewTestLogger(t, "request_debug", true)
	LogRequest(logger2.Logger, "CreateTable", request)

	output2 := capture2.GetOutput()
	if !strings.Contains(output2, "CreateTable request") {
		t.Error("Expected debug mode to show detailed request")
	}
}

func TestLogResponse(t *testing.T) {
	logger, capture := NewTestLogger(t, "response", false)

	// Test successful response
	response := map[string]interface{}{
		"success": true,
		"id":      "table_123",
	}

	LogResponse(logger.Logger, "CreateTable", response, nil)

	output := capture.GetOutput()
	if !strings.Contains(output, "CreateTable completed successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Test error response
	capture.Clear()
	err := fmt.Errorf("connection failed")
	LogResponse(logger.Logger, "CreateTable", nil, err)

	output2 := capture.GetOutput()
	if !strings.Contains(output2, "CreateTable failed: connection failed") {
		t.Errorf("Expected error message, got: %s", output2)
	}
}

func TestLogProviderOperation(t *testing.T) {
	logger, capture := NewTestLogger(t, "operation", false)

	context := ProviderContext{
		ProviderName: "postgres",
		Operation:    "CreateResource",
		ResourceType: "table",
		ResourceName: "users",
		StartTime:    time.Now(),
	}

	// Test successful operation
	err := LogProviderOperation(logger.Logger, context, func() error {
		time.Sleep(10 * time.Millisecond) // Simulate work
		return nil
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	lines := capture.GetLines()
	if len(lines) != 2 {
		t.Errorf("Expected 2 log lines (start and complete), got %d", len(lines))
	}

	capture.AssertContains(t, "Starting CreateResource operation on table 'users'")
	capture.AssertContains(t, "Completed CreateResource operation on table 'users'")

	// Test failed operation
	capture.Clear()
	testError := fmt.Errorf("test error")
	err2 := LogProviderOperation(logger.Logger, context, func() error {
		return testError
	})

	if err2 != testError {
		t.Errorf("Expected original error to be returned")
	}

	capture.AssertContains(t, "Failed CreateResource operation on table 'users'")
	capture.AssertContains(t, "test error")
}

func TestLogConnectionAttempt(t *testing.T) {
	logger, capture := NewTestLogger(t, "connection", false)

	// Test successful connection
	endpoint := "postgresql://user:password@localhost:5432/db"
	LogConnectionAttempt(logger.Logger, endpoint, nil)

	capture.AssertContains(t, "Successfully connected to postgresql://user:***@localhost:5432/db")

	// Test failed connection
	capture.Clear()
	err := fmt.Errorf("connection timeout")
	LogConnectionAttempt(logger.Logger, endpoint, err)

	capture.AssertContains(t, "Failed to connect to postgresql://user:***@localhost:5432/db")
	capture.AssertContains(t, "connection timeout")
}

func TestLogSchemaValidation(t *testing.T) {
	logger, capture := NewTestLogger(t, "validation", true)

	// Test successful validation
	LogSchemaValidation(logger.Logger, "table", []string{}, []string{})
	capture.AssertContains(t, "Schema validation passed for table")

	// Test validation with errors
	capture.Clear()
	errors := []string{"missing primary key", "invalid column type"}
	LogSchemaValidation(logger.Logger, "table", errors, []string{})

	capture.AssertContains(t, "Schema validation failed for table: 2 errors")
	capture.AssertContains(t, "missing primary key")
	capture.AssertContains(t, "invalid column type")

	// Test validation with warnings
	capture.Clear()
	warnings := []string{"recommended index missing", "column should be NOT NULL"}
	LogSchemaValidation(logger.Logger, "table", []string{}, warnings)

	capture.AssertContains(t, "Schema validation passed for table")
	capture.AssertContains(t, "Schema validation warnings for table: 2 warnings")
	capture.AssertContains(t, "recommended index missing")
}

func TestLogDiscoveryResult(t *testing.T) {
	logger, capture := NewTestLogger(t, "discovery", false)

	// Test discovery with results
	duration := 500 * time.Millisecond
	LogDiscoveryResult(logger.Logger, "table", 5, duration)

	capture.AssertContains(t, "Discovered 5 table resources in 500ms")

	// Test discovery with no results
	capture.Clear()
	LogDiscoveryResult(logger.Logger, "view", 0, duration)

	capture.AssertContains(t, "No view resources found in 500ms")
}

func TestWithContext(t *testing.T) {
	baseLogger := NewLogger("base")
	context := ProviderContext{
		Operation: "CreateResource",
	}

	contextLogger := WithContext(baseLogger, context)

	if !strings.Contains(contextLogger.GetComponent(), "CreateResource") {
		t.Error("Expected context to be included in component name")
	}
}

func TestMockProviderContext(t *testing.T) {
	context := MockProviderContext("postgres", "CreateTable", "table", "users")

	if context.ProviderName != "postgres" {
		t.Errorf("Expected provider name 'postgres', got '%s'", context.ProviderName)
	}
	if context.Operation != "CreateTable" {
		t.Errorf("Expected operation 'CreateTable', got '%s'", context.Operation)
	}
	if context.ResourceType != "table" {
		t.Errorf("Expected resource type 'table', got '%s'", context.ResourceType)
	}
	if context.ResourceName != "users" {
		t.Errorf("Expected resource name 'users', got '%s'", context.ResourceName)
	}

	if context.StartTime.IsZero() {
		t.Error("Expected start time to be set")
	}
}

func TestComplexJSONStructures(t *testing.T) {
	// Test deeply nested structures
	complexData := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": []interface{}{
					map[string]interface{}{
						"nested_array": []string{"a", "b", "c"},
					},
				},
			},
		},
		"large_array": make([]interface{}, 100),
	}

	// Fill large array
	for i := 0; i < 100; i++ {
		complexData["large_array"].([]interface{})[i] = fmt.Sprintf("item_%d", i)
	}

	result := JSONToHuman(complexData, "complex")

	// Should handle nested structures gracefully
	if !strings.Contains(result, "level1={1 fields}") {
		t.Error("Expected nested structure to be summarized")
	}
	if !strings.Contains(result, "large_array=[100 items]") {
		t.Error("Expected large array to be summarized")
	}
}

func TestMalformedJSONHandling(t *testing.T) {
	logger, capture := NewTestLogger(t, "malformed", true)

	// Test with invalid JSON string
	malformedJSON := `{"incomplete": json`
	result := JSONToHuman(malformedJSON, "malformed")

	// Should not crash and should treat as regular string
	expected := "malformed: {\"incomplete\": json"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	// Test with circular reference (should not cause infinite loop)
	circular := map[string]interface{}{}
	circular["self"] = circular

	// This should not crash
	logger.JSONDebug("circular", circular)
	capture.AssertNotEmpty(t) // Just verify something was logged
}

func TestPerformanceWithLargeData(t *testing.T) {
	logger, _ := NewTestLogger(t, "perf", true)

	// Create large data structure
	largeData := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeData[fmt.Sprintf("key_%d", i)] = strings.Repeat("value", 100)
	}

	// Time the JSON debug operation
	start := time.Now()
	logger.JSONDebug("large data", largeData)
	duration := time.Since(start)

	// Should complete within reasonable time (adjust threshold as needed)
	if duration > 100*time.Millisecond {
		t.Errorf("JSON debug took too long: %v", duration)
	}

	// Test human conversion performance
	start = time.Now()
	JSONToHuman(largeData, "performance test")
	duration = time.Since(start)

	if duration > 50*time.Millisecond {
		t.Errorf("JSONToHuman took too long: %v", duration)
	}
}

func BenchmarkJSONToHuman(b *testing.B) {
	data := map[string]interface{}{
		"name":   "test",
		"value":  123,
		"nested": map[string]interface{}{"inner": "data"},
		"array":  []interface{}{"a", "b", "c"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		JSONToHuman(data, "benchmark")
	}
}

func BenchmarkSummarizeRequest(b *testing.B) {
	request := map[string]interface{}{
		"resource_type": "table",
		"name":          "users",
		"schema":        "public",
		"columns":       []interface{}{"id", "name", "email"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SummarizeRequest(request)
	}
}

func BenchmarkSanitizeEndpoint(b *testing.B) {
	endpoint := "postgresql://user:password123@localhost:5432/mydb"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SanitizeEndpoint(endpoint)
	}
}
