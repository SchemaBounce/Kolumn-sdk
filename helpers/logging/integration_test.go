package logging

import (
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestProviderOperationFlow tests a complete provider operation with logging
func TestProviderOperationFlow(t *testing.T) {
	cleanup := SetupTestLogging(t, false)
	defer cleanup()

	// Simulate provider initialization
	provider := &MockProvider{name: "postgres", version: "1.0.0"}

	// Test configuration
	config := map[string]interface{}{
		"host":     "localhost",
		"port":     5432,
		"database": "testdb",
		"username": "testuser",
		"password": "secret123",
	}

	capture := &testLogCapture{}
	setupLogCapture(capture)
	defer restoreLogOutput()

	// Configure provider
	err := provider.Configure(config)
	if err != nil {
		t.Fatalf("Failed to configure provider: %v", err)
	}

	// Test CRUD operations
	createReq := &MockCreateRequest{
		Name: "users",
		Type: "table",
	}

	resp, err := provider.CreateResource(createReq)
	if err != nil {
		t.Fatalf("Failed to create resource: %v", err)
	}

	if resp.ID == "" {
		t.Error("Expected response to have ID")
	}

	// Verify logging output
	logs := capture.GetLogs()
	t.Logf("captured logs: %v", logs)

	// Should have configuration logs
	if !containsLog(logs, "config", "Configuring provider") {
		t.Error("Expected configuration log")
	}

	// Should have operation logs
	if !containsLog(logs, "handler", "CreateResource request") {
		t.Error("Expected create request log")
	}

	if !containsLog(logs, "handler", "CreateResource completed") {
		t.Error("Expected create completion log")
	}

	// Should sanitize sensitive data
	if containsString(logs, "secret123") {
		t.Error("Password should be sanitized in logs")
	}
}

// TestDebugModeToggling tests dynamic debug mode changes
func TestDebugModeToggling(t *testing.T) {
	// Reset to clean state first
	Configure(&Configuration{
		DefaultLevel:    LevelInfo,
		ComponentLevels: make(map[string]Level),
		EnableDebug:     false,
	})

	cleanup := SetupTestLogging(t, false)
	defer cleanup()

	capture := &testLogCapture{}
	setupLogCapture(capture)
	defer restoreLogOutput()

	provider := &MockProvider{name: "test", version: "1.0.0"}

	// Initially debug should be off
	provider.LogDebugInfo("debug info 1")

	logs1 := capture.GetLogs()
	if containsString(logs1, "debug info 1") {
		t.Error("Debug message should not appear when debug is disabled")
	}

	// Enable debug
	EnableDebug()

	provider.LogDebugInfo("debug info 2")

	logs2 := capture.GetLogs()
	if !containsString(logs2, "debug info 2") {
		t.Error("Debug message should appear when debug is enabled")
	}

	// Disable debug
	DisableDebug()

	capture.Clear()
	provider.LogDebugInfo("debug info 3")

	logs3 := capture.GetLogs()
	if containsString(logs3, "debug info 3") {
		t.Error("Debug message should not appear when debug is disabled again")
	}
}

// TestComponentSpecificLogging tests component-specific debug settings
func TestComponentSpecificLogging(t *testing.T) {
	cleanup := SetupTestLogging(t, false)
	defer cleanup()

	Configure(&Configuration{
		DefaultLevel: LevelInfo,
		ComponentLevels: map[string]Level{
			"handler": LevelDebug,
			"config":  LevelWarn,
		},
	})

	capture := &testLogCapture{}
	setupLogCapture(capture)
	defer restoreLogOutput()

	// Handler should show debug
	HandlerLogger.Debug("handler debug message")

	// Config should NOT show debug (set to warn level)
	ConfigLogger.Debug("config debug message")

	// Provider should NOT show debug (default level)
	ProviderLogger.Debug("provider debug message")

	logs := capture.GetLogs()

	if !containsString(logs, "handler debug message") {
		t.Error("Handler debug message should appear")
	}

	if containsString(logs, "config debug message") {
		t.Error("Config debug message should not appear")
	}

	if containsString(logs, "provider debug message") {
		t.Error("Provider debug message should not appear")
	}
}

// TestLargeDataLogging tests logging with large datasets
func TestLargeDataLogging(t *testing.T) {
	cleanup := SetupTestLogging(t, true) // Enable debug
	defer cleanup()

	capture := &testLogCapture{}
	setupLogCapture(capture)
	defer restoreLogOutput()

	// Create large request
	largeRequest := &MockDiscoveryRequest{
		Query: "large query",
		Data:  make(map[string]interface{}),
	}

	// Add lots of data
	for i := 0; i < 1000; i++ {
		largeRequest.Data[fmt.Sprintf("key_%d", i)] = strings.Repeat("value", 100)
	}

	start := time.Now()
	LogRequest(DiscoveryLogger, "DiscoverResources", largeRequest)
	duration := time.Since(start)

	// Should complete within reasonable time
	if duration > 100*time.Millisecond {
		t.Errorf("Large data logging took too long: %v", duration)
	}

	logs := capture.GetLogs()
	if !containsLog(logs, "discovery", "DiscoverResources request") {
		t.Error("Expected discovery request log")
	}
}

// TestConcurrentProviderLogging tests multiple providers logging simultaneously
func TestConcurrentProviderLogging(t *testing.T) {
	cleanup := SetupTestLogging(t, false)
	defer cleanup()

	capture := &testLogCapture{}
	setupLogCapture(capture)
	defer restoreLogOutput()

	const numProviders = 5
	const operationsPerProvider = 10

	var wg sync.WaitGroup
	wg.Add(numProviders)

	// Start multiple providers concurrently
	for i := 0; i < numProviders; i++ {
		go func(providerID int) {
			defer wg.Done()

			provider := &MockProvider{
				name:    fmt.Sprintf("provider_%d", providerID),
				version: "1.0.0",
			}

			for j := 0; j < operationsPerProvider; j++ {
				req := &MockCreateRequest{
					Name: fmt.Sprintf("resource_%d_%d", providerID, j),
					Type: "table",
				}

				_, err := provider.CreateResource(req)
				if err != nil {
					t.Errorf("Provider %d operation %d failed: %v", providerID, j, err)
				}

				time.Sleep(time.Millisecond) // Small delay to interleave operations
			}
		}(i)
	}

	wg.Wait()

	logs := capture.GetLogs()

	// Verify all providers logged
	for i := 0; i < numProviders; i++ {
		resourcePrefix := fmt.Sprintf("resource_%d_", i)
		if !containsString(logs, resourcePrefix) {
			t.Errorf("No logs found for operations from provider %d", i)
		}
	}

	// Verify correct number of operations
	createCount := countLogMatches(logs, "CreateResource")
	expectedCount := numProviders * operationsPerProvider * 2 // Request + response logs
	if createCount < expectedCount {
		t.Errorf("Expected at least %d create logs, got %d", expectedCount, createCount)
	}
}

// TestErrorHandlingAndLogging tests error scenarios and their logging
func TestErrorHandlingAndLogging(t *testing.T) {
	cleanup := SetupTestLogging(t, false)
	defer cleanup()

	capture := &testLogCapture{}
	setupLogCapture(capture)
	defer restoreLogOutput()

	provider := &MockProvider{name: "error_test", version: "1.0.0"}

	// Test configuration error
	invalidConfig := map[string]interface{}{
		"invalid_field": "value",
	}

	err := provider.Configure(invalidConfig)
	if err == nil {
		t.Error("Expected configuration to fail")
	}

	// Test operation error
	invalidReq := &MockCreateRequest{
		Name: "", // Invalid empty name
		Type: "table",
	}

	_, err = provider.CreateResource(invalidReq)
	if err == nil {
		t.Error("Expected create operation to fail")
	}

	logs := capture.GetLogs()

	// Should have error logs
	if !containsLog(logs, "config", "Configuration failed") {
		t.Error("Expected configuration error log")
	}

	if !containsLog(logs, "handler", "CreateResource failed") {
		t.Error("Expected operation error log")
	}
}

// TestLogFormatConsistencyIntegration tests log format across all components
func TestLogFormatConsistencyIntegration(t *testing.T) {
	withCleanConfig(t)
	capture := &testLogCapture{}
	setupLogCapture(capture)
	defer restoreLogOutput()

	components := []string{
		"provider", "connection", "handler", "validation",
		"security", "state", "discovery", "config",
		"registry", "dispatch", "schema",
	}

	loggers := make(map[string]*Logger, len(components))
	for _, name := range components {
		loggers[name] = NewLogger(name + "_format_test")
	}

	for name, logger := range loggers {
		logger.Info("Test message from %s", name)
	}

	logs := capture.GetLogs()

	for name := range loggers {
		msg := fmt.Sprintf("Test message from %s", name)
		if !containsString(logs, msg) {
			t.Errorf("Expected log entry for %s logger, but none matched", name)
		}
	}
}

// TestEnvironmentVariableIntegration tests environment variable configuration
func TestEnvironmentVariableIntegration(t *testing.T) {
	tests := []struct {
		name                string
		debugEnv            string
		componentsEnv       string
		providerEnv         string
		expectedGlobalDebug bool
		expectedComponents  []string
	}{
		{
			name:                "DEBUG=1",
			debugEnv:            "1",
			componentsEnv:       "",
			providerEnv:         "",
			expectedGlobalDebug: true,
			expectedComponents:  nil,
		},
		{
			name:                "DEBUG=true",
			debugEnv:            "true",
			componentsEnv:       "",
			providerEnv:         "",
			expectedGlobalDebug: true,
			expectedComponents:  nil,
		},
		{
			name:                "DEBUG_COMPONENTS=handler,config",
			debugEnv:            "",
			componentsEnv:       "handler,config",
			providerEnv:         "",
			expectedGlobalDebug: false,
			expectedComponents:  []string{"handler", "config"},
		},
		{
			name:                "DEBUG_PROVIDER=1",
			debugEnv:            "",
			componentsEnv:       "",
			providerEnv:         "1",
			expectedGlobalDebug: false,
			expectedComponents:  []string{"provider", "handler", "discovery"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			withCleanConfig(t)
			// Set environment variables
			t.Setenv("DEBUG", test.debugEnv)
			t.Setenv("DEBUG_COMPONENTS", test.componentsEnv)
			t.Setenv("DEBUG_PROVIDER", test.providerEnv)

			// Reload configuration
			loadEnvironmentConfig()

			// Check global debug status
			if GetGlobalDebugStatus() != test.expectedGlobalDebug {
				t.Errorf("Expected global debug %v, got %v", test.expectedGlobalDebug, GetGlobalDebugStatus())
			}

			// Check component-specific debug
			if test.expectedComponents != nil {
				for _, component := range test.expectedComponents {
					logger := NewLogger(component)
					if !logger.IsDebugEnabled() {
						t.Errorf("Expected debug to be enabled for component %s", component)
					}
				}
			}
		})
	}
}

// Test Utilities

// MockProvider simulates a provider for testing
type MockProvider struct {
	name    string
	version string
}

func (p *MockProvider) Configure(config map[string]interface{}) error {
	ConfigLogger.Info("Configuring provider %s", p.name)
	ConfigLogger.JSONDebug("Configuration", config)

	if _, ok := config["invalid_field"]; ok {
		err := fmt.Errorf("invalid configuration field")
		ConfigLogger.Error("Configuration failed: %v", err)
		return err
	}

	LogConnectionAttempt(ConnectionLogger, "postgres://user:***@localhost:5432/db", nil)
	return nil
}

func (p *MockProvider) CreateResource(req *MockCreateRequest) (*MockCreateResponse, error) {
	LogRequest(HandlerLogger, "CreateResource", req)

	if req.Name == "" {
		err := fmt.Errorf("resource name cannot be empty")
		LogResponse(HandlerLogger, "CreateResource", nil, err)
		return nil, err
	}

	resp := &MockCreateResponse{
		ID:   fmt.Sprintf("%s_%s", req.Type, req.Name),
		Name: req.Name,
	}

	LogResponse(HandlerLogger, "CreateResource", resp, nil)
	return resp, nil
}

func (p *MockProvider) LogDebugInfo(message string) {
	ProviderLogger.Debug(message)
}

type MockCreateRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type MockCreateResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type MockDiscoveryRequest struct {
	Query string                 `json:"query"`
	Data  map[string]interface{} `json:"data"`
}

// testLogCapture captures log output for testing
type testLogCapture struct {
	mu   sync.Mutex
	logs []string
}

func (c *testLogCapture) Write(p []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logs = append(c.logs, string(p))
	return len(p), nil
}

func (c *testLogCapture) GetLogs() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]string, len(c.logs))
	copy(result, c.logs)
	return result
}

func (c *testLogCapture) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logs = nil
}

var (
	originalLogOutput               = log.Writer()
	originalConsoleOutput io.Writer = consoleLogger.Writer()
)

func setupLogCapture(capture *testLogCapture) {
	log.SetOutput(capture)
	originalConsoleOutput = consoleLogger.Writer()
	consoleLogger.SetOutput(capture)
}

func restoreLogOutput() {
	log.SetOutput(originalLogOutput)
	if originalConsoleOutput != nil {
		consoleLogger.SetOutput(originalConsoleOutput)
	}
}

// Helper functions for test assertions

func containsLog(logs []string, component, message string) bool {
	component = strings.ToUpper(component)
	for _, logLine := range logs {
		if strings.Contains(logLine, component) && strings.Contains(logLine, message) {
			return true
		}
	}
	return false
}

func containsString(logs []string, str string) bool {
	aggregate := strings.Join(logs, "")
	return strings.Contains(aggregate, str)
}

func countLogMatches(logs []string, pattern string) int {
	count := 0
	for _, log := range logs {
		if strings.Contains(log, pattern) {
			count++
		}
	}
	return count
}
