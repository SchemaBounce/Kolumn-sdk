// Package examples demonstrates a simple provider implementation using the Kolumn SDK
package examples

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/schemabounce/kolumn/sdk/pdk"
	"github.com/schemabounce/kolumn/sdk/rpc"
	"github.com/schemabounce/kolumn/sdk/types"
)

// SimpleProvider demonstrates a minimal provider implementation
type SimpleProvider struct {
	*pdk.BaseProvider
	config    *SimpleConfig
	connected bool
	metrics   *Metrics
}

// SimpleConfig represents the provider configuration
type SimpleConfig struct {
	Host     string `json:"host" validate:"required"`
	Port     int    `json:"port" validate:"min=1,max=65535"`
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	Database string `json:"database"`
	SSL      bool   `json:"ssl"`
	Timeout  string `json:"timeout"`
}

// Metrics tracks provider metrics
type Metrics struct {
	RequestCount    int64     `json:"request_count"`
	ErrorCount      int64     `json:"error_count"`
	LastRequest     time.Time `json:"last_request"`
	ConnectedSince  time.Time `json:"connected_since"`
	ConnectionCount int64     `json:"connection_count"`
}

// NewSimpleProvider creates a new instance of the simple provider
func NewSimpleProvider() *SimpleProvider {
	provider := &SimpleProvider{
		BaseProvider: pdk.NewBaseProvider("simple", "1.0.0"),
		metrics:      &Metrics{},
	}

	// Register resource handlers would go here
	// provider.RegisterResourceHandler("simple_database", &SimpleDatabaseHandler{provider: provider})

	return provider
}

// Configure implements the UniversalProvider interface
func (p *SimpleProvider) Configure(ctx context.Context, config map[string]interface{}) error {
	// Call base configuration first
	if err := p.BaseProvider.Configure(ctx, config); err != nil {
		return err
	}

	// Validate required fields
	required := []string{"host", "username", "password"}
	if err := pdk.ValidateRequired(config, required); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Parse configuration
	p.config = &SimpleConfig{}
	if err := pdk.ParseConfig(config, p.config); err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Set defaults
	if p.config.Port == 0 {
		p.config.Port = 5432 // Default PostgreSQL port
	}
	if p.config.Database == "" {
		p.config.Database = "postgres"
	}
	if p.config.Timeout == "" {
		p.config.Timeout = "30s"
	}

	// Validate entity names if provided
	if p.config.Database != "" {
		if err := pdk.ValidateEntityName(p.config.Database); err != nil {
			return fmt.Errorf("invalid database name: %w", err)
		}
	}

	// Simulate connection (in a real provider, you would connect to the actual service)
	time.Sleep(100 * time.Millisecond) // Simulate connection time

	p.connected = true
	p.metrics.ConnectedSince = time.Now()
	p.metrics.ConnectionCount++

	return nil
}

// GetSchema implements the UniversalProvider interface
func (p *SimpleProvider) GetSchema() (*types.ProviderSchema, error) {
	schema := pdk.CreateProviderSchema("simple", "0.1.0")
	schema.Provider.Description = "A simple example provider for demonstration"
	schema.Provider.Source = "https://github.com/schemabounce/kolumn/examples"

	// Add universal functions (ping, get_version, etc.)
	pdk.AddUniversalFunctions(schema)

	// Add provider-specific functions
	pdk.AddFunction(schema, "create_table", "Create a new table", false)
	pdk.AddFunction(schema, "list_tables", "List all tables", true)
	pdk.AddFunction(schema, "describe_table", "Get table schema", true)
	pdk.AddFunction(schema, "drop_table", "Drop a table", false)
	pdk.AddFunction(schema, "insert_data", "Insert data into table", false)
	pdk.AddFunction(schema, "query_data", "Query data from table", true)

	// Add resource types
	schema.ResourceTypes = []string{"table", "index", "view"}

	// Add capabilities
	schema.Capabilities = []string{"transactions", "ssl", "backup", "replication"}

	return schema, nil
}

// CallFunction implements the UniversalProvider interface
func (p *SimpleProvider) CallFunction(ctx context.Context, function string, input json.RawMessage) (json.RawMessage, error) {
	p.metrics.RequestCount++
	p.metrics.LastRequest = time.Now()

	if !p.connected {
		p.metrics.ErrorCount++
		return pdk.CreateErrorResponse("provider not configured")
	}

	switch function {
	// Universal functions
	case "ping":
		return p.ping(ctx)
	case "get_version":
		return p.getVersion(ctx)
	case "health_check":
		return p.healthCheck(ctx)
	case "get_metrics":
		return p.getMetrics(ctx)
	case "validate_config":
		return p.validateConfig(ctx)
	case "get_capabilities":
		return p.getCapabilities(ctx)

	// Provider-specific functions
	case "create_table":
		return p.createTable(ctx, input)
	case "list_tables":
		return p.listTables(ctx)
	case "describe_table":
		return p.describeTable(ctx, input)
	case "drop_table":
		return p.dropTable(ctx, input)
	case "insert_data":
		return p.insertData(ctx, input)
	case "query_data":
		return p.queryData(ctx, input)

	default:
		p.metrics.ErrorCount++
		return pdk.CreateErrorResponse(fmt.Sprintf("unsupported function: %s", function))
	}
}

// Close implements the UniversalProvider interface
func (p *SimpleProvider) Close() error {
	if p.connected {
		// Simulate cleanup (in a real provider, you would close connections)
		time.Sleep(50 * time.Millisecond)
		p.connected = false
	}
	return nil
}

// Universal function implementations

func (p *SimpleProvider) ping(ctx context.Context) (json.RawMessage, error) {
	start := time.Now()

	// Simulate ping (in a real provider, you would ping the actual service)
	time.Sleep(10 * time.Millisecond)

	latency := time.Since(start).Milliseconds()
	status := "ok"
	if !p.connected {
		status = "disconnected"
	}

	metadata := map[string]interface{}{
		"host":     p.config.Host,
		"port":     p.config.Port,
		"database": p.config.Database,
		"ssl":      p.config.SSL,
	}

	return pdk.CreatePingResponse(status, latency, metadata)
}

func (p *SimpleProvider) getVersion(ctx context.Context) (json.RawMessage, error) {
	version := map[string]interface{}{
		"provider_version": "0.1.0",
		"sdk_version":      "0.1.0",
		"api_version":      "v1",
		"go_version":       "1.24",
		"build_date":       "2024-01-01T00:00:00Z",
	}
	return pdk.CreateSuccessResponse(version)
}

func (p *SimpleProvider) healthCheck(ctx context.Context) (json.RawMessage, error) {
	health := map[string]interface{}{
		"status":     "healthy",
		"connected":  p.connected,
		"uptime":     time.Since(p.metrics.ConnectedSince).String(),
		"requests":   p.metrics.RequestCount,
		"errors":     p.metrics.ErrorCount,
		"error_rate": float64(p.metrics.ErrorCount) / float64(p.metrics.RequestCount),
		"last_check": time.Now().UTC().Format(time.RFC3339),
	}

	if !p.connected {
		health["status"] = "unhealthy"
	}

	return pdk.CreateSuccessResponse(health)
}

func (p *SimpleProvider) getMetrics(ctx context.Context) (json.RawMessage, error) {
	return pdk.CreateSuccessResponse(p.metrics)
}

func (p *SimpleProvider) validateConfig(ctx context.Context) (json.RawMessage, error) {
	valid := true
	var issues []string

	if p.config == nil {
		valid = false
		issues = append(issues, "configuration not provided")
	} else {
		if p.config.Host == "" {
			valid = false
			issues = append(issues, "host is required")
		}
		if p.config.Username == "" {
			valid = false
			issues = append(issues, "username is required")
		}
		if p.config.Password == "" {
			valid = false
			issues = append(issues, "password is required")
		}
	}

	result := map[string]interface{}{
		"valid":  valid,
		"issues": issues,
	}

	return pdk.CreateSuccessResponse(result)
}

func (p *SimpleProvider) getCapabilities(ctx context.Context) (json.RawMessage, error) {
	capabilities := map[string]interface{}{
		"transactions": true,
		"ssl":          true,
		"backup":       true,
		"replication":  false,
		"clustering":   false,
		"partitioning": false,
	}
	return pdk.CreateSuccessResponse(capabilities)
}

// Provider-specific function implementations (simplified for demo)

func (p *SimpleProvider) createTable(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var req struct {
		Name    string `json:"name"`
		Columns []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"columns"`
	}

	if err := json.Unmarshal(input, &req); err != nil {
		return pdk.CreateErrorResponse("invalid input format", err.Error())
	}

	if err := pdk.ValidateEntityName(req.Name); err != nil {
		return pdk.CreateErrorResponse("invalid table name", err.Error())
	}

	// Simulate table creation
	time.Sleep(100 * time.Millisecond)

	result := map[string]interface{}{
		"table":   req.Name,
		"created": true,
		"columns": len(req.Columns),
	}

	return pdk.CreateSuccessResponse(result)
}

func (p *SimpleProvider) listTables(ctx context.Context) (json.RawMessage, error) {
	// Simulate fetching tables
	time.Sleep(50 * time.Millisecond)

	tables := []string{"users", "products", "orders", "customers"}
	return pdk.CreateSuccessResponse(map[string]interface{}{
		"tables": tables,
		"count":  len(tables),
	})
}

func (p *SimpleProvider) describeTable(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var req struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal(input, &req); err != nil {
		return pdk.CreateErrorResponse("invalid input format", err.Error())
	}

	// Simulate table description
	time.Sleep(50 * time.Millisecond)

	schema := map[string]interface{}{
		"table": req.Name,
		"columns": []map[string]interface{}{
			{"name": "id", "type": "INTEGER", "primary_key": true},
			{"name": "name", "type": "VARCHAR(255)", "nullable": false},
			{"name": "created_at", "type": "TIMESTAMP", "default": "CURRENT_TIMESTAMP"},
		},
		"indexes": []map[string]interface{}{
			{"name": "idx_name", "columns": []string{"name"}},
		},
	}

	return pdk.CreateSuccessResponse(schema)
}

func (p *SimpleProvider) dropTable(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var req struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal(input, &req); err != nil {
		return pdk.CreateErrorResponse("invalid input format", err.Error())
	}

	// Simulate table drop
	time.Sleep(100 * time.Millisecond)

	result := map[string]interface{}{
		"table":   req.Name,
		"dropped": true,
	}

	return pdk.CreateSuccessResponse(result)
}

func (p *SimpleProvider) insertData(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var req struct {
		Table string                 `json:"table"`
		Data  map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(input, &req); err != nil {
		return pdk.CreateErrorResponse("invalid input format", err.Error())
	}

	// Simulate data insertion
	time.Sleep(75 * time.Millisecond)

	result := map[string]interface{}{
		"table":    req.Table,
		"inserted": true,
		"rows":     1,
	}

	return pdk.CreateSuccessResponse(result)
}

func (p *SimpleProvider) queryData(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var req struct {
		Table string `json:"table"`
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}

	if err := json.Unmarshal(input, &req); err != nil {
		return pdk.CreateErrorResponse("invalid input format", err.Error())
	}

	// Simulate query execution
	time.Sleep(200 * time.Millisecond)

	// Mock data
	rows := []map[string]interface{}{
		{"id": 1, "name": "John Doe", "created_at": "2024-01-01T00:00:00Z"},
		{"id": 2, "name": "Jane Smith", "created_at": "2024-01-02T00:00:00Z"},
	}

	result := map[string]interface{}{
		"table": req.Table,
		"rows":  rows,
		"count": len(rows),
	}

	return pdk.CreateSuccessResponse(result)
}

// RunSimpleProvider - serves the provider as an RPC plugin
func RunSimpleProvider() {
	provider := NewSimpleProvider()

	rpc.ServeProvider(&rpc.ServeConfig{
		Provider: provider,
		Debug:    false,
	})
}

// Ensure SimpleProvider implements UniversalProvider
var _ rpc.UniversalProvider = (*SimpleProvider)(nil)
