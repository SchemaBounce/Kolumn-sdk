package state

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/schemabounce/kolumn/sdk/types"
)

func TestIntegrationExample(t *testing.T) {
	ctx := context.Background()

	// Create manager with default configuration
	config := DefaultManagerConfig()
	config.BackendType = "memory"
	config.WorkspaceName = "example"
	config.Environment = "development"

	manager := NewManager(config)

	// Initialize the manager
	err := manager.Initialize(ctx, map[string]interface{}{
		"enable_drift_detection": true,
	})
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Create a sample state
	state := &types.UniversalState{
		Version:          1,
		TerraformVersion: "1.0.0",
		Serial:           1,
		Lineage:          "example-lineage-12345",
		Resources: []types.UniversalResource{
			{
				ID:       "postgres.table.users",
				Type:     "table",
				Name:     "users",
				Provider: "postgres",
				Mode:     types.ResourceModeManaged,
				Instances: []types.ResourceInstance{
					{
						Status: types.StatusReady,
						Attributes: map[string]interface{}{
							"name":   "users",
							"schema": "public",
						},
						Metadata: map[string]interface{}{
							"created_by": "kolumn-sdk",
						},
					},
				},
				DependsOn:  []string{},
				References: []types.ResourceReference{},
				Metadata:   map[string]interface{}{},
			},
		},
		Providers: map[string]types.ProviderState{
			"postgres": {
				Config: map[string]interface{}{
					"host": "localhost",
				},
				Version: "1.0.0",
			},
		},
		Dependencies: []types.Dependency{},
		Metadata: types.StateMetadata{
			Format:    "kolumn",
			Workspace: "example",
		},
		Governance: types.GovernanceState{},
		Checksums:  map[string]string{},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Store the state
	stateName := "example-state"
	err = manager.PutState(ctx, stateName, state)
	if err != nil {
		t.Fatalf("Failed to store state: %v", err)
	}

	t.Logf("✓ Stored state '%s' with %d resources", stateName, len(state.Resources))

	// Retrieve the state
	retrievedState, err := manager.GetState(ctx, stateName)
	if err != nil {
		t.Fatalf("Failed to retrieve state: %v", err)
	}

	t.Logf("✓ Retrieved state '%s' with lineage: %s", stateName, retrievedState.Lineage)

	// List all states
	states, err := manager.ListStates(ctx)
	if err != nil {
		t.Fatalf("Failed to list states: %v", err)
	}

	t.Logf("✓ Found %d states: %v", len(states), states)

	// Create a backup
	backupID, err := manager.Backup(ctx, stateName)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	t.Logf("✓ Created backup with ID: %s", backupID)

	// Export state
	exportData, err := manager.Export(ctx, stateName)
	if err != nil {
		t.Fatalf("Failed to export state: %v", err)
	}

	t.Logf("✓ Exported state data (%d bytes)", len(exportData))

	// Test drift detection
	driftDetector := manager.GetDriftDetector()
	if driftDetector != nil {
		driftAnalysis, err := driftDetector.DetectDrift(ctx, retrievedState)
		if err != nil {
			t.Logf("Drift detection failed: %v", err)
		} else {
			t.Logf("✓ Drift detection completed - Has drift: %t, Items: %d",
				driftAnalysis.HasDrift, len(driftAnalysis.DriftItems))
		}
	}

	// Test collection manager
	collectionMgr := manager.GetCollectionManager()
	if collectionMgr != nil {
		t.Log("✓ Collection manager available")
	}

	// Test dependency manager
	depMgr := manager.GetDependencyManager()
	if depMgr != nil {
		// Analyze dependencies
		analysis, err := depMgr.AnalyzeGraph(ctx, stateName)
		if err != nil {
			t.Logf("Dependency analysis failed: %v", err)
		} else {
			t.Logf("✓ Dependency analysis - Nodes: %d, Edges: %d",
				analysis.NodeCount, analysis.EdgeCount)
		}
	}

	t.Log("✓ Integration test completed successfully!")
}

func TestCustomAdapterIntegration(t *testing.T) {
	ctx := context.Background()

	// Create manager
	manager := NewManager(DefaultManagerConfig())

	// Initialize with memory backend
	err := manager.Initialize(ctx, map[string]interface{}{
		"backend_type": "memory",
	})
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Create and register a custom adapter
	customAdapter := &testAdapter{}
	err = manager.RegisterAdapter("test-provider", customAdapter)
	if err != nil {
		t.Fatalf("Failed to register adapter: %v", err)
	}

	// Get the adapter
	adapter, err := manager.GetAdapter("test-provider")
	if err != nil {
		t.Fatalf("Failed to get adapter: %v", err)
	}

	// Use the adapter
	testState := &types.UniversalState{
		Version:      1,
		Serial:       1,
		Lineage:      "test",
		Resources:    []types.UniversalResource{},
		Providers:    make(map[string]types.ProviderState),
		Dependencies: []types.Dependency{},
		Metadata:     types.StateMetadata{},
		Governance:   types.GovernanceState{},
		Checksums:    make(map[string]string),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	providerState, err := adapter.FromUniversalState(testState)
	if err != nil {
		t.Fatalf("Failed to convert state: %v", err)
	}

	t.Logf("✓ Custom adapter working - Provider state: %+v", providerState)

	// Convert back
	convertedState, err := adapter.ToUniversalState(providerState)
	if err != nil {
		t.Fatalf("Failed to convert back: %v", err)
	}

	t.Logf("✓ Round-trip conversion successful - Version: %d", convertedState.Version)
}

// testAdapter demonstrates a simple state adapter implementation for testing
type testAdapter struct{}

func (t *testAdapter) ToUniversalState(providerState interface{}) (*types.UniversalState, error) {
	return &types.UniversalState{
		Version:      1,
		Serial:       1,
		Lineage:      "test-lineage",
		Resources:    []types.UniversalResource{},
		Providers:    make(map[string]types.ProviderState),
		Dependencies: []types.Dependency{},
		Metadata:     types.StateMetadata{},
		Governance:   types.GovernanceState{},
		Checksums:    make(map[string]string),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

func (t *testAdapter) FromUniversalState(universalState *types.UniversalState) (interface{}, error) {
	return map[string]interface{}{
		"version": universalState.Version,
		"serial":  universalState.Serial,
		"lineage": universalState.Lineage,
	}, nil
}

func (t *testAdapter) ExtractDependencies(providerState interface{}) ([]types.Dependency, error) {
	return []types.Dependency{}, nil
}

func (t *testAdapter) ValidateState(state *types.UniversalState) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	return nil
}

func (t *testAdapter) SerializeState(state *types.UniversalState) ([]byte, error) {
	return []byte("{}"), nil
}

func (t *testAdapter) DeserializeState(data []byte) (*types.UniversalState, error) {
	return &types.UniversalState{}, nil
}
