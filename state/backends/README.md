# Kolumn SDK State Backends

This package provides concrete implementations of state storage backends for the Kolumn SDK. External providers can use these backends to persist their state across operations.

## Available Backends

### Memory Backend
- **Type**: `memory`
- **Use Case**: Testing and development
- **Configuration**: None required
- **Locking**: Full support
- **Features**: In-memory storage, automatic cleanup on process exit

```go
backend := backends.NewMemoryBackend()
```

### Local Filesystem Backend
- **Type**: `local`
- **Use Case**: Single-machine development, simple deployments
- **Configuration**: File path, backup options
- **Locking**: File-based locking
- **Features**: Atomic writes, backup support, workspace isolation

```go
config := map[string]interface{}{
    "path":         "/path/to/state.klstate",
    "backup_dir":   "/path/to/backups",
    "backup_count": 5,
    "permissions":  0644,
}
```

### PostgreSQL Backend
- **Type**: `postgres`
- **Use Case**: Production deployments, team collaboration
- **Configuration**: Database connection details
- **Locking**: Database-level locking with unique constraints
- **Features**: ACID transactions, concurrent access, versioning

```go
config := map[string]interface{}{
    "host":     "localhost",
    "port":     5432,
    "database": "kolumn_state",
    "username": "kolumn_user",
    "password": "secure_password",
    "schema":   "public",
    "ssl_mode": "prefer",
}
```

### Amazon S3 Backend
- **Type**: `s3`
- **Use Case**: Cloud deployments, high availability
- **Configuration**: S3 bucket and AWS credentials
- **Locking**: Basic implementation (DynamoDB recommended for production)
- **Features**: Encryption, versioning, cross-region replication

```go
config := map[string]interface{}{
    "bucket":     "my-terraform-state",
    "key_prefix": "projects/my-project",
    "region":     "us-west-2",
    "encrypt":    true,
    "kms_key_id": "arn:aws:kms:...",
}
```

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/schemabounce/kolumn/sdk/state/backends"
    "github.com/schemabounce/kolumn/sdk/types"
)

func main() {
    // Create backend using factory
    factory := backends.NewBackendFactory()
    backend, err := factory.CreateBackend(backends.BackendTypeMemory)
    if err != nil {
        log.Fatal(err)
    }
    
    ctx := context.Background()
    
    // Create state
    state := &types.UniversalState{
        Serial:  1,
        Lineage: "my-project-123",
    }
    
    // Store state
    err = backend.PutState(ctx, "my-project", state)
    if err != nil {
        log.Fatal(err)
    }
    
    // Retrieve state
    retrievedState, err := backend.GetState(ctx, "my-project")
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Retrieved state with lineage: %s", retrievedState.Lineage)
}
```

### Configuration and Validation

```go
// Get default configuration for a backend type
defaultConfig := backends.GetDefaultConfig(backends.BackendTypeLocal)

// Validate configuration
err := backends.ValidateConfig(backends.BackendTypeLocal, config)
if err != nil {
    log.Printf("Invalid configuration: %v", err)
}

// Create and configure backend
backend, err := backends.CreateAndConfigureBackend(
    ctx, 
    backends.BackendTypeLocal, 
    config,
)
```

### State Locking

```go
// Create lock info
lockInfo := &state.LockInfo{
    ID:        "unique-lock-id",
    Path:      "my-project",
    Who:       "user@example.com",
    Version:   "1.0.0",
    Created:   time.Now().Format(time.RFC3339),
    Reason:    "Terraform apply operation",
    Operation: "apply",
}

// Acquire lock
lockID, err := backend.Lock(ctx, lockInfo)
if err != nil {
    log.Printf("Failed to acquire lock: %v", err)
    return
}

// Perform operations...

// Release lock
err = backend.Unlock(ctx, lockID, lockInfo)
if err != nil {
    log.Printf("Failed to release lock: %v", err)
}
```

## Backend Interface

All backends implement the `StateBackend` interface:

```go
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
```

## Configuration Reference

### Memory Backend
No configuration required.

### Local Backend
| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `path` | string | Yes | `kolumn.klstate` | Path to state file |
| `workspace_dir` | string | No | - | Directory for workspace-specific states |
| `backup_dir` | string | No | - | Directory for backups |
| `backup_count` | int | No | 10 | Number of backups to keep |
| `permissions` | int | No | 0644 | File permissions (octal) |

### PostgreSQL Backend
| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `host` | string | No | `localhost` | Database host |
| `port` | int | No | 5432 | Database port |
| `database` | string | Yes | - | Database name |
| `username` | string | Yes | - | Database username |
| `password` | string | No | - | Database password |
| `ssl_mode` | string | No | `prefer` | SSL mode |
| `schema` | string | No | `public` | Database schema |
| `table_name` | string | No | `kolumn_state` | State table name |
| `lock_table` | string | No | `kolumn_locks` | Lock table name |

### S3 Backend
| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `bucket` | string | Yes | - | S3 bucket name |
| `key_prefix` | string | No | - | Key prefix for states |
| `region` | string | No | `us-east-1` | AWS region |
| `encrypt` | bool | No | true | Enable encryption |
| `kms_key_id` | string | No | - | KMS key ID for encryption |
| `storage_class` | string | No | `STANDARD` | S3 storage class |
| `access_key` | string | No | - | AWS access key |
| `secret_key` | string | No | - | AWS secret key |

## Error Handling

All backends return structured errors for common scenarios:

- **State Not Found**: When attempting to retrieve a non-existent state
- **State Locked**: When attempting to acquire a lock that's already held
- **Configuration Error**: When backend configuration is invalid
- **Connection Error**: When unable to connect to external storage

## Testing

The package includes comprehensive tests for all backends:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific backend tests
go test -run TestMemoryBackend ./...
```

## Thread Safety

- **Memory Backend**: Thread-safe with internal locking
- **Local Backend**: Thread-safe for single process, file locking for multiple processes
- **PostgreSQL Backend**: Thread-safe with database-level locking
- **S3 Backend**: Eventually consistent, basic locking support

## Production Considerations

### Local Backend
- Use SSD storage for better performance
- Configure backup directory on separate disk
- Monitor disk space usage
- Consider file system limitations

### PostgreSQL Backend
- Use connection pooling for high concurrency
- Configure appropriate timeouts
- Monitor database performance
- Use SSL in production
- Regular database maintenance

### S3 Backend
- Configure bucket versioning
- Use KMS encryption for sensitive data
- Consider cross-region replication
- Monitor API costs
- Use IAM roles instead of access keys

## Migration

When migrating between backends, use the export/import functionality:

```go
// Export from old backend
data, err := oldBackend.GetState(ctx, "my-project")
if err != nil {
    log.Fatal(err)
}

// Import to new backend
err = newBackend.PutState(ctx, "my-project", data)
if err != nil {
    log.Fatal(err)
}
```

## Custom Backends

To implement a custom backend, implement the `StateBackend` interface and register it with the factory:

```go
type CustomBackend struct{}

func (b *CustomBackend) GetState(ctx context.Context, name string) (*types.UniversalState, error) {
    // Implementation
}

// Register with factory
factory.RegisterBackend("custom", func() state.StateBackend {
    return &CustomBackend{}
})
```