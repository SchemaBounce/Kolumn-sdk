// Package state provides the StateBackendProvider interface for Kolumn providers
//
// This interface enables providers to act as state storage backends,
// implementing the provider-as-backend architecture.
package state

import (
	"context"
	"time"
)

// StateBackendProvider defines the interface for providers that can act as state storage backends
// External providers implement this interface to provide state storage capabilities
type StateBackendProvider interface {
	// State Operations
	LoadState(ctx context.Context) (*UniversalState, error)
	SaveState(ctx context.Context, state *UniversalState) error
	DeleteState(ctx context.Context) error
	StateExists(ctx context.Context) (bool, error)

	// Backend Properties
	IsPrimary() bool
	IsBackend() bool
	GetProviderID() string
	GetProviderType() string

	// Configuration and Validation
	ValidateConfig(ctx context.Context) error
	ValidateState(ctx context.Context, state *UniversalState) error

	// Health and Monitoring
	GetHealth(ctx context.Context) (*ProviderHealth, error)

	// Backup Operations
	CreateBackup(ctx context.Context, version string) error
	ListBackups(ctx context.Context) ([]string, error)
	RestoreFromBackup(ctx context.Context, version string) error
}

// ProviderHealth represents the health status of a state backend provider
type ProviderHealth struct {
	Status        string                 `json:"status"`        // "healthy", "degraded", "unhealthy"
	LastCheck     time.Time              `json:"last_check"`    // When health was last checked
	ResponseTime  time.Duration          `json:"response_time"` // Response time for health check
	ErrorCount    int                    `json:"error_count"`   // Number of recent errors
	StorageUsed   int64                  `json:"storage_used"`  // Storage space used in bytes
	StateVersion  string                 `json:"state_version"` // Current state version
	CanRead       bool                   `json:"can_read"`      // Can read state operations
	CanWrite      bool                   `json:"can_write"`     // Can write state operations
	CanBackup     bool                   `json:"can_backup"`    // Can create backups
	BackupCount   int                    `json:"backup_count"`  // Number of available backups
	LastBackup    *time.Time             `json:"last_backup"`   // When last backup was created
	Uptime        time.Duration          `json:"uptime"`        // Provider uptime
	Configuration map[string]interface{} `json:"configuration"` // Provider-specific config info
}

// StateBackendCapabilities defines what capabilities a state backend provider supports
type StateBackendCapabilities struct {
	SupportsVersioning   bool  `json:"supports_versioning"`   // Supports state versioning
	SupportsBackups      bool  `json:"supports_backups"`      // Supports backup/restore
	SupportsEncryption   bool  `json:"supports_encryption"`   // Supports encryption at rest
	SupportsCompression  bool  `json:"supports_compression"`  // Supports state compression
	SupportsLocking      bool  `json:"supports_locking"`      // Supports state locking
	SupportsTransactions bool  `json:"supports_transactions"` // Supports atomic transactions
	MaxStateSize         int64 `json:"max_state_size"`        // Maximum supported state size in bytes
	MaxBackups           int   `json:"max_backups"`           // Maximum number of backups supported
	BackupRetention      int   `json:"backup_retention"`      // Backup retention period in days
}

// StateBackendConfig provides configuration for state backend providers
type StateBackendConfig struct {
	EnableBackups     bool   `json:"enable_backups"`
	BackupInterval    string `json:"backup_interval"`
	EncryptionEnabled bool   `json:"encryption_enabled"`
	CompressionLevel  int    `json:"compression_level"`
	MaxRetries        int    `json:"max_retries"`
	TimeoutSeconds    int    `json:"timeout_seconds"`
}

// DefaultStateBackendConfig returns a default configuration for state backends
func DefaultStateBackendConfig() *StateBackendConfig {
	return &StateBackendConfig{
		EnableBackups:     true,
		BackupInterval:    "24h",
		EncryptionEnabled: true,
		CompressionLevel:  6,
		MaxRetries:        3,
		TimeoutSeconds:    30,
	}
}

// StateBackendError represents an error from state backend operations
type StateBackendError struct {
	Operation string
	Provider  string
	Cause     error
	Retryable bool
}

func (e *StateBackendError) Error() string {
	if e.Cause != nil {
		return e.Operation + " failed for provider " + e.Provider + ": " + e.Cause.Error()
	}
	return e.Operation + " failed for provider " + e.Provider
}

func (e *StateBackendError) Unwrap() error {
	return e.Cause
}

// NewStateBackendError creates a new StateBackendError
func NewStateBackendError(operation, provider string, cause error, retryable bool) *StateBackendError {
	return &StateBackendError{
		Operation: operation,
		Provider:  provider,
		Cause:     cause,
		Retryable: retryable,
	}
}

// IsRetryable returns whether the error is retryable
func (e *StateBackendError) IsRetryable() bool {
	return e.Retryable
}
