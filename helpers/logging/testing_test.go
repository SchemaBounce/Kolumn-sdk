package logging

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewTestLogger(t *testing.T) {
	logger, capture := NewTestLogger(t, "test", false)

	// Verify logger was created correctly
	if logger.GetComponent() != "test" {
		t.Errorf("Expected component 'test', got '%s'", logger.GetComponent())
	}

	// Verify capture was created correctly
	if capture == nil {
		t.Fatal("Expected capture to be created")
	}

	// Test basic logging
	logger.Info("test message")
	if !strings.Contains(capture.GetOutput(), "test message") {
		t.Error("Expected test message to be captured")
	}
}

func TestTestLoggerWithDebug(t *testing.T) {
	logger, capture := NewTestLogger(t, "debug_test", true)

	// Debug should be enabled
	if !logger.IsDebugEnabled() {
		t.Error("Expected debug to be enabled")
	}

	logger.Debug("debug message")
	capture.AssertContains(t, "debug message")
}

func TestLogCapture(t *testing.T) {
	logger, capture := NewTestLogger(t, "capture", false)

	// Test basic capture
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	// Test GetOutput
	output := capture.GetOutput()
	if !strings.Contains(output, "info message") {
		t.Error("Expected info message in output")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Expected warn message in output")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Expected error message in output")
	}

	// Test GetLines
	lines := capture.GetLines()
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	// Test Clear
	capture.Clear()
	if capture.GetOutput() != "" {
		t.Error("Expected output to be cleared")
	}
	if len(capture.GetLines()) != 0 {
		t.Error("Expected lines to be cleared")
	}
}

func TestLogCaptureAssertions(t *testing.T) {
	logger, capture := NewTestLogger(t, "assertions", false)

	logger.Info("test message")
	logger.Error("error occurred")

	// Test AssertContains
	capture.AssertContains(t, "test message")
	capture.AssertContains(t, "error occurred")

	// Test AssertNotContains
	capture.AssertNotContains(t, "should not exist")

	// Test AssertLevel
	capture.AssertLevel(t, LevelInfo, "assertions")
	capture.AssertLevel(t, LevelError, "assertions")

	// Test AssertNoLevel (debug should not be present)
	capture.AssertNoLevel(t, LevelDebug, "assertions")

	// Test AssertNotEmpty
	capture.AssertNotEmpty(t)

	// Test with empty capture
	capture.Clear()
	capture.AssertEmpty(t)
}

func TestLogCaptureAssertionFailures(t *testing.T) {
	// This test verifies that assertion failures are detected
	// We'll use a mock testing.T to capture failures

	logger, capture := NewTestLogger(t, "failures", false)
	logger.Info("actual message")

	// Create a mock test that tracks failures
	mockT := &mockTesting{}

	// These should "fail" the mock test
	capture.AssertContains(mockT, "missing message")
	if !mockT.failed {
		t.Error("Expected AssertContains to fail for missing message")
	}

	mockT.Reset()
	capture.AssertNotContains(mockT, "actual message")
	if !mockT.failed {
		t.Error("Expected AssertNotContains to fail for present message")
	}

	mockT.Reset()
	capture.AssertLevel(mockT, LevelDebug, "failures")
	if !mockT.failed {
		t.Error("Expected AssertLevel to fail for missing debug level")
	}

	mockT.Reset()
	capture.AssertEmpty(mockT)
	if !mockT.failed {
		t.Error("Expected AssertEmpty to fail for non-empty capture")
	}

	// Test with empty capture
	capture.Clear()
	mockT.Reset()
	capture.AssertNotEmpty(mockT)
	if !mockT.failed {
		t.Error("Expected AssertNotEmpty to fail for empty capture")
	}
}

func TestCountLevel(t *testing.T) {
	logger, capture := NewTestLogger(t, "count", true)

	// Log multiple messages at different levels
	logger.Info("info 1")
	logger.Info("info 2")
	logger.Debug("debug 1")
	logger.Error("error 1")

	// Test count by level
	infoCount := capture.CountLevel(LevelInfo, "count")
	if infoCount != 2 {
		t.Errorf("Expected 2 info messages, got %d", infoCount)
	}

	debugCount := capture.CountLevel(LevelDebug, "count")
	if debugCount != 1 {
		t.Errorf("Expected 1 debug message, got %d", debugCount)
	}

	errorCount := capture.CountLevel(LevelError, "count")
	if errorCount != 1 {
		t.Errorf("Expected 1 error message, got %d", errorCount)
	}

	warnCount := capture.CountLevel(LevelWarn, "count")
	if warnCount != 0 {
		t.Errorf("Expected 0 warn messages, got %d", warnCount)
	}
}

func TestSetupTestLogging(t *testing.T) {
	// Test that cleanup function works
	cleanup := SetupTestLogging(t, true)

	// Debug should be enabled
	if !GetGlobalDebugStatus() {
		t.Error("Expected debug to be enabled after setup")
	}

	// Test that cleanup restores state
	cleanup()

	// Debug status may vary depending on environment, but cleanup should not crash
}

func TestMockProviderContextCreation(t *testing.T) {
	context := MockProviderContext("postgres", "CreateTable", "table", "users")

	if context.ProviderName != "postgres" {
		t.Errorf("Expected provider 'postgres', got '%s'", context.ProviderName)
	}
	if context.Operation != "CreateTable" {
		t.Errorf("Expected operation 'CreateTable', got '%s'", context.Operation)
	}
	if context.ResourceType != "table" {
		t.Errorf("Expected type 'table', got '%s'", context.ResourceType)
	}
	if context.ResourceName != "users" {
		t.Errorf("Expected name 'users', got '%s'", context.ResourceName)
	}

	// Start time should be recent
	if time.Since(context.StartTime) > time.Second {
		t.Error("Expected start time to be recent")
	}
}

func TestTestLogLevelsHelper(t *testing.T) {
	logger, capture := NewTestLogger(t, "levels", false)

	// Test the helper function
	ValidateLogLevels(t, logger.Logger, capture)

	// Verify all expected levels were tested
	lines := capture.GetLines()
	if len(lines) < 3 {
		t.Errorf("Expected at least 3 log lines from TestLogLevels, got %d", len(lines))
	}

	capture.AssertContains(t, "test info message")
	capture.AssertContains(t, "test error message")
	capture.AssertContains(t, "test warn message")
}

func TestTestDebugLevelHelper(t *testing.T) {
	// Test with debug enabled
	logger, capture := NewTestLogger(t, "debug", true)
	ValidateDebugLevel(t, logger.Logger, capture, true)
	capture.AssertContains(t, "test debug message")

	// Test with debug disabled
	logger2, capture2 := NewTestLogger(t, "no_debug", false)
	ValidateDebugLevel(t, logger2.Logger, capture2, false)
	capture2.AssertNotContains(t, "test debug message")
}

func TestTestStructuredLoggingHelper(t *testing.T) {
	logger, capture := NewTestLogger(t, "structured", false)

	ValidateStructuredLogging(t, logger.Logger, capture)

	output := capture.GetOutput()
	if !strings.Contains(output, "test message") {
		t.Error("Expected test message in structured logging test")
	}
	if !strings.Contains(output, "key1=value1") {
		t.Error("Expected key1=value1 in structured logging test")
	}
	if !strings.Contains(output, "key2=value2") {
		t.Error("Expected key2=value2 in structured logging test")
	}
}

func TestTestLoggerRestore(t *testing.T) {
	// Create test logger
	logger, _ := NewTestLogger(t, "restore", false)

	// Log something to verify capture is working
	logger.Info("before restore")

	// Manually restore (normally done by t.Cleanup)
	logger.Restore()

	// After restore, logging should go to normal output
	// This is hard to test without capturing os.Stdout, but we can at least
	// verify that Restore() doesn't crash
}

func TestConcurrentTestLoggers(t *testing.T) {
	// Test multiple test loggers don't interfere with each other
	logger1, capture1 := NewTestLogger(t, "concurrent1", false)
	logger2, capture2 := NewTestLogger(t, "concurrent2", false)

	logger1.Info("message from logger 1")
	logger2.Info("message from logger 2")

	// Each capture should only have its own logger's messages
	capture1.AssertContains(t, "message from logger 1")
	capture1.AssertNotContains(t, "message from logger 2")

	capture2.AssertContains(t, "message from logger 2")
	capture2.AssertNotContains(t, "message from logger 1")
}

func TestExampleLogOutput(t *testing.T) {
	// Test that the example function doesn't crash
	// This is mainly a compilation test
	ExampleLogOutput()
}

// mockTesting implements a subset of testing.T for testing assertion failures
type mockTesting struct {
	failed bool
	errors []string
}

func (m *mockTesting) Reset() {
	m.failed = false
	m.errors = nil
}

func (m *mockTesting) Errorf(format string, args ...interface{}) {
	m.failed = true
	m.errors = append(m.errors, fmt.Sprintf(format, args...))
}

func (m *mockTesting) Error(args ...interface{}) {
	m.failed = true
	m.errors = append(m.errors, fmt.Sprint(args...))
}

func TestLogCaptureEdgeCases(t *testing.T) {
	logger, capture := NewTestLogger(t, "edge", false)

	// Test with no logging
	capture.AssertEmpty(t)

	// Test with very long log message
	longMessage := strings.Repeat("A", 10000)
	logger.Info(longMessage)

	output := capture.GetOutput()
	if !strings.Contains(output, longMessage) {
		t.Error("Expected long message to be captured")
	}

	// Test with special characters
	logger.Info("Special: \n\t\r\"'\\")
	capture.AssertContains(t, "Special:")

	// Test with unicode
	logger.Info("Unicode: 你好 🚀")
	capture.AssertContains(t, "Unicode:")
}

func TestLogCapturePerformance(t *testing.T) {
	logger, capture := NewTestLogger(t, "performance", false)

	// Log many messages quickly
	start := time.Now()
	for i := 0; i < 1000; i++ {
		logger.Info("Message %d", i)
	}
	duration := time.Since(start)

	// Should complete quickly
	if duration > time.Second {
		t.Errorf("Logging 1000 messages took too long: %v", duration)
	}

	// Verify all messages were captured
	lines := capture.GetLines()
	if len(lines) != 1000 {
		t.Errorf("Expected 1000 lines, got %d", len(lines))
	}
}

func BenchmarkTestLogger(b *testing.B) {
	logger, _ := NewTestLogger(&testing.T{}, "bench", false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("Benchmark message %d", i)
	}
}

func BenchmarkLogCapture(b *testing.B) {
	logger, capture := NewTestLogger(&testing.T{}, "bench", false)

	// Fill with some data
	for i := 0; i < 100; i++ {
		logger.Info("Setup message %d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		capture.GetOutput()
		capture.GetLines()
	}
}
