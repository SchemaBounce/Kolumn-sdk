package backends

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/schemabounce/kolumn/sdk/state"
	"github.com/schemabounce/kolumn/sdk/types"
)

// PostgresBackend implements state storage in PostgreSQL
type PostgresBackend struct {
	db         *sql.DB
	config     *PostgresConfig
	tableName  string
	lockTable  string
	configured bool
}

// PostgresConfig contains PostgreSQL backend configuration
type PostgresConfig struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Database     string `json:"database"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	SSLMode      string `json:"ssl_mode"`
	Schema       string `json:"schema"`
	TableName    string `json:"table_name"`
	LockTable    string `json:"lock_table"`
	ConnTimeout  int    `json:"conn_timeout"`
	MaxConns     int    `json:"max_conns"`
	MaxIdleConns int    `json:"max_idle_conns"`
}

// NewPostgresBackend creates a new PostgreSQL backend
func NewPostgresBackend() *PostgresBackend {
	return &PostgresBackend{
		tableName: "kolumn_state",
		lockTable: "kolumn_locks",
	}
}

// Configure sets up the PostgreSQL backend
func (b *PostgresBackend) Configure(ctx context.Context, config map[string]interface{}) error {
	// Parse configuration
	pgConfig, err := parsePostgresConfig(config)
	if err != nil {
		return fmt.Errorf("invalid PostgreSQL configuration: %w", err)
	}

	b.config = pgConfig

	// Override table names if configured
	if pgConfig.TableName != "" {
		b.tableName = pgConfig.TableName
	}
	if pgConfig.LockTable != "" {
		b.lockTable = pgConfig.LockTable
	}

	// Build connection string
	connStr := b.buildConnectionString()

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	if pgConfig.MaxConns > 0 {
		db.SetMaxOpenConns(pgConfig.MaxConns)
	}
	if pgConfig.MaxIdleConns > 0 {
		db.SetMaxIdleConns(pgConfig.MaxIdleConns)
	}
	if pgConfig.ConnTimeout > 0 {
		db.SetConnMaxLifetime(time.Duration(pgConfig.ConnTimeout) * time.Second)
	}

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	b.db = db

	// Create necessary tables
	if err := b.createTables(ctx); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	b.configured = true
	return nil
}

// GetState retrieves state by name
func (b *PostgresBackend) GetState(ctx context.Context, name string) (*types.UniversalState, error) {
	if !b.configured {
		return nil, fmt.Errorf("backend not configured")
	}

	query := fmt.Sprintf(`
		SELECT state_data, created_at, updated_at 
		FROM %s.%s 
		WHERE workspace = $1 
		ORDER BY serial DESC 
		LIMIT 1`,
		b.config.Schema, b.tableName)

	var stateData []byte
	var createdAt, updatedAt time.Time

	err := b.db.QueryRowContext(ctx, query, name).Scan(&stateData, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("state '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// Parse state JSON
	var st types.UniversalState
	if err := json.Unmarshal(stateData, &st); err != nil {
		return nil, fmt.Errorf("failed to parse state JSON: %w", err)
	}

	// Update timestamps from database
	st.CreatedAt = createdAt
	st.UpdatedAt = updatedAt

	return &st, nil
}

// PutState stores state by name
func (b *PostgresBackend) PutState(ctx context.Context, name string, st *types.UniversalState) error {
	if !b.configured {
		return fmt.Errorf("backend not configured")
	}

	if st == nil {
		return fmt.Errorf("state cannot be nil")
	}

	// Update timestamp
	st.UpdatedAt = time.Now()
	if st.CreatedAt.IsZero() {
		st.CreatedAt = st.UpdatedAt
	}

	// Serialize state to JSON
	stateData, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}

	// Insert or update state
	query := fmt.Sprintf(`
		INSERT INTO %s.%s (workspace, serial, lineage, state_data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (workspace) 
		DO UPDATE SET 
			serial = EXCLUDED.serial,
			lineage = EXCLUDED.lineage,
			state_data = EXCLUDED.state_data,
			updated_at = EXCLUDED.updated_at`,
		b.config.Schema, b.tableName)

	_, err = b.db.ExecContext(ctx, query,
		name,
		st.Serial,
		st.Lineage,
		stateData,
		st.CreatedAt,
		st.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// DeleteState removes state by name
func (b *PostgresBackend) DeleteState(ctx context.Context, name string) error {
	if !b.configured {
		return fmt.Errorf("backend not configured")
	}

	query := fmt.Sprintf(`DELETE FROM %s.%s WHERE workspace = $1`, b.config.Schema, b.tableName)

	_, err := b.db.ExecContext(ctx, query, name)
	if err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}

	return nil
}

// ListStates lists all available states
func (b *PostgresBackend) ListStates(ctx context.Context) ([]string, error) {
	if !b.configured {
		return nil, fmt.Errorf("backend not configured")
	}

	query := fmt.Sprintf(`SELECT workspace FROM %s.%s ORDER BY workspace`, b.config.Schema, b.tableName)

	rows, err := b.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list states: %w", err)
	}
	defer rows.Close()

	var states []string
	for rows.Next() {
		var workspace string
		if err := rows.Scan(&workspace); err != nil {
			return nil, fmt.Errorf("failed to scan workspace: %w", err)
		}
		states = append(states, workspace)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over results: %w", err)
	}

	return states, nil
}

// Lock acquires a lock on the state
func (b *PostgresBackend) Lock(ctx context.Context, info *state.LockInfo) (string, error) {
	if !b.configured {
		return "", fmt.Errorf("backend not configured")
	}

	if info == nil {
		return "", fmt.Errorf("lock info cannot be nil")
	}

	// Try to acquire lock
	query := fmt.Sprintf(`
		INSERT INTO %s.%s (lock_id, workspace, operation, info, who, version, created_at, path)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		b.config.Schema, b.lockTable)

	_, err := b.db.ExecContext(ctx, query,
		info.ID,
		info.Path,
		info.Operation,
		info.Reason,
		info.Who,
		info.Version,
		info.Created,
		info.Path,
	)

	if err != nil {
		// Check for unique constraint violation (state already locked)
		if isUniqueViolation(err) {
			return "", fmt.Errorf("state is already locked")
		}
		return "", fmt.Errorf("failed to acquire lock: %w", err)
	}

	return info.ID, nil
}

// Unlock releases a lock on the state
func (b *PostgresBackend) Unlock(ctx context.Context, lockID string, info *state.LockInfo) error {
	if !b.configured {
		return fmt.Errorf("backend not configured")
	}

	if info == nil {
		return fmt.Errorf("lock info cannot be nil")
	}

	query := fmt.Sprintf(`
		DELETE FROM %s.%s 
		WHERE lock_id = $1 AND workspace = $2`,
		b.config.Schema, b.lockTable)

	result, err := b.db.ExecContext(ctx, query, lockID, info.Path)
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		// Lock was not found, but that's fine
		return nil
	}

	return nil
}

// Helper methods

func (b *PostgresBackend) buildConnectionString() string {
	config := b.config

	// Set defaults
	if config.Host == "" {
		config.Host = "localhost"
	}
	if config.Port == 0 {
		config.Port = 5432
	}
	if config.SSLMode == "" {
		config.SSLMode = "prefer"
	}

	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.Username, config.Password, config.Database, config.SSLMode)
}

func (b *PostgresBackend) createTables(ctx context.Context) error {
	// Create schema if it doesn't exist
	if b.config.Schema != "" && b.config.Schema != "public" {
		createSchema := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", b.config.Schema)
		if _, err := b.db.ExecContext(ctx, createSchema); err != nil {
			return fmt.Errorf("failed to create schema: %w", err)
		}
	}

	// Create state table
	createStateTable := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.%s (
			workspace VARCHAR(255) PRIMARY KEY,
			serial BIGINT NOT NULL,
			lineage VARCHAR(255) NOT NULL,
			state_data JSONB NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL
		)`, b.config.Schema, b.tableName)

	if _, err := b.db.ExecContext(ctx, createStateTable); err != nil {
		return fmt.Errorf("failed to create state table: %w", err)
	}

	// Create lock table
	createLockTable := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.%s (
			lock_id VARCHAR(255) NOT NULL,
			workspace VARCHAR(255) NOT NULL,
			operation VARCHAR(100) NOT NULL,
			info TEXT,
			who VARCHAR(255) NOT NULL,
			version VARCHAR(100),
			created_at TIMESTAMP WITH TIME ZONE NOT NULL,
			path VARCHAR(1000),
			PRIMARY KEY (workspace)
		)`, b.config.Schema, b.lockTable)

	if _, err := b.db.ExecContext(ctx, createLockTable); err != nil {
		return fmt.Errorf("failed to create lock table: %w", err)
	}

	// Create indexes
	createIndexes := []string{
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_serial ON %s.%s (serial)",
			b.tableName, b.config.Schema, b.tableName),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_updated_at ON %s.%s (updated_at)",
			b.tableName, b.config.Schema, b.tableName),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_lock_id ON %s.%s (lock_id)",
			b.lockTable, b.config.Schema, b.lockTable),
	}

	for _, indexSQL := range createIndexes {
		if _, err := b.db.ExecContext(ctx, indexSQL); err != nil {
			// Don't fail on index creation errors
		}
	}

	return nil
}

// isUniqueViolation checks if the error is a unique constraint violation
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	// Simple check for PostgreSQL unique violation
	errStr := err.Error()
	return containsAny(errStr, []string{
		"duplicate key value",
		"violates unique constraint",
		"UNIQUE constraint",
	})
}

// containsAny checks if a string contains any of the given substrings
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

func parsePostgresConfig(config map[string]interface{}) (*PostgresConfig, error) {
	cfg := &PostgresConfig{
		Schema: "public",
	}

	// Parse configuration map
	if host, ok := config["host"].(string); ok {
		cfg.Host = host
	}

	if port, ok := config["port"].(float64); ok {
		cfg.Port = int(port)
	} else if port, ok := config["port"].(int); ok {
		cfg.Port = port
	}

	if database, ok := config["database"].(string); ok {
		cfg.Database = database
	}

	if username, ok := config["username"].(string); ok {
		cfg.Username = username
	}

	if password, ok := config["password"].(string); ok {
		cfg.Password = password
	}

	if sslMode, ok := config["ssl_mode"].(string); ok {
		cfg.SSLMode = sslMode
	}

	if schema, ok := config["schema"].(string); ok {
		cfg.Schema = schema
	}

	if tableName, ok := config["table_name"].(string); ok {
		cfg.TableName = tableName
	}

	if lockTable, ok := config["lock_table"].(string); ok {
		cfg.LockTable = lockTable
	}

	if connTimeout, ok := config["conn_timeout"].(float64); ok {
		cfg.ConnTimeout = int(connTimeout)
	} else if connTimeout, ok := config["conn_timeout"].(int); ok {
		cfg.ConnTimeout = connTimeout
	}

	if maxConns, ok := config["max_conns"].(float64); ok {
		cfg.MaxConns = int(maxConns)
	} else if maxConns, ok := config["max_conns"].(int); ok {
		cfg.MaxConns = maxConns
	}

	if maxIdleConns, ok := config["max_idle_conns"].(float64); ok {
		cfg.MaxIdleConns = int(maxIdleConns)
	} else if maxIdleConns, ok := config["max_idle_conns"].(int); ok {
		cfg.MaxIdleConns = maxIdleConns
	}

	// Validate required fields
	if cfg.Database == "" {
		return nil, fmt.Errorf("database name is required")
	}

	if cfg.Username == "" {
		return nil, fmt.Errorf("username is required")
	}

	return cfg, nil
}
