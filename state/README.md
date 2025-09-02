# Kolumn SDK State Management

This package provides a comprehensive state management system for the Kolumn SDK, enabling external providers to manage state across multiple backends with enhanced features for resource collections, dependency tracking, and drift detection.

## Key Components

### StateManager (`manager.go`)

The concrete implementation of the StateManager interface providing:

- **State lifecycle management** (load, save, validate)
- **Backend integration** with multiple storage types
- **Resource collection management** 
- **Cross-provider dependency tracking**
- **State migration and versioning support**
- **Import/export capabilities**
- **Backup and restore functionality**

### Enhanced Types (`enhanced_types.go`)

Extended type definitions supporting:

- **EnhancedResourceState** - Multi-provider resources with metadata
- **ResourceCollection** - Grouped resource management
- **ResourceDependency** - Cross-provider dependencies
- **ValidationRule** - Custom validation logic
- **ResourceGraph** - Dependency graph structures

### Resource Collections (`collections.go`)

Management of related resource groups:

- **Collection lifecycle** - Create, update, delete collections
- **Resource assignment** - Assign/unassign resources to collections
- **Health monitoring** - Track collection status and health
- **Dependency validation** - Ensure collection integrity

### Dependency Management (`dependencies.go`)

Graph-based dependency analysis:

- **Execution ordering** - Topological sort for resource operations
- **Cycle detection** - Identify circular dependencies
- **Impact analysis** - Understand change propagation
- **Dependency validation** - Verify dependency satisfaction

### Drift Detection (`drift_detector.go`)

Automated drift detection and resolution:

- **State comparison** - Compare desired vs actual state
- **Confidence scoring** - Assess drift detection accuracy  
- **Auto-resolution** - Automatically resolve low-risk drift
- **Resolution strategies** - Multiple drift handling approaches

### Backend Integration

Uses the factory pattern for backend creation:

- **Memory** - In-memory storage for development/testing
- **Local** - File-based storage
- **PostgreSQL** - Database backend with ACID properties
- **S3** - Object storage with versioning support

## Usage Examples

### Basic State Manager Setup

```go
import (
    "context"
    "github.com/schemabounce/kolumn/sdk/state"
)

// Create manager with default configuration
config := state.DefaultManagerConfig()
config.BackendType = "postgres"
config.BackendConfig = map[string]interface{}{
    "database": "kolumn_state",
    "username": "kolumn",
    "password": "secret",
}

manager := state.NewManager(config)

// Initialize with custom settings
err := manager.Initialize(context.Background(), map[string]interface{}{
    "workspace_name": "production",
    "environment": "prod",
    "enable_drift_detection": true,
})
```

### State Operations

```go
// Store state
state := &types.UniversalState{
    Version:   1,
    Serial:    1,
    Lineage:   "unique-lineage-id",
    Resources: []types.UniversalResource{},
    // ... other fields
}

err := manager.PutState(ctx, "my-state", state)

// Retrieve state
retrievedState, err := manager.GetState(ctx, "my-state")

// List all states
states, err := manager.ListStates(ctx)
```

### Resource Collections

```go
// Get collection manager
collectionMgr := manager.GetCollectionManager()

// Create collection
definition := &state.CollectionDefinition{
    ID:   "data-pipeline",
    Name: "Data Processing Pipeline",
    Type: "pipeline",
    Resources: []state.ResourceReference{
        {Provider: "postgres", Type: "table", Name: "customers"},
        {Provider: "kafka", Type: "topic", Name: "events"},
        {Provider: "s3", Type: "bucket", Name: "data-lake"},
    },
}

collection, err := collectionMgr.CreateCollection(ctx, definition)
```

### Dependency Analysis

```go
// Get dependency manager
depMgr := manager.GetDependencyManager()

// Analyze resource dependencies
analysis, err := depMgr.AnalyzeGraph(ctx, "my-state")

// Find execution order
resourceIDs := []string{"postgres.table.users", "kafka.topic.events"}
batches, err := depMgr.FindExecutionOrder(ctx, "my-state", resourceIDs)
```

### Drift Detection

```go
// Get drift detector
driftDetector := manager.GetDriftDetector()

// Detect drift
state, err := manager.GetState(ctx, "my-state")
analysis, err := driftDetector.DetectDrift(ctx, state)

if analysis.HasDrift {
    // Resolve drift automatically
    err = driftDetector.ResolveDrift(ctx, analysis)
}
```

### Custom State Adapter

```go
// Implement StateAdapter interface
type MyProviderAdapter struct{}

func (a *MyProviderAdapter) ToUniversalState(providerState interface{}) (*types.UniversalState, error) {
    // Convert provider-specific state to universal format
}

func (a *MyProviderAdapter) FromUniversalState(universalState *types.UniversalState) (interface{}, error) {
    // Convert universal state to provider-specific format
}

// Register adapter
adapter := &MyProviderAdapter{}
err := manager.RegisterAdapter("my-provider", adapter)
```

## Configuration Options

### ManagerConfig

```go
type ManagerConfig struct {
    WorkspaceName        string        // Workspace identifier
    Environment          string        // Environment (dev/staging/prod)
    BackendType          string        // Backend type (memory/local/postgres/s3)
    BackendConfig        map[string]interface{} // Backend-specific config
    EnableDriftDetection bool          // Enable automatic drift detection
    DriftCheckInterval   time.Duration // How often to check for drift
    MaxBackups          int           // Maximum number of backups to keep
    StateVersion        int64         // State format version
}
```

### Backend Configurations

**PostgreSQL Backend:**
```go
backendConfig := map[string]interface{}{
    "host":     "localhost",
    "port":     5432,
    "database": "kolumn_state",
    "username": "kolumn",
    "password": "secret",
    "ssl_mode": "require",
    "schema":   "state",
}
```

**S3 Backend:**
```go
backendConfig := map[string]interface{}{
    "bucket":        "my-state-bucket",
    "region":        "us-east-1",
    "key_prefix":    "kolumn/state/",
    "storage_class": "STANDARD",
    "encrypt":       true,
}
```

## State Migration

The state manager supports automatic migration between state format versions:

```go
// Migrate state to newer version
err := manager.Migrate(ctx, "my-state", "2")

// View migration history
history := manager.GetMigrationHistory()
for _, record := range history {
    fmt.Printf("Migration: %s -> %s (%s)\n", 
        record.FromVersion, record.ToVersion, record.Status)
}
```

## Error Handling

The state manager provides detailed error information:

```go
state, err := manager.GetState(ctx, "non-existent")
if err != nil {
    // Handle different error types
    switch err.(type) {
    case *StateNotFoundError:
        // State doesn't exist
    case *LockError:
        // State is locked
    case *ValidationError:
        // State validation failed
    default:
        // Other errors
    }
}
```

## Thread Safety

The DefaultManager is thread-safe and can be used concurrently:

- Internal mutex protection for shared state
- Backend operations are atomic where supported
- Collection and dependency operations are synchronized

## Testing

The package includes comprehensive tests demonstrating usage patterns:

```go
go test ./sdk/state/...
```

Tests cover:
- Manager initialization and configuration
- State CRUD operations  
- Collection management
- Dependency analysis
- Drift detection
- Backend integration
- Error scenarios

## Integration with External Providers

External providers can integrate with the state management system by:

1. **Implementing StateAdapter** - Convert between provider and universal state formats
2. **Registering with Manager** - Register adapters for specific provider types
3. **Using Enhanced Types** - Leverage rich metadata and dependency tracking
4. **Participating in Collections** - Group related resources across providers
5. **Supporting Drift Detection** - Enable automatic state reconciliation

This design allows providers to focus on their core functionality while leveraging the shared state management infrastructure provided by the SDK.