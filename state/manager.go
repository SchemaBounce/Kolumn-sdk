// Package state - manager.go provides the concrete StateManager implementation for the Kolumn SDK
package state

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/schemabounce/kolumn/sdk/types"
)

// BackendFactory creates state backends
type BackendFactory interface {
	CreateBackend(backendType string) (StateBackend, error)
	CreateAndConfigureBackend(ctx context.Context, backendType string, config map[string]interface{}) (StateBackend, error)
}

// DefaultManager provides a concrete implementation of StateManager
type DefaultManager struct {
	// Core components
	backend        StateBackend
	backendFactory BackendFactory
	adapters       map[string]StateAdapter
	collectionMgr  *ResourceCollectionManager
	dependencyMgr  *DependencyManager
	driftDetector  DriftDetector

	// Configuration
	config *ManagerConfig

	// Internal state
	initialized  bool
	migrationLog []MigrationRecord

	// Thread safety
	mu sync.RWMutex
}

// ManagerConfig contains configuration for the state manager
type ManagerConfig struct {
	WorkspaceName        string                 `json:"workspace_name"`
	Environment          string                 `json:"environment"`
	BackendType          string                 `json:"backend_type"`
	BackendConfig        map[string]interface{} `json:"backend_config"`
	EnableDriftDetection bool                   `json:"enable_drift_detection"`
	DriftCheckInterval   time.Duration          `json:"drift_check_interval"`
	MaxBackups           int                    `json:"max_backups"`
	StateVersion         int64                  `json:"state_version"`
}

// DefaultManagerConfig provides default configuration values
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		WorkspaceName:        "default",
		Environment:          "development",
		BackendType:          "memory",
		BackendConfig:        make(map[string]interface{}),
		EnableDriftDetection: true,
		DriftCheckInterval:   15 * time.Minute,
		MaxBackups:           10,
		StateVersion:         1,
	}
}

// NewManager creates a new state manager instance
func NewManager(config *ManagerConfig) *DefaultManager {
	if config == nil {
		config = DefaultManagerConfig()
	}

	return &DefaultManager{
		backendFactory: nil, // Will be set during initialization
		adapters:       make(map[string]StateAdapter),
		config:         config,
		migrationLog:   make([]MigrationRecord, 0),
		initialized:    false,
	}
}

// NewManagerWithFactory creates a new state manager with a custom backend factory
func NewManagerWithFactory(config *ManagerConfig, factory BackendFactory) *DefaultManager {
	if config == nil {
		config = DefaultManagerConfig()
	}

	return &DefaultManager{
		backendFactory: factory,
		adapters:       make(map[string]StateAdapter),
		config:         config,
		migrationLog:   make([]MigrationRecord, 0),
		initialized:    false,
	}
}

// Initialize initializes the state manager with the provided configuration
func (m *DefaultManager) Initialize(ctx context.Context, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.initialized {
		return fmt.Errorf("state manager already initialized")
	}

	// Update configuration from provided config
	if err := m.updateConfigFromMap(config); err != nil {
		return fmt.Errorf("failed to update configuration: %w", err)
	}

	// Set default factory if not provided
	if m.backendFactory == nil {
		m.backendFactory = &defaultBackendFactory{}
	}

	// Create and configure backend
	backend, err := m.backendFactory.CreateAndConfigureBackend(ctx, m.config.BackendType, m.config.BackendConfig)
	if err != nil {
		return fmt.Errorf("failed to create backend: %w", err)
	}

	m.backend = backend

	// Initialize component managers
	m.collectionMgr = NewResourceCollectionManager(m)
	m.dependencyMgr = NewDependencyManager(m)

	// Initialize drift detector if enabled
	if m.config.EnableDriftDetection {
		m.driftDetector = NewDefaultDriftDetector(m)
	}

	m.initialized = true
	return nil
}

// GetAdapter returns the state adapter for a provider type
func (m *DefaultManager) GetAdapter(providerType string) (StateAdapter, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	adapter, exists := m.adapters[providerType]
	if !exists {
		return nil, fmt.Errorf("no adapter registered for provider type: %s", providerType)
	}

	return adapter, nil
}

// GetBackend returns the configured state backend
func (m *DefaultManager) GetBackend() StateBackend {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.backend
}

// RegisterAdapter registers a state adapter for a provider type
func (m *DefaultManager) RegisterAdapter(providerType string, adapter StateAdapter) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if adapter == nil {
		return fmt.Errorf("adapter cannot be nil")
	}

	m.adapters[providerType] = adapter
	return nil
}

// GetState retrieves state by name
func (m *DefaultManager) GetState(ctx context.Context, name string) (*types.UniversalState, error) {
	if !m.initialized {
		return nil, fmt.Errorf("state manager not initialized")
	}

	return m.backend.GetState(ctx, name)
}

// PutState stores state by name
func (m *DefaultManager) PutState(ctx context.Context, name string, state *types.UniversalState) error {
	if !m.initialized {
		return fmt.Errorf("state manager not initialized")
	}

	// Update timestamps
	now := time.Now()
	state.UpdatedAt = now
	if state.CreatedAt.IsZero() {
		state.CreatedAt = now
	}

	// Increment serial number
	state.Serial++

	// Update metadata
	if state.Metadata.Workspace == "" {
		state.Metadata.Workspace = m.config.WorkspaceName
	}
	if state.Metadata.Environment == "" {
		state.Metadata.Environment = m.config.Environment
	}

	return m.backend.PutState(ctx, name, state)
}

// DeleteState removes state by name
func (m *DefaultManager) DeleteState(ctx context.Context, name string) error {
	if !m.initialized {
		return fmt.Errorf("state manager not initialized")
	}

	return m.backend.DeleteState(ctx, name)
}

// ListStates lists all available states
func (m *DefaultManager) ListStates(ctx context.Context) ([]string, error) {
	if !m.initialized {
		return nil, fmt.Errorf("state manager not initialized")
	}

	return m.backend.ListStates(ctx)
}

// Lock acquires a lock on state
func (m *DefaultManager) Lock(ctx context.Context, name string, info *LockInfo) (string, error) {
	if !m.initialized {
		return "", fmt.Errorf("state manager not initialized")
	}

	// Set defaults
	if info.Path == "" {
		info.Path = name
	}
	if info.Created == "" {
		info.Created = time.Now().Format(time.RFC3339)
	}
	if info.Version == "" {
		info.Version = fmt.Sprintf("%d", m.config.StateVersion)
	}

	return m.backend.Lock(ctx, info)
}

// Unlock releases a lock on state
func (m *DefaultManager) Unlock(ctx context.Context, lockID string, info *LockInfo) error {
	if !m.initialized {
		return fmt.Errorf("state manager not initialized")
	}

	return m.backend.Unlock(ctx, lockID, info)
}

// Import imports state from an external source
func (m *DefaultManager) Import(ctx context.Context, name string, data []byte) error {
	if !m.initialized {
		return fmt.Errorf("state manager not initialized")
	}

	// Deserialize the state data
	// For now, assume it's JSON formatted universal state
	var state types.UniversalState
	if err := serializeUniversalState(data, &state); err != nil {
		return fmt.Errorf("failed to deserialize state data: %w", err)
	}

	// Validate the imported state
	if err := m.validateState(&state); err != nil {
		return fmt.Errorf("imported state validation failed: %w", err)
	}

	// Store the imported state
	return m.PutState(ctx, name, &state)
}

// Export exports state to an external format
func (m *DefaultManager) Export(ctx context.Context, name string) ([]byte, error) {
	if !m.initialized {
		return nil, fmt.Errorf("state manager not initialized")
	}

	// Get the state
	state, err := m.GetState(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	// Serialize the state
	data, err := deserializeUniversalState(state)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize state: %w", err)
	}

	return data, nil
}

// Migrate migrates state between versions
func (m *DefaultManager) Migrate(ctx context.Context, name string, targetVersion string) error {
	if !m.initialized {
		return fmt.Errorf("state manager not initialized")
	}

	// Get current state
	currentState, err := m.GetState(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get current state: %w", err)
	}

	// Record migration start
	migrationRecord := MigrationRecord{
		StateName:   name,
		FromVersion: fmt.Sprintf("%d", currentState.Version),
		ToVersion:   targetVersion,
		StartTime:   time.Now(),
		Status:      "in_progress",
	}

	// Create backup before migration
	backupID, err := m.Backup(ctx, name)
	if err != nil {
		migrationRecord.Status = "failed"
		migrationRecord.Error = err.Error()
		migrationRecord.EndTime = time.Now()
		m.addMigrationRecord(migrationRecord)
		return fmt.Errorf("failed to create backup before migration: %w", err)
	}
	migrationRecord.BackupID = backupID

	// Perform migration (placeholder for actual migration logic)
	migratedState, err := m.performMigration(currentState, targetVersion)
	if err != nil {
		migrationRecord.Status = "failed"
		migrationRecord.Error = err.Error()
		migrationRecord.EndTime = time.Now()
		m.addMigrationRecord(migrationRecord)
		return fmt.Errorf("migration failed: %w", err)
	}

	// Save migrated state
	if err := m.PutState(ctx, name, migratedState); err != nil {
		migrationRecord.Status = "failed"
		migrationRecord.Error = err.Error()
		migrationRecord.EndTime = time.Now()
		m.addMigrationRecord(migrationRecord)
		return fmt.Errorf("failed to save migrated state: %w", err)
	}

	// Record successful migration
	migrationRecord.Status = "completed"
	migrationRecord.EndTime = time.Now()
	m.addMigrationRecord(migrationRecord)

	return nil
}

// Backup creates a backup of state
func (m *DefaultManager) Backup(ctx context.Context, name string) (string, error) {
	if !m.initialized {
		return "", fmt.Errorf("state manager not initialized")
	}

	// Get current state
	state, err := m.GetState(ctx, name)
	if err != nil {
		return "", fmt.Errorf("failed to get state: %w", err)
	}

	// Generate backup ID
	backupID := fmt.Sprintf("%s-backup-%d", name, time.Now().Unix())

	// Store backup
	if err := m.PutState(ctx, backupID, state); err != nil {
		return "", fmt.Errorf("failed to store backup: %w", err)
	}

	return backupID, nil
}

// Restore restores state from a backup
func (m *DefaultManager) Restore(ctx context.Context, name string, backupID string) error {
	if !m.initialized {
		return fmt.Errorf("state manager not initialized")
	}

	// Get backup state
	backupState, err := m.GetState(ctx, backupID)
	if err != nil {
		return fmt.Errorf("failed to get backup state: %w", err)
	}

	// Restore the state
	return m.PutState(ctx, name, backupState)
}

// GetCollectionManager returns the resource collection manager
func (m *DefaultManager) GetCollectionManager() *ResourceCollectionManager {
	return m.collectionMgr
}

// GetDependencyManager returns the dependency manager
func (m *DefaultManager) GetDependencyManager() *DependencyManager {
	return m.dependencyMgr
}

// GetDriftDetector returns the drift detector
func (m *DefaultManager) GetDriftDetector() DriftDetector {
	return m.driftDetector
}

// ValidateState validates a universal state
func (m *DefaultManager) ValidateState(ctx context.Context, state *types.UniversalState) error {
	return m.validateState(state)
}

// GetMigrationHistory returns the migration history
func (m *DefaultManager) GetMigrationHistory() []MigrationRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	history := make([]MigrationRecord, len(m.migrationLog))
	copy(history, m.migrationLog)
	return history
}

// Helper methods

func (m *DefaultManager) updateConfigFromMap(config map[string]interface{}) error {
	if workspace, ok := config["workspace_name"].(string); ok {
		m.config.WorkspaceName = workspace
	}
	if env, ok := config["environment"].(string); ok {
		m.config.Environment = env
	}
	if backendType, ok := config["backend_type"].(string); ok {
		m.config.BackendType = backendType
	}
	if backendConfig, ok := config["backend_config"].(map[string]interface{}); ok {
		m.config.BackendConfig = backendConfig
	}
	if enableDrift, ok := config["enable_drift_detection"].(bool); ok {
		m.config.EnableDriftDetection = enableDrift
	}
	if maxBackups, ok := config["max_backups"].(float64); ok {
		m.config.MaxBackups = int(maxBackups)
	}

	return nil
}

func (m *DefaultManager) validateState(state *types.UniversalState) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}

	if state.Version <= 0 {
		return fmt.Errorf("invalid state version: %d", state.Version)
	}

	if state.Lineage == "" {
		return fmt.Errorf("state lineage cannot be empty")
	}

	// Validate resources
	for i, resource := range state.Resources {
		if err := m.validateResource(&resource); err != nil {
			return fmt.Errorf("invalid resource at index %d: %w", i, err)
		}
	}

	return nil
}

func (m *DefaultManager) validateResource(resource *types.UniversalResource) error {
	if resource.ID == "" {
		return fmt.Errorf("resource ID cannot be empty")
	}
	if resource.Type == "" {
		return fmt.Errorf("resource type cannot be empty")
	}
	if resource.Name == "" {
		return fmt.Errorf("resource name cannot be empty")
	}
	if resource.Provider == "" {
		return fmt.Errorf("resource provider cannot be empty")
	}

	return nil
}

func (m *DefaultManager) performMigration(currentState *types.UniversalState, targetVersion string) (*types.UniversalState, error) {
	// This is a placeholder for actual migration logic
	// In a real implementation, you would have version-specific migration functions

	migratedState := *currentState // Copy the state

	// Update version info based on target
	// This would contain actual migration logic for different versions
	switch targetVersion {
	case "2":
		migratedState.Version = 2
		// Apply v2-specific migrations
	case "3":
		migratedState.Version = 3
		// Apply v3-specific migrations
	default:
		return nil, fmt.Errorf("unsupported target version: %s", targetVersion)
	}

	migratedState.UpdatedAt = time.Now()
	return &migratedState, nil
}

func (m *DefaultManager) addMigrationRecord(record MigrationRecord) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.migrationLog = append(m.migrationLog, record)

	// Keep only the last N records
	if len(m.migrationLog) > m.config.MaxBackups {
		m.migrationLog = m.migrationLog[len(m.migrationLog)-m.config.MaxBackups:]
	}
}

// MigrationRecord tracks state migrations
type MigrationRecord struct {
	StateName   string    `json:"state_name"`
	FromVersion string    `json:"from_version"`
	ToVersion   string    `json:"to_version"`
	BackupID    string    `json:"backup_id,omitempty"`
	Status      string    `json:"status"` // in_progress, completed, failed
	Error       string    `json:"error,omitempty"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
}

// defaultBackendFactory provides a simple backend factory implementation
type defaultBackendFactory struct{}

func (f *defaultBackendFactory) CreateBackend(backendType string) (StateBackend, error) {
	switch backendType {
	case "memory":
		return NewMemoryBackend(), nil
	default:
		return nil, fmt.Errorf("unsupported backend type: %s", backendType)
	}
}

func (f *defaultBackendFactory) CreateAndConfigureBackend(ctx context.Context, backendType string, config map[string]interface{}) (StateBackend, error) {
	backend, err := f.CreateBackend(backendType)
	if err != nil {
		return nil, err
	}

	// Configure the backend if it supports configuration
	if configurableBackend, ok := backend.(ConfigurableBackend); ok {
		if err := configurableBackend.Configure(ctx, config); err != nil {
			return nil, fmt.Errorf("failed to configure %s backend: %w", backendType, err)
		}
	}

	return backend, nil
}

// ConfigurableBackend represents a backend that can be configured
type ConfigurableBackend interface {
	Configure(ctx context.Context, config map[string]interface{}) error
}

// NewMemoryBackend creates a simple in-memory backend for testing
func NewMemoryBackend() StateBackend {
	return &memoryBackend{
		states: make(map[string]*types.UniversalState),
		locks:  make(map[string]string),
	}
}

// memoryBackend provides a simple in-memory implementation for testing
type memoryBackend struct {
	states map[string]*types.UniversalState
	locks  map[string]string
	mu     sync.RWMutex
}

func (m *memoryBackend) GetState(ctx context.Context, name string) (*types.UniversalState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.states[name]
	if !exists {
		return nil, fmt.Errorf("state not found: %s", name)
	}

	// Return a copy to prevent external modification
	stateCopy := *state
	return &stateCopy, nil
}

func (m *memoryBackend) PutState(ctx context.Context, name string, state *types.UniversalState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store a copy to prevent external modification
	stateCopy := *state
	m.states[name] = &stateCopy
	return nil
}

func (m *memoryBackend) DeleteState(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.states[name]; !exists {
		return fmt.Errorf("state not found: %s", name)
	}

	delete(m.states, name)
	return nil
}

func (m *memoryBackend) ListStates(ctx context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.states))
	for name := range m.states {
		names = append(names, name)
	}
	return names, nil
}

func (m *memoryBackend) Lock(ctx context.Context, info *LockInfo) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existingLockID, exists := m.locks[info.Path]; exists {
		return "", fmt.Errorf("state is already locked with ID: %s", existingLockID)
	}

	lockID := info.ID
	if lockID == "" {
		lockID = fmt.Sprintf("lock-%d", time.Now().UnixNano())
	}

	m.locks[info.Path] = lockID
	return lockID, nil
}

func (m *memoryBackend) Unlock(ctx context.Context, lockID string, info *LockInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existingLockID, exists := m.locks[info.Path]
	if !exists {
		return fmt.Errorf("no lock found for path: %s", info.Path)
	}

	if existingLockID != lockID {
		return fmt.Errorf("lock ID mismatch: expected %s, got %s", existingLockID, lockID)
	}

	delete(m.locks, info.Path)
	return nil
}
