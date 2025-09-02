// Package state provides state management interfaces for the Kolumn SDK
package state

import (
	"context"

	"github.com/schemabounce/kolumn/sdk/types"
)

// StateAdapter defines the interface for provider state management
// Providers can implement this to handle state operations
type StateAdapter interface {
	// ToUniversalState converts provider-specific state to universal format
	ToUniversalState(providerState interface{}) (*types.UniversalState, error)

	// FromUniversalState converts universal state to provider-specific format
	FromUniversalState(universalState *types.UniversalState) (interface{}, error)

	// ExtractDependencies extracts dependencies from provider-specific state
	ExtractDependencies(providerState interface{}) ([]types.Dependency, error)

	// ValidateState validates the consistency of the state
	ValidateState(state *types.UniversalState) error

	// SerializeState serializes state for storage
	SerializeState(state *types.UniversalState) ([]byte, error)

	// DeserializeState deserializes state from storage
	DeserializeState(data []byte) (*types.UniversalState, error)
}

// StateBackend defines the interface for state storage backends
type StateBackend interface {
	// GetState retrieves state by name
	GetState(ctx context.Context, name string) (*types.UniversalState, error)

	// PutState stores state by name
	PutState(ctx context.Context, name string, state *types.UniversalState) error

	// DeleteState removes state by name
	DeleteState(ctx context.Context, name string) error

	// ListStates lists all available states
	ListStates(ctx context.Context) ([]string, error)

	// Lock acquires a lock on the state
	Lock(ctx context.Context, info *LockInfo) (string, error)

	// Unlock releases a lock on the state
	Unlock(ctx context.Context, lockID string, info *LockInfo) error
}

// LockInfo contains information about a state lock
type LockInfo struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	Who       string `json:"who"`
	Version   string `json:"version"`
	Created   string `json:"created"`
	Reason    string `json:"reason"`
	Operation string `json:"operation"`
}

// StateManager provides high-level state management operations
type StateManager interface {
	// Initialize initializes the state manager
	Initialize(ctx context.Context, config map[string]interface{}) error

	// GetAdapter returns the state adapter for a provider
	GetAdapter(providerType string) (StateAdapter, error)

	// GetBackend returns the state backend
	GetBackend() StateBackend

	// Import imports state from an external source
	Import(ctx context.Context, name string, data []byte) error

	// Export exports state to an external format
	Export(ctx context.Context, name string) ([]byte, error)

	// Migrate migrates state between versions
	Migrate(ctx context.Context, name string, targetVersion string) error

	// Backup creates a backup of state
	Backup(ctx context.Context, name string) (string, error)

	// Restore restores state from a backup
	Restore(ctx context.Context, name string, backupID string) error

	// GetState returns the current state
	GetState(ctx context.Context) (*types.UniversalState, error)

	// SaveState saves the state
	SaveState(ctx context.Context, state *types.UniversalState) error
}

// DriftDetector detects drift between desired and actual state
type DriftDetector interface {
	// DetectDrift detects drift in state
	DetectDrift(ctx context.Context, state *types.UniversalState) (*DriftAnalysis, error)

	// ResolveDrift attempts to resolve detected drift
	ResolveDrift(ctx context.Context, analysis *DriftAnalysis) error
}

// DriftAnalysis contains the results of drift detection
type DriftAnalysis struct {
	HasDrift   bool               `json:"has_drift"`
	DriftItems []DriftItem        `json:"drift_items"`
	Summary    DriftSummary       `json:"summary"`
	Resolution ResolutionStrategy `json:"resolution"`
	Timestamp  string             `json:"timestamp"`
}

// DriftItem represents a single drift item
type DriftItem struct {
	ResourceID     string        `json:"resource_id"`
	ResourceType   string        `json:"resource_type"`
	DriftType      DriftType     `json:"drift_type"`
	Field          string        `json:"field"`
	StateValue     interface{}   `json:"state_value"`
	ActualValue    interface{}   `json:"actual_value"`
	Severity       DriftSeverity `json:"severity"`
	Confidence     float64       `json:"confidence"`
	AutoResolvable bool          `json:"auto_resolvable"`
}

// DriftType represents the type of drift
type DriftType string

const (
	DriftTypeCreate DriftType = "create"
	DriftTypeUpdate DriftType = "update"
	DriftTypeDelete DriftType = "delete"
	DriftTypeModify DriftType = "modify"
)

// DriftSeverity represents the severity of drift
type DriftSeverity string

const (
	DriftSeverityLow      DriftSeverity = "low"
	DriftSeverityMedium   DriftSeverity = "medium"
	DriftSeverityHigh     DriftSeverity = "high"
	DriftSeverityCritical DriftSeverity = "critical"
)

// DriftSummary provides a summary of drift analysis
type DriftSummary struct {
	TotalItems   int                   `json:"total_items"`
	BySeverity   map[DriftSeverity]int `json:"by_severity"`
	ByType       map[DriftType]int     `json:"by_type"`
	Resolvable   int                   `json:"resolvable"`
	ManualReview int                   `json:"manual_review"`
}

// ResolutionStrategy defines how to resolve drift
type ResolutionStrategy struct {
	Strategy     ResolveStrategy    `json:"strategy"`
	AutoResolve  []string           `json:"auto_resolve"`
	ManualReview []string           `json:"manual_review"`
	Actions      []ResolutionAction `json:"actions"`
}

// ResolveStrategy represents a resolution strategy
type ResolveStrategy string

const (
	StrategyUpdateState    ResolveStrategy = "update_state"
	StrategyUpdateResource ResolveStrategy = "update_resource"
	StrategyPromptUser     ResolveStrategy = "prompt_user"
	StrategyIgnore         ResolveStrategy = "ignore"
)

// ResolutionAction represents a specific resolution action
type ResolutionAction struct {
	Type       string                 `json:"type"`
	ResourceID string                 `json:"resource_id"`
	Field      string                 `json:"field,omitempty"`
	NewValue   interface{}            `json:"new_value,omitempty"`
	Reason     string                 `json:"reason"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}
