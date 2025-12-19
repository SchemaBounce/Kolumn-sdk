# Database Discovery Interface

The Kolumn Provider SDK includes a comprehensive database discovery interface that allows providers to implement full database introspection capabilities.

## Overview

Database discovery enables providers to:
- Scan entire databases for objects (tables, views, indexes, functions, etc.)
- Gather statistics about database objects (size, row count, last modified)
- Filter discovered objects by schema, type, or other criteria
- Provide comprehensive metadata about database structure

## Architecture

Discovery is implemented through the unified `CallFunction` interface:

```go
// Providers handle discovery through CallFunction
func (p *MyProvider) CallFunction(ctx context.Context, function string, input []byte) ([]byte, error) {
    switch function {
    case "DiscoverDatabase":
        var req core.DiscoveryRequest
        if err := json.Unmarshal(input, &req); err != nil {
            return nil, err
        }
        result, err := p.discoverDatabase(ctx, &req)
        if err != nil {
            return nil, err
        }
        return json.Marshal(result)
    // ... other functions
    }
}
```

## Request Format

### DiscoveryRequest

```go
type DiscoveryRequest struct {
    Schemas       []string // Optional: limit to specific schemas
    ObjectTypes   []string // Optional: limit to specific types (table, view, index, etc.)
    IncludeSystem bool     // Include system objects (pg_*, information_schema, etc.)
    SampleData    bool     // Include sample data for analysis
    MaxObjects    int      // Limit number of objects (0 = no limit)
}
```

### Example Requests

**Discover all user tables:**
```json
{
  "object_types": ["table"],
  "include_system": false,
  "max_objects": 100
}
```

**Discover all objects in specific schemas:**
```json
{
  "schemas": ["public", "analytics"],
  "include_system": false
}
```

**Full database discovery with statistics:**
```json
{
  "include_system": true,
  "sample_data": true,
  "max_objects": 1000
}
```

## Response Format

### DiscoveryResult

```go
type DiscoveryResult struct {
    ProviderType string               // Provider type (postgres, mysql, etc.)
    DatabaseName string               // Database name
    Objects      []DiscoveredObject   // Discovered objects
    Statistics   DiscoveryStatistics  // Summary statistics
    Timestamp    time.Time            // When discovery was performed
    Duration     time.Duration        // How long discovery took
}
```

### DiscoveredObject

```go
type DiscoveredObject struct {
    Type         string                 // "table", "view", "index", "function", etc.
    Schema       string                 // Schema/database name
    Name         string                 // Object name
    FullName     string                 // schema.name
    Definition   map[string]interface{} // Full object definition
    Dependencies []string               // Object dependencies (foreign keys, etc.)
    Statistics   *ObjectStatistics      // Size and usage statistics
    Metadata     map[string]string      // Additional metadata
}
```

### ObjectStatistics

```go
type ObjectStatistics struct {
    RowCount       int64      // Number of rows (for tables)
    SizeBytes      int64      // Object size in bytes
    IndexSizeBytes int64      // Index size in bytes
    LastAnalyzed   *time.Time // When object was last analyzed
    LastModified   *time.Time // When object was last modified
}
```

### DiscoveryStatistics

```go
type DiscoveryStatistics struct {
    TotalObjects   int            // Total objects discovered
    ObjectCounts   map[string]int // Count by object type
    TotalSizeBytes int64          // Total size of all objects
    SchemasCovered []string       // Schemas included in discovery
}
```

## Implementation Guide

### Step 1: Implement Discovery Function

```go
func (p *PostgresProvider) discoverDatabase(ctx context.Context, req *core.DiscoveryRequest) (*core.DiscoveryResult, error) {
    startTime := time.Now()
    helper := core.NewDiscoveryHelper()

    // Gather all objects
    var allObjects []core.DiscoveredObject

    // Discover tables
    tables, err := p.discoverTables(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to discover tables: %w", err)
    }
    allObjects = append(allObjects, tables...)

    // Discover views
    views, err := p.discoverViews(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to discover views: %w", err)
    }
    allObjects = append(allObjects, views...)

    // Discover indexes
    indexes, err := p.discoverIndexes(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to discover indexes: %w", err)
    }
    allObjects = append(allObjects, indexes...)

    // Filter based on request
    filtered := helper.FilterObjects(allObjects, req)

    // Build statistics
    stats := helper.BuildStatistics(filtered)

    return &core.DiscoveryResult{
        ProviderType: "postgres",
        DatabaseName: p.getDatabaseName(),
        Objects:      filtered,
        Statistics:   stats,
        Timestamp:    time.Now(),
        Duration:     time.Since(startTime),
    }, nil
}
```

### Step 2: Implement Object Type Discovery

**PostgreSQL Table Discovery Example:**

```go
func (p *PostgresProvider) discoverTables(ctx context.Context) ([]core.DiscoveredObject, error) {
    query := `
        SELECT
            n.nspname as schema,
            c.relname as name,
            pg_total_relation_size(c.oid) as total_size,
            pg_relation_size(c.oid) as table_size,
            pg_indexes_size(c.oid) as index_size,
            (SELECT count(*) FROM pg_attribute WHERE attrelid = c.oid AND attnum > 0) as column_count,
            obj_description(c.oid, 'pg_class') as description
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE c.relkind = 'r'
        AND n.nspname NOT IN ('pg_catalog', 'information_schema')
        ORDER BY n.nspname, c.relname
    `

    rows, err := p.db.QueryContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var objects []core.DiscoveredObject

    for rows.Next() {
        var (
            schema, name, description sql.NullString
            totalSize, tableSize, indexSize, columnCount sql.NullInt64
        )

        err := rows.Scan(&schema, &name, &totalSize, &tableSize, &indexSize, &columnCount, &description)
        if err != nil {
            return nil, err
        }

        obj := core.DiscoveredObject{
            Type:     "table",
            Schema:   schema.String,
            Name:     name.String,
            FullName: fmt.Sprintf("%s.%s", schema.String, name.String),
            Statistics: &core.ObjectStatistics{
                SizeBytes:      tableSize.Int64,
                IndexSizeBytes: indexSize.Int64,
            },
            Metadata: map[string]string{
                "description":  description.String,
                "column_count": fmt.Sprintf("%d", columnCount.Int64),
            },
        }

        // Get table definition (columns, constraints, etc.)
        definition, err := p.getTableDefinition(ctx, schema.String, name.String)
        if err == nil {
            obj.Definition = definition
        }

        // Get dependencies (foreign keys)
        dependencies, err := p.getTableDependencies(ctx, schema.String, name.String)
        if err == nil {
            obj.Dependencies = dependencies
        }

        objects = append(objects, obj)
    }

    return objects, rows.Err()
}
```

**MySQL Table Discovery Example:**

```go
func (p *MySQLProvider) discoverTables(ctx context.Context) ([]core.DiscoveredObject, error) {
    query := `
        SELECT
            TABLE_SCHEMA,
            TABLE_NAME,
            TABLE_ROWS,
            DATA_LENGTH + INDEX_LENGTH as total_size,
            DATA_LENGTH as data_size,
            INDEX_LENGTH as index_size,
            CREATE_TIME,
            UPDATE_TIME,
            TABLE_COMMENT
        FROM information_schema.TABLES
        WHERE TABLE_SCHEMA NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
        AND TABLE_TYPE = 'BASE TABLE'
        ORDER BY TABLE_SCHEMA, TABLE_NAME
    `

    rows, err := p.db.QueryContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var objects []core.DiscoveredObject

    for rows.Next() {
        var (
            schema, name, comment sql.NullString
            rowCount, totalSize, dataSize, indexSize sql.NullInt64
            createTime, updateTime sql.NullTime
        )

        err := rows.Scan(&schema, &name, &rowCount, &totalSize, &dataSize, &indexSize, &createTime, &updateTime, &comment)
        if err != nil {
            return nil, err
        }

        obj := core.DiscoveredObject{
            Type:     "table",
            Schema:   schema.String,
            Name:     name.String,
            FullName: fmt.Sprintf("%s.%s", schema.String, name.String),
            Statistics: &core.ObjectStatistics{
                RowCount:       rowCount.Int64,
                SizeBytes:      dataSize.Int64,
                IndexSizeBytes: indexSize.Int64,
                LastModified:   &updateTime.Time,
            },
            Metadata: map[string]string{
                "comment":     comment.String,
                "created_at":  createTime.Time.Format(time.RFC3339),
            },
        }

        objects = append(objects, obj)
    }

    return objects, rows.Err()
}
```

### Step 3: Use Discovery Helper

The SDK provides a `DiscoveryHelper` with utility functions:

```go
helper := core.NewDiscoveryHelper()

// Filter objects based on request parameters
filtered := helper.FilterObjects(allObjects, req)

// Build statistics summary
stats := helper.BuildStatistics(filtered)
```

## Best Practices

### 1. Performance Optimization

```go
// Use concurrent discovery for large databases
var wg sync.WaitGroup
var mu sync.Mutex
var allObjects []core.DiscoveredObject

objectTypes := []string{"table", "view", "index", "function"}

for _, objType := range objectTypes {
    wg.Add(1)
    go func(ot string) {
        defer wg.Done()
        objects, err := p.discoverByType(ctx, ot)
        if err != nil {
            log.Printf("Failed to discover %s: %v", ot, err)
            return
        }
        mu.Lock()
        allObjects = append(allObjects, objects...)
        mu.Unlock()
    }(objType)
}

wg.Wait()
```

### 2. Error Handling

```go
// Continue discovery even if individual object types fail
tables, err := p.discoverTables(ctx)
if err != nil {
    log.Printf("Warning: failed to discover tables: %v", err)
    // Continue with other object types
}

views, err := p.discoverViews(ctx)
if err != nil {
    log.Printf("Warning: failed to discover views: %v", err)
}
```

### 3. Resource Limits

```go
// Respect MaxObjects limit
if req.MaxObjects > 0 && len(allObjects) >= req.MaxObjects {
    break
}

// Use database query limits
query := fmt.Sprintf(`
    SELECT * FROM information_schema.tables
    LIMIT %d
`, req.MaxObjects)
```

### 4. Metadata Enrichment

```go
// Add rich metadata to discovered objects
obj.Metadata = map[string]string{
    "owner":           owner,
    "tablespace":      tablespace,
    "row_security":    fmt.Sprintf("%t", hasRowSecurity),
    "has_triggers":    fmt.Sprintf("%t", hasTriggers),
    "partition_key":   partitionKey,
    "compression":     compression,
}
```

## Testing Discovery Implementation

```go
func TestDiscoverDatabase(t *testing.T) {
    provider := setupTestProvider(t)

    req := &core.DiscoveryRequest{
        ObjectTypes:   []string{"table", "view"},
        IncludeSystem: false,
        MaxObjects:    100,
    }

    result, err := provider.discoverDatabase(context.Background(), req)
    require.NoError(t, err)
    require.NotNil(t, result)

    // Verify result structure
    assert.NotEmpty(t, result.ProviderType)
    assert.NotEmpty(t, result.DatabaseName)
    assert.NotEmpty(t, result.Objects)

    // Verify statistics
    assert.Equal(t, len(result.Objects), result.Statistics.TotalObjects)
    assert.NotEmpty(t, result.Statistics.ObjectCounts)

    // Verify object structure
    for _, obj := range result.Objects {
        assert.NotEmpty(t, obj.Type)
        assert.NotEmpty(t, obj.Schema)
        assert.NotEmpty(t, obj.Name)
        assert.Equal(t, fmt.Sprintf("%s.%s", obj.Schema, obj.Name), obj.FullName)
    }
}
```

## Security Considerations

The SDK includes built-in security validations:

1. **Request Size Limits**: MaxObjects is capped at 10,000 objects
2. **Schema Name Validation**: Schema names are validated for length and format
3. **Object Type Validation**: Object types are validated against allowed types
4. **SQL Injection Protection**: Always use parameterized queries

```go
// Good: Parameterized query
query := "SELECT * FROM information_schema.tables WHERE table_schema = ?"
rows, err := db.QueryContext(ctx, query, schemaName)

// Bad: String concatenation (SQL injection risk)
query := fmt.Sprintf("SELECT * FROM information_schema.tables WHERE table_schema = '%s'", schemaName)
```

## Integration with Kolumn Core

Discovery results are automatically integrated with:

- **Schema Catalog**: Discovered objects populate the schema catalog
- **Drift Detection**: Discovery results enable drift detection
- **State Management**: Discovery informs state file generation
- **AI Agents**: Discovery data feeds AI-powered optimization

## Complete Example

See [`examples/postgres-discovery/`](../examples/postgres-discovery/) for a complete working example of database discovery implementation.
