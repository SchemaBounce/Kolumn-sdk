// Package core provides database discovery interfaces and helpers
package core

import "context"

// DatabaseDiscoverer is an optional interface that database providers can implement
// to support full database introspection via the DiscoverDatabase function.
//
// Providers that implement this interface should register a handler in their
// CallFunction implementation to handle the "DiscoverDatabase" function name.
//
// Example implementation:
//
//	func (p *PostgresProvider) CallFunction(ctx context.Context, function string, input []byte) ([]byte, error) {
//	    switch function {
//	    case "DiscoverDatabase":
//	        var req core.DiscoveryRequest
//	        if err := json.Unmarshal(input, &req); err != nil {
//	            return nil, err
//	        }
//	        result, err := p.DiscoverDatabase(ctx, &req)
//	        if err != nil {
//	            return nil, err
//	        }
//	        return json.Marshal(result)
//	    // ... handle other functions
//	    }
//	}
//
//	func (p *PostgresProvider) DiscoverDatabase(ctx context.Context, req *DiscoveryRequest) (*DiscoveryResult, error) {
//	    // Query information_schema or system catalogs
//	    // Build DiscoveredObject instances for each database object
//	    // Return DiscoveryResult with statistics
//	}
type DatabaseDiscoverer interface {
	// DiscoverDatabase performs comprehensive database introspection
	DiscoverDatabase(ctx context.Context, req *DiscoveryRequest) (*DiscoveryResult, error)
}

// DiscoveryHelper provides utility functions for implementing database discovery
type DiscoveryHelper struct{}

// NewDiscoveryHelper creates a new discovery helper
func NewDiscoveryHelper() *DiscoveryHelper {
	return &DiscoveryHelper{}
}

// FilterObjects filters discovered objects based on request parameters
func (h *DiscoveryHelper) FilterObjects(objects []DiscoveredObject, req *DiscoveryRequest) []DiscoveredObject {
	filtered := make([]DiscoveredObject, 0, len(objects))

	for _, obj := range objects {
		// Filter by object types if specified
		if len(req.ObjectTypes) > 0 {
			found := false
			for _, objType := range req.ObjectTypes {
				if obj.Type == objType {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter by schemas if specified
		if len(req.Schemas) > 0 {
			found := false
			for _, schema := range req.Schemas {
				if obj.Schema == schema {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter system objects if not included
		if !req.IncludeSystem && h.isSystemObject(obj) {
			continue
		}

		filtered = append(filtered, obj)

		// Apply max objects limit
		if req.MaxObjects > 0 && len(filtered) >= req.MaxObjects {
			break
		}
	}

	return filtered
}

// isSystemObject checks if an object is a system object
func (h *DiscoveryHelper) isSystemObject(obj DiscoveredObject) bool {
	// Common system schema names across databases
	systemSchemas := []string{
		"information_schema",
		"pg_catalog",
		"mysql",
		"performance_schema",
		"sys",
		"INFORMATION_SCHEMA",
		"PERFORMANCE_SCHEMA",
	}

	for _, sysSchema := range systemSchemas {
		if obj.Schema == sysSchema {
			return true
		}
	}

	// Check for system object prefixes
	systemPrefixes := []string{
		"pg_",
		"sys_",
		"__",
	}

	for _, prefix := range systemPrefixes {
		if len(obj.Name) > len(prefix) && obj.Name[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

// BuildStatistics calculates discovery statistics from a list of objects
func (h *DiscoveryHelper) BuildStatistics(objects []DiscoveredObject) DiscoveryStatistics {
	stats := DiscoveryStatistics{
		TotalObjects:   len(objects),
		ObjectCounts:   make(map[string]int),
		TotalSizeBytes: 0,
		SchemasCovered: make([]string, 0),
	}

	schemaMap := make(map[string]bool)

	for _, obj := range objects {
		// Count by type
		stats.ObjectCounts[obj.Type]++

		// Sum size
		if obj.Statistics != nil {
			stats.TotalSizeBytes += obj.Statistics.SizeBytes
			if obj.Statistics.IndexSizeBytes > 0 {
				stats.TotalSizeBytes += obj.Statistics.IndexSizeBytes
			}
		}

		// Track unique schemas
		if _, exists := schemaMap[obj.Schema]; !exists {
			schemaMap[obj.Schema] = true
			stats.SchemasCovered = append(stats.SchemasCovered, obj.Schema)
		}
	}

	return stats
}

// Example usage documentation
const discoveryExampleUsage = `
Example: Implementing DiscoverDatabase in a PostgreSQL provider

func (p *PostgresProvider) DiscoverDatabase(ctx context.Context, req *core.DiscoveryRequest) (*core.DiscoveryResult, error) {
    startTime := time.Now()
    helper := core.NewDiscoveryHelper()

    // Query database for all objects
    objects := []core.DiscoveredObject{}

    // Discover tables
    tables, err := p.discoverTables(ctx)
    if err != nil {
        return nil, err
    }
    objects = append(objects, tables...)

    // Discover views
    views, err := p.discoverViews(ctx)
    if err != nil {
        return nil, err
    }
    objects = append(objects, views...)

    // Discover indexes
    indexes, err := p.discoverIndexes(ctx)
    if err != nil {
        return nil, err
    }
    objects = append(objects, indexes...)

    // Filter objects based on request
    filtered := helper.FilterObjects(objects, req)

    // Build statistics
    stats := helper.BuildStatistics(filtered)

    return &core.DiscoveryResult{
        ProviderType: "postgres",
        DatabaseName: p.config["database"].(string),
        Objects:      filtered,
        Statistics:   stats,
        Timestamp:    time.Now(),
        Duration:     time.Since(startTime),
    }, nil
}

func (p *PostgresProvider) discoverTables(ctx context.Context) ([]core.DiscoveredObject, error) {
    query := ` + "`" + `
        SELECT
            schemaname,
            tablename,
            pg_total_relation_size(schemaname || '.' || tablename) as size_bytes,
            pg_stat_get_tuples_fetched(c.oid) as row_count
        FROM pg_tables pt
        JOIN pg_class c ON c.relname = pt.tablename
        WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
    ` + "`" + `

    rows, err := p.db.QueryContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    objects := []core.DiscoveredObject{}
    for rows.Next() {
        var schema, name string
        var sizeBytes, rowCount int64

        if err := rows.Scan(&schema, &name, &sizeBytes, &rowCount); err != nil {
            return nil, err
        }

        obj := core.DiscoveredObject{
            Type:     "table",
            Schema:   schema,
            Name:     name,
            FullName: schema + "." + name,
            Statistics: &core.ObjectStatistics{
                SizeBytes: sizeBytes,
                RowCount:  rowCount,
            },
        }

        objects = append(objects, obj)
    }

    return objects, nil
}
`
