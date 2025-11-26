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
//
// ⚠️ CRITICAL: This interface MUST have exactly 4 methods - no more, no less.
// This enforces the 4-method RPC pattern that maintains compatibility with Kolumn core.
//
// ValidateConfig was intentionally REMOVED to maintain interface purity.
// Use validation helpers within Configure() instead of separate validation methods.
//
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
	DisplayName string `json:"display_name,omitempty"` // Optional display name for UI prefixes (e.g., "POSTGRES", "MYSQL")

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
	Type        string              `json:"type"` // "string", "integer", "boolean", etc.
	Description string              `json:"description"`
	Default     interface{}         `json:"default,omitempty"`
	Examples    []string            `json:"examples,omitempty"`
	Validation  *Validation         `json:"validation,omitempty"`
	Enhanced    *EnhancedValidation `json:"enhanced_validation,omitempty"` // Advanced validation
}

// Validation defines validation rules for a property
type Validation struct {
	Pattern     string        `json:"pattern,omitempty"` // regex pattern
	MinLength   *int          `json:"min_length,omitempty"`
	MaxLength   *int          `json:"max_length,omitempty"`
	Minimum     *float64      `json:"minimum,omitempty"`
	Maximum     *float64      `json:"maximum,omitempty"`
	Enum        []interface{} `json:"enum,omitempty"`        // allowed values
	Required    bool          `json:"required,omitempty"`    // whether field is required
	ErrorMsg    string        `json:"error_msg,omitempty"`   // custom error message
	Suggestion  string        `json:"suggestion,omitempty"`  // suggestion for fixing errors
	Example     string        `json:"example,omitempty"`     // example of valid value
	Description string        `json:"description,omitempty"` // validation description
}

// EnhancedValidation provides advanced validation rules using the validation framework
type EnhancedValidation struct {
	Rules       []ConfigValidationRule `json:"rules"`                 // Validation rules from framework
	Suggestions []string               `json:"suggestions,omitempty"` // Fix suggestions
	Examples    []string               `json:"examples,omitempty"`    // Valid examples
	DocLinks    []string               `json:"doc_links,omitempty"`   // Documentation links
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

// =============================================================================
// VALIDATION FRAMEWORK INTEGRATION
// =============================================================================

// ValidateConfig validates provider configuration using the validation framework
func (s *Schema) ValidateConfig(config map[string]interface{}) *ConfigValidationResult {
	validator := NewValidator(s.Name)

	// Convert legacy ObjectType properties to validation rules
	for objTypeName, objType := range s.CreateObjects {
		for propName, prop := range objType.Properties {
			rule := s.convertPropertyToValidationRule(objTypeName, propName, prop)
			validator.AddRule(rule)
		}
	}

	// Convert ResourceTypeDefinition config schemas to validation rules
	for _, resourceType := range s.ResourceTypes {
		rules := s.convertResourceTypeToValidationRules(resourceType)
		validator.AddRules(rules)
	}

	return validator.Validate(config)
}

// convertPropertyToValidationRule converts a Property to a ConfigValidationRule
func (s *Schema) convertPropertyToValidationRule(objType, propName string, prop *Property) ConfigValidationRule {
	rule := ConfigValidationRule{
		Field:       fmt.Sprintf("%s.%s", objType, propName),
		Type:        prop.Type,
		Description: prop.Description,
		Default:     prop.Default,
	}

	// Convert basic validation if present
	if prop.Validation != nil {
		rule.Pattern = prop.Validation.Pattern
		rule.Enum = make([]string, len(prop.Validation.Enum))
		for i, v := range prop.Validation.Enum {
			rule.Enum[i] = fmt.Sprintf("%v", v)
		}
		rule.Required = prop.Validation.Required
		rule.ErrorMsg = prop.Validation.ErrorMsg
		rule.Suggestion = prop.Validation.Suggestion
		rule.Example = prop.Validation.Example

		// Convert range constraints
		if prop.Validation.MinLength != nil {
			rule.Min = *prop.Validation.MinLength
		}
		if prop.Validation.MaxLength != nil {
			rule.Max = *prop.Validation.MaxLength
		}
		if prop.Validation.Minimum != nil {
			rule.Min = *prop.Validation.Minimum
		}
		if prop.Validation.Maximum != nil {
			rule.Max = *prop.Validation.Maximum
		}
	}

	// Use enhanced validation if available
	if prop.Enhanced != nil && len(prop.Enhanced.Rules) > 0 {
		// Use the first enhanced rule as the primary rule
		enhancedRule := prop.Enhanced.Rules[0]
		rule.Required = enhancedRule.Required
		rule.Type = enhancedRule.Type
		rule.Pattern = enhancedRule.Pattern
		rule.Min = enhancedRule.Min
		rule.Max = enhancedRule.Max
		rule.Enum = enhancedRule.Enum
		rule.ErrorMsg = enhancedRule.ErrorMsg
		rule.Suggestion = enhancedRule.Suggestion
		rule.Example = enhancedRule.Example
		rule.Custom = enhancedRule.Custom
	}

	return rule
}

// convertResourceTypeToValidationRules converts ResourceTypeDefinition to validation rules
func (s *Schema) convertResourceTypeToValidationRules(resourceType ResourceTypeDefinition) []ConfigValidationRule {
	rules := []ConfigValidationRule{}

	// Add basic resource type validation
	rules = append(rules, ConfigValidationRule{
		Field:       "resource_type",
		Required:    true,
		Type:        "string",
		Enum:        []string{resourceType.Name},
		Description: fmt.Sprintf("Must be '%s' for %s resources", resourceType.Name, resourceType.Description),
		ErrorMsg:    fmt.Sprintf("Invalid resource type, expected '%s'", resourceType.Name),
		Suggestion:  fmt.Sprintf("Use resource_type = \"%s\"", resourceType.Name),
		Example:     fmt.Sprintf("resource_type = \"%s\"", resourceType.Name),
	})

	// Parse ConfigSchema JSON if available
	if len(resourceType.ConfigSchema) > 0 {
		// This would require JSON schema parsing - for now, add basic validation
		rules = append(rules, ConfigValidationRule{
			Field:       fmt.Sprintf("%s.config", resourceType.Name),
			Required:    false,
			Type:        "map",
			Description: fmt.Sprintf("Configuration for %s", resourceType.Description),
		})
	}

	return rules
}

// AddValidationRule adds a validation rule to a property
func (p *Property) AddValidationRule(rule ConfigValidationRule) {
	if p.Enhanced == nil {
		p.Enhanced = &EnhancedValidation{
			Rules: []ConfigValidationRule{},
		}
	}
	p.Enhanced.Rules = append(p.Enhanced.Rules, rule)
}

// AddValidationSuggestion adds a validation suggestion
func (p *Property) AddValidationSuggestion(suggestion string) {
	if p.Enhanced == nil {
		p.Enhanced = &EnhancedValidation{}
	}
	p.Enhanced.Suggestions = append(p.Enhanced.Suggestions, suggestion)
}

// AddValidationExample adds a validation example
func (p *Property) AddValidationExample(example string) {
	if p.Enhanced == nil {
		p.Enhanced = &EnhancedValidation{}
	}
	p.Enhanced.Examples = append(p.Enhanced.Examples, example)
}

// AddDocumentationLink adds a documentation link
func (p *Property) AddDocumentationLink(link string) {
	if p.Enhanced == nil {
		p.Enhanced = &EnhancedValidation{}
	}
	p.Enhanced.DocLinks = append(p.Enhanced.DocLinks, link)
}

// GetValidationRules returns all validation rules for this property
func (p *Property) GetValidationRules() []ConfigValidationRule {
	rules := []ConfigValidationRule{}

	// Convert basic validation to rule
	if p.Validation != nil {
		rule := ConfigValidationRule{
			Required:   p.Validation.Required,
			Type:       p.Type,
			Pattern:    p.Validation.Pattern,
			Enum:       make([]string, len(p.Validation.Enum)),
			ErrorMsg:   p.Validation.ErrorMsg,
			Suggestion: p.Validation.Suggestion,
			Example:    p.Validation.Example,
		}

		// Convert enum values
		for i, v := range p.Validation.Enum {
			rule.Enum[i] = fmt.Sprintf("%v", v)
		}

		// Convert range constraints
		if p.Validation.MinLength != nil {
			rule.Min = *p.Validation.MinLength
		}
		if p.Validation.MaxLength != nil {
			rule.Max = *p.Validation.MaxLength
		}
		if p.Validation.Minimum != nil {
			rule.Min = *p.Validation.Minimum
		}
		if p.Validation.Maximum != nil {
			rule.Max = *p.Validation.Maximum
		}

		rules = append(rules, rule)
	}

	// Add enhanced validation rules
	if p.Enhanced != nil {
		rules = append(rules, p.Enhanced.Rules...)
	}

	return rules
}

// CreateValidationBuilder creates a new validation rule builder for this property
func (p *Property) CreateValidationBuilder(field string) *ValidationRuleBuilder {
	return NewValidationRule(field).Type(p.Type).Description(p.Description)
}

// =============================================================================
// BASE PROVIDER IMPLEMENTATION
// =============================================================================

// BaseProvider provides default implementations for the Provider interface
// Providers can embed this to get default behavior and only override what they need
type BaseProvider struct {
	schema    *Schema
	config    map[string]interface{}
	validator *Validator
}

// NewBaseProvider creates a new base provider instance
func NewBaseProvider(name string) *BaseProvider {
	return &BaseProvider{
		validator: NewValidator(name),
	}
}

// SetSchema sets the provider schema
func (bp *BaseProvider) SetSchema(schema *Schema) {
	bp.schema = schema
}

// GetSchema returns the provider schema (for use in internal validation)
func (bp *BaseProvider) GetSchema() *Schema {
	return bp.schema
}

// AddValidationRule adds a validation rule to the provider
func (bp *BaseProvider) AddValidationRule(rule ConfigValidationRule) {
	bp.validator.AddRule(rule)
}

// AddValidationRules adds multiple validation rules to the provider
func (bp *BaseProvider) AddValidationRules(rules []ConfigValidationRule) {
	bp.validator.AddRules(rules)
}

// ValidateConfiguration provides a helper method for internal configuration validation using the schema and validation framework
func (bp *BaseProvider) ValidateConfiguration(ctx context.Context, config map[string]interface{}) *ConfigValidationResult {
	// Store config for potential use by other methods
	bp.config = config

	// If we have a schema, use it for validation
	if bp.schema != nil {
		return bp.schema.ValidateConfig(config)
	}

	// If no schema but we have validation rules, use the validator directly
	if len(bp.validator.rules) > 0 {
		return bp.validator.Validate(config)
	}

	// Default: basic validation - just check for common fields
	bp.addCommonValidationRules()
	return bp.validator.Validate(config)
}

// addCommonValidationRules adds basic validation rules for common provider fields
func (bp *BaseProvider) addCommonValidationRules() {
	// Add validation for common provider configuration fields
	commonRules := []ConfigValidationRule{
		{
			Field:       "host",
			Type:        "string",
			Required:    false, // Not all providers need host
			Description: "Provider host address",
			Custom:      ValidateHost,
			Suggestion:  "Provide a valid hostname or IP address",
			Example:     "host = \"localhost\"",
		},
		{
			Field:       "port",
			Type:        "int",
			Required:    false, // Not all providers need port
			Description: "Provider port number",
			Custom:      ValidatePort,
			Suggestion:  "Provide a valid port number (1-65535)",
			Example:     "port = 5432",
		},
		{
			Field:       "database",
			Type:        "string",
			Required:    false, // Not all providers need database
			Description: "Database name",
			Custom:      ValidateDatabaseName,
			Suggestion:  "Provide a valid database name",
			Example:     "database = \"mydb\"",
		},
		{
			Field:       "username",
			Type:        "string",
			Required:    false, // Not all providers need username
			Description: "Username for authentication",
			Suggestion:  "Provide a valid username",
			Example:     "username = \"postgres\"",
		},
		{
			Field:       "password",
			Type:        "string",
			Required:    false, // Not all providers need password
			Description: "Password for authentication",
			Suggestion:  "Provide a valid password",
			Example:     "password = \"secret\"",
		},
	}

	bp.validator.AddRules(commonRules)
}

// GetConfig returns the current provider configuration
func (bp *BaseProvider) GetConfig() map[string]interface{} {
	return bp.config
}

// GetValidator returns the provider's validator instance
func (bp *BaseProvider) GetValidator() *Validator {
	return bp.validator
}
