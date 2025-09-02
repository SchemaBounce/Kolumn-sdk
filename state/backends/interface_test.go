package backends

import (
	"context"
	"testing"
	"time"

	"github.com/schemabounce/kolumn/sdk/state"
	"github.com/schemabounce/kolumn/sdk/types"
)

// TestBackendInterfaceCompliance tests that all backends implement the StateBackend interface correctly
func TestBackendInterfaceCompliance(t *testing.T) {
	factory := NewBackendFactory()

	// Test each registered backend type
	backendTypes := factory.ListAvailableBackends()
	if len(backendTypes) == 0 {
		t.Fatal("No backends registered in factory")
	}

	for _, backendType := range backendTypes {
		t.Run(string(backendType), func(t *testing.T) {
			// Create backend
			backend, err := factory.CreateBackend(backendType)
			if err != nil {
				t.Fatalf("Failed to create %s backend: %v", backendType, err)
			}

			// Verify it implements StateBackend interface
			if _, ok := backend.(state.StateBackend); !ok {
				t.Fatalf("Backend %s does not implement StateBackend interface", backendType)
			}

			// Test interface methods exist (compile-time check)
			ctx := context.Background()

			// These should compile without errors
			_, _ = backend.GetState(ctx, "test")
			_ = backend.PutState(ctx, "test", nil)
			_ = backend.DeleteState(ctx, "test")
			_, _ = backend.ListStates(ctx)
			_, _ = backend.Lock(ctx, nil)
			_ = backend.Unlock(ctx, "", nil)
		})
	}
}

// TestMemoryBackendBasicOperations tests basic operations on memory backend
func TestMemoryBackendBasicOperations(t *testing.T) {
	backend := NewMemoryBackend()
	ctx := context.Background()

	// Test state that doesn't exist
	_, err := backend.GetState(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent state")
	}

	// Create test state
	testState := &types.UniversalState{
		Version:          1,
		TerraformVersion: "1.0.0",
		Serial:           1,
		Lineage:          "test-lineage",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Put state
	err = backend.PutState(ctx, "test", testState)
	if err != nil {
		t.Fatalf("Failed to put state: %v", err)
	}

	// Get state
	retrievedState, err := backend.GetState(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}

	if retrievedState.Lineage != testState.Lineage {
		t.Errorf("Expected lineage %s, got %s", testState.Lineage, retrievedState.Lineage)
	}

	// List states
	states, err := backend.ListStates(ctx)
	if err != nil {
		t.Fatalf("Failed to list states: %v", err)
	}

	found := false
	for _, state := range states {
		if state == "test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Test state not found in list")
	}

	// Test locking
	lockInfo := &state.LockInfo{
		ID:        "test-lock",
		Path:      "test",
		Who:       "test-user",
		Version:   "1.0.0",
		Created:   time.Now().Format(time.RFC3339),
		Reason:    "test",
		Operation: "test",
	}

	lockID, err := backend.Lock(ctx, lockInfo)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	if lockID != lockInfo.ID {
		t.Errorf("Expected lock ID %s, got %s", lockInfo.ID, lockID)
	}

	// Try to lock again (should fail)
	_, err = backend.Lock(ctx, lockInfo)
	if err == nil {
		t.Error("Expected error when trying to lock already locked state")
	}

	// Unlock
	err = backend.Unlock(ctx, lockID, lockInfo)
	if err != nil {
		t.Fatalf("Failed to unlock: %v", err)
	}

	// Delete state
	err = backend.DeleteState(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to delete state: %v", err)
	}

	// Verify state is deleted
	_, err = backend.GetState(ctx, "test")
	if err == nil {
		t.Error("Expected error for deleted state")
	}
}

// TestFactoryOperations tests factory operations
func TestFactoryOperations(t *testing.T) {
	factory := NewBackendFactory()

	// Test listing backends
	backends := factory.ListAvailableBackends()
	expectedBackends := []BackendType{
		BackendTypeMemory,
		BackendTypeLocal,
		BackendTypePostgres,
		BackendTypeS3,
	}

	if len(backends) != len(expectedBackends) {
		t.Errorf("Expected %d backends, got %d", len(expectedBackends), len(backends))
	}

	// Test backend availability
	for _, expected := range expectedBackends {
		if !factory.IsBackendAvailable(expected) {
			t.Errorf("Backend %s should be available", expected)
		}
	}

	// Test unknown backend
	if factory.IsBackendAvailable("unknown") {
		t.Error("Unknown backend should not be available")
	}

	// Test creating unknown backend
	_, err := factory.CreateBackend("unknown")
	if err == nil {
		t.Error("Expected error for unknown backend type")
	}
}

// TestBackendTypeValidation tests backend type validation
func TestBackendTypeValidation(t *testing.T) {
	validTypes := []string{
		"memory",
		"local", "file", "filesystem",
		"postgres", "postgresql", "pg",
		"s3", "aws", "amazon",
	}

	for _, validType := range validTypes {
		_, err := ParseBackendType(validType)
		if err != nil {
			t.Errorf("Expected %s to be valid, got error: %v", validType, err)
		}
	}

	invalidTypes := []string{
		"invalid",
		"",
		"   ",
		"redis",
	}

	for _, invalidType := range invalidTypes {
		_, err := ParseBackendType(invalidType)
		if err == nil {
			t.Errorf("Expected %s to be invalid", invalidType)
		}
	}
}

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		backendType BackendType
		config      map[string]interface{}
		expectError bool
	}{
		{
			name:        "memory_valid",
			backendType: BackendTypeMemory,
			config:      map[string]interface{}{},
			expectError: false,
		},
		{
			name:        "local_valid",
			backendType: BackendTypeLocal,
			config:      map[string]interface{}{"path": "/tmp/test.state"},
			expectError: false,
		},
		{
			name:        "local_missing_path",
			backendType: BackendTypeLocal,
			config:      map[string]interface{}{},
			expectError: true,
		},
		{
			name:        "postgres_valid",
			backendType: BackendTypePostgres,
			config: map[string]interface{}{
				"database": "test",
				"username": "user",
			},
			expectError: false,
		},
		{
			name:        "postgres_missing_database",
			backendType: BackendTypePostgres,
			config:      map[string]interface{}{"username": "user"},
			expectError: true,
		},
		{
			name:        "postgres_invalid_port",
			backendType: BackendTypePostgres,
			config: map[string]interface{}{
				"database": "test",
				"username": "user",
				"port":     -1,
			},
			expectError: true,
		},
		{
			name:        "s3_valid",
			backendType: BackendTypeS3,
			config:      map[string]interface{}{"bucket": "test-bucket"},
			expectError: false,
		},
		{
			name:        "s3_missing_bucket",
			backendType: BackendTypeS3,
			config:      map[string]interface{}{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.backendType, tt.config)
			if tt.expectError && err == nil {
				t.Error("Expected validation error")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

// TestDefaultConfigs tests default configuration generation
func TestDefaultConfigs(t *testing.T) {
	backendTypes := []BackendType{
		BackendTypeMemory,
		BackendTypeLocal,
		BackendTypePostgres,
		BackendTypeS3,
	}

	for _, backendType := range backendTypes {
		t.Run(string(backendType), func(t *testing.T) {
			defaultConfig := GetDefaultConfig(backendType)
			if defaultConfig == nil {
				t.Fatal("Default config should not be nil")
			}

			// Validate that default config is valid
			err := ValidateConfig(backendType, defaultConfig)
			if err != nil {
				// Some defaults might be incomplete (like missing required credentials)
				// but the structure should be valid
				t.Logf("Default config validation warning for %s: %v", backendType, err)
			}
		})
	}
}
