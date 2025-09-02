package state

import (
	"context"
	"testing"
	"time"

	"github.com/schemabounce/kolumn/sdk/types"
)

func TestDefaultManager_Initialize(t *testing.T) {
	// Test basic initialization
	config := DefaultManagerConfig()
	manager := NewManager(config)

	err := manager.Initialize(context.Background(), map[string]interface{}{
		"backend_type":   "memory",
		"workspace_name": "test",
		"environment":    "test",
	})

	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	if !manager.initialized {
		t.Error("Manager should be initialized")
	}

	// Test double initialization should fail
	err = manager.Initialize(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("Double initialization should fail")
	}
}

func TestDefaultManager_StateOperations(t *testing.T) {
	// Setup manager
	config := DefaultManagerConfig()
	manager := NewManager(config)

	err := manager.Initialize(context.Background(), map[string]interface{}{
		"backend_type": "memory",
	})
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Create test state
	testState := &types.UniversalState{
		Version:          1,
		TerraformVersion: "1.0.0",
		Serial:           1,
		Lineage:          "test-lineage",
		Resources:        []types.UniversalResource{},
		Providers:        make(map[string]types.ProviderState),
		Dependencies:     []types.Dependency{},
		Metadata: types.StateMetadata{
			Format:        "kolumn",
			FormatVersion: "1.0",
			Generator:     "kolumn-sdk",
			Workspace:     "test",
			Environment:   "test",
		},
		Governance: types.GovernanceState{},
		Checksums:  make(map[string]string),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	ctx := context.Background()
	stateName := "test-state"

	// Test PutState
	err = manager.PutState(ctx, stateName, testState)
	if err != nil {
		t.Fatalf("Failed to put state: %v", err)
	}

	// Test GetState
	retrievedState, err := manager.GetState(ctx, stateName)
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}

	if retrievedState.Lineage != testState.Lineage {
		t.Errorf("Expected lineage %s, got %s", testState.Lineage, retrievedState.Lineage)
	}

	// Test ListStates
	states, err := manager.ListStates(ctx)
	if err != nil {
		t.Fatalf("Failed to list states: %v", err)
	}

	found := false
	for _, name := range states {
		if name == stateName {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("State %s not found in list", stateName)
	}

	// Test DeleteState
	err = manager.DeleteState(ctx, stateName)
	if err != nil {
		t.Fatalf("Failed to delete state: %v", err)
	}

	// Verify state is deleted
	_, err = manager.GetState(ctx, stateName)
	if err == nil {
		t.Error("Expected error when getting deleted state")
	}
}

func TestDefaultManager_AdapterRegistration(t *testing.T) {
	manager := NewManager(DefaultManagerConfig())

	// Create mock adapter
	mockAdapter := &mockStateAdapter{}

	// Test registration
	err := manager.RegisterAdapter("test-provider", mockAdapter)
	if err != nil {
		t.Fatalf("Failed to register adapter: %v", err)
	}

	// Test retrieval
	adapter, err := manager.GetAdapter("test-provider")
	if err != nil {
		t.Fatalf("Failed to get adapter: %v", err)
	}

	if adapter != mockAdapter {
		t.Error("Retrieved adapter is not the same as registered")
	}

	// Test non-existent adapter
	_, err = manager.GetAdapter("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent adapter")
	}
}

func TestDefaultManager_ImportExport(t *testing.T) {
	// Setup manager
	config := DefaultManagerConfig()
	manager := NewManager(config)

	err := manager.Initialize(context.Background(), map[string]interface{}{
		"backend_type": "memory",
	})
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Create test state
	testState := &types.UniversalState{
		Version:      1,
		Serial:       1,
		Lineage:      "test-lineage",
		Resources:    []types.UniversalResource{},
		Providers:    make(map[string]types.ProviderState),
		Dependencies: []types.Dependency{},
		Metadata:     types.StateMetadata{Format: "kolumn"},
		Governance:   types.GovernanceState{},
		Checksums:    make(map[string]string),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	ctx := context.Background()
	stateName := "test-export-state"

	// Store state
	err = manager.PutState(ctx, stateName, testState)
	if err != nil {
		t.Fatalf("Failed to put state: %v", err)
	}

	// Test Export
	exportData, err := manager.Export(ctx, stateName)
	if err != nil {
		t.Fatalf("Failed to export state: %v", err)
	}

	if len(exportData) == 0 {
		t.Error("Export data should not be empty")
	}

	// Test Import to new state
	importStateName := "test-import-state"
	err = manager.Import(ctx, importStateName, exportData)
	if err != nil {
		t.Fatalf("Failed to import state: %v", err)
	}

	// Verify imported state
	importedState, err := manager.GetState(ctx, importStateName)
	if err != nil {
		t.Fatalf("Failed to get imported state: %v", err)
	}

	if importedState.Lineage != testState.Lineage {
		t.Errorf("Expected lineage %s, got %s", testState.Lineage, importedState.Lineage)
	}
}

func TestDefaultManager_Locking(t *testing.T) {
	// Setup manager
	config := DefaultManagerConfig()
	manager := NewManager(config)

	err := manager.Initialize(context.Background(), map[string]interface{}{
		"backend_type": "memory",
	})
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	ctx := context.Background()
	stateName := "test-lock-state"

	lockInfo := &LockInfo{
		ID:        "test-lock-id",
		Path:      stateName,
		Who:       "test-user",
		Version:   "1",
		Created:   time.Now().Format(time.RFC3339),
		Reason:    "testing",
		Operation: "apply",
	}

	// Test Lock
	lockID, err := manager.Lock(ctx, stateName, lockInfo)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	if lockID == "" {
		t.Error("Lock ID should not be empty")
	}

	// Test Unlock
	err = manager.Unlock(ctx, lockID, lockInfo)
	if err != nil {
		t.Fatalf("Failed to release lock: %v", err)
	}
}

func TestDefaultManager_Backup(t *testing.T) {
	// Setup manager
	config := DefaultManagerConfig()
	manager := NewManager(config)

	err := manager.Initialize(context.Background(), map[string]interface{}{
		"backend_type": "memory",
	})
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Create test state
	testState := &types.UniversalState{
		Version:      1,
		Serial:       1,
		Lineage:      "test-lineage",
		Resources:    []types.UniversalResource{},
		Providers:    make(map[string]types.ProviderState),
		Dependencies: []types.Dependency{},
		Metadata:     types.StateMetadata{Format: "kolumn"},
		Governance:   types.GovernanceState{},
		Checksums:    make(map[string]string),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	ctx := context.Background()
	stateName := "test-backup-state"

	// Store state
	err = manager.PutState(ctx, stateName, testState)
	if err != nil {
		t.Fatalf("Failed to put state: %v", err)
	}

	// Test Backup
	backupID, err := manager.Backup(ctx, stateName)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	if backupID == "" {
		t.Error("Backup ID should not be empty")
	}

	// Verify backup exists
	backupState, err := manager.GetState(ctx, backupID)
	if err != nil {
		t.Fatalf("Failed to get backup state: %v", err)
	}

	if backupState.Lineage != testState.Lineage {
		t.Errorf("Expected backup lineage %s, got %s", testState.Lineage, backupState.Lineage)
	}

	// Test Restore
	// First modify the original state
	testState.Serial = 2
	err = manager.PutState(ctx, stateName, testState)
	if err != nil {
		t.Fatalf("Failed to update state: %v", err)
	}

	// Restore from backup
	err = manager.Restore(ctx, stateName, backupID)
	if err != nil {
		t.Fatalf("Failed to restore state: %v", err)
	}

	// Verify restored state
	restoredState, err := manager.GetState(ctx, stateName)
	if err != nil {
		t.Fatalf("Failed to get restored state: %v", err)
	}

	// The serial number will be incremented when we restore, so we expect it to be higher
	if restoredState.Serial <= 1 {
		t.Errorf("Expected restored serial > 1, got %d", restoredState.Serial)
	}
}

// Mock adapter for testing
type mockStateAdapter struct{}

func (m *mockStateAdapter) ToUniversalState(providerState interface{}) (*types.UniversalState, error) {
	return &types.UniversalState{}, nil
}

func (m *mockStateAdapter) FromUniversalState(universalState *types.UniversalState) (interface{}, error) {
	return map[string]interface{}{}, nil
}

func (m *mockStateAdapter) ExtractDependencies(providerState interface{}) ([]types.Dependency, error) {
	return []types.Dependency{}, nil
}

func (m *mockStateAdapter) ValidateState(state *types.UniversalState) error {
	return nil
}

func (m *mockStateAdapter) SerializeState(state *types.UniversalState) ([]byte, error) {
	return []byte("{}"), nil
}

func (m *mockStateAdapter) DeserializeState(data []byte) (*types.UniversalState, error) {
	return &types.UniversalState{}, nil
}
