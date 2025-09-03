# Kolumn Provider SDK

**Professional SDK for building Kolumn providers** - The complete toolkit for developing production-ready Kolumn data stack providers.

## Overview

The Kolumn Provider SDK enables developers to build external providers that integrate seamlessly with Kolumn's infrastructure-as-code platform for the modern data stack. The SDK provides:

- **RPC Plugin Framework** - HashiCorp go-plugin based architecture
- **Universal Provider Interface** - Simple 4-method interface for all provider types  
- **Shared Type System** - Universal types for state, metadata, and schemas
- **Provider Development Kit** - Helpers, validation, and testing utilities

## Quick Start

### 1. Install SDK

```bash
go mod init my-kolumn-provider
go get github.com/schemabounce/kolumn/sdk
```

### 2. Implement Provider Interface

```go
package main

import (
    "context"
    "encoding/json"
    
    "github.com/schemabounce/kolumn/sdk/rpc"
    "github.com/schemabounce/kolumn/sdk/types"
)

type MyProvider struct {
    configured bool
}

// Implement the UniversalProvider interface
func (p *MyProvider) Configure(ctx context.Context, config map[string]interface{}) error {
    // Parse and validate configuration
    p.configured = true
    return nil
}

func (p *MyProvider) GetSchema() (*types.ProviderSchema, error) {
    return &types.ProviderSchema{
        Provider: types.ProviderSpec{
            Name:    "myprovider",
            Version: "1.0.0",
        },
        Functions: map[string]types.FunctionSpec{
            "ping": {
                Description: "Health check",
            },
        },
    }, nil
}

func (p *MyProvider) CallFunction(ctx context.Context, function string, input json.RawMessage) (json.RawMessage, error) {
    switch function {
    case "ping":
        return json.Marshal(map[string]string{"status": "ok"})
    default:
        return nil, fmt.Errorf("unsupported function: %s", function)
    }
}

func (p *MyProvider) Close() error {
    return nil
}

// Serve the provider as RPC plugin
func main() {
    provider := &MyProvider{}
    
    rpc.ServeProvider(&rpc.ServeConfig{
        Provider: provider,
    })
}
```

### 3. Build Plugin Binary

```bash
go build -o kolumn-provider-myprovider
```

### 4. Test with Kolumn

```hcl
provider "myprovider" {
  # configuration
}
```

## Architecture

### Universal Provider Interface

All providers implement the same 4-method interface:

```go
type UniversalProvider interface {
    Configure(ctx context.Context, config map[string]interface{}) error
    GetSchema() (*types.ProviderSchema, error)
    CallFunction(ctx context.Context, function string, input json.RawMessage) (json.RawMessage, error)
    Close() error
}
```

### Function-Based Dispatch

Instead of resource-specific methods, providers declare supported functions:

```go
// Database functions
"create_table", "drop_table", "insert_data", "query_data"

// Storage functions  
"create_bucket", "upload_object", "download_object"

// Streaming functions
"create_topic", "produce_message", "consume_messages"

// Universal functions (all providers)
"ping", "get_version", "health_check", "get_metrics"
```

## SDK Packages

### `rpc` - RPC Framework
- `UniversalProvider` interface definition
- Plugin serving and client infrastructure
- Protocol definitions

### `types` - Shared Types
- `ProviderSchema` - Provider capability declaration
- `UniversalState` - Cross-provider state format
- `UniversalMetadata` - Metadata type definitions


### `metadata` - Metadata System
- Collector interfaces for metadata extraction
- Schema and lineage type definitions

### `pdk` - Provider Development Kit
- Helper functions and utilities
- Validation and testing tools
- Code generation templates

## Provider Categories

The SDK supports providers across all data stack categories:

- **Database** (11 supported): PostgreSQL, MySQL, SQLite, MongoDB, etc.
- **ETL/ELT** (4 supported): Airbyte, dbt, Fivetran, Spark
- **Streaming** (3 supported): Kafka, Kinesis, Pulsar
- **Orchestration** (4 supported): Airflow, Dagster, Prefect, Temporal
- **Storage** (5 supported): S3, Azure Blob, GCS, Delta Lake, Iceberg
- **Cache** (2 supported): Redis, Elasticsearch
- **Quality** (1 supported): Great Expectations

## Examples

See the `examples/` directory for complete provider implementations:

- `simple_provider.go` - Minimal provider template
- `database_provider.go` - Database provider example
- `storage_provider.go` - Object storage provider example

## Development

### Building the SDK

```bash
cd sdk
make build
```

### Running Tests

```bash
cd sdk  
make test
```

### Generating Documentation

```bash
cd sdk
make docs
```

## Versioning

The SDK uses semantic versioning:

- **v0.1.0** - Initial beta release
- **v1.x.x** - Backward compatible changes
- **v2.x.x** - Breaking changes (rare)

## Support

- **Documentation**: Full API docs and guides
- **Examples**: Working provider implementations
- **Community**: Provider development discussions

The Kolumn Provider SDK enables the ecosystem of external provider development while maintaining compatibility and type safety across the entire data stack.