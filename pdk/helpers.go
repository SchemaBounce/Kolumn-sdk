// Package pdk provides the Provider Development Kit for the Kolumn SDK
package pdk

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/schemabounce/kolumn/sdk/types"
)

// Helper functions for common provider operations

// ParseConfig parses a generic configuration into a typed struct
func ParseConfig(config map[string]interface{}, target interface{}) error {
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := json.Unmarshal(configBytes, target); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	return nil
}

// ValidateRequired validates that required fields are present in config
func ValidateRequired(config map[string]interface{}, required []string) error {
	var missing []string

	for _, field := range required {
		if value, exists := config[field]; !exists || value == nil || value == "" {
			missing = append(missing, field)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))
	}

	return nil
}

// GetString safely gets a string value from config with default
func GetString(config map[string]interface{}, key string, defaultValue string) string {
	if value, exists := config[key]; exists {
		if str, ok := value.(string); ok && str != "" {
			return str
		}
	}
	return defaultValue
}

// GetInt safely gets an int value from config with default
func GetInt(config map[string]interface{}, key string, defaultValue int) int {
	if value, exists := config[key]; exists {
		switch v := value.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		case string:
			// Try to parse string as int
			if i, err := parseStringToInt(v); err == nil {
				return i
			}
		}
	}
	return defaultValue
}

// GetBool safely gets a bool value from config with default
func GetBool(config map[string]interface{}, key string, defaultValue bool) bool {
	if value, exists := config[key]; exists {
		if b, ok := value.(bool); ok {
			return b
		}
		// Try to parse string as bool
		if str, ok := value.(string); ok {
			switch strings.ToLower(str) {
			case "true", "yes", "1", "on":
				return true
			case "false", "no", "0", "off":
				return false
			}
		}
	}
	return defaultValue
}

// GetDuration safely gets a duration value from config with default
func GetDuration(config map[string]interface{}, key string, defaultValue time.Duration) time.Duration {
	if value, exists := config[key]; exists {
		switch v := value.(type) {
		case time.Duration:
			return v
		case string:
			if d, err := time.ParseDuration(v); err == nil {
				return d
			}
		case int:
			return time.Duration(v) * time.Second
		case int64:
			return time.Duration(v) * time.Second
		case float64:
			return time.Duration(v) * time.Second
		}
	}
	return defaultValue
}

// GetStringSlice safely gets a string slice from config with default
func GetStringSlice(config map[string]interface{}, key string, defaultValue []string) []string {
	if value, exists := config[key]; exists {
		switch v := value.(type) {
		case []string:
			return v
		case []interface{}:
			var result []string
			for _, item := range v {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		case string:
			// Split comma-separated string
			return strings.Split(v, ",")
		}
	}
	return defaultValue
}

// CreateProviderSchema creates a basic provider schema
func CreateProviderSchema(name, version string) *types.ProviderSchema {
	return &types.ProviderSchema{
		Provider: types.ProviderSpec{
			Name:    name,
			Version: version,
		},
		Functions:     make(map[string]types.FunctionSpec),
		ResourceTypes: make([]string, 0),
		Capabilities:  make([]string, 0),
	}
}

// AddFunction adds a function to a provider schema
func AddFunction(schema *types.ProviderSchema, name, description string, idempotent bool) {
	schema.Functions[name] = types.FunctionSpec{
		Description: description,
		Idempotent:  idempotent,
	}
}

// AddUniversalFunctions adds standard universal functions to a provider schema
func AddUniversalFunctions(schema *types.ProviderSchema) {
	universalFunctions := map[string]types.FunctionSpec{
		"ping": {
			Description: "Health check - returns provider status",
			Idempotent:  true,
		},
		"get_version": {
			Description: "Returns provider version information",
			Idempotent:  true,
		},
		"health_check": {
			Description: "Comprehensive health check with detailed status",
			Idempotent:  true,
		},
		"get_metrics": {
			Description: "Returns provider performance and usage metrics",
			Idempotent:  true,
		},
		"validate_config": {
			Description: "Validates provider configuration",
			Idempotent:  true,
		},
		"get_capabilities": {
			Description: "Returns provider capabilities and supported features",
			Idempotent:  true,
		},
	}

	for name, spec := range universalFunctions {
		schema.Functions[name] = spec
	}
}

// CreateErrorResponse creates a standard error response
func CreateErrorResponse(message string, details ...string) (json.RawMessage, error) {
	errorResp := map[string]interface{}{
		"error":   true,
		"message": message,
	}

	if len(details) > 0 {
		errorResp["details"] = strings.Join(details, "; ")
	}

	errorResp["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	return json.Marshal(errorResp)
}

// CreateSuccessResponse creates a standard success response
func CreateSuccessResponse(data interface{}) (json.RawMessage, error) {
	successResp := map[string]interface{}{
		"success":   true,
		"data":      data,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	return json.Marshal(successResp)
}

// CreatePingResponse creates a standard ping response
func CreatePingResponse(status string, latencyMs int64, metadata map[string]interface{}) (json.RawMessage, error) {
	pingResp := map[string]interface{}{
		"status":     status,
		"latency_ms": latencyMs,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}

	if metadata != nil {
		pingResp["metadata"] = metadata
	}

	return json.Marshal(pingResp)
}

// ValidateEntityName validates entity names according to common rules
func ValidateEntityName(name string) error {
	if name == "" {
		return fmt.Errorf("entity name cannot be empty")
	}

	if len(name) > 255 {
		return fmt.Errorf("entity name too long (max 255 characters)")
	}

	// Common naming pattern: letters, numbers, underscores, hyphens
	validName := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("entity name must start with letter and contain only letters, numbers, underscores, and hyphens")
	}

	return nil
}

// SanitizeEntityName sanitizes an entity name to make it valid
func SanitizeEntityName(name string) string {
	// Remove invalid characters
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	sanitized := reg.ReplaceAllString(name, "_")

	// Ensure it starts with a letter
	if len(sanitized) > 0 && !regexp.MustCompile(`^[a-zA-Z]`).MatchString(sanitized) {
		sanitized = "entity_" + sanitized
	}

	// Truncate if too long
	if len(sanitized) > 255 {
		sanitized = sanitized[:255]
	}

	return sanitized
}

// MergeMetadata merges multiple metadata maps with later ones taking precedence
func MergeMetadata(metadataMaps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for _, metadata := range metadataMaps {
		for key, value := range metadata {
			result[key] = value
		}
	}

	return result
}

// GetContextTimeout gets timeout from context or returns default
func GetContextTimeout(ctx context.Context, defaultTimeout time.Duration) time.Duration {
	if deadline, ok := ctx.Deadline(); ok {
		timeout := time.Until(deadline)
		if timeout > 0 {
			return timeout
		}
	}
	return defaultTimeout
}

// RetryWithBackoff executes a function with exponential backoff retry
func RetryWithBackoff(ctx context.Context, maxRetries int, initialDelay time.Duration, fn func() error) error {
	var lastErr error
	delay := initialDelay

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue with retry
				delay *= 2 // Exponential backoff
			}
		}

		if err := fn(); err != nil {
			lastErr = err
			if attempt < maxRetries {
				continue
			}
		} else {
			return nil // Success
		}
	}

	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, lastErr)
}

// Helper function to parse string to int
func parseStringToInt(s string) (int, error) {
	switch s {
	case "0":
		return 0, nil
	case "1":
		return 1, nil
	default:
		return 0, fmt.Errorf("invalid int string")
	}
}
