// Package validation provides schema validation helpers for Kolumn providers
package validation

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/schemabounce/kolumn/sdk/core"
)

// SchemaValidator validates configurations against Kolumn schemas
type SchemaValidator struct {
	schema *core.Schema
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator(schema *core.Schema) *SchemaValidator {
	return &SchemaValidator{schema: schema}
}

// ValidateProviderConfig validates provider configuration
func (v *SchemaValidator) ValidateProviderConfig(config map[string]interface{}) error {
	if v.schema.ConfigSchema == nil {
		return nil // No schema to validate against
	}

	return v.validateObjectConfig(config, v.schema.ConfigSchema.Properties, v.schema.ConfigSchema.Required)
}

// ValidateObjectConfig validates configuration for a specific object type
func (v *SchemaValidator) ValidateObjectConfig(objectType string, config map[string]interface{}) error {
	// Check CREATE objects
	if obj, exists := v.schema.CreateObjects[objectType]; exists {
		return v.validateObjectConfig(config, obj.Properties, obj.Required)
	}

	// Check DISCOVER objects
	if obj, exists := v.schema.DiscoverObjects[objectType]; exists {
		return v.validateObjectConfig(config, obj.Properties, obj.Required)
	}

	return fmt.Errorf("unknown object type: %s", objectType)
}

// validateObjectConfig validates configuration against property definitions
func (v *SchemaValidator) validateObjectConfig(config map[string]interface{}, properties map[string]*core.Property, required []string) error {
	// Validate required fields
	for _, field := range required {
		if _, exists := config[field]; !exists {
			return &ValidationError{
				Field:   field,
				Message: "is required",
			}
		}
	}

	// Validate each provided field
	for field, value := range config {
		if prop, exists := properties[field]; exists {
			if err := v.validateProperty(value, prop, field); err != nil {
				return err
			}
		} else {
			// Field not defined in schema - could be warning or error based on policy
			// For now, we'll be permissive and allow unknown fields
		}
	}

	return nil
}

// validateProperty validates a single property value
func (v *SchemaValidator) validateProperty(value interface{}, prop *core.Property, field string) error {
	// Handle nil values
	if value == nil {
		// Nil is allowed unless field is required (checked elsewhere)
		return nil
	}

	// Validate based on property type
	switch prop.Type {
	case "string":
		if err := v.validateStringProperty(value, prop, field); err != nil {
			return err
		}

	case "integer":
		if err := v.validateIntegerProperty(value, prop, field); err != nil {
			return err
		}

	case "number":
		if err := v.validateNumberProperty(value, prop, field); err != nil {
			return err
		}

	case "boolean":
		if err := v.validateBooleanProperty(value, field); err != nil {
			return err
		}

	case "list":
		if err := v.validateListProperty(value, field); err != nil {
			return err
		}

	case "object":
		if err := v.validateObjectProperty(value, field); err != nil {
			return err
		}

	default:
		return &ValidationError{
			Field:   field,
			Value:   value,
			Message: fmt.Sprintf("unknown property type: %s", prop.Type),
		}
	}

	// Apply validation rules if present
	if prop.Validation != nil {
		if err := v.applyValidationRules(value, prop.Validation, field); err != nil {
			return err
		}
	}

	return nil
}

// validateStringProperty validates string properties
func (v *SchemaValidator) validateStringProperty(value interface{}, prop *core.Property, field string) error {
	str, ok := value.(string)
	if !ok {
		return &ValidationError{
			Field:   field,
			Value:   value,
			Message: "must be a string",
		}
	}

	// Apply string-specific validations
	if prop.Validation != nil {
		if prop.Validation.MinLength != nil && len(str) < *prop.Validation.MinLength {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must be at least %d characters", *prop.Validation.MinLength),
			}
		}

		if prop.Validation.MaxLength != nil && len(str) > *prop.Validation.MaxLength {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must be no more than %d characters", *prop.Validation.MaxLength),
			}
		}

		if prop.Validation.Pattern != "" {
			validator := MatchPattern(prop.Validation.Pattern, "")
			if err := validator(value, field); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateIntegerProperty validates integer properties
func (v *SchemaValidator) validateIntegerProperty(value interface{}, prop *core.Property, field string) error {
	var intVal int64
	var ok bool

	switch v := value.(type) {
	case int:
		intVal = int64(v)
		ok = true
	case int32:
		intVal = int64(v)
		ok = true
	case int64:
		intVal = v
		ok = true
	case float64:
		// Allow conversion from float64 if it's a whole number
		if v == float64(int64(v)) {
			intVal = int64(v)
			ok = true
		}
	}

	if !ok {
		return &ValidationError{
			Field:   field,
			Value:   value,
			Message: "must be an integer",
		}
	}

	// Apply numeric validations
	if prop.Validation != nil {
		floatVal := float64(intVal)
		if prop.Validation.Minimum != nil && floatVal < *prop.Validation.Minimum {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must be at least %g", *prop.Validation.Minimum),
			}
		}

		if prop.Validation.Maximum != nil && floatVal > *prop.Validation.Maximum {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must be no more than %g", *prop.Validation.Maximum),
			}
		}
	}

	return nil
}

// validateNumberProperty validates number properties
func (v *SchemaValidator) validateNumberProperty(value interface{}, prop *core.Property, field string) error {
	var numVal float64
	var ok bool

	switch v := value.(type) {
	case int:
		numVal = float64(v)
		ok = true
	case int32:
		numVal = float64(v)
		ok = true
	case int64:
		numVal = float64(v)
		ok = true
	case float32:
		numVal = float64(v)
		ok = true
	case float64:
		numVal = v
		ok = true
	}

	if !ok {
		return &ValidationError{
			Field:   field,
			Value:   value,
			Message: "must be a number",
		}
	}

	// Apply numeric validations
	if prop.Validation != nil {
		if prop.Validation.Minimum != nil && numVal < *prop.Validation.Minimum {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must be at least %g", *prop.Validation.Minimum),
			}
		}

		if prop.Validation.Maximum != nil && numVal > *prop.Validation.Maximum {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must be no more than %g", *prop.Validation.Maximum),
			}
		}
	}

	return nil
}

// validateBooleanProperty validates boolean properties
func (v *SchemaValidator) validateBooleanProperty(value interface{}, field string) error {
	if _, ok := value.(bool); !ok {
		return &ValidationError{
			Field:   field,
			Value:   value,
			Message: "must be a boolean",
		}
	}

	return nil
}

// validateListProperty validates list properties
func (v *SchemaValidator) validateListProperty(value interface{}, field string) error {
	// Use reflection to check if value is a slice or array
	valueType := reflect.TypeOf(value)
	if valueType.Kind() != reflect.Slice && valueType.Kind() != reflect.Array {
		return &ValidationError{
			Field:   field,
			Value:   value,
			Message: "must be a list",
		}
	}

	return nil
}

// validateObjectProperty validates object properties
func (v *SchemaValidator) validateObjectProperty(value interface{}, field string) error {
	if _, ok := value.(map[string]interface{}); !ok {
		return &ValidationError{
			Field:   field,
			Value:   value,
			Message: "must be an object",
		}
	}

	return nil
}

// applyValidationRules applies validation rules from the property definition
func (v *SchemaValidator) applyValidationRules(value interface{}, validation *core.Validation, field string) error {
	// Check enum values
	if len(validation.Enum) > 0 {
		found := false
		for _, enumValue := range validation.Enum {
			if reflect.DeepEqual(value, enumValue) {
				found = true
				break
			}
		}

		if !found {
			return &ValidationError{
				Field:   field,
				Value:   value,
				Message: fmt.Sprintf("must be one of: %v", validation.Enum),
			}
		}
	}

	return nil
}

// ValidateSchema validates that a schema itself is well-formed
func ValidateSchema(schema *core.Schema) error {
	if schema == nil {
		return fmt.Errorf("schema is nil")
	}

	// Validate basic fields
	if schema.Name == "" {
		return fmt.Errorf("schema name is required")
	}

	if schema.Version == "" {
		return fmt.Errorf("schema version is required")
	}

	// Validate CREATE objects
	for name, obj := range schema.CreateObjects {
		if err := validateObjectType(obj, fmt.Sprintf("CREATE object '%s'", name)); err != nil {
			return err
		}

		if obj.Type != core.CREATE {
			return fmt.Errorf("CREATE object '%s' has wrong type: %s", name, obj.Type)
		}
	}

	// Validate DISCOVER objects
	for name, obj := range schema.DiscoverObjects {
		if err := validateObjectType(obj, fmt.Sprintf("DISCOVER object '%s'", name)); err != nil {
			return err
		}

		if obj.Type != core.DISCOVER {
			return fmt.Errorf("DISCOVER object '%s' has wrong type: %s", name, obj.Type)
		}
	}

	// Validate config schema if present
	if schema.ConfigSchema != nil {
		for name, prop := range schema.ConfigSchema.Properties {
			if err := validateProperty(prop, fmt.Sprintf("config property '%s'", name)); err != nil {
				return err
			}
		}

		// Check that all required properties exist
		for _, req := range schema.ConfigSchema.Required {
			if _, exists := schema.ConfigSchema.Properties[req]; !exists {
				return fmt.Errorf("required config property '%s' not defined in properties", req)
			}
		}
	}

	return nil
}

// validateObjectType validates an object type definition
func validateObjectType(obj *core.ObjectType, description string) error {
	if obj == nil {
		return fmt.Errorf("%s is nil", description)
	}

	if obj.Name == "" {
		return fmt.Errorf("%s: name is required", description)
	}

	if obj.Type != core.CREATE && obj.Type != core.DISCOVER {
		return fmt.Errorf("%s: invalid type '%s'", description, obj.Type)
	}

	// Validate properties
	for name, prop := range obj.Properties {
		if err := validateProperty(prop, fmt.Sprintf("%s property '%s'", description, name)); err != nil {
			return err
		}
	}

	// Check that all required properties exist
	for _, req := range obj.Required {
		if _, exists := obj.Properties[req]; !exists {
			return fmt.Errorf("%s: required property '%s' not defined in properties", description, req)
		}
	}

	return nil
}

// validateProperty validates a property definition
func validateProperty(prop *core.Property, description string) error {
	if prop == nil {
		return fmt.Errorf("%s is nil", description)
	}

	if prop.Type == "" {
		return fmt.Errorf("%s: type is required", description)
	}

	validTypes := []string{"string", "integer", "number", "boolean", "list", "object"}
	isValid := false
	for _, validType := range validTypes {
		if prop.Type == validType {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("%s: invalid type '%s', must be one of: %v", description, prop.Type, validTypes)
	}

	// Validate validation rules if present
	if prop.Validation != nil {
		if err := validateValidationRules(prop.Validation, description); err != nil {
			return err
		}
	}

	return nil
}

// validateValidationRules validates validation rule definitions
func validateValidationRules(validation *core.Validation, description string) error {
	// Validate length constraints
	if validation.MinLength != nil && validation.MaxLength != nil {
		if *validation.MinLength > *validation.MaxLength {
			return fmt.Errorf("%s: min_length cannot be greater than max_length", description)
		}
	}

	// Validate numeric constraints
	if validation.Minimum != nil && validation.Maximum != nil {
		if *validation.Minimum > *validation.Maximum {
			return fmt.Errorf("%s: minimum cannot be greater than maximum", description)
		}
	}

	// Validate pattern if present
	if validation.Pattern != "" {
		if _, err := regexp.Compile(validation.Pattern); err != nil {
			return fmt.Errorf("%s: invalid regex pattern '%s': %v", description, validation.Pattern, err)
		}
	}

	return nil
}

// ValidateCreateObject validates a create object configuration
func (v *SchemaValidator) ValidateCreateObject(objectType string, config map[string]interface{}) (bool, []ValidationIssue) {
	var issues []ValidationIssue

	// For now, just do basic validation - this would be expanded with proper object schemas
	if config == nil {
		issues = append(issues, ValidationIssue{
			Field:   "",
			Message: "configuration cannot be nil",
			Code:    "required",
		})
		return false, issues
	}

	// Basic required field validation based on object type
	switch objectType {
	case "table":
		if name, ok := config["name"]; !ok || name == "" {
			issues = append(issues, ValidationIssue{
				Field:   "name",
				Message: "name is required for table objects",
				Code:    "required",
			})
		}
		if columns, ok := config["columns"]; !ok || columns == nil {
			issues = append(issues, ValidationIssue{
				Field:   "columns",
				Message: "columns are required for table objects",
				Code:    "required",
			})
		}
	}

	return len(issues) == 0, issues
}

// CreateValidator creates common validation functions for provider configs
type CreateValidator struct{}

// DatabaseConnectionConfig creates validation for database connection configs
func (cv *CreateValidator) DatabaseConnectionConfig() map[string]ValidationFunc {
	return map[string]ValidationFunc{
		"host":     Compose(NotEmpty(), IsValidHostname()),
		"port":     IsValidPort(),
		"database": Compose(NotEmpty(), IsValidDatabaseName()),
		"username": NotEmpty(),
		"password": NotEmpty(), // In real world, this might come from env vars
		"sslmode":  IsInList([]string{"disable", "require", "verify-ca", "verify-full"}),
		"timeout":  Compose(IsPositive(), InRange(1, 300)), // 1-300 seconds
	}
}

// S3Config creates validation for S3-compatible storage configs
func (cv *CreateValidator) S3Config() map[string]ValidationFunc {
	return map[string]ValidationFunc{
		"region":     IsValidAWSRegion(),
		"access_key": NotEmpty(),
		"secret_key": NotEmpty(),
		"endpoint":   IsValidURL(), // For S3-compatible services
		"bucket":     IsValidS3BucketName(),
		"use_ssl": func(value interface{}, field string) error {
			// Boolean validation is built into schema validation
			return nil
		},
	}
}

// TableConfig provides validation for table configurations
func (cv *CreateValidator) TableConfig() map[string]ValidationFunc {
	return map[string]ValidationFunc{
		"name":        NotEmpty(),
		"schema":      OptionalString(),
		"columns":     RequiredSlice(),
		"constraints": OptionalSlice(),
		"indexes":     OptionalSlice(),
	}
}

// ValidateConfig validates a configuration map using provided validators
func ValidateConfig(config map[string]interface{}, validators map[string]ValidationFunc) error {
	var errors []error

	for field, validator := range validators {
		if value, exists := config[field]; exists {
			if err := validator(value, field); err != nil {
				errors = append(errors, err)
			}
		}
	}

	if len(errors) > 0 {
		return &MultiValidationError{Errors: errors}
	}

	return nil
}

// MultiValidationError represents multiple validation errors
type MultiValidationError struct {
	Errors []error
}

func (e *MultiValidationError) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}

	msg := fmt.Sprintf("validation failed with %d errors:", len(e.Errors))
	for _, err := range e.Errors {
		msg += fmt.Sprintf("\n  - %s", err.Error())
	}

	return msg
}
