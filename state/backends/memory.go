package backends

import (
	"context"
	"fmt"
	"sync"

	"github.com/schemabounce/kolumn/sdk/state"
	"github.com/schemabounce/kolumn/sdk/types"
)

// MemoryBackend implements state storage in memory for testing
type MemoryBackend struct {
	states map[string]*types.UniversalState
	locks  map[string]*state.LockInfo
	mutex  sync.RWMutex
}

// NewMemoryBackend creates a new memory backend
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		states: make(map[string]*types.UniversalState),
		locks:  make(map[string]*state.LockInfo),
	}
}

// GetState retrieves state by name
func (b *MemoryBackend) GetState(ctx context.Context, name string) (*types.UniversalState, error) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	state, exists := b.states[name]
	if !exists {
		return nil, fmt.Errorf("state '%s' not found", name)
	}

	// Return a copy to prevent mutations
	return b.copyState(state), nil
}

// PutState stores state by name
func (b *MemoryBackend) PutState(ctx context.Context, name string, st *types.UniversalState) error {
	if st == nil {
		return fmt.Errorf("state cannot be nil")
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Store a copy to prevent mutations
	b.states[name] = b.copyState(st)
	return nil
}

// DeleteState removes state by name
func (b *MemoryBackend) DeleteState(ctx context.Context, name string) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	delete(b.states, name)
	return nil
}

// ListStates lists all available states
func (b *MemoryBackend) ListStates(ctx context.Context) ([]string, error) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	states := make([]string, 0, len(b.states))
	for name := range b.states {
		states = append(states, name)
	}
	return states, nil
}

// Lock acquires a lock on the state
func (b *MemoryBackend) Lock(ctx context.Context, info *state.LockInfo) (string, error) {
	if info == nil {
		return "", fmt.Errorf("lock info cannot be nil")
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Check if already locked
	if existingLock, exists := b.locks[info.Path]; exists {
		return "", fmt.Errorf("state is already locked by %s (ID: %s)", existingLock.Who, existingLock.ID)
	}

	// Store the lock
	b.locks[info.Path] = &state.LockInfo{
		ID:        info.ID,
		Path:      info.Path,
		Who:       info.Who,
		Version:   info.Version,
		Created:   info.Created,
		Reason:    info.Reason,
		Operation: info.Operation,
	}

	return info.ID, nil
}

// Unlock releases a lock on the state
func (b *MemoryBackend) Unlock(ctx context.Context, lockID string, info *state.LockInfo) error {
	if info == nil {
		return fmt.Errorf("lock info cannot be nil")
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Check if lock exists and matches
	existingLock, exists := b.locks[info.Path]
	if !exists {
		// Already unlocked, which is fine
		return nil
	}

	if existingLock.ID != lockID {
		return fmt.Errorf("lock ID mismatch: expected %s, got %s", existingLock.ID, lockID)
	}

	// Remove the lock
	delete(b.locks, info.Path)
	return nil
}

// copyState creates a deep copy of the state
func (b *MemoryBackend) copyState(original *types.UniversalState) *types.UniversalState {
	if original == nil {
		return nil
	}

	// Create a new state with the same basic fields
	copy := &types.UniversalState{
		Version:          original.Version,
		TerraformVersion: original.TerraformVersion,
		Serial:           original.Serial,
		Lineage:          original.Lineage,
		CreatedAt:        original.CreatedAt,
		UpdatedAt:        original.UpdatedAt,
	}

	// Copy resources
	if original.Resources != nil {
		copy.Resources = make([]types.UniversalResource, len(original.Resources))
		for i, resource := range original.Resources {
			copy.Resources[i] = b.copyResource(resource)
		}
	}

	// Copy providers
	if original.Providers != nil {
		copy.Providers = make(map[string]types.ProviderState)
		for key, provider := range original.Providers {
			copy.Providers[key] = b.copyProviderState(provider)
		}
	}

	// Copy dependencies
	if original.Dependencies != nil {
		copy.Dependencies = make([]types.Dependency, len(original.Dependencies))
		for i, dep := range original.Dependencies {
			copy.Dependencies[i] = types.Dependency{
				ID:             dep.ID,
				ResourceID:     dep.ResourceID,
				DependsOnID:    dep.DependsOnID,
				DependencyType: dep.DependencyType,
				Constraint:     dep.Constraint,
				Optional:       dep.Optional,
			}
		}
	}

	// Copy metadata
	copy.Metadata = b.copyMetadata(original.Metadata)

	// Copy governance
	copy.Governance = b.copyGovernance(original.Governance)

	// Copy checksums
	if original.Checksums != nil {
		copy.Checksums = make(map[string]string)
		for key, value := range original.Checksums {
			copy.Checksums[key] = value
		}
	}

	return copy
}

// copyResource creates a copy of a resource
func (b *MemoryBackend) copyResource(original types.UniversalResource) types.UniversalResource {
	resource := types.UniversalResource{
		ID:            original.ID,
		Type:          original.Type,
		Name:          original.Name,
		Provider:      original.Provider,
		Mode:          original.Mode,
		ProviderState: original.ProviderState, // RawMessage is already immutable
	}

	// Copy depends_on
	if original.DependsOn != nil {
		resource.DependsOn = make([]string, len(original.DependsOn))
		copy(resource.DependsOn, original.DependsOn)
	}

	// Copy references
	if original.References != nil {
		resource.References = make([]types.ResourceReference, len(original.References))
		copy(resource.References, original.References)
	}

	// Copy instances
	if original.Instances != nil {
		resource.Instances = make([]types.ResourceInstance, len(original.Instances))
		for i, instance := range original.Instances {
			resource.Instances[i] = b.copyResourceInstance(instance)
		}
	}

	// Copy metadata
	if original.Metadata != nil {
		resource.Metadata = make(types.ResourceMetadata)
		for key, value := range original.Metadata {
			resource.Metadata[key] = value
		}
	}

	// Copy classifications
	if original.Classifications != nil {
		resource.Classifications = make([]string, len(original.Classifications))
		copy(resource.Classifications, original.Classifications)
	}

	resource.Compliance = original.Compliance // ComplianceStatus is copyable by value

	return resource
}

// copyResourceInstance creates a copy of a resource instance
func (b *MemoryBackend) copyResourceInstance(original types.ResourceInstance) types.ResourceInstance {
	instance := types.ResourceInstance{
		IndexKey:            original.IndexKey,
		Status:              original.Status,
		Private:             original.Private, // RawMessage is already immutable
		Tainted:             original.Tainted,
		Deposed:             original.Deposed,
		CreateBeforeDestroy: original.CreateBeforeDestroy,
	}

	// Copy attributes
	if original.Attributes != nil {
		instance.Attributes = make(map[string]interface{})
		for key, value := range original.Attributes {
			instance.Attributes[key] = value
		}
	}

	// Copy metadata
	if original.Metadata != nil {
		instance.Metadata = make(types.ResourceMetadata)
		for key, value := range original.Metadata {
			instance.Metadata[key] = value
		}
	}

	return instance
}

// copyProviderState creates a copy of provider state
func (b *MemoryBackend) copyProviderState(original types.ProviderState) types.ProviderState {
	provider := types.ProviderState{
		Version: original.Version,
	}

	// Copy config
	if original.Config != nil {
		provider.Config = make(map[string]interface{})
		for key, value := range original.Config {
			provider.Config[key] = value
		}
	}

	// Copy metadata
	if original.Metadata != nil {
		provider.Metadata = make(map[string]interface{})
		for key, value := range original.Metadata {
			provider.Metadata[key] = value
		}
	}

	return provider
}

// copyMetadata creates a copy of state metadata
func (b *MemoryBackend) copyMetadata(original types.StateMetadata) types.StateMetadata {
	metadata := types.StateMetadata{
		Format:           original.Format,
		FormatVersion:    original.FormatVersion,
		Generator:        original.Generator,
		GeneratorVersion: original.GeneratorVersion,
		CreatedBy:        original.CreatedBy,
		Environment:      original.Environment,
		Workspace:        original.Workspace,
	}

	// Copy tags
	if original.Tags != nil {
		metadata.Tags = make(map[string]string)
		for key, value := range original.Tags {
			metadata.Tags[key] = value
		}
	}

	// Copy custom
	if original.Custom != nil {
		metadata.Custom = make(map[string]interface{})
		for key, value := range original.Custom {
			metadata.Custom[key] = value
		}
	}

	return metadata
}

// copyGovernance creates a copy of governance state
func (b *MemoryBackend) copyGovernance(original types.GovernanceState) types.GovernanceState {
	governance := types.GovernanceState{
		ComplianceStatus: original.ComplianceStatus, // ComplianceStatus is copyable by value
		DataLineage:      original.DataLineage,      // DataLineageState is copyable by value
	}

	// Copy classifications
	if original.Classifications != nil {
		governance.Classifications = make([]types.ClassificationState, len(original.Classifications))
		for i, classification := range original.Classifications {
			governance.Classifications[i] = b.copyClassificationState(classification)
		}
	}

	// Copy policies
	if original.Policies != nil {
		governance.Policies = make([]types.PolicyState, len(original.Policies))
		for i, policy := range original.Policies {
			governance.Policies[i] = b.copyPolicyState(policy)
		}
	}

	return governance
}

// copyClassificationState creates a copy of classification state
func (b *MemoryBackend) copyClassificationState(original types.ClassificationState) types.ClassificationState {
	classification := types.ClassificationState{
		ID:          original.ID,
		Name:        original.Name,
		Level:       original.Level,
		LastUpdated: original.LastUpdated,
	}

	// Copy resources
	if original.Resources != nil {
		classification.Resources = make([]string, len(original.Resources))
		copy(classification.Resources, original.Resources)
	}

	// Copy policies
	if original.Policies != nil {
		classification.Policies = make([]string, len(original.Policies))
		copy(classification.Policies, original.Policies)
	}

	// Copy metadata
	if original.Metadata != nil {
		classification.Metadata = make(map[string]interface{})
		for key, value := range original.Metadata {
			classification.Metadata[key] = value
		}
	}

	return classification
}

// copyPolicyState creates a copy of policy state
func (b *MemoryBackend) copyPolicyState(original types.PolicyState) types.PolicyState {
	policy := types.PolicyState{
		ID:            original.ID,
		Name:          original.Name,
		Type:          original.Type,
		Status:        original.Status,
		LastEvaluated: original.LastEvaluated,
	}

	// Copy resources
	if original.Resources != nil {
		policy.Resources = make([]string, len(original.Resources))
		copy(policy.Resources, original.Resources)
	}

	// Copy violations
	if original.Violations != nil {
		policy.Violations = make([]types.PolicyViolation, len(original.Violations))
		copy(policy.Violations, original.Violations)
	}

	return policy
}
