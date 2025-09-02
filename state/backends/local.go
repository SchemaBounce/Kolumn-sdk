package backends

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/schemabounce/kolumn/sdk/state"
	"github.com/schemabounce/kolumn/sdk/types"
)

// LocalBackend implements state storage on local filesystem
type LocalBackend struct {
	config     *LocalConfig
	lockMutex  sync.Mutex
	locks      map[string]*state.LockInfo
	configured bool
}

// LocalConfig contains local filesystem backend configuration
type LocalConfig struct {
	Path         string `json:"path"`
	WorkspaceDir string `json:"workspace_dir"`
	BackupDir    string `json:"backup_dir"`
	BackupCount  int    `json:"backup_count"`
	Permissions  int    `json:"permissions"`
}

// NewLocalBackend creates a new local filesystem backend
func NewLocalBackend() *LocalBackend {
	return &LocalBackend{
		locks: make(map[string]*state.LockInfo),
	}
}

// Configure sets up the local filesystem backend
func (b *LocalBackend) Configure(ctx context.Context, config map[string]interface{}) error {
	// Parse configuration
	localConfig, err := parseLocalConfig(config)
	if err != nil {
		return fmt.Errorf("invalid local configuration: %w", err)
	}

	b.config = localConfig

	// Create directories if they don't exist
	if err := b.createDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	b.configured = true
	return nil
}

// GetState retrieves state by name
func (b *LocalBackend) GetState(ctx context.Context, name string) (*types.UniversalState, error) {
	if !b.configured {
		return nil, fmt.Errorf("backend not configured")
	}

	stateFile := b.getStateFilePath(name)

	// Check if state file exists
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("state '%s' not found", name)
	}

	// Read state file
	stateData, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	// Parse state JSON
	var st types.UniversalState
	if err := json.Unmarshal(stateData, &st); err != nil {
		return nil, fmt.Errorf("failed to parse state JSON: %w", err)
	}

	// Update timestamps from file info
	if fileInfo, err := os.Stat(stateFile); err == nil {
		st.UpdatedAt = fileInfo.ModTime()
	}

	return &st, nil
}

// PutState stores state by name
func (b *LocalBackend) PutState(ctx context.Context, name string, st *types.UniversalState) error {
	if !b.configured {
		return fmt.Errorf("backend not configured")
	}

	if st == nil {
		return fmt.Errorf("state cannot be nil")
	}

	stateFile := b.getStateFilePath(name)

	// Create backup of existing state
	if err := b.createBackup(stateFile); err != nil {
		// Log but don't fail on backup errors
	}

	// Update timestamp
	st.UpdatedAt = time.Now()

	// Serialize state to JSON
	stateData, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}

	// Create temporary file first
	tempFile := stateFile + ".tmp"
	if err := os.WriteFile(tempFile, stateData, fs.FileMode(b.config.Permissions)); err != nil {
		return fmt.Errorf("failed to write temporary state file: %w", err)
	}

	// Atomically replace the state file
	if err := os.Rename(tempFile, stateFile); err != nil {
		// Clean up temporary file
		os.Remove(tempFile)
		return fmt.Errorf("failed to replace state file: %w", err)
	}

	return nil
}

// DeleteState removes state by name
func (b *LocalBackend) DeleteState(ctx context.Context, name string) error {
	if !b.configured {
		return fmt.Errorf("backend not configured")
	}

	stateFile := b.getStateFilePath(name)

	// Create final backup before deletion
	if err := b.createBackup(stateFile); err != nil {
		// Log but don't fail
	}

	// Remove state file
	if err := os.Remove(stateFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete state file: %w", err)
	}

	// Remove lock file if it exists
	lockFile := b.getLockFilePath(name)
	if err := os.Remove(lockFile); err != nil && !os.IsNotExist(err) {
		// Log but don't fail
	}

	return nil
}

// ListStates lists all available states
func (b *LocalBackend) ListStates(ctx context.Context) ([]string, error) {
	if !b.configured {
		return nil, fmt.Errorf("backend not configured")
	}

	stateDir := b.getStateDirectory()

	entries, err := os.ReadDir(stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read state directory: %w", err)
	}

	var states []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Look for .klstate files
		if strings.HasSuffix(name, ".klstate") {
			// Remove extension to get state name
			stateName := strings.TrimSuffix(name, ".klstate")
			states = append(states, stateName)
		}
	}

	return states, nil
}

// Lock acquires a lock on the state
func (b *LocalBackend) Lock(ctx context.Context, info *state.LockInfo) (string, error) {
	if !b.configured {
		return "", fmt.Errorf("backend not configured")
	}

	if info == nil {
		return "", fmt.Errorf("lock info cannot be nil")
	}

	b.lockMutex.Lock()
	defer b.lockMutex.Unlock()

	lockKey := b.getLockKey(info.Path)

	// Check if already locked
	if existingLock, exists := b.locks[lockKey]; exists {
		return "", fmt.Errorf("state is already locked by %s (ID: %s)", existingLock.Who, existingLock.ID)
	}

	// Create lock file
	lockFile := b.getLockFilePath(info.Path)
	lockData, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to serialize lock info: %w", err)
	}

	if err := os.WriteFile(lockFile, lockData, fs.FileMode(b.config.Permissions)); err != nil {
		return "", fmt.Errorf("failed to create lock file: %w", err)
	}

	// Store lock in memory
	b.locks[lockKey] = info

	return info.ID, nil
}

// Unlock releases a lock on the state
func (b *LocalBackend) Unlock(ctx context.Context, lockID string, info *state.LockInfo) error {
	if !b.configured {
		return fmt.Errorf("backend not configured")
	}

	if info == nil {
		return fmt.Errorf("lock info cannot be nil")
	}

	b.lockMutex.Lock()
	defer b.lockMutex.Unlock()

	lockKey := b.getLockKey(info.Path)

	// Check if lock exists and matches
	if existingLock, exists := b.locks[lockKey]; !exists || existingLock.ID != lockID {
		// Already unlocked or different lock, which is fine
		return nil
	}

	// Remove lock file
	lockFile := b.getLockFilePath(info.Path)
	if err := os.Remove(lockFile); err != nil && !os.IsNotExist(err) {
		// Log but don't fail
	}

	// Remove from memory
	delete(b.locks, lockKey)

	return nil
}

// Helper methods

func (b *LocalBackend) createDirectories() error {
	// Create state directory
	stateDir := b.getStateDirectory()
	if err := os.MkdirAll(stateDir, fs.FileMode(b.config.Permissions)); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Create workspace directory if specified
	if b.config.WorkspaceDir != "" {
		if err := os.MkdirAll(b.config.WorkspaceDir, fs.FileMode(b.config.Permissions)); err != nil {
			return fmt.Errorf("failed to create workspace directory: %w", err)
		}
	}

	// Create backup directory if specified
	if b.config.BackupDir != "" {
		if err := os.MkdirAll(b.config.BackupDir, fs.FileMode(b.config.Permissions)); err != nil {
			return fmt.Errorf("failed to create backup directory: %w", err)
		}
	}

	return nil
}

func (b *LocalBackend) getStateDirectory() string {
	if b.config.WorkspaceDir != "" {
		return b.config.WorkspaceDir
	}
	return filepath.Dir(b.config.Path)
}

func (b *LocalBackend) getStateFilePath(name string) string {
	if b.config.WorkspaceDir != "" {
		return filepath.Join(b.config.WorkspaceDir, fmt.Sprintf("%s.klstate", name))
	}

	// For single-file configuration, use the configured path directly
	if name == "default" || name == "" {
		return b.config.Path
	}

	// For named states, use directory of configured path
	dir := filepath.Dir(b.config.Path)
	return filepath.Join(dir, fmt.Sprintf("%s.klstate", name))
}

func (b *LocalBackend) getLockFilePath(name string) string {
	stateFile := b.getStateFilePath(name)
	return stateFile + ".lock"
}

func (b *LocalBackend) getLockKey(path string) string {
	if path == "" {
		return "default"
	}
	return path
}

func (b *LocalBackend) createBackup(stateFile string) error {
	// Skip backup if file doesn't exist
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		return nil
	}

	// Skip backup if no backup directory configured
	if b.config.BackupDir == "" {
		return nil
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := filepath.Base(stateFile)
	backupFile := filepath.Join(b.config.BackupDir, fmt.Sprintf("%s.%s", filename, timestamp))

	// Copy state file to backup
	stateData, err := os.ReadFile(stateFile)
	if err != nil {
		return fmt.Errorf("failed to read state file for backup: %w", err)
	}

	if err := os.WriteFile(backupFile, stateData, fs.FileMode(b.config.Permissions)); err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}

	// Clean up old backups if limit is set
	if b.config.BackupCount > 0 {
		if err := b.cleanupOldBackups(); err != nil {
			// Log but don't fail
		}
	}

	return nil
}

func (b *LocalBackend) cleanupOldBackups() error {
	if b.config.BackupDir == "" {
		return nil
	}

	// Read backup directory
	entries, err := os.ReadDir(b.config.BackupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	// Filter backup files
	var backupFiles []fs.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && strings.Contains(entry.Name(), ".klstate.") {
			backupFiles = append(backupFiles, entry)
		}
	}

	// Remove excess backups (keep newest ones)
	if len(backupFiles) > b.config.BackupCount {
		// For simplicity, remove the excess oldest by name
		excess := len(backupFiles) - b.config.BackupCount
		for i := 0; i < excess; i++ {
			backupPath := filepath.Join(b.config.BackupDir, backupFiles[i].Name())
			if err := os.Remove(backupPath); err != nil {
				// Log but don't fail
			}
		}
	}

	return nil
}

func parseLocalConfig(config map[string]interface{}) (*LocalConfig, error) {
	cfg := &LocalConfig{
		Path:        "kolumn.klstate",
		BackupCount: 10,
		Permissions: 0644,
	}

	// Parse configuration map
	if path, ok := config["path"].(string); ok {
		cfg.Path = path
	}

	if workspaceDir, ok := config["workspace_dir"].(string); ok {
		cfg.WorkspaceDir = workspaceDir
	}

	if backupDir, ok := config["backup_dir"].(string); ok {
		cfg.BackupDir = backupDir
	}

	if backupCount, ok := config["backup_count"].(float64); ok {
		cfg.BackupCount = int(backupCount)
	} else if backupCount, ok := config["backup_count"].(int); ok {
		cfg.BackupCount = backupCount
	}

	if permissions, ok := config["permissions"].(float64); ok {
		cfg.Permissions = int(permissions)
	} else if permissions, ok := config["permissions"].(int); ok {
		cfg.Permissions = permissions
	}

	// Validate configuration
	if cfg.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	// Convert relative paths to absolute
	if !filepath.IsAbs(cfg.Path) {
		abs, err := filepath.Abs(cfg.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path: %w", err)
		}
		cfg.Path = abs
	}

	return cfg, nil
}
