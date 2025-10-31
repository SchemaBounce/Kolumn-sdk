package logging

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
)

// resetGlobalConfig resets global config to default state for testing
func resetGlobalConfig() {
	Configure(&Configuration{
		DefaultLevel:    LevelInfo,
		ComponentLevels: make(map[string]Level),
		EnableDebug:     false,
	})
}

func TestLoggerCreation(t *testing.T) {
	resetGlobalConfig()
	logger := NewLogger("test")

	if logger.GetComponent() != "test" {
		t.Errorf("Expected component 'test', got '%s'", logger.GetComponent())
	}

	if logger.GetLevel() != LevelInfo {
		t.Errorf("Expected default level LevelInfo, got %v (string: %s)", logger.GetLevel(), logger.GetLevel().String())
	}
}

func TestLogLevels(t *testing.T) {
	resetGlobalConfig()
	logger, capture := NewTestLogger(t, "test", false)

	// Test info level (should always show)
	logger.Info("info message")
	capture.AssertContains(t, "info message")
	capture.AssertLevel(t, LevelInfo, "test")

	// Test error level (should always show)
	logger.Error("error message")
	capture.AssertContains(t, "error message")
	capture.AssertLevel(t, LevelError, "test")

	// Test warn level (should always show)
	logger.Warn("warn message")
	capture.AssertContains(t, "warn message")
	capture.AssertLevel(t, LevelWarn, "test")

	// Test debug level (should NOT show without debug enabled)
	capture.Clear()
	logger.Debug("debug message")
	capture.AssertNotContains(t, "debug message")
}

func TestDebugMode(t *testing.T) {
	resetGlobalConfig()
	// Test with debug enabled
	logger, capture := NewTestLogger(t, "test", true)

	logger.Debug("debug message")
	capture.AssertContains(t, "debug message")
	capture.AssertLevel(t, LevelDebug, "test")

	// Test debug disabled
	DisableDebug()
	capture.Clear()
	logger.Debug("debug message 2")
	capture.AssertNotContains(t, "debug message 2")
}

func TestStructuredLogging(t *testing.T) {
	resetGlobalConfig()
	logger, capture := NewTestLogger(t, "test", false)

	logger.InfoWithFields("test message", "key1", "value1", "key2", "value2")

	output := capture.GetOutput()
	if !strings.Contains(output, "test message") {
		t.Error("Expected message not found in output")
	}
	if !strings.Contains(output, "key1=value1") {
		t.Error("Expected key1=value1 not found in output")
	}
	if !strings.Contains(output, "key2=value2") {
		t.Error("Expected key2=value2 not found in output")
	}
}

func TestJSONDebugLogging(t *testing.T) {
	logger, capture := NewTestLogger(t, "test", true)

	testData := map[string]interface{}{
		"name":  "test",
		"value": 123,
		"nested": map[string]interface{}{
			"inner": "data",
		},
	}

	logger.JSONDebug("test context", testData)

	output := capture.GetOutput()
	if !strings.Contains(output, "test context") {
		t.Error("Expected context not found in JSON debug output")
	}
	if !strings.Contains(output, "name") {
		t.Error("Expected JSON data not found in debug output")
	}

	// Test that JSON debug is suppressed when debug is disabled
	DisableDebug()
	capture.Clear()
	logger.JSONDebug("suppressed context", testData)
	capture.AssertEmpty(t)
}

func TestEnvironmentConfiguration(t *testing.T) {
	// Save original env
	originalDebug := os.Getenv("DEBUG")
	originalComponents := os.Getenv("DEBUG_COMPONENTS")

	// Clean up
	defer func() {
		os.Setenv("DEBUG", originalDebug)
		os.Setenv("DEBUG_COMPONENTS", originalComponents)
		loadEnvironmentConfig() // Reload original config
	}()

	// Test DEBUG=1
	os.Setenv("DEBUG", "1")
	os.Setenv("DEBUG_COMPONENTS", "")
	loadEnvironmentConfig()

	if !GetGlobalDebugStatus() {
		t.Error("Expected global debug to be enabled with DEBUG=1")
	}

	// Test DEBUG_COMPONENTS
	os.Setenv("DEBUG", "")
	os.Setenv("DEBUG_COMPONENTS", "test,provider")
	loadEnvironmentConfig()

	// Create logger and test if debug is enabled for specific component
	logger := NewLogger("test")
	if !logger.IsDebugEnabled() {
		t.Error("Expected debug to be enabled for 'test' component")
	}

	logger2 := NewLogger("other")
	if logger2.IsDebugEnabled() {
		t.Error("Expected debug to be disabled for 'other' component")
	}
}

func TestComponentSpecificConfiguration(t *testing.T) {
	resetGlobalConfig()
	cleanup := SetupTestLogging(t, false)
	defer cleanup()

	// Configure specific component levels
	Configure(&Configuration{
		DefaultLevel: LevelInfo,
		ComponentLevels: map[string]Level{
			"debug_component": LevelDebug,
			"warn_component":  LevelWarn,
		},
	})

	debugLogger := NewLogger("debug_component")
	warnLogger := NewLogger("warn_component")
	defaultLogger := NewLogger("default_component")

	if !debugLogger.IsDebugEnabled() {
		t.Error("Expected debug to be enabled for debug_component")
	}

	if warnLogger.IsDebugEnabled() {
		t.Error("Expected debug to be disabled for warn_component")
	}

	if defaultLogger.IsDebugEnabled() {
		t.Error("Expected debug to be disabled for default_component")
	}
}

func TestPreConfiguredLoggers(t *testing.T) {
	loggers := []*Logger{
		ProviderLogger, ConnectionLogger, HandlerLogger, ValidationLogger,
		SecurityLogger, StateLogger, DiscoveryLogger, ConfigLogger,
		RegistryLogger, DispatchLogger, SchemaLogger,
	}

	expectedComponents := []string{
		"provider", "connection", "handler", "validation",
		"security", "state", "discovery", "config",
		"registry", "dispatch", "schema",
	}

	if len(loggers) != len(expectedComponents) {
		t.Fatalf("Expected %d pre-configured loggers, got %d", len(expectedComponents), len(loggers))
	}

	for i, logger := range loggers {
		if logger == nil {
			t.Errorf("Logger %d (%s) is nil", i, expectedComponents[i])
			continue
		}

		if logger.GetComponent() != expectedComponents[i] {
			t.Errorf("Expected component '%s', got '%s'", expectedComponents[i], logger.GetComponent())
		}
	}
}

func TestConcurrentLogging(t *testing.T) {
	logger, capture := NewTestLogger(t, "concurrent", true)

	const numGoroutines = 10
	const messagesPerGoroutine = 5

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				logger.Info("Message from goroutine %d: %d", id, j)
				logger.Debug("Debug from goroutine %d: %d", id, j)
				logger.Warn("Warning from goroutine %d: %d", id, j)
			}
		}(i)
	}

	wg.Wait()

	lines := capture.GetLines()
	expectedMessages := numGoroutines * messagesPerGoroutine * 3 // info, debug, warn

	if len(lines) != expectedMessages {
		t.Errorf("Expected %d log messages, got %d", expectedMessages, len(lines))
	}

	// Verify all goroutines logged
	for i := 0; i < numGoroutines; i++ {
		found := false
		for _, line := range lines {
			if strings.Contains(line, fmt.Sprintf("goroutine %d", i)) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("No log messages found from goroutine %d", i)
		}
	}
}

func TestConfigurationUpdates(t *testing.T) {
	resetGlobalConfig()
	cleanup := SetupTestLogging(t, false)
	defer cleanup()

	logger := NewLogger("dynamic")

	// Initially debug should be disabled
	if logger.IsDebugEnabled() {
		t.Error("Expected debug to be initially disabled")
	}

	// Enable debug and verify it takes effect
	EnableDebug()
	if !logger.IsDebugEnabled() {
		t.Error("Expected debug to be enabled after EnableDebug()")
	}

	// Disable debug and verify it takes effect
	DisableDebug()
	if logger.IsDebugEnabled() {
		t.Error("Expected debug to be disabled after DisableDebug()")
	}

	// Enable component-specific debug
	EnableComponentDebug("dynamic")
	if !logger.IsDebugEnabled() {
		t.Error("Expected debug to be enabled for component after EnableComponentDebug()")
	}
}

func TestLoggerStringRepresentation(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelInfo, "INFO"},
		{LevelDebug, "DEBUG"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{Level(999), "UNKNOWN"}, // Invalid level
	}

	for _, test := range tests {
		if test.level.String() != test.expected {
			t.Errorf("Expected %s for level %d, got %s", test.expected, test.level, test.level.String())
		}
	}
}

func TestOperationLogging(t *testing.T) {
	logger, capture := NewTestLogger(t, "operation", false)

	logger.OperationStart("CreateTable", "users")
	logger.OperationComplete("CreateTable", "users")
	logger.OperationFailed("DeleteTable", "orders", fmt.Errorf("table not found"))

	lines := capture.GetLines()
	if len(lines) != 3 {
		t.Errorf("Expected 3 operation log lines, got %d", len(lines))
	}

	capture.AssertContains(t, "Starting CreateTable operation on users")
	capture.AssertContains(t, "Completed CreateTable operation on users")
	capture.AssertContains(t, "Failed DeleteTable operation on orders")
}

func TestLogFormatConsistency(t *testing.T) {
	logger, capture := NewTestLogger(t, "format", false)

	logger.Info("test message")

	output := capture.GetOutput()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 1 {
		t.Fatalf("Expected 1 log line, got %d", len(lines))
	}

	line := lines[0]

	// Check format: [LEVEL][COMPONENT] message
	if !strings.Contains(line, "[INFO][format] test message") {
		t.Errorf("Log format incorrect. Expected '[INFO][format] test message', got: %s", line)
	}
}

func TestMemoryUsageWithLargeLogs(t *testing.T) {
	logger, capture := NewTestLogger(t, "memory", true)

	// Create large log message
	largeMessage := strings.Repeat("A", 10000)
	largeData := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		largeData[fmt.Sprintf("key_%d", i)] = strings.Repeat("B", 100)
	}

	// Log many large messages
	for i := 0; i < 100; i++ {
		logger.Info("Large message %d: %s", i, largeMessage[:100]) // Truncate for readability
		logger.JSONDebug("large data", largeData)
	}

	// Verify output was captured (basic test that it didn't crash)
	lines := capture.GetLines()
	if len(lines) < 100 {
		t.Errorf("Expected at least 100 log lines, got %d", len(lines))
	}

	// Test that debug mode doesn't crash with large JSON
	capture.Clear()
	logger.JSONDebug("huge json", largeData)
	capture.AssertNotEmpty(t)
}

func TestEdgeCases(t *testing.T) {
	resetGlobalConfig()
	logger, capture := NewTestLogger(t, "edge", true)

	// Test nil values
	logger.Info("nil test: %v", nil)
	logger.JSONDebug("nil json", nil)

	// Test empty strings
	logger.Info("")
	logger.Debug("")

	// Test format strings without args
	logger.Info("no args")

	// Test format strings with mismatched args (temporarily commented)
	// logger.Info("too many args %s", "arg1", "arg2")

	// Test special characters
	logger.Info("Special chars: \n\t\r\"'\\")

	// Test unicode
	logger.Info("Unicode: 你好 🚀 ñoño")

	// Verify no crashes occurred
	lines := capture.GetLines()
	if len(lines) == 0 {
		t.Error("Expected some log output from edge cases")
	}
}

func TestStructuredLoggingEdgeCases(t *testing.T) {
	logger, capture := NewTestLogger(t, "structured", false)

	// Test odd number of fields (should handle gracefully)
	logger.InfoWithFields("odd fields", "key1", "value1", "key2")
	capture.AssertContains(t, "key1=value1")

	// Test no fields
	logger.InfoWithFields("no fields")
	capture.AssertContains(t, "no fields")

	// Test empty field values
	logger.InfoWithFields("empty", "key", "")
	capture.AssertContains(t, "key=")

	// Test nil field values
	logger.InfoWithFields("nil values", "key", nil)
	capture.AssertContains(t, "key=<nil>")
}

func BenchmarkLogInfo(b *testing.B) {
	logger := NewLogger("bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("Benchmark message %d", i)
	}
}

func BenchmarkLogDebugDisabled(b *testing.B) {
	logger := NewLogger("bench")
	// Debug is disabled by default

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Debug("Debug message %d", i)
	}
}

func BenchmarkLogDebugEnabled(b *testing.B) {
	EnableDebug()
	defer DisableDebug()

	logger := NewLogger("bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Debug("Debug message %d", i)
	}
}

func BenchmarkStructuredLogging(b *testing.B) {
	logger := NewLogger("bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.InfoWithFields("Benchmark message", "iteration", i, "type", "benchmark")
	}
}

func BenchmarkJSONDebug(b *testing.B) {
	EnableDebug()
	defer DisableDebug()

	logger := NewLogger("bench")

	testData := map[string]interface{}{
		"name":    "test",
		"value":   123,
		"complex": map[string]interface{}{"nested": "data"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.JSONDebug("benchmark data", testData)
	}
}
