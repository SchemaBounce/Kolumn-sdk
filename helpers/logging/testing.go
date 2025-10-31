package logging

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestLogger provides a logger that captures output for testing
type TestLogger struct {
	*Logger
	buffer   *bytes.Buffer
	original *log.Logger
	mu       sync.Mutex
}

// TestingInterface represents the minimal interface needed for assertions
type TestingInterface interface {
	Errorf(format string, args ...interface{})
	Error(args ...interface{})
}

// LogCapture captures log output for testing and validation
type LogCapture struct {
	buffer   *bytes.Buffer
	original *log.Logger
	mu       sync.Mutex
}

// NewTestLogger creates a logger that captures output for testing
func NewTestLogger(t *testing.T, component string, enableDebug bool) (*TestLogger, *LogCapture) {
	buffer := &bytes.Buffer{}
	original := log.Default()

	// Redirect log output to buffer
	log.SetOutput(buffer)
	log.SetFlags(0)

	// Configure debug mode if requested
	if enableDebug {
		Configure(&Configuration{
			EnableDebug: true,
			ComponentLevels: map[string]Level{
				component: LevelDebug,
			},
		})
	}

	// Create test logger
	logger := NewLogger(component)
	testLogger := &TestLogger{
		Logger:   logger,
		buffer:   buffer,
		original: original,
	}

	// Create capture for assertions
	capture := &LogCapture{
		buffer:   buffer,
		original: original,
	}

	// Set cleanup function
	t.Cleanup(func() {
		testLogger.Restore()
	})

	return testLogger, capture
}

// Restore restores the original logger settings
func (tl *TestLogger) Restore() {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	log.SetOutput(tl.original.Writer())
	log.SetFlags(log.LstdFlags)
}

// GetOutput returns the captured log output
func (tl *TestLogger) GetOutput() string {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	return tl.buffer.String()
}

// ClearOutput clears the captured log output
func (tl *TestLogger) ClearOutput() {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.buffer.Reset()
}

// AssertContains checks that the log output contains the expected message
func (lc *LogCapture) AssertContains(t TestingInterface, expected string) {
	lc.mu.Lock()
	output := lc.buffer.String()
	lc.mu.Unlock()

	if !strings.Contains(output, expected) {
		t.Errorf("Expected log output to contain %q, but got:\n%s", expected, output)
	}
}

// AssertNotContains checks that the log output does not contain the message
func (lc *LogCapture) AssertNotContains(t TestingInterface, unexpected string) {
	lc.mu.Lock()
	output := lc.buffer.String()
	lc.mu.Unlock()

	if strings.Contains(output, unexpected) {
		t.Errorf("Expected log output to NOT contain %q, but got:\n%s", unexpected, output)
	}
}

// AssertLevel checks that the log output contains messages at the expected level
func (lc *LogCapture) AssertLevel(t TestingInterface, level Level, component string) {
	expected := fmt.Sprintf("[%s][%s]", level.String(), component)
	lc.AssertContains(t, expected)
}

// AssertNoLevel checks that the log output does not contain messages at the specified level
func (lc *LogCapture) AssertNoLevel(t TestingInterface, level Level, component string) {
	unexpected := fmt.Sprintf("[%s][%s]", level.String(), component)
	lc.AssertNotContains(t, unexpected)
}

// AssertEmpty checks that no log output was generated
func (lc *LogCapture) AssertEmpty(t TestingInterface) {
	lc.mu.Lock()
	output := lc.buffer.String()
	lc.mu.Unlock()

	if strings.TrimSpace(output) != "" {
		t.Errorf("Expected no log output, but got:\n%s", output)
	}
}

// AssertNotEmpty checks that some log output was generated
func (lc *LogCapture) AssertNotEmpty(t TestingInterface) {
	lc.mu.Lock()
	output := lc.buffer.String()
	lc.mu.Unlock()

	if strings.TrimSpace(output) == "" {
		t.Error("Expected log output, but got none")
	}
}

// GetLines returns the log output split into lines
func (lc *LogCapture) GetLines() []string {
	lc.mu.Lock()
	output := lc.buffer.String()
	lc.mu.Unlock()

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}
	}
	return lines
}

// CountLevel counts the number of log messages at the specified level
func (lc *LogCapture) CountLevel(level Level, component string) int {
	lines := lc.GetLines()
	pattern := fmt.Sprintf("[%s][%s]", level.String(), component)
	count := 0

	for _, line := range lines {
		if strings.Contains(line, pattern) {
			count++
		}
	}

	return count
}

// GetOutput returns the complete captured output
func (lc *LogCapture) GetOutput() string {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return lc.buffer.String()
}

// Clear resets the captured output
func (lc *LogCapture) Clear() {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.buffer.Reset()
}

// SetupTestLogging configures logging for tests and returns a cleanup function
func SetupTestLogging(t *testing.T, enableDebug bool) func() {
	// Save original configuration
	originalConfig := &Configuration{
		DefaultLevel:    globalConfig.DefaultLevel,
		ComponentLevels: make(map[string]Level),
		EnableDebug:     globalConfig.EnableDebug,
	}
	for k, v := range globalConfig.ComponentLevels {
		originalConfig.ComponentLevels[k] = v
	}

	// Configure for testing
	Configure(&Configuration{
		DefaultLevel: LevelInfo,
		EnableDebug:  enableDebug,
		ComponentLevels: map[string]Level{
			"test": LevelDebug,
		},
	})

	// Return cleanup function
	return func() {
		Configure(originalConfig)
	}
}

// MockProviderContext creates a mock provider context for testing
func MockProviderContext(providerName, operation, resourceType, resourceName string) ProviderContext {
	return ProviderContext{
		ProviderName: providerName,
		Operation:    operation,
		ResourceType: resourceType,
		ResourceName: resourceName,
		StartTime:    time.Now(),
	}
}

// ValidateLogLevels tests that log levels work correctly
func ValidateLogLevels(t TestingInterface, logger *Logger, capture *LogCapture) {
	// Test info level (should always show)
	logger.Info("test info message")
	capture.AssertContains(t, "test info message")
	capture.AssertLevel(t, LevelInfo, logger.GetComponent())

	// Test error level (should always show)
	logger.Error("test error message")
	capture.AssertContains(t, "test error message")
	capture.AssertLevel(t, LevelError, logger.GetComponent())

	// Test warn level (should always show)
	logger.Warn("test warn message")
	capture.AssertContains(t, "test warn message")
	capture.AssertLevel(t, LevelWarn, logger.GetComponent())
}

// ValidateDebugLevel tests debug level logging
func ValidateDebugLevel(t TestingInterface, logger *Logger, capture *LogCapture, shouldShow bool) {
	capture.Clear()

	logger.Debug("test debug message")

	if shouldShow {
		capture.AssertContains(t, "test debug message")
		capture.AssertLevel(t, LevelDebug, logger.GetComponent())
	} else {
		capture.AssertNotContains(t, "test debug message")
		capture.AssertNoLevel(t, LevelDebug, logger.GetComponent())
	}
}

// ValidateStructuredLogging tests structured logging with fields
func ValidateStructuredLogging(t TestingInterface, logger *Logger, capture *LogCapture) {
	logger.InfoWithFields("test message", "key1", "value1", "key2", "value2")
	capture.AssertContains(t, "test message")
	capture.AssertContains(t, "key1=value1")
	capture.AssertContains(t, "key2=value2")
}

// ExampleLogOutput demonstrates expected log output format
func ExampleLogOutput() {
	// Example of what the logs should look like:
	fmt.Println("[INFO][provider] Starting CreateResource operation on table 'users'")
	fmt.Println("[DEBUG][handler] CreateResource request for table 'users' name=users schema=public")
	fmt.Println("[INFO][connection] Successfully connected to postgres://user:***@localhost:5432/db")
	fmt.Println("[DEBUG][validation] Schema validation passed for table")
	fmt.Println("[INFO][provider] Completed CreateResource operation on table 'users' in 245ms")
	fmt.Println("[WARN][discovery] Schema validation warnings for table: 2 warnings")
	fmt.Println("[ERROR][provider] Failed CreateResource operation on table 'users' after 1.2s: connection failed")
}
