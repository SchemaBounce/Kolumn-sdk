// Package state provides helper utilities for state management
//
// This package provides utility functions to make implementing
// state backend providers easier.
package state

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// CalculateChecksum calculates a checksum for a UniversalState
func CalculateChecksum(state *UniversalState) (string, error) {
	// Create a normalized representation for checksum calculation
	normalized := struct {
		Version   int                           `json:"version"`
		Resources map[string]*UniversalResource `json:"resources"`
		Providers map[string]*ProviderState     `json:"providers"`
		Metadata  map[string]interface{}        `json:"metadata"`
		Outputs   map[string]interface{}        `json:"outputs"`
	}{
		Version:   state.Version,
		Resources: state.Resources,
		Providers: state.Providers,
		Metadata:  state.Metadata,
		Outputs:   state.Outputs,
	}

	data, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("failed to marshal state for checksum: %w", err)
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}

// ValidateUniversalState performs basic validation on a UniversalState
func ValidateUniversalState(state *UniversalState) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}

	if state.ProviderID == "" {
		return fmt.Errorf("provider ID cannot be empty")
	}

	if state.ProviderType == "" {
		return fmt.Errorf("provider type cannot be empty")
	}

	if state.Resources == nil {
		return fmt.Errorf("resources map cannot be nil")
	}

	// Validate each resource
	for id, resource := range state.Resources {
		if resource == nil {
			return fmt.Errorf("resource %s cannot be nil", id)
		}

		if resource.ID != id {
			return fmt.Errorf("resource %s has mismatched ID %s", id, resource.ID)
		}

		if resource.Type == "" {
			return fmt.Errorf("resource %s has empty type", id)
		}

		if resource.ProviderType == "" {
			return fmt.Errorf("resource %s has empty provider type", id)
		}
	}

	return nil
}

// MergeUniversalStates merges multiple UniversalState objects into one
// The primary state is used as the base, and resources from secondary states are added
func MergeUniversalStates(primary *UniversalState, secondary ...*UniversalState) (*UniversalState, error) {
	if primary == nil {
		return nil, fmt.Errorf("primary state cannot be nil")
	}

	// Clone the primary state
	merged := primary.Clone()

	// Merge resources from secondary states
	for _, state := range secondary {
		if state == nil {
			continue
		}

		for id, resource := range state.Resources {
			// Check for conflicts
			if existing, exists := merged.Resources[id]; exists {
				// If resource exists, use the one with the newer timestamp
				if resource.UpdatedAt.After(existing.UpdatedAt) {
					merged.Resources[id] = resource.Clone()
				}
			} else {
				merged.Resources[id] = resource.Clone()
			}
		}

		// Merge provider states
		for id, provider := range state.Providers {
			if existing, exists := merged.Providers[id]; exists {
				// Use the one with the newer LastSync
				if provider.LastSync.After(existing.LastSync) {
					merged.Providers[id] = provider
				}
			} else {
				merged.Providers[id] = provider
			}
		}

		// Merge metadata (secondary overwrites primary)
		for k, v := range state.Metadata {
			merged.Metadata[k] = v
		}

		// Merge outputs (secondary overwrites primary)
		for k, v := range state.Outputs {
			merged.Outputs[k] = v
		}
	}

	// Update merged state metadata
	merged.LastUpdated = time.Now()
	merged.UpdatedAt = merged.LastUpdated
	merged.Version++

	return merged, nil
}

// CompareUniversalStates compares two UniversalState objects and returns differences
func CompareUniversalStates(old, new *UniversalState) (*StateDiff, error) {
	diff := &StateDiff{
		Added:    make(map[string]*UniversalResource),
		Modified: make(map[string]*ResourceDiff),
		Removed:  make(map[string]*UniversalResource),
	}

	// Compare resources
	oldResources := make(map[string]*UniversalResource)
	if old != nil && old.Resources != nil {
		oldResources = old.Resources
	}

	newResources := make(map[string]*UniversalResource)
	if new != nil && new.Resources != nil {
		newResources = new.Resources
	}

	// Find added and modified resources
	for id, newResource := range newResources {
		if oldResource, exists := oldResources[id]; exists {
			// Check if modified
			if !resourcesEqual(oldResource, newResource) {
				diff.Modified[id] = &ResourceDiff{
					Old: oldResource,
					New: newResource,
				}
			}
		} else {
			// Added resource
			diff.Added[id] = newResource
		}
	}

	// Find removed resources
	for id, oldResource := range oldResources {
		if _, exists := newResources[id]; !exists {
			diff.Removed[id] = oldResource
		}
	}

	return diff, nil
}

// StateDiff represents the differences between two UniversalState objects
type StateDiff struct {
	Added    map[string]*UniversalResource `json:"added"`
	Modified map[string]*ResourceDiff      `json:"modified"`
	Removed  map[string]*UniversalResource `json:"removed"`
}

// ResourceDiff represents the differences for a specific resource
type ResourceDiff struct {
	Old *UniversalResource `json:"old"`
	New *UniversalResource `json:"new"`
}

// HasChanges returns true if the diff contains any changes
func (d *StateDiff) HasChanges() bool {
	return len(d.Added) > 0 || len(d.Modified) > 0 || len(d.Removed) > 0
}

// resourcesEqual compares two UniversalResource objects for equality
func resourcesEqual(a, b *UniversalResource) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Compare basic fields
	if a.ID != b.ID || a.Type != b.Type || a.Name != b.Name ||
		a.ProviderType != b.ProviderType || a.ProviderID != b.ProviderID ||
		a.Version != b.Version || a.Status != b.Status {
		return false
	}

	// Compare data
	aData, _ := json.Marshal(a.Data)
	bData, _ := json.Marshal(b.Data)
	if string(aData) != string(bData) {
		return false
	}

	// Compare metadata
	aMeta, _ := json.Marshal(a.Metadata)
	bMeta, _ := json.Marshal(b.Metadata)
	if string(aMeta) != string(bMeta) {
		return false
	}

	return true
}

// CreateStateSnapshot creates a point-in-time snapshot of a UniversalState
func CreateStateSnapshot(state *UniversalState) (*StateSnapshot, error) {
	if state == nil {
		return nil, fmt.Errorf("state cannot be nil")
	}

	checksum, err := CalculateChecksum(state)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	snapshot := &StateSnapshot{
		ID:          fmt.Sprintf("snapshot-%d", time.Now().Unix()),
		Timestamp:   time.Now(),
		State:       state.Clone(),
		Checksum:    checksum,
		Description: fmt.Sprintf("Snapshot of state version %d", state.Version),
	}

	return snapshot, nil
}

// StateSnapshot represents a point-in-time snapshot of state
type StateSnapshot struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	State       *UniversalState        `json:"state"`
	Checksum    string                 `json:"checksum"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Validate validates the state snapshot
func (s *StateSnapshot) Validate() error {
	if s.State == nil {
		return fmt.Errorf("snapshot state cannot be nil")
	}

	return ValidateUniversalState(s.State)
}

// BackendProviderHelper provides common functionality for implementing StateBackendProvider
type BackendProviderHelper struct {
	providerID   string
	providerType string
	isPrimary    bool
	isBackend    bool
	config       *StateBackendConfig
}

// NewBackendProviderHelper creates a new helper for implementing StateBackendProvider
func NewBackendProviderHelper(providerID, providerType string, isPrimary, isBackend bool) *BackendProviderHelper {
	return &BackendProviderHelper{
		providerID:   providerID,
		providerType: providerType,
		isPrimary:    isPrimary,
		isBackend:    isBackend,
		config:       DefaultStateBackendConfig(),
	}
}

// GetProviderID returns the provider ID
func (h *BackendProviderHelper) GetProviderID() string {
	return h.providerID
}

// GetProviderType returns the provider type
func (h *BackendProviderHelper) GetProviderType() string {
	return h.providerType
}

// IsPrimary returns whether this is the primary provider
func (h *BackendProviderHelper) IsPrimary() bool {
	return h.isPrimary
}

// IsBackend returns whether this provider acts as a backend
func (h *BackendProviderHelper) IsBackend() bool {
	return h.isBackend
}

// SetConfig sets the backend configuration
func (h *BackendProviderHelper) SetConfig(config *StateBackendConfig) {
	h.config = config
}

// GetConfig returns the backend configuration
func (h *BackendProviderHelper) GetConfig() *StateBackendConfig {
	return h.config
}

// ValidateBasicState performs basic validation on state
func (h *BackendProviderHelper) ValidateBasicState(ctx context.Context, state *UniversalState) error {
	return ValidateUniversalState(state)
}
