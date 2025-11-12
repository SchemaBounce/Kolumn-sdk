package logging

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"time"
)

// ProviderContext represents the current provider operation context
type ProviderContext struct {
	ProviderName string
	Operation    string
	ResourceType string
	ResourceName string
	StartTime    time.Time
}

// RequestSummary provides a human-readable summary of a request
type RequestSummary struct {
	Type         string
	ResourceType string
	ResourceName string
	Fields       map[string]interface{}
	Sensitive    []string
}

// ResponseSummary provides a human-readable summary of a response
type ResponseSummary struct {
	Success    bool
	Error      string
	ResultType string
	Count      int
	Summary    string
}

// JSONToHuman converts JSON data to human-readable log format
func JSONToHuman(jsonData interface{}, context string) string {
	if jsonData == nil {
		return fmt.Sprintf("%s: <nil>", context)
	}

	// Handle different types of JSON data
	switch v := jsonData.(type) {
	case string:
		// Try to parse as JSON if it looks like JSON
		if strings.HasPrefix(v, "{") || strings.HasPrefix(v, "[") {
			var parsed interface{}
			if err := json.Unmarshal([]byte(v), &parsed); err == nil {
				return JSONToHuman(parsed, context)
			}
		}
		return fmt.Sprintf("%s: %s", context, v)

	case map[string]interface{}:
		return mapToHuman(v, context)

	case []interface{}:
		return arrayToHuman(v, context)

	default:
		return fmt.Sprintf("%s: %v", context, v)
	}
}

// mapToHuman converts a map to human-readable format
func mapToHuman(data map[string]interface{}, context string) string {
	if len(data) == 0 {
		return fmt.Sprintf("%s: <empty>", context)
	}

	var parts []string
	sensitiveFields := []string{"password", "secret", "token", "key", "credential", "auth"}

	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := data[key]
		// Check if this is a sensitive field
		isSensitive := false
		for _, sensitive := range sensitiveFields {
			if strings.Contains(strings.ToLower(key), sensitive) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			parts = append(parts, fmt.Sprintf("%s=<redacted>", key))
		} else {
			// Format value based on type
			switch v := value.(type) {
			case string:
				if len(v) > 50 {
					parts = append(parts, fmt.Sprintf("%s=%.47s...", key, v))
				} else {
					parts = append(parts, fmt.Sprintf("%s=%s", key, v))
				}
			case map[string]interface{}:
				if len(v) > 0 {
					parts = append(parts, fmt.Sprintf("%s={%d fields}", key, len(v)))
				} else {
					parts = append(parts, fmt.Sprintf("%s={}", key))
				}
			case []interface{}:
				parts = append(parts, fmt.Sprintf("%s=[%d items]", key, len(v)))
			default:
				parts = append(parts, fmt.Sprintf("%s=%v", key, value))
			}
		}
	}

	return fmt.Sprintf("%s: %s", context, strings.Join(parts, " "))
}

// arrayToHuman converts an array to human-readable format
func arrayToHuman(data []interface{}, context string) string {
	if len(data) == 0 {
		return fmt.Sprintf("%s: <empty array>", context)
	}

	// For small arrays, show details; for large arrays, show summary
	if len(data) <= 3 {
		var items []string
		for i, item := range data {
			items = append(items, fmt.Sprintf("[%d]=%v", i, item))
		}
		return fmt.Sprintf("%s: %s", context, strings.Join(items, " "))
	}

	return fmt.Sprintf("%s: [%d items]", context, len(data))
}

// LogRequest logs a request in human-readable format
func LogRequest(logger *Logger, operation string, request interface{}) {
	if logger.IsDebugEnabled() {
		// In debug mode, show full JSON
		logger.JSONDebug(fmt.Sprintf("%s request", operation), request)
	} else {
		// In normal mode, show human-readable summary
		summary := SummarizeRequest(request)
		if summary.ResourceName != "" {
			logger.Info("%s request for %s '%s'", operation, summary.ResourceType, summary.ResourceName)
		} else {
			logger.Info("%s request for %s", operation, summary.ResourceType)
		}
	}
}

// LogResponse logs a response in human-readable format
func LogResponse(logger *Logger, operation string, response interface{}, err error) {
	if err != nil {
		logger.Error("%s failed: %v", operation, err)
		if logger.IsDebugEnabled() {
			logger.JSONDebug(fmt.Sprintf("%s error response", operation), response)
		}
		return
	}

	if logger.IsDebugEnabled() {
		// In debug mode, show full JSON
		logger.JSONDebug(fmt.Sprintf("%s response", operation), response)
	} else {
		// In normal mode, show human-readable summary
		summary := SummarizeResponse(response)
		if summary.Success {
			logger.Info("%s completed successfully: %s", operation, summary.Summary)
		} else {
			logger.Warn("%s completed with issues: %s", operation, summary.Summary)
		}
	}
}

// SummarizeRequest creates a human-readable summary of a request
func SummarizeRequest(request interface{}) RequestSummary {
	summary := RequestSummary{
		Type:   "unknown",
		Fields: make(map[string]interface{}),
	}

	if request == nil {
		return summary
	}

	// Use reflection to extract fields
	v := reflect.ValueOf(request)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Handle map[string]interface{} requests
	if data, ok := request.(map[string]interface{}); ok {
		if resourceType, exists := data["resource_type"]; exists {
			summary.ResourceType = fmt.Sprintf("%v", resourceType)
		}
		if objectType, exists := data["object_type"]; exists {
			summary.ResourceType = fmt.Sprintf("%v", objectType)
		}
		if name, exists := data["name"]; exists {
			summary.ResourceName = fmt.Sprintf("%v", name)
		}
		summary.Fields = data
		return summary
	}

	// Handle struct requests using reflection
	if v.Kind() == reflect.Struct {
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			value := v.Field(i)

			// Extract common fields
			switch strings.ToLower(field.Name) {
			case "name":
				if value.IsValid() && value.CanInterface() {
					summary.ResourceName = fmt.Sprintf("%v", value.Interface())
				}
			case "resourcetype", "objecttype", "type":
				if value.IsValid() && value.CanInterface() {
					summary.ResourceType = fmt.Sprintf("%v", value.Interface())
				}
			}

			// Store field value
			if value.IsValid() && value.CanInterface() {
				summary.Fields[field.Name] = value.Interface()
			}
		}
	}

	return summary
}

// SummarizeResponse creates a human-readable summary of a response
func SummarizeResponse(response interface{}) ResponseSummary {
	summary := ResponseSummary{
		Success: true,
		Summary: "operation completed",
	}

	if response == nil {
		summary.Summary = "no response data"
		return summary
	}

	// Handle map[string]interface{} responses
	if data, ok := response.(map[string]interface{}); ok {
		// Check for error indicators
		if errorMsg, exists := data["error"]; exists && errorMsg != nil {
			summary.Success = false
			summary.Error = fmt.Sprintf("%v", errorMsg)
			summary.Summary = summary.Error
			return summary
		}

		// Check for success indicators
		if success, exists := data["success"]; exists {
			if successBool, ok := success.(bool); ok {
				summary.Success = successBool
			}
		}

		// Count items if it's a list response
		if items, exists := data["items"]; exists {
			if itemList, ok := items.([]interface{}); ok {
				summary.Count = len(itemList)
				summary.Summary = fmt.Sprintf("returned %d items", summary.Count)
			}
		}

		// Extract result type
		if resultType, exists := data["type"]; exists {
			summary.ResultType = fmt.Sprintf("%v", resultType)
		}

		return summary
	}

	// Handle struct responses using reflection
	v := reflect.ValueOf(response)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		// Look for common response fields
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			value := v.Field(i)

			if !value.IsValid() || !value.CanInterface() {
				continue
			}

			switch strings.ToLower(field.Name) {
			case "error":
				if errorValue := value.Interface(); errorValue != nil {
					summary.Success = false
					summary.Error = fmt.Sprintf("%v", errorValue)
					summary.Summary = summary.Error
				}
			case "success":
				if successBool, ok := value.Interface().(bool); ok {
					summary.Success = successBool
				}
			case "items", "results", "data":
				if value.Kind() == reflect.Slice {
					summary.Count = value.Len()
					summary.Summary = fmt.Sprintf("returned %d items", summary.Count)
				}
			}
		}
	}

	return summary
}

// WithContext creates a new logger with additional context
func WithContext(logger *Logger, context ProviderContext) *Logger {
	// Create a new logger with the same component but additional context
	contextLogger := &Logger{
		component: fmt.Sprintf("%s:%s", logger.component, context.Operation),
		level:     logger.level,
		enabled:   make(map[Level]bool),
	}

	// Copy enabled levels (safely)
	logger.mu.RLock()
	for level, enabled := range logger.enabled {
		contextLogger.enabled[level] = enabled
	}
	logger.mu.RUnlock()

	return contextLogger
}

// LogProviderOperation logs the start and completion of a provider operation
func LogProviderOperation(logger *Logger, context ProviderContext, operation func() error) error {
	startTime := time.Now()

	logger.Info("Starting %s operation on %s '%s'",
		context.Operation, context.ResourceType, context.ResourceName)

	err := operation()
	duration := time.Since(startTime)

	if err != nil {
		logger.Error("Failed %s operation on %s '%s' after %v: %v",
			context.Operation, context.ResourceType, context.ResourceName, duration, err)
	} else {
		logger.Info("Completed %s operation on %s '%s' in %v",
			context.Operation, context.ResourceType, context.ResourceName, duration)
	}

	return err
}

// LogConnectionAttempt logs database/service connection attempts
func LogConnectionAttempt(logger *Logger, endpoint string, err error) {
	// Sanitize endpoint for logging (remove credentials)
	sanitized := SanitizeEndpoint(endpoint)

	if err != nil {
		logger.Error("Failed to connect to %s: %v", sanitized, err)
	} else {
		logger.Info("Successfully connected to %s", sanitized)
	}
}

// SanitizeEndpoint removes sensitive information from connection strings
func SanitizeEndpoint(endpoint string) string {
	if endpoint == "" {
		return endpoint
	}

	if strings.Contains(endpoint, "://") {
		if parsed, err := url.Parse(endpoint); err == nil {
			var builder strings.Builder
			builder.WriteString(parsed.Scheme)
			builder.WriteString("://")

			if parsed.User != nil {
				username := parsed.User.Username()
				wroteUserInfo := false
				if username != "" {
					builder.WriteString(username)
					wroteUserInfo = true
				}
				if _, hasPassword := parsed.User.Password(); hasPassword {
					if !wroteUserInfo {
						builder.WriteString(username)
					}
					builder.WriteString(":***")
					wroteUserInfo = true
				}
				if wroteUserInfo {
					builder.WriteString("@")
				}
			}

			builder.WriteString(parsed.Host)
			if parsed.Path != "" {
				builder.WriteString(parsed.EscapedPath())
			}
			if parsed.RawQuery != "" {
				builder.WriteString("?")
				builder.WriteString(parsed.RawQuery)
			}
			if parsed.Fragment != "" {
				builder.WriteString("#")
				builder.WriteString(parsed.EscapedFragment())
			}
			return builder.String()
		}
	}

	// Remove passwords from connection strings
	parts := strings.Split(endpoint, "@")
	if len(parts) > 1 {
		// Has credentials, sanitize them
		credParts := strings.Split(parts[0], "://")
		if len(credParts) > 1 {
			protocol := credParts[0]
			userParts := strings.Split(credParts[1], ":")
			if len(userParts) > 1 {
				// Has password
				return fmt.Sprintf("%s://%s:***@%s", protocol, userParts[0], parts[1])
			}
		}
	}

	return endpoint
}

// LogSchemaValidation logs schema validation results
func LogSchemaValidation(logger *Logger, resourceType string, errors []string, warnings []string) {
	if len(errors) > 0 {
		logger.Error("Schema validation failed for %s: %d errors", resourceType, len(errors))
		if logger.IsDebugEnabled() {
			for _, err := range errors {
				logger.Debug("Validation error: %s", err)
			}
		}
	} else {
		logger.Info("Schema validation passed for %s", resourceType)
	}

	if len(warnings) > 0 {
		logger.Warn("Schema validation warnings for %s: %d warnings", resourceType, len(warnings))
		if logger.IsDebugEnabled() {
			for _, warning := range warnings {
				logger.Debug("Validation warning: %s", warning)
			}
		}
	}
}

// LogDiscoveryResult logs the results of resource discovery
func LogDiscoveryResult(logger *Logger, resourceType string, count int, duration time.Duration) {
	if count > 0 {
		logger.Info("Discovered %d %s resources in %v", count, resourceType, duration)
	} else {
		logger.Info("No %s resources found in %v", resourceType, duration)
	}
}
