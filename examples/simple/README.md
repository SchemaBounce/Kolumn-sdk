# Simple Provider Example

This example demonstrates the basic patterns for implementing a provider using the Kolumn Provider SDK.

## Key Concepts Demonstrated

### 1. **Simple Provider Interface**
```go
// Simple 4-method Provider interface
type Provider interface {
    Configure(ctx context.Context, config Config) error
    Schema() (*Schema, error)
    CallFunction(ctx context.Context, function string, input []byte) ([]byte, error)
    Close() error
}
```

### 2. **Create/Discover Object Categorization**
```go
// CREATE objects - resources the provider can create and manage
CreateObjects: map[string]*core.ObjectType{
    "table": {
        Type: core.CREATE,
        Description: "Database table that can be created and managed",
        // ...
    },
}

// DISCOVER objects - existing infrastructure the provider can find
DiscoverObjects: map[string]*core.ObjectType{
    "existing_tables": {
        Type: core.DISCOVER, 
        Description: "Discover existing database tables",
        // ...
    },
}
```

### 3. **Handler Registration Pattern**
```go
// Register CREATE handlers
createRegistry := create.NewRegistry()
createRegistry.RegisterHandler("table", tableHandler, tableSchema)

// Register DISCOVER handlers  
discoverRegistry := discover.NewRegistry()
discoverRegistry.RegisterHandler("existing_tables", discoverer, discoverySchema)
```

### 4. **Operation Routing**
```go
switch operation {
case "create", "read", "update", "delete", "plan":
    // Route to CREATE object handlers
    return p.createRegistry.CallHandler(ctx, objectType, operation, data)

case "scan", "analyze", "query": 
    // Route to DISCOVER object handlers
    return p.discoverRegistry.CallHandler(ctx, objectType, operation, data)
}
```

## Usage

### Running the Example
```bash
cd examples/simple
go run provider.go
```

### Expected Output
```
Configuring provider with endpoint: postgresql://localhost:5432/mydb
Provider: simple v1.0.0
CREATE objects: 1
DISCOVER objects: 1
Provider example completed successfully!
```

## Implementation Pattern

1. **Create Provider Struct**: Implement the core `Provider` interface
2. **Register Object Types**: Use registries to organize CREATE vs DISCOVER objects
3. **Implement Handlers**: Create handlers for each object type
4. **Route Operations**: Use the CallFunction method to route operations to appropriate handlers

## Next Steps

- Implement your own provider using these patterns
- The Schema() method provides all information needed for documentation

This example keeps things minimal while demonstrating all the core SDK concepts.