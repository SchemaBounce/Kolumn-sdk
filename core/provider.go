// Package core provides the essential interfaces and types for Kolumn Provider SDK
//
// This package defines the core Provider interface that all Kolumn providers must implement.
// It follows a progressive disclosure pattern - start simple and add advanced features as needed.
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/schemabounce/kolumn/sdk/helpers/security"
)

const (
	// SDKVersion represents the current SDK version
	SDKVersion = "v0.1.0"

	// APIVersion represents the API compatibility version
	APIVersion = "v1"

	// ProtocolVersion represents the RPC protocol version
	ProtocolVersion = 1
)

// Provider is the core interface that all Kolumn providers must implement.
// This is the minimum interface - dead simple to get started.
type Provider interface {
	// Configure sets up the provider with the given configuration
	// Updated to accept map[string]interface{} for core compatibility
	Configure(ctx context.Context, config map[string]interface{}) error

	// Schema returns the provider's schema definition
	Schema() (*Schema, error)

	// CallFunction executes a provider function with unified dispatch
	// Supports function names: CreateResource, ReadResource, UpdateResource, DeleteResource, etc.
	CallFunction(ctx context.Context, function string, input []byte) ([]byte, error)

	// Close cleans up provider resources
	Close() error
}

// Config represents provider configuration
type Config interface {
	// Get returns a configuration value by key
	Get(key string) (interface{}, bool)

	// GetString returns a string configuration value
	GetString(key string) (string, error)

	// GetInt returns an integer configuration value
	GetInt(key string) (int, error)

	// GetBool returns a boolean configuration value
	GetBool(key string) (bool, error)

	// Set adds or updates a configuration value
	Set(key string, value interface{})

	// Keys returns all configuration keys
	Keys() []string

	// Validate validates the configuration
	Validate() error
}

// Schema defines the provider's capabilities and supported object types
// Updated to match core expectations with SupportedFunctions and ResourceTypes
type Schema struct {
	// Provider metadata
	Name        string `json:"name"`
	Version     string `json:"version"`
	Protocol    string `json:"protocol"`
	Type        string `json:"type"`
	Description string `json:"description"`

	// Core compatibility fields - these match the core ProviderSchema structure
	SupportedFunctions []string                 `json:"supported_functions"` // Functions this provider implements
	ResourceTypes      []ResourceTypeDefinition `json:"resource_types"`      // Resource types this provider manages
	ConfigSchema       json.RawMessage          `json:"config_schema"`       // JSON schema for provider config

	// Legacy fields for backward compatibility (deprecated - use ResourceTypes instead)
	CreateObjects   map[string]*ObjectType `json:"create_objects,omitempty"`
	DiscoverObjects map[string]*ObjectType `json:"discover_objects,omitempty"`

	// Available functions (deprecated - use SupportedFunctions instead)
	Functions map[string]*Function `json:"functions,omitempty"`
}

// ResourceTypeDefinition describes a resource type the provider can manage
// This matches the core expectation exactly
type ResourceTypeDefinition struct {
	Name         string          `json:"name"`          // Resource type name (table, topic, bucket, etc.)
	Description  string          `json:"description"`   // Human readable description
	ConfigSchema json.RawMessage `json:"config_schema"` // JSON schema for resource config
	StateSchema  json.RawMessage `json:"state_schema"`  // JSON schema for resource state
	Operations   []string        `json:"operations"`    // Supported operations (create, read, update, delete)
}

// ObjectType defines a specific object type the provider supports
type ObjectType struct {
	// Basic metadata
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"` // e.g., "database", "storage", "streaming"

	// Type classification
	Type ObjectClassification `json:"type"` // CREATE or DISCOVER

	// Schema for this object type
	Properties map[string]*Property `json:"properties"`
	Required   []string             `json:"required"`
	Optional   []string             `json:"optional"`

	// Examples specific to this object type
	Examples []*ObjectExample `json:"examples,omitempty"`
}

// ObjectClassification categorizes object types
type ObjectClassification string

const (
	// CREATE objects are resources the provider can create and manage
	CREATE ObjectClassification = "create"

	// DISCOVER objects are existing infrastructure the provider can find and analyze
	DISCOVER ObjectClassification = "discover"
)

// Property defines a property of an object type
type Property struct {
	Type        string      `json:"type"` // "string", "integer", "boolean", etc.
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"`
	Examples    []string    `json:"examples,omitempty"`
	Validation  *Validation `json:"validation,omitempty"`
}

// Validation defines validation rules for a property
type Validation struct {
	Pattern   string        `json:"pattern,omitempty"` // regex pattern
	MinLength *int          `json:"min_length,omitempty"`
	MaxLength *int          `json:"max_length,omitempty"`
	Minimum   *float64      `json:"minimum,omitempty"`
	Maximum   *float64      `json:"maximum,omitempty"`
	Enum      []interface{} `json:"enum,omitempty"` // allowed values
}

// ConfigSchema defines the provider's configuration schema
type ConfigSchema struct {
	Properties map[string]*Property `json:"properties"`
	Required   []string             `json:"required"`
}

// Function defines a provider function
type Function struct {
	Description string               `json:"description"`
	Parameters  map[string]*Property `json:"parameters,omitempty"`
	Returns     *Property            `json:"returns,omitempty"`
	Examples    []*FunctionExample   `json:"examples,omitempty"`
}

// FunctionExample shows how to call a provider function
type FunctionExample struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Input       interface{} `json:"input"`
	Output      interface{} `json:"output"`
}

// secureConfig provides a secure Config implementation with sensitive data protection
type secureConfig struct {
	data      map[string]interface{}
	sensitive map[string]bool
}

// simpleConfig provides a basic Config implementation (deprecated - use NewSecureConfig)
type simpleConfig struct {
	data map[string]interface{}
}

// NewConfig creates a new basic configuration instance (deprecated - use NewSecureConfig)
func NewConfig() Config {
	return &simpleConfig{
		data: make(map[string]interface{}),
	}
}

// NewSecureConfig creates a new secure configuration instance with sensitive data protection
func NewSecureConfig() Config {
	return &secureConfig{
		data:      make(map[string]interface{}),
		sensitive: make(map[string]bool),
	}
}

// MarkSensitive marks a field as containing sensitive data in secureConfig
func (c *secureConfig) MarkSensitive(key string) {
	c.sensitive[key] = true
}

// IsSensitive checks if a field contains sensitive data
func (c *secureConfig) IsSensitive(key string) bool {
	return c.sensitive[key]
}

// GetSanitized returns a sanitized copy of the config for logging/display
func (c *secureConfig) GetSanitized() map[string]interface{} {
	sanitized := make(map[string]interface{})
	for key, value := range c.data {
		if c.sensitive[key] {
			sanitized[key] = "[REDACTED]"
		} else {
			sanitized[key] = value
		}
	}
	return sanitized
}

// =============================================================================
// UNIFIED FUNCTION DISPATCH HELPERS
// =============================================================================

// UnifiedDispatcher helps bridge between new unified function dispatch and existing registries
type UnifiedDispatcher struct {
	createRegistry   CreateRegistry
	discoverRegistry DiscoverRegistry
}

// CreateRegistry interface for create operations
type CreateRegistry interface {
	CallHandler(ctx context.Context, objectType, method string, input []byte) ([]byte, error)
	GetObjectTypes() map[string]*ObjectType
}

// DiscoverRegistry interface for discover operations
type DiscoverRegistry interface {
	CallHandler(ctx context.Context, objectType, method string, input []byte) ([]byte, error)
	GetObjectTypes() map[string]*ObjectType
}

// NewUnifiedDispatcher creates a new dispatcher
func NewUnifiedDispatcher(createReg CreateRegistry, discoverReg DiscoverRegistry) *UnifiedDispatcher {
	return &UnifiedDispatcher{
		createRegistry:   createReg,
		discoverRegistry: discoverReg,
	}
}

// Dispatch handles unified function calls and routes them to appropriate registries
func (d *UnifiedDispatcher) Dispatch(ctx context.Context, function string, input []byte) ([]byte, error) {
	// SECURITY: Validate function name against allowed functions
	allowedFunctions := map[string]bool{
		"CreateResource":    true,
		"ReadResource":      true,
		"UpdateResource":    true,
		"DeleteResource":    true,
		"DiscoverResources": true,
		"Ping":              true,
	}

	if !allowedFunctions[function] {
		return nil, security.NewSecureError(
			"operation not supported",
			fmt.Sprintf("function not allowed: %s", function),
			"INVALID_FUNCTION",
		)
	}

	// Route to appropriate handler with security validation
	switch function {
	case "CreateResource":
		return d.handleCreateResource(ctx, input)
	case "ReadResource":
		return d.handleReadResource(ctx, input)
	case "UpdateResource":
		return d.handleUpdateResource(ctx, input)
	case "DeleteResource":
		return d.handleDeleteResource(ctx, input)
	case "DiscoverResources":
		return d.handleDiscoverResources(ctx, input)
	case "Ping":
		return d.handlePing(ctx, input)
	default:
		// This should never be reached due to validation above
		return nil, security.NewSecureError(
			"operation not supported",
			fmt.Sprintf("unexpected function: %s", function),
			"UNEXPECTED_FUNCTION",
		)
	}
}

func (d *UnifiedDispatcher) handleCreateResource(ctx context.Context, input []byte) ([]byte, error) {
	// SECURITY: Use safe unmarshaling with size and depth limits
	var unifiedReq map[string]interface{}
	if err := security.SafeUnmarshal(input, &unifiedReq); err != nil {
		return nil, security.NewSecureError(
			"invalid request format",
			fmt.Sprintf("create request unmarshal failed: %v", err),
			"INVALID_REQUEST",
		)
	}

	resourceType, ok := unifiedReq["resource_type"].(string)
	if !ok {
		return nil, security.NewSecureError(
			"invalid request format",
			"missing resource_type in request",
			"MISSING_RESOURCE_TYPE",
		)
	}

	// SECURITY: Validate resource type
	if err := security.ValidateObjectType(resourceType); err != nil {
		return nil, security.NewSecureError(
			"invalid resource type",
			fmt.Sprintf("resource type validation failed: %v", err),
			"INVALID_RESOURCE_TYPE",
		)
	}

	// SECURITY: Validate request configuration size
	if config, ok := unifiedReq["config"].(map[string]interface{}); ok {
		validator := &security.InputSizeValidator{}
		if err := validator.ValidateConfigSize(config); err != nil {
			return nil, security.NewSecureError(
				"request too large",
				fmt.Sprintf("create request config validation failed: %v", err),
				"REQUEST_TOO_LARGE",
			)
		}
	}

	// Transform unified request format to create registry format
	createReq := map[string]interface{}{
		"object_type": resourceType, // Transform resource_type -> object_type
		"name":        unifiedReq["name"],
		"config":      unifiedReq["config"],
	}

	// Include optional fields if present
	if deps, ok := unifiedReq["dependencies"]; ok {
		createReq["dependencies"] = deps
	}
	if options, ok := unifiedReq["options"]; ok {
		createReq["options"] = options
	}
	if metadata, ok := unifiedReq["metadata"]; ok {
		createReq["metadata"] = metadata
	}

	// Marshal the transformed request
	transformedInput, err := json.Marshal(createReq)
	if err != nil {
		return nil, security.NewSecureError(
			"request transformation failed",
			fmt.Sprintf("failed to transform request: %v", err),
			"TRANSFORMATION_FAILED",
		)
	}

	if d.createRegistry != nil {
		return d.createRegistry.CallHandler(ctx, resourceType, "create", transformedInput)
	}

	return nil, security.NewSecureError(
		"registry not available",
		fmt.Sprintf("no create registry available for resource type: %s", resourceType),
		"REGISTRY_NOT_FOUND",
	)
}

func (d *UnifiedDispatcher) handleReadResource(ctx context.Context, input []byte) ([]byte, error) {
	// SECURITY: Use safe unmarshaling with size and depth limits
	var unifiedReq map[string]interface{}
	if err := security.SafeUnmarshal(input, &unifiedReq); err != nil {
		return nil, security.NewSecureError(
			"invalid request format",
			fmt.Sprintf("read request unmarshal failed: %v", err),
			"INVALID_REQUEST",
		)
	}

	resourceType, ok := unifiedReq["resource_type"].(string)
	if !ok {
		return nil, security.NewSecureError(
			"invalid request format",
			"missing resource_type in request",
			"MISSING_RESOURCE_TYPE",
		)
	}

	// SECURITY: Validate resource type
	if err := security.ValidateObjectType(resourceType); err != nil {
		return nil, security.NewSecureError(
			"invalid resource type",
			fmt.Sprintf("resource type validation failed: %v", err),
			"INVALID_RESOURCE_TYPE",
		)
	}

	// Transform unified request format to create registry format
	readReq := map[string]interface{}{
		"object_type": resourceType, // Transform resource_type -> object_type
		"resource_id": unifiedReq["resource_id"],
		"name":        unifiedReq["name"],
	}

	transformedInput, err := json.Marshal(readReq)
	if err != nil {
		return nil, security.NewSecureError(
			"request transformation failed",
			fmt.Sprintf("failed to transform request: %v", err),
			"TRANSFORMATION_FAILED",
		)
	}

	if d.createRegistry != nil {
		return d.createRegistry.CallHandler(ctx, resourceType, "read", transformedInput)
	}

	return nil, security.NewSecureError(
		"registry not available",
		fmt.Sprintf("no create registry available for resource type: %s", resourceType),
		"REGISTRY_NOT_FOUND",
	)
}

func (d *UnifiedDispatcher) handleUpdateResource(ctx context.Context, input []byte) ([]byte, error) {
	// SECURITY: Use safe unmarshaling with size and depth limits
	var unifiedReq map[string]interface{}
	if err := security.SafeUnmarshal(input, &unifiedReq); err != nil {
		return nil, security.NewSecureError(
			"invalid request format",
			fmt.Sprintf("update request unmarshal failed: %v", err),
			"INVALID_REQUEST",
		)
	}

	resourceType, ok := unifiedReq["resource_type"].(string)
	if !ok {
		return nil, security.NewSecureError(
			"invalid request format",
			"missing resource_type in request",
			"MISSING_RESOURCE_TYPE",
		)
	}

	// SECURITY: Validate resource type
	if err := security.ValidateObjectType(resourceType); err != nil {
		return nil, security.NewSecureError(
			"invalid resource type",
			fmt.Sprintf("resource type validation failed: %v", err),
			"INVALID_RESOURCE_TYPE",
		)
	}

	// SECURITY: Validate request configuration size
	if config, ok := unifiedReq["config"].(map[string]interface{}); ok {
		validator := &security.InputSizeValidator{}
		if err := validator.ValidateConfigSize(config); err != nil {
			return nil, security.NewSecureError(
				"request too large",
				fmt.Sprintf("update request config validation failed: %v", err),
				"REQUEST_TOO_LARGE",
			)
		}
	}

	// Transform unified request format to create registry format
	updateReq := map[string]interface{}{
		"object_type": resourceType, // Transform resource_type -> object_type
		"resource_id": unifiedReq["resource_id"],
		"name":        unifiedReq["name"],
		"config":      unifiedReq["config"],
	}

	// Include optional fields if present
	if currentState, ok := unifiedReq["current_state"]; ok {
		updateReq["current_state"] = currentState
	}
	if options, ok := unifiedReq["options"]; ok {
		updateReq["options"] = options
	}

	transformedInput, err := json.Marshal(updateReq)
	if err != nil {
		return nil, security.NewSecureError(
			"request transformation failed",
			fmt.Sprintf("failed to transform request: %v", err),
			"TRANSFORMATION_FAILED",
		)
	}

	if d.createRegistry != nil {
		return d.createRegistry.CallHandler(ctx, resourceType, "update", transformedInput)
	}

	return nil, security.NewSecureError(
		"registry not available",
		fmt.Sprintf("no create registry available for resource type: %s", resourceType),
		"REGISTRY_NOT_FOUND",
	)
}

func (d *UnifiedDispatcher) handleDeleteResource(ctx context.Context, input []byte) ([]byte, error) {
	// SECURITY: Use safe unmarshaling with size and depth limits
	var unifiedReq map[string]interface{}
	if err := security.SafeUnmarshal(input, &unifiedReq); err != nil {
		return nil, security.NewSecureError(
			"invalid request format",
			fmt.Sprintf("delete request unmarshal failed: %v", err),
			"INVALID_REQUEST",
		)
	}

	resourceType, ok := unifiedReq["resource_type"].(string)
	if !ok {
		return nil, security.NewSecureError(
			"invalid request format",
			"missing resource_type in request",
			"MISSING_RESOURCE_TYPE",
		)
	}

	// SECURITY: Validate resource type
	if err := security.ValidateObjectType(resourceType); err != nil {
		return nil, security.NewSecureError(
			"invalid resource type",
			fmt.Sprintf("resource type validation failed: %v", err),
			"INVALID_RESOURCE_TYPE",
		)
	}

	// Transform unified request format to create registry format
	deleteReq := map[string]interface{}{
		"object_type": resourceType, // Transform resource_type -> object_type
		"resource_id": unifiedReq["resource_id"],
		"name":        unifiedReq["name"],
	}

	// Include optional fields if present
	if state, ok := unifiedReq["state"]; ok {
		deleteReq["state"] = state
	}
	if options, ok := unifiedReq["options"]; ok {
		deleteReq["options"] = options
	}

	transformedInput, err := json.Marshal(deleteReq)
	if err != nil {
		return nil, security.NewSecureError(
			"request transformation failed",
			fmt.Sprintf("failed to transform request: %v", err),
			"TRANSFORMATION_FAILED",
		)
	}

	if d.createRegistry != nil {
		return d.createRegistry.CallHandler(ctx, resourceType, "delete", transformedInput)
	}

	return nil, security.NewSecureError(
		"registry not available",
		fmt.Sprintf("no create registry available for resource type: %s", resourceType),
		"REGISTRY_NOT_FOUND",
	)
}

func (d *UnifiedDispatcher) handleDiscoverResources(ctx context.Context, input []byte) ([]byte, error) {
	// SECURITY: Use safe unmarshaling with size and depth limits
	var unifiedReq map[string]interface{}
	if err := security.SafeUnmarshal(input, &unifiedReq); err != nil {
		return nil, security.NewSecureError(
			"invalid request format",
			fmt.Sprintf("discover request unmarshal failed: %v", err),
			"INVALID_REQUEST",
		)
	}

	resourceType, ok := unifiedReq["resource_type"].(string)
	if !ok {
		return nil, security.NewSecureError(
			"invalid request format",
			"missing resource_type in request",
			"MISSING_RESOURCE_TYPE",
		)
	}

	// SECURITY: Validate resource type
	if err := security.ValidateObjectType(resourceType); err != nil {
		return nil, security.NewSecureError(
			"invalid resource type",
			fmt.Sprintf("resource type validation failed: %v", err),
			"INVALID_RESOURCE_TYPE",
		)
	}

	// Transform unified request format to discover registry format
	// For discover operations, we primarily use "scan" method
	discoverReq := map[string]interface{}{
		"object_types": []string{resourceType},
	}

	// Include filters if present
	if filters, ok := unifiedReq["filters"]; ok {
		discoverReq["filters"] = filters
	}
	if options, ok := unifiedReq["options"]; ok {
		discoverReq["options"] = options
	}

	transformedInput, err := json.Marshal(discoverReq)
	if err != nil {
		return nil, security.NewSecureError(
			"request transformation failed",
			fmt.Sprintf("failed to transform request: %v", err),
			"TRANSFORMATION_FAILED",
		)
	}

	if d.discoverRegistry != nil {
		return d.discoverRegistry.CallHandler(ctx, resourceType, "scan", transformedInput)
	}

	return nil, security.NewSecureError(
		"registry not available",
		fmt.Sprintf("no discover registry available for resource type: %s", resourceType),
		"REGISTRY_NOT_FOUND",
	)
}

func (d *UnifiedDispatcher) handlePing(ctx context.Context, input []byte) ([]byte, error) {
	response := map[string]interface{}{
		"success": true,
		"status":  "healthy",
	}
	return json.Marshal(response)
}

// BuildCompatibleSchema builds a core-compatible schema from registries
func (d *UnifiedDispatcher) BuildCompatibleSchema(name, version, providerType, description string) *Schema {
	schema := &Schema{
		Name:         name,
		Version:      version,
		Protocol:     "1.0",
		Type:         providerType,
		Description:  description,
		ConfigSchema: json.RawMessage(`{}`), // Basic config schema
	}

	// Build supported functions
	var supportedFunctions []string
	var resourceTypes []ResourceTypeDefinition

	// Add core functions
	supportedFunctions = append(supportedFunctions,
		"CreateResource", "ReadResource", "UpdateResource", "DeleteResource", "Ping")

	// Build resource types from registries
	if d.createRegistry != nil {
		createObjects := d.createRegistry.GetObjectTypes()
		for name, objType := range createObjects {
			resourceTypes = append(resourceTypes, ResourceTypeDefinition{
				Name:         name,
				Description:  objType.Description,
				Operations:   []string{"create", "read", "update", "delete"},
				ConfigSchema: json.RawMessage(`{}`),
				StateSchema:  json.RawMessage(`{}`),
			})
		}
	}

	if d.discoverRegistry != nil {
		supportedFunctions = append(supportedFunctions, "DiscoverResources")
		discoverObjects := d.discoverRegistry.GetObjectTypes()
		for name, objType := range discoverObjects {
			resourceTypes = append(resourceTypes, ResourceTypeDefinition{
				Name:         name,
				Description:  objType.Description,
				Operations:   []string{"discover"},
				ConfigSchema: json.RawMessage(`{}`),
				StateSchema:  json.RawMessage(`{}`),
			})
		}
	}

	schema.SupportedFunctions = supportedFunctions
	schema.ResourceTypes = resourceTypes

	// Maintain backward compatibility
	if d.createRegistry != nil {
		schema.CreateObjects = d.createRegistry.GetObjectTypes()
	}
	if d.discoverRegistry != nil {
		schema.DiscoverObjects = d.discoverRegistry.GetObjectTypes()
	}

	return schema
}

// Get implements Config
func (c *simpleConfig) Get(key string) (interface{}, bool) {
	value, exists := c.data[key]
	return value, exists
}

// GetString implements Config
func (c *simpleConfig) GetString(key string) (string, error) {
	if value, exists := c.data[key]; exists {
		if str, ok := value.(string); ok {
			return str, nil
		}
		return "", fmt.Errorf("value for key '%s' is not a string", key)
	}
	return "", fmt.Errorf("key '%s' not found", key)
}

// GetInt implements Config
func (c *simpleConfig) GetInt(key string) (int, error) {
	if value, exists := c.data[key]; exists {
		if i, ok := value.(int); ok {
			return i, nil
		}
		return 0, fmt.Errorf("value for key '%s' is not an integer", key)
	}
	return 0, fmt.Errorf("key '%s' not found", key)
}

// GetBool implements Config
func (c *simpleConfig) GetBool(key string) (bool, error) {
	if value, exists := c.data[key]; exists {
		if b, ok := value.(bool); ok {
			return b, nil
		}
		return false, fmt.Errorf("value for key '%s' is not a boolean", key)
	}
	return false, fmt.Errorf("key '%s' not found", key)
}

// Set implements Config
func (c *simpleConfig) Set(key string, value interface{}) {
	c.data[key] = value
}

// Keys implements Config
func (c *simpleConfig) Keys() []string {
	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}

// Validate implements Config
func (c *simpleConfig) Validate() error {
	return nil
}

// SECURE CONFIG IMPLEMENTATIONS

// Get implements Config for secureConfig
func (c *secureConfig) Get(key string) (interface{}, bool) {
	value, exists := c.data[key]
	return value, exists
}

// GetString implements Config for secureConfig
func (c *secureConfig) GetString(key string) (string, error) {
	if value, exists := c.data[key]; exists {
		if str, ok := value.(string); ok {
			return str, nil
		}
		return "", fmt.Errorf("value for key '%s' is not a string", key)
	}
	return "", fmt.Errorf("key '%s' not found", key)
}

// GetInt implements Config for secureConfig
func (c *secureConfig) GetInt(key string) (int, error) {
	if value, exists := c.data[key]; exists {
		if i, ok := value.(int); ok {
			return i, nil
		}
		return 0, fmt.Errorf("value for key '%s' is not an integer", key)
	}
	return 0, fmt.Errorf("key '%s' not found", key)
}

// GetBool implements Config for secureConfig
func (c *secureConfig) GetBool(key string) (bool, error) {
	if value, exists := c.data[key]; exists {
		if b, ok := value.(bool); ok {
			return b, nil
		}
		return false, fmt.Errorf("value for key '%s' is not a boolean", key)
	}
	return false, fmt.Errorf("key '%s' not found", key)
}

// Set implements Config for secureConfig with automatic sensitive field detection
func (c *secureConfig) Set(key string, value interface{}) {
	c.data[key] = value

	// Automatically mark common sensitive fields
	lowerKey := strings.ToLower(key)
	if strings.Contains(lowerKey, "password") ||
		strings.Contains(lowerKey, "secret") ||
		strings.Contains(lowerKey, "token") ||
		strings.Contains(lowerKey, "key") ||
		strings.Contains(lowerKey, "credential") {
		c.sensitive[key] = true
	}
}

// Keys implements Config for secureConfig
func (c *secureConfig) Keys() []string {
	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}

// Validate implements Config for secureConfig with enhanced validation
func (c *secureConfig) Validate() error {
	// Enhanced validation for secure config
	for key, value := range c.data {
		// Validate sensitive fields are not empty
		if c.sensitive[key] {
			if value == nil || value == "" {
				return fmt.Errorf("sensitive field '%s' cannot be empty", key)
			}

			// Validate sensitive string length
			if str, ok := value.(string); ok {
				if len(str) < 8 {
					return fmt.Errorf("sensitive field '%s' is too short (minimum 8 characters)", key)
				}
			}
		}
	}
	return nil
}

// NewValidationError creates a validation error for a specific field
func NewValidationError(field, message string) error {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// ValidationRule defines validation for a property
type ValidationRule struct {
	Field       string        `json:"field"`
	Required    bool          `json:"required"`
	Type        string        `json:"type"`
	Pattern     string        `json:"pattern,omitempty"`
	MinValue    *float64      `json:"min_value,omitempty"`
	MaxValue    *float64      `json:"max_value,omitempty"`
	MinItems    *int          `json:"min_items,omitempty"`
	MaxItems    *int          `json:"max_items,omitempty"`
	Enum        []interface{} `json:"enum,omitempty"`
	Description string        `json:"description"`
}

// ValidationIssue represents a validation problem (compatibility alias for ValidationError)
type ValidationIssue struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// =============================================================================
// OBJECT HANDLER INTERFACES
// =============================================================================

// CreateObjectHandler handles CREATE objects
type CreateObjectHandler interface {
	Create(ctx context.Context, req *CreateRequest) (*CreateResponse, error)
	Read(ctx context.Context, req *ReadRequest) (*ReadResponse, error)
	Update(ctx context.Context, req *UpdateRequest) (*UpdateResponse, error)
	Delete(ctx context.Context, req *DeleteRequest) (*DeleteResponse, error)
	Schema() *ObjectType
}

// DiscoverObjectHandler handles DISCOVER objects
type DiscoverObjectHandler interface {
	Discover(ctx context.Context, req *DiscoverRequest) (*DiscoverResponse, error)
	Schema() *ObjectType
}

// =============================================================================
// PROVIDER EXTENSION INTERFACES
// =============================================================================

// DocumentedProvider extends Provider with documentation capabilities
type DocumentedProvider interface {
	Provider
	Documentation() *ProviderDocumentation
	GenerateDocumentation() ([]byte, error)
	ObjectDocumentation(objectType string) (*ObjectDocumentation, error)
}

// EnterpriseProvider extends Provider with enterprise features
type EnterpriseProvider interface {
	Provider
	HealthCheck(ctx context.Context) (*HealthStatus, error)
	Metrics(ctx context.Context) (*ProviderMetrics, error)
}

// =============================================================================
// DOCUMENTATION TYPES
// =============================================================================

// ProviderDocumentation contains comprehensive provider documentation
type ProviderDocumentation struct {
	Name            string                          `json:"name"`
	Version         string                          `json:"version"`
	Description     string                          `json:"description"`
	Overview        string                          `json:"overview"`
	CreateObjects   map[string]*ObjectDocumentation `json:"create_objects"`
	DiscoverObjects map[string]*ObjectDocumentation `json:"discover_objects"`
	Examples        []*ProviderExample              `json:"examples,omitempty"`
	Links           []DocumentationLink             `json:"links,omitempty"`
}

// ObjectDocumentation contains object-specific documentation
type ObjectDocumentation struct {
	Schema        *ObjectType         `json:"schema"`
	Usage         string              `json:"usage,omitempty"`
	Examples      []*ObjectExample    `json:"examples,omitempty"`
	BestPractices []string            `json:"best_practices,omitempty"`
	Links         []DocumentationLink `json:"links,omitempty"`
}

// ProviderExample shows how to use the provider
type ProviderExample struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	HCL         string `json:"hcl"`
	Category    string `json:"category,omitempty"`
}

// DocumentationLink represents a documentation link
type DocumentationLink struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Type  string `json:"type,omitempty"` // "official", "tutorial", "example"
}

// =============================================================================
// HEALTH CHECK AND METRICS TYPES
// =============================================================================

// HealthStatus represents provider health status
type HealthStatus struct {
	Healthy   bool                   `json:"healthy"`
	Status    string                 `json:"status"`
	LastCheck time.Time              `json:"last_check"`
	Uptime    time.Duration          `json:"uptime"`
	Checks    map[string]CheckResult `json:"checks,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// CheckResult represents the result of a health check
type CheckResult struct {
	Name      string        `json:"name"`
	Passed    bool          `json:"passed"`
	Message   string        `json:"message,omitempty"`
	CheckedAt time.Time     `json:"checked_at"`
	Duration  time.Duration `json:"duration,omitempty"`
}

// ProviderMetrics contains provider performance metrics
type ProviderMetrics struct {
	TotalRequests       int64                  `json:"total_requests"`
	SuccessfulRequests  int64                  `json:"successful_requests"`
	FailedRequests      int64                  `json:"failed_requests"`
	AverageResponseTime time.Duration          `json:"average_response_time"`
	StartTime           time.Time              `json:"start_time"`
	LastRequest         time.Time              `json:"last_request"`
	CollectedAt         time.Time              `json:"collected_at"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}
