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
[INFO][provider] Starting CreateResource operation on table 'users'
[INFO][connection] Successfully connected to postgres://user:***@localhost:5432/db
[INFO][handler] Completed CreateResource operation on table 'users' in 245ms
[WARN][validation] Schema validation warnings for table: 2 warnings
[ERROR][handler] Failed CreateResource operation on table 'users': connection failed
```

### Debug Mode (DEBUG=1)
Includes detailed JSON data and debug information:

```
[INFO][provider] Starting CreateResource operation on table 'users'
[DEBUG][handler] CreateResource request: {name: "users", schema: "public", columns: [...]}
[DEBUG][validation] Schema validation passed for table
[INFO][handler] Completed CreateResource operation on table 'users' in 245ms
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
[INFO][provider] PostgreSQL provider v1.2.3 initializing
[INFO][config] Loading configuration from environment
[INFO][connection] Successfully connected to postgres://user:***@db.example.com:5432/production
[INFO][registry] Registered 5 resource handlers: table, view, function, index, trigger
[INFO][handler] Starting CreateResource operation on table 'users'
[INFO][validation] Schema validation passed for table
[INFO][handler] Completed CreateResource operation on table 'users' in 245ms
```

### Debug Mode (DEBUG=1)
```
[INFO][provider] PostgreSQL provider v1.2.3 initializing
[DEBUG][config] Configuration details: {host: "db.example.com", port: 5432, database: "production", username: "user", password: "<redacted>"}
[INFO][connection] Successfully connected to postgres://user:***@db.example.com:5432/production
[DEBUG][connection] Connection pool initialized: max_connections=20 idle_timeout=5m
[INFO][registry] Registered 5 resource handlers: table, view, function, index, trigger
[INFO][handler] Starting CreateResource operation on table 'users'
[DEBUG][handler] CreateResource request: {name: "users", schema: "public", columns: [{name: "id", type: "bigserial", primary_key: true}, {name: "email", type: "varchar(255)", unique: true}]}
[DEBUG][validation] Validating table schema: name=users columns=2 constraints=2
[INFO][validation] Schema validation passed for table
[DEBUG][handler] Executing SQL: CREATE TABLE public.users (id BIGSERIAL PRIMARY KEY, email VARCHAR(255) UNIQUE NOT NULL)
[DEBUG][handler] CreateResource response: {success: true, id: "table_users_123", metadata: {...}}
[INFO][handler] Completed CreateResource operation on table 'users' in 245ms
```

## Performance Considerations

- **Lazy evaluation**: Debug messages are only formatted when debug mode is enabled
- **Structured logging**: Key-value pairs are only processed when the log level is enabled
- **JSON handling**: Large JSON objects are only marshaled in debug mode
- **Sanitization**: Credential sanitization is cached for repeated connection strings

## Integration with Core Kolumn

This SDK logging package is designed to be compatible with core Kolumn's logging system:

- **Same log format**: `[LEVEL][COMPONENT] message key=value`
- **Same environment variables**: `DEBUG`, `DEBUG_COMPONENTS`
- **Same component naming**: Consistent with core component loggers
- **Same structured patterns**: Key-value pairs and field logging

This ensures that provider logs seamlessly integrate with core Kolumn logs in production environments.