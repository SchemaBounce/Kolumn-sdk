// Package validation provides common validation rules for Kolumn providers
//
// This package contains reusable validation functions that provider developers
// can use to validate configuration values and inputs.
package validation

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation failed: %s", e.Message)
}

// ValidationRule represents a validation rule with metadata
type ValidationRule struct {
	Name        string
	Description string
	Validator   ValidationFunc
}

// ValidationIssue represents a validation issue
type ValidationIssue struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ValidationFunc is a function that validates a value
type ValidationFunc func(value interface{}, field string) error

// OptionalString creates a validator that allows empty strings
func OptionalString() ValidationFunc {
	return func(value interface{}, field string) error {
		if value == nil {
			return nil
		}
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%s must be a string", field)
		}
		return nil
	}
}

// RequiredSlice creates a validator that requires a non-empty slice
func RequiredSlice() ValidationFunc {
	return func(value interface{}, field string) error {
		if value == nil {
			return fmt.Errorf("%s is required", field)
		}
		// Check if it's a slice type
		switch v := value.(type) {
		case []interface{}:
			if len(v) == 0 {
				return fmt.Errorf("%s cannot be empty", field)
			}
		case []string:
			if len(v) == 0 {
				return fmt.Errorf("%s cannot be empty", field)
			}
		case []map[string]interface{}:
			if len(v) == 0 {
				return fmt.Errorf("%s cannot be empty", field)
			}
		default:
			return fmt.Errorf("%s must be a slice/array", field)
		}
		return nil
	}
}

// OptionalSlice creates a validator that allows empty slices
func OptionalSlice() ValidationFunc {
	return func(value interface{}, field string) error {
		if value == nil {
			return nil
		}
		// Check if it's a slice type
		switch value.(type) {
		case []interface{}, []string, []map[string]interface{}:
			return nil
		default:
			return fmt.Errorf("%s must be a slice/array", field)
		}
	}
}

// Compose multiple validation functions into one
func Compose(validators ...ValidationFunc) ValidationFunc {
	return func(value interface{}, field string) error {
		for _, validator := range validators {
			if err := validator(value, field); err != nil {
				return err
			}
		}
		return nil
	}
}

// String Validation

// NotEmpty validates that a string is not empty
func NotEmpty() ValidationFunc {
	return func(value interface{}, field string) error {
		str, ok := value.(string)
		if !ok {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "must be a string",
			}
		}

		if strings.TrimSpace(str) == "" {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "cannot be empty",
			}
		}

		return nil
	}
}

// MinLength validates minimum string length
func MinLength(min int) ValidationFunc {
	return func(value interface{}, field string) error {
		str, ok := value.(string)
		if !ok {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "must be a string",
			}
		}

		if len(str) < min {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must be at least %d characters", min),
			}
		}

		return nil
	}
}

// MaxLength validates maximum string length
func MaxLength(max int) ValidationFunc {
	return func(value interface{}, field string) error {
		str, ok := value.(string)
		if !ok {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "must be a string",
			}
		}

		if len(str) > max {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must be no more than %d characters", max),
			}
		}

		return nil
	}
}

// MatchPattern validates that a string matches a regex pattern
func MatchPattern(pattern string, description string) ValidationFunc {
	regex := regexp.MustCompile(pattern)

	return func(value interface{}, field string) error {
		str, ok := value.(string)
		if !ok {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "must be a string",
			}
		}

		if !regex.MatchString(str) {
			msg := fmt.Sprintf("must match pattern %s", pattern)
			if description != "" {
				msg = fmt.Sprintf("must be %s (pattern: %s)", description, pattern)
			}

			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: msg,
			}
		}

		return nil
	}
}

// IsInList validates that a value is in a predefined list
func IsInList(validValues []string) ValidationFunc {
	return func(value interface{}, field string) error {
		str, ok := value.(string)
		if !ok {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "must be a string",
			}
		}

		for _, valid := range validValues {
			if str == valid {
				return nil
			}
		}

		return &ValidationError{
			Field:   field,
			Value:   value,
			Message: fmt.Sprintf("must be one of: %s", strings.Join(validValues, ", ")),
		}
	}
}

// IsValidURL validates that a string is a valid URL
func IsValidURL() ValidationFunc {
	return func(value interface{}, field string) error {
		str, ok := value.(string)
		if !ok {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "must be a string",
			}
		}

		_, err := url.Parse(str)
		if err != nil {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must be a valid URL: %v", err),
			}
		}

		return nil
	}
}

// IsValidEmail validates basic email format
func IsValidEmail() ValidationFunc {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	return func(value interface{}, field string) error {
		str, ok := value.(string)
		if !ok {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "must be a string",
			}
		}

		if !emailRegex.MatchString(str) {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "must be a valid email address",
			}
		}

		return nil
	}
}

// Numeric Validation

// IsPositive validates that a number is positive
func IsPositive() ValidationFunc {
	return func(value interface{}, field string) error {
		var num float64
		var ok bool

		switch v := value.(type) {
		case int:
			num = float64(v)
			ok = true
		case int64:
			num = float64(v)
			ok = true
		case float64:
			num = v
			ok = true
		case string:
			var err error
			num, err = strconv.ParseFloat(v, 64)
			ok = err == nil
		}

		if !ok {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "must be a number",
			}
		}

		if num <= 0 {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "must be positive",
			}
		}

		return nil
	}
}

// InRange validates that a number is within a specified range
func InRange(min, max float64) ValidationFunc {
	return func(value interface{}, field string) error {
		var num float64
		var ok bool

		switch v := value.(type) {
		case int:
			num = float64(v)
			ok = true
		case int64:
			num = float64(v)
			ok = true
		case float64:
			num = v
			ok = true
		case string:
			var err error
			num, err = strconv.ParseFloat(v, 64)
			ok = err == nil
		}

		if !ok {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "must be a number",
			}
		}

		if num < min || num > max {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must be between %g and %g", min, max),
			}
		}

		return nil
	}
}

// Network Validation

// IsValidIPAddress validates IPv4 or IPv6 address
func IsValidIPAddress() ValidationFunc {
	return func(value interface{}, field string) error {
		str, ok := value.(string)
		if !ok {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "must be a string",
			}
		}

		ip := net.ParseIP(str)
		if ip == nil {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "must be a valid IP address",
			}
		}

		return nil
	}
}

// IsValidPort validates port number (1-65535)
func IsValidPort() ValidationFunc {
	return Compose(
		InRange(1, 65535),
	)
}

// IsValidHostPort validates host:port format
func IsValidHostPort() ValidationFunc {
	return func(value interface{}, field string) error {
		str, ok := value.(string)
		if !ok {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "must be a string",
			}
		}

		host, port, err := net.SplitHostPort(str)
		if err != nil {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must be in host:port format: %v", err),
			}
		}

		// Validate port
		if err := IsValidPort()(port, "port"); err != nil {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("invalid port: %v", err),
			}
		}

		// Validate host (can be IP or hostname)
		if host != "" {
			if net.ParseIP(host) == nil {
				// Not an IP, validate as hostname
				if err := IsValidHostname()(host, "host"); err != nil {
					return &ValidationError{
						Field:   field,
						Value:   value,
						Message: fmt.Sprintf("invalid host: %v", err),
					}
				}
			}
		}

		return nil
	}
}

// IsValidHostname validates hostname format
func IsValidHostname() ValidationFunc {
	hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)

	return func(value interface{}, field string) error {
		str, ok := value.(string)
		if !ok {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "must be a string",
			}
		}

		if len(str) > 253 {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "hostname too long (max 253 characters)",
			}
		}

		if !hostnameRegex.MatchString(str) {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "must be a valid hostname",
			}
		}

		return nil
	}
}

// Time Validation

// IsValidDuration validates duration strings (like "30s", "5m", "1h")
func IsValidDuration() ValidationFunc {
	return func(value interface{}, field string) error {
		str, ok := value.(string)
		if !ok {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "must be a string",
			}
		}

		_, err := time.ParseDuration(str)
		if err != nil {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must be a valid duration (e.g., '30s', '5m', '1h'): %v", err),
			}
		}

		return nil
	}
}

// DurationInRange validates that a duration is within a specified range
func DurationInRange(min, max time.Duration) ValidationFunc {
	return func(value interface{}, field string) error {
		str, ok := value.(string)
		if !ok {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "must be a string",
			}
		}

		duration, err := time.ParseDuration(str)
		if err != nil {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must be a valid duration: %v", err),
			}
		}

		if duration < min || duration > max {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("duration must be between %v and %v", min, max),
			}
		}

		return nil
	}
}

// Database-specific Validation

// IsValidDatabaseName validates database naming conventions
func IsValidDatabaseName() ValidationFunc {
	// Most databases allow alphanumeric + underscore, starting with letter or underscore
	dbNameRegex := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

	return Compose(
		NotEmpty(),
		MaxLength(63), // PostgreSQL limit
		MatchPattern(dbNameRegex.String(), "a valid database name"),
	)
}

// IsValidTableName validates table naming conventions
func IsValidTableName() ValidationFunc {
	return Compose(
		NotEmpty(),
		MaxLength(63), // PostgreSQL limit
		MatchPattern(`^[a-zA-Z_][a-zA-Z0-9_]*$`, "a valid table name"),
	)
}

// IsValidColumnName validates column naming conventions
func IsValidColumnName() ValidationFunc {
	return Compose(
		NotEmpty(),
		MaxLength(63), // PostgreSQL limit
		MatchPattern(`^[a-zA-Z_][a-zA-Z0-9_]*$`, "a valid column name"),
	)
}

// Cloud-specific Validation

// IsValidS3BucketName validates S3 bucket naming rules
func IsValidS3BucketName() ValidationFunc {
	return func(value interface{}, field string) error {
		str, ok := value.(string)
		if !ok {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "must be a string",
			}
		}

		// S3 bucket name rules
		if len(str) < 3 || len(str) > 63 {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "S3 bucket name must be between 3 and 63 characters",
			}
		}

		// Must start and end with lowercase letter or number
		if !regexp.MustCompile(`^[a-z0-9].*[a-z0-9]$`).MatchString(str) {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "S3 bucket name must start and end with lowercase letter or number",
			}
		}

		// Can contain lowercase letters, numbers, hyphens, and periods
		if !regexp.MustCompile(`^[a-z0-9.-]+$`).MatchString(str) {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "S3 bucket name can only contain lowercase letters, numbers, hyphens, and periods",
			}
		}

		// Must not contain consecutive periods
		if strings.Contains(str, "..") {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "S3 bucket name must not contain consecutive periods",
			}
		}

		// Must not be formatted as IP address
		if net.ParseIP(str) != nil {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: "S3 bucket name must not be formatted as IP address",
			}
		}

		return nil
	}
}

// IsValidAWSRegion validates AWS region format
func IsValidAWSRegion() ValidationFunc {
	awsRegions := []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1", "eu-north-1",
		"ap-south-1", "ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ap-northeast-2",
		"ca-central-1", "sa-east-1", "af-south-1", "ap-east-1", "me-south-1",
		"us-gov-east-1", "us-gov-west-1",
	}

	return IsInList(awsRegions)
}

// Conditional Validation

// RequiredIf validates that a field is present if a condition is met
func RequiredIf(conditionField string, conditionValue interface{}) func(config map[string]interface{}) ValidationFunc {
	return func(config map[string]interface{}) ValidationFunc {
		return func(value interface{}, field string) error {
			// Check condition
			if configValue, exists := config[conditionField]; exists && configValue == conditionValue {
				// Condition is met, field is required
				if value == nil || value == "" {
					return &ValidationError{
						Field:   field,
						Value:   value,
						Message: fmt.Sprintf("is required when %s is %v", conditionField, conditionValue),
					}
				}
			}
			return nil
		}
	}
}

// ExclusiveWith validates that only one of multiple fields is set
func ExclusiveWith(otherFields ...string) func(config map[string]interface{}) ValidationFunc {
	return func(config map[string]interface{}) ValidationFunc {
		return func(value interface{}, field string) error {
			if value == nil || value == "" {
				return nil // Field is not set, no conflict
			}

			// Check if any other exclusive fields are set
			for _, otherField := range otherFields {
				if otherValue, exists := config[otherField]; exists && otherValue != nil && otherValue != "" {
					return &ValidationError{
						Field:   field,
						Value:   value,
						Message: fmt.Sprintf("cannot be set when %s is also set", otherField),
					}
				}
			}

			return nil
		}
	}
}
