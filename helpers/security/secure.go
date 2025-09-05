// Package security provides security utilities for the Kolumn Provider SDK
//
// This package contains secure implementations of common operations to prevent
// injection attacks, DoS attacks, and other security vulnerabilities.
package security

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Security constants
const (
	// Maximum JSON payload size (1MB)
	MaxJSONSize = 1024 * 1024

	// Maximum JSON nesting depth
	MaxJSONDepth = 10

	// Maximum string length in JSON
	MaxStringLength = 65536

	// Maximum array/object items
	MaxArrayItems = 1000

	// Maximum error message length
	MaxErrorMessageLength = 500
)

var (
	// Common security errors
	ErrInputTooLarge = errors.New("input payload too large")
	ErrInputTooDeep  = errors.New("input nesting too deep")
	ErrInvalidMethod = errors.New("invalid method name")
	ErrInvalidInput  = errors.New("invalid input format")
	ErrStringTooLong = errors.New("string value too long")
	ErrTooManyItems  = errors.New("too many items in array/object")
)

// AllowedMethods defines the whitelist of allowed method names
var AllowedMethods = map[string]bool{
	// CREATE object methods
	"create": true,
	"read":   true,
	"update": true,
	"delete": true,
	"plan":   true,

	// DISCOVER object methods
	"scan":    true,
	"analyze": true,
	"query":   true,
}

// ValidateMethod validates that a method name is allowed
func ValidateMethod(method string) error {
	if method == "" {
		return errors.New("method name cannot be empty")
	}

	if !AllowedMethods[method] {
		return ErrInvalidMethod
	}

	return nil
}

// SafeUnmarshal safely unmarshals JSON with size and depth limits
func SafeUnmarshal(input []byte, v interface{}) error {
	if len(input) > MaxJSONSize {
		return ErrInputTooLarge
	}

	if len(input) == 0 {
		return ErrInvalidInput
	}

	// Create decoder with security settings
	decoder := json.NewDecoder(bytes.NewReader(input))

	// Prevent unknown fields to avoid injection
	decoder.DisallowUnknownFields()

	// First pass: validate structure without unmarshaling
	if err := validateJSONStructure(input); err != nil {
		return err
	}

	// Second pass: actual unmarshaling
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("JSON decode error: %w", err)
	}

	return nil
}

// validateJSONStructure validates JSON structure for security
func validateJSONStructure(input []byte) error {
	var raw interface{}
	if err := json.Unmarshal(input, &raw); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return validateDepthAndSize(raw, 0)
}

// validateDepthAndSize recursively validates JSON depth and size
func validateDepthAndSize(value interface{}, depth int) error {
	if depth > MaxJSONDepth {
		return ErrInputTooDeep
	}

	switch v := value.(type) {
	case string:
		if len(v) > MaxStringLength {
			return ErrStringTooLong
		}

	case []interface{}:
		if len(v) > MaxArrayItems {
			return ErrTooManyItems
		}
		for _, item := range v {
			if err := validateDepthAndSize(item, depth+1); err != nil {
				return err
			}
		}

	case map[string]interface{}:
		if len(v) > MaxArrayItems {
			return ErrTooManyItems
		}
		for _, item := range v {
			if err := validateDepthAndSize(item, depth+1); err != nil {
				return err
			}
		}
	}

	return nil
}

// SafeTypeCastBool safely casts interface{} to bool
func SafeTypeCastBool(value interface{}) (bool, bool) {
	if b, ok := value.(bool); ok {
		return b, true
	}
	return false, false
}

// SafeTypeCastString safely casts interface{} to string
func SafeTypeCastString(value interface{}) (string, bool) {
	if s, ok := value.(string); ok {
		if len(s) > MaxStringLength {
			return "", false // String too long
		}
		return s, true
	}
	return "", false
}

// SafeTypeCastInt safely casts interface{} to int
func SafeTypeCastInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float64:
		// Allow conversion from float64 if it's a whole number
		if v == float64(int(v)) {
			return int(v), true
		}
	}
	return 0, false
}

// SafeTypeCastFloat safely casts interface{} to float64
func SafeTypeCastFloat(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	}
	return 0.0, false
}

// SafeTypeCastMap safely casts interface{} to map[string]interface{}
func SafeTypeCastMap(value interface{}) (map[string]interface{}, bool) {
	if m, ok := value.(map[string]interface{}); ok {
		if len(m) > MaxArrayItems {
			return nil, false // Map too large
		}
		return m, true
	}
	return nil, false
}

// SafeTypeCastSlice safely casts interface{} to []interface{}
func SafeTypeCastSlice(value interface{}) ([]interface{}, bool) {
	if s, ok := value.([]interface{}); ok {
		if len(s) > MaxArrayItems {
			return nil, false // Slice too large
		}
		return s, true
	}
	return nil, false
}

// SanitizeErrorMessage sanitizes error messages to prevent information disclosure
func SanitizeErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	message := err.Error()

	// Truncate long messages
	if len(message) > MaxErrorMessageLength {
		message = message[:MaxErrorMessageLength] + "..."
	}

	// Remove sensitive patterns
	message = sanitizeSensitiveInfo(message)

	return message
}

// sanitizeSensitiveInfo removes sensitive information from strings
func sanitizeSensitiveInfo(message string) string {
	// Remove file paths
	if strings.Contains(message, "/") || strings.Contains(message, "\\") {
		message = "filesystem operation failed"
	}

	// Remove SQL-like patterns
	if strings.Contains(strings.ToLower(message), "sql") ||
		strings.Contains(strings.ToLower(message), "table") ||
		strings.Contains(strings.ToLower(message), "column") {
		message = "database operation failed"
	}

	// Remove stack traces
	if strings.Contains(message, "goroutine") || strings.Contains(message, ".go:") {
		message = "internal processing error"
	}

	// Remove internal handler/registry information
	if strings.Contains(message, "handler") || strings.Contains(message, "registry") {
		message = "operation not supported"
	}

	return message
}

// ValidateObjectType validates object type names
func ValidateObjectType(objectType string) error {
	if objectType == "" {
		return errors.New("object type cannot be empty")
	}

	if len(objectType) > 100 {
		return errors.New("object type name too long")
	}

	// Check for dangerous patterns that could cause security issues
	dangerousPatterns := []string{
		"__proto__", "constructor", "prototype",
		"eval", "function", "class", "import", "require",
		"process", "global", "window", "document",
		"..", "/", "\\", // Path traversal
	}

	lowerObjectType := strings.ToLower(objectType)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerObjectType, pattern) {
			return errors.New("object type contains prohibited pattern")
		}
	}

	// Only allow alphanumeric, underscore, and hyphen
	for _, r := range objectType {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-') {
			return errors.New("object type contains invalid characters")
		}
	}

	return nil
}

// SecureError creates a sanitized error with limited information disclosure
type SecureError struct {
	UserMessage     string
	InternalMessage string
	Code            string
}

func (e *SecureError) Error() string {
	return e.UserMessage
}

// Internal returns the internal error message for logging
func (e *SecureError) Internal() string {
	return e.InternalMessage
}

// NewSecureError creates a new secure error
func NewSecureError(userMsg, internalMsg, code string) *SecureError {
	return &SecureError{
		UserMessage:     SanitizeErrorMessage(errors.New(userMsg)),
		InternalMessage: internalMsg,
		Code:            code,
	}
}

// InputSizeValidator validates input sizes across the board
type InputSizeValidator struct{}

// ValidateConfigSize validates configuration map size
func (v *InputSizeValidator) ValidateConfigSize(config map[string]interface{}) error {
	if len(config) > MaxArrayItems {
		return ErrTooManyItems
	}

	for key, value := range config {
		if len(key) > MaxStringLength {
			return ErrStringTooLong
		}

		if err := validateDepthAndSize(value, 0); err != nil {
			return err
		}
	}

	return nil
}

// RateLimiter provides basic rate limiting functionality
type RateLimiter struct {
	requests map[string]int
	limit    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string]int),
		limit:    limit,
	}
}

// Allow checks if a request is allowed for the given client
func (r *RateLimiter) Allow(clientID string) bool {
	if r.requests[clientID] >= r.limit {
		return false
	}
	r.requests[clientID]++
	return true
}

// Reset resets the counter for a client
func (r *RateLimiter) Reset(clientID string) {
	delete(r.requests, clientID)
}
