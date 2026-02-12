# Kolumn Provider Logging SDK

This package provides a universal logging standard for all Kolumn providers, transforming JSON-heavy logs into human-readable format that matches core Kolumn's logging style.

## Overview

The logging SDK addresses the common issue where providers output verbose JSON logs that are difficult for humans to read. This package provides:

- **Human-readable logs by default** - Clear, structured messages for operations
- **JSON logs only in debug mode** - Full request/response data when debugging
- **Component-specific loggers** - Pre-configured loggers for different provider components
- **Core Kolumn compatibility** - Same logging format and patterns as core Kolumn
- **Environment variable configuration** - Easy setup for development and production

## Quick Start

### Basic Usage

```go
package main

import (
    "github.com/schemabounce/kolumn/sdk/helpers/logging"
)

func main() {
    // Use pre-configured loggers
    logging.ProviderLogger.Info("Provider initialized successfully")
    logging.ConnectionLogger.Info("Connected to database at %s", endpoint)
    logging.HandlerLogger.Info("Registered %d resource handlers", count)

    // Debug information (only shown when DEBUG=1)
    logging.ProviderLogger.Debug("Configuration details: %+v", config)
}
```

### Provider Integration

Replace existing log statements with the appropriate component logger:

**Before:**
```go
log.Printf("Configuring provider with endpoint: %s", endpoint)
log.Printf("Creating table: %s", req.Name)
fmt.Printf("Request data: %+v", req)
```

**After:**
```go
logging.ConfigLogger.Info("Configuring provider with endpoint: %s", sanitizedEndpoint)
logging.HandlerLogger.Info("Creating table: %s", req.Name)
logging.HandlerLogger.JSONDebug("Create table request", req)
```

## Pre-configured Component Loggers

The SDK provides these pre-configured loggers for common provider components:

- **`ProviderLogger`** - Main provider operations (Configure, Schema, Close)
- **`ConnectionLogger`** - Database/service connections and authentication
- **`HandlerLogger`** - Resource handler operations (Create, Read, Update, Delete)
- **`ValidationLogger`** - Schema validation and configuration validation
- **`SecurityLogger`** - Security-related operations and access control
- **`StateLogger`** - State management and persistence operations
- **`DiscoveryLogger`** - Resource discovery and scanning operations
- **`ConfigLogger`** - Configuration parsing and validation
- **`RegistryLogger`** - Handler registration and management
- **`DispatchLogger`** - Function dispatch and routing
- **`SchemaLogger`** - Schema generation and documentation

## Log Levels and Output

### Normal Mode (Default)
Human-readable logs with operation context:

```
[KOLUMN-INFO] PROVIDER          │ Starting CreateResource operation on table 'users'
[KOLUMN-INFO] CONNECTION        │ Successfully connected to postgres://user:***@localhost:5432/db
[KOLUMN-INFO] HANDLER           │ Completed CreateResource operation on table 'users' in 245ms
[KOLUMN-WARNING] VALIDATION       │ Schema validation warnings for table: 2 warnings
[KOLUMN-ERROR] HANDLER           │ Failed CreateResource operation on table 'users': connection failed
```

### Debug Mode (DEBUG=1)
Includes detailed JSON data and debug information:

```
[KOLUMN-INFO] PROVIDER          │ Starting CreateResource operation on table 'users'
[KOLUMN-DEBUG] HANDLER           │ CreateResource request: {name: "users", schema: "public", columns: [...]}
[KOLUMN-DEBUG] VALIDATION        │ Schema validation passed for table
[KOLUMN-INFO] HANDLER           │ Completed CreateResource operation on table 'users' in 245ms
```

## Configuration

### Environment Variables

```bash
# Enable debug logging globally
DEBUG=1

# Enable debug for specific components
DEBUG_COMPONENTS=provider,handler,connection

# Enable debug for all provider components
DEBUG_PROVIDER=1
```

### Programmatic Configuration

```go
import "github.com/schemabounce/kolumn/sdk/helpers/logging"

// Enable debug globally
logging.Configure(&logging.Configuration{
    EnableDebug: true,
})

// Configure specific components
logging.Configure(&logging.Configuration{
    DefaultLevel: logging.LevelInfo,
    ComponentLevels: map[string]logging.Level{
        "provider":   logging.LevelDebug,
        "handler":    logging.LevelDebug,
        "connection": logging.LevelInfo,
    },
})

// Quick helpers
logging.EnableDebug()                           // Enable debug globally
logging.EnableComponentDebug("provider")        // Enable debug for provider component
logging.SetLogLevel("handler", logging.LevelWarn) // Set specific level
```

## Best Practices

### 1. Use Appropriate Component Loggers

```go
// ✅ Good - Use specific component logger
logging.ConnectionLogger.Info("Connected to %s", endpoint)
logging.HandlerLogger.Info("Creating resource: %s", name)
logging.ValidationLogger.Warn("Validation warning: %s", message)

// ❌ Bad - Use generic logging
log.Printf("Connected to %s", endpoint)
fmt.Printf("Creating resource: %s", name)
```

### 2. Use Helper Functions for Common Patterns

```go
// Log requests and responses with automatic JSON handling
logging.LogRequest(logging.HandlerLogger, "CreateResource", request)
logging.LogResponse(logging.HandlerLogger, "CreateResource", response, err)

// Log operations with timing
err := logging.LogProviderOperation(logging.HandlerLogger, context, func() error {
    return createResource(request)
})

// Log connection attempts with endpoint sanitization
logging.LogConnectionAttempt(logging.ConnectionLogger, connectionString, err)
```

### 3. Use Structured Logging for Rich Context

```go
// ✅ Good - Structured logging with fields
logging.HandlerLogger.InfoWithFields("Resource created",
    "type", "table",
    "name", resourceName,
    "schema", schemaName,
    "duration", duration.String(),
)

// ✅ Good - Debug with JSON data
logging.HandlerLogger.JSONDebug("Full request details", request)

// ❌ Bad - Verbose string formatting
logging.HandlerLogger.Info("Resource created: type=table name=%s schema=%s duration=%v",
    resourceName, schemaName, duration)
```

### 4. Sanitize Sensitive Information

```go
// ✅ Good - Sanitize credentials in connection strings
sanitized := logging.SanitizeEndpoint(connectionString)
logging.ConnectionLogger.Info("Connecting to %s", sanitized)

// ✅ Good - Automatic sensitive field redaction in structured logging
logging.ConfigLogger.InfoWithFields("Configuration loaded",
    "host", config.Host,
    "password", config.Password, // Automatically redacted
    "database", config.Database,
)
```

### 5. Use Context for Operation Tracking

```go
// Create operation context
context := logging.ProviderContext{
    ProviderName: "postgres",
    Operation:    "CreateResource",
    ResourceType: "table",
    ResourceName: request.Name,
    StartTime:    time.Now(),
}

// Log operation with automatic timing and error handling
err := logging.LogProviderOperation(logging.HandlerLogger, context, func() error {
    return p.createTable(request)
})
```

## Migration Guide

### Step 1: Replace Existing Log Statements

**Old pattern:**
```go
import "log"

func (p *Provider) Create(req *CreateRequest) (*CreateResponse, error) {
    log.Printf("Creating resource: %s", req.Name)
    // ... implementation
    log.Printf("Created successfully")
    return response, nil
}
```

**New pattern:**
```go
import "github.com/schemabounce/kolumn/sdk/helpers/logging"

func (p *Provider) Create(req *CreateRequest) (*CreateResponse, error) {
    logging.HandlerLogger.Info("Creating resource: %s", req.Name)
    // ... implementation
    logging.HandlerLogger.Info("Created successfully")
    return response, nil
}
```

### Step 2: Add Structured Request/Response Logging

```go
func (p *Provider) Create(req *CreateRequest) (*CreateResponse, error) {
    // Log request (JSON only in debug mode)
    logging.LogRequest(logging.HandlerLogger, "CreateResource", req)

    response, err := p.createResource(req)

    // Log response with error handling
    logging.LogResponse(logging.HandlerLogger, "CreateResource", response, err)

    return response, err
}
```

### Step 3: Add Operation Context and Timing

```go
func (p *Provider) Create(req *CreateRequest) (*CreateResponse, error) {
    context := logging.ProviderContext{
        ProviderName: p.name,
        Operation:    "CreateResource",
        ResourceType: req.ResourceType,
        ResourceName: req.Name,
        StartTime:    time.Now(),
    }

    var response *CreateResponse
    err := logging.LogProviderOperation(logging.HandlerLogger, context, func() error {
        var err error
        response, err = p.createResource(req)
        return err
    })

    return response, err
}
```

## Testing

The logging package includes comprehensive testing utilities:

```go
import (
    "testing"
    "github.com/schemabounce/kolumn/sdk/helpers/logging"
)

func TestProviderLogging(t *testing.T) {
    // Create test logger with capture
    logger, capture := logging.NewTestLogger(t, "provider", true)

    // Test logging
    logger.Info("test message")
    logger.Debug("debug message")

    // Assert log output
    capture.AssertContains(t, "test message")
    capture.AssertLevel(t, logging.LevelInfo, "provider")

    // Test debug mode
    logging.TestDebugLevel(t, logger, capture, true)
}
```

## Example Output

### Production Mode
```
[KOLUMN-INFO] PROVIDER          │ PostgreSQL provider v1.2.3 initializing
[KOLUMN-INFO] CONFIG            │ Loading configuration from environment
[KOLUMN-INFO] CONNECTION        │ Successfully connected to postgres://user:***@db.example.com:5432/production
[KOLUMN-INFO] REGISTRY          │ Registered 5 resource handlers: table, view, function, index, trigger
[KOLUMN-INFO] HANDLER           │ Starting CreateResource operation on table 'users'
[KOLUMN-INFO] VALIDATION        │ Schema validation passed for table
[KOLUMN-INFO] HANDLER           │ Completed CreateResource operation on table 'users' in 245ms
```

### Debug Mode (DEBUG=1)
```
[KOLUMN-INFO] PROVIDER          │ PostgreSQL provider v1.2.3 initializing
[KOLUMN-DEBUG] CONFIG            │ Configuration details: {host: "db.example.com", port: 5432, database: "production", username: "user", password: "<redacted>"}
[KOLUMN-INFO] CONNECTION        │ Successfully connected to postgres://user:***@db.example.com:5432/production
[KOLUMN-DEBUG] CONNECTION        │ Connection pool initialized: max_connections=20 idle_timeout=5m
[KOLUMN-INFO] REGISTRY          │ Registered 5 resource handlers: table, view, function, index, trigger
[KOLUMN-INFO] HANDLER           │ Starting CreateResource operation on table 'users'
[KOLUMN-DEBUG] HANDLER           │ CreateResource request: {name: "users", schema: "public", columns: [{name: "id", type: "bigserial", primary_key: true}, {name: "email", type: "varchar(255)", unique: true}]}
[KOLUMN-DEBUG] VALIDATION        │ Validating table schema: name=users columns=2 constraints=2
[KOLUMN-INFO] VALIDATION        │ Schema validation passed for table
[KOLUMN-DEBUG] HANDLER           │ Executing SQL: CREATE TABLE public.users (id BIGSERIAL PRIMARY KEY, email VARCHAR(255) UNIQUE NOT NULL)
[KOLUMN-DEBUG] HANDLER           │ CreateResource response: {success: true, id: "table_users_123", metadata: {...}}
[KOLUMN-INFO] HANDLER           │ Completed CreateResource operation on table 'users' in 245ms
```

## Performance Considerations

- **Lazy evaluation**: Debug messages are only formatted when debug mode is enabled
- **Structured logging**: Key-value pairs are only processed when the log level is enabled
- **JSON handling**: Large JSON objects are only marshaled in debug mode
- **Sanitization**: Credential sanitization is cached for repeated connection strings

## Plan Resource Evaluation Patterns

When implementing the `Plan` function in providers, follow these patterns to ensure high-quality, actionable CLI output. The goal is **brief, actionable logs**: one line per unchanged item and SQL-focused output for changes.

### Required Functions

Every provider should implement these helper functions for plan evaluation:

#### 1. `stripTemplateContext` - Remove Template Pollution

The `_template_context` field often pollutes config snapshots. Strip it before processing:

```go
// stripTemplateContext removes the _template_context field from config maps
// to prevent template metadata from polluting config snapshots and logs.
func stripTemplateContext(config map[string]interface{}) map[string]interface{} {
    result := make(map[string]interface{}, len(config))
    for k, v := range config {
        if k == "_template_context" {
            continue
        }
        result[k] = v
    }
    return result
}
```

#### 2. `checkInsertRowExists` - NOOP Detection for Inserts

For SQL databases, check if insert data already exists to avoid false CREATE reports:

```go
// checkInsertRowExists checks if a row already exists in the database
// based on unique key columns to enable accurate NOOP detection.
func (p *Provider) checkInsertRowExists(ctx context.Context, tableName string, values map[string]interface{}, uniqueKeys []string) (bool, error) {
    if len(uniqueKeys) == 0 {
        return false, nil
    }

    // Build WHERE clause from unique keys
    var conditions []string
    var args []interface{}
    for i, key := range uniqueKeys {
        if val, ok := values[key]; ok {
            conditions = append(conditions, fmt.Sprintf("%s = $%d", key, i+1))
            args = append(args, val)
        }
    }

    if len(conditions) == 0 {
        return false, nil
    }

    query := fmt.Sprintf("SELECT 1 FROM %s WHERE %s LIMIT 1",
        tableName, strings.Join(conditions, " AND "))

    var exists int
    err := p.db.QueryRowContext(ctx, query, args...).Scan(&exists)
    if err == sql.ErrNoRows {
        return false, nil
    }
    if err != nil {
        return false, err
    }
    return true, nil
}
```

**NoSQL Variants:**

- **MongoDB**: `checkInsertDocumentExists` - Uses `_id` or all fields for matching
- **DynamoDB**: `checkInsertItemExists` - Uses partition/sort key for GetItem
- **InfluxDB**: Skip existence check (append-only time-series)

#### 3. `evaluate[Provider]ResourceForPlan` - Resource Summary Generation

Generate concise resource summaries for CLI logging:

```go
// evaluatePostgresResourceForPlan evaluates a resource and returns a summary
// with NOOP detection for existing data.
func (p *Provider) evaluatePostgresResourceForPlan(ctx context.Context, resource rpc.PlanResource) map[string]interface{} {
    summary := map[string]interface{}{
        "resource_type": resource.ResourceType,
        "name":          resource.Name,
        "action":        resource.Action,
    }

    // Strip template context from config
    cleanConfig := stripTemplateContext(resource.Config)
    summary["config_snapshot"] = cleanConfig

    // NOOP detection for insert resources
    if strings.HasSuffix(resource.ResourceType, "_insert") && resource.Action == "create" {
        tableName, _ := cleanConfig["table"].(string)
        values, _ := cleanConfig["values"].(map[string]interface{})
        uniqueKeys, _ := cleanConfig["unique_keys"].([]string)

        if tableName != "" && len(values) > 0 {
            exists, err := p.checkInsertRowExists(ctx, tableName, values, uniqueKeys)
            if err == nil && exists {
                summary["action"] = "noop"
                summary["reason"] = "row already exists"
            }
        }
    }

    return summary
}
```

### handlePlan Implementation

The `handlePlan` function should build resource summaries for CLI logging:

```go
func (p *Provider) handlePlan(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
    var req struct {
        Resources []struct {
            ResourceType string                 `json:"resource_type"`
            Name         string                 `json:"name"`
            Config       map[string]interface{} `json:"config"`
            Action       string                 `json:"action"`
        } `json:"resources"`
    }

    if err := json.Unmarshal(input, &req); err != nil {
        return nil, fmt.Errorf("failed to parse plan request: %w", err)
    }

    // Build resource summaries for CLI logging
    resourceSummaries := make([]map[string]interface{}, 0, len(req.Resources))
    for _, resource := range req.Resources {
        planResource := rpc.PlanResource{
            ResourceType: resource.ResourceType,
            Name:         resource.Name,
            Config:       resource.Config,
            Action:       resource.Action,
        }
        summary := p.evaluatePostgresResourceForPlan(ctx, planResource)
        resourceSummaries = append(resourceSummaries, summary)
    }

    response := map[string]interface{}{
        "success":            true,
        "resource_count":     len(req.Resources),
        "resource_summaries": resourceSummaries,
    }

    return json.Marshal(response)
}
```

### Expected CLI Output

With these patterns, CLI output transforms from verbose JSON to concise, actionable logs:

**Before (verbose):**
```
Planning postgres_insert.seed_roles...
  Action: CREATE
  Config: {"table":"roles","values":{"id":"admin","name":"Administrator","permissions":["read","write","delete"],"created_at":"2024-01-01T00:00:00Z","_template_context":{"source":"seeds/roles.csv","line":1}}}
Planning postgres_insert.seed_users...
  Action: CREATE
  Config: {"table":"users","values":{"id":1,"email":"admin@example.com","role_id":"admin","_template_context":{"source":"seeds/users.csv","line":1}}}
```

**After (actionable):**
```
  postgres_insert.seed_roles: NOOP (row already exists)
  postgres_insert.seed_users: NOOP (row already exists)
  postgres_table.audit_log: CREATE
    + CREATE TABLE audit_log (id BIGSERIAL PRIMARY KEY, ...)
```

### Provider-Specific Considerations

| Provider | Existence Check | Unique Key Source |
|----------|-----------------|-------------------|
| PostgreSQL | `SELECT 1 ... LIMIT 1` | `unique_keys` config or primary key |
| MySQL | `SELECT 1 ... LIMIT 1` | `unique_keys` config or primary key |
| SQLite | `SELECT 1 ... LIMIT 1` | `unique_keys` config or primary key |
| MSSQL | `SELECT TOP 1 1 ...` | `unique_keys` config or primary key |
| Snowflake | `SELECT 1 ... LIMIT 1` | `unique_keys` config or primary key |
| Redshift | `SELECT 1 ... LIMIT 1` | `unique_keys` config or primary key |
| BigQuery | `SELECT 1 ... LIMIT 1` | `unique_keys` config or primary key |
| MongoDB | `FindOne` with `_id` or doc match | `_id` field or full document |
| DynamoDB | `GetItem` with key | Partition key + sort key |
| InfluxDB | N/A (append-only) | Time-series, no existence check |
| DuckDB | `SELECT 1 ... LIMIT 1` | `unique_keys` config or primary key |
| Databricks | `SELECT 1 ... LIMIT 1` | `unique_keys` config or primary key |

### Integration with CallFunction

Add the `Plan` case to your provider's `CallFunction` dispatch:

```go
func (p *Provider) CallFunction(ctx context.Context, function string, input json.RawMessage) (json.RawMessage, error) {
    switch function {
    case "Plan":
        return p.handlePlan(ctx, input)
    case "CreateResource":
        return p.handleCreateResource(ctx, input)
    // ... other cases
    }
}
```

## Integration with Core Kolumn

This SDK logging package is designed to be compatible with core Kolumn's logging system:

- **Same log format**: `[LEVEL][COMPONENT] message key=value`
- **Same environment variables**: `DEBUG`, `DEBUG_COMPONENTS`
- **Same component naming**: Consistent with core component loggers
- **Same structured patterns**: Key-value pairs and field logging

This ensures that provider logs seamlessly integrate with core Kolumn logs in production environments.
