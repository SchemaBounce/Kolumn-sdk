package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/schemabounce/kolumn-sdk/testing"
)

// ExampleProvider demonstrates how to implement a Kolumn provider
// with proper schema consistency testing
type ExampleProvider struct {
	endpoint string
	apiKey   string
	timeout  string
}

// NewExampleProvider creates a new example provider instance
func NewExampleProvider() *ExampleProvider {
	return &ExampleProvider{}
}

// Ensure ExampleProvider implements the testing.SchemaProvider interface
var _ testing.SchemaProvider = (*ExampleProvider)(nil)

// Schema returns the provider's schema definition
func (p *ExampleProvider) Schema() (*testing.ProviderSchema, error) {
	return &testing.ProviderSchema{
		Name:     "example",
		Version:  "1.0.0",
		Protocol: "rpc",
		Type:     "database",
		SupportedFunctions: []string{
			"create_table", "read_table", "update_table", "delete_table",
			"create_user", "read_user", "update_user", "delete_user",
		},
		ResourceTypes: []testing.ResourceTypeDefinition{
			{
				Name:        "example_table",
				Description: "Database table resource",
				Operations:  []string{"create", "read", "update", "delete"},
				ConfigSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"name": {"type": "string", "description": "Table name"},
						"schema": {"type": "string", "description": "Database schema"},
						"columns": {"type": "array", "description": "Table columns"}
					},
					"required": ["name"]
				}`),
				StateSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"id": {"type": "string", "description": "Resource ID"},
						"name": {"type": "string", "description": "Table name"},
						"schema": {"type": "string", "description": "Database schema"},
						"columns": {"type": "array", "description": "Table columns"},
						"created_at": {"type": "string", "description": "Creation timestamp"},
						"updated_at": {"type": "string", "description": "Last update timestamp"}
					}
				}`),
			},
			{
				Name:        "example_user",
				Description: "Database user resource",
				Operations:  []string{"create", "read", "update", "delete"},
				ConfigSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"username": {"type": "string", "description": "Username"},
						"password": {"type": "string", "description": "Password"},
						"roles": {"type": "array", "description": "User roles"}
					},
					"required": ["username", "password"]
				}`),
				StateSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"id": {"type": "string", "description": "Resource ID"},
						"username": {"type": "string", "description": "Username"},
						"roles": {"type": "array", "description": "User roles"},
						"created_at": {"type": "string", "description": "Creation timestamp"},
						"last_login": {"type": "string", "description": "Last login timestamp"}
					}
				}`),
			},
		},
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"endpoint": {
					"type": "string",
					"description": "Database endpoint URL"
				},
				"api_key": {
					"type": "string",
					"description": "API authentication key"
				},
				"timeout": {
					"type": "string",
					"description": "Connection timeout duration"
				}
			}
		}`),
	}, nil
}

// Configure initializes the provider with the given configuration
func (p *ExampleProvider) Configure(ctx context.Context, config map[string]interface{}) error {
	if endpoint, ok := config["endpoint"].(string); ok {
		p.endpoint = endpoint
	}
	if apiKey, ok := config["api_key"].(string); ok {
		p.apiKey = apiKey
	}
	if timeout, ok := config["timeout"].(string); ok {
		p.timeout = timeout
	}

	log.Printf("Configured example provider: endpoint=%s, timeout=%s", p.endpoint, p.timeout)
	return nil
}

// CallFunction handles function calls by routing to appropriate handlers
func (p *ExampleProvider) CallFunction(ctx context.Context, function string, input json.RawMessage) (json.RawMessage, error) {
	// Parse function name to determine resource type and action
	switch function {
	case "create_table":
		return p.handleCreateTable(ctx, input)
	case "read_table":
		return p.handleReadTable(ctx, input)
	case "update_table":
		return p.handleUpdateTable(ctx, input)
	case "delete_table":
		return p.handleDeleteTable(ctx, input)
	case "create_user":
		return p.handleCreateUser(ctx, input)
	case "read_user":
		return p.handleReadUser(ctx, input)
	case "update_user":
		return p.handleUpdateUser(ctx, input)
	case "delete_user":
		return p.handleDeleteUser(ctx, input)
	default:
		return nil, fmt.Errorf("unsupported function: %s", function)
	}
}

// Table handlers
func (p *ExampleProvider) handleCreateTable(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var req map[string]interface{}
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	// Validate required fields
	name, ok := req["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("table name is required")
	}

	// Create table logic would go here
	log.Printf("Creating table: %s", name)

	// Return result
	result := map[string]interface{}{
		"id":         "table_" + name,
		"name":       name,
		"created_at": "2024-01-01T00:00:00Z",
		"updated_at": "2024-01-01T00:00:00Z",
	}

	return json.Marshal(result)
}

func (p *ExampleProvider) handleReadTable(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	// Implementation would read table details
	result := map[string]interface{}{
		"id":   "table_example",
		"name": "example_table",
	}
	return json.Marshal(result)
}

func (p *ExampleProvider) handleUpdateTable(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	// Implementation would update table
	result := map[string]interface{}{
		"id":         "table_example",
		"name":       "example_table",
		"updated_at": "2024-01-01T01:00:00Z",
	}
	return json.Marshal(result)
}

func (p *ExampleProvider) handleDeleteTable(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	// Implementation would delete table
	result := map[string]interface{}{
		"deleted": true,
	}
	return json.Marshal(result)
}

// User handlers
func (p *ExampleProvider) handleCreateUser(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var req map[string]interface{}
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	username, ok := req["username"].(string)
	if !ok || username == "" {
		return nil, fmt.Errorf("username is required")
	}

	password, ok := req["password"].(string)
	if !ok || password == "" {
		return nil, fmt.Errorf("password is required")
	}

	log.Printf("Creating user: %s", username)

	result := map[string]interface{}{
		"id":         "user_" + username,
		"username":   username,
		"created_at": "2024-01-01T00:00:00Z",
	}

	return json.Marshal(result)
}

func (p *ExampleProvider) handleReadUser(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	result := map[string]interface{}{
		"id":       "user_example",
		"username": "example_user",
	}
	return json.Marshal(result)
}

func (p *ExampleProvider) handleUpdateUser(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	result := map[string]interface{}{
		"id":       "user_example",
		"username": "example_user",
	}
	return json.Marshal(result)
}

func (p *ExampleProvider) handleDeleteUser(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	result := map[string]interface{}{
		"deleted": true,
	}
	return json.Marshal(result)
}

func main() {
	provider := NewExampleProvider()
	fmt.Printf("Example provider created: %+v\n", provider)

	// Example usage
	ctx := context.Background()

	// Configure the provider
	config := map[string]interface{}{
		"endpoint": "localhost:5432",
		"api_key":  "example-key",
		"timeout":  "30s",
	}

	if err := provider.Configure(ctx, config); err != nil {
		log.Fatalf("Failed to configure provider: %v", err)
	}

	// Get schema
	schema, err := provider.Schema()
	if err != nil {
		log.Fatalf("Failed to get schema: %v", err)
	}

	fmt.Printf("Provider schema: %s v%s\n", schema.Name, schema.Version)
	fmt.Printf("Supported functions: %v\n", schema.SupportedFunctions)
}
