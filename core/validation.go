package core

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// ValidationRule defines validation constraints for provider configuration fields
type ValidationRule struct {
	Field       string                  `json:"field"`
	Required    bool                    `json:"required"`
	Type        string                  `json:"type"`        // "string", "int", "bool", "float", "slice", "map"
	Pattern     string                  `json:"pattern"`     // Regex pattern for strings
	Min         interface{}             `json:"min"`         // Minimum value (for numbers) or length (for strings/slices)
	Max         interface{}             `json:"max"`         // Maximum value (for numbers) or length (for strings/slices)
	Enum        []string                `json:"enum"`        // Valid enum values
	Default     interface{}             `json:"default"`     // Default value if not provided
	Custom      func(interface{}) error `json:"-"`           // Custom validation function
	ErrorMsg    string                  `json:"error_msg"`   // Custom error message
	Suggestion  string                  `json:"suggestion"`  // Suggestion for fixing the error
	Example     string                  `json:"example"`     // Example of correct value
	Description string                  `json:"description"` // Field description
}

// FieldError represents a validation error for a specific field
type FieldError struct {
	Field      string      `json:"field"`
	Value      interface{} `json:"value"`
	Error      string      `json:"error"`
	Suggestion string      `json:"suggestion"`
	Example    string      `json:"example"`
	Line       int         `json:"line,omitempty"`   // Line number in source file
	Column     int         `json:"column,omitempty"` // Column number in source file
	Severity   string      `json:"severity"`         // "error", "warning", "info"
	Code       string      `json:"code"`             // Error code for programmatic handling
}

// ValidationResult contains the results of validating a configuration
type ValidationResult struct {
	Valid       bool         `json:"valid"`
	Errors      []FieldError `json:"errors"`
	Warnings    []FieldError `json:"warnings"`
	FixCommands []string     `json:"fix_commands,omitempty"`
}

// Validator provides configuration validation capabilities for providers
type Validator struct {
	rules        []ValidationRule
	providerName string
}

// NewValidator creates a new validator for a provider
func NewValidator(providerName string) *Validator {
	return &Validator{
		rules:        []ValidationRule{},
		providerName: providerName,
	}
}

// AddRule adds a validation rule to the validator
func (v *Validator) AddRule(rule ValidationRule) *Validator {
	v.rules = append(v.rules, rule)
	return v
}

// AddRules adds multiple validation rules to the validator
func (v *Validator) AddRules(rules []ValidationRule) *Validator {
	v.rules = append(v.rules, rules...)
	return v
}

// Validate validates a configuration map against the defined rules
func (v *Validator) Validate(config map[string]interface{}) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []FieldError{},
		Warnings: []FieldError{},
	}

	// Track which fields were validated
	validatedFields := make(map[string]bool)

	// Validate each rule
	for _, rule := range v.rules {
		fieldError := v.validateField(rule, config)
		if fieldError != nil {
			if fieldError.Severity == "error" {
				result.Valid = false
				result.Errors = append(result.Errors, *fieldError)
			} else if fieldError.Severity == "warning" {
				result.Warnings = append(result.Warnings, *fieldError)
			}
		}
		validatedFields[rule.Field] = true
	}

	// Check for unknown fields (fields not in validation rules)
	for field := range config {
		if !validatedFields[field] {
			result.Warnings = append(result.Warnings, FieldError{
				Field:      field,
				Value:      config[field],
				Error:      fmt.Sprintf("Unknown field '%s'", field),
				Suggestion: fmt.Sprintf("Remove '%s' or check %s provider documentation", field, v.providerName),
				Severity:   "warning",
				Code:       "UNKNOWN_FIELD",
			})
		}
	}

	// Generate fix commands if there are errors
	if len(result.Errors) > 0 {
		result.FixCommands = v.generateFixCommands(result.Errors)
	}

	return result
}

// validateField validates a single field against its rule
func (v *Validator) validateField(rule ValidationRule, config map[string]interface{}) *FieldError {
	value, exists := config[rule.Field]

	// Check if required field is missing
	if rule.Required && (!exists || value == nil) {
		return &FieldError{
			Field:      rule.Field,
			Value:      nil,
			Error:      fmt.Sprintf("Required field '%s' is missing", rule.Field),
			Suggestion: rule.Suggestion,
			Example:    rule.Example,
			Severity:   "error",
			Code:       "REQUIRED_FIELD_MISSING",
		}
	}

	// If field is optional and not provided, use default value
	if !exists && rule.Default != nil {
		config[rule.Field] = rule.Default
		return nil
	}

	// Skip validation if field doesn't exist and isn't required
	if !exists {
		return nil
	}

	// Type validation
	if rule.Type != "" {
		if err := v.validateType(rule, value); err != nil {
			return &FieldError{
				Field:      rule.Field,
				Value:      value,
				Error:      err.Error(),
				Suggestion: rule.Suggestion,
				Example:    rule.Example,
				Severity:   "error",
				Code:       "TYPE_MISMATCH",
			}
		}
	}

	// Pattern validation (for strings)
	if rule.Pattern != "" && rule.Type == "string" {
		if str, ok := value.(string); ok {
			matched, err := regexp.MatchString(rule.Pattern, str)
			if err != nil || !matched {
				return &FieldError{
					Field:      rule.Field,
					Value:      value,
					Error:      fmt.Sprintf("Field '%s' does not match required pattern: %s", rule.Field, rule.Pattern),
					Suggestion: rule.Suggestion,
					Example:    rule.Example,
					Severity:   "error",
					Code:       "PATTERN_MISMATCH",
				}
			}
		}
	}

	// Range validation
	if err := v.validateRange(rule, value); err != nil {
		return &FieldError{
			Field:      rule.Field,
			Value:      value,
			Error:      err.Error(),
			Suggestion: rule.Suggestion,
			Example:    rule.Example,
			Severity:   "error",
			Code:       "RANGE_VIOLATION",
		}
	}

	// Enum validation
	if len(rule.Enum) > 0 {
		if err := v.validateEnum(rule, value); err != nil {
			return &FieldError{
				Field:      rule.Field,
				Value:      value,
				Error:      err.Error(),
				Suggestion: fmt.Sprintf("Valid values are: %s", strings.Join(rule.Enum, ", ")),
				Example:    rule.Example,
				Severity:   "error",
				Code:       "INVALID_ENUM_VALUE",
			}
		}
	}

	// Custom validation
	if rule.Custom != nil {
		if err := rule.Custom(value); err != nil {
			errorMsg := err.Error()
			if rule.ErrorMsg != "" {
				errorMsg = rule.ErrorMsg
			}
			return &FieldError{
				Field:      rule.Field,
				Value:      value,
				Error:      errorMsg,
				Suggestion: rule.Suggestion,
				Example:    rule.Example,
				Severity:   "error",
				Code:       "CUSTOM_VALIDATION_FAILED",
			}
		}
	}

	return nil
}

// validateType validates the type of a field value
func (v *Validator) validateType(rule ValidationRule, value interface{}) error {
	valueType := reflect.TypeOf(value)
	if valueType == nil {
		return fmt.Errorf("field '%s' cannot be nil", rule.Field)
	}

	switch rule.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' must be a string, got %T", rule.Field, value)
		}
	case "int":
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			// Valid integer types
		case float64:
			// JSON numbers are parsed as float64, check if it's a whole number
			if f, ok := value.(float64); ok {
				if f != float64(int64(f)) {
					return fmt.Errorf("field '%s' must be an integer, got float %v", rule.Field, f)
				}
			}
		default:
			return fmt.Errorf("field '%s' must be an integer, got %T", rule.Field, value)
		}
	case "float":
		switch value.(type) {
		case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			// Valid numeric types
		default:
			return fmt.Errorf("field '%s' must be a number, got %T", rule.Field, value)
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field '%s' must be a boolean, got %T", rule.Field, value)
		}
	case "slice":
		if valueType.Kind() != reflect.Slice && valueType.Kind() != reflect.Array {
			return fmt.Errorf("field '%s' must be an array, got %T", rule.Field, value)
		}
	case "map":
		if valueType.Kind() != reflect.Map {
			return fmt.Errorf("field '%s' must be an object, got %T", rule.Field, value)
		}
	default:
		return fmt.Errorf("unknown type '%s' for field '%s'", rule.Type, rule.Field)
	}

	return nil
}

// validateRange validates min/max constraints
func (v *Validator) validateRange(rule ValidationRule, value interface{}) error {
	switch rule.Type {
	case "string":
		if str, ok := value.(string); ok {
			length := len(str)
			if rule.Min != nil {
				if min, ok := rule.Min.(int); ok && length < min {
					return fmt.Errorf("field '%s' must be at least %d characters long", rule.Field, min)
				}
			}
			if rule.Max != nil {
				if max, ok := rule.Max.(int); ok && length > max {
					return fmt.Errorf("field '%s' must be at most %d characters long", rule.Field, max)
				}
			}
		}
	case "int", "float":
		var numValue float64
		switch v := value.(type) {
		case int:
			numValue = float64(v)
		case int64:
			numValue = float64(v)
		case float64:
			numValue = v
		case float32:
			numValue = float64(v)
		default:
			return nil // Type validation should catch this
		}

		if rule.Min != nil {
			min := v.convertToFloat64(rule.Min)
			if numValue < min {
				return fmt.Errorf("field '%s' must be at least %v", rule.Field, rule.Min)
			}
		}
		if rule.Max != nil {
			max := v.convertToFloat64(rule.Max)
			if numValue > max {
				return fmt.Errorf("field '%s' must be at most %v", rule.Field, rule.Max)
			}
		}
	case "slice":
		if reflect.TypeOf(value).Kind() == reflect.Slice {
			length := reflect.ValueOf(value).Len()
			if rule.Min != nil {
				if min, ok := rule.Min.(int); ok && length < min {
					return fmt.Errorf("field '%s' must have at least %d elements", rule.Field, min)
				}
			}
			if rule.Max != nil {
				if max, ok := rule.Max.(int); ok && length > max {
					return fmt.Errorf("field '%s' must have at most %d elements", rule.Field, max)
				}
			}
		}
	}

	return nil
}

// validateEnum validates that a value is in the allowed enum values
func (v *Validator) validateEnum(rule ValidationRule, value interface{}) error {
	strValue := fmt.Sprintf("%v", value)
	for _, enumValue := range rule.Enum {
		if strValue == enumValue {
			return nil
		}
	}
	return fmt.Errorf("field '%s' has invalid value '%v'. Valid values are: %s",
		rule.Field, value, strings.Join(rule.Enum, ", "))
}

// convertToFloat64 converts various numeric types to float64
func (v *Validator) convertToFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case float64:
		return v
	case float32:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0
}

// generateFixCommands generates command suggestions for fixing validation errors
func (v *Validator) generateFixCommands(errors []FieldError) []string {
	commands := []string{}
	commandSet := make(map[string]bool)

	for _, err := range errors {
		var cmd string
		switch err.Code {
		case "TYPE_MISMATCH":
			cmd = "kolumn validate config --fix-types"
		case "PATTERN_MISMATCH":
			cmd = "kolumn validate config --fix-patterns"
		case "REQUIRED_FIELD_MISSING":
			cmd = fmt.Sprintf("# Add required field: %s", err.Field)
		case "INVALID_ENUM_VALUE":
			cmd = "kolumn validate config --fix-enums"
		default:
			cmd = "kolumn validate config --fix"
		}

		if !commandSet[cmd] {
			commands = append(commands, cmd)
			commandSet[cmd] = true
		}
	}

	return commands
}

// ValidationRuleBuilder provides a fluent interface for building validation rules
type ValidationRuleBuilder struct {
	rule ValidationRule
}

// NewValidationRule creates a new validation rule builder
func NewValidationRule(field string) *ValidationRuleBuilder {
	return &ValidationRuleBuilder{
		rule: ValidationRule{
			Field: field,
		},
	}
}

// Required marks the field as required
func (b *ValidationRuleBuilder) Required() *ValidationRuleBuilder {
	b.rule.Required = true
	return b
}

// Type sets the expected type
func (b *ValidationRuleBuilder) Type(t string) *ValidationRuleBuilder {
	b.rule.Type = t
	return b
}

// Pattern sets a regex pattern for validation
func (b *ValidationRuleBuilder) Pattern(pattern string) *ValidationRuleBuilder {
	b.rule.Pattern = pattern
	return b
}

// Min sets the minimum value or length
func (b *ValidationRuleBuilder) Min(min interface{}) *ValidationRuleBuilder {
	b.rule.Min = min
	return b
}

// Max sets the maximum value or length
func (b *ValidationRuleBuilder) Max(max interface{}) *ValidationRuleBuilder {
	b.rule.Max = max
	return b
}

// Enum sets the allowed values
func (b *ValidationRuleBuilder) Enum(values ...string) *ValidationRuleBuilder {
	b.rule.Enum = values
	return b
}

// Default sets the default value
func (b *ValidationRuleBuilder) Default(value interface{}) *ValidationRuleBuilder {
	b.rule.Default = value
	return b
}

// Custom adds a custom validation function
func (b *ValidationRuleBuilder) Custom(fn func(interface{}) error) *ValidationRuleBuilder {
	b.rule.Custom = fn
	return b
}

// ErrorMessage sets a custom error message
func (b *ValidationRuleBuilder) ErrorMessage(msg string) *ValidationRuleBuilder {
	b.rule.ErrorMsg = msg
	return b
}

// Suggestion sets a suggestion for fixing errors
func (b *ValidationRuleBuilder) Suggestion(suggestion string) *ValidationRuleBuilder {
	b.rule.Suggestion = suggestion
	return b
}

// Example sets an example value
func (b *ValidationRuleBuilder) Example(example string) *ValidationRuleBuilder {
	b.rule.Example = example
	return b
}

// Description sets the field description
func (b *ValidationRuleBuilder) Description(desc string) *ValidationRuleBuilder {
	b.rule.Description = desc
	return b
}

// Build returns the constructed validation rule
func (b *ValidationRuleBuilder) Build() ValidationRule {
	return b.rule
}

// Common validation helpers

// ValidateHost validates a hostname or IP address
func ValidateHost(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("host must be a string")
	}

	if str == "" {
		return fmt.Errorf("host cannot be empty")
	}

	// Basic hostname/IP validation
	hostPattern := `^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`
	matched, err := regexp.MatchString(hostPattern, str)
	if err != nil || !matched {
		// Try IP address pattern
		ipPattern := `^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`
		matched, err = regexp.MatchString(ipPattern, str)
		if err != nil || !matched {
			return fmt.Errorf("invalid hostname or IP address")
		}
	}

	return nil
}

// ValidatePort validates a network port number
func ValidatePort(value interface{}) error {
	var port int64

	switch v := value.(type) {
	case int:
		port = int64(v)
	case int64:
		port = v
	case float64:
		port = int64(v)
	default:
		return fmt.Errorf("port must be a number")
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	return nil
}

// ValidateDatabaseName validates a database name
func ValidateDatabaseName(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("database name must be a string")
	}

	if str == "" {
		return fmt.Errorf("database name cannot be empty")
	}

	// Basic database name validation (alphanumeric, underscore, hyphen)
	pattern := `^[a-zA-Z0-9_-]+$`
	matched, err := regexp.MatchString(pattern, str)
	if err != nil || !matched {
		return fmt.Errorf("database name can only contain letters, numbers, underscores, and hyphens")
	}

	return nil
}
