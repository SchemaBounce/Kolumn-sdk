// Package main demonstrates minimal usage of the Kolumn Provider SDK
//
// This example shows the basic patterns for implementing a provider using the SDK.
// It demonstrates the create/discover object categorization and progressive interface disclosure.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/schemabounce/kolumn/sdk/core"
	"github.com/schemabounce/kolumn/sdk/create"
	"github.com/schemabounce/kolumn/sdk/discover"
	"github.com/schemabounce/kolumn/sdk/helpers/security"
)

// SimpleProvider demonstrates the minimal Provider interface
type SimpleProvider struct {
	configured bool
	config     core.Config

	// Registries for object types
	createRegistry   *create.Registry
	discoverRegistry *discover.Registry
}

// NewSimpleProvider creates a new example provider
func NewSimpleProvider() *SimpleProvider {
	provider := &SimpleProvider{
		createRegistry:   create.NewRegistry(),
		discoverRegistry: discover.NewRegistry(),
	}

	// Register CREATE object types (things we can create and manage)
	provider.registerCreateObjects()

	// Register DISCOVER object types (existing infrastructure we can find)
	provider.registerDiscoverObjects()

	return provider
}

// Configure implements the core.Provider interface with secure logging
func (p *SimpleProvider) Configure(ctx context.Context, config core.Config) error {
	// SECURITY: Validate configuration size before processing
	if simpleConfig, ok := config.(*SimpleConfig); ok {
		validator := &security.InputSizeValidator{}
		if err := validator.ValidateConfigSize(simpleConfig.data); err != nil {
			secErr := security.NewSecureError(
				"configuration too large",
				fmt.Sprintf("config validation failed: %v", err),
				"CONFIG_TOO_LARGE",
			)
			return secErr
		}
	}

	// Validate required configuration
	endpoint, err := config.GetString("endpoint")
	if err != nil {
		secErr := security.NewSecureError(
			"missing required configuration",
			fmt.Sprintf("endpoint validation failed: %v", err),
			"MISSING_ENDPOINT",
		)
		return secErr
	}

	// SECURITY: Sanitize endpoint for logging (remove credentials if present)
	sanitizedEndpoint := sanitizeEndpoint(endpoint)
	log.Printf("Configuring provider with endpoint: %s", sanitizedEndpoint)

	p.config = config
	p.configured = true
	return nil
}

// sanitizeEndpoint removes credentials from endpoint URLs for safe logging
func sanitizeEndpoint(endpoint string) string {
	// Simple sanitization - in production you'd use proper URL parsing
	if len(endpoint) > 50 {
		return endpoint[:20] + "..." + endpoint[len(endpoint)-10:]
	}
	return endpoint
}

// Schema implements the core.Provider interface
func (p *SimpleProvider) Schema() (*core.Schema, error) {
	return &core.Schema{
		Name:        "simple",
		Version:     "1.0.0",
		Description: "A simple provider demonstrating SDK usage patterns",

		// CREATE objects - resources this provider can create and manage
		CreateObjects: map[string]*core.ObjectType{
			"table": {
				Name:        "table",
				Description: "Database table that can be created and managed",
				Type:        core.CREATE,
				Category:    "database",
				Properties: map[string]*core.Property{
					"name": {
						Type:        "string",
						Description: "Name of the table",
					},
					"columns": {
						Type:        "list",
						Description: "List of column definitions",
					},
				},
				Required: []string{"name"},
				Examples: []*core.ObjectExample{
					{
						Name:        "basic_table",
						Description: "Basic table creation",
						Config: map[string]interface{}{
							"name":    "users",
							"columns": []string{"id", "name", "email"},
						},
					},
				},
			},
		},

		// DISCOVER objects - existing infrastructure this provider can find
		DiscoverObjects: map[string]*core.ObjectType{
			"existing_tables": {
				Name:        "existing_tables",
				Description: "Discover existing database tables",
				Type:        core.DISCOVER,
				Category:    "database",
				Properties: map[string]*core.Property{
					"schema": {
						Type:        "string",
						Description: "Database schema to scan",
					},
					"pattern": {
						Type:        "string",
						Description: "Table name pattern to match",
					},
				},
				Examples: []*core.ObjectExample{
					{
						Name:        "scan_all_tables",
						Description: "Discover all tables in public schema",
						Config: map[string]interface{}{
							"schema":  "public",
							"pattern": "*",
						},
					},
				},
			},
		},

		// Configuration schema
		ConfigSchema: &core.ConfigSchema{
			Properties: map[string]*core.Property{
				"endpoint": {
					Type:        "string",
					Description: "Database connection endpoint",
				},
				"timeout": {
					Type:        "integer",
					Description: "Connection timeout in seconds",
				},
			},
			Required: []string{"endpoint"},
		},
	}, nil
}

// CallFunction implements the core.Provider interface with comprehensive security
func (p *SimpleProvider) CallFunction(ctx context.Context, function string, input []byte) ([]byte, error) {
	if !p.configured {
		secErr := security.NewSecureError(
			"provider not ready",
			"provider not configured",
			"NOT_CONFIGURED",
		)
		return nil, secErr
	}

	// SECURITY: Validate function name
	if function == "" {
		secErr := security.NewSecureError(
			"invalid request",
			"function name is empty",
			"EMPTY_FUNCTION",
		)
		return nil, secErr
	}

	// Parse function call to determine object type and operation with secure unmarshaling
	var request struct {
		ObjectType string          `json:"object_type"`
		Operation  string          `json:"operation"`
		Data       json.RawMessage `json:"data"`
	}

	// SECURITY: Use safe unmarshaling with size limits
	if err := security.SafeUnmarshal(input, &request); err != nil {
		secErr := security.NewSecureError(
			"invalid request format",
			fmt.Sprintf("function call unmarshal failed: %v", err),
			"INVALID_REQUEST",
		)
		return nil, secErr
	}

	// SECURITY: Validate object type and operation
	if err := security.ValidateObjectType(request.ObjectType); err != nil {
		secErr := security.NewSecureError(
			"invalid object type",
			fmt.Sprintf("object type validation failed: %v", err),
			"INVALID_OBJECT_TYPE",
		)
		return nil, secErr
	}

	if err := security.ValidateMethod(request.Operation); err != nil {
		secErr := security.NewSecureError(
			"operation not supported",
			fmt.Sprintf("operation validation failed: %v", err),
			"INVALID_OPERATION",
		)
		return nil, secErr
	}

	// Route to appropriate handler based on object type with secure error handling
	switch request.Operation {
	case "create", "read", "update", "delete", "plan":
		// CREATE object operations
		result, err := p.createRegistry.CallHandler(ctx, request.ObjectType, request.Operation, request.Data)
		if err != nil {
			// Error is already a SecureError from the registry
			return nil, err
		}
		return result, nil

	case "scan", "analyze", "query":
		// DISCOVER object operations
		result, err := p.discoverRegistry.CallHandler(ctx, request.ObjectType, request.Operation, request.Data)
		if err != nil {
			// Error is already a SecureError from the registry
			return nil, err
		}
		return result, nil

	default:
		// This should never be reached due to validation above
		secErr := security.NewSecureError(
			"operation not supported",
			fmt.Sprintf("unexpected operation: %s", request.Operation),
			"UNEXPECTED_OPERATION",
		)
		return nil, secErr
	}
}

// Close implements the core.Provider interface
func (p *SimpleProvider) Close() error {
	log.Println("Closing simple provider")
	return nil
}

// registerCreateObjects demonstrates registering CREATE object handlers
func (p *SimpleProvider) registerCreateObjects() {
	// Create a simple table handler
	tableHandler := &SimpleTableHandler{}

	// Define the schema for this object type
	tableSchema := &core.ObjectType{
		Name:        "table",
		Description: "Database table",
		Type:        core.CREATE,
		Category:    "database",
	}

	// Register the handler with the schema
	err := p.createRegistry.RegisterHandler("table", tableHandler, tableSchema)
	if err != nil {
		log.Printf("Failed to register table handler: %v", err)
	}
}

// registerDiscoverObjects demonstrates registering DISCOVER object handlers
func (p *SimpleProvider) registerDiscoverObjects() {
	// Create a table discovery handler
	tableDiscoverer := &SimpleTableDiscoverer{}

	// Define the schema for this discovery type
	discoverySchema := &core.ObjectType{
		Name:        "existing_tables",
		Description: "Discover existing database tables",
		Type:        core.DISCOVER,
		Category:    "database",
	}

	// Register the discoverer with the schema
	err := p.discoverRegistry.RegisterHandler("existing_tables", tableDiscoverer, discoverySchema)
	if err != nil {
		log.Printf("Failed to register table discoverer: %v", err)
	}
}

// SimpleTableHandler demonstrates implementing CREATE object operations
type SimpleTableHandler struct{}

func (h *SimpleTableHandler) Create(ctx context.Context, req *create.CreateRequest) (*create.CreateResponse, error) {
	log.Printf("Creating table: %s", req.Name)

	return &create.CreateResponse{
		State: map[string]interface{}{
			"id":     fmt.Sprintf("table_%s", req.Name),
			"name":   req.Config["name"],
			"status": "created",
		},
	}, nil
}

func (h *SimpleTableHandler) Read(ctx context.Context, req *create.ReadRequest) (*create.ReadResponse, error) {
	log.Printf("Reading table: %s", req.Name)

	return &create.ReadResponse{
		State: map[string]interface{}{
			"name":   "example_table",
			"status": "active",
		},
	}, nil
}

func (h *SimpleTableHandler) Update(ctx context.Context, req *create.UpdateRequest) (*create.UpdateResponse, error) {
	log.Printf("Updating table: %s", req.Name)

	return &create.UpdateResponse{
		NewState: map[string]interface{}{
			"name":   req.Config["name"],
			"status": "updated",
		},
	}, nil
}

func (h *SimpleTableHandler) Delete(ctx context.Context, req *create.DeleteRequest) (*create.DeleteResponse, error) {
	log.Printf("Deleting table: %s", req.Name)

	return &create.DeleteResponse{
		Success: true,
	}, nil
}

func (h *SimpleTableHandler) Plan(ctx context.Context, req *create.PlanRequest) (*create.PlanResponse, error) {
	log.Printf("Planning changes for table: %s", req.Name)

	return &create.PlanResponse{
		Changes: []create.PlannedChange{
			{
				Action:          "create",
				Property:        "name",
				NewValue:        req.DesiredConfig["name"],
				RequiresReplace: false,
				RiskLevel:       "low",
				Description:     "Create new table",
			},
		},
		Valid: true,
		Summary: &core.PlanSummary{
			TotalChanges:    1,
			ByAction:        map[string]int{"create": 1},
			RequiresReplace: false,
			RiskLevel:       "low",
		},
	}, nil
}

// SimpleTableDiscoverer demonstrates implementing DISCOVER object operations
type SimpleTableDiscoverer struct{}

func (d *SimpleTableDiscoverer) Scan(ctx context.Context, req *discover.ScanRequest) (*discover.ScanResponse, error) {
	log.Printf("Scanning for existing tables")

	// Mock discovery results
	objects := []*discover.DiscoveredObject{
		{
			ID:       "existing_table_1",
			Name:     "users",
			Type:     "table",
			Category: "database",
			Properties: map[string]interface{}{
				"row_count":    1000,
				"column_count": 5,
				"schema":       "public",
			},
			Source: &discover.Source{
				System:   "postgresql",
				Location: "public.users",
			},
		},
		{
			ID:       "existing_table_2",
			Name:     "orders",
			Type:     "table",
			Category: "database",
			Properties: map[string]interface{}{
				"row_count":    5000,
				"column_count": 8,
				"schema":       "public",
			},
			Source: &discover.Source{
				System:   "postgresql",
				Location: "public.orders",
			},
		},
	}

	return &discover.ScanResponse{
		Objects: objects,
		Summary: &discover.ScanSummary{
			TotalObjects: len(objects),
			ObjectTypes:  map[string]int{"table": len(objects)},
			Duration:     "1.2s",
		},
	}, nil
}

func (d *SimpleTableDiscoverer) Analyze(ctx context.Context, req *discover.AnalyzeRequest) (*discover.AnalyzeResponse, error) {
	log.Printf("Analyzing %d objects", len(req.Objects))

	results := make([]*discover.AnalysisResult, len(req.Objects))
	for i, obj := range req.Objects {
		results[i] = &discover.AnalysisResult{
			Object: obj,
			Analysis: map[string]interface{}{
				"performance_score": 85,
				"optimization_opportunities": []string{
					"Consider adding index on frequently queried columns",
					"Table could benefit from partitioning",
				},
			},
		}
	}

	return &discover.AnalyzeResponse{
		Results: results,
		Insights: []*discover.Insight{
			{
				Type:        "performance",
				Title:       "Index Optimization Opportunity",
				Description: "Several tables are missing indexes on frequently queried columns",
				Impact:      "medium",
				Confidence:  0.8,
			},
		},
	}, nil
}

func (d *SimpleTableDiscoverer) Query(ctx context.Context, req *discover.QueryRequest) (*discover.QueryResponse, error) {
	log.Printf("Querying for objects matching: %s", req.Query)

	// Mock search results
	objects := []*discover.DiscoveredObject{
		{
			ID:       "query_result_1",
			Name:     "large_table",
			Type:     "table",
			Category: "database",
			Properties: map[string]interface{}{
				"row_count":   100000,
				"matches":     true,
				"query_field": req.Query,
			},
		},
	}

	return &discover.QueryResponse{
		Objects:       objects,
		TotalCount:    1,
		ExecutionTime: "0.5s",
	}, nil
}

// main demonstrates how to use the SDK to create and serve a provider
func main() {
	// Create a provider using the SDK
	provider := NewSimpleProvider()

	// Example: Configure the provider
	config := NewSimpleConfig(map[string]interface{}{
		"endpoint": "postgresql://localhost:5432/mydb",
		"timeout":  30,
	})

	ctx := context.Background()
	if err := provider.Configure(ctx, config); err != nil {
		log.Fatalf("Failed to configure provider: %v", err)
	}

	// Example: Get provider schema
	schema, err := provider.Schema()
	if err != nil {
		log.Fatalf("Failed to get schema: %v", err)
	}

	log.Printf("Provider: %s v%s", schema.Name, schema.Version)
	log.Printf("CREATE objects: %d", len(schema.CreateObjects))
	log.Printf("DISCOVER objects: %d", len(schema.DiscoverObjects))

	// In a real implementation, you would serve this provider via RPC
	// For this example, we just demonstrate the patterns
	log.Println("Provider example completed successfully!")
}

// SimpleConfig implements the core.Config interface for this example
type SimpleConfig struct {
	data map[string]interface{}
}

func NewSimpleConfig(data map[string]interface{}) *SimpleConfig {
	return &SimpleConfig{data: data}
}

func (c *SimpleConfig) Get(key string) (interface{}, bool) {
	value, exists := c.data[key]
	return value, exists
}

func (c *SimpleConfig) GetString(key string) (string, error) {
	value, exists := c.data[key]
	if !exists {
		return "", fmt.Errorf("key %s not found", key)
	}

	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("key %s is not a string", key)
	}

	return str, nil
}

func (c *SimpleConfig) GetInt(key string) (int, error) {
	value, exists := c.data[key]
	if !exists {
		return 0, fmt.Errorf("key %s not found", key)
	}

	switch v := value.(type) {
	case int:
		return v, nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("key %s is not a number", key)
	}
}

func (c *SimpleConfig) GetBool(key string) (bool, error) {
	value, exists := c.data[key]
	if !exists {
		return false, fmt.Errorf("key %s not found", key)
	}

	b, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("key %s is not a boolean", key)
	}

	return b, nil
}

func (c *SimpleConfig) Set(key string, value interface{}) {
	if c.data == nil {
		c.data = make(map[string]interface{})
	}
	c.data[key] = value
}

func (c *SimpleConfig) Keys() []string {
	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}

func (c *SimpleConfig) Validate() error {
	// Simple validation - ensure required keys exist
	requiredKeys := []string{"endpoint"}
	for _, key := range requiredKeys {
		if _, exists := c.data[key]; !exists {
			return fmt.Errorf("required key %s is missing", key)
		}
	}
	return nil
}
