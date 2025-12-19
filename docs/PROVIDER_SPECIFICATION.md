# Kolumn Provider Specification

**Version**: 1.0.0
**Last Updated**: December 2025
**Status**: Authoritative Specification

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [Provider Interface (Core 4-Method Contract)](#2-provider-interface-core-4-method-contract)
3. [Tiered Function Requirements](#3-tiered-function-requirements)
4. [Streaming Functions](#4-streaming-functions)
5. [Request/Response Type Definitions](#5-requestresponse-type-definitions)
6. [Resource Handler Registration](#6-resource-handler-registration)
7. [Schema Definition Requirements](#7-schema-definition-requirements)
8. [State Management](#8-state-management)
9. [Error Handling Standards](#9-error-handling-standards)
10. [Security Requirements](#10-security-requirements)
11. [Configuration Standards](#11-configuration-standards)
12. [Logging Standards](#12-logging-standards)
13. [Testing Requirements](#13-testing-requirements)
14. [Provider Categories and Specifics](#14-provider-categories-and-specifics)
15. [Compliance Checklist](#15-compliance-checklist)
16. [Version History and Migration Guide](#16-version-history-and-migration-guide)

---

## 1. Introduction

### 1.1 Purpose

This specification defines the complete requirements for implementing Kolumn database providers. All providers MUST conform to this specification to ensure consistent behavior, enterprise-grade reliability, and seamless integration with the Kolumn CLI and SchemaBounce platform.

### 1.2 Scope

This specification applies to ALL Kolumn database providers including:

**SQL Relational Databases**:
- PostgreSQL, MySQL, MSSQL, SQLite, CockroachDB

**Analytical Warehouses**:
- Snowflake, BigQuery, Redshift, Databricks, DuckDB

**NoSQL Databases**:
- MongoDB, DynamoDB

**Time-Series Databases**:
- InfluxDB

### 1.3 Target Audience

- Provider developers implementing new database providers
- Contributors maintaining existing providers
- Quality assurance engineers validating provider compliance
- Security auditors reviewing provider implementations

### 1.4 Definitions

| Term | Definition |
|------|------------|
| Provider | A standalone binary that implements database operations via RPC |
| Handler | A function that processes a specific resource type operation |
| Registry | A map of resource types to their handlers |
| Dispatcher | Component that routes function calls to appropriate handlers |
| Resource | A managed database object (table, view, function, etc.) |
| State | Persisted information about managed resources |

---

## 2. Provider Interface (Core 4-Method Contract)

### 2.1 Overview

Every Kolumn provider MUST implement the following 4-method interface. This is enforced by the SDK and required for RPC communication with the Kolumn CLI.

```go
type Provider interface {
    // Configure initializes the provider with configuration
    Configure(context.Context, map[string]interface{}) error

    // Schema returns the provider's resource schema
    Schema() (*rpc.ProviderSchema, error)

    // CallFunction handles all resource operations
    CallFunction(context.Context, string, []byte) ([]byte, error)

    // Close cleans up provider resources
    Close() error
}
```

### 2.2 Configure Method

**Purpose**: Initialize the provider with database connection details and settings.

**Signature**:
```go
Configure(ctx context.Context, config map[string]interface{}) error
```

**Requirements**:
- MUST validate all required configuration fields
- MUST establish database connection
- MUST NOT store plain-text passwords in memory longer than necessary
- MUST return descriptive errors for invalid configuration
- MUST support timeout from context

**Configuration Fields** (minimum required):
```go
type ProviderConfiguration struct {
    // Connection settings
    Host     string `json:"host"`
    Port     int    `json:"port"`
    Database string `json:"database"`
    Username string `json:"username"`
    Password string `json:"password"`

    // Optional settings
    SSLMode         string `json:"ssl_mode"`
    ConnectionPool  int    `json:"connection_pool"`
    QueryTimeout    int    `json:"query_timeout"`
    SafeDestroy     bool   `json:"safe_destroy"`
}
```

**Example Implementation**:
```go
func (p *Provider) Configure(ctx context.Context, config map[string]interface{}) error {
    // Parse configuration
    cfg, err := parseConfiguration(config)
    if err != nil {
        return fmt.Errorf("invalid configuration: %w", err)
    }

    // Validate required fields
    if cfg.Host == "" {
        return errors.New("host is required")
    }

    // Establish connection
    db, err := sql.Open("postgres", cfg.ConnectionString())
    if err != nil {
        return fmt.Errorf("failed to connect: %w", err)
    }

    // Test connection
    if err := db.PingContext(ctx); err != nil {
        return fmt.Errorf("connection test failed: %w", err)
    }

    p.db = db
    p.config = cfg
    return nil
}
```

### 2.3 Schema Method

**Purpose**: Return the provider's complete resource schema for validation and documentation.

**Signature**:
```go
Schema() (*rpc.ProviderSchema, error)
```

**Requirements**:
- MUST return all supported resource types
- MUST include configuration schema for each resource type
- MUST include state schema for each resource type
- MUST be consistent with registered handlers

**Example Implementation**:
```go
func (p *Provider) Schema() (*rpc.ProviderSchema, error) {
    return p.unifiedDispatcher.BuildCompatibleSchema(
        "postgres",              // Provider name
        "1.0.0",                 // Version
        "database",              // Category
        "PostgreSQL provider",   // Description
    ), nil
}
```

### 2.4 CallFunction Method

**Purpose**: Route all operations to appropriate handlers.

**Signature**:
```go
CallFunction(ctx context.Context, function string, input []byte) ([]byte, error)
```

**Requirements**:
- MUST route all Tier 1 functions (see Section 3)
- MUST route all applicable Tier 2 functions
- MUST route all Tier 3 migration functions
- MUST route all streaming functions
- MUST return `unsupported function` error for unknown functions
- MUST propagate context for cancellation and timeouts

**Required Function Routing** (minimum):
```go
func (p *Provider) CallFunction(ctx context.Context, function string, input []byte) ([]byte, error) {
    switch function {
    // Tier 1 - Universal (Required ALL)
    case "CreateResource", "ReadResource", "UpdateResource", "DeleteResource", "DiscoverResources":
        return p.unifiedDispatcher.Dispatch(ctx, function, input)
    case "Plan":
        return p.handlePlan(ctx, input)
    case "Apply":
        return p.handleApply(ctx, input)
    case "GetState":
        return p.handleGetState(ctx, input)
    case "SetState":
        return p.handleSetState(ctx, input)
    case "Ping":
        return p.handlePing(ctx, input)
    case "Version":
        return p.handleVersion(ctx, input)

    // Tier 2 - Enterprise (Required SQL/Analytical)
    case "DetectDrift":
        return p.handleDetectDrift(ctx, input)
    case "ValidateState":
        return p.handleValidateState(ctx, input)
    case "GetAuditLog":
        return p.handleGetAuditLog(ctx, input)

    // Tier 2 - Transactions (Required if supported)
    case "BeginTransaction":
        return p.handleBeginTransaction(ctx, input)
    case "CommitTransaction":
        return p.handleCommitTransaction(ctx, input)
    case "RollbackTransaction":
        return p.handleRollbackTransaction(ctx, input)

    // Tier 3 - Migrations (Required ALL)
    case "PlanMigration":
        return p.handlePlanMigration(ctx, input)
    case "ApplyMigration":
        return p.handleApplyMigration(ctx, input)
    case "RollbackMigration":
        return p.handleRollbackMigration(ctx, input)

    // Streaming (Required ALL)
    case "OutboxHealthMetrics":
        return p.handleOutboxHealthMetrics(ctx, input)
    case "OutboxCleanupRetention":
        return p.handleOutboxCleanupRetention(ctx, input)

    default:
        return nil, fmt.Errorf("unsupported function: %s", function)
    }
}
```

### 2.5 Close Method

**Purpose**: Clean up provider resources and close connections.

**Signature**:
```go
Close() error
```

**Requirements**:
- MUST close database connections
- MUST release connection pool resources
- MUST clear sensitive data from memory
- SHOULD be idempotent (safe to call multiple times)

**Example Implementation**:
```go
func (p *Provider) Close() error {
    if p.db != nil {
        if err := p.db.Close(); err != nil {
            return fmt.Errorf("failed to close database: %w", err)
        }
        p.db = nil
    }

    // Clear sensitive configuration
    if p.config != nil {
        p.config.Password = ""
    }

    return nil
}
```

---

## 3. Tiered Function Requirements

### 3.1 Tier Overview

Functions are organized into tiers based on requirement level:

| Tier | Requirement | Description |
|------|-------------|-------------|
| Tier 1 | Universal | Required for ALL providers |
| Tier 2 | Conditional | Required based on provider type |
| Tier 3 | Migration | Required for ALL providers |

### 3.2 Tier 1 - Universal Functions (Required ALL)

Every provider MUST implement these 11 functions:

| Function | Description | Request Type | Response Type |
|----------|-------------|--------------|---------------|
| `CreateResource` | Create new database resource | CreateResourceRequest | CreateResourceResponse |
| `ReadResource` | Read existing resource state | ReadResourceRequest | ReadResourceResponse |
| `UpdateResource` | Modify existing resource | UpdateResourceRequest | UpdateResourceResponse |
| `DeleteResource` | Remove resource | DeleteResourceRequest | DeleteResourceResponse |
| `DiscoverResources` | Auto-discover infrastructure | DiscoverResourcesRequest | DiscoverResourcesResponse |
| `Plan` | Generate execution plan with dependency tree | PlanRequest | PlanResponse |
| `Apply` | Execute planned changes in dependency order | PlanRequest | ApplyResponse |
| `GetState` | Retrieve provider state | StateRequest | StateResponse |
| `SetState` | Update provider state | SetStateRequest | SetStateResponse |
| `Ping` | Health check connection | PingRequest | PingResponse |
| `Version` | Return provider version | VersionRequest | VersionResponse |

**Dependency Tree Requirement**: The `Plan` and `Apply` functions MUST implement dependency tree analysis. Resources MUST be executed in dependency order (e.g., schemas before tables, tables before foreign keys). The `PlanResponse` MUST include an `execution_order` field, and `Apply` MUST execute resources in that order.

### 3.3 Tier 2 - Conditional Functions

#### 3.3.1 Enterprise Functions (Required: SQL Relational + Analytical)

Required for: PostgreSQL, MySQL, MSSQL, SQLite, CockroachDB, Snowflake, BigQuery, Redshift, Databricks, DuckDB

| Function | Description | Request Type | Response Type |
|----------|-------------|--------------|---------------|
| `DetectDrift` | Detect configuration drift | DriftRequest | DriftResponse |
| `ValidateState` | Validate state consistency | ValidateRequest | ValidateResponse |
| `GetAuditLog` | Retrieve audit trail | AuditRequest | AuditResponse |

#### 3.3.2 Transaction Functions (Required: If Database Supports Transactions)

Required for: PostgreSQL, MySQL, MSSQL, SQLite, CockroachDB, MongoDB (4.0+), DynamoDB (TransactWriteItems)

| Function | Description | Request Type | Response Type |
|----------|-------------|--------------|---------------|
| `BeginTransaction` | Start transaction | TransactionRequest | TransactionResponse |
| `CommitTransaction` | Commit transaction | TransactionRequest | TransactionResponse |
| `RollbackTransaction` | Rollback transaction | TransactionRequest | TransactionResponse |

**Note**: InfluxDB is exempt from transaction requirements due to time-series nature.

### 3.4 Tier 3 - Migration Functions (Required ALL)

Every provider MUST implement these migration functions:

| Function | Description | Request Type | Response Type |
|----------|-------------|--------------|---------------|
| `PlanMigration` | Plan schema migration | MigrationRequest | MigrationPlanResponse |
| `ApplyMigration` | Execute schema migration | MigrationRequest | MigrationApplyResponse |
| `RollbackMigration` | Rollback failed migration | MigrationRequest | MigrationRollbackResponse |

### 3.5 Function Implementation Examples

#### CreateResource
```go
func (p *Provider) handleCreateResource(ctx context.Context, input []byte) ([]byte, error) {
    var req rpc.CreateResourceRequest
    if err := json.Unmarshal(input, &req); err != nil {
        return nil, rpc.NewSecureError("invalid request format")
    }

    // Get handler from registry
    handler := p.createRegistry.GetHandler(req.ResourceType)
    if handler == nil {
        return nil, rpc.NewSecureError("unsupported resource type: %s", req.ResourceType)
    }

    // Execute creation
    result, err := handler.Create(ctx, req.Config)
    if err != nil {
        return nil, err
    }

    return json.Marshal(rpc.CreateResourceResponse{
        Success:     true,
        ResourceID:  result.ID,
        State:       result.State,
    })
}
```

#### Plan
```go
func (p *Provider) handlePlan(ctx context.Context, input []byte) ([]byte, error) {
    var req rpc.PlanRequest
    if err := json.Unmarshal(input, &req); err != nil {
        return nil, rpc.NewSecureError("invalid request format")
    }

    // Initialize enterprise safety frameworks
    capabilityProvider := rpc.NewCapabilityProvider(p.db)
    validator := rpc.NewDataValidationFramework(capabilityProvider)
    dependencyManager := rpc.NewConstraintDependencyManager(capabilityProvider)
    rollbackGenerator := rpc.NewRollbackSQLGenerator(capabilityProvider)

    // Analyze dependencies and build execution order
    dependencyPlan, err := p.analyzeProviderDependencies(req.Resources, dependencyManager)
    if err != nil {
        return nil, err
    }

    // Generate plan with validation
    plan := &rpc.PlanResponse{
        ExecutionOrder: dependencyPlan["execution_order"].([]string),
        Resources:      make([]rpc.PlannedResource, 0),
        Warnings:       make([]string, 0),
    }

    for _, resource := range req.Resources {
        plannedResource, warnings := p.planResource(ctx, resource, validator)
        plan.Resources = append(plan.Resources, plannedResource)
        plan.Warnings = append(plan.Warnings, warnings...)
    }

    return json.Marshal(plan)
}
```

#### Apply
```go
func (p *Provider) handleApply(ctx context.Context, input []byte) ([]byte, error) {
    // CRITICAL: Use same request type as Plan
    var req rpc.PlanRequest
    if err := json.Unmarshal(input, &req); err != nil {
        return nil, rpc.NewSecureError("invalid request format")
    }

    // Initialize IDENTICAL frameworks as Plan
    capabilityProvider := rpc.NewCapabilityProvider(p.db)
    validator := rpc.NewDataValidationFramework(capabilityProvider)
    dependencyManager := rpc.NewConstraintDependencyManager(capabilityProvider)
    rollbackGenerator := rpc.NewRollbackSQLGenerator(capabilityProvider)
    backupFramework := shared.NewBackupIntegrityFramework(p.name, "/tmp/backups")

    // Get execution order from dependency analysis
    dependencyPlan, err := p.analyzeProviderDependencies(req.Resources, dependencyManager)
    if err != nil {
        return nil, err
    }

    // Execute resources in dependency order
    results, err := p.executeResourcesInDependencyOrder(ctx, req.Resources, dependencyPlan)
    if err != nil {
        return nil, err
    }

    return json.Marshal(rpc.ApplyResponse{
        Success: true,
        Results: results,
    })
}
```

---

## 4. Streaming Functions

### 4.1 Overview

All providers MUST support streaming functions for CDC (Change Data Capture) integration with the SchemaBounce platform.

### 4.2 Required Streaming Handlers

| Resource Type | Handler | Description |
|---------------|---------|-------------|
| `stream_sink` | StreamSinkHandler | Handle CDC sink operations |
| `stream_route` | StreamRouteHandler | Handle stream routing |
| `stream_outbox` | StreamOutboxHandler | Handle outbox pattern |

### 4.3 Required Streaming Functions

| Function | Description | Required |
|----------|-------------|----------|
| `OutboxHealthMetrics` | Return outbox health metrics | ALL |
| `OutboxCleanupRetention` | Clean old outbox entries | ALL |

### 4.4 Streaming Handler Registration

```go
// Register streaming handlers in provider initialization
func (p *Provider) registerStreamingHandlers() {
    streamingHandlers := shared.NewStreamingHandlers(p.db, p.logger)

    p.createRegistry.Register("stream_sink", streamingHandlers.StreamSinkHandler)
    p.createRegistry.Register("stream_route", streamingHandlers.StreamRouteHandler)
    p.createRegistry.Register("stream_outbox", streamingHandlers.StreamOutboxHandler)

    p.discoverRegistry.Register("stream_sink", streamingHandlers.StreamSinkDiscoverer)
    p.discoverRegistry.Register("stream_route", streamingHandlers.StreamRouteDiscoverer)
    p.discoverRegistry.Register("stream_outbox", streamingHandlers.StreamOutboxDiscoverer)
}
```

### 4.5 OutboxHealthMetrics Implementation

```go
func (p *Provider) handleOutboxHealthMetrics(ctx context.Context, input []byte) ([]byte, error) {
    metrics := shared.OutboxHealthMetrics{
        TotalMessages:     0,
        PendingMessages:   0,
        ProcessedMessages: 0,
        FailedMessages:    0,
        OldestPending:     nil,
        LastProcessed:     nil,
        Healthy:           true,
    }

    // Query outbox table for metrics
    row := p.db.QueryRowContext(ctx, `
        SELECT
            COUNT(*) as total,
            COUNT(*) FILTER (WHERE status = 'pending') as pending,
            COUNT(*) FILTER (WHERE status = 'processed') as processed,
            COUNT(*) FILTER (WHERE status = 'failed') as failed,
            MIN(created_at) FILTER (WHERE status = 'pending') as oldest_pending,
            MAX(processed_at) as last_processed
        FROM stream_outbox
    `)

    if err := row.Scan(
        &metrics.TotalMessages,
        &metrics.PendingMessages,
        &metrics.ProcessedMessages,
        &metrics.FailedMessages,
        &metrics.OldestPending,
        &metrics.LastProcessed,
    ); err != nil {
        return nil, err
    }

    // Determine health status
    metrics.Healthy = metrics.FailedMessages == 0 && metrics.PendingMessages < 1000

    return json.Marshal(metrics)
}
```

---

## 5. Request/Response Type Definitions

### 5.1 Core Types

```go
// CreateResourceRequest represents a request to create a resource
type CreateResourceRequest struct {
    ResourceType string                 `json:"resource_type"`
    Name         string                 `json:"name"`
    Config       map[string]interface{} `json:"config"`
}

// CreateResourceResponse represents the result of resource creation
type CreateResourceResponse struct {
    Success    bool                   `json:"success"`
    ResourceID string                 `json:"resource_id,omitempty"`
    State      map[string]interface{} `json:"state,omitempty"`
    Error      string                 `json:"error,omitempty"`
}

// ReadResourceRequest represents a request to read a resource
type ReadResourceRequest struct {
    ResourceType string `json:"resource_type"`
    Name         string `json:"name"`
}

// ReadResourceResponse represents the result of reading a resource
type ReadResourceResponse struct {
    Success bool                   `json:"success"`
    Exists  bool                   `json:"exists"`
    State   map[string]interface{} `json:"state,omitempty"`
    Error   string                 `json:"error,omitempty"`
}

// UpdateResourceRequest represents a request to update a resource
type UpdateResourceRequest struct {
    ResourceType string                 `json:"resource_type"`
    Name         string                 `json:"name"`
    Config       map[string]interface{} `json:"config"`
    PriorState   map[string]interface{} `json:"prior_state,omitempty"`
}

// UpdateResourceResponse represents the result of updating a resource
type UpdateResourceResponse struct {
    Success bool                   `json:"success"`
    State   map[string]interface{} `json:"state,omitempty"`
    Error   string                 `json:"error,omitempty"`
}

// DeleteResourceRequest represents a request to delete a resource
type DeleteResourceRequest struct {
    ResourceType string                 `json:"resource_type"`
    Name         string                 `json:"name"`
    State        map[string]interface{} `json:"state,omitempty"`
}

// DeleteResourceResponse represents the result of deleting a resource
type DeleteResourceResponse struct {
    Success           bool                   `json:"success"`
    QuarantineLocation string                `json:"quarantine_location,omitempty"`
    BeforeState       map[string]interface{} `json:"before_state,omitempty"`
    Error             string                 `json:"error,omitempty"`
}

// DiscoverResourcesRequest represents a request to discover resources
type DiscoverResourcesRequest struct {
    ResourceType string                 `json:"resource_type"`
    Filters      map[string]interface{} `json:"filters,omitempty"`
}

// DiscoverResourcesResponse represents discovered resources
type DiscoverResourcesResponse struct {
    Success   bool                     `json:"success"`
    Resources []DiscoveredResource     `json:"resources"`
    Error     string                   `json:"error,omitempty"`
}

// DiscoveredResource represents a discovered resource
type DiscoveredResource struct {
    Type   string                 `json:"type"`
    Name   string                 `json:"name"`
    Config map[string]interface{} `json:"config"`
    State  map[string]interface{} `json:"state"`
}
```

### 5.2 Plan/Apply Types

```go
// PlanRequest represents a request to plan resource operations
type PlanRequest struct {
    Resources []PlanResource `json:"resources"`
}

// PlanResource represents a resource operation to be planned
type PlanResource struct {
    ResourceType string                 `json:"resource_type"`
    Name         string                 `json:"name"`
    Config       map[string]interface{} `json:"config"`
    Action       string                 `json:"action"` // "create", "update", "delete"
}

// PlanResponse represents the execution plan
type PlanResponse struct {
    Success        bool              `json:"success"`
    ExecutionOrder []string          `json:"execution_order"`
    Resources      []PlannedResource `json:"resources"`
    Warnings       []string          `json:"warnings,omitempty"`
    Error          string            `json:"error,omitempty"`
}

// PlannedResource represents a planned resource change
type PlannedResource struct {
    ResourceType string                 `json:"resource_type"`
    Name         string                 `json:"name"`
    Action       string                 `json:"action"`
    Changes      map[string]interface{} `json:"changes,omitempty"`
    Dependencies []string               `json:"dependencies,omitempty"`
}

// ApplyResponse represents the result of applying changes
type ApplyResponse struct {
    Success  bool                   `json:"success"`
    Results  []ApplyResult          `json:"results"`
    Summary  ApplySummary           `json:"summary"`
    Error    string                 `json:"error,omitempty"`
}

// ApplyResult represents the result of a single resource operation
type ApplyResult struct {
    ResourceType string                 `json:"resource_type"`
    Name         string                 `json:"name"`
    Action       string                 `json:"action"`
    Success      bool                   `json:"success"`
    State        map[string]interface{} `json:"state,omitempty"`
    Error        string                 `json:"error,omitempty"`
}

// ApplySummary represents a summary of applied changes
type ApplySummary struct {
    Created   int `json:"created"`
    Updated   int `json:"updated"`
    Deleted   int `json:"deleted"`
    Unchanged int `json:"unchanged"`
    Failed    int `json:"failed"`
}
```

### 5.3 State Types

```go
// StateRequest represents a request for state information
type StateRequest struct {
    // Empty - returns full state
}

// StateResponse represents the provider state
type StateResponse struct {
    Success   bool           `json:"success"`
    State     *ProviderState `json:"state,omitempty"`
    Error     string         `json:"error,omitempty"`
}

// SetStateRequest represents a request to set state
type SetStateRequest struct {
    State *ProviderState `json:"state"`
}

// SetStateResponse represents the result of setting state
type SetStateResponse struct {
    Success bool   `json:"success"`
    Error   string `json:"error,omitempty"`
}

// ProviderState represents the complete provider state
type ProviderState struct {
    Version    string             `json:"version"`
    Provider   string             `json:"provider"`
    Resources  []ResourceInstance `json:"resources"`
    Metadata   map[string]interface{} `json:"metadata,omitempty"`
    UpdatedAt  time.Time          `json:"updated_at"`
}

// ResourceInstance represents a managed resource instance
type ResourceInstance struct {
    Type       string                 `json:"type"`
    Name       string                 `json:"name"`
    Provider   string                 `json:"provider"`
    Status     string                 `json:"status"` // "active", "quarantined", "deleted"
    Config     map[string]interface{} `json:"config"`
    State      map[string]interface{} `json:"state"`
    Metadata   map[string]interface{} `json:"metadata,omitempty"`
    CreatedAt  time.Time              `json:"created_at"`
    UpdatedAt  time.Time              `json:"updated_at"`
}
```

### 5.4 Migration Types

```go
// MigrationRequest represents a migration operation request
type MigrationRequest struct {
    MigrationID   string                 `json:"migration_id"`
    ResourceType  string                 `json:"resource_type"`
    ResourceName  string                 `json:"resource_name"`
    FromState     map[string]interface{} `json:"from_state"`
    ToState       map[string]interface{} `json:"to_state"`
    Options       MigrationOptions       `json:"options,omitempty"`
}

// MigrationOptions represents migration configuration
type MigrationOptions struct {
    DryRun           bool `json:"dry_run"`
    AllowDataLoss    bool `json:"allow_data_loss"`
    BackupBeforeMigration bool `json:"backup_before_migration"`
    Timeout          int  `json:"timeout_seconds"`
}

// MigrationPlanResponse represents a planned migration
type MigrationPlanResponse struct {
    Success      bool              `json:"success"`
    MigrationID  string            `json:"migration_id"`
    Steps        []MigrationStep   `json:"steps"`
    Warnings     []string          `json:"warnings,omitempty"`
    DataLossRisk bool              `json:"data_loss_risk"`
    Error        string            `json:"error,omitempty"`
}

// MigrationStep represents a single migration step
type MigrationStep struct {
    Order       int    `json:"order"`
    Description string `json:"description"`
    SQL         string `json:"sql,omitempty"`
    Reversible  bool   `json:"reversible"`
}

// MigrationApplyResponse represents migration execution result
type MigrationApplyResponse struct {
    Success       bool              `json:"success"`
    MigrationID   string            `json:"migration_id"`
    ExecutedSteps []MigrationStep   `json:"executed_steps"`
    BackupPath    string            `json:"backup_path,omitempty"`
    Duration      time.Duration     `json:"duration"`
    Error         string            `json:"error,omitempty"`
}

// MigrationRollbackResponse represents rollback result
type MigrationRollbackResponse struct {
    Success       bool              `json:"success"`
    MigrationID   string            `json:"migration_id"`
    RolledBackSteps []MigrationStep `json:"rolled_back_steps"`
    Duration      time.Duration     `json:"duration"`
    Error         string            `json:"error,omitempty"`
}
```

### 5.5 Utility Types

```go
// PingRequest represents a health check request
type PingRequest struct {
    // Empty
}

// PingResponse represents health check result
type PingResponse struct {
    Success   bool   `json:"success"`
    Latency   int64  `json:"latency_ms"`
    Message   string `json:"message,omitempty"`
}

// VersionRequest represents a version request
type VersionRequest struct {
    // Empty
}

// VersionResponse represents version information
type VersionResponse struct {
    Success       bool   `json:"success"`
    Version       string `json:"version"`
    BuildDate     string `json:"build_date,omitempty"`
    GitCommit     string `json:"git_commit,omitempty"`
    GoVersion     string `json:"go_version,omitempty"`
    ProviderName  string `json:"provider_name"`
}
```

---

## 6. Resource Handler Registration

### 6.1 Registry Architecture

Providers use a dual-registry system for resource handlers:

```go
type UnifiedDispatcher struct {
    createRegistry   *CreateRegistry   // Handlers for resource creation
    discoverRegistry *DiscoverRegistry // Handlers for resource discovery
    // ... other components
}
```

### 6.2 Registry Parity Rule

**CRITICAL**: Every resource type registered in `createRegistry` MUST have a corresponding handler in `discoverRegistry`.

```go
// CORRECT: Registry parity maintained
createRegistry.Register("table", &TableCreateHandler{})
discoverRegistry.Register("table", &TableDiscoverHandler{})

createRegistry.Register("view", &ViewCreateHandler{})
discoverRegistry.Register("view", &ViewDiscoverHandler{})

// INCORRECT: Missing discover handler
createRegistry.Register("function", &FunctionCreateHandler{})
// Missing: discoverRegistry.Register("function", ...)
```

### 6.3 Handler Interface

```go
// CreateHandler interface for resource creation
type CreateHandler interface {
    Create(ctx context.Context, config map[string]interface{}) (*CreateResult, error)
    Update(ctx context.Context, config map[string]interface{}, priorState map[string]interface{}) (*UpdateResult, error)
    Delete(ctx context.Context, state map[string]interface{}) (*DeleteResult, error)
    Schema() *ObjectType
}

// DiscoverHandler interface for resource discovery
type DiscoverHandler interface {
    Discover(ctx context.Context, filters map[string]interface{}) ([]DiscoveredResource, error)
    Introspect(ctx context.Context, request map[string]interface{}) ([]byte, error)
}
```

### 6.4 Registration Example

```go
func (p *Provider) initializeHandlers() {
    // Core database objects
    p.createRegistry.Register("schema", NewSchemaHandler(p.db))
    p.discoverRegistry.Register("schema", NewSchemaDiscoverer(p.db))

    p.createRegistry.Register("table", NewTableHandler(p.db))
    p.discoverRegistry.Register("table", NewTableDiscoverer(p.db))

    p.createRegistry.Register("view", NewViewHandler(p.db))
    p.discoverRegistry.Register("view", NewViewDiscoverer(p.db))

    p.createRegistry.Register("index", NewIndexHandler(p.db))
    p.discoverRegistry.Register("index", NewIndexDiscoverer(p.db))

    p.createRegistry.Register("function", NewFunctionHandler(p.db))
    p.discoverRegistry.Register("function", NewFunctionDiscoverer(p.db))

    // Streaming handlers (required for all)
    streamingHandlers := shared.NewStreamingHandlers(p.db, p.logger)
    p.createRegistry.Register("stream_sink", streamingHandlers.StreamSinkHandler)
    p.discoverRegistry.Register("stream_sink", streamingHandlers.StreamSinkDiscoverer)

    p.createRegistry.Register("stream_route", streamingHandlers.StreamRouteHandler)
    p.discoverRegistry.Register("stream_route", streamingHandlers.StreamRouteDiscoverer)

    p.createRegistry.Register("stream_outbox", streamingHandlers.StreamOutboxHandler)
    p.discoverRegistry.Register("stream_outbox", streamingHandlers.StreamOutboxDiscoverer)
}
```

---

## 7. Schema Definition Requirements

### 7.1 ProviderSchema Structure

```go
type ProviderSchema struct {
    Name          string                   `json:"name"`
    Version       string                   `json:"version"`
    Category      string                   `json:"category"`
    Description   string                   `json:"description"`
    ResourceTypes []ResourceTypeDefinition `json:"resource_types"`
}
```

### 7.2 ResourceTypeDefinition

```go
type ResourceTypeDefinition struct {
    Name         string       `json:"name"`
    Description  string       `json:"description"`
    ConfigSchema *ObjectType  `json:"config_schema"`
    StateSchema  *ObjectType  `json:"state_schema"`
}
```

### 7.3 ObjectType (Schema Definition)

```go
type ObjectType struct {
    Attributes map[string]*AttributeSchema `json:"attributes"`
}

type AttributeSchema struct {
    Type        string            `json:"type"`        // "string", "int", "bool", "list", "map"
    Description string            `json:"description"`
    Required    bool              `json:"required"`
    Computed    bool              `json:"computed"`    // Computed by provider
    Sensitive   bool              `json:"sensitive"`   // Contains sensitive data
    Default     interface{}       `json:"default,omitempty"`
    Validators  []ValidatorSpec   `json:"validators,omitempty"`
}

type ValidatorSpec struct {
    Type    string      `json:"type"`    // "regex", "enum", "range", "length"
    Value   interface{} `json:"value"`
    Message string      `json:"message"`
}
```

### 7.4 Schema Example

```go
func (h *TableHandler) Schema() *ObjectType {
    return &ObjectType{
        Attributes: map[string]*AttributeSchema{
            "name": {
                Type:        "string",
                Description: "Table name",
                Required:    true,
                Validators: []ValidatorSpec{
                    {Type: "regex", Value: "^[a-zA-Z_][a-zA-Z0-9_]*$", Message: "Invalid table name"},
                    {Type: "length", Value: map[string]int{"max": 63}, Message: "Name too long"},
                },
            },
            "schema": {
                Type:        "string",
                Description: "Schema name",
                Required:    false,
                Default:     "public",
            },
            "columns": {
                Type:        "list",
                Description: "Table columns",
                Required:    true,
            },
            "primary_key": {
                Type:        "list",
                Description: "Primary key columns",
                Required:    false,
            },
            "indexes": {
                Type:        "list",
                Description: "Table indexes",
                Required:    false,
            },
            "created_at": {
                Type:        "string",
                Description: "Creation timestamp",
                Computed:    true,
            },
        },
    }
}
```

---

## 8. State Management

### 8.1 State File Format

```json
{
    "version": "1.0.0",
    "provider": "postgres",
    "resources": [
        {
            "type": "postgres_table",
            "name": "users",
            "provider": "postgres.production",
            "status": "active",
            "config": {
                "schema": "public",
                "name": "users",
                "columns": [...]
            },
            "state": {
                "oid": "16384",
                "row_count": 1500,
                "size_bytes": 262144
            },
            "metadata": {
                "created_by": "kolumn",
                "managed": true
            },
            "created_at": "2025-12-15T10:30:00Z",
            "updated_at": "2025-12-15T10:30:00Z"
        }
    ],
    "metadata": {
        "last_plan": "2025-12-15T10:30:00Z",
        "last_apply": "2025-12-15T10:30:00Z"
    },
    "updated_at": "2025-12-15T10:30:00Z"
}
```

### 8.2 State Operations

#### GetState
```go
func (p *Provider) handleGetState(ctx context.Context, input []byte) ([]byte, error) {
    p.stateMu.RLock()
    defer p.stateMu.RUnlock()

    return json.Marshal(StateResponse{
        Success: true,
        State:   p.state,
    })
}
```

#### SetState
```go
func (p *Provider) handleSetState(ctx context.Context, input []byte) ([]byte, error) {
    var req SetStateRequest
    if err := json.Unmarshal(input, &req); err != nil {
        return nil, rpc.NewSecureError("invalid request format")
    }

    p.stateMu.Lock()
    defer p.stateMu.Unlock()

    // Validate state version
    if p.state != nil && req.State.Version < p.state.Version {
        return nil, rpc.NewSecureError("state version conflict")
    }

    p.state = req.State
    p.state.UpdatedAt = time.Now()

    return json.Marshal(SetStateResponse{Success: true})
}
```

### 8.3 Resource Status Transitions

```
                    ┌─────────────────┐
                    │                 │
     Create ───────►│     active      │◄────── Update
                    │                 │
                    └────────┬────────┘
                             │
                      Delete │ (with safe_destroy)
                             │
                             ▼
                    ┌─────────────────┐
                    │                 │
                    │   quarantined   │───────► Permanent Delete
                    │                 │         (after retention)
                    └─────────────────┘
```

---

## 9. Error Handling Standards

### 9.1 SecureError

All provider errors MUST use the `SecureError` type to prevent sensitive data leakage:

```go
// NewSecureError creates a secure error that masks sensitive information
func NewSecureError(format string, args ...interface{}) error {
    // Sanitize arguments to remove potential secrets
    sanitizedArgs := make([]interface{}, len(args))
    for i, arg := range args {
        sanitizedArgs[i] = sanitize(arg)
    }
    return fmt.Errorf(format, sanitizedArgs...)
}

// sanitize removes or masks sensitive patterns
func sanitize(v interface{}) interface{} {
    s, ok := v.(string)
    if !ok {
        return v
    }

    // Mask connection strings
    if strings.Contains(s, "password=") {
        return "[REDACTED CONNECTION STRING]"
    }

    // Mask tokens
    if strings.HasPrefix(s, "sk-") || strings.HasPrefix(s, "token-") {
        return "[REDACTED TOKEN]"
    }

    return s
}
```

### 9.2 Error Categories

| Category | Description | Example |
|----------|-------------|---------|
| ValidationError | Invalid input/configuration | "column name required" |
| ConnectionError | Database connectivity issues | "connection refused" |
| OperationError | Operation execution failed | "table already exists" |
| StateError | State management issues | "state version conflict" |
| PermissionError | Authorization failures | "permission denied" |

### 9.3 Error Response Format

```go
type ErrorResponse struct {
    Success bool   `json:"success"`
    Error   string `json:"error"`
    Code    string `json:"code,omitempty"`    // Error code for programmatic handling
    Details map[string]interface{} `json:"details,omitempty"` // Additional context
}
```

### 9.4 Error Handling Example

```go
func (h *TableHandler) Create(ctx context.Context, config map[string]interface{}) (*CreateResult, error) {
    name, ok := config["name"].(string)
    if !ok || name == "" {
        return nil, rpc.NewSecureError("table name is required")
    }

    // Validate name format
    if !isValidIdentifier(name) {
        return nil, rpc.NewSecureError("invalid table name format")
    }

    // Check for existing table
    exists, err := h.tableExists(ctx, name)
    if err != nil {
        return nil, rpc.NewSecureError("failed to check table existence")
    }
    if exists {
        return nil, rpc.NewSecureError("table already exists: %s", name)
    }

    // Create table
    if err := h.createTable(ctx, config); err != nil {
        // Log detailed error internally
        h.logger.Error("table creation failed", "table", name, "error", err)
        // Return sanitized error to client
        return nil, rpc.NewSecureError("failed to create table")
    }

    return &CreateResult{Success: true}, nil
}
```

---

## 10. Security Requirements

### 10.1 Credential Handling

**MANDATORY**: All providers MUST use `SecureString` for sensitive data:

```go
type SecureString struct {
    value []byte
}

// NewSecureString creates a new secure string
func NewSecureString(s string) *SecureString {
    return &SecureString{value: []byte(s)}
}

// String returns the value (use sparingly)
func (s *SecureString) String() string {
    return string(s.value)
}

// Clear zeros out the memory
func (s *SecureString) Clear() {
    for i := range s.value {
        s.value[i] = 0
    }
    s.value = nil
}
```

### 10.2 Memory Zeroization

```go
func (p *Provider) Close() error {
    // Clear sensitive configuration
    if p.config != nil {
        if p.config.Password != nil {
            p.config.Password.Clear()
        }
        if p.config.APIKey != nil {
            p.config.APIKey.Clear()
        }
    }

    // Close database connection
    if p.db != nil {
        return p.db.Close()
    }

    return nil
}
```

### 10.3 SQL Injection Prevention

**MANDATORY**: All SQL queries MUST use parameterized statements:

```go
// CORRECT: Parameterized query
func (h *TableHandler) selectByID(ctx context.Context, id int64) (*Table, error) {
    query := "SELECT * FROM tables WHERE id = $1"
    row := h.db.QueryRowContext(ctx, query, id)
    // ...
}

// INCORRECT: String concatenation (FORBIDDEN)
func (h *TableHandler) selectByID(ctx context.Context, id string) (*Table, error) {
    query := "SELECT * FROM tables WHERE id = " + id  // SQL INJECTION VULNERABILITY
    // ...
}
```

### 10.4 Logging Restrictions

**MANDATORY**: Sensitive data MUST NEVER appear in logs:

```go
// CORRECT: Safe logging
func (p *Provider) Configure(ctx context.Context, config map[string]interface{}) error {
    logging.ConfigLogger.Info("Configuring provider",
        "host", config["host"],
        "database", config["database"],
        // password NOT logged
    )
}

// INCORRECT: Logging password (FORBIDDEN)
func (p *Provider) Configure(ctx context.Context, config map[string]interface{}) error {
    logging.ConfigLogger.Info("Configuring provider",
        "host", config["host"],
        "password", config["password"],  // SECURITY VIOLATION
    )
}
```

### 10.5 Zero Simulation Code Policy

**ABSOLUTE PROHIBITION**: No simulation, mock, stub, or fake code in production paths.

Prohibited patterns:
- Functions containing "simulate", "mock", "fake", "stub"
- Hardcoded success responses
- Demo/development mode bypasses
- Incomplete "not implemented" functions

---

## 11. Configuration Standards

### 11.1 Standard Configuration Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `host` | string | Yes* | Database host |
| `port` | int | No | Database port (provider default) |
| `database` | string | Yes* | Database name |
| `username` | string | Yes* | Authentication username |
| `password` | string | Yes* | Authentication password |
| `ssl_mode` | string | No | SSL/TLS mode |
| `connection_pool` | int | No | Connection pool size |
| `query_timeout` | int | No | Query timeout in seconds |
| `safe_destroy` | bool | No | Enable destroy operations |

*Required fields may vary by provider (e.g., SQLite only requires `database`)

### 11.2 Environment Variable Support

Providers SHOULD support environment variable substitution:

```go
func (p *Provider) resolveConfigValue(value interface{}) interface{} {
    s, ok := value.(string)
    if !ok {
        return value
    }

    // Support ${ENV_VAR} syntax
    if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
        envVar := s[2 : len(s)-1]
        return os.Getenv(envVar)
    }

    return value
}
```

### 11.3 Connection Pool Configuration

```go
func (p *Provider) configurePool(db *sql.DB, config *ProviderConfiguration) {
    if config.ConnectionPool > 0 {
        db.SetMaxOpenConns(config.ConnectionPool)
        db.SetMaxIdleConns(config.ConnectionPool / 2)
    } else {
        db.SetMaxOpenConns(10)  // Default
        db.SetMaxIdleConns(5)
    }

    db.SetConnMaxLifetime(30 * time.Minute)
    db.SetConnMaxIdleTime(5 * time.Minute)
}
```

---

## 12. Logging Standards

### 12.1 Logger Types

| Logger | Purpose | Example Usage |
|--------|---------|---------------|
| `ConfigLogger` | Configuration parsing | "Parsing configuration", "Validating fields" |
| `ConnectionLogger` | Database connections | "Connecting to database", "Connection established" |
| `HandlerLogger` | Resource operations | "Creating table", "Updating view" |
| `DiscoveryLogger` | Resource discovery | "Discovering tables", "Found 15 resources" |
| `DispatchLogger` | Request routing | "Routing to CreateResource", "Dispatch completed" |
| `StateLogger` | State management | "Loading state", "Saving state" |

### 12.2 Structured Logging Format

```go
// CORRECT: Structured logging with key-value pairs
logging.HandlerLogger.Info("Creating table",
    "table", tableName,
    "schema", schemaName,
    "columns", len(columns),
)

// INCORRECT: Format strings (AVOID)
logging.HandlerLogger.Info(fmt.Sprintf("Creating table %s in schema %s", tableName, schemaName))
```

### 12.3 Log Levels

| Level | Usage |
|-------|-------|
| Debug | Detailed debugging information |
| Info | Normal operational messages |
| Warn | Warning conditions |
| Error | Error conditions |

### 12.4 Logging Example

```go
func (h *TableHandler) Create(ctx context.Context, config map[string]interface{}) (*CreateResult, error) {
    tableName := config["name"].(string)
    schemaName := config["schema"].(string)

    logging.HandlerLogger.Debug("Starting table creation",
        "table", tableName,
        "schema", schemaName,
    )

    startTime := time.Now()

    if err := h.createTableSQL(ctx, config); err != nil {
        logging.HandlerLogger.Error("Table creation failed",
            "table", tableName,
            "error", err.Error(),
            "duration", time.Since(startTime),
        )
        return nil, err
    }

    logging.HandlerLogger.Info("Table created successfully",
        "table", tableName,
        "schema", schemaName,
        "duration", time.Since(startTime),
    )

    return &CreateResult{Success: true}, nil
}
```

---

## 13. Testing Requirements

### 13.1 Coverage Requirements

| Test Type | Minimum Coverage | Description |
|-----------|------------------|-------------|
| Unit Tests | 80% | Individual function/method testing |
| Integration Tests | Required | Database interaction testing |
| E2E Tests | Required | Full provider lifecycle testing |

### 13.2 Test Organization

```
providers/{name}/
├── tests/
│   ├── unit/
│   │   ├── handler_test.go
│   │   ├── translator_test.go
│   │   └── schema_test.go
│   ├── integration/
│   │   ├── crud_test.go
│   │   └── discovery_test.go
│   └── e2e/
│       ├── lifecycle_test.go
│       └── streaming_test.go
```

### 13.3 Required Test Scenarios

#### Unit Tests
- Configuration parsing and validation
- Schema generation
- SQL translation (for SQL providers)
- Error handling

#### Integration Tests
- CRUD operations for each resource type
- Resource discovery
- State management
- Transaction support (if applicable)

#### E2E Tests
- Full provider lifecycle (configure → plan → apply → destroy)
- Multi-resource dependency ordering
- Streaming handler operations
- Migration operations

### 13.4 Zero Simulation Code

**MANDATORY**: All tests MUST use real database connections:

```go
// CORRECT: Real database testing
func TestTableCreate(t *testing.T) {
    ctx := context.Background()

    // Use real test database
    db, cleanup := setupTestDatabase(t)
    defer cleanup()

    provider := NewProvider()
    err := provider.Configure(ctx, map[string]interface{}{
        "host":     os.Getenv("TEST_DB_HOST"),
        "database": os.Getenv("TEST_DB_NAME"),
        "username": os.Getenv("TEST_DB_USER"),
        "password": os.Getenv("TEST_DB_PASS"),
    })
    require.NoError(t, err)

    // Test real table creation
    result, err := provider.CallFunction(ctx, "CreateResource", createTableRequest)
    assert.NoError(t, err)
    // ...
}

// INCORRECT: Mock/simulation testing (FORBIDDEN)
func TestTableCreate(t *testing.T) {
    mockDB := &MockDatabase{}  // FORBIDDEN
    mockDB.On("Exec", mock.Anything).Return(nil)  // FORBIDDEN
    // ...
}
```

### 13.5 Test Utilities

```go
// setupTestDatabase creates a real test database
func setupTestDatabase(t *testing.T) (*sql.DB, func()) {
    db, err := sql.Open("postgres", os.Getenv("TEST_DATABASE_URL"))
    require.NoError(t, err)

    // Create test schema
    _, err = db.Exec("CREATE SCHEMA IF NOT EXISTS test_schema")
    require.NoError(t, err)

    cleanup := func() {
        db.Exec("DROP SCHEMA IF EXISTS test_schema CASCADE")
        db.Close()
    }

    return db, cleanup
}
```

---

## 14. Provider Categories and Specifics

### 14.1 SQL Relational Databases

**Providers**: PostgreSQL, MySQL, MSSQL, SQLite, CockroachDB

**Requirements**:
- All Tier 1 functions
- All Tier 2 Enterprise functions (DetectDrift, ValidateState, GetAuditLog)
- All Tier 2 Transaction functions
- All Tier 3 Migration functions
- All Streaming functions

**Resource Types** (minimum):
- `schema`, `table`, `view`, `index`, `function`, `trigger`, `sequence`
- `user`, `role`, `grant` (if supported)
- `stream_sink`, `stream_route`, `stream_outbox`

### 14.2 Analytical Warehouses

**Providers**: Snowflake, BigQuery, Redshift, Databricks, DuckDB

**Requirements**:
- All Tier 1 functions
- Tier 2 Enterprise functions (DetectDrift, ValidateState)
- Tier 3 Migration functions (may have limitations)
- All Streaming functions
- Batch operation support

**Resource Types** (minimum):
- `schema`, `table`, `view`, `function` (if supported)
- `warehouse`, `stage` (provider-specific)
- `stream_sink`, `stream_route`, `stream_outbox`

**Notes**:
- Transaction support varies (check provider documentation)
- Migration operations may have limitations due to data volumes

### 14.3 NoSQL Document Databases

**Providers**: MongoDB, DynamoDB

**Requirements**:
- All Tier 1 functions
- Transaction functions (if supported by database version)
- Tier 3 Migration functions (document schema migrations)
- All Streaming functions

**Resource Types**:

**MongoDB**:
- `database`, `collection`, `index`, `validation`
- `replica_set`, `shard` (if applicable)
- `stream_sink`, `stream_route`, `stream_outbox`

**DynamoDB**:
- `table`, `global_secondary_index`, `local_secondary_index`
- `stream`, `backup`, `global_table`
- `stream_sink`, `stream_route`, `stream_outbox`

### 14.4 Time-Series Databases

**Providers**: InfluxDB

**Requirements**:
- All Tier 1 functions
- Tier 3 Migration functions
- All Streaming functions
- NO transaction requirement (exempt)

**Resource Types**:
- `bucket`, `measurement`, `retention_policy`
- `task`, `check`, `notification_endpoint`
- `stream_sink`, `stream_route`, `stream_outbox`

---

## 15. Compliance Checklist

Use this checklist to verify provider compliance with the specification:

### Core Interface
- [ ] `Configure()` method implemented
- [ ] `Schema()` method implemented
- [ ] `CallFunction()` method implemented
- [ ] `Close()` method implemented

### Tier 1 Functions (ALL Required)
- [ ] `CreateResource` routed and implemented
- [ ] `ReadResource` routed and implemented
- [ ] `UpdateResource` routed and implemented
- [ ] `DeleteResource` routed and implemented
- [ ] `DiscoverResources` routed and implemented
- [ ] `Plan` routed and implemented with dependency tree analysis
- [ ] `Apply` routed and implemented with dependency-ordered execution
- [ ] `GetState` routed and implemented
- [ ] `SetState` routed and implemented
- [ ] `Ping` routed and implemented
- [ ] `Version` routed and implemented

### Dependency Tree (All 13 Providers Compliant - December 2025)
- [x] `analyzeProviderDependencies()` implemented
- [x] `PlanResponse` includes `execution_order` field
- [x] `Apply` executes resources in dependency order
- [x] Plan and Apply use identical enterprise safety frameworks

### Tier 2 Functions (Conditional)
- [ ] `DetectDrift` implemented (SQL/Analytical)
- [ ] `ValidateState` implemented (SQL/Analytical)
- [ ] `GetAuditLog` implemented (SQL only)
- [ ] `BeginTransaction` implemented (if DB supports)
- [ ] `CommitTransaction` implemented (if DB supports)
- [ ] `RollbackTransaction` implemented (if DB supports)

### Tier 3 Functions (ALL Required)
- [ ] `PlanMigration` routed and implemented
- [ ] `ApplyMigration` routed and implemented
- [ ] `RollbackMigration` routed and implemented

### Streaming Functions (ALL Required)
- [ ] `stream_sink` handler registered
- [ ] `stream_route` handler registered
- [ ] `stream_outbox` handler registered
- [ ] `OutboxHealthMetrics` implemented
- [ ] `OutboxCleanupRetention` implemented

### Registry Parity
- [ ] All create handlers have matching discover handlers
- [ ] Schema() reflects all registered handlers

### Security
- [ ] SecureString used for credentials
- [ ] Parameterized SQL queries only
- [ ] No sensitive data in logs
- [ ] Memory cleared on Close()
- [ ] No simulation/mock code in production

### Testing
- [ ] Unit test coverage >= 80%
- [ ] Integration tests with real database
- [ ] E2E lifecycle tests
- [ ] No mock/simulation in tests

### Documentation
- [ ] PROVIDER_STATUS.md exists
- [ ] README.md with usage examples
- [ ] Configuration options documented

---

## 16. Version History and Migration Guide

### 16.1 Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | December 2025 | Initial specification release |

### 16.2 Future Considerations

- Additional streaming patterns
- Enhanced migration capabilities
- Cross-provider reference resolution
- Advanced caching strategies

### 16.3 Migration Guide

When updating providers to comply with this specification:

1. **Audit Current Implementation**
   - Run compliance checklist
   - Identify missing functions
   - Document gaps

2. **Implement Missing Functions**
   - Add Tier 1 functions first
   - Add applicable Tier 2 functions
   - Add Tier 3 migration functions
   - Add streaming functions

3. **Verify Registry Parity**
   - Ensure all create handlers have discover counterparts
   - Update Schema() method

4. **Security Audit**
   - Review credential handling
   - Audit SQL queries
   - Check logging statements

5. **Update Tests**
   - Add missing test coverage
   - Remove any mock/simulation code
   - Verify with real database

6. **Documentation**
   - Update PROVIDER_STATUS.md
   - Update README.md
   - Document new capabilities

---

## Appendix A: Quick Reference

### Required Functions by Provider Type

| Function | SQL | Analytical | NoSQL | Time-Series |
|----------|-----|------------|-------|-------------|
| CreateResource | ✅ | ✅ | ✅ | ✅ |
| ReadResource | ✅ | ✅ | ✅ | ✅ |
| UpdateResource | ✅ | ✅ | ✅ | ✅ |
| DeleteResource | ✅ | ✅ | ✅ | ✅ |
| DiscoverResources | ✅ | ✅ | ✅ | ✅ |
| Plan | ✅ | ✅ | ✅ | ✅ |
| Apply | ✅ | ✅ | ✅ | ✅ |
| GetState | ✅ | ✅ | ✅ | ✅ |
| SetState | ✅ | ✅ | ✅ | ✅ |
| Ping | ✅ | ✅ | ✅ | ✅ |
| Version | ✅ | ✅ | ✅ | ✅ |
| DetectDrift | ✅ | ✅ | ❌ | ❌ |
| ValidateState | ✅ | ✅ | ❌ | ❌ |
| GetAuditLog | ✅ | ❌ | ❌ | ❌ |
| BeginTransaction | ✅ | ⚠️ | ⚠️ | ❌ |
| CommitTransaction | ✅ | ⚠️ | ⚠️ | ❌ |
| RollbackTransaction | ✅ | ⚠️ | ⚠️ | ❌ |
| PlanMigration | ✅ | ✅ | ✅ | ✅ |
| ApplyMigration | ✅ | ✅ | ✅ | ✅ |
| RollbackMigration | ✅ | ✅ | ✅ | ✅ |
| OutboxHealthMetrics | ✅ | ✅ | ✅ | ✅ |
| OutboxCleanupRetention | ✅ | ✅ | ✅ | ✅ |

**Legend**: ✅ Required | ⚠️ If Supported | ❌ Not Required

---

**End of Specification**

*This document is the authoritative reference for Kolumn provider development. All providers MUST comply with these requirements.*
