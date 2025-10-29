// Package state provides the core state management interfaces for Kolumn providers
//
// This package defines the universal state format and interfaces that enable
// providers to act as state backends in the Kolumn ecosystem.
package state

import (
	"time"
)

// UniversalState represents the complete state of a Kolumn deployment
// This is the universal format for state storage across all providers
type UniversalState struct {
	// Metadata
	Version     int       `json:"version"`
	Checksum    string    `json:"checksum"`
	LastUpdated time.Time `json:"last_updated"`
	CreatedAt   time.Time `json:"created_at"`

	// Provider information
	ProviderID   string `json:"provider_id"`
	ProviderType string `json:"provider_type"`

	// Resources - the core state data
	Resources map[string]*UniversalResource `json:"resources"`

	// Providers - provider state information
	Providers map[string]*ProviderState `json:"providers,omitempty"`

	// UpdatedAt alias for compatibility
	UpdatedAt time.Time `json:"updated_at"`

	// Metadata and lineage
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Dependencies map[string][]string    `json:"dependencies,omitempty"`
	Outputs      map[string]interface{} `json:"outputs,omitempty"`

	// State management
	LockInfo *StateLock           `json:"lock_info,omitempty"`
	Backups  []string             `json:"backups,omitempty"`
	History  []*StateHistoryEntry `json:"history,omitempty"`
}

// UniversalResource represents a single resource in the state
type UniversalResource struct {
	// Core identification
	ID           string `json:"id"`
	Type         string `json:"type"`
	Name         string `json:"name"`
	ProviderType string `json:"provider_type"`
	ProviderID   string `json:"provider_id"`

	// State management
	Version int                    `json:"version"`
	Status  ResourceStatus         `json:"status"`
	Data    map[string]interface{} `json:"data"`

	// Lineage and relationships
	Dependencies []string `json:"dependencies,omitempty"`
	DependsOn    []string `json:"depends_on,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Change tracking
	ChangeInfo *ResourceChangeInfo `json:"change_info,omitempty"`
}

// ResourceStatus represents the status of a resource
type ResourceStatus string

const (
	ResourceStatusUnknown  ResourceStatus = "unknown"
	ResourceStatusCreating ResourceStatus = "creating"
	ResourceStatusActive   ResourceStatus = "active"
	ResourceStatusUpdating ResourceStatus = "updating"
	ResourceStatusDeleting ResourceStatus = "deleting"
	ResourceStatusDeleted  ResourceStatus = "deleted"
	ResourceStatusError    ResourceStatus = "error"
	ResourceStatusDrifted  ResourceStatus = "drifted"
)

// ResourceChangeInfo tracks changes to a resource
type ResourceChangeInfo struct {
	ChangeType    ChangeType             `json:"change_type"`
	Before        map[string]interface{} `json:"before,omitempty"`
	After         map[string]interface{} `json:"after,omitempty"`
	ChangedFields []string               `json:"changed_fields,omitempty"`
	ChangeReason  string                 `json:"change_reason,omitempty"`
	ChangedBy     string                 `json:"changed_by,omitempty"`
	ChangedAt     time.Time              `json:"changed_at"`
}

// ChangeType represents the type of change
type ChangeType string

const (
	ChangeTypeCreate ChangeType = "create"
	ChangeTypeUpdate ChangeType = "update"
	ChangeTypeDelete ChangeType = "delete"
	ChangeTypeNoOp   ChangeType = "no-op"
	ChangeTypeDrift  ChangeType = "drift"
)

// StateLock represents a lock on the state
type StateLock struct {
	ID        string    `json:"id"`
	Operation string    `json:"operation"`
	Info      string    `json:"info"`
	Who       string    `json:"who"`
	Created   time.Time `json:"created"`
	Path      string    `json:"path,omitempty"`
}

// StateHistoryEntry represents an entry in the state history
type StateHistoryEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Operation string                 `json:"operation"`
	User      string                 `json:"user,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Changes   []*ResourceChange      `json:"changes,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Checksum  string                 `json:"checksum"`
}

// ResourceChange represents a change to a resource
type ResourceChange struct {
	ResourceID string                 `json:"resource_id"`
	Action     ChangeType             `json:"action"`
	Before     map[string]interface{} `json:"before,omitempty"`
	After      map[string]interface{} `json:"after,omitempty"`
}

// ProviderState represents the state of a provider
type ProviderState struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Configuration map[string]interface{} `json:"configuration"`
	IsBackend     bool                   `json:"is_backend"`
	IsPrimary     bool                   `json:"is_primary"`
	Status        string                 `json:"status"`
	LastSync      time.Time              `json:"last_sync"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceState represents the state of a resource (alias for UniversalResource for compatibility)
type ResourceState = UniversalResource

// NewUniversalState creates a new UniversalState with default values
func NewUniversalState(providerID, providerType string) *UniversalState {
	now := time.Now()

	return &UniversalState{
		Version:      1,
		LastUpdated:  now,
		UpdatedAt:    now,
		CreatedAt:    now,
		ProviderID:   providerID,
		ProviderType: providerType,
		Resources:    make(map[string]*UniversalResource),
		Providers:    make(map[string]*ProviderState),
		Metadata:     make(map[string]interface{}),
		Dependencies: make(map[string][]string),
		Outputs:      make(map[string]interface{}),
		Backups:      make([]string, 0),
		History:      make([]*StateHistoryEntry, 0),
	}
}

// NewUniversalResource creates a new UniversalResource with default values
func NewUniversalResource(id, resourceType, name, providerType, providerID string) *UniversalResource {
	now := time.Now()

	return &UniversalResource{
		ID:           id,
		Type:         resourceType,
		Name:         name,
		ProviderType: providerType,
		ProviderID:   providerID,
		Version:      1,
		Status:       ResourceStatusCreating,
		Data:         make(map[string]interface{}),
		Dependencies: make([]string, 0),
		DependsOn:    make([]string, 0),
		CreatedAt:    now,
		UpdatedAt:    now,
		Metadata:     make(map[string]interface{}),
	}
}

// AddResource adds a resource to the state
func (us *UniversalState) AddResource(resource *UniversalResource) {
	if us.Resources == nil {
		us.Resources = make(map[string]*UniversalResource)
	}

	us.Resources[resource.ID] = resource
	us.LastUpdated = time.Now()
	us.Version++
}

// RemoveResource removes a resource from the state
func (us *UniversalState) RemoveResource(resourceID string) {
	if us.Resources != nil {
		delete(us.Resources, resourceID)
		us.LastUpdated = time.Now()
		us.Version++
	}
}

// GetResource retrieves a resource by ID
func (us *UniversalState) GetResource(resourceID string) (*UniversalResource, bool) {
	if us.Resources == nil {
		return nil, false
	}

	resource, exists := us.Resources[resourceID]
	return resource, exists
}

// ListResources returns all resources in the state
func (us *UniversalState) ListResources() []*UniversalResource {
	resources := make([]*UniversalResource, 0, len(us.Resources))

	for _, resource := range us.Resources {
		resources = append(resources, resource)
	}

	return resources
}

// Clone creates a deep copy of the UniversalState
func (us *UniversalState) Clone() *UniversalState {
	clone := &UniversalState{
		Version:      us.Version,
		Checksum:     us.Checksum,
		LastUpdated:  us.LastUpdated,
		UpdatedAt:    us.UpdatedAt,
		CreatedAt:    us.CreatedAt,
		ProviderID:   us.ProviderID,
		ProviderType: us.ProviderType,
		Resources:    make(map[string]*UniversalResource),
		Providers:    make(map[string]*ProviderState),
		Metadata:     make(map[string]interface{}),
		Dependencies: make(map[string][]string),
		Outputs:      make(map[string]interface{}),
		Backups:      make([]string, len(us.Backups)),
		History:      make([]*StateHistoryEntry, len(us.History)),
	}

	// Deep copy resources
	for id, resource := range us.Resources {
		clone.Resources[id] = resource.Clone()
	}

	// Deep copy providers
	for id, provider := range us.Providers {
		providerClone := &ProviderState{
			ID:            provider.ID,
			Type:          provider.Type,
			Configuration: make(map[string]interface{}),
			IsBackend:     provider.IsBackend,
			IsPrimary:     provider.IsPrimary,
			Status:        provider.Status,
			LastSync:      provider.LastSync,
			Metadata:      make(map[string]interface{}),
		}

		// Copy configuration
		for k, v := range provider.Configuration {
			providerClone.Configuration[k] = v
		}

		// Copy metadata
		for k, v := range provider.Metadata {
			providerClone.Metadata[k] = v
		}

		clone.Providers[id] = providerClone
	}

	// Copy metadata
	for k, v := range us.Metadata {
		clone.Metadata[k] = v
	}

	// Copy dependencies
	for k, deps := range us.Dependencies {
		clone.Dependencies[k] = make([]string, len(deps))
		copy(clone.Dependencies[k], deps)
	}

	// Copy outputs
	for k, v := range us.Outputs {
		clone.Outputs[k] = v
	}

	// Copy backups
	copy(clone.Backups, us.Backups)

	// Copy history
	copy(clone.History, us.History)

	// Copy lock info
	if us.LockInfo != nil {
		clone.LockInfo = &StateLock{
			ID:        us.LockInfo.ID,
			Operation: us.LockInfo.Operation,
			Info:      us.LockInfo.Info,
			Who:       us.LockInfo.Who,
			Created:   us.LockInfo.Created,
			Path:      us.LockInfo.Path,
		}
	}

	return clone
}

// Clone creates a deep copy of the UniversalResource
func (ur *UniversalResource) Clone() *UniversalResource {
	clone := &UniversalResource{
		ID:           ur.ID,
		Type:         ur.Type,
		Name:         ur.Name,
		ProviderType: ur.ProviderType,
		ProviderID:   ur.ProviderID,
		Version:      ur.Version,
		Status:       ur.Status,
		Data:         make(map[string]interface{}),
		Dependencies: make([]string, len(ur.Dependencies)),
		DependsOn:    make([]string, len(ur.DependsOn)),
		CreatedAt:    ur.CreatedAt,
		UpdatedAt:    ur.UpdatedAt,
		Metadata:     make(map[string]interface{}),
	}

	// Deep copy data
	for k, v := range ur.Data {
		clone.Data[k] = v
	}

	// Copy dependencies
	copy(clone.Dependencies, ur.Dependencies)
	copy(clone.DependsOn, ur.DependsOn)

	// Copy metadata
	for k, v := range ur.Metadata {
		clone.Metadata[k] = v
	}

	// Copy change info
	if ur.ChangeInfo != nil {
		clone.ChangeInfo = &ResourceChangeInfo{
			ChangeType:    ur.ChangeInfo.ChangeType,
			Before:        make(map[string]interface{}),
			After:         make(map[string]interface{}),
			ChangedFields: make([]string, len(ur.ChangeInfo.ChangedFields)),
			ChangeReason:  ur.ChangeInfo.ChangeReason,
			ChangedBy:     ur.ChangeInfo.ChangedBy,
			ChangedAt:     ur.ChangeInfo.ChangedAt,
		}

		// Copy before/after data
		for k, v := range ur.ChangeInfo.Before {
			clone.ChangeInfo.Before[k] = v
		}
		for k, v := range ur.ChangeInfo.After {
			clone.ChangeInfo.After[k] = v
		}
		copy(clone.ChangeInfo.ChangedFields, ur.ChangeInfo.ChangedFields)
	}

	return clone
}

// Update updates the resource with new data
func (ur *UniversalResource) Update(data map[string]interface{}) {
	ur.Data = data
	ur.UpdatedAt = time.Now()
	ur.Version++
}

// SetStatus sets the resource status
func (ur *UniversalResource) SetStatus(status ResourceStatus) {
	ur.Status = status
	ur.UpdatedAt = time.Now()
}

// AddDependency adds a dependency to the resource
func (ur *UniversalResource) AddDependency(dependency string) {
	for _, dep := range ur.Dependencies {
		if dep == dependency {
			return // Already exists
		}
	}
	ur.Dependencies = append(ur.Dependencies, dependency)
}

// RemoveDependency removes a dependency from the resource
func (ur *UniversalResource) RemoveDependency(dependency string) {
	for i, dep := range ur.Dependencies {
		if dep == dependency {
			ur.Dependencies = append(ur.Dependencies[:i], ur.Dependencies[i+1:]...)
			break
		}
	}
}

// GetProviderID returns the provider ID of the state
func (us *UniversalState) GetProviderID() string {
	return us.ProviderID
}
